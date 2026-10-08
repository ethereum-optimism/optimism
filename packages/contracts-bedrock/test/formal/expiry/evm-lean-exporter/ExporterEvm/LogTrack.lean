import ExporterEvm.CallMem

/-!
# Tracking the last log entry (`RDL`)

EquiVM's `RD` invariant does not record the substate, so a `LOG3` it steps over leaves no trace in
the conclusion. `RDL … e …` is `RD` plus "the reached state's log series ends with the entry
`e`". `RDL.log3` steps a `LOG3` off an `RD` cursor and records the entry it appends (EVMLean's
`stLog3`); the other combinators are EquiVM's (same proofs, copied from `Reasoning/Reach.lean` at
`b0e9d55a`) with the log series carried along: none of these opcodes touches the substate, so the
extra conjunct is preserved definitionally. `RDretL` is the matching success terminal.
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach Mem

/-- `RD` whose reached state's log series ends with `e`. -/
def RDL (code : ByteArray) (ee : ExecutionEnv) (g : Sat256) (s0 : State)
    (pc : UInt256) (stk : List UInt256) (mem : ByteArray) (aw : UInt256) (rdata : ByteArray)
    (acc : AccountMap) (e : LogEntry) (k C : ℕ) : Prop :=
  X (g.toNat + 1) (D_J code 0) s0 = .error .OutOfGass
  ∨ ∃ s : State,
      X (g.toNat + 1) (D_J code 0) s0 = X (g.toNat + 1 - k) (D_J code 0) s
    ∧ s.executionEnv.code = code
    ∧ s.machineState.pc = pc
    ∧ s.machineState.stack = stk
    ∧ s.machineState.gasAvailable = g.subNat C
    ∧ k ≤ C ∧ C ≤ g.toNat
    ∧ s.machineState.memory = mem ∧ s.machineState.activeWords = aw ∧ s.machineState.returnData = rdata
    ∧ s.accountMap = acc
    ∧ s.executionEnv = ee
    ∧ s.σ₀ = s0.σ₀
    ∧ ∃ L : LogSeries, s.substate.logSeries = L.push e

/-- Success terminal: the run returns `o` with account map `acc`, and the final substate's log
    series ends with `e`. -/
def RDretL (code : ByteArray) (g : Sat256) (s0 : State) (acc : AccountMap) (o : ByteArray)
    (e : LogEntry) : Prop :=
  X (g.toNat + 1) (D_J code 0) s0 = .error .OutOfGass
  ∨ ∃ s', X (g.toNat + 1) (D_J code 0) s0 = .ok (.success s' o) ∧ s'.accountMap = acc ∧
      ∃ L : LogSeries, s'.substate.logSeries = L.push e

theorem RDretL.toRDret {code : ByteArray} {g : Sat256} {s0 : State} {acc : AccountMap}
    {o : ByteArray} {e : LogEntry} (h : RDretL code g s0 acc o e) : RDret code g s0 acc o := by
  rcases h with h | ⟨s', hX, hacc, _⟩
  · exact Or.inl h
  · exact Or.inr ⟨s', hX, hacc⟩

