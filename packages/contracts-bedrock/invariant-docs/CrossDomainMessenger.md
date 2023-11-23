# `CrossDomainMessenger` Invariants

## A call to `relayMessage` should succeed if at least the minimum gas limit can be supplied to the target context, there is enough gas to complete execution of `relayMessage` after the target context's execution is finished, and the target context did not revert.
**Test:** [`CrossDomainMessenger.t.sol#L137`](../test/invariants/CrossDomainMessenger.t.sol#L137)

There are two minimum gas limits here: 
- The outer min gas limit is for the call from the `OptimismPortal` to the `L1CrossDomainMessenger`,  and it can be retrieved by calling the xdm's `baseGas` function with the `message` and inner limit. 
- The inner min gas limit is for the call from the `L1CrossDomainMessenger` to the target contract. 

## A call to `relayMessage` should assign the message hash to the `failedMessages` mapping if not enough gas is supplied to forward `minGasLimit` to the target context or if there is not enough gas to complete execution of `relayMessage` after the target context's execution is finished.
**Test:** [`CrossDomainMessenger.t.sol#L170`](../test/invariants/CrossDomainMessenger.t.sol#L170)

There are two minimum gas limits here: 
- The outer min gas limit is for the call from the `OptimismPortal` to the `L1CrossDomainMessenger`,  and it can be retrieved by calling the xdm's `baseGas` function with the `message` and inner limit. 
- The inner min gas limit is for the call from the `L1CrossDomainMessenger` to the target contract. 