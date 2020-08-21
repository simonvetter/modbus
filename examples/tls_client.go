package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/simonvetter/modbus"
)

/*
 * Modbus client with TLS example.
 *
 * This file is intended to be a demo of the modbus client in TCP+TLS
 * mode. It shows how to load certificates from files and how to
 * configure the client to use them.
 */

func main() {
	var client         *modbus.ModbusClient
	var err            error
	var clientKeyPair  tls.Certificate
	var serverCertPool *x509.CertPool
	var regs           []uint16

	// load the client certificate and its associated private key, which
	// are used to authenticate the client to the server
	clientKeyPair, err = tls.LoadX509KeyPair(
		"certs/client.cert.pem", "certs/client.key.pem")
	if err != nil {
		fmt.Printf("failed to load client key pair: %v\n", err)
		os.Exit(1)
	}

	// load either the server certificate or the certificate of the CA
	// (Certificate Authority) which signed the server certificate
	serverCertPool, err = modbus.LoadCertPool("certs/server.cert.pem")
	if err != nil {
		fmt.Printf("failed to load server certificate/CA: %v\n", err)
		os.Exit(1)
	}

	// create a client targetting host secure-plc on port 802 using
	// modbus TCP over TLS (MBAPS)
	client, err = modbus.NewClient(&modbus.ClientConfiguration{
		// tcp+tls is the moniker for MBAPS (modbus/tcp encapsulated in
		// TLS),
		// 802/tcp is the IANA-registered port for MBAPS.
		URL:           "tcp+tls://secure-plc:802",
		// set the client-side cert and key
		TLSClientCert: &clientKeyPair,
		// set the server/CA certificate
		TLSRootCAs:    serverCertPool,
	})
	if err != nil {
		fmt.Printf("failed to create modbus client: %v\n", err)
		os.Exit(1)
	}

	// now that the client is created and configured, attempt to connect
	err = client.Open()
	if err != nil {
		fmt.Printf("failed to connect: %v\n", err)
		os.Exit(2)
	}

	// read two 16-bit holding registers at address 0x4000
	regs, err = client.ReadRegisters(0x4000, 2, modbus.HOLDING_REGISTER)
	if err != nil {
		fmt.Printf("failed to read registers 0x4000 and 0x4001: %v\n", err)
	} else {
		fmt.Printf("register 0x4000: 0x%04x\n", regs[0])
		fmt.Printf("register 0x4001: 0x%04x\n", regs[1])
	}

	// set register 0x4002 to 500
	err = client.WriteRegister(0x4002, 500)
	if err != nil {
		fmt.Printf("failed to write to register 0x4002: %v\n", err)
	} else {
		fmt.Printf("set register 0x4002 to 500\n")
	}

	// close the connection
	err = client.Close()
	if err != nil {
		fmt.Printf("failed to close connection: %v\n", err)
	}

	os.Exit(0)
}
