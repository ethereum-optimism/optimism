# House style for OP Stack release notes

This file is the source of truth. Changing the style means editing it, not inferring a new
convention from one release.

The target is a **curated change list**, as in
[`op-node/v1.19.6`](https://github.com/ethereum-optimism/optimism/releases/tag/op-node%2Fv1.19.6)
and
[`op-reth/v2.4.3`](https://github.com/ethereum-optimism/optimism/releases/tag/op-reth%2Fv2.4.3).
Read those two before drafting.

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
`> [!IMPORTANT]` for either recommended level, `> [!CAUTION]` for `required`. Those three and
no others. This block is also the note's only callout — if you reach for a second, the
content is an entry in the change list, or belongs in `## Breaking changes`. The one exception
is `kona-node`, whose callout should always be:

```markdown
> [!NOTE]
>
> `kona-node` is not production-ready. Production deployments should use `op-node`.
```

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

For a `DevFeatures` bit, the default is the answer: we ship the defaults for feature toggles,
so anything behind a bit that is not forced to `true` is dormant unless a chain has
explicitly set it. `docs/ai/devfeatures.md` lists which are default-on.

For anything expressed neither as a hardfork nor a DevFeature — dispute game types, say —
there is no equivalent lookup, so ask the release manager rather than guessing.

A change that turns out to be confined to an unreleased feature is cut — see "A change with
no user-visible impact" below.

**Do not narrate an attack the code path cannot currently suffer.** State what the fix aligns
or corrects; leave the exploit narrative out until the path is live.

## Curating the change list

**Add domain headings** beyond the Shape's spine where they carry more meaning. Drop any
heading that would be empty, and use a flat list for a short release.

**Group PRs that are one logical change** into one entry with all their numbers:
`... are no longer required when only permissioned game types are configured (#21270, #21681)`.

**Write self-contained entries.** The PR title is usually not enough on its own — that is why
the raw list is being replaced. Say what changed and why an operator cares:

> - Tear down the whole VM process group when `--vm-timeout` is hit, preventing orphaned VM processes from lingering after a timeout (#21268)

**A change with no user-visible impact does not appear at all** — not under
`### Miscellaneous`, not as a summarising line. Three cases recur:

- *Internal churn.* A reader gains nothing from being told that something they cannot
  observe was rearranged.
- *Go-module importability.* We do not maintain releases of the monorepo as a Go module, so
  a change that only unblocks downstream importers — moving a symbol to a leaf package,
  shrinking a build closure — is cut like any other churn, however much work it was.
- *Anything confined to an unreleased feature.* No `(not yet in production)` heading, no
  explanatory Note in the Overview. The test is whether the change reaches a live path, not
  what motivated it: a fix written for an interop scenario that also alters pre-interop
  derivation stays in, described by its live effect; the same fix, if it only fires once the
  fork activates, does not.

The one exception is a release that would otherwise have an empty change list, where one
summarising line is more honest than publishing nothing.

**Only mention a PR more than once** if it included multiple logical changes worth
describing separately.

**Reference PRs as bare `(#NNNNN)`** at the end of the entry — GitHub links them
automatically in release bodies. Drop `by @author`.

**Drop git-cliff's `## New Contributors` section.** It tells a reader nothing about what is
in the release.

## Breaking changes

When a change requires operator action before upgrading, it gets its own section, with a bold
short name and the required action stated plainly:

```markdown
## Breaking changes

- **`--cannon-kona-experimental-witness-endpoint` flag removed** (#20498). The `debug_executePayload`-based witness path is now the default for kona-cannon games. Operators passing this flag must remove it before upgrading — op-challenger will reject it as unknown.
```

**State the action, not why the old behaviour was wrong.** The entry exists so an operator
knows what to do before upgrading; the history of the flag or field belongs in the PR.

Go-API-only changes are **not** breaking changes for this purpose, and are not entries
anywhere else either — see "A change with no user-visible impact" above.

## Chain Configuration

A release that moves a chain's embedded config gets its own section — most often a new
hardfork activation time arriving with a superchain-registry pin bump. Name the chains and
the exact values, and say what happens to a node that upgrades late:

```markdown
## Chain Configuration

- **New World Chain Karst activation times** (#22624). The embedded superchain registry gains:

  - `sepolia/worldchain` — `karst_time = 1788868800` (Tue 8 Sep 2026 12:00:00 UTC)
  - `mainnet/worldchain` — `karst_time = 1789992000` (Mon 21 Sep 2026 12:00:00 UTC)

  A World Chain node on an earlier release, running the built-in `--network` config rather than an explicit rollup config, will not activate Karst and will diverge from the chain. No other chain's activation times change in this release.
```

Saying which chains are *not* affected matters as much as which are: most readers of the
note operate a different chain and should be able to stop reading at that sentence.

## Tags, links and images

The release **title** is `<component> <version>`, with a space — `op-node v1.19.6`, not the
tag's slash form `op-node/v1.19.6`.

For a finalized release the heading, the compare link's right side and the image tag all
carry the plain version — never `-rc.N`.

A published note **must** compare finalized tag to finalized tag. git-cliff generates an RC
base, and `scripts/retarget-tag.sh` does not touch it — set it to the previous finalized tag
by hand, with three dots:

```markdown
**Full Changelog**: https://github.com/ethereum-optimism/optimism/compare/op-node/v1.19.5...op-node/v1.19.6
```

If a release carries recurring boilerplate from the previous release, do not copy it forward
blindly — such blocks are often self-limiting, or imply a second image line. Ask first.
