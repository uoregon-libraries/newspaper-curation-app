// Package logger implements global functions for logging so we have an easy
// way to centrally configure the logging mechanism used by NCA without relying
// on the old logger package that had global state.  Yes, this still has global
// state, but it's global only to NCA - a dependency of NCA won't be able to
// modify it.
package logger

import (
	"os"
	"strings"

	l "github.com/uoregon-libraries/gopkg/logger"
)

// Logger is the global logging object for all of NCA to use.  If we need to
// change the log level or otherwise customize it, this can be overwritten.
var Logger = l.New(l.Debug, false)

// Debugf logs a debug-level message. This should be disabled in production
// except when... well... debugging.
func Debugf(format string, args ...any) {
	Logger.Debugf(format, args...)
}

// Infof logs an info-level message. These should just give general info that
// helps uncover problems or tells us a train of some process we sometimes need
// to manually check.
func Infof(format string, args ...any) {
	Logger.Infof(format, args...)
}

// Warnf logs a warn-level message. Use this when something goes wrong, but
// doesn't really cause any problems, or the problems caused aren't something
// that realistically can be addressed in code.
func Warnf(format string, args ...any) {
	Logger.Warnf(format, args...)
}

// Errorf logs an error-level message. This should be used for things that we
// don't expect, and need to fix. The current code is overusing this log level,
// and should be looked at.
func Errorf(format string, args ...any) {
	Logger.Errorf(format, args...)
}

// CriticalFixNeeded logs a critical error, adding information about the error
// that is passed in, and making it clear manual fixes are required. We should
// only use this when something goes so badly wrong that dev intervention is
// likely necessary to fix issues, or something is just so unexpected that
// seeing this message means some kind of attention is needed somewhere ASAP.
func CriticalFixNeeded(message string, err error) {
	message += " (manual intervention is required, error: " + err.Error() + ")"
	message = strings.Replace(message, "%", "%%", -1)
	Logger.Criticalf(message)
}

// Fatalf logs a critical-level message, then exits. The same rules apply here
// as [Criticalf], this just lets us stop a command mid-run. Obviously
// shouldn't be used in any of the daemons unless something just unbelievably
// bad happens.
func Fatalf(format string, args ...any) {
	Logger.Fatalf(format, args...)
	os.Exit(1)
}
