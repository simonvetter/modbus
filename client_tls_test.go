package modbus

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"fmt"
	"net"
	"testing"
	"time"
)

const (
	// note: these certs and associated keys are self-signed
	// and only meant to be used with this test.
	// PLEASE DO NOT USE THEM FOR ANYTHING ELSE, EVER, as they
	// do not provide any kind of security.
	serverCert string = `-----BEGIN CERTIFICATE-----
MIIFkzCCA3ugAwIBAgIUWnvnN1r9czWyX7TGS+AwTNICd4wwDQYJKoZIhvcNAQEL
BQAwKTEnMCUGA1UEAwwebG9jYWxob3N0IFRFU1QgQ0VSVCBETyBOT1QgVVNFMB4X
DTIwMDgyMTA5NTEyMVoXDTQwMDgxNjA5NTEyMVowKTEnMCUGA1UEAwwebG9jYWxo
b3N0IFRFU1QgQ0VSVCBETyBOT1QgVVNFMIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEA4D+a4wqvwxhyMhN4Z6EG7pIU1TfL7hV2MH4Izx1sDHGaUu+318SE
Egn85Zn1PbYAvYqlN+Ti3pCH5/tSJJHD4XVGcXtp3Wswt5MTXX8Ny3f1v3ZeQggp
nTy2tyODTulCQBg5L+8FTgJM2mJR0D+dryswiWgDVBLxg5W9p7icff30n/LHtEGd
jTkVbkaG798iGaIIeI6YS1wjMfsPWGWpG9SVoC3bkHN2NL2apecCLsoZpb+DiKdT
1rBG2pNeDseGpSWKwF/2/HeJsw+tD4okbtfYA7uURmRyqv1rxAXmclZXHFHpUL8l
Vt69g+ER0sXmLavM2Jj3iss6RF2MP6ghVUAcaciPbuDCn6+vnxCE6L2Gyr9G6Aur
rOBl/nRj3BHK9agp0fLzhIKgfCKMzCU5mo/UFlJIKbKIRJcdF5LNF3A9wD0K3Rv/
2bIvaXdWwIgUQ+zX3V3cDuMatCs/F2jGE5FejGaNeA7ixfpdCtybBpzGewLB49NB
AIFBboJBdfW3QuqQBM32GFmbwM4cZpdxr97cZTDJh3Age7e8BSWPO195IJEKWNSn
bnWCDNG5J4G6MBf9AfC/ljJCrOIEN4wTXP6EF4vaMq/VWz674j7QvR9q9aKB1lNn
bdKd/LMH+jmgG8bGuy01Tj12/JBgzgG0KI72364wuJlTjneqkTpCncMCAwEAAaOB
sjCBrzAdBgNVHQ4EFgQUEFEWxaWofRhSTd2+ZWmaH14OKAswHwYDVR0jBBgwFoAU
EFEWxaWofRhSTd2+ZWmaH14OKAswDwYDVR0TAQH/BAUwAwEB/zAsBgNVHREEJTAj
gglsb2NhbGhvc3SHEAAAAAAAAAAAAAAAAAAAAAGHBH8AAAEwCwYDVR0PBAQDAgWg
MBMGA1UdJQQMMAoGCCsGAQUFBwMBMAwGA1UdEwEB/wQCMAAwDQYJKoZIhvcNAQEL
BQADggIBACYfWN0rK/OTXDZEQHLWLe6SDailqJSULVRhzxevo3i07MhsQCARofXH
A37b0q/GHhLsKUEsjB+TYip90eKaRV1XDgnMOdbQxVCKY5IVTmobnoj5ZLja1AgD
4I3CTxx3tqxLxRmV1Bre1tIqHnalA6kC6HJAKW+R0biPSccchIjW2ljB1SXBcS2T
RjAsIfbEl9sDhyLl8jaOaOBLPLS5PNgRs4XJ8/ps9dDCNyeOizGVzAsgaeTxXadC
Y505cxoWQR1atYerjqVr0XolCTkefOapsNJzH3YXF2mxKJCVxARv8Ns8e5WwIWw2
r1ESi6M1qca5CutEgbBdUp7NyF44HJ9O3EsG+CFO6XRn6aaUvmin6vufKk29usRm
L3RWqBH1vz3vQzVLfEzXnJwnxDwZWcBrGx3RjKAL+O+hWHc3Qh6+AfI0yRX4j0MR
7IMHESf2xkCtw58w1t+OA1GBZ7hBX4zRiAQ89hk8UzRMw45yQ3cPkAp9u+PhrY1i
9dcDqvPueaSDoRMl7VvHyQ+2SeQF7mc3Xx6iAm9HPBmuVWVpX32g9jbu0xfzWhng
DXf3U5zg6BsG3gR5omPwbApKBlGckRY+ZuarhxPeczBx6KVIOKgvafybKrCsbso2
oq2sBRSZveoEKZDOmZpsUP2jYrcgrybnurcoN6g1Chl28V5rNITd
-----END CERTIFICATE-----`

	serverKey string = `-----BEGIN PRIVATE KEY-----
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
yAqUh6zMzzcee67KOWWZMTuPQuu1n/m1
-----END PRIVATE KEY-----`

	clientCert string = `
-----BEGIN CERTIFICATE-----
MIIFXzCCA0egAwIBAgIUQzQeLPGsr6OmD5NtrsiYMFVwATwwDQYJKoZIhvcNAQEL
BQAwJjEkMCIGA1UEAwwbVEVTVCBDTElFTlQgQ0VSVCBETyBOT1QgVVNFMB4XDTIw
MDgyMTA5NTI0NVoXDTQwMDgxNjA5NTI0NVowJjEkMCIGA1UEAwwbVEVTVCBDTElF
TlQgQ0VSVCBETyBOT1QgVVNFMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKC
AgEA5yBIvheS8d2T8lBn9BOsdi1mMmhUHyqxdx9YFgwIV0NYb9s3/J83Jf/Focob
DM4fdy7iuSECX8/KUoymQmn26ivzmI4iLJ0LsBbUhTzMO9lo82Vg18E4Ab4GQVMz
LWcxnt1wt2EYJ5nq72c1h9K27pIecDDZ9DtBD1j0d3cuoo8HIVzsUfZRp80H9b5H
WuY2nMPZC3jp6HlsVSHCbkuscs/d7dDGK4tsanmyqVfBNmJhNKvo2GwMM9vf82li
dh6OwrqUNeMXkDU8vQ/xMGWfZ+Xpu0vXx3pQ5SX3WVDZFMuk7/mMBZwUhnXh+yjC
R1MVIRJvjijnkFzSSXctoysl/Mrc3QW5QmPmoa8KVWL8pSc5oaMMdX9bXo9omPf1
XlmjKMvEUu2IbUQcaDFtVAKzR0UGcYEFh/QCIu7WV0pA24DfiX3r0Lv1GHHB6X+3
+zUH7ZcaajzVQM/crCB/VLQiZODg6EgFtx1woll5hES/I6l9Me7UKySlxY4wTjIk
k/cwIN6R0dHGDny30fIkOl0vseo6SKtIkApYfe2tAASatbNdpkJQbMjUkm1IlcYj
ZZdn7yssjLjmAWn6pz19GbW2sAGQ/D4k/rN2hOpCc3NSP2FMIeWwp6TUHUG2ZFck
79j/Fo2bPNTPiFWxUW6h6gWHDgZg8UFz3FXvaeUM0URFOjsCAwEAAaOBhDCBgTAd
BgNVHQ4EFgQUZkTFhQZ4vaia2hiIQrZnSfx2nOAwHwYDVR0jBBgwFoAUZkTFhQZ4
vaia2hiIQrZnSfx2nOAwDwYDVR0TAQH/BAUwAwEB/zALBgNVHQ8EBAMCBaAwEwYD
VR0lBAwwCgYIKwYBBQUHAwIwDAYDVR0TAQH/BAIwADANBgkqhkiG9w0BAQsFAAOC
AgEAhDVloT4TqZL66A/N8GSbiAALYVM4VoQlaiYiNrtwcNBAKrveN/RJlVxvYC9f
ybsvHz+wf/UAkWsQqd7pacKMlctNYoavpooO2ketQErpz+Ysmb+yGlL5u9DwQI36
bsw/sxZjA4uunEM3yySVZf2k5j97HzBhhv24dlgxCPyu5tOAj83bM2QLTc5H7/KZ
ZEhMcrXN0+QUI9np3WYPKAPMJNODSMGD8mMqpjRufxDH0jhPhX4R4qvhHT+/OrLE
CwLTwtgZ8BnRS2b16QEGpvT7bu5EWZda4vgXQEeuMpEgUmwPOm2JS9QZguXrhA6u
Jd/12gbNEowQCt0qig1K2/ouYc3YKvCq/GuDPZnVq0nXEgSom4+g4UpU92zHARSy
CjfEW+rD9ay0ipzl6wxV09ZoQOoFwztf/AO89gl2CDtcw1J+mB8KcP2Pme+lWZ9m
mj7+ed+lubE5kBIK/H2EojEUceGmdluqD/T6bUaAR6edLuS0z4MKFTNlbbZq9QiS
vb6vr137SqCw56gFvYzxxOS2037QHAHk9dZz4+ik6BLXOQmHY1s59y/iAV3CrWwf
wVi6BS05QtOQW1nzeUU4DyMz4aAuBs88iGqDlipzkMreyYTG/66WpKCp/nezSn5H
cufNpBGKcE0Ww/H/GgMvKe/nB7HEJQqoAxVDeq75WFiHQrs=
-----END CERTIFICATE-----
`
	clientKey string = `-----BEGIN PRIVATE KEY-----
MIIJQgIBADANBgkqhkiG9w0BAQEFAASCCSwwggkoAgEAAoICAQDnIEi+F5Lx3ZPy
UGf0E6x2LWYyaFQfKrF3H1gWDAhXQ1hv2zf8nzcl/8WhyhsMzh93LuK5IQJfz8pS
jKZCafbqK/OYjiIsnQuwFtSFPMw72WjzZWDXwTgBvgZBUzMtZzGe3XC3YRgnmerv
ZzWH0rbukh5wMNn0O0EPWPR3dy6ijwchXOxR9lGnzQf1vkda5jacw9kLeOnoeWxV
IcJuS6xyz93t0MYri2xqebKpV8E2YmE0q+jYbAwz29/zaWJ2Ho7CupQ14xeQNTy9
D/EwZZ9n5em7S9fHelDlJfdZUNkUy6Tv+YwFnBSGdeH7KMJHUxUhEm+OKOeQXNJJ
dy2jKyX8ytzdBblCY+ahrwpVYvylJzmhowx1f1tej2iY9/VeWaMoy8RS7YhtRBxo
MW1UArNHRQZxgQWH9AIi7tZXSkDbgN+JfevQu/UYccHpf7f7NQftlxpqPNVAz9ys
IH9UtCJk4ODoSAW3HXCiWXmERL8jqX0x7tQrJKXFjjBOMiST9zAg3pHR0cYOfLfR
8iQ6XS+x6jpIq0iQClh97a0ABJq1s12mQlBsyNSSbUiVxiNll2fvKyyMuOYBafqn
PX0ZtbawAZD8PiT+s3aE6kJzc1I/YUwh5bCnpNQdQbZkVyTv2P8WjZs81M+IVbFR
bqHqBYcOBmDxQXPcVe9p5QzRREU6OwIDAQABAoICABiXYcYAAh2D4uLsVTMuCLKG
QBJq8VBjnYA8MIYf/58xRi6Yl4tkcVy0qxV8yIYDRGvM7EigT31cQX2pA2ObnK7r
wD5iGRbAGudAdpo6jsxrZHRJPBWYtFnTGx1GOfLBwRDTJNQOG6DTCqEwTQzHibk2
iNCNEhOfXlvArjorzyVyrGKLXYWW/Lcq5IbsGPF9/x+M4wIKenDGwpUIQ4SyvoV0
wns0NHGbowxtKGpGMQOVUhxlkh+810uJQHnIo7ZHqA7mBTD6mZ45W94N3S62EVDf
sI/CERJjXEoVUQ0Kwh4pUMJLve823SQ1VLcBbjJij6P2LzJj/cdpaOJyMMPkqmTY
cRUxtM/n5TQ9DVI867BKvDz/TplGaYKFEW1pmMZ2fH2w+YT/gZY8+YkVQsPVuj6c
sedxoAF6fUP4t/ROZkibyMyTJ4v2wF+tjugXddOM5DN9C4BuYJJgZ8tDWy+nf5Oe
weik6cheBXLYJd/LjZop1s+2pGe5/EjDiI16jdoVCwdPKTjNNjeopZr/Od0/C8jj
mljOYyf0wqrQsEBrmOxCtL0QL7gDg6kjYEWR2zbYiAoQu/iDZgdSb0V1RqAagiJP
qLMILSPDi8KHAh+k8z7JjSBXvhffSMtP+zI4iKfibTVAiLw5tUGzDuUrehgysSK7
n9ETpNwwQZuQLx26KJrZAoIBAQD8IlVicwTDjKndJ4an1TOrUCU9qe6OKTO5RQoS
Jma+qOdrcWBIttbWZ8K9CcJUcdhImnp1Es+GTnMqCP2UG9UMysJAZeM3d5uwlpe3
8s6Ju9x2n3yywCzSKgO0mragkdOUEyf+dHF6LgC098EsrfiDchvKinHgSpcFqmSB
1lC9QgyeikvmlSbwJcVWQXWN7dnP4dXL1j6ej6WB9ITKa7cztiBNWmAGAhJE8GUG
tmLG/zk7MV9cCzzJ7a2k9a63C6EGR3b4svs+SFEx8IzRHvzBHH+qdCAdJhzuD7Yp
jASbHNZ0b1yBEJNYTSIUrygFi0+6Ol5AMQusJGGHo0UErrhlAoIBAQDqq32o2sJa
0BsOEyXhAgZgZP+WocMmsduWqfvTCoze+MfD/f6vTBJZ45A98vd3xUHw/CsyS/dM
vjTRQCa0QCflgm1LiwbNR50QhXZ63AJ1ob2+FvDZFrPXJJcykUdMY1UwQh3gHKk/
VYsm/s8L+VDDZXQlrcDQyHxVvML9Pn1GASgtp0NOKy6YA+jR5VJEqhMo8tlGdvfO
vXF24QIwoCYgKe+62jbJ9OB0XLVG8QrA4jV8oKI+U32kUwcDQC5EOw29rq0lbOA5
ig4ry5SmkrlKK6wveOZYjWo3JKKo/o/cJnmZtEYnHzQf6p6m0M9odNgULgm3zO+I
3nXjNNTIg+4fAoIBABI7XVdAH/EQA9x1FjyeoxzZL8g0uIZZHl9gSakkU7untQxE
54R6jDB20lMfGIlIri4Z1Y8PrCf3FkbM3aFPHenN45wKghKpuH1ddl0b1qmJBxkg
0UCPuu37kccGhPw5b0Y+2F6DBw2hs/ViEPrtHZJLtwy/VBq26hLDzn7BA5eb5hO0
xmZHFMi6wnlJRHnd4CkzGGWj+WU31+z8xHlqrpWzrsRJK7ZjgfSwOW3x1FS1cesA
1/ds7JlhcXQDO/4KfjtZAZZcQuSvEAf/b/9TMU25hNXLjeLttZvVUQPSFycsP6mt
v8+pZi41bah3Pfqgp0Q9IkGcCk8JVnAbc0syYy0CggEATeNNedXh3DJmSG2ijOQX
Kbdb/asDErzFnWQd6RX/W6JG645KEfS1wo/9OBKEgIRANrP7wl3kXtxiu3EHZ5xD
obGAhSpHv6qdPvaNNIoBZvmf+I+0sNkQJ8BFTstZVslBZRsMv23D3vmNjgvUvKyr
Wa86tabN8H4ahnp4XYV4HtwTcdOqSy+Z72qcw83RWGj6owS3iOPDrCLEnihgibMd
9F726pWyyaU1Omnq4PjwEMUD67GFKBqeAQRtt2597LeNAAASB/HzGiXwPij71a2t
QijspXUDPzDwqAzI0D5tkSxT/+gNwL5ilpVQwx1bOdhOP6RoJVEnz83GYvsOBN+F
EQKCAQEAo9j9MG+VCz+loz4fUXIJjC63ypfuRfxTCAIBMn4HzohP8chEcQBlWLCH
t0WcguYnwsuxGR4Rhx02UZCx3qNxiroBZ9w1NqTk947ZjKuNzqI7IpIqvtJ18op6
QgQu8piNkf0/etAO0e6IjbZe4WfJCeKsAqE4vCV43baaSiHN/0pfYi6LLJ2YmTF/
+sYY43naHg3zQTL4JbL4c58ebe4ADj4wIdNJ+/H5JgQf6r14iNjpyc6BJOjFuPyx
EJHQKb6499HKFua3QuH/kA6Ogfm9o3Lnwx/VO1lPLFteTv1fBKK00C00SkmyIe1p
iaKCVjivzjP1s/q6adzOOZVlVwm7Xw==
-----END PRIVATE KEY-----
`
)

// TestTCPOVerTLSClient tests the TLS layer of the modbus client.
func TestTCPoverTLSClient(t *testing.T) {
	var err            error
	var client         *ModbusClient
	var serverKeyPair  tls.Certificate
	var clientKeyPair  tls.Certificate
	var clientCp       *x509.CertPool
	var serverCp       *x509.CertPool
	var serverHostPort string
	var serverChan     chan string
	var regs           []uint16

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
	var err            error
	var client         *ModbusClient
	var server         *ModbusServer
	var serverKeyPair  tls.Certificate
	var clientKeyPair  tls.Certificate
	var clientCp       *x509.CertPool
	var serverCp       *x509.CertPool
	var th	           *tlsTestHandler
	var reg            uint16

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
		Timeout:       500 * time.Millisecond,
	}, th)
	if err != nil {
		t.Errorf("failed to create server: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Errorf("failed to start server: %v", err)
	}

	// create the modbus client
	client, err	= NewClient(&ClientConfiguration{
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
	var err         error
	var listener    net.Listener
	var sock        net.Conn
	var reqCount    uint
	var clientCount uint
	var buf         []byte

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
