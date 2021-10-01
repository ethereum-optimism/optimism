# data transport layer

## 0.4.8

### Patch Changes

- e0be02e1: Add fallback provider support to DTL using helper function in core-utils
- Updated dependencies [e0be02e1]
- Updated dependencies [7f7f35c3]
- Updated dependencies [8da04505]
  - @eth-optimism/core-utils@0.6.0
  - @eth-optimism/contracts@0.4.13

## 0.4.7

### Patch Changes

- 21b17edd: Added coverage for packages
- Updated dependencies [888dafca]
- Updated dependencies [eb0854e7]
- Updated dependencies [21b17edd]
- Updated dependencies [dfe3598f]
  - @eth-optimism/contracts@0.4.11
  - @eth-optimism/core-utils@0.5.5

## 0.4.6

### Patch Changes

- 918c08ca: Bump ethers dependency to 5.4.x to support eip1559
- Updated dependencies [918c08ca]
  - @eth-optimism/contracts@0.4.10
  - @eth-optimism/core-utils@0.5.2

## 0.4.5

### Patch Changes

- b5b9fd89: Migrate to using `ethers.StaticJsonRpcProvider`
- Updated dependencies [ecc2f8c1]
  - @eth-optimism/contracts@0.4.9

## 0.4.4

### Patch Changes

- c73c3939: Update the typescript version to `4.3.5`
- Updated dependencies [c73c3939]
  - @eth-optimism/common-ts@0.1.5
  - @eth-optimism/contracts@0.4.5
  - @eth-optimism/core-utils@0.5.1

## 0.4.3

### Patch Changes

- 390fd8a6: Allow the L1 gas price to be fetched from either the sequencer or a L1 provider based on the config `--l1-gas-price-backend` as well as overriding the config by using a query param. Valid values are `l1` or `l2` and it defaults to `l1`
- 049200f4: removed unused functions from core-utils
- Updated dependencies [0313794b]
- Updated dependencies [049200f4]
- Updated dependencies [21e47e1f]
  - @eth-optimism/contracts@0.4.2
  - @eth-optimism/core-utils@0.5.0

## 0.4.2

### Patch Changes

- 70b8ae84: Attach correct TransportDB object to method handler

## 0.4.1

### Patch Changes

- 67eedaf6: Correctly bind the event handlers to the correct `this` in the missing event error path
- Updated dependencies [224b04c0]
  - @eth-optimism/core-utils@0.4.7

## 0.4.0

### Minor Changes

- 2e72fd90: Update AddressSet event to speed search up a bit. Breaks AddressSet API.
- 8582fc16: Define L1 Starting block via OwnershipTransferred (occurring on block 1) rather than AddressSet (occuring on block 2 onwards)

### Patch Changes

- 0b91df42: Adds additional code into the DTL to defend against situations where an RPC provider might be missing an event.
- 8fee7bed: Add extra overflow protection for the DTL types
- ca7d65db: Removes a function that was previously used for backwards compatibility but is no longer necessary
- 16f68159: Have DTL log failed HTTP requests as ERROR instead of INFO
- a415d017: Updates the DTL to use the same L2 chain ID everywhere
- 29431d6a: Add highest L1 and L2 block number Gauge metrics to DTL
- 5c89c45f: Move the metric prefix string to a label #1047
- b8e2d685: Add replica sync test to integration tests; handle 0 L2 blocks in DTL
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

## 0.3.6

### Patch Changes

- baa3b761: Improve Sentry support, initializing as needed and ensuring ERROR logs route to Sentry
- Updated dependencies [baa3b761]
  - @eth-optimism/common-ts@0.1.3

## 0.3.5

### Patch Changes

- 1b692415: incorrect parsing of eth_getBlockRange result

## 0.3.4

### Patch Changes

- f1b27318: Represent gaslimit as a string to avoid an overflow
- 750a5021: Remove dead imports from core-utils
- 1293825c: Fix gasLimit overflow
- a75f05b7: Fixes a bug that prevented verifiers from syncing properly with the DTL
- e52ccd98: Logs the error stacktrace for a failed HTTP request
- 8ac4c74c: improve slow blocking JSON parsing that occurs during l2 sync
- Updated dependencies [a64f8161]
- Updated dependencies [4e03f8a9]
- Updated dependencies [8e2bfd07]
- Updated dependencies [750a5021]
- Updated dependencies [c2b6e14b]
- Updated dependencies [245136f1]
  - @eth-optimism/core-utils@0.4.5
  - @eth-optimism/contracts@0.3.5

## 0.3.3

### Patch Changes

- e4c3b4b: Add Sentry and Metrics switches and environment tag to DTL
- Updated dependencies [5e5d4a1]
  - @eth-optimism/contracts@0.3.3

## 0.3.2

### Patch Changes

- f5185bb: Fix bug with replica syncing where contract creations would fail in replicas but pass in the sequencer. This was due to the change from a custom batched tx serialization to the batch serialzation for txs being regular RLP encoding
- Updated dependencies [7dd2f72]
  - @eth-optimism/contracts@0.3.2

## 0.3.1

### Patch Changes

- e28cec7: Fixes a bug where L2 synced transactions were not RLP encoded
- 96a586e: Migrate bcfg interface to core-utils
- fa4898a: Explicitly log error messages so that they do not show as empty objects
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

- b799caa: Parse and index the value field in the data transport layer
- b799caa: Account for the off by one with regards to the l2geth block number and the CTC index
- b799caa: Remove legacy transaction deserialization to support RLP batch encoding
- b799caa: Prevent access of null value in L1 transaction deserialization
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
- ce7fa52: Add an additional enum for EthSign transactions as they now are batch submitted with 2 different enum values
- 575bcf6: add environment and network to dtl, move metric init to app from base-service
- Updated dependencies [1d40586]
- Updated dependencies [ce7fa52]
- Updated dependencies [575bcf6]
- Updated dependencies [6dc1877]
  - @eth-optimism/common-ts@0.1.1
  - @eth-optimism/contracts@0.2.10
  - @eth-optimism/core-utils@0.4.1

## 0.2.4

### Patch Changes

- 47e40a2: Update the config parsing so that it gives a better error message
- a0a0052: Parse and index the value field in the data transport layer
- 34ab776: Better error logging in the DTL
- e6350e2: add metrics to measure http endpoint latency
- 28dc442: move metrics, logger, and base-service to new common-ts package
- a0a0052: Prevent access of null value in L1 transaction deserialization
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
- 01a2e7d: Clean up config parsing to match CLI argument configuration
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
