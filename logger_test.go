package modbus

import (
	"bytes"
	"log"
	"testing"
)

func TestClientCustomLogger(t *testing.T) {
	var buf bytes.Buffer
	var logger *log.Logger

	logger = log.New(&buf, "external-prefix: ", 0)

	_, _ = NewClient(&ClientConfiguration{
		Logger: logger,
		URL:    "sometype://sometarget",
	})

	if buf.String() != "external-prefix: modbus-client(sometarget) [error]: unsupported client type 'sometype'\n" {
		t.Errorf("unexpected logger output '%s'", buf.String())
	}

	return
}

func TestServerCustomLogger(t *testing.T) {
	var buf bytes.Buffer
	var logger *log.Logger

	logger = log.New(&buf, "external-prefix: ", 0)

	_, _ = NewServer(&ServerConfiguration{
		Logger: logger,
		URL:    "tcp://",
	}, nil)

	if buf.String() != "external-prefix: modbus-server() [error]: missing host part in URL 'tcp://'\n" {
		t.Errorf("unexpected logger output '%s'", buf.String())
	}

	return
}
