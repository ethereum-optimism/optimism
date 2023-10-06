# Superchain Config

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Superchain Config Parameters](#superchain-config-parameters)
- [Roles](#roles)
  - [System Owner](#system-owner)
  - [Initiator and vetoer](#initiator-and-vetoer)
  - [Guardian](#guardian)
- [Sequencer allowlist](#sequencer-allowlist)
  - [Updating the allowlist](#updating-the-allowlist)
- [Pausability status and parameters](#pausability-status-and-parameters)
  - [Effects of a pause](#effects-of-a-pause)
  - [Management of the paused status](#management-of-the-paused-status)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

The `SuperchainConfig` is a contract that manages the configuration of the Superchain system.

## Superchain Config Parameters

The parameters defined in the superchain config contract can be divided into three categories:

1. Roles
1. Batcher allowlist
1. Pausability status and parameters.

## Roles

The superchain config contract defines the following roles:

### System Owner

The `systemOwner` account has the ability to to remove entries from the batcher allow list. The
Optimism Superchain system will be configured such that the system owner is a
[`DelayedVetoable`](./delayed-vetoable.md) contract.

The system owner and all other roles in the Superchain Config contract are only modifiable by an
upgrade, and therefore cannot be modified without being subject to a delay and potential veto.

### Initiator and vetoer

Both the `initiator` and `vetoer` roles are read by the `DelayedVetoable` contract for
authorization.

### Guardian

The `guardian` entity can pause the system in case of an emergency.

## Sequencer allowlist

The Sequencer allowlist is a list of authorized sequencers managed by the system owner.

A Sequencer is represented by a [key pair](./system_config.md#sequencer-key-pair) in this allowlist,
and only allowed sequencers can be set as the sequencer in a system config contract.

### Updating the allowlist

Adding to the allowlist can be done instantly by the initiator. Removing from the list is done by the
system owner, and so is subject to delay.

## Pausability status and parameters

The entire Superchain system is designed to be easily pausable in an emergency situation. The
superchain config contract enables this by exposing the following pause related getter functions:

1. `pausedUntil()`: Returns the timestamp until which the system is paused.
1. `paused()`: A boolean value indicating whether the system is paused.
1. `maxPause()`: The maximum duration for which the system can be paused.

### Effects of a pause

When `paused()` returns true, all functions which enable the withdrawal of assets from any OP Chain
MUST revert. This is implemented by having the following functions read the paused status of the
superchain config contract:

1. `OptimismPortal.proveWithdrawalTransaction()`
2. `OptimismPortal.finalizeWithdrawalTransaction()`
3. `L1CrossDomainMessenger.relayMessage()`

Disabling the `L1CrossDomainMessenger`'s `relayMessage()` function, by virtue of the presence of the
[`onlyOtherBridge`](https://github.com/ethereum-optimism/optimism/blob/5e7be62478b48524963a2f23b93956ecd1651249/packages/contracts-bedrock/src/universal/StandardBridge.sol#L115)
modifier, also has the effect of disabling all withdrawal related function on the `L1StandardBridge`
and `L1ERC721Bridge`.

### Management of the paused status

The paused status of the system is always temporary. When first activated, it will last for for the
specified `duration`, which can be at most up to the `maxPause` value.

If the pause needs to be extended to allow time to address the cause of the emergency, the
`guardian` may call pause again. In the event that the system is already paused, then the pause will
be extended by adding the `duration` to the current time that the pause is set to end.

Although the `pause()` function is only callable by the `guardian`, the intention is to pre-sign
transactions which call `pause()`. These pre-signed transactions will be securely distributed to a
small number of parties so that they can quickly pause the system if they learn of an emergency. An
additional `identifier` string argument can be supplied allow for easy identification of which
pre-signed transaction was used.
