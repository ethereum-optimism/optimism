# OPRM Implementation Backlog

This document breaks `docs/oprm-design.md` into issue-sized implementation tasks for the MVP.

## Current Status Snapshot

The following snapshot refers to the OPRM prototype / implementation branch, not necessarily to `develop`. Treat it as branch-specific groundwork until OPRM is merged.

- `oprm` command skeleton under `./oprm/cmd/oprm` on the prototype branch
- config loading
- shell / `git` / `gh` provider abstractions
- markdown run journal
- doctor checks, including repo push-access checks and monorepo tag fetch from `origin`
- component registry
- version parsing / bumping / RC handling
- release discovery, including draft RC detection
- change detection and review-range generation
- component-selection TUI stage
- initial task TUI with `review-diff`, `local-tag`, `push-tag`, `github-draft-release`, `docker-build`, `stack.rollout`, `retry`, `skip`, and `externally-satisfied`

The backlog below remains useful as a decomposition, but some items are now partially or fully complete and should be updated if converted into GitHub issues.

## Near-Term Next Steps

The highest-value remaining implementation steps from the current state are:

1. add explicit tag/release reconciliation states to the TUI (local tag, remote tag, draft release URL)
2. harden inconsistent-state handling for tag/release mismatches
3. add external checkout support for `op-geth` so local tag/release execution works there too
4. enrich the manual build confirmation step with direct build/release links and operator notes

Scope is limited to release process steps 1-6:

1. select components
2. determine versions
3. handle `op-geth` first when selected
4. verify `develop`
5. tag, push, and create/update draft releases
6. manually confirm release builds are ready

## Suggested Labels

- `oprm`
- `release-tooling`
- `go`
- `tui`
- `github`
- `git`
- `documentation`

## Suggested Milestone Order

1. Foundation
2. Run state + journal
3. Planning engine
4. TUI shell
5. Monorepo release actions
6. `op-geth` integration
7. Hardening

---

## Epic 1: Foundation

### 1. Create the `oprm` Go command skeleton
**Size:** S  
**Depends on:** none

#### Summary
Create the initial Go command structure for the `oprm` binary in this monorepo.

#### Deliverables
- new top-level command directory for `oprm`
- `main.go`
- basic `urfave/cli/v2` app wiring
- placeholder subcommands:
  - `doctor`
  - `run`
  - `resume`
  - `status`
  - `retry`
  - `skip`
  - `satisfy`
  - `log`

#### Acceptance Criteria
- `go run ./...` still works for touched packages
- `go run <oprm-main-pkg> --help` shows all planned subcommands
- subcommands return a placeholder message instead of failing due to missing wiring

---

### 2. Add `oprm` configuration loading
**Size:** S  
**Depends on:** 1

#### Summary
Implement configuration loading with defaults, file-based config, env, and flags.

#### Deliverables
- config struct
- default config values
- optional config file at `.oprm/config.yaml`
- env overrides
- flag overrides for key values

#### Acceptance Criteria
- `runs_dir` defaults to `.oprm/releases`
- config can be loaded without a config file present
- config precedence is documented and tested

---

### 3. Implement shell/provider abstractions for `git` and `gh`
**Size:** M  
**Depends on:** 1

#### Summary
Create thin provider interfaces for invoking external tools in a testable way.

#### Deliverables
- shell runner abstraction
- git provider abstraction
- GitHub CLI provider abstraction
- fake/mock implementations for tests

#### Acceptance Criteria
- code outside provider packages does not directly shell out to `git` and `gh`
- providers can be unit tested with fakes
- stderr/stdout/exit-code handling is normalized

---

### 4. Implement `oprm doctor`
**Size:** M  
**Depends on:** 2, 3

#### Summary
Build the startup prerequisite checks and release manager detection.

#### Deliverables
- `gh` installed check
- `gh auth` check
- `git` installed check
- `git config user.name` and `user.email` resolution
- GitHub login resolution via `gh`
- human-readable output and machine-friendly status model

#### Acceptance Criteria
- `oprm doctor` reports pass/fail for each prerequisite
- detected release manager includes GitHub login, git name, and git email
- failures are actionable and clearly described

---

## Epic 2: Run State and Journal

