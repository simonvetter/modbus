package modbus

import (
	"fmt"
	"io"
	"log"
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
	lastTxnID uint16
}

// Returns a new TCP transport.
func newTCPTransport(socket net.Conn, timeout time.Duration, customLogger *log.Logger) (tt *tcpTransport) {
	tt = &tcpTransport{
		socket:  socket,
		timeout: timeout,
		logger:  newLogger(fmt.Sprintf("tcp-transport(%s)", socket.RemoteAddr()), customLogger),
	}

	return
}

// Closes the underlying tcp socket.
func (tt *tcpTransport) Close() (err error) {
	err = tt.socket.Close()

	return
}

// Runs a request across the socket and returns a response.
func (tt *tcpTransport) ExecuteRequest(req *pdu) (res *pdu, err error) {
	// set an i/o deadline on the socket (read and write)
	err = tt.socket.SetDeadline(time.Now().Add(tt.timeout))
	if err != nil {
		return
	}

	// increase the transaction ID counter
	tt.lastTxnID++

	_, err = tt.socket.Write(tt.assembleMBAPFrame(tt.lastTxnID, req))
	if err != nil {
		return
	}

	res, err = tt.readResponse()

	return
}

// Reads a request from the socket.
func (tt *tcpTransport) ReadRequest() (req *pdu, err error) {
	var txnID uint16

	// set an i/o deadline on the socket (read and write)
	err = tt.socket.SetDeadline(time.Now().Add(tt.timeout))
	if err != nil {
		return
	}

	req, txnID, err = tt.readMBAPFrame()
	if err != nil {
		return
	}

	// store the incoming transaction id
	tt.lastTxnID = txnID

	return
}

// Writes a response to the socket.
func (tt *tcpTransport) WriteResponse(res *pdu) (err error) {
	_, err = tt.socket.Write(tt.assembleMBAPFrame(tt.lastTxnID, res))
	if err != nil {
		return
	}

	return
}

// Reads as many MBAP+modbus frames as necessary until either the response
// matching tt.lastTxnId is received or an error occurs.
func (tt *tcpTransport) readResponse() (res *pdu, err error) {
	var txnID uint16

	for {
		// grab a frame
		res, txnID, err = tt.readMBAPFrame()

		// ignore unknown protocol identifiers
		if err == ErrUnknownProtocolID {
			continue
		}

		// abort on any other erorr
		if err != nil {
			return
		}

		// ignore unknown transaction identifiers
		if tt.lastTxnID != txnID {
			tt.logger.Warningf("received unexpected transaction id "+
				"(expected 0x%04x, received 0x%04x)",
				tt.lastTxnID, txnID)
			continue
		}

		break
	}

	return
}

// Reads an entire frame (MBAP header + modbus PDU) from the socket.
func (tt *tcpTransport) readMBAPFrame() (p *pdu, txnID uint16, err error) {
	var rxbuf []byte
	var bytesNeeded int
	var protocolID uint16
	var unitID uint8

	// read the MBAP header
	rxbuf = make([]byte, mbapHeaderLength)
	_, err = io.ReadFull(tt.socket, rxbuf)
	if err != nil {
		return
	}

	// decode the transaction identifier
	txnID = bytesToUint16(BigEndian, rxbuf[0:2])
	// decode the protocol identifier
	protocolID = bytesToUint16(BigEndian, rxbuf[2:4])
	// store the source unit id
	unitID = rxbuf[6]

	// determine how many more bytes we need to read
	bytesNeeded = int(bytesToUint16(BigEndian, rxbuf[4:6]))

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
	_, err = io.ReadFull(tt.socket, rxbuf)
	if err != nil {
		return
	}

	// validate the protocol identifier
	if protocolID != 0x0000 {
		err = ErrUnknownProtocolID
		tt.logger.Warningf("received unexpected protocol id 0x%04x", protocolID)
		return
	}

	// store unit id, function code and payload in the PDU object
	p = &pdu{
		unitID:       unitID,
		functionCode: rxbuf[0],
		payload:      rxbuf[1:],
	}

	return
}

// Turns a PDU into an MBAP frame (MBAP header + PDU) and returns it as bytes.
func (tt *tcpTransport) assembleMBAPFrame(txnID uint16, p *pdu) (payload []byte) {
	// transaction identifier
	payload = uint16ToBytes(BigEndian, txnID)
	// protocol identifier (always 0x0000)
	payload = append(payload, 0x00, 0x00)
	// length (covers unit identifier + function code + payload fields)
	payload = append(payload, uint16ToBytes(BigEndian, uint16(2+len(p.payload)))...)
	// unit identifier
	payload = append(payload, p.unitID)
	// function code
	payload = append(payload, p.functionCode)
	// payload
	payload = append(payload, p.payload...)

	return
}
