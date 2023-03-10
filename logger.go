package mbserver

import (
	"fmt"
	"os"
)

type LeveledLogger interface {
	Info(msg string)
	Infof(format string, msg ...interface{})
	Warning(msg string)
	Warningf(format string, msg ...interface{})
	Error(msg string)
	Errorf(format string, msg ...interface{})
	Fatal(msg string)
	Fatalf(format string, msg ...interface{})
}

var _ LeveledLogger = (*logger)(nil)

type logger struct {
	prefix string
}

func newLogger(prefix string) (l *logger) {
	l = &logger{
		prefix: prefix,
	}

	return
}

func (l *logger) Info(msg string) {
	l.write(false, fmt.Sprintf("%s [info]: %s\n", l.prefix, msg))
}

func (l *logger) Infof(format string, msg ...interface{}) {
	l.write(false, fmt.Sprintf("%s [info]: %s\n", l.prefix, fmt.Sprintf(format, msg...)))
}

func (l *logger) Warning(msg string) {
	l.write(false, fmt.Sprintf("%s [warn]: %s\n", l.prefix, msg))
}

func (l *logger) Warningf(format string, msg ...interface{}) {
	l.write(false, fmt.Sprintf("%s [warn]: %s\n", l.prefix, fmt.Sprintf(format, msg...)))
}

func (l *logger) Error(msg string) {
	l.write(false, fmt.Sprintf("%s [error]: %s\n", l.prefix, msg))
}

func (l *logger) Errorf(format string, msg ...interface{}) {
	l.write(false, fmt.Sprintf("%s [error]: %s\n", l.prefix, fmt.Sprintf(format, msg...)))
}

func (l *logger) Fatal(msg string) {
	l.Error(msg)
	os.Exit(1)
}

func (l *logger) Fatalf(format string, msg ...interface{}) {
	l.Errorf(format, msg...)
	os.Exit(1)
}

func (l *logger) write(stderr bool, msg string) {
	if stderr {
		os.Stderr.WriteString(msg)
	} else {
		os.Stdout.WriteString(msg)
	}
}
