# Updating the reth dependency

The Rust workspace pins ~70 `reth-*` crates from `paradigmxyz/reth` to a single
git ref in [`rust/Cargo.toml`](Cargo.toml) — `tag = "vX.Y.Z"` when tracking a
release (the normal case; self-documenting), or `rev = "<sha>"` when pinning a
non-release commit. This guide describes how to bump that pin safely and keep
the shared `revm`/`alloy` versions in sync with it.

## When to update

- Picking up a fix or feature merged upstream that op-reth depends on.
- Periodic catch-up to limit drift before it becomes painful.
- After your own reth-side change has merged upstream.

Prefer the latest upstream release tag when one exists that contains the
changes you need. Tags are stable artifacts with corresponding upstream
releases and the most predictable downstream behavior. Fall back to a merge
commit on `main` only when no tagged release covers the needed change yet
(e.g. urgently picking up a fix). Avoid pinning to an unmerged PR branch for
anything we want to land — the merge commit on `main` is at least the version
main's CI actually validated.

## Procedure

1. Pick the new rev. For a specific upstream PR, take its merge commit (see
   `gh pr view <N> --repo paradigmxyz/reth --json mergeCommit`). Otherwise take
   the current head of `paradigmxyz/reth` `main`.

2. Audit what the current pin carries before moving off it. If the current pin
   is **not an ancestor of the target** (e.g. it was an unmerged-PR commit),
   find what it carried — the monorepo commit that introduced it
   (`git log -S '<pin>' -- rust/Cargo.toml`) and the upstream PR
   (`gh api repos/paradigmxyz/reth/commits/<sha>/pulls`) — and verify each
   carried change is contained in, or explicitly superseded by, the target.
   Never silently drop a fix the pin was carrying.

3. Update the pin. The ref is pinned in **four** manifests, not just the main
   workspace: `op-rbuilder` and `rollup-boost` are separate Cargo workspaces
   that path-depend on the op-reth crates while also pinning reth directly.
   Bumping only `rust/Cargo.toml` leaves them on the old ref, so their
   dependency graphs contain two reth versions and fail with E0308 type
   mismatches, breaking the `op-rbuilder-checks` / `rollup-boost-checks` gates.

   When moving to a release tag (the normal case):

   ```bash
   cd rust
   sed -i 's/tag = "<OLD_TAG>"/tag = "<NEW_TAG>"/g' \
     Cargo.toml \
     op-rbuilder/Cargo.toml \
     rollup-boost/crates/rollup-boost/Cargo.toml \
     rollup-boost/crates/flashblocks-rpc/Cargo.toml
   ```

   When pinning a non-release commit instead, use `rev = "<sha>"` in the same
   way (and switch back to `tag = ...` at the next release catch-up). Find any
   stragglers with `grep -rl '<OLD_TAG_OR_REV>' rust --include='Cargo.toml'`.
   The lockfiles record the resolved commit either way, so builds stay
   reproducible even if an upstream tag were to move.

