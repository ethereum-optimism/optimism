import KonaSp1.Proofs.Spec

/-! Soundness of the super-aggregation checks: an accepted aggregation covers every
timestamp from the starting super root to the claim with one contiguous chain of verified
range and consolidation steps. -/

open Aeneas Aeneas.Std Result ControlFlow WP

namespace KonaSp1
open _root_.KonaSp1.super_root

/-- Chain IDs of the starting super root, as `U256`. -/
def startChainIds (A : SuperAggregationInputs) : List U256 :=
  A.starting_super_root_proof.super_root.output_roots.val.map (fun o => u256OfU64 o.chain_id)

/-- What `SuperAggregationInputs::validate` establishes on `Ok`. -/
def ValidateOk (A : SuperAggregationInputs) : Prop :=
  let t0 := A.starting_super_root_proof.super_root.timestamp.val
  _root_.super_root.hash_super_root_proof A.starting_super_root_proof =
    ok (core.result.Result.Ok A.starting_root_hash) ∧
  A.range_outputs.val ≠ [] ∧ A.consolidation_outputs.val ≠ [] ∧
  t0 + 1 < 2 ^ 64 ∧ t0 + 1 ≤ A.l2_sequence_number.val ∧
  ConsChain A.l2_sequence_number.val A.root_claim (startChainIds A) (t0 + 1)
    A.starting_root_hash A.consolidation_outputs.val ∧
  RangeChain A (t0 + 1) (startChainIds A) (t0 + 1) A.range_outputs.val

theorem bind_eq_ok {α β} {m : Result α} {k : α → Result β} {x : β}
    (h : Std.bind m k = ok x) : ∃ y, m = ok y ∧ k y = ok x := by
  induction m using Result.cases with
  | ret y => exact ⟨y, rfl, by simpa using h⟩
  | vis i k' =>
    rw [Std.bind_vis] at h
    have := congrArg Result.match h
    simp [Result.match.vis, Result.match.ok] at this
  | div =>
    rw [Std.bind_div] at h
    have := congrArg Result.match h
    simp [Result.match.div, Result.match.ok] at this

theorem spec_of_eq_ok {α} {m : Result α} {p : Post α} {x : α} (hs : m ⦃ p ⦄) (h : m = ok x) :
    p x := by
  subst h; simpa using hs

theorem validate_ok (A : SuperAggregationInputs) (h : A.validate = ok (.Ok ())) :
    ValidateOk A := by
  unfold SuperAggregationInputs.validate at h
  obtain ⟨hr, hhash, hk⟩ := bind_eq_ok h
  cases hr with
  | Err e =>
    simp [core.result.Result.Insts.CoreOpsTry.branch,
      core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual] at hk
  | Ok val =>
    clear h
    simp only [core.result.Result.Insts.CoreOpsTry.branch, Std.bind_ok, Std.bind_tc_ok,
      core.cmp.PartialEq.ne.trait_default, core.cmp.PartialEq.ne.default,
      alloy_primitives.bits.fixed.FixedBytes.Insts.CoreCmpPartialEqFixedBytes.eq,
      alloc.vec.Vec.is_empty, lift, core.option.Option.ok_or,
      core.result.Result.Insts.CoreOpsTry_traitFromResidualResult.from_residual] at hk
    split_ifs at hk with hne hre hce
    all_goals try simp at hk
    have hstart : val = A.starting_root_hash := by simpa using hne
    subst hstart
    have hca := U64.checked_add_bv_spec A.starting_super_root_proof.super_root.timestamp 1#u64
    cases hc : A.starting_super_root_proof.super_root.timestamp.checked_add 1#u64 with
    | none => simp [hc] at hk
    | some val1 =>
      rw [hc] at hca
      obtain ⟨hmax, hval1, -⟩ := hca
      simp only [hc, Std.bind_ok] at hk
      split_ifs at hk with hgt
      · simp at hk
      simp only [core.slice.Slice.iter, core.iter.traits.iterator.Iterator.map.default,
        Std.bind_ok] at hk
      have hg : ∀ c (x : kona_interop.root.OutputRootWithChain),
          SuperAggregationInputs.validate.closure.Insts.CoreOpsFunctionFnMutTupleShared0OutputRootWithChainUint2564.call_mut
            c x = ok (u256OfU64 x.chain_id, c) := by
        intro c x
        have hlt : x.chain_id.val < 2 ^ 256 := by scalar_tac
        simp [SuperAggregationInputs.validate.closure.Insts.CoreOpsFunctionFnMutTupleShared0OutputRootWithChainUint2564.call_mut,
          ruint.from.Uint.from, ruint.Uint.Insts.RuintFromUintTryFrom.uint_try_from,
          ruint.Uint.Insts.CoreConvertTryFromU64ToUintErrorUint.try_from, u256OfU64]
        split
        · simp
        · rename_i h; exact absurd (by simpa using hlt) h
      obtain ⟨cids, hcol, hk⟩ := bind_eq_ok hk
      have hcids := spec_of_eq_ok (collect_map_slice_iter_spec _ _ hg _ _) hcol
      have hids : (alloc.vec.Vec.deref cids).val = startChainIds A := by
        simp [startChainIds, hcids, alloc.vec.Vec.deref, Slice.from_val]
      obtain ⟨r2, hv2, hk⟩ := bind_eq_ok hk
      cases r2 with
      | Err e => simp at hk
      | Ok u =>
        simp only [Std.bind_ok] at hk
        have hcons := spec_of_eq_ok (validate_consolidation_outputs_spec A val1 _) hv2 rfl
        have hrange := spec_of_eq_ok (validate_range_outputs_spec A val1 _) hk rfl
        rw [hids] at hcons hrange
        have hv1 : val1.val = A.starting_super_root_proof.super_root.timestamp.val + 1 := by
          simpa using hval1
        rw [hv1] at hcons hrange
        refine ⟨hhash, by simpa using hre, by simpa using hce, ?_, by omega, hcons, hrange⟩
        scalar_tac

