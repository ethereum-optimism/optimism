# Upstream mirrors: where OP code duplicates reth / revm / alloy

Some OP code **reproduces upstream logic instead of calling it** — a trait-method body
derived from an upstream default, a helper rebuilt locally, a set we enumerate by hand.
Each one is a place where upstream can move and nothing in our tree changes: no diff, no
failing test, no discussion to review. The recurring failure is always the same shape —
*we duplicated upstream logic, upstream moved, nothing told us.*

Every such site carries an **`UPSTREAM-MIRROR` tag** in its doc comment. The tag records
which upstream symbol the code mirrors and **the upstream version it was last verified
against**, so a bump produces a worklist proportional to what actually drifted rather than
a checklist to tick.

The tags are the source of truth. This file is the guide to using them; it deliberately
does not list the sites, because a list in a doc goes stale the moment someone moves a
function.

## The tag

```rust
/// UPSTREAM-MIRROR(<kind>): <crate>@<version> <upstream::symbol::path>
///
/// <prose: what differs from upstream, and what to look at when re-checking>
```

| Field | |
| --- | --- |
| `<kind>` | one of the five below — it says what "re-verify" means here |
| `<crate>` | crates.io package name (`revm-handler`, `alloy-evm`), or `reth` for the git-pinned family |
| `<version>` | exact resolved semver for crates.io; for `reth`, the pin token: `rev:<sha7>`, `v<tag>`, or `pre-<pr>` for something upstream has since deleted |
| `<symbol>` | full upstream path, so the checker and the reader can both find it |

Put the tag on the smallest item that owns the duplication — the method, not the `impl`
block, unless the whole impl is the mirror. Repeat the tag if one item mirrors two upstream
sources. The prose underneath is the part a machine can't carry: say what we changed and
what to look at, not that a mirror exists.

Write it on one line and run `just fmt-fix`. This workspace sets `wrap_comments`, so
rustfmt will split a tag that exceeds `comment_width` onto a second `///` line — the
checker rejoins the doc paragraph before parsing, so both forms work and you never have to
hand-manage the wrapping.

### Kinds

| Kind | What it means | Re-verify by |
| --- | --- | --- |
| `override` | We implement a method upstream also defaults. | Re-deriving as *(old upstream default → new upstream default)* applied onto our body, leaving OP-specific branches byte-identical. |
| `copy` | We reproduce an upstream function body inline. | Diffing the two bodies. Then asking whether the copy can be deleted in favour of calling upstream. |
| `delegate` | We wrap an upstream type and forward to it. | Checking for **newly defaulted** methods we now inherit — the risk is what we *don't* list. |
| `set` | We enumerate an upstream set (variants, addresses, RPC methods). | Diffing membership. Prefer a test driven by upstream's `VARIANTS` over a hand-written list. |
| `port` | A one-time port pinned to an old upstream version, deliberately frozen. | Nothing on a routine bump. Reported as `frozen`, never as work. |

## Tooling

```bash
cd rust
just mirrors            # every mirror, with status
just mirrors stale      # only what's behind the pin — the bump worklist
just mirrors --json     # for CI comments
```

`just check-upstream-mirrors` (wired into `just lint`) fails on tags that are **malformed**,
name a **crate we no longer depend on**, or claim a verified version **newer than the pin**.
It does *not* fail on stale tags — see below.

Statuses: `current`, `stale` (behind the pin — review it), `frozen` (a `port`, expected to
lag), `ahead` / `unknown-crate` / `malformed` (tag bugs).

## Using it on a bump

1. Bump the pin and refresh the lockfiles as usual (`rust/UPDATING-RETH.md`).
2. `just mirrors stale` — this is the worklist, and it is scoped to the crates that
   actually moved. A revm-only bump does not put the reth mirrors in front of you.
3. For each entry, get the upstream diff for that symbol between the tag's version and the
   new pin, and apply the kind's re-verify action.
4. **Advance the version in the tag only after you have actually re-checked it.** That edit
   is a one-line diff sitting next to the mirror — the ideal place for a reviewer to ask
   whether the re-derivation really happened.
5. If you deleted a mirror by calling upstream instead, delete the tag. That is the best
   outcome available and the number going down is the metric worth watching.

### Which repo you diff a `reth` mirror against

`reth` tags name the *family*, not the repo. The git source is read from `rust/Cargo.toml`
and printed by `just mirrors`, so a move between remotes doesn't invalidate every tag —
check the output for what we currently build against.

That source is OP's own reth fork — read the current remote out of `just mirrors` rather
than assuming the one written here. **It is not a drift source.** Its branch points at
an upstream reth commit — preferably a release — and carries temporary cherry-picks of work
that has not merged upstream yet. It does not accumulate divergence over time, so diffing a
mirror against the pinned rev is very nearly diffing against upstream reth, and the delta is
those cherry-picks.

