package modbus

import (
	"fmt"
	"time"
	"strings"
	"sync"
)

type RegType	uint
type Endianness uint
const (
	PARITY_NONE		uint	= 0
	PARITY_EVEN		uint	= 1
	PARITY_ODD		uint	= 2

	HOLDING_REGISTER	RegType	= 0
	INPUT_REGISTER		RegType	= 1

	BIG_ENDIAN		Endianness	= 0x00
	LITTLE_ENDIAN		Endianness	= 0x01
)

type ClientConfiguration struct {
	URL		string
	Speed		uint
	DataBits	uint
	Parity		uint
	StopBits	uint
	Timeout		time.Duration
}

type ModbusClient struct {
	conf		ClientConfiguration
	logger		*logger
	lock		sync.Mutex
	endianness	Endianness
	transport	transport
	unitId		uint8
}

func NewClient(conf *ClientConfiguration) (mc *ModbusClient, err error) {
	mc = &ModbusClient{
		conf:	*conf,
	}

	switch {
	case strings.HasPrefix(mc.conf.URL, "rtu://"):
		mc.conf.URL	= strings.TrimPrefix(mc.conf.URL, "rtu://")

		// set useful defaults
		if mc.conf.Speed == 0 {
			mc.conf.Speed	= 9600
		}

		// note: the "modbus over serial line v1.02" document specifies an
		// 11-bit character frame, with even parity and 1 stop bit as default,
		// and mandates the use of 2 stop bits when no parity is used.
		// This stack defaults to 8/N/2 as most devices seem to use no parity,
		// but giving 8/N/1, 8/E/1 and 8/O/1 a shot may help with serial
		// issues.
		if mc.conf.DataBits == 0 {
			mc.conf.DataBits = 8
		}

		if mc.conf.StopBits == 0 {
			if mc.conf.Parity == PARITY_NONE {
				mc.conf.StopBits = 2
			} else {
				mc.conf.StopBits = 1
			}
		}

		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 300 * time.Millisecond
		}

		// create the RTU transport
		mc.transport = newRTUTransport(&mc.conf)

	case strings.HasPrefix(mc.conf.URL, "tcp://"):
		mc.conf.URL	= strings.TrimPrefix(mc.conf.URL, "tcp://")

		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		// create the TCP transport
		mc.transport = newTCPTransport(&mc.conf)

	default:
		err	= ErrConfigurationError
		return
	}

	mc.unitId	= 1
	mc.logger	= newLogger(fmt.Sprintf("modbus-client(%s)", mc.conf.URL))

	return
}

// Opens the underlying transport (tcp socket or serial line).
func (mc *ModbusClient) Open() (err error) {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	err = mc.transport.Open()

	return
}

// Closes the underlying transport.
func (mc *ModbusClient) Close() (err error) {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	err = mc.transport.Close()

	return
}

// Sets the unit id of subsequent requests.
func (mc *ModbusClient) SetUnitId(id uint8) (err error) {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	mc.unitId	= id

	return
}

// Sets the endianness of subsequent requests.
func (mc *ModbusClient) SetEndianness(endianness Endianness) (err error) {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	mc.endianness	= endianness

	return
}

// Reads multiple 16-bit registers.
func (mc *ModbusClient) ReadRegisters(addr uint16, quantity uint16, regType RegType) (values []uint16, err error) {
	var mbPayload	[]byte

	// read 1 uint16 register, as bytes
	mbPayload, err	= mc.readRegisters(addr, quantity, regType)
	if err != nil {
		return
	}

	// decode payload bytes as uint16s
	values	= bytesToUint16s(mc.endianness, mbPayload)

	return
}

// Reads a single 16-bit register.
func (mc *ModbusClient) ReadRegister(addr uint16, regType RegType) (value uint16, err error) {
	var values	[]uint16

	values, err	= mc.ReadRegisters(addr, 1, regType)
	if err == nil {
		value = values[0]
	}

	return
}

// Writes a single 16-bit register (function code 0x06).
func (mc *ModbusClient) WriteRegister(addr uint16, value uint16) (err error) {
	var req		*request
	var res		*response

	mc.lock.Lock()
	defer mc.lock.Unlock()

	// create and fill in the request object
	req	= &request{
		unitId:		mc.unitId,
		functionCode:	FC_WRITE_SINGLE_REGISTER,
	}

	// register address
	req.payload	= uint16ToBytes(BIG_ENDIAN, addr)
	// register value
	req.payload	= append(req.payload, uint16ToBytes(mc.endianness, value)...)

	// run the request across the transport and wait for a response
	res, err	= mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.responseCode == req.functionCode:
		// expect 4 bytes (2 byte of address + 2 bytes of value)
		if len(res.payload) != 4 ||
		   // bytes 1-2 should be the register address
		   bytesToUint16(BIG_ENDIAN, res.payload[0:2]) != addr ||
		   // bytes 3-4 should be the value
		   bytesToUint16(mc.endianness, res.payload[2:4]) != value {
			   err = ErrProtocolError
			   return
		   }

	case res.responseCode == (req.functionCode | 0x80):
		err	= mapExceptionCodeToError(res.payload[0])

	default:
		err	= ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.responseCode)
	}

	return
}

