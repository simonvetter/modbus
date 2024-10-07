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

	"go.bug.st/serial"
)

// RegisterType is either HoldingRegister or InputRegister.
type RegisterType uint

// Endianness is either BigEndian or LittleEndian.
type Endianness uint

// WordOrder is either HighWordFirst of LowWordFirst.
type WordOrder uint

const (
	// HoldingRegister is a writable 16 bit register.
	HoldingRegister RegisterType = 0
	// InputRegister is a read-only 16 bit register.
	InputRegister RegisterType = 1

	// BigEndian means that the most significant bit is first, this is the default.
	BigEndian Endianness = 1
	// LittleEndian means that the least significant bit is first.
	LittleEndian Endianness = 2

	// HighWordFirst means that the most significant register is first, this is the default.
	HighWordFirst WordOrder = 1
	// LowWordFirst means that the least significant register is first.
	LowWordFirst WordOrder = 2
)

// Configuration stores the configuration needed to create a Modbus client.
type Configuration struct {
	// URL sets the client mode and target location in the form
	// <mode>://<serial device or host:port> e.g. tcp://plc:502.
	URL string
	// Speed sets the serial link speed (in bps, rtu only).
	Speed int
	// DataBits sets the number of bits per serial character (rtu only).
	DataBits int
	// Parity sets the serial link parity mode (rtu only).
	Parity serial.Parity
	// StopBits sets the number of serial stop bits (rtu only).
	StopBits serial.StopBits
	// Timeout sets the request timeout value.
	Timeout time.Duration
	// TLSClientCert sets the client-side TLS key pair (tcp+tls only).
	TLSClientCert *tls.Certificate
	// TLSRootCAs sets the list of CA certificates used to authenticate
	// the server (tcp+tls only). Leaf (i.e. server) certificates can also
	// be used in case of self-signed certs, or if cert pinning is required.
	TLSRootCAs *x509.CertPool
	// Logger provides a custom sink for log messages.
	// If nil, messages will be written to stdout.
	Logger *log.Logger
}

// Client is the actual client for a Modbus transport.
type Client struct {
	conf          Configuration
	logger        *logger
	lock          sync.Mutex
	unitID        uint8
	endianness    Endianness
	wordOrder     WordOrder
	transport     transport
	transportType transportType
}

// NewClient creates, configures and returns a modbus client object.
func NewClient(conf *Configuration) (mc *Client, err error) {
	var clientType string
	var splitURL []string

	mc = &Client{
		conf:       *conf,
		unitID:     1,
		endianness: BigEndian,
		wordOrder:  HighWordFirst,
	}

	splitURL = strings.SplitN(mc.conf.URL, "://", 2)
	if len(splitURL) == 2 {
		clientType = splitURL[0]
		mc.conf.URL = splitURL[1]
	}

	mc.logger = newLogger(
		fmt.Sprintf("modbus-client(%s)", mc.conf.URL), conf.Logger)

	switch clientType {
	case "rtu":
		// set useful defaults
		if mc.conf.Speed == 0 {
			mc.conf.Speed = 19200
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

		if mc.conf.Parity == serial.NoParity {
			mc.conf.StopBits = serial.TwoStopBits
		} else {
			mc.conf.StopBits = serial.OneStopBit
		}

		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 300 * time.Millisecond
		}

		mc.transportType = modbusRTU

	case "rtuovertcp":
		if mc.conf.Speed == 0 {
			mc.conf.Speed = 19200
		}

		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		mc.transportType = modbusRTUOverTCP

	case "rtuoverudp":
		if mc.conf.Speed == 0 {
			mc.conf.Speed = 19200
		}

		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		mc.transportType = modbusRTUOverUDP

	case "tcp":
		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		mc.transportType = modbusTCP

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

		mc.transportType = modbusTCPOverTLS

	case "udp":
		if mc.conf.Timeout == 0 {
			mc.conf.Timeout = 1 * time.Second
		}

		mc.transportType = modbusTCPOverUDP

	default:
		if len(splitURL) != 2 {
			mc.logger.Errorf("missing client type in URL '%s'", mc.conf.URL)
		} else {
			mc.logger.Errorf("unsupported client type '%s'", clientType)
		}
		err = ErrConfigurationError
		return
	}

	return
}

