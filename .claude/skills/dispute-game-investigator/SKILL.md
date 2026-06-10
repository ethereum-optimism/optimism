---
name: dispute-game-investigator
description: Investigate an OP Stack fault dispute game — why a challenger is disagreeing, doing a lot of moves, or contradicting itself; whether a proposal is valid; which node is at fault; and who wins the bonds. Read-only.
---

# Dispute Game Investigator

## When to Use

- A challenger is disagreeing with a proposal or another challenger, performing many
  moves, or contradicting itself (posting both roots / attacking its own claims).
- You need to know whether the in-progress proposals for a chain are valid.
- You need to diagnose which op-node is responsible, or work out the bond outcome.

## Guide

@docs/ai/dispute-game-investigation.md

## Tooling

- `op-challenger list-games`/`list-claims --format json` — enumerate games/claims.
- `op-chain-ops/cmd/check-output-root` and `check-super-root` — canonical output root.
- `op-challenger/scripts/game-proposal-outputs.sh` — per-game `outputAtBlock` +
  `safeHeadAtL1Block` from a node (spots an incomplete safe-head DB).
- `op-challenger/scripts/check-game-block-hashes.sh` — block-hash cross-check, node vs
  reference.

Read-only: investigate and explain; never make moves, resolve, fund, or restart a
challenger.
