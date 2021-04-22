# data transport layer

## 0.2.1

### Patch Changes

- a3dc553: Adds a release version to batch-submitter and data-transport-layer usage of Sentry
- 27f32ca: Allow the DTL to provide data from either L1 or L2, configurable via a query param sent by the client.
  The config option `default-backend` can be used to specify the backend to be
  used if the query param is not specified. This allows it to be backwards
  compatible with how the DTL was previously used.
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

## 0.2.0

### Minor Changes

- 91460d9: add Metrics and use in base-service, rename DTL services to avoid spaces

### Patch Changes

- 0497d7d: Re-organize event typings to core-utils
- Updated dependencies [91460d9]
- Updated dependencies [a0a7956]
- Updated dependencies [0497d7d]
  - @eth-optimism/core-utils@0.3.0
  - @eth-optimism/contracts@0.2.5

## 0.1.6

### Patch Changes

- 35b99b0: add Sentry to TypeScript services for error tracking
- Updated dependencies [35b99b0]
  - @eth-optimism/core-utils@0.2.3

## 0.1.5

### Patch Changes

- 01eaf2c: added extra logs to base-service / dtl to improve observability
- Updated dependencies [01eaf2c]
  - @eth-optimism/core-utils@0.2.2

## 0.1.4

### Patch Changes

- 3b00b7c: bump private package versions to try triggering a tag

## 0.1.3

### Patch Changes

- Updated dependencies [6cbc54d]
  - @eth-optimism/core-utils@0.2.0
  - @eth-optimism/contracts@0.2.2

## v0.1.2

- Fix bug in L2 sync

## v0.1.1

- Prioritize L2 synced API requests
- Stop syncing L2 at a certain height

## v0.1.0

- Sync From L1
- Sync From L2
- Initial Release
