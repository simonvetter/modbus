## Go modbus server

This repo is a simplified version of [github.com/simonvetter/modbus](https://github.com/simonvetter/modbus)

The main changes are:

- simplified the modbus server
- removed TLS handling (this should be done in the calling application)
- added an `Option` interface to creating the server
- added a `DummyHandler`
- removed modbus client (use [github.com/grid-x/modbus](https://github.com/grid-x/modbus))

### Description

This package is a go implementation of the modbus protocol.
It aims to provide a simple-to-use, high-level API to interact with modbus
devices using native Go types.

Please note that UDP transports are not part of the Modbus specification.
Some devices expect MBAP (modbus TCP) framing in UDP packets while others
use RTU frames instead. The client support both so if unsure, try with
both udp:// and rtuoverudp:// schemes.

The server supports:
- modbus TCP (a.k.a. MBAP)

### License

MIT.
