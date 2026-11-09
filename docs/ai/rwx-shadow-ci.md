# RWX shadow CI

RWX (rwx.com) runs an optional shadow of the CircleCI PR workloads so the team can
compare the two providers before any decision to migrate. CircleCI stays the only
required CI: it owns `ci-gate`, `required-contracts-ci`, `required-rust-ci` and
`required-rust-e2e`, and nothing in RWX gates a merge. RWX changes nothing that
CircleCI runs.

## When RWX runs

| Trigger | Definition | Condition |
| --- | --- | --- |
| GitHub push | `.rwx/pr-gates.yml`, `.rwx/rust-e2e.yml`, `.rwx/selector-upload.yml` | Any branch push to this repository; tag pushes start no run |
| Cache rebuild | Every definition with `on.cache-rebuild` | Branch is `develop`; RWX starts these to re-warm caches |
| CLI | Every definition | An operator runs `rwx run` |

Like CircleCI, RWX runs on every branch push, so a PR from a branch of this
repository gets RWX's optional checks next to CircleCI's. RWX routes each push
with CircleCI's own routing scripts, so it selects the same workloads. A new push
to a branch cancels the older RWX run on that branch; `develop` runs every
commit.

Fork PRs cannot push to this repository, so they never start a run. CircleCI
runs them only after a maintainer comments `/ci authorize`; RWX has no
equivalent, so fork PRs get no RWX checks. Never push a commit you have not
reviewed, such as a fork PR's head, to a branch of this repository: RWX runs it
with that branch's vault access.

RWX reads the definitions from the pushed commit. A trigger change therefore
takes effect for the next push that contains it.

## Turning RWX off

- **From the repository:** set the `if:` of the three `on.github.push` triggers
  to `false`, or delete the `github:` blocks, and merge. Later pushes of branches
  that contain the change start no runs; older branches keep their old
  triggers until they merge develop. The `cache-rebuild` triggers stop being useful without pushes;
  delete them too for a full stop.
- **Immediately, without a merge:** a GitHub organization owner suspends the RWX
  GitHub App installation, or removes this repository from its access list.
- **Permanently:** delete `.rwx/`, `ops/ci/`, this guide and the `/.rwx/` and
  `/ops/ci/` lines in `.github/CODEOWNERS`. Nothing else depends on them.

## Layout

| Path | Content |
| --- | --- |
| `.rwx/pr-gates.yml` | Coordinator. Calls one definition per workload, then mirrors the CircleCI gates as optional checks |
| `.rwx/*.yml` | One definition per workload. `go-rollup.yml` is CLI-only |
| `.rwx/packages/` | Reusable compile and verdict packages that the definitions call |
| `ops/ci/runtime/` | Scripts that RWX tasks run, the gate policy `pr-gates.json` and timing seeds |
| `ops/ci/tests/` | Unit tests for `ops/ci/runtime/` |
| `ops/ci/migration/` | Transitional tools tied to CircleCI: gate alignment, the L2 chains sync check, the CI result comparer and the private selector-upload rehearsal |

## Routing

`ops/ci/runtime/rwx-metadata.sh` selects workloads with CircleCI's own routing
data and policy (`.circleci/routing.yml`,
`.circleci/scripts/collect-params.sh` and
`.circleci/scripts/compute-workflow-conditions.sh`), which it runs read-only. The
two providers therefore select the same workloads for the same commit. RWX adds
one rule: a change under `.rwx/` or `ops/ci/`, or an uncommitted change uploaded
by `rwx run`, selects every workload. On `develop` every workload runs, as on
CircleCI. Branches compare against `develop`.

## Gates

`ops/ci/runtime/pr-gates.json` lists the prerequisites of `ci-gate`,
`required-contracts-ci` and `required-rust-ci`, and the RWX task that shadows
each one. `ops/ci/migration/circle-alignment.py`, which runs in RWX's
`ci-unit-tests` task, fails when one of those gates' `requires:` lists no longer
matches. After a CircleCI gate change, update `pr-gates.json`.

`required-rust-e2e` is shadowed by `.rwx/rust-e2e.yml`, whose job list lives in
`JOBS` in `ops/ci/runtime/rust-e2e.py`. The same alignment check fails when
CircleCI's `required-rust-e2e` stops requiring exactly its release builds
(`release_jobs` of `e2e_gate` in `ops/ci/migration/circle-gates.json`) and the
`circle_job` of each `JOBS` entry. CircleCI splits the release workspace build
across two jobs; RWX builds the whole workspace in one task. After a CircleCI E2E change, update `JOBS` and
`.rwx/rust-e2e.yml`.

