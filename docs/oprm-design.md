# OPRM Design Doc

`oprm` is a Go command in this monorepo that manages the OP Stack release flow for selected components. It is a stateful, resumable, confirmation-driven workflow runner with a terminal UI and a markdown-backed audit log.

This document covers the MVP scope for steps 1-6 of the current release process. Rollout/finalization are explicitly out of scope for v1.

Implementation backlog: see `docs/oprm-backlog.md`.

## Prototype Status Snapshot

As of 2026-04-10, the `oprm` prototype branch already has a working foundation and an initial interactive workflow. This status does not imply that `oprm` is merged into `develop` yet, and agents should not assume `./oprm/...` exists in every checkout.

- planned binary path after merge: `./oprm/cmd/oprm`
- must be invoked from the monorepo root
  - monorepo-root validation should use stable repository markers (for example: `git rev-parse --show-toplevel` and the root `go.mod` module path)
  - monorepo-root validation must not depend on the presence of unmerged `./oprm/...` files
- markdown-backed run journal under `.oprm/releases/` by default
- default local `op-geth` checkout path: `../op-geth`
- `doctor` checks for:
  - `gh` installed
  - `gh` authenticated
  - `git` installed
  - `git config user.name`
  - `git config user.email`
  - local monorepo tags fetched from the local git remote that matches the configured monorepo target repo
  - local `op-geth` tags fetched from the local git remote that matches the configured `op-geth` target repo
  - monorepo checkout on its base branch and on a releasable commit from the matching local remote's `<base-branch>`
    - exact match is fine
    - an older ancestor commit is also fine so an in-progress release can be resumed after new commits land remotely
  - `op-geth` checkout on `optimism` and on a releasable commit from the matching local remote's `optimism` branch
  - GitHub release manager identity
  - push access to the configured monorepo target (defaults to `ethereum-optimism/optimism`)
  - push access to the configured `op-geth` target (defaults to `ethereum-optimism/op-geth`)
- component registry for:
  - `op-geth`
  - `op-node`
  - `op-batcher`
  - `kona-node`
  - `op-reth`
- GitHub release discovery for latest stable release, latest RC, and latest draft RC
- first-release bootstrap behavior:
  - if a component has no prior release or RC history in the configured target repo, `oprm` proposes `v0.0.1-rc.1` targeting `v0.0.1`
- change detection against component dependency scopes
- review-range generation:
  - previous stable release → draft RC, if a draft RC exists
  - previous stable release → base branch otherwise
- draft-resume semantics:
  - if a draft RC exists, that RC is treated as the active in-progress release to resume
  - `oprm` does not auto-bump to the next RC in that case
- a Bubble Tea TUI with:
  - component selection
  - task inspection / execution
  - target repo / branch header context
  - local checkout path in task context
  - remote tag-state highlighting for target release / proposed RC
  - retry / skip / externally-satisfied actions
- initial per-component task flow:
  - `component.review-diff`
  - `component.prepare-release-notes`
  - `component.create-tag`
  - `component.push-tag`
  - `component.github-draft-release`
  - `component.docker-build`
  - `component.rollout`
  - `component.finalize-release`
- release-notes artifact generation under `.oprm/releases/<run-id>/release-notes/`

The current intended operator flow is:

```bash
oprm run
```

This starts a run, auto-detects changed candidate components, and opens the TUI so the release manager can choose which components to release.

If `oprm run` is invoked in a non-TTY context, it will persist the run and print a message telling the operator to resume it later:

```bash
oprm resume <run-id>
```

`oprm plan` still exists, but is now primarily a power-user / scripting interface rather than the preferred interactive entrypoint.

## Goals

- Automate the currently manual release flow for:
  - `op-geth`
  - `op-node`
  - `op-batcher`
  - `kona-node`
  - `op-reth`
- Keep a resumable release run state on disk.
- Require explicit user confirmation before each mutating action.
- Handle tasks that were already performed elsewhere.
- Support `retry` and `skip` semantics.
- Record an audit trail in markdown, including the current release manager.
- Provide a modifiable terminal UI.
- Port existing tagging/release logic instead of shelling out to `op-workbench`.

## Non-Goals for v1

- Automating rollout into k8s repos.
- Full CircleCI automation. In v1, image/job monitoring is a manual confirmation task.
- Managing arbitrary release processes outside the supported component set.

## Product Requirements

### Startup / release manager detection

On startup, `oprm` must:

