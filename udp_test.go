package modbus

import (
	"net"
	"os"
	"testing"
	"time"
)

func TestUDPSockWrapper(t *testing.T) {
	var err error
	var usw *udpSockWrapper
	var sock1 *net.UDPConn
	var sock2 *net.UDPConn
	var addr *net.UDPAddr
	var txchan chan []byte
	var rxbuf []byte
	var count int

	addr, err = net.ResolveUDPAddr("udp", "localhost:5502")
	if err != nil {
		t.Errorf("failed to resolve udp address: %v", err)
		return
	}

	txchan = make(chan []byte, 4)
	// get a pair of UDP sockets ready to talk to each other
	sock1, err = net.ListenUDP("udp", addr)
	if err != nil {
		t.Errorf("failed to listen on udp socket: %v", err)
		return
	}
	err = sock1.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		t.Errorf("failed to set deadline on udp socket: %v", err)
		return
	}

	sock2, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Errorf("failed to open udp socket: %v", err)
		return
	}
	// the feedTestPipe goroutine will forward any slice of bytes
	// pushed into txchan over UDP to our test UDP sock wrapper object
	go feedTestPipe(t, txchan, sock2)

	usw = newUDPSockWrapper(sock1)
	// push a valid RTU response (illegal data address) to the test pipe
	txchan <- []byte{
		0x31, 0x82, // unit id and response code
		0x02,       // exception code
		0xc1, 0x6e, // CRC
	}
	// then push random junk
	txchan <- []byte{
		0xaa, 0xbb, 0xcc,
	}
	// then some more
	txchan <- []byte{
		0xdd, 0xee,
	}

	// attempt to read 3 bytes: we should get them as the first datagram
	// is 5 bytes long
	rxbuf = make([]byte, 3)
	count, err = usw.Read(rxbuf)
	if err != nil {
		t.Errorf("usw.Read() should have succeeded, got: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 bytes, got: %v", count)
	}
	for idx, val := range []byte{
		0x31, 0x82, 0x02,
	} {
		if rxbuf[idx] != val {
			t.Errorf("expected 0x%02x at pos %v, got: 0x%02x",
				val, idx, rxbuf[idx])
		}
	}

	// attempt to read 1 byte: we should get the 4th byte of the
	// first datagram, of which we've been holding on to bytes #4 and 5
	rxbuf = make([]byte, 1)
	count, err = usw.Read(rxbuf)
	if err != nil {
		t.Errorf("usw.Read() should have succeeded, got: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 byte, got: %v", count)
	}
	if rxbuf[0] != 0xc1 {
		t.Errorf("expected 0xc1 at pos 0, got: 0x%02x", rxbuf[0])
	}

	// attempt to read 5 bytes: we should get the last byte of the
	// first datagram, which the udpSockWrapper object still holds in
	// its buffer
	rxbuf = make([]byte, 5)
	count, err = usw.Read(rxbuf)
	if err != nil {
		t.Errorf("usw.Read() should have succeeded, got: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 byte, got: %v", count)
	}
	if rxbuf[0] != 0x6e {
		t.Errorf("expected 0x6e at pos 0, got: 0x%02x", rxbuf[0])
	}

	// attempt to read 10 bytes: we should get all 3 bytes of the 2nd
	// datagram
	rxbuf = make([]byte, 10)
	count, err = usw.Read(rxbuf)
	if err != nil {
		t.Errorf("usw.Read() should have succeeded, got: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 bytes, got: %v", count)
	}
	for idx, val := range []byte{
		0xaa, 0xbb, 0xcc,
	} {
		if rxbuf[idx] != val {
			t.Errorf("expected 0x%02x at pos %v, got: 0x%02x",
				val, idx, rxbuf[idx])
		}
	}

	// attempt to read 40 bytes: we should get both bytes of the 3rd
	// datagram
	rxbuf = make([]byte, 40)
	count, err = usw.Read(rxbuf)
	if err != nil {
		t.Errorf("usw.Read() should have succeeded, got: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 bytes, got: %v", count)
	}
	for idx, val := range []byte{
		0xdd, 0xee,
	} {
		if rxbuf[idx] != val {
			t.Errorf("expected 0x%02x at pos %v, got: 0x%02x",
				val, idx, rxbuf[idx])
		}
	}

	// attempt to read 7 bytes: we should get a read timeout as we've
	// consumed all bytes from all datagrams and no more are coming
	rxbuf = make([]byte, 7)
	count, err = usw.Read(rxbuf)
	if !os.IsTimeout(err) {
		t.Errorf("usw.Read() should have failed with a timeout error, got: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 bytes, got: %v", count)
	}

	// cleanup
	sock1.Close()
	sock2.Close()

	return
}
