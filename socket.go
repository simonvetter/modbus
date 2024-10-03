package modbus

import (
	"io"
	"net"
	"time"
)

type socketWrapper struct {
	socket net.Conn
}

func newSocketWrapper(s net.Conn) (sw *socketWrapper) {
	sw = &socketWrapper{
		socket: s,
	}

	return
}

// Closes the socket.
func (sw *socketWrapper) Close() (err error) {
	err = sw.socket.Close()

	return
}

// Reset the input and output buffer
func (sw *socketWrapper) Reset() (err error) {
	// Discards the contents of the link's rx buffer, eating up to 1kB of data.
	// Note that on a serial line, this call may block for up to serialConf.Timeout
	// i.e. 10ms.
	var rxbuf = make([]byte, 1024)

	sw.SetDeadline(time.Now().Add(500 * time.Microsecond))
	io.ReadFull(sw.socket, rxbuf)

	return nil
}

// Reads bytes from the underlying serial port.
// If Read() is called after the deadline, a timeout error is returned without
// attempting to read from the serial port.
// If Read() is called before the deadline, a read attempt to the serial port
// is made. At this point, one of two things can happen:
//   - the serial port's receive buffer has one or more bytes and port.Read()
//     returns immediately (partial or full read),
//   - the serial port's receive buffer is empty: port.Read() blocks for
//     up to 10ms and returns serial.ErrTimeout. The serial timeout error is
//     masked and Read() returns with no data.
//
// As the higher-level methods use io.ReadFull(), Read() will be called
// as many times as necessary until either enough bytes have been read or an
// error is returned (ErrRequestTimedOut or any other i/o error).
func (sw *socketWrapper) Read(rxbuf []byte) (cnt int, err error) {
	cnt, err = sw.socket.Read(rxbuf)

	return
}

// Sends the bytes over the wire.
func (sw *socketWrapper) Write(txbuf []byte) (cnt int, err error) {
	cnt, err = sw.socket.Write(txbuf)

	return
}

// Saves the i/o deadline (only used by Read).
func (sw *socketWrapper) SetDeadline(deadline time.Time) (err error) {
	sw.socket.SetDeadline(deadline)

	return
}
