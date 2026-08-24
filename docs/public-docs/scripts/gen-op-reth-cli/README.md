# gen-op-reth-cli — generated op-reth CLI reference for docs.optimism.io

Regenerates the op-reth CLI reference tree
(`docs/public-docs/node-operators/op-reth/cli/op-reth.mdx` and everything under
`.../cli/op-reth/`) plus its docs.json nav fragment from the `--help` output of
a pinned op-reth release binary, so the published command catalog cannot
silently fall behind a release.

The pages entered the monorepo as a one-time snapshot of the retired Vocs site
(optimism#19778) with no regeneration pipeline, and drifted (optimism#21845
deleted two stale `db settings set` pages by hand). This generator is the
pipeline: it is a port of upstream reth's `docs/cli/update.sh` +
`docs/cli/help.rs` — the exact tooling whose output format the snapshot
matches — emitting Mintlify MDX (frontmatter with `diataxis: reference`
instead of a Markdown heading) and the docs.json nav fragment instead of a
Vocs sidebar.

## Usage

Build the binary at a finalized op-reth release tag, then run from anywhere
(paths resolve relative to this directory):

```bash
# in a checkout of the release tag
cargo build --bin op-reth --manifest-path rust/op-reth/bin/Cargo.toml

node docs/public-docs/scripts/gen-op-reth-cli/main.mjs \
  --bin rust/target/debug/op-reth \
  --tag op-reth/vX.Y.Z
```

To verify the committed tree against a binary without rewriting anything
(exits nonzero on any mismatch):

```bash
node docs/public-docs/scripts/gen-op-reth-cli/main.mjs --bin <op-reth> --check
```

## How it works

- Walks the binary's `--help` tree recursively (subcommands parsed from each
  help output's `Commands:` section, `help` itself skipped), with
  `NO_COLOR=1 COLUMNS=100 LINES=10000` for stable formatting.
- Writes one MDX page per command: quoted `title`, `diataxis: reference`
  frontmatter, the command's one-line description, and the verbatim help text
  in a `txt` code block. Environment-dependent output (home paths, version
  and target triple, CPU-count defaults) is scrubbed with the same
  replacements as upstream `help.rs`, producing the `<CACHE_DIR>` /
  `<VERSION>` / `<OS>` placeholders.
- Splices the regenerated command tree into the `op-reth CLI reference` group
  of `docs.json` (the collapsed group under Node Operators > Reference),
  keeping the hand-written `cli/overview` page as the group's first entry.
  Nested subcommands become nested groups named by the full command, in
  `--help` order. The hand-written overview page is never touched.
- Deletes pages whose command disappeared from `--help` and prints their
  URLs: **every deleted page must gain a redirect in `docs.json` in the same
  PR** (see `REDIRECTS_GUIDE.md`); the redirect lint enforces this.
- `manifest.json` records the finalized release tag the tree was generated
  at, the binary's reported version line, and a SHA-256 over the generated
  pages. All are rewritten together, only on regeneration.

### What `--check` verifies

The tree documents a *released* binary: it is generated at the manifest tag,
and `develop` routinely moves past that tag, so byte-comparing against a
binary built from an arbitrary commit is only meaningful at (or at parity
with) the tag. Given a binary, `--check` recomputes the whole tree in memory
and fails on any difference against the committed pages, the docs.json nav
fragment, or the manifest hash — which also makes any hand edit to a
generated page fail the next check.

## Regenerating for a new release

When a new finalized (non-rc) `op-reth/vX.Y.Z` tag is published:

1. Check out the tag and build the binary with the default feature set
   (`cargo build --bin op-reth --manifest-path rust/op-reth/bin/Cargo.toml`).
   Never regenerate from unreleased code: if the CLI changed since the tag,
   the pages would document behavior no released binary has.
2. Run the generator with `--tag` set to the new tag (from the tag checkout,
   against the docs tree you are committing from).
3. If the run deleted pages, add one redirect per deleted URL to `docs.json`
   in the same change, pointing at the nearest surviving command page.
4. Run the docs lints (`pnpm lint:nav`, `pnpm lint:redirects`,
   `node scripts/lint-link-policy.mjs --baseline scripts/lint-link-policy.baseline.json`)
   and commit pages, nav fragment, redirects, and `manifest.json` together.

Running the generator twice in a row against the same binary is a no-op.

## Automation registration

Regeneration runs as a review-gated Mintlify docs automation (the same
mechanism as the nav and redirect lints in `DOCS_CONTRIBUTING.md` — content
updates proposed for human review, never CI): on its weekly schedule it
compares the newest finalized `op-reth/v*` tag against the tag in
`manifest.json` and, when a newer release exists, regenerates per the steps
above and proposes the change for human review, including the redirect
entries for any deleted pages. Local `--check` runs cover the gap between
scheduled runs. The automation prompt is recorded in the docs automation
registry alongside the gen-flags and gen-deploy-config entries, not held as
tribal knowledge.

## Ownership

Everything in this pipeline lives under `docs/public-docs/`, so it is covered
by the existing CODEOWNERS rule
(`/docs/public-docs/  @ethereum-optimism/solutions @ethereum-optimism/monorepo-reviewers`)
— no dedicated entries are needed. Per the ownership model in the Solutions
repo's `projects/docs-improver/plans/option-b-truth-from-source.md`:

| Artifact | Author of record | Reviewer | Stale-reference triage |
| --- | --- | --- | --- |
| Generator code + docs automation (this directory, the Mintlify automation config) | Matthew Cruz (@sbvegan), docs owner — proposed, pending confirmation | @ethereum-optimism/solutions (via the `/docs/public-docs/` CODEOWNERS rule) | Author of record |
| Generated pages (`node-operators/op-reth/cli/op-reth*`) and the nav fragment | The pipeline — nobody hand-edits; `--check` fails on hand edits by construction | @ethereum-optimism/solutions review the automation's regeneration PRs | A tree that cannot be regenerated cleanly is filed as an `accuracy`-labelled issue on the Solutions board |
| CLI facts (`rust/op-reth` and the pinned upstream reth crates) | Component engineers | Component team | Component team; the docs tree follows at the next finalized release |

Known residual gaps (accepted, by design):

- The tree documents the manifest release tag, not `develop`. CLI changes
  merged after a release are intentionally not reflected until the next
  finalized tag is published and the automation (or a maintainer)
  regenerates.
- `--check` needs a binary, so it cannot run as a pure content lint; the
  redirect and nav lints still guard the generated pages' URLs and nav
  entries on every PR.
- The hand-written `cli/overview.mdx` explainer references commands by name;
  if a top-level command is ever renamed, the regeneration PR must update the
  overview by hand (the nav and link lints catch dangling links).
