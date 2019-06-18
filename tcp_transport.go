package modbus

import (
	"fmt"
	"io"
	"net"
	"time"
)

const (
	maxTCPFrameLength	int = 260
	mbapHeaderLength	int = 7
)

type tcpTransport struct {
	conf		*ClientConfiguration
	logger		*logger
	socket		net.Conn
	lastTxnId	uint16
}

// Returns a new TCP transport.
func newTCPTransport(conf *ClientConfiguration) (tt *tcpTransport) {
	tt = &tcpTransport{
		conf:	conf,
		logger:	newLogger(fmt.Sprintf("tcp-transport(%s)", conf.URL)),
	}

	return
}

// Attempts to connect to the remote host.
func (tt *tcpTransport) Open() (err error) {
	var socket	net.Conn

	socket, err	= net.DialTimeout("tcp", tt.conf.URL, 5 * time.Second)
	if err == nil {
		tt.socket = socket
	}

	return
}

// Closes the underlying tcp socket.
func (tt *tcpTransport) Close() (err error) {
	err  = tt.socket.Close()

	return
}

// Writes an MBAP header + modbus request to the underlying socket.
func (tt *tcpTransport) WriteRequest(req *request) (err error) {
	var payload		[]byte
	var bytesWritten	int

	payload	= tt.assembleMBAPFrame(req)

	err	= tt.socket.SetDeadline(time.Now().Add(tt.conf.Timeout))
	if err != nil {
		return
	}

	bytesWritten, err	= tt.socket.Write(payload)
	if err != nil {
		return
	}

	if bytesWritten != len(payload) {
		err = ErrShortResponse
		return
	}

	return
}

// Reads as many MBAP+modbus frames as necessary until either the response
// matching tt.lastTxnId is received, or a timeout occurs.
func (tt *tcpTransport) ReadResponse() (res *response, err error) {
	for {
		res, err = tt.readSingleRepsonse()
		if err == ErrBadTransactionId || err == ErrUnknownProtocolId {
			continue
		}

		break
	}

	return
}

// Reads a single MBAP header + modbus response from the socket.
// Transaction and protocol ID validation is done as a last step to avoid
// erroring out halfway through reading the frame, which would
// render the connection unuseable for subsequent requests.
func (tt *tcpTransport) readSingleRepsonse() (res *response, err error) {
	var rxbuf	[]byte
	var bytesNeeded	int
	var protocolId	uint16
	var txnId	uint16

	res = &response{}

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
	res.unitId	= rxbuf[6]

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

	// the smallest valid modbus response is at least 2 bytes long
	if bytesNeeded < 2 {
		err	= ErrShortResponse
		return
	}

	// validate the protocol identifier
	if protocolId != 0x0000 {
		err = ErrUnknownProtocolId
		tt.logger.Warningf("received unexpected protocol id 0x%04x", protocolId)
		return
	}

	// validate the transaction identifier
	if tt.lastTxnId != txnId {
		err = ErrBadTransactionId
		tt.logger.Warningf("received unexpected transaction id " +
				   "(expected 0x%04x, received 0x%04x)",
				   tt.lastTxnId, txnId)
		return
	}

	// store response code and payload in the response object
	res.responseCode	= rxbuf[0]
	res.payload		= rxbuf[1:]

	return
}

func (tt *tcpTransport) assembleMBAPFrame(req *request) (payload []byte) {
	// transaction identifier
	tt.lastTxnId++
	payload	= uint16ToBytes(BIG_ENDIAN, tt.lastTxnId)
	// protocol identifier (always 0x0000)
	payload	= append(payload, 0x00, 0x00)
	// length (covers unit identifier + function code + payload fields)
	payload = append(payload, uint16ToBytes(BIG_ENDIAN, uint16(2 + len(req.payload)))...)
	// unit identifier
	payload	= append(payload, req.unitId)
	// request payload (function code + payload)
	payload	= append(payload, req.functionCode)
	payload	= append(payload, req.payload...)

	return
}
