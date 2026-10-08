import ExpiryEvm.ExpireMessage
import ExpiryEvm.Post
import ExpiryEvm.KernelRun

/-!
# Concrete runs (reachability witnesses and boundary checks)

EVMLean is executable. These runs execute the pinned bytecode with `Ξ` on concrete states:
the L2ToL2CrossDomainMessenger code at 0x4200..0023, a mock L2CrossDomainMessenger at
0x4200..0007 whose code returns the same address for every call (so
`xDomainMessageSender() == otherMessenger()`), and `sentMessageTimestamps[H] = 5`.
They show the success branch of the theorems is reachable and check the `>` boundary.
They are checked by `native_decide` (compiled evaluation) and are tests, not part of the proof
of the headline theorems.
-/

namespace ExpiryEvm.Concrete

open Ethereum Ethereum.EVM

def messengerAddr : AccountAddress := AccountAddress.ofUInt256 (UInt256.ofNat 0x4200000000000000000000000000000000000023)

/-- `PUSH20 0x..beef PUSH0 MSTORE PUSH1 0x20 PUSH0 RETURN`: returns the word 0x..beef. -/
def mockL2cdmCode : ByteArray :=
  ⟨#[0x73, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xbe, 0xef,
     0x5f, 0x52, 0x60, 0x20, 0x5f, 0xf3]⟩

def H : UInt256 := UInt256.ofNat 0x1234

def calldataOf (t : ℕ) : ByteArray :=
  ⟨#[0x76, 0x3a, 0x1c, 0xb7]⟩ ++ UInt256.toByteArray H ++ UInt256.toByteArray (UInt256.ofNat t)

def σ : AccountMap :=
  (∅ : AccountMap)
    |>.insert messengerAddr
      { (default : Account) with
          code := l2tol2Runtime
          storage := (∅ : Storage).insert (sentAtSlot H) (UInt256.ofNat 5) }
    |>.insert l2cdm { (default : Account) with code := mockL2cdmCode }

def env (caller : AccountAddress) (t : ℕ) : ExecutionEnv :=
  { (default : ExecutionEnv) with
      codeOwner := messengerAddr
      sender := caller
      source := caller
      weiValue := ⟨0⟩
      calldata := calldataOf t
      code := l2tol2Runtime
      depth := 1
      perm := true }

def run (caller : AccountAddress) (t : ℕ) :=
  Ξ σ σ (UInt256.ofNat 1000000) default (env caller t)

