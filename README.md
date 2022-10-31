## Go modbus stack

### Description
This package is a go implementation of the modbus protocol.
It aims to provide a simple-to-use, high-level API to interact with modbus
devices using native Go types.

Both client and server components are available.

The client supports the following modes:
- modbus RTU (serial, over both RS-232 and RS-485),
- modbus TCP (a.k.a. MBAP),
- modbus TCP over TLS (a.k.a. MBAPS or Modbus Security),
- modbus TCP over UDP (a.k.a. MBAP over UDP),
- modbus RTU over TCP (RTU tunneled in TCP for use with e.g. remote serial
  ports or cheap TCP to serial bridges),
- modbus RTU over UDP (RTU tunneled in UDP).

Please note that UDP transports are not part of the Modbus specification.
Some devices expect MBAP (modbus TCP) framing in UDP packets while others
use RTU frames instead. The client support both so if unsure, try with
both udp:// and rtuoverudp:// schemes.

The server supports:
- modbus TCP (a.k.a. MBAP),
- modbus TCP over TLS (a.k.a. MBAPS or Modbus Security).

A CLI client is available in cmd/modbus-cli.go and can be built with
```bash
$ go build -o modbus-cli cmd/modbus-cli.go
$ ./modbus-cli --help
```

### Getting started
```bash
$ go get github.com/simonvetter/modbus
```

### Using the client

```golang
import (
    "github.com/simonvetter/modbus"
)

func main() {
    var client  *modbus.ModbusClient
    var err      error

    // for a TCP endpoint
    // (see examples/tls_client.go for TLS usage and options)
    client, err = modbus.NewClient(&modbus.ClientConfiguration{
        URL:      "tcp://hostname-or-ip-address:502",
        Timeout:  1 * time.Second,
    })
    // note: use udp:// for modbus TCP over UDP

    // for an RTU (serial) device/bus
    client, err = modbus.NewClient(&modbus.ClientConfiguration{
        URL:      "rtu:///dev/ttyUSB0",
        Speed:    19200,                   // default
        DataBits: 8,                       // default, optional
        Parity:   modbus.PARITY_NONE,      // default, optional
        StopBits: 2,                       // default if no parity, optional
        Timeout:  300 * time.Millisecond,
    })

    // for an RTU over TCP device/bus (remote serial port or
    // simple TCP-to-serial bridge)
    client, err = modbus.NewClient(&modbus.ClientConfiguration{
        URL:      "rtuovertcp://hostname-or-ip-address:502",
        Speed:    19200,                   // serial link speed
        Timeout:  1 * time.Second,
    })
    // note: use rtuoverudp:// for modbus RTU over UDP

    if err != nil {
        // error out if client creation failed
    }

    // now that the client is created and configured, attempt to connect
    err = client.Open()
    if err != nil {
        // error out if we failed to connect/open the device
        // note: multiple Open() attempts can be made on the same client until
        // the connection succeeds (i.e. err == nil), calling the constructor again
        // is unnecessary.
        // likewise, a client can be opened and closed as many times as needed.
    }

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

    // read the same 4 consecutive 16-bit input registers as 2 32-bit integers
    var reg32s  []uint32
    reg32s, err = client.ReadUint32s(100, 2, modbus.INPUT_REGISTER)

    // read the same 4 consecutive 16-bit registers as a single 64-bit integer
    var reg64   uint64
    reg64, err  = client.ReadUint64(100, modbus.INPUT_REGISTER)

    // read the same 4 consecutive 16-bit registers as a slice of bytes
    var regBs   []byte
    regBs, err  = client.ReadBytes(100, 8, modbus.INPUT_REGISTER)

    // by default, 16-bit integers are decoded as big-endian and 32/64-bit values as
    // big-endian with the high word first.
    // change the byte/word ordering of subsequent requests to little endian, with
    // the low word first (note that the second argument only affects 32/64-bit values)
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

    // write 0x0102030405060708 to 16-bit (holding) registers 10 through 13
    // (8 bytes i.e. 4 consecutive modbus registers)
    err         = client.WriteBytes(10, []byte{
        0x01, 0x02, 0x03, 0x04,
        0x05, 0x06, 0x07, 0x08,
    })

    // close the TCP connection/serial port
    client.Close()
}
```
### Using the server component
See:
* [examples/tcp_server.go](examples/tcp_server.go) for a modbus TCP example
* [examples/tls_server.go](examples/tls_server.go) for TLS and Modbus Security features

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

Go object types:
* Booleans (coils and discrete inputs)
* Bytes (input and holding registers)
* Signed/Unisgned 16-bit integers (input and holding registers)
* Signed/Unsigned 32-bit integers (input and holding registers)
* 32-bit floating point numbers (input and holding registers)
* Signed/Unsigned 64-bit integers (input and holding registers)
* 64-bit floating point numbers (input and holding registers)

Byte encoding/endianness/word ordering:
* Little and Big endian for byte slices and 16-bit integers
* Little and Big endian, with and without word swap for 32 and 64-bit
  integers and floating point numbers.

### Logging ###
Both client and server objects will log to stdout by default.
This behavior can be overriden by passing a log.Logger object
through the Logger property of ClientConfiguration/ServerConfiguration.

### TODO (in no particular order)
* Add RTU (serial) support to the server
* Add more tests
* Add diagnostics register support
* Add fifo register support
* Add file register support

### Dependencies
* [github.com/goburrow/serial](https://github.com/goburrow/serial) for access to the serial port (thanks!)

### License
MIT.
