package main

import (
	"crypto/tls"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/munnik/modbus"
	"go.bug.st/serial"
)

func main() {
	var err error
	var help bool
	var client *modbus.Client
	var config *modbus.Configuration
	var target string
	var caPath string   // path to TLS CA/server certificate
	var certPath string // path to TLS client certificate
	var keyPath string  // path to TLS client key
	var clientKeyPair tls.Certificate
	var speed int
	var dataBits int
	var parity string
	var stopBits string
	var endianness string
	var wordOrder string
	var timeout string
	var cEndianess modbus.Endianness
	var cWordOrder modbus.WordOrder
	var unitID uint
	var runList []operation

	flag.StringVar(&target, "target", "", "target device to connect to (e.g. tcp://somehost:502) [required]")
	flag.IntVar(&speed, "speed", 19200, "serial bus speed in bps (rtu)")
	flag.IntVar(&dataBits, "data-bits", 8, "number of bits per character on the serial bus (rtu)")
	flag.StringVar(&parity, "parity", "none", "parity bit <none|even|odd> on the serial bus (rtu)")
	flag.StringVar(&stopBits, "stop-bits", "2", "number of stop bits <0|1|1.5|2>) on the serial bus (rtu)")
	flag.StringVar(&timeout, "timeout", "3s", "timeout value")
	flag.StringVar(&endianness, "endianness", "big", "register endianness <little|big>")
	flag.StringVar(&wordOrder, "word-order", "highfirst", "word ordering for 32-bit registers <highfirst|hf|lowfirst|lf>")
	flag.UintVar(&unitID, "unit-id", 1, "unit/slave id to use")
	flag.StringVar(&certPath, "cert", "", "path to TLS client certificate")
	flag.StringVar(&keyPath, "key", "", "path to TLS client key")
	flag.StringVar(&caPath, "ca", "", "path to TLS CA/server certificate")
	flag.BoolVar(&help, "help", false, "show a wall-of-text help message")
	flag.Parse()

	if help {
		displayHelp()
		os.Exit(0)
	}

	if target == "" {
		fmt.Printf("no target specified, please use --target\n")
		os.Exit(1)
	}

	// create and populate the client configuration object
	config = &modbus.Configuration{
		URL:      target,
		Speed:    speed,
		DataBits: dataBits,
	}

	switch parity {
	case "none":
		config.Parity = serial.NoParity
	case "odd":
		config.Parity = serial.OddParity
	case "even":
		config.Parity = serial.EvenParity
	default:
		fmt.Printf("unknown parity setting '%s' (should be one of none, odd or even)\n",
			parity)
		os.Exit(1)
	}
	switch stopBits {
	case "1":
		config.StopBits = serial.OneStopBit
	case "1.5":
		config.StopBits = serial.OnePointFiveStopBits
	case "2":
		config.StopBits = serial.TwoStopBits
	default:
		fmt.Printf("unknown stop-bits setting '%s' (should be one of 1, 1.5 or 2)\n",
			stopBits)
		os.Exit(1)
	}

	config.Timeout, err = time.ParseDuration(timeout)
	if err != nil {
		fmt.Printf("failed to parse timeout setting '%s': %v\n", timeout, err)
		os.Exit(1)
	}

	// parse encoding (endianness and word order) settings
	switch endianness {
	case "big":
		cEndianess = modbus.BigEndian
	case "little":
		cEndianess = modbus.LittleEndian
	default:
		fmt.Printf("unknown endianness setting '%s' (should either be big or little)\n",
			endianness)
		os.Exit(1)
	}

	switch wordOrder {
	case "highfirst", "hf":
		cWordOrder = modbus.HighWordFirst
	case "lowfirst", "lf":
		cWordOrder = modbus.LowWordFirst
	default:
		fmt.Printf("unknown word order setting '%s' (should be one of highfirst, hf, littlefirst, lf)\n",
			wordOrder)
		os.Exit(1)
	}

	// handle TLS options
	if strings.HasPrefix(target, "tcp+tls://") {
		if certPath == "" {
			fmt.Print("TLS requested but no client certificate given, please use --cert\n")
			os.Exit(1)
		}

		if keyPath == "" {
			fmt.Print("TLS requested but no client key given, please use --key\n")
			os.Exit(1)
		}

		if caPath == "" {
			fmt.Print("TLS requested but no CA/server cert given, please use --ca\n")
			os.Exit(1)
		}

		clientKeyPair, err = tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			fmt.Printf("failed to load client tls key pair: %v\n", err)
			os.Exit(1)
		}
		config.TLSClientCert = &clientKeyPair

		config.TLSRootCAs, err = modbus.LoadCertPool(caPath)
		if err != nil {
			fmt.Printf("failed to load tls CA/server certificate: %v\n", err)
			os.Exit(1)
		}
	}

	if len(flag.Args()) == 0 {
		fmt.Printf("nothing to do.\n")
		os.Exit(0)
	}

	// parse arguments and build a list of objects
	for _, arg := range flag.Args() {
		var splitArgs []string
		var o operation

		splitArgs = strings.Split(arg, ":")
		if len(splitArgs) < 2 && splitArgs[0] != "repeat" && splitArgs[0] != "date" {
			fmt.Printf("illegal command format (should be command:arg1:arg2..., e.g. rh:uint32:0x1000+5)\n")
			os.Exit(2)
		}

		switch splitArgs[0] {
		case "rc", "readCoil", "readCoils",
			"rdi", "readDiscreteInput", "readDiscreteInputs":

			if len(splitArgs) != 2 {
				fmt.Printf("need exactly 1 argument after rc/rdi, got %v\n",
					len(splitArgs)-1)
				os.Exit(2)
			}

			if splitArgs[0] == "rc" || splitArgs[0] == "readCoil" || splitArgs[0] == "readCoils" {
				o.isCoil = true
			}

			o.op = readBools
			o.addr, o.quantity, err = parseAddressAndQuantity(splitArgs[1])
			if err != nil {
				fmt.Printf("failed to parse address ('%v'): %v\n", splitArgs[1], err)
				os.Exit(2)
			}

		case "rh", "readHoldingRegister", "readHoldingRegisters",
			"ri", "readInputRegister", "readInputRegisters":

			if len(splitArgs) != 3 {
				fmt.Printf("need exactly 2 arguments after rh/ri, got %v\n",
					len(splitArgs)-1)
				os.Exit(2)
			}

			if splitArgs[0] == "rh" || splitArgs[0] == "readHoldingRegister" ||
				splitArgs[0] == "readHoldingRegisters" {
				o.isHoldingReg = true
			}

			switch splitArgs[1] {
			case "uint16":
				o.op = readUint16
			case "int16":
				o.op = readInt16
			case "uint32":
				o.op = readUint32
			case "int32":
				o.op = readInt32
			case "float32":
				o.op = readFloat32
			case "uint64":
				o.op = readUint64
			case "int64":
				o.op = readInt64
			case "float64":
				o.op = readFloat64
			case "bytes":
				o.op = readBytes
			default:
				fmt.Printf("unknown register type '%v' (should be one of "+
					"[u]unt16, [u]int32, [u]int64, float32, float64, bytes)\n",
					splitArgs[1])
				os.Exit(2)
			}

			o.addr, o.quantity, err = parseAddressAndQuantity(splitArgs[2])
			if err != nil {
				fmt.Printf("failed to parse address ('%v'): %v\n", splitArgs[2], err)
				os.Exit(2)
			}

		case "wc", "writeCoil":
			if len(splitArgs) != 3 {
				fmt.Printf("need exactly 2 arguments after writeCoil, got %v\n",
					len(splitArgs)-1)
				os.Exit(2)
			}

			o.op = writeCoil
			o.addr, err = parseUint16(splitArgs[1])
			if err != nil {
				fmt.Printf("failed to parse address ('%v'): %v\n", splitArgs[1], err)
				os.Exit(2)
			}

			switch splitArgs[2] {
			case "true":
				o.coil = true
			case "false":
				o.coil = false
			default:
				fmt.Printf("failed to parse coil value '%v' (should either be true or false)\n",
					splitArgs[2])
				os.Exit(2)
			}

		case "wr", "writeRegister":
			if len(splitArgs) != 4 {
				fmt.Printf("need exactly 3 arguments after writeRegister, got %v\n",
					len(splitArgs)-1)
				os.Exit(2)
			}

			o.addr, err = parseUint16(splitArgs[2])
			if err != nil {
				fmt.Printf("failed to parse address ('%v'): %v\n", splitArgs[2], err)
				os.Exit(2)
			}

			switch splitArgs[1] {
			case "uint16":
				o.op = writeUint16
				o.u16, err = parseUint16(splitArgs[3])

			case "int16":
				o.op = writeInt16
				o.u16, err = parseInt16(splitArgs[3])

			case "uint32":
				o.op = writeUint32
				o.u32, err = parseUint32(splitArgs[3])

			case "int32":
				o.op = writeInt32
				o.u32, err = parseInt32(splitArgs[3])

			case "float32":
				o.op = writeFloat32
				o.f32, err = parseFloat32(splitArgs[3])

			case "uint64":
				o.op = writeUint64
				o.u64, err = parseUint64(splitArgs[3])

			case "int64":
				o.op = writeInt64
				o.u64, err = parseInt64(splitArgs[3])

			case "float64":
				o.op = writeFloat64
				o.f64, err = parseFloat64(splitArgs[3])

			case "bytes":
				o.op = writeBytes
				o.bytes, err = parseHexBytes(splitArgs[3])

			case "string":
				o.op = writeBytes
				o.bytes = []byte(splitArgs[3])
				err = nil

			default:
				fmt.Printf("unknown register type '%v' (should be one of "+
					"[u]unt16, [u]int32, [u]int64, float32, float64, bytes, string)\n",
					splitArgs[1])
				os.Exit(2)
			}

			if err != nil {
				fmt.Printf("failed to parse '%s' as %s: %v\n", splitArgs[3], splitArgs[1], err)
				os.Exit(2)
			}

		case "sleep":
			if len(splitArgs) != 2 {
				fmt.Printf("need exactly 1 argument after sleep, got %v\n",
					len(splitArgs)-1)
				os.Exit(2)
			}

			o.op = sleep
			o.duration, err = time.ParseDuration(splitArgs[1])
			if err != nil {
				fmt.Printf("failed to parse '%s' as duration: %v\n", splitArgs[1], err)
				os.Exit(2)
			}

		case "suid", "setUnitId", "sid":
			if len(splitArgs) != 2 {
				fmt.Printf("need exactly 1 argument after setUnitId, got %v\n",
					len(splitArgs)-1)
				os.Exit(2)
			}

			o.op = setUnitId
			o.unitID, err = parseUnitId(splitArgs[1])
			if err != nil {
				fmt.Printf("failed to parse '%s' as unit id: %v\n", splitArgs[1], err)
				os.Exit(2)
			}

		case "repeat":
			if len(splitArgs) != 1 {
				fmt.Printf("repeat takes no arguments, got %v\n",
					len(splitArgs)-1)
				os.Exit(2)
			}

			o.op = repeat

		case "date":
			if len(splitArgs) != 1 {
				fmt.Printf("date takes no arguments, got %v\n",
					len(splitArgs)-1)
				os.Exit(2)
			}

			o.op = date

		case "scan":
			if len(splitArgs) != 2 {
				fmt.Printf("need exactly 1 argument after scan, got %v\n",
					len(splitArgs)-1)
				os.Exit(2)
			}

			switch splitArgs[1] {
			case "c", "coils":
				o.op = scanBools
				o.isCoil = true
			case "di", "discreteInputs":
				o.op = scanBools
				o.isCoil = false
			case "h", "hr", "holding", "holdingRegisters":
				o.op = scanRegisters
				o.isHoldingReg = true
			case "i", "ir", "input", "inputRegisters":
				o.op = scanRegisters
				o.isHoldingReg = false
			case "s", "sid":
				o.op = scanUnitId
			default:
				fmt.Printf("unknown scan/register type '%s' (valid options <coils|di|hr|ir|s>\n",
					splitArgs[1])
				os.Exit(2)
			}

		case "ping":
			if len(splitArgs) < 2 || len(splitArgs) > 3 {
				fmt.Printf("need 1 or 2 arguments after ping, got %v\n",
					len(splitArgs)-1)
				os.Exit(2)
			}

			o.op = ping
			o.quantity, err = parseUint16(splitArgs[1])
			if err != nil {
				fmt.Printf("failed to parse ping count ('%v'): %v\n", splitArgs[1], err)
				os.Exit(2)
			}

			if o.quantity == 0 {
				fmt.Printf("illegal ping count value (must be >= 1)\n")
				os.Exit(2)
			}

			if len(splitArgs) == 3 {
				o.duration, err = time.ParseDuration(splitArgs[2])
				if err != nil {
					fmt.Printf("failed to parse '%s' as duration: %v\n", splitArgs[2], err)
					os.Exit(2)
				}
			}

		default:
			fmt.Printf("unsupported command '%v'\n", splitArgs[0])
			os.Exit(2)
		}

		runList = append(runList, o)

	}

	// create the modbus client
	client, err = modbus.NewClient(config)
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
	if unitID > 0xff {
		fmt.Printf("set unit id: value '%v' out of range\n", unitID)
		os.Exit(1)
	}
	client.SetUnitID(uint8(unitID))

	// connect to the remote host/open the serial port
	err = client.Open()
	if err != nil {
		fmt.Printf("failed to open client: %v\n", err)
		os.Exit(2)
	}

	for opIdx := 0; opIdx < len(runList); opIdx++ {
		var o *operation = &runList[opIdx]

		switch o.op {
		case readBools:
			var res []bool

			if o.isCoil {
				res, err = client.ReadCoils(o.addr, o.quantity+1)
			} else {
				res, err = client.ReadDiscreteInputs(o.addr, o.quantity+1)
			}
			if err != nil {
				fmt.Printf("failed to read coils/discrete inputs: %v\n", err)
			} else {
				for idx := range res {
					fmt.Printf("0x%04x\t%-5v : %v\n", o.addr+uint16(idx),
						o.addr+uint16(idx),
						res[idx])
				}
			}

		case readUint16, readInt16:
			var res []uint16

			if o.isHoldingReg {
				res, err = client.ReadRegisters(o.addr, o.quantity+1, modbus.HoldingRegister)
			} else {
				res, err = client.ReadRegisters(o.addr, o.quantity+1, modbus.InputRegister)
			}
			if err != nil {
				fmt.Printf("failed to read holding/input registers: %v\n", err)
			} else {
				for idx := range res {
					if o.op == readUint16 {
						fmt.Printf("0x%04x\t%-5v : 0x%04x\t%v\n",
							o.addr+uint16(idx),
							o.addr+uint16(idx),
							res[idx], res[idx])
					} else {
						fmt.Printf("0x%04x\t%-5v : 0x%04x\t%v\n",
							o.addr+uint16(idx),
							o.addr+uint16(idx),
							res[idx], int16(res[idx]))
					}
				}
			}

		case readUint32, readInt32:
			var res []uint32

			if o.isHoldingReg {
				res, err = client.ReadUint32s(o.addr, o.quantity+1, modbus.HoldingRegister)
			} else {
				res, err = client.ReadUint32s(o.addr, o.quantity+1, modbus.InputRegister)
			}
			if err != nil {
				fmt.Printf("failed to read holding/input registers: %v\n", err)
			} else {
				for idx := range res {
					if o.op == readUint32 {
						fmt.Printf("0x%04x\t%-5v : 0x%08x\t%v\n",
							o.addr+(uint16(idx)*2),
							o.addr+(uint16(idx)*2),
							res[idx], res[idx])
					} else {
						fmt.Printf("0x%04x\t%-5v : 0x%08x\t%v\n",
							o.addr+(uint16(idx)*2),
							o.addr+(uint16(idx)*2),
							res[idx], int32(res[idx]))
					}
				}
			}

		case readFloat32:
			var res []float32

			if o.isHoldingReg {
				res, err = client.ReadFloat32s(o.addr, o.quantity+1, modbus.HoldingRegister)
			} else {
				res, err = client.ReadFloat32s(o.addr, o.quantity+1, modbus.InputRegister)
			}
			if err != nil {
				fmt.Printf("failed to read holding/input registers: %v\n", err)
			} else {
				for idx := range res {
					fmt.Printf("0x%04x\t%-5v : %f\n",
						o.addr+(uint16(idx)*2),
						o.addr+(uint16(idx)*2),
						res[idx])
				}
			}

		case readUint64, readInt64:
			var res []uint64

			if o.isHoldingReg {
				res, err = client.ReadUint64s(o.addr, o.quantity+1, modbus.HoldingRegister)
			} else {
				res, err = client.ReadUint64s(o.addr, o.quantity+1, modbus.InputRegister)
			}
			if err != nil {
				fmt.Printf("failed to read holding/input registers: %v\n", err)
			} else {
				for idx := range res {
					if o.op == readUint64 {
						fmt.Printf("0x%04x\t%-5v : 0x%016x\t%v\n",
							o.addr+(uint16(idx)*4),
							o.addr+(uint16(idx)*4),
							res[idx], res[idx])
					} else {
						fmt.Printf("0x%04x\t%-5v : 0x%016x\t%v\n",
							o.addr+(uint16(idx)*4),
							o.addr+(uint16(idx)*4),
							res[idx], int64(res[idx]))
					}
				}
			}

		case readFloat64:
			var res []float64

			if o.isHoldingReg {
				res, err = client.ReadFloat64s(o.addr, o.quantity+1, modbus.HoldingRegister)
			} else {
				res, err = client.ReadFloat64s(o.addr, o.quantity+1, modbus.InputRegister)
			}
			if err != nil {
				fmt.Printf("failed to read holding/input registers: %v\n", err)
			} else {
				for idx := range res {
					fmt.Printf("0x%04x\t%-5v : %f\n",
						o.addr+(uint16(idx)*4),
						o.addr+(uint16(idx)*4),
						res[idx])
				}
			}

		case readBytes:
			var res []byte

			if o.isHoldingReg {
				res, err = client.ReadBytes(o.addr, o.quantity+1, modbus.HoldingRegister)
			} else {
				res, err = client.ReadBytes(o.addr, o.quantity+1, modbus.InputRegister)
			}
			if err != nil {
				fmt.Printf("failed to read holding/input registers: %v\n", err)
			} else {
				for idx := range res {
					if (idx % 16) == 0 {
						fmt.Printf("0x%04x\t%-5v : ",
							o.addr+(uint16(idx/2)), o.addr+(uint16(idx/2)))
					}
					fmt.Printf("%02x", res[idx])

					if (idx%16) == 15 || idx == len(res)-1 {
						fmt.Printf(" <%s>\n",
							decodeString(res[(idx/16*16):(idx/16*16)+(idx%16)+1]))
					} else if (idx % 16) == 7 {
						fmt.Printf(" ")
					}
				}
			}

		case writeCoil:
			err = client.WriteCoil(o.addr, o.coil)
			if err != nil {
				fmt.Printf("failed to write %v at coil address 0x%04x: %v\n",
					o.coil, o.addr, err)
			} else {
				fmt.Printf("wrote %v at coil address 0x%04x\n",
					o.coil, o.addr)
			}

		case writeUint16:
			err = client.WriteRegister(o.addr, o.u16)
			if err != nil {
				fmt.Printf("failed to write %v at register address 0x%04x: %v\n",
					o.u16, o.addr, err)
			} else {
				fmt.Printf("wrote %v at register address 0x%04x\n",
					o.u16, o.addr)
			}

		case writeInt16:
			err = client.WriteRegister(o.addr, o.u16)
			if err != nil {
				fmt.Printf("failed to write %v at register address 0x%04x: %v\n",
					int16(o.u16), o.addr, err)
			} else {
				fmt.Printf("wrote %v at register address 0x%04x\n",
					int16(o.u16), o.addr)
			}

		case writeUint32:
			err = client.WriteUint32(o.addr, o.u32)
			if err != nil {
				fmt.Printf("failed to write %v at address 0x%04x: %v\n",
					o.u32, o.addr, err)
			} else {
				fmt.Printf("wrote %v at address 0x%04x\n",
					o.u32, o.addr)
			}

		case writeInt32:
			err = client.WriteUint32(o.addr, o.u32)
			if err != nil {
				fmt.Printf("failed to write %v at address 0x%04x: %v\n",
					int32(o.u32), o.addr, err)
			} else {
				fmt.Printf("wrote %v at address 0x%04x\n",
					int32(o.u32), o.addr)
			}

		case writeFloat32:
			err = client.WriteFloat32(o.addr, o.f32)
			if err != nil {
				fmt.Printf("failed to write %f at address 0x%04x: %v\n",
					o.f32, o.addr, err)
			} else {
				fmt.Printf("wrote %f at address 0x%04x\n",
					o.f32, o.addr)
			}

		case writeUint64:
			err = client.WriteUint64(o.addr, o.u64)
			if err != nil {
				fmt.Printf("failed to write %v at address 0x%04x: %v\n",
					o.u64, o.addr, err)
			} else {
				fmt.Printf("wrote %v at address 0x%04x\n",
					o.u64, o.addr)
			}

		case writeInt64:
			err = client.WriteUint64(o.addr, o.u64)
			if err != nil {
				fmt.Printf("failed to write %v at address 0x%04x: %v\n",
					int64(o.u64), o.addr, err)
			} else {
				fmt.Printf("wrote %v at address 0x%04x\n",
					int64(o.u64), o.addr)
			}

		case writeFloat64:
			err = client.WriteFloat64(o.addr, o.f64)
			if err != nil {
				fmt.Printf("failed to write %f at address 0x%04x: %v\n",
					o.f64, o.addr, err)
			} else {
				fmt.Printf("wrote %f at address 0x%04x\n",
					o.f64, o.addr)
			}

		case writeBytes:
			err = client.WriteBytes(o.addr, o.bytes)
			if err != nil {
				fmt.Printf("failed to write %v at address 0x%04x: %v\n",
					o.bytes, o.addr, err)
			} else {
				fmt.Printf("wrote %v bytes at address 0x%04x\n",
					len(o.bytes), o.addr)
			}

		case sleep:
			time.Sleep(o.duration)

		case setUnitId:
			client.SetUnitID(o.unitID)

		case repeat:
			// start over
			opIdx = -1

		case date:
			fmt.Printf("%s\n", time.Now().Format(time.RFC3339))

		case scanBools:
			performBoolScan(client, o.isCoil)

		case scanRegisters:
			performRegisterScan(client, o.isHoldingReg)

		case scanUnitId:
			performUnitIdScan(client)

		case ping:
			performPing(client, o.quantity, o.duration)

		default:
			fmt.Printf("unknown operation %v\n", o)
			os.Exit(100)
		}
	}

	return
}

