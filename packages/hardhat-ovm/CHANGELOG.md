# @eth-optimism/hardhat-ovm

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
