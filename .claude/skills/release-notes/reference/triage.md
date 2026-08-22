# Deciding which PRs belong in a release note

## Why the draft is noisy

`just release-notes` selects commits by include-path, and those paths are deliberately wide.
For the Go services:

```
--include-path "<component>/**/*"  --include-path "go.*"
--include-path "op-core/**/*"      --include-path "op-service/**/*"
```

`op-core/` and `op-service/` hold code shared by every Go service, by the test harnesses
(`op-devstack`, `op-e2e`, `op-acceptance-tests`) and by tools that ship separately
(`op-deployer`, `op-chain-ops`). So a draft picks up many commits that never reach the
binary — the op-batcher v1.16.13 draft listed 21 PRs, of which 8 touched nothing op-batcher
compiles. The kona-* components filter on all of `rust/kona/**`, `rust/op-alloy/**` and
`rust/alloy-op*/**`, so a kona-node draft picks up kona-client, kona-host and kona-sp1 work
too.

## The test that matters

**Not** "did this PR touch `<component>/`". Shared code is compiled into the binary, so a
change under `op-service/` or `op-core/` can absolutely change how the component behaves —
`op-service/txmgr`, `op-service/bgpo` and `op-core/fees` all reach op-batcher.

The right question: **does this PR change code compiled into this binary, and does that
change alter behaviour an operator or downstream importer would notice?**

`scripts/pr-facts.sh` answers the first half exactly and tags every PR:

| Tag | Meaning | Default |
| --- | --- | --- |
| `LINKED` | Changed a package in the binary's transitive dependency set; the listed packages are the ones that reach it | Candidate — apply the judgment pass |
| `DEPS` | Changed only the dependency manifests | Drop, unless it is a security bump |
| `--` | Touched nothing the binary compiles | Drop |
| `?` | Dependencies could not be resolved | Judge by hand |

## The judgment pass on LINKED rows

A PR whose substance is test infrastructure often nudges a linked package in passing. For
each `LINKED` row, look only at the part of the diff landing in the listed packages:

```bash
gh pr view <N> --json title,body      # intent
gh pr diff <N>                        # when the intent is unclear
```

`_test.go` files are already excluded from linkage, so a Go PR that only adds coverage is
tagged `--`. Rust test code lives behind `#[cfg(test)]` inside the crate and cannot be
excluded by path, so `LINKED` Rust rows still need this check by hand.

**Linkage proves the package is compiled in, not that the changed function is on the
component's runtime path.** This is the trap that survives the mechanical pass.
`op-core/fees` is genuinely linked into op-batcher, but the Jovian DA-footprint work
(#22163, #22219) is reached from the `op-chain-ops/cmd/check-*` tools, not from anything the
batcher runs — both were cut with the note "doesn't affect the batcher". Similarly,
`op-node/rollup/derive` is linked into op-batcher, but a derivation-only change there is not
batcher-facing. Ask what the component does, not just what it links:

```bash
grep -rn "<ChangedSymbol>" <component>/ --include='*.go' | grep -v _test.go
```

**Keep** when the change to the linked package:

- alters runtime behaviour — what gets built, submitted, derived, gossiped, or logged
- changes a flag, env var, config key, default, metric, or RPC surface
- fixes a bug, panic, race, or correctness issue reachable in production
- is a security fix

**Drop** when it is:

- test-only, fixtures, mocks, or `testutils` churn
- a pure rename/move with no behaviour change
- a dev-feature toggle removal for a feature never on in production
- comment, TODO, or docs cleanup
- another component's work that merely brushed a shared package
- a Go API change with no operator-facing effect — mention under `### Other` at most

**Keep a cross-component PR** when the component embeds the other one. op-supernode runs
virtual op-nodes, so op-node's follow-source reorg metrics belong in the supernode notes even
though the PR touches no `op-supernode/` path.

**A change to a dormant feature** gets one line under a `### <Feature> (not yet in
production)` heading if it affects this component once the feature activates, and is **cut
entirely** if it is a no-op here and only matters to another consumer of the shared code.
Check liveness against the registry — see `house-style.md`.

## Fixes for bugs that never shipped

Before letting a scary-looking fix set the upgrade recommendation, check whether the bug is
inside the same release range:

```bash
git tag --contains <sha-that-introduced-the-bug> | grep '<component>/v'
```

Empty output means the bug never reached a release: keep the PR in the list, but do not let
it set the recommendation — and do not mention it in the note. "Both land in this release, so
no published version is affected" is a triage conclusion, not something the reader needs.

## Dependency and security bumps

`DEPS` rows are usually noise, but a bump patching a CVE in a library the binary links is
worth a line, and if it is the most serious thing in the release it raises the
recommendation. Check the module is really in the binary first:

```bash
go list -deps ./<component>/cmd | grep <module-path>
```

Renovate/dependabot bumps with no security label: drop.

## Ground truth

Two published releases, checked against their regenerated raw drafts:

- **`op-node/v1.19.4`** — 32 PRs raw, 15 published. Dropped: op-dispute-mon and op-challenger
  work, kona changes, dependabot bumps, test fixes, a stale-TODO removal, two dev-feature
  toggle removals.
- **`op-batcher/v1.16.12`** — 23 PRs raw, 1 published. Almost the entire draft was other
  components' work reaching op-batcher's include paths.

Regenerate any published release's raw draft to check a call:

```bash
GITHUB_TOKEN=$(gh auth token) just release-notes op-node v1.19.3 v1.19.4
```

The v1.19.5 train is the closest reference for current practice. Surviving PR counts:
op-node 10 of 30, op-batcher 6 of 21, op-supernode 8 of 24, kona-node 6 of 17.

## Drafts with more than one section

If earlier RCs were never published, git-cliff emits several `## What's Changed in <tag>`
sections in one draft. For a finalized release, merge them under the final tag, dedupe by PR
number, and triage the union — the release covers all of it.
