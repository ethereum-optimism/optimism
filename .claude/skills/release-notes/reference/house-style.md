# House style for OP Stack release notes

The target style is a **curated change list**, as in
[`op-challenger/v1.9.4`](https://github.com/ethereum-optimism/optimism/releases/tag/op-challenger%2Fv1.9.4)
and
[`op-contracts/v8.0.0-rc.2`](https://github.com/ethereum-optimism/optimism/releases/tag/op-contracts%2Fv8.0.0-rc.2).
Those two are the reference; read them before drafting.

A curated note is not the git-cliff list with prose bolted on top. The PR list is
*replaced* by grouped, self-contained entries. Because each entry explains itself, the
stack of callouts that older notes used to supply context is unnecessary.

This file goes stale. Run `scripts/recent-style.sh` first, and when recent releases
disagree with anything here, they win — then update this file.

## Shape

```markdown
## Overview

> [!NOTE]
> This is a <patch|minor|major> release of <component> containing <what kind of changes>. It is an <recommendation> upgrade for all users.

Note: <feature> is not yet live in production. Changes to those code paths in this release are preparation for future functionality and do not affect operators today.

## Breaking changes            <!-- only when there are any -->

- **<Short name>** (#NNNNN). What changed, and what the operator must do before upgrading.

## What's Changed

### Features
- <Self-contained description> (#NNNNN)

### Bug fixes
- <Self-contained description> (#NNNNN, #NNNNN)

### <Domain grouping, e.g. "Super dispute games (not yet in production)">
- ...

### Other
- ...

**Full Changelog**: https://github.com/ethereum-optimism/optimism/compare/<prev-finalized>...<this-finalized>

**🚢 Docker Image:**

- https://us-docker.pkg.dev/oplabs-tools-artifacts/images/<component>:<version>
```

## The Overview sentence

One standard sentence carries the upgrade recommendation, inside a callout at the top of
`## Overview`. Sentence one names the release type and what it contains; sentence two gives
the recommendation, using exactly one of these:

| Recommendation | Use when |
| --- | --- |
| `It is an optional upgrade for all users.` | Nothing operator-visible; refactors, internals, tooling |
| `It is an optional but recommended upgrade for all users.` | Contains fixes or features worth having, none urgent |
| `It is a recommended upgrade for all users.` | Contains a fix operators are likely to want — a bug they could hit, a meaningful performance improvement |
| `It is a required upgrade for <scope>.` | Not upgrading risks a halt, consensus divergence, or stalled safe-head progression. Name the scope, and say who is unaffected |

Derive patch/minor/major from the semver difference against the previous finalized tag.

**The callout is always `> [!NOTE]`, whatever the recommendation** — the one exception being
`required`, which uses `> [!CAUTION]`. Holding the block type constant is the point: how
urgent the release is comes from the fixed sentence vocabulary, while the block only makes
that sentence findable. Varying the block with the severity is how notes ended up flagging
a routine recommended upgrade as a `[!WARNING]`, which overstates it and makes the
recommendation harder to read, not easier.

```markdown
## Overview

> [!NOTE]
> This is a patch release of op-supernode containing bug fixes and improved observability. It is an optional but recommended upgrade for all users.
```

For a required upgrade, scope it — name the affected chains or configurations and say who
is unaffected, as `op-node/v1.19.3` does with "Mode, Metal, and Zora ... OP Mainnet is
unaffected":

```markdown
## Overview

> [!CAUTION]
> This is a patch release of op-node containing a registry config fix. It is a required upgrade for operators of Mode, Metal, and Zora using the built-in `--network` configs — an op-node on v1.19.2 or earlier can derive a different post-Karst gas limit and halt safe-head progression. OP Mainnet is unaffected.
```

## Proportionality

The single most common correction is calling undue attention to small things. A new metric,
a version string reported wrongly, a rare corner-case fix — these are **bullets in the
list**, not callouts, and not Overview material. If a reader would shrug, it does not get
a heading of its own.

Two questions before you highlight anything:

1. **Is the affected feature live in production?** Interop/Lagoon is not enabled anywhere.
   Neither are ZK dispute games or super dispute games. A change to a dormant code path
   cannot affect operators today, however alarming its description sounds. Put such changes
   under a `### <Feature> (not yet in production)` heading and add the standard Note
   paragraph to the Overview:

   > Note: ZK dispute games and super dispute games (interop) are not yet live in production. Changes to those code paths in this release are preparation for future functionality and do not affect operators today.

2. **How likely is an operator to hit this?** "Closes a very rare corner case" and "fixes a
   bug that freezes block production" both describe bug fixes, and they do not deserve the
   same prominence.

## Curating the change list

**Group by domain or change type.** `### Features`, `### Bug fixes`, `### Other` is the
default spine; add domain headings (`### Dispute games`, `### L1 contracts`,
`### Super dispute games (not yet in production)`) when they carry more meaning. Drop any
heading that would hold nothing. A short release needs no headings at all — see
`op-dispute-mon/v1.5.2`, which is one flat list.

**Group PRs that are one logical change.** Several PRs that together deliver one thing get
one entry with all their numbers: `... are no longer required when only permissioned game
types are configured (#21270, #21681)`.

**Write self-contained entries.** The PR title is usually not enough on its own — that is
the reason the raw list is being replaced. Say what changed and why an operator cares:

> - Tear down the whole VM process group when `--vm-timeout` is hit, preventing orphaned VM processes from lingering after a timeout (#21268)

not

> - op-challenger: tear down VM process group on timeout by @author in [#21268](...)

**Mention each PR exactly once.** No entry should be explained in a callout and then listed
again below. There is no separate raw PR list to fall back on. The one exception is a
`## New Contributors` row, which credits a person rather than describing the change.

**Reference PRs as bare `(#NNNNN)` at the end of the entry.** GitHub links them
automatically in release bodies. Drop `by @author` — contributor credit lives in
`## New Contributors`.

## Breaking changes

When a change requires operator action before upgrading, it gets its own `## Breaking
changes` section above `## What's Changed`, with a bold short name and the required action
stated plainly (from `op-challenger/v1.9.3`):

```markdown
## Breaking changes

- **`--cannon-kona-experimental-witness-endpoint` flag removed** (#20498). The `debug_executePayload`-based witness path is now the default for kona-cannon games. Operators passing this flag (or setting `CANNON_KONA_EXPERIMENTAL_WITNESS_ENDPOINT` in their config) must remove it before upgrading — op-challenger will reject it as unknown.
```

Go-API-only changes are **not** breaking changes for this purpose. They affect downstream
importers, not operators; put them under `### Other` with a note that no operator action is
needed, or leave them out when they are pure internal churn.

## Callouts

**Exactly one callout per note** — the Overview block described above. Nothing else.

The curated list carries everything else: each entry explains itself, so there is nothing
left for a second callout to add. If you find yourself reaching for one, the content is an
entry in the list, or it belongs in `## Breaking changes`.

Callout stacks are what this style replaces. Notes that opened with three or four
`[!IMPORTANT]` blocks were compensating for a PR list that could not speak for itself.

## Standing notices

Recurring boilerplate carried between releases — the APKO migration block was one — is not
safe to copy forward blindly; its text said "in this release (and only in this release)".
`scripts/recent-style.sh` shows whether the previous release carried one. Ask before
including it, and copy it verbatim along with whatever image lines it implies.

## Tags, links and images

For a finalized release the heading, the compare link's right side and the image tag all
carry the plain version — never `-rc.N`. `scripts/retarget-tag.sh` does this and refuses
when it cannot verify that the finalized tag matches the RC commit.

The compare link's **base** is the previous *finalized* tag, with three dots:

```markdown
**Full Changelog**: https://github.com/ethereum-optimism/optimism/compare/op-node/v1.19.4...op-node/v1.19.5
```

git-cliff generates an RC base and older notes still carry one; the current convention is
finalized-to-finalized.

## New Contributors

Keep git-cliff's section when the contributor's PR survived curation — with authorship gone
from the entries, it is the only credit left. Drop bot rows (`@dependabot[bot]`,
`@oplabs-renovate[bot]`, `@claude[bot]`) and rows whose PR was cut, since the contributor is
credited in whichever component's release actually ships their work.

## Working notes

Keep the raw git-cliff bullets for cut PRs as HTML comments at the bottom of the draft
while iterating, each with a short reason. They make review easy — a reviewer can see what
was considered and reinstate an entry in one edit — and are deleted before publishing, or
kept if the release manager prefers.

```markdown
<!--* op-core/fees: add Jovian DA-footprint calculation by @claude[bot] in [#22163](...) — doesn't affect the batcher-->
```
