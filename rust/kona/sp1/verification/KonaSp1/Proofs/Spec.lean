import KonaSp1.Proofs.Leaf

/-! Pure predicates that mirror the aggregation checks, and the specs that tie each
extracted function to them. Every spec states what an `Ok` result implies. -/

open Aeneas Aeneas.Std Result ControlFlow WP

namespace KonaSp1
open _root_.KonaSp1.super_root

abbrev B256 := alloy_primitives.bits.fixed.FixedBytes 32#usize
abbrev U256 := ruint.Uint 256#usize 4#usize

/-- `U256::from(u64)`. -/
def u256OfU64 (x : Std.U64) : U256 :=
  ⟨x.val, lt_of_lt_of_le x.hBounds (Nat.pow_le_pow_right (by decide) (by decide))⟩

/-- Number of timestamps in an inclusive span. -/
abbrev spanCount (s : TimestampSpan) : Nat := s.end.val - s.start.val + 1

/-! ## Level 1: leaf helpers -/

theorem TimestampSpan.contains_spec (s : TimestampSpan) (t : Std.U64) :
    s.contains t ⦃ b => b ↔ (s.start.val ≤ t.val ∧ t.val ≤ s.end.val) ⦄ := by
  unfold TimestampSpan.contains
  split <;> simp_all

theorem inclusive_timestamp_count_spec (span : TimestampSpan) (chains : Std.Usize) :
    inclusive_timestamp_count span chains ⦃ r =>
      ∀ v, r = .Ok v → span.start.val ≤ span.end.val ∧ v.val = spanCount span ⦄ := by
  unfold inclusive_timestamp_count
  simp only [lift, core.option.Option.and_then,
    inclusive_timestamp_count.closure.Insts.CoreOpsFunctionFnOnceTupleU64OptionU64.call_once]
  have hsub := U64.checked_sub_bv_spec span.end span.start
  cases hs : U64.checked_sub span.end span.start with
  | none => simp [core.option.Option.ok_or]
  | some z =>
    rw [hs] at hsub
    obtain ⟨hle, hz, -⟩ := hsub
    have hadd := U64.checked_add_bv_spec z 1#u64
    simp only [Std.bind_ok]
    cases ha : U64.checked_add z 1#u64 with
    | none => simp [core.option.Option.ok_or]
    | some w =>
      rw [ha] at hadd
      obtain ⟨-, hw, -⟩ := hadd
      simp only [core.option.Option.ok_or, spec_ok, core.result.Result.Ok.injEq]
      intro v hv
      subst hv
      refine ⟨hle, ?_⟩
      simp only [spanCount]
      scalar_tac

theorem consolidation_timestamp_count_spec (span : TimestampSpan) :
    consolidation_timestamp_count span ⦃ r =>
      ∀ v, r = .Ok v → span.start.val ≤ span.end.val ∧ v.val = spanCount span ⦄ := by
  unfold consolidation_timestamp_count
  apply spec_bind (inclusive_timestamp_count_spec span 0#usize)
  intro r hr
  cases r with
  | Ok v =>
    obtain ⟨hle, hv⟩ := hr v rfl
    simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok,
      Usize.Insts.CoreConvertTryFromU64TryFromIntError.try_from, core.num.tryFromUScalar]
    split_ifs with hm
    · simp only [Std.bind_ok, core.result.Result.map_err, spec_ok, core.result.Result.Ok.injEq]
      intro w hw
      subst hw
      refine ⟨hle, ?_⟩
      rw [UScalar.cast_val_mod_pow_of_inBounds_eq]
      · exact hv
      · have : 0 < 2 ^ UScalarTy.Usize.numBits := by simp
        simp only [UScalar.max] at hm
        omega
    · simp [core.result.Result.map_err,
        consolidation_timestamp_count.closure.Insts.CoreOpsFunctionFnOnceTupleTryFromIntErrorSuperRootError.call_once]
  | Err e =>
    simp [core.result.Result.Insts.CoreOpsTry.branch,
      core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]

theorem slice_next_lt {T : Type} (s : Slice T) (k : Nat) (h : k < s.val.length) :
    core.slice.iter.IteratorSliceIter.next ⟨s, k⟩ = ok (some s.val[k], ⟨s, k + 1⟩) := by
  simp [core.slice.iter.IteratorSliceIter.next, h]; rfl

theorem slice_next_ge {T : Type} (s : Slice T) (k : Nat) (h : ¬ k < s.val.length) :
    core.slice.iter.IteratorSliceIter.next ⟨s, k⟩ = ok (none, ⟨s, k⟩) := by
  simp [core.slice.iter.IteratorSliceIter.next, h]

/-- Per-index property checked by `ensure_consolidation_transitions` (timestamps wrap). -/
def consTransitionOK (span : TimestampSpan) (ids : Slice U256)
    (ts : Slice SuperConsolidationTransition) (j : Nat) : Prop :=
  ∃ tr, ts.val[j]? = some tr ∧ tr.timestamp.val = (span.start.val + j) % 2 ^ 64 ∧
    tr.optimistic_blocks.val.map (·.chain_id) = ids.val

