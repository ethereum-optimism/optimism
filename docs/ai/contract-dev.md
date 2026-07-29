# Smart Contract Development

This document provides guidance for AI agents working with smart contracts in the OP Stack.

## Non-Idempotent Initializers

When reviewing `initialize()` or `reinitializer` functions, check whether the function is **idempotent** — calling it multiple times with the same arguments should produce the same end state as calling it once.

### The Risk

Proxied contracts in the OP Stack can be re-initialized during upgrades (via `reinitializer(version)`). Orchestrators like `OPContractsManagerV2._apply()` call `initialize()` on contracts that may already hold state from a previous initialization. If the initializer is not idempotent, re-initialization can corrupt state.

**Example**: `ETHLockbox.initialize()` calls `_authorizePortal()` for each portal passed in. Currently safe because `_authorizePortal()` is idempotent — setting `authorizedPortals[portal] = true` twice has the same effect as once. But if someone later added a portal count that increments on each authorization, re-initialization would double-count portals.

### What Makes an Initializer Non-Idempotent

- Incrementing counters or nonces
- Appending to arrays (creates duplicates on re-init)
- External calls with lasting side-effects (e.g., minting tokens, sending ETH)
- Operations that depend on prior state (e.g., "add 10 to balance" vs "set balance to 10")


### Other Reasons an Initializer may be Unsafe to Re-Run

- Emitting events that trigger off-chain actions (e.g., indexers that process each event exactly once)
- Overwriting a variable that other contracts or off-chain systems already depend on (e.g., resetting a registry address that live contracts are pointing to, or changing a config value that should be immutable after first init)

### Rule

Non-idempotent or unsafe-to-rerun behavior in `initialize()` / `reinitializer` functions is **disallowed** unless the consequences are explicitly acknowledged in a `@notice` comment on the function. The comment must explain why the non-idempotent behavior is safe given how callers use the function.

Without this comment, the code must not be approved.

### Review Checklist

When reviewing changes to `initialize()` or its callers:

1. **Is every operation in this initializer idempotent?** Assigning a variable to a fixed value is idempotent. Incrementing, appending, or calling external contracts may not be.
2. **Could overwriting any variable be unsafe?** Some values should only be set once — overwriting them during re-initialization could break other contracts or systems that depend on the original value.
3. **Can this contract be re-initialized?** Check for `reinitializer` modifier. If it only uses `initializer` (one-shot), the risk does not apply.
4. **If non-idempotent or unsafe behavior exists, is there a `@notice` comment acknowledging it?** The comment must explain why it's safe. If the comment is missing, flag it as a blocking issue.

## Scope and Architecture

L1 and L2 smart contracts in the OP Stack live primarily in `packages/contracts-bedrock/`.
These contracts secure real value — OP Mainnet, Base, and other Superchain members — so every
change carries risk, especially changes to the implementations in
`packages/contracts-bedrock/src/`. The canonical Solidity style guide is at
`packages/contracts-bedrock/book/src/contributing/style-guide.md`, the interface policy at
`packages/contracts-bedrock/book/src/contributing/interfaces.md`, and the versioning and upgrade
policies under `packages/contracts-bedrock/book/src/policies/`.

### Proxy System

All protocol contracts live behind EIP-1967 transparent proxies. The proxy implementation is
custom (not OpenZeppelin), located at `src/universal/Proxy.sol`. A `ProxyAdmin` contract manages
upgrades for all proxies in a system.

Key properties:

- Admin calls are blocked from being proxied (transparent proxy pattern).
- `msg.sender == address(0)` check allows `eth_call` simulation.
- Atomic upgrade-and-call via `upgradeToAndCall()`.
- Legacy support for CHUGSPLASH and RESOLVED proxy types.

### Cross-Chain Messaging

Two messaging systems:

- **L1<>L2**: `CrossDomainMessenger` (abstract base) with L1 and L2 specializations.
- **L2<>L2**: `L2ToL2CrossDomainMessenger` (predeploy at `0x4200...0023`).

The message nonce encodes a version in the upper 16 bits (lower 240 bits are the nonce). V1
messages include sender, target, value, gasLimit, and data. Gas overhead constants account for
EIP-150 63/64 forwarding.

### Key Invariants

- **Proxy upgrade safety**: storage layout must never change incompatibly.
- **Initializer guards**: contracts must not be re-initializable without the StorageSetter flow.
- **Bridge message integrity**: cross-domain messages must be provably relayed.
- **Deposit transaction ordering**: deposits are processed in L1 inclusion order.
- **No duplicate message relay**: the `successfulMessages` mapping prevents replays.
- **Reentrancy safety**: transient storage guards on all message relay paths.

