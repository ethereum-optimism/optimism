# `Burn.Gas` Invariants

## `gas(uint256)` always burns at least the amount of gas passed.
**Test:** [`Burn.Gas.t.sol#L64`](../test/invariants/Burn.Gas.t.sol#L64)

Asserts that when `Burn.gas(uint256)` is called, it always burns at least the amount of gas passed to the function. 