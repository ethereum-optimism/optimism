import ExporterEvm.Concrete

/-!
# Non-vacuity of every headline theorem

For each theorem checked by `#assert_headline` in `Axioms.lean` there is a `nonvacuous_<name>`
here: concrete values satisfying **all** of the theorem's hypotheses jointly, the theorem
instantiated on them, and its conclusion derived. `Axioms.lean` fails the build if a headline
theorem has no partner.

Trust: every step is kernel-checked except the facts that need *evaluating `Ξ` on the concrete
bytecode* (a successful run, a reverting run and a statically entered run). Those are isolated as
the named `native_decide` lemmas `native_run_success`, `native_run_relayed`, `native_run_static`.
`#assert_headline` checks that no headline theorem depends on them and that every non-standard
axiom of a partner comes from a `native_*` lemma (so renaming one of them breaks the build). The witness is `Concrete.σ0` /
`Concrete.env0` (exporter code at 0x…30, mock messengers; see `Concrete.lean`).
-/

namespace ExporterEvm.NonVacuous

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach Mem ExporterEvm.Concrete

/-! ## The common hypotheses on the concrete environments (kernel-checked) -/

theorem w_code : env0.code = exporterRuntime := rfl

theorem w_sel : selectorWord env0 = exportSelector := by decide +kernel

theorem w_cds : env0.calldata.size < 2 ^ 63 := by decide +kernel

theorem w_args : ArgsOk env0 :=
  ⟨by decide +kernel, by decide +kernel, by decide +kernel, by decide +kernel, by decide +kernel,
    by decide +kernel, by decide +kernel, by decide +kernel, by decide +kernel, by decide +kernel⟩

/-- The statically entered environment (same calldata). -/
abbrev envS : ExecutionEnv := env 0xaaaa 0 false

theorem w_codeS : envS.code = exporterRuntime := rfl
theorem w_selS : selectorWord envS = exportSelector := by decide +kernel
theorem w_cdsS : envS.calldata.size < 2 ^ 63 := by decide +kernel

/-- The world in which the mock messenger reports the message as relayed. -/
abbrev σR : AccountMap := σx (mockL2l2Code Hcast) mockCdmCode

/-- The world whose 0x4200…0007 code just stops (succeeds also in a static frame). -/
abbrev σS : AccountMap := σx (mockL2l2Code (Hcast + UInt256.ofNat 1)) ⟨#[0x00]⟩

/-! ## Facts that need `Ξ` evaluated on the concrete bytecode (compiled evaluation, trusted) -/

abbrev XiRes := Except ExecutionException (ExecutionResult (AccountMap × UInt256 × Substate))

def isSuccess (r : XiRes) : Bool :=
  match r with
  | .ok (.success _ _) => true
  | _ => false

def isRevert (r : XiRes) : Bool :=
  match r with
  | .ok (.revert _ _) => true
  | _ => false

def isStaticViolation (r : XiRes) : Bool :=
  match r with
  | .error .StaticModeViolation => true
  | _ => false

/-- `Ξ` on the concrete success scenario succeeds. -/
theorem native_run_success : isSuccess (runx σ0 env0) = true := by native_decide

/-- `Ξ` on the relayed scenario reverts. -/
theorem native_run_relayed : isRevert (runx σR env0) = true := by native_decide

/-- `Ξ` entered statically raises `StaticModeViolation`. -/
theorem native_run_static : isStaticViolation (runx σS envS) = true := by native_decide

theorem success_of_isSuccess {r : XiRes} (h : isSuccess r = true) :
    ∃ σ' g' A' o, r = .ok (.success (σ', g', A') o) := by
  rcases r with e | res
  · simp [isSuccess] at h
  · rcases res with ⟨⟨σ', g', A'⟩, o⟩ | ⟨g, o⟩
    · exact ⟨σ', g', A', o, rfl⟩
    · simp [isSuccess] at h

theorem revert_of_isRevert {r : XiRes} (h : isRevert r = true) : ∃ g' o, r = .ok (.revert g' o) := by
  rcases r with e | res
  · simp [isRevert] at h
  · rcases res with ⟨⟨σ', g', A'⟩, o⟩ | ⟨g, o⟩
    · simp [isRevert] at h
    · exact ⟨g, o, rfl⟩

theorem static_of_isStatic {r : XiRes} (h : isStaticViolation r = true) :
    r = .error .StaticModeViolation := by
  rcases r with e | res
  · cases e <;> simp_all [isStaticViolation]
  · simp [isStaticViolation] at h

