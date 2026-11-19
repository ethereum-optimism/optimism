# RWX shadow CI

RWX (rwx.com) runs an optional shadow of the CircleCI PR workloads so the team can
compare the two providers before any decision to migrate. CircleCI stays the only
required CI: it owns `ci-gate`, `required-contracts-ci`, `required-rust-ci` and
`required-rust-e2e`, and nothing in RWX gates a merge. RWX changes nothing that
CircleCI runs.

## When RWX runs

| Trigger | Definition | Condition |
| --- | --- | --- |
| GitHub push | `.rwx/pr-gates.yml` | Any branch push to this repository; tag pushes start no run |
| Cache rebuild | Every definition with `on.cache-rebuild` | Branch is `develop`; RWX starts these to re-warm caches |
| CLI | Every definition | An operator runs `rwx run` |

Like CircleCI, RWX runs on every branch push, so a PR from a branch of this
repository gets RWX's optional checks next to CircleCI's. A new push to a branch
cancels the older RWX run on that branch; `develop` runs every commit.

Fork PRs cannot push to this repository, so they never start a run. CircleCI
runs them only after a maintainer comments `/ci authorize`; RWX has no
equivalent, so fork PRs get no RWX checks. Never push a commit you have not
reviewed, such as a fork PR's head, to a branch of this repository: RWX runs it
with that branch's vault access.

RWX reads the definitions from the pushed commit. A trigger change therefore
takes effect for the next push that contains it.

## Turning RWX off

- **From the repository:** set the `if:` of the `on.github.push` trigger in
  `.rwx/pr-gates.yml` to `false`, or delete the `github:` block, and merge. Later
  pushes of branches that contain the change start no runs; older branches keep
  their old trigger until they merge develop. The `cache-rebuild` triggers stop being useful without pushes;
  delete them too for a full stop.
- **Immediately, without a merge:** a GitHub organization owner suspends the RWX
  GitHub App installation, or removes this repository from its access list.
- **Permanently:** delete `.rwx/`, `ops/ci/` and this guide, and remove the
  guide's index line from `AGENTS.md`. Nothing else depends on them.

## Layout

| Path | Content |
| --- | --- |
| `.rwx/pr-gates.yml` | Coordinator. Calls one definition per workload, then mirrors the CircleCI gates as optional checks |
| `.rwx/*.yml` | One definition per workload |
| `.rwx/packages/` | Reusable tool, build and verdict packages that the definitions call |
| `ops/ci/runtime/` | Scripts that RWX tasks run, the gate policy `pr-gates.json` and the Go test timing seed |
| `ops/ci/tests/` | Unit tests for `ops/ci/runtime/` |
| `ops/ci/migration/` | Transitional tools tied to CircleCI: the gate alignment check and the CI result comparer |

A task that mirrors a CircleCI job runs the command of that job, and its exit
status is the verdict. Follow that shape for a new task: `run:` the recipe, add
the environment that the CircleCI job sets, and name the job in a comment. Do
not wrap the command in a script that re-checks what the recipe or RWX already
guarantees. "Known differences" lists where a task departs from this.

## Routing

`ops/ci/runtime/rwx-metadata.sh` selects workloads with CircleCI's own routing
data and policy (`.circleci/routing.yml`,
`.circleci/scripts/collect-params.sh` and
`.circleci/scripts/compute-workflow-conditions.sh`), which it runs read-only.
RWX adds one rule: a change under `.rwx/` or `ops/ci/`, or an uncommitted change
uploaded by `rwx run`, selects every workload. On `develop` every workload runs,
as on CircleCI.

RWX compares every branch with `develop`. CircleCI compares a PR with its target
branch. The two providers therefore select the same workloads for a PR into
`develop`; for a PR into a release branch, RWX can select more.

## Gates

`ops/ci/runtime/pr-gates.json` lists the prerequisites of `ci-gate`,
`required-contracts-ci`, `required-rust-ci` and `required-rust-e2e`, and the RWX
tasks that mirror each one. `ops/ci/migration/circle-alignment.py`, which runs in RWX's
`ci-unit-tests` task, fails when one of those gates' `requires:` lists no longer
names the same jobs, and prints each difference. After a CircleCI gate change,
update `pr-gates.json`.

Each workload has a receipt task (`ops/ci/runtime/pr-gate.py receipt`). It
passes if the route selected the workload and every mirrored task succeeded, or
the route did not select it and every mirrored task was skipped. The
coordinator has one status task per gate (`pr-gate.py status`). It passes if
every receipt of the gate succeeded, and it records each CircleCI prerequisite
with its outcome. `ops/ci/tests/test_pr_gate.py` fails when `pr-gates.json` and
the definitions name different tasks.

CircleCI splits the release build that `required-rust-e2e` requires across two
jobs, so that the E2E jobs wait only for the binaries they run. RWX has the
same two tasks: `binaries`, and `release`, which starts from the output of
`binaries` and builds the rest. `JOBS` in
`ops/ci/runtime/rust-e2e.py` holds the package, shard count and timeout of each
E2E suite. After a CircleCI E2E change, update it together with `pr-gates.json`
and `.rwx/rust-e2e.yml`.

