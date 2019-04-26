// Package shell centralizes common exec.Cmd functionality
package shell

import (
	"bytes"

	"os/exec"
	"strings"
	"syscall"

	"github.com/uoregon-libraries/gopkg/logger"
)

func _exec(cmd *exec.Cmd, binary string, jobLogger *logger.Logger, args ...string) (ok bool) {
	jobLogger.Debugf(`Running "%s %s"`, binary, strings.Replace(strings.Join(args, " "), "%", "%%", -1))
	var output, err = cmd.CombinedOutput()
	if err != nil {
		jobLogger.Log.Errorf(`Failed to run "%s %s": %s`, binary, strings.Join(args, " "), err)
		for _, line := range bytes.Split(output, []byte("\n")) {
			jobLogger.Debugf("--> %s", line)
		}

		return false
	}

	return true
}

// Exec attempts to run the given command, using logger to give consistent
// formatting to whatever the command spits out if an error occurs
func Exec(binary string, jobLogger *logger.Logger, args ...string) (ok bool) {
	var cmd = exec.Command(binary, args...)
	return _exec(cmd, binary, jobLogger, args...)
}

// ExecSubgroup is just like Exec, but sets the process to run in its own group
// so it doesn't get killed on CTRL+C
func ExecSubgroup(binary string, jobLogger *logger.Logger, args ...string) (ok bool) {
	var cmd = exec.Command(binary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return _exec(cmd, binary, jobLogger, args...)
}