### Key Contracts

| Contract | Purpose | Location |
|----------|---------|----------|
| OptimismPortal2 | L1 deposit/withdrawal portal | `src/L1/` |
| SystemConfig | On-chain system configuration | `src/L1/` |
| SuperchainConfig | Global Superchain config (pause, guardian) | `src/L1/` |
| ETHLockbox | Unified ETH liquidity for authorized portals | `src/L1/` |
| L1CrossDomainMessenger | L1 cross-domain messaging | `src/L1/` |
| L1StandardBridge | L1 token bridge | `src/L1/` |
| L1ERC721Bridge | L1 ERC-721 bridge | `src/L1/` |
| OPContractsManager | Manages L1 contract deployments and upgrades (impl: `OPContractsManagerV2`) | `src/L1/opcm/` |
| L2ContractsManager | L2CM — manages upgrades of the L2 predeploys | `src/L2/` |
| CrossDomainMessenger | Abstract base messenger | `src/universal/` |
| StandardBridge | Abstract base bridge | `src/universal/` |
| Proxy | EIP-1967 transparent proxy | `src/universal/` |
| ProxyAdmin | Proxy management | `src/universal/` |
| DisputeGameFactory | Factory creating/registering dispute games | `src/dispute/` |
| FaultDisputeGame | Fault proof dispute resolution | `src/dispute/` |
| AnchorStateRegistry | Stores the latest anchor state per game type | `src/dispute/` |

### Utility Libraries

| Library | Purpose |
|---------|---------|
| Hashing | Cross-domain message hashing, deposit source hashing |
| Encoding | RLP encoding, versioned nonce encoding |
| SafeCall | Gas-safe external calls with EIP-150 accounting |
| Constants | Protocol-wide constants and addresses |
| Predeploys | L2 predeploy addresses |
| Storage | Low-level storage access (sload/sstore) |
| TransientContext | Transient storage reentrancy guards |
| SemverComp | Runtime semver comparison |

### Source Directory Structure

```
src/
├── L1/           # L1 protocol contracts (OptimismPortal2, SystemConfig, bridges)
│   └── opcm/     # OPContractsManager implementations
├── L2/           # L2 predeploy contracts (GasPriceOracle, messengers, bridges)
├── universal/    # Shared by L1 and L2 (Proxy, ProxyAdmin, StandardBridge, CrossDomainMessenger)
├── libraries/    # Pure utility libraries (Hashing, Encoding, SafeCall, Constants, Predeploys)
├── dispute/      # Fault proof dispute game contracts
├── governance/   # Governance contracts
├── safe/         # Safe multisig extensions
├── cannon/       # Cannon VM contracts
├── periphery/    # Peripheral contracts
├── integration/  # Integration utilities
├── vendor/       # External vendored code
└── legacy/       # Deprecated contracts
```

## Proxy and Upgradeability

Every new implementation contract must follow this pattern:

1. Extend OpenZeppelin's `Initializable`.
2. Include `initialize()` with the `initializer` modifier.
3. In the constructor: call `_disableInitializers()` and set immutables only.
4. Extend `ReinitializableBase(N)` with the current init version.
5. Never use `reinitializer(uint64 version)` — this codebase does not use it.

### Upgrade Process (Atomic 3-Step)

1. Upgrade implementation to `StorageSetter`.
2. Use StorageSetter to zero the initialized slot (typically slot 0).
3. Upgrade to the new implementation and call `initialize()`.

This is done atomically via `ProxyAdmin.upgradeAndCall()`.

### Storage Layout

- Never modify existing storage slot assignments.
- Use `private` spacer variables for removed fields: `spacer_<slot>_<offset>_<length>`.
- Tag spacers with `@custom:legacy` and `@custom:spacer`.
- Storage gaps for inheritance: `uint256[N] private __gap`.
- CI validates storage layout via snapshots in `snapshots/storageLayout/`.
- Deterministic storage slots are used in SystemConfig (via `keccak256("systemconfig.fieldname")`).

### Access Control

- `ProxyAdminOwnedBase` for proxy admin ownership checks.
- `CrossDomainOwnable3` for L2 contracts with cross-domain ownership.
- `onlyEOA()` modifier to prevent smart contract wallet calls.
- `onlyOtherBridge()` for bridge message validation.
- Use `ICrossDomainMessenger.xDomainMessageSender()` for cross-chain caller verification.

