# Cannon Go shadow

The optional `optimism-cannon-go-shadow` implements Main's
`cannon-go-lint-and-test` through the shared `run-main` routing. This occurrence
is **pending hosted verification** and does not yet increase verified coverage
above 74/86. Circle retains the original job name, dependency, notifications,
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
Full hosted suite and cache evidence remain pending.

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
[first-failure evidence](rwx-cannon-go-evidence/first-failures.json) records their
original hashes, settings and rejected setup request.
The shared artifact restorer now verifies matching destination bytes and the
corresponding archived payload before reusing an existing regular file. It
preserves the read-only directory/file modes and rejects corrupt archive bytes
even when the existing destination is correct. Six artifact helper fixtures
pass, including repeated read-only restoration, and the live Go/Forge fixtures
exercise this exact cache shape. Full hosted parity after the fix remains pending.
