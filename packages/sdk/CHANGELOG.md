# @eth-optimism/sdk

## 1.0.2

### Patch Changes

- d49feca1: Comment out non-functional getMessagesByAddress function
- Updated dependencies [88601cb7]
  - @eth-optimism/contracts@0.5.18

## 1.0.1

### Patch Changes

- 7ae1c67f: Update package json to include correct repo link
- 47e5d118: Tighten type restriction on ProviderLike
- Updated dependencies [175ae0bf]
  - @eth-optimism/contracts@0.5.17

## 1.0.0

### Major Changes

- 84f63c49: Update README and bump SDK to 1.0.0

### Patch Changes

- 42227d69: Fix typo in constructor docstring

## 0.2.5

### Patch Changes

- b66e3131: Add a function for waiting for a particular message status
- Updated dependencies [962f36e4]
- Updated dependencies [f2179e37]
- Updated dependencies [b6a4fa4b]
- Updated dependencies [b7c0a5ca]
- Updated dependencies [5a6f539c]
- Updated dependencies [27d8942e]
  - @eth-optimism/contracts@0.5.16
  - @eth-optimism/core-utils@0.8.1

## 0.2.4

### Patch Changes

- 44420939: 1. Fix a bug in `L2Provider.getL1GasPrice()`
  2. Make it easier to get correct estimates from `L2Provider.estimateL1Gas()` and `L2.estimateL2GasCost`.

## 0.2.3

### Patch Changes

- f37c283c: Have SDK properly handle case when no batches are submitted yet
- 3f4d3c13: Have SDK wait for transactions in getMessagesByTransaction
- 0c54e60e: Add approval functions to the SDK
- Updated dependencies [0b4453f7]
- Updated dependencies [78298782]
  - @eth-optimism/core-utils@0.8.0
  - @eth-optimism/contracts@0.5.15

## 0.2.2

### Patch Changes

- fd6ea3ee: Adds support for depositing or withdrawing to a target address
- 5ffb5fcf: Removes the getTokenBridgeMessagesByAddress function
- dd4b2055: This update implements the asL2Provider function
- f08c06a8: Updates the SDK to include default bridges for the local Optimism network (31337)
- da53dc64: Have SDK sort deposits/withdrawals descending by block number
- Updated dependencies [b4165299]
- Updated dependencies [3c2acd91]
  - @eth-optimism/core-utils@0.7.7
  - @eth-optimism/contracts@0.5.14

## 0.2.1

### Patch Changes

- Updated dependencies [438bc78a]
  - @eth-optimism/contracts@0.5.13

## 0.2.0

### Minor Changes

- dd9683bb: Correctly export SDK contents

## 0.1.0

### Minor Changes

- cb65f3d8: Beta release of the Optimism SDK

### Patch Changes

- ba14c59d: Updates various ethers dependencies to their latest versions
- 64e746b6: Have SDK include ethers as a peer dependency
- Updated dependencies [ba14c59d]
  - @eth-optimism/contracts@0.5.12
  - @eth-optimism/core-utils@0.7.6

## 0.0.7

### Patch Changes

- Updated dependencies [e631c39c]
  - @eth-optimism/contracts@0.5.11

## 0.0.6

### Patch Changes

- Updated dependencies [ad94b9d1]
  - @eth-optimism/core-utils@0.7.5
  - @eth-optimism/contracts@0.5.10

## 0.0.5

### Patch Changes

- Updated dependencies [ba96a455]
- Updated dependencies [c3e85fef]
  - @eth-optimism/core-utils@0.7.4
  - @eth-optimism/contracts@0.5.9

## 0.0.4

### Patch Changes

- Updated dependencies [b3efb8b7]
- Updated dependencies [279603e5]
- Updated dependencies [b6040bb3]
  - @eth-optimism/contracts@0.5.8

## 0.0.3

### Patch Changes

- Updated dependencies [b6f89fad]
  - @eth-optimism/contracts@0.5.7

## 0.0.2

### Patch Changes

- Updated dependencies [bbd42e03]
- Updated dependencies [453f0774]
  - @eth-optimism/contracts@0.5.6
