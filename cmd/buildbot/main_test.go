package main

import (
	"borring/cs495/buildbot/ticket"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

var testdir = "../../testdir"

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

func TestBuildbotHandler(t *testing.T) {
	testHandler := func(bsig ticket.BuildSignal, ecount int) {
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

		c := make(chan int, 2)
		handler := newBuildbotHandler(NewMockTicketfunc(c))
		handler(rec, req)

		count := <-c
		if count != ecount {
			t.Fatalf("Failed to insert a ticket: expected to create %d tickets, made %d",
				ecount, count)
		}
	}

	bsig := ticket.BuildSignal{
		CommitHsh: "28302c98c83693e9455835f4e5b76e9aebb7daa6",
		CommitMsg: "test: fix directory reference for buildfail target",
		GitRoot:   testdir,
		Branch:    "master",
	}

	bsig.Category = "buildfail"
	testHandler(bsig, 1)

	bsig.Category = "testfail"
	testHandler(bsig, 1)

	bsig.Category = "testok"
	testHandler(bsig, 0)
}

// Mock ticket =================================================================

type mock_ticket struct {
	ticket.Ticket
	comm chan int
}

func NewMockTicketfunc(c chan int) func() ticket.Ticket {
	return func() ticket.Ticket {
		return &mock_ticket{
			Ticket: ticket.NewTicket(),
			comm:   c,
		}
	}
}

func (tk *mock_ticket) SubmitTicket(bsig ticket.BuildSignal) error {
	if !tk.IsErr() {
		tk.comm <- 0
		return nil
	}

	tk.comm <- 1
	return nil
}
