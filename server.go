package modbus

import (
	"fmt"
	"time"
	"net"
	"strings"
	"sync"
)

// Server configuration object.
type ServerConfiguration struct {
	URL		string		// where to listen at e.g. tcp://[::]:502
	Timeout		time.Duration	// idle session timeout (client connection will be
					// closed if idle for this long)
	MaxClients	uint		// maximum number of concurrent client connections
}

// Request object passed to the coil handler.
type CoilsRequest struct {
	ClientAddr	string	// the source (client) IP address
	UnitId		uint8	// the requested unit id (slave id)
	Addr		uint16	// the base coil address requested
	Quantity	uint16	// the number of consecutive coils covered by this request
				// (first address: Addr, last address: Addr + Quantity - 1)
	IsWrite		bool	// true if the request is a write, false if a read
	Args		[]bool	// a slice of bool values of the coils to be set, ordered
				// from Addr to Addr + Quantity - 1 (for writes only)
}

// Request object passed to the discrete input handler.
type DiscreteInputsRequest struct {
	ClientAddr	string	// the source (client) IP address
	UnitId		uint8	// the requested unit id (slave id)
	Addr		uint16	// the base discrete input address requested
	Quantity	uint16	// the number of consecutive discrete inputs
				// covered by this request
}

// Request object passed to the holding register handler.
type HoldingRegistersRequest struct {
	ClientAddr	string	// the source (client) IP address
	UnitId		uint8	// the requested unit id (slave id)
	Addr		uint16	// the base register address requested
	Quantity	uint16	// the number of consecutive registers covered by this request
	IsWrite		bool	// true if the request is a write, false if a read
	Args		[]uint16 // a slice of register values to be set, ordered from
				 // Addr to Addr + Quantity - 1 (for writes only)
}

// Request object passed to the input register handler.
type InputRegistersRequest struct {
	ClientAddr	string	// the source (client) IP address
	UnitId		uint8	// the requested unit id (slave id)
	Addr		uint16	// the base register address requested
	Quantity	uint16	// the number of consecutive registers covered by this request
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
	HandleCoils	(req *CoilsRequest) (res []bool, err error)

	// HandleDiscreteInputs handles the read discrete inputs (0x02) function code.
	// A DiscreteInputsRequest oibject is passed to the handler (see above).
	//
	// Expected return values:
	// - res:	a slice of bools containing the discrete input values to be
	//		sent back to the client,
	// - err:	either nil if no error occurred, a modbus error (see
	//		mapErrorToExceptionCode() in modbus.go for a complete list),
	//		or any other error.
	HandleDiscreteInputs	(req *DiscreteInputsRequest) (res []bool, err error)

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
	HandleHoldingRegisters	(req *HoldingRegistersRequest) (res []uint16, err error)

	// HandleInputRegisters handles the read input registers (0x04) function code.
	// An InputRegistersRequest object is passed to the handler (see above).
	//
	// Expected return values:
	// - res:	a slice of uint16 containing the register values to be sent
	//		back to the client,
	// - err:	either nil if no error occurred, a modbus error (see
	//		mapErrorToExceptionCode() in modbus.go for a complete list),
	//		or any other error.
	HandleInputRegisters	(req *InputRegistersRequest) (res []uint16, err error)
}

// Modbus server object.
type ModbusServer struct {
	conf		ServerConfiguration
	logger		*logger
	lock		sync.Mutex
	started		bool
	handler		RequestHandler
	tcpListener	net.Listener
	tcpClients	[]net.Conn
	transportType	transportType
}

// Returns a new modbus server.
// reqHandler should be a user-provided handler object satisfying the RequestHandler
// interface.
func NewServer(conf *ServerConfiguration, reqHandler RequestHandler) (
	ms *ModbusServer, err error) {

	ms = &ModbusServer{
		conf:		*conf,
		handler:	reqHandler,
		logger:		newLogger("modbus-server"),
	}

	switch {
	case strings.HasPrefix(ms.conf.URL, "tcp://"):
		ms.conf.URL	= strings.TrimPrefix(ms.conf.URL, "tcp://")

		if ms.conf.Timeout == 0 {
			ms.conf.Timeout = 120 * time.Second
		}

		if ms.conf.MaxClients == 0 {
			ms.conf.MaxClients = 10
		}

		ms.transportType	= modbusTCP

	default:
		err	= ErrConfigurationError
		return
	}

	ms.logger	= newLogger(fmt.Sprintf("modbus-server(%s)", ms.conf.URL))

	return
}

