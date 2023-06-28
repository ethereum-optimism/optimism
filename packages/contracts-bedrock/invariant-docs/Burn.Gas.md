# `Burn.Gas` Invariants

## `gas(uint256)` always burns at least the amount of gas passed.
**Test:** [`Burn.Gas.t.sol#L68`](../contracts/test/invariants/Burn.Gas.t.sol#L68)

Asserts that when `Burn.gas(uint256)` is called, it always burns at least the amount of gas passed to the function. 
