# Investigating fault dispute games

How to investigate an OP Stack fault dispute game when something looks wrong — a
challenger **disagreeing** with a proposal or another challenger, **performing a
lot of moves**, **contradicting itself** (posting both roots / attacking its own
claims), or when you need to know whether a proposal is **valid** and who will win
the bonds. The goal is a conclusion about *which side is correct and why*, grounded
in recomputed chain data.

This is **read-only**: investigate and explain; never `move`, `resolve`,
`resolveClaim`, `create`, fund a signer, or restart/stop a challenger.

## What you need

- **L1 EL RPC** (the games live on L1).
- **An L2 EL RPC** for the chain — to recompute output roots from block headers.
- **The chain's op-node rollup RPC** (ideal) — `optimism_outputAtBlock`,
  `optimism_safeHeadAtL1Block`, `optimism_syncStatus`. Often cluster-internal.
- `op-challenger` built locally: `go build -o /tmp/op-challenger ./op-challenger/cmd`.

## 1. Identify the actors

Addresses alone don't tell you who is who, and a single logical actor may use a
**key pool** (several EOAs). In a fault game *anyone* can attack or defend *any*
claim — the chess-clock parity is by depth, not by address. Resolve roles from
deployment/config (e.g. the chain's proposer and challenger addresses, and the
`DisputeGameFactoryProxy` from the superchain-registry).

## 2. Enumerate games and claims

```bash
op-challenger list-games  --game-factory-address <factory> --l1-eth-rpc <l1> --format json
op-challenger list-claims --game-address <game>            --l1-eth-rpc <l1> --format json
```

Use `--format json` for reliable parsing (full values, exact bonds, structured
resolution fields). In the text view the `Value` column is `Hash.TerminalString()`
(first+last 3 bytes), so the *same* short hash repeating across positions means the
*same* output root (clamping — see below).

## 3. Establish the canonical output root (ground truth)

Prefer the existing tooling:

```bash
go run ./op-chain-ops/cmd/check-output-root --help   # output-root games
go run ./op-chain-ops/cmd/check-super-root  --help    # super-root games
```

If you only have a public EL RPC (no archive / no `eth_getProof`), recompute from
the **block header** — on Isthmus the header `withdrawalsRoot` **equals** the
L2ToL1MessagePasser storage root:

```
outputRoot = keccak256( version(32×0) ‖ stateRoot ‖ withdrawalsRoot ‖ blockHash )
```

i.e. `eth.OutputRoot(&eth.OutputV0{StateRoot, MessagePasserStorageRoot: header.withdrawalsRoot, BlockHash})`.
Needs only `eth_getBlockByNumber`, no archive node.

## 4. Classify every claim (correct vs invalid)

Map a claim's position to its L2 block, then compare its value to canonical:

- `traceIdx = (idxAtDepth+1)·2^(splitDepth−depth) − 1` (depth ≤ split depth).
- attack child `= t − 2^(splitDepth−d−1)`; defend child `= t + 2^(splitDepth−d−1)`.
- `block = rangeStart + traceIdx + 1`, **clamped** at the claimed L2 block.
- Beyond the chain tip every position clamps to the final output root (why one hash
  repeats). A claim is **invalid** iff its value ≠ canonical(block).

## 5. Diagnose the responsible op-node

If two parties disagree, at least one node is wrong. Compare nodes with the bundled
scripts (`op-challenger/scripts/`, chain-agnostic, `curl` + `python3`):

- **`game-proposal-outputs.sh <rollup-rpc> <l1-rpc> --factory <addr>`** — per game,
  prints the node's `optimism_outputAtBlock` *and* `optimism_safeHeadAtL1Block` at the
  game's `l1Head`. **Identical output roots but a safe head BELOW the proposed block**
  is the fingerprint of an **incomplete safe-head DB / lagging node**: the challenger
  clamps the max valid L2 block to its safe head and disputes everything beyond it.
- **`check-game-block-hashes.sh <node-rpc> <ref-rpc> <blocks…>`** — confirms a node is
  on canonical history for the proposal blocks (a mismatch = real divergence).

Behind a **load balancer, check each backend individually** — one bad backend
produces intermittent, contradictory behavior (a single challenger emitting *both*
roots and attacking its own claims). A clean cross-check: every claim's value should
be either the canonical root **or** the single clamped value of the bad node — any
third value implies a different fault.

## 6. Why some invalid claims are intentionally left uncountered

A correct challenger does **not** counter every invalid claim. Per
`op-challenger/game/fault/solver/solver.go` `shouldCounter`, it counters a dishonest
claim only when:

- the claim's **parent is honest** (it attacks our line), **or**
- the parent was itself countered by us **and** the claim is at/left of our counter.

It **ignores** dishonest claims to the **right** of its leftmost honest counter and
**all descendants of an ignored claim** ("do not respond to a claim countering a
claim the honest actor ignored") — countering them is unnecessary (they can't change
resolution) and risks **prestate/agreement poisoning** (the "freeloader" solver
tests). So uncountered invalid claims are usually *correct behavior*, not a gap.

## 7. Bond outcome

`packages/contracts-bedrock/src/dispute/FaultDisputeGame.sol` `resolveClaim`: a
claim's bond goes to its **claimant** if **uncountered**, else to the claimant of the
**leftmost correctly-positioned uncountered child**. Therefore:

- invalid claims an honest actor countered → bond to the counterer;
- invalid claims left **uncountered** (§6) → bond **returns to the claimant**;
- valid claims → bond returns to the claimant.

Read bonds straight from `claimData[i].bond`; read `l1Head()` / `l2BlockNumber()` /
`status()` from the game. Distribution is final once all challengers are up to date
(modulo on-chain resolution after the chess clocks expire; refund mode returns bonds
to original posters). Note **resolution** (status flips, clocks expired) precedes
**finalization** (an additional air-gap before credits can be withdrawn).

## Common root causes

- **Incomplete safe-head DB / lagging op-node behind a load balancer** — the
  challenger clamps to its safe head and disputes valid proposals; LB flapping makes
  one challenger emit *both* roots and attack its own claims (looks like a bug, isn't).
- **Genuinely diverged node** (wrong chain) — block hashes mismatch (§5).
- **Misconfiguration** — wrong rollup config / L1 / game type.

## Reading the signals

- Same `outputRoot`, different `safeHead` across nodes ⇒ chain state fine, one node's
  safe-head DB incomplete ⇒ it clamps and disputes valid proposals.
- A single challenger posting two roots / attacking its own claims ⇒ often one
  challenger behind a load balancer with one bad backend, **not** a second bug.
- A challenger making many moves to counter a faulty challenger is **by design**;
  bonds escalate against the loser.
