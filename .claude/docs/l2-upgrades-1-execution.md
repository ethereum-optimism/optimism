# L2 Upgrade Execution

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of Contents**

- [Overview](#overview)
- [Upgrade Process](#upgrade-process)
  - [Overview](#overview-1)
  - [Definitions](#definitions)
    - [Fork Activation Timestamp](#fork-activation-timestamp)
  - [Assumptions](#assumptions)
    - [aUP-001: Fork Activation Time is Coordinated](#aup-001-fork-activation-time-is-coordinated)
      - [Mitigations](#mitigations)
    - [aUP-002: Testing Environments Match Production](#aup-002-testing-environments-match-production)
      - [Mitigations](#mitigations-1)
    - [aUP-003: Net-new Configuration Values will not be Set During Upgrades](#aup-003-net-new-configuration-values-will-not-be-set-during-upgrades)
      - [Mitigations](#mitigations-2)
    - [aUP-004: Transaction Payloads are Identical Across Chains](#aup-004-transaction-payloads-are-identical-across-chains)
      - [Mitigations](#mitigations-3)
  - [Invariants](#invariants)
    - [iUP-001: Atomic Upgrade Execution](#iup-001-atomic-upgrade-execution)
      - [Impact](#impact)
    - [iUP-002: Deterministic Execution](#iup-002-deterministic-execution)
      - [Impact](#impact-1)
    - [iUP-003: Network-specific configuration must be preserved](#iup-003-network-specific-configuration-must-be-preserved)
      - [Impact](#impact-2)
    - [iUP-004: Verifiable Upgrade Execution](#iup-004-verifiable-upgrade-execution)
      - [Impact](#impact-3)
  - [Upgrade Release Process](#upgrade-release-process)
  - [Transaction Execution Sequence](#transaction-execution-sequence)
- [Network Upgrade Transaction Bundle](#network-upgrade-transaction-bundle)
  - [Overview](#overview-2)
  - [Definitions](#definitions-1)
    - [Network Upgrade Transaction (NUT)](#network-upgrade-transaction-nut)
    - [Fork Activation Block](#fork-activation-block)
    - [Bundle Generation Script](#bundle-generation-script)
  - [Assumptions](#assumptions-1)
    - [aNUTB-001: Solidity Compiler is Deterministic](#anutb-001-solidity-compiler-is-deterministic)
      - [Mitigations](#mitigations-4)
    - [aNUTB-001b: Build Toolchain is Not Compromised](#anutb-001b-build-toolchain-is-not-compromised)
      - [Mitigations](#mitigations-5)
    - [aNUTB-002: Bundle Generation Script is Pure](#anutb-002-bundle-generation-script-is-pure)
      - [Mitigations](#mitigations-6)
    - [aNUTB-003: Git Repository is Authoritative Source](#anutb-003-git-repository-is-authoritative-source)
      - [Mitigations](#mitigations-7)
    - [aNUTB-004: JSON Format is Correctly Parsed](#anutb-004-json-format-is-correctly-parsed)
      - [Mitigations](#mitigations-8)
  - [Invariants](#invariants-1)
    - [iNUTB-001: Deterministic Bundle Generation](#inutb-001-deterministic-bundle-generation)
      - [Impact](#impact-4)
    - [iNUTB-002: Transaction Completeness](#inutb-002-transaction-completeness)
      - [Impact](#impact-5)
    - [iNUTB-003: Transaction Ordering](#inutb-003-transaction-ordering)
      - [Impact](#impact-6)
    - [iNUTB-004: Valid Transaction Format](#inutb-004-valid-transaction-format)
      - [Impact](#impact-7)
    - [iNUTB-005: Upgrade transactions do not revert](#inutb-005-upgrade-transactions-do-not-revert)
      - [Impact](#impact-8)
  - [Bundle Format](#bundle-format)
  - [Bundle Generation Process](#bundle-generation-process)
  - [Bundle Verification Process](#bundle-verification-process)
- [Custom Upgrade Block Gas Limit](#custom-upgrade-block-gas-limit)
  - [Overview](#overview-3)
  - [Definitions](#definitions-2)
    - [System Transaction Gas Limit](#system-transaction-gas-limit)
    - [Upgrade Block Gas Allocation](#upgrade-block-gas-allocation)
    - [Derivation Pipeline](#derivation-pipeline)
  - [Assumptions](#assumptions-2)
    - [aUBGL-001: Upgrade Gas Requirements Are Bounded](#aubgl-001-upgrade-gas-requirements-are-bounded)
      - [Mitigations](#mitigations-9)
    - [aUBGL-003: Custom Gas Does Not Affect Consensus](#aubgl-003-custom-gas-does-not-affect-consensus)
      - [Mitigations](#mitigations-10)
  - [Invariants](#invariants-2)
    - [iUBGL-001: Sufficient Gas Availability](#iubgl-001-sufficient-gas-availability)
      - [Impact](#impact-9)
    - [iUBGL-002: Deterministic Gas Allocation](#iubgl-002-deterministic-gas-allocation)
      - [Impact](#impact-10)
    - [iUBGL-003: Gas Limit Independence from Block Gas Limit](#iubgl-003-gas-limit-independence-from-block-gas-limit)
      - [Impact](#impact-11)
    - [iUBGL-004: Gas Allocation Only for Upgrade Blocks](#iubgl-004-gas-allocation-only-for-upgrade-blocks)
      - [Impact](#impact-12)
  - [Gas Allocation Specification](#gas-allocation-specification)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

This specification defines the execution mechanism for L2 contract upgrades, covering the bundle format, gas
allocation, and complete upgrade lifecycle. These components work together with the
[L2 Upgrade Contracts](./l2-upgrades-2-contracts.md) specification to enable deterministic, verifiable upgrades of L2
predeploy contracts across all OP Stack chains.

The upgrade execution system ensures that upgrade transactions are properly formatted, have sufficient gas to execute,
and follow a well-defined process from development through verification.

## Upgrade Process

### Overview

The Upgrade Process defines the complete lifecycle of an L2 predeploy upgrade, from initial development through fork
activation and execution. The process ensures that upgrades are developed safely, tested thoroughly, and executed
deterministically across all OP Stack chains.

This end-to-end process integrates all components of the upgrade system: contract development, bundle generation,
testing, verification, and execution at fork activation.

### Definitions

#### Fork Activation Timestamp

The L2 block timestamp at which a fork becomes active and upgrade transactions are executed. This timestamp is
specified in the protocol configuration and used by the [derivation pipeline](#derivation-pipeline) to identify when to
inject upgrade transactions.

### Assumptions

#### aUP-001: Fork Activation Time is Coordinated

All OP Stack chains coordinate fork activation times to enable consistent upgrade rollout. The fork activation
timestamp is communicated well in advance of activation to allow node operators to prepare.

##### Mitigations

- Fork activation timestamps are defined in the superchain-registry
- Activation times are set and communicated far enough in advance to allow preparation

#### aUP-002: Testing Environments Match Production

Fork-based testing environments accurately represent production chain state, allowing upgrade testing to catch issues
that would occur in production.

##### Mitigations

- Fork tests should use actual mainnet chain state (e.g., OP Mainnet) as a starting point
- Testing should validate against chains with different configurations (e.g., custom gas token, alt-DA)
- CI/CD should run fork tests to catch regressions
- Manual testing on testnet chains before mainnet activation

#### aUP-003: Net-new Configuration Values will not be Set During Upgrades

If a new configuration value is being added to a contract by the upgrade, then it should
initially be set to a default which applies to all chains.

##### Mitigations

- All previous hard fork activate contract upgrades have met this assumption.

#### aUP-004: Transaction Payloads are Identical Across Chains

The upgrade transaction payloads are identical for all chains executing the upgrade. While there may be
ephemeral differences in execution state (such as data read from existing contracts), the transaction
data itself is deterministic and chain-independent.

##### Mitigations

- Bundle generation uses only deterministic inputs (CREATE2 addresses, compiled bytecode)
- CI validates bundle determinism through repeated generation
- Fork tests validate execution across different chain configurations

### Invariants

#### iUP-001: Atomic Upgrade Execution

All transactions in the upgrade bundle MUST execute atomically within the
[fork activation block](#fork-activation-block), and the entire upgrade must execute successfully.

##### Impact

**Severity: Critical**

A failed upgrade can lead to a chain halt, for example if a new function is not added to the L1Block
contract, causing the L1 Attributes deposit transactions to fail.

#### iUP-002: Deterministic Execution

The upgrade transactions must be deterministically generated across all chains, regardless of chain-specific
configuration or chain state at the time of execution. The transaction payloads (calldata) must be identical
across all chains. While execution may result in different storage or memory state (e.g., contracts storing
`block.number` or reading existing chain-specific configuration), the transaction calldata itself must be identical.

##### Impact

**Severity: Critical**

Non-determinism in transaction payloads could result in a chain split.

#### iUP-003: Network-specific configuration must be preserved

All pre-existing storage values corresponding to chain-specific configuration (ie. the L1StandardBridge address),
must be preserved after the upgrade.

##### Impact

**Severity: Critical**

Unexpected changes to configuration could result in loss of funds or chain halts. In general the only
storage modifications should be pointers to implementation addresses.

#### iUP-004: Verifiable Upgrade Execution

After fork activation, it MUST be possible to verify that the executed upgrade transactions match the committed bundle
and that the resulting contract state matches expectations.

##### Impact

**Severity: High**

If upgrades cannot be verified post-execution, there is no way to audit whether the correct upgrade was performed,
breaking transparency and making troubleshooting difficult. If post-execution verification reveals that contracts do not
match expectations, this would violate other invariants and likely require another upgrade to address the issues.

### Upgrade Release Process

The upgrade process follows this flow:

1. **Implementation & Bundle Generation**: Contract changes are implemented and the
   [bundle generation script](#bundle-generation-script) produces a deterministic JSON bundle containing all upgrade
   transactions. This bundle will be checked into the monorepo at
   `packages/contracts-bedrock/snapshots/current-l2-upgrade-bundle.json` (or similar). CI will enforce that this
   bundle always corresponds with bundle generated from the source code in the current commit.

2. **Continuous Release Readiness**: Contracts should always be release-ready. New functionality should be behind
   feature flags that are only activated after a fork. This approach means there is no need to copy the bundle to a
   separate fork-named file during development iteration. The `current-l2-upgrade-bundle.json` is always the
   authoritative source that gets released with each contracts version.

3. **Client Integration**: The canonical bundle JSON is embedded into L2 client binaries at build time.

4. **Fork Activation**: At the [fork activation timestamp](#fork-activation-timestamp), nodes execute the bundle transactions.

### Transaction Execution Sequence

Within the fork activation block, transactions will execute in this order:

0. **L1 Info Deposit transaction**: This is the transaction which occurs at the start of each block, it is unchanged by
   this work.
1. **ConditionalDeployer Deployment** (one-time only): Deploy the ConditionalDeployer contract
2. **ConditionalDeployer Upgrade** (one-time only): Upgrade the ConditionalDeployer implementation
3. **Implementation Deployments**: For each predeploy being upgraded, deploy new implementation via
   ConditionalDeployer
4. **ProxyAdmin Upgrade** (one-time only): Upgrade the L2ProxyAdmin implementation
5. **L2ContractsManager Deployment**: Deploy the L2ContractsManager for this upgrade
6. **Upgrade Execution**: Call `L2ProxyAdmin.upgradePredeploys(l2ContractsManagerAddress)` which will atomically:
   - Executes DELEGATECALL to L2ContractsManager.upgrade()
   - L2ContractsManager gathers configuration from existing predeploys
   - For each predeploy, calls `proxy.upgradeTo()` or `proxy.upgradeToAndCall()`
   - Verifies all upgrades completed successfully

All of these transactions execute before any user-submitted transactions in the block.

**Note:** Transactions which are labeled as "one-time only" are necessary to enable infrastructure which will
support this mechanism of upgrades in general. Once these steps are complete they do not need to be included in
future upgrades, thus they should be kept out of the bundle itself and instead hardcoded in the client
implementation.

## Network Upgrade Transaction Bundle

### Overview

The Network Upgrade Transaction (NUT) Bundle is a JSON-formatted data structure containing the complete set of
transactions that must be executed at a specific fork activation block. The bundle is generated deterministically from
Solidity scripts, tracked in git, and executed by all L2 client implementations to upgrade predeploy contracts.

The bundle format enables verification that upgrade transactions correspond to specific source code commits, ensuring
transparency and auditability across all OP Stack chains executing the upgrade.

### Definitions

#### Network Upgrade Transaction (NUT)

A system transaction injected by the protocol at a specific fork block height, executed with the
[Depositor Account](./l2-upgrades-2-contracts.md#depositor-account) as the sender. These transactions bypass normal
transaction pool processing and are deterministically included in the fork activation block.

Examples of previous NUTs can be seen in the op-node (ie.
[ecotone_upgrade_transactions.go](https://github.com/ethereum-optimism/optimism/blob/develop/op-node/rollup/derive/ecotone_upgrade_transactions.go)).
This spec builds on that precedent with an improved method for generating and inserting NUTs.

#### Fork Activation Block

The L2 block at which a protocol upgrade becomes active, identified by a specific L2 block timestamp. This block
contains the Network Upgrade Transactions that implement the protocol changes.

#### Bundle Generation Script

A Solidity script (typically using Forge scripting) that deterministically computes all transaction data for an
upgrade. The script computes CREATE2 addresses, generates deployment initcode, and assembles transaction calldata
into a JSON file written to disk.

### Assumptions

#### aNUTB-001: Solidity Compiler is Deterministic

The Solidity compiler produces identical bytecode when given identical source code and compiler settings. This enables
verification that bundle contents match the source code on a specific commit.

##### Mitigations

- Use pinned compiler versions specified in foundry.toml
- Verification process rebuilds contracts with identical settings and compares bytecode

#### aNUTB-001b: Build Toolchain is Not Compromised

The Solidity compiler (solc), Forge, and other build tools used to compile contracts and generate bundles are not
compromised and produce trustworthy output. Compromised build tools could inject malicious code or alter bytecode.

##### Mitigations

- Use pinned, verified versions of build tools from official sources
- Build in isolated, reproducible environments
- Multiple independent parties verify bundle generation
- Compare bytecode against known-good reference builds

#### aNUTB-002: Bundle Generation Script is Pure

The [bundle generation script](#bundle-generation-script) does not depend on external state that could vary between
executions. All addresses are computed deterministically using CREATE2, and all transaction data is derived from
compiled bytecode.

##### Mitigations

- Bundle generation scripts are reviewed to ensure they contain no external dependencies
- Scripts use only deterministic address computation (CREATE2)
- CI validates bundle regeneration produces identical output

#### aNUTB-003: Git Repository is Authoritative Source

The git repository containing the bundle JSON files and source code serves as the authoritative source of truth for
upgrade transactions.

##### Mitigations

- Bundles are committed to git alongside the source code that generates them
- Repository is hosted on GitHub with branch protection and audit logs

#### aNUTB-004: JSON Format is Correctly Parsed

All L2 client implementations (Go, Rust, etc.) correctly parse the JSON bundle format and extract transaction fields
identically. Parsing inconsistencies would cause consensus failures.

##### Mitigations

- JSON schema is simple and uses standard field types
- Test vectors validate parsing across client implementations
- Acceptance tests should validate bundle execution across implementations

### Invariants

#### iNUTB-001: Deterministic Bundle Generation

Running the [bundle generation script](#bundle-generation-script) multiple times on the same source code commit MUST
produce byte-for-byte identical JSON output. No aspect of bundle generation may depend on timestamps, random values, or
external state.

##### Impact

**Severity: Critical**

If bundle generation is non-deterministic, it becomes impossible to verify that a given bundle corresponds to specific
source code, potentially allowing unverified or malicious transactions to be included.

#### iNUTB-002: Transaction Completeness

The bundle MUST contain all transactions required to complete the upgrade. Missing transactions would cause the upgrade
to fail partially, leaving the system in an inconsistent state.

##### Impact

**Severity: Critical**

If the bundle is incomplete, the fork activation would fail, potentially halting the chain or leaving predeploys in
partially upgraded states.

#### iNUTB-003: Transaction Ordering

Transactions in the bundle MUST be ordered such that dependencies are satisfied. For example, contract deployments MUST
occur before transactions that call those contracts.

##### Impact

**Severity: Critical**

If transactions are misordered, executions will fail when attempting to call non-existent contracts, causing the entire
upgrade to fail at fork activation potentially halting the chain.

#### iNUTB-004: Valid Transaction Format

All transactions in the bundle MUST conform to the expected transaction format for
[Network Upgrade Transactions](#network-upgrade-transaction-nut), including correct sender
([Depositor Account](./l2-upgrades-2-contracts.md#depositor-account)), appropriate gas limits, and valid calldata
encoding.

##### Impact

**Severity: Critical**

If transactions are malformed, they will fail to execute at fork activation, causing the upgrade to fail and
potentially halting the chain.

#### iNUTB-005: Upgrade transactions do not revert

The upgrade transactions must successfully execute without reverting.

##### Impact

**Severity: Critical**

Reverting would likely cause a chain halt.

### Bundle Format

The bundle is a JSON file with the following structure:

```json
{
  "metadata": {
    "version": "1.0.0"
  },
  "transactions": [
    {
      "to": "0x1234...",
      "data": "0xabcd...",
      "gasLimit": "1000000",
      "from": "0x0000000000000000000000000000000000000000"
    }
  ]
}
```

**Field Requirements:**

- `metadata.version`: Bundle format version for compatibility tracking
- `transactions`: Array of transaction objects in execution order
- `transactions[].to`: Target address (contract being called)
- `transactions[].data`: Transaction calldata as hex string
- `transactions[].gasLimit`: Gas limit for this transaction
- `transactions[].from`: (Optional) Sender address. Defaults to the [Depositor Account](./l2-upgrades-2-contracts.md#depositor-account).
  Must be set to `address(0)` for the L2ProxyAdmin upgrade transaction to utilize the zero-address upgrade path in the
  Proxy.sol implementation

A `value` field MUST NOT be included in transaction objects. All NUT transactions are calls with zero ETH value, which
is enforced by the execution layer rather than specified per-transaction.

### Bundle Generation Process

Bundle generation MUST follow this process:

1. **Compile Contracts**: Build all contracts with deterministic compiler settings
2. **Compute Addresses**: Calculate implementation addresses using CREATE2 with deterministic salts
3. **Generate Transaction Data**: Construct calldata for each transaction using computed addresses
4. **Assemble Bundle**: Create JSON structure with transactions in dependency order
5. **Write Bundle File**: Output JSON to the designated path in the repository

### Bundle Verification Process

To verify a bundle matches source code:

1. **Check Out Commit**: Check out the commit specified in the upgrade release documentation
2. **Build Contracts**: Compile contracts using the build process documented in the repository
3. **Regenerate Bundle**: Run the bundle generation script
4. **Compare Output**: Verify byte-for-byte match with committed bundle

## Custom Upgrade Block Gas Limit

### Overview

The Custom Upgrade Block Gas Limit mechanism provides guaranteed gas availability for executing upgrade transactions at
fork activation, independent of the regular block gas limit and system transaction gas constraints. This ensures that
complex multi-contract upgrades can execute completely within the [fork activation block](#fork-activation-block)
without running out of gas.

Standard L2 blocks are constrained by `systemTxMaxGas` (typically 1,000,000 gas), which is insufficient for executing
the deployment and upgrade transactions in a typical predeploy upgrade. The custom gas limit bypasses this constraint
for upgrade blocks specifically.

Note: In practice, past upgrades that consumed more than 1M gas were possible because remaining gas in the block
(after the ~20M user deposit gas allocation) was also available. However, this relied on the implicit assumption
that chains had sufficient gas available via their block gas limit. This specification makes the gas allocation
explicit to avoid such implicit dependencies.

### Definitions

#### System Transaction Gas Limit

The maximum gas available for system transactions (transactions from the
[Depositor Account](./l2-upgrades-2-contracts.md#depositor-account)) in a normal L2 block, defined by
`resourceConfig.systemTxMaxGas`. This limit is typically set to 1,000,000 gas.

#### Upgrade Block Gas Allocation

The total gas available for executing upgrade transactions in a [fork activation block](#fork-activation-block). This
value is set significantly higher than the [system transaction gas limit](#system-transaction-gas-limit) to accommodate
complex upgrade operations.

#### Derivation Pipeline

The component of L2 client implementations responsible for constructing L2 blocks from L1 data and protocol rules. The
derivation pipeline determines block attributes including gas limits and inserts upgrade transactions at fork
activations.

### Assumptions

#### aUBGL-001: Upgrade Gas Requirements Are Bounded

The gas required to execute all upgrade transactions in a [bundle](#network-upgrade-transaction-bundle) is finite and
can be estimated before deployment. Upgrades do not contain unbounded loops or operations that could consume arbitrary
amounts of gas.

##### Mitigations

- Fork-based testing measures actual gas consumption of upgrade transactions
- Bundle generation process includes gas estimation for all transactions
- Upgrade complexity is bounded by the number of predeploys and deployment operations
- Gas profiling is performed during development to identify expensive operations

#### aUBGL-003: Custom Gas Does Not Affect Consensus

Providing additional gas for upgrade blocks does not violate consensus rules or create divergence between clients. The
gas allocation is deterministic and applied consistently across all implementations.

##### Mitigations

- Gas allocation is part of the protocol specification
- All client implementations follow the same derivation rules
- Gas allocation logic is simple and deterministic

### Invariants

#### iUBGL-001: Sufficient Gas Availability

The [upgrade block gas allocation](#upgrade-block-gas-allocation) MUST be sufficient to execute all transactions in the
[Network Upgrade Transaction Bundle](#network-upgrade-transaction-bundle) without running out of gas. No upgrade
transaction should fail due to insufficient gas.

##### Impact

**Severity: Critical**

If insufficient gas is allocated, upgrade transactions will fail mid-execution, leaving predeploys in partially
upgraded states and halting the chain at fork activation.

#### iUBGL-002: Deterministic Gas Allocation

The gas allocation for upgrade blocks MUST be deterministic and identical across all L2 nodes executing the fork
activation. The allocation must depend only on consensus-critical inputs (fork identification) and not on
node-specific state or configuration.

##### Impact

**Severity: Critical**

If gas allocation is non-deterministic, different nodes could allocate different gas amounts, causing a consensus
failure and chain split.

#### iUBGL-003: Gas Limit Independence from Block Gas Limit

The [upgrade block gas allocation](#upgrade-block-gas-allocation) MUST be independent of the chain's configured block
gas limit and the [system transaction gas limit](#system-transaction-gas-limit). Upgrade transactions must execute even
if block gas limits are set to minimum values.

##### Impact

**Severity: High**

If upgrade gas depends on block gas limits, chains with different configurations could have inconsistent upgrade
execution, breaking the goal of deterministic upgrades across the Superchain.

#### iUBGL-004: Gas Allocation Only for Upgrade Blocks

The custom gas allocation MUST only apply to [fork activation blocks](#fork-activation-block) containing upgrade
transactions. Regular blocks must continue to use standard gas limits without modification.

##### Impact

**Severity: High**

If custom gas allocation applies to non-upgrade blocks, it could enable DOS attacks by allowing transactions to consume
excessive gas or bypass fee markets.

### Gas Allocation Specification

The custom upgrade block gas allocation is implemented in the derivation pipeline with the following behavior:

**Gas Allocation Value:**

- The upgrade block gas allocation is specified in the bundle JSON for each upgrade
- The value should be set significantly higher than the measured gas consumption to provide safety margin
- The specific value is read from the bundle and applied by the derivation pipeline for the corresponding fork

**Allocation Conditions:**

- Custom gas allocation MUST only apply when processing a fork activation block
- Fork activation is identified by L2 block timestamp matching or exceeding the fork activation timestamp
- Allocation applies to the entire upgrade transaction bundle, not per-transaction

**Implementation Requirements:**

- Implemented in `op-node/rollup/derive/attributes.go` or equivalent derivation logic
- Gas allocation is applied when constructing the payload attributes for the fork activation block
