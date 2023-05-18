# services/l1/bridge/bridge.go

[bridge.go](./bridge.go) provides an interface and implementation for interacting with Ethereum bridges.
It is designed to support interaction with different Ethereum chains such as Mainnet and Goerli, allowing deposit and withdrawal operations, among others.

- **Example**

The following example demonstrates how to get a map of custom Ethereum bridges for a given chain ID (Mainnet in this case):

```go
bridges, err := BridgesByChainID(big.NewInt(1), client, addrs)
if err != nil {
    log.Fatal(err)
}
```
In this code snippet, `client` is the Ethereum client and `addrs` is an instance of `services.AddressManager`. The function `BridgesByChainID` returns a map of bridge interfaces.

- **Usage**

This package provides several functionalities such as:

1. Getting deposits by a range of blocks via `GetDepositsByBlockRange` method of the `Bridge` interface.

```go
deposits, err := bridge.GetDepositsByBlockRange(ctx, startBlock, endBlock)
if err != nil {
    log.Fatal(err)
}
```

2. Creating a state commitment chain scanner, which filters Ethereum events related to state commitment.

```go
scanner, err := StateCommitmentChainScanner(client, addrs)
if err != nil {
    log.Fatal(err)
}
```

Here, the `StateCommitmentChainScanner` function takes in an Ethereum client and an instance of `services.AddressManager` and returns a new filter for the state commitment chain.

- **See also:** 

- Implementation in [bridge.go](./bridge.go)
- Unit tests in [bridge_test.go](../bridge/bridge_test.go)
- External library docs: [go-ethereum docs](https://pkg.go.dev/github.com/ethereum/go-ethereum@v1.10.8), [optimism docs](https://pkg.go.dev/github.com/ethereum-optimism/optimism)
