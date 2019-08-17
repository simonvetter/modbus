package main

import (
	"errors"
	"fmt"
	"flag"
	"os"
	"strings"
	"strconv"
	"time"

	"github.com/simonvetter/modbus"
)

func main() {
	var err		error
	var help	bool
	var client	*modbus.ModbusClient
	var config	*modbus.ClientConfiguration
	var target	string
	var speed	uint
	var dataBits	uint
	var parity	string
	var stopBits	uint
	var endianness	string
	var wordOrder	string
	var timeout	string
	var cEndianess	modbus.Endianness
	var cWordOrder	modbus.WordOrder
	var unitId	uint
	var runList	[]operation

	flag.StringVar(&target, "target", "rtu:///dev/ttyUSB0", "target device to connect to (e.g. tcp://somehost:502) [required]")
	flag.UintVar(&speed, "speed", 9600, "serial bus speed in bps (rtu)")
	flag.UintVar(&dataBits, "data-bits", 8, "number of bits per character on the serial bus (rtu)")
	flag.StringVar(&parity, "parity", "none", "parity bit <none|even|odd> on the serial bus (rtu)")
	flag.UintVar(&stopBits, "stop-bits", 2, "number of stop bits <0|1|2>) on the serial bus (rtu)")
	flag.StringVar(&timeout, "timeout", "3s", "timeout value")
	flag.StringVar(&endianness, "endianness", "big", "register endianness <little|big>")
	flag.StringVar(&wordOrder, "word-order", "highfirst", "word ordering for 32-bit registers <highfirst|hf|lowfirst|lf>")
	flag.UintVar(&unitId, "unit-id", 1, "unit/slave id to use")
	flag.BoolVar(&help, "help", false, "show a wall-of-text help message")
	flag.Parse()

	if help {
		displayHelp()
		os.Exit(0)
	}

	// create and populate the client configuration object
	config		= &modbus.ClientConfiguration{
		URL:		target,
		Speed:		speed,
		DataBits:	dataBits,
		StopBits:	stopBits,
	}

	switch parity {
	case "none":	config.Parity	= modbus.PARITY_NONE
	case "odd":	config.Parity	= modbus.PARITY_ODD
	case "even":	config.Parity	= modbus.PARITY_EVEN
	default:
		fmt.Printf("unknown parity setting '%s' (should be one of none, odd or even)\n",
	                   parity)
		os.Exit(1)
	}

	config.Timeout, err = time.ParseDuration(timeout)
	if err != nil {
		fmt.Printf("failed to parse timeout setting '%s': %v\n", timeout, err)
		os.Exit(1)
	}

	// parse encoding (endianness and word order) settings
	switch endianness {
	case "big":	cEndianess	= modbus.BIG_ENDIAN
	case "little":  cEndianess	= modbus.LITTLE_ENDIAN
	default:
		fmt.Printf("unknown endianness setting '%s' (should either be big or little)\n",
			   endianness)
		os.Exit(1)
	}

	switch wordOrder {
	case "highfirst", "hf": cWordOrder	= modbus.HIGH_WORD_FIRST
	case "lowfirst", "lf":  cWordOrder	= modbus.LOW_WORD_FIRST
	default:
		fmt.Printf("unknown word order setting '%s' (should be one of highfirst, hf, littlefirst, lf)\n",
			   wordOrder)
		os.Exit(1)
	}

	if len(flag.Args()) == 0 {
		fmt.Printf("nothing to do.\n")
		os.Exit(0)
	}

	// parse arguments and build a list of objects
	for _, arg := range flag.Args() {
		var splitArgs	[]string
		var o		operation

		splitArgs	= strings.Split(arg, ":")
		if len(splitArgs) < 2 && splitArgs[0] != "repeat" && splitArgs[0] != "date" {
			fmt.Printf("illegal command format (should be command:arg1:arg2..., e.g. rh:uint32:0x1000+5)\n")
			os.Exit(2)
		}

		switch splitArgs[0] {
		case "rc", "readCoil", "readCoils",
		     "rdi", "readDiscreteInput", "readDiscreteInputs":

			if len(splitArgs) != 2 {
				fmt.Printf("need exactly 1 arguments after rc/rdi, got %v\n",
					   len(splitArgs) - 1)
				os.Exit(2)
			}

			if splitArgs[0] == "rc" || splitArgs[0] == "readCoil" || splitArgs[0] == "readCoils" {
				o.isCoil	= true
			}

			o.op			= readBools
			o.addr, o.quantity, err	= parseAddressAndQuantity(splitArgs[1])
			if err != nil {
				fmt.Printf("failed to parse address ('%v'): %v\n", splitArgs[1], err)
				os.Exit(2)
			}

		case "rh", "readHoldingRegister", "readHoldingRegisters",
		     "ri", "readInputRegister", "readInputRegisters":

			if len(splitArgs) != 3 {
				fmt.Printf("need exactly 2 arguments after rh/ri, got %v\n",
					   len(splitArgs) - 1)
				os.Exit(2)
			}

			if splitArgs[0] == "rh" || splitArgs[0] == "readHoldingRegister" ||
			   splitArgs[0] == "readHoldingRegisters" {
				   o.isHoldingReg	= true
			}

			switch splitArgs[1] {
			case "uint16":	o.op	= readUint16
			case "int16":	o.op	= readInt16
			case "uint32":	o.op	= readUint32
			case "int32":	o.op	= readInt32
			case "float32":	o.op	= readFloat32
			default:
				fmt.Printf("unknown register type '%v' (should be one of [u]unt16, [u]int32, float32)\n",
					   splitArgs[1])
				os.Exit(2)
			}

			o.addr, o.quantity, err	= parseAddressAndQuantity(splitArgs[2])
			if err != nil {
				fmt.Printf("failed to parse address ('%v'): %v\n", splitArgs[2], err)
				os.Exit(2)
			}

		case "wc", "writeCoil":
			if len(splitArgs) != 3 {
				fmt.Printf("need exactly 2 arguments after writeCoil, got %v\n",
					   len(splitArgs) - 1)
				os.Exit(2)
			}

			o.op			= writeCoil
			o.addr, err		= parseUint16(splitArgs[1])
			if err != nil {
				fmt.Printf("failed to parse address ('%v'): %v\n", splitArgs[1], err)
				os.Exit(2)
			}

			switch splitArgs[2] {
			case "true":	o.coil	= true
			case "false":	o.coil	= false
			default:
				fmt.Printf("failed to parse coil value '%v' (should either be true or false)\n",
					   splitArgs[2])
				os.Exit(2)
			}

		case "wr", "writeRegister":
			if len(splitArgs) != 4 {
				fmt.Printf("need exactly 3 arguments after writeRegister, got %v\n",
					   len(splitArgs) - 1)
				os.Exit(2)
			}

			o.addr, err	= parseUint16(splitArgs[2])
			if err != nil {
				fmt.Printf("failed to parse address ('%v'): %v\n", splitArgs[2], err)
				os.Exit(2)
			}

			switch splitArgs[1] {
			case "uint16":
				o.op		= writeUint16
				o.u16, err	= parseUint16(splitArgs[3])

			case "int16":
				o.op		= writeInt16
				o.u16, err	= parseInt16(splitArgs[3])

			case "uint32":
				o.op		= writeUint32
				o.u32, err	= parseUint32(splitArgs[3])

			case "int32":
				o.op		= writeInt32
				o.u32, err	= parseInt32(splitArgs[3])

			case "float32":
				o.op		= writeFloat32
				o.f32, err	= parseFloat32(splitArgs[3])

			default:
				fmt.Printf("unknown register type '%v' (should be one of [u]unt16, [u]int32, float32)\n",
					   splitArgs[1])
				os.Exit(2)
			}

			if err != nil {
				fmt.Printf("failed to parse '%s' as %s: %v\n", splitArgs[3], splitArgs[1], err)
				os.Exit(2)
			}

		case "sleep":
			if len(splitArgs) != 2 {
				fmt.Printf("need exactly 1 arguments after sleep, got %v\n",
					   len(splitArgs) - 1)
				os.Exit(2)
			}

			o.op		= sleep
			o.duration, err	= time.ParseDuration(splitArgs[1])
			if err != nil {
				fmt.Printf("failed to parse '%s' as duration: %v\n", splitArgs[1], err)
				os.Exit(2)
			}

		case "suid", "setUnitId", "sid":
			if len(splitArgs) != 2 {
				fmt.Printf("need exactly 1 arguments after setUnitId, got %v\n",
					   len(splitArgs) - 1)
				os.Exit(2)
			}

			o.op		= setUnitId
			o.unitId, err	= parseUnitId(splitArgs[1])
			if err != nil {
				fmt.Printf("failed to parse '%s' as unit id: %v\n", splitArgs[1], err)
				os.Exit(2)
			}

		case "repeat":
			if len(splitArgs) != 1 {
				fmt.Printf("repeat takes no argument, got %v\n",
					   len(splitArgs) - 1)
				os.Exit(2)
			}

			o.op		= repeat

		case "date":
			if len(splitArgs) != 1 {
				fmt.Printf("date takes no argument, got %v\n",
					   len(splitArgs) - 1)
			os.Exit(2)
		   }

		   o.op        = date

		case "scan":
			if len(splitArgs) != 2 {
				fmt.Printf("need exactly 1 arguments after scan, got %v\n",
					   len(splitArgs) - 1)
				os.Exit(2)
			}

			switch splitArgs[1] {
			case "c", "coils":
				o.op		= scanBools
				o.isCoil	= true
			case "di", "discreteInputs":
				o.op		= scanBools
				o.isCoil	= false
			case "h", "hr", "holding", "holdingRegisters":
				o.op		= scanRegisters
				o.isHoldingReg	= true
			case "i", "ir", "input", "inputRegisters":
				o.op		= scanRegisters
				o.isHoldingReg	= false
			default:
				fmt.Printf("unknown register type '%v' (valid options <coils|di|hr|ir>\n",
					   splitArgs[1])
				os.Exit(2)
			}

		default:
			fmt.Printf("unsupported command '%v'\n", splitArgs[0])
			os.Exit(2)
		}

		runList	= append(runList, o)

	}

	// create the modbus client
	client, err	= modbus.NewClient(config)
	if err != nil {
		fmt.Printf("failed to create client: %v\n", err)
		os.Exit(1)
	}

	err = client.SetEncoding(cEndianess, cWordOrder)
	if err != nil {
		fmt.Printf("failed to set encoding: %v\n", err)
		os.Exit(1)
	}

	// set the initial unit id (note: this can be changed later at runtime through
	// the setUnitId command)
	if unitId > 0xff {
		fmt.Printf("set unit id: value '%v' out of range\n", unitId)
		os.Exit(1)
	}
	client.SetUnitId(uint8(unitId))

	// connect to the remote host/open the serial port
	err		= client.Open()
	if err != nil {
		fmt.Printf("failed to open client: %v\n", err)
		os.Exit(2)
	}

	for opIdx := 0; opIdx < len(runList); opIdx++ {
		var o *operation = &runList[opIdx]

		switch o.op {
		case readBools:
			var res	[]bool

			if o.isCoil {
				res, err = client.ReadCoils(o.addr, o.quantity + 1)
			} else {
				res, err = client.ReadDiscreteInputs(o.addr, o.quantity + 1)
			}
			if err != nil {
				fmt.Printf("failed to read coils/discrete inputs: %v\n", err)
			} else {
				for idx := range res {
					fmt.Printf("0x%04x\t%-5v : %v\n", o.addr + uint16(idx),
									   o.addr + uint16(idx),
								           res[idx])
				}
			}

		case readUint16, readInt16:
			var res	[]uint16

			if o.isHoldingReg {
				res, err = client.ReadRegisters(o.addr, o.quantity + 1, modbus.HOLDING_REGISTER)
			} else {
				res, err = client.ReadRegisters(o.addr, o.quantity + 1, modbus.INPUT_REGISTER)
			}
			if err != nil {
				fmt.Printf("failed to read holding/input registers: %v\n", err)
			} else {
				for idx := range res {
					if o.op == readUint16 {
						fmt.Printf("0x%04x\t%-5v : 0x%04x\t%v\n",
							   o.addr + uint16(idx),
							   o.addr + uint16(idx),
							   res[idx], res[idx])
					} else {
						fmt.Printf("0x%04x\t%-5v : 0x%04x\t%v\n",
							   o.addr + uint16(idx),
							   o.addr + uint16(idx),
							   res[idx], int16(res[idx]))
					}
				}
			}

		case readUint32, readInt32:
			var res	[]uint32

			if o.isHoldingReg {
				res, err = client.ReadUint32s(o.addr, o.quantity + 1, modbus.HOLDING_REGISTER)
			} else {
				res, err = client.ReadUint32s(o.addr, o.quantity + 1, modbus.INPUT_REGISTER)
			}
			if err != nil {
				fmt.Printf("failed to read holding/input registers: %v\n", err)
			} else {
				for idx := range res {
					if o.op == readUint32 {
						fmt.Printf("0x%04x\t%-5v : 0x%08x\t%v\n",
							   o.addr + (uint16(idx) * 2),
							   o.addr + (uint16(idx) * 2),
							   res[idx], res[idx])
					} else {
						fmt.Printf("0x%04x\t%-5v : 0x%08x\t%v\n",
							   o.addr + (uint16(idx) * 2),
							   o.addr + (uint16(idx) * 2),
							   res[idx], int32(res[idx]))
					}
				}
			}

		case readFloat32:
			var res	[]float32

			if o.isHoldingReg {
				res, err = client.ReadFloat32s(o.addr, o.quantity + 1, modbus.HOLDING_REGISTER)
			} else {
				res, err = client.ReadFloat32s(o.addr, o.quantity + 1, modbus.INPUT_REGISTER)
			}
			if err != nil {
				fmt.Printf("failed to read holding/input registers: %v\n", err)
			} else {
				for idx := range res {
					fmt.Printf("0x%04x\t%-5v : %f\n",
						   o.addr + (uint16(idx) * 2),
						   o.addr + (uint16(idx) * 2),
						   res[idx])
				}
			}

		case writeCoil:
			err	= client.WriteCoil(o.addr, o.coil)
			if err != nil {
				fmt.Printf("failed to write %v at coil address 0x%04x: %v\n",
					   o.coil, o.addr, err)
			} else {
				fmt.Printf("wrote %v at coil address 0x%04x\n",
					   o.coil, o.addr)
			}

		case writeUint16:
			err	= client.WriteRegister(o.addr, o.u16)
			if err != nil {
				fmt.Printf("failed to write %v at register address 0x%04x: %v\n",
					   o.u16, o.addr, err)
			} else {
				fmt.Printf("wrote %v at register address 0x%04x\n",
					   o.u16, o.addr)
			}

		case writeInt16:
			err	= client.WriteRegister(o.addr, o.u16)
			if err != nil {
				fmt.Printf("failed to write %v at register address 0x%04x: %v\n",
					   int16(o.u16), o.addr, err)
			} else {
				fmt.Printf("wrote %v at register address 0x%04x\n",
					   int16(o.u16), o.addr)
			}

		case writeUint32:
			err	= client.WriteUint32(o.addr, o.u32)
			if err != nil {
				fmt.Printf("failed to write %v at address 0x%04x: %v\n",
					   o.u32, o.addr, err)
			} else {
				fmt.Printf("wrote %v at address 0x%04x\n",
					   o.u32, o.addr)
			}

		case writeInt32:
			err	= client.WriteUint32(o.addr, o.u32)
			if err != nil {
				fmt.Printf("failed to write %v at address 0x%04x: %v\n",
					   int32(o.u32), o.addr, err)
			} else {
				fmt.Printf("wrote %v at address 0x%04x\n",
					   int32(o.u32), o.addr)
			}

		case writeFloat32:
			err	= client.WriteFloat32(o.addr, o.f32)
			if err != nil {
				fmt.Printf("failed to write %f at address 0x%04x: %v\n",
					   o.f32, o.addr, err)
			} else {
				fmt.Printf("wrote %f at address 0x%04x\n",
					   o.f32, o.addr)
			}

		case sleep:
			time.Sleep(o.duration)

		case setUnitId:
			client.SetUnitId(o.unitId)

		case repeat:
			// start over
			opIdx = -1

		case date:
			fmt.Printf("%s\n", time.Now().Format(time.RFC3339))

		case scanBools:
			performBoolScan(client, o.isCoil)

		case scanRegisters:
			performRegisterScan(client, o.isHoldingReg)

		default:
			fmt.Printf("unknown operation %v\n", o)
			os.Exit(100)
		}
	}

	return
}

