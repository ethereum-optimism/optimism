# CI Config Review

Checklist for reviewing changes to `.circleci/` and `.github/workflows/`. The
repo-specific items are the high-priority ones — they're where the real bugs
hide. For each changed file, walk the relevant items and look for the bad pattern.

## How CI is wired here

- **`config.yml` is a setup pipeline** (`setup: true`). `prepare-continuation-config`
  detects changed paths, runs a decision tree, merges the continuation fragments,
  and continues the pipeline. Only workflows whose `c-run_*` flag the tree set to
  `true` execute.
- **The real config is merged from fragments** under `.circleci/continue/`
  (`helpers.yml` → `main.yml` → `rust-ci.yml` → `rust-e2e.yml`) by
  `merge-configs.sh`. **Merge is later-wins**: a key (job, command, anchor)
  redefined in a later fragment silently overrides the earlier one.
- **Change detection**: `collect-params.sh` turns `c-*` env vars into params;
  `detect` is true if *any* changed file matches the regex, `detect_all` only if
  *every* file matches. `workflow-helpers.sh` sets the `c-run_*` flags;
  `test-decision-tree.sh` asserts the tree.
- **The gate**: the GitHub `enforce-ci-checks-develop` ruleset requires exactly
  four checks — `ci-gate`, `required-contracts-ci`, `required-rust-ci`,
  `required-rust-e2e`. These are fan-in jobs (no work, just `requires:`). A merge
  is gated *only* by what they transitively require; anything outside their
  `requires:` chain can fail without blocking merge. Gates use `utils/ci-gate`.
- **Continuation limits**: a setup pipeline continues exactly once, within 6h, no
  setup→setup. A param declared in both `config.yml` and a fragment with
  different defaults fails with "Conflicting pipeline parameters".

## Choosing where a new job runs

When a diff adds a job, the first question is cadence, not correctness. Options
here, fastest-signal/highest-cost first:

- **PR-blocking** — wired into a gate's `requires:` chain (items 1–3). Use only for
  checks that are fast, deterministic, and catch a regression class a reviewer
  can't eyeball. Every blocking job is a tax on every PR.
- **Non-blocking on PR** — runs on the PR but sits outside any gate's `requires:`.
  Treat as a staging area for a check not yet trusted to block, not a permanent
  home — non-blocking failures get ignored. Have a plan to promote it to blocking
  or move it off the PR.
- **develop-only** — `filters:` restricting to the `develop`/`main` branch; runs
  post-merge. For checks too slow, flaky, or expensive to block every PR but where
  you still want fast signal on the integration branch.
- **Scheduled** — a `scheduled_*` workflow gated on a `c-run_scheduled_*` param and
  dispatched by the schedule-name mapping in `config.yml` (`build_four_hours` /
  `build_daily` / `build_weekly`). For exhaustive/expensive suites (full Cannon,
  heavy fuzz, reproducibility, link checks). A new scheduled job not added to that
  mapping never fires — verify the wiring, not just the workflow definition.

Default heuristic: block if fast + deterministic + guards a real regression;
otherwise push to develop-only or scheduled by cost and how quickly the signal is
needed.

## Repo-specific checklist (high priority)

Each item produces a silently-green-but-untested merge — the worst failure mode
here. Treat items 1–4 as **blocking**.

1. **Gate coverage.** Any job that should gate merge must appear (by exact name,
   incl. matrix suffix like `contracts-bedrock-tests main`) in the `requires:` of
   its gate. Renaming a job silently drops it. Wire merge-queue-only jobs
   (`gh-readonly-queue`) too. Beware intermediate fan-in helpers that aren't
   themselves a required check — they look like they gate but don't.

2. **Skip paths must still emit every required check.** Required checks match by
   *name*; if a fast path skips the workflow that produces one, the check never
   reports and the PR is permanently unmergeable. A skip path must run the same
   gate job with `always-succeed: true`. Check every required check name is
   produced on every alternate path the diff adds/changes.

3. **`always-succeed` semantics.** `utils/ci-gate` defaults to
   `always-succeed: false` → it queries the API for upstream job IDs and verifies
   them. A gate with no `requires:` and no `always-succeed: true` errors out
   (`no dependency IDs found`). So: empty `requires:` ⇒ must set
   `always-succeed: true`; real `requires:` ⇒ must *not* set it (would green the
   check without verifying deps).

4. **Path filtering must be all-match, not exclusion.** Detect a limited change
   set (e.g. docs-only) with `detect_all` (true iff *every* file matches the
   narrow pattern), never by excluding known categories (`docs && !contracts &&
   !rust`) — unenumerated paths (new dirs, Go files) would slip through and skip
   real tests. Be suspicious of negative lookaheads (`^(?!...)`). Confirm
   `test-decision-tree.sh` covers the "undetected code" case.

