# Changelog

## 0.3.14

### Patch Changes

- 39cea8fd: Removes the call to `appendQueueBatch` from the batch submitter
- Updated dependencies [e0be02e1]
- Updated dependencies [7f7f35c3]
- Updated dependencies [8da04505]
  - @eth-optimism/core-utils@0.6.0
  - @eth-optimism/contracts@0.4.13

## 0.3.13

### Patch Changes

- 7482d09c: Fixes a bug in the batch submitted that would cause it to submit transactions with increasing nonces

## 0.3.12

### Patch Changes

- 21b17edd: Added coverage for packages
- 78ca518b: Add loglines for eip1559 related fields before sending a transaction
- Updated dependencies [888dafca]
- Updated dependencies [eb0854e7]
- Updated dependencies [21b17edd]
- Updated dependencies [dfe3598f]
  - @eth-optimism/contracts@0.4.11
  - @eth-optimism/core-utils@0.5.5

## 0.3.11

### Patch Changes

- 918c08ca: Bump ethers dependency to 5.4.x to support eip1559
- Updated dependencies [918c08ca]
  - @eth-optimism/contracts@0.4.10
  - @eth-optimism/core-utils@0.5.2

## 0.3.10

### Patch Changes

- b5b9fd89: Migrate to using `ethers.StaticJsonRpcProvider`
- Updated dependencies [ecc2f8c1]
  - @eth-optimism/contracts@0.4.9

## 0.3.9

### Patch Changes

- 3b132974: Fix tx resubmission estimateGas bug in batch submitter
- Updated dependencies [7f26667d]
- Updated dependencies [77511b68]
  - @eth-optimism/contracts@0.4.7

## 0.3.8

### Patch Changes

- c73c3939: Update the typescript version to `4.3.5`
- Updated dependencies [c73c3939]
  - @eth-optimism/common-ts@0.1.5
  - @eth-optimism/contracts@0.4.5
  - @eth-optimism/core-utils@0.5.1

## 0.3.7

### Patch Changes

- 8a1e63dd: Prevent batch submitter from submitting batches if low on ETH
- Updated dependencies [0313794b]
- Updated dependencies [049200f4]
- Updated dependencies [21e47e1f]
  - @eth-optimism/contracts@0.4.2
  - @eth-optimism/core-utils@0.5.0

## 0.3.6

### Patch Changes

- f87a2d00: Use dashes instead of colons in contract names
- 52d02b14: Add failure metrics to batch submitter
- 31f517a2: Improved logging of batch submission timeout logs
- 5c89c45f: Move the metric prefix string to a label #1047
- Updated dependencies [25f09abd]
- Updated dependencies [dd8edc7b]
- Updated dependencies [c87e4c74]
- Updated dependencies [db0dbfb2]
- Updated dependencies [7f5936a8]
- Updated dependencies [f87a2d00]
- Updated dependencies [85da4979]
- Updated dependencies [57ca21a2]
- Updated dependencies [5fc728da]
- Updated dependencies [2e72fd90]
- Updated dependencies [c43b33ec]
- Updated dependencies [26bc63ad]
- Updated dependencies [a0d9e565]
- Updated dependencies [2bd49730]
- Updated dependencies [38355a3b]
- Updated dependencies [3c2c32e1]
- Updated dependencies [d9644c34]
- Updated dependencies [48ece14c]
- Updated dependencies [e04de624]
- Updated dependencies [014dea71]
- Updated dependencies [fa29b03e]
- Updated dependencies [6b46c8ba]
- Updated dependencies [e045f582]
- Updated dependencies [5c89c45f]
- Updated dependencies [df5ff890]
- Updated dependencies [e29fab10]
- Updated dependencies [c2a04893]
- Updated dependencies [baacda34]
  - @eth-optimism/contracts@0.4.0
  - @eth-optimism/core-utils@0.4.6
  - @eth-optimism/common-ts@0.1.4

## 0.3.5

### Patch Changes

- 7cce55a9: Add status to generic error log to disambiguate errors

## 0.3.4

### Patch Changes

- baa3b761: Improve Sentry support, initializing as needed and ensuring ERROR logs route to Sentry
- cc742715: Fix typo in USE_HARDHAT config
- 98b7839f: Change monotonicity band-aid code to log warnings not errors
- c520100d: Fix a bug in fixMonotonicity auto healer
- 85362d44: Log additional data in monotonicity violation
- Updated dependencies [baa3b761]
  - @eth-optimism/common-ts@0.1.3

## 0.3.3

### Patch Changes

- 750a5021: Remove dead imports from core-utils
- Updated dependencies [a64f8161]
- Updated dependencies [4e03f8a9]
- Updated dependencies [8e2bfd07]
- Updated dependencies [750a5021]
- Updated dependencies [c2b6e14b]
- Updated dependencies [245136f1]
  - @eth-optimism/core-utils@0.4.5
  - @eth-optimism/contracts@0.3.5

## 0.3.2

### Patch Changes

- 4340bb1: Fix: correctly read Batch Submitter env var defaults

## 0.3.1

### Patch Changes

