# op-bindings

This package contains built go bindings of the smart contracts. It must be
updated after any changes to the smart contracts to ensure that the bindings are
up to date.

The bindings include the bytecode for each contract so that go based tests
can deploy the contracts. There are also `more` files that include the deployed
bytecode as well as the storage layout. These are used to dynamically set
bytecode and storage slots in state.

## Usage

```bash
make
```

## Dependencies

- `abigen` version 1.10.25
- `make`

To check the version of `abigen`, run the command `abigen --version`.

## abigen

The `abigen` tool is part of `go-ethereum` and can be used to build go bindings
for smart contracts. It can be installed with go using the commands:

```bash
$ go get -u github.com/ethereum/go-ethereum
$ cd $GOPATH/src/github.com/ethereum/go-ethereum/
$ make devtools
```

The geth docs for `abigen` can be found [here](https://geth.ethereum.org/docs/dapp/native-bindings).

## See also

TypeScript bindings are also generated in [@eth-optimism/contracts-ts](../packages/contracts-ts/)
