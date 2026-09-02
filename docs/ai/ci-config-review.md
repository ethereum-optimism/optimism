# CI Config Review

Checklist for reviewing changes to `.circleci/` and `.github/workflows/`. The
repo-specific items are the high-priority ones — they're where the real bugs
hide. For each changed file, walk the relevant items and look for the bad pattern.

## How CI is wired here

- **`config.yml` is a setup pipeline** (`setup: true`). `prepare-continuation-config`
  detects changed paths, runs the routing policy (`compute-workflow-conditions.sh`),
  merges the continuation fragments, and continues the pipeline. Only workflows whose
  `c-run_*` flag the policy set to `true` execute. It also front-loads the mise
  toolset (`utils/install-mise`): it always finishes before any continuation job
  starts, so on a cold cache it is the only job that installs over the network —
  continuation jobs restore the mise cache it saved.
- **Routing is data + logic split**: `routing.yml` holds the declarative data
  (schedule→workflows, API dispatch flag→workflows, change-detection patterns,
  passthrough params); `compute-workflow-conditions.sh` holds the conditions that
  decide which entries fire. Add a schedule/dispatch/pattern by editing `routing.yml`.
- **The real config is merged from fragments** under `.circleci/continue/`
  (`helpers.yml` → `main.yml` → `rust-ci.yml` → `rust-e2e.yml` →
  `rust-nightly-bump.yml`) by `merge-configs.sh`. **Merge is later-wins**: a key
  (job, command, anchor) redefined in a later fragment silently overrides the
  earlier one.
- **Change detection**: `collect-params.sh str`/`bool` turn `c-*` env vars into
  params; `detect`/`detect_all` match the `routing.yml` change patterns against the
  changed files (`detect` true if *any* file matches, `detect_all` only if *every*
  file matches). `workflow-helpers.sh` sets the `c-run_*` flags;
  `test-decision-tree.sh` asserts the routing policy.
- **The gate**: the GitHub `enforce-ci-checks-develop` ruleset requires exactly
  four checks — `ci-gate`, `required-contracts-ci`, `required-rust-ci`,
  `required-rust-e2e`. These are fan-in jobs (no work, just `requires:`). A merge
  is gated *only* by what they transitively require; anything outside their
  `requires:` chain can fail without blocking merge. Gates use `utils/ci-gate`.
- **Continuation limits**: a setup pipeline continues exactly once, within 6h, no
  setup→setup. A param declared in both `config.yml` and a fragment with
  different defaults fails with "Conflicting pipeline parameters".

## Validating a change locally

Because the real config is merged from fragments, validate the **merged** file, not a
single fragment:

```bash
# 1. Merge the fragments into /tmp/merged-config.yml (uses mise's yq; resolves anchors).
mise exec -- bash .circleci/scripts/merge-configs.sh

# 2. Validate it. --org-slug is REQUIRED: the private org orb
#    ethereum-optimism/circleci-utils won't resolve without it (and the CLI
#    needs CIRCLECI_CLI_TOKEN set to resolve --org-slug).
export CIRCLECI_CLI_TOKEN="<your token>"
circleci config validate --org-slug gh/ethereum-optimism /tmp/merged-config.yml

# 3. The setup config imports the private orb too, so it needs the same flag.
circleci config validate --org-slug gh/ethereum-optimism .circleci/config.yml
```

Install the CLI without sudo:
`curl -fLSs https://raw.githubusercontent.com/CircleCI-Public/circleci-cli/main/install.sh | DESTDIR="$HOME/.local/bin" bash`.

