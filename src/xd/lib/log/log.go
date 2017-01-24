package log

import (
	"fmt"
	native "log"
)

const (
	debug = 0
	info  = 1
	warn  = 2
	error = 3
	fatal = 4
)

var level = debug

func accept(lvl int) bool {
	return lvl >= level
}

func log(lvl int, f string, args ...interface{}) {
	if accept(lvl) {
		m := fmt.Sprintf(f, args...)
		native.Printf("[%d] %s", lvl, m)
		if lvl == fatal {
			panic(m)
		}
	}
}

func Debug(msg string) {
	log(debug, msg)
}

func Debugf(f string, args ...interface{}) {
	log(debug, f, args...)
}

func Info(msg string) {
	log(info, msg)
}

func Infof(f string, args ...interface{}) {
	log(info, f, args...)
}

func Warn(msg string) {
	log(warn, msg)
}

func Warnf(f string, args ...interface{}) {
	log(warn, f, args...)
}

func Error(msg string) {
	log(error, msg)
}

func Errorf(f string, args ...interface{}) {
	log(error, f, args...)
}

func Fatal(msg string) {
	log(fatal, msg)
}

func Fatalf(f string, args ...interface{}) {
	log(fatal, f, args...)
}
