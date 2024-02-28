# go-schnorrkel
  <a href="https://discord.gg/zy8eRF7FG2">
    <img alt="Discord" src="https://img.shields.io/discord/593655374469660673.svg?style=flat&label=Discord&logo=discord" />
  </a>


This repo contains the Go implementation of the sr25519 signature algorithm (schnorr over ristretto25519). The existing Rust implementation is [here.](https://github.com/w3f/schnorrkel)

This library is currently able to create sr25519 keys, import sr25519 keys, and sign and verify messages. It is interoperable with the Rust implementation. 

The BIP39 implementation in this library is compatible with the rust [substrate-bip39](https://github.com/paritytech/substrate-bip39) implementation.  Note that this is not a standard bip39 implementation.

This library has been audited as of August 2021 and is production-ready.

### dependencies

go 1.16

### usage

Example: key generation, signing, and verification

```
package main 

import (
	"fmt"
	
	schnorrkel "github.com/ChainSafe/go-schnorrkel"
)

func main() {
	msg := []byte("hello friends")
	signingCtx := []byte("example")

	signingTranscript := schnorrkel.NewSigningContext(signingCtx, msg)
	verifyTranscript := schnorrkel.NewSigningContext(signingCtx, msg)

	priv, pub, err := schnorrkel.GenerateKeypair()
	if err != nil {
		panic(err)
	}

	sig, err := priv.Sign(signingTranscript)
	if err != nil {
		panic(err)
	}

	ok := pub.Verify(sig, verifyTranscript)
	if !ok {
		fmt.Println("failed to verify signature")
		return
	}

	fmt.Println("verified signature")
}
```

Please see the [godocs](https://pkg.go.dev/github.com/ChainSafe/go-schnorrkel) for more usage examples.
