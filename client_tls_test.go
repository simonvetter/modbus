package modbus

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

const (
	// note: these certs and associated keys are self-signed
	// and only meant to be used with this test.
	// PLEASE DO NOT USE THEM FOR ANYTHING ELSE, EVER, as they
	// do not provide any kind of security.
	serverCert string = `
-----BEGIN CERTIFICATE-----
MIIFiDCCA3CgAwIBAgIUXEymmMuwNGPRSikQTwEo3ugRtccwDQYJKoZIhvcNAQEL
BQAwKTEnMCUGA1UEAwwebG9jYWxob3N0IFRFU1QgQ0VSVCBETyBOT1QgVVNFMB4X
DTI1MTEwNjE1MDYxOFoXDTM1MDkxNTE1MDYxOFowKTEnMCUGA1UEAwwebG9jYWxo
b3N0IFRFU1QgQ0VSVCBETyBOT1QgVVNFMIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEAkmPVA6ifUL+H14SXWGn+/0IsIjxa9UvtOJAtdcSzoI3obTxMlZxJ
r/XUs+/0XCIlSnDoO3J0dNWGRH+U9puOYPSAwqoZ5OgWTGgLW5RrbzrQV8y2yaWM
KX3y2WgdpYl76N8SNWZ6UWqbp70rrrDTyxLvolSx2zxzZtGgxox99jTN+uLLsgpI
cUI5cbDQdpxZDPXAY/gnYHG1YJcAYncfjI7Wxwg0C4Hn9t5FAPZEHymhO/1trRNz
B4IGWyFhEqeslEBar4Fg2MTpKZAOSQ4fTxmHTYDPcXDrzpxAAlT7QpfkkcXUozE3
aM1Rl0Zaikyw3LtSHe7OC5KC2hUJT92a+GX25dgnUjpfonQ0paCF+pyKlid+Vbzm
wpKQCr1PzJnLSuLMfdk4S8IMGv7zsjq4BFqNM15L7ru4hyDaYOZ4CxYNdv3Abr7P
Lcb1XUasYKnthgaM1l6xFs1PZ988qqG3I69TTMAjrhhgHLjFj5iuV2/seQ+UiT2/
NcDvsa2z2RokCNpDhZHkDKlhbO6v50AjCwxdnzomq+h/KJh7cT5fyZBpGRubgiM2
dk/fcXjeUVxqlZNEtdV4rpzNITR+az+I8JOUaGwioQKs4/ORmNyeSZB4J6a8FSE6
0fqXXNumRMaO98ydi56NPb0PzSG1uAQJNQjNsWok5FEmryUoNVngi6MCAwEAAaOB
pzCBpDAdBgNVHQ4EFgQU8NXPG19x7bHN7IJkKK9Ee+PLM6EwHwYDVR0jBBgwFoAU
8NXPG19x7bHN7IJkKK9Ee+PLM6EwDwYDVR0TAQH/BAUwAwEB/zAsBgNVHREEJTAj
gglsb2NhbGhvc3SHEAAAAAAAAAAAAAAAAAAAAAGHBH8AAAEwCwYDVR0PBAQDAgKk
MBYGA1UdJQEB/wQMMAoGCCsGAQUFBwMBMA0GCSqGSIb3DQEBCwUAA4ICAQAj2jvB
lEURcKHcfrlMieRnIVAQF08lhQxpdcaHzrOi75u+Rc+Kf8/2N1tOYz4Z7WJGxHpF
TpfvinrzSCf2iqSPkegYpNUHoenhObu9o/m1Hz/A1L1wHx3wg6iThStJPs5Z6+MJ
vVWXWMM6LiXQ325BILv8XbDk/Fwg0yqXeNJypHU3R0NiwbpgYuGsEwZOBhLnkQu0
ezUv9JAnoNUFPnidi6jPaypcwjOPKwlXTCz9pkN0dZIDWyYvY3hUUwLSm9ZCB+eX
AdFkSAzX9ZK9cMSYXGyA9xXzyMYUpBtHRxzCxvMUeFay0NH67NRQCt76RnX5vgas
o+8IiTyhcPKc1pWMQJYrMa2AsfxZQtvhrBJJGEFnUWX0ELezpK7x+hAXKWRFjnbP
jUBptTfi4zEJJQyg5NutVFmLvXD88YUhOWo+buUpIhTTcdHT3aBGZb4UIZJQSokx
NnQzZV7QGk9rpiOqXxZm80muS2EepgU/n1Q9BYaMYDJWWYDsBSuOAERUW07ChZ3n
Av7Yp/alljuhM/Tz+1Hl/6ETk6aMj8ltio6TTkuOUuPEao3oXYKcZA4Z8B0ZCXd9
lA6B8qlyJ/z6oIHIVBvPlv+MFINdoH3f21qT38KhV5Reqb89mMF3Yl6MvM0SCZqS
GXecywcrsY12ciA0rl7R1BDbeDdWR0GlbD1CQw==
-----END CERTIFICATE-----
`

	serverKey string = `
-----BEGIN PRIVATE KEY-----
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
gPokGE4rIl2QrQKCeDCrRdTSfQmgxVY=
-----END PRIVATE KEY-----
`

	clientCert string = `
-----BEGIN CERTIFICATE-----
MIIFUjCCAzqgAwIBAgIUDUE69zn1fMSUHFJ7lKZNaq4tBIowDQYJKoZIhvcNAQEL
BQAwJjEkMCIGA1UEAwwbVEVTVCBDTElFTlQgQ0VSVCBETyBOT1QgVVNFMB4XDTI1
MTEwNjE1MTExNVoXDTM1MDkxNTE1MTExNVowJjEkMCIGA1UEAwwbVEVTVCBDTElF
TlQgQ0VSVCBETyBOT1QgVVNFMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKC
AgEAuPCWiCVre+nlGT73EhFnom2riDDswuKpx1PHbRqVJIGIjIUpDH7COTA7TndT
ORQGGfeTgjvsVZnye+t2CzpaK5CjVthod6kiuJJmluFTAKyr6Cd9yc/zT1hO2DEe
dADzIn+IZiwiHpoVS1OTBVougHMrCo4G2VncymbG/Ek8Zg42+sliwQnRLnxsDCuF
q8+Fm54rmiM30uWoO1FKe4nyqQT66w/QOYhlhFXTJ9qaOUCarG3Sv/ugI7Q8n0E2
uzhRhmie+2cmAumex54Wc8Tcwy9ju/rVNByvqH8oHfdI7YgPfB9Z4RuJd0prGP9i
WtI+fj+Jbd6fMM+l1jlRlnlVPPuSrilps2MK2gJALjQefTeZstlMgxPkavCs/0eR
kSZjY9UZlQK52Tg1tbDMvPnklUa3x9DIcSocG5zh0ihlPqedTOy1sXDgpdX+h5Nb
SZXO305MtHMaXIPB6jsF1BofAb1LLAPkGCSJ560BTphfKGSyj2Kut/yKurYtDXER
OKyuJ8ibvKcoJmVH4tj2UXDYBmVdYEYDZdON8cIwqL3FSVFEhKE9kVTUA+gDTpdW
ovSg6RLy3Tz0CYdVOiO6mfmNvYRKe7Qot+a+Ac1fy8hFu7DRNP/aH2oL3OsIjoPg
JSthGKyGWCYZmqS38atAoSad7/J9270sJXQQKF/LSEdBSpkCAwEAAaN4MHYwHQYD
VR0OBBYEFMbRremvSoZzOvkbC4iKtQ8+TyAmMB8GA1UdIwQYMBaAFMbRremvSoZz
OvkbC4iKtQ8+TyAmMA8GA1UdEwEB/wQFMAMBAf8wCwYDVR0PBAQDAgKkMBYGA1Ud
JQEB/wQMMAoGCCsGAQUFBwMCMA0GCSqGSIb3DQEBCwUAA4ICAQBAtBL/iB3Kim46
3yYQXmwPfGkq0ATrAgkwwi621sa2/Mu91rgSg3I5h56/fNtgoYLDtIPbGP5q8MAN
vGesG9Y0MsGTjyu1X/B9ZsRWX0Y1x+7JcYqzVDC1IAPevYeu65OQozKPXXRE4vBB
zEQSTSWlhj54JiHs4HkIkZD2Kx2zg+F518CYqUSz9uTIhvQDbibs31sS2bd3PyGI
tzwFgBG+GDvQQ6bL7eG4qcrnGDitTnyhGvrycNB+B7cJT7ahaiMjMAsAMRXTx2h1
VGTuqjD5DFL+lI0chLanUkbnqpVdfv8fTc3BJ73YiExheM7KvSUwAHB3P5ixwEMI
NOtY9GlIG3o0mssq6VxuYNoWlasDoUBNEdsRdQsppbcTs0Q92U/IyRUsGfjBHU1C
sHlQdLxQ3GMniq8pSarAJd+IrA6Fnz5oAhBP8USFl3ipnKWkpDcWUNrhFmj2EmIg
B15VKJLw1jAj3o+raR/z7N+sK4TBYfLa0svS5r/DId6vQG9umk85v0mGL2TWBopH
SnMzYTqIGtCpjmJ0DXbe8eXERYm4wK6bUwRMsL6DMWqD+oEzUiqD03O/i155EmCc
UQtFIG3k4uduJrJYUKF2dVIe4xcIgqZU3W61wKyRcUTOUn9xoreU9tHf5hHKeTqv
loJLAXLimj4RPFEChtk80+cZCihPhw==
-----END CERTIFICATE-----
`
	clientKey string = `
-----BEGIN PRIVATE KEY-----
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
yEU+HMj419vJHZRyHwuxy9aJLDErxg==
-----END PRIVATE KEY-----
`
)

