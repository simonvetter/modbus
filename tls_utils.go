package modbus

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
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
