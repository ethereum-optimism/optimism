# kona-sp1 super-aggregation verification

Lean proofs about the super-aggregation checks in
`crates/client/src/super_root.rs` (`SuperAggregationInputs::validate`), extracted with
[Charon](https://github.com/AeneasVerif/charon) and
[Aeneas](https://github.com/AeneasVerif/aeneas). CI does not build this project yet.

## Main result

`KonaSp1.aggregation_sound` (`KonaSp1/Proofs/Main.lean`): if `validate` returns `Ok`, the
starting super root has at least one chain, and every range and consolidation child output is
sound, then:

- the starting root hash is the hash of the starting super-root proof;
- a chain of super roots runs from the starting root (at the starting timestamp) to the root
  claim (at `l2_sequence_number`);
- every timestamp in between has one range-derived block per chain of the starting super
  root, each derived from the previous super root with the game's L1 head, and one
  consolidation step from the previous super root to the next with those blocks.

The conclusion does not depend on how the range outputs split the work: one output can cover
several timestamps, or several outputs can cover the chains of one timestamp. The proofs hold
for 32-bit and 64-bit `usize`.

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
- **Chain list.** `SuperRootProof::validate` rejects an empty chain list, but it runs inside
  the opaque hash, so the theorem takes a non-empty list as a hypothesis. With no chains the
  range checks are empty, and the consolidation checks accept coverage that ends at
  `u64::MAX - 1` for a claim of `u64::MAX` (`saturating_add`).
- **Opaque hash.** `hash_super_root_proof` (keccak256 of the fixed-width super-root encoding)
  is an axiom.
- **Translation.** Charon and Aeneas, including the Aeneas library models (`Range`,
  `RangeInclusive`, `Enumerate`, `Zip`, slice iteration, indexing and `get`, `Option::map`,
  the `IntoIterator` identity for iterators, `collect`).
- **Hand-written models** in `KonaSp1/Code/TypesExternal.lean` and `FunsExternal.lean`, among
  them `flat_map` (with the unspecialized `Fuse`; see the comment on its `next`), `Option`
  equality, `Option::copied`, slice `first` and `last`, and `B256` clone. The models follow
  the `rust/` lockfile (ruint 1.20.0); the guest locks ruint 1.17.2, whose modelled functions
  behave the same. `#print axioms` on a theorem lists every unmodelled function it uses;
  `aggregation_sound` uses only `super_root.hash_super_root_proof`.
- **Arithmetic.** The extraction runs with overflow checks off, like the release guest.
- **Mathlib oleans** from `lake exe cache get`. Build Mathlib from source to avoid them.

## Editing `super_root.rs`

Aeneas rejects:

- a `return` or `?` inside a loop when a `?` precedes the loop in the same function;
- a `return` inside a nested loop;
- a closure that returns a borrow of the data it captures;
- a loop that mutates a value that holds borrows.

The verified functions keep such loops in their own helpers, and the `expected` iterator of
`validate_range_outputs` owns its state and yields chain indices. Keep that shape. After any
edit to `super_root.rs`, rerun `scripts/extract.sh`: the generated files record source line
numbers.

## Regenerate and check

```sh
CHARON=... AENEAS=... scripts/extract.sh   # pinned commits are in the script
lake exe cache get && lake build
```

`KonaSp1/Code/Funs.lean` and `Types.lean` are generated; do not edit them by hand.
