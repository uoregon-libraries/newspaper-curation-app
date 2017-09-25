// Package shell centralizes common exec.Cmd functionality
package shell

import (
	"bytes"
	"logger"
	"os/exec"
	"strings"
)

// Exec attempts to run the given command, using logger to give consistent
// formatting to whatever the command spits out if an error occurs
func Exec(binary string, args ...string) (ok bool) {
	var cmd = exec.Command(binary, args...)
	logger.Debug(`Running "%s %s"`, binary, strings.Replace(strings.Join(args, " "), "%", "%%", -1))
	var output, err = cmd.CombinedOutput()
	if err != nil {
		logger.Error(`Failed to run "%s %s": %s`, binary, strings.Join(args, " "), err)
		for _, line := range bytes.Split(output, []byte("\n")) {
			logger.Debug("--> %s", line)
		}

		return false
	}

	return true
}
