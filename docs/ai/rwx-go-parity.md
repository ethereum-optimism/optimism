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
script preparation, the embedded deployer artifact archive, Kona host/client/node and op-zk-proposer release
binaries, plus op-reth, and all configured reproducible prestates.
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
name. The shared Go runner adds `-count=1` while retaining Circle's existing
`nproc` concurrency. An exploratory fresh run exposed `nproc=32` inside the
8-CPU Circle container. RWX concurrency is independently tunable through
`test-parallel` (initially 8), with 16 and 32 as candidates on the same
8 CPU / 16 GiB verdict workers. Preserve the complete workload, fresh execution,
timeout and retry limits; record execution-setting differences explicitly.
Circle retains tagged discovery,
complete selection, effective settings, per-node assignments, original Go JSON,
JUnit and per-test logs. Benchmark both providers at the same immutable SHA.

## Stage closeout (October 2, 2026)

The selected configuration is **12 duration-balanced shards, `-parallel=8`,
`-p=4`, 16 CPU / 32 GiB compilation and 8 CPU / 16 GiB verdict workers**.
The full native rollup trigger is retired; its CLI mode and regression coverage
remain. Circle continues to own required gates.

Same-revision fresh parity is verified at
`cf7f3f2d51ea5e75b9eb21adcb4cf7a860a15bce` between
[RWX](https://cloud.rwx.com/optimism/runs/fee5ce5d0613477397c69755dded1950)
and [Circle job 5629727](https://circleci.com/gh/ethereum-optimism/optimism/5629727).
Both selected all 459 packages exactly once, including packages without tests,
and reported 11,613 identities: 11,490 passes, 123 skips and zero retries.
No identities are missing or extra. The strict comparison reports seven differing
skip messages in flaky-handling self-tests. Each has identical annotations and
source-relative traces; provider log routing and absolute workspace paths account
for the differences. All seven are investigated, with zero unresolved differences.
The retained [comparison evidence](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-go-evidence/parity.json) records each resolution;
it does not change the original reports or claim strict textual equivalence.

| Observation at that SHA | Configuration | Wall time | Measurement boundary |
| --- | --- | ---: | --- |
| RWX native default | 12 shards, parallel 8 | 1,075s | RWX run start through completion |
| RWX concurrency trial | 12 shards, parallel 32 | 1,054s | RWX run start through completion |
| Circle fresh validation | 12 nodes, effective parallel 32 | 1,222.7s | Pipeline creation through final Go job |

The [default run](https://cloud.rwx.com/optimism/runs/3fbfa393c684449eb73f3d6372943f5f)
passed all 11,613 cases. These are individual observations with different provider
creation boundaries and variable producer cache state, not repeated warm medians
or a proven speed win. The 21-second concurrency difference does not establish
a repeatable improvement at no greater whole-run cost. At the user's request,
performance experimentation ends here: repeated three-run benchmarks and 24-shard
selection are deferred. The CLI retains both shard configurations for later use.

Validated evidence includes:

- Full hosted runtime Go/Forge fixtures and RPC preflight, alongside helper tests
  for unavailable RPC inputs, stale settings/SHA, corrupt binaries, discovery,
  assignments, runtime paths, report collection, signals and cancellation.
- [Isolated intentional failure](https://cloud.rwx.com/optimism/runs/c8144a8d36614520a3274f78d50a78b0):
  initial execution plus three fresh retries, original failures and a failed CLI
  run. A native GitHub webhook failure rehearsal remains an operational follow-up.
- [Empty-target Kona sccache](https://cloud.rwx.com/optimism/runs/ea88798123de459686653de6b95e96d4):
  1,568 Rust hits and eight misses; complete statistics retain other compiler
  feature probes and cache errors rather than presenting them as all hits.
- Isolated dirty-snapshot invalidation probes for
  [Go/prestate](https://cloud.rwx.com/optimism/runs/5e980f4644f84328b2cc4b89e9ff070e),
  [Rust/prestate](https://cloud.rwx.com/optimism/runs/b9569fc31bcb48958df03508c5240e59),
  [contracts](https://cloud.rwx.com/optimism/runs/47b445bedeb44b3891cae01ed00e6910), and
  [toolchain](https://cloud.rwx.com/optimism/runs/91ce4d1927294bea9e4dbd912c589f47).
  These ran zero tests and are cache diagnostics, not clean-SHA benchmarks.
- Unchanged Go/contract artifact reuse, native compiler caches, and zero-test
  warming rehearsals. An actual protected `develop` cache-rebuild event requires
  merge and remains unobserved.
- Both configured prestate hashes matched Circle job 5629322:
  `kona-client=0x03e384aad91052e86a9912ad763cd32027586d99b5e8e2024115c606f35b6afa`,
  `kona-client-int=0x03d56f7fd7d39b381efc142e127f41109709812d788c436ebad8f6de0633bc39`.

Original JSON, JUnit, per-test logs, effective settings and manifests remain in
provider artifacts and retained comparison archives. The checked-in comparison
is a compact evidence index; provider retention is finite. Preserve downloaded
archives before provider retention expires when auditing or resuming benchmarks.

Follow-up work is acceptance-suite parity, operational webhook/gate rehearsals,
and optional performance work. Current bottlenecks are the longest Go packages
and Docker prestate cache transfer (about 5.6 GB and 138–143 seconds despite an
approximately two-second warm build). Full producer output reuse across CLI/native
contexts also needs investigation before making stronger cache or cost claims.
