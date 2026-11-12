---
name: update-reth
description: Use when asked to bump, update, or upgrade the reth dependency pin in the OP Stack Rust workspaces, whether to a release tag, commit, or merged upstream PR.
---

# Update reth

Bump the `reth` pin (normally to the latest upstream release tag) and adapt the
OP Stack Rust workspaces to upstream API changes. The pin may point at an OP-maintained
fork rather than upstream directly — read the source out of `rust/Cargo.toml`.

The complete procedure — pin audit, workspace manifest, both lockfiles,
shared-dependency sync, compile-and-adapt loop, and verification — lives in
[`rust/UPDATING-RETH.md`](../../../rust/UPDATING-RETH.md). **Read it in full
first and follow it exactly**; this skill only adds the agent workflow around
it. Don't restate the guide to the user — execute it.

## Arguments

Optional target: a release tag (`v2.4.0`), a commit sha, or an upstream PR number.
Default: the latest upstream release tag (`gh release list --repo paradigmxyz/reth`).

## Workflow

1. **Orient.** Read the guide and the pinned reth git source from
   `rust/Cargo.toml`. Reuse or clone that source, ensure a remote points to
   `https://github.com/paradigmxyz/reth`, then fetch upstream `main`, tags, the
   old pin, and the selected target. Validate reused remotes before diffing.
   Skim upstream fixes newer than the candidate target (the forward sweep in
   `docs/ai/reth-update-review.md`) so a retarget decision comes before the
   adaptation work.

2. **Isolate.** Work in a fresh git worktree (or jj workspace) based on latest
   `develop` — never on the main checkout's working copy. Run `mise trust` in the
   new directory.

3. **Execute the guide's procedure**, including the current-pin audit and
   `cd rust && just mirrors stale` before verification. If fork patches
   survive the audit, or the target is not an upstream release tag, the pin
   moves to a tag on a new `op-*` fork branch (the guide's "The reth fork").
   Push that branch at the upstream base and prepare it in one prep PR: the
   CI commit and the OP `.github/CODEOWNERS` carried over from the previous
   branch, every retained patch and forward-sweep pick, and any fixes the new
   base needs, as separate commits. Re-run the forward sweep from its
   recorded upstream tip, then rebase-merge it, wait for fork CI to pass
   on the PR's `mergeCommit` (its last rebased commit), tag that commit, and
   confirm the fork's `codeowners/errors` check is empty. Meanwhile, the
   update PR may pin the prep PR's head by `rev`, in draft only. Work every
   stale mirror and iterate the compile loop to green; don't ask the user to
   confirm each adaptation.

4. **Compound.** If the bump taught you something the guide doesn't cover, fold
   it into `rust/UPDATING-RETH.md` before review.

5. **Review and resolve.** Commit a review candidate and run code, security, and
   the **`reth-update-reviewer` agent** (`docs/ai/reth-update-review.md`),
   including its partitioned full-range sweep over every bumped family; the
   mirror funnel alone does not clear a bump. Include its forward sweep for
   upstream fixes newer than the target, and get the user's decision:
   cherry-pick the fixes onto the fork branch (UPDATING-RETH, "The reth
   fork", then back to step 3 here), or retarget to a newer upstream release or commit (back to
   step 1 here with the new target, reusing the worktree). Complete its
   all-severities triage and selected investigations. Fix findings, then
   rerun every matching reviewer until the candidate is clean.

6. **Final verification.** Verify everything the guide lists on the reviewed
   head. On memory-constrained machines, use
   `systemd-run --user --scope -p MemoryMax=<n>G` with reduced `-j`, not
   `ulimit -v`. Any later edit returns to review, then verification.

7. **Ship.** Use one commit (`rust: update reth to <version>`) describing the pin
   move, shared-dependency sync, and API adaptations. Push only the verified
   reviewed head, then open the PR to `develop` with verification results.
   Open the companion optimism-premium PR in draft (guide step 10).

   **Merge gate: the pin is a tag.** A `rev` pin is allowed during
   development, but the update PR stays in draft as long as it has one; say so
   in its description. The required `rust-reth-tag-pin` check stays red until
   then. Only once the fork branch is fully prepared and its `op-<base>-<n>`
   tag exists, replace the `rev` with the tag, confirm the tag's commit is on
   the fork's `op-<base>` branch (the guide's "Pinning a fork change"), rerun
   the guide's step 2 check of the old branch, and then move the PR out of
   draft. Never mark the PR ready, enable auto-merge, or enqueue it with a
   `rev` pin, even with a repin planned as a follow-up.

8. **After merge.** Repin the premium PR to the merge commit (the guide's
   "After the monorepo PR merges").
