# `SafeCall` Invariants

## `callWithMinGas` forwards at least `minGas` if the call succeeds.
**Test:** [`SafeCall.t.sol#L30`](../contracts/test/invariants/SafeCall.t.sol#L30)

If the call to `SafeCall.callWithMinGas` succeeds, then the call must have received at *least* `minGas` gas. If there is not enough gas in the callframe to supply the minimum amount of gas to the call, it must revert. 


## `callWithMinGas` reverts if there is not enough gas to pass to the call.
**Test:** [`SafeCall.t.sol#L61`](../contracts/test/invariants/SafeCall.t.sol#L61)

If there is not enough gas in the callframe to ensure that `SafeCall.callWithMinGas` will receive at least `minGas` gas, then the call must revert. 
