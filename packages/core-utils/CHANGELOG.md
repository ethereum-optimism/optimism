# @eth-optimism/core-utils

## 0.12.0

### Minor Changes

- c975c9620: Add suppory for finalizing legacy withdrawals after the Bedrock migration

### Patch Changes

- 136ea1785: Refactors the L2OutputOracle to key the l2Outputs mapping by index instead of by L2 block number.

## 0.11.0

### Minor Changes

- 1e76cdb86: Changes the type for Bedrock withdrawal proofs

## 0.10.1

### Patch Changes

- 7215f4ce: Bump ethers to 5.7.0 globally
- 206f6033: Fix outdated references to 'withdrawal contract'

## 0.10.0

### Minor Changes

- dbfea116: Removes ethers as a dependency in favor of individual ethers sub-packages

## 0.9.3

### Patch Changes

- 0df744f6: Implement basic OpNodeProvider
- 8ae39154: Update deposit transaction type
- dac4a9f0: Updates the SDK to be compatible with Bedrock (via the "bedrock: true" constructor param). Updates the build pipeline for contracts-bedrock to export a properly formatted dist folder that matches our other packages.

## 0.9.2

### Patch Changes

- 0bf3b9b4: Add encoding and hashing functions for bedrock
- 8d26459b: Remove subversion byte from deposit tx
- 4477fe9f: Update deposit transaction serialization

## 0.9.1

### Patch Changes

- f9fee446: Move the `DepositTx` type to `core-utils`. This way it can be more easily used across projects

## 0.9.0

### Minor Changes

- 700dcbb0: Update geth's Genesis type to work with modern geth

## 0.8.7

### Patch Changes

- 29ff7462: Revert es target back to 2017

## 0.8.6

### Patch Changes

- 17962ca9: Update geth genesis type

## 0.8.5

### Patch Changes

- d18ae135: Updates all ethers versions in response to BN.js bug

## 0.8.4

### Patch Changes

- 5cb3a5f7: Add a `calldataCost` function that computes the cost of calldata
- 6b9fc055: Adds a one-liner for getting chain ID from provider

## 0.8.3

### Patch Changes

- b57014d1: Update to typescript@4.6.2

## 0.8.2

### Patch Changes

- c1957126: Update Dockerfile to use Alpine

## 0.8.1

### Patch Changes

- 5a6f539c: Add toJSON methods to the batch primitives
- 27d8942e: Update batch serialization with typed batches and zlib compression

## 0.8.0

### Minor Changes

- 0b4453f7: Deletes the Watcher and injectL2Context functions. Use the SDK instead.

## 0.7.7

### Patch Changes

- b4165299: Added tests and docstrings to misc functions
- 3c2acd91: Refactor folder structure of @eth-optimism/core-utils.

## 0.7.6

### Patch Changes

- ba14c59d: Updates various ethers dependencies to their latest versions

## 0.7.5

### Patch Changes

- ad94b9d1: test/docs: Improve docstrings and tests for utils inside of hex-strings.ts

## 0.7.4

### Patch Changes

- ba96a455: Improved docstrings for BCFG typings
- c3e85fef: Cleans up the internal file and folder structure for the typings exported by core-utils

## 0.7.3

### Patch Changes

- 584cbc25: Clean up the L1 => L2 address aliasing utilities

## 0.7.2

### Patch Changes

- 8e634b49: Fix package JSON issues

## 0.7.1

### Patch Changes

- 243f33e5: Standardize package json file format

## 0.7.0

### Minor Changes

- 896168e2: Parse optimistic ethereum specific fields on transaction receipts
- 83a449c4: Change the expectApprox interface to allow setting an absoluteexpected deviation range
- 81ccd6e4: `regenesis/0.5.0` release

### Patch Changes

- 3ce62c81: Export bnToAddress
- cee2a464: Add awaitCondition to core utils
- 222a3eef: Add 'User-Agent' to the http headers for ethers providers
- 7c352b1e: Add bytes32ify
- b70ee70c: upgraded to solidity 0.8.9
- 20c8969b: Correctly move chai into deps instead of dev deps
- 6d32d701: Expose lower level API for tx fees

## 0.6.1

### Patch Changes

- 6d3e1d7f: Update dependencies
- 2e929aa9: Parse the L1 timestamp in `injectContext`

## 0.6.0

### Minor Changes

- 8da04505: Allow a configurable L1 and L2 blocks to fetch in the watcher

### Patch Changes

- e0be02e1: Add fallback provider support to DTL using helper function in core-utils

## 0.5.5

### Patch Changes

- eb0854e7: increased coverage of core-utils
- 21b17edd: Added coverage for packages
- dfe3598f: Lower per tx fee overhead to more accurately represent L1 costs

## 0.5.4

### Patch Changes

- 085b35ba: Watcher: Even lower num blocks to fetch

## 0.5.3

### Patch Changes

- 2aa4416e: Watcher: Make blocks to fetch a config option
- 0b8180b0: Lower NUM_BLOCKS_TO_FETCH in Watcher

## 0.5.2

### Patch Changes

- 918c08ca: Bump ethers dependency to 5.4.x to support eip1559

## 0.5.1

### Patch Changes

- c73c3939: Update the typescript version to `4.3.5`

## 0.5.0

### Minor Changes

- 049200f4: removed unused functions from core-utils

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
