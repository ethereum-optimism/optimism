# `AddressAliasing` Invariants

## Address aliases are always able to be undone.
**Test:** [`FuzzAddressAliasing.sol#L32`](../contracts/echidna/FuzzAddressAliasing.sol#L32)

Asserts that an address that has been aliased with `applyL1ToL2Alias` can always be unaliased with `undoL1ToL2Alias`. 
