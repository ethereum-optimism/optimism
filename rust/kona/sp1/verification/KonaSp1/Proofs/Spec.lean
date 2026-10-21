import KonaSp1.Proofs.Expected

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

theorem usize_max_lt_size : Usize.max < UScalar.size .Usize := by
  rw [UScalar.size_UScalarTyUsize, Usize.max_def, Usize.size_def]
  have : 0 < 2 ^ Usize.numBits := Nat.two_pow_pos _
  omega

open Classical in
theorem option_optblock_eq (a b : Option SuperOptimisticBlock) :
    core.option.Option.Insts.CoreCmpPartialEqOption.eq
      (core.cmp.PartialEqShared SuperOptimisticBlock.Insts.CoreCmpPartialEqSuperOptimisticBlock)
      a b = ok (decide (a = b)) := by
  cases a <;> cases b <;> simp [core.option.Option.Insts.CoreCmpPartialEqOption.eq, optblock_eq]

/-! ## Level 3: the range outputs -/

/-- What `validate_range_output_transitions` checks for transition `tr` of output `r` against
the expected item `x = (timestamp, chain index)`. -/
def TransOK (A : SuperAggregationInputs) (first : Nat) (ids : List U256) (r : SuperRangeOutputs)
    (tr : SuperRangeTransition) (x : Nat × Nat) : Prop :=
  tr.timestamp.val = x.1 ∧ ids[x.2]? = some tr.optimistic_block.chain_id ∧
  r.span.start.val ≤ x.1 ∧
  (∃ h, r.previous_super_roots.val[x.1 - r.span.start.val]? = some h ∧
    superRootBefore A x.1 first = some h) ∧
  ∃ ct, consLookup A.consolidation_outputs.val x.1 = some ct ∧
    ct.optimistic_blocks.val[x.2]? = some tr.optimistic_block

