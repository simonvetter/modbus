package mbserver

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"
)

// Request object passed to the coil handler.
type CoilsRequest struct {
	WriteFuncCode uint8  // the function code of the write request
	ClientAddr    string // the source (client) IP address
	UnitId        uint8  // the requested unit id (slave id)
	Addr          uint16 // the base coil address requested
	Quantity      uint16 // the number of consecutive coils covered by this request
	// (first address: Addr, last address: Addr + Quantity - 1)
	IsWrite bool   // true if the request is a write, false if a read
	Args    []bool // a slice of bool values of the coils to be set, ordered
	// from Addr to Addr + Quantity - 1 (for writes only)
}

// Request object passed to the discrete input handler.
type DiscreteInputsRequest struct {
	ClientAddr string // the source (client) IP address
	UnitId     uint8  // the requested unit id (slave id)
	Addr       uint16 // the base discrete input address requested
	Quantity   uint16 // the number of consecutive discrete inputs covered by this request
}

// Request object passed to the holding register handler.
type HoldingRegistersRequest struct {
	WriteFuncCode uint8    // the function code of the write request
	ClientAddr    string   // the source (client) IP address
	UnitId        uint8    // the requested unit id (slave id)
	Addr          uint16   // the base register address requested
	Quantity      uint16   // the number of consecutive registers covered by this request
	IsWrite       bool     // true if the request is a write, false if a read
	Args          []uint16 // a slice of register values to be set, ordered from
	// Addr to Addr + Quantity - 1 (for writes only)
}

// Request object passed to the input register handler.
type InputRegistersRequest struct {
	ClientAddr string // the source (client) IP address
	UnitId     uint8  // the requested unit id (slave id)
	Addr       uint16 // the base register address requested
	Quantity   uint16 // the number of consecutive registers covered by this request
}

// The RequestHandler interface should be implemented by the handler
// object passed to NewServer (see reqHandler in NewServer()).
// After decoding and validating an incoming request, the server will
// invoke the appropriate handler function, depending on the function code
// of the request.
type RequestHandler interface {
	// HandleCoils handles the read coils (0x01), write single coil (0x05)
	// and write multiple coils (0x0f) function codes.
	// A CoilsRequest object is passed to the handler (see above).
	//
	// Expected return values:
	// - res:	a slice of bools containing the coil values to be sent to back
	//		to the client (only sent for reads),
	// - err:	either nil if no error occurred, a modbus error (see
	//		mapErrorToExceptionCode() in modbus.go for a complete list),
	//		or any other error.
	//		If nil, a positive modbus response is sent back to the client
	//		along with the returned data.
	//		If non-nil, a negative modbus response is sent back, with the
	//		exception code set depending on the error
	//		(again, see mapErrorToExceptionCode()).
	HandleCoils(*CoilsRequest) ([]bool, error)

	// HandleDiscreteInputs handles the read discrete inputs (0x02) function code.
	// A DiscreteInputsRequest oibject is passed to the handler (see above).
	//
	// Expected return values:
	// - res:	a slice of bools containing the discrete input values to be
	//		sent back to the client,
	// - err:	either nil if no error occurred, a modbus error (see
	//		mapErrorToExceptionCode() in modbus.go for a complete list),
	//		or any other error.
	HandleDiscreteInputs(*DiscreteInputsRequest) ([]bool, error)

	// HandleHoldingRegisters handles the read holding registers (0x03),
	// write single register (0x06) and write multiple registers (0x10).
	// A HoldingRegistersRequest object is passed to the handler (see above).
	//
	// Expected return values:
	// - res:	a slice of uint16 containing the register values to be sent
	//		to back to the client (only sent for reads),
	// - err:	either nil if no error occurred, a modbus error (see
	//		mapErrorToExceptionCode() in modbus.go for a complete list),
	//		or any other error.
	HandleHoldingRegisters(*HoldingRegistersRequest) ([]uint16, error)

	// HandleInputRegisters handles the read input registers (0x04) function code.
	// An InputRegistersRequest object is passed to the handler (see above).
	//
	// Expected return values:
	// - res:	a slice of uint16 containing the register values to be sent
	//		back to the client,
	// - err:	either nil if no error occurred, a modbus error (see
	//		mapErrorToExceptionCode() in modbus.go for a complete list),
	//		or any other error.
	HandleInputRegisters(*InputRegistersRequest) ([]uint16, error)
}

