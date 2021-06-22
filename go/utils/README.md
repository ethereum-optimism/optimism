# utils

This package is meant to hold utilities used by
[Optimistic Ethereum](https://github.com/ethereum-optimism/optimism) written in
Golang.

## Packages

### Fees

Package fees includes helpers for dealing with fees on Optimistic Ethereum

#### `EncodeTxGasLimit(data []byte, l1GasPrice, l2GasLimit, l2GasPrice *big.Int) *big.Int`

Encodes `tx.gasLimit` based on the variables that are used to determine it.

`data` - Calldata of the transaction being sent. This data should *not* include the full signed RLP transaction.

`l1GasPrice` - gas price on L1 in wei

`l2GasLimit` - amount of gas provided for execution in L2. Notably, accounts are charged for execution based on this gasLimit, even if the gasUsed ends up being less.

`l2GasPrice` - gas price on L2 in wei

#### `DecodeL2GasLimit(gasLimit *big.Int) *big.Int`

Accepts the return value of `eth_estimateGas` and decodes the L2 gas limit that
is encoded in the return value. This is the gas limit that is passed to the user
contract within the OVM.
