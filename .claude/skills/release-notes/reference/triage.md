# Deciding which PRs belong in a release note

## Why the draft is noisy

`just release-notes` selects commits by include-path, and those paths are deliberately
wide. For the Go services:

```
--include-path "<component>/**/*"  --include-path "go.*"
--include-path "op-core/**/*"      --include-path "op-service/**/*"
```

`op-core/` and `op-service/` hold shared code used by every Go service *and* by the test
harnesses (`op-devstack`, `op-e2e`, `op-acceptance-tests`) and by tools that ship
separately (`op-deployer`, `op-chain-ops`). So a component's draft picks up a large number
of commits that never reach that component's binary — the op-batcher v1.16.13-rc.2 draft
listed 21 PRs, of which 8 touched nothing op-batcher compiles.

The kona-* components filter on all of `rust/kona/**`, `rust/op-alloy/**` and
`rust/alloy-op*/**`, so a kona-node draft picks up kona-client, kona-host, kona-sp1 and
prestate-build work too.

## The test that matters

**Not** "did this PR touch `<component>/`". Shared code is compiled into the binary, so a
change under `op-service/` or `op-core/` can absolutely change how the component behaves —
`op-service/txmgr`, `op-service/bgpo` and `op-core/fees` all reach op-batcher and all can
alter what it does on chain.

The right question is: **does this PR change code that is compiled into this binary, and
does that change alter behaviour an operator or downstream importer would notice?**

`scripts/pr-facts.sh` answers the first half exactly, via `go list -deps`, and tags every
PR:

| Tag | Meaning | Default |
| --- | --- | --- |
| `LINKED` | Changed a package in the binary's transitive dependency set. The listed packages are the ones that reach it. | Candidate — apply the judgment pass below |
| `DEPS` | Changed only `go.mod`/`go.sum` | Drop, unless it is a security bump (see below) |
| `--` | Touched nothing the binary compiles | Drop |
| `?` | Non-Go component, or no component given | Judge by hand |

## The judgment pass on LINKED rows

`LINKED` is necessary but not sufficient — a PR whose substance is test infrastructure
often also nudges a linked package in passing. For each `LINKED` row, look only at the
part of the diff that lands in the listed packages:

```bash
gh pr view <N> --json title,body      # intent
gh pr diff <N>                        # when the intent is unclear
```

`_test.go` files are already excluded from linkage, so a Go PR that only adds coverage to
a linked package is tagged `--`. Rust test code lives behind `#[cfg(test)]` inside the
crate and cannot be excluded by path, so `LINKED` Rust rows still need this check by hand.

**Linkage proves the package is compiled in, not that the changed function is on the
component's runtime path.** This is the trap that survives the mechanical pass.
`op-core/fees` is genuinely linked into op-batcher, but the Jovian DA-footprint work
(#22163, #22219) is reached from the `op-chain-ops/cmd/check-*` tools, not from anything
the batcher runs — the release manager cut both from op-batcher v1.16.13 with the note
"doesn't affect the batcher". When a linked change is in shared code, check that the
component actually calls it:

```bash
# Does anything the component compiles reach the changed symbol?
grep -rn "<ChangedSymbol>" <component>/ --include='*.go' | grep -v _test.go
```

Similarly, `op-node/rollup/derive` is linked into op-batcher, but a derivation-only change
there (#22101) is not batcher-facing. Ask what the component does, not just what it links.

**Keep** when the change to the linked package:

- alters runtime behaviour — what gets built, submitted, derived, gossiped, or logged
- changes a flag, env var, config key, default, metric, or RPC surface
- fixes a bug, panic, race, or correctness issue reachable in production
- changes a Go API that downstream importers use — these belong under `### Other` with a
  note that no operator action is needed, never in `## Breaking changes`
- is a security fix

**Drop** when it is:

- test-only, fixtures, mocks, or `*_test.go` and `testutils` churn
- a pure rename/move with no behaviour change *and* no exported-API change
- a dev-feature toggle removal for a feature that was never on in production
- comment, TODO, or docs cleanup
- another component's work that merely brushed a shared package

**Keep a cross-component PR** when the component embeds the other one. op-supernode runs
virtual op-nodes, so op-node's follow-source reorg metrics (#22106) belong in the supernode
notes even though the PR touches no `op-supernode/` path.

**Keep a change to a dormant feature**, but section it. Interop/Lagoon, ZK dispute games
and super dispute games are not live anywhere, so those changes cannot affect operators
today however alarming they sound. They belong under a
`### <Feature> (not yet in production)` heading, not in the Overview.

**Keep it anyway** in one case: if pruning would empty the list, keep the most substantial
refactor and let the Overview say plainly that nothing functional changed —
`op-batcher/v1.16.12` shipped exactly one entry, a shared-helper refactor. An empty change
list is worse than an honest one.

## Fixes for bugs that never shipped

Before letting a scary-looking fix set the upgrade recommendation, check whether the bug it
fixes is inside the same release range. `op-node/v1.19.5-rc.2` contains #22360, which
fixes a permanent block-production freeze — but the freeze was introduced by #22241, also
in that range, so no published version was ever affected. Describing it as a fix operators
need would be wrong.

```bash
git tag --contains <sha-that-introduced-the-bug> | grep '<component>/v'
```

Empty output means the bug never reached a release: keep the PR in the list, but do not
let it set the severity.

## Dependency and security bumps

`DEPS` rows are usually noise, but a bump that patches a CVE in a library the binary
actually links is worth a line — and if it is the most serious thing in the release, it
raises the upgrade recommendation. Check whether the bumped module is really in the
binary before promoting it:

```bash
go list -deps ./<component>/cmd | grep <module-path>
```

Renovate/dependabot bumps with no security label: drop.

## Ground truth

Two published releases, checked against their regenerated raw drafts:

- **`op-node/v1.19.4`** — 32 PRs in the raw draft, 15 published. Dropped: op-dispute-mon
  and op-challenger work, kona changes, dependabot bumps, `op-acceptance-tests`/`op-e2e`
  fixes, a stale-TODO removal, and two dev-feature toggle removals.
- **`op-batcher/v1.16.12`** — 23 PRs in the raw draft, 1 published. Almost the entire
  draft was other components' work reaching op-batcher's include paths.

Regenerate any published release's raw draft to check a call:

```bash
GITHUB_TOKEN=$(gh auth token) just release-notes op-node v1.19.3 v1.19.4
```

Note that these two were pruned harder than the rule above strictly implies — v1.16.12
dropped shared-code changes that were genuinely linked. Treat them as the floor for how
aggressive pruning can be, not as a target. When a linked change really does alter
behaviour, keep it.

The v1.19.5 train is the closest reference for current practice, since its notes were
drafted by this skill and then corrected by the release manager. Surviving PR counts:
op-node 10 of 30, op-batcher 6 of 21, op-supernode 8 of 24, kona-node 6 of 17.

## Drafts with more than one section

If earlier RCs were never published, git-cliff emits several
`## What's Changed in <tag>` sections in the same draft (the op-supernode v1.0.1-rc.2
draft carries a v1.0.0-rc.7 section below it). For a finalized release, **merge them into
a single section** under the final tag, dedupe by PR number, and triage the union. The
release covers all of it, and an operator should not have to reconstruct the range.
