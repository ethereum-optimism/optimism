# go-subkey
[![Go Reference](https://pkg.go.dev/badge/github.com/vedhavyas/go-subkey.svg)](https://pkg.go.dev/github.com/vedhavyas/go-subkey)

Subkey port in Go

## Usage

### Generate Key pair

#### Sr25519
```go
    kr, err := sr25519.Scheme{}.Generate()
```

#### Ed25519
```go
    kr, err := ed25519.Scheme{}.Generate()
```

#### Ecdsa
```go
    kr, err := ecdsa.Scheme{}.Generate()
```


### Deriving keypair from a mnemonic or seed

#### Mnemonic
```go
    uri := "crowd swamp sniff machine grid pretty client emotion banana cricket flush soap//foo//42///password"
    scheme := sr25519.Scheme{}
    kr, err := subkey.DeriveKeyPair(scheme, uri)
```

#### Hex encoded Seed
```go
    uri := "0x6ea8835d60351a39a1e2293b2902d7bd6e12e526e72c46f4fda4a233809c4379"
    scheme := sr25519.Scheme{}
    kr, err := subkey.DeriveKeyPair(scheme, uri)
```

#### Hex encoded Seed with derivation
```go
    uri := "0x6ea8835d60351a39a1e2293b2902d7bd6e12e526e72c46f4fda4a233809c4379//foo//42///password"
    scheme := sr25519.Scheme{}
    kr, err := subkey.DeriveKeyPair(scheme, uri)
```


### Sign and verify using Keypair
```go
    kr, err := ed25519.Scheme{}.Generate()
    msg := []byte("test message")
    sig, err := kr.Sign(msg)
    ok := kr.Verify(msg, sig)
```
