# `CrossDomainMessenger` Invariants

## A call to `relayMessage` should never revert if at least the proper minimum gas limits are supplied.
**Test:** [`CrossDomainMessenger.t.sol#L126`](../contracts/test/invariants/CrossDomainMessenger.t.sol#L126)

There are two minimum gas limits here: 
- The outer min gas limit is for the call from the `OptimismPortal` to the `L1CrossDomainMessenger`,  and it can be retrieved by calling the xdm's `baseGas` function with the `message` and inner limit. 
- The inner min gas limit is for the call from the `L1CrossDomainMessenger` to the target contract. 