The RWX checks (`optimism-rwx-pr-coordinator` and the `optimism-*-shadow`
checks) are not required by any ruleset.

## Caches and credentials

- Tool caches use the `optimism-op-reth-shadow` vault. Only
  `refs/heads/develop` may write. Other branches restore the `develop` baseline
  read-only and compile their own changes.
- Go and contract fork tests read archive RPC URLs from the test-only
  `optimism-go-tests-rpc-shadow` vault. A branch needs a grant on this vault
  for those tasks to run. In a grant's branch pattern, `*` matches one path
  segment and `**` matches nothing, so the vault grants `refs/heads/*`,
  `refs/heads/*/*` and `refs/heads/*/*/*` (merge-queue branches have two
  slashes). A branch with more slashes gets no access.
- RWX holds no production publishing credentials. The selector-upload workload
  publishes to a private registry that the run starts itself.

Check the grants with `rwx vaults access list --vault <name> --json`.

## Comparing with CircleCI

RWX does not change CircleCI, so CircleCI produces only its normal artifacts.
Compare RWX results with CircleCI's test results for the same commit.
`ops/ci/migration/compare-ci.py` normalizes a collection file for each provider
and compares the two. A collection names the commit, branch, workload and
features, and lists report files in one of three formats: `junit`, `go-json`
(`go test -json`, as gotestsum retains it) or `circleci-tests` (the response of
CircleCI's job test-metadata API). `ops/ci/migration/tests/test_compare_ci.py`
has complete examples.

```bash
python3 ops/ci/migration/compare-ci.py normalize --input circle.json --output circle.normalized.json
python3 ops/ci/migration/compare-ci.py normalize --input rwx.json --output rwx.normalized.json
python3 ops/ci/migration/compare-ci.py compare --baseline circle.normalized.json \
  --candidate rwx.normalized.json --output report.json --markdown report.md
```

Compare wall time and cost per workload, including setup and cache transfer,
rather than per test.

## Known differences from CircleCI

RWX calls the unchanged `just` recipes and scripts that CircleCI calls. These
differences remain:

- **Acceptance sharding.** CircleCI splits tests across shards by historical
  timing (`circleci tests split` in `ops/scripts/shard-tests.sh`). RWX puts
  `ops/ci/runtime/acceptance-circleci/circleci` first on `PATH`; it answers that
  call with a deterministic assignment from RWX's own test discovery, and fails
  when the recipe lists different tests.
- **Go tests.** RWX compiles test binaries in one task and runs them in a fresh
  verdict task with `-count=1`, so it never reuses a cached test result.
- **cargo-hack partitions.** RWX seeds the package shuffle with the hash of
  `rust/Cargo.lock` instead of the CircleCI workflow ID, so each partition
  checks different crates. All partitions together check the same set.
- **NUT provenance.** RWX verifies every fork; CircleCI verifies only forks whose
  lock entry changed against the PR base.
- **NUT pre-fork, Kontrol.** RWX keeps the recipe's exit status and logs, not a
  per-test report.
- **Flaky-test report.** Not shadowed: it reads CircleCI's Insights API, which
  needs a CircleCI token.
- **Selector upload.** RWX runs the real publisher against a private registry
  that the run starts, never the production one.

## Validating a change

RWX definitions must lint without errors. A warning that a pinned package has a
newer version is informational; bump packages in their own change.

```bash
rwx lint --dir .rwx
```

Unit tests (Python 3.12+, with `yq`, `jq` and `git` on `PATH`; `mise exec yq jq --`
provides the pinned versions):

```bash
python3 -m unittest discover -s ops/ci/tests -p 'test_*.py'
python3 -m unittest discover -s ops/ci/migration/tests -p 'test_*.py'
python3 ops/ci/migration/circle-alignment.py
```

In RWX, each workload's `helper-tests` task runs that workload's tests with its
toolchains, and `ci-unit-tests` in `pr-gates.yml` runs the shared ones. A change
under `.rwx/` or `ops/ci/` selects every workload, so all of them run.
`test_ci_layout.py` fails when no task runs a test file; add a new test file to
its workload's `helper-tests`. Tests that need a live toolchain
(`RWX_LIVE_*_FIXTURE=1`) skip locally unless you set the variable.

A hosted run costs money. An authenticated operator can start one from a pushed
commit with `rwx run .rwx/pr-gates.yml --wait`; a branch push also starts one.
