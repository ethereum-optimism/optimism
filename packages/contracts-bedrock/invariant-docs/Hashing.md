# `Hashing` Invariants

## `hashCrossDomainMessage` reverts if `version` is > `1`.
**Test:** [`L120`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/echidna/FuzzHashing.sol#L120)

The `hashCrossDomainMessage` function should always revert if the `version` passed is > `1`. 


## `version` = `0`: `hashCrossDomainMessage` and `hashCrossDomainMessageV0` are equivalent.
**Test:** [`L131`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/echidna/FuzzHashing.sol#L131)

If the version passed is 0, `hashCrossDomainMessage` and `hashCrossDomainMessageV0` should be equivalent. 


## `version` = `1`: `hashCrossDomainMessage` and `hashCrossDomainMessageV1` are equivalent.
**Test:** [`L143`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/echidna/FuzzHashing.sol#L143)

If the version passed is 1, `hashCrossDomainMessage` and `hashCrossDomainMessageV1` should be equivalent. 
