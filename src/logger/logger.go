// Package logger centralizes logging things in a way that gives similar output
// to the Python tools.  For now, there is no filtering via log levels, and the
// app name is figured out just by pulling os.Args[0] rather than being
// manually set.
package logger

import (
	"fmt"
	"io"
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

// Logger holds basic data to format log messages
type Logger struct {
	TimeFormat string
	AppName    string
	Output     io.Writer
}

// DefaultLogger gives an app semi-sane logging without creating and managing a
// Logger instance
var DefaultLogger = Logger{
	TimeFormat: "2006/01/02 15:04:05.000",
	AppName:    filepath.Base(os.Args[0]),
	Output:     os.Stderr,
}

// log is the central logger for all helpers to use
func (l *Logger) log(level logLevel, message string) {
	var timeString = time.Now().Format(l.TimeFormat)
	var output = fmt.Sprintf("%s - %s - %s - %s\n", timeString, l.AppName, level, message)
	fmt.Fprintf(l.Output, output)
}

// Debugf logs a debug-level message using the default logger
func Debugf(format string, args ...interface{}) {
	DefaultLogger.Debugf(format, args...)
}

// Debugf logs a debug-level message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(debug, fmt.Sprintf(format, args...))
}

// Infof logs an info-level message using the default logger
func Infof(format string, args ...interface{}) {
	DefaultLogger.Infof(format, args...)
}

// Infof logs an info-level message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(info, fmt.Sprintf(format, args...))
}

// Warnf logs a warn-level message using the default logger
func Warnf(format string, args ...interface{}) {
	DefaultLogger.Warnf(format, args...)
}

// Warnf logs a warn-level message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(warn, fmt.Sprintf(format, args...))
}

// Errorf logs an error-level message using the default logger
func Errorf(format string, args ...interface{}) {
	DefaultLogger.Errorf(format, args...)
}

// Errorf logs an error-level message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(err, fmt.Sprintf(format, args...))
}

// Criticalf logs a critical-level message using the default logger
func Criticalf(format string, args ...interface{}) {
	DefaultLogger.Criticalf(format, args...)
}

// Criticalf logs a critical-level message
func (l *Logger) Criticalf(format string, args ...interface{}) {
	l.log(crit, fmt.Sprintf(format, args...))
}

// Fatalf logs a critical-level message using the default logger, then exits
func Fatalf(format string, args ...interface{}) {
	DefaultLogger.Fatalf(format, args...)
}

// Fatalf logs a critical-level message, then exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(crit, fmt.Sprintf(format, args...))
	os.Exit(1)
}
