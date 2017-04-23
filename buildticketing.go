package main

import (
	"fmt"
	"os/exec"
	"regexp"
)

type buildSignal struct {
	CommitHsh string `json:"commit-hash"`
	CommitMsg string `json:"commit-message"`
	GitRoot   string `json:"git-root"`
	Branch    string `json:"branch"`
	Category  string `json:"category"`
}

type ticket struct {
	submitter string
	errmsg    string
	logmsg    []byte
}

var fmtrgx = regexp.MustCompile(`%\w`)

func (tk *ticket) run(cmd *exec.Cmd, errstrf string) bool {
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