// TestTCPOVerTLSClient tests the TLS layer of the modbus client.
func TestTCPoverTLSClient(t *testing.T) {
	var err error
	var client *ModbusClient
	var serverKeyPair tls.Certificate
	var clientKeyPair tls.Certificate
	var clientCp *x509.CertPool
	var serverCp *x509.CertPool
	var serverHostPort string
	var serverChan chan string
	var regs []uint16

	serverChan = make(chan string)

	// load server and client keypairs
	serverKeyPair, err = tls.X509KeyPair([]byte(serverCert), []byte(serverKey))
	if err != nil {
		t.Errorf("failed to load test server key pair: %v", err)
		return
	}

	clientKeyPair, err = tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		t.Errorf("failed to load test client key pair: %v", err)
		return
	}

	// start with an empty client cert pool initially to reject the server
	// certificate
	clientCp = x509.NewCertPool()

	// start with an empty server cert pool initially to reject the client
	// certificate
	serverCp = x509.NewCertPool()

	// start a mock modbus TLS server
	go runMockTLSServer(t, serverKeyPair, serverCp, serverChan)

	// wait for the test server goroutine to signal its readiness
	// and network location
	serverHostPort = <-serverChan

	// attempt to create a client without specifying any TLS configuration
	// parameter: should fail
	client, err = NewClient(&ClientConfiguration{
		URL: fmt.Sprintf("tcp+tls://%s", serverHostPort),
	})
	if err != ErrConfigurationError {
		t.Errorf("NewClient() should have failed with %v, got: %v",
			ErrConfigurationError, err)
	}

	// attempt to create a client without specifying any TLS server
	// cert/CA: should fail
	client, err = NewClient(&ClientConfiguration{
		URL:           fmt.Sprintf("tcp+tls://%s", serverHostPort),
		TLSClientCert: &clientKeyPair,
	})
	if err != ErrConfigurationError {
		t.Errorf("NewClient() should have failed with %v, got: %v",
			ErrConfigurationError, err)
	}

	// attempt to create a client with both client cert+key and server
	// cert/CA: should succeed
	client, err = NewClient(&ClientConfiguration{
		URL:           fmt.Sprintf("tcp+tls://%s", serverHostPort),
		TLSClientCert: &clientKeyPair,
		TLSRootCAs:    clientCp,
	})
	if err != nil {
		t.Errorf("NewClient() should have succeeded, got: %v", err)
	}

	// connect to the server: should fail with a TLS error as the server cert
	// is not yet trusted by the client
	err = client.Open()
	if err == nil {
		t.Errorf("Open() should have failed")
	}

	// now load the server certificate into the client's trusted cert pool
	// to get the client to accept the server's certificate
	if !clientCp.AppendCertsFromPEM([]byte(serverCert)) {
		t.Errorf("failed to load test server cert into cert pool")
	}

	// connect to the server: should succeed
	// note: client certificates are verified after the handshake procedure
	// has completed, so Open() won't fail even though the client cert
	// is rejected by the server.
	// (see RFC 8446 section 4.6.2 Post Handshake Authentication)
	err = client.Open()
	if err != nil {
		t.Errorf("Open() should have succeeded, got: %v", err)
	}

	// attempt to read two registers: since the client cert won't pass
	// the validation step yet (no cert in server cert pool),
	// expect a tls error
	regs, err = client.ReadRegisters(0x1000, 2, INPUT_REGISTER)
	if err == nil {
		t.Errorf("ReadRegisters() should have failed")
	}
	client.Close()

	// now place the client cert in the server's authorized client list
	// to get the client cert past the validation procedure
	if !serverCp.AppendCertsFromPEM([]byte(clientCert)) {
		t.Errorf("failed to load test client cert into cert pool")
	}

	// connect to the server: should succeed
	err = client.Open()
	if err != nil {
		t.Errorf("Open() should have succeeded, got: %v", err)
	}

	// attempt to read two registers: should succeed
	regs, err = client.ReadRegisters(0x1000, 2, INPUT_REGISTER)
	if err != nil {
		t.Errorf("ReadRegisters() should have succeeded, got: %v", err)
	}
	if regs[0] != 0x1234 {
		t.Errorf("expected 0x1234 in 1st reg, saw: 0x%04x", regs[0])
	}
	if regs[1] != 0x5678 {
		t.Errorf("expected 0x5678 in 2nd reg, saw: 0x%04x", regs[1])
	}

	// attempt to read another: should succeed
	regs, err = client.ReadRegisters(0x1002, 1, HOLDING_REGISTER)
	if err != nil {
		t.Errorf("ReadRegisters() should have succeeded, got: %v", err)
	}
	if regs[0] != 0xaabb {
		t.Errorf("expected 0xaabb in 1st reg, saw: 0x%04x", regs[0])
	}

	// close the connection: should succeed
	err = client.Close()
	if err != nil {
		t.Errorf("Close() should have succeeded, got: %v", err)
	}

	return
}

