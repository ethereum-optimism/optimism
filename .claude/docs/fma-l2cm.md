# FMA: L2CM

|                     |            |
| ------------------- | ---------- |
| Author              | Ariel Diaz |
| Created at          | 2026-02-19 |
| Needs Approval From | maurelian  |
| Status              | Final      |

## Introduction

The L2 Contract Manager (L2CM) introduces a new mechanism for upgrading L2 predeploy contracts via hard fork, replacing the current Go-based NUT files with a deterministic JSON bundle generated from Solidity scripts. This FMA covers failure modes specific to the L2CM architecture.

**Key components:** ConditionalDeployer, L2ProxyAdmin (extended with `upgradePredeploys()`), L2ContractsManager (DELEGATECALLed by L2ProxyAdmin), and JSON NUT bundles.

**References:**

- [L2CM Design Doc](https://github.com/ethereum-optimism/design-docs/blob/main/protocol/l2-contract-upgrades.md)
- [L2 Upgrade Execution Spec](https://github.com/ethereum-optimism/specs/blob/7c29e249c62459ce4f6b034fa370f84e94fd9a28/specs/protocol/l2-upgrades-1-execution.md)
- [L2 Upgrade Contracts Spec](https://github.com/ethereum-optimism/specs/blob/7c29e249c62459ce4f6b034fa370f84e94fd9a28/specs/protocol/l2-upgrades-2-contracts.md)
- [OPCMv2 FMA](https://github.com/ethereum-optimism/design-docs/blob/main/security/fma-opcmv2.md)

## Failure Modes

<details>
<summary>FM1: Re-initialization State Corruption</summary>

#### Description

L2CM clears the `initialized` slot and re-calls `initialize()` on predeploy proxies during upgrades. If initialization fails or uses incorrect parameters, a predeploy could be left in an uninitialized or partially initialized state.

#### Risk Assessment

Critical severity / Low likelihood

#### Mitigations

- Upgrade-and-reinitialize is atomic within a single delegatecall from `upgradePredeploys()`
- `initializer` modifier prevents re-initialization once complete
- Config values gathered from on-chain state before upgrades via `gather_config()`
- Fork-based testing validates initialization path

#### Detection

- Fork tests verifying all predeploys are properly initialized post-upgrade

#### Recovery Path

- If a predeploy is left uninitialized: re-call `initialize()` if still callable
- If `initialize()` is open to anyone: critical severity, requires emergency response
- Most likely scenario is a full revert of the upgrade transaction (atomicity preserved)

#### Action Items

- [ ] Fork tests that verify initialization state of all predeploys post-upgrade
- [ ] Use [ProxyAdminOwnedBase.\_assertOnlyProxyAdminOrProxyAdminOwner()](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/src/L1/ProxyAdminOwnedBase.sol#L101) in all Predeploy contract initialize() functions

</details>

<details>
<summary>FM2: Storage Layout Corruption</summary>

#### Description

New predeploy implementations may introduce storage variables that conflict with existing slot positions, causing storage corruption during upgrade. Additionally, the `L2ContractsManager` executes via `DELEGATECALL` in the `L2ProxyAdmin`'s storage context, if the L2CM declares storage variables, they collide with `ProxyAdmin` slots (e.g., `_owner`), potentially locking out admin access permanently.

#### Risk Assessment

High+ severity / Low likelihood

#### Mitigations

- Careful review of storage layout changes during PRs and release process
- `L2ContractsManager` must only use `immutable`/`constant` variables to avoid colliding with `ProxyAdmin` storage
- Fork testing against real chain state
- Storage layout diff tooling (`forge inspect`)

#### Detection

- Fork testing will catch clearly broken storage layouts
- Challenging to detect if subtle

#### Recovery Path

- If storage layout issue causes broken-but-not-unsafe behavior: requires bug fix release ASAP
- If ProxyAdmin's `_owner` is corrupted: could permanently lock out upgrades, potentially unrecoverable without a hardfork to reset state
- If causes other unsafe behavior: potentially critical, may require pause

#### Action Items

- [ ] Establish convention that `L2ContractsManager` must not declare storage variables
- [ ] Investigate automated enforcement (Semgrep rule or Foundry test)

</details>

<details>
<summary>FM3: L2ProxyAdmin Backwards Compatibility Break</summary>

#### Description

`L2ProxyAdmin` replaces the existing `ProxyAdmin` predeploy. Other L2 contracts (`FeeVault`, `L1Withdrawer`, `FeeSplitter`) call `owner()` on it for access control. If existing `ProxyAdmin` functionality breaks, these contracts lose their admin mechanism.

#### Risk Assessment

High severity / Low likelihood

#### Mitigations

- `L2ProxyAdmin` inherits from `ProxyAdmin` without overriding existing functions
- Fork testing against real chains

#### Detection

- Fork tests calling existing `ProxyAdmin` functions post-upgrade

#### Recovery Path

- If backwards compatibility is broken: requires emergency upgrade to restore functionality
- If admin access control breaks for `FeeVault`/ `L1Withdrawer`: owner-gated functions become uncallable until fixed

#### Action Items

- [ ] Differential test verifying all existing `ProxyAdmin` interface functions are present and callable on `L2ProxyAdmin`

</details>

<details>
<summary>FM4: ConditionalDeployer Failure</summary>

#### Description

The ConditionalDeployer wraps `CREATE2` deployments, skipping deployment if bytecode already exists at the target address. Incorrect collision detection could result in implementations not being deployed, or unexpected reverts blocking the entire upgrade. This could lead to a chain halt in certain situations, such as if the L1 attributes transaction call to `L1Block` begins failing.

#### Risk Assessment

High severity / Low likelihood

#### Mitigations

- `ConditionalDeployer` is designed to never revert on properly formed inputs ([iCD-003](https://github.com/ethereum-optimism/specs/blob/7c29e249c62459ce4f6b034fa370f84e94fd9a28/specs/protocol/l2-upgrades-2-contracts.md#icd-003-non-reverting-collision-handling))
- `CREATE2` addresses are deterministic and pre-verifiable

#### Detection

- Fork testing catches deployment failures
- NUT generation script can pre-verify expected addresses

#### Recovery Path

- If deployment fails: the entire upgrade transaction reverts (atomicity preserved)
- If deterministic deployer is missing on a chain: must be deployed before the upgrade fork activates

#### Action Items

- [ ] Verify deterministic-deployment-proxy exists on all target L2 chains

</details>

<details>
<summary>FM5: Upgrade Block Gas Limit Exceeded</summary>

#### Description

The current `systemTxMaxGas` guarantees only 1M gas for system transactions. Upgrading all predeploys will exceed this. All node implementations must agree on the gas allocation for upgrade blocks, or a consensus failure occurs.

#### Risk Assessment

Medium severity / Medium likelihood

#### Mitigations

- Spec defines gas allocation value is read from the bundle JSON and applied by derivation pipeline
- Fork tests measure actual gas usage
- All client implementations must follow the same derivation rules ([aUBGL-003](https://github.com/ethereum-optimism/specs/blob/7c29e249c62459ce4f6b034fa370f84e94fd9a28/specs/protocol/l2-upgrades-1-execution.md#aubgl-003-custom-gas-does-not-affect-consensus))

#### Detection

- Fork testing with gas measurement
- CI tests that fail if upgrade gas exceeds threshold

#### Recovery Path

- If caught before mainnet: adjust gas limit and re-release
- If caught on mainnet: upgrade transactions fail, block is produced without upgrades, requires new fork activation with fixed gas limits

#### Action Items

- [ ] Implement and test upgrade gas mechanism in op-node
- [ ] Coordinate with Kona team to implement matching gas logic

</details>

<details>
<summary>FM6: Upgrade Atomicity Failure</summary>

#### Description

The `L2ContractsManager` upgrades multiple predeploys in a single `delegatecall`. If individual upgrades fail silently (e.g., via try/catch) rather than reverting, the system could be left with a mix of old and new contract versions.

#### Risk Assessment

High+ severity / Low likelihood

#### Mitigations

- L2CM must not use try/catch patterns that swallow failures
- All individual upgrade calls should revert on failure, propagating up

#### Detection

- Fork tests verifying all-or-nothing behavior

#### Recovery Path

- If partial upgrade occurs but system is functional: emergency upgrade to fix inconsistencies
- If partial upgrade causes critical failure: may require chain intervention

#### Action Items

- [ ] Review L2CM implementation for error swallowing patterns

</details>

<details>
<summary>FM7: Malicious or Incorrect NUT Bundle</summary>

#### Description

The NUT JSON bundle contains the exact transactions executed at fork activation. If tampered with or incorrectly generated, malicious or incorrect bytecode could be deployed across all OP chains simultaneously.

#### Risk Assessment

Critical severity / Low likelihood

#### Mitigations

- Bundle stored in monorepo, tracked by git, deterministically regenerable
- The spec requires CI to enforce that the committed bundle matches the source ([Upgrade Release Process](https://github.com/ethereum-optimism/specs/blob/7c29e249c62459ce4f6b034fa370f84e94fd9a28/specs/protocol/l2-upgrades-1-execution.md#upgrade-release-process))
- Code review of bundle generation script changes

#### Detection

- CI check that regenerates and verifies the bundle matches the committed version

#### Recovery Path

- If detected before fork activation: regenerate correct bundle and delay fork if necessary
- If malicious bundle is executed: critical incident, assess damage across all affected chains

#### Action Items

- [ ] CI job that regenerates and verifies NUT bundle on every PR

</details>

<details>
<summary>FM8: Multi-Client NUT Bundle Inconsistency</summary>

#### Description

The NUT bundle must be parsed and executed identically by all node implementations (op-node, kona). Differences in JSON parsing or transaction construction would cause L2 state divergence and a chain split.

#### Risk Assessment

Critical severity / Low likelihood

#### Mitigations

- JSON schema is simple with standard field types
- Shared test vectors across implementations
- Acceptance tests validate bundle execution across implementations

#### Detection

- Cross-client devnet/betanet testing before mainnet activation

#### Recovery Path

- If detected in testing: fix parsing bug in the affected client before activation
- If detected on mainnet: chain split, requires coordinated response across client teams

#### Action Items

- [ ] Cross-client devnet testing before any mainnet upgrade

</details>

<details>
<summary>FM9: Poorly Understood Differences in Prior L2 State Across Chains</summary>

#### Description

The L2CM assumes that certain functionality is available on L2 prior to the execution of the upgrade. If it is not, the upgrade will revert. The risk is increased by the fact that chains were all deployed at different times, and we do not have a good sense of the different contract versions they were each deployed with.

#### Risk Assessment

High severity / Medium likelihood

#### Mitigations

- Fork testing against multiple real OP chains with different deployment histories
- The spec mitigates via [aUP-002](https://github.com/ethereum-optimism/specs/blob/7c29e249c62459ce4f6b034fa370f84e94fd9a28/specs/protocol/l2-upgrades-1-execution.md#aup-002-testing-environments-match-production): testing against chains with different configurations (custom gas token, alt-DA, etc.)

#### Detection

- Fork testing against a diverse set of OP chains
- Post-upgrade checks comparing config values before and after

#### Recovery Path

- If upgrade reverts on a specific chain: exclude that chain from the fork until investigated
- If incorrect config is applied: requires follow-up upgrade to fix

#### Action Items

- [ ] Fork test all OP Chains prior to upgrade. These tests can be executed locally.
- [ ] Add validation/sanity checks in `gather_config()`

</details>

<details>
<summary>FM10: New Predeploys Not Correctly Inserted Into Deploy/Upgrade Path</summary>

#### Description

A new predeploy is written under `packages/contracts-bedrock/src/L2/` but is not properly incorporated into the `L2ContractsManager`. This failure mode is a consequence of the separation between the deploy path (`L2Genesis`) and upgrade path (L2CM) which persists until Milestone 2.

#### Risk Assessment

Medium severity / Medium likelihood

#### Mitigations

- Code review that checks both `L2Genesis` and L2CM when a new predeploy is added
- CI check comparing the set of predeploys in `L2Genesis` vs L2CM and flagging mismatches
- Milestone 2 will structurally fix this by unifying deploy and upgrade paths

#### Detection

- CI diff between predeploy lists in `L2Genesis` and L2CM
- Review checklist item for PRs adding new predeploys

#### Recovery Path

- Missing predeploy is added to L2CM in the next upgrade bundle
- No permanent damage, but delays rollout to existing chains

#### Action Items

- [ ] CI check that compares predeploy lists between `L2Genesis` and L2CM

</details>

## Risks & Uncertainties

- We do not have a good understanding of what code is running on the various L2s (see FM9).
- Other projects occurring in parallel may affect predeploys (e.g., `FeeVault` upgrades).
- Exact set of contracts to upgrade in Milestone 1 is TBD; edge-case contracts may be excluded.
- Standard Genesis generation is reserved for Milestone 2 and not covered by this FMA.
- No post-upgrade state validation mechanism (equivalent to L1's `OPContractsManagerStandardValidator`) has been defined for L2CM. Fork tests provide partial coverage, but a dedicated validator could strengthen mitigations across multiple failure modes.

## Audit Requirements

**This component requires audit.**
