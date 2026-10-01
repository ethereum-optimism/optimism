# CircleCI to RWX migration

CircleCI remains the merge-gating CI provider. The RWX pilot is for comparing
execution, caching, and feedback before moving required checks. It does not change
GitHub rulesets, fork authorization, schedules, or publishing credentials.

## Run the pilot

`.rwx/pilot.yml` runs shared routing tests plus policy-selected Go lint (with the
superchain bundle) and Rust formatting/upstream-mirror checks. Verdict tasks use
`cache: false`; tool setup can be reused. This is partial coverage, with the
optional push status `RWX: optimism-pilot`. No fork PR trigger or vault/token is
configured. Authorized `external-fork/*` pushes follow the existing Bailiff path.
Terminal verdict tasks disable filesystem output to avoid uploading unused state;
logs remain available. Export any future test reports as explicit artifacts.

Install repo tools using [dev-workflow.md](dev-workflow.md). RWX is initially a
standalone pinned CLI, outside the mise toolset. On Linux x86_64, install v3.32.1
into `~/.local/bin` following the [official CLI installation](https://www.rwx.com/docs/cli#pinning-a-version-for-scripts):

```bash
set -euo pipefail
RWX_CLI_DOWNLOAD=$(mktemp)
curl -fsSL --retry 5 --retry-delay 2 -o "$RWX_CLI_DOWNLOAD" https://github.com/rwx-cloud/rwx/releases/download/v3.32.1/rwx-linux-x86_64
printf '%s  %s\n' b67326b892b301f6c0abaaa69287fb9f53869c766216cb568e861131311991ef "$RWX_CLI_DOWNLOAD" | sha256sum -c - && {
  mkdir -p "$HOME/.local/bin"
  install -m 0755 "$RWX_CLI_DOWNLOAD" "$HOME/.local/bin/rwx"
}
rm -f "$RWX_CLI_DOWNLOAD"
export PATH="$HOME/.local/bin:$PATH"
```

The [linter](https://www.rwx.com/docs/cli-reference/rwx-lint) needs an installed
Node.js on `PATH` (22 or newer recommended). From the repository root:

```bash
mise exec -- bash ops/ci/test-decision-tree.sh
mise exec -- bash .circleci/scripts/test-decision-tree.sh
mise exec -- python -m unittest ops/ci/test_rwx_metadata.py
rwx --version
mise exec -- rwx lint .rwx/pilot.yml --warnings-as-errors
rwx login
rwx whoami
mise exec -- rwx run .rwx/pilot.yml --wait
```

Lint does not need an RWX login. A remote run requires an RWX organization and
authenticated CLI. [CLI runs](https://www.rwx.com/docs/cli-reference/rwx-run)
include local changes through Git patching; use a clean, pushed commit when
comparing providers. For automatic pushes, an organization administrator must
connect the repository using the [RWX GitHub App](https://www.rwx.com/docs/getting-started/github).
Account/app setup and remote execution must be verified separately from local lint.

## Go rollup shadow

`.rwx/go-rollup.yml` adds the optional GitHub push status
`RWX: optimism-go-rollup-shadow` and accepts authenticated CLI execution, which
does not post a VCS status. Both paths use the existing `run-main` routing value.
It runs every package under `./op-node/rollup/...` with `-tags=ci`, without `-short` or a
test-name filter. This is a complete component workload, not a replacement for
the aggregate Go gate or its dependent acceptance, Cannon, contract and Rust jobs.

`ops/ci/go-rollup-tests.sh prepare` builds one sorted package manifest and two
round-robin shards. `go-package-shards.py` rejects empty discovery, Go package or
dependency errors, invalid shard inputs, and any assignment that omits or repeats
a package. Each shard validates that manifest before running the existing
gotestsum wrapper, retaining its three failure retries, JUnit, JSON and per-test
logs. The log artifact also includes the file logger's output under `tmp/testlogs`.
`-count=1` and `cache: false` ensure comparison runs execute tests; terminal
tasks disable filesystem output and publish reports as explicit
[artifacts](https://www.rwx.com/docs/artifacts) and
[test results](https://www.rwx.com/docs/test-results). Go JSON workflows should set
parser options `language: Go` and `framework: go test`, as this shadow does.
Each isolated shard uses a literal JSON report path; RWX associates it with the
parallel task. Test-result paths do not expand filename expressions.
Parallel tasks translate
[RWX shard metadata](https://www.rwx.com/docs/parallelism) into provider-neutral
`CI_SHARD_INDEX` and `CI_SHARD_TOTAL`.

The new run reuses the pilot's pinned checkout, full Git history, tool bootstrap
and routing adapter. A component-specific task installs the existing gotestsum
pin; a lockfile-filtered module download task feeds both shards. Source inputs
remain unfiltered because tests also import other Go components and embed NUT
bundle files. This scope needs no RPC credentials, cloud identity, publishers,
Docker, contract artifacts or Rust binaries. GitHub App setup remains necessary
for automatic pushes; authenticated CLI runs can exercise the workload beforehand.

From a trusted checkout, validate and run it with:

```bash
mise exec -- python ops/ci/test_go_package_shards.py
mise exec -- rwx lint .rwx/go-rollup.yml --warnings-as-errors
mise exec -- rwx run .rwx/go-rollup.yml --wait
```

## Standard contracts shadow

`.rwx/contracts.yml` adds the optional GitHub push status
`RWX: optimism-contracts-shadow` and accepts authenticated CLI execution without
posting a VCS status. The complete standard `contracts-bedrock-tests` workload
runs once without feature overrides and in the existing `CUSTOM_GAS_TOKEN`,
`OPTIMISM_PORTAL_INTEROP` and `ZK_DISPUTE_GAME` configurations. The main variant
uses `run-main`; the feature matrix uses shared `c-run_contracts_feature_tests`.
There is no path, test-name or shard filter. `develop` uses `ci`, other branches
use `liteci`; both retain 128 fuzz runs and 64 invariant runs at depth 32.
Fork and activation test flags remain disabled, retaining their existing
conditional skips.

A component bootstrap installs the repository's Forge/Cast 1.2.3 and svm-rs
0.5.19 pins, with solc 0.8.15, 0.8.19, 0.8.25 and 0.8.28. An unfiltered source
producer explicitly initializes public submodules, downloads Go modules and
builds Go FFI through Just. Keep its full source and Git state until narrower
cache inputs have been proved safe. Contract tool, compiler, submodule and Go
module downloads retry.

Each uncached verdict runs `just test` and the existing Go test-convention check.
The runner validates effective Foundry settings and a complete test-file
inventory, removes inherited filters/feature overrides, and preserves the first
test failure while collecting `just test-rerun` traces. Terminal tasks disable
filesystem output and export literal-path JUnit with `Solidity`/`Foundry` parser
labels, compiler output, configuration, inventory, traces and generated
counterexamples/file reports. Verify native parsed counts and failed-run artifact
collection in hosted runs before promoting any check.
This shadow needs no RPC credentials, vault, publisher, Docker or Rust producer.
Coverage, upgrade/fork, heavy-fuzz, snapshot and semver jobs remain outside this
bounded workload; the existing required contracts gate stays in CircleCI.

From a trusted checkout:

```bash
mise exec -- python ops/ci/test_contracts_shadow.py
mise exec -- rwx lint .rwx/contracts.yml --warnings-as-errors
mise exec -- rwx run .rwx/contracts.yml --wait
```

## Compare retained test evidence

Use [ci-comparison.md](ci-comparison.md) to collect reports from both providers
on the same pushed commit, compare test identities, outcomes and skips, and report
observed retry history. The report distinguishes missing or different cases
from evidence that is incomplete or belongs to a different revision, profile or
routing context.

An initial automatic push on October 1, 2026 completed all three optional checks
at commit `02d29aa13f9da3d8e51a1f52a987eb69e1b6bda0`: [pilot](https://cloud.rwx.com/optimism/runs/f8d83016615a4dcabd89b0e79c055610),
[Go rollup](https://cloud.rwx.com/optimism/runs/fbd271ef52184767a0bf8f2c03e1b393),
and [standard contracts](https://cloud.rwx.com/optimism/runs/d10bf0a7eb65404aac0782d41d386ea7). CircleCI's project setting
`build-prs-only` is enabled: a branch-only push starts RWX, while an open PR is
needed for the CircleCI baseline. Keep the actual provider trigger in the
collection; compare the verified shared routing context rather than relabeling
an event. GitHub status details link to the corresponding native RWX run.

Cached RWX tasks may expose execution timestamps and durations from the run that
created the cache entry. Exclude those historical execution values from current
compute totals. Retain cache hits separately, use run start/end timestamps for
elapsed time, and leave billed time and price unknown when the provider does not
return them. A warm run alone does not establish cold-cache performance.

## Migration contract

Preserve the existing routing and test coverage before tuning performance:

- PRs run the main workflow, with contract and Rust suites selected by changed
  paths. Only changes entirely inside `docs/public-docs/` take the docs fast path.
- Merge queue runs the main and contract suites, with Rust selected by changed
  paths. Docs-only merge groups still emit all required gates.
- `develop` runs the full post-merge set unconditionally. Diffing `develop`
  against itself cannot select post-merge suites correctly.
- Tags retain component-specific release filters. Schedules and manual dispatches
  retain their current workload selection.
- Changes to `.circleci/`, `.rwx/`, and `ops/ci/` select CI, contract, and Rust
  validation. A new or otherwise unclassified code path must not take a docs skip.

`ops/ci/routing.yml` and `ops/ci/compute-workflow-conditions.sh` hold shared policy;
CircleCI retains adapters in `.circleci/scripts/` and a routing-data symlink.
`ops/ci/rwx-metadata.sh` maps push/CLI metadata into that policy and exports
`run-main`/`run-rust-ci` values. Extend the shared routing tests for new selection
rules, retaining adapter parity and real changed-file fixtures.
CircleCI's per-job `ops/check-changed/main.py` rebuild policy also includes `.rwx/`
and `ops/ci/`, so selected contract jobs execute on CI changes in stacked PRs.

## Inventory before cutover

For each workload, capture commands, dependencies, shards, resources, timeouts,
caches, output files/results, credentials and merge-blocking status. Compare
coverage, cold/warm duration, critical-path time, cost and failures on the same
commit/event.

Checked-in configuration does not contain all operational state. Export and
review these separately:

- GitHub ruleset contexts, provider identities, branch patterns, and bypass rules.
- CircleCI context variable names and restrictions, project settings, concurrency,
  scheduled trigger cadence/timezone/branch, and API dispatch clients.
- GCP workload identity providers, claim conditions, service accounts and bucket
  permissions; Bailiff's deployed version, team configuration, and webhook setup.

Inventory secret names and permissions without exporting their values into the
repository or run artifacts.

### Historical CircleCI baseline

A read-only API snapshot collected on October 1, 2026 sampled five successful
`develop` pipelines created between `2026-09-30T16:23:15.165Z` and
`2026-10-01T13:43:39.973Z` (UTC). All four core workflows
succeeded in each sample; 435 job records include executor classes and parallelism.

| Scope | Median elapsed time | CircleCI executor / shards |
| --- | --- | --- |
| Main workflow | 25.77 min | Multiple |
| Rust CI workflow | 17.48 min | Multiple |
| Rust E2E workflow | 19.78 min | Multiple |
| Contracts workflow | 16.50 min | Multiple |
| Setup workflow | 47 sec | Single setup job |
| Go lint job | 5.07 min | Docker `large-gen2` / 1 |
| Rust formatting job | 1.37 min | Docker `medium-gen2` / 1 |

Sampled commit SHAs, oldest first:

```text
5686e957e7456831d8891356e6a8ab145f0439c4
031adbfb268906396306a41ffde0d78f0120bf81
777fc39a9220814391ee1fdf99f56b923ad487ca
da6d3252491754837a778061db0cc47236ec13c6
a271fd6b3feb76e49f3453e5a27be4b33a3c7211
```

The main execution path consistently passed through op-reth compilation, sysgo
binary aggregation, and acceptance tests. These successful-run medians are
historical orientation, not performance targets or failure-rate estimates. RWX
uses different CPUs, caches, and checkout history; compare equivalent workloads
on the same commit/event before attributing timing differences to the provider.
Job elapsed time is not total billed time across shards.

The live schedule inventory had five definitions targeting `develop`, while
routing mapped three. `build_sunday_early` (Sunday hour 01 UTC) and `build_mon_thu`
(Monday/Thursday hour 07 UTC) had no routing entry and therefore selected no
continuation workflows under the current policy. Confirm their owners and intent
before carrying them over or removing them. The mapped schedules used four-hourly
hours 00/04/08/12/16/20, daily hour 04, and Sunday hour 00. CircleCI schedules run
within the specified UTC hour rather than guaranteeing its exact start. See
[scheduled trigger timing](https://circleci.com/docs/guides/orchestrate/schedule-triggers/).

All nine referenced contexts were accessible through the restriction API. Six
exposed only the organization-wide All members group. The repository write-token
context additionally had project and branch-expression restrictions; the sccache
context restricted access to the two Optimism repositories; the release context
required the release-managers group. Preserve these restrictions when transferring
credentials. GCP federation and bucket authorization still need a separate audit;
CircleCI context restrictions alone do not establish those permissions.

## Required checks and fork authorization

The `develop` ruleset currently requires CircleCI contexts for `ci-gate`,
`required-contracts-ci`, `required-rust-ci`, and `required-rust-e2e`.
`dependency-review` is a separate GitHub Actions requirement. Match both the
context and its provider when reviewing a proposed ruleset change; preserving a
display name alone does not transfer a requirement to RWX.

The CircleCI gates use terminal dependencies and the private
`ethereum-optimism/circleci-utils` orb to inspect upstream results. RWX replacements
must report a failure after a failed/canceled build or test, even when descendants
never ran. An intentional safe skip must still produce every required context.
Gate coverage includes exact matrix names, not just job templates.

Test reporting on actual PR/merge-group commits, superseded runs and cancellation
before changing requirements. Keep distinct, non-required RWX checks meanwhile.

Legacy rulesets `enforce-circleci-check-old-backports` and
`enforce-circleci-check-porposal-v3` cover `backports/op-deployer/v*` and
`proposal/op-contracts/v*` with older CircleCI and image checks. Backport replacement
config/requirements or retain CircleCI for those branches. Re-read live rulesets
before cutover; these names describe the audited baseline.

Keep human authorization of exact fork commits through Bailiff's internal
`external-fork/*` pushes. New commits require new authorization. Verify check
association with the approved SHA and prevent privileged direct fork execution,
including access to secrets or writable trusted caches. See
[PR authorization](../../ops/book/src/ci/pr-authorization.md) and
[the human-only authorization rule](../handbook/pr-guidelines.md#triggering-ci-on-prs-from-external-forks).

## Integration blockers

| Area | Current dependency | Required replacement |
| --- | --- | --- |
| Toolchain and checkout | Private orb installs mise, warms tools, and handles checkout | Explicit pinned toolchain and checkout tasks, including submodules and required Git history |
| Shards | `circleci tests split`, `CIRCLE_NODE_*`, and CircleCI JUnit timing history | RWX sharding with complete test-set coverage and provider-neutral shard inputs |
| Metadata | Branch, PR, repository, workflow ID, and run URL in CircleCI variables | Explicit shared metadata; preserve true PR base lookup and merge-group SHA |
| Workspaces | Contract outputs, binaries, prestates, and gitignored `superchain-configs.zip` | Explicit producer/consumer dependencies and retained outputs |
| Reporting | JUnit uploads, log/artifact paths, CircleCI Insights flake API | Accessible failed-run artifacts and a replacement for flake history/reporting |
| Cloud identity | `CIRCLE_OIDC_TOKEN` and GCP workload identity | Separate RWX vault subjects and exact-sub GCP bindings; prove equivalent ref restrictions before writer cutover |
| Rust compile cache | sccache GCS reader for all refs; writer only for `develop` | Enforced reader/writer identity split or a separately validated RWX cache design |
| Release and publish | GoReleaser, contract artifacts, Cannon/SP1 prestates | One active publisher per destination, preserving component tag/ref filters |
| Maintenance | TODO/Cannon every four hours; daily suites; weekly nightly PR; labels and stale automation | Explicit ownership, cadence, permissions, and one active trigger |

Audit `justfile`, `ops/scripts/shard-tests.sh`, acceptance/Kona justfiles, and contract
target-branch/semver scripts. Compatibility variables do not replace Circle's CLI
or timing history. `CIRCLECI` also changes dependency builds and Docker pruning.

[RWX OIDC](https://www.rwx.com/docs/oidc) documents issuer
`https://cloud.rwx.com/mint`, a vault-identifying `sub`, audience and run/task IDs;
the documented claims contain no signed repository, ref or trigger identity.
Reader and writer identities need separate vault subjects and exact-sub GCP
bindings. These enforce vault identity without independently verifying GitHub refs.
[Locked vaults](https://www.rwx.com/docs/vaults) restrict repository/ref access but
also permit user/service-account grants. Keep the current CircleCI sccache writer
until denial tests prove CLI runs, local patches and unauthorized refs cannot
access the writer vault, or signed VCS claims/external attestation restores that
policy.

Preserve op-deployer's release settings for fresh tools/modules and disabled caches.

Tests must execute on comparison runs; retain retries/flaky-result reporting.
Dependency/build cache reuse must include its real inputs.

## Rollout and rollback

1. **Pilot:** run the small workload above on an exact commit; compare results.
2. **Shadow:** add Go/contract/Rust suites with equivalent shards and dependencies,
   then compare PRs and merge groups using distinct, non-required RWX checks.
3. **Gates:** inject failures, cancellations and safe skips; exercise internal PRs,
   authorized forks and merge groups. Verify all gates and legacy branches before
   changing requirements.
4. **Post-merge:** validate full `develop`, schedules, dispatches and governance
   writes. Transfer each trigger to one provider.
5. **Publishers:** validate release-candidate artifacts/provenance and identity
   restrictions without publishing, then transfer each destination to one provider.
6. **Retirement:** remove obsolete config/tokens/identities after the rollback window
   and legacy branch obligations. Keep unrelated GitHub Actions workflows.

Keep the CircleCI config runnable, preserve exported rulesets/trigger settings, and
retain its required identities during comparison. A rollback restores previous
required contexts and transfers trigger/publisher ownership back; it must not
activate duplicate publishing or maintenance runs.

Before retirement, update `ci-ops.md`, `ci-config-review.md`, the CI reviewer agent,
the TODO repair skill, authorization/runner runbooks, feature-matrix documentation,
and Kona SP1 publishing guidance. The image-provenance GitHub Action currently
parses the Rust image pin from `.circleci/config.yml`; move its source and path
triggers together if that pin moves.
