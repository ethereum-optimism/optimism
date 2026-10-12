import KonaSp1.Code.Funs

open Aeneas Aeneas.Std Result ControlFlow WP

namespace KonaSp1
open _root_.KonaSp1.super_root

theorem TimestampSpan.validate_spec (s : TimestampSpan) :
    s.validate ⦃ r => r = .Ok () ↔ s.start.val ≤ s.end.val ⦄ := by
  unfold TimestampSpan.validate
  split <;> simp_all

theorem ensure_optimistic_blocks_match_chain_ids_loop_spec
    (bs : Slice SuperOptimisticBlock) (ids : Slice (ruint.Uint 256#usize 4#usize))
    (i0 : Nat) (hlen : bs.val.length = ids.val.length) (hi0 : i0 ≤ bs.val.length) :
    ensure_optimistic_blocks_match_chain_ids_loop ⟨⟨bs, i0⟩, ⟨ids, i0⟩⟩ ⦃ r =>
      r = .Ok () → ∀ j, i0 ≤ j → j < bs.val.length →
        (bs.val[j]?).map (·.chain_id) = ids.val[j]? ⦄ := by
  unfold ensure_optimistic_blocks_match_chain_ids_loop
  apply loop.spec_decr_nat
    (measure := fun z => bs.val.length - z.fst.i)
    (inv := fun z => z.fst.slice = bs ∧ z.snd.slice = ids ∧ z.fst.i = z.snd.i ∧
      i0 ≤ z.fst.i ∧ z.fst.i ≤ bs.val.length ∧
      ∀ j, i0 ≤ j → j < z.fst.i → (bs.val[j]?).map (·.chain_id) = ids.val[j]?)
  · intro z ⟨h1, h2, h3, hk0, hk, hpre⟩
    obtain ⟨⟨s1, k⟩, ⟨s2, k'⟩⟩ := z
    simp only at h1 h2 h3 hk0 hk hpre
    subst h1 h2 h3
    unfold ensure_optimistic_blocks_match_chain_ids_loop.body
    simp only [core.iter.adapters.zip.Zip.Insts.CoreIterTraitsIteratorIteratorPair.next,
      core.slice.iter.IteratorSliceIter.next, core.cmp.PartialEq.ne.trait_default,
      core.cmp.PartialEq.ne.default, ruint.Uint.Insts.CoreCmpPartialEqUint.eq, Slice.len_val]
    by_cases hk' : k < s1.val.length
    · have hk2 : k < s2.val.length := by omega
      simp [hk', hk2]
      split_ifs with h
      · simp only [spec_ok]
        refine ⟨trivial, trivial, trivial, by omega, by omega, ?_, by omega⟩
        intro j hj hj'
        rcases Nat.lt_succ_iff_lt_or_eq.mp hj' with hj' | rfl
        · exact hpre j hj hj'
        · simp [List.getElem?_eq_getElem hk', List.getElem?_eq_getElem hk2]
          exact h
      · simp
    · simp [hk']
      intro j hj hj'
      exact hpre j hj (by omega)
  · refine ⟨rfl, rfl, rfl, le_refl _, hi0, ?_⟩
    intro j hj hj'; simp at hj'; omega

theorem ensure_optimistic_blocks_match_chain_ids_spec
    (bs : Slice SuperOptimisticBlock) (ids : Slice (ruint.Uint 256#usize 4#usize)) :
    ensure_optimistic_blocks_match_chain_ids bs ids ⦃ r =>
      r = .Ok () → bs.val.map (·.chain_id) = ids.val ⦄ := by
  unfold ensure_optimistic_blocks_match_chain_ids
  simp only [core.slice.Slice.iter, core.iter.traits.iterator.Iterator.zip.trait_default,
    core.iter.traits.iterator.Iterator.zip.default,
    SharedSlice.Insts.CoreIterTraitsCollectIntoIteratorSharedIter.into_iter]
  split_ifs with h
  · simp
  · have hlen : bs.val.length = ids.val.length := by
      simp only [bne_iff_ne, ne_eq, Decidable.not_not] at h
      have := congrArg UScalar.val h; simpa using this
    simp only [Std.bind_ok, Std.bind_tc_ok]
    apply spec_mono (ensure_optimistic_blocks_match_chain_ids_loop_spec bs ids 0 hlen (Nat.zero_le _))
    intro r hr hok
    apply List.ext_getElem? 
    intro j
    by_cases hj : j < bs.val.length
    · simpa [List.getElem?_map] using hr hok j (Nat.zero_le _) hj
    · rw [List.getElem?_eq_none (by simpa using hj), List.getElem?_eq_none (by omega)]

end KonaSp1
