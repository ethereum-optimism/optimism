import ExporterEvm.Spec
import ExporterEvm.KernelRun

/-!
# Revert terminals that record the output

EquiVM's `RDrev` says only that a run reverts. The outcome classification of this proof needs
the revert *output*, so `RDrevP code g s0 P` is `RDrev` with the output constrained by `P`, and
`RD.revP` is EquiVM's `RD.rev` (copied, same proof) keeping the output
`mem.readWithPadding off len` that EVMLean's `REVERT` step returns.
-/

/-- Stack-overflow side conditions (`stk.length + n ≤ 1024`). -/
macro "stk_ov" : tactic => `(tactic| first
  | omega
  | (simp only [List.length_cons, List.length_nil] at *; omega)
  | (simp at *; omega))

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach Mem

/-- Halting-revert terminal with the output constrained by `P`: `X (g+1) … s0` runs out of gas,
    or reverts with an output `o` satisfying `P o`. -/
def RDrevP (code : ByteArray) (g : Sat256) (s0 : State) (P : ByteArray → Prop) : Prop :=
  X (g.toNat + 1) (D_J code 0) s0 = .error .OutOfGass
  ∨ ∃ g' o, X (g.toNat + 1) (D_J code 0) s0 = .ok (.revert g' o) ∧ P o

theorem RDrevP.mono {code : ByteArray} {g : Sat256} {s0 : State} {P Q : ByteArray → Prop}
    (h : RDrevP code g s0 P) (hPQ : ∀ o, P o → Q o) : RDrevP code g s0 Q := by
  rcases h with h | ⟨g', o, hX, hP⟩
  · exact Or.inl h
  · exact Or.inr ⟨g', o, hX, hPQ o hP⟩

private theorem terminalOOG' {code : ByteArray} {g : Sat256} {s0 s : State} {k C cost : ℕ}
    {res : Except ExecutionException (State × Option (HaltCause × ByteArray))}
    (hgas : s.machineState.gasAvailable = g.subNat C)
    (hstep : Xstep (D_J code 0) s
              = if s.machineState.gasAvailable.toNat < cost then .error .OutOfGass else res)
    (hk : k ≤ C) (hC : C ≤ g.toNat) (hOOG : g.toNat < C + cost)
    (hX : X (g.toNat + 1) (D_J code 0) s0 = X (g.toNat + 1 - k) (D_J code 0) s) :
    X (g.toNat + 1) (D_J code 0) s0 = .error .OutOfGass := by
  rw [hX]
  have hgg : s.machineState.gasAvailable.toNat < cost := by
    rw [hgas, Sat256.subNat_toNat]
    omega
  have hstepE : Xstep (D_J code 0) s = .error .OutOfGass := by rw [hstep, if_pos hgg]
  have hfuel : g.toNat + 1 - k = (g.toNat + 1 - (k + 1)) + 1 := by omega
  rw [hfuel]; exact Ethereum.EVM.Xstep_X_X_except _ s _ _ hstepE

/-- `REVERT`, keeping the output: `mem[off .. off+len]` as EVMLean reads it. -/
theorem RD.revP {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {k C : ℕ}
    {off len : UInt256} {t : List UInt256}
    (h : RD code ee g s0 pc (off :: len :: t) mem aw rdata acc k C)
    (hdec : decode code pc = some (.REVERT, .none))
    (hov : t.length ≤ 1024) :
    RDrevP code g s0 (fun o => o = mem.readWithPadding off.toNat len.toNat) := by
  unfold RD at h
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, _hrdata, _hacc, _hee⟩
  · exact Or.inl hoog
  · have hmcS : memoryExpansionCost s .REVERT = memExpansionCost aw off len := by
      simp only [memoryExpansionCost, memoryExpansionCost.μᵢ', haw, hstk,
        List.getElem!_cons_zero, List.getElem!_cons_succ, M, memExpansionCost]
    have st := revert_xstep hcode hpc hdec hstk hov
    rw [hmcS, hmem] at st
    by_cases gg : g.toNat < C + memExpansionCost aw off len
    · exact Or.inl (terminalOOG' hgas st hk hC gg hX)
    · exact Or.inr ⟨_, _, hX.trans (stepHaltRevert hgas st hk (by omega)), rfl⟩

/-- `RDrevP` at a top-level run gives the `Ξ` result. -/
theorem RDrevP.xiResult {σ σ₀ A I} {g : UInt256} {code : ByteArray} {P : ByteArray → Prop}
    (hcode : I.code = code)
    (h : RDrevP code (Sat256.ofUInt256 g) (initState σ σ₀ (Sat256.ofUInt256 g) A I) P) :
    Ξ σ σ₀ g A I = .error .OutOfGass
    ∨ ∃ (g' : UInt256) (o : ByteArray), Ξ σ σ₀ g A I = .ok (.revert g' o) ∧ P o := by
  rcases h with hoog | ⟨g', o, hX, hP⟩
  · exact Or.inl (Xi_error_of_X (by rw [hcode]; exact hoog))
  · exact Or.inr ⟨g', o, Xi_revert_of_X (by rw [hcode]; exact hX), hP⟩

/-- A zero-length read is empty. -/
theorem readWithPadding_zero (m : ByteArray) (a : ℕ) : m.readWithPadding a 0 = ByteArray.empty := by
  unfold ByteArray.readWithPadding
  rw [if_neg (by norm_num)]
  unfold ByteArray.readWithoutPadding
  split <;> (apply ByteArray.ext; simp [ByteArray.zeroes])

/-- solc's `revert(0, 0)` (`PUSH1 0; DUP1; REVERT`) reverts with empty output. -/
theorem RD.rev00 {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {stk : List UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {k C : ℕ}
    (h : RD code ee g s0 pc stk mem aw rdata acc k C)
    (h1 : decode code pc = some (.Push .PUSH1, some (UInt256.ofNat 0, 1)))
    (h2 : decode code (pc + UInt256.ofNat 2) = some (.DUP1, .none))
    (h3 : decode code (pc + UInt256.ofNat 2 + ⟨1⟩) = some (.REVERT, .none))
    (hov : stk.length + 2 ≤ 1024) :
    RDrevP code g s0 (fun o => o = ByteArray.empty) := by
  have r1 := h.push1 (UInt256.ofNat 0) h1 (by omega)
  have r2 := r1.dup1 h2 (by omega)
  refine (RD.revP r2 h3 (by omega)).mono ?_
  intro o ho
  rw [ho]
  exact readWithPadding_zero _ _

end ExporterEvm
