package modbus

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type RegType	uint
type Endianness uint
type WordOrder	uint
const (
	PARITY_NONE         uint = 0
	PARITY_EVEN         uint = 1
	PARITY_ODD          uint = 2

	HOLDING_REGISTER    RegType = 0
	INPUT_REGISTER      RegType = 1

	// endianness of 16-bit registers
	BIG_ENDIAN          Endianness = 1
	LITTLE_ENDIAN       Endianness = 2

	// word order of 32-bit registers
	HIGH_WORD_FIRST     WordOrder = 1
	LOW_WORD_FIRST      WordOrder = 2
)

// Modbus client configuration object.
type ClientConfiguration struct {
	// URL sets the client mode and target location in the form
	// <mode>://<serial device or host:port> e.g. tcp://plc:502
	URL           string
	// Speed sets the serial link speed (in bps, rtu only)
	Speed         uint
	// DataBits sets the number of bits per serial character (rtu only)
	DataBits      uint
	// Parity sets the serial link parity mode (rtu only)
	Parity        uint
	// StopBits sets the number of serial stop bits (rtu only)
	StopBits      uint
	// Timeout sets the request timeout value
	Timeout       time.Duration
	// TLSClientCert sets the client-side TLS key pair (tcp+tls only)
	TLSClientCert *tls.Certificate
	// TLSRootCAs sets the list of CA certificates used to authenticate
	// the server (tcp+tls only). Leaf (i.e. server) certificates can also
	// be used in case of self-signed certs, or if cert pinning is required.
	TLSRootCAs    *x509.CertPool
	// Logger provides a custom sink for log messages.
	// If nil, messages will be written to stdout.
	Logger        *log.Logger
}

// Modbus client object.
type ModbusClient struct {
	conf          ClientConfiguration
	logger        *logger
	lock          sync.Mutex
	endianness    Endianness
	wordOrder     WordOrder
	transport     transport
	unitId        uint8
	transportType transportType
}

