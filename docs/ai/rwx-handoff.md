# RWX migration handoff

Updated October 27, 2026. Start here when continuing the CircleCI-to-RWX
evaluation in a fresh session. Read [AGENTS.md](../../AGENTS.md), then this
document and the [evidence index](rwx-evidence-index.md). The
[operator guide](rwx-migration.md) owns implementation details; the
[coverage checklist](rwx-parity-todos.md) owns the dated workload inventory.

## Objective and decision so far

Evaluate moving Optimism's monorepo CI to RWX while preserving selection,
test execution, required checks and normal PR cache permissions. The measured
source-change samples support investing in a staged rollout, especially Rust
and Go. They do not establish whole-repository savings or production readiness.
CircleCI still owns the four required CI gates; dependency review remains in
place. No gate cutover, release migration or production publication was performed.

RWX tasks can reuse results for unchanged inputs, inherit dependency filesystems
with `use`, order work without inheriting state with `after`, narrow inputs with
filters, and recover compiler state through tool caches. This pilot separates
reusable preparation/compilation from fresh verdicts and hands test workers
smaller binary/runtime artifacts. Circle also supports dependency graphs,
caches and workspaces; several optimizations were applied to both services.

## Current state

GitHub was checked on October 27, 2026 at approximately 16:42 EDT. All five PR
heads still match the final benchmark records. **PR #23151 now conflicts with
`develop`**, despite passing checks at its unchanged head. Observed `develop` was
`8b8df208b2a2f51055f8d03e305d597d001a02d0`. These are observations, not promises
about the next checkout or merge result.

