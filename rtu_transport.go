package modbus

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/goburrow/serial"
)

const (
	maxRTUFrameLength	int = 256
)

type rtuTransport struct {
	conf		*ClientConfiguration
	logger		*logger
	isOverTCP	bool
	link		rtuLink
}

type rtuLink interface {
	Close()		(error)
	Read([]byte)	(int, error)
	Write([]byte)	(int, error)
	SetDeadline(time.Time)	(error)
}

// Returns a new RTU transport.
func newRTUTransport(conf *ClientConfiguration, isOverTCP bool) (rt *rtuTransport) {
	rt = &rtuTransport{
		conf:		conf,
		logger:		newLogger(fmt.Sprintf("rtu-transport(%s)", conf.URL)),
		isOverTCP:	isOverTCP,
	}

	return
}

// Opens the rtu link.
func (rt *rtuTransport) Open() (err error) {
	var parity	string
	var port	serial.Port

	if rt.isOverTCP {
		rt.link, err	= net.DialTimeout("tcp", rt.conf.URL, 5 * time.Second)
		if err != nil {
			return
		}
	} else {
		switch rt.conf.Parity {
		case PARITY_NONE:	parity	= "N"
		case PARITY_EVEN:	parity	= "E"
		case PARITY_ODD:	parity	= "O"
		}

		port, err	= serial.Open(&serial.Config{
			Address:	rt.conf.URL,
			BaudRate:	int(rt.conf.Speed),
			DataBits:	int(rt.conf.DataBits),
			Parity:		parity,
			StopBits:	int(rt.conf.StopBits),
			Timeout:	10 * time.Millisecond,
		})

		if err != nil {
			return
		}

		rt.link	= &serialPortWrapper{
			port: port,
		}
	}

	// flush the receive buffer to avoid consuming stale data
	discard(rt.link)

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
	err	= rt.link.SetDeadline(time.Now().Add(rt.conf.Timeout))
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
	// reading requests from RTU links is currently unsupported
	err	= fmt.Errorf("unimplemented")

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
	if rt.conf.Speed == 0 || rt.conf.Speed >= 19200 {
		// for baud rates equal to or greater than 19200 bauds, a fixed
		// inter-frame delay of 1750 uS is specified.
		delay = 1750 * time.Microsecond
	} else {
		// for lower baud rates, the inter-frame delay should be 3.5 character times
		delay = time.Duration(38500000 / rt.conf.Speed) * time.Microsecond
	}

	return
}

// Waits for, reads and decodes a frame from the rtu link.
func (rt *rtuTransport) readRTUFrame() (res *pdu, err error) {
	var rxbuf	[]byte
	var byteCount	int
	var bytesNeeded	int
	var crc		crc

	rxbuf		= make([]byte, maxRTUFrameLength)

	// read the serial ADU header: unit id (1 byte), response code (1 byte) and
	// PDU length/exception code (1 byte)
	byteCount, err	= io.ReadFull(rt.link, rxbuf[0:3])
	if err != nil && err != io.ErrUnexpectedEOF {
		return
	}
	if byteCount != 3 {
		err = ErrShortFrame
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
	case FC_READ_HOLDING_REGISTERS,
	     FC_READ_INPUT_REGISTERS,
	     FC_READ_COILS,
	     FC_READ_DISCRETE_INPUTS:		byteCount = int(responseLength)
	case FC_WRITE_SINGLE_REGISTER,
	     FC_WRITE_MULTIPLE_REGISTERS,
	     FC_WRITE_SINGLE_COIL,
	     FC_WRITE_MULTIPLE_COILS:		byteCount = 3
	case FC_MASK_WRITE_REGISTER:		byteCount = 5
	case FC_READ_HOLDING_REGISTERS | 0x80,
	     FC_READ_INPUT_REGISTERS | 0x80,
	     FC_READ_COILS | 0x80,
	     FC_READ_DISCRETE_INPUTS | 0x80,
	     FC_WRITE_SINGLE_REGISTER | 0x80,
	     FC_WRITE_MULTIPLE_REGISTERS | 0x80,
	     FC_WRITE_SINGLE_COIL | 0x80,
	     FC_WRITE_MULTIPLE_COILS | 0x80,
	     FC_MASK_WRITE_REGISTER | 0x80:	byteCount = 0
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

// serialPortWrapper wraps a seria.Port (i.e. physical port) to
// 1) satisfy the rtuLink interface and
// 2) add Read() deadline/timeout support.
type serialPortWrapper struct {
	port		serial.Port
	deadline	time.Time
}

// Closes the serial port.
func (spw *serialPortWrapper) Close() (err error) {
	err = spw.port.Close()

	return
}

// Reads bytes from the underlying serial port.
// If Read() is called after the deadline, a timeout error is returned without
// attempting to read from the serial port.
// If Read() is called before the deadline, a read attempt to the serial port
// is made. At this point, one of two things can happen:
// - the serial port's receive buffer has one or more bytes and port.Read()
//   returns immediately (partial or full read),
// - the serial port's receive buffer is empty: port.Read() blocks for
//   up to 10ms and returns serial.ErrTimeout. The serial timeout error is
//   masked and Read() returns with no data.
// As the higher-level methods use io.ReadFull(), Read() will be called
// as many times as necessary until either enough bytes have been read or an
// error is returned (serial.ErrTimeout or any other i/o error).
func (spw *serialPortWrapper) Read(rxbuf []byte) (cnt int, err error) {
	// return a timeout if the deadline has passed
	if time.Now().After(spw.deadline) {
		err = serial.ErrTimeout
		return
	}

	cnt, err = spw.port.Read(rxbuf)
	// mask serial.ErrTimeout errors from the serial port
	if err != nil && err == serial.ErrTimeout {
		err = nil
	}

	return
}

// Sends the bytes over the wire.
func (spw *serialPortWrapper) Write(txbuf []byte) (cnt int, err error) {
	cnt, err = spw.port.Write(txbuf)

	return
}

// Saves the i/o deadline (only used by Read).
func (spw *serialPortWrapper) SetDeadline(deadline time.Time) (err error) {
	spw.deadline = deadline

	return
}
