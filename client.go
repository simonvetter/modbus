package modbus

import (
	"fmt"
	"net"
	"time"
	"strings"
	"sync"
)

type RegType	uint
type Endianness uint
type WordOrder	uint
const (
	PARITY_NONE		uint		= 0
	PARITY_EVEN		uint		= 1
	PARITY_ODD		uint		= 2

	HOLDING_REGISTER	RegType		= 0
	INPUT_REGISTER		RegType		= 1

	// endianness of 16-bit registers
	BIG_ENDIAN		Endianness	= 1
	LITTLE_ENDIAN		Endianness	= 2

	// word order of 32-bit registers
	HIGH_WORD_FIRST		WordOrder	= 1
	LOW_WORD_FIRST		WordOrder	= 2
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
	wordOrder	WordOrder
	transport	transport
	unitId		uint8
	transportType	transportType
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

		mc.transportType	= RTU_TRANSPORT

	case strings.HasPrefix(mc.conf.URL, "rtuovertcp://"):
		mc.conf.URL	= strings.TrimPrefix(mc.conf.URL, "rtuovertcp://")

		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		mc.transportType	= RTU_OVER_TCP_TRANSPORT

	case strings.HasPrefix(mc.conf.URL, "tcp://"):
		mc.conf.URL	= strings.TrimPrefix(mc.conf.URL, "tcp://")

		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		mc.transportType	= TCP_TRANSPORT

	default:
		err	= ErrConfigurationError
		return
	}

	mc.unitId	= 1
	mc.endianness	= BIG_ENDIAN
	mc.wordOrder	= HIGH_WORD_FIRST
	mc.logger	= newLogger(fmt.Sprintf("modbus-client(%s)", mc.conf.URL))

	return
}

