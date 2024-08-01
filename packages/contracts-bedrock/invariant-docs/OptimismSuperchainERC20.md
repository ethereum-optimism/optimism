# `OptimismSuperchainERC20` Invariants

## Calls to sendERC20 should always succeed as long as the actor has enough balance. Actor's balance should also not increase out of nowhere.
**Test:** [`OptimismSuperchainERC20.t.sol#L177`](../test/invariants/OptimismSuperchainERC20.t.sol#L177)

