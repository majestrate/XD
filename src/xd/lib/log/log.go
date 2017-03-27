package log

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type logLevel int

const (
	debug = logLevel(0)
	info  = logLevel(1)
	warn  = logLevel(2)
	error = logLevel(3)
	fatal = logLevel(4)
)

func (l logLevel) Int() int {
	return int(l)
}

func (l logLevel) Name() string {

	switch l {
	case debug:
		return "DBG"
	case info:
		return "NFO"
	case warn:
		return "WRN"
	case error:
		return "ERR"
	case fatal:
		return "FTL"
	default:
		return "???"
	}

}

func (l logLevel) Color() string {
	switch l {
	case debug:
		return "\x1b[37;0m"
	case info:
		return "\x1b[37;1m"
	case warn:
		return "\x1b[33;1m"
	default:
		return "\x1b[31;1m"
	}
}

var level = info

// SetLevel sets global logger level
func SetLevel(l string) {
	l = strings.ToLower(l)
	if l == "debug" {
		level = debug
	} else {
		level = info
	}
}

var out = os.Stderr

func accept(lvl logLevel) bool {
	return lvl.Int() >= level.Int()
}

func log(lvl logLevel, f string, args ...interface{}) {
	if accept(lvl) {
		m := fmt.Sprintf(f, args...)
		t := time.Now()
		fmt.Fprintf(out, "%s[%s] %s %s\x1b[0;0m\n", lvl.Color(), lvl.Name(), t, m)
		if lvl == fatal {
			panic(m)
		}
	}
}

// Debug prints debug message
func Debug(msg string) {
	log(debug, msg)
}

// Debugf prints formatted debug message
func Debugf(f string, args ...interface{}) {
	log(debug, f, args...)
}

// Info prints info log message
func Info(msg string) {
	log(info, msg)
}

// Infof prints formatted info log message
func Infof(f string, args ...interface{}) {
	log(info, f, args...)
}

// Warn prints warn log message
func Warn(msg string) {
	log(warn, msg)
}

// Warnf prints formatted warn log message
func Warnf(f string, args ...interface{}) {
	log(warn, f, args...)
}

// Error prints error log message
func Error(msg string) {
	log(error, msg)
}

// Errorf prints formatted error log message
func Errorf(f string, args ...interface{}) {
	log(error, f, args...)
}

// Fatal print fatal error and panic
func Fatal(msg string) {
	log(fatal, msg)
}

// Fatalf print formatted fatal error and panic
func Fatalf(f string, args ...interface{}) {
	log(fatal, f, args...)
}
