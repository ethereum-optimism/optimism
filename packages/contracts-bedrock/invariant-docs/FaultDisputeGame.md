# `FaultDisputeGame` Invariants

## FaultDisputeGame always returns all ETH on total resolution
**Test:** [`FaultDisputeGame.t.sol#L38`](../test/invariants/FaultDisputeGame.t.sol#L38)

The FaultDisputeGame contract should always return all ETH in the contract to the correct recipients upon resolution of all outstanding claims. There may never be any ETH left in the contract after a full resolution. 