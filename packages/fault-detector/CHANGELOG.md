# @eth-optimism/fault-detector

## 0.6.2

### Patch Changes

- f9b579d55: Fixes a bug that would cause the fault detector to error out if no outputs had been proposed yet.
- Updated dependencies [fecd42d67]
- Updated dependencies [66cafc00a]
  - @eth-optimism/common-ts@0.8.1
  - @eth-optimism/sdk@2.0.1

## 0.6.1

### Patch Changes

- Updated dependencies [cb19e2f9c]
  - @eth-optimism/sdk@2.0.0

## 0.6.0

### Minor Changes

- b004d1ad4: Updates the fault detector to support Bedrock networks.

### Patch Changes

- Updated dependencies [0e179781b]
- Updated dependencies [5372c9f5b]
- Updated dependencies [4ae94b412]
  - @eth-optimism/common-ts@0.8.0
  - @eth-optimism/sdk@1.10.2

## 0.5.0

### Minor Changes

- 9b2891852: Refactors BaseServiceV2 slightly, merges standard options with regular options

### Patch Changes

- ab8ec365c: Updates BaseServiceV2 so that options are secret by default. Services will have to explicitly mark options as "public" for those options to be logged and included in the metadata metric.
- c6c9c7dbf: Fault detector will now wait for providers to be connected
- Updated dependencies [e23f60f63]
- Updated dependencies [ab8ec365c]
- Updated dependencies [ba8b94a60]
- Updated dependencies [9b2891852]
- Updated dependencies [d1f9098f9]
- Updated dependencies [c6c9c7dbf]
- Updated dependencies [ffcee1013]
- Updated dependencies [eceb0de1d]
  - @eth-optimism/common-ts@0.7.0
  - @eth-optimism/sdk@1.9.0
  - @eth-optimism/contracts@0.5.40

## 0.4.0

### Minor Changes

- ab5c1b897: Includes a new event caching mechanism for running the fault detector against Geth.

### Patch Changes

- 1d3c749a2: Bumps the version of ts-node used
- Updated dependencies [1d3c749a2]
- Updated dependencies [767585b07]
- Updated dependencies [c975c9620]
- Updated dependencies [1d3c749a2]
- Updated dependencies [136ea1785]
  - @eth-optimism/contracts@0.5.39
  - @eth-optimism/sdk@1.8.0
  - @eth-optimism/core-utils@0.12.0
  - @eth-optimism/common-ts@0.6.8

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
