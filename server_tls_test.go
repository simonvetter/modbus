package modbus

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"testing"
	"time"
)

const (
	clientCertWithRoleOID string = `
-----BEGIN CERTIFICATE-----
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
hG79y2z0UfKJstN+
-----END CERTIFICATE-----
`

	clientKeyWithRoleOID string = `
-----BEGIN PRIVATE KEY-----
MIIJQgIBADANBgkqhkiG9w0BAQEFAASCCSwwggkoAgEAAoICAQCv1ScBlPxYNg64
j7RzEUBSc7Djbaix2GW9vuAYRnDAcw57iT4PQ99BUUPRy11kwFUvg79hY64qSlSU
0FgWq679464VVV0+zzXLW7wRniSlh15OICATlpsA2Uy+dId5IjIUB5TSpLlQ0qti
I7AnMqA1ntItuNnEW92KuOFJftj3ltn07krjHUywHt1RhHPB4fmMfI6CqCPYZxzG
SiGDWemxwPMW/LLveclaEcIVePvOF8ukkoeG7/2/uMRmS5M2XTdeuxaMRZHLdihb
bEbx6W90PotACPa2uH1tyr2HdQl1u124vIulwcjA0m24I7EZ6TuTIYuGHrsV4g1c
qt0SnBC5J0nDwNUBE/cP6W2Ol8Zl5k3a6sO+uKPP7mLkMjE7xCWqgSbktZ/bfZ0o
Q9lyrDxTPhLFFiBzypzD0cNZoocOSl10Av8P4Ri38scy9O2MknsiuBfbR6j9Audb
DFxY59zZiusYBeeCErDY7cOqs5Pm3l0r+HP85DA8waM3WmPY5xjB0u/KSohLfzsN
d4jNqJrFIgjCjo1Xy5bFxAivrkCO3ydPKAeJ+ancVGJyuA4/JyjDDePvdjrWdhVJ
D1OmODcjo4dAHuBb7KZv1DA4nr++z8PMMXUx5rDrEopsXycD8lTEXQ1V90H4Np9U
/8xqB6iS+nMy1EoflCn8Y4kakX8NewIDAQABAoICABMzrufQUmJ7vM3Q+77ZMnIO
qlGb5yFM5Yd8MdLU1nld10YMbdeS7O2gJ0zg7ZkUG/ltZNgI37tElMoPmp8XLqwR
UjCIOv+h91j28qnl4FCnYNgdUANzngfQsz3VUfobjuZ7EXiTfp1h9E9qYFFXiQFy
D7foiPeVpLMCj6/MB3u6YKEL6Oe2imptZHQDh/SzbeI2tAV2wTtfv1e0PsauagP8
c0+eVxgp76BDcjOQG8ec96NIUT6eNNLcJa6aMEBum55fxg2Zh1t10uBxCapfeMl0
Dxb2I6M+sIvt6RbC5D6UMJ79EC8Q45CTKmJCm5Od0eC2eBs0fe/c2OK20h+3JWg0
eybKuX1GXTSkd5padBKZPJumINFIDUlaiQrRCFr4ZpcUJqBcrCkkRq2wyvuwAEiZ
IvNqcJqjqzZFhjhZKrxVIv3C9av6v0JQLdrbquYZbRji6KaSCcU2xCdbOmSvcr9w
909Lz9gwdCFYVWWRCLmfA8FSR+hDGLW8fT2CfbijYYBpGR0W9zLXCR7DD7Hx0bYj
ZJg+ri3Yw8yZ1mTT6ZLmlz3HEtSNJD1kD/QcpCanwvkGYXRO5FOVaMXGx5QVWs33
fS9ChLesujKQ9ye5jFc5tw1rmClcTNtWipllctdPZGzfaEs6a+LfTbpB1/zd2eBK
XkeMAYWp+ES8XXYFUToRAoIBAQDjj3MdNR5sM9m87E/IS3+1GulkfQ/M39BFmCtc
O7gvWiK7asrP2TFU9vjU9zEomo4ABCYKNaJtMqtGxH9EapsTepdjb1UrjYpmKyuJ
SyTk8+iWRLQa3RnyWkE2MXqIKQQH37uB5/Id009is5f1X89mfGOXQuldDxQy3ygC
OeYOAVOl5fZH0NqikTDnpaViKyAqhp7bbRG0KBvqtYBj/k930auL3Ls6eONMlFfB
9IzYeM3lcbiYmfMhOPuYReFgpC62SQDWBALQagcS2Uh4vOP2fSd6ragi8telBeLW
fWl5TSy4tNurbgtYh8WdizBDO+DD0is+b8HSiWUPJfPH7QDDAoIBAQDFzrq8JvMm
cJWnSqsCnWD6CRj8w0CvXe9IjwK7cquUc/xsyZFbHbKe1kKf0+UxywbT+1uSaJ19
ZbvyLAi16+S27aHI5SX3R2SnZQPA2GOWqimSHu5HAMfTOsqqbkLImRUnzl2Vl3lW
+AN/4FMvAlA1HotI3EiuQxLWstG5RNeo58sVMobaiZd7+xnBsov7MKgKC90eBmTR
uxbQmPJUFLhefpTly1E72rbYwZ2a2AOBq/WJ74+Gb7A6DoQQmSFqRHe5X1e5v8L4
nUwiUd60J9ACOMKYCGzPkwXSPvfqcmuSL1KKupsJAcVV+AcC4qmNPfDWI6A87aha
b4a+78g4u3TpAoIBACTZ1SV0tbGGEAu1JRJlj4/PhN4+FnHyCLNMejEchq48ZYV+
PMu9+2wr9o3eXfqaVMaR5Wsf1mbinrP+HDIDJYvY/W0f2WYNLM1wzkMUhSwCh7bV
92imR45kqUzSZGpqYfm4dJAL9Lx5vNBaDxCwbFDHcgVL06i7SWUXmE4L/EJmWppy
DBkDLHTJGGda/tZP74yTcmRMXGKVYDf5HoqS42Ge9a3XmAZXD1AWccO6C5j+rzEp
4l/sBmBp7uxw3Jee3uWsGtONoLsJgI2/3CmZRT1kdSE7wA+wzdUuh9Z+RrdbFRPw
TeaMEpBKpGjn4m/w4Ww0u8YHqRakI1Z5qenFaqsCggEAH3WUf04Wh7uKIZQfhIfx
H3MI9VI8XGetIbYU8ij3nuGfeNHJ+1rKyLY83Fx/7B5lFJu6YZufyIzAinB0ZjKB
KpK6k0/WbPB+0pyfLzF7DUA84k9nCAXYwgBssRReLLckBTOt8JeppapGLDVKJYTR
qtETx9+483YZben8ruGDBwruYo2pouIVJJO38fVqi+WeJBLk9NyBdlWx+DUK/VJa
TDUHi1B9t+49/FU2sqS+UgY+Q9TE19W1ilY6rMUd6l+/Rs0iD5mu8YlazW6F49Md
Iu1SDYnxfEXevCRlm3TdJN+/2e55r8IHV3fd7ZiM7Li4L+Z0mpwVlWR9YqqSBmvR
2QKCAQEAv1P9zlYiOjK5MlpP8rfWyb2CuUCT3DG9k7+RZMPL6QCp5Fc/xINsttJc
bPSwhuWjYYE2DpenZAcn4Mf8JhhdUf+yijLVZYSDINgfUMgrmSTETRB4X28KYrGJ
UG3kz2IQnbIfPPrekFcL87h6dc88lfq5U/inPqSoQYdE99XD8iTY3Tb2ESD8B7Zk
Xh9uF519h8lnUA+/O6r3aLJ/d0ApKoLWancvenrkwe3jgc1MGUG0kjNLNCN310YW
lKNiMZCOhCMEGxo7pm1KBpPxxb+8Mo2ydxC2s4jhX748aMe1MvlTg5+IYUkqVDBq
isPLG4c6aPGxSbHirNfl6tBSngDy+A==
-----END PRIVATE KEY-----
`
)