/-- `expiredMessages[H]` of the messenger after a run, or `none` if the run did not succeed. -/
def expiredAfter (caller : AccountAddress) (t : ℕ) : Option UInt256 :=
  match run caller t with
  | .ok (.success (σ', _, _) _) => some (storageWord σ' messengerAddr (expiredSlot H))
  | _ => none

def reverted (caller : AccountAddress) (t : ℕ) : Bool :=
  match run caller t with
  | .ok (.revert _ _) => true
  | _ => false

#eval expiredAfter l2cdm (5 + P_contract + 1)
#eval reverted l2cdm (5 + P_contract)
#eval reverted (AccountAddress.ofUInt256 (UInt256.ofNat 0x99)) (5 + P_contract + 1)

/-- Reachability witness: at `t = sentAt + P + 1` the call from the L2CrossDomainMessenger
    succeeds and sets `expiredMessages[H]` to 1. -/
theorem success_reachable : expiredAfter l2cdm (5 + P_contract + 1) = some (UInt256.ofNat 1) := by
  native_decide

/-- Boundary: at `t = sentAt + P` (the `≥`-mutation would accept this) the code reverts. -/
theorem boundary_reverts : reverted l2cdm (5 + P_contract) = true := by
  native_decide

/-- Any other caller reverts. -/
theorem other_caller_reverts :
    reverted (AccountAddress.ofUInt256 (UInt256.ofNat 0x99)) (5 + P_contract + 1) = true := by
  native_decide

/-! ## Other branch kinds (witnesses that each disjunct of `expireMessage_outcome` is inhabited) -/

def outcome (σ : AccountMap) (I : ExecutionEnv) (gas : ℕ) : String :=
  match Ξ σ σ (UInt256.ofNat gas) default I with
  | .ok (.success _ _) => "success"
  | .ok (.revert _ _) => "revert"
  | .error .OutOfGass => "oog"
  | .error .StaticModeViolation => "static"
  | .error _ => "other error"

/-- Out of gas: the same successful call with 20000 gas runs out of gas. -/
theorem oog_reachable : outcome σ (env l2cdm (5 + P_contract + 1)) 20000 = "oog" := by
  native_decide

/-- Static mode: entered with `perm = false` (via `STATICCALL`), the run halts at the `SSTORE`. -/
theorem static_reachable :
    outcome σ { env l2cdm (5 + P_contract + 1) with perm := false } 1000000 = "static" := by
  native_decide

/-- A mock L2CrossDomainMessenger whose code is `PUSH0 PUSH0 REVERT`. -/
def σRevertingL2cdm : AccountMap :=
  σ.insert l2cdm { (default : Account) with code := ⟨#[0x5f, 0x5f, 0xfd]⟩ }

/-- Callee failure: if the L2CrossDomainMessenger's view call reverts, `expireMessage` reverts. -/
theorem callee_failure_reverts :
    outcome σRevertingL2cdm (env l2cdm (5 + P_contract + 1)) 1000000 = "revert" := by
  native_decide

/-- `t` of the success witness. -/
def tS : ℕ := 5 + P_contract + 1

/-- The static call `otherMessenger()` from `σ` with 0 forwarded gas. -/
def zeroGasCall :=
  Θ σ σ default (AccountAddress.ofUInt256 (UInt256.ofNat (env l2cdm tS).codeOwner))
    (env l2cdm tS).sender l2cdm (toExecute σ l2cdm) ⟨0⟩
    (UInt256.ofNat (env l2cdm tS).gasPrice) ⟨0⟩ ⟨0⟩ otherMessengerCalldata
    ((env l2cdm tS).depth + 1) (env l2cdm tS).header (env l2cdm tS).blobVersionedHashes
    (env l2cdm tS).blocks false

/-- **Why `CallFailed` is weak.** It holds in `σ`, where the run with `t = sentAt + P + 1`
    succeeds (`success_reachable`): the `otherMessenger()` call with 0 gas fails. Statements
    with a `revert ∧ CallFailed` disjunct therefore do not exclude reverts. -/
theorem callFailed_in_success_state : CallFailed σ σ (env l2cdm tS) := by
  have hz : zeroGasCall.2.2.2.1 = false := by native_decide
  refine Or.inr (Or.inl ⟨zeroGasCall.1, zeroGasCall.2.2.2.2, default, ⟨0⟩, zeroGasCall.2.1,
    zeroGasCall.2.2.1, ?_⟩)
  show _ = zeroGasCall
  rw [← hz]

/-! ## Non-vacuity of the headline hypotheses

The two call summaries (`ReturnsAddress`) are proved — not just executed — for the concrete state
`σ`, and `expireMessage_success` is instantiated on the concrete successful run. -/

section NonVacuity

open Reasoning.Theory Reasoning.Reach

/-- A successful `Θ` message call into code `code` came from a successful `Ξ` run of that code
    whose output is the call's output. -/
theorem theta_code_success {σ σ₀ : AccountMap} {A : Substate} {s o r : AccountAddress}
    {code : ByteArray} {g p v v' : UInt256} {d : ByteArray} {e : Fin 1025} {H : BlockHeader}
    {bvh : List ByteArray} {blocks : ProcessedBlocks}
    {σ' : AccountMap} {g' : UInt256} {A' : Substate} {out : ByteArray}
    (h : (σ', g', A', true, out) = Θ σ σ₀ A s o r (.Code code) g p v v' d e H bvh blocks false) :
    ∃ (σ₁ : AccountMap) (I : ExecutionEnv) (σ'' : AccountMap) (g'' : UInt256) (A'' : Substate),
      I.code = code ∧ Ξ σ₁ σ₀ g A I = .ok (.success (σ'', g'', A'') out) := by
  unfold Θ at h
  simp only [] at h
  split at h
  · exfalso
    have h4 := congrArg (fun x => x.2.2.2.1) h
    simp at h4
  · exfalso
    have h4 := congrArg (fun x => x.2.2.2.1) h
    simp at h4
  · rename_i heq
    have h5 := congrArg (fun x => x.2.2.2.2) h
    simp only [] at h5
    exact ⟨_, _, _, _, _, rfl, by rw [heq, h5]⟩

/-- The word the mock returns. -/
def mockWord : UInt256 := UInt256.ofNat 0xbeef

/-- The address the mock returns for both calls. -/
def vMock : AccountAddress := AccountAddress.ofUInt256 mockWord

/-- Every run of the mock code (any state, gas, calldata) runs out of gas or returns
    exactly the 32-byte word `mockWord`. -/
theorem mock_xi {σ₁ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256}
    (hcode : I.code = mockL2cdmCode) :
    Ξ σ₁ σ₀ g A I = .error .OutOfGass ∨
    ∃ g' A', Ξ σ₁ σ₀ g A I = .ok (.success (σ₁, g', A') (UInt256.toByteArray mockWord)) := by
  have r0 := RD.initState (σ := σ₁) (σ₀ := σ₀) (A := A) (g := Sat256.ofUInt256 g) hcode
  have r1 := kevm_run r0 with [push20 mockWord, push0]
  have r2 := RD.genMstore r1 (by evm_kdecide) (by evm_ov)
  have r3 := kevm_run r2 with [push1 (UInt256.ofNat 32), push0]
  have r4 := RD.genRet r3 (by evm_kdecide) (by evm_ov)
  have hm : (UInt256.toByteArray mockWord).write 0 ByteArray.empty (⟨0⟩ : UInt256).toNat 32 =
      Mem.wordsMem [mockWord] := Mem.wordsMem_write_end [] mockWord 0 rfl
  rw [hm, show (⟨0⟩ : UInt256).toNat = 0 from rfl, show (UInt256.ofNat 32).toNat = 32 by decide,
    Mem.wordsMem_read32 [mockWord] 0 (by simp) 0 rfl] at r4
  exact rdret_xi hcode r4

theorem l2cdm_not_precompile : ¬ (l2cdm ∈ π) := by decide +kernel

theorem sigma_l2cdm_code : (σ.getD l2cdm default).code = mockL2cdmCode := by decide +kernel

/-- From any account map with `σ`'s code, the L2CrossDomainMessenger address runs the mock. -/
theorem toExecute_mock {σc : AccountMap} (hcd : accountCodeStateEq σ σc) :
    toExecute σc l2cdm = .Code mockL2cdmCode := by
  have h := hcd l2cdm
  rw [sigma_l2cdm_code] at h
  unfold toExecute
  rw [if_neg l2cdm_not_precompile]
  cases hg : σc.get? l2cdm with
  | none =>
    exfalso
    have : σc.getD l2cdm default = default := by
      rw [Std.ExtTreeMap.getD_eq_getD_getElem?,
        show σc[l2cdm]? = none by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact hg]; rfl
    rw [this] at h
    exact absurd h (by decide +kernel)
  | some acc =>
    simp only [Id.run]
    have : σc.getD l2cdm default = acc := getD_of_get? hg
    rw [this] at h
    rw [h]

/-- **The call summary holds for the concrete state**, for any calldata, environment and `σ₀`:
    a successful static call to 0x..07 returns exactly the word `mockWord`. -/
theorem mock_returnsAddress (σ₀ : AccountMap) (I : ExecutionEnv) (cd : ByteArray) :
    ReturnsAddress σ σ₀ I cd vMock := by
  intro σc σ' z o _hst hcd hcall hz
  subst hz
  obtain ⟨A_in, cg, g', A', hΘ⟩ := hcall
  rw [toExecute_mock hcd] at hΘ
  obtain ⟨σ₁, I', σ'', g'', A'', hIc, hxi⟩ := theta_code_success hΘ
  rcases mock_xi (σ₁ := σ₁) (σ₀ := σ₀) (A := A_in) (g := cg) hIc with hoog | ⟨g3, A3, hs⟩
  · rw [hxi] at hoog; cases hoog
  · rw [hxi] at hs
    injection hs with hs
    injection hs with _ ho
    subst ho
    exact ⟨by rw [toByteArray_size], by
      rw [toByteArray_extract_all]; rfl⟩

def isSuccess : Except ExecutionException (ExecutionResult (AccountMap × UInt256 × Substate)) → Bool
  | .ok (.success _ _) => true
  | _ => false

/-- **NATIVE (compiled evaluation).** The concrete run succeeds. This is the only fact of the
    success witnesses that is not kernel-checked: the kernel cannot evaluate `Ξ` (its
    well-founded recursion does not reduce). It is used only by `success_instance` and the
    `nonvacuous_*` witnesses (`NonVacuity.lean`), never by a headline theorem. -/
theorem native_xi_success :
    isSuccess (Ξ σ σ (UInt256.ofNat 1000000) default (env l2cdm tS)) = true := by
  native_decide

/-- The selector of the witness calldata (kernel-checked). -/
theorem env_sel : selectorWord (env l2cdm tS) = expireSelector := by
  decide +kernel

/-- The calldata bound of the witness (kernel-checked). -/
theorem env_cds : (env l2cdm tS).calldata.size < 2 ^ 256 := by
  decide +kernel

/-- **Non-vacuity of the headline theorem.** All hypotheses of `expireMessage_success` —
    code, selector, calldata bound and both call summaries — hold for the concrete run, which
    succeeds; so `ExpireConds` and `ExpirePost` hold there. -/
theorem success_instance :
    ∃ σ' g' A' o, Ξ σ σ (UInt256.ofNat 1000000) default (env l2cdm tS) =
        .ok (.success (σ', g', A') o) ∧
      ExpireConds σ (env l2cdm tS) vMock vMock ∧ ExpirePost σ σ' (env l2cdm tS) := by
  have hs := native_xi_success
  cases h : Ξ σ σ (UInt256.ofNat 1000000) default (env l2cdm tS) with
  | error e => rw [h] at hs; cases hs
  | ok r =>
    cases r with
    | revert g' o => rw [h] at hs; cases hs
    | success t o =>
      obtain ⟨σ', g', A'⟩ := t
      obtain ⟨_, hc, hpost, _⟩ := expireMessage_success (σ := σ) (σ₀ := σ) (A := default)
        (I := env l2cdm tS) (g := UInt256.ofNat 1000000) rfl env_sel env_cds
        (mock_returnsAddress σ _ _) (mock_returnsAddress σ _ _) h
      exact ⟨σ', g', A', o, rfl, hc, hpost⟩

end NonVacuity

end ExpiryEvm.Concrete
