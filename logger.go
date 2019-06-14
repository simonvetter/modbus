package modbus

import (
	"fmt"
	"os"
)

type logger struct {
	prefix	string
}

func newLogger(prefix string) (l *logger) {
	l = &logger{
		prefix:	prefix,
	}

	return
}

func (l *logger) Info(msg string) {
	l.write(false, fmt.Sprintf("%s [info]: %s\n", l.prefix, msg))

	return
}

func (l *logger) Infof(format string, msg ...interface{}) {
	l.write(false, fmt.Sprintf("%s [info]: %s\n", l.prefix, fmt.Sprintf(format, msg...)))

	return
}

func (l *logger) Warning(msg string) {
	l.write(false, fmt.Sprintf("%s [warn]: %s\n", l.prefix, msg))

	return
}

func (l *logger) Warningf(format string, msg ...interface{}) {
	l.write(false, fmt.Sprintf("%s [warn]: %s\n", l.prefix, fmt.Sprintf(format, msg...)))

	return
}

func (l *logger) Error(msg string) {
	l.write(false, fmt.Sprintf("%s [error]: %s\n", l.prefix, msg))

	return
}

func (l *logger) Errorf(format string, msg ...interface{}) {
	l.write(false, fmt.Sprintf("%s [error]: %s\n", l.prefix, fmt.Sprintf(format, msg...)))

	return
}

func (l *logger) Fatal(msg string) {
	l.Error(msg)
	os.Exit(1)

	return
}

func (l *logger) Fatalf(format string, msg ...interface{}) {
	l.Errorf(format, msg...)
	os.Exit(1)

	return
}

func (l *logger) write(stderr bool, msg string) {
	if stderr {
		os.Stderr.WriteString(msg)
	} else {
		os.Stdout.WriteString(msg)
	}

	return
}
