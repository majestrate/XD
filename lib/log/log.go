package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"github.com/majestrate/XD/lib/sync"
	//t "github.com/majestrate/XD/lib/translate"
)

var mtx sync.Mutex

type logLevel int

const (
	debug = logLevel(0)
	info  = logLevel(1)
	warn  = logLevel(2)
	err   = logLevel(3)
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
	case err:
		return "ERR"
	case fatal:
		return "FTL"
	default:
		return "???"
	}

}

var level = info

// SetLevel sets global logger level
func SetLevel(l string) {
	l = strings.ToLower(l)
	if l == "debug" {
		level = debug
	} else if l == "info"{
		level = info
	} else if l == "warn"{
		level = warn
	} else if l == "err"{
		level = err
	} else if l == "fatal"{
		level = fatal
	} else {
		panic(fmt.Sprintf("invalid log level: '%s'", l))
	}
}

var out io.Writer = os.Stdout

// SetOutput sets logging to output to a writer
func SetOutput(w io.Writer) {
	out = w
}

func accept(lvl logLevel) bool {
	return lvl.Int() >= level.Int()
}

func log(lvl logLevel, f string, args ...interface{}) {
	if accept(lvl) {
		m := fmt.Sprintf(f, args...)
		t := time.Now()
		mtx.Lock()
		fmt.Fprintf(out, "%s[%s] %s\t%s%s", lvl.Color(), lvl.Name(), t, m, colorReset)
		fmt.Fprintln(out)
		mtx.Unlock()
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
	log(err, msg)
}

// Errorf prints formatted error log message
func Errorf(f string, args ...interface{}) {
	log(err, f, args...)
}

// Fatal print fatal error and panic
func Fatal(msg string) {
	log(fatal, msg)
}

// Fatalf print formatted fatal error and panic
func Fatalf(f string, args ...interface{}) {
	log(fatal, f, args...)
}