// Starts accepting client connections.
func (ms *ModbusServer) Start() (err error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	if ms.started {
		return
	}

	switch ms.transportType {
	case modbusTCP:
		// bind to a TCP socket
		ms.tcpListener, err	= net.Listen("tcp", ms.conf.URL)
		if err != nil {
			return
		}

		// accept client connections in a goroutine
		go ms.acceptTCPClients()

	default:
		err = ErrConfigurationError
		return
	}

	ms.started = true

	return
}

// Stops accepting new client connections and closes any active session.
func (ms *ModbusServer) Stop() (err error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	if !ms.started {
		return
	}

	ms.started = false

	if ms.transportType == modbusTCP {
		// close the server socket if we're listening over TCP
		err	= ms.tcpListener.Close()

		// close all active TCP clients
		for _, sock := range ms.tcpClients{
			sock.Close()
		}
	}

	return
}

// Accepts new client connections if the configured connection limit allows it.
// Each connection is served from a dedicated goroutine to allow for concurrent
// connections.
func (ms *ModbusServer) acceptTCPClients() {
	var sock	net.Conn
	var err		error
	var accepted	bool

	for {
		sock, err = ms.tcpListener.Accept()
		if err != nil {
			// if the server has just been stopped, return here
			if !ms.started {
				return
			}
			ms.logger.Warningf("failed to accept client connection: %v", err)
			continue
		}

		ms.lock.Lock()
		// apply a connection limit
		if uint(len(ms.tcpClients)) < ms.conf.MaxClients {
			accepted	= true
			// add the new client connection to the pool
			ms.tcpClients	= append(ms.tcpClients, sock)
		} else {
			accepted	= false
		}
		ms.lock.Unlock()

		if accepted {
			// spin a client handler goroutine to serve the new client
			go ms.handleTCPClient(sock)
		} else {
			ms.logger.Warningf("max. number of concurrent connections " +
					   "reached, rejecting %v", sock.RemoteAddr())
			// discard the connection
			sock.Close()
		}
	}

	// never reached
	return
}

// Handles a TCP client connection.
// Once handleTransport() returns (i.e. the connection has either closed, timed
// out, or an unrecoverable error happened), the TCP socket is closed and removed
// from the list of active client connections.
func (ms *ModbusServer) handleTCPClient(sock net.Conn) {
	var tt	*tcpTransport

	// create a new transport
	tt = newTCPTransport(sock, ms.conf.Timeout)

	ms.handleTransport(tt, sock.RemoteAddr().String())

	// once done, remove our connection from the list of active client conns
	ms.lock.Lock()
	for i := range ms.tcpClients {
		if ms.tcpClients[i] == sock {
			ms.tcpClients[i] = ms.tcpClients[len(ms.tcpClients)-1]
			ms.tcpClients	 = ms.tcpClients[:len(ms.tcpClients)-1]
			break
		}
	}
	ms.lock.Unlock()

	// close the connection
	sock.Close()

	return
}

