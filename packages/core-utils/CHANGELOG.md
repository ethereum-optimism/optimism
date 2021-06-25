# @eth-optimism/core-utils

## 0.4.7

### Patch Changes

- 224b04c0: Adds a pollInterval delay to watcher.ts

## 0.4.6

### Patch Changes

- d9644c34: Minor fix on watchers to pick up finalization of transactions on L1
- df5ff890: improved watcher ability to find transactions during periods of high load

## 0.4.5

### Patch Changes

- a64f8161: Implement the next fee spec in both geth and in core-utils
- 750a5021: Delete dead transaction coders. These are no longer used now that RLP encoded transactions are used
- c2b6e14b: Implement the latest fee spec such that the L2 gas limit is scaled and the tx.gasPrice/tx.gasLimit show correctly in metamask

## 0.4.4

### Patch Changes

- f091e86: Have watcher correctly handle failed L1 => L2 messages
- f880479: End to end fee integration with recoverable L2 gas limit

## 0.4.3

### Patch Changes

- 96a586e: Migrate bcfg interface to core-utils

## 0.4.2

### Patch Changes

- b799caa: Update toRpcHexString to accept ethers.BigNumber and add tests

## 0.4.1

### Patch Changes

- 1d40586: Removed various unused dependencies
- ce7fa52: Add an additional enum for EthSign transactions as they now are batch submitted with 2 different enum values

## 0.4.0

### Minor Changes

- 28dc442: move metrics, logger, and base-service to new common-ts package

### Patch Changes

- a0a0052: Update toRpcHexString to accept ethers.BigNumber and add tests

## 0.3.2

### Patch Changes

- 6daa408: update hardhat versions so that solc is resolved correctly
- dee74ef: migrate batch submitter types to core-utils
- d64b66d: reformat error context for Sentry

## 0.3.1

### Patch Changes

- 5077441: - Use raw transaction in batch submitter -- incompatible with L2Geth v0.1.2.1
  - Pass through raw transaction in l2context

## 0.3.0

### Minor Changes

- 91460d9: add Metrics and use in base-service, rename DTL services to avoid spaces
- a0a7956: initialize Sentry and streams in Logger, remove Sentry from Batch Submitter

### Patch Changes

- 0497d7d: Re-organize event typings to core-utils

## 0.2.3

### Patch Changes

- 35b99b0: add Sentry to TypeScript services for error tracking

## 0.2.2

### Patch Changes

- 01eaf2c: added extra logs to base-service / dtl to improve observability

## 0.2.1

### Patch Changes

- 5362d38: adds build files which were not published before to npm

## 0.2.0

### Minor Changes

- 6cbc54d: allow injecting L2 transaction and block context via core-utils (this removes the need to import the now deprecated @eth-optimism/provider package)