/-- `validate_range_output_transitions` for any `expected` iterator whose `next` yields the
head of `items`. -/
theorem validate_range_output_transitions_spec {T0 : Type}
    (inst : core.iter.traits.iterator.Iterator T0 (Std.U64 × Std.Usize))
    (items : T0 → List (Nat × Nat)) (wf : T0 → Prop)
    (hnext : ∀ e, wf e → inst.next e ⦃ o e' => wf e' ∧ (o = none → items e = []) ∧
      (∀ x, o = some x → items e = (x.1.val, x.2.val) :: items e') ⦄)
    (A : SuperAggregationInputs) (r : SuperRangeOutputs) (first : Std.U64) (ids : Slice U256)
    (hidx : ∀ e, wf e → ∀ x ∈ items e, x.2 < ids.val.length)
    (e : T0) (he : wf e) :
    A.validate_range_output_transitions inst r first ids e ⦃ res e' =>
      res = .Ok () → wf e' ∧ ∃ pre, items e = pre ++ items e' ∧
        List.Forall₂ (TransOK A first.val ids.val r) r.transitions.val pre ⦄ := by
  unfold SuperAggregationInputs.validate_range_output_transitions
  simp only [SharedAVec.Insts.CoreIterTraitsCollectIntoIteratorSharedATIter.into_iter,
    Std.bind_ok]
  unfold SuperAggregationInputs.validate_range_output_transitions_loop
  apply loop.spec_decr_nat
    (measure := fun z => r.transitions.val.length - z.1.i)
    (inv := fun z => z.1.slice = alloc.vec.Vec.deref r.transitions ∧
      z.1.i ≤ r.transitions.val.length ∧ wf z.2 ∧
      ∃ pre, items e = pre ++ items z.2 ∧
        List.Forall₂ (TransOK A first.val ids.val r) (r.transitions.val.take z.1.i) pre)
  · rintro ⟨⟨s1, k⟩, e1⟩ ⟨h1, hk, hwf1, pre, hpre, hfa⟩
    simp only at h1 hk hwf1 hpre hfa
    subst h1
    unfold SuperAggregationInputs.validate_range_output_transitions_loop.body
    have hS : (alloc.vec.Vec.deref r.transitions).val = r.transitions.val := by
      simp [alloc.vec.Vec.deref]
    simp only
    by_cases hk' : k < r.transitions.val.length
    · rw [slice_next_lt _ k (by rw [hS]; exact hk')]
      simp only [Std.bind_ok, Std.uncurry_apply_pair]
      have htr : (alloc.vec.Vec.deref r.transitions).val[k]'(by rw [hS]; exact hk') =
          r.transitions.val[k] := by simp [alloc.vec.Vec.deref]
      rw [htr]
      generalize htrdef : r.transitions.val[k] = tr
      apply spec_bind (hnext e1 hwf1)
      rintro ⟨o1, e2⟩ h
      simp only [uncurry'_pair] at h
      obtain ⟨hwf2, -, hsome⟩ := h
      cases o1 with
      | none =>
        simp [core.option.Option.ok_or, core.result.Result.Insts.CoreOpsTry.branch,
          core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
      | some x =>
        obtain ⟨t, ci⟩ := x
        have hitems1 := hsome (t, ci) rfl
        have hci : ci.val < ids.val.length := hidx e1 hwf1 (t.val, ci.val) (by rw [hitems1]; simp)
        simp only [core.option.Option.ok_or, core.result.Result.Insts.CoreOpsTry.branch,
          Std.bind_ok, Std.uncurry_apply_pair]
        apply spec_bind (Slice.index_usize_spec ids ci hci)
        intro cid hcid
        by_cases hts : tr.timestamp = t
        swap
        · simp [hts, uncurry'_pair]
        simp only [hts, bne_self_eq_false, Bool.false_eq_true, ↓reduceIte,
          core.cmp.PartialEq.ne.trait_default, core.cmp.PartialEq.ne.default,
          ruint.Uint.Insts.CoreCmpPartialEqUint.eq]
        by_cases hcid' : tr.optimistic_block.chain_id = cid
        swap
        · simp [hcid', uncurry'_pair]
        simp only [hcid', decide_true]
        have hcs := U64.checked_sub_bv_spec t r.span.start
        cases hsub : t.checked_sub r.span.start with
        | none =>
          simp [lift, core.option.Option.and_then,
            core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual, uncurry'_pair]
        | some d =>
          rw [hsub] at hcs
          obtain ⟨hle, hdv, -⟩ := hcs
          simp only [lift, Std.bind_ok, core.option.Option.and_then,
            SuperAggregationInputs.validate_range_output_transitions.closure.Insts.CoreOpsFunctionFnOnceTupleU64OptionUsize.call_once,
            Usize.Insts.CoreConvertTryFromU64TryFromIntError.try_from, core.num.tryFromUScalar]
          by_cases hmax : d.val ≤ UScalar.max .Usize
          swap
          · simp [hmax, core.result.Result.ok,
              core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual,
              uncurry'_pair]
          simp only [hmax, ↓reduceIte, Std.bind_ok, core.result.Result.ok,
            SuperAggregationInputs.validate_range_output_transitions.closure_1.Insts.CoreOpsFunctionFnOnceTupleUsizeOptionFixedBytes32.call_once,
            core.slice.Slice.get, core.slice.index.Usize.get,
            core.option.OptionShared0T.copied, Slice.getElem?_Usize_eq]
          have hcast : (UScalar.cast UScalarTy.Usize d).val = t.val - r.span.start.val := by
            rw [UScalar.cast_val_eq, ← hdv]
            apply Nat.mod_eq_of_lt
            have := usize_max_lt_size
            rw [UScalar.max_USize_eq] at hmax
            simp only [UScalar.size] at this
            omega
          have hprsS : (alloc.vec.Vec.deref r.previous_super_roots).val =
              r.previous_super_roots.val := by simp [alloc.vec.Vec.deref]
          rw [hcast, hprsS]
          simp only [Std.bind_tc_ok, not_true_eq_false, decide_false]
          cases hprv : r.previous_super_roots.val[t.val - r.span.start.val]? with
          | none => simp [core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual,
            uncurry'_pair]
          | some pv =>
            simp only [Std.bind_ok]
            apply spec_bind (super_root_before_timestamp_spec A t first)
            intro r2 h2
            cases r2 with
            | Err e => simp [core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual,
              uncurry'_pair]
            | Ok val2 =>
              have hval2 := (h2 val2).mp rfl
              simp only [Std.bind_ok, Std.bind_tc_ok,
                alloy_primitives.bits.fixed.FixedBytes.Insts.CoreCmpPartialEqFixedBytes.eq]
              by_cases hpv : pv = val2
              swap
              · simp [hpv, uncurry'_pair]
              simp only [hpv, decide_true, not_true_eq_false, decide_false, Bool.false_eq_true,
                ↓reduceIte]
              apply spec_bind (consolidation_transition_at_spec A t)
              intro r3 h3
              cases r3 with
              | Err e => simp [core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual,
                uncurry'_pair]
              | Ok ct =>
                have hct := (h3 ct).mp rfl
                have hblkS : (alloc.vec.Vec.deref ct.optimistic_blocks).val =
                    ct.optimistic_blocks.val := by simp [alloc.vec.Vec.deref]
                simp only [Std.bind_ok, Std.bind_tc_ok, option_optblock_eq]
                by_cases hcb : (alloc.vec.Vec.deref ct.optimistic_blocks).val[ci.val]? =
                  some tr.optimistic_block
                swap
                · simp [hcb, uncurry'_pair]
                simp only [hcb, decide_true, not_true_eq_false, decide_false, Bool.false_eq_true,
                  ↓reduceIte, spec_ok]
                rw [hblkS] at hcb
                refine ⟨⟨trivial, by omega, hwf2, pre ++ [(t.val, ci.val)], ?_, ?_⟩,
                  by omega⟩
                · rw [hpre, hitems1]; simp
                · rw [List.take_add_one, List.getElem?_eq_getElem hk', htrdef]
                  refine List.rel_append hfa (List.Forall₂.cons ?_ List.Forall₂.nil)
                  refine ⟨by rw [hts], ?_, hle, ⟨pv, hprv, by rw [hpv]; exact hval2⟩, ct, hct,
                    hcb⟩
                  rw [hcid', hcid, List.getElem?_eq_getElem hci]
    · have hk2 : k = r.transitions.val.length := by omega
      subst hk2
      rw [slice_next_ge _ _ (by rw [hS]; exact hk')]
      simp only [Std.bind_ok, Std.uncurry_apply_pair, spec_ok, uncurry'_pair]
      intro _
      refine ⟨hwf1, pre, hpre, ?_⟩
      simpa using hfa
  · exact ⟨rfl, Nat.zero_le _, he, [], by simp, by simp⟩

/-- What `validate_range_outputs` establishes on `Ok`: every range output has the game's L1
head, and the transitions of all outputs, in order, match the expected items one to one. -/
def RangeOK (A : SuperAggregationInputs) (first : Nat) (ids : List U256) : Prop :=
  (∀ r ∈ A.range_outputs.val, r.l1_head = A.l1_head) ∧
  List.Forall₂
    (fun (p : SuperRangeOutputs × SuperRangeTransition) x => TransOK A first ids p.1 p.2 x)
    (A.range_outputs.val.flatMap (fun r => r.transitions.val.map (fun tr => (r, tr))))
    (expectedItems first A.l2_sequence_number.val ids.length)

theorem validate_range_outputs_spec (A : SuperAggregationInputs) (first : Std.U64)
    (ids : Slice U256) :
    A.validate_range_outputs first ids ⦃ r => r = .Ok () → RangeOK A first.val ids.val ⦄ := by
  unfold SuperAggregationInputs.validate_range_outputs
  simp only [core.ops.range.RangeInclusive.new, core.iter.traits.iterator.Iterator.flat_map.default,
    SharedAVec.Insts.CoreIterTraitsCollectIntoIteratorSharedATIter.into_iter, Std.bind_ok]
  unfold SuperAggregationInputs.validate_range_outputs_loop
  have hS : (alloc.vec.Vec.deref A.range_outputs).val = A.range_outputs.val := by
    simp [alloc.vec.Vec.deref]
  apply loop.spec_decr_nat
    (measure := fun z => A.range_outputs.val.length - z.1.i)
    (inv := fun z => z.1.slice = alloc.vec.Vec.deref A.range_outputs ∧
      z.1.i ≤ A.range_outputs.val.length ∧ ExpWF ids.val.length z.2 ∧
      (∀ r ∈ A.range_outputs.val.take z.1.i, r.l1_head = A.l1_head) ∧
      ∃ pre, expectedItems first.val A.l2_sequence_number.val ids.val.length =
          pre ++ expItems ids.val.length z.2 ∧
        List.Forall₂
          (fun (p : SuperRangeOutputs × SuperRangeTransition) x =>
            TransOK A first.val ids.val p.1 p.2 x)
          ((A.range_outputs.val.take z.1.i).flatMap
            (fun r => r.transitions.val.map (fun tr => (r, tr))))
          pre)
  · rintro ⟨⟨s1, k⟩, e1⟩ ⟨h1, hk, hwf1, hl1, pre, hpre, hfa⟩
    simp only at h1 hk hwf1 hl1 hpre hfa
    subst h1
    unfold SuperAggregationInputs.validate_range_outputs_loop.body
    simp only
    by_cases hk' : k < A.range_outputs.val.length
    · rw [slice_next_lt _ k (by rw [hS]; exact hk')]
      simp only [Std.bind_ok, Std.uncurry_apply_pair]
      have hro : (alloc.vec.Vec.deref A.range_outputs).val[k]'(by rw [hS]; exact hk') =
          A.range_outputs.val[k] := by simp [alloc.vec.Vec.deref]
      rw [hro]
      generalize hodef : A.range_outputs.val[k] = out
      simp only [core.cmp.PartialEq.ne.trait_default, core.cmp.PartialEq.ne.default,
        alloy_primitives.bits.fixed.FixedBytes.Insts.CoreCmpPartialEqFixedBytes.eq, Std.bind_tc_ok]
      by_cases hl : out.l1_head = A.l1_head
      swap
      · simp [hl]
      simp only [hl, decide_true, not_true_eq_false, decide_false, Bool.false_eq_true, ↓reduceIte,
        Std.bind_ok]
      apply spec_bind (ensure_range_previous_super_root_outputs_spec out.span _)
      intro r0 _
      cases r0 with
      | Err e => simp [core.result.Result.Insts.CoreOpsTry.branch,
        core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
      | Ok u =>
        simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok,
          core.slice.Slice.first, core.slice.Slice.last]
        -- The span check: the transitions run from the span start to the span end.
        cases hfst : (alloc.vec.Vec.deref out.transitions).val.head? <;>
          cases hlst : (alloc.vec.Vec.deref out.transitions).val.getLast? <;>
          simp only [core.option.Option.map,
            SuperAggregationInputs.validate_range_outputs.closure_1.Insts.CoreOpsFunctionFnOnceTupleShared0SuperRangeTransitionU64.call_once,
            SuperAggregationInputs.validate_range_outputs.closure_2.Insts.CoreOpsFunctionFnOnceTupleShared0SuperRangeTransitionU64.call_once,
            Std.bind_ok, Std.bind_tc_ok, core.option.Option.Insts.CoreCmpPartialEqOption.eq,
            liftFun2, core.cmp.impls.PartialEqU64.eq]
        all_goals try (simp; done)
        rename_i a b
        by_cases ha : a.timestamp = out.span.start
        swap
        · simp [ha]
        by_cases hb : b.timestamp = out.span.«end»
        swap
        · simp [ha, hb]
        simp only [ha, hb, decide_true, Bool.false_eq_true, ↓reduceIte,
          not_true_eq_false, decide_false]
        apply spec_bind (validate_range_output_transitions_spec expInst (expItems ids.val.length)
          (ExpWF ids.val.length) (expNext_spec ids.val.length) A out first ids
          (expItems_idx ids.val.length) e1 hwf1)
        rintro ⟨r1, e2⟩ h
        simp only [uncurry'_pair] at h
        cases r1 with
        | Err e => simp [core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual]
        | Ok u1 =>
          obtain ⟨hwf2, pre2, hpre2, hfa2⟩ := h rfl
          simp only [Std.bind_ok, Std.uncurry_apply_pair,
            spec_ok]
          refine ⟨⟨trivial, by omega, hwf2, ?_, pre ++ pre2, ?_, ?_⟩, by omega⟩
          · intro r hr
            rw [List.take_add_one, List.getElem?_eq_getElem hk', hodef] at hr
            rcases List.mem_append.mp hr with hr | hr
            · exact hl1 r hr
            · simp only [Option.toList_some, List.mem_singleton] at hr
              rw [hr]; exact hl
          · rw [hpre, hpre2, List.append_assoc]
          · rw [List.take_add_one, List.getElem?_eq_getElem hk', hodef]
            simp only [Option.toList_some, List.flatMap_append, List.flatMap_singleton]
            exact List.rel_append hfa (List.forall₂_map_left_iff.mpr hfa2)
    · have hk2 : k = A.range_outputs.val.length := by omega
      subst hk2
      rw [slice_next_ge _ _ (by rw [hS]; exact hk')]
      simp only [Std.bind_ok, Std.uncurry_apply_pair]
      apply spec_bind (expNext_spec ids.val.length e1 hwf1)
      rintro ⟨o1, e2⟩ h
      simp only [uncurry'_pair] at h
      obtain ⟨-, hnone, -⟩ := h
      cases o1 with
      | some x =>
        obtain ⟨t, i⟩ := x
        simp [Std.uncurry_apply_pair]
      | none =>
        simp only [Std.uncurry_apply_pair, spec_ok]
        intro _
        rw [List.take_length] at hl1 hfa
        refine ⟨hl1, ?_⟩
        rw [hpre, hnone rfl, List.append_nil]
        exact hfa
  · refine ⟨rfl, Nat.zero_le _, ⟨rfl, ?_, by simp⟩, by simp, [], ?_, by simp⟩
    · intro m hm
      simp only [Option.some.injEq] at hm
      subst hm
      simp
    · simp [expItems, rangeItems, expectedItems]

end KonaSp1
