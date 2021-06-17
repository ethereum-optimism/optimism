# utils

This package is meant to hold utilities used by
[Optimistic Ethereum](https://github.com/ethereum-optimism/optimism) written in
Golang.

## Packages

### Fees

Package fees includes helpers for dealing with fees on Optimistic Ethereum

#### `EncodeTxGasLimit(data []byte, l1GasPrice, l2GasLimit, l2GasPrice *big.Int) *big.Int`

Encodes `tx.gasLimit` based on the variables that are used to determine it.

#### `DecodeL2GasLimit(gasLimit *big.Int) *big.Int`

Accepts the return value of `eth_estimateGas` and decodes the L2 gas limit that
is encoded in the return value. This is the gas limit that is passed to the user
contract within the OVM.
