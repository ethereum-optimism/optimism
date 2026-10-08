import L1cdmEvm.Compose
import L1cdmEvm.Concrete
import L1cdmEvm.KernelRun

/-!
# The summary hypotheses are satisfiable (non-vacuity of `Summaries`)

`relay_success`/`relay_deposit` assume `Summaries σ σ₀ I v`: one `ReturnsWord` hypothesis per view
call. This file proves them, symbolically, for the mock contracts of `Concrete.lean`: the generic
mock `PUSH0 CALLDATALOAD PUSH1 0xe0 SHR SLOAD PUSH0 MSTORE PUSH1 0x20 PUSH0 RETURN` returns exactly
the 32-byte word `SLOAD(selector)` whenever it succeeds (from any account map with the same storage
and code, with any gas and substate). Hence `concrete_summaries`: in `Concrete.world .ok`, the world
where `success_reachable` shows the call succeeding, all seven summaries hold with the values the
mocks are programmed with. So the theorems' hypotheses are not contradictory on that witness.
-/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem L1cdmEvm.Concrete

/-- The word the generic mock returns when run as `I`. -/
def mockWord (σ : AccountMap) (I : ExecutionEnv) : UInt256 :=
  σ.get? I.codeOwner |>.option ⟨0⟩ (fun ac => ac.storage.getD
    (UInt256.shiftRight (uInt256OfByteArray (I.calldata.readBytes (⟨0⟩ : UInt256).toNat 32))
      (UInt256.ofNat 224)) ⟨0⟩)

