# SafeETHBridge validation

Base: latest fetched `ethereum-optimism/optimism` `develop` at
`a3e9e91a83cde7708123d52727711baef270f093`.
Branch: `poc/safe-eth-bridge`. Validation date: September 17, 2026 (EDT).

The implementation is 264 lines, with 8,804 bytes of deployed bytecode. It adds one
contract, its interface, Foundry tests, a two-L2 acceptance test, documentation, and
ABI/storage/semver snapshots. Existing production L2 implementations are unchanged.
The acceptance fixture upgrades the **existing ETH bridge predeploy** on fresh local
chains. Ordinary bridging remains available alongside the optional safe protocol.

## Commands and results

From `packages/contracts-bedrock`:

```sh
mise x -- just test-dev
mise x -- just test
mise x -- just pr
mise x -- just pr -no-cache
mise x -- just test-dev --match-contract SafeETHBridge --fuzz-runs 128
mise x -- just test --match-contract '"SafeETHBridge|SuperchainETHBridge|ETHLiquidity|L2ToL2CrossDomainMessenger|CrossL2Inbox"' --fuzz-runs 128
mise x -- just build-no-tests
mise x -- just snapshots-no-build
mise x -- just lint
mise x -- just semgrep
mise x -- just nut-bundle-check-no-build
mise x -- just interfaces-check-no-build
```

- Full development-profile suite: **2,337 passed, 0 failed, 172 skipped**. Full optimized
  suite: **2,397 passed, 0 failed, 148 skipped**. These full runs cover the earlier
  safe-only version; the final optional ordinary mode is covered by the focused runs below.
- Final development-profile focused run: **35 passed, 0 failed**.
- Final optimized Interop run: **86 passed, 0 failed**, including all 35 new tests and
  the existing ETHLiquidity invariant (64 runs, 32,000 calls). Both final runs configure
  128 fuzz runs. Earlier safe-only optimized runs also passed (27 focused / 78 broader).
- Builds, snapshot generation, formatting, and final Semgrep scan pass (29 rules,
  zero findings). A narrow style-rule exception preserves the legacy bridge error selector.
- Final `just pr -no-cache`: **15 checks passed, one failed** because the upgrade-bundle
  script artifact was missing (`Could not find target contract`). After `build-no-tests`,
  `nut-bundle-check-no-build` passes and the production bundle has no diff. The other
  checks cover Semgrep, formatting, semver, snapshots, imports, pragma, spacers,
  reinitializers, contract size, test names, and Kontrol summary stability.
- Interface checking passes in the pre-PR runner's clean-source retry. After building
  scripts/tests, the standalone whole-package checker again rejects extra-profile
  artifacts for unchanged `ISuperPermissionedDisputeGame` / `IZKDisputeGame`. The new
  bridge/interface also passes the same checker in isolation using real copies of their
  final artifacts and interface source. Initial stale cached artifacts were removed;
  no unrelated generated snapshots are included.

From the repository root:

```sh
mise x -- just build-superchain-go
mise exec -- just lint-go
RUST_BINARY_PATH_OP_RETH=/home/main/agents/workspace/optimism-atomic-analysis/rust/target/debug/op-reth mise x -- go test ./op-acceptance-tests/tests/interop/contract -run '^TestSafeETHBridge' -count=1 -timeout=10m
```

- Full Go lint passes with **zero issues** and a clean module tidiness check.
- **All three final acceptance tests pass** (58.852 seconds): commit through three real
  messenger relays; refund followed by late ACK and ABORT relay; and ordinary bridging
  interleaved with a pending safe transfer. Assertions cover both bridge states,
  reservations, bridge/pool balances, and native ETH payment. The final run follows
  snapshot generation; an earlier concurrent attempt failed during Go embedding while
  snapshots were being regenerated, before any tests executed.
- Reused the existing local op-reth binary through the supported override. Its SHA-256:
  `e62332a5ce6405b1e0d2b83f1c108fa778248554484f936b1b22306101c609db`.
  It reports an unknown build revision; this is **not** validation of an op-reth binary
  freshly built from this `develop` commit. Go services and contracts use this checkout.
  Standard reproduction can use the documented acceptance recipe with `RUST_JIT_BUILD=1`
  and locally built current Rust dependencies.

Additional static analysis, from `packages/contracts-bedrock`:

```sh
FOUNDRY_EVM_VERSION=london mise x -- slither src/periphery/interop/SafeETHBridge.sol --compile-force-framework solc --solc /home/main/.local/share/svm/0.8.15/solc-0.8.15 --filter-paths 'lib/|scripts/|src/libraries/|src/universal/'
```

Slither reports **35 findings**, so this is not a clean Slither result. London is used
because solc 0.8.15 cannot accept `--evm-version cancun`. Findings cover the unused safe
messenger hash return; events after external calls; intended deadline comparisons;
repository parameter naming; mixed pragmas; and compiler-version advisories. Calls go
only to the known messenger/ETHLiquidity and SafeSend; protocol state updates precede
calls, and SafeSend never invokes recipient code. Recovery uses messenger logs rather
than a stored return hash. Compiler advisories remain subject to production audit.

Go, contract/security, and documentation/coverage reviews were completed. Actionable
findings were fixed, including a third-chain test ID and waiting for the supernode's
validated initiating timestamp before relaying. Production migration, reorg/fault-proof
scenarios, and formal verification remain outside this PoC. See the design note for
exact safety assumptions and liveness limits. Raw local logs are in
`~/agents/workspace/optimism-atomic-bridge-validation`.
