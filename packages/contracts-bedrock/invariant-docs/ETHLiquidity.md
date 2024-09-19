# `ETHLiquidity` Invariants

## Calls to mint/burn repeatedly should never cause the actor's balance to increase beyond the starting balance.
**Test:** [`ETHLiquidity.t.sol#L86`](../test/invariants/ETHLiquidity.t.sol#L86)

