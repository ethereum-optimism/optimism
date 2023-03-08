# Network Upgrades

Network upgrades, also known as forks or hardforks, implement consensus-breaking changes.
These changes are transitioned into deterministically across all nodes through an activation rule.

This document lists the network upgrades of the OP Stack, starting after the Bedrock upgrade.
Prospective upgrades may be listed as proposals, but are not governed through these specifications.

Activation rule parameters of network upgrades are configured in respective chain configurations,
and not part of this specification.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Activation rules](#activation-rules)
  - [L2 Block-number based activation](#l2-block-number-based-activation)
  - [L2 Block-timestamp based activation](#l2-block-timestamp-based-activation)
- [Post-Bedrock Network upgrades](#post-bedrock-network-upgrades)
  - [Regolith](#regolith)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Activation rules

The below L2-block based activation rules may be applied in two contexts:

- The rollup node, specified through the rollup configuration (known as `rollup.json`),
  referencing L2 blocks (or block input-attributes) that pass through the derivation pipeline.
- The execution engine, specified through the chain configuration (known as the `config` part of `genesis.json`),
  referencing blocks or input-attributes that are part of, or applied to, the L2 chain.

### L2 Block-number based activation

Activation rule: `x != null && x >= upgradeNumber`

Starting at, and including, the L2 `block` with `block.number == x`, the upgrade rules apply.
If the upgrade block-number `x` is not specified in the configuration, the upgrade is ignored.

This applies to the L2 block number, not to the L1-origin block number.
This means that an L2 upgrade may be inactive, and then active, without changing the L1-origin.

This block number based method has commonly been used in L1 up until the Bellatrix/Paris upgrade, a.k.a. The Merge,
which was upgraded through special rules.

### L2 Block-timestamp based activation

Activation rule: `x != null && x >= upgradeTime`

Starting at, and including, the L2 `block` with `block.timestamp == x`, the upgrade rules apply.
If the upgrade block-timestamp `x` is not specified in the configuration, the upgrade is ignored.

This applies to the L2 block timestamp, not to the L1-origin block timestamp.
This means that an L2 upgrade may be inactive, and then active, without changing the L1-origin.

This timestamp based method has become the default on L1 after the Bellatrix/Paris upgrade, a.k.a. The Merge,
because it can be planned in accordance with beacon-chain epochs and slots.

Note that the L2 version is not limited to timestamps that match L1 beacon-chain slots or epochs.
A timestamp may be chosen to be synchronous with a specific slot or epoch on L1,
but the matching L1-origin information may not be present at the time of activation on L2.

## Post-Bedrock Network upgrades

### Regolith

The Regolith upgrade, named after a material best described as "deposited dust on top of a layer of bedrock",
implements minor changes to deposit processing, based on reports of the Sherlock Audit-contest and findings in
the Bedrock Optimism Goerli testnet.

Summary of changes:

- The `isSystemTx` boolean is disabled, system transactions now use the same gas accounting rules as regular deposits.
- The actual deposit gas-usage is recorded in the receipt of the deposit transaction,
  and subtracted from the L2 block gas-pool.
  Unused gas of deposits is not refunded with ETH however, as it is burned on L1.
- The `nonce` value of the deposit sender account, before the transaction state-transition, is recorded in a new
  optional field (`depositNonce`), extending the transaction receipt (i.e. not present in pre-Regolith receipts).
- The recorded deposit `nonce` is used to correct the transaction and receipt metadata in RPC responses,
  including the `contractAddress` field of deposits that deploy contracts.
- The `gas` and `depositNonce` data is committed to as part of the consensus-representation of the receipt,
  enabling the data to be safely synced between independent L2 nodes.

The [deposit specification](./deposits.md) specifies the changes of the Regolith upgrade in more detail.

The Regolith upgrade uses a *L2 block-timestamp* activation-rule, and is specified in both the
rollup-node (`regolith_time`) and execution engine (`config.regolithTime`).