theorem w_success : ∃ σ' g' A' o,
    Ξ σ0 σ0 (UInt256.ofNat 10000000) default env0 = .ok (.success (σ', g', A') o) :=
  success_of_isSuccess native_run_success

theorem w_relayed : ∃ g' o, Ξ σR σR (UInt256.ofNat 10000000) default env0 = .ok (.revert g' o) :=
  revert_of_isRevert native_run_relayed

theorem w_static : Ξ σS σS (UInt256.ofNat 10000000) default envS = .error .StaticModeViolation :=
  static_of_isStatic native_run_static

theorem w_hg : (Sat256.ofUInt256 (UInt256.ofNat 10000000)).toUInt256 = UInt256.ofNat 10000000 := rfl

/-! ## One partner per headline theorem -/

/-- `export_trace`: hypotheses satisfied (kernel); the theorem applies, and the disjunct realized
    is the success one: the other two would make the concrete run end in out of gas, a revert or a
    static violation, contradicting `native_run_success`. -/
theorem nonvacuous_export_trace :
    env0.code = exporterRuntime ∧ selectorWord env0 = exportSelector ∧ env0.calldata.size < 2 ^ 63 ∧
    ∃ σ', ExportRun σ0 σ0 env0 σ' ∧
      RDretL exporterRuntime (Sat256.ofUInt256 (UInt256.ofNat 10000000))
        (initState σ0 σ0 (Sat256.ofUInt256 (UInt256.ofNat 10000000)) default env0) σ'
        (UInt256.toByteArray (exportHash env0)) (exportedLog env0 (exportHash env0)) := by
  refine ⟨w_code, w_sel, w_cds, ?_⟩
  obtain ⟨σ₁, g₁, A₁, o₁, hres⟩ := w_success
  rcases export_trace (σ := σ0) (σ₀ := σ0) (A := default)
      (g := Sat256.ofUInt256 (UInt256.ofNat 10000000)) w_code w_sel w_cds with hrev | ⟨_, hstat⟩ | h
  · rcases RDrevP.xiResult w_code hrev with h | ⟨_, _, h, _⟩
    · rw [hres] at h; cases h
    · rw [hres] at h; cases h
  · rcases RDstatic.xiResult w_code hstat with h | h
    · rw [w_hg, hres] at h; cases h
    · rw [w_hg, hres] at h; cases h
  · exact h

/-- `export_outcome`: on the concrete success run the outcome is the success disjunct. -/
theorem nonvacuous_export_outcome :
    env0.code = exporterRuntime ∧ selectorWord env0 = exportSelector ∧ env0.calldata.size < 2 ^ 63 ∧
    ∃ σ' g' A', Ξ σ0 σ0 (UInt256.ofNat 10000000) default env0 =
      .ok (.success (σ', g', A') (UInt256.toByteArray (exportHash env0))) ∧ ExportRun σ0 σ0 env0 σ' ∧
      ∃ L : LogSeries, A'.logSeries = L.push (exportedLog env0 (exportHash env0)) := by
  refine ⟨w_code, w_sel, w_cds, ?_⟩
  obtain ⟨σ', g', A', o, hres⟩ := w_success
  rcases export_outcome (σ := σ0) (σ₀ := σ0) (A := default) (g := UInt256.ofNat 10000000)
      w_code w_sel w_cds with h | ⟨_, _, h, _⟩ | ⟨h, _⟩ | h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · exact h

/-- `export_success`: all hypotheses (including a successful `Ξ` run) hold jointly; the theorem
    yields `ExportRun`, the output `abi.encode(H)` and the event; `H` is the cast-computed hash
    (kernel). -/
theorem nonvacuous_export_success :
    ∃ σ' g' A' o,
      env0.code = exporterRuntime ∧ selectorWord env0 = exportSelector ∧ env0.calldata.size < 2 ^ 63 ∧
      Ξ σ0 σ0 (UInt256.ofNat 10000000) default env0 = .ok (.success (σ', g', A') o) ∧
      ExportRun σ0 σ0 env0 σ' ∧ o = UInt256.toByteArray Hcast ∧
      (∃ L : LogSeries, A'.logSeries = L.push (exportedLog env0 Hcast)) := by
  obtain ⟨σ', g', A', o, hres⟩ := w_success
  obtain ⟨hrun, ho, hL⟩ := export_success w_code w_sel w_cds hres
  rw [exportHash_matches_cast] at ho hL
  exact ⟨σ', g', A', o, w_code, w_sel, w_cds, hres, hrun, ho, hL⟩

/-- `export_revert`: all hypotheses hold for the relayed scenario, whose run reverts; the theorem
    classifies the output. -/
theorem nonvacuous_export_revert :
    ∃ g' o, env0.code = exporterRuntime ∧ selectorWord env0 = exportSelector ∧
      env0.calldata.size < 2 ^ 63 ∧
      Ξ σR σR (UInt256.ofNat 10000000) default env0 = .ok (.revert g' o) ∧ ExportRevert σR σR env0 o := by
  obtain ⟨g', o, hres⟩ := w_relayed
  exact ⟨g', o, w_code, w_sel, w_cds, hres, export_revert w_code w_sel w_cds hres⟩

/-- `export_no_other_error`: all hypotheses hold for the statically entered run, which errors
    with `StaticModeViolation`; the theorem applies and yields `perm = false`. -/
theorem nonvacuous_export_no_other_error :
    envS.code = exporterRuntime ∧ selectorWord envS = exportSelector ∧ envS.calldata.size < 2 ^ 63 ∧
    Ξ σS σS (UInt256.ofNat 10000000) default envS = .error .StaticModeViolation ∧
    envS.perm = false := by
  refine ⟨w_codeS, w_selS, w_cdsS, w_static, ?_⟩
  rcases export_no_other_error (σ := σS) (σ₀ := σS) (A := default) (g := UInt256.ofNat 10000000)
      w_codeS w_selS w_cdsS w_static with h | ⟨_, h⟩
  · cases h
  · exact h

/-- `exportPreimage_size`: `ArgsOk env0` (kernel); the preimage of the witness has
    `224 + 64 = 288` bytes. -/
theorem nonvacuous_exportPreimage_size :
    ArgsOk env0 ∧ (exportPreimage env0).size = 224 + roundUp32 (msgLen env0) ∧ roundUp32 (msgLen env0) = 64 :=
  ⟨w_args, exportPreimage_size env0 w_args, by decide +kernel⟩

end ExporterEvm.NonVacuous
