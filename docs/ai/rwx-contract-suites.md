# Standard and changed-file contract suites

The existing optional `optimism-contracts-shadow` now uses one shared adapter for
the four standard feature variants and four changed-file heavy-fuzz variants.
The latter remain uncounted until their complete hosted originals pass
same-revision comparison. Total implementation coverage remains 63/86 (73%),
with 23 occurrences remaining. Circle retains its required gate.

`contract-suites.py` preserves Circle's original `find test -name "*.t.sol"`
selection for standard tests, and the exact `git diff origin/develop...HEAD`
added/modified `.t.sol` selection for heavy fuzzing. Circle still invokes its
actual `circleci tests split --split-by=timings`; the current job has one
partition. Native records an equivalent complete assignment. Both retain the
original file-selection and split logs, configured parameter, target revisions,
full Forge discovery and compiler signatures. Empty changed-file selection
prepares zero tests and disables native verdicts; it cannot establish coverage.

Standard profiles remain `liteci` on PRs and `ci` on `develop`: 128 fuzz runs,
64 invariant runs and depth 32. Modified tests retain `ciheavy`: 20,000 fuzz
runs, 128 invariant runs, depth 512, and 300-second fuzz and invariant timeouts.
The original profile, feature and filter parameters are checked against actual
settings. No inherited environment may silently reduce the workload.

Each suite/feature compiles independently on 16 CPU / 32 GiB with isolated
Foundry and Go compiler caches. Fresh verdicts start from verified compiled
artifacts on 16 CPU / 32 GiB and preserve full source paths, Git history,
submodules, fixtures, toolchain and revision metadata. Runtime Go compilation
has a separate cache. The original `forge test --match-path ... --junit`,
nonempty-JUnit guard and `just lint-forge-tests-check-no-build` execute. Initial
failure evidence remains intact when `just test-rerun` emits diagnostic traces.
Reports, outcomes and generated fixtures are excluded from reusable outputs.
Compiler-only protected warming runs zero verdicts.

The adapter seals complete tracked inputs, effective configuration, producer
settings, compiler outputs, selected signatures, complete original JUnit and
each command/log/exit/signal. Consumers reject stale settings/revisions,
missing/corrupt binaries, changed target history, fixture selection or tools.
Comparison revalidates each file assignment and signature against original
discovery and accounts for deployable cases, abstract bytecode declarations and
whole-contract setup skips. Original skip reasons remain untouched; unresolved
reasonless skips still need diagnostic evidence before final migration closeout.

Submodules are explicitly initialized recursively before provenance validation;
the Circle checkout does not initialize them by itself. Preparation must leave
every tracked input unchanged. `GenerateNUTBundleTest` intentionally invokes the
real script's `run()` and writes
`snapshots/upgrades/current-upgrade-bundle.json`. When that writer is selected,
the adapter retains this exact snapshot as both an initial fixture and a runtime
output. Its before/after payloads and hashes are sealed, source changes are
enumerated, and comparison requires the same generated payload on both providers.
Any other tracked input mutation still fails. Runtime fixture outputs never
enter reusable compiler outputs.

Existing skip conditions now emit their reasons through Foundry's supported
`vm.skip(bool,string)` interface: production bytecode requirements, coverage
instrumentation, fork/ops-repository exclusions, and manual resource-metering
CSV generation. No guard or test assertion is weakened. Complete hosted reports
must verify the emitted reasons and unchanged outcomes.

Linux fixtures execute the real production Just Go FFI recipe, production Go
convention validator, Forge discovery, both profiles and fresh original verdicts.
They reproduce an intentional failure, retain its original XML and diagnostic
rerun, reject a corrupt reused binary before tests, and prove empty changed-file
selection emits zero tests. The identity split fixture exercises stdin/CLI wiring
outside Circle; hosted comparison must use Circle's actual splitter. Comparison
fixtures reject omitted new files, duplicate assignments, matching wrong
commands, reduced fuzzing, stale inputs, corrupt originals, extra unsealed files
and unexplained retries.

The live fixture also executes an invariant in the complete standard manifest
and verifies that the original changed-file selection includes only the modified
unit/fuzz file. It initializes a previously deinitialized submodule and rejects
undeclared source mutation and corrupt tracked fixture evidence. A timed-out
fixture terminates the runner gracefully so Forge receives cancellation before
temporary-file cleanup.

A documentation-only comment in `test/libraries/Bytes.t.sol` exercises the actual
changed-file selector on the pilot branch. All of that existing file's cases
must run under the full heavy profile on both providers. No test behavior is
changed to obtain a passing check.

At `8236d17b`, all four native heavy variants passed all 14 selected cases.
All four native standard test commands and naming validators passed, but the
adapter rejected the intentional tracked snapshot write. A fresh pinned Linux
reproduction retained the exact changed path and before/after hashes. Circle
failed earlier on uninitialized submodules. Its four heavy fallback diagnostics
were canceled after their original preparation failures were retained; they
provide no Circle heavy-suite verdict. The fallback now requires an actual
test-failure cache, preventing an empty `--rerun` from executing the full suite
after preparation failure. These are retained first failures, not parity proof.
The complete original file hashes, provider observations and exact diagnostic
change are retained in [first-failures.json](rwx-contract-suites-evidence/first-failures.json).

The next hosted observation will supersede the older standard runner's
unfiltered Forge invocation and record the exact Circle file filter, settings,
original skips and invocation histories. Coverage, L2 fork and gate equivalents
remain separate work.
