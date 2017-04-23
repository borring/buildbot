package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func testingDB(fn func(*db)) error {
	dbserver := "moymoy:paspass@/tickets_testing?parseTime=true"
	d, err := sql.Open("mysql", dbserver)
	if err != nil {
		return err
	}

	fn(&db{d})

	d.Close()
	return nil
}

func TestNewRoot(t *testing.T) {
	testnewroot := func(tmp string, sf bool) {
		nroot, err := newRoot(tmp, tmprootprefix)
		if err != nil && sf {
			t.Fatalf("Failed to make new tmp root directory: %s\n", err)
		}
		var fst os.FileInfo
		if fst, err = os.Stat(nroot); err != nil && sf {
			t.Fatalf("Failed to stat new tmp root directory: %s\n", err)
		}
		if fst == nil {
			return
		}
		if !fst.IsDir() {
			t.Fatalf("Failed to stat new tmp root directory: tmpdir is not directory\n")
		}

		os.Remove(nroot)
	}

	testnewroot("/tmp", true)
	testnewroot("/tmp/somedir", false)
}

type ticketSchema struct {
	Pid         int       `sql:"pid"`
	Tid         string    `sql:"tid"`
	DateCreated time.Time `sql:"dateCreated"`
	LastUpdated time.Time `sql:"lastUpdated"`
	Status      string    `sql:"status"`
	Title       string    `sql:"title"`
	Description string    `sql:"description"`
	Submitter   string    `sql:"submitter"`
	Priority    int       `sql:"priority"`
	Category    string    `sql:"category"`
}

func TestNewTicket(t *testing.T) {
	bsig := buildSignal{
		CommitHsh: "abcd423",
		CommitMsg: "crappy work",
	}
	tk := ticket{
		submitter: "buildbot",
		errmsg:    "something went wrong",
		logmsg:    []byte("this happened, that happened"),
	}
	r := ticketSchema{}

	if err := testingDB(func(d *db) {
		count, err := testTicketInsertion(d, &r, tk.submitter, func() {
			if err := newTicket(d, bsig, tk); err != nil {
				t.Fatalf("Could not create new ticket %s", err.Error())
			}
		})
		if err != nil {
			t.Errorf("Could not count tickets: %s", err.Error())
		}
		if count != 1 {
			t.Fatalf("Failed to add new ticket")
		}
	}); err != nil {
		t.Fatal(err)
	}

	if !r.DateCreated.Equal(r.LastUpdated) {
		t.Fatalf("Dates don't match")
	}

	if r.Status != "open" {
		t.Errorf("Status is not open: %q", r.Status)
	}

	if len(r.Tid) != 14 {
		t.Errorf("Len of tid is wrong: %q", r.Tid)
	}
}

func TestBuildbotHandler(t *testing.T) {
	testHandler := func(bsig buildSignal, ecount int) {
		r := ticketSchema{}
		jsonbsig, err := json.Marshal(bsig)
		if err != nil {
			t.Errorf("Couldn't marshal json from bsig: %s", err.Error())
		}

		json := strings.NewReader(string(jsonbsig))
		req, err := http.NewRequest("POST", "/build", json)
		if err != nil {
			log.Fatalf("Could not create new Request: %s\n", err.Error())
		}
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		if err := testingDB(func(d *db) {
			handler := newBuildbotHandler(d)
			count, err := testTicketInsertion(d, &r, "buildbot", func() {
				handler(rec, req)
			})
			if err != nil {
				t.Errorf("Could not count tickets: %s", err.Error())
			}
			if count != ecount {
				t.Fatalf("Failed to insert a ticket: expected to create %d tickets, made %d",
					ecount, count)
			}
		}); err != nil {
			t.Fatal(err)
		}
	}

	bsig := buildSignal{
		CommitHsh: "28302c98c83693e9455835f4e5b76e9aebb7daa6",
		CommitMsg: "test: fix directory reference for buildfail target",
		GitRoot:   "./testdir",
		Branch:    "master",
	}

	bsig.Category = "buildfail"
	testHandler(bsig, 1)

	bsig.Category = "testfail"
	testHandler(bsig, 1)

	bsig.Category = "testok"
	testHandler(bsig, 0)
}

func Testfmtrgx(t *testing.T) {
	str := "Some string with substitution: %s\n"
	if !fmtrgx.MatchString(str) {
		t.Errorf("fmtrgx does not match string: %q", str)
	}

	str = "string with no substitution\n"
	if fmtrgx.MatchString(str) {
		t.Errorf("fmtrgx matched string: %q", str)
	}
}

func testTicketInsertion(d *db, rowInfo *ticketSchema, submitter string, fn func()) (int, error) {
	counteri, err := ticketCounter(d, rowInfo, submitter)
	if err != nil {
		return -1, fmt.Errorf("could not count tickets: %s", err.Error())
	}

	fn()

	counter, err := ticketCounter(d, rowInfo, submitter)
	if err != nil {
		return -1, fmt.Errorf("could not count tickets: %s", err.Error())
	}

	return counter - counteri, nil
}

func ticketCounter(d *db, r *ticketSchema, submitter string) (int, error) {
	rows, err := d.Query("SELECT * FROM w_mei_tickets WHERE submitter=?", submitter)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	var counter int
	for rows.Next() {
		if err := rows.Scan(
			&r.Pid,
			&r.Tid,
			&r.DateCreated,
			&r.LastUpdated,
			&r.Status,
			&r.Title,
			&r.Description,
			&r.Submitter,
			&r.Priority,
			&r.Category,
		); err != nil {
			return -1, fmt.Errorf("couldn't scan row: %s\n", err.Error())
		}
		counter++
	}
	return counter, nil
}
