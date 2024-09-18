package modbus

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"
)

// LoadCertPool loads a certificate store from a file into a CertPool object.
func LoadCertPool(filePath string) (cp *x509.CertPool, err error) {
	var buf []byte

	// read the entire cert store, which may contain zero, one
	// or more certificates
	buf, err = ioutil.ReadFile(filePath)
	if err != nil {
		return
	}

	if len(buf) == 0 {
		err = fmt.Errorf("%v: empty file", filePath)
		return
	}

	// add these certs to the pool
	cp = x509.NewCertPool()
	cp.AppendCertsFromPEM(buf)

	// let the caller know if no usable certificate was found
	if len(cp.Subjects()) == 0 {
		err = fmt.Errorf("%v: no certificate found", filePath)
		return
	}

	return
}

// tlsSockWrapper wraps a TLS socket to work around odd error handling in
// TLSConn on internal connection state corruption.
// tlsSockWrapper implements the net.Conn interface to allow its
// use by the modbus TCP transport.
type tlsSockWrapper struct {
	sock net.Conn
}

func newTLSSockWrapper(sock net.Conn) (tsw *tlsSockWrapper) {
	tsw = &tlsSockWrapper{
		sock: sock,
	}

	return
}

func (tsw *tlsSockWrapper) Read(buf []byte) (rlen int, err error) {
	rlen, err = tsw.sock.Read(buf)

	return
}

func (tsw *tlsSockWrapper) Write(buf []byte) (wlen int, err error) {
	wlen, err = tsw.sock.Write(buf)

	// since write timeouts corrupt the internal state of TLS sockets,
	// any subsequent read/write operation will fail and return the same write
	// timeout error (see https://pkg.go.dev/crypto/tls#Conn.SetWriteDeadline).
	// this isn't all that helpful to clients, which may be tricked into
	// retrying forever, treating timeout errors as transient.
	// to avoid this, close the TLS socket after the first write timeout.
	// this ensures that clients 1) get a timeout error on the first write timeout
	// and 2) get an ErrNetClosing "use of closed network connection" on subsequent
	// operations.
	if err != nil && os.IsTimeout(err) {
		tsw.sock.Close()
	}

	return
}

func (tsw *tlsSockWrapper) Close() (err error) {
	err = tsw.sock.Close()

	return
}

func (tsw *tlsSockWrapper) SetDeadline(deadline time.Time) (err error) {
	err = tsw.sock.SetDeadline(deadline)

	return
}

func (tsw *tlsSockWrapper) SetReadDeadline(deadline time.Time) (err error) {
	err = tsw.sock.SetReadDeadline(deadline)

	return
}

func (tsw *tlsSockWrapper) SetWriteDeadline(deadline time.Time) (err error) {
	err = tsw.sock.SetWriteDeadline(deadline)

	return
}

func (tsw *tlsSockWrapper) LocalAddr() (addr net.Addr) {
	addr = tsw.sock.LocalAddr()

	return
}

func (tsw *tlsSockWrapper) RemoteAddr() (addr net.Addr) {
	addr = tsw.sock.RemoteAddr()

	return
}
