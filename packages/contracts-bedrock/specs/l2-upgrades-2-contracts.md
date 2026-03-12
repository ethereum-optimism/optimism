# L2 Upgrade Contracts

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
- [ConditionalDeployer](#conditionaldeployer)
  - [Overview](#overview-1)
  - [Definitions](#definitions)
    - [CREATE2 Collision](#create2-collision)
    - [Deterministic Deployment Proxy](#deterministic-deployment-proxy)
  - [Assumptions](#assumptions)
    - [aCD-001: Deterministic Deployment Proxy is Available and Correct](#acd-001-deterministic-deployment-proxy-is-available-and-correct)
      - [Mitigations](#mitigations)
    - [aCD-002: Initcode is Well-Formed](#acd-002-initcode-is-well-formed)
      - [Mitigations](#mitigations-1)
  - [Invariants](#invariants)
    - [iCD-001: Deterministic Address Derivation](#icd-001-deterministic-address-derivation)
      - [Impact](#impact)
    - [iCD-002: Idempotent Deployment Operations](#icd-002-idempotent-deployment-operations)
      - [Impact](#impact-1)
    - [iCD-003: Non-Reverting Collision Handling](#icd-003-non-reverting-collision-handling)
      - [Impact](#impact-2)
    - [iCD-004: Collision Detection Accuracy](#icd-004-collision-detection-accuracy)
      - [Impact](#impact-3)
    - [iCD-005: Contract Availability After Deployment](#icd-005-contract-availability-after-deployment)
      - [Impact](#impact-4)
- [L2ProxyAdmin](#l2proxyadmin)
  - [Overview](#overview-2)
  - [Definitions](#definitions-1)
    - [Depositor Account](#depositor-account)
    - [Predeploy](#predeploy)
  - [Assumptions](#assumptions-1)
    - [aL2PA-001: Depositor Account is Controlled by Protocol](#al2pa-001-depositor-account-is-controlled-by-protocol)
      - [Mitigations](#mitigations-2)
    - [aL2PA-002: L2ContractsManager Code is Not Malicious](#al2pa-002-l2contractsmanager-code-is-not-malicious)
      - [Mitigations](#mitigations-3)
    - [aL2PA-003: Predeploy Proxies Follow Expected Patterns](#al2pa-003-predeploy-proxies-follow-expected-patterns)
      - [Mitigations](#mitigations-4)
  - [Invariants](#invariants-1)
    - [iL2PA-001: Exclusive Depositor Authorization for Batch Upgrades](#il2pa-001-exclusive-depositor-authorization-for-batch-upgrades)
      - [Impact](#impact-5)
    - [iL2PA-002: Safe Delegation to L2ContractsManager](#il2pa-002-safe-delegation-to-l2contractsmanager)
      - [Impact](#impact-6)
    - [iL2PA-003: Backwards Compatibility Maintained](#il2pa-003-backwards-compatibility-maintained)
      - [Impact](#impact-7)
- [L2ContractsManager](#l2contractsmanager)
  - [Overview](#overview-3)
  - [Definitions](#definitions-2)
    - [Network-Specific Configuration](#network-specific-configuration)
    - [Feature Flag](#feature-flag)
    - [Initialization Parameters](#initialization-parameters)
  - [Assumptions](#assumptions-2)
    - [aL2CM-001: Existing Predeploys Provide Valid Configuration](#al2cm-001-existing-predeploys-provide-valid-configuration)
      - [Mitigations](#mitigations-5)
    - [aL2CM-002: Implementation Addresses Are Pre-Computed Correctly](#al2cm-002-implementation-addresses-are-pre-computed-correctly)
      - [Mitigations](#mitigations-6)
    - [aL2CM-003: Predeploy Proxies Are Upgradeable](#al2cm-003-predeploy-proxies-are-upgradeable)
      - [Mitigations](#mitigations-7)
    - [aL2CM-004: Feature Flags Are Correctly Configured](#al2cm-004-feature-flags-are-correctly-configured)
      - [Mitigations](#mitigations-8)
  - [Invariants](#invariants-2)
    - [iL2CM-001: Deterministic Upgrade Execution](#il2cm-001-deterministic-upgrade-execution)
      - [Impact](#impact-8)
    - [iL2CM-002: Configuration Preservation](#il2cm-002-configuration-preservation)
      - [Impact](#impact-9)
    - [iL2CM-003: Upgrade Atomicity](#il2cm-003-upgrade-atomicity)
      - [Impact](#impact-10)
    - [iL2CM-004: Clear-and-Reinitialize Pattern](#il2cm-004-clear-and-reinitialize-pattern)
      - [Impact](#impact-11)
    - [iL2CM-005: No Storage Corruption During DELEGATECALL](#il2cm-005-no-storage-corruption-during-delegatecall)
      - [Impact](#impact-12)
    - [iL2CM-006: Complete Upgrade Coverage](#il2cm-006-complete-upgrade-coverage)
      - [Impact](#impact-13)
- [Upgrade Execution](#upgrade-execution)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Overview

This specification defines the mechanism for upgrading L2 predeploy contracts through deterministic, hard fork-driven
Network Upgrade Transactions (NUTs). The system enables safe, well-tested upgrades of L2 contracts with both
implementation and upgrade paths written in Solidity, ensuring determinism, verifiability, and testability across all
client implementations.

The upgrade system maintains the existing pattern of injecting Network Upgrade Transactions at specific fork block
heights while improving the development and testing process. Upgrade transactions are defined in JSON bundles (see
[Bundle Format](./l2-upgrades-1-execution.md#bundle-format)) that are tracked in git, generated from Solidity scripts,
and executed deterministically at fork activation.

## ConditionalDeployer

### Overview

The ConditionalDeployer contract enables deterministic deployment of contract implementations while maintaining
idempotency across upgrade transactions. It ensures that unchanged contract bytecode always deploys to the same
address, and that attempting to deploy already-deployed bytecode succeeds silently _rather than reverting_.

This component enables upgrade transactions to unconditionally deploy for all implementation contracts without
requiring developers to manually track which contracts have changed between upgrades.

The ConditionalDeployer is included in the L2Genesis state to ensure availability for all future network upgrades. It is
deployed as a preinstall at a deterministic address and does not require upgradeability.

The deployment function returns an address for off-chain convenience, but this return value is not used in Network
Upgrade Transactions, as deployment addresses must be pre-computed before transaction generation.

### Definitions

#### CREATE2 Collision

A CREATE2 collision occurs when attempting to deploy contract bytecode to an address where a contract with identical
bytecode already exists. This happens when the same initcode and salt are used in multiple deployment attempts.

Note: when CREATE2 targets an address that already has code, the
[zero address is placed on the stack][create2-spec] (execution specs).

[create2-spec]:
https://github.com/ethereum/execution-specs/blob/4ef381a0f75c96b52da635653ab580e731d3882a/src/ethereum/forks/prague/vm/instructions/system.py#L112

#### Deterministic Deployment Proxy

The canonical deterministic deployment proxy contract at address `0x4e59b44847b379578588920cA78FbF26c0B4956C`,
originally deployed by Nick Johnson (Arachnid). This contract provides CREATE2-based deployment with a fixed deployer
address across all chains.

Note: when the deterministic deployment proxy deploys to an address that already has code, [it will revert with no data](https://github.com/Arachnid/deterministic-deployment-proxy/blob/be3c5974db5028d502537209329ff2e730ed336c/source/deterministic-deployment-proxy.yul#L13).
Otherwise the ConditionalDeployer would not be required.

### Assumptions

#### aCD-001: Deterministic Deployment Proxy is Available and Correct

The [Deterministic Deployment Proxy](#deterministic-deployment-proxy) exists at the expected address and correctly
implements CREATE2 deployment semantics. The proxy must deterministically compute deployment addresses and execute
deployments as specified.

##### Mitigations

- The [Deterministic Deployment Proxy](#deterministic-deployment-proxy) is a well-established contract deployed across
  all EVM chains using the same keyless deployment transaction
- The proxy's behavior is verifiable by inspecting its bytecode and testing deployment operations
- The proxy contract is immutable and cannot be upgraded or modified

#### aCD-002: Initcode is Well-Formed

Callers provide valid EVM initcode that, when executed, will either successfully deploy a contract or revert with a
clear error. Malformed initcode that produces undefined behavior is not considered.

##### Mitigations

- Initcode is generated by the Solidity compiler from verified source code
- The upgrade transaction generation process includes validation of all deployment operations
- Fork-based testing exercises all deployments before inclusion in upgrade bundles

### Invariants

#### iCD-001: Deterministic Address Derivation

For any given initcode and salt combination, the ConditionalDeployer MUST always compute the same deployment address,
regardless of whether the contract has been previously deployed. The address calculation MUST match the CREATE2
address that would be computed by the [Deterministic Deployment Proxy](#deterministic-deployment-proxy).

##### Impact

**Severity: Critical**

If address derivation is non-deterministic or inconsistent with CREATE2 semantics, upgrade transactions could deploy
implementations to unexpected addresses, breaking proxy upgrade operations.

#### iCD-002: Idempotent Deployment Operations

Calling the ConditionalDeployer multiple times with identical initcode and salt MUST produce the same outcome: the
first call deploys the contract, and subsequent calls succeed without modification. No operation should revert due to
a [CREATE2 Collision](#create2-collision).

##### Impact

**Severity: Critical**

If deployments are not idempotent, upgrade transactions that attempt to deploy unchanged implementations would revert
or deploy the implementation to an unexpected address, breaking the upgrade.

#### iCD-003: Non-Reverting Collision Handling

When a [CREATE2 Collision](#create2-collision) is detected (contract already deployed at the target address), the
ConditionalDeployer MUST return successfully without reverting and without modifying blockchain state.

##### Impact

**Severity: Medium**

If collisions cause reverts, the presence of reverting transactions in an upgrade block would cause confusion.

#### iCD-004: Collision Detection Accuracy

The ConditionalDeployer MUST correctly distinguish between addresses where no contract exists (deploy needed) and
addresses where a contract already exists (collision detected). False negatives (failing to detect existing contracts)
and false positives (detecting non-existent contracts) are both prohibited.

##### Impact

**Severity: High**

False negatives would cause failed deployments while false positives would prevent legitimate deployments, both
breaking the upgrade process.

#### iCD-005: Contract Availability After Deployment

After execution of the ConditionalDeployer, the address returned by the deployment operation MUST contain the runtime
bytecode derived from the provided initcode. This ensures that contracts deployed through the ConditionalDeployer are
immediately available and functional at their expected addresses.

##### Impact

**Severity: Critical**

If the contract is not properly available at the expected address after deployment, subsequent transactions that
attempt to call or upgrade to that implementation address will fail, causing the upgrade to fail and potentially
halting the chain.

## L2ProxyAdmin

### Overview

The L2ProxyAdmin is the administrative contract responsible for managing proxy upgrades for L2 predeploy contracts. It
is deployed as a predeploy at address `0x4200000000000000000000000000000000000018` and serves as the `admin` for all
upgradeable L2 predeploy proxies.

The upgraded L2ProxyAdmin implementation extends the existing proxy administration interface with a new
`upgradePredeploys()` function that orchestrates batch upgrades of multiple predeploys by delegating to an
[L2ContractsManager](#l2contractsmanager) contract. This design enables deterministic, testable upgrade paths written
entirely in Solidity.

### Definitions

#### Depositor Account

The special system address `0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001` controlled by the L2 protocol's derivation
pipeline. This account is used to submit system transactions including L1 attributes updates and upgrade
transactions.

#### Predeploy

A contract deployed at a predetermined address in the L2 genesis state. Predeploys provide core L2 protocol
functionality and are typically deployed behind proxies to enable upgradability.

### Assumptions

#### aL2PA-001: Depositor Account is Controlled by Protocol

The [Depositor Account](#depositor-account) is exclusively controlled by the L2 protocol's derivation and execution
pipeline. No external parties can submit transactions from this address.

##### Mitigations

- The [Depositor Account](#depositor-account) address has no known private key
- Transactions from this address can only originate from the protocol's derivation pipeline processing L1 deposit events
- The address is hardcoded in the protocol specification and client implementations

#### aL2PA-002: L2ContractsManager Code is Not Malicious

The [L2ContractsManager](#l2contractsmanager) contract that receives the DELEGATECALL from `upgradePredeploys()`
correctly implements the upgrade logic and does not contain malicious code. This includes not corrupting the
L2ProxyAdmin's storage, not performing unauthorized operations, and not introducing vulnerabilities when executing
in the context of the L2ProxyAdmin.

##### Mitigations

- The L2ContractsManager address is deterministically computed and verified during upgrade bundle generation
- The L2ContractsManager implementation is developed, reviewed, and tested alongside the upgrade bundle
- Fork-based testing validates the complete upgrade execution before production deployment
- The L2ContractsManager bytecode is verifiable against source code on a specific commit
- Code review and security audits examine the L2ContractsManager implementation

#### aL2PA-003: Predeploy Proxies Follow Expected Patterns

[Predeploys](#predeploy) being upgraded follow the expected proxy patterns (ERC-1967 or similar) and correctly handle
`upgradeTo()` and `upgradeToAndCall()` operations when called by the ProxyAdmin.

##### Mitigations

- All L2 predeploys use standardized proxy implementations from the contracts-bedrock package
- Proxy implementations are thoroughly tested and audited
- Fork-based testing validates upgrade operations against actual deployed proxies

### Invariants

#### iL2PA-001: Exclusive Depositor Authorization for Batch Upgrades

The `upgradePredeploys()` function MUST only be callable by the [Depositor Account](#depositor-account). No other
address, including the current ProxyAdmin owner, can invoke this function.

##### Impact

**Severity: Critical**

If unauthorized addresses could call `upgradePredeploys()`, attackers could execute arbitrary upgrade, enabling
complete takeover of all L2 predeploy contracts.

#### iL2PA-002: Safe Delegation to L2ContractsManager

When `upgradePredeploys()` executes a DELEGATECALL to the [L2ContractsManager](#l2contractsmanager), the call MUST
preserve the ProxyAdmin's storage context.

##### Impact

**Severity: Critical**

If the DELEGATECALL is not properly executed, upgrades could fail silently or the ProxyAdmin's storage could be
corrupted, resulting in loss of admin control over predeploys.

#### iL2PA-003: Backwards Compatibility Maintained

The upgraded L2ProxyAdmin implementation MUST maintain the existing interface for standard proxy administration
functions. Existing functionality for upgrading individual proxies, changing proxy admins, and querying proxy state
MUST continue to work as before.

Note: Backwards compatibility requires maintaining the full ProxyAdmin interface, but does not require supporting
upgrades of legacy proxy types (ResolvedDelegate and Chugsplash proxies). Currently, no predeploy uses these legacy
proxy types.

##### Impact

**Severity: High**

If backwards compatibility is broken, existing tooling and scripts that interact with the ProxyAdmin could fail,
preventing emergency responses and breaking operational procedures.

## L2ContractsManager

### Overview

The L2ContractsManager is a contract deployed fresh for each upgrade that contains the upgrade logic and coordination
for a specific set of predeploy upgrades. When invoked via DELEGATECALL from the [L2ProxyAdmin](#l2proxyadmin), it
gathers network-specific configuration from existing predeploys, and executes
upgrade operations for all affected predeploys.

Each L2ContractsManager instance is purpose-built for a specific upgrade, deployed via the
[ConditionalDeployer](#conditionaldeployer), and referenced directly in the upgrade transaction. The contract is
stateless and contains all upgrade logic in code, ensuring determinism and verifiability.

The L2ContractsManager assumes that all prerequisite contracts (implementations, ConditionalDeployer, etc.) have
already been deployed and are available in the state before the L2ContractsManager is called. The transaction
execution sequence ensures this ordering.

### Definitions

#### Network-Specific Configuration

Configuration values that vary between L2 chains, such as custom gas token parameters, operator fee configurations, or
chain-specific feature flags. These values are typically stored in system predeploys like `L1Block` and must be
preserved across upgrades.

#### Feature Flag

A boolean or enumerated value that enables or disables optional protocol features. Feature flags allow different
upgrade paths for development environments (alphanets), testing environments, and production chains. Flags are read
from a dedicated FeatureFlags contract during upgrade execution.

#### Initialization Parameters

Constructor arguments or initializer function parameters required by a predeploy implementation. Similar to the OPCMv2
implementation, we will assume that all config values are first read, and then contracts are reinitialized with
those same parameters.

### Assumptions

#### aL2CM-001: Existing Predeploys Provide Valid Configuration

The existing [predeploy](#predeploy) contracts contain valid [network-specific configuration](#network-specific-configuration)
that can be read and used during the upgrade. Configuration values are accurate, properly formatted, and represent the
intended chain configuration.

##### Mitigations

- Configuration is read from well-established predeploys that have been operating correctly
- Fork-based testing validates configuration gathering against real chain state

#### aL2CM-002: Implementation Addresses Are Pre-Computed Correctly

The implementation addresses used by the L2ContractsManager are pre-computed by the off-chain bundle generation script
using the same CREATE2 parameters that will be used by the [ConditionalDeployer](#conditionaldeployer). The
L2ContractsManager receives these addresses via its constructor and does not compute them. Address mismatches would
cause proxies to point to incorrect or non-existent implementations.

##### Mitigations

- Implementation addresses are computed off-chain using deterministic CREATE2 formula during bundle generation
- The computed addresses are provided to the L2ContractsManager constructor at deployment time
- Fork-based testing validates that all implementation addresses exist and contain expected bytecode
- Address computation is isolated in shared libraries to prevent divergence

#### aL2CM-003: Predeploy Proxies Are Upgradeable

All [predeploy](#predeploy) proxies targeted for upgrade support the `upgradeTo()` and `upgradeToAndCall()` functions
and will accept upgrade calls from the [L2ProxyAdmin](#l2proxyadmin) executing the DELEGATECALL.

##### Mitigations

- All L2 predeploys use standardized proxy implementations with well-tested upgrade functions
- Fork-based testing exercises upgrade operations against actual deployed proxies
- Non-upgradeable predeploys (if they exist) will be excluded from the upgrade process

#### aL2CM-004: Feature Flags Are Correctly Configured

When [feature flags](#feature-flag) are used to customize upgrade behavior, the FeatureFlags contract is properly
configured in the environment and returns consistent values throughout the upgrade execution.
Production features which are enabled must be exposed by the L1Block contract's interface.

##### Mitigations

- Fork and local testing validates feature flag behavior across different configurations

### Invariants

#### iL2CM-001: Deterministic Upgrade Execution

The L2ContractsManager's `upgrade()` function MUST execute deterministically, producing identical state changes when
given identical pre-upgrade blockchain state. The function MUST NOT read external state that could vary between
executions (timestamps, block hashes, etc.) and MUST NOT accept runtime parameters.

##### Impact

**Severity: Critical**

If upgrade execution is non-deterministic, different L2 nodes could produce different post-upgrade states, causing
consensus failures and halting the chain.

#### iL2CM-002: Configuration Preservation

All [network-specific configuration](#network-specific-configuration) that exists before the upgrade MUST be preserved
in the upgraded predeploy implementations. Configuration values MUST be read from existing predeploys and properly
passed to new implementations during upgrade.

##### Impact

**Severity: Critical**

If configuration is not preserved, chains could lose critical settings like custom gas token addresses or operator fee
parameters, breaking fee calculations and chain-specific functionality.

#### iL2CM-003: Upgrade Atomicity

All predeploy upgrades within a single L2ContractsManager execution MUST succeed or fail atomically. If any upgrade
operation fails, the entire DELEGATECALL MUST revert, leaving all predeploys in their pre-upgrade state.

##### Impact

**Severity: Critical**

If upgrades are not atomic, a partial failure could leave some predeploys upgraded and others not, creating an
inconsistent system state that breaks inter-contract dependencies.

#### iL2CM-004: Clear-and-Reinitialize Pattern

For each predeploy being upgraded, the L2ContractsManager MUST:
1. use `upgradeTo()` to set the implementation to the StorageSetter
2. Reset the `initialized` value to 0
3. use `upgradeToAndCall()` to call the `initialize()` method.

This ensures storage is properly cleared and reconstructed, avoiding storage layout conflicts.

##### Impact

**Severity: Critical**

If contracts are not properly reinitialized with preserved configuration, chain-specific settings could be lost or
storage corruption could occur, breaking critical system contracts.

#### iL2CM-005: No Storage Corruption During DELEGATECALL

When executing in the [L2ProxyAdmin](#l2proxyadmin) context via DELEGATECALL, the L2ContractsManager MUST NOT corrupt
or modify the ProxyAdmin's own storage. All storage modifications must be directed to the predeploy proxies being
upgraded.

##### Impact

**Severity: Critical**

If the L2ContractsManager corrupts ProxyAdmin storage, it could change the ProxyAdmin's owner or disable future upgrade
capability, compromising the entire upgrade system.

#### iL2CM-006: Complete Upgrade Coverage

The L2ContractsManager MUST upgrade all predeploys intended for the upgrade. It MUST NOT skip predeploys that should
be upgraded, even if their implementations are unchanged, to maintain consistency across all chains executing the
upgrade.

##### Impact

**Severity: High**

If predeploys are skipped incorrectly, chains would have inconsistent contract versions, violating the goal of bringing
all chains to a consistent version.

## Upgrade Execution

The L2ContractsManager contract should be deployed with all implementation addresses provided to its constructor and
stored in an `Implementations` struct. Where a dev feature or feature flag requires a different implementation, both
implementations will be deployed and stored in the L2ContractsManager. The L2ContractsManager will contain branching
logic which will enable a different implementation depending on the configured features.

The upgrade flow of the L2ContractsManager will be similar to the OPCMv2, where the `FullConfig` is first collected from
all contracts, and the config values are provided to the contract's `initializer()` method using `upgradeToAndCall()`.
