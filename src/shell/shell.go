// Package shell centralizes common exec.Cmd functionality
package shell

import (
	"bytes"

	"os/exec"
	"strings"
	"syscall"

	"github.com/uoregon-libraries/gopkg/logger"
)

func _exec(cmd *exec.Cmd, binary string, args ...string) (ok bool) {
	logger.Debugf(`Running "%s %s"`, binary, strings.Replace(strings.Join(args, " "), "%", "%%", -1))
	var output, err = cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(`Failed to run "%s %s": %s`, binary, strings.Join(args, " "), err)
		for _, line := range bytes.Split(output, []byte("\n")) {
			logger.Debugf("--> %s", line)
		}

		return false
	}

	return true
}

// Exec attempts to run the given command, using logger to give consistent
// formatting to whatever the command spits out if an error occurs
func Exec(binary string, args ...string) (ok bool) {
	var cmd = exec.Command(binary, args...)
	return _exec(cmd, binary, args...)
}

// ExecSubgroup is just like Exec, but sets the process to run in its own group
// so it doesn't get killed on CTRL+C
func ExecSubgroup(binary string, args ...string) (ok bool) {
	var cmd = exec.Command(binary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return _exec(cmd, binary, args...)
}

// ExecSubgroupWithContext is just like ExecSubgroup, but with a string context
func ExecSubgroupWithContext(binary string, context string, args ...string) (ok bool) {
	var cmd = exec.Command(binary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	args = append(args, context)
	return _exec(cmd, binary, args...)
}