This checks orb resolution and config structure, but **not** continuation-time param
wiring — a param leak across fragment anchors still only surfaces when the pipeline
actually continues.

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
- **Scheduled** — a `scheduled-*` workflow gated on a `c-run_scheduled_*` param and
  dispatched by the schedule-name mapping in `routing.yml` (`build_four_hours` /
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

6. **Cache keys.** CircleCI caches are **write-once**: once a key is saved it is
   immutable, and a later `save_cache` with the same key is a silent no-op — the
   stale content is served forever. So a key must hash *every* input that affects
   the cached content; if any such input can change without the key changing, the
   cache poisons. The classic mistake is keying build output (compiled artifacts)
   on the lockfile alone: `Cargo.lock`/`go.sum` covers dependency *downloads*, but
   the compiled output also depends on the source, the toolchain, and the build
   profile/features — keying it on the lockfile alone serves a stale build whenever
   source changes but deps don't. Verify the key for each cache covers all of its
   real inputs:
   - **dependency downloads** → lockfile (`Cargo.lock`, `go.sum`, etc.) is enough,
     since the lockfile fully determines what's downloaded;
   - **build output** → lockfile **plus** something that changes with the source
     (e.g. a hash of the source tree / `git rev`) plus toolchain pin and
     profile/features.
   Shared content (dependency downloads) → one shared key, not a per-job prefix
   (avoids each job storing/re-downloading its own copy). Separate caches by
   invalidation cadence (toolchain keyed on toolchain pins, deps on lockfile, build
   output on lockfile+source+profile+features). Fallback `restore` keys are
   deliberate: a chain restores a near-match and recompiles the delta; no fallback
   forces a full refresh (right for download caches, so they can't accrete stale
   versions). Keys carry a version buster (`rust-cache-version`, `go-cache-version`)
   for manual invalidation when the key formula itself is wrong. Check `save` and
   `restore` keys stay consistent.

   Also check **cache coverage**: every job that builds or compiles should restore
   the relevant dependency cache, at minimum — a Go job should restore the Go
   module/build cache, a Rust job the cargo registry/git (and ideally build) cache,
   a Node job the package cache. A job that skips the restore step does a cold build
   every run: it inflates PR-path time and cost (ties into item 8), and — just as
   importantly — it re-downloads every dependency over the network on each run,
   turning transient network failures into CI flakes that a warm cache would have
   avoided. When a diff adds a new build job or a new build step to an existing job,
   confirm it restores the appropriate cache rather than relying on another job to
   have warmed it.

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

   **`validate` is not enough — use `process` with pipeline parameters.** Jobs
   reachable only from a `when: << pipeline.parameters.c-run_* >>` workflow are
   skipped when the parameter is false, which is the default, so a job that fails
   to compile validates clean. Process the merged config with the gated params
   turned on:

   ```sh
   printf '{"c-run_release":true,"c-run_kona_publish_prestates":true}' > /tmp/pp.json
   circleci config process --pipeline-parameters /tmp/pp.json /tmp/merged-config.yml
   ```

   **Never write `<<` inside a `command:`.** CircleCI 2.1 treats `<<` as a
   parameter-tag opener before any shell sees it, so bash here-strings (`<<<`) and
   heredocs (`<<EOF`) fail the whole continuation with `Unclosed '<<' tag` — and a
   continuation that fails to compile produces *no* workflows at all, so every
   check silently disappears from the PR instead of going red. This bites shell
   comments too: a comment merely *mentioning* the token fails compilation.
   Escape as `\<<` if you truly need the literal, or restructure to avoid it —
   but mind what the restructure costs:

   ```sh
   ARR=(); while IFS= read -r L; do ARR+=("$L"); done < <(cmd)
                                    # no '<<', but swallows cmd's exit status:
                                    # a failing cmd reads as an empty array
   TMP=$(mktemp); cmd > "$TMP"      # keeps set -e, and keeps the stream off the
   ARR=(); while IFS= read -r L; do ARR+=("$L"); done < "$TMP"   # Prefer this.
   rm -f "$TMP"
   (( ${#ARR[@]} > 0 )) || { echo "empty" >&2; exit 1; }
   ```

   Process substitution trades the `<<` bug for a swallowed-exit-status bug, so
   assert on the result either way. For a heredoc, write the body to a temp file.

   Read the lines with `while read`, not `mapfile`: `mapfile` is a bash 4
   builtin, and macOS ships bash 3.2, so any snippet copied from a CI command
   into a justfile recipe would break local runs. Same for the rest of bash 4
   (`declare -A`, `${var^^}`, `local -n`).

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
- [both] Caches are write-once/immutable per key — a re-save under an existing key
  is a no-op, so the key must hash every input affecting the content (lockfile for
  downloads; lockfile+source+toolchain+profile for build output) or it serves stale
  content forever. Use broader fallback `restore-keys`; a key with no hashed input
  never invalidates.
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
