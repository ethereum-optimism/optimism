# `SafeCall` Invariants

## If `callWithMinGas` performs a call, then it must always provide at least the specified minimum gas limit to the subcontext.
**Test:** [`SafeCall.t.sol#L31`](../test/invariants/SafeCall.t.sol#L31)

If the check for remaining gas in `SafeCall.callWithMinGas` passes, the subcontext of the call below it must be provided at least `minGas` gas. 

## `callWithMinGas` reverts if there is not enough gas to pass to the subcontext.
**Test:** [`SafeCall.t.sol#L63`](../test/invariants/SafeCall.t.sol#L63)

If there is not enough gas in the callframe to ensure that `callWithMinGas` can provide the specified minimum gas limit to the subcontext of the call, then `callWithMinGas` must revert. 