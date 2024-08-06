# `SuperchainWETH` Invariants

## Calls to sendERC20 should always succeed as long as the actor has less than uint248 wei which is much greater than the total ETH supply. Actor's balance should also not increase out of nowhere.
**Test:** [`SuperchainWETH.t.sol#L181`](../test/invariants/SuperchainWETH.t.sol#L181)

