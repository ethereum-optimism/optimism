# `OptimismPortal` Invariants

## `finalizeWithdrawalTransaction` should revert if the finalization period has not elapsed.
**Test:** [`L86`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/test/invariants/OptimismPortal.t.sol#L86)

A withdrawal that has been proven should not be able to be finalized until after the finalization period has elapsed. 


## `finalizeWithdrawalTransaction` should revert if the withdrawal has already been finalized.
**Test:** [`L123`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/test/invariants/OptimismPortal.t.sol#L123)

Ensures that there is no chain of calls that can be made that allows a withdrawal to be finalized twice. 


## Deposits of any value should always succeed unless `_to` = `address(0)` or `_isCreation` = `true`.
**Test:** [`L37`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/echidna/FuzzOptimismPortal.sol#L37)

All deposits, barring creation transactions and transactions sent to `address(0)`, should always succeed. 
