## Golang modbus client library

### Description
This package is a golang implementation of the modbus protocol.
It aims to provide a simple-to-use, high-level API to interact with modbus
devices using native Go types.

So far, only the client part is implemented, using either TCP or RTU (serial)
as transports.

A CLI client is available in cmd/modbus-cli.go and can be built with
```bash
$ go build -o modbus-cli cmd/modbus-cli.go
$ ./modbus-cli --help
```

### Getting started

```bash
$ go get github.com/simonvetter/modbus
```

```golang
import (
    "github.com/simonvetter/modbus"
)

func main() {
    var client  *modbus.ModbusClient
    var err      error

    // for a TCP endpoint
    client, err = modbus.NewClient(&modbus.ClientConfiguration{
        URL:      "tcp://hostname-or-ip-address:502",
        Timeout:  1 * time.Second,
    })

    // for an RTU (serial) device/bus
    client, err = modbus.NewClient(&modbus.ClientConfiguration{
        URL:      "rtu:///dev/ttyUSB0",
        Speed:    9600,                    // default
        DataBits: 8,                       // default, optional
        Parity:   modbus.PARITY_NONE,      // default, optional
        StopBits: 2,                       // default if no parity, optional
        Timeout:  300 * time.Millisecond,
    })

    // read a single 16-bit holding register at address 100
    var reg16   uint16
    reg16, err  = client.ReadRegister(100, modbus.HOLDING_REGISTER)
    if err != nil {
      // error out
    } else {
      // use value
      fmt.Printf("value: %v", reg16)        // as unsigned integer
      fmt.Printf("value: %v", int16(reg16)) // as signed integer
    }

    // read 4 consecutive 16-bit input registers starting at address 100
    var reg16s  []uint16
    reg16s, err = client.ReadRegisters(100, 4, modbus.INPUT_REGISTER)

    // read the same 4 consecutive 16-bit input registers as 2 32-bit registers
    var reg32s  []uint32
    reg32s, err = client.ReadUint32s(100, 2, modbus.INPUT_REGISTER)

    // by default, 16-bit registers are decoded as big-endian and 32-bit registers
    // as big-endian with the high word first.
    // change the byte/word ordering of subsequent requests to little endian, with
    // the low word first (note that the second argument only affects 32-bit registers)
    client.SetEncoding(modbus.LITTLE_ENDIAN, modbus.LOW_WORD_FIRST)

    // read the same 4 consecutive 16-bit input registers as 2 32-bit floats
    var fl32s   []float32
    fl32s, err  = client.ReadFloat32s(100, 2, modbus.INPUT_REGISTER)

    // write -200 to 16-bit (holding) register 100, as a signed integer
    var s int16 = -200
    err         = client.WriteRegister(100, uint16(s))

    // Switch to unit ID (a.k.a. slave ID) #4
    client.SetUnitId(4)

    // write 3 floats to registers 100 to 105
    err         = client.WriteFloat32s(100, []float32{
	3.14,
        1.1,
	-783.22,
    })

    // close the TCP connection/serial port
    client.Close()
}
```
### Supported function codes, golang object types and endianness/word ordering
Function codes:
* Read coils (0x01)
* Read discrete inputs (0x02)
* Read holding registers (0x03)
* Read input registers (0x04)
* Write single coil (0x05)
* Write single register (0x06)
* Write multiple coils (0x0f)
* Write multiple registers (0x10)

Golang object types:
* Booleans (coils and discrete inputs)
* Signed/Unisgned 16-bit integers (input and holding registers)
* Signed/Unsigned 32-bit integers (input and holding registers)
* 32-bit floating point numbers (input and holding registers)

Byte encoding/endianness/word ordering:
* Little and Big endian for 16-bit integers
* Little and Big endian, with and without word swap for 32-bit integers and 32-bit
  floating point numbers.

### TODO (in no particular order)
* Add integration tests
* Add a server object
* Add RTU over TCP transport support
* Add diagnostics register support
* Add fifo register support
* Add file register support

### Dependencies
* [github.com/goburrow/serial](https://github.com/goburrow/serial) for access to the serial port (thanks!)

### License
MIT.
