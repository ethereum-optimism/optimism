# @eth-optimism/fault-detector

## 0.3.2

### Patch Changes

- 97b5f578c: Fixes how versions are imported for BaseServiceV2 services
  - @eth-optimism/sdk@1.6.11

## 0.3.1

### Patch Changes

- Updated dependencies [1e76cdb86]
  - @eth-optimism/core-utils@0.11.0
  - @eth-optimism/common-ts@0.6.7
  - @eth-optimism/contracts@0.5.38
  - @eth-optimism/sdk@1.6.10

## 0.3.0

### Minor Changes

- 4a5e1832: Updates metrics to use better labels.

### Patch Changes

- Updated dependencies [e2faaa8b]
  - @eth-optimism/sdk@1.6.5

## 0.2.7

### Patch Changes

- 7215f4ce: Bump ethers to 5.7.0 globally
- 17999a54: Adds a fault status API to the Fault Detector.
- 2f058b84: Fixes a small bug in the fault detector that would cause errors for testnets where the fault proof window is extremely short.
- Updated dependencies [7215f4ce]
- Updated dependencies [206f6033]
- Updated dependencies [d7679ca4]
  - @eth-optimism/common-ts@0.6.5
  - @eth-optimism/contracts@0.5.36
  - @eth-optimism/core-utils@0.10.1
  - @eth-optimism/sdk@1.6.4

## 0.2.6

### Patch Changes

- Updated dependencies [b27d0fa7]
- Updated dependencies [dbfea116]
- Updated dependencies [299157e7]
  - @eth-optimism/sdk@1.6.1
  - @eth-optimism/core-utils@0.10.0
  - @eth-optimism/contracts@0.5.34
  - @eth-optimism/common-ts@0.6.4

## 0.2.5

### Patch Changes

- 98206b7e: Properly handle connection failures for L2 node

## 0.2.4

### Patch Changes

- 89d01f2e: Update dev deps
- Updated dependencies [6e3449ba]
- Updated dependencies [f9fee446]
  - @eth-optimism/contracts@0.5.30
  - @eth-optimism/core-utils@0.9.1
  - @eth-optimism/sdk@1.2.1
  - @eth-optimism/common-ts@0.6.1

## 0.2.3

### Patch Changes

- 977493bc: Update SDK version and usage to account for new constructor
- 2296cf81: Fix bug where FD would try to sync beyond local tip
- Updated dependencies [977493bc]
- Updated dependencies [700dcbb0]
- Updated dependencies [3d1cb720]
  - @eth-optimism/sdk@1.2.0
  - @eth-optimism/core-utils@0.9.0
  - @eth-optimism/common-ts@0.6.0
  - @eth-optimism/contracts@0.5.29

## 0.2.2

### Patch Changes

- Updated dependencies [cb71fcde]
- Updated dependencies [10e41522]
  - @eth-optimism/common-ts@0.5.0

## 0.2.1

### Patch Changes

- 29ff7462: Revert es target back to 2017
- Updated dependencies [27234f68]
- Updated dependencies [c201f3f1]
- Updated dependencies [29ff7462]
- Updated dependencies [52b26878]
  - @eth-optimism/contracts@0.5.28
  - @eth-optimism/common-ts@0.4.0
  - @eth-optimism/core-utils@0.8.7
  - @eth-optimism/sdk@1.1.9

## 0.2.0

### Minor Changes

- 84a8934c: BaseServiceV2 exposes service name and version as standard synthetic metric

### Patch Changes

- 37dfe4f6: Smarter starting height for fault-detector
- 6fe58eb2: Fix order in which a metric was bumped then emitted to fix off by one issue
- Updated dependencies [d9e39931]
- Updated dependencies [84a8934c]
  - @eth-optimism/common-ts@0.3.0

## 0.1.1

### Patch Changes

- d18ae135: Updates all ethers versions in response to BN.js bug
- Updated dependencies [f16383f2]
- Updated dependencies [d18ae135]
  - @eth-optimism/common-ts@0.2.8
  - @eth-optimism/core-utils@0.8.5
  - @eth-optimism/sdk@1.1.6

## 0.1.0

### Minor Changes

- 2177c8ef: Releases the first public version of the fault detector

### Patch Changes

- @eth-optimism/sdk@1.1.4
