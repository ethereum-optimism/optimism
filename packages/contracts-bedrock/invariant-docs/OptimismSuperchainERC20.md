# `OptimismSuperchainERC20` Invariants

## Calls to sendERC20 should always succeed as long as the actor has enough balance. Actor's balance should also not increase out of nowhere but instead should decrease by the amount sent.
**Test:** [`OptimismSuperchainERC20.t.sol#L194`](../test/invariants/OptimismSuperchainERC20.t.sol#L194)



## Calls to relayERC20 should always succeeds when a message is received from another chain. Actor's balance should only increase by the amount relayed.
**Test:** [`OptimismSuperchainERC20.t.sol#L212`](../test/invariants/OptimismSuperchainERC20.t.sol#L212)