## Reentrancy Protection

- Transient storage-based guards (EIP-1153) via `TransientReentrancyAware`.
- `nonReentrant` modifier on message relay functions.
- Call depth tracking via `TransientContext.increment()` / `decrement()`.
- The `successfulMessages` mapping prevents duplicate message relay.

## Cross-Chain Messages

- Message versioning embedded in nonce (upper 16 bits = version, lower 240 = nonce).
- V1 encoding: `abi.encode(nonce, sender, target, value, gasLimit, data)`.
- V1 hash: `keccak256(abi.encode(nonce, sender, target, value, gasLimit, data))`.
- Gas overhead constants defined in CrossDomainMessenger (200k relay constant, 5k check buffer).
- 63/64 gas forwarding rule (EIP-150) handled by the SafeCall library.

## Contract Organization

Reference implementations: `SystemConfig.sol` and `OptimismPortal2.sol`.

```solidity
// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { ProxyAdminOwnedBase } from "src/L1/ProxyAdminOwnedBase.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

// Libraries
import { SafeCall } from "src/libraries/SafeCall.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @custom:proxied true
/// @title ContractName
/// @notice Description
contract ContractName is Initializable, ProxyAdminOwnedBase, ReinitializableBase, ISemver {
    // Constants and immutables
    // Custom errors
    // Events
    // State variables (with @custom:network-specific where appropriate)
    // Spacers (with @custom:legacy and @custom:spacer)
    // Constructor (call _disableInitializers())
    // Initializer
    // External functions
    // Internal functions
}
```

## Solidity Standards

### Tooling

- Foundry under the hood, but always drive it through the `just` recipes in
  `packages/contracts-bedrock/justfile` — never call `forge` directly. The recipes wire up
  go-ffi, profiles, and the script cache for you.
- `just build` builds the contracts; `just build-dev` is the faster variant
  (`FOUNDRY_PROFILE=lite`) for local iteration. Builds must produce zero warnings
  (`deny_warnings = true` in `foundry.toml`).
- `just test` runs the suite; `just test-dev` is the faster `lite`-profile variant for local
  iteration. Default 64 fuzz runs; CI uses 128.
- `just lint` formats and checks (`forge fmt` under the hood: 120-char line length, bracket
  spacing, multiline func headers).
- Semgrep for security linting (custom rules in `.semgrep/rules/`, via `just semgrep`).
- Slither for static analysis.

