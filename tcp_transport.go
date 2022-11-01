package mbserver

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	maxTCPFrameLength int = 260
	mbapHeaderLength  int = 7
)

type tcpTransport struct {
	logger    *logger
	socket    net.Conn
	timeout   time.Duration
	lastTxnId uint16
}

// Returns a new TCP transport.
func newTCPTransport(socket net.Conn, timeout time.Duration) *tcpTransport {
	return &tcpTransport{
		socket:  socket,
		timeout: timeout,
		logger:  newLogger(fmt.Sprintf("tcp-transport(%s)", socket.RemoteAddr())),
	}
}

// Closes the underlying tcp socket.
func (tt *tcpTransport) Close() (err error) {
	return tt.socket.Close()
}

// Runs a request across the socket and returns a response.
func (tt *tcpTransport) ExecuteRequest(req *pdu) (*pdu, error) {
	// set an i/o deadline on the socket (read and write)
	if err := tt.socket.SetDeadline(time.Now().Add(tt.timeout)); err != nil {
		return nil, err
	}

	// increase the transaction ID counter
	tt.lastTxnId++

	if _, err := tt.socket.Write(tt.assembleMBAPFrame(tt.lastTxnId, req)); err != nil {
		return nil, err
	}

	return tt.readResponse()
}

// Reads a request from the socket.
func (tt *tcpTransport) ReadRequest() (*pdu, error) {
	// set an i/o deadline on the socket (read and write)
	if err := tt.socket.SetDeadline(time.Now().Add(tt.timeout)); err != nil {
		return nil, err
	}

	req, txnId, err := tt.readMBAPFrame()
	if err != nil {
		return nil, err
	}

	// store the incoming transaction id
	tt.lastTxnId = txnId

	return req, err
}

// Writes a response to the socket.
func (tt *tcpTransport) WriteResponse(res *pdu) (err error) {
	_, err = tt.socket.Write(tt.assembleMBAPFrame(tt.lastTxnId, res))
	return err
}

// Reads as many MBAP+modbus frames as necessary until either the response
// matching tt.lastTxnId is received or an error occurs.
func (tt *tcpTransport) readResponse() (res *pdu, err error) {
	var txnId uint16

	for {
		// grab a frame
		res, txnId, err = tt.readMBAPFrame()

		// ignore unknown protocol identifiers
		if err == ErrUnknownProtocolId {
			continue
		}

		// abort on any other erorr
		if err != nil {
			return
		}

		// ignore unknown transaction identifiers
		if tt.lastTxnId != txnId {
			tt.logger.Warningf("received unexpected transaction id (expected 0x%04x, received 0x%04x)", tt.lastTxnId, txnId)
			continue
		}

		return
	}
}

// Reads an entire frame (MBAP header + modbus PDU) from the socket.
func (tt *tcpTransport) readMBAPFrame() (p *pdu, txnId uint16, err error) {
	// read the MBAP header
	rxbuf := make([]byte, mbapHeaderLength)
	if _, err = io.ReadFull(tt.socket, rxbuf); err != nil {
		return
	}

	// decode the transaction identifier
	txnId = binary.BigEndian.Uint16(rxbuf[0:2])
	// decode the protocol identifier
	protocolId := binary.BigEndian.Uint16(rxbuf[2:4])
	// store the source unit id
	unitId := rxbuf[6]

	// determine how many more bytes we need to read
	bytesNeeded := int(binary.BigEndian.Uint16(rxbuf[4:6]))

	// the byte count includes the unit ID field, which we already have
	bytesNeeded--

	// never read more than the max allowed frame length
	if bytesNeeded+mbapHeaderLength > maxTCPFrameLength {
		err = ErrProtocolError
		return
	}

	// an MBAP length of 0 is illegal
	if bytesNeeded <= 0 {
		err = ErrProtocolError
		return
	}

	// read the PDU
	rxbuf = make([]byte, bytesNeeded)
	if _, err = io.ReadFull(tt.socket, rxbuf); err != nil {
		return
	}

	// validate the protocol identifier
	if protocolId != 0x0000 {
		err = ErrUnknownProtocolId
		tt.logger.Warningf("received unexpected protocol id 0x%04x", protocolId)
		return
	}

	// store unit id, function code and payload in the PDU object
	p = &pdu{
		unitId:       unitId,
		functionCode: rxbuf[0],
		payload:      rxbuf[1:],
	}

	return
}

// Turns a PDU into an MBAP frame (MBAP header + PDU) and returns it as bytes.
func (tt *tcpTransport) assembleMBAPFrame(txnId uint16, p *pdu) []byte {
	payload := make([]byte, 0, 8+len(p.payload))
	// transaction identifier
	payload = append(payload, asBytes(binary.BigEndian, txnId)...)
	// protocol identifier (always 0x0000)
	payload = append(payload, 0x00, 0x00)
	// length (covers unit identifier + function code + payload fields)
	payload = append(payload, asBytes(binary.BigEndian, uint16(2+len(p.payload)))...)
	// unit identifier
	payload = append(payload, p.unitId)
	// function code
	payload = append(payload, p.functionCode)
	// payload
	payload = append(payload, p.payload...)

	return payload
}
