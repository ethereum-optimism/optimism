# Cannon Go shadow

The optional `optimism-cannon-go-shadow` implements Main's
`cannon-go-lint-and-test` through the shared `run-main` routing. This occurrence
passes complete original-report parity at
`5ee3311d889fd889a70d112b46ed00d8a398508c`. Verified occurrence coverage is
now 75/86 (87%): Main 24/32, Contracts 21/23, Rust 21/22 and Rust E2E 9/9.
Circle retains the original job name, dependency, notifications,
cache namespace and required-gate membership.

The shared adapter runs the original `just lint` compatibility recipe, then
`gotestsum-split.sh --format=testname -- -timeout=10m -parallel=$(nproc) ./...`
from `cannon`. PR runs preserve `SKIP_SLOW_TESTS=true`; scheduled Circle runs
preserve `false` and the 45-minute timeout. No `ci` tag, `-short`, explicit
package concurrency, test retries or package exclusions are added. The root
Go lint job remains the actual Cannon linter; its separate occurrence was
already ported.

RWX always supplies `-count=1`, disables verdict task caching, and retains a
dedicated native Go compiler cache. Circle's existing `c-go_fresh_tests`
benchmark parameter now also controls this adapter, defaulting to false.
Both benchmark providers must run freshly. `go list -e -json ./...` errors and
dependency errors fail discovery even if Go exits zero. Complete test listing
includes every initial Test, Example and Fuzz identity and retains packages
without tests. Every selected package and initial test must appear exactly
once in the original verdict events.

The modules-only producer downloads, verifies and discovers the full module
set before creating a source-bound archive. It keeps the checked-in Cannon
embed placeholders intact: this Circle workload does not receive the full
Go producer's generated VM binaries or hello ELF. The existing reusable `ci`
contract producer runs independently. Both producers bind their archive and
file hashes to the source revision, tool pins, settings and actual tools.
Runtime restores reject missing, corrupt or mismatched inputs. Full source,
submodules, relative fixture paths and Git metadata remain available.

