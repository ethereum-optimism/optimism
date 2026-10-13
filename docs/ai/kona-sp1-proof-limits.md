# Kona-SP1 proof limits and completeness review

This guide gives the resource limits of kona-sp1 proof generation and a method to find inputs that
exceed them. It pairs with the
[`kona-sp1-cost-reviewer`](../../.claude/agents/kona-sp1-cost-reviewer.md) agent. Read it before
you review or change:

- A kona-sp1 guest program (`rust/kona/sp1/programs/`) or the guest crates under
  `rust/kona/sp1/crates/`: `client` (witness and oracle) and `ethereum/client` (range and
  consolidation).
- Shared kona crates that a guest runs: `kona-proof`, `kona-proof-interop`, `kona-interop`,
  `kona-driver`, `kona-derive`, `kona-protocol`, `kona-executor`, `kona-mpt`, `kona-preimage`, and
  the configuration crates `kona-genesis`, `kona-hardforks`, `kona-registry`. To check the list, run
  `cargo tree -e normal --target all -p kona-sp1-super-range` in `rust/kona/sp1/programs` and read
  the `kona-` crates.
- The proposer split logic (`RangeSplitCount`).

PGU is the SP1 unit of proving gas. The prover network bills and limits requests in PGU.

## Why completeness is a security property

A guest can be sound and still fail to prove an honest claim. In `ZKDisputeGame`, a challenged
proposal without a proof at its prove deadline resolves as `CHALLENGER_WINS`, and the challenger
gets all bonds. A child game of that game also resolves as `CHALLENGER_WINS`. So an input that
makes an honest proof too large lets an attacker remove a valid proposal and take the proposer
bond.

Report such an issue as a completeness finding, with liveness and fund-loss impact. Do not report
it as a performance note.

## Limits

| Limit | Value | Source | Effect when exceeded |
|---|---|---|---|
| Guest input region | 16 GiB | `INPUT_REGION_SIZE` default (`1 << 34`), `sp1-zkvm` build script | `read_vec_raw` panics: "Input region overflowed." All stdin uses this region, including the witness. |
| Guest heap | Up to 112 GiB: `MAX_MEMORY` (`1 << 37`) minus the input region | `sp1-zkvm` build script and embedded allocator | The allocator fails. Decoded copies of the witness use the heap. |
| Executor memory | 24 GiB | `DEFAULT_MEMORY_LIMIT` in `sp1-core-executor` (`SP1CoreOpts::memory_limit`, env `MEMORY_LIMIT`). The `zkvm-canary` default is the same value. | Local execution and the canary stop. The prover-network limit is not recorded here. |
| Range and consolidation request | 2 × 10¹¹ PGU, 10¹² cycles | Proposer `DEFAULT_RANGE_GAS_LIMIT`, `DEFAULT_PROOF_CYCLE_LIMIT` in `config.rs` | The prover stops. The request gives no proof. |
| Aggregation request | 10⁹ PGU, 10¹² cycles | Proposer `DEFAULT_AGG_GAS_LIMIT`, `DEFAULT_PROOF_CYCLE_LIMIT` | The prover stops. The request gives no proof. |
| Split | At most 128 chunks per game (default 16). Each chunk is one range request and one consolidation request. At 128 chunks, the largest chunk holds ceil(span / 128) timestamps (29 for a 3,600 s game). | `RangeSplitCount::MAX`, `RangeSplitCount::default`, `RangeSplitCount::split` | The proposer cannot split the work of one timestamp. |

The SP1 values are the SP1 6.8.1 defaults: `sp1-zkvm` in `rust/kona/sp1/programs/Cargo.lock`,
`sp1-core-executor` in `rust/Cargo.lock`. Nothing in this repository sets
`SP1_ZKVM_INPUT_REGION_SIZE` or `SP1_ZKVM_MAX_MEMORY`. Check the values again after an SP1 upgrade.

The proposer limits are configuration defaults, and `OP_ZK_PROPOSER_*` env vars override them. The
input region and the one-timestamp floor are fixed for a deployed guest.

The table has no time limits. Proof time depends on prover throughput and the prover-network
market, so a review cannot estimate it. Measure the cost in PGU, witness bytes, and memory.

## Unit costs

Measured with the SP1 6.8.1 executor (`calculate_gas(true)`). Check the values again after an SP1
upgrade.

| Operation | Cost |
|---|---|
| One keccak256 call | About 2,400 PGU |
| Receipts trie walk and decode | 22–49 PGU per witness byte. A full 30M-gas block: 81.2M PGU for a 3.66 MB witness (3,480 logs of 1 KB), 88.9M PGU for 1.80 MB (75,000 empty logs). |
| One EIP-2935 header lookup step | 330K PGU and 7.2 KB of witness. The first lookup to the far end of a 7-day window at 1 s blocks takes about 74 steps. |
| One Merkle opening | 46K PGU (depth 12) to 63K PGU (depth 17) |
| In-guest `verify_sp1_proof` | 84K PGU per call, plus about 1.1M PGU once. Each deferred proof also adds about 1.1M PGU-equivalent of recursion work. The PGU count does not include it. |

