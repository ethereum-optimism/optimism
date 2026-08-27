# gen-flags — generated flag references for docs.optimism.io

Generates the MDX flag-reference tables under `docs/public-docs/snippets/generated/`
from the urfave/cli flag definitions the OP Stack Go services actually register.
The docs pages import these snippets instead of hand-transcribing `--help`
output, so the published flag catalogs cannot silently fall behind a release.

Covered components: op-batcher, op-node, op-proposer, op-challenger,
op-conductor — every first-party Go service with a flag reference page.

## Usage

Run from the monorepo root:

```bash
go run ./docs/public-docs/scripts/gen-flags
```

To limit a run to specific components (used when regenerating one component
at its release tag):

```bash
go run ./docs/public-docs/scripts/gen-flags -only op-node
```

To verify the committed snippets without rewriting them (exits nonzero on any
problem):

```bash
go run ./docs/public-docs/scripts/gen-flags -check
```

The monorepo Go build embeds `op-core/superchain/superchain-configs.zip`
(gitignored). On a fresh checkout, build it first:

```bash
just build-superchain-go
```

## How it works

- `main.go` imports each component's composed flag slice (e.g. for op-batcher,
  `op-batcher/flags.Flags` — including the op-service flag families appended in
  its `init()`) and renders one snippet per component:
  `snippets/generated/<component>-flags.mdx`.
- Flag name, usage, default, and environment variable come from urfave/cli's
  own `DocGenerationFlag` accessors, so the output matches `--help` semantics.
  Hidden flags (including the deprecated op-node flags) are skipped.
- `manifest.json` records, per component, the finalized release tag each
  snippet was generated at (e.g. `op-batcher/v1.16.11`) and the SHA-256 of the
  snippet bytes written at generation time. The snippet's provenance line
  names the same tag. All three are updated together, only on regeneration.

### What `-check` verifies

The snippets document *released* software: each is generated from the source
at its manifest tag, and the components release independently, so the
flag-defining sources on `develop` routinely move past one component's tag
while matching another's. Byte-comparing a committed snippet against a
regeneration from an arbitrary tree is therefore only meaningful at (or at
parity with) that component's tag.

`-check` verifies what can be verified at every commit: that each committed
snippet matches the SHA-256 recorded in `manifest.json` — any hand edit to a
snippet (or a snippet/manifest mismatch) fails. It also reports, per
component, whether the current tree still generates identical bytes; a
mismatch there is informational ("the source has moved past the tag"), not a
failure — it resolves when the next finalized release is published and the
snippet is regenerated.

## Regenerating for a new release

When a new finalized (non-rc, non-pre-release) release tag of a component is
published:

1. Check out that tag and regenerate from it — or, if working from another
   commit, verify the flag-defining packages are identical to the tag first
   (`git diff <tag> HEAD -- <paths>`): the component's `flags/` package (and,
   for op-challenger, its `config/` defaults) plus the op-service flag
   families it appends (`op-service/rpc`, `op-service/log`,
   `op-service/metrics`, `op-service/oppprof`, `op-service/txmgr`, and
   `op-service/flags`, as applicable per component).
2. Update the component's `tag` in `manifest.json` to the new release tag.
3. Run the generator, restricted to that component:
   `go run ./docs/public-docs/scripts/gen-flags -only <component>`
   (from the tag checkout, with `-docs-dir` pointing at the docs root you are
   committing from, if they differ). The snippet, its provenance line, and
   the manifest `sha256` are rewritten together. Commit the manifest and
   snippet in the same change.

Never hand-edit the generated snippets (`-check` fails on any hand edit), and
never regenerate from unreleased code: if the flags changed since the manifest
tag, the regenerated table would document behavior no released binary has.

Enforcement runs as a review-gated Mintlify docs automation on a weekly
schedule: it compares the newest finalized tag of each covered component
(`op-batcher/v*`, `op-node/v*`, `op-proposer/v*`, `op-challenger/v*`,
`op-conductor/v*`) against the tag in `manifest.json` and, when a newer
release exists, regenerates per the steps above and proposes the change for
human review. Local runs of the generator (`-check` for verification) cover
the gap between scheduled runs.

## Adding a component

1. Add the component's flag slice to the `components` table in `main.go`,
   mirroring its required-flag names (the components keep their required
   slices unexported; a runtime sanity check catches renames).
2. Add its latest finalized release tag to `manifest.json`.
3. Run the generator (at that tag, or at a tag-parity-verified commit) and
   import the new snippet from the component's reference page.
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
| Flag facts (each component's `flags/` package, op-service flag families) | Component engineers | Component team | Component team; the docs table follows at the next finalized release |

Known residual gaps (accepted, by design):

- The tables document the manifest release tags, not `develop`. Flag changes
  merged after a release are intentionally not reflected until the next
  finalized tag is published and the automation (or a maintainer) regenerates.
- The required-flag lists are mirrored in `main.go` because the components do
  not export them. A rename fails the generator's sanity check; adding a brand
  new required flag without updating the mirror would list it as optional.
  (op-challenger's mirror also includes `l2-eth-rpc`, which its
  `CheckRequired` enforces unconditionally from outside its required slice.)
- `-check` verifies snippet integrity against the manifest hash, not
  regeneration parity with the tag — a regeneration mistakenly run from
  non-tag source would pass `-check`. The review gate on regeneration PRs and
  the tag-parity procedure above are the guards.
- op-batcher's `--compressor` option list (`compressor.KindKeys`) is built
  from map iteration, so its order changes on every process start — in
  `op-batcher --help` itself, not just here. The generator sorts that one
  list (`canonicalizeUsage` in `main.go`) so output is deterministic; the
  proper fix is sorting `KindKeys` at the source (one line in
  `op-batcher/compressor/compressors.go`), proposed as a follow-up.
- The `--network` usage string (op-node, op-challenger, op-conductor)
  enumerates the chains bundled from the superchain-registry, which changes
  with every registry submodule bump — independent of component releases.
  `canonicalizeUsage` replaces the enumerated list with a stable pointer at
  the registry so the snippets stay byte-stable; `<component> --help` always
  shows the exact bundled list.
