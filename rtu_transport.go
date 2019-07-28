package modbus

import (
	"fmt"
	"time"
)

const (
	maxRTUFrameLength	int = 256
)

type rtuTransport struct {
	logger		*logger
	link		rtuLink
	timeout		time.Duration
	speed		uint
}

type rtuLink interface {
	Close()		(error)
	Read([]byte)	(int, error)
	Write([]byte)	(int, error)
	SetDeadline(time.Time)	(error)
}

// Returns a new RTU transport.
func newRTUTransport(link rtuLink, addr string, speed uint, timeout time.Duration) (rt *rtuTransport) {
	rt = &rtuTransport{
		logger:		newLogger(fmt.Sprintf("rtu-transport(%s)", addr)),
		link:		link,
		timeout:	timeout,
		speed:		speed,
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
	// set an i/o deadline on the link
	err	= rt.link.SetDeadline(time.Now().Add(rt.timeout))
	if err != nil {
		return
	}

	// build an RTU ADU out of the request object and
	// send the final ADU+CRC on the wire
	_, err	= rt.link.Write(rt.assembleRTUFrame(req))
	if err != nil {
		return
	}

	// observe inter-frame delays
	time.Sleep(rt.interFrameDelay())

	// read the response back from the wire
	res, err = rt.readRTUFrame()

	return
}

// Reads a request from the rtu link.
func (rt *rtuTransport) ReadRequest() (req *pdu, err error) {
	// set an i/o deadline on the link
	err	= rt.link.SetDeadline(time.Now().Add(rt.timeout))
	if err != nil {
		return
	}

	// read the request from the wire
	req, err = rt.readRTUFrame()

	return
}

// Writes a response to the rtu link.
func (rt *rtuTransport) WriteResponse(res *pdu) (err error) {
	// build an RTU ADU out of the request object and
	// send the final ADU+CRC on the wire
	_, err	= rt.link.Write(rt.assembleRTUFrame(res))
	if err != nil {
		return
	}

	// observe inter-frame delays
	time.Sleep(rt.interFrameDelay())

	return
}

// Returns the inter-frame gap duration.
func (rt *rtuTransport) interFrameDelay() (delay time.Duration) {
	if rt.speed == 0 || rt.speed >= 19200 {
		// for baud rates equal to or greater than 19200 bauds, a fixed
		// inter-frame delay of 1750 uS is specified.
		delay = 1750 * time.Microsecond
	} else {
		// for lower baud rates, the inter-frame delay should be 3.5 character times
		delay = time.Duration(38500000 / rt.speed) * time.Microsecond
	}

	return
}

// Waits for, reads and decodes a response from the rtu link.
func (rt *rtuTransport) readRTUFrame() (res *pdu, err error) {
	var rxbuf	[]byte
	var byteCount	int
	var n		int
	var crc		crc
	var start	time.Time

	rxbuf		= make([]byte, maxRTUFrameLength)

	// mark the time
	start		= time.Now()

	// attempt to read for as long as we're allowed to
	for time.Since(start) < rt.timeout {

		// grab as many bytes as we can in ~1.5 character times
		n, err	= rt.link.Read(rxbuf[byteCount:])
		if err != nil {
			return
		} else {
			byteCount += n
		}

		// stop reading if either:
		// - we've read the max number of bytes an RTU frame is allowed to have,
		// - we detected the end of the frame (1.5 character times has elapsed
		//   without any data being transmitted)
		if byteCount > 0 && n == 0 || byteCount == maxRTUFrameLength {
			break
		}
	}

	if byteCount == 0 {
		err = ErrRequestTimedOut
		return
	}

	// we need at the very least a unit ID + function code + 2 bytes of CRC
	if byteCount <= 4 {
		err = ErrShortFrame
		return
	}

	// compute the CRC on the entire frame, excluding the CRC
	crc.init()
	crc.add(rxbuf[0:byteCount - 2])

	// compare CRC values
	if !crc.isEqual(rxbuf[byteCount - 2], rxbuf[byteCount - 1]) {
		err = ErrBadCRC
		return
	}

	res	= &pdu{
		unitId:		rxbuf[0],
		functionCode:	rxbuf[1],
		// pass the byte count + trailing data as payload, withtout the CRC
		payload:	rxbuf[2:byteCount - 2],
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
	default: err = fmt.Errorf("unexpected response code (%v)", responseCode)
	}

	return
}

// Discards the contents of the link's rx buffer, eating up to 1kB of data.
// Note that on a serial line, this call may block for up to serialConf.Timeout
// i.e. 10ms.
func discard(link rtuLink) {
	var rxbuf	= make([]byte, 1024)

	link.SetDeadline(time.Now().Add(time.Millisecond))
	link.Read(rxbuf)
	return
}