5. **No duplicate command/anchor defs across fragments.** Later-wins merge means
   a redefinition in a later fragment silently shadows the canonical one, dropping
   its behavior with no error. `helpers.yml` is the home for shared commands. If a
   diff adds a `commands:`/`executors:`/anchor, grep the other fragments for the
   same key.

6. **Cache keys.** Shared content (dependency downloads) → one shared key, not a
   per-job prefix (avoids each job storing/re-downloading its own copy). Separate
   caches by invalidation cadence (toolchain keyed on toolchain pins, deps on
   lockfile, build output on lockfile+profile+features). Fallback `restore` keys
   are deliberate: a chain restores a near-match and recompiles the delta; no
   fallback forces a full refresh (right for download caches, so they can't
   accrete stale versions). Keys carry a version buster (`-v16-`,
   `go-cache-version`). Check `save` and `restore` keys stay consistent.

7. **Resource class / concurrency / timeouts.** Right-size `resource_class` with a
   stated reason; bound parallelism and shard memory-hungry suites rather than
   over-subscribing one runner. Set `no_output_timeout` above healthy runtime but
   tight enough to catch hangs.

8. **CI time.** Weigh what a change does to PR-path wall-clock and cost: a new
   heavy job added to a gate's `requires:`, reduced parallelism, a larger
   `resource_class`, or a removed/narrowed cache all add up. Don't eyeball it —
   capture actual before/after numbers. The reliable way is a draft PR: push the
   change, then compare job durations (and the critical-path total) against the
   base. Cite the real numbers in review rather than guessing.

9. **Validate locally.** Never `circleci config validate` a single fragment (fails
   on duplicate keys). Always produce the merged output by running the repo's own
   script — `bash .circleci/scripts/merge-configs.sh`, which writes
   `/tmp/merged-config.yml` — so you validate exactly what CI builds; don't
   re-run the `yq` merge by hand. Then validate `/tmp/merged-config.yml`, first
   stubbing the private `ethereum-optimism/circleci-utils` orb (inline orb with
   `checkout-with-mise`, `ci-gate`, `github-event-handler-setup`, `github-stale`;
   `name` is reserved). Also run `bash .circleci/scripts/test-decision-tree.sh`.
   Validation catches schema and missing-`requires:`-target errors, not semantics
   (items 1–6).

## General best practices

Repo is CircleCI-primary; GitHub Actions footprint is small but these apply there.

**Security**
- [GHA] Pin third-party actions/reusable workflows to a full commit SHA, not a
  tag/branch — tags are mutable.
- [GHA] Never interpolate untrusted `${{ github.event.* }}` (PR title/body,
  branch, commit msg) into `run:` — script injection. Pass via `env:`, use `"$VAR"`.
- [GHA] `pull_request_target`/`workflow_run` run privileged (write token +
  secrets); never check out and run PR-head code under them. Build fork code under
  plain `pull_request`.
- [GHA] Least-privilege `permissions:` — `contents: read` at top, widen per-job.
  Prefer OIDC over long-lived cloud secrets.
- [CCI] Secrets in restricted contexts, not org-wide project vars; pin orbs to an
  exact version (never `@volatile`).
- [both] Never `echo`/CLI-pass secrets; auto-redaction misses transformed values.

**Correctness**
- [both] Cache keys hash the lockfile with broader fallback `restore-keys`; a key
  with no hashed input never invalidates.
- [GHA] Required check + `paths-ignore` deadlocks the PR (check stays Pending) —
  same class as items 2–4; use an always-passing companion with the required name.
- [both] Set explicit timeouts — GHA's default job timeout is 6h.
- [GHA] `concurrency` group must include `github.workflow`; `cancel-in-progress`
  for CI, not for prod deploys.
- [GHA] Matrix `fail-fast` defaults to true; set false for full results, cap
  `max-parallel`.
- [both] Retry only genuine transients; "passes on retry" is a flake to fix.

**Maintainability**
- [both] DRY via reusable workflows/composites [GHA] or orbs/anchors [CCI].
  `secrets: inherit` passes everything — prefer explicit per-secret.
- [both] Pin runner images (`ubuntu-latest` drifts) and Docker base images by
  `@sha256:` digest.
- [GHA] `continue-on-error` / `set +e` mark steps green — branch on
  `steps.<id>.outcome`; set `shell: bash` so piped failures aren't swallowed.
