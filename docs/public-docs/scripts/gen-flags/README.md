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

To verify the committed snippets match what the current source tree generates
without rewriting them (exits nonzero on any difference):

```bash
go run ./docs/public-docs/scripts/gen-flags -check
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
- `manifest.json` records the finalized release tag each snippet was generated
  at (e.g. `op-batcher/v1.16.11`), and the snippet's provenance line names the
  same tag. The two are updated together, only on regeneration.

## Regenerating for a new release

The snippets document *released* software, so regeneration is keyed to
finalized release tags — not to every change on `develop`. When a new
finalized (non-rc, non-pre-release) `op-batcher/v*` tag is published:

1. Check out that tag — or, if working from another commit, verify the
   flag-defining packages are identical to the tag first:
   `op-batcher/flags/` and the flag families it appends (`op-service/rpc`,
   `op-service/log`, `op-service/metrics`, `op-service/oppprof`,
   `op-service/txmgr`, `op-alt-da`), e.g. with
   `git diff <tag> HEAD -- <paths>`.
2. Update the component's `tag` in `manifest.json` to the new release tag.
3. Run the generator; the snippet and its provenance line are rewritten
   together. Commit the manifest and snippet in the same change.

Never hand-edit the generated snippets (`-check` fails on any hand edit), and
never regenerate from unreleased code: if the flags changed since the manifest
tag, the regenerated table would document behavior no released binary has.

Enforcement runs as a review-gated Mintlify docs automation on a weekly
schedule: it compares the newest finalized `op-batcher/v*` tag against the
tag in `manifest.json` and, when a newer release exists, regenerates per the
steps above and proposes the change for human review. Local runs of the
generator (`-check` for verification) cover the gap between scheduled runs.

## Adding a component

1. Add the component's flag slice to the `components` table in `main.go`,
   mirroring its required-flag names (the components keep their required
   slices unexported; a runtime sanity check catches renames).
2. Add its latest finalized release tag to `manifest.json`.
3. Run the generator and import the new snippet from the component's
   reference page.
4. Extend the docs automation prompt to watch the new component's release
   tag pattern.

## Ownership

Everything in this pipeline lives under `docs/public-docs/`, so it is covered
by the existing CODEOWNERS rule
(`/docs/public-docs/  @ethereum-optimism/solutions`) — no dedicated entries
are needed. Per the ownership model in the Solutions repo's
`projects/docs-improver/plans/option-b-truth-from-source.md`:

| Artifact | Author of record | Reviewer | Stale-reference triage |
| --- | --- | --- | --- |
| Generator code + docs automation (this directory, the Mintlify automation config) | Matthew Cruz (@sbvegan), docs owner — proposed, pending confirmation | @ethereum-optimism/solutions (via the `/docs/public-docs/` CODEOWNERS rule) | Author of record |
| Generated snippets (`snippets/generated/`) | The pipeline — nobody hand-edits; `-check` fails on hand edits by construction | @ethereum-optimism/solutions review the automation's regeneration PRs | A snippet that can't be regenerated cleanly is filed as an `accuracy`-labelled issue on the Solutions board |
| Flag facts (`op-batcher/flags/`, op-service flag families) | Component engineers | Component team | Component team; the docs table follows at the next finalized release |

Known residual gaps (accepted, by design):

- The table documents the manifest release tag, not `develop`. Flag changes
  merged after a release are intentionally not reflected until the next
  finalized tag is published and the automation (or a maintainer) regenerates.
- The required-flag list is mirrored in `main.go` because the components do
  not export it. A rename fails the generator's sanity check; adding a brand
  new required flag without updating the mirror would list it as optional.
- op-batcher's `--compressor` option list (`compressor.KindKeys`) is built
  from map iteration, so its order changes on every process start — in
  `op-batcher --help` itself, not just here. The generator sorts that one
  list (`canonicalizeUsage` in `main.go`) so output is deterministic; the
  proper fix is sorting `KindKeys` at the source (one line in
  `op-batcher/compressor/compressors.go`), proposed as a follow-up.
