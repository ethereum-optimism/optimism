# `Hashing` Invariants

## `hashCrossDomainMessage` reverts if `version` is > `1`.
**Test:** [`Hashing.t.sol#L137`](../test/invariants/Hashing.t.sol#L137)

The `hashCrossDomainMessage` function should always revert if the `version` passed is > `1`. 

## `version` = `0`: `hashCrossDomainMessage` and `hashCrossDomainMessageV0` are equivalent.
**Test:** [`Hashing.t.sol#L147`](../test/invariants/Hashing.t.sol#L147)

If the version passed is 0, `hashCrossDomainMessage` and `hashCrossDomainMessageV0` should be equivalent. 

## `version` = `1`: `hashCrossDomainMessage` and `hashCrossDomainMessageV1` are equivalent.
**Test:** [`Hashing.t.sol#L158`](../test/invariants/Hashing.t.sol#L158)

If the version passed is 1, `hashCrossDomainMessage` and `hashCrossDomainMessageV1` should be equivalent. 