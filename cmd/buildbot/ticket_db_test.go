package main

import (
	"borring/cs495/buildbot/ticket"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

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
	Assignee    string    `sql:"assignee"`
}

func TestSubmitTicket(t *testing.T) {
	bsig := ticket.BuildSignal{
		CommitHsh: "d7cbe0ac4098f56338c74859d2b39498b64ba224",
		CommitMsg: "pa04: fix nil pointer reference\n changes commited\n\tmodified: bull.sh\n\tmodified: crazy.bat",
		Branch:    "feature-v2",
		Category:  "mock-module",
	}
	cmd := exec.Command(filepath.Join(testdir, "failcmd"))

	r := ticketSchema{}

	testingDB(t, func(d *db) {

		tk := NewTicketFunc(d)()
		tk.SetSubmitter("buildbot")
		tk.Run(cmd, "something went wrong")

		count, err := testTicketInsertion(d, &r, tk.GetSubmitter(), func() {
			if err := tk.SubmitTicket(bsig); err != nil {
				t.Fatalf("Could not create new ticket %s", err.Error())
			}
		})
		if err != nil {
			t.Errorf("Could not count tickets: %s", err.Error())
		}
		if count != 1 {
			t.Fatalf("Failed to add new ticket")
		}
	})

	if !r.DateCreated.Equal(r.LastUpdated) {
		t.Fatalf("Dates don't match")
	}

	if r.Status != "Open" {
		t.Errorf("Status is not Open: %q", r.Status)
	}

	if len(r.Tid) != 14 {
		t.Errorf("Len of tid is wrong: %q", r.Tid)
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
			&r.Assignee,
		); err != nil {
			return -1, fmt.Errorf("couldn't scan row: %s\n", err.Error())
		}
		counter++
	}
	return counter, nil
}
