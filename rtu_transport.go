package modbus

import (
	"fmt"
	"io"
	"log"
	"time"
)

const (
	maxRTUFrameLength	int = 256
)

type rtuTransport struct {
	logger       *logger
	link         rtuLink
	timeout      time.Duration
	lastActivity time.Time
	t35          time.Duration
	t1           time.Duration
}

type rtuLink interface {
	Close()		(error)
	Read([]byte)	(int, error)
	Write([]byte)	(int, error)
	SetDeadline(time.Time)	(error)
}

// Returns a new RTU transport.
func newRTUTransport(link rtuLink, addr string, speed uint, timeout time.Duration, customLogger *log.Logger) (rt *rtuTransport) {
	rt = &rtuTransport{
		logger:  newLogger(fmt.Sprintf("rtu-transport(%s)", addr), customLogger),
		link:    link,
		timeout: timeout,
		t1:      serialCharTime(speed),
	}

	if speed >= 19200 {
		// for baud rates equal to or greater than 19200 bauds, a fixed value of
		// 1750 uS is specified for t3.5.
		rt.t35 = 1750 * time.Microsecond
	} else {
		// for lower baud rates, the inter-frame delay should be 3.5 character times
		rt.t35 = (serialCharTime(speed) * 35) / 10
	}

	return
}

// Closes the rtu link.
func (rt *rtuTransport) Close() (err error) {
	err = rt.link.Close()

	return
}

// Runs a request across the rtu link and returns a response.
func (rt *rtuTransport) ExecuteRequest(req *pdu) (res *pdu, err error) {
	var ts time.Time
	var t  time.Duration
	var n  int

	// set an i/o deadline on the link
	err	= rt.link.SetDeadline(time.Now().Add(rt.timeout))
	if err != nil {
		return
	}

	// if the line was active less than 3.5 char times ago,
	// let t3.5 expire before transmitting
	t = time.Since(rt.lastActivity.Add(rt.t35))
	if t < 0 {
		time.Sleep(t * (-1))
	}

	ts = time.Now()

	// build an RTU ADU out of the request object and
	// send the final ADU+CRC on the wire
	n, err	= rt.link.Write(rt.assembleRTUFrame(req))
	if err != nil {
		return
	}

	// estimate how long the serial line was busy for.
	// note that on most platforms, Write() will be buffered and return
	// immediately rather than block until the buffer is drained
	rt.lastActivity = ts.Add(time.Duration(n) * rt.t1)

	// observe inter-frame delays
	time.Sleep(rt.lastActivity.Add(rt.t35).Sub(time.Now()))

	// read the response back from the wire
	res, err = rt.readRTUFrame()

	if err == ErrBadCRC || err == ErrProtocolError || err == ErrShortFrame {
		// wait for and flush any data coming off the link to allow
		// devices to re-sync
		time.Sleep(time.Duration(maxRTUFrameLength) * rt.t1)
		discard(rt.link)
	}

	// mark the time if we heard anything back
	if err != ErrRequestTimedOut {
		rt.lastActivity = time.Now()
	}

	return
}

// Reads a request from the rtu link.
func (rt *rtuTransport) ReadRequest() (req *pdu, err error) {
	// reading requests from RTU links is currently unsupported
	err	= fmt.Errorf("unimplemented")

	return
}

// Writes a response to the rtu link.
func (rt *rtuTransport) WriteResponse(res *pdu) (err error) {
	var n int

	// build an RTU ADU out of the request object and
	// send the final ADU+CRC on the wire
	n, err	= rt.link.Write(rt.assembleRTUFrame(res))
	if err != nil {
		return
	}

	rt.lastActivity = time.Now().Add(rt.t1 * time.Duration(n))

	return
}

