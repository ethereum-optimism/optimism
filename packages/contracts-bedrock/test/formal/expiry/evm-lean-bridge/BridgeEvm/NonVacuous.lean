import BridgeEvm.Concrete

/-!
# Non-vacuity of every headline theorem

For each theorem checked by `#assert_std_axioms` in `Axioms.lean` there is a `nonvacuous_<name>`
here: concrete values satisfying **all** of the theorem's hypotheses jointly, the theorem
instantiated on them, and its conclusion derived. `Axioms.lean` fails the build if a headline
theorem has no partner.

Trust: every step is kernel-checked except the facts that need *evaluating `Ξ` on the concrete
bytecode* (a successful run and a static-mode run). Those are isolated as the named
`native_decide` lemmas `native_run_success`, `native_run_static`; `Axioms.lean`
(`#assert_headline`) checks that every non-standard axiom of a partner comes from a `native_*`
lemma and that no headline theorem depends on one. The concrete hash
(`Concrete.refundHash_matches_cast`, keccak256 of the 352-byte preimage) and the slot facts are
kernel-checked. The witness is `Concrete.σ 0` / `Concrete.env 7` (bridge code at 0x…24, mock
messenger, mock ETHLiquidity; see `Concrete.lean`).

Hypotheses quantified over all states or callees: none remain in the headline theorems
(`StaticCall`, `CallTo`, `CreateStep` are existential facts about the actual run; the earlier
universal `MintFrame`/`CreateFrame` were removed). Every hypothesis below is discharged by the
concrete witness.
-/

namespace BridgeEvm.NonVacuous

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach Mem BridgeEvm.Concrete

/-! ## The common hypotheses on the concrete environment (kernel-checked) -/

theorem w_code : (env 7).code = ethbridgeRuntime := rfl

theorem w_sel : selectorWord (env 7) = refundSelector := by decide +kernel

theorem w_cds : (env 7).calldata.size < 2 ^ 256 := by decide +kernel

/-! ## Facts that need `Ξ` evaluated on the concrete bytecode (compiled evaluation, trusted) -/

def isSuccess (r : Except ExecutionException (ExecutionResult (AccountMap × UInt256 × Substate))) :
    Bool :=
  match r with
  | .ok (.success _ _) => true
  | _ => false

def isStaticViolation
    (r : Except ExecutionException (ExecutionResult (AccountMap × UInt256 × Substate))) : Bool :=
  match r with
  | .error .StaticModeViolation => true
  | _ => false

/-- `Ξ` on the concrete success scenario succeeds (compiled evaluation). -/
theorem native_run_success : isSuccess (run 0 7) = true := by native_decide

/-- `Ξ` on the concrete scenario entered statically raises `StaticModeViolation`. -/
theorem native_run_static :
    isStaticViolation (runx (σ 0) { env 7 with perm := false }) = true := by native_decide

theorem success_of_isSuccess {r : Except ExecutionException (ExecutionResult (AccountMap × UInt256 × Substate))}
    (h : isSuccess r = true) : ∃ σ' g' A' o, r = .ok (.success (σ', g', A') o) := by
  rcases r with e | res
  · simp [isSuccess] at h
  · rcases res with ⟨⟨σ', g', A'⟩, o⟩ | ⟨g, o⟩
    · exact ⟨σ', g', A', o, rfl⟩
    · simp [isSuccess] at h

theorem static_of_isStatic {r : Except ExecutionException (ExecutionResult (AccountMap × UInt256 × Substate))}
    (h : isStaticViolation r = true) : r = .error .StaticModeViolation := by
  rcases r with e | res
  · cases e <;> simp_all [isStaticViolation]
  · simp [isStaticViolation] at h

