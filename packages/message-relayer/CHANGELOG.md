# @eth-optimism/message-relayer

## 0.1.13

### Patch Changes

- e504bb47: added coverage for getMerkleTreeProof
- Updated dependencies [e0be02e1]
- Updated dependencies [7f7f35c3]
- Updated dependencies [8da04505]
  - @eth-optimism/core-utils@0.6.0
  - @eth-optimism/contracts@0.4.13

## 0.1.12

### Patch Changes

- 21b17edd: Added coverage for packages
- Updated dependencies [888dafca]
- Updated dependencies [eb0854e7]
- Updated dependencies [21b17edd]
- Updated dependencies [dfe3598f]
  - @eth-optimism/contracts@0.4.11
  - @eth-optimism/core-utils@0.5.5

## 0.1.11

### Patch Changes

- 918c08ca: Bump ethers dependency to 5.4.x to support eip1559
- Updated dependencies [918c08ca]
  - @eth-optimism/contracts@0.4.10
  - @eth-optimism/core-utils@0.5.2

## 0.1.10

### Patch Changes

- ecc2f8c1: Patch so contracts package will correctly use the browser-compatible contract artifacts import
- Updated dependencies [ecc2f8c1]
  - @eth-optimism/contracts@0.4.9

## 0.1.9

### Patch Changes

- e4fea5e0: Use latest contracts package for browser compatibility support
- Updated dependencies [e4fea5e0]
  - @eth-optimism/contracts@0.4.8

## 0.1.8

### Patch Changes

- c73c3939: Update the typescript version to `4.3.5`
- Updated dependencies [c73c3939]
  - @eth-optimism/common-ts@0.1.5
  - @eth-optimism/contracts@0.4.5
  - @eth-optimism/core-utils@0.5.1

## 0.1.7

### Patch Changes

- fd86043a: Adds a new l2 to l1 message relaying utility within the message relayer package
- Updated dependencies [0313794b]
- Updated dependencies [049200f4]
- Updated dependencies [21e47e1f]
  - @eth-optimism/contracts@0.4.2
  - @eth-optimism/core-utils@0.5.0

## 0.1.6

### Patch Changes

- 735cd78f: Update relayer package JSON to correctly export all files in dist
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

## 0.1.5

### Patch Changes

- baa3b761: Improve Sentry support, initializing as needed and ensuring ERROR logs route to Sentry
- Updated dependencies [baa3b761]
  - @eth-optimism/common-ts@0.1.3

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