1. Check that `gh` is installed.
2. Check that `gh` is authenticated.
3. Check that `git` is installed.
4. Read `git config user.name` and `git config user.email`.
5. Resolve the current GitHub user via `gh api user` or equivalent.
6. Persist that identity as the release manager for the run.

If any required prerequisite is missing, the run is blocked until resolved.

### Confirmations

Every task has a detection phase and an execution phase:

- `Detect` may run automatically.
- Any mutating or state-advancing action must require explicit user confirmation.
- `retry`, `skip`, and `mark externally satisfied` also require confirmation.

### Retry / skip / external state reconciliation

Tasks must be idempotent and reconciliation-first:

- First inspect current real-world state.
- If already done elsewhere, mark as `externally-satisfied`.
- If not done, plan the next action.
- If the action fails, allow `retry`.
- If the operator wants to move on, allow `skip` with a reason.

### Release log

A release run must write a markdown log that contains:

- run metadata
- current release manager
- selected components
- proposed versions
- task states
- appended action log with timestamps and operator decisions

Default storage location:

- `.oprm/releases/`

This location must be configurable.

## Architecture Overview

`oprm` should be implemented as a workflow engine with four major layers:

1. **Component registry**
   - defines per-component repo, tag, change-detection paths, release-note scope, and special hooks
2. **Workflow engine**
   - builds a task DAG from selected components and version decisions
3. **Providers**
   - `git`, `gh`, local filesystem, and later CircleCI API
4. **Terminal UI**
   - task list, details, confirmations, logs, and resume/retry/skip flows

Recommended implementation stack:

- Go
- `urfave/cli/v2` for command entrypoints
- Bubble Tea + Lip Gloss for TUI

## Proposed Repository Layout

A new top-level Go command directory:

```text
oprm/
  cmd/
    main.go
  flags/
    flags.go
  manager/
    app.go
    config.go
  workflow/
    task.go
    dag.go
    runner.go
    statuses.go
  components/
    registry.go
    op_geth.go
    op_node.go
    op_batcher.go
    kona_node.go
    op_reth.go
  providers/
    git/
    github/
    shell/
  release/
    run.go
    versions.go
    ordering.go
  journal/
    markdown.go
  tui/
    model.go
    views/
      summary.go
      tasks.go
      task_detail.go
      confirm.go
```

Binary name: `oprm`.

## Workflow Model

A release run is built from:

- selected components
- current release manager
- proposed target versions
- a task DAG generated from those choices

Each task follows the same lifecycle:

1. `Detect`
2. `Plan`
3. `Confirm`
4. `Execute`
5. `Verify`
6. `Record`

### Task Statuses

```text
pending
ready
blocked
needs-confirmation
running
completed
skipped
failed
externally-satisfied
```

### Why `externally-satisfied` matters

`skip` means: we are choosing not to perform this task in this run.

`externally-satisfied` means: the task objective was already satisfied by external action, and the workflow may proceed without loss of correctness.

## Task Interface

Suggested Go interface:

```go
type Task interface {
    ID() string
    Title() string
    Description() string
    Component() string
    Dependencies() []string

    Detect(ctx context.Context, run *Run) (Observation, error)
    Plan(ctx context.Context, run *Run, obs Observation) (Plan, error)
    Execute(ctx context.Context, run *Run, plan Plan) (Result, error)
    Verify(ctx context.Context, run *Run, result Result) (Verification, error)
}
```

Auxiliary concepts:

- `Observation`: what exists now
- `Plan`: what `oprm` intends to do next
- `Result`: what action was attempted
- `Verification`: whether the desired state now holds

## Release Run Data Model

The source of truth for a run is a markdown journal file with YAML frontmatter.

### Run file shape

```md
---
run_id: 2026-04-10-01
status: in_progress
created_at: 2026-04-10T12:00:00Z
updated_at: 2026-04-10T12:30:00Z
repo: ethereum-optimism/optimism
base_branch: develop
release_manager:
  gh_login: alice
  git_name: Alice Example
  git_email: alice@example.com
config:
  runs_dir: .oprm/releases
components:
  - op-geth
  - op-node
versions:
  op-geth:
    latest_release: v1.101605.0
    latest_rc: v1.101605.1-rc.1
    bump: patch
    proposed: v1.101605.1-rc.2
    manual_override: false
  op-node:
    latest_release: v1.14.2
    latest_rc: v1.14.3-rc.1
    bump: patch
    proposed: v1.14.3-rc.2
    manual_override: false
tasks:
  - id: doctor.git
    status: completed
  - id: doctor.gh-cli
    status: completed
  - id: op-geth.tag-rc
    status: pending
---

# OPRM Release Run 2026-04-10-01

## Timeline

- 2026-04-10T12:00:01Z detected release manager `alice`
- 2026-04-10T12:03:00Z selected components `op-geth`, `op-node`
- 2026-04-10T12:05:00Z proposed `op-geth/v1.101605.1-rc.2`
```

