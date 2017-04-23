package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	tmpdir        = "/tmp"
	tmprootprefix = "tmpbuild-"
)

func newTicket(d *db, bsig buildSignal, tk ticket) error {
	tnow := time.Now()
	tid := tnow.Format("20060102150405")
	timestamp := tnow.Format("2006-01-02 15:04:05")

	ver := fmt.Sprintf("Branch: %s\nCommit: %s\n\n%s\n\n",
		bsig.CommitHsh, bsig.CommitMsg, bsig.Branch)

	tk.logmsg = append([]byte(ver), tk.logmsg...)

	err := d.transaction(func(t *sql.Tx) ([]interface{}, *sql.Stmt, error) {
		str := "INSERT into w_mei_tickets (tid, dateCreated, lastUpdated, status, title, description, submitter, priority, category) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
		args := []interface{}{tid, timestamp, timestamp, "open", tk.errmsg, string(tk.logmsg), "buildbot", 2, "3"}
		stmt, err := t.Prepare(str)
		return args, stmt, err
	})
	return err
}

type readercpy struct {
	io.ReadCloser
	store []byte
}

func (r *readercpy) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	r.store = append(r.store, p...)
	return n, err
}

func newBuildbotHandler(d *db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			return
		}

		newreader := &readercpy{
			ReadCloser: r.Body,
		}

		io.Copy(w, newreader)
		log.Printf("%s\n", string(newreader.store))

		decoder := json.NewDecoder(bytes.NewBuffer(newreader.store))
		var bsig buildSignal
		if err := decoder.Decode(&bsig); err != nil {
			log.Printf("Could not decode json: %s\n", err.Error())
			return
		}
		defer newreader.Close()

		var t ticket
		defer func() {
			if t.errmsg == "" {
				return
			}

			log.Printf("%s\n", t.errmsg)
			log.Printf("Generating ticket\n")
			if err := newTicket(d, bsig, t); err != nil {
				log.Printf("Could not submit ticket: %s\n", err.Error())
			}
		}()

		nroot, err := newRoot(tmpdir, tmprootprefix)
		if err != nil {
			t.errmsg = err.Error()
			return
		}

		t.run(
			exec.Command("git", "clone", bsig.GitRoot, nroot),
			"Failed to copy repository",
		)

		gitcobcmd := exec.Command("git", "checkout", bsig.Branch)
		gitcobcmd.Dir = nroot
		t.run(
			gitcobcmd,
			fmt.Sprintf("Failed to checkout branch: %q", bsig.Branch),
		)

		gitcocmd := exec.Command("git", "checkout", bsig.CommitHsh)
		gitcocmd.Dir = nroot
		t.run(
			gitcocmd,
			fmt.Sprintf("Failed to checkout commit: %q", bsig.CommitHsh),
		)

		makecmd := exec.Command("make", bsig.Category)
		makecmd.Dir = nroot
		t.run(
			makecmd,
			fmt.Sprintf("Build Failed! %s", bsig.CommitHsh),
		)

		testcmd := exec.Command("make", bsig.Category+"-"+"test")
		testcmd.Dir = nroot
		t.run(
			testcmd,
			fmt.Sprintf("Test Failed! %s", bsig.CommitHsh),
		)
	}
}

func newRoot(dir, prefix string) (string, error) {
	fst, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("could not stat tmpdir: %s\n", err.Error())
	}
	if !fst.IsDir() {
		return "", fmt.Errorf("tmpdir %s exists but is not directory\n", fst.Name())
	}
	return ioutil.TempDir(dir, prefix)
}

func main() {
	dbname := flag.String("db", "tickets", "Which DB to connect to")
	flag.Parse()
	dbserver := fmt.Sprintf("moymoy:paspass@/%s?parseTime=true", *dbname)
	d, err := sql.Open("mysql", dbserver)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("BUILDBOT!\n")
	http.HandleFunc("/build", newBuildbotHandler(&db{d}))
	http.ListenAndServe(":8077", nil)
}