// Modbus server object.
type ModbusServer struct {
	// Timeout sets the idle session timeout (client connections will
	// be closed if idle for this long)
	Timeout time.Duration
	// MaxClients sets the maximum number of concurrent client connections
	MaxClients  uint
	logger      LeveledLogger
	lock        sync.Mutex
	handler     RequestHandler
	tcpListener net.Listener
	tcpClients  []net.Conn
}

type Option func(*ModbusServer) error

// Logger is the modbus server logger option
func Logger(logger LeveledLogger) func(*ModbusServer) error {
	return func(ms *ModbusServer) error {
		ms.logger = logger
		return nil
	}
}

// Timeout is the modbus server timeout option
func Timeout(timeout time.Duration) func(*ModbusServer) error {
	return func(ms *ModbusServer) error {
		ms.Timeout = timeout
		return nil
	}
}

// MaxClients is the modbus server maximum concurrent clients option
func MaxClients(max uint) func(*ModbusServer) error {
	return func(ms *ModbusServer) error {
		ms.MaxClients = max
		return nil
	}
}

// Returns a new modbus server.
// reqHandler should be a user-provided handler object satisfying the RequestHandler
// interface.
func New(reqHandler RequestHandler, opts ...Option) (*ModbusServer, error) {
	ms := &ModbusServer{
		Timeout: 30 * time.Second,
		handler: reqHandler,
		logger:  newLogger("modbus-server"),
	}

	for _, o := range opts {
		if err := o(ms); err != nil {
			return ms, err
		}
	}

	return ms, nil
}

// Starts accepting client connections.
func (ms *ModbusServer) Start(l net.Listener) error {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	if ms.tcpListener != nil {
		return errors.New("already started")
	}
	ms.tcpListener = l

	go ms.acceptTCPClients()

	return nil
}

// Stops accepting new client connections and closes any active session.
func (ms *ModbusServer) Stop() (err error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	if ms.tcpListener == nil {
		return errors.New("not started")
	}

	// close the server socket if we're listening over TCP
	err = ms.tcpListener.Close()

	// close all active TCP clients
	for _, sock := range ms.tcpClients {
		sock.Close()
	}

	ms.tcpListener = nil

	return
}

// Accepts new client connections if the configured connection limit allows it.
// Each connection is served from a dedicated goroutine to allow for concurrent
// connections.
func (ms *ModbusServer) acceptTCPClients() {
	var sock net.Conn
	var err error
	var accepted bool

	for {
		sock, err = ms.tcpListener.Accept()
		if err != nil {
			// if the server has just been stopped, return here
			ms.lock.Lock()
			if ms.tcpListener == nil {
				ms.lock.Unlock()
				return
			}
			ms.lock.Unlock()
			ms.logger.Warningf("failed to accept client connection: %v", err)
			continue
		}

		ms.lock.Lock()
		// apply a connection limit
		if ms.MaxClients == 0 || uint(len(ms.tcpClients)) < ms.MaxClients {
			accepted = true
			// add the new client connection to the pool
			ms.tcpClients = append(ms.tcpClients, sock)
		} else {
			accepted = false
		}
		ms.lock.Unlock()

		if accepted {
			// spin a client handler goroutine to serve the new client
			go ms.handleTCPClient(sock)
		} else {
			ms.logger.Warningf("max. number of concurrent connections reached, rejecting %v", sock.RemoteAddr())
			// discard the connection
			sock.Close()
		}
	}
}