### Optional per-run artifact directory

To avoid overloading the markdown file, the run may also have an artifact directory:

```text
.oprm/releases/<run-id>/
  run.md
  release-notes/
    op-geth-v1.101605.1-rc.2.md
    op-node-v1.14.3-rc.2.md
  evidence/
    task-op-geth-tag-rc.json
```

`run.md` remains the canonical audit log.

## Configuration Model

Default config file:

```text
.oprm/config.yaml
```

Suggested schema:

```yaml
runs_dir: .oprm/releases
base_branch: develop
github:
  owner: ethereum-optimism
  repo: optimism
op_geth:
  owner: ethereum-optimism
  repo: op-geth
  checkout_path: ../op-geth
```

For local testing against a fork, point the monorepo target at your fork instead of the upstream repo:

```yaml
github:
  owner: nonsense
  repo: optimism
```

The same values may also be overridden via CLI flags or environment variables:

- `--github-owner` / `OPRM_GITHUB_OWNER`
- `--github-repo` / `OPRM_GITHUB_REPO`
- `--op-geth-owner` / `OPRM_OP_GETH_OWNER`
- `--op-geth-repo` / `OPRM_OP_GETH_REPO`
- `--op-geth-checkout` / `OPRM_OP_GETH_CHECKOUT`

Note: release discovery currently reads GitHub Releases from the configured target repo. If you point `github.owner` / `github.repo` at a fork, make sure that fork has the release history you want to test against.

`base_branch` currently applies to the monorepo-backed components (`op-node`, `op-batcher`, `kona-node`, `op-reth`). `op-geth` uses its component default branch, which is currently `optimism`.

`op_geth.checkout_path` defaults to `../op-geth` and is resolved relative to the monorepo root.

Override sources:

1. CLI flags
2. environment variables
3. `.oprm/config.yaml`
4. built-in defaults

## Component Registry Schema

The registry should be code-backed, but shaped as structured data so it can be extended safely.

Suggested schema:

```go
type ComponentSpec struct {
    ID                string
    Kind              string // monorepo-go, monorepo-rust, external-go
    DisplayName       string

    GitHubOwner       string
    GitHubRepo        string
    BaseBranch        string
    TagPrefix         string // e.g. op-node or op-geth

    ChangeScope       []string
    ReleaseNotesScope []string

    Versioning        VersionPolicy
    Dependencies      []ComponentDependency
    Hooks             ComponentHooks
}
```

### Version policy

```go
type VersionPolicy struct {
    SupportsMajor bool
    SupportsMinor bool
    SupportsPatch bool
    AutoIncrementRC bool
}
```

### Dependency example

```go
type ComponentDependency struct {
    ComponentID string
    Kind        string // release-order, source-version-reference
}
```

## Initial Component Specs

### `op-geth`

- GitHub repo: configurable via `op_geth.owner` / `op_geth.repo` (default: `ethereum-optimism/op-geth`)
- Default comparison branch: `optimism`
- Tag prefix: `v` in the external repo release model
- Versioning: auto-increment RC, allow manual override
- Special hook:
  - if `op-node` is in the run, inject `op-node` dependency update tasks

### `op-node`

- GitHub repo: configurable via `github.owner` / `github.repo` (default: `ethereum-optimism/optimism`)
- Tag prefix: `op-node`
- Change scope:
  - `op-node/**`
  - `go.mod`
  - `go.sum`
  - `op-core/**`
  - `op-service/**`
  - `op-chain-ops/**` if it is on the dependency graph for release tooling/builds
- Special hook:
  - may need `op-geth` version bump in `go.mod`

### `op-batcher`

- GitHub repo: configurable via `github.owner` / `github.repo` (default: `ethereum-optimism/optimism`)
- Tag prefix: `op-batcher`
- Change scope:
  - `op-batcher/**`
  - `go.mod`
  - `go.sum`
  - shared Go libs on dependency graph

### `kona-node`

