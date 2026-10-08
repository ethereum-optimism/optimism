import BridgeEvm.Concrete

/-!
# Non-vacuity of every headline theorem

For each theorem checked by `#assert_std_axioms` in `Axioms.lean` there is a `nonvacuous_<name>`
here: concrete values satisfying **all** of the theorem's hypotheses jointly, the theorem
instantiated on them, and its conclusion derived. `Axioms.lean` fails the build if a headline
theorem has no partner.

Trust: every step is kernel-checked except the facts that need *evaluating `Ξ` on the concrete
bytecode* (a successful run, a static-mode run, and the concrete hash). Those are isolated as the
named `native_decide` lemmas `run_success_native`, `run_static_native` (and `Concrete.*`), which
appear only here and in `Concrete.lean`, never in a headline theorem's axiom cone (the assertion
in `Axioms.lean` proves this). The witness is `Concrete.σ 0` / `Concrete.env 7` (bridge code at
0x…24, mock messenger, mock ETHLiquidity; see `Concrete.lean`).

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
theorem run_success_native : isSuccess (run 0 7) = true := by native_decide

/-- `Ξ` on the concrete scenario entered statically raises `StaticModeViolation`. -/
theorem run_static_native :
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
  success_of_isSuccess run_success_native

/-! ## One partner per headline theorem -/

/-- `refundETH_trace`: hypotheses satisfied; conclusion instantiated. -/
theorem nonvacuous_refundETH_trace :
    let g : Sat256 := Sat256.ofUInt256 (UInt256.ofNat 10000000)
    RDrev ethbridgeRuntime g (initState (σ 0) (σ 0) g default (env 7)) ∨
    ((env 7).perm = false ∧ RDstatic ethbridgeRuntime g (initState (σ 0) (σ 0) g default (env 7))) ∨
    (∃ σ', RefundRun (σ 0) (σ 0) (env 7) σ' ∧
      RDret ethbridgeRuntime g (initState (σ 0) (σ 0) g default (env 7)) σ' ByteArray.empty) :=
  refundETH_trace w_code w_sel w_cds

/-- `refundETH_outcome`: on the concrete success run the outcome is the success disjunct. -/
theorem nonvacuous_refundETH_outcome :
    ∃ σ' g' A', Ξ (σ 0) (σ 0) (UInt256.ofNat 10000000) default (env 7) =
      .ok (.success (σ', g', A') ByteArray.empty) ∧ RefundRun (σ 0) (σ 0) (env 7) σ' := by
  obtain ⟨σ', g', A', o, hres⟩ := w_success
  rcases refundETH_outcome (σ := σ 0) (σ₀ := σ 0) (A := default) (g := UInt256.ofNat 10000000)
      w_code w_sel w_cds with h | ⟨_, _, h⟩ | ⟨h, _⟩ | h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · exact h

/-- `refundETH_success`: all hypotheses (incl. a successful `Ξ` run) hold; `RefundRun` follows. -/
theorem nonvacuous_refundETH_success :
    ∃ σ', RefundRun (σ 0) (σ 0) (env 7) σ' := by
  obtain ⟨σ', g', A', o, hres⟩ := w_success
  exact ⟨σ', (refundETH_success w_code w_sel w_cds hres).1⟩

/-- `refundETH_no_other_error`: the statically entered concrete run errors; the theorem says the
    error is a static-mode violation with `perm = false`. -/
theorem nonvacuous_refundETH_no_other_error :
    let I := { env 7 with perm := false }
    (ExecutionException.StaticModeViolation = .OutOfGass ∨
      (ExecutionException.StaticModeViolation = .StaticModeViolation ∧ I.perm = false)) :=
  refundETH_no_other_error (σ := σ 0) (σ₀ := σ 0) (A := default) (g := UInt256.ofNat 10000000)
    (I := { env 7 with perm := false }) rfl w_sel w_cds (static_of_isStatic run_static_native)

/-- The `RefundRun` of the concrete success run, unpacked. -/
theorem w_run : ∃ σ', RefundRun (σ 0) (σ 0) (env 7) σ' := nonvacuous_refundETH_success

/-- `createStep_success`: its hypotheses (`CreateStep` and `x ≠ 0`) are those of the concrete run. -/
theorem nonvacuous_createStep_success :
    ∃ σ₃ σ' x rd', CreateStep (env 7) (σ 0) σ₃ (argAmount (env 7)) (safeSendDeploy (argFrom (env 7)))
      x σ' rd' ∧ x ≠ UInt256.ofNat 0 ∧
      ((σ₃.get? (env 7).codeOwner |>.getD default).nonce.toNat < 2 ^ 64 - 1) := by
  obtain ⟨σ', _, _, σ₁, oE, _, _, _, _, _, _, _, σ₃, oM, _, x, rd', hcs, hx⟩ := w_run
  exact ⟨σ₃, σ', x, rd', hcs, hx, (createStep_success hcs hx).1⟩

/-- The concrete pre-state: the executing account has code (needed by the post-state lemmas). -/
theorem w_hascode : ((σ 0).getD (env 7).codeOwner default).code.size ≠ 0 := bridge_has_code

/-- `storedMap_post`: `σ₁` from the concrete run, with `hst`, `hcd`, `hcode` all satisfied. -/
theorem nonvacuous_storedMap_post :
    ∃ σ₁, UInt256.land (UInt256.ofNat 255)
      (refundedWord (storedMap σ₁ (env 7) (refundHash (env 7))) (env 7) (refundHash (env 7))) =
        UInt256.ofNat 1 := by
  obtain ⟨σ', _, _, σ₁, oE, _, _, _, hst, hcd, _⟩ := w_run
  exact ⟨σ₁, (storedMap_post hst hcd w_hascode).2.2.2⟩

/-- `refundETH_store`: hypotheses (`RefundRun`, code present) satisfied by the concrete run. -/
theorem nonvacuous_refundETH_store :
    ∃ σ' σ₂ σ₃, RefundRun (σ 0) (σ 0) (env 7) σ' ∧
      UInt256.land (UInt256.ofNat 255) (refundedWord σ₂ (env 7) (refundHash (env 7))) = UInt256.ofNat 1 ∧
      (∃ oM, CallTo (σ 0) (env 7) ethLiq (mintCalldata (argAmount (env 7))) σ₂ σ₃ true oM) := by
  obtain ⟨σ', hrun⟩ := w_run
  obtain ⟨σ₂, σ₃, _, _, _, h4, h5, _⟩ := refundETH_store hrun w_hascode
  exact ⟨σ', σ₂, σ₃, hrun, h4, h5⟩

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
