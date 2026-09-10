# Upstream mirrors: where OP code duplicates reth / revm / alloy

Some OP code **reproduces upstream logic instead of calling it** — a trait-method body
derived from an upstream default, a helper rebuilt locally, a set we enumerate by hand.
Each one is a place where upstream can move and nothing in our tree changes: no diff, no
failing test, no discussion to review. The recurring failure is always the same shape —
*we duplicated upstream logic, upstream moved, nothing told us.*

Every in-scope production mirror carries an **`UPSTREAM-MIRROR` tag** in its doc
comment. The tag records which upstream symbol the code mirrors and **the
upstream version it was last verified against**, so a bump identifies what
could have drifted rather than relying on a generic checklist.

The tags are the source of truth. This file is the guide to using them; it deliberately
does not list the sites, because a list in a doc goes stale the moment someone moves a
function.

## The tag

```rust
/// UPSTREAM-MIRROR(<kind>): <crate>@<version> `<upstream::symbol::path>`
///
/// <prose: what differs from upstream, and what to look at when re-checking>
```

| Field | |
| --- | --- |
| `<kind>` | one of the five below — it says what "re-verify" means here |
| `<crate>` | crates.io package name (`revm-handler`, `alloy-evm`), or `reth` for the git-pinned family |
| `<version>` | exact resolved semver for crates.io; for `reth`, the pin token: `rev:<sha7>`, `v<semver>`, or `pre-<pr>` for something upstream has since deleted |
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
| `delegate` | We wrap an upstream type and forward selected methods to it. | Checking newly defaulted methods and whether the inner type overrides them; inheriting the generic default can differ from forwarding to the inner implementation. |
| `set` | We enumerate an upstream set (variants, addresses, RPC methods). | Diffing membership. Prefer a test driven by upstream's `VARIANTS` over a hand-written list. |
| `port` | A one-time port pinned to an old upstream version, deliberately frozen. | Nothing on a routine bump. Reported as `frozen`, never as work. |

## Tooling

```bash
cd rust
just mirrors            # every mirror, with status
just mirrors stale      # only stale mirrors — the bump worklist
just mirrors --json     # machine-readable output
```

`just check-upstream-mirrors` runs its contract tests and validates the tags in
required Rust CI. It fails on malformed tags, unknown crates, ambiguous
resolved versions, or versions newer than an ordered pin. It does not fail on
stale tags.

Statuses: `current`, `stale` (review it), `frozen` (an intentionally old
`port`), and the tag errors `ahead`, `unknown-crate`, `ambiguous-version`, and
`malformed`.

## Using it on a bump

1. Bump the pin and refresh the lockfiles as usual (`rust/UPDATING-RETH.md`).
2. From the repository root, run `cd rust && just mirrors stale`. It lists
   every mirror whose recorded token differs from the currently resolved pin.
   Use the lockfile and tag diff from the merge base to distinguish entries
   made stale by this bump from pre-existing staleness. A reth rev is one opaque
   family pin, so moving it marks every non-port reth mirror stale.
3. Diff each named upstream symbol from the recorded version to the new pin and
   apply the kind-specific re-verification.
4. Advance the tag only after that check. The reviewer separately inspects every
   tag change relative to the merge base, so a current head tag is not proof of
   re-verification.
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

Find a fork pin's upstream base with
`git merge-base <pin> <upstream-remote>/main`; then
`git log <base>..<pin>` lists its cherry-picks and their files.

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

For crates.io mirrors and reth release tags, the `ahead` check catches an
overshot ordered version. Git revisions have no manifest-level ordering, so the
checker cannot detect an “ahead” rev. The reviewer-side control is mandatory for
every source: inspect each tag change relative to the merge base and require the
corresponding upstream diff or an explicit explanation.

## What is not tagged

- **Intra-repo duplicates** — ours copied from ours. Same failure mode, but there is no
  upstream version to record, so they get a plain doc note pointing at the original.
  `kona/bin/client/src/fpvm_evm/tx.rs` is the live example.
- **Mirrored tests.** `op-reth/crates/evm/src/lib.rs` reproduces upstream test fixtures.
  Real drift, low stakes; tagging them would bury the consensus-relevant entries.

## Adding a mirror

Don't, if calling upstream is possible. When duplication is unavoidable, write
the tag and its maintenance prose in the same commit, on the smallest item that
owns the duplicated behavior. Record the version you actually verified against.

## See also

- [reth-update-review.md](reth-update-review.md) — the bump review guide: the lockfile
  funnel, the precondition question, and the risk taxonomy these mirrors sit inside.
- [../../rust/UPDATING-RETH.md](../../rust/UPDATING-RETH.md) — the bump procedure.
- `rust/scripts/upstream-mirrors.py` — the checker.
