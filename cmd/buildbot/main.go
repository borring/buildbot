package main

import (
	"borring/cs495/buildbot/ticket"
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

	_ "github.com/go-sql-driver/mysql"
	mysql "github.com/go-sql-driver/mysql"
)

var (
	tmpdir        = "/tmp"
	tmprootprefix = "tmpbuild-"
)

type readercpy struct {
	io.ReadCloser
	store []byte
}

func (r *readercpy) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	r.store = append(r.store, p...)
	return n, err
}

func newBuildbotHandler(fn func() ticket.Ticket) http.HandlerFunc {
	NewTicket = fn
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
		var bsig ticket.BuildSignal
		if err := decoder.Decode(&bsig); err != nil {
			log.Printf("Could not decode json: %s\n", err.Error())
			return
		}
		defer newreader.Close()

		t := NewTicket()
		defer func() {
			// Don't attempt to submit if request body is empty
			// or JSON is invalid
			if bsig.CommitHsh == "" {
				return
			}
			if err := t.SubmitTicket(bsig); err != nil {
				log.Printf("Could not submit ticket: %s\n", err.Error())
			}
		}()

		nroot, err := newRoot(tmpdir, tmprootprefix)
		if err != nil {
			t.SetErr(err.Error())
			return
		}

		t.Run(
			exec.Command("git", "clone", bsig.GitRoot, nroot),
			"Failed to copy repository",
		)

		gitcobcmd := exec.Command("git", "checkout", bsig.Branch)
		gitcobcmd.Dir = nroot
		t.Run(
			gitcobcmd,
			fmt.Sprintf("Failed to checkout branch: %q", bsig.Branch),
		)

		gitcocmd := exec.Command("git", "checkout", bsig.CommitHsh)
		gitcocmd.Dir = nroot
		t.Run(
			gitcocmd,
			fmt.Sprintf("Failed to checkout commit: %q", bsig.CommitHsh),
		)

		makecmd := exec.Command("make", bsig.Category)
		makecmd.Dir = nroot
		t.Run(
			makecmd,
			fmt.Sprintf("Build Failed! %s", bsig.CommitHsh),
		)

		testcmd := exec.Command("make", bsig.Category+"-"+"test")
		testcmd.Dir = nroot
		t.Run(
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

var NewTicket func() ticket.Ticket

func main() {
	host := flag.String("h", "localhost:3306", "Database host to connect to")
	dbname := flag.String("db", "tickets", "Which DB to connect to")
	flag.Parse()

	dsn := &mysql.Config{
		User:      "moymoy",
		Passwd:    "paspass",
		DBName:    *dbname,
		Net:       "tcp",
		Addr:      *host,
		ParseTime: true,
	}
	d, err := sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("BUILDBOT!\n")
	http.HandleFunc("/build", newBuildbotHandler(NewTicketFunc(&db{d})))
	http.ListenAndServe(":8077", nil)
}
