# @eth-optimism/replica-healthcheck

## 1.1.12

### Patch Changes

- 97b5f578c: Fixes how versions are imported for BaseServiceV2 services

## 1.1.11

### Patch Changes

- Updated dependencies [1e76cdb86]
  - @eth-optimism/core-utils@0.11.0
  - @eth-optimism/common-ts@0.6.7

## 1.1.10

### Patch Changes

- Updated dependencies [ce7da914]
  - @eth-optimism/common-ts@0.6.6

## 1.1.9

### Patch Changes

- 7215f4ce: Bump ethers to 5.7.0 globally
- Updated dependencies [7215f4ce]
- Updated dependencies [206f6033]
- Updated dependencies [d7679ca4]
  - @eth-optimism/common-ts@0.6.5
  - @eth-optimism/core-utils@0.10.1

## 1.1.8

### Patch Changes

- Updated dependencies [dbfea116]
  - @eth-optimism/core-utils@0.10.0
  - @eth-optimism/common-ts@0.6.4

## 1.1.7

### Patch Changes

- Updated dependencies [0df744f6]
- Updated dependencies [8ae39154]
- Updated dependencies [dac4a9f0]
  - @eth-optimism/core-utils@0.9.3
  - @eth-optimism/common-ts@0.6.3

## 1.1.6

### Patch Changes

- Updated dependencies [0bf3b9b4]
- Updated dependencies [8d26459b]
- Updated dependencies [4477fe9f]
  - @eth-optimism/core-utils@0.9.2
  - @eth-optimism/common-ts@0.6.2

## 1.1.5

### Patch Changes

- Updated dependencies [f9fee446]
  - @eth-optimism/core-utils@0.9.1
  - @eth-optimism/common-ts@0.6.1

## 1.1.4

### Patch Changes

- Updated dependencies [700dcbb0]
- Updated dependencies [3d1cb720]
  - @eth-optimism/core-utils@0.9.0
  - @eth-optimism/common-ts@0.6.0

## 1.1.3

### Patch Changes

- Updated dependencies [cb71fcde]
- Updated dependencies [10e41522]
  - @eth-optimism/common-ts@0.5.0

## 1.1.2

### Patch Changes

- 29ff7462: Revert es target back to 2017
- Updated dependencies [c201f3f1]
- Updated dependencies [29ff7462]
- Updated dependencies [52b26878]
  - @eth-optimism/common-ts@0.4.0
  - @eth-optimism/core-utils@0.8.7

## 1.1.1

### Patch Changes

- Updated dependencies [9ba869a7]
- Updated dependencies [050859fd]
  - @eth-optimism/common-ts@0.3.1

## 1.1.0

### Minor Changes

- 84a8934c: BaseServiceV2 exposes service name and version as standard synthetic metric

### Patch Changes

- Updated dependencies [d9e39931]
- Updated dependencies [84a8934c]
  - @eth-optimism/common-ts@0.3.0

## 1.0.9

### Patch Changes

- Updated dependencies [9ecbf3e5]
  - @eth-optimism/common-ts@0.2.10

## 1.0.8

### Patch Changes

- Updated dependencies [17962ca9]
  - @eth-optimism/core-utils@0.8.6
  - @eth-optimism/common-ts@0.2.9

## 1.0.7

### Patch Changes

- d18ae135: Updates all ethers versions in response to BN.js bug
- Updated dependencies [f16383f2]
- Updated dependencies [d18ae135]
  - @eth-optimism/common-ts@0.2.8
  - @eth-optimism/core-utils@0.8.5

## 1.0.6

### Patch Changes

- Updated dependencies [5cb3a5f7]
- Updated dependencies [6b9fc055]
  - @eth-optimism/core-utils@0.8.4
  - @eth-optimism/common-ts@0.2.7

## 1.0.5

### Patch Changes

- bc289e91: Fixes a bug that would cause the service to stop properly checking blocks when the target client consistently leads the reference client

## 1.0.4

### Patch Changes

- b57014d1: Update to typescript@4.6.2
- Updated dependencies [b57014d1]
  - @eth-optimism/common-ts@0.2.6
  - @eth-optimism/core-utils@0.8.3

## 1.0.3

### Patch Changes

- c1957126: Update Dockerfile to use Alpine
- Updated dependencies [e36b085c]
- Updated dependencies [c1957126]
- Updated dependencies [51673b90]
- Updated dependencies [7a179003]
  - @eth-optimism/common-ts@0.2.5
  - @eth-optimism/core-utils@0.8.2

## 1.0.2

### Patch Changes

