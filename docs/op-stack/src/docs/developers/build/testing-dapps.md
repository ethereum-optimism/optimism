---
title: Testing Decentralized Applications with Optimism
lang: en-US
---

For the most part running applications on Optimism is identical to running them on Ethereum, so the testing is identical too.
In this article you learn the best practices for Optimism testing where there are differences.


## Unit tests and single layer integration tests

The vast majority of tests do not involve any Optimism-specific features.
In those cases, while you *could* test everything on Optimism, that would normally be inefficient.
Most Ethereum development stacks include features that make testing easier, which normal Ethereum clients, such as geth (and our modified version, l2geth) don't support.
Therefore, it is a good idea to run the majority of tests, which do not rely on Optimism-specific features, in the development stack.
It is a lot faster.

Ideally you would want to be able to run some tests on Optimism (either a [local development environment](dev-node.md) or [the test network](../../useful-tools/networks.md#optimism-goerli).
This would be a much slower process, but it would let you identify cases where [the equivalence between Optimism and Ethereum breaks down](differences.md) (or the equivalence between Ethereum itself and the development stack, for that matter).

## Multilayer integration tests

Some dapps need Optimism-specific features that aren't available as part of the development stack.
For example, if your decentralized application relies on [inter-domain communication](../bridge/messaging.md), the effort of developing a stub to let you debug it in a development stack is probably greater than the hassle of having the automated test go to [a local development environment](dev-node.md) each time.


## Integration with other products

In many cases a decentralized application requires the services of other contracts. 
For example, [Perpetual v. 2](https://v2docs.perp.fi/benefits-of-v2) cannot function without [Uniswap v. 3](https://uniswap.org/blog/uniswap-v3).

If that is the case you can use [mainnet forking](https://hardhat.org/hardhat-network/guides/mainnet-forking.html).
It works with Optimism with the exception of transactions that use the `L1BlockNumber` opcode directly.
Alternatively, you can connect to our [test network](../../useful-tools/networks.md#optimism-goerli) if those contracts are also deployed there (in many cases they are).
