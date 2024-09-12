# `OptimismPortal2` Invariants

## Deposits of any value should always succeed unless `_to` = `address(0)` or `_isCreation` = `true`.
**Test:** [`OptimismPortal2.t.sol#L163`](../test/invariants/OptimismPortal2.t.sol#L163)

All deposits, barring creation transactions and transactions sent to `address(0)`, should always succeed. 

## `finalizeWithdrawalTransaction` should revert if the proof maturity period has not elapsed.
**Test:** [`OptimismPortal2.t.sol#L185`](../test/invariants/OptimismPortal2.t.sol#L185)

A withdrawal that has been proven should not be able to be finalized until after the proof maturity period has elapsed. 

## `finalizeWithdrawalTransaction` should revert if the withdrawal has already been finalized.
**Test:** [`OptimismPortal2.t.sol#L214`](../test/invariants/OptimismPortal2.t.sol#L214)

Ensures that there is no chain of calls that can be made that allows a withdrawal to be finalized twice. 

## A withdrawal should **always** be able to be finalized `PROOF_MATURITY_DELAY_SECONDS` after it was successfully proven, if the game has resolved and passed the air-gap.
**Test:** [`OptimismPortal2.t.sol#L242`](../test/invariants/OptimismPortal2.t.sol#L242)

This invariant asserts that there is no chain of calls that can be made that will prevent a withdrawal from being finalized exactly `PROOF_MATURITY_DELAY_SECONDS` after it was successfully proven and the game has resolved and passed the air-gap. 