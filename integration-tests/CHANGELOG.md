# @eth-optimism/integration-tests

## 0.0.5

### Patch Changes

- 467d6cb: Adds a test for contract deployments that run out of gas

## 0.0.4

### Patch Changes

- b799caa: Add support for parsed revert reasons in DoEstimateGas
- b799caa: Update minimum response from estimate gas
- b799caa: Add value transfer support to ECDSAContractAccount
- b799caa: Update expected gas prices based on minimum of 21k value

## 0.0.3

### Patch Changes

- 6daa408: update hardhat versions so that solc is resolved correctly
- 5b9be2e: Correctly set the OVM context based on the L1 values during `eth_call`. This will also set it during `eth_estimateGas`. Add tests for this in the integration tests

## 0.0.2

### Patch Changes

- 6bcf22b: Add contracts for OVM context test coverage and add tests
