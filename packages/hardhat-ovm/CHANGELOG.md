# @eth-optimism/hardhat-ovm

## 0.2.0

### Minor Changes

- b799caa: Updates to use RLP encoded transactions in batches for the `v0.3.0` release

## 0.1.2

### Patch Changes

- 1d40586: Removed various unused dependencies

## 0.1.1

### Patch Changes

- d32d915: default to 0 gasPrice if none provided in the network config
- cc4b096: Ensure hardhat does not fail if no input sources provided
- daf975f: fix(hh-ovm): Working compilation for M1 macs

## 0.1.0

### Minor Changes

- 122df8c: allow overriding the ethers polling interval
- 9a7dd60: export ovm typechain bindings to types-ovm via hardhat-ovm

### Patch Changes

- 6daa408: update hardhat versions so that solc is resolved correctly

## 0.0.3

### Patch Changes

- c75a0fc: Use optimistic-solc to compile the SequencerEntrypoint. Also introduces a cache invalidation mechanism for hardhat-ovm so that we can push new compiler versions.

## 0.0.2

### Patch Changes

- 5362d38: adds build files which were not published before to npm
