# Pipeline model — design doc

Status: draft / not yet implemented. Describes a future refactor of the
graph + selector that treats CI as a uniform dataflow pipeline. See
"Migration" below for the phased path from today's model.

## Why

Today's graph mixes several orthogonal mechanisms:

- `imports` edges (source→source, from adapters)
- `tested_by` edges (source→check, from builder — "run this check
  against this source")
- `prerequisite` edges (check→check, from catalog — "run A before B")
- `generates` edges (source→artifact/source, for things like
  `src/Foo.sol → interfaces/IFoo.sol`)
- `produces` edges (check→source, from catalog's `produces:` field —
  "this check validates the freshness of this file")
- `triggers:` field (check→path glob, for "fire this check on these
  paths")
- `blast_radius_patterns` policy (path globs that trigger everything)
- `profiles.triggers` (per-profile variant of `triggers`)

Each mechanism was added to solve a specific gap, and they overlap
awkwardly:

- Per-check `triggers:` and `inputs:` of a conceptual pipeline step
  are the same thing.
- `prerequisite` (coarse, check→check) and the `produces`-driven
  per-file prerequisites (fine, discovered at walk time) say the same
  thing at two different granularities.
- `blast_radius_patterns` is a blunt workaround for toolchain/config
  files that in a cleaner model would be ordinary inputs to the checks
  that actually depend on them.
- `generates` edges are shorthand for "some implicit regeneration
  step produces this artifact", but that step isn't modeled as a
  node — it's elided.

Adding new kinds of derived artifacts (Go bindings, cannon prestates,
snapshot JSON) requires picking from this menu case-by-case.

The pipeline model collapses all of the above into one shape: every
check has explicit inputs and outputs, and selection is a dataflow
walk over those edges.

## Target model

### Node kinds

| kind | contents | examples |
|---|---|---|
| `source:` | handwritten files in the repo | `source:src/L1/OptimismPortal2.sol`, `source:op-node/rollup/derive.go` |
| `artifact:` | generated data, whether checked-in or transient | `artifact:forge-artifacts/OptimismPortal2.sol/OptimismPortal2.json`, `artifact:interfaces/L1/IOptimismPortal2.sol`, `artifact:op-e2e/bindings/optimismportal2.go`, `artifact:toolchain/forge`, `artifact:toolchain/go` |
| `check:` | a runnable unit of CI | `check:forge-build`, `check:gen-go-bindings`, `check:mise-setup` |
| `module:` | external code (unchanged) | `mod:github.com/stretchr/testify` |

`source:` absorbs today's `sol:`/`go:`/`rs:` — the language becomes a
property (`properties.language`). Rationale: the pipeline model cares
about dataflow, not language origin. Path + extension determines
adapter behavior, not the node kind.

Artifact nodes are used for three distinct things, all handled
uniformly:

1. **Transient artifacts** (not in repo): `forge-artifacts/*.json`,
   Go test binaries, coverage reports.
2. **Checked-in generated files**: `interfaces/**/*.sol`,
   `op-e2e/bindings/**/*.go`, `snapshots/**/*.json`.
3. **Toolchain state**: `artifact:toolchain/forge`,
   `artifact:toolchain/mise`, etc. — virtual nodes representing the
   ambient tool versions declared in `mise.toml` /
   `.circleci/continue/*.yml`.

For (2), the file has a single node of kind `artifact:`. Tests that
import `interfaces/IFoo.sol` have an `imports` edge to
`artifact:interfaces/IFoo.sol`. No separate source/artifact split for
the same file.

### Edge kinds

| kind | direction | meaning |
|---|---|---|
| `consumes` | check → source/artifact | check reads this |
| `produces` | check → artifact | check writes this |
| `imports` | source → source/artifact | structural code dependency (stays) |

Derived / eliminated:

- `prerequisite` disappears — "A must run before B" is derivable: if A
  produces X and B consumes X, A runs first.
- `tested_by` disappears — replaced by `consumes`.
- `generates` disappears — every `src → generated` is mediated by a
  generator check with `consumes: src`, `produces: generated`.
- `observed_correlation` / `ai_annotated` stay as signal sources, not
  dataflow.

Implementation note: the existing walk follows `imports` (reverse) and
`generates` (forward); the new walk follows `consumes` (reverse) and
`produces` (forward). Same shape, different edge kinds.

## Catalog schema

```yaml
- id: <check-id>
  name: <human name>
  command: <shell command>
  scopeable: <bool>
  scope_flag: <cli flag, optional>
  scope_type: paths|packages|crates

  inputs:
    - <path glob OR artifact ref>
  outputs:
    - <path glob OR artifact ref>

  tools:            # sugar: expands into inputs: [artifact:toolchain/X]
    - <tool name>

  # Unchanged fields:
  kind, language, avg_duration, per_unit_duration, knobs, ci_job_names
```

Inputs and outputs accept:
- **Path globs** (resolved at build time against the file tree):
  `"packages/contracts-bedrock/src/**/*.sol"`.
- **Artifact refs** (declared, don't need to exist on disk):
  `"artifact:forge-artifacts/**"`, `"artifact:toolchain/forge"`.

Examples:

```yaml
- id: mise-setup
  command: "mise install"
  inputs:
    - "mise.toml"
  outputs:
    - "artifact:toolchain/forge"
    - "artifact:toolchain/go"
    - "artifact:toolchain/cargo"
    - "artifact:toolchain/rust"
    - "artifact:toolchain/node"
    - "artifact:toolchain/pnpm"

- id: forge-build
  command: "cd packages/contracts-bedrock && mise exec -- forge build"
  tools: [forge, mise]
  inputs:
    - "packages/contracts-bedrock/src/**/*.sol"
    - "packages/contracts-bedrock/foundry.toml"
  outputs:
    - "artifact:forge-artifacts/**/*.json"

- id: interfaces-regen    # currently doesn't exist as a separate check
  command: "cd packages/contracts-bedrock && mise exec -- just interfaces-snapshots"
  tools: [forge, mise]
  inputs:
    - "artifact:forge-artifacts/**/*.json"
  outputs:
    - "artifact:interfaces/**/*.sol"

- id: interfaces-check
  command: "cd packages/contracts-bedrock && mise exec -- just interfaces-check-no-build"
  tools: [forge, mise]
  inputs:
    - "artifact:forge-artifacts/**/*.json"
    - "artifact:interfaces/**/*.sol"
  outputs: []

- id: gen-go-bindings
  command: "cd op-e2e && just gen-bindings-all"
  tools: [go, jq]
  inputs:
    - "artifact:forge-artifacts/**/*.json"
  outputs:
    - "artifact:op-e2e/bindings/**/*.go"
    - "artifact:op-service/txintent/bindings/**/*.go"

- id: forge-test
  command: "cd packages/contracts-bedrock && mise exec -- forge test"
  tools: [forge, mise]
  scopeable: true
  scope_flag: "--match-path"
  scope_type: paths
  inputs:
    - "packages/contracts-bedrock/{src,test,interfaces}/**/*.sol"
    - "artifact:forge-artifacts/**/*.json"
  outputs: []

- id: go-test
  command: "go test"
  tools: [go]
  scopeable: true
  scope_type: packages
  inputs:
    - "**/*.go"
    - "go.mod"
    - "go.sum"
    - "artifact:op-e2e/bindings/**/*.go"
  outputs: []
```

## Selection algorithm

Input: a diff (set of changed source paths).

```
invalidated = { source:<path> for each changed path }
selected    = {}
queue       = list(invalidated)

while queue not empty:
  node = queue.pop()
  for edge in g.EdgesTo(node) where kind == "consumes":
    check = edge.From
    if check not in selected:
      selected.add(check)
      # All of check's outputs are now stale
      for out_edge in g.EdgesFrom(check) where kind == "produces":
        artifact = out_edge.To
        if artifact not in invalidated:
          invalidated.add(artifact)
          queue.push(artifact)

# Ordering: topo-sort on produces/consumes edges.
# Scoping: unchanged — coverage / import walk still picks per-test-file
# or per-package scope for scopeable checks. The selection step says
# WHICH checks; the scoping step says WHAT SCOPE within each check.
```

Scoping stays orthogonal. Coverage and import edges still drive which
specific test files / packages / crates run under a selected check.
The pipeline walk answers "is this check selected at all"; the scope
walk answers "with what args".

## Concrete behavioral changes vs today

1. **Blast radius shrinks.** `mise.toml` change no longer fires every
   check — it fires only checks that declare that tool. Same for
   `Dockerfile` (per-image scoping), `.circleci/*.yml` (per-workflow
   scoping if we model CI config as producing workflow-artifacts),
   `foundry.toml` (forge-* only).

2. **Generator checks become selectable.** `gen-go-bindings` can be
   added as a catalog check_type; it's selected whenever a bindings
   consumer is selected. Today there's no mechanism for this without
   special casing.

3. **Transitive artifact staleness.** A diff to `src/Foo.sol`
   propagates through forge-artifacts, interfaces, bindings, snapshots
   as explicit invalidations. Today we have several overlapping
   mechanisms (generates edges, produces edges, triggers) that
   approximate this piecewise.

4. **Simpler mental model for new check_types.** Declare (inputs,
   outputs, command) — everything else follows. No choice to make
   between triggers / prerequisites / produces / blast.

## Non-changes

- Scoping logic (coverage, import walk, per-crate / per-package) stays
  unchanged. These drive scope *within* a selected check; pipeline
  dataflow drives *which* checks are selected.
- Knob policy, stage miss_cost, tier thresholds all stay.
- Recall on the 500-pipeline calibration is expected to hold at 29/29.
  Selection answers should be equivalent to today's for every
  migrated check.
- Execution (the `checks run` command) stays layered by prereq; the
  layering just derives from the dataflow DAG instead of an explicit
  prereq edge.

## Migration

The refactor is large enough to need phasing. Each phase is
independently verifiable against the calibration sample.

### Phase A — schema only, behavior-preserving

- Add `inputs:` and `outputs:` as *optional* fields on CheckType.
- Builder: when `inputs:`/`outputs:` are present, emit `consumes` and
  `produces` edges alongside the existing `triggers:` /
  `prerequisites:` / `produces:` / `tested_by` mechanisms.
- Selector: add a new `selectViaDataflow` function that uses only
  consumes/produces. Run it in parallel with the existing selector on
  every calibration diff; assert identical candidate sets.
- No user-visible change. Proves the model is faithful.

Exit criteria: dataflow-only selection matches existing selection on
every diff in the 500-pipeline sample, including blast-radius
scenarios.

### Phase B — migrate check_types one at a time

For each check_type in the catalog:

1. Translate `triggers:` / `prerequisites:` / `produces:` into
   `inputs:` / `outputs:`.
2. Delete the old fields for this check.
3. Re-run calibration; verify recall and savings unchanged.
4. Commit.

Rough order (cheapest to most impactful):

1. `mise-setup` (new check_type). Introduces `artifact:toolchain/*`
   nodes. Other checks gain `tools: [...]` entries one at a time.
2. Pure validators with no outputs: `lint-forge-tests`, `semgrep`,
   `go-mod-tidy`, `snapshots-check`, `semver-check`, `validate-spacers`,
   `reinitializer-check`.
3. `forge-build` (gains `artifact:forge-artifacts/**` outputs).
4. Everything that consumes forge-artifacts: `interfaces-check`,
   `forge-test`, `gen-go-bindings` (new).
5. Go/Rust checks gain explicit tool inputs.
6. `.circleci/*.yml` becomes inputs to per-workflow check_types;
   `blast_radius_patterns` loses entries one at a time until the
   policy file's `blast_radius` is empty.

### Phase C — cleanup

Once every check_type uses the new schema:

- Delete `CheckType.Triggers`, `CheckType.Prerequisites`,
  `CheckType.Produces`, `TestProfile.Triggers`.
- Delete `blast_radius_patterns` from policy.
- Delete `EdgePrerequisite`, `EdgeGenerates`, `EdgeTestedBy`,
  `EdgeProduces` (the check→source variant) — replaced by `consumes`
  and `produces` (check→artifact).
- Consolidate walks: one `walkDataflow` function replaces
  `walkForScope`, `importScopeCandidates`, `profileTriggerCandidates`,
  `blastRadiusCandidates`. Scoping logic stays untouched.

### Phase D — new check_types that the old model couldn't express

After the refactor is complete, several latent gaps become easy to
close:

- `gen-go-bindings` — regenerate Go bindings from forge artifacts.
  Selected whenever a bindings consumer is selected.
- `interfaces-regen` — regenerate `interfaces/**` from forge
  artifacts. Distinct from `interfaces-check` (validation).
- `cannon-prestate-regen` — rebuild cannon prestates from
  op-program + cannon source.
- `snapshots-regen` — regenerate ABI / storage-layout / semver-lock
  JSON.

Each of these today requires special-case plumbing; in the new model
they're just catalog entries.

## Open questions

1. **CI job model.** CI today runs jobs in parallel with job-level
   `requires:`. Our per-file dataflow is finer than that. When we
   generate an execution plan for a CI run, do we collapse back to
   job granularity, or do we propose finer-grained CI orchestration?
   Probably collapse for now; finer granularity is a later axis.

2. **Content-hash identity.** Artifact identity could be path-only
   (what the refactor assumes) or content-hashed (enables skip-if-
   unchanged caching). Start with path-only; content-hashing is a
   separate future feature that layers on top.

3. **Nondeterministic outputs.** The pipeline model assumes a check's
   outputs are a function of its inputs. Flaky tests or tests with
   timestamp-dependent artifacts break that. For selection purposes,
   this doesn't matter (we're not caching yet); but it would matter
   later.

4. **Multiple producers of the same artifact.** What if two checks
   both declare `outputs: [artifact:X]`? Catalog validation should
   reject this. But some artifacts are genuinely produced by multiple
   checks (e.g. coverage reports might be produced by
   `forge-test-coverage` and `go-test-coverage` under different
   namespaces). Disambiguate via namespacing in the artifact ref.

5. **Coverage reports as artifacts.** The selector reads coverage at
   build time for scoping. If coverage is modeled as a pipeline
   artifact, there's a cycle (forge-test consumes coverage to decide
   what to run, but produces coverage to update it). In practice the
   selector uses *past* coverage to decide present runs, so there's no
   real cycle — but it's worth documenting that coverage sits slightly
   outside the dataflow model (or on a lag of one pipeline).

6. **`triggers:` as sugar or removal?** During migration, `triggers:`
   entries translate directly to `inputs:` entries. Do we keep
   `triggers:` as sugar post-migration for readability, or just use
   `inputs:` everywhere? My instinct: one field is better than two.

7. **Per-profile artifacts.** Today each Solidity feature profile
   runs the same tests under different env vars. A profile doesn't
   produce different files, but it does exercise different code paths
   and produce different coverage. Does coverage-per-profile need to
   be a different artifact node? Probably: `artifact:coverage/forge-
   test:opcm_v2.json` vs `artifact:coverage/forge-test:main.json`.

8. **External contract bindings.** `op-e2e/bindings/safe.go` etc. are
   for external contracts whose src isn't in the repo. They have no
   upstream `src/` node. In the pipeline model they're unsourced
   artifacts — consumers still import them, but nothing invalidates
   them from a diff. Probably fine; they'd only become relevant if
   `lib/` submodule versions shifted.

## Sizing

Order-of-magnitude estimate for a complete implementation:

- Schema + catalog migration: ~300 LoC
- Builder changes (emit consumes/produces from globs): ~200 LoC
- Dataflow walker: ~150 LoC (simpler than current walks combined)
- Selector glue (assemble candidates from walker, retain scoping):
  ~100 LoC
- Cleanup / deletions in Phase C: net negative LoC
- Tests: ~400 LoC

~1000 LoC net change. Most of the effort is catalog migration and
verification against the calibration sample, not the code changes
themselves.
