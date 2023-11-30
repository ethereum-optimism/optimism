# Superchain Configuration

The SuperchainConfig contract is used to manage global configuration values for multiple OP Chains within
a single Superchain network.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Configurable values](#configurable-values)
- [Configuration data flow](#configuration-data-flow)
  - [Pausability](#pausability)
    - [Paused identifiers](#paused-identifiers)
    - [Scope of pausability](#scope-of-pausability)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Configurable values

Configurability of the Superchain is currently limited to two values:

The `SuperchainConfig` contract manages the following configuration values:

- `PAUSED_SLOT`: A boolean value indicating whether the Superchain is paused.
- `GUARDIAN_SLOT`: The address of the guardian, which can pause and unpause the system.

## Configuration data flow

All contracts which read from the `SuperchainConfig` contract hold its address as storage values
in the proxy account, and call directly to it when reading configuration data.

```mermaid
flowchart TD
StandardBridge --> SuperchainConfig
L1ERC721Bridge --> SuperchainConfig
L1CrossDomainMessenger --> SuperchainConfig
OptimismPortal --> SuperchainConfig
```

### Pausability

The Superchain pause feature is a safety mechanism designed to temporarily halt withdrawals from the system in
an emergency situation. The Guardian role is authorized to pause and unpause the system.

#### Paused identifiers

The Guardian may distributed a set of presigned transactions to trusted partners, so that the `pause()`
method may be called as quickly as possible in the event of an emergency. Although this increases the risk of
the system being paused unnecessarily, this is preferable to not pausing when assets are legitimately
vulnerable.

When the system is paused the `Paused(string identifier)` event is emitted, which enables easy attribution
of which partner triggered the pause.

#### Scope of pausability

The pause applies specifically to withdrawals of assets from the L1 bridge contracts. The L2 bridge contracts
are not pausable, on the basis that issues on L2 can be addressed more easily by a hard fork in the consensus
layer.

When the Pause is activated, the following methods are disabled:

1. `OptimismPortal.proveWithdrawalTransaction()`
2. `OptimismPortal.finalizeWithdrawalTransaction()`
3. `StandardBridge.finalizeBridgeERC20()`
4. `StandardBridge.finalizeBridgeETH()`
5. `L1ERC721Bridge.finalizeBridgeERC721()`
6. `L1CrossDomainMessenger.relayMessage()`
