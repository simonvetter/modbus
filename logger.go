package modbus

import (
	"fmt"
	"log"
	"os"
)

type logger struct {
	prefix       string
	customLogger *log.Logger
}

func newLogger(prefix string, customLogger *log.Logger) (l *logger) {
	l = &logger{
		prefix:       prefix,
		customLogger: customLogger,
	}

	return
}

func (l *logger) Info(msg string) {
	l.write(fmt.Sprintf("%s [info]: %s\n", l.prefix, msg))
}

func (l *logger) Infof(format string, msg ...interface{}) {
	l.write(fmt.Sprintf("%s [info]: %s\n", l.prefix, fmt.Sprintf(format, msg...)))

}

func (l *logger) Warning(msg string) {
	l.write(fmt.Sprintf("%s [warn]: %s\n", l.prefix, msg))
}

func (l *logger) Warningf(format string, msg ...interface{}) {
	l.write(fmt.Sprintf("%s [warn]: %s\n", l.prefix, fmt.Sprintf(format, msg...)))
}

func (l *logger) Error(msg string) {
	l.write(fmt.Sprintf("%s [error]: %s\n", l.prefix, msg))
}

func (l *logger) Errorf(format string, msg ...interface{}) {
	l.write(fmt.Sprintf("%s [error]: %s\n", l.prefix, fmt.Sprintf(format, msg...)))
}

func (l *logger) Fatal(msg string) {
	l.Error(msg)
	os.Exit(1)
}

func (l *logger) Fatalf(format string, msg ...interface{}) {
	l.Errorf(format, msg...)
	os.Exit(1)
}

func (l *logger) write(msg string) {
	if l.customLogger == nil {
		os.Stdout.WriteString(msg)
	} else {
		l.customLogger.Print(msg)
	}
}