In `packages/contracts-bedrock`, run every recipe through `mise` so it uses the pinned toolchain
(see [Build and Test Commands](#build-and-test-commands) below): e.g. `mise x -- just build-dev`.

### Pragma

- **Unified Solidity version** across the codebase (currently `0.8.15` for most contracts).
- **Pin the final derived contracts and scripts** — concrete (non-abstract) production contracts
  and deploy scripts must use an exact pragma (`pragma solidity 0.8.15;`). The `strict-pragma`
  CI check enforces this on files containing a concrete contract.
- **Floating pragmas are fine for reusable libraries** — a `^0.8.0` pragma is common and
  acceptable for `src/libraries/`, abstract base contracts, and interfaces. These are not pinned
  because they're consumed by the pinned contracts that derive from them, and CI deliberately
  exempts libraries, interfaces, and abstract contracts from the strict-pragma check.
- Never introduce a new Solidity version without a formal design-doc proposal.
- New versions must be at least 6 months old before adoption.

### Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Function parameters | `_underscorePrefix` | `function set(address _newOwner)` |
| Return values | `underscoreSuffix_` | `returns (uint256 balance_)` |
| Event parameters | `camelCase`, no prefix | `event Transfer(address from, address to)` |
| Custom errors | `ContractName_Description` | `error SystemConfig_InvalidCaller()` |
| Immutables | `SCREAMING_SNAKE_CASE`, `internal` | `address internal immutable OWNER_ADDRESS` |
| Constants | `SCREAMING_SNAKE_CASE` | `uint256 internal constant DEPOSIT_VERSION = 0` |
| Spacers | `spacer_<slot>_<offset>_<length>`, `private` | `bytes32 private spacer_52_0_32` |
| Struct storage vars | `_underscorePrefix`, `internal` | `Config internal _config` |

### Immutables

- Must be `internal` (never `public`).
- Must have a handwritten getter returning the lowercase name.
- This decouples the ABI from whether a value is stored or immutable.

```solidity
address internal immutable OWNER_ADDRESS;
function ownerAddress() public view returns (address) { return OWNER_ADDRESS; }
```

### Struct Storage Variables

- Must be `internal` with a `_` prefix.
- Must have a handwritten getter returning the struct type (not a tuple).
- Solidity auto-generated getters return tuples, which breaks ergonomics.

```solidity
Config internal _config;
function config() public view returns (Config memory) { return _config; }
```

### Errors

- Custom Solidity errors for all new code.
- Format: `error ContractName_ErrorDescription()`.
- Revert with `revert ContractName_ErrorDescription()`.
- No `require(condition, "string")` or `revert("string")` in new code.

### NatSpec Comments

- Triple-slash `///` style.
- Use `@notice` exclusively (never `@dev`).
- Newline between `@notice` and the first `@param`.
- Newline between `@param` and the first `@return`.
- 100-character line length for comments.

Custom tags:

- `@custom:proxied` — contract lives behind a proxy.
- `@custom:upgradeable` — contract meant to be inherited by upgradeable implementations.
- `@custom:semver` — version variable (semver format).
- `@custom:legacy` — function/event exists only for backwards compatibility.
- `@custom:network-specific` — state variables that vary between OP Chains.
- `@custom:spacer` — spacer variables for removed storage.

### Import Order

Imports must be grouped and ordered — contracts first, libraries second, interfaces last:

```solidity
// Contracts (first)
import { StandardBridge } from "src/universal/StandardBridge.sol";

// Libraries (second)
import { SafeCall } from "src/libraries/SafeCall.sol";

// Interfaces (last)
import { ISemver } from "interfaces/universal/ISemver.sol";
```

### Interface Policy

- Source contracts must NOT inherit from their own interfaces.
- Contracts CAN import interfaces for OTHER contracts.
- Every source contract must have a corresponding interface in `interfaces/`.
- Interfaces must include a `__constructor__()` pseudo-constructor.
- CI enforces a 1:1 ABI match between source and interface.

### Versioning

- All non-library, non-abstract contracts must implement `ISemver`.
- Expose `string public constant version = "X.Y.Z";` with the `@custom:semver` tag.
- Patch: comment-only changes (no bytecode change except the version string).
- Minor: bytecode or ABI expansion (non-breaking).
- Major: breaking interface or security model changes.
- `version >= 1.0.0` required for production readiness.
- **Bump once per PR, not per commit** — PRs are squash-merged, so only one commit appears in
  history. The final version should reflect the total change from the PR's base branch.

### Events

- All state-changing functions must emit a corresponding event.
- Events enable transparent monitoring and log reconstruction.

## Testing Conventions

### Function Naming

Format: `[method]_[FunctionName]_[reason]_[status]`

- `[method]`: `test`, `testFuzz`, or `testDiff`.
- `[FunctionName]`: function or behavior being tested.
- `[reason]`: optional description (required for `reverts` / `fails`).
- `[status]`: `succeeds`, `reverts`, `works`, `fails`, or `benchmark`.

Rules: camelCase per part, no double underscores, exactly 3 or 4 parts.

```solidity
// Valid
function test_transfer_succeeds() external { }
function test_transfer_insufficientBalance_reverts() external { }
function testFuzz_balanceOf_randomAccount_succeeds(address _account) external { }

// Invalid
function test_transfer_reverts() external { }         // Missing reason
function test_TRANSFER_succeeds() external { }        // Not camelCase
function testTransferSucceeds() external { }          // No underscores
```

### Contract Naming

- `<ContractName>_<FunctionName>_Test` — tests for a specific function.
- `<ContractName>_TestInit` — reusable initialization/setup.
- `<ContractName>_Harness` — expose internal functions for testing.
- `<ContractName>_Uncategorized_Test` — miscellaneous tests.

### Test File Organization

- Files in `test/` with the `.t.sol` extension.
- Mirror the `src/` directory structure.
- One test contract per function being tested.
- All tests inherit from `CommonTest` (provides a full OP Stack deployment).

### Testing Infrastructure

- `CommonTest` base class deploys the full OP Stack (L1 + L2).
- Pre-configured actors: alice and bob with 10,000 ETH each.
- Feature flags for testing variants: altDA, interop, revenue sharing, custom gas token.
- Fork test support: automatic detection via the `FORK_TEST` env var.
- Invariant tests in `test/invariants/` with guided and unguided fuzz modes.
- Kontrol formal verification in `test/kontrol/`.
- Go FFI for off-chain computation in tests.

## Foundry Configuration Highlights

- Default optimizer: 999,999 runs.
- Dispute/OPCM contracts: 5,000 runs (bytecode size management).
- EVM version: cancun.
- Extra output: devdoc, userdoc, metadata, storageLayout.
- FFI enabled for scripts/tests.
- Gas limit: max int64 (for large tests).
- Fuzz runs: 64 (default), 128 (CI), 20,000 (ciheavy).

| Profile | Optimizer | Fuzz Runs | Use Case |
|---------|-----------|-----------|----------|
| default | 999,999 runs | 64 | Production builds |
| lite | disabled | 8 | Fast dev iteration |
| ci | 999,999 runs | 128 | CI testing |
| ciheavy | 999,999 runs | 20,000 | Stress testing |
| cicoverage | disabled | 1 | Coverage only |
| kprove | default | — | Kontrol formal verification |

Dispute games, OPCM, OptimismPortal2, and ProtocolVersions compile with 5,000 optimizer runs
for bytecode size management.

## Build and Test Commands

All commands in `packages/contracts-bedrock` must run through `mise` so they use the pinned
versions of forge, solc, go, etc. Never run a bare `just <target>` or `forge <cmd>` — these
bypass the pinned toolchain.

```bash
mise x -- just build-dev          # Fast dev build (preferred for local work)
mise x -- just test-dev           # Fast dev tests (preferred for local work)
mise x -- just lint               # Format fix + check
mise x -- just pr                 # Full pre-PR suite: build, lint, all checks
mise x -- just test-upgrade       # Fork tests against mainnet state (needs ETH_RPC_URL)
mise x -- just semver-lock        # Regenerate semver-lock.json
mise x -- just snapshots          # Regenerate all snapshots
mise x -- just semver-lock-no-build  # Regenerate from existing artifacts (faster)
```

`just build` and `just test` run production builds with full optimization — slower. Use
`just build-dev` and `just test-dev` for day-to-day iteration. Recipes forward extra args to
`forge`, so pare a run down to what you're working on with `--match-contract` / `--match-test`
(e.g. `mise x -- just test-dev --match-contract OptimismPortal2_Test`) — much faster than the
whole suite. Run the full `just test` before opening a PR, since that's what CI runs.

`just test-upgrade` forks mainnet (or Sepolia) at a weekly-pinned block, applies the upgrade
path, and runs tests in `test/{L1,dispute,cannon}/`. It verifies that upgrades work against
real deployed state — the actual upgrade path, not just a clean deployment. Requires
`ETH_RPC_URL`. Run it when modifying upgradeable contracts or the upgrade flow itself.

## CI Checks

All of these must pass:

- `forge fmt --check` — formatting.
- Semgrep scan — security rules.
- Snapshot generation — ABI + storage layout + semver lock.
- Semver diff — version bump required when bytecode changes.
- Unused imports — no dead imports.
- Strict pragma — no floating pragmas.
- Storage spacers — spacer naming and placement.
- Reinitializer modifiers — proper upgrade guards.
- Interface correctness — 1:1 ABI match.
- Contract size — within the EIP-170 limit.
- Test naming — conventions enforced by the validation script.

## Rebase Conflicts in Generated Snapshot Files

`snapshots/semver-lock.json` is generated — **both sides of a conflict are wrong**. The hashes
must be recomputed from the actual compiled artifacts after the rebase lands. Even hashes the
branch author pre-computed may be stale.

1. Accept either side to clear the conflict markers.
2. Run `mise x -- just semver-lock` in `packages/contracts-bedrock/` to regenerate.
3. Stage the regenerated file and amend the commit (or continue the rebase).

The same rule applies to any other generated snapshot file (`snapshots/storageLayout/`, etc.).
If the build fails (no network/solc), the rebase cannot be completed correctly — say so rather
than leaving any manually-chosen hashes in place.

## Before Committing Contract Changes

1. `just test-dev` — zero test failures (fast `lite`-profile iteration). While iterating on a
   specific change, narrow the run with `forge`'s filters to avoid the whole suite, e.g.
   `just test-dev --match-contract OptimismPortal2_Test` or
   `just test-dev --match-test test_finalizeWithdrawalTransaction_succeeds`. Run the unfiltered
   `just test-dev` (and `just test`, the full-optimization variant CI runs) before opening the PR.
2. `just pr` — full pre-PR suite: build, lint, all checks.
3. Bump the contract version if bytecode changed (once per PR, not per commit — PRs are
   squash-merged).
4. Review with a security focus — these contracts secure real value.
