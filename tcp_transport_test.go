package modbus

import (
	"net"
	"testing"
	"time"
)

func TestAssembleMBAPFrame(t *testing.T) {
	var tt		*tcpTransport
	var frame	[]byte

	tt		= newTCPTransport(&ClientConfiguration{})
	tt.lastTxnId	= 0x9218

	frame		= tt.assembleMBAPFrame(&request{
		unitId:		0x33,
		functionCode:	0x11,
		payload:	[]byte{0x22, 0x33, 0x44, 0x55},
	})
	// expect 7 bytes of MBAP header + 1 bytes of function code + 4 bytes of payload
	if len(frame) != 12 {
		t.Errorf("expected 12 bytes, got %v", len(frame))
	}
	for i, b := range []byte{
		0x92, 0x19, // transaction identifier (big endian)
		0x00, 0x00, // protocol identifier
		0x00, 0x06, // length (big endian)
		0x33, 0x11, // unit id and function code
		0x22, 0x33, // payload
		0x44, 0x55, // payload
	} {
		if frame[i] != b {
			t.Errorf("expected 0x%02x at position %v, got 0x%02x", b, i, frame[i])
		}
	}

	frame		= tt.assembleMBAPFrame(&request{
		unitId:		0x31,
		functionCode:	0x06,
		payload:	[]byte{0x12, 0x34},
	})
	// expect 7 bytes of MBAP header + 1 bytes of function code + 2 bytes of payload
	if len(frame) != 10 {
		t.Errorf("expected 10 bytes, got %v", len(frame))
	}
	for i, b := range []byte{
		0x92, 0x1a, // transaction identifier (big endian)
		0x00, 0x00, // protocol identifier
		0x00, 0x04, // length (big endian)
		0x31, 0x06, // unit id and function code
		0x12, 0x34, // payload
	} {
		if frame[i] != b {
			t.Errorf("expected 0x%02x at position %v, got 0x%02x", b, i, frame[i])
		}
	}

	return
}

func TestTCPTransportReadResponse(t *testing.T) {
	var tt		*tcpTransport
	var p1, p2	net.Conn
	var txchan	chan []byte
	var err		error
	var res		*response

	txchan		= make(chan []byte, 2)
	p1, p2		= net.Pipe()
	go feedTestPipe(t, txchan, p1)


	tt		= newTCPTransport(&ClientConfiguration{
		Timeout: 10 * time.Millisecond,
	})
	tt.lastTxnId	= 0x9218
	tt.socket	= p2

	// read a valid response
	txchan		<- []byte{
		0x92, 0x18, // transaction identifier (big endian)
		0x00, 0x00, // protocol identifier
		0x00, 0x04, // length (big endian)
		0x31, 0x06, // unit id and function code
		0x12, 0x34, // payload
	}
	res, err	= tt.ReadResponse()
	if err != nil {
		t.Errorf("ReadResponse() should have succeeded, got %v", err)
	}
	if res.unitId != 0x31 {
		t.Errorf("expected 0x31 as unit id, got 0x%02x", res.unitId)
	}
	if res.responseCode != 0x06 {
		t.Errorf("expected 0x06 as function code, got 0x%02x", res.responseCode)
	}
	if len(res.payload) != 2 {
		t.Errorf("expected a length of 2, got %v", len(res.payload))
	}
	if res.payload[0] != 0x12 || res.payload[1] != 0x34 {
		t.Errorf("expected {0x12, 0x34} as payload, got {0x%02x, 0x%02x}",
			 res.payload[0], res.payload[1])
	}

	// read a frame with an unexpected transaction id followed by a frame with a
	// matching transaction id: the first frame should be silently skipped
	txchan		<- []byte{
		0x92, 0x19, // transaction identifier (big endian)
		0x00, 0x00, // protocol identifier
		0x00, 0x04, // length (big endian)
		0x31, 0x06, // unit id and function code
		0x12, 0x34, // payload
	}
	txchan		<- []byte{
		0x92, 0x18, // transaction identifier (big endian)
		0x00, 0x00, // protocol identifier
		0x00, 0x04, // length (big endian)
		0x39, 0x02, // unit id and function code
		0x10, 0x01, // payload
	}
	res, err	= tt.ReadResponse()
	if err != nil {
		t.Errorf("ReadResponse() should have succeeded, got %v", err)
	}
	if res.unitId != 0x39 {
		t.Errorf("expected 0x39 as unit id, got 0x%02x", res.unitId)
	}
	if res.responseCode != 0x02 {
		t.Errorf("expected 0x02 as function code, got 0x%02x", res.responseCode)
	}
	if len(res.payload) != 2 {
		t.Errorf("expected a length of 2, got %v", len(res.payload))
	}
	if res.payload[0] != 0x10 || res.payload[1] != 0x01 {
		t.Errorf("expected {0x10, 0x01 as payload, got {0x%02x, 0x%02x}",
			 res.payload[0], res.payload[1])
	}

	// read a frame with an illegal length, preceded by a frame with an unexpected
	// protocol ID. While the first frame should be skipped without error,
	// the second should yield an ErrProtocolError.
	txchan		<- []byte{
		0x92, 0x18, // transaction identifier (big endian)
		0x00, 0x01, // protocol identifier
		0x00, 0x04, // length (big endian)
		0x31, 0x06, // unit id and function code
		0x12, 0x34, // payload
	}
	txchan		<- []byte{
		0x92, 0x18, // transaction identifier (big endian)
		0x00, 0x00, // protocol identifier
		0x00, 0x01, // length (big endian)
		0x31,       // unit id
	}
	res, err	= tt.ReadResponse()
	if err != ErrProtocolError {
		t.Errorf("ReadResponse() should have returned ErrProtocolError, got %v", err)
	}

	// read a valid frame again
	txchan		<- []byte{
		0x92, 0x18, // transaction identifier (big endian)
		0x00, 0x00, // protocol identifier
		0x00, 0x0a, // length (big endian)
		0x31, 0x32, // unit id and function code
		0x44, 0x55, // payload
		0x66, 0x77, // payload
		0x88, 0x99, // payload
		0xaa, 0xbb, // payload
	}
	res, err	= tt.ReadResponse()
	if err != nil {
		t.Errorf("ReadResponse() should have succeeded, got %v", err)
	}
	if res.unitId != 0x31 {
		t.Errorf("expected 0x31 as unit id, got 0x%02x", res.unitId)
	}
	if res.responseCode != 0x32 {
		t.Errorf("expected 0x32 as response code, got 0x%02x", res.responseCode)
	}
	if len(res.payload) != 8 {
		t.Errorf("expected a length of 8, got %v", len(res.payload))
	}
	for i, b := range []byte{
		0x44, 0x55,
		0x66, 0x77,
		0x88, 0x99,
		0xaa, 0xbb,
	} {
		if res.payload[i] != b {
			t.Errorf("expected 0x%02x at position %v, got 0x%02x",
				 b, i, res.payload[i])
		}
	}

	// read a huge frame
	txchan		<- []byte{
		0x92, 0x18, // transaction identifier (big endian)
		0x00, 0x00, // protocol identifier
		0x10, 0x0a, // length (big endian)
		0x31,       // unit id
	}
	res, err	= tt.ReadResponse()
	if err != ErrProtocolError {
		t.Errorf("ReadResponse() should have returned ErrProtocolError, got %v", err)
	}

	p1.Close()
	p2.Close()

	return
}