The RWX checks (`optimism-rwx-pr-coordinator` and the `optimism-*-shadow`
checks) are not required by any ruleset.

## Caches and credentials

- Tool caches use the `optimism-op-reth-shadow` vault. Only
  `refs/heads/develop` may write. Other branches restore the `develop` baseline
  read-only and compile their own changes.
- The Go tests and the L1 contract upgrade tests read archive RPC URLs from the
  test-only `optimism-go-tests-rpc-shadow` vault. A branch needs a grant on this
  vault for those tasks to run. In a grant's branch pattern, `*` matches one path
  segment and `**` matches nothing, so the vault grants `refs/heads/*`,
  `refs/heads/*/*` and `refs/heads/*/*/*` (merge-queue branches have two
  slashes). A branch with more slashes gets no access.
- RWX holds no production publishing credentials.

Check the grants with `rwx vaults access list --vault <name> --json`.

## Comparing with CircleCI

RWX does not change CircleCI, so CircleCI produces only its normal artifacts.
Compare RWX results with CircleCI's test results for the same commit and
workload. `ops/ci/migration/compare-ci.py` takes the report files of each
provider, one per shard, and prints the test cases that are missing from the
candidate, extra in it, or different:

```bash
python3 ops/ci/migration/compare-ci.py --baseline circle-tests.json \
  --candidate rwx-shard-0.xml rwx-shard-1.xml
```

A report is JUnit XML, `go test -json` events (as gotestsum retains them), or
one complete response of CircleCI's job test-metadata API. That API returns a
large job in pages; join the `items` of every page first. The exit status is 0
for equivalent, 1 for different, 2 for rejected input and 3 when the two sides
have no case in common. The tool does not check that both sides are from the
same commit. Nothing collects the report files yet: download them by hand.

The lint, check and build jobs publish no per-test results on either provider.
For those, compare the task's command and its conclusion.

Compare wall time and cost per workload, including setup and cache transfer,
rather than per test.

## Speed and cost

RWX bills each vCPU by the second. A larger runner costs the same for the same
work only when the task uses its CPUs. Choose a runner size, a shard count and
a cache key for RWX from a measurement on RWX. A CircleCI resource class or
`parallelism` value is not a default. The largest runner that this RWX
organization can start has 16 CPUs.

- **Dependency builds.** `ops/ci/runtime/go-dependency-source.py plan` names,
  for each dependency, the commit that last changed its build inputs. The
  dependencies are the Go dependency, the contracts, kona, op-reth, the SP1
  executor, the prestate and the Go compiler seed. Every workload passes that
  commit and the same tool tasks to the build package (`.rwx/packages/go-*.yml`),
  and the build checks the commit out itself. The build then has one cache key:
  a later commit that leaves the inputs unchanged reuses it, and so do the other
  workloads of the run. A task passes on every layer that it uses, and the tool
  tasks carry the checkout of the run. So the build has an input `filter` that
  leaves it only `mise.toml` and its own outputs, and it writes its checkout
  itself: the run's checkout stays out of the build's cache key, and the build's
  checkout stays out of its filtered output. The Go compiler seeds work the same
  way. Each tool package has an input `filter` for the same reason: a tool task
  with the checkout in its cache key runs again for each commit, and gives each
  task that uses it a new cache key.
  Before a task restores an output, `go-artifacts.py` checks that the inputs
  equal those of the task's commit, and that the task's checkout has no
  modified input. `SCOPES` in the script lists the inputs; add a file there when
  a build starts to read it. Of `ops/ci/runtime`, a scope names only the scripts
  that the build runs, and `test_go_dependency_source.py` fails when a script
  that they reach is missing. `.rwx` and the tool scripts are not inputs,
  because RWX's own cache key covers a task's definition and its tool tasks.
  `test_rwx_local_packages.py` fails when a workload builds a dependency in
  another way.
- **Rust baselines.** `ops/ci/runtime/rust-target-cache.py base` names the
  commit in the same way for the nextest archive and the cargo-hack partitions.
- **Runner sizes.** A task on the critical path of a gate gets the size at
  which it stops scaling; a task off that path stays small. Measured: clippy
  takes 9:12 on 4 CPUs and 2:50 on 16, udeps 9:16 and 2:43, the workspace build
  9:57 on 8 CPUs and 6:14 on 16, and the slowest cargo-hack baseline 31:08 on 4
  CPUs and 17:52 on 16. no-std is no faster on 16 CPUs than on 8. The L2 fork
  compile spends eight of its nine minutes in one solc process, so it has 4
  CPUs.
- **Floors.** More shards do not shorten the main gate: one acceptance test,
  `TestExecuteStep_CannonKona`, runs for six minutes, after each shard has
  listed the tests for one minute. The four longest Go test packages each have
  a shard to themselves. The contracts gate waits for the L2 fork compile and
  then for the paced fork tests.

