import KonaSp1.Proofs.Leaf

/-! The `expected` iterator of `validate_range_outputs`: the `flat_map` model yields the
`(timestamp, chain index)` grid in timestamp-major order. -/

open Aeneas Aeneas.Std Result ControlFlow WP

namespace KonaSp1
open _root_.KonaSp1.super_root

/-- `(0..chain_count).map(move |index| (timestamp, index))`; the closure holds `timestamp`. -/
abbrev InnerIter := core.iter.adapters.map.Map (core.ops.range.Range Std.Usize)
  SuperAggregationInputs.validate_range_outputs.closure.closure

/-- The `flat_map` iterator; its closure holds `chain_count`. -/
abbrev ExpIter := core.iter.adapters.flatten.FlatMap (core.ops.range.RangeInclusive Std.U64)
  InnerIter SuperAggregationInputs.validate_range_outputs.closure (Std.U64 × Std.Usize) InnerIter

abbrev rangeInst := core.ops.range.RangeInclusive.Insts.CoreIterTraitsIteratorIterator
  core.iter.range.StepU64

abbrev innerInst := core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator
  (core.iter.traits.iterator.IteratorRange core.iter.range.StepUsize)
  SuperAggregationInputs.validate_range_outputs.closure.closure.Insts.CoreOpsFunctionFnMutTupleUsizePairU64Usize

abbrev intoInst := core.iter.traits.collect.IntoIterator.Blanket innerInst

abbrev outerInst :=
  SuperAggregationInputs.validate_range_outputs.closure.Insts.CoreOpsFunctionFnMutTupleU64MapRangeUsizeclosure

abbrev expInst := core.iter.adapters.flatten.FlatMap.Insts.CoreIterTraitsIteratorIterator
  rangeInst intoInst outerInst

/-- Every `(t, i)` with `t` in `ts` and `i < n`, timestamp-major. -/
def gridItems (n : Nat) (ts : List Nat) : List (Nat × Nat) :=
  ts.flatMap (fun t => (List.range n).map (fun i => (t, i)))

