package main

import (
	"crypto/x509"
	"crypto/tls"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/simonvetter/modbus"
)

/* Modbus TCP+TLS (MBAPS or Modbus Security) server example.
 *
 * This file is intended to be a demo of the modbus server in a tcp+tls
 * configuration.
 * It shows how to configure and start a server, as well as how to use
 * client roles to perform authorization in the handler.
 * Feel free to use it as boilerplate for simple servers.
 *
 * This server simulates a simple wall clock device, exposing a 32-bit unix
 * timestamp in holding registers #0 and 1.
 * The timestamp is incremented every second by the main loop.
 *
 * Access control is done by way of Modbus Roles, which are encoded in the
 * client certificate as an X509 extension:
 * - any client can read the clock regardless of their role, provided that their
 *   certificate is accepted by the server,
 * - only clients with the "operator" role specified in their certificate can
 *   set the time.
 *
 * Certificates with no, invalid or multiple Modbus Role extensions will have
 * their role set to an empty string (req.ClientRole == "").
 *
 * Requests from clients with certificates not passing TLS verification are
 * rejected at the TLS layer (i.e. before reaching the Modbus layer).
 *
 *
 * The following commands can be used to create self-signed server and client
 * certificates:
 * $ mkdir certs
 *
 * create the server key pair:
 * $ openssl req -x509 -newkey rsa:4096 -sha256 -days 360 -nodes \
 *   -keyout certs/server.key.pem -out certs/server.cert.pem \
 *   -subj "/CN=TEST SERVER CERT DO NOT USE/" -addext "subjectAltName=DNS:localhost" \
 *   -addext "keyUsage=keyCertSign,digitalSignature,keyEncipherment" \
 *   -addext "extendedKeyUsage=critical,serverAuth"
 *
 * create a client certificate with the "user" role:
 * $ openssl req -x509 -newkey rsa:4096 -sha256 -days 360 -nodes \
 *   -keyout certs/user-client.key.pem -out certs/user-client.cert.pem \
 *   -subj "/CN=TEST CLIENT CERT DO NOT USE/" \
 *   -addext "keyUsage=keyCertSign,digitalSignature,keyEncipherment" \
 *   -addext "extendedKeyUsage=critical,clientAuth" \
 *   -addext "1.3.6.1.4.1.50316.802.1=ASN1:UTF8String:user"
 *
 * create another client certificate with the "operator" role:
 * $ openssl req -x509 -newkey rsa:4096 -sha256 -days 360 -nodes \
 *   -keyout certs/operator-client.key.pem -out certs/operator-client.cert.pem \
 *   -subj "/CN=TEST CLIENT CERT DO NOT USE/" \
 *   -addext "keyUsage=keyCertSign,digitalSignature,keyEncipherment" \
 *   -addext "extendedKeyUsage=critical,clientAuth" \
 *   -addext "1.3.6.1.4.1.50316.802.1=ASN1:UTF8String:operator"
 *
 * create a file containing both client certificates (for use by the server as an
 * 'allowed client list'):
 * $ cat certs/user-client.cert.pem certs/operator-client.cert.pem >certs/clients.cert.pem
 *
 * start the server:
 * $ go run examples/tls_server.go
 *
 * in another shell, read the clock with modbus-cli as the 'user' role:
 * $ go run cmd/modbus-cli.go --target tcp+tls://localhost:5802 --cert certs/user-client.cert.pem \
 *   --key certs/user-client.key.pem --ca certs/server.cert.pem rh:uint32:0
 *
 * attempting to set the clock as 'user' should fail with Illegal Function:
 * $ go run cmd/modbus-cli.go --target tcp+tls://localhost:5802 --cert certs/user-client.cert.pem \
 *   --key certs/user-client.key.pem --ca certs/server.cert.pem wr:uint32:0:1598692358
 *
 * setting the clock as 'operator' should succeed:
 * $ go run cmd/modbus-cli.go --target tcp+tls://localhost:5802 --cert certs/operator-client.cert.pem \
 *   --key certs/operator-client.key.pem --ca certs/server.cert.pem wr:uint32:0:1598692358
 *
 * reading the cock as 'operator' should also work:
 * $ go run cmd/modbus-cli.go --target tcp+tls://localhost:5802 --cert certs/operator-client.cert.pem \
 *   --key certs/operator-client.key.pem --ca certs/server.cert.pem rh:uint32:0
 */