theorem mock_run {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    (hcode : I.code = mockCode) :
    RDret mockCode g (initState σ σ₀ g A I) σ (UInt256.toByteArray (mockWord σ I)) := by
  have r0 := RD.initState (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode
  have r1 := kevm_run r0 with [push0]
  have r2 := RD.calldataload r1 (by evm_kdecide) (by evm_ov)
  have r3 := kevm_run r2 with [push1 (UInt256.ofNat 224), shr]
  obtain ⟨k4, C4, r4⟩ := RD.sload r3 (by evm_kdecide) (by evm_ov)
  have r5 := kevm_run r4 with [push0]
  have r6 := RD.genMstore r5 (by evm_kdecide) (by evm_ov)
  have r7 := kevm_run r6 with [push1 (UInt256.ofNat 32), push0]
  have hF := frame_mstore_var (frame_nil [mockWord σ I] ByteArray.empty 0) 0 (mockWord σ I) rfl 0
  have hrd := frame_read_eq hF 0 32 (by norm_num) (vsyms 0) (by decide +kernel) (by decide +kernel)
  rw [Mem_vsyms] at hrd
  exact RD.ret _ _ r7 (by evm_kdecide) rfl (by simpa using hrd) (by evm_ov)

/-- A successful value-0 call into code `c` (any caller) is a successful `Ξ` run of `c` on the
    unchanged account map, in the frame `Θ` builds. -/
theorem theta_call_success {σc σ₀ σ' : AccountMap} {A_in A' : Substate} {s o r : AccountAddress}
    {c cd out : ByteArray} {callGas g' p : UInt256} {e : Fin 1025} {H : BlockHeader}
    {bvh : List ByteArray} {bl : ProcessedBlocks} {w : Bool}
    (hr : ∃ acc, σc.get? r = some acc)
    (hΘ : (σ', g', A', true, out) = Θ σc σ₀ A_in s o r (ToExecute.Code c) callGas p ⟨0⟩ ⟨0⟩ cd e H bvh
      bl w) :
    ∃ A'', Ξ σc σ₀ callGas A_in
      { codeOwner := r, sender := o, gasPrice := p.toNat, calldata := cd, source := s,
        weiValue := ⟨0⟩, depth := e, perm := w, code := c, header := H, blobVersionedHashes := bvh,
        blocks := bl } = .ok (.success (σ', g', A'') out) := by
  obtain ⟨acc, hacc⟩ := hr
  unfold Θ at hΘ
  simp only [hacc, add_zero'] at hΘ
  have e1 : σc.insert r { acc with balance := acc.balance } = σc := insert_same hacc
  simp only [e1] at hΘ
  cases hs : σc.get? s with
  | none =>
    simp only [hs] at hΘ
    generalize hR : Ξ σc σ₀ callGas A_in _ = R at hΘ
    cases R with
    | error e => simp at hΘ
    | ok res =>
      cases res with
      | revert g2 o2 => simp at hΘ
      | success tr o2 =>
        obtain ⟨a, b, c'⟩ := tr
        by_cases ha : (a == ∅) = true
        · simp [ha] at hΘ
        · simp only [ha, Prod.mk.injEq] at hΘ
          obtain ⟨h1, h2, h3, _, h5⟩ := hΘ
          subst h1 h2 h5
          exact ⟨c', rfl⟩
  | some a2 =>
    have e2 : σc.insert s { a2 with balance := a2.balance } = σc := insert_same hs
    simp only [hs, sub_zero', e2] at hΘ
    generalize hR : Ξ σc σ₀ callGas A_in _ = R at hΘ
    cases R with
    | error e => simp at hΘ
    | ok res =>
      cases res with
      | revert g2 o2 => simp at hΘ
      | success tr o2 =>
        obtain ⟨a, b, c'⟩ := tr
        by_cases ha : (a == ∅) = true
        · simp [ha] at hΘ
        · simp only [ha, Prod.mk.injEq] at hΘ
          obtain ⟨h1, h2, h3, _, h5⟩ := hΘ
          subst h1 h2 h5
          exact ⟨c', rfl⟩

/-- The selector a mock dispatches on. -/
def cdSel (cd : ByteArray) : UInt256 :=
  UInt256.shiftRight (uInt256OfByteArray (cd.readBytes (⟨0⟩ : UInt256).toNat 32)) (UInt256.ofNat 224)

/-- The portal mock, on any selector other than `depositTransaction`, behaves like the generic mock. -/
theorem portal_run {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    (hcode : I.code = portalCode) (hsel : cdSel I.calldata ≠ UInt256.ofNat 0xe9e05c42) :
    RDret portalCode g (initState σ σ₀ g A I) σ (UInt256.toByteArray (mockWord σ I)) := by
  have r0 := RD.initState (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode
  have r1 := kevm_run r0 with [push0]
  have r2 := RD.calldataload r1 (by evm_kdecide) (by evm_ov)
  have r3 := kevm_run r2 with [push1 (UInt256.ofNat 224), shr, dup1, push4 (UInt256.ofNat 0xe9e05c42), eq,
    push1 (UInt256.ofNat 22)]
  have hb : UInt256.eq (UInt256.ofNat 0xe9e05c42) (cdSel I.calldata) = ⟨0⟩ :=
    Words.eq_eq0.mpr (Ne.symm hsel)
  have r4 := RD.jumpiNT r3 (by evm_kdecide) hb (by evm_ov)
  obtain ⟨k5, C5, r5⟩ := RD.sload r4 (by evm_kdecide) (by evm_ov)
  have r6 := kevm_run r5 with [push0]
  have r7 := RD.genMstore r6 (by evm_kdecide) (by evm_ov)
  have r8 := kevm_run r7 with [push1 (UInt256.ofNat 32), push0]
  have hF := frame_mstore_var (frame_nil [mockWord σ I] ByteArray.empty 0) 0 (mockWord σ I) rfl 0
  have hrd := frame_read_eq hF 0 32 (by norm_num) (vsyms 0) (by decide +kernel) (by decide +kernel)
  rw [Mem_vsyms] at hrd
  exact RD.ret _ _ r8 (by evm_kdecide) rfl (by simpa using hrd) (by evm_ov)

/-- **The summary holds for a mock.** If `target` holds `code` (the generic mock, or the portal
    mock on a selector other than `depositTransaction`) and its storage slot `cdSel cd` holds `w`,
    then `ReturnsWord σ σ₀ I target cd w`. -/
theorem mock_returnsWord {σ σ₀ : AccountMap} {I : ExecutionEnv} {target : AccountAddress}
    {cd : ByteArray} {w : UInt256} {code : ByteArray}
    (hrun : ∀ (σx σ₀x : AccountMap) (Ax : Substate) (Ix : ExecutionEnv) (gx : Sat256),
      Ix.code = code → Ix.calldata = cd →
        RDret code gx (initState σx σ₀x gx Ax Ix) σx (UInt256.toByteArray (mockWord σx Ix)))
    (hne : code.size ≠ 0) (hcode : (σ.getD target default).code = code) (hnp : target ∉ π)
    (hval : storageWord σ target (cdSel cd) = w) :
    ReturnsWord σ σ₀ I target cd w := by
  intro σc σ' z o hst hcd hcall hz
  subst hz
  obtain ⟨A_in, callGas, g', A', hΘ⟩ := hcall
  have hcodec : (σc.getD target default).code = code := by rw [← hcd target]; exact hcode
  rw [toExecute_code hne hnp hcodec] at hΘ
  obtain ⟨acc, hacc, _⟩ := getD_code_get hne hcodec
  obtain ⟨A'', hX⟩ := theta_call_success ⟨acc, hacc⟩ hΘ
  have hr := hrun σc σ₀ A_in (ExecutionEnv.mk target I.sender
    (AccountAddress.ofUInt256 (UInt256.ofNat I.codeOwner)) ⟨0⟩ cd code (UInt256.ofNat I.gasPrice).toNat
    I.header (I.depth + 1) false I.blobVersionedHashes I.blocks) (Sat256.ofUInt256 callGas) rfl rfl
  rcases RDret.xiResult (code := code) rfl hr with hoog | ⟨g2, A2, hX2⟩
  · rw [show (Sat256.ofUInt256 callGas).toUInt256 = callGas from rfl, hX] at hoog; cases hoog
  · rw [show (Sat256.ofUInt256 callGas).toUInt256 = callGas from rfl, hX] at hX2
    cases hX2
    have hmw : ∀ Ix : ExecutionEnv, Ix.codeOwner = target → Ix.calldata = cd → mockWord σc Ix = w := by
      intro Ix h1 h2
      unfold mockWord
      rw [h1, h2, optWord_eq, storageWord_eq_of_storageEq hst]
      exact hval
    rw [hmw _ rfl rfl]
    exact ⟨by rw [toByteArray_size]; norm_num, fun _ =>
      List.take_of_length_le (le_of_eq (by rw [Array.length_toList]; exact toByteArray_size w))⟩

/-- **Non-vacuity of the hypotheses.** In the concrete world where `Concrete.success_reachable`
    shows `relayUndeliveredMessage` succeeding, all seven summaries hold, with the values the mocks
    return. -/
theorem concrete_summaries :
    Summaries (world .ok) (world .ok) env
      ⟨UInt256.ofNat 1, word P_B, word SC_B, word CALLER, word LB, UInt256.ofNat 1, exporterWord⟩ := by
  have gm : ∀ (σx σ₀x : AccountMap) (Ax : Substate) (Ix : ExecutionEnv) (gx : Sat256) (cd : ByteArray),
      Ix.code = mockCode → Ix.calldata = cd →
        RDret mockCode gx (initState σx σ₀x gx Ax Ix) σx (UInt256.toByteArray (mockWord σx Ix)) :=
    fun _ _ _ _ _ _ h _ => mock_run h
  refine ⟨?_, ?_, ?_, ?_, ?_, ?_, ?_⟩
  · exact mock_returnsWord (gm · · · · · _) (by decide) (by decide +kernel) (by decide +kernel)
      (by decide +kernel)
  · exact mock_returnsWord (gm · · · · · _) (by decide) (by decide +kernel) (by decide +kernel)
      (by decide +kernel)
  · exact mock_returnsWord (gm · · · · · _) (by decide) (by decide +kernel) (by decide +kernel)
      (by decide +kernel)
  · exact mock_returnsWord (gm · · · · · _) (by decide) (by decide +kernel) (by decide +kernel)
      (by decide +kernel)
  · exact mock_returnsWord (code := portalCode)
      (fun _ _ _ _ _ h hc => portal_run h (by rw [hc]; decide +kernel)) (by decide) (by decide +kernel)
      (by decide +kernel) (by decide +kernel)
  · exact mock_returnsWord (gm · · · · · _) (by decide) (by decide +kernel) (by decide +kernel)
      (by decide +kernel)
  · exact mock_returnsWord (gm · · · · · _) (by decide) (by decide +kernel) (by decide +kernel)
      (by decide +kernel)

/-- The other two hypotheses of `relay_deposit` hold in the concrete world as well. -/
theorem concrete_self : ((world .ok).getD SELF default).code = l1cdmRuntime ∧ SELF ∉ π := by
  refine ⟨?_, by decide +kernel⟩
  rfl

end L1cdmEvm
