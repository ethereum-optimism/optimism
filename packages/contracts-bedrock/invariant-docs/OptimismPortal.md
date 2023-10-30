# `OptimismPortal` Invariants

## Deposits of any value should always succeed unless `_to` = `address(0)` or `_isCreation` = `true`.
**Test:** [`OptimismPortal.t.sol#L148`](../test/invariants/OptimismPortal.t.sol#L148)

All deposits, barring creation transactions and transactions sent to `address(0)`, should always succeed. 

## `finalizeWithdrawalTransaction` should revert if the finalization period has not elapsed.
**Test:** [`OptimismPortal.t.sol#L171`](../test/invariants/OptimismPortal.t.sol#L171)

A withdrawal that has been proven should not be able to be finalized until after the finalization period has elapsed. 

## `finalizeWithdrawalTransaction` should revert if the finalization period has not elapsed.
**Test:** [`OptimismPortal.t.sol#L196`](../test/invariants/OptimismPortal.t.sol#L196)

A withdrawal that has been proven should not be able to be finalized until after the finalization period has elapsed. 

## `finalizeWithdrawalTransaction` should revert if the finalization period has not elapsed.
**Test:** [`OptimismPortal.t.sol#L224`](../test/invariants/OptimismPortal.t.sol#L224)

A withdrawal that has been proven should not be able to be finalized until after the finalization period has elapsed. 

## `finalizeWithdrawalTransaction` should revert if the withdrawal has already been finalized.
**Test:** [`OptimismPortal.t.sol#L254`](../test/invariants/OptimismPortal.t.sol#L254)

Ensures that there is no chain of calls that can be made that allows a withdrawal to be finalized twice. 

## A withdrawal should **always** be able to be finalized `FINALIZATION_PERIOD_SECONDS` after it was successfully proven.
**Test:** [`OptimismPortal.t.sol#L283`](../test/invariants/OptimismPortal.t.sol#L283)

This invariant asserts that there is no chain of calls that can be made that will prevent a withdrawal from being finalized exactly `FINALIZATION_PERIOD_SECONDS` after it was successfully proven. 