theorem ensure_consolidation_transitions_loop_spec (span : TimestampSpan) (ids : Slice U256)
    (ts : Slice SuperConsolidationTransition) :
    ensure_consolidation_transitions_loop ⟨⟨ts, 0⟩, 0#usize⟩ span ids ⦃ r =>
      r = .Ok () → ∀ j, j < ts.val.length → consTransitionOK span ids ts j ⦄ := by
  unfold ensure_consolidation_transitions_loop
  apply loop.spec_decr_nat
    (measure := fun e => ts.val.length - e.iter.i)
    (inv := fun e => e.iter.slice = ts ∧ e.count.val = e.iter.i ∧ e.iter.i ≤ ts.val.length ∧
      ∀ j, j < e.iter.i → consTransitionOK span ids ts j)
  · intro e ⟨h1, hc, hk, hpre⟩
    obtain ⟨⟨s1, k⟩, c⟩ := e
    simp only at h1 hc hk hpre
    subst h1
    unfold ensure_consolidation_transitions_loop.body
    simp only [core.iter.adapters.enumerate.IteratorEnumerate.next]
    by_cases hk' : k < s1.val.length
    · have hmax : c.val + (1#usize).val ≤ Usize.max := by
        have := Slice.length_ineq s1
        scalar_tac
      rw [slice_next_lt s1 k hk']
      simp only [Std.bind_tc_ok]
      simp only [Std.bind_tc_eq, Std.bind_assoc, Std.bind_ok]
      apply spec_bind (Usize.add_spec hmax)
      intro c' hc'
      simp only [lift, Std.bind_ok]
      simp only [uncurry_apply_pair]
      have hcast : (UScalar.cast UScalarTy.U64 c).val = k := by
        rw [UScalar.cast_val_mod_pow_greater_numBits_eq]
        · exact hc
        · simp only [UScalarTy.Usize_numBits_eq, UScalarTy.U64_numBits_eq]
          cases System.Platform.numBits_eq <;> simp_all
      split_ifs with hts
      · simp
      · simp only [bne_iff_ne, ne_eq, Decidable.not_not] at hts
        apply spec_bind (ensure_optimistic_blocks_match_chain_ids_spec _ ids)
        intro r hr
        cases r with
        | Ok u =>
          cases u
          simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok, spec_ok]
          refine ⟨⟨trivial, ?_, by omega, ?_⟩, by omega⟩
          · simp at hc'; omega
          · intro j hj
            rcases Nat.lt_succ_iff_lt_or_eq.mp hj with hj | hj
            · exact hpre j hj
            · rw [hj]
              refine ⟨s1.val[k], by simp [List.getElem?_eq_getElem hk'], ?_, ?_⟩
              · rw [hts, U64.wrapping_add_val_eq, hcast]; simp [UScalar.size_def]
              · simpa [alloc.vec.Vec.deref] using hr rfl
        | Err e =>
          simp [core.result.Result.Insts.CoreOpsTry.branch,
            core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
    · rw [slice_next_ge s1 k hk']
      simp
      intro j hj
      exact hpre j (by omega)
  · refine ⟨rfl, rfl, Nat.zero_le _, ?_⟩
    intro j hj; simp at hj

theorem ensure_consolidation_transition_coverage_spec (span : TimestampSpan)
    (ids : Slice U256) (ts : Slice SuperConsolidationTransition) :
    ensure_consolidation_transition_coverage span ids ts ⦃ r =>
      r = .Ok () →
        span.start.val ≤ span.end.val ∧ ts.val.length = spanCount span ∧
        ∀ j (hj : j < ts.val.length),
          (ts.val[j]).timestamp.val = span.start.val + j ∧
          (ts.val[j]).optimistic_blocks.val.map (·.chain_id) = ids.val ⦄ := by
  unfold ensure_consolidation_transition_coverage
  apply spec_bind (consolidation_timestamp_count_spec span)
  intro r hr
  cases r with
  | Ok v =>
    obtain ⟨hle, hv⟩ := hr v rfl
    simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok]
    split_ifs with h
    · simp
    · simp only [bne_iff_ne, ne_eq, Decidable.not_not] at h
      have hlen : ts.val.length = spanCount span := by
        have := congrArg UScalar.val h
        simp only [Slice.len_val, Slice.length] at this
        omega
      unfold ensure_consolidation_transitions
      simp only [core.slice.Slice.iter, core.iter.traits.iterator.Iterator.enumerate.trait_default,
        core.iter.traits.iterator.Iterator.enumerate.default, Std.bind_ok]
      apply spec_mono (ensure_consolidation_transitions_loop_spec span ids ts)
      intro r hr hok
      refine ⟨hle, hlen, ?_⟩
      intro j hj
      obtain ⟨tr, htr, hts, hblocks⟩ := hr hok j hj
      rw [List.getElem?_eq_getElem hj, Option.some.injEq] at htr
      subst htr
      refine ⟨?_, hblocks⟩
      rw [hts]
      apply Nat.mod_eq_of_lt
      have := span.end.hBounds
      simp only [UScalarTy.U64_numBits_eq] at this
      simp only [spanCount] at hlen
      omega
  | Err e =>
    simp [core.result.Result.Insts.CoreOpsTry.branch,
      core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]

theorem ensure_range_previous_super_root_outputs_spec (span : TimestampSpan)
    (prs : Slice B256) :
    ensure_range_previous_super_root_outputs span prs ⦃ r =>
      r = .Ok () → span.start.val ≤ span.end.val ∧ prs.val.length = spanCount span ⦄ := by
  unfold ensure_range_previous_super_root_outputs
  apply spec_bind (consolidation_timestamp_count_spec span)
  intro r hr
  cases r with
  | Ok v =>
    obtain ⟨hle, hv⟩ := hr v rfl
    simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok]
    split_ifs with h
    · simp
    · simp only [bne_iff_ne, ne_eq, Decidable.not_not] at h
      simp only [spec_ok, forall_const]
      refine ⟨hle, ?_⟩
      have := congrArg UScalar.val h
      simp only [Slice.len_val, Slice.length] at this
      omega
  | Err e =>
    simp [core.result.Result.Insts.CoreOpsTry.branch,
      core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]

/-- What `ensure_range_transition_coverage` establishes on `Ok`: transitions are grouped by
timestamp, and each group lists one transition per chain in `ids` order. -/
def RangeCoverage (span : TimestampSpan) (ids : List U256)
    (ts : List SuperRangeTransition) : Prop :=
  span.start.val ≤ span.end.val ∧ ts.length = spanCount span * ids.length ∧
  ∀ j k, j < spanCount span → (hk : k < ids.length) →
    ∃ tr, ts[j * ids.length + k]? = some tr ∧
      tr.timestamp.val = span.start.val + j ∧ tr.optimistic_block.chain_id = ids[k]

theorem ensure_timestamp_transition_order_loop_spec (t : Std.U64)
    (ids : Slice U256) (ts : Slice SuperRangeTransition)
    (i0 : Nat) (hlen : ts.val.length = ids.val.length) (hi0 : i0 ≤ ids.val.length) :
    ensure_timestamp_transition_order_loop ⟨⟨ids, i0⟩, ⟨ts, i0⟩⟩ t ⦃ r =>
      r = .Ok () → ∀ k, i0 ≤ k → k < ids.val.length →
        ∃ tr, ts.val[k]? = some tr ∧ tr.timestamp = t ∧
          ids.val[k]? = some tr.optimistic_block.chain_id ⦄ := by
  unfold ensure_timestamp_transition_order_loop
  apply loop.spec_decr_nat
    (measure := fun z => ids.val.length - z.fst.i)
    (inv := fun z => z.fst.slice = ids ∧ z.snd.slice = ts ∧ z.fst.i = z.snd.i ∧
      i0 ≤ z.fst.i ∧ z.fst.i ≤ ids.val.length ∧
      ∀ k, i0 ≤ k → k < z.fst.i →
        ∃ tr, ts.val[k]? = some tr ∧ tr.timestamp = t ∧
          ids.val[k]? = some tr.optimistic_block.chain_id)
  · intro z ⟨h1, h2, h3, hk0, hk, hpre⟩
    obtain ⟨⟨s1, k⟩, ⟨s2, k'⟩⟩ := z
    simp only at h1 h2 h3 hk0 hk hpre
    subst h1 h2 h3
    unfold ensure_timestamp_transition_order_loop.body
    simp only [core.iter.adapters.zip.Zip.Insts.CoreIterTraitsIteratorIteratorPair.next,
      core.slice.iter.IteratorSliceIter.next, core.cmp.PartialEq.ne.trait_default,
      core.cmp.PartialEq.ne.default, ruint.Uint.Insts.CoreCmpPartialEqUint.eq, Slice.len_val]
    by_cases hk' : k < s1.val.length
    · have hk2 : k < s2.val.length := by omega
      simp [hk', hk2]
      split_ifs with h h'
      · simp only [spec_ok]
        refine ⟨trivial, trivial, trivial, by omega, by omega, ?_, by omega⟩
        intro j hj hj'
        rcases Nat.lt_succ_iff_lt_or_eq.mp hj' with hj' | rfl
        · exact hpre j hj hj'
        · refine ⟨s2.val[j], List.getElem?_eq_getElem hk2, ?_, ?_⟩
          · exact UScalar.eq_of_val_eq h
          · simp only [List.getElem?_eq_getElem hk', Option.some.injEq]
            exact h'.symm
      · simp
      · simp
    · simp [hk']
      intro j hj hj'
      exact hpre j hj (by omega)
  · refine ⟨rfl, rfl, rfl, le_refl _, hi0, ?_⟩
    intro j hj hj'; simp at hj'; omega

theorem ensure_timestamp_transition_order_spec (t : Std.U64)
    (ids : Slice U256) (ts : Slice SuperRangeTransition)
    (hlen : ts.val.length = ids.val.length) :
    ensure_timestamp_transition_order t ids ts ⦃ r =>
      r = .Ok () → ∀ k, k < ids.val.length →
        ∃ tr, ts.val[k]? = some tr ∧ tr.timestamp = t ∧
          ids.val[k]? = some tr.optimistic_block.chain_id ⦄ := by
  unfold ensure_timestamp_transition_order
  simp only [core.slice.Slice.iter, core.iter.traits.iterator.Iterator.zip.trait_default,
    core.iter.traits.iterator.Iterator.zip.default,
    SharedSlice.Insts.CoreIterTraitsCollectIntoIteratorSharedIter.into_iter, Std.bind_ok,
    Std.bind_tc_ok]
  apply spec_mono (ensure_timestamp_transition_order_loop_spec t ids ts 0 hlen (Nat.zero_le _))
  intro r hr hok k hk
  exact hr hok k (Nat.zero_le _) hk

/-- Timestamps of the range already processed by `ensure_range_transition_order_loop`. -/
def rangeOrderDone (S E : Std.U64) (it : core.ops.range.RangeInclusive Std.U64) : Nat :=
  if it.exhausted then E.val - S.val + 1 else it.start.val - S.val

/-- Loop invariant of `ensure_range_transition_order_loop`. -/
def rangeOrderInv (S E : Std.U64) (ids : Slice U256) (ts : Slice SuperRangeTransition)
    (it : core.ops.range.RangeInclusive Std.U64) (idx : Std.Usize) : Prop :=
  it.end = E ∧ S.val ≤ it.start.val ∧ it.start.val ≤ E.val ∧
  idx.val = rangeOrderDone S E it * ids.val.length ∧
  ∀ j k, j < rangeOrderDone S E it → k < ids.val.length →
    ∃ tr, ts.val[j * ids.val.length + k]? = some tr ∧
      tr.timestamp.val = S.val + j ∧ ids.val[k]? = some tr.optimistic_block.chain_id

theorem ensure_range_transition_order_loop_spec
    (S E : Std.U64) (ids : Slice U256) (ts : Slice SuperRangeTransition)
    (hse : S.val ≤ E.val)
    (hlen : ts.val.length = (E.val - S.val + 1) * ids.val.length)
    (it : core.ops.range.RangeInclusive Std.U64) (idx : Std.Usize)
    (hinv : rangeOrderInv S E ids ts it idx) :
    ensure_range_transition_order_loop it ids ts idx ⦃ r =>
      r = .Ok () → ∀ j k, j < E.val - S.val + 1 → k < ids.val.length →
        ∃ tr, ts.val[j * ids.val.length + k]? = some tr ∧
          tr.timestamp.val = S.val + j ∧ ids.val[k]? = some tr.optimistic_block.chain_id ⦄ := by
  unfold ensure_range_transition_order_loop
  apply loop.spec_decr_nat
    (measure := fun p => if p.1.exhausted then 0 else E.val - p.1.start.val + 1)
    (inv := fun p => rangeOrderInv S E ids ts p.1 p.2)
  · rintro ⟨it, idx⟩ ⟨hend, hS, hE, hidx, hpre⟩
    unfold ensure_range_transition_order_loop.body
    apply spec_bind (core.ops.range.RangeInclusive.Insts.CoreIterTraitsIteratorIterator.next_U64_spec it)
    rintro ⟨o, it'⟩ hnext
    simp only [uncurry'_pair] at hnext
    simp only at hend hS hE hidx hpre ⊢
    by_cases hempty : it.exhausted = true ∨ it.end.val < it.start.val
    · simp only [eq_true hempty, ↓reduceIte] at hnext
      obtain ⟨rfl, h'⟩ := hnext
      rw [h']
      simp
      intro j k hj hk
      have hex : it.exhausted = true := by
        rcases hempty with h | h
        · exact h
        · rw [hend] at h; omega
      exact hpre j k (by simp only [rangeOrderDone, hex, ite_true]; omega) hk
    · simp only [eq_false hempty, ↓reduceIte] at hnext
      have hex : it.exhausted = false := by simp_all
      have hle : it.start.val ≤ E.val := hE
      have ho : o = some it.start := by split_ifs at hnext <;> exact hnext.1
      subst ho
      have hd : rangeOrderDone S E it = it.start.val - S.val := by simp [rangeOrderDone, hex]
      rw [hd] at hidx hpre
      have hdn : (it.start.val - S.val + 1) * ids.val.length ≤ ts.val.length := by
        rw [hlen]; apply Nat.mul_le_mul_right; omega
      have hts_max : ts.val.length ≤ Usize.max := ts.length_ineq
      have hni : (idx.wrapping_add ids.len).val = idx.val + ids.val.length := by
        rw [Usize.wrapping_add_val_eq, Slice.len_val]
        apply Nat.mod_eq_of_lt
        have : Usize.max < UScalar.size .Usize := by
          simp [Usize.max_def, UScalar.size_def, Usize.numBits_def]
        rw [Nat.add_mul] at hdn
        simp only [Slice.length] at *
        omega
      simp only [lift, Std.bind_ok, core.slice.index.Slice.index]
      have h1 : idx ≤ idx.wrapping_add ids.len := by rw [UScalar.le_equiv, hni]; omega
      have h2 : (idx.wrapping_add ids.len).val ≤ ts.length := by
        simp only [Slice.length]; rw [hni, hidx]; rw [Nat.add_mul, Nat.one_mul] at hdn; omega
      apply spec_bind (core.slice.index.SliceIndexRangeUsizeSlice.index.step_spec
        { start := idx, «end» := idx.wrapping_add ids.len } ts h1 h2)
      rintro sl ⟨hs, hslen⟩
      have hslen' : sl.val.length = ids.val.length := by
        simp only [Slice.length] at hslen; rw [hslen, hni]; omega
      apply spec_bind (ensure_timestamp_transition_order_spec it.start ids sl hslen')
      intro r hr
      cases r with
      | Err e =>
        simp [core.result.Result.Insts.CoreOpsTry.branch,
          core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
      | Ok u =>
        cases u
        simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok, spec_ok]
        simp only at hs
        have hext : ∀ j k, j < it.start.val - S.val + 1 → k < ids.val.length →
            ∃ tr, ts.val[j * ids.val.length + k]? = some tr ∧
              tr.timestamp.val = S.val + j ∧ ids.val[k]? = some tr.optimistic_block.chain_id := by
          intro j k hj hk
          rcases Nat.lt_succ_iff_lt_or_eq.mp hj with hj | rfl
          · exact hpre j k hj hk
          · obtain ⟨tr, h1', h2', h3'⟩ := hr rfl k hk
            refine ⟨tr, ?_, ?_, h3'⟩
            · rw [← h1', hs, List.slice, List.getElem?_take_of_lt (by rw [hni]; omega),
                List.getElem?_drop, hidx]
            · rw [h2']; omega
        split_ifs at hnext with hlt
        · obtain ⟨-, hst, hen, hexh⟩ := hnext
          have hd' : rangeOrderDone S E it' = it.start.val - S.val + 1 := by
            simp only [rangeOrderDone, hexh, Bool.false_eq_true, ite_false, hst]; omega
          refine ⟨⟨by rw [hen, hend], by omega, by rw [hend] at hlt; omega, ?_, ?_⟩, ?_⟩
          · rw [hd', hni, hidx, Nat.add_mul, Nat.one_mul]
          · rw [hd']; exact hext
          · simp only [hexh, hex, Bool.false_eq_true, ite_false, hst]
            rw [hend] at hlt
            have key : ∀ a b : Nat, a < b → b - (a + 1) + 1 < b - a + 1 := by intros; omega
            exact key _ _ hlt
        · obtain ⟨-, hst, hen, hexh⟩ := hnext
          have hEq : it.start.val = E.val := by rw [hend] at hlt; omega
          have hd' : rangeOrderDone S E it' = it.start.val - S.val + 1 := by
            simp only [rangeOrderDone, hexh, ite_true]; omega
          refine ⟨⟨by rw [hen, hend], by rw [hst]; exact hS, by rw [hst]; exact hE, ?_, ?_⟩, ?_⟩
          · rw [hd', hni, hidx, Nat.add_mul, Nat.one_mul]
          · rw [hd']; exact hext
          · simp [hexh, hex]
  · exact hinv

theorem ensure_range_transition_order_spec (span : TimestampSpan) (ids : Slice U256)
    (ts : Slice SuperRangeTransition) (hse : span.start.val ≤ span.end.val)
    (hlen : ts.val.length = spanCount span * ids.val.length) :
    ensure_range_transition_order span ids ts ⦃ r =>
      r = .Ok () → ∀ j k, j < spanCount span → k < ids.val.length →
        ∃ tr, ts.val[j * ids.val.length + k]? = some tr ∧
          tr.timestamp.val = span.start.val + j ∧
          ids.val[k]? = some tr.optimistic_block.chain_id ⦄ := by
  unfold ensure_range_transition_order
  simp only [core.ops.range.RangeInclusive.new, Std.bind_ok]
  apply ensure_range_transition_order_loop_spec span.start span.end ids ts hse hlen
  refine ⟨rfl, le_refl _, hse, ?_, ?_⟩
  · simp [rangeOrderDone]
  · intro j k hj; simp [rangeOrderDone] at hj

theorem ensure_range_transition_coverage_spec (span : TimestampSpan) (ids : Slice U256)
    (ts : Slice SuperRangeTransition) :
    ensure_range_transition_coverage span ids ts ⦃ r =>
      r = .Ok () → RangeCoverage span ids.val ts.val ⦄ := by
  unfold ensure_range_transition_coverage
  apply spec_bind (inclusive_timestamp_count_spec span ids.len)
  intro r hr
  cases r with
  | Err e =>
    simp [core.result.Result.Insts.CoreOpsTry.branch,
      core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
  | Ok v =>
    obtain ⟨hse, hv⟩ := hr v rfl
    simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok,
      Usize.Insts.CoreConvertTryFromU64TryFromIntError.try_from, core.num.tryFromUScalar]
    by_cases hmax : v.val ≤ UScalar.max .Usize
    · simp only [hmax, ite_true, Std.bind_ok, core.result.Result.ok, core.option.Option.and_then,
          ensure_range_transition_coverage.closure.Insts.CoreOpsFunctionFnOnceTupleUsizeOptionUsize.call_once]
      have hcast : (UScalar.cast UScalarTy.Usize v).val = v.val := by
        apply UScalar.cast_val_mod_pow_of_inBounds_eq
        have : UScalar.max .Usize < 2 ^ UScalarTy.Usize.numBits := by
          simp [UScalar.max_def]
        omega
      have hmul := Usize.checked_mul_bv_spec (UScalar.cast UScalarTy.Usize v) ids.len
      cases hm : Usize.checked_mul (UScalar.cast UScalarTy.Usize v) ids.len with
      | none =>
        simp [core.option.Option.ok_or,
          core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
      | some w =>
        rw [hm] at hmul
        obtain ⟨-, hw, -⟩ := hmul
        simp only [core.option.Option.ok_or, Std.bind_ok]
        split_ifs with hneq
        · simp
        · have hlen : ts.val.length = spanCount span * ids.val.length := by
            simp only [bne_iff_ne, ne_eq, Decidable.not_not] at hneq
            have := congrArg UScalar.val hneq
            rw [Slice.len_val, hw, hcast, Slice.len_val, hv] at this
            exact this
          apply spec_mono (ensure_range_transition_order_spec span ids ts hse hlen)
          intro r hr' hok
          refine ⟨hse, hlen, ?_⟩
          intro j k hj hk
          obtain ⟨tr, h1, h2, h3⟩ := hr' hok j k hj hk
          refine ⟨tr, h1, h2, ?_⟩
          rw [List.getElem?_eq_getElem hk, Option.some.injEq] at h3
          exact h3.symm
    · simp [hmax, core.result.Result.ok, core.option.Option.and_then, core.option.Option.ok_or,
        core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]

/-! ## Level 2: lookups and the two output loops -/

/-- Mirror of `consolidation_transition_at`: the first output whose span contains `t`,
indexed by `(t - start) as usize`. -/
def consLookup : List SuperConsolidationOutputs → Nat → Option SuperConsolidationTransition
  | [], _ => none
  | c :: cs, t =>
    if c.span.start.val ≤ t ∧ t ≤ c.span.end.val then
      c.transitions.val[(t - c.span.start.val) % 2 ^ UScalarTy.Usize.numBits]?
    else consLookup cs t

theorem U64_size : U64.size = 2 ^ 64 := by
  simp [U64.size, U64.numBits]

theorem wrapping_sub_val (x y : Std.U64) (h : y.val ≤ x.val) :
    (x.wrapping_sub y).val = x.val - y.val := by
  rw [U64.wrapping_sub_val_eq, UScalar.size_UScalarTyU64]
  have hx : x.val < U64.size := by have := x.hBounds; simpa [U64_size] using this
  rw [show x.val + (U64.size - y.val) = (x.val - y.val) + U64.size by omega, Nat.add_mod_right]
  exact Nat.mod_eq_of_lt (by omega)

theorem consLookup_drop (l : List SuperConsolidationOutputs) (k t : Nat)
    (hk : k < l.length) :
    consLookup (l.drop k) t =
      if l[k].span.start.val ≤ t ∧ t ≤ l[k].span.end.val then
        l[k].transitions.val[(t - l[k].span.start.val) % 2 ^ UScalarTy.Usize.numBits]?
      else consLookup (l.drop (k + 1)) t := by
  rw [List.drop_eq_getElem_cons hk]
  rfl

theorem consolidation_transition_at_loop_spec (s : Slice SuperConsolidationOutputs)
    (t : Std.U64) :
    SuperAggregationInputs.consolidation_transition_at_loop ⟨s, 0⟩ t ⦃ r =>
      ∀ tr, r = .Ok tr ↔ consLookup s.val t.val = some tr ⦄ := by
  unfold SuperAggregationInputs.consolidation_transition_at_loop
  apply loop.spec_decr_nat
    (measure := fun z => s.val.length - z.i)
    (inv := fun z => z.slice = s ∧ z.i ≤ s.val.length ∧
      consLookup (s.val.drop z.i) t.val = consLookup s.val t.val)
  · intro z ⟨h1, hk, hinv⟩
    obtain ⟨s1, k⟩ := z
    simp only at h1 hk hinv
    subst h1
    unfold SuperAggregationInputs.consolidation_transition_at_loop.body
    simp only [core.slice.iter.IteratorSliceIter.next]
    by_cases hk' : k < s1.val.length
    · rw [consLookup_drop _ _ _ hk'] at hinv
      simp [hk', TimestampSpan.contains]
      have hout' : s1.val[k] = s1[k] := rfl
      rw [hout'] at hinv
      generalize s1[k] = out at hinv ⊢
      split_ifs with hc1
      · simp only [Std.bind_ok]
        split_ifs with hc2
        · have hc : out.span.start.val ≤ t.val ∧ t.val ≤ out.span.end.val :=
            ⟨hc1, by simpa using hc2⟩
          rw [ite_eq_left_of_eq_true _ _ (eq_true hc)] at hinv
          have hidx : (UScalar.cast UScalarTy.Usize (t.wrapping_sub out.span.start)).val =
              (t.val - out.span.start.val) % 2 ^ UScalarTy.Usize.numBits := by
            rw [UScalar.cast_val_eq, wrapping_sub_val _ _ hc1]
          simp only [lift, Std.bind_ok, core.slice.Slice.get, core.slice.index.Usize.get,
            Slice.getElem?_Usize_eq, hidx]
          simp only [alloc.vec.Vec.deref, Slice.from_val]
          cases hget : out.transitions.val[(t.val - out.span.start.val) %
              2 ^ UScalarTy.Usize.numBits]? <;>
            simp [core.option.Option.ok_or] <;> rw [← hinv, hget] <;> simp
        · have hc : ¬(out.span.start.val ≤ t.val ∧ t.val ≤ out.span.end.val) := by
            intro h; simp at hc2; omega
          rw [ite_eq_right_of_eq_false _ _ (eq_false hc)] at hinv
          simp only [spec_ok]
          exact ⟨trivial, by omega, hinv, by omega⟩
      · have hc : ¬(out.span.start.val ≤ t.val ∧ t.val ≤ out.span.end.val) := by
          intro h; exact hc1 h.1
        rw [ite_eq_right_of_eq_false _ _ (eq_false hc)] at hinv
        simp only [Std.bind_ok, Bool.false_eq_true, ite_false, spec_ok]
        exact ⟨trivial, by omega, hinv, by omega⟩
    · have hk2 : k = s1.val.length := by omega
      subst hk2
      simp at hinv ⊢
      intro tr h
      rw [← hinv] at h
      simp [consLookup] at h
  · exact ⟨rfl, Nat.zero_le _, rfl⟩

theorem consolidation_transition_at_spec (A : SuperAggregationInputs) (t : Std.U64) :
    A.consolidation_transition_at t ⦃ r =>
      ∀ tr, r = .Ok tr ↔ consLookup A.consolidation_outputs.val t.val = some tr ⦄ := by
  unfold SuperAggregationInputs.consolidation_transition_at
  simp only [SharedAVec.Insts.CoreIterTraitsCollectIntoIteratorSharedATIter.into_iter,
    Std.bind_ok]
  apply spec_mono (consolidation_transition_at_loop_spec _ t)
  intro r hr tr
  simpa [alloc.vec.Vec.deref] using hr tr

/-- Mirror of `super_root_before_timestamp` (`timestamp - 1` wraps). -/
def superRootBefore (A : SuperAggregationInputs) (t first : Nat) : Option B256 :=
  if t = first then some A.starting_root_hash
  else (consLookup A.consolidation_outputs.val ((t + (2 ^ 64 - 1)) % 2 ^ 64)).map (·.super_root)

theorem super_root_before_timestamp_spec (A : SuperAggregationInputs) (t first : Std.U64) :
    A.super_root_before_timestamp t first ⦃ r =>
      ∀ h, r = .Ok h ↔ superRootBefore A t.val first.val = some h ⦄ := by
  unfold SuperAggregationInputs.super_root_before_timestamp superRootBefore
  by_cases h : t = first
  · subst h; simp
  · have hv : t.val ≠ first.val := fun e => h (UScalar.eq_of_val_eq e)
    simp only [h, hv, ite_false, lift, Std.bind_ok]
    apply spec_bind (consolidation_transition_at_spec _ _)
    intro r hr
    have hi : (t.wrapping_sub 1#u64).val = (t.val + (2 ^ 64 - 1)) % 2 ^ 64 := by
      rw [U64.wrapping_sub_val_eq, UScalar.size_UScalarTyU64, U64_size]
      rfl
    rw [hi] at hr
    cases r with
    | Ok tr =>
      have htr := (hr tr).mp rfl
      norm_num at htr
      simp [core.result.Result.Insts.CoreOpsTry.branch, htr]
    | Err e =>
      simp [core.result.Result.Insts.CoreOpsTry.branch,
        core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
      intro tr htr
      have := (hr tr).mpr (by norm_num; exact htr)
      simp at this

/-- Mirror of the `validate_consolidation_outputs` loop, from the expected span start and
expected previous super root. -/
def ConsChain (claim : Nat) (rootClaim : B256) (ids : List U256) :
    Nat → B256 → List SuperConsolidationOutputs → Prop
  | start, prev, [] => start = min (claim + 1) (2 ^ 64 - 1) ∧ prev = rootClaim
  | start, prev, c :: cs =>
    c.span.start.val = start ∧ c.span.start.val ≤ c.span.end.val ∧ c.span.end.val ≤ claim ∧
    c.previous_super_root = prev ∧
    c.transitions.val.length = spanCount c.span ∧
    (∀ j (hj : j < c.transitions.val.length),
      (c.transitions.val[j]).timestamp.val = c.span.start.val + j ∧
      (c.transitions.val[j]).optimistic_blocks.val.map (·.chain_id) = ids) ∧
    c.span.end.val + 1 < 2 ^ 64 ∧
    ConsChain claim rootClaim ids (c.span.end.val + 1)
      ((c.transitions.val.getLast?.map (·.super_root)).getD prev) cs

theorem getLast_drop_step {α β : Type} (l : List α) (k : Nat) (hk : k < l.length)
    (f : α → β) (p : β) :
    ((l.drop k).getLast?.map f).getD p = ((l.drop (k + 1)).getLast?.map f).getD (f l[k]) := by
  rw [List.drop_eq_getElem_cons hk, List.getLast?_cons]
  cases (l.drop (k + 1)).getLast? <;> rfl

theorem validate_consolidation_outputs_inner_spec (s : Slice SuperConsolidationTransition)
    (prev : B256) :
    SuperAggregationInputs.validate_consolidation_outputs_loop0_loop0 ⟨s, 0⟩ prev ⦃ r =>
      r = (s.val.getLast?.map (·.super_root)).getD prev ⦄ := by
  unfold SuperAggregationInputs.validate_consolidation_outputs_loop0_loop0
  apply loop.spec_decr_nat
    (measure := fun z => s.val.length - z.1.i)
    (inv := fun z => z.1.slice = s ∧ z.1.i ≤ s.val.length ∧
      ((s.val.drop z.1.i).getLast?.map (·.super_root)).getD z.2 =
        (s.val.getLast?.map (·.super_root)).getD prev)
  · intro z ⟨h1, hk, hinv⟩
    obtain ⟨⟨s1, k⟩, p⟩ := z
    simp only at h1 hk hinv
    subst h1
    unfold SuperAggregationInputs.validate_consolidation_outputs_loop0_loop0.body
    simp only [core.slice.iter.IteratorSliceIter.next]
    by_cases hk' : k < s1.val.length
    · rw [getLast_drop_step _ _ hk'] at hinv
      simp [hk']
      refine ⟨?_, by omega⟩
      rw [← hinv]
      cases (List.drop (k + 1) s1.val).getLast? <;> rfl
    · have hk2 : k = s1.val.length := by omega
      subst hk2
      simp at hinv ⊢
      exact hinv
  · exact ⟨rfl, Nat.zero_le _, rfl⟩

theorem saturating_add_one (x : Std.U64) :
    (core.num.U64.saturating_add x 1#u64).val = min (x.val + 1) (2 ^ 64 - 1) := by
  have hx : x.val < 2 ^ 64 := by have := x.hBounds; simpa using this
  have h1 : (1#u64).val = 1 := by rfl
  show (BitVec.ofNat _ (min (UScalar.max .U64) (x.val + (1#u64).val))).toNat = _
  rw [BitVec.toNat_ofNat, h1, UScalar.max_UScalarTy_U64_eq, U64.max_eq]
  simp only [UScalarTy.numBits]
  omega

theorem validate_consolidation_outputs_loop_spec (S : Slice SuperConsolidationOutputs)
    (fb : B256) (i : Std.U64) (ids : Slice U256) (st0 : Std.U64) (prev0 : B256) :
    SuperAggregationInputs.validate_consolidation_outputs_loop0 ⟨S, 0⟩ fb i ids st0 prev0 ⦃ r =>
      r = .Ok () → ConsChain i.val fb ids.val st0.val prev0 S.val ⦄ := by
  unfold SuperAggregationInputs.validate_consolidation_outputs_loop0
  apply loop.spec_decr_nat
    (measure := fun z => S.val.length - z.1.i)
    (inv := fun z => z.1.slice = S ∧ z.1.i ≤ S.val.length ∧
      (ConsChain i.val fb ids.val z.2.1.val z.2.2 (S.val.drop z.1.i) →
        ConsChain i.val fb ids.val st0.val prev0 S.val))
  · intro z ⟨h1, hk, hinv⟩
    obtain ⟨⟨s1, k⟩, est, eprev⟩ := z
    simp only at h1 hk hinv
    subst h1
    unfold SuperAggregationInputs.validate_consolidation_outputs_loop0.body
    simp only [core.slice.iter.IteratorSliceIter.next]
    by_cases hk' : k < s1.val.length
    · simp [hk']
      rw [List.drop_eq_getElem_cons hk'] at hinv
      have hout' : s1.val[k] = s1[k] := rfl
      rw [hout'] at hinv
      generalize s1[k] = out at hinv ⊢
      apply spec_bind (TimestampSpan.validate_spec out.span)
      intro r hr
      cases r with
      | Err e =>
        simp [core.result.Result.Insts.CoreOpsTry.branch,
          core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
      | Ok u =>
        have hse := hr.mp rfl
        simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok]
        split_ifs with hst hend
        · simp
        · simp only [core.cmp.PartialEq.ne.trait_default, core.cmp.PartialEq.ne.default,
            alloy_primitives.bits.fixed.FixedBytes.Insts.CoreCmpPartialEqFixedBytes.eq, Std.bind_ok,
            Std.bind_tc_ok, decide_not, Bool.not_eq_eq_eq_not, Bool.not_true, decide_eq_false_iff_not]
          by_cases hprev' : out.previous_super_root = eprev
          · simp only [hprev', decide_true, not_true_eq_false, ite_false]
            apply spec_bind (ensure_consolidation_transition_coverage_spec out.span ids
              (alloc.vec.Vec.deref out.transitions))
            intro r1 hr1
            cases r1 with
            | Err e =>
              simp [core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
            | Ok u1 =>
              have hcov := hr1 rfl
              simp only [alloc.vec.Vec.deref, Slice.from_val] at hcov
              simp only [Std.bind_ok,
                SharedAVec.Insts.CoreIterTraitsCollectIntoIteratorSharedATIter.into_iter]
              apply spec_bind (validate_consolidation_outputs_inner_spec _ eprev)
              intro pnext hpnext
              simp only [alloc.vec.Vec.deref, Slice.from_val] at hpnext
              have hca := U64.checked_add_bv_spec out.span.end 1#u64
              cases hc : out.span.end.checked_add 1#u64 with
              | none =>
                simp [lift, core.option.Option.ok_or,
                  core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
              | some z =>
                rw [hc] at hca
                simp only [lift, Std.bind_ok, core.option.Option.ok_or, spec_ok]
                refine ⟨trivial, by omega, ?_, by omega⟩
                intro hrest
                apply hinv
                have h1 : (1#u64).val = 1 := rfl
                have hz : z.val = out.span.end.val + 1 := by rw [hca.2.1, h1]
                have hlt : out.span.end.val + 1 < 2 ^ 64 := by
                  have := hca.1; rw [h1, U64.max_eq] at this; omega
                refine ⟨by simpa using hst, hse, by simpa using hend, hprev', hcov.2.1, hcov.2.2,
                  hlt, ?_⟩
                rw [← hz, ← hpnext]
                exact hrest
          · simp [hprev']
        · simp
    · have hk2 : k = s1.val.length := by omega
      subst hk2
      simp only [List.drop_length] at hinv
      simp [lift, saturating_add_one, core.cmp.PartialEq.ne.trait_default,
        core.cmp.PartialEq.ne.default,
        alloy_primitives.bits.fixed.FixedBytes.Insts.CoreCmpPartialEqFixedBytes.eq]
      split_ifs with he hp
      · simp only [spec_ok]
        intro _
        exact hinv ⟨by simpa using he, hp⟩
      · simp
      · simp
  · exact ⟨rfl, Nat.zero_le _, by simp⟩

theorem validate_consolidation_outputs_spec (A : SuperAggregationInputs)
    (first : Std.U64) (ids : Slice U256) :
    A.validate_consolidation_outputs first ids ⦃ r =>
      r = .Ok () →
        ConsChain A.l2_sequence_number.val A.root_claim ids.val first.val
          A.starting_root_hash A.consolidation_outputs.val ⦄ := by
  unfold SuperAggregationInputs.validate_consolidation_outputs
  simp only [SharedAVec.Insts.CoreIterTraitsCollectIntoIteratorSharedATIter.into_iter,
    Std.bind_ok]
  apply spec_mono (validate_consolidation_outputs_loop_spec _ _ _ _ _ _)
  intro r hr hok
  simpa [alloc.vec.Vec.deref] using hr hok

/-- What `validate_range_output_timestamps` establishes on `Ok`, for each offset `j`: the
previous super root matches the consolidated chain, and the range blocks for that
timestamp equal the consolidated blocks. -/
def RangeTimestamps (A : SuperAggregationInputs) (first : Nat) (n : Nat)
    (r : SuperRangeOutputs) : Prop :=
  ∀ j, j < spanCount r.span →
    r.previous_super_roots.val[j]? = superRootBefore A (r.span.start.val + j) first ∧
    ∃ tr, consLookup A.consolidation_outputs.val (r.span.start.val + j) = some tr ∧
      ((r.transitions.val.drop (j * n)).take n).map (·.optimistic_block) =
        tr.optimistic_blocks.val

theorem map_next_slice {T B F : Type} (fInst : core.ops.function.FnMut F T B)
    (g : T → B) (hg : ∀ c x, fInst.call_mut c x = ok (g x, c)) (s : Slice T) (f : F)
    (i : Nat) (hi : i < s.val.length) :
    core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator.next
      (core.iter.traits.iterator.IteratorSliceIter T) fInst ⟨⟨s, i⟩, f⟩ =
      ok (some (g s.val[i]), ⟨⟨s, i + 1⟩, f⟩) := by
  have hnext2 : (core.iter.traits.iterator.IteratorSliceIter T).next =
      core.slice.iter.IteratorSliceIter.next := rfl
  simp [core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator.next, 
    core.slice.iter.IteratorSliceIter.next, hi, hg]
  rfl

theorem map_next_slice_end {T B F : Type} (fInst : core.ops.function.FnMut F T B)
    (s : Slice T) (f : F) (i : Nat) (hi : ¬ i < s.val.length) :
    core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator.next
      (core.iter.traits.iterator.IteratorSliceIter T) fInst ⟨⟨s, i⟩, f⟩ =
      ok (none, ⟨⟨s, i⟩, f⟩) := by
  have hnext2 : (core.iter.traits.iterator.IteratorSliceIter T).next =
      core.slice.iter.IteratorSliceIter.next := rfl
  simp [core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator.next, 
    core.slice.iter.IteratorSliceIter.next, hi]

theorem iterToList_map {T B F : Type} (fInst : core.ops.function.FnMut F T B)
    (g : T → B) (hg : ∀ c x, fInst.call_mut c x = ok (g x, c)) (s : Slice T) (f : F) :
    ∀ (n i : Nat), s.val.length - i = n → ∀ (acc : List B),
    alloc.vec.FromIteratorVec.iterToList
      (core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator
        (core.iter.traits.iterator.IteratorSliceIter T) fInst) ⟨⟨s, i⟩, f⟩ acc =
      ok (acc.reverse ++ (s.val.drop i).map g) := by
  have hnext : (core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator
      (core.iter.traits.iterator.IteratorSliceIter T) fInst).next =
      core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator.next
        (core.iter.traits.iterator.IteratorSliceIter T) fInst := rfl
  intro n
  induction n with
  | zero =>
    intro i hi acc
    rw [alloc.vec.FromIteratorVec.iterToList, hnext, map_next_slice_end fInst s f i (by omega)]
    simp only [Std.bind_tc_ok]
    simp [List.drop_eq_nil_of_le (show s.val.length ≤ i by omega)]
  | succ n ih =>
    intro i hi acc
    rw [alloc.vec.FromIteratorVec.iterToList, hnext, map_next_slice fInst g hg s f i (by omega)]
    simp only [Std.bind_tc_ok]
    rw [ih (i + 1) (by omega)]
    congr 1
    rw [List.drop_eq_getElem_cons (show i < s.val.length by omega), List.map_cons,
      List.reverse_cons, List.append_assoc]
    rfl

/-- `collect()` of `map` over a slice iterator whose closure is a pure function. -/
theorem collect_map_slice_iter_spec {T B F : Type} (fInst : core.ops.function.FnMut F T B)
    (g : T → B) (hg : ∀ c x, fInst.call_mut c x = ok (g x, c)) (s : Slice T) (f : F) :
    core.iter.traits.iterator.Iterator.collect.default
      (core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator
        (core.iter.traits.iterator.IteratorSliceIter T) fInst)
      (core.iter.traits.collect.FromIteratorVec B) ⟨⟨s, 0⟩, f⟩ ⦃ v =>
      v.val = s.val.map g ⦄ := by
  have h := iterToList_map fInst g hg s f (s.val.length - 0) 0 rfl []
  simp only [List.reverse_nil, List.nil_append, List.drop_zero] at h
  simp only [core.iter.traits.iterator.Iterator.collect.default,
    alloc.vec.FromIteratorVec.from_iter,
    
    core.iter.traits.collect.IntoIterator.Blanket.into_iter, Std.bind_tc_ok]
  rw [h]
  simp only [Std.bind_tc_ok]
  have hlen : (s.val.map g).length ≤ Usize.max := by
    simp
  simp []

open Classical in
theorem optblock_eq (a b : SuperOptimisticBlock) :
    SuperOptimisticBlock.Insts.CoreCmpPartialEqSuperOptimisticBlock.eq a b = ok (decide (a = b)) := by
  obtain ⟨c1, h1, o1⟩ := a
  obtain ⟨c2, h2, o2⟩ := b
  simp only [SuperOptimisticBlock.Insts.CoreCmpPartialEqSuperOptimisticBlock.eq,
    ruint.Uint.Insts.CoreCmpPartialEqUint.eq,
    alloy_primitives.bits.fixed.FixedBytes.Insts.CoreCmpPartialEqFixedBytes.eq, SuperOptimisticBlock.mk.injEq]
  by_cases hc : c1 = c2 <;> by_cases hh : h1 = h2 <;> by_cases ho : o1 = o2 <;> simp [hc, hh, ho]

open Classical in
theorem anyM_zip_ne_aux {T : Type} :
    ∀ (l1 l2 : List T), l1.length = l2.length →
      List.anyM (fun (x : T × T) => (ok (decide (x.1 ≠ x.2)) : Result Bool)) (List.zip l1 l2) =
        ok (decide (l1 ≠ l2)) := by
  intro l1
  induction l1 with
  | nil =>
    intro l2 h
    cases l2 with
    | nil => simp_all; rfl
    | cons _ _ => simp_all
  | cons a l1 ih =>
    intro l2 h
    cases l2 with
    | nil => simp at h
    | cons b l2 =>
      simp only [List.zip_cons_cons, List.anyM_cons, Std.bind_tc_ok]
      by_cases hab : a = b
      · subst hab
        simp only [ne_eq, not_true_eq_false, decide_false, Bool.false_eq_true, ↓reduceIte]
        rw [ih l2 (by simpa using h)]
        simp
      · simp [hab]
        rfl

open Classical in
theorem anyM_zip_ne {T : Type} (inst : core.cmp.PartialEq T T)
    (hne : ∀ a b, inst.ne a b = ok (decide (a ≠ b))) (l1 l2 : List T) (h : l1.length = l2.length) :
      List.anyM (fun (x : T × T) => inst.ne x.1 x.2) (List.zip l1 l2) = ok (decide (l1 ≠ l2)) := by
  have hf : (fun (x : T × T) => inst.ne x.1 x.2) = fun x => ok (decide (x.1 ≠ x.2)) := by
    funext x; exact hne x.1 x.2
  rw [hf]
  exact anyM_zip_ne_aux l1 l2 h

open Classical in
theorem vec_ne_optblock (v0 v1 : alloc.vec.Vec SuperOptimisticBlock) :
    alloc.vec.partial_eq.PartialEqVec.ne
      SuperOptimisticBlock.Insts.CoreCmpPartialEqSuperOptimisticBlock v0 v1 =
      ok (decide (v0.val ≠ v1.val)) := by
  have hne : ∀ a b, SuperOptimisticBlock.Insts.CoreCmpPartialEqSuperOptimisticBlock.ne a b =
      ok (decide (a ≠ b)) := by
    intro a b
    simp [
      core.cmp.PartialEq.ne.default, optblock_eq]
  unfold alloc.vec.partial_eq.PartialEqVec.ne
  split_ifs with h
  · exact anyM_zip_ne _ hne _ _ h
  · have : v0.val ≠ v1.val := fun e => h (congrArg List.length e)
    simp [this]

theorem enum_range_next
    (x : core.iter.adapters.enumerate.Enumerate (core.ops.range.RangeInclusive Std.U64))
    (hc : x.iter.exhausted = false → x.iter.start.val ≤ x.iter.«end».val →
      x.count.val + 1 ≤ Usize.max) :
    core.iter.adapters.enumerate.IteratorEnumerate.next
      (core.ops.range.RangeInclusive.Insts.CoreIterTraitsIteratorIterator core.iter.range.StepU64) x
    ⦃ o x' =>
      ((x.iter.exhausted = true ∨ x.iter.«end».val < x.iter.start.val) ∧ o = none) ∨
      (x.iter.exhausted = false ∧ x.iter.start.val ≤ x.iter.«end».val ∧
        o = some (x.count, x.iter.start) ∧
        x'.count.val = x.count.val + 1 ∧ x'.iter.«end» = x.iter.«end» ∧
        (x.iter.start.val < x.iter.«end».val →
          x'.iter.start.val = x.iter.start.val + 1 ∧ x'.iter.exhausted = false) ∧
        (¬ x.iter.start.val < x.iter.«end».val → x'.iter.exhausted = true)) ⦄ := by
  unfold core.iter.adapters.enumerate.IteratorEnumerate.next
  have hn : (core.ops.range.RangeInclusive.Insts.CoreIterTraitsIteratorIterator
      core.iter.range.StepU64).next =
      core.ops.range.RangeInclusive.Insts.CoreIterTraitsIteratorIterator.next
        core.iter.range.StepU64 := rfl
  rw [hn]
  apply spec_bind
    (core.ops.range.RangeInclusive.Insts.CoreIterTraitsIteratorIterator.next_U64_spec x.iter)
  rintro ⟨o, r'⟩ h
  by_cases he : x.iter.exhausted = true ∨ x.iter.«end».val < x.iter.start.val
  · simp only [he, ↓reduceIte] at h
    obtain ⟨rfl, rfl⟩ := h
    simp [he]
  · simp only [he, ↓reduceIte] at h
    have hex : x.iter.exhausted = false := by simpa using (not_or.mp he).1
    have hle : x.iter.start.val ≤ x.iter.«end».val := by have := (not_or.mp he).2; omega
    have hcnt := hc hex hle
    by_cases hlt : x.iter.start.val < x.iter.«end».val
    · simp only [hlt, ↓reduceIte] at h
      obtain ⟨rfl, h1, h2, h3⟩ := h
      simp only
      apply spec_bind (Usize.add_spec (x := x.count) (y := 1#usize) (by simpa using hcnt))
      intro c hcv
      simp only [spec_ok]
      right
      exact ⟨hex, hle, rfl, by simpa using hcv, h2, fun _ => ⟨h1, h3⟩, fun h' => absurd hlt h'⟩
    · simp only [hlt, ↓reduceIte] at h
      obtain ⟨rfl, h1, h2, h3⟩ := h
      simp only
      apply spec_bind (Usize.add_spec (x := x.count) (y := 1#usize) (by simpa using hcnt))
      intro c hcv
      simp only [spec_ok]
      right
      exact ⟨hex, hle, rfl, by simpa using hcv, h2, fun h' => absurd h' hlt, fun _ => h3⟩

theorem usize_max_lt_size : Usize.max < UScalar.size .Usize := by
  rw [UScalar.size_UScalarTyUsize, Usize.max_def, Usize.size_def]
  have : 0 < 2 ^ Usize.numBits := Nat.two_pow_pos _
  omega

/-- Property of offset `j` that the timestamps loop establishes. -/
def rangeTimestampOK (A : SuperAggregationInputs) (first n : Nat) (r : SuperRangeOutputs) (j : Nat) : Prop :=
  r.previous_super_roots.val[j]? = superRootBefore A (r.span.start.val + j) first ∧
  ∃ tr, consLookup A.consolidation_outputs.val (r.span.start.val + j) = some tr ∧
    ((r.transitions.val.drop (j * n)).take n).map (·.optimistic_block) = tr.optimistic_blocks.val

/-- Loop invariant of `validate_range_output_timestamps_loop`. -/
def rangeTimestampsInv (A : SuperAggregationInputs) (first n : Nat) (r : SuperRangeOutputs)
    (x : core.iter.adapters.enumerate.Enumerate (core.ops.range.RangeInclusive Std.U64)) : Prop :=
  x.iter.«end» = r.span.end ∧ x.count.val ≤ spanCount r.span ∧
  (x.iter.exhausted = true → x.count.val = spanCount r.span) ∧
  (x.iter.exhausted = false → x.count.val < spanCount r.span ∧
    x.iter.start.val = r.span.start.val + x.count.val) ∧
  ∀ j, j < x.count.val → rangeTimestampOK A first n r j

theorem validate_range_output_timestamps_loop_spec (A : SuperAggregationInputs)
    (r : SuperRangeOutputs) (first : Std.U64) (ids : Slice U256)
    (hse : r.span.start.val ≤ r.span.end.val)
    (hprs : r.previous_super_roots.val.length = spanCount r.span)
    (hts : r.transitions.val.length = spanCount r.span * ids.val.length)
    (x0 : core.iter.adapters.enumerate.Enumerate (core.ops.range.RangeInclusive Std.U64))
    (h0 : rangeTimestampsInv A first.val ids.val.length r x0) :
    SuperAggregationInputs.validate_range_output_timestamps_loop x0 A r.previous_super_roots r.transitions first ids
    ⦃ res => res = .Ok () → ∀ j, j < spanCount r.span → rangeTimestampOK A first.val ids.val.length r j ⦄ := by
  unfold SuperAggregationInputs.validate_range_output_timestamps_loop
  apply loop.spec_decr_nat
    (measure := fun x => spanCount r.span - x.count.val)
    (inv := rangeTimestampsInv A first.val ids.val.length r)
  · intro x hinv
    obtain ⟨hend, hcnt, hexT, hexF, hpre⟩ := hinv
    unfold SuperAggregationInputs.validate_range_output_timestamps_loop.body
    have hcmax : spanCount r.span ≤ Usize.max := by
      have := alloc.vec.Vec.len_ineq r.previous_super_roots
      omega
    apply spec_bind (enum_range_next x (by
      intro hex _
      have := (hexF hex).1
      omega))
    rintro ⟨o, x'⟩ h
    rcases h with ⟨hemp, rfl⟩ | ⟨hex, hle, rfl, hc', hend', hlt1, hlt2⟩
    · show spec (ok (done (core.result.Result.Ok ()))) _
      rw [spec_ok]
      intro _ j hj
      cases hx : x.iter.exhausted
      · obtain ⟨h1, h2⟩ := hexF hx
        rcases hemp with h | h
        · simp [hx] at h
        · rw [hend] at h; unfold spanCount at *; omega
      · exact hpre j (by have := hexT hx; omega)
    · obtain ⟨hcx, hsx⟩ := hexF hex
      dsimp only
      apply spec_bind (super_root_before_timestamp_spec A x.iter.start first)
      intro r1 h1
      cases r1 with
      | Err e =>
        simp [core.result.Result.Insts.CoreOpsTry.branch,
          core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
      | Ok val =>
        have hval := (h1 val).mp rfl
        simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok]
        have hj : x.count.val < r.previous_super_roots.val.length := by omega
        rw [alloc.vec.Vec.index_slice_index]
        have hidx : alloc.vec.Vec.index_usize r.previous_super_roots x.count =
            ok (r.previous_super_roots.val[x.count.val]) := by
          simp [alloc.vec.Vec.index_usize, hj]
        rw [hidx]
        simp only [Std.bind_tc_ok, core.cmp.PartialEq.ne.trait_default, core.cmp.PartialEq.ne.default,
          
          alloy_primitives.bits.fixed.FixedBytes.Insts.CoreCmpPartialEqFixedBytes.eq]
        by_cases hpv : r.previous_super_roots.val[x.count.val] = val
        · simp only [hpv, Std.bind_ok, decide_true, decide_not, Bool.not_true,
            Bool.false_eq_true, ↓reduceIte, lift]
          have hn : (Slice.len ids).val = ids.val.length := Slice.len_val ids
          have hjn : (x.count.val + 1) * ids.val.length ≤ r.transitions.val.length := by
            rw [hts]; exact Nat.mul_le_mul_right _ (by omega)
          have htmax := alloc.vec.Vec.len_ineq r.transitions
          have hsz := usize_max_lt_size
          have htsv : (x.count.wrapping_mul (Slice.len ids)).val = x.count.val * ids.val.length := by
            rw [Usize.wrapping_mul_val_eq, hn]
            apply Nat.mod_eq_of_lt
            have : x.count.val * ids.val.length ≤ (x.count.val + 1) * ids.val.length :=
              Nat.mul_le_mul_right _ (Nat.le_succ _)
            omega
          have htev : ((x.count.wrapping_mul (Slice.len ids)).wrapping_add (Slice.len ids)).val =
              x.count.val * ids.val.length + ids.val.length := by
            rw [Usize.wrapping_add_val_eq, htsv, hn]
            apply Nat.mod_eq_of_lt
            have : x.count.val * ids.val.length + ids.val.length =
              (x.count.val + 1) * ids.val.length := by ring
            omega
          simp only [core.slice.Slice.get,
            core.slice.index.SliceIndexRangeUsizeSlice.get]
          have hle1 : x.count.wrapping_mul (Slice.len ids) ≤
              (x.count.wrapping_mul (Slice.len ids)).wrapping_add (Slice.len ids) := by
            scalar_tac
          have hle2 : ((x.count.wrapping_mul (Slice.len ids)).wrapping_add (Slice.len ids)).val ≤
              (alloc.vec.Vec.deref r.transitions).length := by
            rw [htev]; simp [alloc.vec.Vec.deref]
            have : x.count.val * ids.val.length + ids.val.length =
              (x.count.val + 1) * ids.val.length := by ring
            omega
          rw [ite_eq_left ⟨hle1, hle2⟩]
          simp only [Std.bind_ok, core.option.Option.ok_or, core.slice.Slice.iter,
            core.iter.traits.iterator.Iterator.map.default]
          apply spec_bind (collect_map_slice_iter_spec _
            (fun t : SuperRangeTransition => t.optimistic_block) (fun _ _ => rfl) _ ())
          intro rb hrb
          apply spec_bind (consolidation_transition_at_spec A x.iter.start)
          intro r2 h2
          cases r2 with
          | Err e =>
            simp [
              core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
          | Ok val2 =>
            have hval2 := (h2 val2).mp rfl
            simp only [Std.bind_ok, 
              vec_ne_optblock]
            by_cases heq : rb.val = val2.optimistic_blocks.val
            · simp only [heq, ne_eq, not_true_eq_false, decide_false, Bool.false_eq_true,
                ↓reduceIte, spec_ok]
              have hendv : x.iter.«end».val = r.span.«end».val := by rw [hend]
              have hblocks : ((r.transitions.val.drop (x.count.val * ids.val.length)).take
                  ids.val.length).map (·.optimistic_block) = val2.optimistic_blocks.val := by
                rw [← heq, hrb]
                simp [List.slice, htsv, htev, alloc.vec.Vec.deref]
              refine ⟨⟨hend'.trans hend, by omega, ?_, ?_, ?_⟩, by omega⟩
              · intro hx'
                by_cases hlt : x.iter.start.val < x.iter.«end».val
                · simp [(hlt1 hlt).2] at hx'
                · unfold spanCount; omega
              · intro hx'
                by_cases hlt : x.iter.start.val < x.iter.«end».val
                · refine ⟨by unfold spanCount; omega, ?_⟩
                  rw [(hlt1 hlt).1, hc', hsx]; omega
                · simp [hlt2 hlt] at hx'
              · intro j hj'
                rw [hc'] at hj'
                rcases Nat.lt_succ_iff_lt_or_eq.mp hj' with hj' | rfl
                · exact hpre j hj'
                · refine ⟨?_, val2, ?_, hblocks⟩
                  · rw [← hsx, hval, List.getElem?_eq_getElem hj, hpv]
                  · rw [← hsx]; exact hval2
            · simp [heq]
        · simp [hpv]
  · exact h0

theorem validate_range_output_timestamps_spec (A : SuperAggregationInputs)
    (r : SuperRangeOutputs) (first : Std.U64) (ids : Slice U256)
    (hse : r.span.start.val ≤ r.span.end.val)
    (hprs : r.previous_super_roots.val.length = spanCount r.span)
    (hts : r.transitions.val.length = spanCount r.span * ids.val.length) :
    A.validate_range_output_timestamps r first ids ⦃ res =>
      res = .Ok () → RangeTimestamps A first.val ids.val.length r ⦄ := by
  unfold SuperAggregationInputs.validate_range_output_timestamps
  simp only [core.ops.range.RangeInclusive.new,
    core.iter.traits.iterator.Iterator.enumerate.trait_default,
    core.iter.traits.iterator.Iterator.enumerate.default, Std.bind_ok]
  apply spec_mono (validate_range_output_timestamps_loop_spec A r first ids hse hprs hts _ ?_)
  · intro res h hok
    exact h hok
  · refine ⟨rfl, Nat.zero_le _, ?_, ?_, ?_⟩
    · intro h; simp at h
    · intro _; refine ⟨by simp, ?_⟩; simp
    · intro j hj; simp at hj

/-- Mirror of the `validate_range_outputs` loop, from the expected span start. -/
def RangeChain (A : SuperAggregationInputs) (first : Nat) (ids : List U256) :
    Nat → List SuperRangeOutputs → Prop
  | start, [] => start = min (A.l2_sequence_number.val + 1) (2 ^ 64 - 1)
  | start, r :: rs =>
    r.l1_head = A.l1_head ∧ r.span.start.val = start ∧ r.span.start.val ≤ r.span.end.val ∧
    r.span.end.val ≤ A.l2_sequence_number.val ∧
    r.previous_super_roots.val.length = spanCount r.span ∧
    RangeCoverage r.span ids r.transitions.val ∧
    RangeTimestamps A first ids.length r ∧
    r.span.end.val + 1 < 2 ^ 64 ∧
    RangeChain A first ids (r.span.end.val + 1) rs

theorem sat_add_val (x y : Std.U64) :
    (core.num.U64.saturating_add x y).val = min (x.val + y.val) (2 ^ 64 - 1) := by
  have hm : U64.max = 2^64 - 1 := by simp [U64.max, U64.numBits]
  simp only [core.num.U64.saturating_add, UScalar.saturating_add, UScalar.val, BitVec.toNat_ofNat]
  rw [Nat.mod_eq_of_lt (by have := min_le_left (UScalar.max .U64) (x.val + y.val); simp at *; omega)]
  simp [min_comm]
  rw [hm]; rfl

theorem slice_next_some {T : Type} (s : Slice T) (k : Nat) (h : k < s.val.length) :
    core.slice.iter.IteratorSliceIter.next ⟨s, k⟩ = ok (some s.val[k], ⟨s, k + 1⟩) := by
  simp [core.slice.iter.IteratorSliceIter.next, h]
  rfl

theorem slice_next_none {T : Type} (s : Slice T) (k : Nat) (h : ¬ k < s.val.length) :
    core.slice.iter.IteratorSliceIter.next ⟨s, k⟩ = ok (none, ⟨s, k⟩) := by
  simp [core.slice.iter.IteratorSliceIter.next, h]

/-- Loop invariant of `validate_range_outputs_loop`. -/
def rangeOutputsInv (A : SuperAggregationInputs) (first : Std.U64) (ids : Slice U256)
    (p : core.slice.iter.Iter SuperRangeOutputs × Std.U64) : Prop :=
  p.1.slice = alloc.vec.Vec.deref A.range_outputs ∧ p.1.i ≤ A.range_outputs.val.length ∧
  (RangeChain A first.val ids.val p.2.val (A.range_outputs.val.drop p.1.i) →
    RangeChain A first.val ids.val first.val A.range_outputs.val)

theorem validate_range_outputs_spec (A : SuperAggregationInputs) (first : Std.U64)
    (ids : Slice U256) :
    A.validate_range_outputs first ids ⦃ r =>
      r = .Ok () → RangeChain A first.val ids.val first.val A.range_outputs.val ⦄ := by
  unfold SuperAggregationInputs.validate_range_outputs
  simp only [SharedAVec.Insts.CoreIterTraitsCollectIntoIteratorSharedATIter.into_iter, Std.bind_ok
    ]
  unfold SuperAggregationInputs.validate_range_outputs_loop
  apply loop.spec_decr_nat
    (measure := fun p => A.range_outputs.val.length - p.1.i)
    (inv := rangeOutputsInv A first ids)
  · rintro ⟨⟨sl, k⟩, es⟩ ⟨h1, hk, himp⟩
    simp only at h1 hk himp
    subst h1
    have hdl : (alloc.vec.Vec.deref A.range_outputs).val = A.range_outputs.val := by
      simp [alloc.vec.Vec.deref]
    unfold SuperAggregationInputs.validate_range_outputs_loop.body
    by_cases hk' : k < A.range_outputs.val.length
    · have hk2 : k < (alloc.vec.Vec.deref A.range_outputs).val.length := by rw [hdl]; exact hk'
      dsimp only
      rw [slice_next_some _ k hk2]
      apply spec_bind (Pₘ := fun p => p = (some (alloc.vec.Vec.deref A.range_outputs).val[k],
        ⟨alloc.vec.Vec.deref A.range_outputs, k + 1⟩)) ((spec_ok _).mpr rfl)
      rintro ⟨o, it⟩ hp
      simp only [Prod.mk.injEq] at hp
      obtain ⟨rfl, rfl⟩ := hp
      simp only [Std.uncurry_apply_pair]
      generalize hout : (alloc.vec.Vec.deref A.range_outputs).val[k] = out
      have hout' : A.range_outputs.val[k] = out := by
        rw [← hout]; simp [alloc.vec.Vec.deref]
      simp only [core.cmp.PartialEq.ne.trait_default, core.cmp.PartialEq.ne.default,
        
        alloy_primitives.bits.fixed.FixedBytes.Insts.CoreCmpPartialEqFixedBytes.eq,
        Std.bind_ok, Std.bind_tc_ok]
      by_cases hl1 : out.l1_head = A.l1_head
      swap
      · simp [hl1]
      simp only [hl1, decide_true, not_true_eq_false, decide_false, 
        Bool.false_eq_true, ↓reduceIte]
      by_cases hs : out.span.start = es
      swap
      · simp [hs]
      simp only [hs, bne_self_eq_false, Bool.false_eq_true, ↓reduceIte]
      by_cases he : out.span.end > A.l2_sequence_number
      · simp [he]
      simp only [he, ↓reduceIte]
      have hderef : ∀ {T : Type} (v : alloc.vec.Vec T), (alloc.vec.Vec.deref v).val = v.val := by
        intro T v; simp [alloc.vec.Vec.deref]
      apply spec_bind (ensure_range_previous_super_root_outputs_spec out.span _)
      intro r0 h0
      cases r0 with
      | Err e =>
        simp [core.result.Result.Insts.CoreOpsTry.branch,
          core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
      | Ok u =>
        cases u
        obtain ⟨hse, hprs⟩ := h0 rfl
        rw [hderef] at hprs
        simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok]
        apply spec_bind (ensure_range_transition_coverage_spec out.span ids _)
        intro r1 h1
        cases r1 with
        | Err e =>
          simp [
            core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
        | Ok u =>
          cases u
          have hcov := h1 rfl
          rw [hderef] at hcov
          simp only [Std.bind_ok]
          apply spec_bind (validate_range_output_timestamps_spec A out first ids hse hprs hcov.2.1)
          intro r2 h2
          cases r2 with
          | Err e =>
            simp [
              core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
          | Ok u =>
            cases u
            have hrt := h2 rfl
            simp only [Std.bind_ok, lift]
            have hcb := U64.checked_add_bv_spec out.span.end 1#u64
            cases hco : U64.checked_add out.span.end 1#u64 with
            | none =>
              simp [core.option.Option.ok_or, 
                core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
            | some v =>
              rw [hco] at hcb
              obtain ⟨hmax, hv, _⟩ := hcb
              simp only [core.option.Option.ok_or, Std.bind_ok, 
                spec_ok]
              have hm : U64.max = 2 ^ 64 - 1 := by simp [U64.max, U64.numBits]
              refine ⟨⟨rfl, show k + 1 ≤ _ by omega, ?_⟩, show _ - (k + 1) < _ - k by omega⟩
              intro hrest
              apply himp
              rw [List.drop_eq_getElem_cons hk', hout']
              simp only [RangeChain]
              refine ⟨hl1, by rw [hs], hse, by scalar_tac, hprs, hcov, hrt, by simp at hmax; omega, ?_⟩
              simp at hv
              rw [← hv]
              exact hrest
    · have hk2 : ¬ k < (alloc.vec.Vec.deref A.range_outputs).val.length := by rw [hdl]; exact hk'
      dsimp only
      rw [slice_next_none _ k hk2]
      apply spec_bind (Pₘ := fun p => p = (none, ⟨alloc.vec.Vec.deref A.range_outputs, k⟩))
        ((spec_ok _).mpr rfl)
      rintro ⟨o, it⟩ hp
      simp only [Prod.mk.injEq] at hp
      obtain ⟨rfl, rfl⟩ := hp
      simp only [Std.uncurry_apply_pair, Std.bind_ok, lift]
      by_cases hes : es = core.num.U64.saturating_add A.l2_sequence_number 1#u64
      · simp only [hes, bne_self_eq_false, Bool.false_eq_true, ↓reduceIte, spec_ok]
        intro _
        apply himp
        rw [List.drop_eq_nil_of_le (by omega)]
        simp only [RangeChain]
        rw [hes, sat_add_val]
        simp
      · simp [hes]
  · refine ⟨rfl, Nat.zero_le _, ?_⟩
    simp

end KonaSp1
