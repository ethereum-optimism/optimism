# House style for OP Stack release notes

The target is a **curated change list**, as in
[`op-challenger/v1.9.4`](https://github.com/ethereum-optimism/optimism/releases/tag/op-challenger%2Fv1.9.4)
and
[`op-contracts/v8.0.0-rc.2`](https://github.com/ethereum-optimism/optimism/releases/tag/op-contracts%2Fv8.0.0-rc.2).
Read those two before drafting for how entries are written — but take the section layout
from the Shape below, not from them. Every published release predates it.

A curated note is not the git-cliff list with prose bolted on top: the PR list is *replaced*
by grouped, self-contained entries. Because each entry explains itself, the stack of
callouts older notes used to supply context is unnecessary.

This file is the source of truth. Changing the style means editing it, not inferring a new
convention from one release.

## Shape

```markdown
## Overview

<!-- block type follows the recommendation: NOTE / IMPORTANT / CAUTION -->
> [!IMPORTANT]
> This is a <recommendation> release <for whom>. It contains <what kind of changes>.

## Breaking changes            <!-- only when there are any -->

- **<Short name>** (#NNNNN). What changed, and what the operator must do before upgrading.

## Chain Configuration         <!-- only when a chain's embedded config moved -->

- **<Short name>** (#NNNNN). Which chains, which values, and what happens to a node that upgrades late.

## Other changes

### Derivation                 <!-- op-node or kona derivation changes, first -->
- <Self-contained description> (#NNNNN)

### Features
- <Self-contained description> (#NNNNN)

### Bug fixes
- <Self-contained description> (#NNNNN, #NNNNN)

### <Domain grouping, e.g. "Sequencer">
- ...

### Miscellaneous
- ...

**Full Changelog**: https://github.com/ethereum-optimism/optimism/compare/<prev-finalized>...<this-finalized>

**🚢 Docker Image:**

- https://us-docker.pkg.dev/oplabs-tools-artifacts/images/<component>:<version>
```

## The Overview sentence

One standard sentence carries the upgrade recommendation, in a callout at the top of
`## Overview`. Sentence one says what kind of release this is and for whom; sentence two
says what it contains.

**Never name a semver release type.** "This is a minor release" tells the reader nothing —
our tags are not strict semver, so the version numbers do not carry that meaning. The
recommendation is the classification.

Exactly one of:

| Recommendation | Use when |
| --- | --- |
| `This is an optional release ...` | Nothing operator-visible |
| `This is a recommended release ...` | Fixes or features worth having, none urgent |
| `This is a strongly recommended release ...` | A fix operators are likely to want |
| `This is a required release for <scope> ...` | Not upgrading risks a halt, consensus divergence, or stalled safe-head progression. Name the scope, and say who is unaffected |

Scope it to a role when only that role benefits — "a required release for World Chain
operators running the built-in `--network` configs, and a strongly recommended one for
everyone else" — rather than pitching it at everyone on the strength of something most
operators never see.

**The callout type follows the recommendation**: `> [!NOTE]` for `optional`,
`> [!IMPORTANT]` for either recommended level, `> [!CAUTION]` for `required`. Nothing else —
`[!WARNING]` is not in the vocabulary, and reaching for it is how a routine recommended
upgrade once ended up overstated.

## Impact, not implementation

**Release notes say what changes for the reader. They do not explain how the code works.**
This is the correction made most often, and the easiest to slide back on, because the PR
descriptions read during triage are written the other way round.

Out: threads and goroutines, event loops, call paths, locking; internal type, function and
package names; "moved X into Y"; latency mechanics; release-internal history such as which
PR introduced the bug another PR fixes, or which PR is stacked on which.

In: the symptom that appears or disappears, and who sees it; flag, env var, metric, RPC and
config names; values that shift, and dashboards that need checking; anything the reader must
do.

```markdown
<!-- too much: architecture the reader cannot see -->
- Block production now runs on its own goroutine and calls the engine controller directly,
  instead of competing with derivation for the shared driver loop. Sequencer latency under
  derivation load drops from a full event-queue drain to at most one engine critical section.

<!-- right: the symptom that goes away, and where it was felt -->
- Block production now runs concurrently with the derivation pipeline, alleviating the
  long-standing issue where block production can slow immediately following inclusion of a
  batcher transaction on L1 — most noticeable on chains with relatively low throughput
  (#22241, #22238, #22360)
```

The line is not "no technical detail". An entry may keep that `SuggestBlobTipCap` sampled one
block more than `maxBlocks`, because that *is* the observable change — it says why suggested
tips move. Detail describing a computation whose result the operator sees stays; detail
describing how the program is put together goes.

A useful test: if a sentence would sit just as comfortably in the PR description, it is
probably implementation.

## Proportionality

Calling undue attention to small things is the second most common correction. A new metric, a
version reported wrongly, a rare corner-case fix — these are **bullets in the list**, not
callouts, and not Overview material.

**Is the affected feature live in production?** A change to a dormant code path cannot affect
operators today, however alarming its description sounds. For a hardfork, check the registry
rather than trusting any list written here — a chain with an activation time is a chain
running the fork:

```bash
# zero hits across both networks => not live anywhere
grep -rl '<fork>_time' superchain-registry/superchain/configs/mainnet/
grep -rl '<fork>_time' superchain-registry/superchain/configs/sepolia/
```

For a `DevFeatures` bit, the default is the answer: anything behind a DevFeature that isn't
forced to `true` should be considered disabled and not yet in production — we ship the
defaults for feature toggles. `docs/ai/devfeatures.md` lists them; today only
`SuperRootGamesMigration` is default-on, so everything else behind a bit is dormant unless a
chain has explicitly set it.

For anything expressed neither as a hardfork nor a DevFeature — dispute game types, say —
there is no equivalent lookup, so ask the release manager rather than guessing.

**A change confined to an unreleased feature is cut entirely.** No `(not yet in production)`
heading, no explanatory Note in the Overview — a reader upgrading today cannot act on it, and
it competes for attention with the changes they can.

The test is whether the change reaches a live path, not what motivated it. A fix written for
an interop scenario that also alters pre-interop derivation stays in, described by its live
effect; the same fix, if it only fires once the fork activates, does not. So read what the
change does, not the feature name in its PR title.

**Do not narrate an attack the code path cannot currently suffer.** State what the fix aligns
or corrects; leave the exploit narrative out until the path is live.

## Curating the change list

**Group by domain or change type.** `### Features` / `### Bug fixes` / `### Miscellaneous`
is the default spine under `## Other changes`; add domain headings where they carry more
meaning. Drop any heading that would be empty, and use a flat list for a short release.

**Derivation changes get `### Derivation`, listed first.** This holds for op-node and kona
alike: derivation is the part of a release most likely to change what a node computes, so it
is not left to fall into `### Bug fixes` among unrelated entries.

**Group PRs that are one logical change** into one entry with all their numbers:
`... are no longer required when only permissioned game types are configured (#21270, #21681)`.

**Write self-contained entries.** The PR title is usually not enough on its own — that is why
the raw list is being replaced. Say what changed and why an operator cares:

> - Tear down the whole VM process group when `--vm-timeout` is hit, preventing orphaned VM processes from lingering after a timeout (#21268)

**A change with no user-visible impact does not appear at all.** Not under
`### Miscellaneous`, not as a summarising line — a reader gains nothing from being told that
something they cannot observe was rearranged. The one exception is a release that would
otherwise have an empty change list, where one summarising line is more honest than
publishing nothing.

Importability of the monorepo **as a Go module is not user impact**. We do not maintain
releases of it as a Go module, so a change that only unblocks downstream importers — moving
a symbol to a leaf package, shrinking a build closure — is cut like any other internal
churn, however much work it was.

**Only mention a PR more than once** if it included multiple logical changes worth
describing separately.

**Reference PRs as bare `(#NNNNN)`** at the end of the entry — GitHub links them
automatically in release bodies. Drop `by @author`.

**Drop git-cliff's `## New Contributors` section.** It tells a reader nothing about what is
in the release.

## Breaking changes

When a change requires operator action before upgrading, it gets its own section directly
below `## Overview`, with a bold short name and the required action stated plainly:

```markdown
## Breaking changes

- **`--cannon-kona-experimental-witness-endpoint` flag removed** (#20498). The `debug_executePayload`-based witness path is now the default for kona-cannon games. Operators passing this flag must remove it before upgrading — op-challenger will reject it as unknown.
```

**State the action, not why the old behaviour was wrong.** The entry exists so an operator
knows what to do before upgrading; the history of the flag or field belongs in the PR.

Go-API-only changes are **not** breaking changes for this purpose, and are not entries
anywhere else either — see "A change with no user-visible impact" above.

## Chain Configuration

A release that moves a chain's embedded config gets its own `## Chain Configuration` section,
after `## Breaking changes` when there is one and directly below `## Overview` when there is
not — most often a new hardfork activation time arriving with a superchain-registry pin bump.
It stands alone rather than nesting under breaking changes, because a registry bump
frequently ships without one. Name the chains and the exact values, and say what happens to a
node that upgrades late, since that is the whole reason the section exists:

```markdown
## Chain Configuration

- **New World Chain Karst activation times** (#22624). The embedded superchain registry gains:

  - `sepolia/worldchain` — `karst_time = 1788868800` (Tue 8 Sep 2026 12:00:00 UTC)
  - `mainnet/worldchain` — `karst_time = 1789992000` (Mon 21 Sep 2026 12:00:00 UTC)

  A World Chain node on an earlier release, running the built-in `--network` config rather than an explicit rollup config, will not activate Karst and will diverge from the chain. No other chain's activation times change in this release.
```

Saying which chains are *not* affected matters as much as which are: most readers of the
note operate a different chain and should be able to stop reading at that sentence.

## Callouts

**Exactly one callout per note** — the Overview block. Nothing else. The curated list carries
everything else, so if you find yourself reaching for a second, the content is an entry in
the list or belongs in `## Breaking changes`.

## Tags, links and images

For a finalized release the heading, the compare link's right side and the image tag all
carry the plain version — never `-rc.N`. `scripts/retarget-tag.sh` does this.

The compare link's **base** is the previous *finalized* tag, with three dots:

```markdown
**Full Changelog**: https://github.com/ethereum-optimism/optimism/compare/op-node/v1.19.4...op-node/v1.19.5
```

git-cliff generates an RC base and older notes still carry one; the current convention is
finalized-to-finalized.

If a release carries recurring boilerplate from the previous release — the APKO migration
block was one — do not copy it forward blindly. Its text was self-limiting, and it also
implied a second image line. Ask before including it.

## Working notes

Keep the raw git-cliff bullets for cut PRs as HTML comments at the bottom of the draft while
iterating, each with a short reason. A reviewer can then see what was considered and
reinstate an entry in one edit. Delete them before publishing, or keep them if the release
manager prefers.

```markdown
<!--* op-core/fees: add Jovian DA-footprint calculation (#22163) — doesn't affect the batcher-->
```
