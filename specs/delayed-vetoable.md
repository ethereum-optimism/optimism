# The Delayed Vetoable Contract

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Purpose of the Delayed Vetoable contract](#purpose-of-the-delayed-vetoable-contract)
  - [Behaviour of the Delayed Vetoable contract](#behaviour-of-the-delayed-vetoable-contract)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Purpose of the Delayed Vetoable contract

The `DelayedVetoable` contract will be deployed in two distinct ways in the Superchain:

1. Once per OP Chain as the owner of the `ProxyAdmin` for that OP Chain.
2. Once as the account authorized to perform upgrades on the
   [`SuperchainConfig`](./superchain_config.md) contract itself.

It has two authorized roles:

1. The `initiator` who submits a call to be forwarded after a two week delay.
2. The `vetoer` who may veto a pending call.

### Behaviour of the Delayed Vetoable contract

The implementation of `DelayedVetoable` enforces the following properties:

1. At the outset there is a 'setup period' to facilitate deploying and configuring a new system.
   During this period all calls are forwarded instantly.
1. Either the  `INITIATOR` or `VETOER` can trigger the delay by submitting a call with null `data`.
   Once set, the delay cannot be disabled and applies to all calls moving forward.
1. The contract is ‘transparent’ similar to a transparent proxy, meaning it initiates, vetoes or
   forwards all calls based on the address of the caller.
    - The one exception to this property is `data` for which the first 4 bytes collide with the
      `_queuedAt(byte32)` selector.
1. The `INITIATOR` can initiate a call by sending data to the contract. The call is recorded in the
   `_queuedAt` mapping with the current timestamp. The timestamp is used to enforce the delay before
   a call can be forwarded to the `TARGET`.
1. The `VETOER` can veto a call at any time by sending the same data as the initiated call. This
   deletes the call from the `_queuedAt` mapping.
1. Calls can only be forwarded to the `TARGET` if it has been initiated by the `INITIATOR` and the
   delay period has passed.
1. After a call is forwarded it is deleted from the `_queuedAt` mapping to prevent replays.
1. If a call is forwarded to the `TARGET`, the contract ‘bubbles up’ the return or revert and
   associated data.

The implementation also has some limitations. The contract does not:

1. support value transfers, only data is forwarded,
1. enforce ordering of transactions (ie. there is no nonce), however the time between transactions
   being queued and becoming forwardable makes the ordering likely to hold.
1. allow for multiple calls with identical data to be queued up. If the `INITIATOR` resubmits, it
   will be treated as an attempt to forward the call (which will be allowed subject to the delay).
