import ExporterEvm.TraceLog

/-!
# `exportUndeliveredMessage` refinement theorems (EVM level)

The deployed runtime code `exporterRuntime` of `UndeliveredMessageExporter`, run by EVMLean's
code-execution function `Ξ` on calldata that selects
`exportUndeliveredMessage(address,uint256,uint256,address,address,bytes,uint32)`, from any account
map, caller, value, depth, gas and permission, with calldata shorter than `2^63` bytes:

* `export_trace` — the whole symbolic execution, at the level of EquiVM's `RD` invariant.
* `export_outcome` — every run ends in out-of-gas, a revert with an output allowed by
  `ExportRevert`, a static-mode violation (only in a static frame, `I.perm = false`), or success
  returning exactly `abi.encode(H)` with `ExportRun`.
* `export_success`, `export_revert`, `export_no_other_error` — its three projections. On success
  the final substate's log series ends with the `UndeliveredMessageExported` entry.

No external call is assumed to succeed or to return anything: `ExportRun` *states* that the two
calls happened with exact inputs and the outputs that make the code proceed.
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach Mem

/-- **What a successful `exportUndeliveredMessage(...)` run implies**, with `H = exportHash I`,
    pre-state `σ` and final account map `σ'`:
    1. the ABI conditions `ArgsOk` and a non-static frame;
    2. a `STATICCALL` to the L2ToL2CrossDomainMessenger 0x4200…0023 with calldata exactly
       `successfulMessages(H)` succeeded and returned ≥ 32 bytes whose first word is `0` (false);
       it left storage and code unchanged (σ₁);
    3. the L2CrossDomainMessenger 0x4200…0007 had code in σ₁;
    4. a `CALL` to it with value 0 and calldata exactly
       `sendMessage(_sourceMessenger, relayUndeliveredMessage(H, block.timestamp), _minGasLimit)`
       ran from σ₁, succeeded, and its resulting account map is the final `σ'` (nothing after the
       call changes the account map: the event and the return touch only the substate and
       memory). -/
