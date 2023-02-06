---
title: Transaction fees on L2
lang: en-US
---

## Understanding the basics

Transaction fees on Optimism work a lot like fees on Ethereum.
However, Layer 2 introduces some new paradigms that means it can never be exactly like Ethereum.
Luckily, Optimism's [EVM equivalence](https://medium.com/ethereum-optimism/introducing-evm-equivalence-5c2021deb306) makes these differences easy to understand and even easier to handle within your app.
Let's take a look at the two sources of cost in a transaction on Optimism: the L2 execution fee and the L1 data/security fee.

### The L2 execution fee

Just like on Ethereum, transactions on Optimism have to pay **gas** for the amount of computation and storage that they use.
Every L2 transaction will pay some **execution fee**, equal to the amount of gas used by the transaction multiplied by the gas price attached to the transaction.
This is exactly how fees work on Ethereum with the added bonus that gas prices on Optimism are seriously low.

Here's the (simple) math:

```
l2_execution_fee = transaction_gas_price * l2_gas_used
```

The amount of L2 gas used depends on the particular transaction that you're trying to send.
Thanks to [EVM equivalence](https://medium.com/ethereum-optimism/introducing-evm-equivalence-5c2021deb306), transactions typically use approximately the same amount of gas on Optimism as they do on Ethereum.
Gas prices fluctuate with time and congestion, but you can always check the current estimated L2 gas price on the [public Optimism dashboard](https://public-grafana.optimism.io/d/9hkhMxn7z/public-dashboard?orgId=1&refresh=5m).

### The L1 data fee

Optimism differs from Ethereum because all transactions on Optimism are also published to Ethereum.
This step is crucial to the security properties of Optimism because it means that all of the data you need to sync an Optimism node is always publicly available on Ethereum.
It's what makes Optimism an L2.

Users on Optimism have to pay for the cost of submitting their transactions to Ethereum.
We call this the **L1 data fee**, and it's the primary discrepancy between Optimism (and other L2s) and Ethereum.
Because the cost of gas is so expensive on Ethereum, the L1 data fee typically dominates the total cost of a transaction on Optimism.
This fee is based on four factors:

1. The current gas price on Ethereum.
2. The gas cost to publish the transaction to Ethereum. This scales roughly with the size of the transaction (in bytes).
3. A fixed overhead cost denominated in gas. This is currently set to 2100.
4. A dynamic overhead cost which scales the L1 fee paid by a fixed number. This is currently set to 1.0.

Here's the math:

```
l1_data_fee = l1_gas_price * (tx_data_gas + fixed_overhead) * dynamic_overhead
```

Where `tx_data_gas` is:

```
tx_data_gas = count_zero_bytes(tx_data) * 4 + count_non_zero_bytes(tx_data) * 16
```

You can read the parameter values from the [gas oracle contract](https://explorer.optimism.io/address/0x420000000000000000000000000000000000000F#readContract).

::: warning NOTE
Ethereum has limited support for adding custom transaction types.
As a result, unlike the L2 execution fee, **users are not able to set limits for the L1 data fee that they may be charged**.
The L1 gas price used to charge the data fee is automatically updated when new data is received from Ethereum.
**Spikes in Ethereum gas prices may result in users paying a higher or lower than estimated L1 data fee, by up to 25%.**

[See here for a detailed explanation why the difference is capped at 25%](https://help.optimism.io/hc/en-us/articles/4416677738907-What-happens-if-the-L1-gas-price-spikes-while-a-transaction-is-in-process).
:::


## Stuff to keep in mind

### Sending transactions

The process of sending a transaction on Optimism is identical to the process of sending a transaction on Ethereum.
When sending a transaction, you should provide a gas price greater than or equal to the current L2 gas price.
Like on Ethereum, you can query this gas price with the `eth_gasPrice` RPC method.
Similarly, you should set your transaction gas limit in the same way that you would set your transaction gas limit on Ethereum (e.g. via `eth_estimateGas`).

### Responding to gas price updates

Gas prices on L2 default to 0.001 Gwei but can increase dynamically if the network is congested.
When this happens, the lowest fee that the network will accept increases.
Unlike Ethereum, Optimism currently does not have a mempool to hold transactions with too low a fee.
Instead, Optimism nodes will reject the transaction with the message `Fee too low`.
You may need to handle this case explicitly and retry the transaction with a new gas price when this happens.

### Displaying fees to users

Many Ethereum applications display estimated fees to users by multiplying the gas price by the gas limit.
However, as discussed earlier, users on Optimism are charged both an L2 execution fee and an L1 data fee.
As a result, you should display the sum of both of these fees to give users the most accurate estimate of the total cost of a transaction.

[See here for a code sample using the JavaScript SDK](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/sdk-estimate-gas)

#### Estimating the L2 execution fee

You can estimate the L2 execution fee by multiplying the gas price by the gas limit, just like on Ethereum.

#### Estimating the L1 data fee

You can use the SDK [(see here)](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/sdk-estimate-gas).
Alternatively, you can estimate the L1 data fee using the `GasPriceOracle` predeployed smart contract located at [`0x420000000000000000000000000000000000000F`](https://explorer.optimism.io/address/0x420000000000000000000000000000000000000F).
[The `GasPriceOracle` contract](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts/contracts/L2/predeploys/OVM_GasPriceOracle.sol) is located at the same address on every Optimism network (mainnet and testnet).
To do so, call `GasPriceOracle.getL1Fee(<unsigned RLP encoded transaction>)`.

#### Estimating the total fee

You can estimate the total fee by combining your estimates for the L2 execution fee and L1 data fee.

### Sending max ETH

Sending the maximum amount of ETH that a user has in their wallet is a relatively common use case.
When doing this, you will need to subtract the estimated L2 execution fee and the estimated L1 data fee from the amount of ETH you want the user to send.
Use the logic described above for estimating the total fee.

## Common RPC Errors

### Insufficient funds

- Error code: `-32000`
- Error message: `invalid transaction: insufficient funds for l1Fee + l2Fee + value`

You'll get this error when attempting to send a transaction and you don't have enough ETH to pay for the value of the transaction, the L2 execution fee, and the L1 data fee.
You might get this error when attempting to send max ETH if you aren't properly accounting for both the L2 execution fee and the L1 data fee.

### Gas price too low

- Error code: `-32000`
- Error message: `gas price too low: X wei, use at least tx.gasPrice = Y wei`

This is a custom RPC error that Optimism returns when a transaction is rejected because the gas price is too low.
See the section on [Responding to gas price updates](#responding-to-gas-price-updates) for more information.

### Gas price too high
- Error code: `-32000`
- Error message: `gas price too high: X wei, use at most tx.gasPrice = Y wei`

This is a custom RPC error that Optimism returns when a transaction is rejected because the gas price is too high.
We include this as a safety measure to prevent users from accidentally sending a transaction with an extremely high L2 gas price.
See the section on [Responding to gas price updates](#responding-to-gas-price-updates) for more information.
