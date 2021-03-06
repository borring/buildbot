package main

import (
	"borring/cs495/buildbot/ticket"
	"database/sql"
	"fmt"
	"log"
	"time"
)

var (
	tidfmt       = "20060102150405"
	timestampfmt = "2006-01-02 15:04:05"
)

type ticketdb struct {
	ticket.Ticket
	d *db
}

func NewTicketFunc(d *db) func() ticket.Ticket {
	return func() ticket.Ticket {
		return &ticketdb{
			Ticket: ticket.NewTicket(),
			d:      d,
		}
	}
}

func (tk *ticketdb) SubmitTicket(bsig ticket.BuildSignal) error {
	if !tk.IsErr() {
		return nil
	}

	log.Printf("%s\n", tk.GetErr())
	log.Printf("Generating ticket\n")

	tnow := time.Now()
	tid := tnow.Format(tidfmt)
	timestamp := tnow.Format(timestampfmt)

	ver := fmt.Sprintf("Branch: %s\nCommit: %s\n%s\n\n",
		bsig.Branch, bsig.CommitHsh, bsig.CommitMsg)

	logmsg := tk.GetLog()
	logmsg = append([]byte(ver), logmsg...)

	err := tk.d.transaction(func(t *sql.Tx) ([]interface{}, *sql.Stmt, error) {
		str := "INSERT into w_mei_tickets (tid, dateCreated, lastUpdated, status, title, description, submitter, priority, category, assignee) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		args := []interface{}{tid, timestamp, timestamp, "Open", tk.GetErr(), string(logmsg), "buildbot", 2, bsig.Category, ""}
		stmt, err := t.Prepare(str)
		return args, stmt, err
	})
	return err
}