def ExportRun (σ σ₀ : AccountMap) (I : ExecutionEnv) (σ' : AccountMap) : Prop :=
  ArgsOk I ∧ I.perm = true ∧
  ∃ σ₁ oS, StaticCall σ₀ I l2l2 (successfulCalldata (exportHash I)) σ σ₁ true oS ∧
    32 ≤ oS.size ∧ returnWord oS = UInt256.ofNat 0 ∧
    accountStorageStateEq σ σ₁ ∧ accountCodeStateEq σ σ₁ ∧
    extCodeSizeWord σ₁ l2cdmWord ≠ ⟨0⟩ ∧
    ∃ oC, CallTo σ₀ I l2cdm
      (sendMessageCd (argSrcMessenger I) (exportHash I) (tsWord I) (argMinGas I)) σ₁ σ' true oC

/-- **The only possible revert outputs**:
    1. empty (`revert(0, 0)`: ETH attached, malformed calldata, call depth 1024, return data
       shorter than 32 bytes or not a clean bool, or no code at 0x4200…0007);
    2. the return data of a *failed* `successfulMessages(H)` static call, bubbled;
    3. exactly `UndeliveredMessageExporter_MessageRelayed()` (`0xccc3f3b0`), and then
       `successfulMessages(H)` returned `true`;
    4. the return data of a *failed* `sendMessage(...)` call, bubbled, after
       `successfulMessages(H)` returned `false` (and 0x4200…0007 had code).
    This is a necessary condition on the output, not a unique cause: the alternatives overlap
    (a callee may itself revert with empty data or with `0xccc3f3b0`). -/
def ExportRevert (σ σ₀ : AccountMap) (I : ExecutionEnv) (o : ByteArray) : Prop :=
  o = ByteArray.empty ∨
  (ArgsOk I ∧ ∃ σ₁ rd, StaticCall σ₀ I l2l2 (successfulCalldata (exportHash I)) σ σ₁ false rd ∧
    o = bubble rd) ∨
  (ArgsOk I ∧ o = messageRelayedError ∧ ∃ σ₁ oS,
    StaticCall σ₀ I l2l2 (successfulCalldata (exportHash I)) σ σ₁ true oS ∧
    32 ≤ oS.size ∧ returnWord oS = UInt256.ofNat 1) ∨
  (ArgsOk I ∧ ∃ σ₁ oS, StaticCall σ₀ I l2l2 (successfulCalldata (exportHash I)) σ σ₁ true oS ∧
    32 ≤ oS.size ∧ returnWord oS = UInt256.ofNat 0 ∧
    accountStorageStateEq σ σ₁ ∧ accountCodeStateEq σ σ₁ ∧ extCodeSizeWord σ₁ l2cdmWord ≠ ⟨0⟩ ∧
    ∃ σ₂ rd, CallTo σ₀ I l2cdm
      (sendMessageCd (argSrcMessenger I) (exportHash I) (tsWord I) (argMinGas I)) σ₁ σ₂ false rd ∧
      o = bubble rd)

/-- The preimage has `7·32 + roundUp32(len)` bytes (`abi.encode` of five words and a `bytes`). -/
theorem exportPreimage_size (I : ExecutionEnv) (hargs : ArgsOk I) :
    (exportPreimage I).size = 224 + roundUp32 (msgLen I) := by
  rw [exportPreimage_eq I hargs]
  simp only [ofL_size, List.length_append, length_wb, msgData_length I hargs, padZ, List.length_replicate]
  have := roundUp32_ge (msgLen I)
  omega

theorem endPtr_lt (I : ExecutionEnv) (hargs : ArgsOk I) (hcds : I.calldata.size < 2 ^ 63) :
    endPtr I < 2 ^ 139 := by
  have := msgLen_lt I hargs hcds
  have := roundUp32_lt (msgLen I)
  unfold endPtr p1; omega

set_option maxHeartbeats 4000000 in
theorem export_trace {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    (hcode : I.code = exporterRuntime) (hsel : selectorWord I = exportSelector)
    (hcds : I.calldata.size < 2 ^ 63) :
    RDrevP exporterRuntime g (initState σ σ₀ g A I) (ExportRevert σ σ₀ I) ∨
    (I.perm = false ∧ RDstatic exporterRuntime g (initState σ σ₀ g A I)) ∨
    (∃ σ', ExportRun σ σ₀ I σ' ∧
      RDretL exporterRuntime g (initState σ σ₀ g A I) σ' (UInt256.toByteArray (exportHash I))
        (exportedLog I (exportHash I))) := by
  rcases seg_entry (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode hsel hcds with hrev | ⟨hargs, aw, k, C, r1⟩
  · exact Or.inl (hrev.mono (fun _ ho => Or.inl ho))
  obtain ⟨B, aw2, k2, C2, hB, r2⟩ := seg_hash hargs hcds r1
  rcases seg_static hB (endPtr_lt I hargs hcds) (by simp) r2 with hrev |
      ⟨σ₁, oS, hcall, hlen, _, hw0, hst, hcd, B2, fp2, aw3, k3, C3, hB2, hfp2, r3⟩
  · left
    refine hrev.mono ?_
    rintro o (ho | ⟨σ₁, rd, hc, ho⟩ | ⟨ho, σ₁, oS, hc, hl, hw⟩)
    · exact Or.inl ho
    · exact Or.inr (Or.inl ⟨hargs, σ₁, rd, hc, ho⟩)
    · exact Or.inr (Or.inr (Or.inl ⟨hargs, ho, σ₁, oS, hc, hl, hw⟩))
  rcases seg_call hargs hB2 hfp2 r3 with hrev | ⟨hx, σ₂, oC, hcall2, Q, aw4, k4, C4, hQ, r4⟩
  · left
    refine hrev.mono ?_
    rintro o (ho | ⟨hx, σ₂, rd, hc, ho⟩)
    · exact Or.inl ho
    · exact Or.inr (Or.inr (Or.inr ⟨hargs, σ₁, oS, hcall, hlen, hw0, hst, hcd, hx, σ₂, rd, hc, ho⟩))
  rcases seg_log hargs hB2 hfp2 hQ r4 with ⟨hp, hret⟩ | ⟨hp, hstat⟩
  · exact Or.inr (Or.inr ⟨σ₂, ⟨hargs, hp, σ₁, oS, hcall, hlen, hw0, hst, hcd, hx, oC, hcall2⟩, hret⟩)
  · exact Or.inr (Or.inl ⟨hp, hstat⟩)

/-- **Outcome theorem.** Every run of the compiled exporter on `exportUndeliveredMessage`
    calldata ends in exactly one of: out of gas; a revert whose output satisfies `ExportRevert`;
    a static-mode violation (only in a static frame, `I.perm = false`); or success returning exactly the
    32-byte word `H = exportHash I` with `ExportRun` relating the pre-state `σ` to the final
    account map `σ'`. -/
theorem export_outcome {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256}
    (hcode : I.code = exporterRuntime) (hsel : selectorWord I = exportSelector)
    (hcds : I.calldata.size < 2 ^ 63) :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    (∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o) ∧ ExportRevert σ σ₀ I o) ∨
    (Ξ σ σ₀ g A I = .error .StaticModeViolation ∧ I.perm = false) ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') (UInt256.toByteArray (exportHash I))) ∧
      ExportRun σ σ₀ I σ' ∧ ∃ L : LogSeries, A'.logSeries = L.push (exportedLog I (exportHash I))) := by
  have hg : (Sat256.ofUInt256 g).toUInt256 = g := rfl
  rcases export_trace (σ := σ) (σ₀ := σ₀) (A := A) (g := Sat256.ofUInt256 g) hcode hsel hcds with
    hrev | ⟨hp, hstat⟩ | ⟨σ', hrun, hret⟩
  · rcases RDrevP.xiResult hcode hrev with hoog | ⟨g', o, hr, hP⟩
    · exact Or.inl hoog
    · exact Or.inr (Or.inl ⟨g', o, hr, hP⟩)
  · rcases RDstatic.xiResult hcode hstat with hoog | hs
    · left; rw [← hg]; exact hoog
    · right; right; left; exact ⟨by rw [← hg]; exact hs, hp⟩
  · rcases rdretL_xi hcode hret with hoog | ⟨g', A', hr, hL⟩
    · exact Or.inl hoog
    · exact Or.inr (Or.inr (Or.inr ⟨σ', g', A', hr, hrun, hL⟩))

/-- **Soundness.** If the compiled exporter succeeds on `exportUndeliveredMessage(...)` from
    pre-state `σ` with final account map `σ'`, then `ExportRun σ σ₀ I σ'` holds and the output is
    exactly the message hash `H` as one 32-byte word. -/
theorem export_success {σ σ₀ σ' : AccountMap} {A A' : Substate} {I : ExecutionEnv}
    {g g' : UInt256} {o : ByteArray}
    (hcode : I.code = exporterRuntime) (hsel : selectorWord I = exportSelector)
    (hcds : I.calldata.size < 2 ^ 63)
    (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    ExportRun σ σ₀ I σ' ∧ o = UInt256.toByteArray (exportHash I) ∧
      ∃ L : LogSeries, A'.logSeries = L.push (exportedLog I (exportHash I)) := by
  rcases export_outcome (g := g) (A := A) hcode hsel hcds with
    h | ⟨_, _, h, _⟩ | ⟨h, _⟩ | ⟨σ'', g'', A'', h, hrun, hL⟩
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h
    cases h
    exact ⟨hrun, rfl, hL⟩

/-- **Revert classification.** A reverting run's output is one of the four `ExportRevert`
    alternatives: empty, the custom error `UndeliveredMessageExporter_MessageRelayed()`, or a
    callee's failure bubbled. -/
theorem export_revert {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv}
    {g g' : UInt256} {o : ByteArray}
    (hcode : I.code = exporterRuntime) (hsel : selectorWord I = exportSelector)
    (hcds : I.calldata.size < 2 ^ 63)
    (hres : Ξ σ σ₀ g A I = .ok (.revert g' o)) :
    ExportRevert σ σ₀ I o := by
  rcases export_outcome (g := g) (A := A) hcode hsel hcds with
    h | ⟨_, _, h, hP⟩ | ⟨h, _⟩ | ⟨_, _, _, h, _, _⟩
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h; exact hP
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h

/-- No exceptional halt other than out-of-gas or (in a static frame, `I.perm = false`) a static-mode
    violation: no `INVALID`, stack under/overflow, bad jump, etc. -/
theorem export_no_other_error {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv}
    {g : UInt256} {e : ExecutionException}
    (hcode : I.code = exporterRuntime) (hsel : selectorWord I = exportSelector)
    (hcds : I.calldata.size < 2 ^ 63)
    (he : Ξ σ σ₀ g A I = .error e) :
    e = .OutOfGass ∨ (e = .StaticModeViolation ∧ I.perm = false) := by
  rcases export_outcome (g := g) (A := A) hcode hsel hcds with
    h | ⟨_, _, h, _⟩ | ⟨h, hp⟩ | ⟨_, _, _, h, _, _⟩
  · rw [he] at h; cases h; exact Or.inl rfl
  · rw [he] at h; cases h
  · rw [he] at h; cases h; exact Or.inr ⟨rfl, hp⟩
  · rw [he] at h; cases h

end ExporterEvm
