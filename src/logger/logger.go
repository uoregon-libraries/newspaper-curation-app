// package logger centralizes logging things in a way that gives similar output
// to the Python tools.  For now, there is no filtering via log levels, and the
// app name is figured out just by pulling os.Args[0] rather than being
// manually set.
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type logLevel string

const (
	debug logLevel = "DEBUG"
	info  logLevel = "INFO"
	warn  logLevel = "WARN"
	err   logLevel = "ERROR"
	crit  logLevel = "CRIT"
)

// log is the central logger for all helpers to use
func log(level logLevel, message string) {
	var timeString = time.Now().Format("2006/01/02 15:04:05.000")
	var appName = filepath.Base(os.Args[0])
	var output = fmt.Sprintf("%s - %s - %s - %s\n", timeString, appName, level, message)
	fmt.Fprintf(os.Stderr, output)
}

// Debug logs a debug-level message
func Debug(format string, args ...interface{}) {
	log(debug, fmt.Sprintf(format, args...))
}

// Info logs an info-level message
func Info(format string, args ...interface{}) {
	log(info, fmt.Sprintf(format, args...))
}

// Warn logs a warn-level message
func Warn(format string, args ...interface{}) {
	log(warn, fmt.Sprintf(format, args...))
}

// Error logs an error-level message
func Error(format string, args ...interface{}) {
	log(err, fmt.Sprintf(format, args...))
}

// Critical logs a critical-level message
func Critical(format string, args ...interface{}) {
	log(crit, fmt.Sprintf(format, args...))
}

func Fatal(format string, args ...interface{}) {
	log(crit, fmt.Sprintf(format, args...))
	os.Exit(1)
}