theorem RDL.normalizePC {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc pc' : UInt256} {stk : List UInt256} {mem : ByteArray} {aw : UInt256}
    {rdata : ByteArray} {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    (h : RDL code ee g s0 pc stk mem aw rdata acc lg k C) (hpc : pc = pc') :
    RDL code ee g s0 pc' stk mem aw rdata acc lg k C := by
  subst pc'
  exact h

/-- `LOG3` from an `RD` cursor: the appended entry is the executing account, the three topics
    and `mem[a .. a+b]`. -/
theorem RDL.log3 {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {k C : ℕ}
    {a b c d e : UInt256} {t : List UInt256}
    (h : RD code ee g s0 pc (a :: b :: c :: d :: e :: t) mem aw rdata acc k C)
    (hdec : decode code pc = some (.LOG3, .none)) (hperm : ee.perm = true)
    (hov : t.length ≤ 1024) :
    ∃ k' C', RDL code ee g s0 (pc + ⟨1⟩) t mem (M aw a b) rdata acc
      ⟨ee.codeOwner, #[c, d, e], mem.readWithPadding a.toNat b.toNat⟩ k' C' := by
  unfold RD at h
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀⟩
  · exact ⟨k, C, Or.inl hoog⟩
  · have hmcS : memoryExpansionCost s .LOG3 = memExpansionCost aw a b := by
      simp only [memoryExpansionCost, memoryExpansionCost.μᵢ', haw, hstk,
        List.getElem!_cons_zero, List.getElem!_cons_succ, M, memExpansionCost]
    have hperms : s.executionEnv.perm = true := by rw [hee]; exact hperm
    have st := log3_xstep hcode hpc hdec hperms hstk hov
    rw [hmcS] at st
    by_cases gg : g.toNat < C + (memExpansionCost aw a b
        + (GasConstants.Glog + GasConstants.Glogdata * b.toNat + 3 * GasConstants.Glogtopic))
    · exact ⟨k, C, Or.inl (hX.trans (stepOOG hgas st hk hC (by omega)))⟩
    · refine ⟨k + 1, C + (memExpansionCost aw a b
        + (GasConstants.Glog + GasConstants.Glogdata * b.toNat + 3 * GasConstants.Glogtopic)),
        Or.inr ⟨stLog3 s a b c d e t,
        hX.trans (stepContinue hgas st hk (Nat.not_lt.mp gg)), ?_, ?_, ?_, ?_,
          (by have : 1 ≤ GasConstants.Glog := (by decide); omega), by omega,
          ?_, ?_, ?_, ?_, ?_, ?_, ?_⟩⟩
      · simp only [stLog3]; exact hcode
      · simp only [stLog3]; rw [hpc]
      · simp only [stLog3]
      · simp only [stLog3, hmcS]
        rw [hgas, Sat256.subNat_sub_add_of_sub_sub, Sat256.subNat_sub_add_of_sub_sub]
      · simp only [stLog3]; exact hmem
      · simp only [stLog3]; rw [haw]
      · simp only [stLog3]; exact hrdata
      · simp only [stLog3]; exact hacc
      · simp only [stLog3]; exact hee
      · simp only [stLog3]; exact hσ₀
      · exact ⟨s.substate.logSeries, by simp only [stLog3]; rw [hee, hmem]⟩

theorem RDL.stepSwap {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    {stkIn stkOut : List UInt256}
    (h : RDL code ee g s0 pc stkIn mem aw rdata acc lg k C)
    (hstep : ∀ s : State, s.executionEnv.code = code → s.machineState.pc = pc →
        s.machineState.stack = stkIn →
        Xstep (D_J code 0) s =
          if s.machineState.gasAvailable.toNat < 3 then .error .OutOfGass
          else .ok (stSwap s stkOut, .none)) :
    RDL code ee g s0 (pc + ⟨1⟩) stkOut mem aw rdata acc lg (k + 1) (C + 3) := by
  unfold RDL at h ⊢
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀, hL⟩
  · exact Or.inl hoog
  · have st := hstep s hcode hpc hstk
    by_cases gg : g.toNat < C + 3
    · exact Or.inl (hX.trans (stepOOG hgas st hk hC (by omega)))
    · refine Or.inr ⟨stSwap s stkOut,
        hX.trans (stepContinue hgas st hk (Nat.not_lt.mp gg)), ?_, ?_, ?_, ?_, by omega, by omega,
        ?_, ?_, ?_, ?_, ?_, ?_, hL⟩
      · simp only [stSwap]; exact hcode
      · simp only [stSwap]; rw [hpc]
      · rfl
      · simp only [stSwap]; rw [hgas, Sat256.subNat_sub_add_of_sub_sub]
      · simp only [stSwap]; exact hmem
      · simp only [stSwap]; exact haw
      · simp only [stSwap]; exact hrdata
      · simp only [stSwap]; exact hacc
      · exact hee
      · exact hσ₀

theorem RDL.stepBinop {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    {a b res : UInt256} {t : List UInt256}
    (h : RDL code ee g s0 pc (a :: b :: t) mem aw rdata acc lg k C)
    (hstep : ∀ s : State, s.executionEnv.code = code → s.machineState.pc = pc →
        s.machineState.stack = a :: b :: t →
        Xstep (D_J code 0) s =
          if s.machineState.gasAvailable.toNat < 3 then .error .OutOfGass
          else .ok (stBinop s res t, .none)) :
    RDL code ee g s0 (pc + ⟨1⟩) (res :: t) mem aw rdata acc lg (k + 1) (C + 3) := by
  unfold RDL at h ⊢
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀, hL⟩
  · exact Or.inl hoog
  · have st := hstep s hcode hpc hstk
    by_cases gg : g.toNat < C + 3
    · exact Or.inl (hX.trans (stepOOG hgas st hk hC (by omega)))
    · refine Or.inr ⟨stBinop s res t,
        hX.trans (stepContinue hgas st hk (Nat.not_lt.mp gg)), ?_, ?_, ?_, ?_, by omega, by omega,
        ?_, ?_, ?_, ?_, ?_, ?_, hL⟩
      · simp only [stBinop]; exact hcode
      · simp only [stBinop]; rw [hpc]
      · rfl
      · simp only [stBinop]; rw [hgas, Sat256.subNat_sub_add_of_sub_sub]
      · simp only [stBinop]; exact hmem
      · simp only [stBinop]; exact haw
      · simp only [stBinop]; exact hrdata
      · simp only [stBinop]; exact hacc
      · exact hee
      · exact hσ₀

theorem RDL.pop {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    {a : UInt256} {t : List UInt256}
    (h : RDL code ee g s0 pc (a :: t) mem aw rdata acc lg k C)
    (hdec : decode code pc = some (.POP, .none))
    (hov : t.length ≤ 1024) :
    RDL code ee g s0 (pc + ⟨1⟩) t mem aw rdata acc lg (k + 1) (C + 2) := by
  unfold RDL at h ⊢
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀, hL⟩
  · exact Or.inl hoog
  · have st := pop_xstep hcode hpc hdec hstk hov
    by_cases gg : g.toNat < C + 2
    · exact Or.inl (hX.trans (stepOOG hgas st hk hC (by omega)))
    · refine Or.inr ⟨stPop s t,
        hX.trans (stepContinue hgas st hk (Nat.not_lt.mp gg)), ?_, ?_, ?_, ?_, by omega, by omega,
        ?_, ?_, ?_, ?_, ?_, ?_, hL⟩
      · simp only [stPop]; exact hcode
      · simp only [stPop]; rw [hpc]
      · rfl
      · simp only [stPop]; rw [hgas, Sat256.subNat_sub_add_of_sub_sub]
      · simp only [stPop]; exact hmem
      · simp only [stPop]; exact haw
      · simp only [stPop]; exact hrdata
      · simp only [stPop]; exact hacc
      · exact hee
      · exact hσ₀

theorem RDL.jump {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    {a : UInt256} {t : List UInt256}
    (h : RDL code ee g s0 pc (a :: t) mem aw rdata acc lg k C)
    (hdec : decode code pc = some (.JUMP, .none))
    (hjd : (D_J code 0).contains a = true)
    (hov : t.length ≤ 1024) :
    RDL code ee g s0 a t mem aw rdata acc lg (k + 1) (C + 8) := by
  unfold RDL at h ⊢
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀, hL⟩
  · exact Or.inl hoog
  · have st := jump_xstep hcode hpc hdec hstk hjd hov
    by_cases gg : g.toNat < C + 8
    · exact Or.inl (hX.trans (stepOOG hgas st hk hC (by omega)))
    · refine Or.inr ⟨stJump s a t,
        hX.trans (stepContinue hgas st hk (Nat.not_lt.mp gg)), ?_, ?_, ?_, ?_, by omega, by omega,
        ?_, ?_, ?_, ?_, ?_, ?_, hL⟩
      · simp only [stJump]; exact hcode
      · rfl
      · rfl
      · simp only [stJump]; rw [hgas, Sat256.subNat_sub_add_of_sub_sub]
      · simp only [stJump]; exact hmem
      · simp only [stJump]; exact haw
      · simp only [stJump]; exact hrdata
      · simp only [stJump]; exact hacc
      · exact hee
      · exact hσ₀

theorem RDL.jumpdest {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {stk : List UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    (h : RDL code ee g s0 pc stk mem aw rdata acc lg k C)
    (hdec : decode code pc = some (.JUMPDEST, .none))
    (hov : stk.length ≤ 1024) :
    RDL code ee g s0 (pc + ⟨1⟩) stk mem aw rdata acc lg (k + 1) (C + 1) := by
  unfold RDL at h ⊢
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀, hL⟩
  · exact Or.inl hoog
  · have st := jumpdest_xstep hcode hpc hdec (by rw [hstk]; exact hov)
    by_cases gg : g.toNat < C + 1
    · exact Or.inl (hX.trans (stepOOG hgas st hk hC (by omega)))
    · refine Or.inr ⟨stJumpdest s,
        hX.trans (stepContinue hgas st hk (Nat.not_lt.mp gg)), ?_, ?_, ?_, ?_, by omega, by omega,
        ?_, ?_, ?_, ?_, ?_, ?_, hL⟩
      · simp only [stJumpdest]; exact hcode
      · simp only [stJumpdest]; rw [hpc]
      · simp only [stJumpdest]; exact hstk
      · simp only [stJumpdest]; rw [hgas, Sat256.subNat_sub_add_of_sub_sub]
      · simp only [stJumpdest]; exact hmem
      · simp only [stJumpdest]; exact haw
      · simp only [stJumpdest]; exact hrdata
      · simp only [stJumpdest]; exact hacc
      · exact hee
      · exact hσ₀

theorem RDL.push1 {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {stk : List UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    (h : RDL code ee g s0 pc stk mem aw rdata acc lg k C) (argv : UInt256)
    (hdec : decode code pc = some (.Push .PUSH1, some (argv, 1)))
    (hov : stk.length + 1 ≤ 1024) :
    RDL code ee g s0 (pc + UInt256.ofNat 2) (argv :: stk) mem aw rdata acc lg (k + 1) (C + 3) := by
  unfold RDL at h ⊢
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀, hL⟩
  · exact Or.inl hoog
  · have st := push1_xstep hcode hpc hdec hstk hov
    by_cases gg : g.toNat < C + 3
    · exact Or.inl (hX.trans (stepOOG hgas st hk hC (by omega)))
    · refine Or.inr ⟨stPush1 s argv,
        hX.trans (stepContinue hgas st hk (Nat.not_lt.mp gg)), ?_, ?_, ?_, ?_, by omega, by omega,
        ?_, ?_, ?_, ?_, ?_, ?_, hL⟩
      · simp only [stPush1]; exact hcode
      · simp only [stPush1]; rw [hpc]
      · simp only [stPush1]; rw [hstk]
      · simp only [stPush1]; rw [hgas, Sat256.subNat_sub_add_of_sub_sub]
      · simp only [stPush1]; exact hmem
      · simp only [stPush1]; exact haw
      · simp only [stPush1]; exact hrdata
      · simp only [stPush1]; exact hacc
      · exact hee
      · exact hσ₀

theorem RDL.dup1 {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    {a : UInt256} {t : List UInt256}
    (h : RDL code ee g s0 pc (a :: t) mem aw rdata acc lg k C)
    (hdec : decode code pc = some (.DUP1, .none)) (hov : t.length + 2 ≤ 1024) :
    RDL code ee g s0 (pc + ⟨1⟩) (a :: a :: t) mem aw rdata acc lg (k + 1) (C + 3) := by
  unfold RDL at h ⊢
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀, hL⟩
  · exact Or.inl hoog
  · have st := dup1_xstep hcode hpc hdec hstk (by simp only [List.length_cons]; omega)
    by_cases gg : g.toNat < C + 3
    · exact Or.inl (hX.trans (stepOOG hgas st hk hC (by omega)))
    · refine Or.inr ⟨stDup1 s a t,
        hX.trans (stepContinue hgas st hk (Nat.not_lt.mp gg)), ?_, ?_, ?_, ?_, by omega, by omega,
        ?_, ?_, ?_, ?_, ?_, ?_, hL⟩
      · simp only [stDup1]; exact hcode
      · simp only [stDup1]; rw [hpc]
      · rfl
      · simp only [stDup1]; rw [hgas, Sat256.subNat_sub_add_of_sub_sub]
      · simp only [stDup1]; exact hmem
      · simp only [stDup1]; exact haw
      · simp only [stDup1]; exact hrdata
      · simp only [stDup1]; exact hacc
      · exact hee
      · exact hσ₀

theorem RDL.genMstore {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    {a b : UInt256} {t : List UInt256}
    (h : RDL code ee g s0 pc (a :: b :: t) mem aw rdata acc lg k C)
    (hdec : decode code pc = some (.MSTORE, .none)) (hov : t.length ≤ 1024) :
    RDL code ee g s0 (pc + ⟨1⟩) t (b.toByteArray.write 0 mem a.toNat 32) (M aw a ⟨32⟩) rdata acc lg
      (k + 1) (C + (memExpansionCost aw a ⟨32⟩ + 3)) := by
  unfold RDL at h ⊢
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀, hL⟩
  · exact Or.inl hoog
  · have hmcS : memoryExpansionCost s .MSTORE = memExpansionCost aw a ⟨32⟩ := by
      simp only [memoryExpansionCost, memoryExpansionCost.μᵢ', haw, hstk,
        List.getElem!_cons_zero, M, memExpansionCost]; rfl
    have st := mstore_xstep hcode hpc hdec hstk hov
    rw [hmcS] at st
    by_cases gg : g.toNat < C + (memExpansionCost aw a ⟨32⟩ + 3)
    · exact Or.inl (hX.trans (stepOOG hgas st hk hC (by omega)))
    · refine Or.inr ⟨stMStore s a b t,
        hX.trans (stepContinue hgas st hk (Nat.not_lt.mp gg)), ?_, ?_, ?_, ?_, by omega, by omega,
        ?_, ?_, ?_, ?_, ?_, ?_, hL⟩
      · simp only [stMStore]; exact hcode
      · simp only [stMStore]; rw [hpc]
      · rfl
      · simp only [stMStore, hmcS]
        rw [hgas, Sat256.subNat_sub_add_of_sub_sub, Sat256.subNat_sub_add_of_sub_sub]
      · simp only [stMStore]; rw [hmem]
      · simp only [stMStore]; rw [haw]; rfl
      · simp only [stMStore]; exact hrdata
      · simp only [stMStore]; exact hacc
      · exact hee
      · exact hσ₀

theorem RDL.genMload {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    {a : UInt256} {t : List UInt256}
    (h : RDL code ee g s0 pc (a :: t) mem aw rdata acc lg k C)
    (hdec : decode code pc = some (.MLOAD, .none)) (hov : t.length + 1 ≤ 1024) :
    RDL code ee g s0 (pc + ⟨1⟩) (memLoad a mem :: t) mem (M aw a ⟨32⟩) rdata acc lg (k + 1)
      (C + (memExpansionCost aw a ⟨32⟩ + 3)) := by
  unfold RDL at h ⊢
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, hrdata, hacc, hee, hσ₀, hL⟩
  · exact Or.inl hoog
  · have hmcS : memoryExpansionCost s .MLOAD = memExpansionCost aw a ⟨32⟩ := by
      simp only [memoryExpansionCost, memoryExpansionCost.μᵢ', haw, hstk,
        List.getElem!_cons_zero, M, memExpansionCost]; rfl
    have st := mload_xstep hcode hpc hdec hstk hov
    rw [hmcS] at st
    by_cases gg : g.toNat < C + (memExpansionCost aw a ⟨32⟩ + 3)
    · exact Or.inl (hX.trans (stepOOG hgas st hk hC (by omega)))
    · refine Or.inr ⟨stMLoad s a t,
        hX.trans (stepContinue hgas st hk (Nat.not_lt.mp gg)), ?_, ?_, ?_, ?_, by omega, by omega,
        ?_, ?_, ?_, ?_, ?_, ?_, hL⟩
      · simp only [stMLoad]; exact hcode
      · simp only [stMLoad]; rw [hpc]
      · simp only [stMLoad]; rw [hmem]; rfl
      · simp only [stMLoad, hmcS]
        rw [hgas, Sat256.subNat_sub_add_of_sub_sub, Sat256.subNat_sub_add_of_sub_sub]
      · simp only [stMLoad]; rw [hmem]
      · simp only [stMLoad]; rw [haw]; rfl
      · simp only [stMLoad]; exact hrdata
      · simp only [stMLoad]; exact hacc
      · exact hee
      · exact hσ₀

/-- `RETURN` from an `RDL` cursor. -/
theorem RDL.ret {code : ByteArray} {ee : ExecutionEnv} {g : Sat256} {s0 : State}
    {pc : UInt256} {mem : ByteArray} {aw : UInt256} {rdata : ByteArray}
    {acc : AccountMap} {lg : LogEntry} {k C : ℕ}
    {off len : UInt256} {t : List UInt256}
    (h : RDL code ee g s0 pc (off :: len :: t) mem aw rdata acc lg k C)
    (hdec : decode code pc = some (.RETURN, .none))
    (hov : t.length ≤ 1024) :
    RDretL code g s0 acc (mem.readWithPadding off.toNat len.toNat) lg := by
  unfold RDL at h
  rcases h with hoog | ⟨s, hX, hcode, hpc, hstk, hgas, hk, hC, hmem, haw, _hrdata, hacc, _hee, _hσ₀, hL⟩
  · exact Or.inl hoog
  · have hmcS : memoryExpansionCost s .RETURN = memExpansionCost aw off len := by
      simp only [memoryExpansionCost, memoryExpansionCost.μᵢ', haw, hstk,
        List.getElem!_cons_zero, List.getElem!_cons_succ, M, memExpansionCost]
    have st := return_xstep hcode hpc hdec hstk hov
    rw [hmcS, hmem] at st
    by_cases gg : g.toNat < C + memExpansionCost aw off len
    · left
      rw [hX]
      have hgg : s.machineState.gasAvailable.toNat < memExpansionCost aw off len := by
        rw [hgas, Sat256.subNat_toNat]; omega
      have hstepE : Xstep (D_J code 0) s = .error .OutOfGass := by rw [st, if_pos hgg]
      have hfuel : g.toNat + 1 - k = (g.toNat + 1 - (k + 1)) + 1 := by omega
      rw [hfuel]; exact Ethereum.EVM.Xstep_X_X_except _ s _ _ hstepE
    · exact Or.inr ⟨stReturn s off len t, hX.trans (stepHaltSuccess hgas st hk (by omega)),
        by simp only [stReturn]; exact hacc, hL⟩

/-- `RDretL` at a top-level run gives the `Ξ` result with the final substate's log series. -/
theorem rdretL_xi {σ σ₀ acc : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256}
    {code o : ByteArray} {e : LogEntry} (hcode : I.code = code)
    (h : RDretL code (Sat256.ofUInt256 g) (initState σ σ₀ (Sat256.ofUInt256 g) A I) acc o e) :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    ∃ (g' : UInt256) (A' : Substate), Ξ σ σ₀ g A I = .ok (.success (acc, g', A') o) ∧
      ∃ L : LogSeries, A'.logSeries = L.push e := by
  rcases h with hoog | ⟨s, hX, hacc, hL⟩
  · left
    exact Xi_error_of_X (by rw [hcode]; exact hoog)
  · right
    have hxi := Xi_success_of_X (σ := σ) (σ₀ := σ₀) (A := A) (I := I) (g := g)
      (by rw [hcode]; exact hX)
    rw [hacc] at hxi
    exact ⟨_, _, hxi, hL⟩

end ExporterEvm
