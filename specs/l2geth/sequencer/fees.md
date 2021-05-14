# Transaction Fees Spec

## Goals

- Allow projects to easily query how high of a gasPrice their users need to set to get their transaction approved
- Allow the sequencer to only accept transactions that at least cover their cost to the system (L1 calldata gas costs + L2 gas costs)

## Non-goals

- Minimum possible fees
- Fee calculation that is completely automated
- Congestion Fees

## Why a separate service?

N/A

## Pre-launch timeline

- [ ] getGasPrice endpoint in geth + rejecting underfunded txs 2 weeks before launch
- [ ] Paying fees in Kovan -both Synthetix and Kovan by 1.5 week before launch
- [ ] Paying fees in Mainnet 1 week before launch

## Inputs & dependencies

- OVM_ETH (for fee payment)
- ETH Deposit Withdrawal Dapp
- Seeing your L2 WETH in SNX frontend

### getGasPrice(tx)

- Input is tx calldata

## Outputs

### estimateGasPrice(tx, ?gasLimit)

- returns `gasPrice` in wei to set for a given tx, taking into account the current `dataPrice` and `l2GasPrice` along with the `tx` calldata.

### getDataPrice()

- returns the `dataPrice` in wei (cost per zero byte). This will approximately track 4 \* l1 gasPrice

### getGasPrice()

- returns the gasPrice for L2 execution

### validateTx

- returns error message for eth_sendRawTransaction
  - If tx is underfunded, "Error: Inadequate transaction fee. For this transaction to be accepted, set a gasPrice of \_\_ gwei"
  - If tx has a gasLimit !== 9m, "Error: Expected a gas limit of 9m. Please set your tx gasLimit to 9m."

## Internals

Geth sets a dataPrice and a gasPrice internally. This can be displayed publicly on an `*.optimism.io` site.

- `dataPrice` in wei is the cost per zero byte of calldata. A non-zero byte of calldata will cost `dataPrice * 4` in wei.
- `gasPrice` in wei is the cost per unit of gas consumed during L2 execution.

```
// estimates the gas price based on a txs size and its gasLimit so gasPrice * gasLimit = intended fee
function estimateGasPrice(tx, ?gasLimit=9000000) {
    const dataCost = dataPrice * (tx.zeroBytes + (tx.nonZeroBytes * 4))
    const gasCost = gasLimit * gasPrice
    return ((gasCost + dataCost) / gasLimit)
}
```