// Waits for, reads and decodes a frame from the rtu link.
func (rt *rtuTransport) readRTUFrame() (res *pdu, err error) {
	var rxbuf	[]byte
	var byteCount	int
	var bytesNeeded	int
	var crc		crc

	rxbuf		= make([]byte, maxRTUFrameLength)

	// read the serial ADU header: unit id (1 byte), function code (1 byte) and
	// PDU length/exception code (1 byte)
	byteCount, err	= io.ReadFull(rt.link, rxbuf[0:3])
	if (byteCount > 0 || err == nil) && byteCount != 3 {
		err = ErrShortFrame
		return
	}
	if err != nil && err != io.ErrUnexpectedEOF {
		return
	}

	// figure out how many further bytes to read
	bytesNeeded, err = expectedResponseLenth(uint8(rxbuf[1]), uint8(rxbuf[2]))
	if err != nil {
		return
	}

	// we need to read 2 additional bytes of CRC after the payload
	bytesNeeded	+= 2

	// never read more than the max allowed frame length
	if byteCount + bytesNeeded > maxRTUFrameLength {
		err	= ErrProtocolError
		return
	}

	byteCount, err	= io.ReadFull(rt.link, rxbuf[3:3 + bytesNeeded])
	if err != nil && err != io.ErrUnexpectedEOF {
		return
	}
	if byteCount != bytesNeeded {
		rt.logger.Warningf("expected %v bytes, received %v", bytesNeeded, byteCount)
		err = ErrShortFrame
		return
	}

	// compute the CRC on the entire frame, excluding the CRC
	crc.init()
	crc.add(rxbuf[0:3 + bytesNeeded - 2])

	// compare CRC values
	if !crc.isEqual(rxbuf[3 + bytesNeeded - 2], rxbuf[3 + bytesNeeded - 1]) {
		err = ErrBadCRC
		return
	}

	res	= &pdu{
		unitId:		rxbuf[0],
		functionCode:	rxbuf[1],
		// pass the byte count + trailing data as payload, withtout the CRC
		payload:	rxbuf[2:3 + bytesNeeded  - 2],
	}

	return
}

// Turns a PDU object into bytes.
func (rt *rtuTransport) assembleRTUFrame(p *pdu) (adu []byte) {
	var crc		crc

	adu	= append(adu, p.unitId)
	adu	= append(adu, p.functionCode)
	adu	= append(adu, p.payload...)

	// run the ADU through the CRC generator
	crc.init()
	crc.add(adu)

	// append the CRC to the ADU
	adu	= append(adu, crc.value()...)

	return
}

// Computes the expected length of a modbus RTU response.
func expectedResponseLenth(responseCode uint8, responseLength uint8) (byteCount int, err error) {
	switch responseCode {
	case fcReadHoldingRegisters,
	     fcReadInputRegisters,
	     fcReadCoils,
	     fcReadDiscreteInputs:            byteCount = int(responseLength)
	case fcWriteSingleRegister,
	     fcWriteMultipleRegisters,
	     fcWriteSingleCoil,
	     fcWriteMultipleCoils:            byteCount = 3
	case fcMaskWriteRegister:             byteCount = 5
	case fcReadHoldingRegisters | 0x80,
	     fcReadInputRegisters | 0x80,
	     fcReadCoils | 0x80,
	     fcReadDiscreteInputs | 0x80,
	     fcWriteSingleRegister | 0x80,
	     fcWriteMultipleRegisters | 0x80,
	     fcWriteSingleCoil | 0x80,
	     fcWriteMultipleCoils | 0x80,
	     fcMaskWriteRegister | 0x80:      byteCount = 0
	default: err = ErrProtocolError
	}

	return
}

// Discards the contents of the link's rx buffer, eating up to 1kB of data.
// Note that on a serial line, this call may block for up to serialConf.Timeout
// i.e. 10ms.
func discard(link rtuLink) {
	var rxbuf = make([]byte, 1024)

	link.SetDeadline(time.Now().Add(500 * time.Microsecond))
	io.ReadFull(link, rxbuf)

	return
}

// Returns how long it takes to send 1 byte on a serial line at the
// specified baud rate.
func serialCharTime(rate_bps uint) (ct time.Duration) {
	// note: an RTU byte on the wire is:
	// - 1 start bit,
	// - 8 data bits,
	// - 1 parity or stop bit
	// - 1 stop bit
	ct = (11) * time.Second / time.Duration(rate_bps)

	return
}
