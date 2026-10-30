package modbus

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestTCPServerReadQuantityExceptions(t *testing.T) {
	for _, fc := range []byte{fcReadCoils, fcReadDiscreteInputs, fcReadHoldingRegisters, fcReadInputRegisters} {
		limit := uint16(125)
		if fc == fcReadCoils || fc == fcReadDiscreteInputs {
			limit = 2000
		}
		for _, quantity := range []uint16{0, limit + 1, 0xffff, limit} {
			t.Run(fmt.Sprintf("function_%02x_quantity_%d", fc, quantity), func(t *testing.T) {
				server, err := NewServer(&ServerConfiguration{URL: "tcp://127.0.0.1:0"}, &tcpTestHandler{})
				if err != nil {
					t.Fatal(err)
				}
				if err := server.Start(); err != nil {
					t.Fatal(err)
				}
				defer server.Stop()
				conn, err := net.Dial("tcp", server.tcpListener.Addr().String())
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()
				if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
					t.Fatal(err)
				}
				// A complete MBAP frame with an invalid read quantity must get
				// an exception, with its transaction and unit identifiers intact.
				request := []byte{0, 1, 0, 0, 0, 6, 9, fc, 0, 0, byte(quantity >> 8), byte(quantity)}
				if _, err := conn.Write(request); err != nil {
					t.Fatal(err)
				}
				got := make([]byte, 9)
				if _, err := io.ReadFull(conn, got); err != nil {
					t.Fatalf("reading exception response: %v", err)
				}
				exception := exIllegalDataValue
				if quantity == limit {
					// A legal quantity reaches the test handler, whose data map
					// has only ten entries, and gets an address exception instead.
					exception = exIllegalDataAddress
				}
				want := []byte{0, 1, 0, 0, 0, 3, 9, fc | 0x80, exception}
				if !bytes.Equal(got, want) {
					t.Fatalf("response = %x, want %x", got, want)
				}
				// The same connection must still accept a subsequent valid read.
				request[1], request[10], request[11] = 2, 0, 1
				if _, err := conn.Write(request); err != nil {
					t.Fatal(err)
				}
				want = []byte{0, 2, 0, 0, 0, 4, 9, fc, 1, 0}
				if fc == fcReadHoldingRegisters || fc == fcReadInputRegisters {
					want = []byte{0, 2, 0, 0, 0, 5, 9, fc, 2, 0, 0}
				}
				got = make([]byte, len(want))
				if _, err := io.ReadFull(conn, got); err != nil {
					t.Fatalf("reading valid response after exception: %v", err)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("subsequent response = %x, want %x", got, want)
				}
			})
		}
	}
}
