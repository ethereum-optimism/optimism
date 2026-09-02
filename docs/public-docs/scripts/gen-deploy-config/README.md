# gen-deploy-config — generated deploy-config schema for docs.optimism.io

Generates the MDX schema reference for the OP Stack `DeployConfig` — the flat
JSON deployment configuration — under `docs/public-docs/snippets/generated/`,
from the `DeployConfig` struct tree in `op-chain-ops/genesis`. The docs page
(`chain-operators/reference/rollup-deployment-configuration.mdx`) imports the
snippet instead of hand-transcribing the struct, so the published schema
cannot silently fall behind the source.

This is the config-schema sibling of [`../gen-flags`](../gen-flags/README.md)
and reuses its conventions: a do-not-edit header, a provenance line naming a
finalized release tag, and a `manifest.json` recording `{tag, sha256}` per
generated snippet. Consolidating the two tools behind one shared core is a
noted follow-up; they are kept separate so each can land as one reviewable
change.

## Usage

Run from the monorepo root:

```bash
go run ./docs/public-docs/scripts/gen-deploy-config
```

To verify the committed snippet matches the SHA-256 recorded in
`manifest.json` without rewriting it (hand-edit detection; exits nonzero on
any mismatch):

```bash
go run ./docs/public-docs/scripts/gen-deploy-config -check
```

The tool parses Go source (`go/ast`) — it does not build the package, so it
needs no `just build-superchain-go` step and can run against a bare source
checkout of any tag.

## How it works

- `main.go` parses the genesis package source (default `op-chain-ops/genesis`,
  override with `-source`), finds the `DeployConfig` struct, and flattens its
  embedded config structs (`L2CoreDeployConfig`, `FaultProofDeployConfig`,
  `UpgradeScheduleDeployConfig`, …) in declaration order — mirroring how
  `encoding/json` flattens the embedded fields into one flat JSON object.
- Each embedded struct becomes a section titled from the `groupTitles` table,
  introduced by the struct's Go doc comment; each field becomes a table row
  with its JSON key (from the `json:"..."` struct tag), a friendly type from
  the `typeLabels` table, and its Go doc comment. Fields tagged `json:"-"`
  are skipped.
- The generator fails loudly on anything it cannot map — an embedded struct
  missing from `groupTitles`, a field type missing from `typeLabels`, a field
  without a `json` tag, a cross-package embed, or a duplicate JSON key — so
  upstream schema changes force a conscious docs decision instead of silent
  drift or leaked Go type names.
- `manifest.json` records the finalized release tag the snippet was generated
  at and the snippet's SHA-256; the provenance line names the same tag. All
  three are updated together, only on regeneration.

## Why the tag is an op-deployer tag

`op-chain-ops` is a library, not a released binary — it has no release tags of
its own. The released tool whose pipeline derives and emits the `DeployConfig`
(and the deploy path the docs frame as the default) is **op-deployer**, so the
snippet documents the schema as of the newest finalized (non-rc, non-alpha)
`op-deployer/v*` tag, and regeneration is keyed to those releases.

## Regenerating for a new release

The snippet documents *released* software, so never regenerate from unreleased
code. When a new finalized `op-deployer/v*` tag is published:

1. Update the `tag` in `manifest.json` to the new release tag.
2. Generate from the schema source at that tag. Either check out the tag, or
   extract just the package the tool parses and point `-source` at it:

   ```bash
   git archive op-deployer/vX.Y.Z op-chain-ops/genesis | tar -x -C /tmp/opd-tag
   go run ./docs/public-docs/scripts/gen-deploy-config -source /tmp/opd-tag/op-chain-ops/genesis
   ```

   (If `op-chain-ops/genesis` is identical between the tag and your current
   checkout — `git diff op-deployer/vX.Y.Z HEAD -- op-chain-ops/genesis` —
   running without `-source` is equivalent.)
3. Commit the regenerated snippet and `manifest.json` in the same change.

Never hand-edit the generated snippet (`-check` fails on any hand edit). Note
that `-check` verifies the committed snippet against the manifest hash rather
than against a regeneration from the current tree: the schema source on
`develop` may legitimately move past the tag between releases, so tree parity
is reported informationally only.

Enforcement runs as a review-gated Mintlify docs automation on a weekly
schedule: it compares the newest finalized `op-deployer/v*` tag against the
tag in `manifest.json` and, when a newer release exists, regenerates per the
steps above and proposes the change for human review. Local `-check` runs
cover the gap between scheduled runs.

## Ownership

Everything in this pipeline lives under `docs/public-docs/`, and the generator
code in this directory is owned by solutions via the CODEOWNERS rule
(`/docs/public-docs/scripts/  @ethereum-optimism/solutions`). Per the ownership
model in the Solutions repo's
`projects/docs-improver/plans/option-b-truth-from-source.md`:

| Artifact | Author of record | Reviewer | Stale-reference triage |
| --- | --- | --- | --- |
| Generator code + docs automation (this directory, the Mintlify automation config) | Matthew Cruz (@sbvegan), docs owner — proposed, pending confirmation | @ethereum-optimism/solutions (via the `/docs/public-docs/scripts/` CODEOWNERS rule) | Author of record |
| Generated snippet (`snippets/generated/deploy-config-schema.mdx`) | The pipeline — nobody hand-edits; `-check` fails on hand edits by construction | @ethereum-optimism/solutions review the automation's regeneration PRs | A snippet that can't be regenerated cleanly is filed as an `accuracy`-labelled issue on the Solutions board |
| Schema facts (`op-chain-ops/genesis` struct tags + doc comments) | Component engineers | Component team | Component team; the docs schema follows at the next finalized op-deployer release |

Known residual gaps (accepted, by design):

- The schema documents the manifest release tag, not `develop`. Struct changes
  merged after a release are intentionally not reflected until the next
  finalized tag is published and the automation (or a maintainer) regenerates.
- Descriptions are the source's Go doc comments; fields without a doc comment
  (the dev-only L1/L2 genesis block fields, `channelTimeoutGranite`) render an
  em dash. Improving them means improving the doc comments upstream — by
  design, not by hand-editing the snippet.
- Validation rules (which values must be nonzero, conditional requirements)
  live in the structs' `Check` methods and are not extracted; the reference
  page's hand-written guidance covers the operationally important ones.
- Requirement facts that are policy rather than code (standard-configuration
  requirements, recommended values) stay hand-written on the reference page.