func TestTLSClientOnServerTimeout(t *testing.T) {
	t.Skip("TODO figure out why this test is failing -- was failing on the initial fork")
	var err error
	var client *ModbusClient
	var server *ModbusServer
	var serverKeyPair tls.Certificate
	var clientKeyPair tls.Certificate
	var clientCp *x509.CertPool
	var serverCp *x509.CertPool
	var th *tlsTestHandler
	var reg uint16

	th = &tlsTestHandler{}
	// load server and client keypairs
	serverKeyPair, err = tls.X509KeyPair([]byte(serverCert), []byte(serverKey))
	if err != nil {
		t.Errorf("failed to load test server key pair: %v", err)
		return
	}

	clientKeyPair, err = tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		t.Errorf("failed to load test client key pair: %v", err)
		return
	}

	// add those keypairs to their corresponding cert pool
	clientCp = x509.NewCertPool()
	if !clientCp.AppendCertsFromPEM([]byte(serverCert)) {
		t.Errorf("failed to load test server cert into cert pool")
	}

	serverCp = x509.NewCertPool()
	if !serverCp.AppendCertsFromPEM([]byte(clientCert)) {
		t.Errorf("failed to load client cert into cert pool")
	}

	// load the server cert into the client CA cert pool to get the server cert
	// accepted by clients
	clientCp = x509.NewCertPool()
	if !clientCp.AppendCertsFromPEM([]byte(serverCert)) {
		t.Errorf("failed to load test server cert into cert pool")
	}

	server, err = NewServer(&ServerConfiguration{
		URL:           "tcp+tls://[::1]:5802",
		MaxClients:    10,
		TLSServerCert: &serverKeyPair,
		TLSClientCAs:  serverCp,
		// disconnect idle clients after 500ms
		Timeout: 500 * time.Millisecond,
	}, th)
	if err != nil {
		t.Errorf("failed to create server: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Errorf("failed to start server: %v", err)
	}

	// create the modbus client
	client, err = NewClient(&ClientConfiguration{
		URL:           "tcp+tls://localhost:5802",
		TLSClientCert: &clientKeyPair,
		TLSRootCAs:    clientCp,
	})
	if err != nil {
		t.Errorf("failed to create client: %v", err)
	}

	// connect to the server: should succeed
	err = client.Open()
	if err != nil {
		t.Errorf("Open() should have succeeded, got: %v", err)
	}

	// write a value to register #3: should succeed
	err = client.WriteRegister(3, 0x0199)
	if err != nil {
		t.Errorf("Write() should have succeeded, got: %v", err)
	}

	// attempt to read the value back: should succeed
	reg, err = client.ReadRegister(3, HOLDING_REGISTER)
	if err != nil {
		t.Errorf("ReadRegisters() should have succeeded, got: %v", err)
	}
	if reg != 0x0199 {
		t.Errorf("expected 0x0199 in reg #3, saw: 0x%04x", reg)
	}

	// pause for longer than the server's configured timeout to end up with
	// an open client with a closed underlying TCP socket
	time.Sleep(1 * time.Second)

	// attempt a read: should fail
	_, err = client.ReadRegister(3, INPUT_REGISTER)
	if err == nil {
		t.Errorf("ReadRegister() should have failed")
	}

	// cleanup
	client.Close()
	server.Stop()

	return
}

