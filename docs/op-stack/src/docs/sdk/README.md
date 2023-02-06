---
title: SDK
lang: en-US
---

In most ways Optimism is [EVM equivalent](https://medium.com/ethereum-optimism/introducing-evm-equivalence-5c2021deb306).
However, the are [a few differences](../developers/build/differences/), which sometimes require decentralized applications to access Optimism-specific services.

For example, decentralized applications might need to estimate gas costs.
The standard Ethereum tooling assumes that gas cost is proportional to the gas used by the transaction, which is correct on L1, but not on Optimism.
[Our gas costs are predominately the cost of writing the transaction to L1](../developers/build/transaction-fees.md), which depends on the transaction size, not the amount of processing required.
This difference requires us to have separate methods to provide gas estimates.

There are three ways to access Optimism services:

1. [On chain contract calls](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts/docs). 
   This is the way your contracts can get Optimism information or services directly.
1. [The JavaScript SDK](js-client.md). For use when you write JavaScript or TypeScript code, either in the client or a Node.js server.
1. [Off chain, using RPC](../developers/build/json-rpc.md). Which is more complicated but usable from any development stack (Python, Rust, etc.).


::: tip Improving the SDK
If you find a bug, or if there's a feature you think we should add, there are several ways to inform us.

- [Go on our Discord](https://discord-gateway.optimism.io/), and then ask in **#dev-support**.
- Submit an issue on [our Github](https://github.com/ethereum-optimism/optimism/issues).
:::
