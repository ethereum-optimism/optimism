package subkey

import (
	"fmt"
)

// Scheme represents a cryptography scheme.
type Scheme interface {
	fmt.Stringer
	Generate() (KeyPair, error)
	FromSeed(seed []byte) (KeyPair, error)
	FromPhrase(phrase, password string) (KeyPair, error)
	Derive(pair KeyPair, djs []DeriveJunction) (KeyPair, error)
}

// DeriveKeyPair derives the Keypair from the URI using the provided cryptography scheme.
func DeriveKeyPair(scheme Scheme, uri string) (kp KeyPair, err error) {
	phrase, path, pwd, err := splitURI(uri)
	if err != nil {
		return nil, err
	}

	if b, ok := DecodeHex(phrase); ok {
		kp, err = scheme.FromSeed(b)
	} else {
		kp, err = scheme.FromPhrase(phrase, pwd)
	}
	if err != nil {
		return nil, err
	}

	djs, err := deriveJunctions(derivePath(path))
	if err != nil {
		return nil, err
	}

	return scheme.Derive(kp, djs)
}