Two practical consequences:

- Diff against **what we build** — the pinned rev in the fork. That is the code our mirrors
  have to match.
- If a mirrored symbol lives in a file a cherry-pick touches, say so in the review. That is
  the one place fork and upstream disagree, and the disagreement is temporary: it
  disappears when the patch merges upstream and the fork rebases onto it.

Confirming the delta is cheap — the cherry-picks are the commits the pinned rev has that
its upstream base does not, so `git log <upstream-base>..<pin>` in a checkout of the fork
names them and their files.

## Should CI block a bump while a mirror is stale?

**No — report, don't block.** The incentive shape is right (a bump costs what we have
duplicated, which discourages duplicating) but a hard gate is satisfiable with one `sed`,
and that makes things *worse*, not merely no better: before the sed the tag honestly says
"unverified since 41.0.0"; after it, the tag asserts verification that never happened. A
gate that is cheaper to launder than to satisfy converts a true signal into a false one.

The rigidity compounds it. Most bumps move crates nowhere near a mirror, so most gate
failures would be noise, and the `sed` becomes routine rather than exceptional.

So the split above: CI **blocks** only on tag bugs, which are unambiguous defects and can't
be waved through by editing a version. Staleness is surfaced as a worklist and reviewed
like any other finding.

What gives it teeth instead is the `ahead` check: a blind global find/replace overshoots
the pin and fails `just lint`, which catches the specific gaming move the gate would have
invited. Advancing a single tag deliberately still works — and it is a one-line diff next
to the mirror, which is where a reviewer should be asking whether the re-derivation
happened.

## Next step: derive the worklist from the upstream diff

Everything above shares one structural weakness. **The tags record where our copy is; they
cannot record whether upstream moved.** That is a fact about upstream, and no field we
maintain by hand can be authoritative about it — which is exactly why the staleness gate
was gameable.

The fix is a check that reads the upstream sources directly: for each tag, diff its named
symbol between the tag's verified version and the new pin, and report the ones that
actually changed. That signal is **immune to what our tags claim** — advancing a version
does not make a real upstream change disappear — so it can carry the weight the string
comparison could not:

- It is the honest blocking gate, if we want one. "This mirrored symbol changed upstream
  and the mirror was not touched in this PR" is a true statement about risk, not a
  bureaucratic one.
- It degrades usefully. A symbol that can no longer be resolved upstream has been renamed,
  moved or deleted — itself a finding, and one the current checker cannot see.

**Cost, so whoever picks this up knows what they are signing up for.** It needs upstream
sources in CI at *two* versions. For the crates.io mirrors (`revm-*`, `alloy-evm`) that is
cheap — both versions are fetchable from the registry. For the `reth` mirrors it means a
clone, which is the expensive part; a blobless or shallow fetch of just the two revs is the
obvious mitigation but has not been measured. A reasonable first cut is to implement it for
the crates.io mirrors only and leave the reth ones on the version comparison, since that
split follows the cost.

**Diff `reth` mirrors against the pinned rev in the fork**, not against upstream reth. The
fork is what we build, so it is what our mirrors must match, and per the section above it
tracks upstream closely enough that the two are near-equivalent. Diffing against upstream
instead would report the cherry-picks as drift on every run — noise that has nothing to do
with whether our copy is stale. A checker that wants to distinguish the two cases can name
the cherry-picked files separately, since they are exactly the commits between the fork's
upstream base and the pin.

## What is not tagged

- **Intra-repo duplicates** — ours copied from ours. Same failure mode, but there is no
  upstream version to record, so they get a plain doc note pointing at the original.
  `kona/bin/client/src/fpvm_evm/tx.rs` is the live example.
- **Mirrored tests.** `op-reth/crates/evm/src/lib.rs` reproduces upstream test fixtures.
  Real drift, low stakes; tagging them would bury the consensus-relevant entries.

## Adding a mirror

Don't, if calling upstream is possible — that is the whole point of the inventory. Two
examples of the choice, both in kona, both doing the same job: the SP1 precompile provider
calls `precompile_output_to_interpreter_result`; the FPVM one rebuilds it and is tagged
`copy` as a result.

When it genuinely isn't possible, write the tag and the prose in the same commit as the
duplication, and record the version you verified against — not the version you happened to
be on.

## See also

- [reth-update-review.md](reth-update-review.md) — the bump review guide: the lockfile
  funnel, the precondition question, and the risk taxonomy these mirrors sit inside.
- [../../rust/UPDATING-RETH.md](../../rust/UPDATING-RETH.md) — the bump procedure.
- `rust/scripts/upstream-mirrors.py` — the checker.
