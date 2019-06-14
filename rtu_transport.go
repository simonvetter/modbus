package modbus

import (
	"fmt"
	"io"
	"time"

	"github.com/goburrow/serial"
)

const (
	maxRTUFrameLength	int = 256
)

type rtuTransport struct {
	port		io.ReadWriteCloser
	conf		*ClientConfiguration
	logger		*logger
}

// Returns a new RTU transport.
func newRTUTransport(conf *ClientConfiguration) (rt *rtuTransport) {
	rt = &rtuTransport{
		conf:	conf,
		logger:	newLogger(fmt.Sprintf("rtu-transport(%s)", conf.URL)),
	}

	return
}

// Opens the serial line.
func (rt *rtuTransport) Open() (err error) {
	var serialConf	*serial.Config

	serialConf	= &serial.Config{
		Address:	rt.conf.URL,
		BaudRate:	int(rt.conf.Speed),
		DataBits:	int(rt.conf.DataBits),
		StopBits:	int(rt.conf.StopBits),
		Timeout:	rt.conf.Timeout,
	}

	switch rt.conf.Parity {
	case PARITY_NONE:	serialConf.Parity	= "N"
	case PARITY_EVEN:	serialConf.Parity	= "E"
	case PARITY_ODD:	serialConf.Parity	= "O"
	}

	rt.port, err	= serial.Open(serialConf)
	if err != nil {
		rt.logger.Errorf("failed to open serial device: %v", err)
		return
	}

	return
}

// Closes the serial line.
func (rt *rtuTransport) Close() (err error) {
	err = rt.port.Close()
	if err != nil {
		rt.logger.Errorf("failed to close serial device: %v", err)
		return
	}

	return
}

// Serializes and writes a request out to the serial line.
func (rt *rtuTransport) WriteRequest(req *request) (err error) {
	var byteCount	int
	var adu		[]byte

	// build an RTU ADU out of the request object
	adu		= rt.assembleRTUFrame(req)

	// send the final ADU+CRC on the wire
	byteCount, err	= rt.port.Write(adu)
	if byteCount != len(adu) {
		return
	}

	// enforce inter-frame delays
	if rt.conf.Speed >= 19200 {
		// for baud rates equal to or greater than 19200 bauds, a fixed
		// inter-frame delay of 1750 uS is specified.
		time.Sleep(1750 * time.Microsecond)
	} else {
		// for lower baud rates, the inter-frame delay should be 3.5 character times
		time.Sleep(time.Duration(38500000 / rt.conf.Speed) * time.Microsecond)
	}

	return
}

// Waits for, reads and decodes a response from the serial line.
func (rt *rtuTransport) ReadResponse() (res *response, err error) {
	var rxbuf	[]byte
	var byteCount	int
	var bytesNeeded	int
	var crc		crc

	rxbuf		= make([]byte, maxRTUFrameLength)

	// read the serial ADU header: unit id (1 byte), response code (1 byte) and
	// PDU length/exception code (1 byte)
	byteCount, err	= io.ReadFull(rt.port, rxbuf[0:3])
	if err != nil && err != io.ErrUnexpectedEOF {
		return
	}
	if byteCount != 3 {
		err = ErrShortResponse
		return
	}

	// figure out how many further bytes to read
	bytesNeeded, err	= expectedResponseLenth(uint8(rxbuf[1]), uint8(rxbuf[2]))
	if err != nil {
		return
	}

	// we need to read 2 additional bytes of CRC after the payload
	bytesNeeded		+= 2

	// never read more than the max allowed frame length
	if byteCount + bytesNeeded > maxRTUFrameLength {
		err	= ErrProtocolError
		return
	}

	byteCount, err		= io.ReadFull(rt.port, rxbuf[3:3 + bytesNeeded])
	if err != nil && err != io.ErrUnexpectedEOF {
		return
	}
	if byteCount != bytesNeeded {
		rt.logger.Warningf("expected %v bytes, received %v", bytesNeeded, byteCount)
		err = ErrShortResponse
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

	res	= &response{
		unitId:		rxbuf[0],
		responseCode:	rxbuf[1],
		// pass the byte count + trailing data as payload, withtout the CRC
		payload:	rxbuf[2:3 + bytesNeeded  - 2],
	}

	return
}

func (rt *rtuTransport) assembleRTUFrame(req *request) (adu []byte) {
	var crc		crc

	adu	= append(adu, req.unitId)
	adu	= append(adu, req.functionCode)
	adu	= append(adu, req.payload...)

	// run the ADU through the CRC generator
	crc.init()
	crc.add(adu)

	// append the CRC to the ADU
	adu	= append(adu, crc.value()...)

	return
}

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
