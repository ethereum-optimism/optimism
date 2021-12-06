# @eth-optimism/gas-oracle

## 0.1.3

### Patch Changes

- 3af7ce3f: Meter gas usage based on gas used in block instead of assuming max gas usage per block

## 0.1.2

### Patch Changes

- 5a3996ec: Fixed gas-oacle tx/not_significant metric name

## 0.1.1

### Patch Changes

- e4067d4c: Fix the gas oracle gas price prometheus metric

## 0.1.0

### Minor Changes

- d89b5005: Add L1 base fee, add breaking config options
- 81ccd6e4: `regenesis/0.5.0` release

### Patch Changes

- d7fa6809: Bumps the go-ethereum dependency version to v1.10.9
- b70ee70c: upgraded to solidity 0.8.9
- 4f805355: Bump go-ethereum dep to v1.10.10
- 1527cf6f: Use the configured gas price when updating the L1 base fee in L2 state

## 0.0.3

### Patch Changes

- 8c4f479c: Add additional logging in the `gas-oracle`

## 0.0.2

### Patch Changes

- ce3c353b: Initial implementation of the `gas-oracle`
