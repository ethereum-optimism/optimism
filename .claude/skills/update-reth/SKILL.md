---
name: update-reth
description: Update the pinned upstream reth dependency across the Rust workspaces — pin bump, shared dependency sync, compile-and-adapt loop, full verification, and PR. Use when asked to bump, update, or upgrade reth (e.g. "update to reth v2.4.0").
---

# Update reth

Bump the `paradigmxyz/reth` pin (normally to the latest release tag) and adapt the
OP Stack Rust workspaces to upstream API changes.

The complete procedure — pin audit, the four manifests, shared dependency sync,
lockfiles, the compile-and-adapt loop, and verification — lives in
[`rust/UPDATING-RETH.md`](../../../rust/UPDATING-RETH.md). **Read it in full first
and follow it exactly**; this skill only adds the agent workflow around it. Don't
restate the guide to the user — execute it.

## Arguments

Optional target: a release tag (`v2.4.0`), a commit sha, or an upstream PR number.
Default: the latest upstream release tag (`gh release list --repo paradigmxyz/reth`).

## Workflow

1. **Orient.** Read the guide. Find a local `paradigmxyz/reth` checkout (ask the
   user if you can't find one) and fetch its tags — you'll need it for tag
   resolution, API archaeology, and diffing replicated code.

2. **Isolate.** Work in a fresh git worktree (or jj workspace) based on latest
   `develop` — never on the main checkout's working copy. Run `mise trust` in the
   new directory.

3. **Execute the guide's procedure**, including the current-pin audit before
   moving off it. Iterate the compile loop to green rather than enumerating
   breakages up front; don't ask the user to confirm each adaptation.

4. **Verify everything the guide lists** before any push. On memory-constrained
   machines, wrap heavy builds in `systemd-run --user --scope -p MemoryMax=<n>G`
   with reduced `-j` (not `ulimit -v`, which SIGABRTs rustc).

5. **Ship.** One commit (`rust: update reth to <version>`) describing the pin
   move, each shared dependency sync, and every API adaptation. Run the AI review
   (code + security) before pushing; PR to `develop` with verification results
   spelled out.

6. **Compound.** If this bump taught you something the guide doesn't cover, fold
   it into `rust/UPDATING-RETH.md` in the same PR.
