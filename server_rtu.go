package modbus

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/goburrow/serial"
)

type ModbusRtuServer struct {
	conf    *ModbusRtuServerConfig
	handler RequestHandler
	lock    sync.Mutex
	port    serial.Port
	started bool
	logger  *logger
}

type ModbusRtuServerConfig struct {
	// What is my modbus address?
	// I only listen to messages addressed to me...
	ModbusAddress uint8
	// Defines where to listen
	TTYPath string
	// Set Baudrate (default 19200)
	BaudRate uint
	// How many data bits? (default 8)
	DataBits uint
	// How many stop bits? (default 1)
	StopBits uint
	// Is parity bit used? [Y/N] (default 'N')
	Parity string
	// Logger provides a custom sink for log messages.
	// If nil, messages will be written to stdout.
	Logger *log.Logger
}

func NewRTUServer(config *ModbusRtuServerConfig, reqHandler RequestHandler) (
	ms *ModbusRtuServer, err error) {

	if config == nil || reqHandler == nil {
		err = fmt.Errorf("Config and request handler must not be nil!")
		return
	}
	if config.TTYPath == "" {
		err = fmt.Errorf("TTYPath must not be an empty string!")
		return
	}
	if config.ModbusAddress == 0 {
		err = fmt.Errorf("My modbus address must be specified!")
		return
	}
	if config.BaudRate == 0 {
		config.BaudRate = 19200
	}
	if config.DataBits == 0 {
		config.DataBits = 8
	}
	if config.StopBits == 0 {
		config.StopBits = 1
	}
	if config.Parity != "N" && config.Parity != "Y" {
		config.Parity = "N"
	}

	ms = &ModbusRtuServer{
		conf:    config,
		handler: reqHandler,
	}

	ms.logger = newLogger("Modbus RTU:", ms.conf.Logger)

	return
}

func (ms *ModbusRtuServer) Start() (err error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	if ms.started {
		return
	}

	var startBit uint = 1
	var parityBit uint = 0
	if ms.conf.Parity == "Y" {
		parityBit = 1
	}
	bitsPerSymbol := uint(startBit + ms.conf.DataBits + ms.conf.StopBits + parityBit)
	minPauseDuration := time.Duration((3.5*(1/float64(ms.conf.BaudRate/bitsPerSymbol)))*1e6) * time.Microsecond
	if ms.conf.BaudRate > 19200 {
		minPauseDuration = 1750 * time.Microsecond
	}
	ms.logger.Infof("Min pause time at %v baud with %v bits per symbol: %v", ms.conf.BaudRate, bitsPerSymbol, minPauseDuration)

	serialCfg := serial.Config{
		Address:  ms.conf.TTYPath,
		BaudRate: int(ms.conf.BaudRate),
		DataBits: int(ms.conf.DataBits),
		StopBits: int(ms.conf.StopBits),
		Parity:   ms.conf.Parity,
		Timeout:  minPauseDuration,
	}

	ms.logger.Infof("Connecting %v", serialCfg)
	ms.port, err = serial.Open(&serialCfg)
	if err != nil {
		ms.logger.Errorf("Connecting failed: %v", err)
		return
	}
	ms.logger.Infof("Connected!")
	ms.started = true

	go ms.listenAndServe()

	return
}

func (ms *ModbusRtuServer) Stop() (err error) {
	ms.lock.Lock()
	defer ms.lock.Unlock()

	if !ms.started {
		return
	}

	err = ms.port.Close()
	if err != nil {
		ms.logger.Errorf("Closing failed: %v", err)
		return
	}

	ms.logger.Infof("Closed!")
	ms.started = false
	return
}

func (ms *ModbusRtuServer) messageIsForMe(message []byte) (yes bool) {
	if message == nil {
		return false
	}
	return message[0] == ms.conf.ModbusAddress
}

func (ms *ModbusRtuServer) sendErrorMessage(originalMessage []byte, errorCode uint8) {
	if originalMessage == nil || !ms.started {
		return
	}

	var errorMsg = []byte{originalMessage[0], originalMessage[1] | 0x80, errorCode}
	var c crc
	c.init()
	c.add(errorMsg)
	errorMsg = append(errorMsg, c.value()...)

	ms.logger.Infof("Written %v bytes: 0x[% X]", len(errorMsg), errorMsg)
	if _, err := ms.port.Write(errorMsg); err != nil {
		ms.logger.Errorf("Send answer failed! (%v)", err)
	}

	return
}