| Role | PR | Head commit | State at handoff |
| --- | --- | --- | --- |
| Migration, base `develop` | [#23151](https://github.com/ethereum-optimism/optimism/pull/23151) | `5552a61cccd569961d9ae100f896c28bc440ea9d` | Open, ready for review, conflicts with base |
| Harness, base migration | [#23533](https://github.com/ethereum-optimism/optimism/pull/23533) | `f8b1c43bfb7a489f8eb5a2aa9d2a16ea19dffa0c` | Open draft |
| Go fixture, base harness | [#23513](https://github.com/ethereum-optimism/optimism/pull/23513) | `3c99ebf34eb00d5478e2ebb50493ccb08524b8cc` | Open draft |
| Rust fixture, base harness | [#23517](https://github.com/ethereum-optimism/optimism/pull/23517) | `de695f31eb719bbc6ca924184a5dc1b0fcf29209` | Open draft |
| Solidity fixture, base harness | [#23519](https://github.com/ethereum-optimism/optimism/pull/23519) | `13a35ccd4af7a665e047af20467b3fc236513324` | Open draft |

Latest named checks reported by `gh pr checks` were 183 passing plus one neutral
for migration; 112 passing plus one skipped for the harness; and 25, 20 and 20
passing for Go, Rust and Solidity respectively. Dependency review succeeded for
all five. Raw `statusCheckRollup` also includes superseded attempts, including
an old cancelled Rust subject check; do not count those as new failures.
No CI was rerun while preparing this handoff.

Remote branches, in table order, are `codex/rwx-ci-pilot`,
`codex/rwx-source-benchmark-base`, `codex/rwx-bench-go-20261019`,
`codex/rwx-bench-rust-20261019` and `codex/rwx-bench-solidity-20261019`.
The initial handoff was captured from local branch
`codex/rwx-sdk-benchmark-go-20261021` at the Go fixture SHA. The documentation
is retained on `codex/rwx-handoff-docs-20261027`, based on migration head
`5552a61cccd569961d9ae100f896c28bc440ea9d`, with only four documentation files
changed and no PR opened. Other local worktrees and branch refs may be stale;
inspect them before using them.

The fixture PRs are benchmark evidence, not production changes to merge.
The shared harness advanced after the Rust/Solidity measurements. Their frozen
commits remain the source of truth even though their PR base branch has moved.
Preserve those commits and provider run identities when updating migration code.

## What the benchmarks establish

| Selected workload | CircleCI | RWX | Less wall time | Qualification |
| --- | ---: | ---: | ---: | --- |
| Go | 17m01s | 13m36s | 20.1% | New source edit after verified warm parent |
| Rust | 17m35s | 7m03s | 59.9% | New source edit after verified warm parent |
| Solidity | 15m35s | 9m56s | 36.3% | Normal PR permissions; caches not uniformly warm |

The [evidence index](rwx-evidence-index.md#representative-source-change-benchmarks)
links the exact runs, source edits, warm parents and scope. Wall time starts at
provider pipeline/run creation/start and ends at selected verdict completion,
including prerequisites, queueing, setup and transfers. It excludes unrelated
checks and native GitHub status-publication delay. Go covers the full 12-worker
Go job and seven prerequisites; Rust covers workspace tests, beacon, doctests
and ten feature partitions; Solidity covers the standard `main` contract suite.
Rust feature checks compile test configurations; they do not execute tests.

Each language has one measured sample on its own frozen revision. Advertised
CPU/RAM match per build/verdict phase; hardware, OS, storage, usable memory and
peak aggregate concurrency can differ. Tests execute freshly on both services.
Solidity's Circle queue contributed about 170 seconds. Cache telemetry does not
measure equivalent total network traffic: parallel totals are not critical-path
time, and RWX log counters can omit lazy filesystem reads.

Go and Rust verify warmed parent reuse, not prewarming the changed commit.
The compiler checkpoints match the parent's tracked Go source files / Rust
tree; broader accumulated source-only deltas are not established by these
samples. The earlier Go comparison with SDK downloads is diagnostic history,
not the final warm-cache result. Cost analysis is private and modeled rather
than verified invoice savings.

## Important fixes already implemented

| Area | Change and reason | Entry point |
| --- | --- | --- |
| Rust timestamps | Restore unchanged source mtimes at whole-second precision; stamp changed inputs into the next second and wait past that stamp before Cargo. Prevents false freshness after filesystem timestamp quantization. | [rust-target-cache.py](../../ops/ci/runtime/rust-target-cache.py) |
| Rust compiler reuse and handoffs | Reusable per-partition compiler producers feed fresh consumers; unit workers receive execution archives, while beacon/doctests retain required compiler state. Avoid passing full targets to every test worker. | [rust.yml](../../.rwx/rust.yml), [rust-workspace.sh](../../ops/ci/runtime/rust-workspace.sh) |
| System-file read timeouts | Materialize host linker inputs into fresh inodes with integrity checks and bounded retries of timed-out reads. Provider-side root cause remains unconfirmed. | [rwx-materialize-linker.py](../../ops/ci/runtime/rwx-materialize-linker.py) |
| Contract RPC 502s | Shared relay bounds concurrency and retries transient HTTP failures; nested shells preserve the relay environment. Permanent/exhausted failures still fail tests, and original attempts are retained. | [l2-rpc-proxy.py](../../ops/ci/runtime/l2-rpc-proxy.py), [contracts-test-env.sh](../../ops/ci/runtime/contracts-test-env.sh) |
| Go dependency reuse | Reuse source ancestors only after build-input identity checks; preserve current-source discovery after layer assembly; inherit modules and pass sealed runtime assets. | [go-dependency-source.py](../../ops/ci/runtime/go-dependency-source.py), [go-compiled-tests.py](../../ops/ci/runtime/go-compiled-tests.py) |
| Go test splitting | Share a 12-worker plan across providers and divide the slow deployer CLI package into four top-level groups without dropping tests. | [go-package-shards.py](../../ops/ci/runtime/go-package-shards.py), [go-test-names.go](../../ops/ci/runtime/go-test-names.go) |
| Circle Go cache correctness | Align `GOMODCACHE` with the restored directory across executors; include `go.mod`, `go.sum` and `mise.toml` in the immutable module key; retain per-node compiler caches. | [CI operations](ci-ops.md#go-module-cache-on-circle-machines) |

The latest migration head also includes a `develop` merge and refreshed generated
fetcher artifacts. Do not undo Circle cache fixes to make a comparison look better.

RWX support reported raising this organization's filesystem-layer limit to
200 GiB during the pilot. Older 100 GiB references in the operator guide describe
earlier incidents; the current account limit was not rechecked for this handoff.
Support also recommended binary/runtime artifacts and acknowledged ongoing
permission/hardlink work. Keep those open operational questions distinct from
the timestamp and local materialization fixes already validated.

## Next steps, in order

1. **Recheck state and preserve evidence.** Read the index, inspect local changes,
   and fetch current PR heads. The documentation branch preserves this handoff;
   use an appropriate migration checkout for implementation. Keep benchmark
   fixtures frozen; do not rerun experiments merely to reconstruct context.
2. **Resolve migration drift.** Reconcile #23151 with current `develop` in an
   isolated checkout. Read the CI review guidance, inventory changed workflows,
   gates and matrices, run affected validation, then watch checks and dependency
   review to terminal states on the resulting SHA. Historical green checks do
   not validate that new merge.
3. **Refresh coverage before claiming completion.** The 86/86 baseline is an
   October 1 inventory with later original-report parity. The October 19 guide
   mentions 88 occurrences, but later upstream jobs and variants need a new
   inventory, including SP1 smoke/action jobs. Audit current gate dependencies,
   routing, feature matrices, fork authorization, merge queue, denied cache
   access, schedules and publication boundaries.
4. **Plan a bounded shadow rollout.** Start with Rust and Go, retain Circle as
   the required provider, and observe ordinary PRs for latency, cold fallbacks,
   flakes, cancellation and artifact completeness. Define the acceptance and
   rollback criteria before changing required-gate ownership.
5. **Close commercial and operational questions privately.** Reconcile billed
   usage with the models, confirm evidence retention/backups, and follow up on
   filesystem reliability and permission/hardlink behavior. Do not publish raw
   reports, credentials, pricing details or signed download URLs.

The requested engineering report and copyable Slack draft were delivered in the
conversation. A slide deck was requested but that turn was interrupted before
creation; it is not an existing deliverable.

## Resume safely

Read [CI operations](ci-ops.md), [CI review](ci-config-review.md), and the
[operator guide](rwx-migration.md) before changing pipeline behavior. Inspect
the current checkout and current remote state with read-only commands:

```bash
git status --short --branch
git rev-parse HEAD
git worktree list
gh pr view 23151 --repo ethereum-optimism/optimism \
  --json headRefName,headRefOid,baseRefName,state,isDraft,mergeable
gh pr checks 23151 --repo ethereum-optimism/optimism
```

The installed GitHub CLI supports the commands above but lacks
`gh pr checks --json`. Use `gh pr view --json statusCheckRollup` for structured
observations and account for superseded attempts. The connector lacked org SSO
access during handoff; the existing CLI authentication worked.

Use existing authentication without copying token values into files or chat.
CLI RPC-vault access is separate from tool-cache write permission; recheck both
before any new hosted run. Normal PRs must not gain protected cache writes or
spoof `develop`. This handoff does not renew vault grants or run CI.

The private companion index holds machine paths, collection scripts, original
checksums, billing methodology and raw evidence. It is outside Git and requires
operator access. Public PRs should retain source/run links, validation summaries
and caveats. A new session should begin by reporting changes since this snapshot
and the next required action, without treating this document as live CI state.
