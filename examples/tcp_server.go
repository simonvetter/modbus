package main

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/simonvetter/modbus"
)

const (
	MINUS_ONE int16 = -1
)

/*
* Simple modbus server example.
*
* This file is intended to be a demo of the modbus server.
* It shows how to create and start a server, as well as how
* to write a handler object.
* Feel free to use it as boilerplate for simple servers.
 */

// run this with go run examples/tcp_server.go
func main() {
	var server *modbus.ModbusServer
	var err error
	var eh *exampleHandler
	var ticker *time.Ticker

	// create the handler object
	eh = &exampleHandler{}

	// create the server object
	server, err = modbus.NewServer(&modbus.ServerConfiguration{
		// listen on localhost port 5502
		URL: "tcp://localhost:5502",
		// close idle connections after 30s of inactivity
		Timeout: 30 * time.Second,
		// accept 5 concurrent connections max.
		MaxClients: 5,
	}, eh)
	if err != nil {
		fmt.Printf("failed to create server: %v\n", err)
		os.Exit(1)
	}

	// start accepting client connections
	// note that Start() returns as soon as the server is started
	err = server.Start()
	if err != nil {
		fmt.Printf("failed to start server: %v\n", err)
		os.Exit(1)
	}

	// increment a 32-bit uptime counter every second.
	// (this counter is exposed as input registers 200-201 for demo purposes)
	ticker = time.NewTicker(1 * time.Second)
	for {
		<-ticker.C

		// since the handler methods are called from multiple goroutines,
		// use locking where appropriate to avoid concurrency issues.
		eh.lock.Lock()
		eh.uptime++
		eh.lock.Unlock()
	}

	// never reached
	return
}

// Example handler object, passed to the NewServer() constructor above.
type exampleHandler struct {
	// this lock is used to avoid concurrency issues between goroutines, as
	// handler methods are called from different goroutines
	// (1 goroutine per client)
	lock sync.RWMutex

	// simple uptime counter, incremented in the main() above and exposed
	// as a 32-bit input register (2 consecutive 16-bit modbus registers).
	uptime uint32

	// these are here to hold client-provided (written) values, for both coils and
	// holding registers
	coils       [100]bool
	holdingReg1 uint16
	holdingReg2 uint16

	// this is a 16-bit signed integer
	holdingReg3 int16

	// this is a 32-bit unsigned integer
	holdingReg4 uint32
}

// Coil handler method.
// This method gets called whenever a valid modbus request asking for a coil operation is
// received by the server.
// It exposes 100 read/writable coils at addresses 0-99, except address 80 which is
// read-only.
// (read them with ./modbus-cli --target tcp://localhost:5502 rc:0+99, write to register n
// with ./modbus-cli --target tcp://localhost:5502 wr:n:<true|false>)
func (eh *exampleHandler) HandleCoils(req *modbus.CoilsRequest) (res []bool, err error) {
	if req.UnitId != 1 {
		// only accept unit ID #1
		// note: we're merely filtering here, but we could as well use the unit
		// ID field to support multiple register maps in a single server.
		err = modbus.ErrIllegalFunction
		return
	}

	// make sure that all registers covered by this request actually exist
	if int(req.Addr)+int(req.Quantity) > len(eh.coils) {
		err = modbus.ErrIllegalDataAddress
		return
	}

	// since we're manipulating variables shared between multiple goroutines,
	// acquire a lock to avoid concurrency issues.
	eh.lock.Lock()
	// release the lock upon return
	defer eh.lock.Unlock()

	// loop through `req.Quantity` registers, from address `req.Addr` to
	// `req.Addr + req.Quantity - 1`, which here is conveniently `req.Addr + i`
	for i := 0; i < int(req.Quantity); i++ {
		// ignore the write if the current register address is 80
		if req.IsWrite && int(req.Addr)+i != 80 {
			// assign the value
			eh.coils[int(req.Addr)+i] = req.Args[i]
		}
		// append the value of the requested register to res so they can be
		// sent back to the client
		res = append(res, eh.coils[int(req.Addr)+i])
	}

	return
}

// Discrete input handler method.
// Note that we're returning ErrIllegalFunction unconditionally.
// This will cause the client to receive "illegal function", which is the modbus way of
// reporting that this server does not support/implement the discrete input type.
func (eh *exampleHandler) HandleDiscreteInputs(req *modbus.DiscreteInputsRequest) (res []bool, err error) {
	// this is the equivalent of saying
	// "discrete inputs are not supported by this device"
	// (try it with modbus-cli --target tcp://localhost:5502 rdi:1)
	err = modbus.ErrIllegalFunction

	return
}

