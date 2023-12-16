# Superchain Upgrades

Superchain upgrades, also known as forks or hardforks, implement consensus-breaking changes.

A Superchain upgrade requires the node software to support up to a given Protocol Version.
The version indicates support, the upgrade indicates the activation of new functionality.

This document lists the protocol versions of the OP-Stack, starting at the Bedrock upgrade,
as well as the default Superchain Targets.

Activation rule parameters of network upgrades are configured as part of the Superchain Target specification:
chains following the same Superchain Target upgrade synchronously.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Protocol Version](#protocol-version)
  - [Protocol Version Format](#protocol-version-format)
    - [Build identifier](#build-identifier)
    - [Major versions](#major-versions)
    - [Minor versions](#minor-versions)
    - [Patch versions](#patch-versions)
    - [Pre-releases](#pre-releases)
  - [Protocol Version Exposure](#protocol-version-exposure)
- [Superchain Target](#superchain-target)
  - [Superchain Version signaling](#superchain-version-signaling)
  - [`ProtocolVersions` L1 contract](#protocolversions-l1-contract)
- [Activation rules](#activation-rules)
  - [L2 Block-number based activation (deprecated)](#l2-block-number-based-activation-deprecated)
  - [L2 Block-timestamp based activation](#l2-block-timestamp-based-activation)
- [OP-Stack Protocol versions](#op-stack-protocol-versions)
- [Post-Bedrock Network upgrades](#post-bedrock-network-upgrades)
  - [Regolith](#regolith)
- [Canyon](#canyon)
- [Delta](#delta)
- [Eclipse](#eclipse)
- [Fjord](#fjord)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Protocol Version

The Protocol Version documents the progression of the total set of canonical OP-Stack specifications.
Components of the OP-Stack implement the subset of their respective protocol component domain,
up to a given Protocol Version of the OP-Stack.

OP-Stack mods, i.e. non-canonical extensions to the OP-Stack, are not included in the versioning of the Protocol.
Instead, mods must specify which upstream Protocol Version they are based on and where breaking changes are made.
This ensures tooling of the OP-Stack can be shared and collaborated on with OP-Stack mods.

The Protocol Version is NOT a hardfork identifier, but rather indicates software-support for a well-defined set
of features introduced in past and future hardforks, not the activation of said hardforks.

Changes that can be included in prospective Protocol Versions may be included in the specifications as proposals,
with explicit notice of the Protocol Version they are based on.
This enables an iterative integration process into the canonical set of specifications,
but does not guarantee the proposed specifications become canonical.

Note that the Protocol Version only applies to the Protocol specifications with the Superchain Targets specified within.
This versioning is independent of the [Semver] versioning used in OP Stack smart-contracts,
and the [Semver]-versioned reference software of the OP-Stack.

### Protocol Version Format

The Protocol Version is [Semver]-compatible.
It is encoded as a single 32 bytes long `<protocol version>`.
The version must be encoded as 32 bytes of `DATA` in JSON RPC usage.

The encoding is typed, to ensure future-compatibility.

```text
<protocol version> ::= <version-type><typed-payload>
<version-type> ::= <uint8>
<typed-payload> ::= <31 bytes>
```

version-type `0`:

```text
<reserved><build><major><minor><patch><pre-release>
<reserved> ::= <7 zeroed bytes>
<build> ::= <8 bytes>
<major> ::= <big-endian uint32>
<minor> ::= <big-endian uint32>
<patch> ::= <big-endian uint32>
<pre-release> ::= <big-endian uint32>
```

The `<reserved>` bytes of the Protocol Version are reserved for future extensions.

Protocol versions with a different `<version-type>` should not be compared directly.

[Semver]: https://semver.org/

#### Build identifier

The `<build>` identifier, as defined by [Semver], is ignored when determining version precedence.
The `<build>` must be non-zero to apply to the protocol version.

Modifications of the OP-Stack should define a `<build>` to distinguish from the canonical protocol feature-set.
Changes to the `<build>` may be encoded in the `<build>` itself to stay aligned with the upstream protocol.
The major/minor/patch versions should align with that of the upstream protocol that the modifications are based on.
Users of the protocol can choose to implement custom support for the alternative `<build>`,
but may work out of the box if the major features are consistent with that of the upstream protocol version.

The 8 byte `<build>` identifier may be presented as a string for human readability if the contents are alpha-numeric,
including `-` and `.`, as outlined in the [Semver] format specs. Trailing `0` bytes can be used for padding.
It may be presented as `0x`-prefixed hex string otherwise.

#### Major versions

Major version changes indicate support for new consensus-breaking functionality.
Major versions should retain support for functionality of previous major versions for
syncing/indexing of historical chain data.
Implementations may drop support for previous Major versions, when there are viable alternatives,
e.g. `l2geth` for pre-Bedrock data.

#### Minor versions

Minor version changes indicate support for backward compatible extensions,
including backward-compatible additions to the set of chains in a Superchain Target.
Backward-compatibility is defined by the requirement for existing end-users to upgrade nodes and tools or not.
Minor version changes may also include optional offchain functionality, such as additional syncing protocols.

#### Patch versions

Patch version changes indicate backward compatible bug fixes and improvements.

#### Pre-releases

Pre-releases of the protocol are proposals: these are not stable targets for production usage.
A pre-release might not satisfy the intended compatibility requirements as denoted by its associated normal version.
The `<pre-release>` must be non-zero to apply to the protocol version.
The `<pre-release>` `0`-value is reserved for non-prereleases, i.e. `v3.1.0` is higher than `v3.1.0-1`.

Node-software may support a pre-release, but must not activate any protocol changes without the user explicitly
opting in through the means of a feature-flag or configuration change.

A pre-release is not an official version and is meant for protocol developers to communicate an experimental changeset
before the changeset is reviewed by governance. Pre-releases are subject to change.

### Protocol Version Exposure

The Protocol Version is not exposed to the application-layer environment:
hardforks already expose the change of functionality upon activation as required,
and the Protocol Version is meant for offchain usage only.
The protocol version indicates support rather than activation of functionality.
There is one exception however: signaling by onchain components to offchain components.
More about this in [Superchain Version signaling].

## Superchain Target

Changes to the L2 state-transition function are transitioned into deterministically across all nodes
through an **activation rule**.

Changes to L1 smart-contracts must be compatible with the latest activated L2 functionality,
and are executed through **L1 contract-upgrades**.

A Superchain Target defines a set of activation rules and L1 contract upgrades shared between OP-Stack chains,
to upgrade the chains collectively.

### Superchain Version signaling

Each Superchain Target tracks the protocol changes, and signals the `recommended` and `required`
Protocol Version ahead of activation of new breaking functionality.

- `recommended`: a signal in advance of a network upgrade, to alert users of the protocol change to be prepared for.
  Node software is recommended to signal the recommendation to users through logging and metrics.
- `required`: a signal shortly in advance of a breaking network upgrade, to alert users of breaking changes.
  Users may opt in to elevated alerts or preventive measures, to ensure consistency with the upgrade.

Signaling is done through a L1 smart-contract that is monitored by the OP-Stack software.
Not all components of the OP-Stack are required to directly monitor L1 however:
cross-component APIs like the Engine API may be used to forward the Protocol Version signals,
to keep components encapsulated from L1.
See [`engine_signalOPStackVersionV1`](./exec-engine.md#enginesignalopstackversionv1).

### `ProtocolVersions` L1 contract

The `ProtocolVersions` contract on L1 enables L2 nodes to pick up on superchain protocol version signals.

The interface is:

- Required storage slot: `bytes32(uint256(keccak256("protocolversion.required")) - 1)`
- Recommended storage slot: `bytes32(uint256(keccak256("protocolversion.recommended")) - 1)`
- Required getter: `required()` returns `ProtocolVersion`
- Recommended getter `recommended()` returns `ProtocolVersion`
- Version updates also emit a typed event:
  `event ConfigUpdate(uint256 indexed version, UpdateType indexed updateType, bytes data)`

## Activation rules

The below L2-block based activation rules may be applied in two contexts:

- The rollup node, specified through the rollup configuration (known as `rollup.json`),
  referencing L2 blocks (or block input-attributes) that pass through the derivation pipeline.
- The execution engine, specified through the chain configuration (known as the `config` part of `genesis.json`),
  referencing blocks or input-attributes that are part of, or applied to, the L2 chain.

For both types of configurations, some activation parameters may apply to all chains within the superchain,
and are then retrieved from the superchain target configuration.

### L2 Block-number based activation (deprecated)

Activation rule: `upgradeNumber != null && block.number >= upgradeNumber`

Starting at, and including, the L2 `block` with `block.number >= upgradeNumber`, the upgrade rules apply.
If the upgrade block-number `upgradeNumber` is not specified in the configuration, the upgrade is ignored.

This block number based method has commonly been used in L1 up until the Bellatrix/Paris upgrade, a.k.a. The Merge,
which was upgraded through special rules.

This method is not superchain-compatible, as the activation-parameter is chain-specific
(different chains may have different block-heights at the same moment in time).

This applies to the L2 block number, not to the L1-origin block number.
This means that an L2 upgrade may be inactive, and then active, without changing the L1-origin.

### L2 Block-timestamp based activation

Activation rule: `upgradeTime != null && block.timestamp >= upgradeTime`

Starting at, and including, the L2 `block` with `block.timestamp >= upgradeTime`, the upgrade rules apply.
If the upgrade block-timestamp `upgradeTime` is not specified in the configuration, the upgrade is ignored.

This is the preferred superchain upgrade activation-parameter type:
it is synchronous between all L2 chains and compatible with post-Merge timestamp-based chain upgrades in L1.

This applies to the L2 block timestamp, not to the L1-origin block timestamp.
This means that an L2 upgrade may be inactive, and then active, without changing the L1-origin.

This timestamp based method has become the default on L1 after the Bellatrix/Paris upgrade, a.k.a. The Merge,
because it can be planned in accordance with beacon-chain epochs and slots.

Note that the L2 version is not limited to timestamps that match L1 beacon-chain slots or epochs.
A timestamp may be chosen to be synchronous with a specific slot or epoch on L1,
but the matching L1-origin information may not be present at the time of activation on L2.

## OP-Stack Protocol versions

- `v1.0.0`: 2021 Jan 16th - Mainnet Soft Launch, based on OVM.
  ([announcement](https://medium.com/ethereum-optimism/mainnet-soft-launch-7cacc0143cd5))
- `v1.1.0`: 2021 Aug 19th - Community launch.
  ([announcement](https://medium.com/ethereum-optimism/community-launch-7c9a2a9d3e84))
- `v2.0.0`: 2021 Nov 12th - the EVM-Equivalence update, also known as OVM 2.0 and chain regenesis.
  ([announcement](https://twitter.com/optimismfnd/status/1458953238867165192))
- `v2.1.0`: 2022 May 31st - Optimism Collective.
  ([announcement](https://optimism.mirror.xyz/gQWKlrDqHzdKPsB1iUnI-cVN3v0NvsWnazK7ajlt1fI)).
- `v3.0.0-1`: 2023 Jan 13th - Bedrock pre-release, deployed on OP-Goerli, and later Base-Goerli.
- `v3.0.0`: 2023 Jun 6th - Bedrock, including the Regolith hardfork improvements, first deployed on OP-Mainnet.
- `v4.0.0`: TBD - Canyon.
  [Governance proposal](https://gov.optimism.io/t/final-upgrade-proposal-2-canyon-network-upgrade/7088).
- `v5.0.0-1`: Delta - Experimental, devnet pre-release stage.

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
- The L1-cost function was corrected to more closely match pre-Bedrock behavior.

The [deposit specification](./deposits.md) specifies the deposit changes of the Regolith upgrade in more detail.
The [execution engine specification](./exec-engine.md) specifies the L1 cost function difference.

The Regolith upgrade uses a *L2 block-timestamp* activation-rule, and is specified in both the
rollup-node (`regolith_time`) and execution engine (`config.regolithTime`).

## Canyon

The Canyon upgrade contains the Shapella upgrade from L1 and some minor protocol fixes.

- Shapella Upgrade
  - [EIP-3651: Warm COINBASE](https://eips.ethereum.org/EIPS/eip-3651)
  - [EIP-3855: PUSH0 instruction](https://eips.ethereum.org/EIPS/eip-3855)
  - [EIP-3860: Limit and meter initcode](https://eips.ethereum.org/EIPS/eip-3860)
  - [EIP-4895: Beacon chain push withdrawals as operations](https://eips.ethereum.org/EIPS/eip-4895)
    - [Withdrawals are prohibited in P2P Blocks](./rollup-node-p2p.md#block-validation)
    - [Withdrawals should be set to the empty array with Canyon](./derivation.md#building-individual-payload-attributes)
  - [EIP-6049: Deprecate SELFDESTRUCT](https://eips.ethereum.org/EIPS/eip-6049)
- [Modifies the EIP-1559 Denominator](./exec-engine.md#1559-parameters)
- [Channel Ordering Fix](./derivation.md#reading)
- [Adds the deposit nonce & deposit nonce version to the deposit receipt hash](./deposits.md#deposit-receipt)
- [Deploys the create2Deployer to `0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2`](./predeploys.md#create2deployer)

The Canyon upgrade uses a *L2 block-timestamp* activation-rule, and is specified in both the
rollup-node (`canyon_time`) and execution engine (`config.canyonTime`). Shanghai time in the
execution engine should be set to the same time as the Canyon time.

## Delta

The Delta upgrade consists of a single consensus-layer feature: [Span Batches](./span-batches.md).

The Delta upgrade uses a *L2 block-timestamp* activation-rule, and is specified only in the rollup-node (`delta_time`).

## Eclipse

Name of the next upgrade after Delta. Placeholder for development coordination.

## Fjord

Name of the next upgrade after Eclipse. Placeholder for development coordination.
