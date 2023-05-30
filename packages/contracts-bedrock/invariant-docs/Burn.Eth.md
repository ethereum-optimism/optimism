# `Burn.Eth` Invariants

## `eth(uint256)` always burns the exact amount of eth passed.
**Test:** [`Burn.Eth.t.sol#L68`](../contracts/test/invariants/Burn.Eth.t.sol#L68)

Asserts that when `Burn.eth(uint256)` is called, it always burns the exact amount of ETH passed to the function. 
