# `Encoding` Invariants

## `convertRoundTripAToB` never fails.
**Test:** [`Encoding.t.sol#L76`](../contracts/test/invariants/Encoding.t.sol#L76)

Asserts that a raw versioned nonce can be encoded / decoded to reach the same raw value. 


## `convertRoundTripBToA` never fails.
**Test:** [`Encoding.t.sol#L87`](../contracts/test/invariants/Encoding.t.sol#L87)

Asserts that an encoded versioned nonce can always be decoded / re-encoded to reach the same encoded value. 