// Opens the underlying transport (tcp socket or serial line).
func (mc *ModbusClient) Open() (err error) {
	var spw		*serialPortWrapper
	var sock	net.Conn

	mc.lock.Lock()
	defer mc.lock.Unlock()

	switch mc.transportType {
	case RTU_TRANSPORT:
		// create a serial port wrapper object
		spw = newSerialPortWrapper(&serialPortConfig{
			Device:		mc.conf.URL,
			Speed:		mc.conf.Speed,
			DataBits:	mc.conf.DataBits,
			Parity:		mc.conf.Parity,
			StopBits:	mc.conf.StopBits,
		})

		// open the serial device
		err = spw.Open()
		if err != nil {
			return
		}

		// discard potentially stale serial data
		discard(spw)

		// create the RTU transport
		mc.transport = newRTUTransport(
			spw, mc.conf.URL, mc.conf.Speed, mc.conf.Timeout)

	case RTU_OVER_TCP_TRANSPORT:
		// connect to the remote host
		sock, err	= net.DialTimeout("tcp", mc.conf.URL, 5 * time.Second)
		if err != nil {
			return
		}

		// discard potentially stale serial data
		discard(sock)

		// create the RTU transport
		mc.transport = newRTUTransport(
			sock, mc.conf.URL, mc.conf.Speed, mc.conf.Timeout)

	case TCP_TRANSPORT:
		// connect to the remote host
		sock, err	= net.DialTimeout("tcp", mc.conf.URL, 5 * time.Second)
		if err != nil {
			return
		}

		// create the TCP transport
		mc.transport = newTCPTransport(sock, mc.conf.Timeout)

	default:
		// should never happen
		err = ErrConfigurationError
	}

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

// Sets the encoding (endianness and word ordering) of subsequent requests.
func (mc *ModbusClient) SetEncoding(endianness Endianness, wordOrder WordOrder) (err error) {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	if endianness != BIG_ENDIAN && endianness != LITTLE_ENDIAN {
		mc.logger.Errorf("unknown endianness value %v", endianness)
		err	= ErrUnexpectedParameters
		return
	}

	if wordOrder != HIGH_WORD_FIRST && wordOrder != LOW_WORD_FIRST {
		mc.logger.Errorf("unknown word order value %v", wordOrder)
		err	= ErrUnexpectedParameters
		return
	}

	mc.endianness	= endianness
	mc.wordOrder	= wordOrder

	return
}

// Reads multiple coils (function code 01).
func (mc *ModbusClient) ReadCoils(addr uint16, quantity uint16) (values []bool, err error) {
	values, err	= mc.readBools(addr, quantity, false)

	return
}

// Reads a single coil (function code 01).
func (mc *ModbusClient) ReadCoil(addr uint16) (value bool, err error) {
	var values	[]bool

	values, err	= mc.readBools(addr, 1, false)
	if err == nil {
		value = values[0]
	}

	return
}

// Reads multiple discrete inputs (function code 02).
func (mc *ModbusClient) ReadDiscreteInputs(addr uint16, quantity uint16) (values []bool, err error) {
	values, err	= mc.readBools(addr, quantity, true)

	return
}

// Reads a single discrete input (function code 02).
func (mc *ModbusClient) ReadDiscreteInput(addr uint16) (value bool, err error) {
	var values	[]bool

	values, err	= mc.readBools(addr, 1, true)
	if err == nil {
		value = values[0]
	}

	return
}

// Reads multiple 16-bit registers (function code 03 or 04).
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

// Reads a single 16-bit register (function code 03 or 04).
func (mc *ModbusClient) ReadRegister(addr uint16, regType RegType) (value uint16, err error) {
	var values	[]uint16

	values, err	= mc.ReadRegisters(addr, 1, regType)
	if err == nil {
		value = values[0]
	}

	return
}

// Reads multiple 32-bit registers.
func (mc *ModbusClient) ReadUint32s(addr uint16, quantity uint16, regType RegType) (values []uint32, err error) {
	var mbPayload	[]byte

	// read 2 * quantity uint16 registers, as bytes
	mbPayload, err	= mc.readRegisters(addr, quantity * 2, regType)
	if err != nil {
		return
	}

	// decode payload bytes as uint32s
	values	= bytesToUint32s(mc.endianness, mc.wordOrder, mbPayload)

	return
}

// Reads a single 32-bit register.
func (mc *ModbusClient) ReadUint32(addr uint16, regType RegType) (value uint32, err error) {
	var values	[]uint32

	values, err	= mc.ReadUint32s(addr, 1, regType)
	if err == nil {
		value	= values[0]
	}

	return
}

// Reads multiple 32-bit float registers.
func (mc *ModbusClient) ReadFloat32s(addr uint16, quantity uint16, regType RegType) (values []float32, err error) {
	var mbPayload	[]byte

	// read 2 * quantity uint16 registers, as bytes
	mbPayload, err	= mc.readRegisters(addr, quantity * 2, regType)
	if err != nil {
		return
	}

	// decode payload bytes as float32s
	values	= bytesToFloat32s(mc.endianness, mc.wordOrder, mbPayload)

	return
}

// Reads a single 32-bit float register.
func (mc *ModbusClient) ReadFloat32(addr uint16, regType RegType) (value float32, err error) {
	var values	[]float32

	values, err	= mc.ReadFloat32s(addr, 1, regType)
	if err == nil {
		value	= values[0]
	}

	return
}

// Writes a single coil (function code 05)
func (mc *ModbusClient) WriteCoil(addr uint16, value bool) (err error) {
	var req		*pdu
	var res		*pdu

	mc.lock.Lock()
	defer mc.lock.Unlock()

	// create and fill in the request object
	req	= &pdu{
		unitId:		mc.unitId,
		functionCode:	FC_WRITE_SINGLE_COIL,
	}

	// coil address
	req.payload	= uint16ToBytes(BIG_ENDIAN, addr)
	// coil value
	if value {
		req.payload	= append(req.payload, 0xff, 0x00)
	} else {
		req.payload	= append(req.payload, 0x00, 0x00)
	}

	// run the request across the transport and wait for a response
	res, err	= mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.functionCode == req.functionCode:
		// expect 4 bytes (2 byte of address + 2 bytes of value)
		if len(res.payload) != 4 ||
		   // bytes 1-2 should be the coil address
		   bytesToUint16(BIG_ENDIAN, res.payload[0:2]) != addr ||
		   // bytes 3-4 should either be {0xff, 0x00} or {0x00, 0x00}
		   // depending on the coil value
		   (value == true && res.payload[2] != 0xff) ||
		   res.payload[3] != 0x00 {
			   err = ErrProtocolError
			   return
		   }

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err	= ErrProtocolError
			return
		}

		err	= mapExceptionCodeToError(res.payload[0])

	default:
		err	= ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

// Writes multiple coils (function code 15)
func (mc *ModbusClient) WriteCoils(addr uint16, values []bool) (err error) {
	var req			*pdu
	var res			*pdu
	var quantity		uint16
	var encodedValues	[]byte

	mc.lock.Lock()
	defer mc.lock.Unlock()

	quantity	= uint16(len(values))
	if quantity == 0 {
		err	= ErrUnexpectedParameters
		mc.logger.Error("quantity of coils is 0")
		return
	}

	if quantity > 0x7b0 {
		err	= ErrUnexpectedParameters
		mc.logger.Error("quantity of coils exceeds 1968")
		return
	}

	if uint32(addr) + uint32(quantity) - 1 > 0xffff {
		err	= ErrUnexpectedParameters
		mc.logger.Error("end coil address is past 0xffff")
		return
	}

	encodedValues	= encodeBools(values)

	// create and fill in the request object
	req	= &pdu{
		unitId:		mc.unitId,
		functionCode:	FC_WRITE_MULTIPLE_COILS,
	}

	// start address
	req.payload	= uint16ToBytes(BIG_ENDIAN, addr)
	// quantity
	req.payload	= append(req.payload, uint16ToBytes(BIG_ENDIAN, quantity)...)
	// byte count
	req.payload	= append(req.payload, byte(len(encodedValues)))
	// payload
	req.payload	= append(req.payload, encodedValues...)

	// run the request across the transport and wait for a response
	res, err	= mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.functionCode == req.functionCode:
		// expect 4 bytes (2 byte of address + 2 bytes of quantity)
		if len(res.payload) != 4 ||
		   // bytes 1-2 should be the base coil address
		   bytesToUint16(BIG_ENDIAN, res.payload[0:2]) != addr ||
		   // bytes 3-4 should be the quantity of coils
		   bytesToUint16(BIG_ENDIAN, res.payload[2:4]) != quantity {
			   err = ErrProtocolError
			   return
		   }

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err	= ErrProtocolError
			return
		}

		err	= mapExceptionCodeToError(res.payload[0])

	default:
		err	= ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

// Writes a single 16-bit register (function code 06).
func (mc *ModbusClient) WriteRegister(addr uint16, value uint16) (err error) {
	var req		*pdu
	var res		*pdu

	mc.lock.Lock()
	defer mc.lock.Unlock()

	// create and fill in the request object
	req	= &pdu{
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
	case res.functionCode == req.functionCode:
		// expect 4 bytes (2 byte of address + 2 bytes of value)
		if len(res.payload) != 4 ||
		   // bytes 1-2 should be the register address
		   bytesToUint16(BIG_ENDIAN, res.payload[0:2]) != addr ||
		   // bytes 3-4 should be the value
		   bytesToUint16(mc.endianness, res.payload[2:4]) != value {
			   err = ErrProtocolError
			   return
		   }

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err	= ErrProtocolError
			return
		}

		err	= mapExceptionCodeToError(res.payload[0])

	default:
		err	= ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

// Writes multiple 16-bit registers (function code 16).
func (mc *ModbusClient) WriteRegisters(addr uint16, values []uint16) (err error) {
	var payload	[]byte

	// turn registers to bytes
	for _, value := range values {
		payload	= append(payload, uint16ToBytes(mc.endianness, value)...)
	}

	err = mc.writeRegisters(addr, payload)

	return
}

// Writes multiple 32-bit registers.
func (mc *ModbusClient) WriteUint32s(addr uint16, values []uint32) (err error) {
	var payload	[]byte

	// turn registers to bytes
	for _, value := range values {
		payload	= append(payload, uint32ToBytes(mc.endianness, mc.wordOrder, value)...)
	}

	err = mc.writeRegisters(addr, payload)

	return
}

// Writes a single 32-bit register.
func (mc *ModbusClient) WriteUint32(addr uint16, value uint32) (err error) {
	err = mc.writeRegisters(addr, uint32ToBytes(mc.endianness, mc.wordOrder, value))

	return
}

// Writes multiple 32-bit float registers.
func (mc *ModbusClient) WriteFloat32s(addr uint16, values []float32) (err error) {
	var payload	[]byte

	// turn registers to bytes
	for _, value := range values {
		payload	= append(payload, float32ToBytes(mc.endianness, mc.wordOrder, value)...)
	}

	err = mc.writeRegisters(addr, payload)

	return
}

// Writes a single 32-bit float register.
func (mc *ModbusClient) WriteFloat32(addr uint16, value float32) (err error) {
	err = mc.writeRegisters(addr, float32ToBytes(mc.endianness, mc.wordOrder, value))

	return
}

/*** unexported methods ***/
// Reads and returns quantity booleans.
// Digital inputs are read if di is true, otherwise coils are read.
func (mc *ModbusClient) readBools(addr uint16, quantity uint16, di bool) (values []bool, err error) {
	var req		*pdu
	var res		*pdu
	var expectedLen	int

	mc.lock.Lock()
	defer mc.lock.Unlock()

	if quantity == 0 {
		err	= ErrUnexpectedParameters
		mc.logger.Error("quantity of coils/discrete inputs is 0")
		return
	}

	if quantity > 2000 {
		err	= ErrUnexpectedParameters
		mc.logger.Error("quantity of coils/discrete inputs exceeds 2000")
		return
	}

	if uint32(addr) + uint32(quantity) - 1 > 0xffff {
		err	= ErrUnexpectedParameters
		mc.logger.Error("end coil/discrete input address is past 0xffff")
		return
	}

	// create and fill in the request object
	req	= &pdu{
		unitId:	mc.unitId,
	}

	if di {
		req.functionCode	= FC_READ_DISCRETE_INPUTS
	} else {
		req.functionCode	= FC_READ_COILS
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
	case res.functionCode == req.functionCode:
		// expect a payload of 1 byte (byte count) + 1 byte for 8 coils/discrete inputs)
		expectedLen	= 1
		expectedLen	+= int(quantity) / 8
		if quantity % 8 != 0 {
			expectedLen++
		}

		if len(res.payload) != expectedLen {
			err = ErrProtocolError
			return
		}

		// validate the byte count field
		if int(res.payload[0]) + 1 != expectedLen {
			err = ErrProtocolError
			return
		}

		// turn bits into a bool slice
		values = decodeBools(quantity, res.payload[1:])


	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err	= ErrProtocolError
			return
		}

		err	= mapExceptionCodeToError(res.payload[0])

	default:
		err	= ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}


// Reads and returns quantity registers of type regType, as bytes.
func (mc *ModbusClient) readRegisters(addr uint16, quantity uint16, regType RegType) (bytes []byte, err error) {
	var req		*pdu
	var res		*pdu

	mc.lock.Lock()
	defer mc.lock.Unlock()

	// create and fill in the request object
	req	= &pdu{
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
	case res.functionCode == req.functionCode:
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

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err	= ErrProtocolError
			return
		}

		err	= mapExceptionCodeToError(res.payload[0])

	default:
		err	= ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

// Writes multiple registers starting from base address addr.
// Register values are passed as bytes, each value being exactly 2 bytes.
func (mc *ModbusClient) writeRegisters(addr uint16, values []byte) (err error) {
	var req			*pdu
	var res			*pdu
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
	req	= &pdu{
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
	case res.functionCode == req.functionCode:
		// expect 4 bytes (2 byte of address + 2 bytes of quantity)
		if len(res.payload) != 4 ||
		   // bytes 1-2 should be the base register address
		   bytesToUint16(BIG_ENDIAN, res.payload[0:2]) != addr ||
		   // bytes 3-4 should be the quantity of registers (2 bytes per register)
		   bytesToUint16(BIG_ENDIAN, res.payload[2:4]) != quantity {
			   err = ErrProtocolError
			   return
		   }

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err	= ErrProtocolError
			return
		}

		err	= mapExceptionCodeToError(res.payload[0])

	default:
		err	= ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

func (mc *ModbusClient) executeRequest(req *pdu) (res *pdu, err error) {
	// send the request over the wire, wait for and decode the response
	res, err	= mc.transport.ExecuteRequest(req)
	if err != nil {
		return
	}

	// make sure the source unit id matches that of the request
	if (res.functionCode & 0x80) == 0x00 && res.unitId != req.unitId {
		err = ErrBadUnitId
		return
	}
	// accept errors from gateway devices (using special unit id #255)
	if (res.functionCode & 0x80) == 0x80 &&
		(res.unitId != req.unitId && res.unitId != 0xff) {
		err = ErrBadUnitId
		return
	}

	return
}
