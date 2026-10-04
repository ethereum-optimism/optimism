# L1 upgrade shadow

The seven Circle PR upgrade occurrences are implemented by
`.rwx/contract-upgrades.yml` and `ops/ci/contract-upgrades.py`. They remain
unchecked in the parity inventory until hosted, same-revision original reports
agree. The optional check is `optimism-contract-upgrades-shadow`; Circle keeps
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

Twenty execution/comparison fixtures pass with pinned Linux Go, Forge and
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

[First-failure evidence](rwx-contract-upgrades-evidence/first-failure.json)
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