- c79dc8b: Add impersonate account debug config.
- 0c16805: add metrics server to common-ts and batch submitter
- fa4898a: Explicitly log error messages so that they do not show as empty objects
- 96a586e: Updates the configuration to use bcfg in a backwards compatible way
- c79dc8b: Make BLOCK_OFFSET configurable.
- Updated dependencies [96a586e]
- Updated dependencies [0c16805]
- Updated dependencies [775118a]
  - @eth-optimism/core-utils@0.4.3
  - @eth-optimism/common-ts@0.1.2
  - @eth-optimism/contracts@0.3.1

## 0.3.0

### Minor Changes

- b799caa: Updates to use RLP encoded transactions in batches for the `v0.3.0` release

### Patch Changes

- 751e2be: Add the support for different sequencer & proposer keys in the batch submitter.
- Updated dependencies [b799caa]
- Updated dependencies [6132e7a]
- Updated dependencies [b799caa]
- Updated dependencies [b799caa]
- Updated dependencies [b799caa]
- Updated dependencies [20747fd]
- Updated dependencies [b799caa]
- Updated dependencies [b799caa]
  - @eth-optimism/contracts@0.3.0
  - @eth-optimism/core-utils@0.4.2

## 0.2.5

### Patch Changes

- 1d40586: Removed various unused dependencies
- Updated dependencies [1d40586]
- Updated dependencies [ce7fa52]
- Updated dependencies [575bcf6]
- Updated dependencies [6dc1877]
  - @eth-optimism/common-ts@0.1.1
  - @eth-optimism/contracts@0.2.10
  - @eth-optimism/core-utils@0.4.1

## 0.2.4

### Patch Changes

- 12dbd81: add key metrics to batch submitter
- 28dc442: move metrics, logger, and base-service to new common-ts package
- 79df44e: Add skipped deposit auto heal
- Updated dependencies [28dc442]
- Updated dependencies [d2091d4]
- Updated dependencies [a0a0052]
- Updated dependencies [0ef3069]
  - @eth-optimism/common-ts@0.1.0
  - @eth-optimism/core-utils@0.4.0
  - @eth-optimism/contracts@0.2.9

## 0.2.3

### Patch Changes

- 6daa408: update hardhat versions so that solc is resolved correctly
- dee74ef: migrate batch submitter types to core-utils
- d64b66d: reformat error context for Sentry
- Updated dependencies [6daa408]
- Updated dependencies [ea4041b]
- Updated dependencies [f1f5bf2]
- Updated dependencies [dee74ef]
- Updated dependencies [9ec3ec0]
- Updated dependencies [d64b66d]
- Updated dependencies [5f376ee]
- Updated dependencies [eef1df4]
- Updated dependencies [a76cde5]
- Updated dependencies [e713cd0]
- Updated dependencies [572dcbc]
- Updated dependencies [6014ec0]
  - @eth-optimism/contracts@0.2.8
  - @eth-optimism/core-utils@0.3.2

## 0.2.2

### Patch Changes

- 6d31324: Update release tag for Sentry compatability
- a2f6e83: add default metrics to all batch submitters

## 0.2.1

### Patch Changes

- ab285e4: properly start the batch submitter instead of instantly exiting

## 0.2.0

### Minor Changes

- 5077441: - Use raw transaction in batch submitter -- incompatible with L2Geth v0.1.2.1
  - Pass through raw transaction in l2context

### Patch Changes

- a3dc553: Adds a release version to batch-submitter and data-transport-layer usage of Sentry
- b95dc22: log errors for monotonicity violations
- c7bc0ce: Correctly formatted error object to log exceptions
- Updated dependencies [ce5d596]
- Updated dependencies [1a55f64]
- Updated dependencies [6e8fe1b]
- Updated dependencies [8d4aae4]
- Updated dependencies [c75a0fc]
- Updated dependencies [d4ee2d7]
- Updated dependencies [edb4346]
- Updated dependencies [5077441]
  - @eth-optimism/contracts@0.2.6
  - @eth-optimism/core-utils@0.3.1

## 0.1.12

### Patch Changes

- a0a7956: initialize Sentry and streams in Logger, remove Sentry from Batch Submitter
- Updated dependencies [91460d9]
- Updated dependencies [a0a7956]
- Updated dependencies [0497d7d]
  - @eth-optimism/core-utils@0.3.0
  - @eth-optimism/contracts@0.2.5

## 0.1.11

### Patch Changes

- 35b99b0: add Sentry to TypeScript services for error tracking
- Updated dependencies [35b99b0]
  - @eth-optimism/core-utils@0.2.3

## 0.1.10

### Patch Changes

- 962e31b: removed unused l1 block number logic, added debug logging to batch submitter

## 0.1.9

### Patch Changes

- 3b00b7c: bump private package versions to try triggering a tag

## 0.1.8

### Patch Changes

- 6cbc54d: allow injecting L2 transaction and block context via core-utils (this removes the need to import the now deprecated @eth-optimism/provider package)
- Updated dependencies [6cbc54d]
  - @eth-optimism/core-utils@0.2.0
  - @eth-optimism/contracts@0.2.2

## v0.1.3

- Add tx resubmission logic
- Log when the batch submitter runs low on ETH

## v0.1.2

Adds mnemonic config parsing

## v0.1.1

Final fixes before minnet release.

- Add batch submission timeout
- Log sequencer address
- remove ssh

## v0.1.0

The inital release
