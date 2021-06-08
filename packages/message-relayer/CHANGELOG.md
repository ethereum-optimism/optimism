# @eth-optimism/message-relayer

## 0.1.4

### Patch Changes

- 9d39121b: Adds a README and cleans up the interface for generating messages and proofs
- 86708bb5: Adds a new set of tools for generating messages to be relayed and their proofs
- 064c03af: Removes spreadsheet mode from the message relayer
- Updated dependencies [a64f8161]
- Updated dependencies [4e03f8a9]
- Updated dependencies [8e2bfd07]
- Updated dependencies [750a5021]
- Updated dependencies [c2b6e14b]
- Updated dependencies [245136f1]
  - @eth-optimism/core-utils@0.4.5
  - @eth-optimism/contracts@0.3.5

## 0.1.3

### Patch Changes

- e3b138b: Fix to avoid getting OOM killed when the relayer runs for a long period of time
- Updated dependencies [5e5d4a1]
  - @eth-optimism/contracts@0.3.3

## 0.1.2

### Patch Changes

- 96a586e: Migrate bcfg interface to core-utils
- fa4898a: Explicitly log error messages so that they do not show as empty objects
- Updated dependencies [96a586e]
- Updated dependencies [0c16805]
- Updated dependencies [775118a]
  - @eth-optimism/core-utils@0.4.3
  - @eth-optimism/common-ts@0.1.2
  - @eth-optimism/contracts@0.3.1

## 0.1.1

### Patch Changes

- aedf931: Add updated config parsing in a backwards compatible way
- d723b2a: Don't log the config options at startup because it contains secrets

## 0.1.0

### Minor Changes

- b799caa: Updates to use RLP encoded transactions in batches for the `v0.3.0` release

### Patch Changes

- 33fcd84: Add a check for `OVM_L2MessageRelayer` in the AddressManager before attempting to relay messages to help surface errors more quickly
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

## 0.0.5

### Patch Changes

- 28dc442: move metrics, logger, and base-service to new common-ts package
- Updated dependencies [28dc442]
- Updated dependencies [d2091d4]
- Updated dependencies [a0a0052]
- Updated dependencies [0ef3069]
  - @eth-optimism/common-ts@0.1.0
  - @eth-optimism/core-utils@0.4.0
  - @eth-optimism/contracts@0.2.9

## 0.0.4

### Patch Changes

- Updated dependencies [91460d9]
- Updated dependencies [a0a7956]
- Updated dependencies [0497d7d]
  - @eth-optimism/core-utils@0.3.0
  - @eth-optimism/contracts@0.2.5

## 0.0.3

### Patch Changes

- 3b00b7c: bump private package versions to try triggering a tag

## 0.0.2

### Patch Changes

- Updated dependencies [6cbc54d]
  - @eth-optimism/core-utils@0.2.0
  - @eth-optimism/contracts@0.2.2