### 5. Define the release run model
**Size:** S  
**Depends on:** 1, 2

#### Summary
Implement the in-memory model for a release run.

#### Deliverables
- run metadata types
- release manager types
- component selection types
- version proposal types
- task status types

#### Acceptance Criteria
- run model matches `docs/oprm-design.md`
- status enum includes `externally-satisfied`
- model is serializable for journal persistence

---

### 6. Implement markdown journal read/write
**Size:** M  
**Depends on:** 5

#### Summary
Create the canonical on-disk run format using markdown with YAML frontmatter.

#### Deliverables
- serialize run to markdown
- deserialize markdown to run
- append timeline entries
- stable run file naming convention

#### Acceptance Criteria
- a run can be created, saved, loaded, and appended to without data loss
- journal lives under `.oprm/releases/` by default
- corrupted or partial frontmatter yields a clear error

---

### 7. Create run lifecycle commands: `run`, `resume`, `status`, `log`
**Size:** M  
**Depends on:** 4, 5, 6

#### Summary
Implement the non-interactive lifecycle commands for creating and inspecting runs.

#### Deliverables
- create new run id
- persist new run file
- resume existing run by id
- print summarized status
- print markdown log path / contents

#### Acceptance Criteria
- `oprm run` creates a persisted run record before any mutating action
- `oprm resume <run-id>` loads the same run
- `oprm status <run-id>` shows task states
- `oprm log <run-id>` exposes the journal cleanly

---

## Epic 3: Planning Engine

### 8. Implement component registry
**Size:** M  
**Depends on:** 5

#### Summary
Add a code-backed registry for the five supported MVP components.

#### Deliverables
- registry API
- specs for:
  - `op-geth`
  - `op-node`
  - `op-batcher`
  - `kona-node`
  - `op-reth`
- change scopes
- release notes scopes
- repo/tag metadata

#### Acceptance Criteria
- all five components can be resolved by id
- spec data matches `docs/oprm-design.md`
- shared lib scopes are encoded for each component

---

### 9. Implement version parsing, bumping, and RC auto-increment
**Size:** M  
**Depends on:** 8

#### Summary
Add semver-ish release version logic including release candidates.

#### Deliverables
- parse release and RC versions
- bump patch/minor/major
- auto-increment `-rc.N`
- allow manual override values

#### Acceptance Criteria
- latest full release and latest RC can be compared
- `vX.Y.Z-rc.N` increments correctly
- invalid manual versions are rejected with clear errors

---

### 10. Implement GitHub/tag release discovery
**Size:** M  
**Depends on:** 3, 8, 9

#### Summary
Discover the latest release and latest RC for each component.

#### Deliverables
- resolve latest full release tag
- resolve latest RC tag
- support monorepo tag prefixes
- support external `op-geth` repo

#### Acceptance Criteria
- discovery works for monorepo components and `op-geth`
- missing-release cases are handled gracefully
- results feed version proposal logic

---

### 11. Implement component change detection
**Size:** M  
**Depends on:** 8, 10

#### Summary
Determine whether a component changed since its last full release.

#### Deliverables
- path-based diffing against latest release tag
- shared dependency path inclusion
- initial reasoning output for why a component is considered changed

#### Acceptance Criteria
- direct changes under a component path are detected
- shared library changes for in-scope dependencies are detected
- docs-only changes outside the dependency graph do not trigger release-needed

---

### 12. Implement release plan generation
**Size:** M  
**Depends on:** 8, 9, 10, 11

#### Summary
Build a plan from selected components and discovered versions.

#### Deliverables
- component selection model
- proposed version generation
- ordering constraints
- per-component plan summary

#### Acceptance Criteria
- `op-geth` is ordered before `op-node` when both are selected
- unchanged components can be excluded or marked no-release
- plan is persisted to the run journal

---

### 13. Implement task DAG and task statuses
**Size:** M  
**Depends on:** 5, 12

#### Summary
Turn the release plan into a dependency-aware task graph.

#### Deliverables
- task graph data structure
- dependency resolution
- valid status transitions
- generic task execution envelope

#### Acceptance Criteria
- blocked/ready tasks are computed correctly
- retries do not duplicate dependency edges
- task graph can be rendered in status output

