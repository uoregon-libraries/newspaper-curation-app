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

// Debug logs a debug-level message using the default logger
func Debug(format string, args ...interface{}) {
	DefaultLogger.Debug(format, args...)
}

// Debug logs a debug-level message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(debug, fmt.Sprintf(format, args...))
}

// Info logs an info-level message using the default logger
func Info(format string, args ...interface{}) {
	DefaultLogger.Info(format, args...)
}

// Info logs an info-level message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(info, fmt.Sprintf(format, args...))
}

// Warn logs a warn-level message using the default logger
func Warn(format string, args ...interface{}) {
	DefaultLogger.Warn(format, args...)
}

// Warn logs a warn-level message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(warn, fmt.Sprintf(format, args...))
}

// Error logs an error-level message using the default logger
func Error(format string, args ...interface{}) {
	DefaultLogger.Error(format, args...)
}

// Error logs an error-level message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(err, fmt.Sprintf(format, args...))
}

// Critical logs a critical-level message using the default logger
func Critical(format string, args ...interface{}) {
	DefaultLogger.Critical(format, args...)
}

// Critical logs a critical-level message
func (l *Logger) Critical(format string, args ...interface{}) {
	l.log(crit, fmt.Sprintf(format, args...))
}

// Fatal logs a critical-level message using the default logger, then exits
func Fatal(format string, args ...interface{}) {
	DefaultLogger.Fatal(format, args...)
}

// Fatal logs a critical-level message, then exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(crit, fmt.Sprintf(format, args...))
	os.Exit(1)
}