/-- The items `validate_range_outputs` expects: chains `0..n` at each timestamp in
`first..=claim`. -/
def expectedItems (first claim n : Nat) : List (Nat × Nat) :=
  gridItems n (List.range' first (claim + 1 - first))

/-- Items left in an inner iterator. -/
def innerItems (m : InnerIter) : List (Nat × Nat) :=
  (List.range' m.iter.start.val (m.iter.«end».val - m.iter.start.val)).map (fun i => (m.f.val, i))

/-- Timestamps left in a `RangeInclusive`. -/
def rangeItems (r : core.ops.range.RangeInclusive Std.U64) : List Nat :=
  if r.exhausted then [] else List.range' r.start.val (r.«end».val + 1 - r.start.val)

/-- Items left in the `flat_map` iterator. -/
def expItems (n : Nat) (s : ExpIter) : List (Nat × Nat) :=
  (s.frontiter.map innerItems).getD [] ++
    (s.iter.map (fun m => gridItems n (rangeItems m.iter))).getD []

/-- States that `validate_range_outputs` reaches with `chain_count = n`. -/
def ExpWF (n : Nat) (s : ExpIter) : Prop :=
  s.backiter = none ∧ (∀ m, s.iter = some m → m.f.val = n) ∧
    (∀ m, s.frontiter = some m → m.iter.«end».val = n)

theorem gridItems_cons (n t : Nat) (ts : List Nat) :
    gridItems n (t :: ts) = (List.range n).map (fun i => (t, i)) ++ gridItems n ts := by
  simp [gridItems]

theorem mem_gridItems {n : Nat} {ts : List Nat} {x : Nat × Nat} :
    x ∈ gridItems n ts ↔ x.1 ∈ ts ∧ x.2 < n := by
  obtain ⟨t, i⟩ := x
  simp [gridItems]

theorem mem_expectedItems {first claim n : Nat} {x : Nat × Nat} :
    x ∈ expectedItems first claim n ↔ first ≤ x.1 ∧ x.1 ≤ claim ∧ x.2 < n := by
  simp only [expectedItems, mem_gridItems, List.mem_range'_1]
  omega

theorem expItems_idx (n : Nat) (s : ExpIter) (hwf : ExpWF n s) :
    ∀ x ∈ expItems n s, x.2 < n := by
  obtain ⟨-, hiter, hfront⟩ := hwf
  intro x hx
  simp only [expItems, List.mem_append] at hx
  rcases hx with hx | hx
  · cases hf : s.frontiter with
    | none => simp [hf] at hx
    | some m =>
      simp only [hf, Option.map_some, Option.getD_some, innerItems, List.mem_map,
        List.mem_range'] at hx
      obtain ⟨i, ⟨k, hk, rfl⟩, rfl⟩ := hx
      have := hfront m hf
      simp only
      omega
  · cases hi : s.iter with
    | none => simp [hi] at hx
    | some m =>
      simp only [hi, Option.map_some, Option.getD_some, mem_gridItems] at hx
      exact hx.2

theorem inner_next (m : InnerIter) :
    innerInst.next m ⦃ o m' => m'.iter.«end» = m.iter.«end» ∧
      (o = none → innerItems m = []) ∧
      (∀ x, o = some x → innerItems m = (x.1.val, x.2.val) :: innerItems m') ⦄ := by
  show core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator.next _ _ m ⦃ _ ⦄
  unfold core.iter.adapters.map.Map.Insts.CoreIterTraitsIteratorIterator.next
  have hn : (core.iter.traits.iterator.IteratorRange core.iter.range.StepUsize).next =
      core.iter.range.IteratorRange.next core.iter.range.StepUsize := rfl
  rw [hn]
  apply spec_bind (core.iter.range.IteratorRange.next_Usize_spec m.iter)
  rintro ⟨o, r'⟩ ⟨h, hend⟩
  by_cases hlt : m.iter.start.val < m.iter.«end».val
  · simp only [hlt, ↓reduceIte] at h
    obtain ⟨rfl, hs⟩ := h
    simp only [Std.uncurry_apply_pair,
      SuperAggregationInputs.validate_range_outputs.closure.closure.Insts.CoreOpsFunctionFnMutTupleUsizePairU64Usize.call_mut,
      Std.bind_ok, spec_ok, uncurry'_pair]
    refine ⟨hend, fun h => by simp at h, ?_⟩
    intro x hx
    simp only [Option.some.injEq] at hx
    subst hx
    simp only [innerItems, hs, hend]
    rw [show m.iter.«end».val - m.iter.start.val =
      (m.iter.«end».val - (m.iter.start.val + 1)) + 1 by omega, List.range'_succ]
    simp
  · simp only [hlt, ↓reduceIte] at h
    obtain ⟨rfl, -⟩ := h
    simp only [Std.uncurry_apply_pair, spec_ok, uncurry'_pair]
    refine ⟨hend, fun _ => ?_, fun x hx => by simp at hx⟩
    simp [innerItems, show m.iter.«end».val - m.iter.start.val = 0 by omega]

/-- Loop measure of the `flat_map` model. -/
def expMeasure (s : ExpIter) : Nat :=
  2 * (s.iter.map (fun m => (rangeItems m.iter).length)).getD 0 +
    (if s.frontiter.isSome then 1 else 0)

theorem rangeItems_next (r : core.ops.range.RangeInclusive Std.U64) :
    rangeInst.next r ⦃ o r' =>
      (o = none → rangeItems r = []) ∧
      (∀ t, o = some t → rangeItems r = t.val :: rangeItems r') ⦄ := by
  show core.ops.range.RangeInclusive.Insts.CoreIterTraitsIteratorIterator.next _ r ⦃ _ ⦄
  apply spec_mono
    (core.ops.range.RangeInclusive.Insts.CoreIterTraitsIteratorIterator.next_U64_spec r)
  rintro ⟨o, r'⟩ h
  simp only [uncurry'_pair] at h ⊢
  by_cases he : r.exhausted = true ∨ r.«end».val < r.start.val
  · simp only [he, ↓reduceIte] at h
    obtain ⟨rfl, -⟩ := h
    refine ⟨fun _ => ?_, fun t ht => by simp at ht⟩
    rcases he with he | he
    · simp [rangeItems, he]
    · simp [rangeItems, show r.«end».val + 1 - r.start.val = 0 by omega]
  · simp only [he, ↓reduceIte] at h
    have hex : r.exhausted = false := by simpa using (not_or.mp he).1
    have hle : r.start.val ≤ r.«end».val := by have := (not_or.mp he).2; omega
    by_cases hlt : r.start.val < r.«end».val
    · simp only [hlt, ↓reduceIte] at h
      obtain ⟨rfl, h1, h2, h3⟩ := h
      refine ⟨fun h => by simp at h, ?_⟩
      intro t ht
      simp only [Option.some.injEq] at ht
      subst ht
      simp only [rangeItems, hex, h3, h1, h2, Bool.false_eq_true, ↓reduceIte]
      rw [show r.«end».val + 1 - r.start.val = (r.«end».val + 1 - (r.start.val + 1)) + 1 by
        omega, List.range'_succ]
    · simp only [hlt, ↓reduceIte] at h
      obtain ⟨rfl, h1, h2, h3⟩ := h
      refine ⟨fun h => by simp at h, ?_⟩
      intro t ht
      simp only [Option.some.injEq] at ht
      subst ht
      simp only [rangeItems, hex, h3, Bool.false_eq_true, ↓reduceIte]
      rw [show r.«end».val + 1 - r.start.val = 1 by omega]
      simp

/-- The `flat_map` model yields the head of `expItems`. -/
theorem expNext_spec (n : Nat) (s : ExpIter) (hwf : ExpWF n s) :
    expInst.next s ⦃ o s' => ExpWF n s' ∧
      (o = none → expItems n s = []) ∧
      (∀ x, o = some x → expItems n s = (x.1.val, x.2.val) :: expItems n s') ⦄ := by
  show core.iter.adapters.flatten.FlatMap.Insts.CoreIterTraitsIteratorIterator.next _ _ _ s ⦃ _ ⦄
  unfold core.iter.adapters.flatten.FlatMap.Insts.CoreIterTraitsIteratorIterator.next
  simp only
  apply loop.spec_decr_nat (measure := expMeasure)
    (inv := fun s' => ExpWF n s' ∧ expItems n s' = expItems n s)
  · rintro s' ⟨⟨hback, hiter, hfront⟩, hitems⟩
    cases hf : s'.frontiter with
    | some inner =>
      apply spec_bind (inner_next inner)
      rintro ⟨o, inner1⟩ h
      simp only [uncurry'_pair] at h
      obtain ⟨hend, hnone, hsome⟩ := h
      cases o with
      | some x =>
        simp only [Std.uncurry_apply_pair, spec_ok, uncurry'_pair]
        refine ⟨⟨hback, hiter, ?_⟩, fun h => by simp at h, ?_⟩
        · intro m hm
          simp only [Option.some.injEq] at hm
          subst hm
          rw [hend]; exact hfront inner hf
        · intro y hy
          simp only [Option.some.injEq] at hy
          subst hy
          rw [← hitems]
          simp only [expItems, hf, Option.map_some, Option.getD_some, hsome _ rfl,
            List.cons_append]
      | none =>
        simp only [Std.uncurry_apply_pair, spec_ok]
        refine ⟨⟨⟨hback, hiter, by simp⟩, ?_⟩, ?_⟩
        · rw [← hitems]
          simp [expItems, hf, hnone rfl]
        · simp [expMeasure, hf]
    | none =>
      cases hi : s'.iter with
      | none =>
        simp only [hback, spec_ok, uncurry'_pair]
        refine ⟨⟨hback, hiter, hfront⟩, fun _ => ?_, fun x hx => by simp at hx⟩
        rw [← hitems]
        simp [expItems, hf, hi]
      | some m =>
        apply spec_bind (rangeItems_next m.iter)
        rintro ⟨o, it⟩ h
        simp only [uncurry'_pair] at h
        obtain ⟨hnone, hsome⟩ := h
        cases o with
        | none =>
          simp only [Std.uncurry_apply_pair, hback, spec_ok, uncurry'_pair]
          refine ⟨⟨by simp, by simp, by simp⟩, fun _ => ?_, fun x hx => by simp at hx⟩
          rw [← hitems]
          simp [expItems, hf, hi, hnone rfl, gridItems]
        | some t =>
          have hmf := hiter m hi
          simp only [Std.uncurry_apply_pair,
            SuperAggregationInputs.validate_range_outputs.closure.Insts.CoreOpsFunctionFnMutTupleU64MapRangeUsizeclosure.call_mut,
            core.iter.traits.iterator.Iterator.map.default, Std.bind_ok,
            core.iter.traits.collect.IntoIterator.Blanket.into_iter, spec_ok]
          refine ⟨⟨⟨hback, ?_, ?_⟩, ?_⟩, ?_⟩
          · intro m' hm'
            simp only [Option.some.injEq] at hm'
            subst hm'
            exact hmf
          · intro m' hm'
            simp only [Option.some.injEq] at hm'
            subst hm'
            exact hmf
          · rw [← hitems]
            simp only [expItems, hf, hi, Option.map_some, Option.getD_some, Option.map_none,
              Option.getD_none, List.nil_append, hsome t rfl, gridItems_cons, innerItems]
            congr 1
            simp [hmf, List.range_eq_range']
          · simp only [expMeasure, hf, hi, Option.map_some, Option.getD_some, Option.isSome_some,
              Option.isSome_none, hsome t rfl, List.length_cons]
            simp
            omega
  · exact ⟨hwf, rfl⟩

end KonaSp1
