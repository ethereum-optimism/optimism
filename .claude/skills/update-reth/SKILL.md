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

2. **Isolate.** Work in a fresh git worktree (or jj workspace) based on latest
   `develop` — never on the main checkout's working copy. Run `mise trust` in the
   new directory.

3. **Execute the guide's procedure**, including the current-pin audit and
   `cd rust && just mirrors stale` before verification. Work every stale mirror
   and iterate the compile loop to green; don't ask the user to confirm each
   adaptation.

4. **Compound.** If the bump taught you something the guide doesn't cover, fold
   it into `rust/UPDATING-RETH.md` before review.

5. **Review and resolve.** Commit a review candidate and run code, security, and
   the **`reth-update-reviewer` agent** (`docs/ai/reth-update-review.md`).
   Complete its all-severities triage and selected investigations. Fix findings,
   then rerun every matching reviewer until the candidate is clean.

6. **Final verification.** Verify everything the guide lists on the reviewed
   head. On memory-constrained machines, use
   `systemd-run --user --scope -p MemoryMax=<n>G` with reduced `-j`, not
   `ulimit -v`. Any later edit returns to review, then verification.

7. **Ship.** Use one commit (`rust: update reth to <version>`) describing the pin
   move, shared-dependency sync, and API adaptations. Push only the verified
   reviewed head, then open the PR to `develop` with verification results.
