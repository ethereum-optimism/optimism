# House style for OP Stack release notes

The target is a **curated change list**, as in
[`op-challenger/v1.9.4`](https://github.com/ethereum-optimism/optimism/releases/tag/op-challenger%2Fv1.9.4)
and
[`op-contracts/v8.0.0-rc.2`](https://github.com/ethereum-optimism/optimism/releases/tag/op-contracts%2Fv8.0.0-rc.2).
Read those two before drafting.

A curated note is not the git-cliff list with prose bolted on top: the PR list is *replaced*
by grouped, self-contained entries. Because each entry explains itself, the stack of
callouts older notes used to supply context is unnecessary.

This file is the source of truth. Changing the style means editing it, not inferring a new
convention from one release.

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

One standard sentence carries the upgrade recommendation, in a callout at the top of
`## Overview`. Sentence one names the release type and what it contains; sentence two gives
the recommendation.

Pick the release type from **what is in the diff**, not by comparing version numbers — our
tags are not strict semver, so the numbers do not tell you this:

| Type | Use when |
| --- | --- |
| patch | No new features — fixes, dependency work, internal change |
| minor | Small feature work alongside the fixes |
| major | Something significant, or a large volume of change |

Then exactly one of:

| Recommendation | Use when |
| --- | --- |
| `It is an optional upgrade for all users.` | Nothing operator-visible |
| `It is an optional but recommended upgrade for all users.` | Fixes or features worth having, none urgent |
| `It is a recommended upgrade for all users.` | A fix operators are likely to want |
| `It is a required upgrade for <scope>.` | Not upgrading risks a halt, consensus divergence, or stalled safe-head progression. Name the scope, and say who is unaffected |

Scope it to a role when only that role benefits — "It is an optional upgrade, but
recommended for sequencers" — rather than recommending it to everyone on the strength of
something most operators never see.

**The callout is always `> [!NOTE]`**, the one exception being `required`, which uses
`> [!CAUTION]`. Holding the block type constant is the point: urgency comes from the
sentence's fixed vocabulary, and the block only makes that sentence findable. Varying the
block with severity is how a routine recommended upgrade ended up flagged as `[!WARNING]`,
which overstates it and makes the recommendation harder to read.

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

- If the change does nothing for *this component*, and only matters to another consumer of
  the shared code, **cut it entirely**.
- If it affects the component but only once the feature activates, give it one line under a
  `### <Feature> (not yet in production)` heading and add the standard Note paragraph to the
  Overview.

**Do not narrate an attack the code path cannot currently suffer.** State what the fix aligns
or corrects; leave the exploit narrative out until the path is live.

## Curating the change list

**Group by domain or change type.** `### Features` / `### Bug fixes` / `### Other` is the
default spine; add domain headings where they carry more meaning. Drop any heading that would
be empty, and use a flat list for a short release.

**Group PRs that are one logical change** into one entry with all their numbers:
`... are no longer required when only permissioned game types are configured (#21270, #21681)`.

**Write self-contained entries.** The PR title is usually not enough on its own — that is why
the raw list is being replaced. Say what changed and why an operator cares:

> - Tear down the whole VM process group when `--vm-timeout` is hit, preventing orphaned VM processes from lingering after a timeout (#21268)

**Omit pure internal churn.** A reader gains nothing from being told that something they
cannot observe was rearranged. The one exception is a release that would otherwise have an
empty change list, where one summarising line is more honest than publishing nothing.

**Only mention a PR more than once** if it included multiple logical changes worth
describing separately.

**Reference PRs as bare `(#NNNNN)`** at the end of the entry — GitHub links them
automatically in release bodies. Drop `by @author`.

**Drop git-cliff's `## New Contributors` section.** It tells a reader nothing about what is
in the release.

## Breaking changes

When a change requires operator action before upgrading, it gets its own section above
`## What's Changed`, with a bold short name and the required action stated plainly:

```markdown
## Breaking changes

- **`--cannon-kona-experimental-witness-endpoint` flag removed** (#20498). The `debug_executePayload`-based witness path is now the default for kona-cannon games. Operators passing this flag must remove it before upgrading — op-challenger will reject it as unknown.
```

Go-API-only changes are **not** breaking changes for this purpose. They affect downstream
importers, not operators; mention them under `### Other` if they are worth mentioning at all.

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