func main() {
	var err            error
	var eh             *exampleHandler
	var server         *modbus.ModbusServer
	var serverKeyPair  tls.Certificate
	var clientCertPool *x509.CertPool
	var ticker         *time.Ticker

	// create the handler object
	eh = &exampleHandler{}

	// load the server certificate and its associated private key, which
	// are used to authenticate the server to the client.
	// note that a tls.Certificate object can contain both the cert and its key,
	// which is the case here.
	serverKeyPair, err = tls.LoadX509KeyPair(
		"certs/server.cert.pem", "certs/server.key.pem")
	if err != nil {
		fmt.Printf("failed to load server key pair: %v\n", err)
		os.Exit(1)
	}

	// load TLS client authentication material, which could either be:
	// - the CA (Certificate Authority) certificate(s) used to sign client certs,
	// - the list of allowed client certs, if client certificates are self-signed or
	//   if client certificate pinning is required.
	clientCertPool, err = modbus.LoadCertPool("certs/clients.cert.pem")
	if err != nil {
		fmt.Printf("failed to load CA/client certificates: %v\n", err)
		os.Exit(1)
	}

	// create the server object
	server, err = modbus.NewServer(&modbus.ServerConfiguration{
		// listen on localhost port 5802
		URL:           "tcp+tls://localhost:5802",
		// accept 10 concurrent connections max.
		MaxClients:    10,
		// close idle connections after 1min of inactivity
		Timeout:       60 * time.Second,
		// use serverKeyPair as server certificate + server private key
		TLSServerCert: &serverKeyPair,
		// use the client cert/CA pool to verify client certificates
		TLSClientCAs:  clientCertPool,
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
	fmt.Println("server started")

	ticker	= time.NewTicker(1 * time.Second)
	for {
		<-ticker.C

		// increment the clock every second.
		// lock the handler object while updating the clock register to avoid
		// concurrency issues as each client is served from a dedicated goroutine.
		eh.lock.Lock()
		eh.clock++
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
	lock  sync.RWMutex

	// unix timestamp register, incremented in the main() function above and exposed
	// as a 32-bit holding register (2 consecutive 16-bit modbus registers).
	clock uint32
}

// Holding register handler method.
// This method gets called whenever a valid modbus request asking for a holding register
// operation is received by the server.
func (eh *exampleHandler) HandleHoldingRegisters(req *modbus.HoldingRegistersRequest) (res []uint16, err error) {
	var regAddr	uint16

	// require the "operator" role for write operations (i.e. set the clock).
	if req.IsWrite && req.ClientRole != "operator" {
		fmt.Printf("write access denied: client %s missing the 'operator' role (role: '%s')\n",
			req.ClientAddr, req.ClientRole)
		err = modbus.ErrIllegalFunction
		return
	}

	// since we're manipulating variables accessed from multiple goroutines,
	// acquire a lock to avoid concurrency issues.
	eh.lock.Lock()
	// release the lock upon return
	defer eh.lock.Unlock()

	// loop through `quantity` registers
	for i := 0; i < int(req.Quantity); i++ {
		// compute the target register address
		regAddr	= req.Addr + uint16(i)

		switch regAddr {
		// expose the 16 most-significant bits of the clock in register #0
		case 0:
			if req.IsWrite {
				eh.clock =
					((uint32(req.Args[i]) << 16) & 0xffff0000 |
					 (eh.clock & 0x0000ffff))
			}
			res = append(res, uint16((eh.clock >> 16) & 0x0000ffff))

		// expose the 16 least-significant bits of the clock in register #1
		case 1:
			if req.IsWrite {
				eh.clock =
					(uint32(req.Args[i]) & 0x0000ffff |
					 (eh.clock & 0xffff0000))
			}
			res = append(res, uint16(eh.clock & 0x0000ffff))

		// any other address is unknown
		default:
			err = modbus.ErrIllegalDataAddress
			return
		}
	}

	return
}

// input registers are not used by this server.
func (eh *exampleHandler) HandleInputRegisters(req *modbus.InputRegistersRequest) (res []uint16, err error) {
	// this is the equivalent of saying
	// "input registers are not supported by this device"
	err = modbus.ErrIllegalFunction
	return
}

// coils are not used by this server.
func (eh *exampleHandler) HandleCoils(req *modbus.CoilsRequest) (res []bool, err error) {
	// this is the equivalent of saying
	// "coils are not supported by this device"
	err = modbus.ErrIllegalFunction
	return
}

// discrete inputs are not used by this server.
func (eh *exampleHandler) HandleDiscreteInputs(req *modbus.DiscreteInputsRequest) (res []bool, err error) {
	// this is the equivalent of saying
	// "discrete inputs are not supported by this device"
	err = modbus.ErrIllegalFunction
	return
}