---

### 14. Implement `retry`, `skip`, and `externally-satisfied` state transitions
**Size:** S  
**Depends on:** 13

#### Summary
Add workflow control operations for recovering from failures and external actions.

#### Deliverables
- retry command behavior
- skip command behavior with reason capture
- satisfy command behavior with reason capture
- journal entries for all overrides

#### Acceptance Criteria
- transitions are validated against current task state
- reason is required for skip and satisfy
- run state persists correctly after transitions

---

## Epic 4: TUI Shell

### 15. Build the initial TUI application shell
**Size:** M  
**Depends on:** 7, 13

#### Summary
Create the Bubble Tea app shell for viewing and operating a run.

#### Deliverables
- root TUI model
- header / summary panel
- task list panel
- detail panel
- keybinding footer

#### Acceptance Criteria
- a run can be opened in the TUI
- task selection updates detail view
- app can quit cleanly without corrupting run state

---

### 16. Add confirmation UI for mutating task actions
**Size:** M  
**Depends on:** 15

#### Summary
Require explicit confirmation before mutating actions.

#### Deliverables
- reusable confirmation dialog/component
- action preview text
- cancel/confirm flow

#### Acceptance Criteria
- no mutating action runs without confirmation
- confirmation copy clearly states what will happen
- cancellations are logged when appropriate

---

### 17. Add run planning UI: select components and approve versions
**Size:** M  
**Depends on:** 12, 15

#### Summary
Support initial release plan construction through the TUI.

#### Deliverables
- component selection UI
- version suggestion UI
- bump choice UI: patch/minor/major/manual
- final plan confirmation screen

#### Acceptance Criteria
- operator can select any subset of supported components
- operator can override auto-proposed versions manually
- confirmed plan is persisted before task execution begins

---

## Epic 5: Monorepo Release Actions

### 18. Implement repository state verification task(s)
**Size:** M  
**Depends on:** 3, 13

#### Summary
Add pre-tag checks to verify repo state before release actions.

#### Deliverables
- clean working tree check
- base branch verification (`develop` by default)
- fetch/update verification
- commit SHA reporting

#### Acceptance Criteria
- tag/release tasks remain blocked if verification fails
- operator sees clear evidence of current repo state
- successful verification is journaled

---

### 19. Port release notes generation logic into Go
**Size:** L  
**Depends on:** 8, 10, 12

#### Summary
Port the release notes path-scoping logic currently used by `just release-notes`.

#### Deliverables
- monorepo component include-path definitions in Go
- git range resolution
- release note body generation helper
- artifact output under run directory

#### Acceptance Criteria
- generated release notes match current conventions closely enough for operator review
- notes are generated for `op-node`, `op-batcher`, `kona-node`, and `op-reth`
- release notes artifacts are stored under the run directory

---

### 20. Implement tag reconciliation logic
**Size:** M  
**Depends on:** 18, 19

#### Summary
Detect whether a tag already exists and reconcile that state safely.

#### Deliverables
- local tag existence check
- remote tag existence check
- create/update/no-op decision model
- explicit blocked state for inconsistent cases

#### Acceptance Criteria
- handles the four cases in the design doc
- rerunning after partial completion is safe
- operator sees whether `oprm` is creating or reconciling

---

### 21. Implement draft GitHub release reconciliation logic
**Size:** M  
**Depends on:** 3, 19, 20

#### Summary
Detect and reconcile draft releases on GitHub.

#### Deliverables
- draft release lookup by tag/version
- create draft release
- update existing draft release
- detect inconsistent tag/release states

#### Acceptance Criteria
- operator is told whether an existing draft is being updated or a new one is being created
- existing draft release detection works for supported components
- release body comes from generated notes or approved content

---

### 22. Implement monorepo tag creation and push tasks
**Size:** M  
**Depends on:** 20

#### Summary
Add the mutating git steps for creating and pushing tags.

#### Deliverables
- annotated tag creation
- remote push
- verification after push
- journal entries before and after mutation

#### Acceptance Criteria
- tag creation is confirmation-gated
- pushed tag is verified remotely
- rerun after successful tag push becomes no-op or externally-satisfied

---

