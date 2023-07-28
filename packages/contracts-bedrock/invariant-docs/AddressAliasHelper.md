# `AddressAliasHelper` Invariants

## Address aliases are always able to be undone.
**Test:** [`AddressAliasHelper.t.sol#L46`](../test/invariants/AddressAliasHelper.t.sol#L46)

Asserts that an address that has been aliased with `applyL1ToL2Alias` can always be unaliased with `undoL1ToL2Alias`. 