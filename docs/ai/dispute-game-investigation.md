# Investigating fault dispute games

Read-only playbook for when a dispute game looks wrong — a challenger disagreeing,
doing lots of moves, contradicting itself, or when you need to know whether a
proposal is valid and who wins the bonds. Investigate and explain; never `move`,
`resolve`, `resolveClaim`, `create`, fund a signer, or restart/stop a challenger.

## Inputs

- L1 EL RPC (games live on L1) and an L2 EL RPC for the chain. The chain's op-node
  rollup RPC is useful but optional.
- `op-challenger` built locally: `go build -o /tmp/op-challenger ./op-challenger/cmd`.

## Steps

1. **Identify the actors.** Resolve proposer / challenger(s) / our node and the
   `DisputeGameFactoryProxy` from deployment config + the superchain-registry. A
   signer may be a key pool, and anyone can move on any claim — never infer "side"
   from an address; chess-clock parity is by depth.

2. **Enumerate.** `op-challenger list-games --game-factory-address <f> --l1-eth-rpc <l1> --format json`
   and `op-challenger list-claims --game-address <g> --l1-eth-rpc <l1> --format json`.
   (In text, `Value` is `Hash.TerminalString()` — a repeated short hash means the same
   output root, from clamping.)

3. **Canonical output root (ground truth).** Use the EL-only tools — they need just an
   L2 EL RPC and work against public nodes:
   - output-root games: `go run ./op-chain-ops/cmd/check-output-root --l2-eth-rpc <l2> --block-num <n>`
   - super-root games: `go run ./op-chain-ops/cmd/check-super-root --rpc-endpoints <l2…>`

4. **Classify each claim correct-vs-invalid.** Map position → L2 block:
   `traceIdx = (idxAtDepth+1)·2^(splitDepth−depth) − 1`; attack child `t − 2^(splitDepth−d−1)`,
   defend child `t + 2^(splitDepth−d−1)`; `block = rangeStart + traceIdx + 1`. The honest
   value at a position is the canonical output root at that block **clamped to the safe head
   as of the game's `l1Head`** (`optimism_safeHeadAtL1Block(l1Head)`), not the chain tip —
   positions beyond that safe head clamp to its output root (why a hash repeats). A claim is
   invalid iff its value differs. Crucially this means a claim/proposal can carry a genuinely
   canonical output root and still be **invalid** if its block was not yet safe at the game's
   `l1Head` (e.g. proposing ahead of the safe head) — so always clamp to the safe head, not
   just to the claimed block.

5. **Diagnose the responsible op-node** (`op-challenger/scripts/`, chain-agnostic,
   `curl` + `python3`):
   - `game-proposal-outputs.sh <rollup-rpc> <l1-rpc> --factory <f>` — per game, the node's
     `optimism_outputAtBlock` and `optimism_safeHeadAtL1Block` at the game's `l1Head`.
     Identical output roots but a **safe head below the proposed block** = an incomplete
     safe-head DB / lagging node: the challenger clamps to its safe head and disputes
     everything beyond it.
   - `check-game-block-hashes.sh <node-rpc> <ref-rpc> <blocks…>` — block-hash cross-check
     (a mismatch = real divergence).

   Behind a load balancer, check each backend separately — one bad backend makes a
   single challenger emit *both* roots and attack its own claims. Clean test: every claim
   value should be either canonical or the bad node's single clamped value; anything else
   is a different fault.

6. **Uncountered invalid claims are usually correct.** `op-challenger/game/fault/solver/solver.go`
   `shouldCounter` counters a dishonest claim only when its parent is honest, or the
   parent was countered by us and the claim is at/left of our counter. It ignores claims
   to the right of our leftmost counter and all descendants of an ignored claim — avoids
   wasted bonds and prestate/agreement poisoning.

7. **Bond outcome.** `packages/contracts-bedrock/src/dispute/FaultDisputeGame.sol`
   `resolveClaim`: a claim's bond goes to its claimant if uncountered, else to the
   leftmost correctly-positioned uncountered child's claimant. So invalid-but-countered →
   counterer; invalid-but-ignored → back to the claimant; valid → back to the claimant.
   Read bonds from `claimData[i].bond`. Resolution (clocks expired) precedes finalization
   (an extra air-gap before credits can be withdrawn).

## Common causes

- Incomplete safe-head DB / lagging op-node behind a load balancer — clamps and disputes
  valid proposals; LB flapping makes one challenger emit both roots and self-contradict
  (looks like a bug, isn't).
- Diverged node (wrong chain — block hashes mismatch), or misconfig (wrong rollup config /
  L1 / game type).
- Many moves to counter a faulty challenger is by design; bonds escalate against the loser.
