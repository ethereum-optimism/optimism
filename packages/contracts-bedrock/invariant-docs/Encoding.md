# `Encoding` Invariants

## `testRoundTripAToB` never fails.
**Test:** [`L56`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/echidna/FuzzEncoding.sol#L56)

Asserts that a raw versioned nonce can be encoded / decoded to reach the same raw value. 


## `testRoundTripBToA` never fails.
**Test:** [`L67`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/echidna/FuzzEncoding.sol#L67)

Asserts that an encoded versioned nonce can always be decoded / re-encoded to reach the same encoded value. 
