# Aggregate Go RWX shadow

The full Circle `go-tests` workload is implemented in `.rwx/go-tests.yml`, with
optional status `optimism-go-tests-shadow`. Circle still owns every required
gate. Acceptance, Cannon's dedicated test suite, Rust suites, and
`op-deployer/pkg/deployer/forge` remain outside this workload, exactly as excluded
by `just list-test-packages`. Packages without tests remain in the selection.

## Execution contract

`go-suite.py` runs the shared Just selector, validates every selected package
with `go list -e -tags=ci -json`, and produces an exhaustive, duplicate-free
manifest. Duration estimates affect assignment only. Historical Circle job
5625852 seeds the balancer; cached historical observations are estimates and
cannot establish fresh parity.

Each of the initial 12 shards compiles on 16 CPU / 32 GiB, starting after the
Go support and `ci` contract producers finish. Fresh verdicts start independently
on 8 CPU / 16 GiB once their binaries and all runtime artifacts are ready.
Compilation never invokes TestMain. The runner executes verified binaries in
the original package working directories with `-test.count=1`, `-parallel=8`,
a 40-minute package timeout, and at most four concurrent packages. Gotestsum
retains the original events and permits three retries with a 50-failure ceiling.
Native reporting consumes a compact projection that preserves every verdict and
retry event, and bounded failure/skip output. Its metadata hashes the unchanged
complete original JSON, which remains available with JUnit and per-test logs.
The runner restores default interrupt/quit signal dispositions before executing
each binary: `test2json` command mode otherwise leaks ignored signals into
subprocess fixtures. It also retains Go's one-minute backup cleanup grace.
The CLI retains `--suite go-rollup` compatibility.

[Captain's Go support](https://www.rwx.com/docs/captain/test-frameworks/go/go-test)
does not currently provide partitioning, so assignment uses the repository's
package balancer while RWX consumes native Go JSON reports.

## Dependency and cache boundaries

Local packages produce Go modules, the verified superchain ZIP, Cannon binaries
and embeds, hello ELF (Go 1.24.13), contracts with the Circle `ci` profile and
script preparation, the embedded deployer artifact archive, four Kona release
binaries, op-reth plus its SDM fixture, and all configured reproducible prestates.
The existing op-reth shadow calls the same release package.

Native Go objects, Foundry state, Cargo targets and sccache use isolated tool
cache keys. The prestate task uses `docker: preserve-data`, retaining Docker and
BuildKit data through [RWX's supported cache](https://www.rwx.com/docs/docker).
Verdicts restore dependency archives after checking revision, tool pins,
settings, archive hash and individual file hashes. Compiler metadata additionally
binds suite, shard assignment, effective settings, binary hashes, Go version and
absolute source path. Test results and logs never enter reusable outputs.
Runtime Go builds have a separate compiler cache.

Only runtime preflight and verdict tasks receive the two existing archive RPC
inputs from the locked `optimism-go-tests-rpc-shadow` vault. Repository access is
restricted to `codex/rwx-ci-pilot` and `develop`. Protected develop warming targets
producers and compilation only, executing zero tests.

## Fresh Circle comparison

Dispatch Circle with public parameter `c-go_fresh_tests: true`. Its default is
false. Setup forwards it as `c-go_fresh_tests_effective` to avoid Circle's conflict
when a nondefault setup parameter is also passed to continuation under the same
name. The shared Go runner adds `-count=1` and pins fresh benchmark concurrency
to `-parallel=8`. Normal runs retain their existing `nproc` behavior. An
exploratory fresh run exposed `nproc=32` inside the 8-CPU Circle container;
that run is not a comparable benchmark against RWX's explicit parallelism.
Circle retains tagged discovery,
complete selection, effective settings, per-node assignments, original Go JSON,
JUnit and per-test logs. Benchmark both providers at the same immutable SHA.

RWX CLI accepts `shard-total=12` or `shard-total=24`. Keep all six samples (three
warm runs of each), resources, cache classifications, queue time, setup time,
compilation, artifact transfers and fresh execution. Select the lower median
among parity-passing configurations, retaining 12 on a tie. Compare with three
fresh Circle samples measured from pipeline creation to the last Go verdict.
Do not sum concurrent task durations to claim an end-to-end speed.

## Remaining validation

- [ ] Hosted full-suite parity at the benchmark SHA, including complete original
      reports, skips, outcomes and retry histories; investigate each discrepancy.
- [ ] Relevant Go, Rust, contract, toolchain and prestate invalidation probes;
      unchanged reuse, empty-target sccache, and Circle prestate hash comparison.
- [ ] Runtime Go/Forge fixtures, RPC availability, failure reports, intentional
      fresh failure/retry/cancellation behavior, and failing shadow status.
- [ ] Three warm Circle samples, three RWX 12-shard samples, and three RWX
      24-shard samples with phase measurements and runner resources.
- [ ] Replace the narrower native rollup trigger after full-suite validation;
      retain its CLI mode and regression tests.
- [ ] Final PR checks reach terminal states; update the PR with measured results
      and explicit remaining speed bottlenecks.

Implementation alone does not establish hosted parity or a speed improvement.
