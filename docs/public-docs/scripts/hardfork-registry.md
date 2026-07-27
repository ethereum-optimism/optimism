# Hardfork registry frontmatter schema

Every OP Stack hardfork has exactly one permanent registry page at
`op-stack/protocol/hardforks/<fork>.mdx`. The machine-readable facts live in
that page's frontmatter; `scripts/generate-hardforks.ts` validates them against
the [superchain-registry](https://github.com/ethereum-optimism/superchain-registry)
(the monorepo submodule at `superchain-registry/`) and renders them into the
generated snippets under `snippets/generated/hardforks/` (a summary table plus
one per-fork facts table). Nobody hand-edits an activation table: **a
frontmatter edit followed by regeneration is the only way to change an
activation row.**

This registry is shared infrastructure: it was built by the Option B program
(slice B4, "Truth from Source") and is consumed by the Option A program
(slice 6, protocol anchor). There is exactly one implementation.

## Commands

From `docs/public-docs/`:

```bash
pnpm gen:hardforks         # validate + regenerate snippets
pnpm gen:hardforks:check   # drift check (fails when snippets are stale)
```

The superchain-registry submodule must be initialized
(`git submodule update --init superchain-registry` from the monorepo root), or
pass `--registry <path>` pointing at a checkout.

## Schema

Keys are deliberately flat (scalar `key: value` plus block string lists) so the
generator needs no YAML dependency. Timestamps are unix seconds, UTC.

| Key | Required | Meaning |
| --- | --- | --- |
| `hardfork_name` | yes | Lowercase fork name; must match the filename (`karst` for `karst.mdx`). |
| `hardfork_lifecycle` | yes | `active` (activated on the mainnet superchain default), `scheduled` (timestamps set, in the future), or `development` (spec in progress, no activations). |
| `hardfork_spec` | yes | Governing spec, as a rendered `https://specs.optimism.io/...` URL on its current path (see the [cross-repo link policy](../op-stack/contribute/link-policy.mdx)). |
| `hardfork_activation_mainnet` | unless `development` | Superchain-wide mainnet default from `superchain/configs/mainnet/superchain.toml`. `0` means genesis-active / at Bedrock (predates the superchain defaults; e.g. Regolith). |
| `hardfork_activation_sepolia` | unless `development` | Superchain-wide sepolia default from `superchain/configs/sepolia/superchain.toml`. Same `0` convention. |
| `hardfork_governance` | no | Governance approval URL (`gov.optimism.io` thread or `vote.optimism.io` proposal). |
| `hardfork_notice` | no | Root-relative link to the operator notice for the upgrade (e.g. `/notices/archive/upgrade-19`). |
| `hardfork_upgrade_number` | no | The governance "Upgrade N" number, only where a notice in this repo confirms it. |
| `hardfork_min_versions` | no | Block list of `"<component> <version>"` strings — minimum component versions to follow the fork. |
| `hardfork_min_versions_source` | with `hardfork_min_versions` | Markdown link citing where the versions come from (the upgrade notice or release notes). |

Example:

```yaml
---
title: Karst
description: Hardfork registry entry for the Karst network upgrade.
diataxis: reference
hardfork_name: karst
hardfork_lifecycle: active
hardfork_spec: https://specs.optimism.io/protocol/karst/overview.html
hardfork_activation_mainnet: 1783526401
hardfork_activation_sepolia: 1781712001
hardfork_notice: /notices/archive/upgrade-19
hardfork_upgrade_number: 19
hardfork_min_versions:
  - op-node v1.19.1
  - op-reth v2.3.3
hardfork_min_versions_source: "[Upgrade 19 notice](/notices/archive/upgrade-19)"
---
```

## Validation rules

*   Every fork scheduled in the superchain-registry's `[hardforks]` sections
    must have a registry page, and the page's activation values must equal the
    registry's — the superchain-registry is the source of truth.
*   A page may not state a nonzero activation that the superchain-registry does
    not contain.
*   Keys in `[hardforks]` that are not L2 hardforks in the fork series
    (`pectra_blob_schedule_time`, `keep_karst_upgrade_gas`) are ignored.
*   Rendered tables show timestamps (and their UTC dates) only. Hardforks
    activate by L2 block timestamp and block heights differ per chain, so
    block-number estimates are deliberately not rendered.

## Adding a new hardfork

1.  Create `op-stack/protocol/hardforks/<fork>.mdx` with `hardfork_lifecycle: development`
    and the spec link; import its generated snippet
    (`/snippets/generated/hardforks/<fork>.mdx`).
2.  Add the page to the `docs.json` nav (OP Stack → Protocol Information →
    Network Upgrades group).
3.  When the superchain-registry schedules activation times, update the
    frontmatter to `scheduled` (later `active`) with those timestamps.
4.  Run `pnpm gen:hardforks` and commit the page and regenerated snippets
    together; the drift check fails otherwise.
