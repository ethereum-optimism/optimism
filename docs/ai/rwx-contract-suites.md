# Standard and changed-file contract suites

The existing optional `optimism-contracts-shadow` now uses one shared adapter for
the four standard feature variants and four changed-file heavy-fuzz variants.
All eight now pass complete same-revision original-report comparison.
All eight occurrences have verified parity. The current
[inventory](rwx-parity-todos.md) is 74/86 (86%), with twelve remaining after the
coverage and static-check closeouts. Circle retains its required gate.

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
whole-contract setup skips. Original skip reasons remain untouched. The first
benchmark retains two reasonless Interop L1Block skips with their exact source
guard; the follow-up below verifies the explicit messages on both providers.

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
change are retained in [first-failures.json](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-contract-suites-evidence/first-failures.json).

At `c702cfb2d41a0220383cebdf9013530dd25d50c0`,
[RWX run 96f1589c](https://cloud.rwx.com/optimism/runs/96f1589c68bc418eacd5d37bfc259585)
and [Circle pipeline 135567](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135567)
passed all eight complete original comparisons. Standard selection contains all
164 files and 2,882 signatures, including 23 non-executable abstract declarations.
The 2,859 executable outcomes per feature are:

| Feature | Pass | Skip |
| --- | ---: | ---: |
| main | 2,313 | 546 |
| Custom gas token | 2,345 | 514 |
| Interop | 2,419 | 440 |
| ZK dispute game | 2,438 | 421 |

Each heavy variant selects the actual modified Bytes and ResourceMetering files,
with all 28 cases: 27 passes and one explicitly skipped manual CSV generator.
All settings, file assignments, original XML/skip details, source/fixture hashes,
selected compiler signatures and invocation histories agree. No diagnostic rerun
or native task retry occurred in these successful verdicts. All 154 GitHub checks
at this benchmark are terminal: 153 successful and one neutral, including the
four required Circle gates and every optional RWX check.

Restored Foundry outputs contain 25 additional obsolete line-numbered
`VmContractHelper` interfaces. Every one has empty creation bytecode and no test
or invariant selector. Complete original bindings are retained; comparison checks
every executable contract and test-bearing abstract declaration and records these
non-test interface differences explicitly. Regression coverage rejects any extra
executable binding or test/invariant selector. The two original Interop L1Block
skips lack emitted reasons; their exact existing feature condition is sealed in
the evidence. All other original skip records have reasons. No original reason
is invented or rewritten.

## Current-source follow-up

All eight occurrences also pass complete original-report comparison at
`ffefdb34638470dd1126cbf2aaebb4644400b6fb` on
[native run ffab3bbe](https://cloud.rwx.com/optimism/runs/ffab3bbedad3416eb8424baef0d13abe)
and [Circle pipeline 135572](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135572).
The [follow-up index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-contract-suites-evidence/ffef-parity.json) retains every
selection, outcome, skip, source/tool setting, compiler binding, fixture mutation
and original report hash. All 950 original files verify. Full originals remain
under `.ci/rwx-contract-suites-evidence/ffef/` and the provider artifacts.

Standard selection and outcomes remain the complete 164-file/2,882-signature
workload above. The changed-file authority now selects three real files:
`ResourceMetering.t.sol`, `L1Block.t.sol` and `Bytes.t.sol`. The additional changed
L1Block file brings heavy selection to 58 signatures: 56 executable cases and two
abstract declarations. The full `ciheavy` counts/timeouts remain unchanged.

| Heavy feature | Pass | Skip |
| --- | ---: | ---: |
| main | 50 | 8 |
| CUSTOM_GAS_TOKEN | 53 | 5 |
| OPTIMISM_PORTAL_INTEROP | 48 | 10 |
| ZK_DISPUTE_GAME | 50 | 8 |

The two Interop L1Block cases now emit the exact original reason
`Interop is already enabled by the dev feature` in both providers' standard
JUnit. Their existing feature condition and assertions remain unchanged. The
comparison confirms these messages directly; the historical reasonless records
remain intact in the first benchmark evidence.

[Compiler-only rehearsal ef8b816c](https://cloud.rwx.com/optimism/runs/ef8b816ca9fe470f8e34369cdc9d8fa7)
passed all eight producers at the original `c702cfb2` benchmark revision with zero tests and no
verdict tasks. Every build reports unchanged compilation skipped. Every contract
artifact and Go FFI binary is byte-identical; only
`cache/solidity-files-cache.json` changes. These tasks executed and reused native
compiler data; they were not filesystem task-cache hits. This is a CLI rehearsal,
not an observed protected `develop` cache-rebuild event or a speed comparison.

[parity.json](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-contract-suites-evidence/parity.json) retains the complete shared
source hashes, authoritative selections, per-case outcomes and original skips,
all provider report hashes, generated fixture changes, interface reconciliation,
final checks and the eight preparation-only original inventories. Raw originals
remain in both hosted runs and in `.ci/rwx-contract-suites-evidence/c702`.
Case and skip-reason catalogs deduplicate repeated details; decoding is checked
against every complete original comparison result without dropping fields.
Coverage, L2 fork and gate equivalents remain separate work.
