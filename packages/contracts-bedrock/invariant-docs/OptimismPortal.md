# `OptimismPortal` Invariants

## `finalizeWithdrawalTransaction` should revert if the finalization period has not elapsed.
**Test:** [`OptimismPortal.t.sol#L85`](../contracts/test/invariants/OptimismPortal.t.sol#L85)

A withdrawal that has been proven should not be able to be finalized until after the finalization period has elapsed. 


## `finalizeWithdrawalTransaction` should revert if the withdrawal has already been finalized.
**Test:** [`OptimismPortal.t.sol#L122`](../contracts/test/invariants/OptimismPortal.t.sol#L122)

Ensures that there is no chain of calls that can be made that allows a withdrawal to be finalized twice. 


## A withdrawal should **always** be able to be finalized `FINALIZATION_PERIOD_SECONDS` after it was successfully proven.
**Test:** [`OptimismPortal.t.sol#L157`](../contracts/test/invariants/OptimismPortal.t.sol#L157)

This invariant asserts that there is no chain of calls that can be made that will prevent a withdrawal from being finalized exactly `FINALIZATION_PERIOD_SECONDS` after it was successfully proven. 


## Deposits of any value should always succeed unless `_to` = `address(0)` or `_isCreation` = `true`.
**Test:** [`FuzzOptimismPortal.sol#L42`](../contracts/echidna/FuzzOptimismPortal.sol#L42)

All deposits, barring creation transactions and transactions sent to `address(0)`, should always succeed. 
