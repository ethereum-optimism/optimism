# `Hashing` Invariants

## `hashCrossDomainMessage` reverts if `version` is > `1`.
**Test:** [`Hashing.t.sol#L115`](../contracts/test/invariants/Hashing.t.sol#L115)

The `hashCrossDomainMessage` function should always revert if the `version` passed is > `1`. 


## `version` = `0`: `hashCrossDomainMessage` and `hashCrossDomainMessageV0` are equivalent.
**Test:** [`Hashing.t.sol#L127`](../contracts/test/invariants/Hashing.t.sol#L127)

If the version passed is 0, `hashCrossDomainMessage` and `hashCrossDomainMessageV0` should be equivalent. 


## `version` = `1`: `hashCrossDomainMessage` and `hashCrossDomainMessageV1` are equivalent.
**Test:** [`Hashing.t.sol#L140`](../contracts/test/invariants/Hashing.t.sol#L140)

If the version passed is 1, `hashCrossDomainMessage` and `hashCrossDomainMessageV1` should be equivalent. 


## `hashCrossDomainMessage` reverts if `version` is > `1`.
**Test:** [`FuzzHashing.sol#L94`](../contracts/echidna/FuzzHashing.sol#L94)

The `hashCrossDomainMessage` function should always revert if the `version` passed is > `1`. 


## `version` = `0`: `hashCrossDomainMessage` and `hashCrossDomainMessageV0` are equivalent.
**Test:** [`FuzzHashing.sol#L106`](../contracts/echidna/FuzzHashing.sol#L106)

If the version passed is 0, `hashCrossDomainMessage` and `hashCrossDomainMessageV0` should be equivalent. 


## `version` = `1`: `hashCrossDomainMessage` and `hashCrossDomainMessageV1` are equivalent.
**Test:** [`FuzzHashing.sol#L119`](../contracts/echidna/FuzzHashing.sol#L119)

If the version passed is 1, `hashCrossDomainMessage` and `hashCrossDomainMessageV1` should be equivalent. 