// For each request read from the transport, performs decoding and validation,
// calls the user-provided handler, then encodes and writes the response
// to the transport.
func (ms *ModbusServer) handleTransport(t transport, clientAddr string) {
	var req		*pdu
	var res		*pdu
	var err		error
	var addr	uint16
	var quantity	uint16

	for {
		req, err = t.ReadRequest()
		if err != nil {
			return
		}

		switch req.functionCode {
		case fcReadCoils, fcReadDiscreteInputs:
			var coils	[]bool
			var resCount	int

			if len(req.payload) != 4 {
				err = ErrProtocolError
				break
			}

			// decode address and quantity fields
			addr		= bytesToUint16(BIG_ENDIAN, req.payload[0:2])
			quantity	= bytesToUint16(BIG_ENDIAN, req.payload[2:4])

			// ensure the reply never exceeds the maximum PDU length and we
			// never read past 0xffff
			if quantity > 2000 || quantity == 0 {
				err	= ErrProtocolError
				break
			}
			if uint32(addr) + uint32(quantity) - 1 > 0xffff {
				err	= ErrIllegalDataAddress
				break
			}

			// invoke the appropriate handler
			if req.functionCode == fcReadCoils {
				coils, err	= ms.handler.HandleCoils(&CoilsRequest{
					ClientAddr:	clientAddr,
					UnitId:		req.unitId,
					Addr:		addr,
					Quantity:	quantity,
					IsWrite:	false,
					Args:		nil,
				})
			} else {
				coils, err	= ms.handler.HandleDiscreteInputs(
					&DiscreteInputsRequest{
						ClientAddr:	clientAddr,
						UnitId:		req.unitId,
						Addr:		addr,
						Quantity:	quantity,
					})
			}
			resCount	= len(coils)

			// make sure the handler returned the expected number of items
			if err == nil && resCount != int(quantity) {
				ms.logger.Errorf("handler returned %v bools, " +
					         "expected %v", resCount, quantity)
				err = ErrServerDeviceFailure
				break
			}

			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:		req.unitId,
				functionCode:	req.functionCode,
				payload:	[]byte{0},
			}

			// byte count (1 byte for 8 coils)
			res.payload[0]	= uint8(resCount / 8)
			if resCount % 8 != 0 {
				res.payload[0]++
			}

			// coil values
			res.payload	= append(res.payload, encodeBools(coils)...)

		case fcWriteSingleCoil:
			if len(req.payload) != 4 {
				err = ErrProtocolError
				break
			}

			// decode the address field
			addr	= bytesToUint16(BIG_ENDIAN, req.payload[0:2])

			// validate the value field (should be either 0xff00 or 0x0000)
			if ((req.payload[2] != 0xff && req.payload[2] != 0x00) ||
			    req.payload[3] != 0x00) {
				err = ErrProtocolError
				break
			}

			// invoke the coil handler
			_, err	= ms.handler.HandleCoils(&CoilsRequest{
				ClientAddr:	clientAddr,
				UnitId:		req.unitId,
				Addr:		addr,
				Quantity:	1, // request for a single coil
				IsWrite:	true, // this is a write request
				Args:		[]bool{(req.payload[2] == 0xff)},
			})

			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:		req.unitId,
				functionCode:	req.functionCode,
			}

			// echo the address and value in the response
			res.payload	= append(res.payload,
						 uint16ToBytes(BIG_ENDIAN, addr)...)
			res.payload	= append(res.payload,
						 req.payload[2], req.payload[3])

		case fcWriteMultipleCoils:
			var expectedLen	int

			if len(req.payload) < 6 {
				err = ErrProtocolError
				break
			}

			// decode address and quantity fields
			addr		= bytesToUint16(BIG_ENDIAN, req.payload[0:2])
			quantity	= bytesToUint16(BIG_ENDIAN, req.payload[2:4])

			// ensure the reply never exceeds the maximum PDU length and we
			// never read past 0xffff
			if quantity > 0x7b0 || quantity == 0 {
				err	= ErrProtocolError
				break
			}
			if uint32(addr) + uint32(quantity) - 1 > 0xffff {
				err	= ErrIllegalDataAddress
				break
			}

			// validate the byte count field (1 byte for 8 coils)
			expectedLen	= int(quantity) / 8
			if quantity % 8 != 0 {
				expectedLen++
			}

			if req.payload[4] != uint8(expectedLen) {
				err	= ErrProtocolError
				break
			}

			// make sure we have enough bytes
			if len(req.payload) - 5 != expectedLen {
				err	= ErrProtocolError
				break
			}

			// invoke the coil handler
			_, err	= ms.handler.HandleCoils(&CoilsRequest{
				ClientAddr:	clientAddr,
				UnitId:		req.unitId,
				Addr:		addr,
				Quantity:	quantity,
				IsWrite:	true, // this is a write request
				Args:		decodeBools(quantity, req.payload[5:]),
			})

			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:		req.unitId,
				functionCode:	req.functionCode,
			}

			// echo the address and quantity in the response
			res.payload	= append(res.payload,
						 uint16ToBytes(BIG_ENDIAN, addr)...)
			res.payload	= append(res.payload,
						 uint16ToBytes(BIG_ENDIAN, quantity)...)

		case fcReadHoldingRegisters, fcReadInputRegisters:
			var regs	[]uint16
			var resCount	int

			if len(req.payload) != 4 {
				err = ErrProtocolError
				break
			}

			// decode address and quantity fields
			addr		= bytesToUint16(BIG_ENDIAN, req.payload[0:2])
			quantity	= bytesToUint16(BIG_ENDIAN, req.payload[2:4])

			// ensure the reply never exceeds the maximum PDU length and we
			// never read past 0xffff
			if quantity > 0x007d || quantity == 0 {
				err	= ErrProtocolError
				break
			}
			if uint32(addr) + uint32(quantity) - 1 > 0xffff {
				err	= ErrIllegalDataAddress
				break
			}

			// invoke the appropriate handler
			if req.functionCode == fcReadHoldingRegisters {
				regs, err	= ms.handler.HandleHoldingRegisters(
					&HoldingRegistersRequest{
						ClientAddr:	clientAddr,
						UnitId:		req.unitId,
						Addr:		addr,
						Quantity:	quantity,
						IsWrite:	false,
						Args:		nil,
					})
			} else {
				regs, err	= ms.handler.HandleInputRegisters(
					&InputRegistersRequest{
						ClientAddr:	clientAddr,
						UnitId:		req.unitId,
						Addr:		addr,
						Quantity:	quantity,
					})
			}
			resCount	= len(regs)

			// make sure the handler returned the expected number of items
			if err == nil && resCount != int(quantity) {
				ms.logger.Errorf("handler returned %v 16-bit values, " +
					         "expected %v", resCount, quantity)
				err = ErrServerDeviceFailure
				break
			}

			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:		req.unitId,
				functionCode:	req.functionCode,
				payload:	[]byte{0},
			}

			// byte count (2 bytes per register)
			res.payload[0]	= uint8(resCount * 2)

			// register values
			res.payload	= append(res.payload,
						 uint16sToBytes(BIG_ENDIAN, regs)...)

		case fcWriteSingleRegister:
			var value	uint16

			if len(req.payload) != 4 {
				err = ErrProtocolError
				break
			}

			// decode address and value fields
			addr	= bytesToUint16(BIG_ENDIAN, req.payload[0:2])
			value	= bytesToUint16(BIG_ENDIAN, req.payload[2:4])

			// invoke the handler
			_, err	= ms.handler.HandleHoldingRegisters(
				&HoldingRegistersRequest{
					ClientAddr:	clientAddr,
					UnitId:		req.unitId,
					Addr:		addr,
					Quantity:	1, // request for a single register
					IsWrite:	true, // request is a write
					Args:		[]uint16{value},
				})

			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:		req.unitId,
				functionCode:	req.functionCode,
			}

			// echo the address and value in the response
			res.payload	= append(res.payload,
						 uint16ToBytes(BIG_ENDIAN, addr)...)
			res.payload	= append(res.payload,
						 uint16ToBytes(BIG_ENDIAN, value)...)

		case fcWriteMultipleRegisters:
			var expectedLen	int

			if len(req.payload) < 6 {
				err = ErrProtocolError
				break
			}

			// decode address and quantity fields
			addr		= bytesToUint16(BIG_ENDIAN, req.payload[0:2])
			quantity	= bytesToUint16(BIG_ENDIAN, req.payload[2:4])

			// ensure the reply never exceeds the maximum PDU length and we
			// never read past 0xffff
			if quantity > 0x007b || quantity == 0 {
				err	= ErrProtocolError
				break
			}
			if uint32(addr) + uint32(quantity) - 1 > 0xffff {
				err	= ErrIllegalDataAddress
				break
			}

			// validate the byte count field (2 bytes per register)
			expectedLen	= int(quantity) * 2

			if req.payload[4] != uint8(expectedLen) {
				err	= ErrProtocolError
				break
			}

			// make sure we have enough bytes
			if len(req.payload) - 5 != expectedLen {
				err	= ErrProtocolError
				break
			}

			// invoke the holding register handler
			_, err		= ms.handler.HandleHoldingRegisters(
				&HoldingRegistersRequest{
					ClientAddr:	clientAddr,
					UnitId:		req.unitId,
					Addr:		addr,
					Quantity:	quantity,
					IsWrite:	true, // this is a write request
					Args:		bytesToUint16s(BIG_ENDIAN, req.payload[5:]),
				})
			if err != nil {
				break
			}

			// assemble a response PDU
			res = &pdu{
				unitId:		req.unitId,
				functionCode:	req.functionCode,
			}

			// echo the address and quantity in the response
			res.payload	= append(res.payload,
						 uint16ToBytes(BIG_ENDIAN, addr)...)
			res.payload	= append(res.payload,
						 uint16ToBytes(BIG_ENDIAN, quantity)...)

		default:
			res = &pdu{
				// reply with the request target unit ID
				unitId:		req.unitId,
				// set the error bit
				functionCode:	(0x80 | req.functionCode),
				// set the exception code to illegal function to indicate that
				// the server does not know how to handle this function code.
				payload:	[]byte{exIllegalFunction},
			}
		}

		// if there was no error processing the request but the response is nil
		// (which should never happen), emit a server failure exception code
		// and log an error
		if err == nil && res == nil {
			err = ErrServerDeviceFailure
			ms.logger.Errorf("internal server error (req: %v, res: %v, err: %v)",
					 req, res, err)
		}

		// map go errors to modbus errors, unless the error is a protocol error,
		// in which case close the transport and return.
		if err != nil {
			if err == ErrProtocolError {
				ms.logger.Warningf(
					"protocol error, closing link (client address: '%s')",
					clientAddr)
				t.Close()
				return
			} else {
				res = &pdu{
					unitId:		req.unitId,
					functionCode:	(0x80 | req.functionCode),
					payload:	[]byte{mapErrorToExceptionCode(err)},
				}
			}
		}

		// write the response to the transport
		err	= t.WriteResponse(res)
		if err != nil {
			ms.logger.Warningf("failed to write response: %v", err)
		}

		// avoid holding on to stale data
		req	= nil
		res	= nil
	}

	// never reached
	return
}