// TestTLSServer tests the TLS layer of the modbus server.
func TestTLSServer(t *testing.T) {
	var err error
	var server *ModbusServer
	var serverKeyPair tls.Certificate
	var client1KeyPair tls.Certificate
	var client2KeyPair tls.Certificate
	var clientCp *x509.CertPool
	var serverCp *x509.CertPool
	var th *tlsTestHandler
	var c1 *ModbusClient
	var c2 *ModbusClient
	var regs []uint16
	var coils []bool

	th = &tlsTestHandler{}

	// load server keypair (from client_tls_test.go)
	serverKeyPair, err = tls.X509KeyPair([]byte(serverCert), []byte(serverKey))
	if err != nil {
		t.Errorf("failed to load test server key pair: %v", err)
		return
	}

	// load the first client keypair (from client_tls_test.go)
	// this client cert doesn't have any Modbus Role extension
	client1KeyPair, err = tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		t.Errorf("failed to load test client key pair: %v", err)
		return
	}

	// load the second client keypair (defined above)
	// this client cert has an "operator2" Modbus Role extension
	client2KeyPair, err = tls.X509KeyPair(
		[]byte(clientCertWithRoleOID), []byte(clientKeyWithRoleOID))
	if err != nil {
		t.Errorf("failed to load test client key pair: %v", err)
		return
	}

	// load the server cert into the client CA cert pool to get the server cert
	// accepted by clients
	clientCp = x509.NewCertPool()
	if !clientCp.AppendCertsFromPEM([]byte(serverCert)) {
		t.Errorf("failed to load test server cert into cert pool")
	}

	// start with an empty server cert pool initially to reject the client
	// certificate
	serverCp = x509.NewCertPool()

	server, err = NewServer(&ServerConfiguration{
		URL:           "tcp+tls://localhost:5802",
		MaxClients:    2,
		TLSServerCert: &serverKeyPair,
		TLSClientCAs:  serverCp,
	}, th)
	if err != nil {
		t.Errorf("failed to create server: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Errorf("failed to start server: %v", err)
	}

	// create 2 modbus clients
	c1, err = NewClient(&ClientConfiguration{
		URL:           "tcp+tls://localhost:5802",
		TLSClientCert: &client1KeyPair,
		TLSRootCAs:    clientCp,
	})
	if err != nil {
		t.Errorf("failed to create client: %v", err)
	}
	c2, err = NewClient(&ClientConfiguration{
		URL:           "tcp+tls://localhost:5802",
		TLSClientCert: &client2KeyPair,
		TLSRootCAs:    clientCp,
	})
	if err != nil {
		t.Errorf("failed to create client: %v", err)
	}

	// attempt to connect and use the first client. since its cert
	// is not trusted by the server, a TLS error should occur on the first
	// request.
	err = c1.Open()
	if err != nil {
		t.Errorf("c1.Open() should have succeeded")
	}
	coils, err = c1.ReadCoils(0, 5)
	if err == nil {
		t.Error("c1.ReadCoils() should have failed")
	}
	c1.Close()

	// now place both client certs in the server's authorized client list
	// to get them past the TLS client cert validation procedure
	if !serverCp.AppendCertsFromPEM([]byte(clientCert)) {
		t.Errorf("failed to load client#1 cert into cert pool")
	}
	if !serverCp.AppendCertsFromPEM([]byte(clientCertWithRoleOID)) {
		t.Errorf("failed to load client#2 cert into cert pool")
	}

	// connect both clients: should succeed
	err = c1.Open()
	if err != nil {
		t.Error("c1.Open() should have succeeded")
	}

	err = c2.Open()
	if err != nil {
		t.Error("c2.Open() should have succeeded")
	}

	// client #2 (with 'operator2' role) should have read/write access to coils while
	// client #1 (without role) should only be able to read.
	err = c1.WriteCoil(0, true)
	if err != ErrIllegalFunction {
		t.Errorf("c1.WriteCoil() should have failed with %v, got: %v",
			ErrIllegalFunction, err)
	}

	coils, err = c1.ReadCoils(0, 5)
	if err != nil {
		t.Errorf("c1.ReadCoils() should have succeeded, got: %v", err)
	}
	if coils[0] {
		t.Errorf("coils[0] should have been false")
	}

	err = c2.WriteCoil(0, true)
	if err != nil {
		t.Errorf("c2.WriteCoil() should have succeeded, got: %v", err)
	}

	coils, err = c2.ReadCoils(0, 5)
	if err != nil {
		t.Errorf("c2.ReadCoils() should have succeeded, got: %v", err)
	}
	if !coils[0] {
		t.Errorf("coils[0] should have been true")
	}

	coils, err = c1.ReadCoils(0, 5)
	if err != nil {
		t.Errorf("c1.ReadCoils() should have succeeded, got: %v", err)
	}
	if !coils[0] {
		t.Errorf("coils[0] should have been true")
	}

	// client #1 should only be allowed access to holding registers of unit id #1
	// while client#2 should be allowed access to holding registers of unit ids #1 and #4
	c1.SetUnitId(1)
	err = c1.WriteRegister(2, 100)
	if err != nil {
		t.Errorf("c1.WriteRegister() should have succeeded, got: %v", err)
	}

	c1.SetUnitId(4)
	err = c1.WriteRegister(2, 200)
	if err != ErrIllegalFunction {
		t.Errorf("c1.WriteRegister() should have failed with %v, got: %v",
			ErrIllegalFunction, err)
	}

	c2.SetUnitId(1)
	regs, err = c2.ReadRegisters(1, 2, HOLDING_REGISTER)
	if err != nil {
		t.Errorf("c2.ReadRegisters() should have succeeded, got: %v", err)
	}
	if regs[0] != 0 || regs[1] != 100 {
		t.Errorf("unexpected register values: %v", regs)
	}

	c2.SetUnitId(4)
	err = c2.WriteRegister(2, 200)
	if err != nil {
		t.Errorf("c2.WriteRegister() should have succeeded, got: %v", err)
	}

	regs, err = c2.ReadRegisters(1, 2, HOLDING_REGISTER)
	if err != nil {
		t.Errorf("c2.ReadRegisters() should have succeeded, got: %v", err)
	}
	if regs[0] != 0 || regs[1] != 200 {
		t.Errorf("unexpected register values: %v", regs)
	}

	// close the server and all client connections
	server.Stop()

	// make sure all underlying TCP client connections have been freed
	time.Sleep(10 * time.Millisecond)
	server.lock.Lock()
	if len(server.tcpClients) != 0 {
		t.Errorf("expected 0 client connections, saw: %v", len(server.tcpClients))
	}
	server.lock.Unlock()

	// cleanup
	c1.Close()
	c2.Close()

	return
}