const (
	readBools uint = iota + 1
	readUint16
	readInt16
	readUint32
	readInt32
	readFloat32
	readUint64
	readInt64
	readFloat64
	readBytes
	writeCoil
	writeCoils
	writeUint16
	writeInt16
	writeInt32
	writeUint32
	writeFloat32
	writeInt64
	writeUint64
	writeFloat64
	writeBytes
	setUnitId
	sleep
	repeat
	date
	scanBools
	scanRegisters
	scanUnitId
	ping
)

type operation struct {
	op           uint
	addr         uint16
	isCoil       bool
	isHoldingReg bool
	quantity     uint16
	coil         bool
	u16          uint16
	u32          uint32
	f32          float32
	u64          uint64
	f64          float64
	bytes        []byte
	duration     time.Duration
	unitID       uint8
}

func parseUint16(in string) (u16 uint16, err error) {
	var val uint64

	val, err = strconv.ParseUint(in, 0, 16)
	if err == nil {
		u16 = uint16(val)
		return
	}

	return
}

func parseInt16(in string) (u16 uint16, err error) {
	var val int64

	val, err = strconv.ParseInt(in, 0, 16)
	if err == nil {
		u16 = uint16(int16(val))
	}

	return
}

func parseUint32(in string) (u32 uint32, err error) {
	var val uint64

	val, err = strconv.ParseUint(in, 0, 32)
	if err == nil {
		u32 = uint32(val)
		return
	}

	return
}