/-! ## Pure consequences of `ConsChain` and `RangeChain` -/

theorem usize_max_add_one : Usize.max + 1 = 2 ^ UScalarTy.Usize.numBits := by
  have : 1 ≤ 2 ^ UScalarTy.Usize.numBits := Nat.one_le_two_pow
  simp only [Usize.max_def, Usize.numBits_def]
  omega

theorem vec_length_lt_pow {α} (v : alloc.vec.Vec α) :
    v.val.length < 2 ^ UScalarTy.Usize.numBits := by
  have := v.len_ineq
  have := usize_max_add_one
  omega

/-- The super root before transition `j` of a consolidation output. -/
def consPrev (c : SuperConsolidationOutputs) : Nat → B256
  | 0 => c.previous_super_root
  | j + 1 => (c.transitions.val[j]?.map (·.super_root)).getD c.previous_super_root

theorem consLookup_cons_of_mem {c : SuperConsolidationOutputs} {cs t}
    (h1 : c.span.start.val ≤ t) (h2 : t ≤ c.span.end.val)
    (hlen : c.transitions.val.length = spanCount c.span) :
    consLookup (c :: cs) t = c.transitions.val[t - c.span.start.val]? := by
  have hlt : t - c.span.start.val < 2 ^ UScalarTy.Usize.numBits := by
    have := vec_length_lt_pow c.transitions
    simp only [spanCount] at hlen
    omega
  have hlt' : t - c.span.start.val < 2 ^ System.Platform.numBits := by
    simpa [UScalarTy.numBits] using hlt
  simp [consLookup, h1, h2, Nat.mod_eq_of_lt hlt']

theorem consLookup_cons_of_gt {c : SuperConsolidationOutputs} {cs t}
    (h : c.span.end.val < t) : consLookup (c :: cs) t = consLookup cs t := by
  have : ¬(c.span.start.val ≤ t ∧ t ≤ c.span.end.val) := by omega
  simp [consLookup, this]

section ConsChain
variable {claim : Nat} {rc : B256} {ids : List U256}

/-- Every timestamp from `start` to `claim` has a consolidation transition, which chains from
the previous timestamp's super root; the last super root is the root claim. -/
theorem ConsChain.lookup (hclaim : claim + 1 < 2 ^ 64) :
    ∀ (cs : List SuperConsolidationOutputs) (start : Nat) (prev : B256),
    ConsChain claim rc ids start prev cs →
    (∀ t, start ≤ t → t ≤ claim →
      ∃ c ∈ cs, ∃ j, ∃ hj : j < c.transitions.val.length,
        c.span.start.val + j = t ∧
        consLookup cs t = some (c.transitions.val[j]) ∧
        (c.transitions.val[j]).timestamp.val = t ∧
        (c.transitions.val[j]).optimistic_blocks.val.map (·.chain_id) = ids ∧
        consPrev c j =
          if t = start then prev
          else ((consLookup cs (t - 1)).map (·.super_root)).getD prev) ∧
    (start ≤ claim → (consLookup cs claim).map (·.super_root) = some rc) ∧
    (claim < start → prev = rc) := by
  intro cs
  induction cs with
  | nil =>
    intro start prev h
    obtain ⟨hstart, hprev⟩ := h
    have : start = claim + 1 := by omega
    refine ⟨fun t h1 h2 => by omega, fun h => by omega, fun _ => hprev⟩
  | cons c rest ih =>
    intro start prev h
    obtain ⟨hs, hse, hec, hprev, hlen, hcov, hend, hrest⟩ := h
    have hpos : 0 < c.transitions.val.length := by simp only [spanCount] at hlen; omega
    have hlast : c.transitions.val.getLast? =
        some (c.transitions.val[c.transitions.val.length - 1]'(by omega)) := by
      rw [List.getLast?_eq_getElem?]; simp [List.getElem?_eq_getElem (show
        c.transitions.val.length - 1 < c.transitions.val.length by omega)]
    obtain ⟨ih1, ih2, ih3⟩ := ih _ _ hrest
    refine ⟨?_, ?_, ?_⟩
    · intro t h1 h2
      by_cases hin : t ≤ c.span.end.val
      · have hj : t - start < c.transitions.val.length := by
          simp only [spanCount] at hlen; omega
        refine ⟨c, List.mem_cons_self .., t - start, hj, by omega, ?_, ?_, ?_, ?_⟩
        · rw [consLookup_cons_of_mem (by omega) hin hlen, hs]
          simp [List.getElem?_eq_getElem hj]
        · have := (hcov _ hj).1; omega
        · exact (hcov _ hj).2
        · by_cases ht : t = start
          · subst ht; simp [consPrev, hprev]
          · obtain ⟨j', hj'⟩ : ∃ j', t - start = j' + 1 := ⟨t - start - 1, by omega⟩
            have hj'lt : j' < c.transitions.val.length := by omega
            rw [hj', ite_eq_right ht, consLookup_cons_of_mem (by omega) (by omega) hlen]
            have : t - 1 - c.span.start.val = j' := by omega
            simp [consPrev, this, List.getElem?_eq_getElem hj'lt]
      · obtain ⟨c', hc', j, hj, hst, hlk, hts, hids, hprev'⟩ := ih1 t (by omega) h2
        refine ⟨c', List.mem_cons_of_mem _ hc', j, hj, hst, ?_, hts, hids, ?_⟩
        · rw [consLookup_cons_of_gt (by omega)]; exact hlk
        · rw [hprev', ite_eq_right (by omega : t ≠ start)]
          by_cases ht : t = c.span.end.val + 1
          · rw [ite_eq_left ht, ht, Nat.add_sub_cancel,
              consLookup_cons_of_mem (by omega) (le_refl _) hlen, hlast]
            have : c.span.end.val - c.span.start.val = c.transitions.val.length - 1 := by
              simp only [spanCount] at hlen; omega
            simp [this, List.getElem?_eq_getElem (show
              c.transitions.val.length - 1 < c.transitions.val.length by omega)]
          · rw [ite_eq_right ht, consLookup_cons_of_gt (by omega : c.span.end.val < t - 1)]
            obtain ⟨c'', -, j'', hj'', -, hlk', -⟩ := ih1 (t - 1) (by omega) (by omega)
            simp [hlk']
    · intro _
      by_cases hc : claim ≤ c.span.end.val
      · have hce : claim = c.span.end.val := by omega
        have := ih3 (by omega)
        rw [consLookup_cons_of_mem (by omega) hc hlen, ← this, hlast]
        have : claim - c.span.start.val = c.transitions.val.length - 1 := by
          simp only [spanCount] at hlen; omega
        simp [this, List.getElem?_eq_getElem (show
          c.transitions.val.length - 1 < c.transitions.val.length by omega)]
      · rw [consLookup_cons_of_gt (by omega)]
        exact ih2 (by omega)
    · intro h; omega

end ConsChain

section RangeChain
variable {A : SuperAggregationInputs} {first : Nat} {ids : List U256}

/-- Every timestamp from `start` to the claim lies in exactly one checked range output. -/
theorem RangeChain.lookup (hclaim : A.l2_sequence_number.val + 1 < 2 ^ 64) :
    ∀ (rs : List SuperRangeOutputs) (start : Nat), RangeChain A first ids start rs →
    ∀ t, start ≤ t → t ≤ A.l2_sequence_number.val →
      ∃ r ∈ rs, ∃ j, j < spanCount r.span ∧ r.span.start.val + j = t ∧
        r.l1_head = A.l1_head ∧
        RangeCoverage r.span ids r.transitions.val ∧ RangeTimestamps A first ids.length r := by
  intro rs
  induction rs with
  | nil =>
    intro start h t h1 h2
    simp only [RangeChain] at h
    omega
  | cons r rest ih =>
    intro start h t h1 h2
    obtain ⟨hl1, hs, hse, hec, -, hcov, hts, -, hrest⟩ := h
    by_cases hin : t ≤ r.span.end.val
    · exact ⟨r, List.mem_cons_self .., t - start, by simp only [spanCount]; omega,
        by omega, hl1, hcov, hts⟩
    · obtain ⟨r', hr', j, hj, hst, hl1', hcov', hts'⟩ := ih _ hrest t (by omega) h2
      exact ⟨r', List.mem_cons_of_mem _ hr', j, hj, hst, hl1', hcov', hts'⟩

end RangeChain

/-! ## Soundness -/

section Soundness

-- `RangeStep l1Head prevRoot t block`: the range program derived `block` for timestamp `t`
-- from the super root `prevRoot` with L1 head `l1Head`.
variable (RangeStep : B256 → B256 → Nat → SuperOptimisticBlock → Prop)

-- `ConsolidateStep prevRoot t blocks postRoot`: the consolidation program moved from
-- `prevRoot` to `postRoot` at timestamp `t` with the per-chain `blocks`.
variable (ConsolidateStep : B256 → Nat → List SuperOptimisticBlock → B256 → Prop)

/-- Statement of a verified range-mode proof: every transition was derived from the
previous super root that the output records for its timestamp. -/
def RangeSound (r : SuperRangeOutputs) : Prop :=
  ∀ tr ∈ r.transitions.val, ∀ h,
    r.previous_super_roots.val[tr.timestamp.val - r.span.start.val]? = some h →
    RangeStep r.l1_head h tr.timestamp.val tr.optimistic_block

/-- Statement of a verified consolidation-mode proof: each transition consolidates from the
super root before it. -/
def ConsSound (c : SuperConsolidationOutputs) : Prop :=
  ∀ j (hj : j < c.transitions.val.length),
    ConsolidateStep (consPrev c j) (c.transitions.val[j]).timestamp.val
      (c.transitions.val[j]).optimistic_blocks.val (c.transitions.val[j]).super_root

/-- If `validate` accepts and every child proof is sound, then a chain of super roots runs
from the starting root to the root claim, and each timestamp in between has one range step
per chain and one consolidation step. -/
theorem aggregation_sound (A : SuperAggregationInputs)
    (hok : A.validate = ok (.Ok ()))
    (hclaim : A.l2_sequence_number.val < 2 ^ 64 - 1)
    (hrange : ∀ r ∈ A.range_outputs.val, RangeSound RangeStep r)
    (hcons : ∀ c ∈ A.consolidation_outputs.val, ConsSound ConsolidateStep c) :
    _root_.super_root.hash_super_root_proof A.starting_super_root_proof =
      ok (core.result.Result.Ok A.starting_root_hash) ∧
    ∃ root : Nat → B256,
      root A.starting_super_root_proof.super_root.timestamp.val = A.starting_root_hash ∧
      root A.l2_sequence_number.val = A.root_claim ∧
      ∀ t, A.starting_super_root_proof.super_root.timestamp.val < t →
        t ≤ A.l2_sequence_number.val →
        ∃ blocks : List SuperOptimisticBlock,
          blocks.map (·.chain_id) = startChainIds A ∧
          (∀ b ∈ blocks, RangeStep A.l1_head (root (t - 1)) t b) ∧
          ConsolidateStep (root (t - 1)) t blocks (root t) := by
  obtain ⟨hhash, -, -, hfirst, hfc, hcc, hrc⟩ := validate_ok A hok
  refine ⟨hhash, ?_⟩
  set t0 := A.starting_super_root_proof.super_root.timestamp.val with ht0
  set claim := A.l2_sequence_number.val with hclaimdef
  set cs := A.consolidation_outputs.val
  set ids := startChainIds A
  set n := ids.length
  have hclaim' : claim + 1 < 2 ^ 64 := by omega
  obtain ⟨hc1, hc2, -⟩ := ConsChain.lookup hclaim' cs (t0 + 1) A.starting_root_hash hcc
  let root : Nat → B256 := fun t =>
    if t = t0 then A.starting_root_hash
    else ((consLookup cs t).map (·.super_root)).getD A.starting_root_hash
  -- The root before `t` in the chain is the super root that `superRootBefore` reports.
  have hbefore : ∀ t, t0 < t → t ≤ claim →
      superRootBefore A t (t0 + 1) = some (root (t - 1)) ∧
      (if t = t0 + 1 then A.starting_root_hash
        else ((consLookup cs (t - 1)).map (·.super_root)).getD A.starting_root_hash) =
        root (t - 1) := by
    intro t h1 h2
    by_cases ht : t = t0 + 1
    · simp [superRootBefore, ht, root]
    · obtain ⟨c, -, j, hj, -, hlk, -⟩ := hc1 (t - 1) (by omega) (by omega)
      have hwrap : (t + (2 ^ 64 - 1)) % 2 ^ 64 = t - 1 := by
        rw [show t + (2 ^ 64 - 1) = (t - 1) + 2 ^ 64 by omega, Nat.add_mod_right,
          Nat.mod_eq_of_lt (by omega)]
      have hne : t - 1 ≠ t0 := by omega
      have hlk' : consLookup A.consolidation_outputs.val (t - 1) = some (c.transitions.val[j]) :=
        hlk
      have hroot1 : root (t - 1) = (c.transitions.val[j]).super_root := by
        simp only [root, ite_eq_right hne, hlk, Option.map_some, Option.getD_some]
      refine ⟨?_, ?_⟩
      · simp only [superRootBefore, ite_eq_right ht]
        rw [hwrap, hroot1, hlk', Option.map_some]
      · rw [ite_eq_right ht, hroot1, hlk, Option.map_some, Option.getD_some]
  refine ⟨root, by simp [root], ?_, ?_⟩
  · have hne : claim ≠ t0 := by omega
    simp [root, hne, hc2 hfc]
  · intro t h1 h2
    obtain ⟨c, hc, jc, hjc, -, hlk, hts, hids, hprev⟩ := hc1 t (by omega) h2
    obtain ⟨r, hr, j, hj, hst, hl1, hcov, hrts⟩ :=
      RangeChain.lookup (by omega) A.range_outputs.val (t0 + 1) hrc t (by omega) h2
    obtain ⟨hprs, tr, hlk', hblocks⟩ := hrts j hj
    rw [hst] at hprs hlk'
    rw [hlk] at hlk'
    obtain rfl := Option.some.inj hlk'
    obtain ⟨hb, hroot⟩ := hbefore t h1 h2
    refine ⟨(c.transitions.val[jc]).optimistic_blocks.val, hids, ?_, ?_⟩
    · intro b hbmem
      rw [← hblocks] at hbmem
      obtain ⟨tr', htr'mem, rfl⟩ := List.mem_map.mp hbmem
      obtain ⟨k, hk, hk'⟩ := List.getElem_of_mem htr'mem
      have hklen : k < ids.length := by
        have := hk; simp only [List.length_take] at this; omega
      obtain ⟨hse, hlen, hcovk⟩ := hcov
      obtain ⟨trk, htrk, htrk_ts, -⟩ := hcovk j k hj hklen
      have hidx' : j * ids.length + k < r.transitions.val.length := by
        have := hk; simp only [List.length_take, List.length_drop] at this; omega
      have htrk_eq : trk = tr' := by
        rw [List.getElem?_eq_getElem hidx'] at htrk
        rw [← Option.some.inj htrk, ← hk']
        simp [List.getElem_take, List.getElem_drop]
      subst htrk_eq
      have hmem : trk ∈ r.transitions.val := List.mem_of_getElem? htrk
      have hidx : trk.timestamp.val - r.span.start.val = j := by omega
      have := hrange r hr trk hmem (root (t - 1)) (by rw [hidx, hprs, hb])
      rwa [hl1, htrk_ts, hst] at this
    · have hcs := hcons c hc jc hjc
      rw [hprev, hroot, hts] at hcs
      have hne : t ≠ t0 := by omega
      have hrt : root t = (c.transitions.val[jc]).super_root := by simp [root, hne, hlk]
      rw [hrt]
      exact hcs

end Soundness

end KonaSp1