## Known differences from CircleCI

- **Skipped jobs.** CircleCI halts some jobs unless their paths changed
  (`check-changed`): Cannon Go, the Kontrol build, NUT provenance, the fast
  contract checks, both generated-mock checks and the forge version check. RWX
  runs them whenever their workflow is selected.
- **Base branch.** RWX uses `develop` for routing and for the op-reth codec
  baseline. The fast contract checks find a merge-queue base, and otherwise use
  `develop`. CircleCI uses the PR's target branch.
- **Go tests.** RWX compiles the test binaries in one task and runs them in a
  fresh verdict task with `-count=1`, with the flags of `just go-tests-ci`. It
  places packages on shards with the committed timings in
  `ops/ci/runtime/go-tests-timings.json`; CircleCI uses its own timing history.
- **Rust E2E.** RWX compiles each suite once and runs the binaries with the
  flags of the recipes. It shards tests by count.
- **Rust unit and op-reth integration tests.** RWX builds a nextest archive in
  one task and runs it in another, with the recipe's test filter.
- **Acceptance sharding.** CircleCI splits tests across shards by historical
  timing (`circleci tests split` in `ops/scripts/shard-tests.sh`). RWX puts
  `ops/ci/runtime/acceptance-circleci/circleci` first on `PATH`; it gives each
  shard the lines of the recipe's test list whose position, modulo the shard
  count, is the shard index.
- **Contract tests.** RWX compiles in a task of its own before the test recipe.
  On `develop`, CircleCI shards the standard suite; RWX does not.
- **L2 fork tests.** CircleCI forks a private RPC. RWX has no such URL in a
  vault, so it forks the public `https://mainnet.optimism.io` through
  `ops/ci/runtime/rpc-pacer.py`, which starts at most two requests per second.
  At forge's own rate the public RPC answers HTTP 429 for longer than forge
  retries. The recipe and its arguments are CircleCI's. To remove the pacer,
  add the private URL to an RWX vault and pass it as `L2_FORK_RPC_URL`.
- **cargo-hack partitions.** RWX seeds the package shuffle with the hash of
  `rust/Cargo.lock` instead of the CircleCI workflow ID, so each partition
  checks different crates. All partitions together check the same set.
- **NUT provenance.** RWX verifies every fork, each with the forge and just
  versions pinned at the fork's recorded commit. CircleCI verifies only forks
  whose lock entry changed against the PR base.
- **SP1 guest.** RWX builds the ELFs in a clone at CircleCI's paths
  (`/home/circleci/project`), so the bytes can match CircleCI's.
  `ops/ci/runtime/sp1-guest-toolchain.sh` is a hand copy of CircleCI's
  toolchain command; nothing compares the two.
- **Flaky-test report.** Not shadowed: it reads CircleCI's Insights API, which
  needs a CircleCI token.
- **Selector upload.** Not shadowed: the CircleCI job publishes to the public
  selector registry, and no gate requires it.

## Validating a change

RWX definitions must lint without errors. A warning that a pinned package has a
newer version is informational; bump packages in their own change.

```bash
rwx lint --dir .rwx
```

The lint does not check the order of a `use` list. Name a task after the tasks
that it uses itself. RWX applies a task above the tasks it uses whatever the
list says, so another order has no effect, and each run prints a warning for
the task.

Write a `test-results` path without a `${{ }}` expression: a literal path or a
glob. RWX does not expand an expression there, finds no file, and reports no
tests without a warning. It does expand one in an `artifacts` path.

Unit tests (Python 3.12+, with `yq`, `jq`, `just` and `git` on `PATH`;
`mise exec yq jq just --` provides the pinned versions):

```bash
python3 -m unittest discover -s ops/ci/tests -p 'test_*.py'
python3 -m unittest discover -s ops/ci/migration/tests -p 'test_*.py'
python3 ops/ci/migration/circle-alignment.py
```

In RWX, a workload's `helper-tests` task runs the tests of that workload's
scripts, and `ci-unit-tests` in `pr-gates.yml` runs the shared ones. A workload
whose tasks run recipes directly has no scripts and no `helper-tests`. A change
under `.rwx/` or `ops/ci/` selects every workload, so all of them run.
`test_ci_layout.py` fails when no task runs a test file; add a new test file to
its workload's `helper-tests`. Tests that need a live toolchain
(`RWX_LIVE_*_FIXTURE=1`) skip locally unless you set the variable.

Several task families in the definitions are YAML merges: the first task of the
family has an anchor, and each later task lists `<<: *anchor` first and then
only the fields that differ.

A hosted run costs money. An authenticated operator can start one from a pushed
commit with `rwx run .rwx/pr-gates.yml --wait`; a branch push also starts one.
Either one cancels a run of `pr-gates.yml` that is still in progress for the
same branch, even when the new run names only some tasks with `--target`.