Estimate first: multiply the unit costs by the worst-case input counts. Measure with the SP1
executor only when the estimate is within 10× of a limit. The `kona-sp1-super-range-executor`
binary (`just build-super-range-executor` in `rust/kona/sp1`) runs the super-range guest in execute
mode.

## Triage

Do this step first. Continue only if the diff adds or changes one of these items on a guest path:

- A witness or oracle read.
- A loop over witness data.
- A cache or its key.
- A data structure that the guest decodes.
- The proof split.
- The guest cost of EVM execution: precompile routing
  (`rust/kona/sp1/crates/client/src/precompiles/`) or the SP1 crypto patches (`[patch.crates-io]`
  in `rust/kona/sp1/programs/Cargo.toml`).

Otherwise, report `No guest cost change` and one sentence of reason, and stop. Most diffs stop
here.

## Review method

For each guest path that passes triage:

1. Write the guest cost of the path as a function of its inputs: PGU, witness bytes, and peak
   guest memory.
2. For each input, name who controls it and what it costs them. An unprivileged L2 user pays L2
   gas and DA. The sequencer, L1, and the dependency set are separate cases.
3. Find the largest value of each input under protocol limits: the block gas limit, the DA
   footprint limit, the message expiry window (at most 7 days), the number of chains, and the
   number of timestamps in one proof.
4. Compute the worst case for one proof request at the smallest split that the proposer can make.
   Compare it with each row of the limits table.
5. Report the amplification: the worst-case PGU per gas that the attacker pays. If the author
   gives a baseline from the canary (`kona_zkvm_canary_report_pgu{mode="range"}` divided by
   `kona_zkvm_canary_executed_l2_gas`), compare with it.

Look for these patterns:

- **Cost from data that the attacker references but does not pay for.** The guest loads a whole
  object (all receipts of a block, a trie, a block body) to read one item that a user names. The
  user pays per reference. The prover pays per byte of the object.
- **Multipliers that a per-block view hides.** No cache across timestamps in one proof. One check
  per block-time step. Replacement rounds. Several chains at one timestamp.
- **Work below the split floor.** The proposer cannot split cost that grows inside one timestamp.
- **Witness size.** Each preimage that the guest reads goes into stdin. Check the witness bytes
  as well as the PGU.

Do not dismiss a candidate for these reasons:

- The result is sound. Completeness is a separate property.
- A configuration limit can go higher. Check the fixed limits first.
- Typical traffic is small. The attacker chooses the input.
- The attacker must pay for the referenced data. Check whether any data in the window, from any
  user, is a valid reference.

## Known case: interop receipts in consolidation

The consolidation checks each executing message against its initiating block.
`MessageGraph::check_single_dependency` (`kona-interop`) calls
`OracleInteropProvider::receipts_by_number` (`kona-proof-interop`). Its `derive_receipts` walks
the receipts trie and decodes every receipt of the block. `MessageGraph::resolve` sorts the
messages by initiating block, so each distinct initiating block loads once per resolve. Any log in
the expiry window is a valid target, so the attacker references many large blocks that exist.

- One full 30M-gas initiating block costs 81–89M PGU to load.
- A 30M-gas executing block holds 2,011–3,239 messages (DA-bound or gas-bound).
- One chain at one timestamp costs 1.8–2.9 × 10¹¹ PGU, which is near or above the range request
  limit.
- At 60M gas, the estimated witness for one chain at one timestamp (29–47 GB) exceeds the 16 GiB
  input region.

The planned fix is per-block log-commitment sub-proofs. The consolidation then checks one Merkle
opening per message. Do not report this case again unless the diff makes it worse.

## Output

Start with the triage result. Then give one block for each candidate:

```text
### Short candidate title

Path: Guest program, entry point, and the file and symbol of the costly operation
Input: The input that drives the cost, who controls it, and what it costs them
Worst case: Formula with unit costs and input counts, and the result in PGU, witness bytes, and memory
Limit: The limits-table row, and the worst case as a fraction of it
Amplification: Worst-case PGU per gas that the attacker pays
Evidence: Estimate only, or the executor command and its measured result
Verdict: Completeness finding | No finding, with the margin to the nearest limit
```

Report a completeness finding when the worst case exceeds a limit. If no candidate exceeds a limit,
report `No verified findings.` after the candidate blocks.

## Related

- [fault-proofs.md](fault-proofs.md): Cannon, kona-client, and the dispute games.
- [interop-review.md](interop-review.md): interop message verification.