func (ms *ModbusRtuServer) listenAndServe() {
	if !ms.started {
		return
	}
	ms.lock.Lock()
	defer ms.lock.Unlock()

	// Receive strategy is as follow:
	// Read single byte after byte. If pause time between two bytes is longer than minimal
	// pause time, the message is complete (according to Modbus RTU standard.)
	var receivedData []byte

	ms.logger.Infof("Start listening....")

	for {
		// Read one byte and append to buffer if we have received one.
		// Listen for the next byte immediately.
		buf := make([]byte, 1)
		n, _ := ms.port.Read(buf)
		if n > 0 {
			receivedData = append(receivedData, buf...)
			continue
		}

		// Continue receiving when we have nothing received yet...
		// Even if the timeout has expired.
		if len(receivedData) == 0 {
			continue
		}

		ms.logger.Infof("Received bytes: %v\n", receivedData)

		var raw any
		var err error
		var response any
		var bytesToSend []byte

		if err = messageIsValid(receivedData); err != nil {
			ms.logger.Warningf("Can't execute request! (%v)", err)
			receivedData = nil
			continue
		}

		if !ms.messageIsForMe(receivedData) {
			ms.logger.Infof("Ignoring message. Not for me...")
			receivedData = nil
			continue
		}

		if raw, err = createRequestFromBytes(receivedData); err != nil {
			ms.logger.Warningf("Can't execute request! (%v)", err)
			ms.sendErrorMessage(receivedData, exIllegalFunction)
			receivedData = nil
			continue
		}

		switch raw.(type) {
		case CoilsRequest:
			request := raw.(CoilsRequest)
			response, err = ms.handler.HandleCoils(&request)
		case DiscreteInputsRequest:
			request := raw.(DiscreteInputsRequest)
			response, err = ms.handler.HandleDiscreteInputs(&request)
		case HoldingRegistersRequest:
			request := raw.(HoldingRegistersRequest)
			response, err = ms.handler.HandleHoldingRegisters(&request)
		case InputRegistersRequest:
			request := raw.(InputRegistersRequest)
			response, err = ms.handler.HandleInputRegisters(&request)
		default:
			err = fmt.Errorf("Function code not implemented!")
			ms.logger.Warningf("Can't execute request! (%v)", err)
			ms.sendErrorMessage(receivedData, exIllegalFunction)
			receivedData = nil
			continue
		}

		if err != nil {
			ms.logger.Warningf("Request execution failed! (%v)", err)
			ms.sendErrorMessage(receivedData, exIllegalDataAddress)
			receivedData = nil
			continue
		}

		ms.logger.Infof("Response = %v", response)
		if bytesToSend, err = createBytesFromRequest(receivedData, response); err != nil {
			ms.logger.Warningf("Compose answer failed! (%v)", err)
			ms.sendErrorMessage(receivedData, exIllegalFunction)
			receivedData = nil
			continue
		}

		ms.logger.Infof("Written %v bytes: 0x[% X]", len(bytesToSend), bytesToSend)
		if _, err = ms.port.Write(bytesToSend); err != nil {
			ms.logger.Errorf("Send answer failed! (%v)", err)
		}

		// Request executed and answer sent. Ready to receivce next message.
		receivedData = nil
	}
}

func messageIsValid(message []byte) error {
	if len(message) <= 2 {
		return fmt.Errorf("Message must be larger than 2 bytes.")
	}

	var c crc
	c.init()
	c.add(message[:len(message)-2])
	isEqual := c.isEqual(message[len(message)-2], message[len(message)-1])
	if isEqual {
		return nil
	} else {
		return fmt.Errorf("Checking CRC of [% X]. WRONG CRC!", message)
	}
}