- GitHub repo: configurable via `github.owner` / `github.repo` (default: `ethereum-optimism/optimism`)
- Tag prefix: `kona-node`
- Change scope:
  - `rust/kona/**`
  - `rust/Cargo.toml`
  - `rust/op-alloy/**`
  - `rust/alloy-op*/**`

### `op-reth`

- GitHub repo: configurable via `github.owner` / `github.repo` (default: `ethereum-optimism/optimism`)
- Tag prefix: `op-reth`
- Change scope:
  - `rust/op-reth/**`
  - `rust/Cargo.toml`
  - `rust/op-alloy/**`
  - `rust/alloy-op*/**`

## Change Detection Rules

For v1, a component is considered changed if there are relevant code or dependency-graph changes since the last release tag.

This includes:

- direct code changes under the component path
- changes in shared libraries used by the component
- top-level module manifest changes (`go.mod`, `go.sum`, `rust/Cargo.toml`) when relevant

This excludes:

- docs-only changes
- unrelated repo changes outside the component's scope

`oprm` should reuse existing repo metadata where possible:

- `.github/docker-images.json`
- current `just release-notes` include-path conventions
- existing release-note and version helper scripts where useful

## Version Discovery and Selection

For each selected component:

1. Detect latest full release.
2. Detect latest RC release.
3. Compute whether the component changed since the latest full release.
4. Suggest the next version.
5. Auto-increment `-rc.N` if an RC series already exists.
6. Require operator confirmation.
7. Allow manual override before the workflow is materialized.

### Proposed version UX

For each component, the operator should be able to choose:

- no release
- patch bump
- minor bump
- major bump
- manual version input

Then `oprm` computes the RC tag:

- `vX.Y.Z-rc.N`

with `N` auto-incremented from GitHub/tag state.

## Workflow DAG for MVP

### Current implemented workflow stages

#### Global doctor checks

1. `doctor.git`
2. `doctor.gh-cli`
3. `doctor.git-fetch-tags-monorepo`
4. `doctor.git-fetch-tags-op-geth`
5. `doctor.monorepo-base-branch-synced`
6. `doctor.op-geth-base-branch-synced`
7. `doctor.release-manager-detected`
8. `doctor.repo-push-permissions`

#### Component planning and selection

After doctor passes, `oprm run` currently:

1. discovers candidate components
2. detects changed components
3. discovers latest stable release / latest RC / latest draft RC
4. computes the review range and proposed target RC
   - if no prior release or RC history exists for a component, bootstrap at `v0.0.1-rc.1` targeting `v0.0.1`
5. opens the TUI in component-selection mode

In component-selection mode:

- all changed components are preselected
- any component with a draft RC is also preselected
- the release manager confirms which components are part of the run

#### Current per-component task model

For each selected component, the currently implemented task flow is:

1. `component.review-diff`
2. `component.prepare-release-notes`
3. `component.create-tag`
4. `component.push-tag`
5. `component.github-draft-release`
6. `component.docker-build`

Current behavior:

- `component.review-diff` is confirmation-driven and combines:
  - reviewing the diff
  - approving that diff as the intended release scope
- local checkout / branch verification is now handled during startup doctor checks rather than as a per-component task
- `component.prepare-release-notes` writes a release-notes artifact under `.oprm/releases/<run-id>/release-notes/`
- `component.create-tag` creates the proposed RC tag locally only
- `component.push-tag` pushes the already-created local RC tag to the remote repository and verifies remote visibility
- `component.github-draft-release` creates or updates the GitHub draft release using the generated release-notes artifact
- `component.docker-build` is a manual confirmation checkpoint that happens only after the RC tag exists remotely and the draft release is present

Task confirmation should happen on the concrete task itself, not via a separate meta-task.

### Planned next per-component tasks

The next layer of tasks to add after `component.github-draft-release` is:

1. additional reconciliation-hardening for inconsistent tag/release states
2. richer build/link surfacing for the manual confirmation step
3. explicit release URL / remote tag / local tag status surfacing in the TUI

### `op-geth` -> `op-node` special path

If both `op-geth` and `op-node` are selected, insert:

1. `op-geth.tag-rc`
2. `op-geth.push-tag`
3. `op-geth.create-draft-release`
4. `op-node.detect-op-geth-version-gap`
5. `op-node.update-go-mod-op-geth`
6. `op-node.go-mod-tidy`
7. `op-node.create-update-branch`
8. `op-node.commit-update-branch`
9. `op-node.push-update-branch`
10. `op-node.create-pr-to-develop`
11. `op-node.wait-for-pr-confirmation`
12. `op-node.refresh-develop-and-verify`
13. continue with `op-node` tagging tasks