func parseInt32(in string) (u32 uint32, err error) {
	var val int64

	val, err = strconv.ParseInt(in, 0, 32)
	if err == nil {
		u32 = uint32(int32(val))
	}

	return
}

func parseFloat32(in string) (f32 float32, err error) {
	var val float64

	val, err = strconv.ParseFloat(in, 32)
	if err == nil {
		f32 = float32(val)
	}

	return
}

func parseUint64(in string) (u64 uint64, err error) {
	var val uint64

	val, err = strconv.ParseUint(in, 0, 64)
	if err == nil {
		u64 = val
	}

	return
}

func parseInt64(in string) (u64 uint64, err error) {
	var val int64

	val, err = strconv.ParseInt(in, 0, 64)
	if err == nil {
		u64 = uint64(val)
	}

	return
}

func parseFloat64(in string) (f64 float64, err error) {
	var val float64

	val, err = strconv.ParseFloat(in, 64)
	if err == nil {
		f64 = val
	}

	return
}

func parseAddressAndQuantity(in string) (addr uint16, quantity uint16, err error) {
	var split = strings.Split(in, "+")

	switch {
	case len(split) == 1:
		addr, err = parseUint16(in)

	case len(split) == 2:
		addr, err = parseUint16(split[0])
		if err != nil {
			return
		}
		quantity, err = parseUint16(split[1])
	default:
		err = errors.New("illegal format")
	}

	return
}