const (
	readBools		uint	= iota + 1
	readUint16
	readInt16
	readUint32
	readInt32
	readFloat32
	writeCoil
	writeCoils
	writeUint16
	writeInt16
	writeInt32
	writeUint32
	writeFloat32
	setUnitId
	sleep
	repeat
	date
	scanBools
	scanRegisters
)

type operation struct {
	op		uint
	addr		uint16
	isCoil		bool
	isHoldingReg	bool
	quantity	uint16
	coil		bool
	u16		uint16
	u32		uint32
	f32		float32
	duration	time.Duration
	unitId		uint8
}

func parseUint16(in string) (u16 uint16, err error) {
	var val uint64

	val, err = strconv.ParseUint(in, 0, 16)
	if err == nil {
		u16	= uint16(val)
		return
	}

	return
}

func parseInt16(in string) (u16 uint16, err error) {
	var val	int64

	val, err = strconv.ParseInt(in, 10, 16)
	if err == nil {
		u16	= uint16(int16(val))
	}

	return
}

func parseUint32(in string) (u32 uint32, err error) {
	var val	uint64

	val, err = strconv.ParseUint(in, 0, 32)
	if err == nil {
		u32	= uint32(val)
		return
	}

	return
}

func parseInt32(in string) (u32 uint32, err error) {
	var val	int64

	val, err = strconv.ParseInt(in, 0, 32)
	if err == nil {
		u32	= uint32(int32(val))
	}

	return
}