type tlsTestHandler struct {
	coils      [10]bool
	holdingId1 [10]uint16
	holdingId4 [10]uint16
}

func (th *tlsTestHandler) HandleCoils(req *CoilsRequest) (res []bool, err error) {
	// coils access is allowed to any client with a valid cert, but
	// the "operator2" role is required to write
	if req.IsWrite && req.ClientRole != "operator2" {
		err = ErrIllegalFunction
		return
	}

	if req.Addr+req.Quantity > uint16(len(th.coils)) {
		err = ErrIllegalDataAddress
		return
	}

	for i := 0; i < int(req.Quantity); i++ {
		if req.IsWrite {
			th.coils[int(req.Addr)+i] = req.Args[i]
		}
		res = append(res, th.coils[int(req.Addr)+i])
	}

	return
}

func (th *tlsTestHandler) HandleDiscreteInputs(req *DiscreteInputsRequest) (res []bool, err error) {
	// there are no digital inputs on this device
	err = ErrIllegalDataAddress

	return
}

func (th *tlsTestHandler) HandleHoldingRegisters(req *HoldingRegistersRequest) (res []uint16, err error) {
	// gate unit id #4 behind the "operator2" role while access to unit id #1
	// is allowed to any valid cert
	if req.UnitId == 0x04 {
		if req.ClientRole != "operator2" {
			err = ErrIllegalFunction
			return
		}

		if req.Addr+req.Quantity > uint16(len(th.holdingId4)) {
			err = ErrIllegalDataAddress
			return
		}

		for i := 0; i < int(req.Quantity); i++ {
			if req.IsWrite {
				th.holdingId4[int(req.Addr)+i] = req.Args[i]
			}
			res = append(res, th.holdingId4[int(req.Addr)+i])
		}
	} else if req.UnitId == 0x01 {
		if req.Addr+req.Quantity > uint16(len(th.holdingId1)) {
			err = ErrIllegalDataAddress
			return
		}

		for i := 0; i < int(req.Quantity); i++ {
			if req.IsWrite {
				th.holdingId1[int(req.Addr)+i] = req.Args[i]
			}
			res = append(res, th.holdingId1[int(req.Addr)+i])
		}
	} else {
		err = ErrIllegalFunction
		return
	}

	return
}

