# gen-flags — generated flag references for docs.optimism.io

Generates the MDX flag-reference tables under `docs/public-docs/snippets/generated/`
from the urfave/cli flag definitions the OP Stack Go services actually register.
The docs pages import these snippets instead of hand-transcribing `--help`
output, so the published flag catalogs cannot silently fall behind a release.

## Usage

Run from the monorepo root:

```bash
go run ./docs/public-docs/scripts/gen-flags
```

The monorepo Go build embeds `op-core/superchain/superchain-configs.zip`
(gitignored). On a fresh checkout, build it first:

```bash
just build-superchain-go
```

## How it works

- `main.go` imports each component's composed flag slice (for op-batcher,
  `op-batcher/flags.Flags` — including the op-service flag families appended in
  its `init()`: rpc, log, metrics, pprof, txmgr, altda) and renders one snippet
  per component: `snippets/generated/<component>-flags.mdx`.
- Flag name, usage, default, and environment variable come from urfave/cli's
  own `DocGenerationFlag` accessors, so the output matches `--help` semantics.
  Hidden flags are skipped.
- `manifest.json` records the release tag named in each snippet's provenance
  line (e.g. `op-batcher/v1.16.11`). Bump it when a new component release is
  cut; the drift check keeps the table itself in sync with the source tree at
  every commit, tag or not.
- The drift check (`.github/workflows/docs-flag-drift.yml`) regenerates the
  snippets and fails on any diff, so a change to a component's flag
  definitions must land together with the regenerated snippet, and hand edits
  to generated files always fail CI.

## Adding a component

1. Add the component's flag slice to the `components` table in `main.go`,
   mirroring its required-flag names (the components keep their required
   slices unexported; a runtime sanity check catches renames).
2. Add its release tag to `manifest.json`.
3. Run the generator and import the new snippet from the component's
   reference page.
4. Add the component's flag-definition path to the workflow's path filters.

## Ownership

Per the ownership model in the Solutions repo's
`projects/docs-improver/plans/option-b-truth-from-source.md`:

| Artifact | Author of record | Reviewer | Drift triage |
| --- | --- | --- | --- |
| Generator code + CI job (this directory, `.github/workflows/docs-flag-drift.yml`) | Matthew Cruz (@sbvegan), docs owner — proposed, pending confirmation | @ethereum-optimism/solutions (via CODEOWNERS) | Author of record |
| Generated snippets (`snippets/generated/`) | The pipeline — nobody hand-edits; hand edits fail the drift check by construction | — | Unresolved drift is filed as an `accuracy`-labelled issue on the Solutions board |
| Flag facts (`op-batcher/flags/`, op-service flag families) | Component engineers — the drift check fails on the PR that changes the flags, so the docs delta lands in the same change | Docs team reviews | Component team |

Known residual gaps (accepted, by design):

- The drift workflow is path-filtered to `op-batcher/flags/` and the docs
  paths. A flag change made purely inside an op-service flag family
  (rpc/log/metrics/pprof/txmgr/altda) does not trigger it on that PR; the next
  triggering PR or manual `workflow_dispatch` run surfaces the drift.
- The required-flag list is mirrored in `main.go` because the components do
  not export it. A rename fails the generator's sanity check; adding a brand
  new required flag without updating the mirror would list it as optional.
