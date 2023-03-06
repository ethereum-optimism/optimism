# @eth-optimism/gas-oracle

## 0.1.13

### Patch Changes

- 9b61c84c9: build(deps): bump golang.org/x/net from 0.0.0-20211112202133-69e39bad7dc2 to 0.7.0 in /gas-oracle
- f13b31e04: build(deps): bump golang.org/x/sys from 0.0.0-20220310020820-b874c991c1a5 to 0.1.0 in /gas-oracle

## 0.1.12

### Patch Changes

- 6f458607: Bump go-ethereum to 1.10.17

## 0.1.11

### Patch Changes

- 160f4c3d: Update docker image to use golang 1.18.0

## 0.1.10

### Patch Changes

- 162ff89c: Fixes a bug that would cause the service to crash on startup if the RPC URLs were not immediately available

## 0.1.9

### Patch Changes

- c535b3a5: Allow configurable base fee update poll time with `GAS_PRICE_ORACLE_L1_BASE_FEE_EPOCH_LENGTH_SECONDS`

## 0.1.8

### Patch Changes

- 88601cb7: Refactored Dockerfiles

## 0.1.7

### Patch Changes

- fed748e0: Update to go-ethereum v1.10.16

## 0.1.6

### Patch Changes

- b3efb8b7: String update to change the system name from OE to Optimism

## 0.1.5

### Patch Changes

- 40b6c5bd: Update the flag parsing of the average block gas limit

## 0.1.4

### Patch Changes

- 9eed33c4: fix rounding error in average gas/epoch calculation

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
