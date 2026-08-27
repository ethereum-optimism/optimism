# @eth-optimism/viem

This package is a TypeScript extension for Viem that provides actions and utilities for working with OP stack chains. The goal of this package is to upstream as many actions and utilities as possible directly into Viem. You can view this as a playground for new features that haven't hit mainnet yet or more experimental features in the OP stack.

### Documentation

- [SDK Reference](./docs/README.md)

### Code Snippets

- [Interop](./docs/actions/interop/README.md)

### Running Tests

Before you can run the unit tests you'll need [supersim](https://github.com/ethereum-optimism/supersim) installed. Once you have supersim installed you can run `pnpm nx run @eth-optimism/viem:test` from the root of the monorepo to get the tests running.
