# @eth-optimism/common-ts

## 0.6.7

### Patch Changes

- Updated dependencies [1e76cdb86]
  - @eth-optimism/core-utils@0.11.0

## 0.6.6

### Patch Changes

- ce7da914: Minor update to BaseServiceV2 to keep the raw body around when requests are made.

## 0.6.5

### Patch Changes

- 7215f4ce: Bump ethers to 5.7.0 globally
- d7679ca4: Add source maps
- Updated dependencies [7215f4ce]
- Updated dependencies [206f6033]
  - @eth-optimism/core-utils@0.10.1

## 0.6.4

### Patch Changes

- Updated dependencies [dbfea116]
  - @eth-optimism/core-utils@0.10.0

## 0.6.3

### Patch Changes

- Updated dependencies [0df744f6]
- Updated dependencies [8ae39154]
- Updated dependencies [dac4a9f0]
  - @eth-optimism/core-utils@0.9.3

## 0.6.2

### Patch Changes

- Updated dependencies [0bf3b9b4]
- Updated dependencies [8d26459b]
- Updated dependencies [4477fe9f]
  - @eth-optimism/core-utils@0.9.2

## 0.6.1

### Patch Changes

- Updated dependencies [f9fee446]
  - @eth-optimism/core-utils@0.9.1

## 0.6.0

### Minor Changes

- 3d1cb720: Add version to healthz for convenience

### Patch Changes

- Updated dependencies [700dcbb0]
  - @eth-optimism/core-utils@0.9.0

## 0.5.0

### Minor Changes

- cb71fcde: Make typescript type more permissive for MetricsV2

### Patch Changes

- 10e41522: Fix potential metrics DoS vector in recent commit to BSV2

## 0.4.0

### Minor Changes

- 52b26878: More gracefully shut down base service

### Patch Changes

- c201f3f1: Collect default node metrics
- 29ff7462: Revert es target back to 2017
- Updated dependencies [29ff7462]
  - @eth-optimism/core-utils@0.8.7

## 0.3.1

### Patch Changes

- 9ba869a7: Log server messages to logger instead of stdout
- 050859fd: Include default options in metadata metric

## 0.3.0

### Minor Changes

- d9e39931: Minor upgrade to BaseServiceV2 to expose a full customizable server, instead of just metrics.
- 84a8934c: BaseServiceV2 exposes service name and version as standard synthetic metric

## 0.2.10

### Patch Changes

- 9ecbf3e5: Expose service internal options as environment or cli options

## 0.2.9

### Patch Changes

- Updated dependencies [17962ca9]
  - @eth-optimism/core-utils@0.8.6

## 0.2.8

### Patch Changes

- f16383f2: Have legacy BaseService metrics bind to 0.0.0.0 by default
- d18ae135: Updates all ethers versions in response to BN.js bug
- Updated dependencies [d18ae135]
  - @eth-optimism/core-utils@0.8.5

## 0.2.7

### Patch Changes

- Updated dependencies [5cb3a5f7]
- Updated dependencies [6b9fc055]
  - @eth-optimism/core-utils@0.8.4

## 0.2.6

### Patch Changes

- b57014d1: Update to typescript@4.6.2
- Updated dependencies [b57014d1]
  - @eth-optimism/core-utils@0.8.3

## 0.2.5

### Patch Changes

- e36b085c: Adds hard stop to BaseServiceV2 when multiple exit signals are received
- c1957126: Update Dockerfile to use Alpine
- 51673b90: Have BaseServiceV2 throw when options are undefined
- 7a179003: Adds the jsonRpcProvider validator as an input validator
- Updated dependencies [c1957126]
  - @eth-optimism/core-utils@0.8.2

## 0.2.4

### Patch Changes

- f981b8da: Properly exposes metrics as part of a metrics server at port 7300

## 0.2.3

### Patch Changes

- f7761058: Update log lines for service shutdown
- 5ae15042: Update metric names to include proper snake_case for strings that include "L1" or "L2"
- 5cd1e996: Have BaseServiceV2 add spaces to environment variable names

## 0.2.2

### Patch Changes

- b3f9bdef: Have BaseServiceV2 gracefully catch exit signals
- e53b5783: Introduces the new BaseServiceV2 class.

## 0.2.1

### Patch Changes

- 243f33e5: Standardize package json file format

## 0.2.0

### Minor Changes

- 81ccd6e4: `regenesis/0.5.0` release

## 0.1.6

### Patch Changes

- 6d3e1d7f: Update dependencies

## 0.1.5

### Patch Changes

- c73c3939: Update the typescript version to `4.3.5`

## 0.1.4

### Patch Changes

- 5c89c45f: Move the metric prefix string to a label #1047

## 0.1.3

### Patch Changes

- baa3b761: Improve Sentry support, initializing as needed and ensuring ERROR logs route to Sentry

## 0.1.2

### Patch Changes

- 0c16805: add metrics server to common-ts and batch submitter

## 0.1.1

### Patch Changes

- 1d40586: Removed various unused dependencies
- 575bcf6: add environment and network to dtl, move metric init to app from base-service

## 0.1.0

### Minor Changes

- 28dc442: move metrics, logger, and base-service to new common-ts package