// Handles a TCP client connection.
// Once handleTransport() returns (i.e. the connection has either closed, timed
// out, or an unrecoverable error happened), the TCP socket is closed and removed
// from the list of active client connections.
func (ms *ModbusServer) handleTCPClient(sock net.Conn) {
	ms.handleTransport(newTCPTransport(sock, ms.Timeout), sock.RemoteAddr().String())

	// once done, remove our connection from the list of active client conns
	ms.lock.Lock()
	for i := range ms.tcpClients {
		if ms.tcpClients[i] == sock {
			ms.tcpClients[i] = ms.tcpClients[len(ms.tcpClients)-1]
			ms.tcpClients = ms.tcpClients[:len(ms.tcpClients)-1]
			break
		}
	}
	ms.lock.Unlock()

	// close the connection
	sock.Close()
}

// For each request read from the transport, performs decoding and validation,
// calls the user-provided handler, then encodes and writes the response
// to the transport.
func (ms *ModbusServer) handleTransport(t transport, clientAddr string) {
	for {
		req, err := t.ReadRequest()
		if err != nil {
			return
		}

		var res *pdu

		switch req.functionCode {
		case fcReadCoils, fcReadDiscreteInputs:
			var coils []bool
			var resCount int

			if len(req.payload) != 4 {
				err = ErrProtocolError
				break
			}

			// decode address and quantity fields
			addr := binary.BigEndian.Uint16(req.payload[0:2])
			quantity := binary.BigEndian.Uint16(req.payload[2:4])

			// ensure the reply never exceeds the maximum PDU length and we
			// never read past 0xffff
			if quantity > 2000 || quantity == 0 {
				err = ErrProtocolError
				break
			}
			if uint32(addr)+uint32(quantity)-1 > 0xffff {
				err = ErrIllegalDataAddress
				break
			}

			// invoke the appropriate handler
			if req.functionCode == fcReadCoils {
				coils, err = ms.handler.HandleCoils(&CoilsRequest{
					ClientAddr: clientAddr,
					UnitId:     req.unitId,
					Addr:       addr,
					Quantity:   quantity,
					IsWrite:    false,
					Args:       nil,
				})
			} else {
				coils, err = ms.handler.HandleDiscreteInputs(
					&DiscreteInputsRequest{
						ClientAddr: clientAddr,
						UnitId:     req.unitId,
						Addr:       addr,
						Quantity:   quantity,
					})
			}
			resCount = len(coils)

			// make sure the handler returned the expected number of items
			if err == nil && resCount != int(quantity) {
				ms.logger.Errorf("handler returned %v bools, expected %v", resCount, quantity)
				err = ErrServerDeviceFailure
				break
			}

			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:       req.unitId,
				functionCode: req.functionCode,
				payload:      []byte{0},
			}

			// byte count (1 byte for 8 coils)
			res.payload[0] = uint8(resCount / 8)
			if resCount%8 != 0 {
				res.payload[0]++
			}

			// coil values
			res.payload = append(res.payload, encodeBools(coils)...)

		case fcWriteSingleCoil:
			if len(req.payload) != 4 {
				err = ErrProtocolError
				break
			}

			// decode the address field
			addr := binary.BigEndian.Uint16(req.payload[0:2])

			// validate the value field (should be either 0xff00 or 0x0000)
			if (req.payload[2] != 0xff && req.payload[2] != 0x00) ||
				req.payload[3] != 0x00 {
				err = ErrProtocolError
				break
			}

			// invoke the coil handler
			_, err = ms.handler.HandleCoils(&CoilsRequest{
				WriteFuncCode: fcWriteSingleCoil,
				ClientAddr:    clientAddr,
				UnitId:        req.unitId,
				Addr:          addr,
				Quantity:      1,    // request for a single coil
				IsWrite:       true, // this is a write request
				Args:          []bool{(req.payload[2] == 0xff)},
			})

			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:       req.unitId,
				functionCode: req.functionCode,
			}

			// echo the address and value in the response
			res.payload = append(res.payload, asBytes(binary.BigEndian, addr)...)
			res.payload = append(res.payload, req.payload[2], req.payload[3])

		case fcWriteMultipleCoils:
			var expectedLen int

			if len(req.payload) < 6 {
				err = ErrProtocolError
				break
			}

			// decode address and quantity fields
			addr := binary.BigEndian.Uint16(req.payload[0:2])
			quantity := binary.BigEndian.Uint16(req.payload[2:4])

			// ensure the reply never exceeds the maximum PDU length and we
			// never read past 0xffff
			if quantity > 0x7b0 || quantity == 0 {
				err = ErrProtocolError
				break
			}
			if uint32(addr)+uint32(quantity)-1 > 0xffff {
				err = ErrIllegalDataAddress
				break
			}

			// validate the byte count field (1 byte for 8 coils)
			expectedLen = int(quantity) / 8
			if quantity%8 != 0 {
				expectedLen++
			}

			if req.payload[4] != uint8(expectedLen) {
				err = ErrProtocolError
				break
			}

			// make sure we have enough bytes
			if len(req.payload)-5 != expectedLen {
				err = ErrProtocolError
				break
			}

			// invoke the coil handler
			_, err = ms.handler.HandleCoils(&CoilsRequest{
				WriteFuncCode: fcWriteMultipleCoils,
				ClientAddr:    clientAddr,
				UnitId:        req.unitId,
				Addr:          addr,
				Quantity:      quantity,
				IsWrite:       true, // this is a write request
				Args:          decodeBools(quantity, req.payload[5:]),
			})

			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:       req.unitId,
				functionCode: req.functionCode,
			}

			// echo the address and quantity in the response
			res.payload = append(res.payload, asBytes(binary.BigEndian, addr)...)
			res.payload = append(res.payload, asBytes(binary.BigEndian, quantity)...)

		case fcReadHoldingRegisters, fcReadInputRegisters:
			var regs []uint16
			var resCount int

			if len(req.payload) != 4 {
				err = ErrProtocolError
				break
			}

			// decode address and quantity fields
			addr := binary.BigEndian.Uint16(req.payload[0:2])
			quantity := binary.BigEndian.Uint16(req.payload[2:4])

			// ensure the reply never exceeds the maximum PDU length and we
			// never read past 0xffff
			if quantity > 0x007d || quantity == 0 {
				err = ErrProtocolError
				break
			}
			if uint32(addr)+uint32(quantity)-1 > 0xffff {
				err = ErrIllegalDataAddress
				break
			}

			// invoke the appropriate handler
			if req.functionCode == fcReadHoldingRegisters {
				regs, err = ms.handler.HandleHoldingRegisters(
					&HoldingRegistersRequest{
						ClientAddr: clientAddr,
						UnitId:     req.unitId,
						Addr:       addr,
						Quantity:   quantity,
						IsWrite:    false,
						Args:       nil,
					})
			} else {
				regs, err = ms.handler.HandleInputRegisters(
					&InputRegistersRequest{
						ClientAddr: clientAddr,
						UnitId:     req.unitId,
						Addr:       addr,
						Quantity:   quantity,
					})
			}
			resCount = len(regs)

			// make sure the handler returned the expected number of items
			if err == nil && resCount != int(quantity) {
				ms.logger.Errorf("handler returned %v 16-bit values, expected %v", resCount, quantity)
				err = ErrServerDeviceFailure
				break
			}

			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:       req.unitId,
				functionCode: req.functionCode,
				payload:      []byte{0},
			}

			// byte count (2 bytes per register)
			res.payload[0] = uint8(resCount * 2)

			// register values
			res.payload = append(res.payload, uint16ToBytes(binary.BigEndian, regs)...)

		case fcWriteSingleRegister:
			if len(req.payload) != 4 {
				err = ErrProtocolError
				break
			}

			// decode address and value fields
			addr := binary.BigEndian.Uint16(req.payload[0:2])
			value := binary.BigEndian.Uint16(req.payload[2:4])

			// invoke the handler
			_, err = ms.handler.HandleHoldingRegisters(
				&HoldingRegistersRequest{
					WriteFuncCode: fcWriteSingleRegister,
					ClientAddr:    clientAddr,
					UnitId:        req.unitId,
					Addr:          addr,
					Quantity:      1,    // request for a single register
					IsWrite:       true, // request is a write
					Args:          []uint16{value},
				})

			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:       req.unitId,
				functionCode: req.functionCode,
			}

			// echo the address and value in the response
			res.payload = append(res.payload, asBytes(binary.BigEndian, addr)...)
			res.payload = append(res.payload, asBytes(binary.BigEndian, value)...)

		case fcWriteMultipleRegisters:
			if len(req.payload) < 6 {
				err = ErrProtocolError
				break
			}

			// decode address and quantity fields
			addr := binary.BigEndian.Uint16(req.payload[0:2])
			quantity := binary.BigEndian.Uint16(req.payload[2:4])

			// ensure the reply never exceeds the maximum PDU length and we
			// never read past 0xffff
			if quantity > 0x007b || quantity == 0 {
				err = ErrProtocolError
				break
			}
			if uint32(addr)+uint32(quantity)-1 > 0xffff {
				err = ErrIllegalDataAddress
				break
			}

			// validate the byte count field (2 bytes per register)
			expectedLen := int(quantity) * 2

			if req.payload[4] != uint8(expectedLen) {
				err = ErrProtocolError
				break
			}

			// make sure we have enough bytes
			if len(req.payload)-5 != expectedLen {
				err = ErrProtocolError
				break
			}

			// invoke the holding register handler
			_, err = ms.handler.HandleHoldingRegisters(
				&HoldingRegistersRequest{
					WriteFuncCode: fcWriteMultipleRegisters,
					ClientAddr:    clientAddr,
					UnitId:        req.unitId,
					Addr:          addr,
					Quantity:      quantity,
					IsWrite:       true, // this is a write request
					Args:          bytesToUint16(binary.BigEndian, req.payload[5:]),
				})
			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:       req.unitId,
				functionCode: req.functionCode,
			}

			// echo the address and quantity in the response
			res.payload = append(res.payload, asBytes(binary.BigEndian, addr)...)
			res.payload = append(res.payload, asBytes(binary.BigEndian, quantity)...)

		default:
			res = &pdu{
				// reply with the request target unit ID
				unitId: req.unitId,
				// set the error bit
				functionCode: (0x80 | req.functionCode),
				// set the exception code to illegal function to indicate that
				// the server does not know how to handle this function code.
				payload: []byte{exIllegalFunction},
			}
		}

		// if there was no error processing the request but the response is nil
		// (which should never happen), emit a server failure exception code
		// and log an error
		if err == nil && res == nil {
			err = ErrServerDeviceFailure
			ms.logger.Errorf("internal server error (req: %v, res: %v, err: %v)", req, res, err)
		}

		// map go errors to modbus errors, unless the error is a protocol error,
		// in which case close the transport and return.
		if err != nil {
			if err == ErrProtocolError {
				ms.logger.Warningf("protocol error, closing link (client address: '%s')", clientAddr)
				t.Close()
				return
			}

			res = &pdu{
				unitId:       req.unitId,
				functionCode: (0x80 | req.functionCode),
				payload:      []byte{mapErrorToExceptionCode(err)},
			}
		}

		// write the response to the transport
		if err := t.WriteResponse(res); err != nil {
			ms.logger.Warningf("failed to write response: %v", err)
		}
	}
}