func (th *tlsTestHandler) HandleInputRegisters(req *InputRegistersRequest) (res []uint16, err error) {
	// there are no inputs registers on this device
	err = ErrIllegalDataAddress

	return
}

func TestServerExtractRole(t *testing.T) {
	var ms *ModbusServer
	var pemBlock *pem.Block
	var x509Cert *x509.Certificate
	var err error
	var role string

	ms = &ModbusServer{
		logger: newLogger("test-server-role-extraction", nil),
	}

	// load a client cert without role OID
	pemBlock, _ = pem.Decode([]byte(clientCert))
	if err != nil {
		t.Errorf("failed to decode client cert: %v", err)
		return
	}

	x509Cert, err = x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		t.Errorf("failed to parse client cert: %v", err)
		return
	}

	// calling extractRole on a cert without role extension should return an
	// empty string (see R-23 of the MBAPS spec)
	role = ms.extractRole(x509Cert)
	if role != "" {
		t.Errorf("role should have been empty, got: '%s'", role)
	}

	// load a certificate with a single role extension of "operator2"
	pemBlock, _ = pem.Decode([]byte(clientCertWithRoleOID))
	if err != nil {
		t.Errorf("failed to decode client cert: %v", err)
		return
	}

	x509Cert, err = x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		t.Errorf("failed to parse client cert: %v", err)
		return
	}

	role = ms.extractRole(x509Cert)
	if role != "operator2" {
		t.Errorf("role should have been 'operator2', got: '%s'", role)
	}

	// build a certificate with multiple Modbus Role extensions: they should
	// all be rejected
	x509Cert = &x509.Certificate{
		Extensions: []pkix.Extension{
			{
				Id: modbusRoleOID,
				Value: []byte{
					0x0c, 0x04, 0x66, 0x77, 0x67, 0x78,
					// ^ ASN1:UTF8String
					//     ^ length
					//          ^ 4-byte string 'fwgx'
				},
			},
			{
				Id: modbusRoleOID,
				Value: []byte{
					0x0c, 0x02, 0x66, 0x67,
					// ^ ASN1:UTF8String
					//     ^ length
					//          ^ 2-byte string 'fwwf'
				},
			},
		},
	}

	role = ms.extractRole(x509Cert)
	if role != "" {
		t.Errorf("role should have been empty, got: '%s'", role)
	}

	// build a certificate with a single Modbus Role extension of the wrong
	// type: the role should be rejected
	x509Cert = &x509.Certificate{
		Extensions: []pkix.Extension{
			{
				Id: modbusRoleOID,
				Value: []byte{
					0x13, 0x04, 0x66, 0x77, 0x67, 0x78,
					// ^ ASN1:PrintableString
					//     ^ length
					//          ^ 4-byte string 'fwgx'
				},
			},
		},
	}

	role = ms.extractRole(x509Cert)
	if role != "" {
		t.Errorf("role should have been empty, got: '%s'", role)
	}

	// build a certificate with a single, short Modbus Role extension: the role
	// should be rejected
	x509Cert = &x509.Certificate{
		Extensions: []pkix.Extension{
			{
				Id: modbusRoleOID,
				Value: []byte{
					0x0c,
					// ^ ASN1:UTF8String
					//    ^ missing length + payload bytes
				},
			},
		},
	}

	role = ms.extractRole(x509Cert)
	if role != "" {
		t.Errorf("role should have been empty, got: '%s'", role)
	}

	// build a certificate with one bad Modbus Role extension (short) and one
	// valid: they should both be rejected
	x509Cert = &x509.Certificate{
		Extensions: []pkix.Extension{
			{
				Id: modbusRoleOID,
				Value: []byte{
					0x0c,
					// ^ ASN1:UTF8String
					//    ^ missing length + payload bytes
				},
			},
			{
				Id: modbusRoleOID,
				Value: []byte{
					0x0c, 0x02, 0x66, 0x67,
					// ^ ASN1:UTF8String
					//     ^ length
					//          ^ 2-byte string 'fwwf'
				},
			},
		},
	}

	role = ms.extractRole(x509Cert)
	if role != "" {
		t.Errorf("role should have been empty, got: '%s'", role)
	}

	// build a certificate with a single, valid Modbus Role extension: it should be
	// accepted
	x509Cert = &x509.Certificate{
		Extensions: []pkix.Extension{
			{
				Id: modbusRoleOID,
				Value: []byte{
					0x0c, 0x04, 0x66, 0x77, 0x67, 0x78,
					// ^ ASN1:UTF8String
					//     ^ length
					//          ^ 4-byte string 'fwgx'
				},
			},
		},
	}

	role = ms.extractRole(x509Cert)
	if role != "fwgx" {
		t.Errorf("role should have been 'fwgx', got: '%s'", role)
	}

	return
}