The PR step is confirmation-driven. `oprm` does the mechanical work, but the operator confirms each action.

## Preflight Verification

`component.review-diff` owns release-scope review:

- which commits/files are in scope
- latest release / latest RC / latest draft RC
- proposed target release and proposed RC

Startup doctor checks own local checkout verification:

- expected branch
- current HEAD SHA / releasable relationship to the remote base branch

Before creating any tag, `oprm` must verify and show:

- target repo and branch
- current HEAD SHA
- whether working tree is clean
- latest release tag and latest RC tag
- change summary since the prior release
- generated release notes preview path
- whether a matching tag already exists
- whether a matching draft GitHub release already exists

The operator must explicitly confirm on the concrete mutating task before the release step begins.

## Tagging and Draft Releases

For v1, `oprm` should port the tagging/release creation logic rather than shelling out to `op-workbench`.

### Required behaviors

- create annotated tag if needed
- push tag if needed
- create draft release if none exists
- update draft release if one already exists
- clearly explain whether it is creating or updating
- safely re-run if tag or release already exists

### Reconciliation logic

When a component/version task runs, detect these cases:

1. tag missing, release missing
   - create both
2. tag exists, release missing
   - if the existing tag points to the intended release commit, mark the relevant tag task `externally-satisfied` and create the release only
   - if the existing tag points somewhere else, block and ask the operator to reconcile
3. tag missing, release exists
   - block and ask operator to reconcile
4. tag exists, release exists
   - if the existing tag points to the intended release commit, mark tag tasks `externally-satisfied` or update notes after confirmation
   - otherwise block and ask the operator to reconcile

## GitHub Release Notes

For monorepo components, `oprm` should generate release notes using the same include-path logic currently encoded in `just release-notes`.

This logic should be ported into Go, not shelled out.

For `op-geth`, the notes generator may initially be simpler if the external repo structure differs, but it must still support:

- latest release lookup
- latest RC lookup
- changelog range generation
- draft release body generation

## CircleCI Handling in v1

CircleCI is intentionally manual in the MVP.

Each component gets a task:

- `component.docker-build`

This task should:

- display a description telling the operator what to check
- show any relevant tag/version context
- allow the operator to mark the task complete after manual verification
- write that confirmation and optional notes to the run log

Later, this task can be replaced by an automated CircleCI poller without changing the rest of the task graph.

## Terminal UI

The TUI is the primary operator experience. It is task-driven, but also has an explicit component-selection stage before task execution begins.

### Current UI stages

#### Stage 1: component selection

The left pane shows release candidates, and the right pane shows review details for the highlighted component.

Displayed information includes:

- whether the component changed
- latest stable release
- latest RC
- latest draft RC
- whether `oprm` is resuming an existing draft RC
- review range
- compare URL
- commit summaries

Current selection controls:

- `j` / `k` or arrows: move
- `space`: toggle component selection
- `a`: select all changed components
- `enter`: confirm component selection
- `g`: refresh run state
- `q`: quit

#### Stage 2: task execution

After selection is confirmed, the left pane shows tasks and the right pane shows details for the selected task/component.

Current task controls:

- `enter`: confirm and execute the selected task
- `r`: retry/reset the selected task
- `s`: skip the selected task with a reason
- `e`: mark the selected task externally satisfied with a reason
- `g`: refresh run state
- `q`: quit

### Why task-driven UI

A task-driven UI is easier to modify than a linear wizard because:

- dependencies are explicit
- retry/skip/external satisfaction are first-class
- new tasks can be added without rewriting the whole flow

## CLI UX

### Current intended commands

```text
oprm doctor
oprm run
oprm resume <run-id>
oprm status <run-id>
oprm plan <run-id> [component-id ...]
oprm retry <run-id> <task-id>
oprm skip <run-id> <task-id> <reason>
oprm satisfy <run-id> <task-id> <reason>
oprm log <run-id>
```

### Current intended operator flow

```bash
oprm doctor
oprm run
```

`oprm run` is the preferred entrypoint. It creates the run, auto-plans candidate components, and opens the TUI.

If the command is executed in a non-TTY environment, `oprm` will not open the TUI automatically and will instead print the run id so the operator can continue later with:

```bash
oprm resume <run-id>
```

### Advanced / scripting usage