// Writes multiple 16-bit registers (function code 0x10).
func (mc *ModbusClient) WriteRegisters(addr uint16, values []uint16) (err error) {
	var payload	[]byte

	// turn registers to bytes
	for _, value := range values {
		payload	= append(payload, uint16ToBytes(mc.endianness, value)...)
	}

	err = mc.writeRegisters(addr, payload)

	return
}

/*** unexported methods ***/
// Reads and returns quantity registers of type regType, as bytes.
func (mc *ModbusClient) readRegisters(addr uint16, quantity uint16, regType RegType) (bytes []byte, err error) {
	var req		*request
	var res		*response

	mc.lock.Lock()
	defer mc.lock.Unlock()

	// create and fill in the request object
	req	= &request{
		unitId:	mc.unitId,
	}

	switch regType {
	case HOLDING_REGISTER:	req.functionCode = FC_READ_HOLDING_REGISTERS
	case INPUT_REGISTER:	req.functionCode = FC_READ_INPUT_REGISTERS
	default:
		err = ErrUnexpectedParameters
		mc.logger.Errorf("unexpected register type (%v)", regType)
		return
	}

	if quantity == 0 {
		err = ErrUnexpectedParameters
		mc.logger.Error("quantity of registers is 0")
	}

	if quantity > 123 {
		err = ErrUnexpectedParameters
		mc.logger.Error("quantity of registers exceeds 123")
	}

	if uint32(addr) + uint32(quantity) - 1 > 0xffff {
		err = ErrUnexpectedParameters
		mc.logger.Error("end register address is past 0xffff")
		return
	}

	// start address
	req.payload	= uint16ToBytes(BIG_ENDIAN, addr)
	// quantity
	req.payload	= append(req.payload, uint16ToBytes(BIG_ENDIAN, quantity)...)

	// run the request across the transport and wait for a response
	res, err	= mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.responseCode == req.functionCode:
		// make sure the payload length is what we expect
		// (1 byte of length + 2 bytes per register)
		if len(res.payload) != 1 + 2 * int(quantity) {
			err = ErrProtocolError
			return
		}

		// validate the byte count field
		// (2 bytes per register * number of registers)
		if uint(res.payload[0]) != 2 * uint(quantity) {
			err = ErrProtocolError
			return
		}

		// remove the byte count field from the returned slice
		bytes	= res.payload[1:]

	case res.responseCode == (req.functionCode | 0x80):
		err	= mapExceptionCodeToError(res.payload[0])

	default:
		err	= ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.responseCode)
	}

	return
}

// Writes multiple registers starting from base address addr.
// Register values are passed as bytes, each value being exactly 2 bytes.
func (mc *ModbusClient) writeRegisters(addr uint16, values []byte) (err error) {
	var req			*request
	var res			*response
	var payloadLength	uint16
	var quantity		uint16

	mc.lock.Lock()
	defer mc.lock.Unlock()

	payloadLength	= uint16(len(values))
	quantity	= payloadLength / 2

	if quantity == 0 {
		err = ErrUnexpectedParameters
		mc.logger.Errorf("quantity of registers is 0")
		return
	}

	if quantity > 123 {
		err = ErrUnexpectedParameters
		mc.logger.Errorf("quantity of registers exceeds 123")
		return
	}

	if uint32(addr) + uint32(quantity) - 1 > 0xffff {
		err = ErrUnexpectedParameters
		mc.logger.Errorf("end register address is past 0xffff")
		return
	}

	// create and fill in the request object
	req	= &request{
		unitId:		mc.unitId,
		functionCode:	FC_WRITE_MULTIPLE_REGISTERS,
	}

	// base address
	req.payload	= uint16ToBytes(BIG_ENDIAN, addr)
	// quantity of registers (2 bytes per register)
	req.payload	= append(req.payload, uint16ToBytes(BIG_ENDIAN, quantity)...)
	// byte count
	req.payload	= append(req.payload, byte(payloadLength))
	// registers value
	req.payload	= append(req.payload, values...)

	// run the request across the transport and wait for a response
	res, err	= mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.responseCode == req.functionCode:
		// expect 4 bytes (2 byte of address + 2 bytes of quantity)
		if len(res.payload) != 4 ||
		   // bytes 1-2 should be the base register address
		   bytesToUint16(BIG_ENDIAN, res.payload[0:2]) != addr ||
		   // bytes 3-4 should be the quantity of registers (2 bytes per register)
		   bytesToUint16(BIG_ENDIAN, res.payload[2:4]) != quantity {
			   err = ErrProtocolError
			   return
		   }

	case res.responseCode == (req.functionCode | 0x80):
		err	= mapExceptionCodeToError(res.payload[0])

	default:
		err	= ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.responseCode)
	}

	return
}

func (mc *ModbusClient) executeRequest(req *request) (res *response, err error) {
	// send the request out
	err		= mc.transport.WriteRequest(req)

	// wait for, read and decode the response
	res, err	= mc.transport.ReadResponse()
	if err != nil {
		return
	}

	// make sure the source unit id matches that of the request
	if (res.responseCode & 0x80) == 0x00 && res.unitId != req.unitId {
		err = ErrBadUnitId
		return
	}
	// accept errors from gateway devices (using special unit id #255)
	if (res.responseCode & 0x80) == 0x80 &&
		(res.unitId != req.unitId && res.unitId != 0xff) {
		err = ErrBadUnitId
		return
	}

	return
}
