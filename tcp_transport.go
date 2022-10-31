package modbus

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const (
	maxTCPFrameLength	int = 260
	mbapHeaderLength	int = 7
)

type tcpTransport struct {
	logger		*logger
	socket		net.Conn
	timeout		time.Duration
	lastTxnId	uint16
}

// Returns a new TCP transport.
func newTCPTransport(socket net.Conn, timeout time.Duration, customLogger *log.Logger) (tt *tcpTransport) {
	tt = &tcpTransport{
		socket:		socket,
		timeout:	timeout,
		logger:		newLogger(fmt.Sprintf("tcp-transport(%s)", socket.RemoteAddr()), customLogger),
	}

	return
}

// Closes the underlying tcp socket.
func (tt *tcpTransport) Close() (err error) {
	err  = tt.socket.Close()

	return
}

// Runs a request across the socket and returns a response.
func (tt *tcpTransport) ExecuteRequest(req *pdu) (res *pdu, err error) {
	// set an i/o deadline on the socket (read and write)
	err	= tt.socket.SetDeadline(time.Now().Add(tt.timeout))
	if err != nil {
		return
	}

	// increase the transaction ID counter
	tt.lastTxnId++

	_, err	= tt.socket.Write(tt.assembleMBAPFrame(tt.lastTxnId, req))
	if err != nil {
		return
	}

	res, err = tt.readResponse()

	return
}

// Reads a request from the socket.
func (tt *tcpTransport) ReadRequest() (req *pdu, err error) {
	var txnId	uint16

	// set an i/o deadline on the socket (read and write)
	err	= tt.socket.SetDeadline(time.Now().Add(tt.timeout))
	if err != nil {
		return
	}

	req, txnId, err	= tt.readMBAPFrame()
	if err != nil {
		return
	}

	// store the incoming transaction id
	tt.lastTxnId	= txnId

	return
}

// Writes a response to the socket.
func (tt *tcpTransport) WriteResponse(res *pdu) (err error) {
	_, err	= tt.socket.Write(tt.assembleMBAPFrame(tt.lastTxnId, res))
	if err != nil {
		return
	}

	return
}

// Reads as many MBAP+modbus frames as necessary until either the response
// matching tt.lastTxnId is received or an error occurs.
func (tt *tcpTransport) readResponse() (res *pdu, err error) {
	var txnId	uint16

	for {
		// grab a frame
		res, txnId, err	= tt.readMBAPFrame()

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
			tt.logger.Warningf("received unexpected transaction id " +
					   "(expected 0x%04x, received 0x%04x)",
					   tt.lastTxnId, txnId)
			continue
		}

		break
	}

	return
}

// Reads an entire frame (MBAP header + modbus PDU) from the socket.
func (tt *tcpTransport) readMBAPFrame() (p *pdu, txnId uint16, err error) {
	var rxbuf	[]byte
	var bytesNeeded	int
	var protocolId	uint16
	var unitId	uint8

	// read the MBAP header
	rxbuf		= make([]byte, mbapHeaderLength)
	_, err		= io.ReadFull(tt.socket, rxbuf)
	if err != nil {
		return
	}

	// decode the transaction identifier
	txnId		= bytesToUint16(BIG_ENDIAN, rxbuf[0:2])
	// decode the protocol identifier
	protocolId	= bytesToUint16(BIG_ENDIAN, rxbuf[2:4])
	// store the source unit id
	unitId		= rxbuf[6]

	// determine how many more bytes we need to read
	bytesNeeded	= int(bytesToUint16(BIG_ENDIAN, rxbuf[4:6]))

	// the byte count includes the unit ID field, which we already have
	bytesNeeded--

	// never read more than the max allowed frame length
	if bytesNeeded + mbapHeaderLength > maxTCPFrameLength {
		err = ErrProtocolError
		return
	}

	// an MBAP length of 0 is illegal
	if bytesNeeded <= 0 {
		err = ErrProtocolError
		return
	}

	// read the PDU
	rxbuf		= make([]byte, bytesNeeded)
	_, err		= io.ReadFull(tt.socket, rxbuf)
	if err != nil {
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
		unitId:		unitId,
		functionCode:	rxbuf[0],
		payload:	rxbuf[1:],
	}

	return
}

// Turns a PDU into an MBAP frame (MBAP header + PDU) and returns it as bytes.
func (tt *tcpTransport) assembleMBAPFrame(txnId uint16, p *pdu) (payload []byte) {
	// transaction identifier
	payload	= uint16ToBytes(BIG_ENDIAN, txnId)
	// protocol identifier (always 0x0000)
	payload	= append(payload, 0x00, 0x00)
	// length (covers unit identifier + function code + payload fields)
	payload = append(payload, uint16ToBytes(BIG_ENDIAN, uint16(2 + len(p.payload)))...)
	// unit identifier
	payload	= append(payload, p.unitId)
	// function code
	payload	= append(payload, p.functionCode)
	// payload
	payload	= append(payload, p.payload...)

	return
}
