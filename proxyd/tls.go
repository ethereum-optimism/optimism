package proxyd

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
)

func CreateTLSClient(ca string) (*tls.Config, error) {
	pem, err := os.ReadFile(ca)
	if err != nil {
		return nil, wrapErr(err, "error reading CA")
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(pem)
	if !ok {
		return nil, errors.New("error parsing TLS client cert")
	}

	return &tls.Config{
		RootCAs: roots,
	}, nil
}

func ParseKeyPair(crt, key string) (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		return tls.Certificate{}, wrapErr(err, "error loading x509 key pair")
	}
	return cert, nil
}
