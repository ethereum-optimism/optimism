# Devdocs Decision Memo — DRAFT

> **Status: DRAFT — no decision has been made.** This memo records the
> working default and the cross-team ask, per Slice 3 of the Option A
> canonical-homes plan
> (`ethereum-optimism/solutions` → `projects/docs-improver/plans/option-a-canonical-homes.md`).
> The decision itself requires the owning teams (see "Asks") and must not be
> executed from this draft. Tracking: ethereum-optimism/solutions#956.

## Problem

devdocs.optimism.io hosts developer books that overlap the canonical docs
site (docs.optimism.io, source in this repo under `docs/public-docs/`):

1. **The contracts-bedrock book** (`devdocs.optimism.io/contracts-bedrock`).
   Genuinely deep, maintained-with-source material — but linked from exactly
   one page on the docs site (`releases/op-contracts.mdx`), so readers of
   `smart-contracts.mdx` and the bridging explainers never find it.
2. **A legacy OP Deployer book** (`devdocs.optimism.io/op-deployer/`). The
   canonical op-deployer docs now live on the docs site at
   `/chain-operators/tools/op-deployer/overview`; the devdocs book duplicates
   now-canonical content. External references still point at it — e.g. the
   superchain-registry's `docs/ops.md` ("Adding a custom chain" section) links
   `https://devdocs.optimism.io/op-deployer/` while the same document's
   standard-chain section links `docs.optimism.io/chain-operators/tools/op-deployer`
   (checked at superchain-registry commit `738212e`).

Under the content guide's dual-sourcing ban
(`/op-stack/contribute/content-guide`), two live homes for the same content is
the failure mode, whichever of them is better.

## Options

- **Fold**: import the contracts-bedrock book's content into docs.optimism.io
  and retire the devdocs book. Maximal one-home compliance; large migration
  and an ongoing sync obligation against contract source — needs the
  contracts team to own it.
- **Link (default)**: keep the book where it is maintained, beside its
  source, and add prominent links from the docs site's contract-facing pages
  (`op-stack/protocol/smart-contracts.mdx`, the bridging explainers). Zero
  migration; the book remains the canonical home for contract internals per
  the content guide's component-internals row.

## Recorded Default

Per the Option A plan's risk table ("nothing blocks on cross-team items"),
**the default is Link**: prominent links from `smart-contracts.mdx` and the
bridging explainers to the contracts-bedrock book. If the fold is never
agreed, the default stands and satisfies the plan's goal G4/G5 intent
(joined, not mirrored).

Independently of fold-vs-link, **the legacy OP Deployer book should be
retired** (deleted or redirected to
`docs.optimism.io/chain-operators/tools/op-deployer/overview`), since its
canonical successor already exists. This needs filing with the book's owning
team, including a heads-up to superchain-registry maintainers to re-point the
`docs/ops.md` link.

## Asks (cross-team — owners to be named)

1. **Contracts/platform team**: decide fold vs link for the contracts-bedrock
   book; name an owner for the decision. Absent a decision, the Link default
   above is applied.
2. **Owning team of devdocs.optimism.io**: file and execute the retirement of
   the legacy OP Deployer book (delete or redirect).
3. **Superchain-registry maintainers**: after (2), re-point the
   `docs/ops.md` custom-chain link to the canonical op-deployer docs.

## Verification Notes (for reviewers)

- `releases/op-contracts.mdx` → `devdocs.optimism.io/contracts-bedrock` link:
  verified in this repo (the only `devdocs.optimism.io` reference in
  `docs/public-docs/`).
- superchain-registry `docs/ops.md` links: verified at registry commit
  `738212ef2932d26f9274e3317699471534d2b3ac`.
- devdocs.optimism.io itself was **not** reachable from the drafting
  environment (egress-restricted); live-site contents should be spot-checked
  by a human before the asks are filed.
