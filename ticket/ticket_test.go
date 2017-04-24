package ticket

import (
	"os/exec"
	"strings"
	"testing"
)

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

func TestRun(t *testing.T) {
	cmdnerr := exec.Command("grep", `a`)
	cmdnerr.Stdin = strings.NewReader("a")

	tk := NewTicket()

	tk.Run(cmdnerr, "Ayyy this error shouldn't be here")
	if tk.IsErr() {
		t.Errorf("Run logged an error when there shouldn't be any: %s",
			tk.GetErr())
	}

	cmderr := exec.Command("grep", `a`)
	cmderr.Stdin = strings.NewReader("b")
	tk.Run(cmderr, "This error should be here")
	if !tk.IsErr() {
		t.Errorf("cmd finished successfully when it shouldn't have")
	}
}
