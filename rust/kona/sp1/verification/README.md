# kona-sp1 super-aggregation verification

Lean proofs about the super-aggregation checks in
`crates/client/src/super_root.rs` (`SuperAggregationInputs::validate`), extracted with
[Charon](https://github.com/AeneasVerif/charon) and
[Aeneas](https://github.com/AeneasVerif/aeneas). CI does not build this project yet.

## Main result

`KonaSp1.aggregation_sound` (`KonaSp1/Proofs/Main.lean`): if `validate` returns `Ok`, the
claim is below `u64::MAX`, and every range and consolidation child output is sound, then:

- the starting root hash is the hash of the starting super-root proof;
- a chain of super roots runs from the starting root (at the starting timestamp) to the root
  claim (at `l2_sequence_number`);
- every timestamp in between has one range-derived block per chain of the starting super
  root, each derived from the previous super root with the game's L1 head, and one
  consolidation step from the previous super root to the next with those blocks.

The proofs hold for 32-bit and 64-bit `usize`.

## Trust base

- **Child outputs.** `RangeSound` and `ConsSound` are hypotheses. They stand for:
  - SP1 recursive verification against `SUPER_RANGE_VKEY`;
  - collision resistance of the public-values digest. Range and consolidation outputs share
    the vkey and differ only in the bincode enum tag;
  - the super-range guest committing only outputs that satisfy `RangeStep` and
    `ConsolidateStep`.
- **Not extracted.** `aggregate` in `programs/super-aggregation/src/main.rs` verifies one
  child proof per output. SP1, the on-chain verifier, and the field mapping of
  `zk_dispute_game_public_values` (an axiom) are trusted too.
- **Claim bound.** `validate` accepts a claim of `u64::MAX` whose coverage ends at
  `u64::MAX - 1` (`saturating_add`). The theorem excludes that case with its
  `l2_sequence_number < u64::MAX` hypothesis. The case needs child proofs near timestamp
  2^64, so it is not reachable in practice.
- **Opaque hash.** `hash_super_root_proof` (keccak256 of the fixed-width super-root encoding)
  is an axiom. The proof does not see the chain-list checks of `SuperRootProof::validate`
  inside it.
- **Translation.** Charon and Aeneas, including the Aeneas library models (`Enumerate`,
  `RangeInclusive`, slice `get`, `Vec` equality, `collect`).
- **Hand-written models** in `KonaSp1/Code/TypesExternal.lean` and `FunsExternal.lean`. They
  follow the `rust/` lockfile (ruint 1.20.0); the guest locks ruint 1.17.2, whose modelled
  functions behave the same. `#print axioms` on a theorem lists every unmodelled function it
  uses; `aggregation_sound` uses only `super_root.hash_super_root_proof`.
- **Arithmetic.** The extraction runs with overflow checks off, like the release guest.
- **Mathlib oleans** from `lake exe cache get`. Build Mathlib from source to avoid them.

## Editing `super_root.rs`

Aeneas rejects a `return` or `?` inside a loop when a `?` precedes the loop in the same
function, and a `return` inside a nested loop. The verified functions keep such loops in
their own helpers; keep that shape. After any edit to `super_root.rs`, rerun
`scripts/extract.sh`: the generated files record source line numbers.

## Regenerate and check

```sh
CHARON=... AENEAS=... scripts/extract.sh   # pinned commits are in the script
lake exe cache get && lake build
```

`KonaSp1/Code/Funs.lean` and `Types.lean` are generated; do not edit them by hand.
