// Package logger implements global functions for logging so we have an easy
// way to centrally configure the logging mechanism used by NCA without relying
// on the old logger package that had global state.  Yes, this still has global
// state, but it's global only to NCA - a dependency of NCA won't be able to
// modify it.
package logger

import (
	"fmt"
	"os"

	l "github.com/uoregon-libraries/gopkg/logger"
)

// Logger is the global logging object for all of NCA to use.  If we need to
// change the log level or otherwise customize it, this can be overwritten.
var Logger = l.New(l.Debug)

// Debugf logs a debug-level message
func Debugf(format string, args ...interface{}) {
	Logger.Debugf(fmt.Sprintf(format, args...))
}

// Infof logs an info-level message
func Infof(format string, args ...interface{}) {
	Logger.Infof(fmt.Sprintf(format, args...))
}

// Warnf logs a warn-level message
func Warnf(format string, args ...interface{}) {
	Logger.Warnf(fmt.Sprintf(format, args...))
}

// Errorf logs an error-level message
func Errorf(format string, args ...interface{}) {
	Logger.Errorf(fmt.Sprintf(format, args...))
}

// Criticalf logs a critical-level message
func Criticalf(format string, args ...interface{}) {
	Logger.Criticalf(fmt.Sprintf(format, args...))
}

// Fatalf logs a critical-level message, then exits
func Fatalf(format string, args ...interface{}) {
	Logger.Fatalf(fmt.Sprintf(format, args...))
	os.Exit(1)
}
