import Reasoning.Reach

/-! # `CREATE` as an `RD` step

EquiVM's block generator stops at `CREATE` (no `RD` lemma exists for it). `RD.create` below is
proved from EVMLean's own `step_create` (the semantics of the opcode) in the style of EquiVM's
`RD.call`: from a reached cursor at a `CREATE` with `[value, offset, size, …]` on the stack, the
next cursor is at `pc + 1` with the pushed result `x`, unchanged memory, the account map `σ'` and
return data `rd'` of the creation, related by `CreateStep` to EVMLean's `Lambda` (the
contract-creation function), or the whole run is out of gas. Nothing is assumed. -/

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach

/-- What the `CREATE` step computes, as EVMLean's `step_create` defines it, for the executing
    account `I.codeOwner` with account map `σ`, endowment `value` and init code `i`, under *some*
    substate `A` and *some* forwarded gas `gas`:
    * creator nonce `≥ 2^64 - 1`: nothing happens, `x = 0`;
    * insufficient balance, depth 1024, or init code longer than 49152 bytes: nothing happens,
      `x = 0`;
    * otherwise `Lambda` runs from `σ` with the creator's nonce incremented; `σ'` is its resulting
      account map, `x` is the new address if it succeeded (`z = true`) and `0` otherwise, and
      the return data is empty on success and `Lambda`'s output on failure. -/
def CreateStep (I : ExecutionEnv) (σ₀ σ : AccountMap) (value : UInt256) (i : ByteArray)
    (x : UInt256) (σ' : AccountMap) (rd' : ByteArray) : Prop :=
  ∃ (A : Substate) (gas : UInt256),
    let σa : Account := σ.get? I.codeOwner |>.getD default
    (σa.nonce.toNat ≥ 2 ^ 64 - 1 ∧ x = ⟨0⟩ ∧ σ' = σ ∧ rd' = .empty) ∨
    (σa.nonce.toNat < 2 ^ 64 - 1 ∧
      ¬ (value ≤ (σ.get? I.codeOwner |>.option ⟨0⟩ (·.balance)) ∧ I.depth < 1024 ∧ i.size ≤ 49152) ∧
      x = ⟨0⟩ ∧ σ' = σ ∧ rd' = .empty) ∨
    (σa.nonce.toNat < 2 ^ 64 - 1 ∧
      ∃ (hD : value ≤ (σ.get? I.codeOwner |>.option ⟨0⟩ (·.balance)) ∧ I.depth < 1024 ∧
          i.size ≤ 49152) (a : AccountAddress) (g' : UInt256) (A' : Substate) (z : Bool)
          (o : ByteArray),
        Lambda (σ.insert I.codeOwner {σa with nonce := σa.nonce + ⟨1⟩}) σ₀ A I.codeOwner I.sender
          gas (UInt256.ofNat I.gasPrice) value i ⟨I.depth.val + 1, Nat.succ_lt_succ hD.2.1⟩ none
          I.header I.blobVersionedHashes I.blocks I.perm = (a, σ', g', A', z, o) ∧
        x = (if z then UInt256.ofNat a else ⟨0⟩) ∧ rd' = (if z then .empty else o))

theorem ite3_or {α : Type _} {P Q R : Prop} [Decidable P] [Decidable Q] [Decidable R] {Y X : α} :
    (if P then Y else if Q then Y else if R then Y else X) = if (P ∨ Q ∨ R) then Y else X := by
  by_cases hP : P <;> by_cases hQ : Q <;> by_cases hR : R <;> simp [hP, hQ, hR]

/-- Rebuild the `RD` invariant after a `CREATE` step whose `Xstep` has `step_create`'s shape
    (memory-expansion guard, base-cost guard, the post-creation gas guard, then the successor `S`). -/
theorem rd_after_create {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 s S : State}
    {k C : ℕ} {pc : UInt256} {stk : List UInt256} {mem : ByteArray} {aw : UInt256}
    {rd : ByteArray} {σ' : AccountMap} {mc gc G : ℕ}
    (hX : X (g.toNat + 1) (D_J code 0) s0 = X (g.toNat + 1 - k) (D_J code 0) s)
    (hgas : s.machineState.gasAvailable = g.subNat C) (hk : k ≤ C) (hC : C ≤ g.toNat)
    (hgc : 1 ≤ gc)
    (st : Xstep (D_J code 0) s =
      if s.machineState.gasAvailable.toNat < mc then .error .OutOfGass
      else if (s.machineState.gasAvailable.subNat mc).toNat < gc then .error .OutOfGass
      else if ((s.machineState.gasAvailable.subNat mc).subNat gc).toNat + G <
          L ((s.machineState.gasAvailable.subNat mc).subNat gc).toNat then .error .OutOfGass
      else .ok (S, .none))
    (hSgas : S.machineState.gasAvailable = ((s.machineState.gasAvailable.subNat mc).subNat gc).subNat
      (L ((s.machineState.gasAvailable.subNat mc).subNat gc).toNat - G))
    (hcode : S.executionEnv.code = code) (hpc : S.machineState.pc = pc)
    (hstk : S.machineState.stack = stk) (hmem : S.machineState.memory = mem)
    (haw : S.machineState.activeWords = aw) (hrd : S.machineState.returnData = rd)
    (hacc : S.accountMap = σ') (hee : S.executionEnv = ee) (hσ₀ : S.σ₀ = s0.σ₀) :
    ∃ k' C', RD code ee g s0 pc stk mem aw rd σ' k' C' := by
  rw [ite3_or] at st
  have hfuel : g.toNat + 1 - k = (g.toNat - k) + 1 := by omega
  have hXP := hX.trans (hfuel.symm ▸ X_peel (f := g.toNat - k) st)
  split at hXP
  · exact ⟨k, C, Or.inl hXP⟩
  · rename_i hP
    simp only [not_or, not_lt] at hP
    obtain ⟨h1, h2, h3⟩ := hP
    have hgN : s.machineState.gasAvailable.toNat = g.toNat - C := by rw [hgas, Sat256.subNat_toNat]
    rw [Sat256.subNat_toNat] at h2
    simp only [Sat256.subNat_toNat] at h3
    have hL : ∀ n, L n ≤ n := fun n => by unfold L; omega
    refine ⟨k + 1, C + (mc + (gc + (L (g.toNat - C - mc - gc) - G))), Or.inr ⟨S, ?_, hcode, hpc, hstk,
      ?_, ?_, ?_, hmem, haw, hrd, hacc, hee, hσ₀⟩⟩
    · rw [hXP]; congr 1; omega
    · rw [hSgas, hgas, Sat256.subNat_subNat, Sat256.subNat_subNat, Sat256.subNat_subNat]
      congr 1
    · omega
    · have := hL (g.toNat - C - mc - gc)
      omega

set_option maxHeartbeats 2000000 in
theorem RD.create {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray} {σ : AccountMap} {k C : ℕ}
    {value offset size : UInt256} {t : List UInt256}
    (h : RD code ee g s0 pc (value :: offset :: size :: t) mem aw rdata σ k C)
    (hdec : decode code pc = some (.CREATE, .none))
    (hperm : ee.perm = true) (hsz : size.toNat ≤ 49152) (hov : t.length + 1 ≤ 1024) :
    ∃ (x : UInt256) (σ' : AccountMap) (rd' : ByteArray) (k' C' : ℕ),
      CreateStep ee s0.σ₀ σ value (mem.readWithPadding offset.toNat size.toNat) x σ' rd' ∧
      RD code ee g s0 (pc + ⟨1⟩) (x :: t) mem
        (UInt256.ofNat (MachineState.M aw.toNat offset.toNat size.toNat)) rd' σ' k' C' := by
  unfold RD at h
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀⟩
  · by_cases hn : ((σ.get? ee.codeOwner).getD default).nonce.toNat ≥ 2 ^ 64 - 1
    · exact ⟨⟨0⟩, σ, .empty, k, C, ⟨default, ⟨0⟩, Or.inl ⟨hn, rfl, rfl, rfl⟩⟩, Or.inl hoog⟩
    have hn' : ((σ.get? ee.codeOwner).getD default).nonce.toNat < 2 ^ 64 - 1 := by omega
    by_cases hD : value ≤ (σ.get? ee.codeOwner |>.option ⟨0⟩ (·.balance)) ∧ ee.depth < 1024 ∧
        (mem.readWithPadding offset.toNat size.toNat).size ≤ 49152
    · obtain ⟨⟨a, σ', g', A', z, o⟩, hres⟩ : ∃ res, Lambda (σ.insert ee.codeOwner
          {((σ.get? ee.codeOwner).getD default) with
            nonce := ((σ.get? ee.codeOwner).getD default).nonce + ⟨1⟩}) s0.σ₀ default ee.codeOwner
          ee.sender ⟨0⟩ (UInt256.ofNat ee.gasPrice) value (mem.readWithPadding offset.toNat size.toNat)
          ⟨ee.depth.val + 1, Nat.succ_lt_succ hD.2.1⟩ none ee.header ee.blobVersionedHashes ee.blocks
          ee.perm = res := ⟨_, rfl⟩
      exact ⟨_, σ', _, k, C, ⟨default, ⟨0⟩, Or.inr (Or.inr ⟨hn', hD, a, g', A', z, o, hres, rfl, rfl⟩)⟩,
        Or.inl hoog⟩
    · exact ⟨⟨0⟩, σ, .empty, k, C, ⟨default, ⟨0⟩, Or.inr (Or.inl ⟨hn', hD, rfl, rfl, rfl⟩)⟩, Or.inl hoog⟩
  · subst hacc hee hmem haw hpc hcode
    have st := step_create s hdec
    rw [hstk] at st
    have hovF : ((value :: offset :: size :: t).length - 3 + 1 > 1024) = False :=
      eq_false (by simp; omega)
    have hpF : (¬ s.executionEnv.perm = true) = False := eq_false (by simp [hperm])
    have hszF : (size > (⟨49152⟩ : UInt256)) = False := eq_false (by
      intro hc
      have : (⟨49152⟩ : UInt256).val < size.val := hc
      rw [Fin.lt_def] at this
      have h2 : (⟨49152⟩ : UInt256).val.val = 49152 := rfl
      unfold UInt256.toNat at hsz; omega)
    simp only [hovF, hpF, hszF, if_false] at st
    set mc := memoryExpansionCost s Operation.CREATE with hmc
    set gc := GasConstants.Gcreate + R size.toNat with hgc
    set g2 := (s.machineState.gasAvailable.subNat mc).subNat gc with hg2
    set σa := (s.accountMap.get? s.executionEnv.codeOwner).getD default with hσa
    set i := s.machineState.memory.readWithPadding offset.toNat size.toNat with hi
    have hgc1 : 1 ≤ gc := by rw [hgc]; simp [GasConstants.Gcreate]; omega
    by_cases hn : σa.nonce.toNat ≥ 2 ^ 64 - 1
    · rw [if_pos hn] at st
      simp only [true_or, if_true, Bool.false_eq_true, if_false] at st
      obtain ⟨k', C', hrd⟩ := rd_after_create hX hgas hk hC hgc1 st rfl rfl rfl rfl rfl rfl rfl rfl rfl hσ₀
      exact ⟨_, _, _, k', C', ⟨s.substate, ⟨0⟩, Or.inl ⟨hn, rfl, rfl, rfl⟩⟩, hrd⟩
    · rw [if_neg hn] at st
      have hn' : σa.nonce.toNat < 2 ^ 64 - 1 := by omega
      by_cases hD : value ≤ Option.option ⟨0⟩ (fun x => x.balance)
            (s.accountMap.get? s.executionEnv.codeOwner) ∧
          s.executionEnv.depth.val < 1024 ∧ i.size ≤ 49152
      · rw [dif_pos hD] at st
        generalize hres : Lambda (s.accountMap.insert s.executionEnv.codeOwner
            {σa with nonce := σa.nonce + ⟨1⟩}) s.σ₀ s.substate s.executionEnv.codeOwner
            s.executionEnv.sender (UInt256.ofNat (L g2.toNat)) (UInt256.ofNat s.executionEnv.gasPrice)
            value i ⟨s.executionEnv.depth.val + 1, Nat.succ_lt_succ hD.2.1⟩ none s.executionEnv.header
            s.executionEnv.blobVersionedHashes s.executionEnv.blocks s.executionEnv.perm = res at st
        obtain ⟨a, σ', g', A', z, o⟩ := res
        simp only at st
        have hx : (if z = false ∨ (s.executionEnv.depth : ℕ) = 1024 ∨
              value > Option.option ⟨0⟩ (fun x => x.balance)
                (s.accountMap.get? s.executionEnv.codeOwner) ∨ i.size > 49152
            then (⟨0⟩ : UInt256) else UInt256.ofNat a) = (if z then UInt256.ofNat a else ⟨0⟩) := by
          have h1 : ¬ ((s.executionEnv.depth : ℕ) = 1024) := by omega
          have h2 : ¬ (value > Option.option ⟨0⟩ (fun x => x.balance)
              (s.accountMap.get? s.executionEnv.codeOwner)) := fun hc =>
            Nat.lt_irrefl _ (Nat.lt_of_lt_of_le hc hD.1)
          have h3 : ¬ (i.size > 49152) := by omega
          cases z
          · rw [if_pos (Or.inl rfl)]; rfl
          · rw [if_neg (fun hc => by
              rcases hc with hc | hc | hc | hc
              · exact Bool.noConfusion hc
              · exact h1 hc
              · exact h2 hc
              · exact h3 hc)]
            rfl
        rw [hx] at st
        rw [hσ₀] at hres
        obtain ⟨k', C', hrd⟩ := rd_after_create hX hgas hk hC hgc1 st rfl rfl rfl rfl rfl rfl rfl rfl rfl hσ₀
        exact ⟨_, _, _, k', C', ⟨s.substate, UInt256.ofNat (L g2.toNat),
          Or.inr (Or.inr ⟨hn', hD, a, g', A', z, o, hres, rfl, rfl⟩)⟩, hrd⟩
      · rw [dif_neg hD] at st
        simp only [true_or, if_true, Bool.false_eq_true, if_false] at st
        obtain ⟨k', C', hrd⟩ := rd_after_create hX hgas hk hC hgc1 st rfl rfl rfl rfl rfl rfl rfl rfl rfl hσ₀
        exact ⟨_, _, _, k', C', ⟨s.substate, ⟨0⟩, Or.inr (Or.inl ⟨hn', hD, rfl, rfl, rfl⟩)⟩, hrd⟩

end BridgeEvm
