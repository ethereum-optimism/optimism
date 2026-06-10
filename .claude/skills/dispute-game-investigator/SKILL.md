---
name: dispute-game-investigator
description: Investigate an OP Stack fault dispute game during (or after) an incident — answer the user's questions and diagnose: why a challenger is disagreeing, doing a lot of moves, or contradicting itself; whether a proposal is valid; which op-node is at fault; and (after the fact) who wins the bonds. Read-only.
---

# Dispute Game Investigator

## When to Use

- **During an incident**, to answer questions and diagnose: a challenger disagreeing
  with a proposal or another challenger, performing many moves, or contradicting itself;
  whether the in-progress proposals are valid; which op-node is responsible.
- **After the fact**, for specifics like the bond outcome.

## How to use it

Answer the user's actual question using only the **relevant** parts of the guide — you do
not need to run every step. Triage first (which game(s)? whose node? valid or not?) and pull
in only what bears on the question; some parts are situational (e.g. the bond outcome in §7 is
usually a post-incident question, not live triage). For a full end-to-end written analysis,
use the **`dispute-game-investigator` agent** instead.

Read-only: investigate and explain; never make moves, resolve, fund, or restart a challenger.

## Guide

@docs/ai/dispute-game-investigation.md

## Tooling

- `op-challenger list-games`/`list-claims --format json` — enumerate games/claims.
- `op-chain-ops/cmd/check-output-root` and `check-super-root` — canonical output root (EL-only, work against public nodes).
- `op-challenger/scripts/game-proposal-outputs.sh` — per-game `outputAtBlock` +
  `safeHeadAtL1Block` from a node (spots an incomplete safe-head DB).
- `op-challenger/scripts/check-game-block-hashes.sh` — block-hash cross-check, node vs reference.
