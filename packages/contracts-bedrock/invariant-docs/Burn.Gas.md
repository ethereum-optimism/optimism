# `Burn.Gas` Invariants

## `gas(uint256)` always burns at least the amount of gas passed.
**Test:** [`Burn.Gas.t.sol#L66`](../test/invariants/Burn.Gas.t.sol#L66)

Asserts that when `Burn.gas(uint256)` is called, it always burns at least the amount of gas passed to the function. 