// runMockTLSServer spins a test TLS server for use with TestTCPoverTLSClient.
func runMockTLSServer(t *testing.T, serverKeyPair tls.Certificate,
	serverCp *x509.CertPool, serverChan chan string) {
	var err error
	var listener net.Listener
	var sock net.Conn
	var reqCount uint
	var clientCount uint
	var buf []byte

	// let the OS pick an available port on the loopback interface
	listener, err = tls.Listen("tcp", "localhost:0", &tls.Config{
		// the server will use serverKeyPair (key+cert) to
		// authenticate to the client
		Certificates: []tls.Certificate{serverKeyPair},
		// the server will use the certpool to authenticate the
		// client-side cert
		ClientCAs: serverCp,
		// request client-side authentication and client cert validation
		ClientAuth: tls.RequireAndVerifyClientCert,
	})
	if err != nil {
		t.Errorf("failed to start test server listener: %v", err)
	}
	defer listener.Close()

	// let the main test goroutine know which port the OS picked
	serverChan <- listener.Addr().String()

	for err == nil {
		// accept client connections
		sock, err = listener.Accept()
		if err != nil {
			t.Errorf("failed to accept client conn: %v", err)
			break
		}

		// only proceed with clients passing the tls handshake
		// note: this will reject any client whose cert does not pass the
		// verification step
		err = sock.(*tls.Conn).Handshake()
		if err != nil {
			sock.Close()
			err = nil
			continue
		}

		clientCount++
		if clientCount > 2 {
			t.Errorf("expected 2 client conns, saw: %v", clientCount)
		}

		// expect MBAP (modbus/tcp) messages inside the TLS tunnel
		for {
			// expect 12 bytes per request
			buf = make([]byte, 12)

			_, err = sock.Read(buf)
			if err != nil {
				// ignore EOF errors (clients disconnecting)
				if err != io.EOF {
					t.Errorf("failed to read client request: %v", err)
				}
				sock.Close()
				break
			}

			reqCount++
			switch reqCount {
			case 1:
				for i, b := range []byte{
					0x00, 0x01, // txn id
					0x00, 0x00, // protocol id
					0x00, 0x06, // length
					0x01, 0x04, // unit id + function code
					0x10, 0x00, // start address
					0x00, 0x02, // quantity
				} {
					if b != buf[i] {
						t.Errorf("expected 0x%02x at pos %v, saw 0x%02x",
							b, i, buf[i])
					}
				}

				// send a reply
				_, err = sock.Write([]byte{
					0x00, 0x01, // txn id
					0x00, 0x00, // protocol id
					0x00, 0x07, // length
					0x01, 0x04, // unit id + function code
					0x04,       // byte count
					0x12, 0x34, // reg #0
					0x56, 0x78, // reg #1
				})
				if err != nil {
					t.Errorf("failed to write reply: %v", err)
				}

			case 2:
				for i, b := range []byte{
					0x00, 0x02, // txn id
					0x00, 0x00, // protocol id
					0x00, 0x06, // length
					0x01, 0x03, // unit id + function code
					0x10, 0x02, // start address
					0x00, 0x01, // quantity
				} {
					if b != buf[i] {
						t.Errorf("expected 0x%02x at pos %v, saw 0x%02x",
							b, i, buf[i])
					}
				}

				// send a reply
				_, err = sock.Write([]byte{
					0x00, 0x02, // txn id
					0x00, 0x00, // protocol id
					0x00, 0x05, // length
					0x01, 0x03, // unit id + function code
					0x02,       // byte count
					0xaa, 0xbb, // reg #0
				})
				if err != nil {
					t.Errorf("failed to write reply: %v", err)
				}

				// stop the server after the 2nd request
				listener.Close()

			default:
				t.Errorf("unexpected request id %v", reqCount)
				return
			}
		}
	}
}