func parseFloat32(in string) (f32 float32, err error) {
	var val float64

	val, err	= strconv.ParseFloat(in, 32)
	if err == nil {
		f32	= float32(val)
	}

	return
}

func parseAddressAndQuantity(in string) (addr uint16, quantity uint16, err error) {
	var split = strings.Split(in, "+")

	switch {
	case len(split) == 1:
		addr, err	= parseUint16(in)

	case len(split) == 2:
		addr, err	= parseUint16(split[0])
		if err != nil {
			return
		}
		quantity, err	= parseUint16(split[1])
	default:
		err		= errors.New("illegal format")
	}

	return
}

func parseUnitId(in string) (addr uint8, err error) {
	var val uint64

	val, err	= strconv.ParseUint(in, 0, 8)
	if err == nil {
		addr = uint8(val)
	}

	return
}

func performBoolScan(client *modbus.ModbusClient, isCoil bool) {
	var err		error
	var addr	uint32
	var val		bool
	var count	uint
	var regType	string

	if isCoil {
		regType	= "coil"
	} else {
		regType	= "discrete input"
	}

	fmt.Printf("starting %s scan\n", regType)

	for addr = 0; addr <= 0xffff; addr++ {
		if isCoil {
			val, err	= client.ReadCoil(uint16(addr))
		} else {
			val, err	= client.ReadDiscreteInput(uint16(addr))
		}
		if err == modbus.ErrIllegalDataAddress || err == modbus.ErrIllegalFunction {
			// the register does not exist
			continue
		} else if err != nil {
			fmt.Printf("failed to read %s at address 0x%04x: %v\n",
			           regType, addr, err)
		} else {
			// we found a coil: display its address and value
			fmt.Printf("0x%04x\t%-5v : %v\n", addr, addr, val)
			count++
		}
	}

	fmt.Printf("found %v %ss\n", count, regType)

	return
}

