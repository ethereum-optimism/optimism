# `Encoding` Invariants

## `convertRoundTripAToB` never fails.
**Test:** [`Encoding.t.sol#L71`](../test/invariants/Encoding.t.sol#L71)

Asserts that a raw versioned nonce can be encoded / decoded to reach the same raw value. 

## `convertRoundTripBToA` never fails.
**Test:** [`Encoding.t.sol#L80`](../test/invariants/Encoding.t.sol#L80)

Asserts that an encoded versioned nonce can always be decoded / re-encoded to reach the same encoded value. 