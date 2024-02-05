# `Burn.Eth` Invariants

## `eth(uint256)` always burns the exact amount of eth passed.
**Test:** [`Burn.Eth.t.sol#L66`](../test/invariants/Burn.Eth.t.sol#L66)

Asserts that when `Burn.eth(uint256)` is called, it always burns the exact amount of ETH passed to the function. 