// Open opens the underlying transport (network socket or serial line).
func (mc *Client) Open() (err error) {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	var spw *serialPortWrapper
	var sw *socketWrapper
	var sock net.Conn

	if mc.transport != nil {
		return ErrTransportIsAlreadyOpen
	}

	switch mc.transportType {
	case modbusRTU:
		// create a serial port wrapper object
		spw = newSerialPortWrapper(&serialPortConfig{
			Device:   mc.conf.URL,
			Speed:    mc.conf.Speed,
			DataBits: mc.conf.DataBits,
			Parity:   mc.conf.Parity,
			StopBits: mc.conf.StopBits,
		})

		// open the serial device
		err = spw.Open()
		if err != nil {
			return
		}

		// discard potentially stale serial data
		err = spw.Reset()
		if err != nil {
			return err
		}

		// create the RTU transport
		mc.transport = newRTUTransport(
			spw, mc.conf.URL, mc.conf.Speed, mc.conf.Timeout, mc.conf.Logger)

	case modbusRTUOverTCP:
		// connect to the remote host
		sock, err = net.DialTimeout("tcp", mc.conf.URL, 5*time.Second)
		if err != nil {
			return
		}

		sw = newSocketWrapper(sock)
		// discard potentially stale data
		sw.Reset()

		// create the RTU transport
		mc.transport = newRTUTransport(
			sw, mc.conf.URL, mc.conf.Speed, mc.conf.Timeout, mc.conf.Logger)

	case modbusRTUOverUDP:
		// open a socket to the remote host (note: no actual connection is
		// being made as UDP is connection-less)
		sock, err = net.DialTimeout("udp", mc.conf.URL, 5*time.Second)
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
		sock, err = net.DialTimeout("tcp", mc.conf.URL, 5*time.Second)
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
				RootCAs: mc.conf.TLSRootCAs,
				// mandate TLS 1.2 or higher (see R-01 of the MBAPS spec)
				MinVersion: tls.VersionTLS12,
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

		// create the TCP transport, wrapping the TLS socket in
		// an adapter to work around write timeouts corrupting internal
		// state (see https://pkg.go.dev/crypto/tls#Conn.SetWriteDeadline)
		mc.transport = newTCPTransport(
			newTLSSockWrapper(sock), mc.conf.Timeout, mc.conf.Logger)

	case modbusTCPOverUDP:
		// open a socket to the remote host (note: no actual connection is
		// being made as UDP is connection-less)
		sock, err = net.DialTimeout("udp", mc.conf.URL, 5*time.Second)
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

// Close closes the underlying transport.
func (mc *Client) Close() error {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	if mc.transport == nil {
		return ErrTransportIsAlreadyClosed
	}

	err := mc.transport.Close()
	if err != nil {
		return err
	}

	mc.transport = nil
	return nil
}

// SetUnitID sets the unit id of subsequent requests.
func (mc *Client) SetUnitID(id uint8) (err error) {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	mc.unitID = id

	return
}

// SetEncoding sets the encoding (endianness and word ordering) of subsequent requests.
func (mc *Client) SetEncoding(endianness Endianness, wordOrder WordOrder) (err error) {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	if endianness != BigEndian && endianness != LittleEndian {
		mc.logger.Errorf("unknown endianness value %v", endianness)
		err = ErrUnexpectedParameters
		return
	}

	if wordOrder != HighWordFirst && wordOrder != LowWordFirst {
		mc.logger.Errorf("unknown word order value %v", wordOrder)
		err = ErrUnexpectedParameters
		return
	}

	mc.endianness = endianness
	mc.wordOrder = wordOrder

	return
}

// WithUnitID is an option that can be passed to the Read and Write functions
func WithUnitID(unitID uint8) func(*Client) {
	return func(mc *Client) {
		mc.unitID = unitID
	}
}

// WithEndianess is an option that can be passed to the Read and Write functions
func WithEndianess(endianness Endianness) func(*Client) {
	return func(mc *Client) {
		mc.endianness = endianness
	}
}

// WithWordOrder is an option that can be passed to the Read and Write functions
func WithWordOrder(wordOrder WordOrder) func(*Client) {
	return func(mc *Client) {
		mc.wordOrder = wordOrder
	}
}

// ReadCoils reads multiple coils (function code 01).
func (mc *Client) ReadCoils(addr uint16, quantity uint16, options ...func(*Client)) (values []bool, err error) {
	values, err = mc.readBools(addr, quantity, false, options...)

	return
}

// ReadCoil reads a single coil (function code 01).
func (mc *Client) ReadCoil(addr uint16, options ...func(*Client)) (value bool, err error) {
	var values []bool

	values, err = mc.readBools(addr, 1, false, options...)
	if err == nil {
		value = values[0]
	}

	return
}

// ReadDiscreteInputs reads multiple discrete inputs (function code 02).
func (mc *Client) ReadDiscreteInputs(addr uint16, quantity uint16, options ...func(*Client)) (values []bool, err error) {
	values, err = mc.readBools(addr, quantity, true, options...)

	return
}

// ReadDiscreteInput reads a single discrete input (function code 02).
func (mc *Client) ReadDiscreteInput(addr uint16, options ...func(*Client)) (value bool, err error) {
	var values []bool

	values, err = mc.readBools(addr, 1, true, options...)
	if err == nil {
		value = values[0]
	}

	return
}

// ReadRegisters reads multiple 16-bit registers (function code 03 or 04).
func (mc *Client) ReadRegisters(addr uint16, quantity uint16, regType RegisterType, options ...func(*Client)) (values []uint16, err error) {
	var mbPayload []byte

	// read quantity uint16 registers, as bytes
	mbPayload, err = mc.readRegisters(addr, quantity, regType, options...)
	if err != nil {
		return
	}

	// decode payload bytes as uint16s
	values = bytesToUint16s(mc.endianness, mbPayload)

	return
}

// ReadRegister reads a single 16-bit register (function code 03 or 04).
func (mc *Client) ReadRegister(addr uint16, regType RegisterType, options ...func(*Client)) (value uint16, err error) {
	var values []uint16

	// read 1 uint16 register, as bytes
	values, err = mc.ReadRegisters(addr, 1, regType, options...)
	if err == nil {
		value = values[0]
	}

	return
}

// ReadUint32s reads multiple 32-bit registers.
func (mc *Client) ReadUint32s(addr uint16, quantity uint16, regType RegisterType, options ...func(*Client)) (values []uint32, err error) {
	var mbPayload []byte

	// read 2 * quantity uint16 registers, as bytes
	mbPayload, err = mc.readRegisters(addr, quantity*2, regType, options...)
	if err != nil {
		return
	}

	// decode payload bytes as uint32s
	values = bytesToUint32s(mc.endianness, mc.wordOrder, mbPayload)

	return
}

// ReadUint32 reads a single 32-bit register.
func (mc *Client) ReadUint32(addr uint16, regType RegisterType, options ...func(*Client)) (value uint32, err error) {
	var values []uint32

	values, err = mc.ReadUint32s(addr, 1, regType, options...)
	if err == nil {
		value = values[0]
	}

	return
}

// ReadFloat32s reads multiple 32-bit float registers.
func (mc *Client) ReadFloat32s(addr uint16, quantity uint16, regType RegisterType, options ...func(*Client)) (values []float32, err error) {
	var mbPayload []byte

	// read 2 * quantity uint16 registers, as bytes
	mbPayload, err = mc.readRegisters(addr, quantity*2, regType, options...)
	if err != nil {
		return
	}

	// decode payload bytes as float32s
	values = bytesToFloat32s(mc.endianness, mc.wordOrder, mbPayload)

	return
}

// ReadFloat32 reads a single 32-bit float register.
func (mc *Client) ReadFloat32(addr uint16, regType RegisterType, options ...func(*Client)) (value float32, err error) {
	var values []float32

	values, err = mc.ReadFloat32s(addr, 1, regType, options...)
	if err == nil {
		value = values[0]
	}

	return
}

// ReadUint64s reads multiple 64-bit registers.
func (mc *Client) ReadUint64s(addr uint16, quantity uint16, regType RegisterType, options ...func(*Client)) (values []uint64, err error) {
	var mbPayload []byte

	// read 4 * quantity uint16 registers, as bytes
	mbPayload, err = mc.readRegisters(addr, quantity*4, regType, options...)
	if err != nil {
		return
	}

	// decode payload bytes as uint64s
	values = bytesToUint64s(mc.endianness, mc.wordOrder, mbPayload)

	return
}

// ReadUint64 reads a single 64-bit register.
func (mc *Client) ReadUint64(addr uint16, regType RegisterType, options ...func(*Client)) (value uint64, err error) {
	var values []uint64

	values, err = mc.ReadUint64s(addr, 1, regType, options...)
	if err == nil {
		value = values[0]
	}

	return
}

// ReadFloat64s reads multiple 64-bit float registers.
func (mc *Client) ReadFloat64s(addr uint16, quantity uint16, regType RegisterType, options ...func(*Client)) (values []float64, err error) {
	var mbPayload []byte

	// read 4 * quantity uint16 registers, as bytes
	mbPayload, err = mc.readRegisters(addr, quantity*4, regType, options...)
	if err != nil {
		return
	}

	// decode payload bytes as float64s
	values = bytesToFloat64s(mc.endianness, mc.wordOrder, mbPayload)

	return
}

// ReadFloat64 reads a single 64-bit float register.
func (mc *Client) ReadFloat64(addr uint16, regType RegisterType, options ...func(*Client)) (value float64, err error) {
	var values []float64

	values, err = mc.ReadFloat64s(addr, 1, regType, options...)
	if err == nil {
		value = values[0]
	}

	return
}

// ReadBytes reads one or multiple 16-bit registers (function code 03 or 04) as bytes.
// A per-register byteswap is performed if endianness is set to LittleEndian.
func (mc *Client) ReadBytes(addr uint16, quantity uint16, regType RegisterType, options ...func(*Client)) (values []byte, err error) {
	values, err = mc.readBytes(addr, quantity, regType, true, options...)

	return
}

// ReadRawBytes reads one or multiple 16-bit registers (function code 03 or 04) as bytes.
// No byte or word reordering is performed: bytes are returned exactly as they come
// off the wire, allowing the caller to handle encoding/endianness/word order manually.
func (mc *Client) ReadRawBytes(addr uint16, quantity uint16, regType RegisterType, options ...func(*Client)) (values []byte, err error) {
	values, err = mc.readBytes(addr, quantity, regType, false, options...)

	return
}

// WriteCoil writes a single coil (function code 05).
func (mc *Client) WriteCoil(addr uint16, value bool, options ...func(*Client)) (err error) {
	var req *pdu
	var res *pdu

	mc.lock.Lock()
	defer mc.lock.Unlock()
	for _, option := range options {
		option(mc)
	}

	// create and fill in the request object
	req = &pdu{
		unitID:       mc.unitID,
		functionCode: fcWriteSingleCoil,
	}

	// coil address
	req.payload = uint16ToBytes(BigEndian, addr)
	// coil value
	if value {
		req.payload = append(req.payload, 0xff, 0x00)
	} else {
		req.payload = append(req.payload, 0x00, 0x00)
	}

	// run the request across the transport and wait for a response
	res, err = mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.functionCode == req.functionCode:
		if len(res.payload) != 4 {
			err = ErrProtocolError
			mc.logger.Warningf("the length of the payload is not 4 but %d", len(res.payload))
			return
		}
		if bytesToUint16(BigEndian, res.payload[0:2]) != addr {
			err = ErrProtocolError
			mc.logger.Warningf("bytes 1-2 are not the expected coil address %d, but %d", addr, bytesToUint16(BigEndian, res.payload[0:2]))
			return
		}
		if value && (res.payload[2] != 0xff || res.payload[3] != 0x00) {
			err = ErrProtocolError
			mc.logger.Warningf("expected % x but got % x", []byte{0xff, 0x00}, res.payload[2:4])
			return
		}
		if !value && (res.payload[2] != 0x00 || res.payload[3] != 0x00) {
			err = ErrProtocolError
			mc.logger.Warningf("expected % x but got % x", []byte{0x00, 0x00}, res.payload[2:4])
			return
		}

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err = ErrProtocolError
			mc.logger.Warningf("expected the length of the payload to be 1, but the length is %d", len(res.payload))
			return
		}

		err = mapExceptionCodeToError(res.payload[0])

	default:
		err = ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

// WriteCoils writes multiple coils (function code 15).
func (mc *Client) WriteCoils(addr uint16, values []bool, options ...func(*Client)) (err error) {
	var req *pdu
	var res *pdu
	var quantity uint16
	var encodedValues []byte

	mc.lock.Lock()
	defer mc.lock.Unlock()
	for _, option := range options {
		option(mc)
	}

	quantity = uint16(len(values))
	if quantity == 0 {
		err = ErrUnexpectedParameters
		mc.logger.Error("quantity of coils is 0")
		return
	}

	if quantity > 0x7b0 {
		err = ErrUnexpectedParameters
		mc.logger.Error("quantity of coils exceeds 1968")
		return
	}

	if uint32(addr)+uint32(quantity)-1 > 0xffff {
		err = ErrUnexpectedParameters
		mc.logger.Error("end coil address is past 0xffff")
		return
	}

	encodedValues = encodeBools(values)

	// create and fill in the request object
	req = &pdu{
		unitID:       mc.unitID,
		functionCode: fcWriteMultipleCoils,
	}

	// start address
	req.payload = uint16ToBytes(BigEndian, addr)
	// quantity
	req.payload = append(req.payload, uint16ToBytes(BigEndian, quantity)...)
	// byte count
	req.payload = append(req.payload, byte(len(encodedValues)))
	// payload
	req.payload = append(req.payload, encodedValues...)

	// run the request across the transport and wait for a response
	res, err = mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.functionCode == req.functionCode:
		if len(res.payload) != 4 {
			err = ErrProtocolError
			mc.logger.Warningf("the length of the payload is not 4 but %d", len(res.payload))
			return
		}
		if bytesToUint16(BigEndian, res.payload[0:2]) != addr {
			err = ErrProtocolError
			mc.logger.Warningf("bytes 1-2 are not the expected coil address %d, but %d", addr, bytesToUint16(BigEndian, res.payload[0:2]))
			return
		}
		if bytesToUint16(BigEndian, res.payload[2:4]) != quantity {
			err = ErrProtocolError
			mc.logger.Warningf("bytes 3-4 are not the expected quantity of coils %d, but %d", quantity, bytesToUint16(BigEndian, res.payload[2:4]))
			return
		}

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err = ErrProtocolError
			mc.logger.Warningf("expected the length of the payload to be 1, but the length is %d", len(res.payload))
			return
		}

		err = mapExceptionCodeToError(res.payload[0])

	default:
		err = ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

// WriteRegister writes a single 16-bit register (function code 06).
func (mc *Client) WriteRegister(addr uint16, value uint16, options ...func(*Client)) (err error) {
	var req *pdu
	var res *pdu

	mc.lock.Lock()
	defer mc.lock.Unlock()
	for _, option := range options {
		option(mc)
	}

	// create and fill in the request object
	req = &pdu{
		unitID:       mc.unitID,
		functionCode: fcWriteSingleRegister,
	}

	// register address
	req.payload = uint16ToBytes(BigEndian, addr)
	// register value
	req.payload = append(req.payload, uint16ToBytes(mc.endianness, value)...)

	// run the request across the transport and wait for a response
	res, err = mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.functionCode == req.functionCode:
		if len(res.payload) != 4 {
			err = ErrProtocolError
			mc.logger.Warningf("the length of the payload is not 4 but %d", len(res.payload))
			return
		}
		if bytesToUint16(BigEndian, res.payload[0:2]) != addr {
			err = ErrProtocolError
			mc.logger.Warningf("bytes 1-2 are not the expected coil address %d, but %d", addr, bytesToUint16(BigEndian, res.payload[0:2]))
			return
		}
		if bytesToUint16(mc.endianness, res.payload[2:4]) != value {
			err = ErrProtocolError
			mc.logger.Warningf("bytes 3-4 are not the expected value %d, but %d", value, bytesToUint16(mc.endianness, res.payload[2:4]))
			return
		}

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err = ErrProtocolError
			mc.logger.Warningf("expected the length of the payload to be 1, but the length is %d", len(res.payload))
			return
		}

		err = mapExceptionCodeToError(res.payload[0])

	default:
		err = ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

// WriteRegisters writes multiple 16-bit registers (function code 16).
func (mc *Client) WriteRegisters(addr uint16, values []uint16, options ...func(*Client)) (err error) {
	var payload []byte

	// turn registers to bytes
	for _, value := range values {
		payload = append(payload, uint16ToBytes(mc.endianness, value)...)
	}

	err = mc.writeRegisters(addr, payload, options...)

	return
}

// WriteUint32s writes multiple 32-bit registers.
func (mc *Client) WriteUint32s(addr uint16, values []uint32, options ...func(*Client)) (err error) {
	mc.lock.Lock()
	for _, option := range options {
		option(mc)
	}
	var payload []byte
	// turn registers to bytes
	for _, value := range values {
		payload = append(payload, uint32ToBytes(mc.endianness, mc.wordOrder, value)...)
	}
	mc.lock.Unlock()

	err = mc.writeRegisters(addr, payload, options...)

	return
}

// WriteUint32 writes a single 32-bit register.
func (mc *Client) WriteUint32(addr uint16, value uint32, options ...func(*Client)) (err error) {
	mc.lock.Lock()
	for _, option := range options {
		option(mc)
	}
	payload := uint32ToBytes(mc.endianness, mc.wordOrder, value)
	mc.lock.Unlock()
	err = mc.writeRegisters(addr, payload, options...)

	return
}

// WriteFloat32s writes multiple 32-bit float registers.
func (mc *Client) WriteFloat32s(addr uint16, values []float32, options ...func(*Client)) (err error) {
	mc.lock.Lock()
	for _, option := range options {
		option(mc)
	}
	var payload []byte
	// turn registers to bytes
	for _, value := range values {
		payload = append(payload, float32ToBytes(mc.endianness, mc.wordOrder, value)...)
	}
	mc.lock.Unlock()

	err = mc.writeRegisters(addr, payload, options...)

	return
}

// WriteFloat32 writes a single 32-bit float register.
func (mc *Client) WriteFloat32(addr uint16, value float32, options ...func(*Client)) (err error) {
	mc.lock.Lock()
	for _, option := range options {
		option(mc)
	}
	payload := float32ToBytes(mc.endianness, mc.wordOrder, value)
	mc.lock.Unlock()

	err = mc.writeRegisters(addr, payload, options...)

	return
}

// WriteUint64s writes multiple 64-bit registers.
func (mc *Client) WriteUint64s(addr uint16, values []uint64, options ...func(*Client)) (err error) {
	mc.lock.Lock()
	for _, option := range options {
		option(mc)
	}
	var payload []byte
	// turn registers to bytes
	for _, value := range values {
		payload = append(payload, uint64ToBytes(mc.endianness, mc.wordOrder, value)...)
	}
	mc.lock.Unlock()

	err = mc.writeRegisters(addr, payload, options...)

	return
}

// WriteUint64 writes a single 64-bit register.
func (mc *Client) WriteUint64(addr uint16, value uint64, options ...func(*Client)) (err error) {
	mc.lock.Lock()
	for _, option := range options {
		option(mc)
	}
	payload := uint64ToBytes(mc.endianness, mc.wordOrder, value)
	mc.lock.Unlock()

	err = mc.writeRegisters(addr, payload, options...)

	return
}

// WriteFloat64s writes multiple 64-bit float registers.
func (mc *Client) WriteFloat64s(addr uint16, values []float64, options ...func(*Client)) (err error) {
	mc.lock.Lock()
	for _, option := range options {
		option(mc)
	}
	var payload []byte
	// turn registers to bytes
	for _, value := range values {
		payload = append(payload, float64ToBytes(mc.endianness, mc.wordOrder, value)...)
	}
	mc.lock.Unlock()

	err = mc.writeRegisters(addr, payload, options...)

	return
}

// WriteFloat64 writes a single 64-bit float register.
func (mc *Client) WriteFloat64(addr uint16, value float64, options ...func(*Client)) (err error) {
	mc.lock.Lock()
	for _, option := range options {
		option(mc)
	}
	payload := float64ToBytes(mc.endianness, mc.wordOrder, value)
	mc.lock.Unlock()

	err = mc.writeRegisters(addr, payload, options...)

	return
}

// WriteBytes writes the given slice of bytes to 16-bit registers starting at addr.
// A per-register byteswap is performed if endianness is set to LittleEndian.
// Odd byte quantities are padded with a null byte to fall on 16-bit register boundaries.
func (mc *Client) WriteBytes(addr uint16, values []byte, options ...func(*Client)) (err error) {
	err = mc.writeBytes(addr, values, true, options...)

	return
}

// WriteRawBytes writes the given slice of bytes to 16-bit registers starting at addr.
// No byte or word reordering is performed: bytes are pushed to the wire as-is,
// allowing the caller to handle encoding/endianness/word order manually.
// Odd byte quantities are padded with a null byte to fall on 16-bit register boundaries.
func (mc *Client) WriteRawBytes(addr uint16, values []byte, options ...func(*Client)) (err error) {
	err = mc.writeBytes(addr, values, false, options...)

	return
}

// Reads one or multiple 16-bit registers (function code 03 or 04) as bytes.
func (mc *Client) readBytes(addr uint16, quantity uint16, regType RegisterType, observeEndianness bool, options ...func(*Client)) (values []byte, err error) {
	// read enough registers to get the requested number of bytes
	// (2 bytes per reg)
	regCount := (quantity / 2) + (quantity % 2)

	values, err = mc.readRegisters(addr, regCount, regType, options...)
	if err != nil {
		return
	}

	mc.lock.Lock()
	for _, option := range options {
		option(mc)
	}
	// swap bytes on register boundaries if requested by the caller
	// and endianness is set to little endian
	if observeEndianness && mc.endianness == LittleEndian {
		for i := 0; i < len(values); i += 2 {
			values[i], values[i+1] = values[i+1], values[i]
		}
	}
	mc.lock.Unlock()

	// pop the last byte on odd quantities
	if quantity%2 == 1 {
		values = values[0 : len(values)-1]
	}

	return
}

// Writes the given slice of bytes to 16-bit registers starting at addr.
func (mc *Client) writeBytes(addr uint16, values []byte, observeEndianness bool, options ...func(*Client)) (err error) {
	// pad odd quantities to make for full registers
	if len(values)%2 == 1 {
		values = append(values, 0x00)
	}

	mc.lock.Lock()
	for _, option := range options {
		option(mc)
	}
	// swap bytes on register boundaries if requested by the caller
	// and endianness is set to little endian
	if observeEndianness && mc.endianness == LittleEndian {
		for i := 0; i < len(values); i += 2 {
			values[i], values[i+1] = values[i+1], values[i]
		}
	}
	mc.lock.Unlock()

	err = mc.writeRegisters(addr, values, options...)

	return
}

// Reads and returns quantity booleans.
// Digital inputs are read if di is true, otherwise coils are read.
func (mc *Client) readBools(addr uint16, quantity uint16, di bool, options ...func(*Client)) (values []bool, err error) {
	var req *pdu
	var res *pdu
	var expectedLen int

	mc.lock.Lock()
	defer mc.lock.Unlock()
	for _, option := range options {
		option(mc)
	}

	if quantity == 0 {
		err = ErrUnexpectedParameters
		mc.logger.Error("quantity of coils/discrete inputs is 0")
		return
	}

	if quantity > 2000 {
		err = ErrUnexpectedParameters
		mc.logger.Error("quantity of coils/discrete inputs exceeds 2000")
		return
	}

	if uint32(addr)+uint32(quantity)-1 > 0xffff {
		err = ErrUnexpectedParameters
		mc.logger.Error("end coil/discrete input address is past 0xffff")
		return
	}

	// create and fill in the request object
	req = &pdu{
		unitID: mc.unitID,
	}

	if di {
		req.functionCode = fcReadDiscreteInputs
	} else {
		req.functionCode = fcReadCoils
	}

	// start address
	req.payload = uint16ToBytes(BigEndian, addr)
	// quantity
	req.payload = append(req.payload, uint16ToBytes(BigEndian, quantity)...)

	// run the request across the transport and wait for a response
	res, err = mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.functionCode == req.functionCode:
		// expect a payload of 1 byte (byte count) + 1 byte for 8 coils/discrete inputs)
		expectedLen = 1
		expectedLen += int(quantity) / 8
		if quantity%8 != 0 {
			expectedLen++
		}

		if len(res.payload) != expectedLen {
			err = ErrProtocolError
			mc.logger.Warningf("expected the length of the payload to be %d, but the length is %d", expectedLen, len(res.payload))
			return
		}

		// validate the byte count field
		if int(res.payload[0])+1 != expectedLen {
			err = ErrProtocolError
			mc.logger.Warningf("expected %d in the byte count field, but got %d", expectedLen, int(res.payload[0])+1)
			return
		}

		// turn bits into a bool slice
		values = decodeBools(quantity, res.payload[1:])

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err = ErrProtocolError
			mc.logger.Warningf("expected the length of the payload to be 1, but the length is %d", len(res.payload))
			return
		}

		err = mapExceptionCodeToError(res.payload[0])

	default:
		err = ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

// Reads and returns quantity registers of type regType, as bytes.
func (mc *Client) readRegisters(addr uint16, quantity uint16, regType RegisterType, options ...func(*Client)) (bytes []byte, err error) {
	var req *pdu
	var res *pdu

	mc.lock.Lock()
	defer mc.lock.Unlock()
	for _, option := range options {
		option(mc)
	}

	// create and fill in the request object
	req = &pdu{
		unitID: mc.unitID,
	}

	switch regType {
	case HoldingRegister:
		req.functionCode = fcReadHoldingRegisters
	case InputRegister:
		req.functionCode = fcReadInputRegisters
	default:
		err = ErrUnexpectedParameters
		mc.logger.Errorf("unexpected register type (%v)", regType)
		return
	}

	if quantity == 0 {
		err = ErrUnexpectedParameters
		mc.logger.Error("quantity of registers is 0")
		return
	}

	if quantity > 125 {
		err = ErrUnexpectedParameters
		mc.logger.Error("quantity of registers exceeds 125")
		return
	}

	if uint32(addr)+uint32(quantity)-1 > 0xffff {
		err = ErrUnexpectedParameters
		mc.logger.Error("end register address is past 0xffff")
		return
	}

	// start address
	req.payload = uint16ToBytes(BigEndian, addr)
	// quantity
	req.payload = append(req.payload, uint16ToBytes(BigEndian, quantity)...)

	// run the request across the transport and wait for a response
	res, err = mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.functionCode == req.functionCode:
		// make sure the payload length is what we expect
		// (1 byte of length + 2 bytes per register)
		if len(res.payload) != 1+2*int(quantity) {
			err = ErrProtocolError
			mc.logger.Warningf("expected the length of the payload to be %d, but the length is %d", 1+2*int(quantity), len(res.payload))
			return
		}

		// validate the byte count field
		// (2 bytes per register * number of registers)
		if uint(res.payload[0]) != 2*uint(quantity) {
			err = ErrProtocolError
			mc.logger.Warningf("expected %d in the byte count field, but got %d", 2*uint(quantity), int(res.payload[0]))
			return
		}

		// remove the byte count field from the returned slice
		bytes = res.payload[1:]

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err = ErrProtocolError
			mc.logger.Warningf("expected the length of the payload to be 1, but the length is %d", len(res.payload))
			return
		}

		err = mapExceptionCodeToError(res.payload[0])

	default:
		if len(res.payload) != 4 {
			err = ErrProtocolError
			mc.logger.Warningf("the length of the payload is not 4 but %d", len(res.payload))
			return
		}
		if bytesToUint16(BigEndian, res.payload[0:2]) != addr {
			err = ErrProtocolError
			mc.logger.Warningf("bytes 1-2 are not the expected coil address %d, but %d", addr, bytesToUint16(BigEndian, res.payload[0:2]))
			return
		}
		if bytesToUint16(BigEndian, res.payload[2:4]) != quantity {
			err = ErrProtocolError
			mc.logger.Warningf("bytes 3-4 are not the expected quantity of coils %d, but %d", quantity, bytesToUint16(BigEndian, res.payload[2:4]))
			return
		}

		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

// Writes multiple registers starting from base address addr.
// Register values are passed as bytes, each value being exactly 2 bytes.
func (mc *Client) writeRegisters(addr uint16, values []byte, options ...func(*Client)) (err error) {
	var req *pdu
	var res *pdu
	var payloadLength uint16
	var quantity uint16

	mc.lock.Lock()
	defer mc.lock.Unlock()
	for _, option := range options {
		option(mc)
	}

	payloadLength = uint16(len(values))
	quantity = payloadLength / 2

	if quantity == 0 {
		err = ErrUnexpectedParameters
		mc.logger.Error("quantity of registers is 0")
		return
	}

	if quantity > 123 {
		err = ErrUnexpectedParameters
		mc.logger.Error("quantity of registers exceeds 123")
		return
	}

	if uint32(addr)+uint32(quantity)-1 > 0xffff {
		err = ErrUnexpectedParameters
		mc.logger.Error("end register address is past 0xffff")
		return
	}

	// create and fill in the request object
	req = &pdu{
		unitID:       mc.unitID,
		functionCode: fcWriteMultipleRegisters,
	}

	// base address
	req.payload = uint16ToBytes(BigEndian, addr)
	// quantity of registers (2 bytes per register)
	req.payload = append(req.payload, uint16ToBytes(BigEndian, quantity)...)
	// byte count
	req.payload = append(req.payload, byte(payloadLength))
	// registers value
	req.payload = append(req.payload, values...)

	// run the request across the transport and wait for a response
	res, err = mc.executeRequest(req)
	if err != nil {
		return
	}

	// validate the response code
	switch {
	case res.functionCode == req.functionCode:
		if len(res.payload) != 4 {
			err = ErrProtocolError
			mc.logger.Warningf("the length of the payload is not 4 but %d", len(res.payload))
			return
		}
		if bytesToUint16(BigEndian, res.payload[0:2]) != addr {
			err = ErrProtocolError
			mc.logger.Warningf("bytes 1-2 are not the expected coil address %d, but %d", addr, bytesToUint16(BigEndian, res.payload[0:2]))
			return
		}
		if bytesToUint16(BigEndian, res.payload[2:4]) != quantity {
			err = ErrProtocolError
			mc.logger.Warningf("bytes 3-4 are not the expected value %d, but %d", quantity, bytesToUint16(mc.endianness, res.payload[2:4]))
			return
		}

	case res.functionCode == (req.functionCode | 0x80):
		if len(res.payload) != 1 {
			err = ErrProtocolError
			mc.logger.Warningf("expected the length of the payload to be 1, but the length is %d", len(res.payload))
			return
		}

		err = mapExceptionCodeToError(res.payload[0])

	default:
		err = ErrProtocolError
		mc.logger.Warningf("unexpected response code (%v)", res.functionCode)
	}

	return
}

func (mc *Client) executeRequest(req *pdu) (res *pdu, err error) {
	// send the request over the wire, wait for and decode the response
	res, err = mc.transport.ExecuteRequest(req)
	if err != nil {
		// map i/o timeouts to ErrRequestTimedOut
		if os.IsTimeout(err) {
			err = ErrRequestTimedOut
		}
		return
	}

	// make sure the source unit id matches that of the request
	if (res.functionCode&0x80) == 0x00 && res.unitID != req.unitID {
		err = ErrBadUnitID
		return
	}
	// accept errors from gateway devices (using special unit id #255)
	if (res.functionCode&0x80) == 0x80 &&
		(res.unitID != req.unitID && res.unitID != 0xff) {
		err = ErrBadUnitID
		return
	}

	return
}
