
# @eth-optimism/sdk

[![codecov](https://codecov.io/gh/ethereum-optimism/optimism/branch/master/graph/badge.svg?token=0VTG7PG7YR&flag=sdk-tests)](https://codecov.io/gh/ethereum-optimism/optimism)

The `@eth-optimism/sdk` package provides a set of tools for interacting with Optimism.

## Warning!!!

`@eth-optimism/sdk` has been superseded by `op-viem`. For most developers we suggest you migrate to [viem](https://viem.sh/op-stack) which has native built in op-stack support built in. It also has additional benefits.

**The OP Labs team has no plans to update @eth-optimism/sdk and it is in maintenance mode at the moment**

- an intuitive API that learned from this package and is now revamped
- great treeshaking with a 10x+ improvement to bundlesize
- Better performance
- Updated to use the latest op stack contracts. At times it will save you gas compared to using viem.

If viem does not have what you need please let us know by opening an issue in the viem repo or here. Letting us know helps us advocate to upstream more functionality to viem. Viem is missing the following functionality:

- ERC20 support

If viem doesn't have what you need, the extensions for viem, [op-viem extensions](https://github.com/base-org/op-viem), likely have it too.

## Installation

```
npm install @eth-optimism/sdk
```

## Docs

You can find auto-generated API documentation over at [sdk.optimism.io](https://sdk.optimism.io).

## Contributing

Most of the core functionality is in the [CrossChainMessenger](./src/cross-chain-messenger.ts) file.

## Using the SDK

### CrossChainMessenger

The [`CrossChainMessenger`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/cross-chain-messenger.ts) class simplifies the process of moving assets and data between Ethereum and Optimism.
You can use this class to, for example, initiate a withdrawal of ERC20 tokens from Optimism back to Ethereum, accurately track when the withdrawal is ready to be finalized on Ethereum, and execute the finalization transaction after the challenge period has elapsed.
The `CrossChainMessenger` can handle deposits and withdrawals of ETH and any ERC20-compatible token.
Detailed API descriptions can be found at [sdk.optimism.io](https://sdk.optimism.io/classes/crosschainmessenger).
The `CrossChainMessenger` automatically connects to all relevant contracts so complex configuration is not necessary.

### L2Provider and related utilities

The Optimism SDK includes [various utilities](https://github.com/ethereum-optimism/optimism/blob/develop/packages/sdk/src/l2-provider.ts) for handling Optimism's [transaction fee model](https://community.optimism.io/docs/developers/build/transaction-fees/).
For instance, [`estimateTotalGasCost`](https://sdk.optimism.io/modules.html#estimateTotalGasCost) will estimate the total cost (in wei) to send at transaction on Optimism including both the L2 execution cost and the L1 data cost.
You can also use the [`asL2Provider`](https://sdk.optimism.io/modules.html#asL2Provider) function to wrap an ethers Provider object into an `L2Provider` which will have all of these helper functions attached.

### Other utilities

The SDK contains other useful helper functions and constants.
For a complete list, refer to the auto-generated [SDK documentation](https://sdk.optimism.io/)