func performRegisterScan(client *modbus.ModbusClient, isHoldingReg bool) {
	var err		error
	var addr	uint32
	var val		uint16
	var count	uint
	var regType	string

	if isHoldingReg {
		regType	= "holding register"
	} else {
		regType	= "input register"
	}

	fmt.Printf("starting %s scan\n", regType)

	for addr = 0; addr <= 0xffff; addr++ {
		if isHoldingReg {
			val, err	= client.ReadRegister(uint16(addr), modbus.HOLDING_REGISTER)
		} else {
			val, err	= client.ReadRegister(uint16(addr), modbus.INPUT_REGISTER)
		}
		if err == modbus.ErrIllegalDataAddress || err == modbus.ErrIllegalFunction {
			// the register does not exist
			continue
		} else if err != nil {
			fmt.Printf("failed to read %s at address 0x%04x: %v\n",
			           regType, addr, err)
		} else {
			// we found a register: display its address and value
			fmt.Printf("0x%04x\t%-5v : 0x%04x\t%v\n",
				   addr, addr, val, val)
			count++
		}
	}

	fmt.Printf("found %v %ss\n", count, regType)

	return
}

func displayHelp() {
	fmt.Println(
`
This tool is a modbus command line interface client meant to allow quick and easy
interaction with modbus devices (e.g. probing or troubleshooting).

Available options:`)
	flag.PrintDefaults()
	fmt.Printf(
`

Command strings must be given as trailing arguments after any options.

Example: modbus-cli --target=tcp://somehost:502 --timeout=3s rh:uint16:0x100+5 wc:12:true
	 Read 6 holding registers at address 0x100 then set the coil at address 12 to true
	 on modbus/tcp device somehost port 502, with a timeout of 3s.

Available commands:
* <rc|readCoils>:<addr>[+additional quantity]
  Read coil at address <addr>, plus any additional coils if specified.

  rc:0x100+199	reads 200 coils starting at address 0x100 (hex)
  rc:300	reads 1 coil at address 300 (decimal)

* <rdi|readDiscreteInputs>:<addr>[+additional quantity]
  Read discrete input at address <addr>, plus any additional discrete inputs if specified.

  rdi:0x100+199	reads 200 discrete inputs starting at address 0x100 (hex)
  rdi:300	reads 1 discrete input at address 300 (decimal)

* <rh|readHoldingRegisters>:<type>:<addr>[+additional quantity]
  Read holding registers at address <addr>, plus any additional registers if specified,
  decoded as <type> which should be one of:
  - uint16: unsigned 16-bit integer,
  - int16: signed 16-bit integer,
  - uint32: unsigned 32-bit integer (2 contiguous modbus registers),
  - int32: signed 32-bit integer (2 contiguous modbus registers),
  - float32: 32-bit floating point number (2 contiguous modbus registers).

  rh:int16:0x300+1 	reads 2 consecutive 16-bit signed integers at addresses 0x300 and 0x301
  rh:uint32:20		reads a 32-bit unsigned integer at adresses 20-21 (2 modbus registers)
  rh:float32:500+10	reads 11 32-bit floating point numbers at adresses 500-521
			(11 * 32bit make for 22 16-bit contiguous modbus registers)

* <ri:readInputRegisters>:<type>:<addr>[+additional quanitity]
  Read input registers at address <addr>, plus any additional registers if specified, decoded
  in the same way as explained above.

  ri:uint16:0x300+1	reads 2 consecutive 16-bit unsigned integers at adresses 0x300 and 0x301
  ri:int32:20		reads a 32-bit signed integer at adresses 20-21 (2 modbus registers)

* <wc|writeCoil>:<addr>:<value>
  Set the coil at address <addr> to either true or false, depending on <value>.

  wc:1:true	writes true to the coil at address 1
  wc:2:false	writes false to the coil at address 2

* <wr:writeRegister>:<type>:<addr>:<value>
  Write <value> to register(s) at address <addr>, using the encoding given by <type>.

  wr:int16:0xf100:-10	writes -10 as a 16-bit signed integer at address 0xf100
                        (1 modbus register)
  wr:int32:0xff00:0xff	writes 0xff as a 32-bit signed integer at addresses 0xff00-0xff01
			(2 consecutive modbus registers)
  wr:float32:100:-3.2	writes -3.2 as a 32-bit float at addresses 100-101
			(2 consecutive modbus registers)

* sleep:<duration>
  Pause for <duration>, specified as a golang duration string.

  sleep:300s		sleeps for 300 seconds
  sleep:3m		sleeps for 3 minutes
  sleep:3ms		sleeps for 3 milliseconds

* <setUnitId|suid|sid>:<unit id>
  Switch to unit id (slave id) <unit id> for subsequent requests.

  sid:10		selects unit id #10

* repeat
  Restart execution of the given commands.

  rh:uint32:100 sleep:1s repeat   reads a 32-bit unsigned integer at adresses 100-101 and
				  pauses for one second, forever in a loop.

* date
  Print the current date and time (can be useful for long-running scripts).

* scan:<type>
  Perform a modbus "scan" of the modbus type <type>, which can be one of:
  - "c", "coils",
  - "di", "discreteInputs",
  - "hr", "holdingRegisters",
  - "ir", "inputRegisters"

  scan:hr	scans the device for holding registers.
  scan:di	scans the device for discrete inputs.

  Read requests are made over the entire adress space (65535 adresses).
  Adresses for which a non-error response is received are listed, along with the value received.
  Errors other than Illegal Data Address and Illegal Function are also shown, as they should
  not happen in sane implementations.

Register endianness and word order:
  The endianness of holding/input registers can be specified with --endianness <big|little> and
  defaults to big endian (as per the modbus spec).
  For constructs spanning two consecutive registers (namely int32, uint32 and float32), the word
  order can be set with --word-order <highfirst|lowfirst> and arbitrarily defaults to highfirst
  (i.e. most significant word first).

Examples:
  $ modbus-cli --target tcp://10.100.0.10:502 rh:uint32:0x100+5 rc:0+10 wc:3:true
  Connect to 10.100.0.10 port 502, read 6 consecutive 32-bit unsigned integers at addresses
  0x100-0x10b (12 modbus registers) and 11 coils at addresses 0-10, then set the coil at
  address 3 to true.

  $ modbus-cli --target rtu:///dev/ttyUSB0 --speed 19200 suid:2 rh:uint16:0+7 \
    wr:uint16:0x2:0x0605 suid:3 ri:int16:0+1 sleep:1s repeat
  Open serial port /dev/ttyUSB0 at a speed of 19200 bps and repeat forever:
    select unit id (slave id) 2, read holding registers at adresses 0-7 as 16 bit unsigned
    integers, write 0x605 as a 16-bit unsigned integer at address 2,
    change for unit id 3, read input registers 0-1 as 16-bit signed integers,
    pause for 1s.

  $ modbus-cli --target tcp://somehost:502 scan:hr scan:ir scan:di scan:coils
  Connect to somehost port 502 and perform a scan of all modbus types (namely
  holding registers, input registers, discrete inputs and coils).

`)

	return
}
