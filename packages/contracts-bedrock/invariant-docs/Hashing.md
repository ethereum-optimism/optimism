# `Hashing` Invariants

## `hashCrossDomainMessage` reverts if `version` is > `1`.
**Test:** [`FuzzHashing.sol#L120`](../contracts/echidna/FuzzHashing.sol#L120)

The `hashCrossDomainMessage` function should always revert if the `version` passed is > `1`. 


## `version` = `0`: `hashCrossDomainMessage` and `hashCrossDomainMessageV0` are equivalent.
**Test:** [`FuzzHashing.sol#L132`](../contracts/echidna/FuzzHashing.sol#L132)

If the version passed is 0, `hashCrossDomainMessage` and `hashCrossDomainMessageV0` should be equivalent. 


## `version` = `1`: `hashCrossDomainMessage` and `hashCrossDomainMessageV1` are equivalent.
**Test:** [`FuzzHashing.sol#L145`](../contracts/echidna/FuzzHashing.sol#L145)

If the version passed is 1, `hashCrossDomainMessage` and `hashCrossDomainMessageV1` should be equivalent. 