/-- The concrete successful run, as an equation about `Ξ`. -/
theorem w_success : ∃ σ' g' A' o,
    Ξ (σ 0) (σ 0) (UInt256.ofNat 10000000) default (env 7) = .ok (.success (σ', g', A') o) :=
  success_of_isSuccess native_run_success

/-! ## Kernel-checked facts of the witness -/

/-- `refunded[Hgood]` is unset in the pre-state (kernel: keccak slot lookup). -/
theorem w_unrefunded : refundedWord (σ 0) (env 7) Hgood = ⟨0⟩ := by decide +kernel

/-- The bridge's storage is empty in the pre-state. -/
theorem w_storage_empty : ((σ 0).getD (env 7).codeOwner default).storage = ∅ := by decide +kernel

theorem w_setTrue : setTrueWord ⟨0⟩ = UInt256.ofNat 1 := by decide +kernel

/-- The `CREATE`'s endowment and beneficiary argument in the witness. -/
theorem w_args : argAmount (env 7) = UInt256.ofNat amount ∧ argFrom (env 7) = UInt256.ofNat 0xf0f0 := by
  decide +kernel

theorem w_hg : (Sat256.ofUInt256 (UInt256.ofNat 10000000)).toUInt256 = UInt256.ofNat 10000000 := rfl

/-! ## One partner per headline theorem -/

/-- `refundETH_trace`: hypotheses satisfied (kernel); the theorem applies, and the disjunct realized
    is the success one (`RefundRun` and `RDret`): the other two would make the concrete run end in
    out of gas, a revert or a static violation, contradicting `native_run_success`. -/
theorem nonvacuous_refundETH_trace :
    (env 7).code = ethbridgeRuntime ∧ selectorWord (env 7) = refundSelector ∧
    (env 7).calldata.size < 2 ^ 256 ∧
    ∃ σ', RefundRun (σ 0) (σ 0) (env 7) σ' ∧
      RDret ethbridgeRuntime (Sat256.ofUInt256 (UInt256.ofNat 10000000))
        (initState (σ 0) (σ 0) (Sat256.ofUInt256 (UInt256.ofNat 10000000)) default (env 7)) σ'
        ByteArray.empty := by
  refine ⟨w_code, w_sel, w_cds, ?_⟩
  obtain ⟨σ₁, g₁, A₁, o₁, hres⟩ := w_success
  rcases refundETH_trace (σ := σ 0) (σ₀ := σ 0) (A := default)
      (g := Sat256.ofUInt256 (UInt256.ofNat 10000000)) w_code w_sel w_cds with hrev | ⟨_, hstat⟩ | h
  · rcases RDrev.xiResult w_code hrev with h | ⟨_, _, h⟩
    · rw [w_hg, hres] at h; cases h
    · rw [w_hg, hres] at h; cases h
  · rcases RDstatic.xiResult w_code hstat with h | h
    · rw [w_hg, hres] at h; cases h
    · rw [w_hg, hres] at h; cases h
  · exact h

/-- `refundETH_outcome`: on the concrete success run the outcome is the success disjunct. -/
theorem nonvacuous_refundETH_outcome :
    (env 7).code = ethbridgeRuntime ∧ selectorWord (env 7) = refundSelector ∧
    (env 7).calldata.size < 2 ^ 256 ∧
    ∃ σ' g' A', Ξ (σ 0) (σ 0) (UInt256.ofNat 10000000) default (env 7) =
      .ok (.success (σ', g', A') ByteArray.empty) ∧ RefundRun (σ 0) (σ 0) (env 7) σ' := by
  refine ⟨w_code, w_sel, w_cds, ?_⟩
  obtain ⟨σ', g', A', o, hres⟩ := w_success
  rcases refundETH_outcome (σ := σ 0) (σ₀ := σ 0) (A := default) (g := UInt256.ofNat 10000000)
      w_code w_sel w_cds with h | ⟨_, _, h⟩ | ⟨h, _⟩ | h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · exact h

/-- `refundETH_success`: all hypotheses (including a successful `Ξ` run) hold jointly; the theorem
    yields `RefundRun` and empty output. Its "`refunded[H]` was false" clause is consistent with the
    kernel-checked pre-state (`refundHash (env 7) = Hgood`, `refunded[Hgood] = 0`). -/
theorem nonvacuous_refundETH_success :
    ∃ σ' g' A' o,
      (env 7).code = ethbridgeRuntime ∧ selectorWord (env 7) = refundSelector ∧
      (env 7).calldata.size < 2 ^ 256 ∧
      Ξ (σ 0) (σ 0) (UInt256.ofNat 10000000) default (env 7) = .ok (.success (σ', g', A') o) ∧
      RefundRun (σ 0) (σ 0) (env 7) σ' ∧ o = ByteArray.empty ∧
      refundHash (env 7) = Hgood ∧ refundedWord (σ 0) (env 7) Hgood = ⟨0⟩ := by
  obtain ⟨σ', g', A', o, hres⟩ := w_success
  obtain ⟨hrun, ho⟩ := refundETH_success w_code w_sel w_cds hres
  exact ⟨σ', g', A', o, w_code, w_sel, w_cds, hres, hrun, ho, refundHash_matches_cast, w_unrefunded⟩

/-- `refundETH_no_other_error`: all hypotheses hold for the statically entered concrete run, which
    errors with `StaticModeViolation` (`he`); the theorem applies and yields `perm = false`. -/
theorem nonvacuous_refundETH_no_other_error :
    let I : ExecutionEnv := { env 7 with perm := false }
    I.code = ethbridgeRuntime ∧ selectorWord I = refundSelector ∧ I.calldata.size < 2 ^ 256 ∧
    Ξ (σ 0) (σ 0) (UInt256.ofNat 10000000) default I = .error .StaticModeViolation ∧
    I.perm = false := by
  intro I
  have he : Ξ (σ 0) (σ 0) (UInt256.ofNat 10000000) default I = .error .StaticModeViolation :=
    static_of_isStatic native_run_static
  refine ⟨rfl, w_sel, w_cds, he, ?_⟩
  rcases refundETH_no_other_error (σ := σ 0) (σ₀ := σ 0) (A := default)
      (g := UInt256.ofNat 10000000) (I := I) rfl w_sel w_cds he with h | ⟨_, h⟩
  · cases h
  · exact h

/-- The `RefundRun` of the concrete success run, unpacked. -/
theorem w_run : ∃ σ', RefundRun (σ 0) (σ 0) (env 7) σ' := by
  obtain ⟨σ', _, _, _, hres⟩ := w_success
  exact ⟨σ', (refundETH_success w_code w_sel w_cds hres).1⟩

/-- `createStep_success`: its hypotheses (`CreateStep` and `x ≠ 0`) are those of the concrete
    successful run (endowment `amount`, beneficiary `0xf0f0`, kernel); the theorem yields the
    nonce bound, the created address and empty return data. -/
theorem nonvacuous_createStep_success :
    argAmount (env 7) = UInt256.ofNat amount ∧ argFrom (env 7) = UInt256.ofNat 0xf0f0 ∧
    ∃ σ₃ σ' x rd', CreateStep (env 7) (σ 0) σ₃ (argAmount (env 7)) (safeSendDeploy (argFrom (env 7)))
      x σ' rd' ∧ x ≠ UInt256.ofNat 0 ∧
      (σ₃.get? (env 7).codeOwner |>.getD default).nonce.toNat < 2 ^ 64 - 1 ∧
      ∃ a : AccountAddress, x = UInt256.ofNat a ∧ rd' = .empty := by
  obtain ⟨σ', _, _, σ₁, oE, _, _, _, _, _, _, _, σ₃, oM, _, x, rd', hcs, hx⟩ := w_run
  obtain ⟨hn, _, _, _, a, _, _, _, _, hxa, hrd⟩ := createStep_success hcs hx
  exact ⟨w_args.1, w_args.2, σ₃, σ', x, rd', hcs, hx, hn, a, hxa, hrd⟩

/-- The concrete pre-state: the executing account has code (needed by the post-state lemmas). -/
theorem w_hascode : ((σ 0).getD (env 7).codeOwner default).code.size ≠ 0 := bridge_has_code

/-- `storedMap_post`: `σ₁` from the concrete run, with `hst`, `hcd`, `hcode` all satisfied; the
    theorem applies at `H = Hgood`, and with the kernel-checked pre-state (empty storage,
    `refunded[Hgood] = 0`) the bridge's storage after the store is exactly `{refunded[Hgood] ↦ 1}`. -/
theorem nonvacuous_storedMap_post :
    ∃ σ₁, accountStorageStateEq (σ 0) σ₁ ∧ accountCodeStateEq (σ 0) σ₁ ∧
      ((σ 0).getD (env 7).codeOwner default).code.size ≠ 0 ∧
      ((storedMap σ₁ (env 7) Hgood).getD (env 7).codeOwner default).storage =
        (∅ : Storage).insert (refundedSlot Hgood) (UInt256.ofNat 1) ∧
      UInt256.land (UInt256.ofNat 255) (refundedWord (storedMap σ₁ (env 7) Hgood) (env 7) Hgood) =
        UInt256.ofNat 1 := by
  obtain ⟨σ', _, _, σ₁, oE, _, _, _, hst, hcd, _⟩ := w_run
  obtain ⟨h1, _, _, h4⟩ := storedMap_post (H := Hgood) hst hcd w_hascode
  refine ⟨σ₁, hst, hcd, w_hascode, ?_, h4⟩
  rw [h1, w_storage_empty, w_unrefunded, w_setTrue]

/-- `refundETH_store`: hypotheses (`RefundRun`, code present) satisfied by the concrete run; the
    theorem yields the store, the `mint(amount)` call and the SafeSend creation. -/
theorem nonvacuous_refundETH_store :
    ∃ σ' σ₂ σ₃, RefundRun (σ 0) (σ 0) (env 7) σ' ∧
      UInt256.land (UInt256.ofNat 255) (refundedWord σ₂ (env 7) Hgood) = UInt256.ofNat 1 ∧
      (∃ oM, CallTo (σ 0) (env 7) ethLiq (mintCalldata (argAmount (env 7))) σ₂ σ₃ true oM) ∧
      (∃ x rd', CreateStep (env 7) (σ 0) σ₃ (argAmount (env 7)) (safeSendDeploy (argFrom (env 7)))
        x σ' rd' ∧ x ≠ UInt256.ofNat 0) := by
  obtain ⟨σ', hrun⟩ := w_run
  obtain ⟨σ₂, σ₃, _, _, _, h4, h5, h6⟩ := refundETH_store hrun w_hascode
  rw [refundHash_matches_cast] at h4
  exact ⟨σ', σ₂, σ₃, hrun, h4, h5, h6⟩

/-! `RD.create` on a one-byte program `CREATE` with stack `[0, 0, 0]`, entered at pc 0
(all kernel-checked): the `RD` hypothesis is built directly with `RD.start`. -/

def createCode : ByteArray := ⟨#[0xf0]⟩

def createEnv : ExecutionEnv := { (default : ExecutionEnv) with code := createCode, perm := true }

def createState : State :=
  { (default : State) with
      executionEnv := createEnv
      machineState := { (default : MachineState) with
        stack := [⟨0⟩, ⟨0⟩, ⟨0⟩], gasAvailable := Sat256.ofUInt256 (UInt256.ofNat 100000) } }

theorem w_create_rd :
    RD createCode createEnv (Sat256.ofUInt256 (UInt256.ofNat 100000)) createState ⟨0⟩ [⟨0⟩, ⟨0⟩, ⟨0⟩]
      createState.machineState.memory createState.machineState.activeWords
      createState.machineState.returnData createState.accountMap 0 0 :=
  RD.start (s0 := createState) (s := createState) rfl rfl rfl (by simp [createState]) (le_refl _)
    (Nat.zero_le _) (by simp) rfl

theorem w_create_dec : decode createCode ⟨0⟩ = some (.CREATE, .none) := by evm_kdecide

/-- `RD.create`: its hypotheses (an `RD` cursor at a `CREATE`, the decode, `perm`, size bound,
    stack bound) are satisfied; the conclusion is instantiated. -/
theorem nonvacuous_RD_create :
    ∃ (x : UInt256) (σ' : AccountMap) (rd' : ByteArray) (k' C' : ℕ),
      CreateStep createEnv createState.σ₀ createState.accountMap ⟨0⟩
        (createState.machineState.memory.readWithPadding (⟨0⟩ : UInt256).toNat (⟨0⟩ : UInt256).toNat)
        x σ' rd' ∧
      RD createCode createEnv (Sat256.ofUInt256 (UInt256.ofNat 100000)) createState (⟨0⟩ + ⟨1⟩) [x]
        createState.machineState.memory
        (UInt256.ofNat (MachineState.M createState.machineState.activeWords.toNat 0 0))
        rd' σ' k' C' :=
  RD.create w_create_rd w_create_dec rfl (by decide) (by simp)

/-- `refundPreimage_size` (no hypotheses): instantiated on the concrete environment. -/
theorem nonvacuous_refundPreimage_size : (refundPreimage (env 7)).size = 352 :=
  refundPreimage_size (env 7)

end BridgeEvm.NonVacuous