### 23. Implement monorepo draft release create/update tasks
**Size:** M  
**Depends on:** 21, 22

#### Summary
Create or update GitHub draft releases for monorepo components.

#### Deliverables
- release creation task
- release update task
- release URL capture
- journal integration

#### Acceptance Criteria
- draft releases can be created for `op-node`, `op-batcher`, `kona-node`, and `op-reth`
- release URL is shown in UI and saved in log
- rerun is reconciliation-safe

---

### 24. Add manual CircleCI confirmation task
**Size:** S  
**Depends on:** 23

#### Summary
Implement the manual operator checkpoint for verifying builds/images are ready.

#### Deliverables
- task description template
- optional operator notes field
- confirmation action

#### Acceptance Criteria
- task includes component and tag context
- operator can add freeform notes before marking complete
- completion is journaled with timestamp and operator identity

---

## Epic 6: `op-geth` Integration

### 25. Add external repository support for `op-geth`
**Size:** M  
**Depends on:** 3, 8, 10

#### Summary
Support release operations against `ethereum-optimism/op-geth`.

#### Deliverables
- external repo configuration
- git operations in external repo context
- GitHub release discovery in external repo context

#### Acceptance Criteria
- `op-geth` latest release and latest RC can be discovered
- `op-geth` tasks can operate without confusing monorepo state with external repo state
- repo context is explicit in logs and UI

---

### 26. Implement `op-geth` release notes generation
**Size:** M  
**Depends on:** 25

#### Summary
Generate draft release notes for `op-geth` from the external repo.

#### Deliverables
- git range discovery for `op-geth`
- release note generation helper for external repo
- artifact output for notes

#### Acceptance Criteria
- release notes can be generated from `op-geth` repo history
- generated output is stored under the run artifact directory
- task integrates with the same draft release flow model

---

### 27. Implement `op-geth` tag + draft release tasks
**Size:** M  
**Depends on:** 25, 26

#### Summary
Enable RC tagging and draft release creation/updating for `op-geth`.

#### Deliverables
- external repo tag creation/push
- external repo draft release create/update
- reconciliation logic reused or adapted from monorepo path

#### Acceptance Criteria
- `op-geth` can be released as part of the same run
- release URLs are captured in the journal
- reruns are safe and state-aware

---

### 28. Detect `op-node` dependency on released `op-geth` version
**Size:** M  
**Depends on:** 8, 25, 27

#### Summary
Detect whether `op-node` needs its `go.mod` updated to reference the selected `op-geth` RC.

#### Deliverables
- read current `go.mod`
- find current `op-geth` replacement/reference
- compare with selected `op-geth` version

#### Acceptance Criteria
- task reports whether `op-node` is already aligned with the selected `op-geth` version
- no-op path works if update is already present
- evidence is shown to operator before mutation

---

### 29. Implement `op-node` `go.mod` update and `go mod tidy`
**Size:** M  
**Depends on:** 28

#### Summary
Perform the `op-node` dependency update after `op-geth` RC selection.

#### Deliverables
- update `go.mod`
- run `go mod tidy`
- verify diff
- capture resulting file changes

#### Acceptance Criteria
- action is confirmation-gated
- resulting diff only includes expected dependency changes
- failure leaves enough evidence for retry

---

### 30. Implement branch, commit, and push flow for the `op-node` dependency update
**Size:** M  
**Depends on:** 29

#### Summary
Automate the git branch workflow for the `op-geth` dependency update PR.

#### Deliverables
- branch naming strategy
- branch creation
- commit creation
- remote push

#### Acceptance Criteria
- branch name is deterministic or clearly derived from run/component/version
- commit message is consistent and understandable
- push result is verified and journaled

---

### 31. Implement PR creation to `develop` for the `op-node` dependency update
**Size:** M  
**Depends on:** 30

#### Summary
Create a PR from the update branch into `develop`.

#### Deliverables
- PR title/body template
- PR creation via `gh`
- PR URL capture

#### Acceptance Criteria
- PR creation is confirmation-gated
- created PR targets `develop`
- PR URL is shown in the UI and recorded in the journal

---

### 32. Implement wait-for-PR-confirmation task
**Size:** S  
**Depends on:** 31