func createRequestFromBytes(message []byte) (result any, err error) {
	reqType := message[1]

	switch reqType {
	case fcReadCoils:
		fallthrough
	case fcWriteSingleCoil:
		fallthrough
	case fcWriteMultipleCoils:
		var req = CoilsRequest{}
		req.ClientAddr = "127.0.0.1"
		req.UnitId = message[0]
		req.Addr = uint16(message[2])<<8 + uint16(message[3])
		req.Quantity = uint16(message[4])<<8 + uint16(message[5])

		if reqType == fcWriteSingleCoil {
			req.IsWrite = true
			req.Quantity = 1
			req.Args = []bool{true}
		}
		if reqType == fcWriteMultipleCoils {
			req.IsWrite = true
			count := 0
			for i := uint(0); i < uint(message[6]); i++ {
				byteval := message[7+i]
				for k := 0; k < 8; k++ {
					bitval := (byteval >> k) & 0x01
					if bitval == 1 && count < int(req.Quantity) {
						req.Args = append(req.Args, true)
						count++
					} else if bitval == 0 && count < int(req.Quantity) {
						req.Args = append(req.Args, false)
						count++
					}
				}
			}
		}
		return req, nil

	case fcReadDiscreteInputs:
		var req = DiscreteInputsRequest{}
		req.ClientAddr = "127.0.0.1"
		req.UnitId = message[0]
		req.Addr = uint16(message[2])<<8 + uint16(message[3])
		req.Quantity = uint16(message[4])<<8 + uint16(message[5])
		return req, nil

	case fcReadHoldingRegisters:
		fallthrough
	case fcWriteSingleRegister:
		// Write holding register
		fallthrough
	case fcWriteMultipleRegisters:
		// Write multiple holding register
		var req = HoldingRegistersRequest{}
		req.ClientAddr = "127.0.0.1"
		req.UnitId = message[0]
		req.Addr = uint16(message[2])<<8 + uint16(message[3])
		req.Quantity = uint16(message[4])<<8 + uint16(message[5])

		if reqType == fcWriteSingleRegister {
			req.IsWrite = true
			req.Quantity = 1
			req.Args = []uint16{uint16(message[4])<<8 + uint16(message[5])}
		}
		if reqType == fcWriteMultipleRegisters {
			if (req.Quantity%2 != 0) || (message[6]%2 != 0) {
				return nil, fmt.Errorf("Expected an even number as requested quantity")
			}
			req.IsWrite = true
			for i := uint(0); i < uint(message[6]); i += 2 {
				val := uint16(message[7+i])<<8 + uint16(message[7+i+1])
				req.Args = append(req.Args, val)
			}
		}
		return req, nil

	case fcReadInputRegisters:
		var req = InputRegistersRequest{}
		req.ClientAddr = "127.0.0.1"
		req.UnitId = message[0]
		req.Addr = uint16(message[2])<<8 + uint16(message[3])
		req.Quantity = uint16(message[4])<<8 + uint16(message[5])
		return req, nil

	default:
		return nil, fmt.Errorf("Function code not supported 0x%X", reqType)
	}
}

func createBytesFromRequest(originalMessage []byte, requestResult any) (result []byte, err error) {
	reqType := originalMessage[1]
	var c crc
	c.init()

	switch reqType {
	case fcReadCoils:
		fallthrough
	case fcReadDiscreteInputs:
		result = append(result, originalMessage[0])
		result = append(result, originalMessage[1])
		nofBytes := int(float64(len(requestResult.([]bool)))/8 + 0.5)
		result = append(result, uint8(nofBytes&0x0F))
		for i := 0; i < nofBytes; i++ {
			byteVal := 0
			for k := 0; k < 8; k++ {
				pos := i*8 + k
				if pos >= len(requestResult.([]bool)) {
					break
				}
				if requestResult.([]bool)[pos] == true {
					byteVal += 1 << k
				}
			}
			result = append(result, byte(byteVal))
		}

	case fcReadHoldingRegisters:
		fallthrough
	case fcReadInputRegisters:
		result = append(result, originalMessage[0])
		result = append(result, originalMessage[1])
		nofBytes := len(requestResult.([]uint16)) * 2
		result = append(result, uint8(nofBytes&0x0F))
		for i := 0; i < nofBytes/2; i++ {
			result = append(result, byte(requestResult.([]uint16)[i]>>8))
			result = append(result, byte(requestResult.([]uint16)[i]&0xFF))
		}

	case fcWriteSingleCoil:
		fallthrough
	case fcWriteSingleRegister:
		// Write holding register
		if requestResult != nil {
			result = append(result, originalMessage...)
			return
		}

	case fcWriteMultipleCoils:
		fallthrough
	case fcWriteMultipleRegisters:
		// Write multiple holding register
		result = append(result, originalMessage[:6]...)
	default:
		return nil, fmt.Errorf("Could not compute bytes. Function code not supported 0x%X", reqType)
	}

	c.add(result)
	result = append(result, c.value()...)
	return
}