func parseUnitId(in string) (addr uint8, err error) {
	var val uint64

	val, err = strconv.ParseUint(in, 0, 8)
	if err == nil {
		addr = uint8(val)
	}

	return
}

func parseHexBytes(in string) (out []byte, err error) {
	out, err = hex.DecodeString(in)

	return
}

func performBoolScan(client *modbus.Client, isCoil bool) {
	var err error
	var addr uint32
	var val bool
	var count uint
	var regType string

	if isCoil {
		regType = "coil"
	} else {
		regType = "discrete input"
	}

	fmt.Printf("starting %s scan\n", regType)

	for addr = 0; addr <= 0xffff; addr++ {
		if isCoil {
			val, err = client.ReadCoil(uint16(addr))
		} else {
			val, err = client.ReadDiscreteInput(uint16(addr))
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
}

func performRegisterScan(client *modbus.Client, isHoldingReg bool) {
	var err error
	var addr uint32
	var val uint16
	var count uint
	var regType string

	if isHoldingReg {
		regType = "holding register"
	} else {
		regType = "input register"
	}

	fmt.Printf("starting %s scan\n", regType)

	for addr = 0; addr <= 0xffff; addr++ {
		if isHoldingReg {
			val, err = client.ReadRegister(uint16(addr), modbus.HoldingRegister)
		} else {
			val, err = client.ReadRegister(uint16(addr), modbus.InputRegister)
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
}

func performUnitIdScan(client *modbus.Client) {
	var err error
	var countOk uint
	var countErr uint
	var countTimeout uint
	var countGWTimeout uint

	fmt.Println("starting unit id scan")

	for unitID := uint(0); unitID <= 0xff; unitID++ {
		client.SetUnitID(uint8(unitID))

		_, err = client.ReadRegister(0, modbus.InputRegister)
		switch err {
		case nil,
			modbus.ErrIllegalDataAddress,
			modbus.ErrIllegalFunction,
			modbus.ErrIllegalDataValue:
			fmt.Printf("0x%02x (%3v): ok\n", unitID, unitID)
			countOk++

		case modbus.ErrRequestTimedOut:
			countTimeout++

		case modbus.ErrGWTargetFailedToRespond:
			countGWTimeout++

		default:
			fmt.Printf("0x%02x (%3v): %v\n", unitID, unitID, err)
			countErr++
		}
	}

	fmt.Printf("found %v devices (%v errors, %v timeouts, %v gateway timeouts)\n",
		countOk, countErr, countTimeout, countGWTimeout)
}

func performPing(client *modbus.Client, count uint16, interval time.Duration) {
	var err error
	var okCount uint
	var timeoutCount uint
	var otherErrCount uint
	var startTs time.Time
	var ts time.Time
	var rtt time.Duration
	var minRTT time.Duration
	var maxRTT time.Duration
	var avgRTT time.Duration

	fmt.Printf("ping: sending %v requests...\n", count)

	startTs = time.Now()

	for run := uint16(0); run < count; run++ {
		ts = time.Now()
		_, err = client.ReadRegister(0x0000, modbus.HoldingRegister)

		rtt = time.Since(ts)
		avgRTT += rtt

		if run == 0 || rtt < minRTT {
			minRTT = rtt
		}

		if rtt > maxRTT {
			maxRTT = rtt
		}

		switch err {
		// mask illegal data address and illegal function errors since we
		// only care about getting a response from the target device
		// (on which holding reg #0 may or may not exist)
		case nil, modbus.ErrIllegalDataAddress, modbus.ErrIllegalFunction:
			okCount++
			fmt.Printf("ok: seq = %v, time: %v\n",
				run+1, rtt.Round(time.Microsecond))

		case modbus.ErrRequestTimedOut, modbus.ErrGWTargetFailedToRespond:
			timeoutCount++
			fmt.Printf("timeout (%v): seq = %v, time: %v\n",
				err, run+1, rtt.Round(time.Microsecond))

		default:
			otherErrCount++
			fmt.Printf("error (%v): seq = %v, time: %v\n",
				err, run+1, rtt.Round(time.Microsecond))
		}

		if interval > 0 {
			time.Sleep(interval)
		}
	}

	fmt.Printf("--- ping statistics ---\n"+
		"%v queries, %v target replies, %v transmission errors, %v timeouts, time: %v\n",
		count, okCount, otherErrCount, timeoutCount,
		time.Since(startTs).Round(time.Millisecond))

	fmt.Printf("rtt min/avg/max = %v/%v/%v\n",
		minRTT.Round(time.Microsecond),
		(avgRTT / time.Duration(count)).Round(time.Microsecond),
		maxRTT.Round(time.Microsecond))
}

func decodeString(in []byte) (out string) {
	var dec []byte
	var b byte

	for idx := range in {
		if in[idx] >= 0x20 && in[idx] <= 0x7e {
			b = in[idx]
		} else {
			b = '.'
		}

		dec = append(dec, b)
	}

	out = string(dec)

	return
}

func displayHelp() {
	flag.CommandLine.SetOutput(os.Stdout)

	fmt.Println(
		`This tool is a modbus command line interface client meant to allow quick and easy
interaction with modbus devices (e.g. for probing or troubleshooting).

Available options:`)
	flag.PrintDefaults()
	fmt.Printf(
		`

Commands must be given as trailing arguments after any options.

Example: modbus-cli --target=tcp://somehost:502 --timeout=3s rh:uint16:0x100+5 wc:12:true
         Read 6 holding registers at address 0x100 then set the coil at address 12 to true
         on modbus/tcp device somehost port 502, with a timeout of 3s.

Available commands:
* <rc|readCoils>:<addr>[+additional quantity]
  Read coil at address <addr>, plus any additional coils if specified.

  rc:0x100+199         reads 200 coils starting at address 0x100 (hex)
  rc:300               reads 1 coil at address 300 (decimal)

* <rdi|readDiscreteInputs>:<addr>[+additional quantity]
  Read discrete input at address <addr>, plus any additional discrete inputs if specified.

  rdi:0x100+199        reads 200 discrete inputs starting at address 0x100 (hex)
  rdi:300              reads 1 discrete input at address 300 (decimal)

* <rh|readHoldingRegisters>:<type>:<addr>[+additional quantity]
  Read holding registers at address <addr>, plus any additional registers if specified,
  decoded as <type> which should be one of:
  - uint16:            unsigned 16-bit integer,
  - int16:             signed 16-bit integer,
  - uint32:            unsigned 32-bit integer (2 contiguous modbus registers),
  - int32:             signed 32-bit integer (2 contiguous modbus registers),
  - float32:           32-bit floating point number (2 contiguous modbus registers),
  - uint64:            unsigned 64-bit integer (4 contiguous modbus registers),
  - int64:             signed 64-bit integer (4 contiguous modbus registers),
  - float64:           64-bit floating point number (4 contiguous modbus registers),
  - bytes:             string of bytes (2 bytes per modbus register).

  rh:int16:0x300+1     reads 2 consecutive 16-bit signed integers at addresses 0x300 and 0x301
  rh:uint32:20         reads a 32-bit unsigned integer at addresses 20-21 (2 modbus registers)
  rh:float32:500+10    reads 11 32-bit floating point numbers at addresses 500-521
                       (11 * 32bit make for 22 16-bit contiguous modbus registers)

* <ri:readInputRegisters>:<type>:<addr>[+additional quanitity]
  Read input registers at address <addr>, plus any additional registers if specified, decoded
  in the same way as explained above.

  ri:uint16:0x300+1    reads 2 consecutive 16-bit unsigned integers at addresses 0x300 and 0x301
  ri:int32:20          reads a 32-bit signed integer at addresses 20-21 (2 modbus registers)

* <wc|writeCoil>:<addr>:<value>
  Set the coil at address <addr> to either true or false, depending on <value>.

  wc:1:true            writes true to the coil at address 1
  wc:2:false           writes false to the coil at address 2

* <wr:writeRegister>:<type>:<addr>:<value>
  Write <value> to register(s) at address <addr>, using the encoding given by <type>.

  wr:int16:0xf100:-10  writes -10 as a 16-bit signed integer at address 0xf100
                       (1 modbus register)
  wr:int32:0xff00:0xff writes 0xff as a 32-bit signed integer at addresses 0xff00-0xff01
                       (2 consecutive modbus registers)
  wr:float64:100:-3.2  writes -3.2 as a 64-bit float at addresses 100-103
                       (4 consecutive modbus registers)
  wr:bytes:5:fafbfcfd  writes 0xfafbfcfd as a 4-byte string at addresses 5-6
                       (2 consecutive modbus registers)

* sleep:<duration>
  Pause for <duration>, specified as a golang duration string.

  sleep:300s           sleeps for 300 seconds
  sleep:3m             sleeps for 3 minutes
  sleep:3ms            sleeps for 3 milliseconds

* <setUnitId|suid|sid>:<unit id>
  Switch to unit id (slave id) <unit id> for subsequent requests.

  sid:10               selects unit id #10

* repeat
  Restart execution of the given commands.

  rh:uint32:100 sleep:1s repeat  reads a 32-bit unsigned integer at addresses 100-101 and
                                 pauses for one second, forever in a loop.

* date
  Print the current date and time (can be useful for long-running scripts).

* scan:<type>
  Perform a modbus "scan" of the modbus type <type>, which can be one of:
  - "c", "coils",
  - "di", "discreteInputs",
  - "hr", "holdingRegisters",
  - "ir", "inputRegisters",
  - "s", "sid".

  scan:hr              scans the device for holding registers.
  scan:di              scans the device for discrete inputs.

  Read requests are made over the entire address space (65535 addresses).
  Adresses for which a non-error response is received are listed, along with the value received.
  Errors other than Illegal Data Address and Illegal Function are also shown, as they should
  not happen in sane implementations.

  scan:sid             scans the target for devices.

  Scans all unit IDs (0 to 255) using a single read input register request. Addresses responding
  positively or with non-timeout errors are shown, while timeouts and gateway timeouts are ignored.
  This command can be used to find active nodes on RS485 buses, behind gateways or in composite
  devices.

* ping:<count>[:interval]
  Executes <count> modbus reads (1 holding register at address 0x0000), either back to back or
  separated by [interval] if specified, then prints timing and outcome statistics.
  This command can be used to troubleshoot network or serial connections.

Register endianness and word order:
  The endianness of holding/input registers can be specified with --endianness <big|little> and
  defaults to big endian (as per the modbus spec).
  For constructs spanning multiple consecutive registers (namely [u]int32, float32, [u]int64 and
  float64), the word order can be set with --word-order <highfirst|lowfirst> and arbitrarily
  defaults to highfirst (i.e. most significant word first).

Supported transports and associated target schemes:
  - Modbus RTU using a local serial device:               rtu:///path/to/device
  - Modbus RTU over TCP (RTU framing over a TCP socket):  rtuovertcp://host:port
  - Modbus RTU over UDP (RTU framing over an UDP socket): rtuoverudp://host:port
  - Modbus TCP (MBAP):                                    tcp://host:port
  - Modbus TCP over TLS (MBAPS or Modbus Security):       tcp+tls://host:port
  - Modbus TCP over UDP (MBAP over UDP):                  udp://host:port
Note that UDP transports are not part of the Modbus protocol specification.

Examples:
  $ modbus-cli --target tcp://10.100.0.10:502 rh:uint32:0x100+5 rc:0+10 wc:3:true
  Connect to 10.100.0.10 port 502, read 6 consecutive 32-bit unsigned integers at addresses
  0x100-0x10b (12 modbus registers) and 11 coils at addresses 0-10, then set the coil at
  address 3 to true.

  $ modbus-cli --target rtu:///dev/ttyUSB0 --speed 19200 suid:2 rh:uint16:0+7 \
    wr:uint16:0x2:0x0605 suid:3 ri:int16:0+1 sleep:1s repeat
  Open serial port /dev/ttyUSB0 at a speed of 19200 bps and repeat forever:
    select unit id (slave id) 2, read holding registers at addresses 0-7 as 16 bit unsigned
    integers, write 0x605 as a 16-bit unsigned integer at address 2,
    change for unit id 3, read input registers 0-1 as 16-bit signed integers,
    pause for 1s.

  $ modbus-cli --target tcp://somehost:502 scan:hr scan:ir scan:di scan:coils
  Connect to somehost port 502 and perform a scan of all modbus types (namely
  holding registers, input registers, discrete inputs and coils).

  $ modbus-cli --target tcp+tls://securehost:802 --cert client.cert.pem --key client.key.pem \
    --ca ca.cert.pem rh:uint32:0x3000
  Connect to securehost port 802 using modbus/TCP over TLS, using client.cert.pem and
  client.key.pem to authenticate to the server (client auth) and ca.cert.pem to authenticate
  the server, then read holding registers 0x3000-0x3001 as a 32-bit unsigned integer.
  Note that ca.cert.pem can either be a CA (Certificate Authority) or the server (leaf)
  certificate.
`)
}
