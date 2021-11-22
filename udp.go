package modbus

import (
	"net"
	"time"
)

// udpSockWrapper wraps a net.UDPConn (UDP socket) to
// allow transports to consume data off the network socket on
// a byte per byte basis rather than datagram by datagram.
type udpSockWrapper struct {
	leftoverCount int
	rxbuf         []byte
	sock          *net.UDPConn
}

func newUDPSockWrapper(sock net.Conn) (usw *udpSockWrapper) {
	usw = &udpSockWrapper{
		rxbuf: make([]byte, maxTCPFrameLength),
		sock:  sock.(*net.UDPConn),
	}

	return
}

func (usw *udpSockWrapper) Read(buf []byte) (rlen int, err error) {
	var copied int

	if usw.leftoverCount > 0 {
		// if we're holding onto any bytes from a previous datagram,
		// use them to satisfy the read (potentially partially)
		copied = copy(buf, usw.rxbuf[0:usw.leftoverCount])

		if usw.leftoverCount > copied {
			// move any leftover bytes to the beginning of the buffer
			copy(usw.rxbuf, usw.rxbuf[copied:usw.leftoverCount])
		}
		// make a note of how many leftover bytes we have in the buffer
		usw.leftoverCount -= copied
	} else {
		// read up to maxTCPFrameLength bytes from the socket
		rlen, err = usw.sock.Read(usw.rxbuf)
		if err != nil {
			return
		}
		// copy as many bytes as possible to satisfy the read
		copied = copy(buf, usw.rxbuf[0:rlen])

		if rlen > copied {
			// move any leftover bytes to the beginning of the buffer
			copy(usw.rxbuf, usw.rxbuf[copied:rlen])
		}
		// make a note of how many leftover bytes we have in the buffer
		usw.leftoverCount = rlen - copied
	}

	rlen = copied

	return
}

func (usw *udpSockWrapper) Close() (err error) {
	err = usw.sock.Close()

	return
}

func (usw *udpSockWrapper) Write(buf []byte) (wlen int, err error) {
	wlen, err = usw.sock.Write(buf)

	return
}

func (usw *udpSockWrapper) SetDeadline(deadline time.Time) (err error) {
	err = usw.sock.SetDeadline(deadline)

	return
}

func (usw *udpSockWrapper) SetReadDeadline(deadline time.Time) (err error) {
	err = usw.sock.SetReadDeadline(deadline)

	return
}

func (usw *udpSockWrapper) SetWriteDeadline(deadline time.Time) (err error) {
	err = usw.sock.SetWriteDeadline(deadline)

	return
}

func (usw *udpSockWrapper) LocalAddr() (addr net.Addr) {
	addr = usw.sock.LocalAddr()

	return
}

func (usw *udpSockWrapper) RemoteAddr() (addr net.Addr) {
	addr = usw.sock.RemoteAddr()

	return
}