// Holding register handler method.
// This method gets called whenever a valid modbus request asking for a holding register
// operation (either read or write) received by the server.
func (eh *exampleHandler) HandleHoldingRegisters(req *modbus.HoldingRegistersRequest) (res []uint16, err error) {
	var regAddr uint16

	if req.UnitId != 1 {
		// only accept unit ID #1
		err = modbus.ErrIllegalFunction
		return
	}

	// since we're manipulating variables shared between multiple goroutines,
	// acquire a lock to avoid concurrency issues.
	eh.lock.Lock()
	// release the lock upon return
	defer eh.lock.Unlock()

	// loop through `quantity` registers
	for i := 0; i < int(req.Quantity); i++ {
		// compute the target register address
		regAddr = req.Addr + uint16(i)

		switch regAddr {
		// expose the static, read-only value of 0xff00 in register 100
		case 100:
			res = append(res, 0xff00)

		// expose holdingReg1 in register 101 (RW)
		case 101:
			if req.IsWrite {
				eh.holdingReg1 = req.Args[i]
			}
			res = append(res, eh.holdingReg1)

		// expose holdingReg2 in register 102 (RW)
		case 102:
			if req.IsWrite {
				// only accept values 2 and 4
				switch req.Args[i] {
				case 2, 4:
					eh.holdingReg2 = req.Args[i]

					// make note of the change (e.g. for auditing purposes)
					fmt.Printf("%s set reg#102 to %v\n", req.ClientAddr, eh.holdingReg2)
				default:
					// if the written value is neither 2 nor 4,
					// return a modbus "illegal data value" to
					// let the client know that the value is
					// not acceptable.
					err = modbus.ErrIllegalDataValue
					return
				}
			}
			res = append(res, eh.holdingReg2)

		// expose eh.holdingReg3 in register 103 (RW)
		// note: eh.holdingReg3 is a signed 16-bit integer
		case 103:
			if req.IsWrite {
				// cast the 16-bit unsigned integer passed by the server
				// to a 16-bit signed integer when writing
				eh.holdingReg3 = int16(req.Args[i])
			}
			// cast the 16-bit signed integer from the handler to a 16-bit unsigned
			// integer so that we can append it to `res`.
			res = append(res, uint16(eh.holdingReg3))

		// expose the 16 most-significant bits of eh.holdingReg4 in register 200
		case 200:
			if req.IsWrite {
				eh.holdingReg4 =
					((uint32(req.Args[i])<<16)&0xffff0000 |
						(eh.holdingReg4 & 0x0000ffff))
			}
			res = append(res, uint16((eh.holdingReg4>>16)&0x0000ffff))

		// expose the 16 least-significant bits of eh.holdingReg4 in register 201
		case 201:
			if req.IsWrite {
				eh.holdingReg4 =
					(uint32(req.Args[i])&0x0000ffff |
						(eh.holdingReg4 & 0xffff0000))
			}
			res = append(res, uint16(eh.holdingReg4&0x0000ffff))

		// any other address is unknown
		default:
			err = modbus.ErrIllegalDataAddress
			return
		}
	}

	return
}

// Input register handler method.
// This method gets called whenever a valid modbus request asking for an input register
// operation is received by the server.
// Note that input registers are always read-only as per the modbus spec.
func (eh *exampleHandler) HandleInputRegisters(req *modbus.InputRegistersRequest) (res []uint16, err error) {
	var unixTs_s uint32
	var minusOne int16 = -1

	if req.UnitId != 1 {
		// only accept unit ID #1
		err = modbus.ErrIllegalFunction
		return
	}

	// get the current unix timestamp, converted as a 32-bit unsigned integer for
	// simplicity
	unixTs_s = uint32(time.Now().Unix() & 0xffffffff)

	// loop through all register addresses from req.addr to req.addr + req.Quantity - 1
	for regAddr := req.Addr; regAddr < req.Addr+req.Quantity; regAddr++ {
		switch regAddr {
		case 100:
			// return the static value 0x1111 at address 100, as an unsigned
			// 16-bit integer
			// (read it with modbus-cli --target tcp://localhost:5502 ri:uint16:100)
			res = append(res, 0x1111)

		case 101:
			// return the static value -1 at address 101, as a signed 16-bit
			// integer
			// (read it with modbus-cli --target tcp://localhost:5502 ri:int16:101)
			res = append(res, uint16(minusOne))

		// expose our uptime counter, encoded as a 32-bit unsigned integer in
		// input registers 200-201
		// (read it with modbus-cli --target tcp://localhost:5502 ri:uint32:200)
		case 200:
			// return the 16 most significant bits of the uptime counter
			// (using locking to avoid concurrency issues)
			eh.lock.RLock()
			res = append(res, uint16((eh.uptime>>16)&0xffff))
			eh.lock.RUnlock()

		case 201:
			// return the 16 least significant bits of the uptime counter
			// (again, using locking to avoid concurrency issues)
			eh.lock.RLock()
			res = append(res, uint16(eh.uptime&0xffff))
			eh.lock.RUnlock()

		// expose the current unix timestamp, encoded as a 32-bit unsigned integer
		// in input registers 202-203
		// (read it with modbus-cli --target tcp://localhost:5502 ri:uint32:202)
		case 202:
			// return the 16 most significant bits of the current unix time
			res = append(res, uint16((unixTs_s>>16)&0xffff))

		case 203:
			// return the 16 least significant bits of the current unix time
			res = append(res, uint16(unixTs_s&0xffff))

		// return 3.1415, encoded as a 32-bit floating point number in input
		// registers 300-301
		// (read it with modbus-cli --target tcp://localhost:5502 ri:float32:300)
		case 300:
			// returh the 16 most significant bits of the number
			res = append(res, uint16((math.Float32bits(3.1415)>>16)&0xffff))

		case 301:
			// returh the 16 least significant bits of the number
			res = append(res, uint16((math.Float32bits(3.1415))&0xffff))

		// attempting to access any input register address other than
		// those defined above will result in an illegal data address
		// exception client-side.
		default:
			err = modbus.ErrIllegalDataAddress
			return
		}
	}

	return
}
