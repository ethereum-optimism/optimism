# plasma-verifier
`plasma-verifier` is a library that makes it possible for plasma chain clients to execute [predicate contracts](https://medium.com/@plasma_group/towards-a-general-purpose-plasma-f1cc4d49c1f4). In a nutshell, predicate contracts are special smart contracts that run on a plasma chain. Clients need to be able to execute these contracts in order to check the validity of certain state transitions.

At its core, `plasma-verifier` basically just executes the EVM via `ethereumjs-vm`. All predicate contracts must implement a standard contract interface, so we just need the contract's bytecode in order to execute a transaction. 

`plasma-verifier` *currently* only supports predicate contracts that are pure functions - meaning they don't rely on any external state. However, this is **not** a restriction of predicate contracts in general. It's entirely possible to build predicate contracts that *do* read external state by pulling the state from Ethereum and then loading it into our local EVM instance. We plan to add support for this in the near future.