// NewClient creates, configures and returns a modbus client object.
func NewClient(conf *ClientConfiguration) (mc *ModbusClient, err error) {
	var clientType string
	var splitURL   []string

	mc = &ModbusClient{
		conf: *conf,
	}

	splitURL = strings.SplitN(mc.conf.URL, "://", 2)
	if len(splitURL) == 2 {
		clientType  = splitURL[0]
		mc.conf.URL = splitURL[1]
	}

	mc.logger = newLogger(
		fmt.Sprintf("modbus-client(%s)", mc.conf.URL), conf.Logger)

	switch clientType {
	case "rtu":
		// set useful defaults
		if mc.conf.Speed == 0 {
			mc.conf.Speed	= 19200
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

		mc.transportType    = modbusRTU

	case "rtuovertcp":
		if mc.conf.Speed == 0 {
			mc.conf.Speed   = 19200
		}

		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		mc.transportType    = modbusRTUOverTCP

	case "rtuoverudp":
		if mc.conf.Speed == 0 {
			mc.conf.Speed   = 19200
		}

		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		mc.transportType    = modbusRTUOverUDP

	case "tcp":
		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		mc.transportType    = modbusTCP

	case "tcp+tls":
		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		// expect a client-side certificate for mutual auth as the
		// modbus/mpab protocol has no inherent auth facility.
		// (see requirements R-08 and R-19 of the MBAPS spec)
		if mc.conf.TLSClientCert == nil {
			mc.logger.Errorf("missing client certificate")
			err = ErrConfigurationError
			return
		}

		// expect a CertPool object containing at least 1 CA or
		// leaf certificate to validate the server-side cert
		if mc.conf.TLSRootCAs == nil {
			mc.logger.Errorf("missing CA/server certificate")
			err = ErrConfigurationError
			return
		}

		mc.transportType    = modbusTCPOverTLS

	case "udp":
		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		mc.transportType    = modbusTCPOverUDP

	default:
		if len(splitURL) != 2 {
			mc.logger.Errorf("missing client type in URL '%s'", mc.conf.URL)
		} else {
			mc.logger.Errorf("unsupported client type '%s'", clientType)
		}
		err	= ErrConfigurationError
		return
	}

	mc.unitId     = 1
	mc.endianness = BIG_ENDIAN
	mc.wordOrder  = HIGH_WORD_FIRST

	return
}

// Opens the underlying transport (network socket or serial line).
func (mc *ModbusClient) Open() (err error) {
	var spw		*serialPortWrapper
	var sock	net.Conn

	mc.lock.Lock()
	defer mc.lock.Unlock()

	switch mc.transportType {
	case modbusRTU:
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
			spw, mc.conf.URL, mc.conf.Speed, mc.conf.Timeout, mc.conf.Logger)

	case modbusRTUOverTCP:
		// connect to the remote host
		sock, err = net.DialTimeout("tcp", mc.conf.URL, 5 * time.Second)
		if err != nil {
			return
		}

		// discard potentially stale serial data
		discard(sock)

		// create the RTU transport
		mc.transport = newRTUTransport(
			sock, mc.conf.URL, mc.conf.Speed, mc.conf.Timeout, mc.conf.Logger)

	case modbusRTUOverUDP:
		// open a socket to the remote host (note: no actual connection is
		// being made as UDP is connection-less)
		sock, err = net.DialTimeout("udp", mc.conf.URL, 5 * time.Second)
		if err != nil {
			return
		}

		// create the RTU transport, wrapping the UDP socket in
		// an adapter to allow the transport to read the stream of
		// packets byte per byte
		mc.transport = newRTUTransport(
			newUDPSockWrapper(sock),
			mc.conf.URL, mc.conf.Speed, mc.conf.Timeout, mc.conf.Logger)

	case modbusTCP:
		// connect to the remote host
		sock, err = net.DialTimeout("tcp", mc.conf.URL, 5 * time.Second)
		if err != nil {
			return
		}

		// create the TCP transport
		mc.transport = newTCPTransport(sock, mc.conf.Timeout, mc.conf.Logger)

	case modbusTCPOverTLS:
		// connect to the remote host with TLS
		sock, err = tls.DialWithDialer(
			&net.Dialer{
				Deadline: time.Now().Add(15 * time.Second),
			}, "tcp", mc.conf.URL,
			&tls.Config{
				Certificates: []tls.Certificate{
					*mc.conf.TLSClientCert,
				},
				RootCAs:     mc.conf.TLSRootCAs,
				// mandate TLS 1.2 or higher (see R-01 of the MBAPS spec)
				MinVersion:  tls.VersionTLS12,
			})
		if err != nil {
			return
		}

		// force the TLS handshake
		err = sock.(*tls.Conn).Handshake()
		if err != nil {
			sock.Close()
			return
		}

		// create the TCP transport
		mc.transport = newTCPTransport(sock, mc.conf.Timeout, mc.conf.Logger)

	case modbusTCPOverUDP:
		// open a socket to the remote host (note: no actual connection is
		// being made as UDP is connection-less)
		sock, err = net.DialTimeout("udp", mc.conf.URL, 5 * time.Second)
		if err != nil {
			return
		}

		// create the TCP transport, wrapping the UDP socket in
		// an adapter to allow the transport to read the stream of
		// packets byte per byte
		mc.transport = newTCPTransport(
			newUDPSockWrapper(sock), mc.conf.Timeout, mc.conf.Logger)

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

	// read quantity uint16 registers, as bytes
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

	// read 1 uint16 register, as bytes
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

// Reads multiple 64-bit registers.
func (mc *ModbusClient) ReadUint64s(addr uint16, quantity uint16, regType RegType) (values []uint64, err error) {
	var mbPayload	[]byte

	// read 4 * quantity uint16 registers, as bytes
	mbPayload, err	= mc.readRegisters(addr, quantity * 4, regType)
	if err != nil {
		return
	}

	// decode payload bytes as uint64s
	values	= bytesToUint64s(mc.endianness, mc.wordOrder, mbPayload)

	return
}

// Reads a single 64-bit register.
func (mc *ModbusClient) ReadUint64(addr uint16, regType RegType) (value uint64, err error) {
	var values	[]uint64

	values, err	= mc.ReadUint64s(addr, 1, regType)
	if err == nil {
		value	= values[0]
	}

	return
}

// Reads multiple 64-bit float registers.
func (mc *ModbusClient) ReadFloat64s(addr uint16, quantity uint16, regType RegType) (values []float64, err error) {
	var mbPayload	[]byte

	// read 4 * quantity uint16 registers, as bytes
	mbPayload, err	= mc.readRegisters(addr, quantity * 4, regType)
	if err != nil {
		return
	}

	// decode payload bytes as float64s
	values	= bytesToFloat64s(mc.endianness, mc.wordOrder, mbPayload)

	return
}

// Reads a single 64-bit float register.
func (mc *ModbusClient) ReadFloat64(addr uint16, regType RegType) (value float64, err error) {
	var values	[]float64

	values, err	= mc.ReadFloat64s(addr, 1, regType)
	if err == nil {
		value	= values[0]
	}

	return
}

// Reads one or multiple 16-bit registers (function code 03 or 04) as bytes.
// A per-register byteswap is performed if endianness is set to LITTLE_ENDIAN.
func (mc *ModbusClient) ReadBytes(addr uint16, quantity uint16, regType RegType) (values []byte, err error) {
	values, err = mc.readBytes(addr, quantity, regType, true)

	return
}

// Reads one or multiple 16-bit registers (function code 03 or 04) as bytes.
// No byte or word reordering is performed: bytes are returned exactly as they come
// off the wire, allowing the caller to handle encoding/endianness/word order manually.
func (mc *ModbusClient) ReadRawBytes(addr uint16, quantity uint16, regType RegType) (values []byte, err error) {
	values, err = mc.readBytes(addr, quantity, regType, false)

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
		unitId:	      mc.unitId,
		functionCode: fcWriteSingleCoil,
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
		unitId:       mc.unitId,
		functionCode: fcWriteMultipleCoils,
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
		unitId:	      mc.unitId,
		functionCode: fcWriteSingleRegister,
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

// Writes multiple 64-bit registers.
func (mc *ModbusClient) WriteUint64s(addr uint16, values []uint64) (err error) {
	var payload	[]byte

	// turn registers to bytes
	for _, value := range values {
		payload	= append(payload, uint64ToBytes(mc.endianness, mc.wordOrder, value)...)
	}

	err = mc.writeRegisters(addr, payload)

	return
}

// Writes a single 64-bit register.
func (mc *ModbusClient) WriteUint64(addr uint16, value uint64) (err error) {
	err = mc.writeRegisters(addr, uint64ToBytes(mc.endianness, mc.wordOrder, value))

	return
}

// Writes multiple 64-bit float registers.
func (mc *ModbusClient) WriteFloat64s(addr uint16, values []float64) (err error) {
	var payload	[]byte

	// turn registers to bytes
	for _, value := range values {
		payload	= append(payload, float64ToBytes(mc.endianness, mc.wordOrder, value)...)
	}

	err = mc.writeRegisters(addr, payload)

	return
}

// Writes a single 64-bit float register.
func (mc *ModbusClient) WriteFloat64(addr uint16, value float64) (err error) {
	err = mc.writeRegisters(addr, float64ToBytes(mc.endianness, mc.wordOrder, value))

	return
}

// Writes the given slice of bytes to 16-bit registers starting at addr.
// A per-register byteswap is performed if endianness is set to LITTLE_ENDIAN.
// Odd byte quantities are padded with a null byte to fall on 16-bit register boundaries.
func (mc *ModbusClient) WriteBytes(addr uint16, values []byte) (err error) {
	err = mc.writeBytes(addr, values, true)

	return
}

// Writes the given slice of bytes to 16-bit registers starting at addr.
// No byte or word reordering is performed: bytes are pushed to the wire as-is,
// allowing the caller to handle encoding/endianness/word order manually.
// Odd byte quantities are padded with a null byte to fall on 16-bit register boundaries.
func (mc *ModbusClient) WriteRawBytes(addr uint16, values []byte) (err error) {
	err = mc.writeBytes(addr, values, false)

	return
}

/*** unexported methods ***/
// Reads one or multiple 16-bit registers (function code 03 or 04) as bytes.
func (mc *ModbusClient) readBytes(addr uint16, quantity uint16, regType RegType, observeEndianness bool) (values []byte, err error) {
	var regCount uint16

	// read enough registers to get the requested number of bytes
	// (2 bytes per reg)
	regCount = (quantity / 2) + (quantity % 2)

	values, err = mc.readRegisters(addr, regCount, regType)
	if err != nil {
		return
	}

	// swap bytes on register boundaries if requested by the caller
	// and endianness is set to little endian
	if observeEndianness && mc.endianness == LITTLE_ENDIAN {
		for i := 0; i < len(values); i += 2 {
			values[i], values[i+1] = values[i+1], values[i]
		}
	}

	// pop the last byte on odd quantities
	if quantity % 2 == 1 {
		values = values[0:len(values) - 1]
	}

	return
}

// Writes the given slice of bytes to 16-bit registers starting at addr.
func (mc *ModbusClient) writeBytes(addr uint16, values []byte, observeEndianness bool) (err error) {
	// pad odd quantities to make for full registers
	if len(values) % 2 == 1 {
		values = append(values, 0x00)
	}

	// swap bytes on register boundaries if requested by the caller
	// and endianness is set to little endian
	if observeEndianness && mc.endianness == LITTLE_ENDIAN {
		for i := 0; i < len(values); i += 2 {
			values[i], values[i+1] = values[i+1], values[i]
		}
	}

	err = mc.writeRegisters(addr, values)

	return
}

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
		req.functionCode = fcReadDiscreteInputs
	} else {
		req.functionCode = fcReadCoils
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
	case HOLDING_REGISTER: req.functionCode = fcReadHoldingRegisters
	case INPUT_REGISTER:   req.functionCode = fcReadInputRegisters
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
	var req           *pdu
	var res           *pdu
	var payloadLength uint16
	var quantity      uint16

	mc.lock.Lock()
	defer mc.lock.Unlock()

	payloadLength = uint16(len(values))
	quantity      = payloadLength / 2

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
		unitId:	      mc.unitId,
		functionCode: fcWriteMultipleRegisters,
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
		// map i/o timeouts to ErrRequestTimedOut
		if os.IsTimeout(err) {
			err = ErrRequestTimedOut
		}
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