`oprm plan` remains available for power-user and scripting workflows, especially when manually supplying bumps or target versions.

## Confirmation Policy

The policy should be strict and simple:

- detection can run without a prompt
- any mutation requires confirmation
- any state override requires confirmation
- manual tasks require confirmation to advance

Mutations include:

- editing `go.mod`
- running `go mod tidy`
- creating branches
- committing changes
- pushing branches
- creating PRs
- creating tags
- pushing tags
- creating or updating GitHub draft releases

## Audit Log Format

Every task action appends a markdown log entry with:

- timestamp
- task id
- action type
- operator identity
- summary
- result
- links or artifact paths when relevant

Example:

```md
- 2026-04-10T12:20:00Z `op-node.create-pr-to-develop` confirmed by `alice`
  - branch: `oprm/op-node-op-geth-v1.101605.1-rc.2`
  - pr: https://github.com/ethereum-optimism/optimism/pull/12345
  - result: created
```

## Failure Model

A task failure should not corrupt the run.

Rules:

- every task writes pre-action intent before mutation
- every task writes post-action result after mutation
- reruns always begin with fresh detection
- partial completion is legal if verification can prove state
- operator can retry, skip, or mark externally satisfied

## Testing Strategy

### Unit tests

- version parsing and bump logic
- RC auto-increment logic
- tag/release reconciliation logic
- task DAG construction
- journal serialization/deserialization
- change-scope matching

### Integration tests

- fake git repositories for tag/branch workflows
- mock `gh` provider for release/PR flows
- resume after partial failure
- `op-geth` -> `op-node` dependency update flow

### Manual MVP validation

- dry-run release plan for each supported component
- full RC flow for a monorepo Go component
- full RC flow for a monorepo Rust component
- `op-geth` external repo flow
- `op-geth` + `op-node` combined flow with PR generation

## MVP Milestones

### Milestone 1: skeleton

- command scaffolding
- config loading
- markdown journal format
- provider wrappers for `git` and `gh`
- `doctor` command

### Milestone 2: run planning

- component registry
- version discovery
- change detection
- RC auto-increment
- initial task DAG generation

### Milestone 3: TUI

- run summary screen
- task list/details
- confirmation prompts
- resume existing runs

### Milestone 4: monorepo release actions

- tag creation and push
- draft release create/update
- release note generation
- manual CircleCI confirmation task

### Milestone 5: `op-geth` integration

- external repo access
- release flow in `ethereum-optimism/op-geth`
- `op-node` `go.mod` update flow
- branch, commit, push, PR creation to `develop`

### Milestone 6: hardening

- retry/skip/external-satisfaction correctness
- integration tests
- better error messaging
- docs and onboarding

## Next Implementation Steps

The highest-priority next steps are now the first mutating release actions after `component.review-diff`.

### Near-term next steps

1. **Preflight verification task**
   - verify branch / HEAD / clean working tree
   - show whether the target tag already exists
   - show whether the draft release already exists

2. **Release notes preparation task**
   - generate release-note content for the selected review range
   - persist the generated notes under the run artifacts directory
   - show a preview path in the TUI

3. **Tag reconciliation + execution task**
   - detect whether the RC tag already exists
   - if missing, create and push it after confirmation
   - if present, mark it as resumed/external state rather than blindly bumping

4. **Draft GitHub release reconciliation task**
   - detect whether the draft release already exists
   - if missing, create it
   - if present, update/continue it
   - clearly explain whether `oprm` is creating or resuming

5. **Manual build-confirmation task**
   - keep CircleCI manual for now
   - add explicit operator acknowledgment when images/builds are ready

### Medium-term next steps

6. **`op-geth` -> `op-node` dependency-update workflow**
   - detect `op-geth` draft/resume state
   - update `op-node` `go.mod`
   - run `go mod tidy`
   - create branch / commit / PR to `develop`

7. **Richer TUI ergonomics**
   - scrolling in the details pane
   - open compare URL in browser
   - clearer badges for `changed` vs `resume draft`
   - stable layout around long lines and long commit messages

8. **Task DAG expansion**
   - model post-review tasks explicitly in the journal
   - add dependencies between verification, notes, tag, release, and build tasks

## Open Follow-Ups

These are intentionally deferred, not unresolved blockers:

- automated CircleCI monitoring
- rollout PR generation into k8s repos
- final release promotion after RC validation
- optional dry-run mode for all mutating tasks
- support for more OP Stack components