#### Summary
Add a manual checkpoint after PR creation so the operator can confirm merge/readiness before continuing with `op-node` release tasks.

#### Deliverables
- task for manual confirmation
- optional notes/reason capture
- ability to refresh/re-detect merge state later

#### Acceptance Criteria
- `op-node` release tasks remain blocked until operator confirms readiness
- operator can mark task externally satisfied if the PR was handled elsewhere
- decision is persisted to journal

---

## Epic 7: Hardening

### 33. Add unit tests for version and task-state logic
**Size:** M  
**Depends on:** 9, 13, 14

#### Summary
Add focused tests for the most error-prone pure logic.

#### Deliverables
- version parsing tests
- bumping tests
- RC increment tests
- task transition tests
- DAG dependency tests

#### Acceptance Criteria
- all critical pure logic has coverage
- invalid transitions are tested explicitly

---

### 34. Add integration tests for git/tag/release reconciliation
**Size:** L  
**Depends on:** 20, 21, 22, 23

#### Summary
Exercise the release flow with fake repos/providers.

#### Deliverables
- fake repo fixtures
- tag exists/release missing case
- tag missing/release exists case
- rerun-after-partial-success case

#### Acceptance Criteria
- reconciliation cases from the design doc are covered
- reruns do not duplicate successful state
- failures surface enough context to debug

---

### 35. Add integration tests for the `op-geth` -> `op-node` update flow
**Size:** L  
**Depends on:** 28, 29, 30, 31, 32

#### Summary
Test the special cross-repo dependency update path end-to-end with mocks/fakes.

#### Deliverables
- external repo fixture for `op-geth`
- monorepo fixture for `op-node`
- PR creation mock
- resume-after-failure scenario

#### Acceptance Criteria
- flow can pause and resume around the PR step
- `go.mod` update logic is verified
- branch/commit/PR metadata is journaled correctly

---

### 36. Improve operator-facing errors and logging
**Size:** S  
**Depends on:** 23, 27, 31

#### Summary
Polish the operator UX for failures and recoverable states.

#### Deliverables
- consistent error formatting
- remediation hints
- log messages for reconciliation decisions

#### Acceptance Criteria
- common failure modes show next-step guidance
- logs clearly distinguish detect/plan/execute/verify phases

---

### 37. Write operator documentation for `oprm`
**Size:** M  
**Depends on:** 24, 32, 36

#### Summary
Document usage, configuration, and recovery workflows.

#### Deliverables
- command usage doc
- run/resume/retry/skip/satisfy doc
- config doc
- run journal doc

#### Acceptance Criteria
- a release manager can follow the docs to start a new run and resume a failed one
- docs explain manual CircleCI confirmation in v1
- docs explain `skip` vs `externally-satisfied`

---

## Recommended Next Issues to Open

If you want to sequence the project pragmatically from the current implementation state, open these next:

1. Add a preflight verification task after `component.review-diff`
2. Port monorepo release-notes generation into Go and surface it in the TUI
3. Add tag reconciliation / create / push tasks for monorepo components
4. Add draft GitHub release reconciliation / create / update tasks for monorepo components
5. Add a manual build-confirmation task after draft release creation/update
6. Improve TUI ergonomics:
   - scrolling details pane
   - open compare URL
   - clearer badges for `resume draft`
7. Add task-DAG persistence for post-review mutating tasks
8. Add `op-geth` external-repo tag / draft-release execution tasks
9. Add `op-geth` -> `op-node` `go.mod` / PR workflow
10. Add integration tests for tag/release reconciliation and resume flows

In other words: the project has moved beyond foundation work and the next milestone is turning the review workflow into a real release execution workflow.

## Suggested Stretch Split for Large Issues

These tasks are likely to be the biggest and may need to be split further during implementation:

- 19. Port release notes generation logic into Go
- 34. Add integration tests for git/tag/release reconciliation
- 35. Add integration tests for the `op-geth` -> `op-node` update flow

## Notes on Parallelism

Work that can likely proceed in parallel after the foundations land:

- TUI shell work can start once run state and task graph basics are stable.
- Monorepo release-note generation can proceed in parallel with journal/status command work.
- `op-geth` external repo support can proceed in parallel with monorepo tag/release reconciliation work.
