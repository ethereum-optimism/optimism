<h3 align="center">
  <br />
  <img src="https://user-images.githubusercontent.com/168240/51436479-a4cf9e80-1c42-11e9-9af3-3bb827b8f2de.png" alt="logo" width="600" />
  <br />
  <br />
  <br />
</h3>

# go-ethereum-hdwallet

> Ethereum HD Wallet derivations from [mnemonic] seed in Go (golang). Implements the [go-ethereum](https://github.com/ethereum/go-ethereum) [`accounts.Wallet`](https://github.com/ethereum/go-ethereum/blob/master/accounts/accounts.go) interface.

[![License](http://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/ethereum-optimism/go-ethereum-hdwallet/master/LICENSE)
[![Build Status](https://travis-ci.org/ethereum-optimism/go-ethereum-hdwallet.svg?branch=master)](https://travis-ci.org/miguelmota/go-ethereum-hdwallet)
[![Go Report Card](https://goreportcard.com/badge/github.com/ethereum-optimism/go-ethereum-hdwallet?)](https://goreportcard.com/report/github.com/miguelmota/go-ethereum-hdwallet)
[![GoDoc](https://godoc.org/github.com/ethereum-optimism/go-ethereum-hdwallet?status.svg)](https://godoc.org/github.com/miguelmota/go-ethereum-hdwallet)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#contributing)

## Install

```bash
go get -u github.com/ethereum-optimism/go-ethereum-hdwallet
```

## Documenation

[https://godoc.org/github.com/ethereum-optimism/go-ethereum-hdwallet](https://godoc.org/github.com/miguelmota/go-ethereum-hdwallet)

## Getting started

```go
package main

import (
	"fmt"
	"log"

	"github.com/ethereum-optimism/go-ethereum-hdwallet"
)

func main() {
	mnemonic := "tag volcano eight thank tide danger coast health above argue embrace heavy"
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		log.Fatal(err)
	}

	path := hdwallet.MustParseDerivationPath("m/44'/60'/0'/0/0")
	account, err := wallet.Derive(path, false)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(account.Address.Hex()) // 0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947

	path = hdwallet.MustParseDerivationPath("m/44'/60'/0'/0/1")
	account, err = wallet.Derive(path, false)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(account.Address.Hex()) // 0x8230645aC28A4EdD1b0B53E7Cd8019744E9dD559
}
```

### Signing transaction

```go
package main

import (
	"log"
	"math/big"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum-optimism/go-ethereum-hdwallet"
)

func main() {
	mnemonic := "tag volcano eight thank tide danger coast health above argue embrace heavy"
	wallet, err := hdwallet.NewFromMnemonic(mnemonic)
	if err != nil {
		log.Fatal(err)
	}

	path := hdwallet.MustParseDerivationPath("m/44'/60'/0'/0/0")
	account, err := wallet.Derive(path, true)
	if err != nil {
		log.Fatal(err)
	}

	nonce := uint64(0)
	value := big.NewInt(1000000000000000000)
	toAddress := common.HexToAddress("0x0")
	gasLimit := uint64(21000)
	gasPrice := big.NewInt(21000000000)
	var data []byte

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)
	signedTx, err := wallet.SignTx(account, tx, nil)
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(signedTx)
}
```

## CLI

```bash
go install github.com/ethereum-optimism/go-ethereum-hdwallet/cmd/geth-hdwallet@latest
```

```bash
$ geth-hdwallet -mnemonic "tag volcano eight thank tide danger coast health above argue embrace heavy" -path "m/44'/60'/0'/0/0"

public address: 0xC49926C4124cEe1cbA0Ea94Ea31a6c12318df947
private key: 63e21d10fd50155dbba0e7d3f7431a400b84b4c2ac1ee38872f82448fe3ecfb9
```

## Test

```bash
make test
```

## Contributing

Pull requests are welcome!

For contributions please create a new branch and submit a pull request for review.

## License

[MIT](LICENSE)