4. Sync shared dependency versions to the new rev's pins. reth and the OP Stack
   share the `revm`/`revm-*`, `alloy-*` (core and main), `alloy-eip7928`, and
   the published reth-core crate families (`reth-primitives-traits`,
   `reth-codecs`, `reth-rpc-traits`, `reth-zstd-compressors`). reth pins them
   in its own workspace `Cargo.toml`; ours must declare the same versions so we
   build against the same types reth's APIs expose. Read reth's pins at the new
   rev and update the matching lines in `rust/Cargo.toml`:

   ```bash
   # from a reth checkout at the new rev:
   git show <NEW_REV>:Cargo.toml | grep -E '^(revm|alloy-|reth-)'
   ```

   Apply the same bumps in `op-rbuilder/Cargo.toml` (which additionally pins
   `revm-context`, `revm-context-interface`, and `revm-inspector` — take their
   versions from reth's `Cargo.lock` at the new rev) and
   `rollup-boost/crates/flashblocks-rpc/Cargo.toml`.

   Bump only these crates.io ecosystem crates. Leave the OP-internal path
   crates (`op-revm`, `op-alloy*`, `alloy-op-evm`, `alloy-op-hardforks`) alone —
   they live in-tree under `rust/`, not on crates.io.

   A version mismatch in the reth-core crates shows up as baffling
   `expected SealedHeader, found a different SealedHeader` errors with a
   "multiple different versions of crate `reth_primitives_traits`" note.
   Verify a single version survives unification:

   ```bash
   mise exec -- cargo tree -i reth-primitives-traits@<OLD_MINOR>  # should not match
   ```

   **Easy to miss:** cargo may have *already* floated these up in `Cargo.lock`
   when you bumped the rev — our declared versions are caret ranges (`"2.0.4"`
   admits `2.0.5`) and reth's higher pin wins unification. So the lock can be
   correct while the declared versions silently lag, and this step's
   `Cargo.lock` diff is often empty. Update the declared versions anyway: it
   keeps the manifest honest about what we actually build against and signals
   the sync to downstream consumers (e.g. Hardhat tracking `op-revm`).

5. Refresh the lockfiles — all three workspaces have their own. `cargo update
   -p reth` does **not** work — there is no top-level crate literally named
   `reth` in the dep graph; the workspace depends on `reth-*` subcrates. Pass
   any real reth subcrate; cargo cascades to every git dep sharing the same
   source:

   ```bash
   for d in . op-rbuilder rollup-boost; do
     (cd $d && mise exec -- cargo update reth-chainspec)
   done
   ```

6. Revisit the slot-preimage layout reference.
   `op-reth/crates/cli/src/commands/slot_preimages_seed.rs` replicates reth's
   private `SlotPreimages` MDBX layout and carries the rev it was copied from.
   Diff the upstream source between the revs; if unchanged, just update the rev
   in the comment, otherwise port the layout change:

   ```bash
   git -C <reth-checkout> diff <OLD_REV> <NEW_REV> -- \
     crates/stages/stages/src/stages/execution/slot_preimages.rs
   ```

7. Compile and adapt:

   ```bash
   mise exec -- cargo check --workspace --tests
   ```

   Fix each compile error, then re-run. Don't try to predict the full set of
   breakages in advance — let the compiler walk the dep graph. Each pass
   surfaces the next crate's errors. Remember the in-tree forks of upstream
   crates (`op-revm`, `alloy-op-evm`, kona's `fpvm_evm`) — when upstream
   reworks an API, the right fix is usually to mirror what the new upstream
   default/eth implementation does, found in the new rev's sources or the
   cargo registry sources under `~/.cargo/registry/src/`.

   **Gas accounting and other consensus-adjacent overrides** deserve extra
   rigor: derive the change as *(old upstream default → new upstream default)*
   applied onto our override, leaving the OP-specific branches byte-identical —
   never invent logic. Any new semantic branch (even fork-gated and inert
   today) gets a unit test, verified red against a deliberately broken variant
   and green against the real code.

   Then repeat for the vendored workspaces:

   ```bash
   (cd op-rbuilder && mise exec -- cargo check --workspace --tests)
   (cd rollup-boost && mise exec -- cargo check --workspace --tests)
   ```

8. Build, format, and test before pushing:

   ```bash
   mise exec -- cargo build -p op-reth
   just fmt-fix && just lint
   just test   # unit + doc tests — CI runs both, and doc examples call reth APIs too
   ```

   Only the `just` recipes use the repo-pinned nightly rustfmt — a bare
   `cargo fmt` on the stable toolchain reformats unrelated files and fails
   CI's `rust-fmt` gate. Also run the test suites of any vendored-workspace
   crate whose source you touched.

## Expect upstream churn beyond your target change

A rev bump is rarely "just a rev change." Upstream reth iterates trait
signatures, struct fields, re-exports, and feature flags between commits, and
any of them can require op-reth-side adaptation. Recent examples observed in
the wild:

- Trait methods gaining new parameters (e.g. `FullConsensus::validate_block_post_execution`
  gained `block_access_list_hash: Option<B256>`; `PayloadTypes::block_to_payload`
  gained `bal: Option<Bytes>`).
- Associated-type bounds (e.g. `PayloadTypes::ExecutionData` gaining a
  `From<Self::BuiltPayload>` bound, requiring a new `From` impl).
- Provider trait additions (e.g. `BalProvider` becoming a required bound on
  the engine API's `Provider`).
- Struct destructuring (e.g. `BlockBuilderOutcome` gaining a
  `block_access_list` field; `BuiltPayloadExecutedBlock.hashed_state` /
  `trie_updates` losing their `Either<Arc<...>, _>` wrappers).
- Renames (e.g. `ComputedTrieData::without_trie_input` → `::new`).
- Removed re-exports — sometimes the trait itself is deleted upstream while
  op-reth (and the wider OP Stack) still relies on it. In that case the
  smallest correct fix is to vendor the trait locally in the consuming crate
  (with a comment pointing at the upstream PR that removed it). See
  `rust/op-reth/crates/rpc/src/witness.rs` for an example.
- Type changes in test fixtures (e.g. struct fields flipping from `u16` to
  `Option<u16>`).

None of these are deep redesigns — most fixes are one to three lines — but
they need to be done before CI will pass.

## Picking the right target commit

In order of preference:

1. **Latest upstream release tag** that contains the change you need. Tags
   correspond to actual upstream releases and have the most predictable
   downstream behavior. List them with `git -C <reth-checkout> tag --sort=-v:refname | head`
   (or `gh release list --repo paradigmxyz/reth`).

2. **Merge commit on `main`** of a specific PR — for cases where the needed
   change has been merged but isn't in a release yet (e.g. urgent fix). Find
   it via `gh pr view <N> --repo paradigmxyz/reth --json mergeCommit`. The
   merge commit is stable (PR rebases no longer affect it) and is the version
   main's CI actually validated.

3. **Current `main` HEAD** — for periodic catch-up bumps when no specific PR
   is the trigger.

Avoid pinning to an unmerged PR branch tip for anything we want to land. It
moves under us when the PR rebases, may sit on a much newer main than our
current pin, and isn't a version any upstream CI has signed off on as a
release artifact.

If the only upstream change you need is a single PR but the PR branch is based
on much newer main than our pin, consider asking the upstream author to rebase
onto an older base, or accept the broader catch-up work as part of the bump.

## See also

- `docs/ai/rust-dev.md` — broader Rust workflow (build, test, lint).
- `rust/Cargo.toml` — where the pin lives (~70 occurrences), plus
  `rust/op-rbuilder/Cargo.toml` and the two `rust/rollup-boost` crate
  manifests.
- `rust/op-reth/crates/rpc/src/witness.rs` — example of vendoring a trait that
  upstream removed.
- `rust/op-reth/crates/cli/src/commands/slot_preimages_seed.rs` — replicated
  upstream MDBX layout; revisit on every bump.
