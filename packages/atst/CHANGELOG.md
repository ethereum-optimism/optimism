# @eth-optimism/atst

## 0.2.0

### Minor Changes

- dcd13eec1: Update readAttestations and prepareWriteAttestation to handle keys longer than 32 bytes
- 9fd5be8e2: Remove broken allowFailures as option
- 3f4a43542: Move react api to @eth-optimism/atst/react so react isn't required to run the core sdk
- 71727eae9: Fix main and module in atst package.json
- 3d5f26c49: Deprecate parseAttestationBytes and createRawKey in favor for createKey, createValue

### Patch Changes

- 68bbe48b6: Update docs
- 6fea2f2db: Fixed bug with atst not defaulting to currently connected chain

## 0.1.0

### Minor Changes

- a312af15d: Make type parsing more intuitive
- 82a033fed: Fix string type that should be `0x${string}`

### Patch Changes

- 11bb01851: Add new atst package
- 7c37d262a: Release ATST