The initial verdict worker has 8 CPUs and 16 GiB, matching the original
`xlarge.gen2` Docker allocation in the
[Circle resource reference](https://circleci.com/docs/reference/configuration-reference/).
Its CPU count, default
package concurrency and effective `-parallel` are recorded. This stage adds
no shard or resource tuning matrix. Warm-only events on protected `develop`
target the two producers and select zero tests or helper fixtures. Native
test reporting uses the bounded projection; complete original JSON, JUnit,
per-test logs, discovery, commands, effective environment and source hashes
remain in the `reports` artifact. Failed and cancelled commands retain their
real exit/signal and available original reports without inventing verdicts.

`compare-cannon-go.py` validates every original file seal, rederives discovery
and coverage, verifies native report projection and runtime dependency
provenance, then compares both providers' complete selection, settings,
initial assignments, outcomes, skip reasons and retry histories on one SHA.
Only execution durations and bound absolute workspace prefixes differ in the
comparison view, apart from explicitly retained CPU visibility differences
under the original `nproc` rule; original bytes remain unchanged. A failed package or
TestMain, a missing terminal verdict, a foreign package, unexpected retry or
altered command prevents parity.

The live helper fixtures use actual Go, gotestsum and Forge. They cover new
packages and packages without tests, repeated fresh execution, source-relative
contract fixtures, runtime builds, original failure evidence, cancellation,
and stale source/settings or corrupt dependency archives. Hosted fixture
artifacts are retained separately from the complete real-workload reports.
Run both helper test modules, shared routing/adapter fixtures, ShellCheck,
RWX lint and merged Circle validation before pushing. Final hosted evidence,
cache observations and check states belong here after execution; scaffolding
and fixture results do not establish full workload parity.

The implementation passed eight Linux helper fixtures (including four actual
Go/Forge execution fixtures), ten complete-report comparison fixtures and
35 shared routing/adapter fixtures. The existing wrapper and routing scripts
passed ShellCheck. RWX lint checked the new run and both dependency packages
with zero problems. Circle's merged 5,289-line config and setup config validated,
and processing with all PR workflows and fresh Go tests enabled succeeded.
Hosted execution and cache observations follow below.

The initial Circle benchmark request (135575) was rejected during setup because
it supplied the continuation's `c-main_dispatch` instead of setup's
`main_dispatch`. No test workload ran in that request. Benchmark dispatch uses
the setup parameter. The shared adapter explicitly accepts Circle's `0`/`1`
boolean environment values as well as `true`/`false`; an actual Circle-style
fixture exercises the fresh setting through the environment without a CLI
override.

Circle's corrected fresh benchmark 135577 at `6df00e26ca5e1ed99162ef0714b22b11f0d985ef`
executed all 16 packages, 112 initial tests and 2,881 case identities, all passing
without retries. Its original `nproc` returned 32 inside the 8-vCPU Docker
allocation. RWX returned 8 on its requested 8-CPU worker. The adapter now retains
the original CPU command/output and the comparer records this difference while
checking every other setting and each provider's exact effective arguments.
The module's toolchain directive selects Go 1.26.6 on both providers despite
the Mise bootstrap's 1.26.5 pin; the actual toolchains and Go environment agree.

The initial native Cannon run at `c833ae75459faf15fbf0fea3c6fbdbcc9562618c`
completed both producers and helpers, then failed before tests because archive
restoration tried overwriting a materialized read-only Go toolchain cache file.
The next head reproduced that preflight failure; neither run counts as test
coverage. Complete original failure reports and task logs remain retained;
[first-failure evidence](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-cannon-go-evidence/first-failures.json) records their
original hashes, settings and rejected setup request.
The shared artifact restorer now verifies matching destination bytes and the
corresponding archived payload before reusing an existing regular file. It
preserves the read-only directory/file modes and rejects corrupt archive bytes
even when the existing destination is correct. Six artifact helper fixtures
pass, including repeated read-only restoration, and the live Go/Forge fixtures
exercise this exact cache shape. The corrected full hosted comparison passes.

[RWX run f536dccd](https://cloud.rwx.com/optimism/runs/f536dccd0d9c419bac8bea355153c09c)
and fresh [Circle pipeline 135579, job 5635468](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135579)
executed the same source revision. Both select all **16 packages, 112 initial
tests and 2,881 case identities**, all passing without skips or retries. The
[complete parity index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-cannon-go-evidence/parity.json) verifies every selected
source file, initial assignment, original command, effective setting, package
verdict, JUnit identity and split-log bytes. It seals **2,989 Circle and 2,991 RWX
original files**. Circle's hash-manifest-declared empty files are explicitly
recorded; missing nonempty reports fail validation.

The sole effective concurrency difference is retained explicitly: Circle's
original CPU discovery returns 32 and RWX's returns 8. Both retain the original
`nproc` rule, their exact arguments and outputs, Go 1.26.6, Forge, gotestsum and
Just versions. Every other workload setting, complete selection, result and
retry history agrees. Complete reports remain under
`.ci/rwx-cannon-go-evidence/5ee3/{circle,rwx}`; the index records every original
hash and native dependency binding. The hosted helper task also passed all
eight execution fixtures and ten comparison fixtures; its original fixture
reports are retained separately. These fixtures do not add occurrence coverage.

Two exact-source CLI dependency-only rehearsals completed with zero verdicts or
helper tests. The first [warm run 358c75a0](https://cloud.rwx.com/optimism/runs/358c75a0894f4179bf4aa10baf73a63d)
executed the producers with reusable tool caches. The unchanged
[replay b1ec0c4a](https://cloud.rwx.com/optimism/runs/b1ec0c4a5cc54e2ebd204d883fd0a84a)
reused both complete module and contract producer task outputs. The
[cache and fixture index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-cannon-go-evidence/cache-and-fixtures.json) retains
their original task/cache identities, tool-cache declarations, source parameters,
zero-test selections and all eight original fixture report seals. Protected
`develop` cache-rebuild events remain unobserved. These are cache correctness
observations; no warm median or speed win is claimed.
