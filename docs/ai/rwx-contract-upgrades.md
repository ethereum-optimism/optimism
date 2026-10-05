# L1 upgrade shadow

The seven Circle PR upgrade occurrences are implemented by
`.rwx/contract-upgrades.yml` and `ops/ci/contract-upgrades.py`. All seven pass complete hosted, same-revision original-report parity at
`58a81fbdb7f1c6184b881417f4d74c50c73f13d6`. The optional check is `optimism-contract-upgrades-shadow`; Circle keeps
its required contract gate.

| Occurrence | Chain | Feature |
| --- | --- | --- |
| feature-main | op | main |
| feature-CUSTOM_GAS_TOKEN | op | CUSTOM_GAS_TOKEN |
| feature-OPTIMISM_PORTAL_INTEROP | op | OPTIMISM_PORTAL_INTEROP |
| feature-ZK_DISPUTE_GAME | op | ZK_DISPUTE_GAME |
| chain-op | op | main |
| chain-ink | ink | main |
| chain-unichain | unichain | main |

The two op/main occurrences execute separately. Each has a dedicated compiler
and verdict task. Compilers and verdicts use 16 CPU / 32 GiB, with isolated
Foundry/Go compilation caches. Module preparation is shared and verified.
Fresh verdicts retain only the separate Foundry RPC cache; results and logs are
artifacts. Protected `develop` warming targets compilers and executes zero tests.

The shared adapter preserves `liteci` on PR branches, `ci` on `develop`, fuzz
seed 42424242 and one fuzz run. The original `just test-upgrade` recipe selects
all L1, dispute and Cannon tests. Compiler preparation explicitly builds the
contracts before `forge test --list --json`: Forge 1.8.3's list mode alone
produces ABI-only artifacts on a cold checkout. Full method identifiers resolve
bare discovery names, including overloads, into the complete JUnit signatures.
Discovery also lists inherited tests on abstract contracts. The complete list
is retained; only declarations bound to empty compiler creation bytecode are
classified as non-executable. A skipped `setUp()` expands into skipped members
of that same deployable contract, with the original setup verdict and skip
reason retained. Every other absent, extra or conflicting verdict fails.
Runtime workers verify the revision, tools, settings, complete recursive
submodule commits, source inputs and every compiler artifact before execution.

The existing test-only L1 archive vault is used by RPC preflight and runtime
workers. Preflight precedes expensive builds, verifies chain ID 1 and the
original Just daily 00:00 UTC block, and records its height, hash and timestamp.
Runtime verifies that same block again. Compiler tasks receive no RPC input.
Authentication is masked before dependent output is streamed, saved or hashed.
Circle uses the same adapter and retains the original JUnit plus separate
failed-test diagnostics; a passing diagnostic cannot replace a failed verdict.
Cancellation preserves the process signal and partial original output.

Twenty-one execution/comparison fixtures pass with pinned Linux Go, Forge and
Just. The complete adapter fixture uses the production build/runtime recipes,
real Go FFI compilation, real Solidity tests and a local read-only RPC server.
Two runs from the same compiled manifest each execute the FFI test afresh;
corrupt FFI bytes fail before execution. An intentional failure retains both
original and diagnostic JUnit. Other fixtures exercise full signature discovery,
individual and whole-contract setup skips, abstract and inherited tests,
wrong/unavailable block inputs, redaction, process-group cancellation,
corrupt originals, stale revision/settings, missing cases and uninvestigated
retries. Routing and Circle adapters pass all 35 scenarios. RWX lint, shell
blocks and merged/activated Circle configuration validation pass.

Hosted comparison must retain every original report, complete selection,
compiler/signature bindings, settings, exact block and retry history for all
seven occurrences on the same SHA. `ops/ci/compare-contract-upgrades.py` rejects
missing, extra, corrupt, failed or different inputs and outcomes. A date-boundary
block difference requires a new comparable observation. No workload has been
added to the verified count by these local fixtures.

[First-failure evidence](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-contract-upgrades-evidence/first-failure.json)
retains the initial shared-client RPC error on both providers. All seven
Circle preparations discovered 1,359 complete test signatures each, while the
native preflight prevented compilation and verdict execution. Archive-client
headers now match the passing Go preflight, with safe numeric error categories.
The CLI identity correctly remains unable to unlock this test-only archive
vault; original reports are retrieved through the existing authorized user
access without changing credential permissions.

The follow-up observation at `59f3f33f72354f2521525a26a331fa771b6c1edc`
passed RPC preflight, every compiler and all seven original Just upgrade
commands on both providers. Complete raw JUnit identities, outcomes and skip
reasons match for every variant. The new adapter nevertheless failed because
it expected individual verdicts for setup-skipped contracts and four methods
on an abstract test initializer. The corrected compiler/skip classification
above is covered by actual pinned Forge execution. This failed observation
remains in the evidence index and adds zero verified occurrences pending a
successful hosted run of the corrected adapter.

The initial compiler-only warm rehearsal also exposed skipped-preflight
propagation: referring to its absent `ready` value skipped every compiler.
Producers now use a status-only `after` condition that accepts successful or
intentionally skipped preflight, plus the existing contract-route/warm-only
condition. Failed preflight cannot start producers. RWX's
[task dependency documentation](https://www.rwx.com/docs/after) describes the
status scope; run-initiation validation rejects initialization parameters
inside that expression. A skipped-only run is not evidence of cache warming.

The first hosted bytecode-classification attempt at `1926d60b` rejected multiple
compiler contexts before verdicts. The original artifacts include twenty
contracts compiled with more than one Solc version; identical method signatures
can have distinct creation bytecodes. Every artifact now retains its own size
and hash. Only a contract whose complete bound artifact set has empty creation
bytecode is non-executable. Cross-provider comparison requires equal signatures
and deployability while preserving each provider's compiler artifact hashes.

The successful benchmark is [native run bcab8b6f](https://cloud.rwx.com/optimism/runs/bcab8b6f456b42da908aefb5d4867773)
and [Circle pipeline 135563](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135563).
The [parity index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-contract-upgrades-evidence/parity.json) retains complete
selection, original outcomes, skip reasons, settings, source/tool/compile
bindings, block identities and original-report hashes. Every variant has
1,359 discovered signatures: 1,355 executable and four non-executable abstract
declarations. Main, CGT and each chain occurrence report 861 passes / 494 skips;
Interop reports 983 / 372 and ZK reports 986 / 369. All seven original commands
succeeded with no diagnostic rerun or task retry. This adds seven verified
implementation occurrences, reaching 56/86 (65%); thirty remain.

The compiler-only rehearsal completed all seven targets with zero tests. A
separate intentional preflight failure kept producers and verdicts skipped.
Protected `develop` cache-rebuild events remain unobserved until these
definitions reach that branch. The complete compiler-context rehearsal verified
the executed helper hash: an explicit CLI `commit-sha` override suppresses
uncommitted helper patches, so patched rehearsals must omit that override.
These rehearsals do not substitute for source-matched hosted verdicts.

All four required Circle gates, dependency review and all nine optional RWX
checks passed at the benchmark revision: 146 successful checks, one neutral,
zero unfinished or failed checks. Later pushes require their own observations.