- f981b8da: Fixes a bug in the replica-healthcheck docker file
- 032731b5: Add checks and metrics for dead networks
- Updated dependencies [f981b8da]
  - @eth-optimism/common-ts@0.2.4

## 1.0.1

### Patch Changes

- 1c685f76: Fixes a bug in the replica-healthcheck dockerfile
- 5cd1e996: Have BaseServiceV2 add spaces to environment variable names
- Updated dependencies [f7761058]
- Updated dependencies [5ae15042]
- Updated dependencies [5cd1e996]
  - @eth-optimism/common-ts@0.2.3

## 1.0.0

### Major Changes

- e264f03f: Rewrite replica-healthcheck with BaseServiceV2

### Patch Changes

- Updated dependencies [b3f9bdef]
- Updated dependencies [e53b5783]
  - @eth-optimism/common-ts@0.2.2

## 0.3.11

### Patch Changes

- Updated dependencies [42227d69]
- Updated dependencies [84f63c49]
  - @eth-optimism/sdk@1.0.0

## 0.3.10

### Patch Changes

- dad644b4: Fix bug in replica healthcheck dockerfile
- Updated dependencies [b66e3131]
- Updated dependencies [5a6f539c]
- Updated dependencies [27d8942e]
  - @eth-optimism/sdk@0.2.5
  - @eth-optimism/core-utils@0.8.1

## 0.3.9

### Patch Changes

- d4b0e193: Fix bug in replica healthcheck dockerfile
- Updated dependencies [44420939]
  - @eth-optimism/sdk@0.2.4

## 0.3.8

### Patch Changes

- d3d70291: Use asL2Provider instead of injectL2Context in bss and healthcheck service.
- Updated dependencies [f37c283c]
- Updated dependencies [3f4d3c13]
- Updated dependencies [0b4453f7]
- Updated dependencies [0c54e60e]
  - @eth-optimism/sdk@0.2.3
  - @eth-optimism/core-utils@0.8.0

## 0.3.7

### Patch Changes

- Updated dependencies [b4165299]
- Updated dependencies [3c2acd91]
  - @eth-optimism/core-utils@0.7.7

## 0.3.6

### Patch Changes

- ba14c59d: Updates various ethers dependencies to their latest versions
- Updated dependencies [ba14c59d]
  - @eth-optimism/core-utils@0.7.6

## 0.3.5

### Patch Changes

- Updated dependencies [ad94b9d1]
  - @eth-optimism/core-utils@0.7.5

## 0.3.4

### Patch Changes

- Updated dependencies [ba96a455]
- Updated dependencies [c3e85fef]
  - @eth-optimism/core-utils@0.7.4

## 0.3.3

### Patch Changes

- Updated dependencies [584cbc25]
  - @eth-optimism/core-utils@0.7.3

## 0.3.2

### Patch Changes

- 8e634b49: Fix package JSON issues
- Updated dependencies [8e634b49]
  - @eth-optimism/core-utils@0.7.2

## 0.3.1

### Patch Changes

- 243f33e5: Standardize package json file format
- Updated dependencies [243f33e5]
  - @eth-optimism/common-ts@0.2.1
  - @eth-optimism/core-utils@0.7.1

## 0.3.0

### Minor Changes

- 81ccd6e4: `regenesis/0.5.0` release

### Patch Changes

- 222a3eef: Add 'User-Agent' to the http headers for ethers providers
- a98a1884: Fixes dependencies instead of using caret constraints
- Updated dependencies [3ce62c81]
- Updated dependencies [cee2a464]
- Updated dependencies [222a3eef]
- Updated dependencies [896168e2]
- Updated dependencies [7c352b1e]
- Updated dependencies [b70ee70c]
- Updated dependencies [20c8969b]
- Updated dependencies [83a449c4]
- Updated dependencies [81ccd6e4]
- Updated dependencies [6d32d701]
  - @eth-optimism/core-utils@0.7.0
  - @eth-optimism/common-ts@0.2.0

## 0.2.4

### Patch Changes

- 6d3e1d7f: Update dependencies
- Updated dependencies [6d3e1d7f]
- Updated dependencies [2e929aa9]
  - @eth-optimism/common-ts@0.1.6
  - @eth-optimism/core-utils@0.6.1

## 0.2.3

### Patch Changes

- Updated dependencies [e0be02e1]
- Updated dependencies [8da04505]
  - @eth-optimism/core-utils@0.6.0

## 0.2.2

### Patch Changes

- 4262ea2c: Add tx write latency cron check

## 0.2.1

### Patch Changes

- 91c6287e: Bug fix from leftover error during testing

## 0.2.0

### Minor Changes

- 4319e455: Add replica-healthcheck to monorepo
