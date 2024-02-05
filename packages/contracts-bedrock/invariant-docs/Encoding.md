# `Encoding` Invariants

## `convertRoundTripAToB` never fails.
**Test:** [`Encoding.t.sol#L73`](../test/invariants/Encoding.t.sol#L73)

Asserts that a raw versioned nonce can be encoded / decoded to reach the same raw value. 

## `convertRoundTripBToA` never fails.
**Test:** [`Encoding.t.sol#L82`](../test/invariants/Encoding.t.sol#L82)

Asserts that an encoded versioned nonce can always be decoded / re-encoded to reach the same encoded value. 