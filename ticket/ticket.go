package ticket

import (
	"fmt"
	"os/exec"
	"regexp"
)

type BuildSignal struct {
	CommitHsh string `json:"commit-hash"`
	CommitMsg string `json:"commit-message"`
	GitRoot   string `json:"git-root"`
	Branch    string `json:"branch"`
	Category  string `json:"category"`
}

type Ticket interface {
	Run(*exec.Cmd, string) bool
	SubmitTicket(BuildSignal) error
	SetSubmitter(string)
	GetSubmitter() string
	IsErr() bool
	SetErr(string)
	GetErr() string
	GetLog() []byte
}

type ticket struct {
	submitter string
	errmsg    string
	logmsg    []byte
}

var fmtrgx = regexp.MustCompile(`%\w`)

func NewTicket() Ticket {
	return &ticket{}
}

func (tk *ticket) Run(cmd *exec.Cmd, errstrf string) bool {
	if tk.errmsg != "" {
		return false
	}

	output, err := cmd.CombinedOutput()
	if err == nil {
		return true // ran successfully
	}

	if fmtrgx.MatchString(errstrf) {
		tk.errmsg = fmt.Sprintf(errstrf, err)
	} else {
		tk.errmsg = errstrf
	}
	tk.logmsg = output

	return false
}

func (tk *ticket) SubmitTicket(bsig BuildSignal) error {
	return nil
}

func (tk *ticket) SetSubmitter(s string) {
	tk.submitter = s
}

func (tk *ticket) GetSubmitter() string {
	return tk.submitter
}

func (tk *ticket) IsErr() bool {
	return tk.errmsg != ""
}

func (tk *ticket) SetErr(s string) {
	if tk.IsErr() {
		return
	}
	tk.errmsg = s
}

func (tk *ticket) GetErr() string {
	return tk.errmsg
}

func (tk *ticket) GetLog() []byte {
	return tk.logmsg
}
