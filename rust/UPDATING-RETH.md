# Updating the reth dependency

The Rust workspace pins ~70 `reth-*` crates from `paradigmxyz/reth` to a single
git rev in [`rust/Cargo.toml`](Cargo.toml). This guide describes how to bump
that rev safely.

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

2. Update the rev. All ~70 references share a single rev string, so a single
   replacement covers them:

   ```bash
   cd rust
   sed -i 's/rev = "<OLD_REV>"/rev = "<NEW_REV>"/g' Cargo.toml
   ```

3. Refresh `Cargo.lock`. `cargo update -p reth` does **not** work — there is no
   top-level crate literally named `reth` in the dep graph; the workspace
   depends on `reth-*` subcrates. Pass any real reth subcrate; cargo cascades
   to every git dep sharing the same source:

   ```bash
   mise exec -- cargo update reth-chainspec
   ```

4. Compile and adapt:

   ```bash
   mise exec -- cargo check --workspace --tests
   ```

   Fix each compile error, then re-run. Don't try to predict the full set of
   breakages in advance — let the compiler walk the dep graph. Each pass
   surfaces the next crate's errors.

5. Build, format, and test before pushing:

   ```bash
   mise exec -- cargo build -p op-reth
   just fmt-fix && just lint
   just test-unit
   ```

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
- `rust/Cargo.toml` — where the rev string lives (~70 occurrences).
- `rust/op-reth/crates/rpc/src/witness.rs` — example of vendoring a trait that
  upstream removed.
