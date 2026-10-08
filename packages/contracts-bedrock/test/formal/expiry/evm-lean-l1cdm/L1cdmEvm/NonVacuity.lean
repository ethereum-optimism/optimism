import L1cdmEvm.Mock

/-!
# Non-vacuity witnesses, one per headline theorem

For every headline theorem `T` (the list in `Axioms.lean`), `nonvacuous_T` exhibits one concrete
instance on which **all** of `T`'s hypotheses hold **jointly**, applies `T` to it, and shows which
case of `T`'s conclusion the instance realizes. The instance is `Concrete.world .ok`: the pinned
`L1CrossDomainMessenger` runtime at `SELF`, the mock SystemConfig / portals / ETHLockbox / caller
(B's messenger) of `Concrete.lean`, and the call `relayUndeliveredMessage(H, T)` from `CALLER`
with 3·10^6 gas; for the inner frame, the self-call frame `selfCallEnv env (sendMessageCd H T)`.

**Kernel-checked:** the code (`rfl`), the selector and calldata bound, the inner frame's calldata
and value, all seven call summaries (`concrete_summaries`, an `RD` proof of each mock's bytecode),
`hself` and `hnp` (`concrete_self`), and every field of `RelayConds` evaluated directly on the
concrete views (`witness_conds`).

**Compiled evaluation (`native_decide`):** only facts "`Ξ` on this concrete input ends in success /
out of gas": `native_relay_success`, `native_relay_oog`, `native_send_success`. The kernel cannot
evaluate `Ξ` (well-founded recursion). They supply the hypotheses about `Ξ`'s result (`hres`, `he`)
and identify which disjunct of an outcome theorem is realized. `Axioms.lean` checks that every
non-standard axiom of a witness comes from a `native_*` theorem and that no headline theorem
depends on one.
-/

namespace L1cdmEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.Concrete

namespace NV

/-- The views the mocks return in `world .ok`. -/
def v₁ : Views :=
  ⟨UInt256.ofNat 1, word P_B, word SC_B, word CALLER, word LB, UInt256.ofNat 1, exporterWord⟩

/-- The self-call frame `relayUndeliveredMessage` creates. -/
abbrev Iₛ : ExecutionEnv := selfCallEnv env (sendMessageCd H T) l1cdmRuntime

abbrev R := Except ExecutionException (ExecutionResult (AccountMap × UInt256 × Substate))

def isSuccess : R → Bool
  | .ok (.success _ _) => true
  | _ => false

def isOOG : R → Bool
  | .error .OutOfGass => true
  | _ => false

/-- **NATIVE (compiled evaluation).** The outer call succeeds in `world .ok`. -/
theorem native_relay_success : isSuccess (run .ok) = true := by native_decide

/-- **NATIVE (compiled evaluation).** The outer call with 100 gas runs out of gas. -/
theorem native_relay_oog :
    isOOG (Ξ (world .ok) (world .ok) (UInt256.ofNat 100) default env) = true := by native_decide

/-- **NATIVE (compiled evaluation).** The self-call frame `sendMessage(...)` succeeds in
    `world .ok` (it calls the portal mock's `depositTransaction`). -/
theorem native_send_success :
    isSuccess (Ξ (world .ok) (world .ok) (UInt256.ofNat 3000000) default Iₛ) = true := by
  native_decide

theorem of_isSuccess {r : R} (h : isSuccess r = true) :
    ∃ σ' g' A' o, r = .ok (.success (σ', g', A') o) := by
  cases r with
  | error e => cases h
  | ok r =>
    cases r with
    | revert g' o => cases h
    | success t o => obtain ⟨σ', g', A'⟩ := t; exact ⟨σ', g', A', o, rfl⟩

theorem of_isOOG {r : R} (h : isOOG r = true) : r = .error .OutOfGass := by
  unfold isOOG at h
  split at h <;> first | rfl | cases h

/-! Kernel-checked facts of the witness. -/

theorem env_code : env.code = l1cdmRuntime := rfl
theorem env_sel : selectorWord env = relaySelector := by decide +kernel
theorem env_cds : env.calldata.size < 2 ^ 256 := by decide +kernel
theorem s_code : Iₛ.code = l1cdmRuntime := rfl
theorem s_cd : Iₛ.calldata = sendMessageCd H T := rfl
theorem s_val : Iₛ.weiValue = ⟨0⟩ := rfl
theorem env_args : argHash env = H ∧ argTime env = T := by decide +kernel

/-- Every field of `RelayConds`, evaluated directly on the concrete environment and views (no `Ξ`
    run): the three checks (a), (b), (c) and the INTEROP feature hold in the witness. -/
theorem witness_conds : RelayConds env v₁ where
  noValue := rfl
  calldataLen := by decide +kernel
  interop := rfl
  callerPortalClean := show _ < _ by decide +kernel
  callerSCClean := show _ < _ by decide +kernel
  lockboxClean := show _ < _ by decide +kernel
  isMessenger := rfl
  authorized := rfl
  fromExporter := rfl

end NV

open NV

/-- All hypotheses of `relay_outcome` hold jointly (kernel-checked); the theorem applies, and the
    disjunct realized is success with `RelayConds` and `RelayPost`. -/
theorem nonvacuous_relay_outcome :
    env.code = l1cdmRuntime ∧ selectorWord env = relaySelector ∧ env.calldata.size < 2 ^ 256 ∧
    Summaries (world .ok) (world .ok) env v₁ ∧
    ∃ σ' g' A', run .ok = .ok (.success (σ', g', A') ByteArray.empty) ∧
      RelayConds env v₁ ∧ RelayPost (world .ok) (world .ok) env σ' := by
  refine ⟨env_code, env_sel, env_cds, concrete_summaries, ?_⟩
  obtain ⟨σ₁, g₁, A₁, o₁, hs⟩ := of_isSuccess native_relay_success
  rcases relay_outcome (g := UInt256.ofNat 3000000) (A := default) env_code env_sel env_cds
      concrete_summaries with h | ⟨_, _, h⟩ | h
  · exact absurd (h.symm.trans hs) (by intro h; cases h)
  · exact absurd (h.symm.trans hs) (by intro h; cases h)
  · exact h

/-- All hypotheses of `relay_success`, including a successful run, hold jointly; the theorem
    yields `RelayConds` (which `witness_conds` also checks directly) and `RelayPost`. -/
theorem nonvacuous_relay_success :
    ∃ σ' g' A' o,
      env.code = l1cdmRuntime ∧ selectorWord env = relaySelector ∧ env.calldata.size < 2 ^ 256 ∧
      Summaries (world .ok) (world .ok) env v₁ ∧
      run .ok = .ok (.success (σ', g', A') o) ∧
      RelayConds env v₁ ∧ RelayPost (world .ok) (world .ok) env σ' ∧ o = ByteArray.empty := by
  obtain ⟨σ', g', A', o, hs⟩ := of_isSuccess native_relay_success
  exact ⟨σ', g', A', o, env_code, env_sel, env_cds, concrete_summaries, hs,
    relay_success env_code env_sel env_cds concrete_summaries hs⟩

/-- All hypotheses of `relay_no_other_error` hold jointly for a run that does end in an exceptional
    halt (100 gas: out of gas); the theorem applies. -/
theorem nonvacuous_relay_no_other_error :
    env.code = l1cdmRuntime ∧ selectorWord env = relaySelector ∧ env.calldata.size < 2 ^ 256 ∧
    Summaries (world .ok) (world .ok) env v₁ ∧
    Ξ (world .ok) (world .ok) (UInt256.ofNat 100) default env = .error .OutOfGass ∧
    ExecutionException.OutOfGass = .OutOfGass := by
  have he := of_isOOG native_relay_oog
  exact ⟨env_code, env_sel, env_cds, concrete_summaries, he,
    relay_no_other_error env_code env_sel env_cds concrete_summaries he⟩

/-- All hypotheses of `send_outcome` hold jointly on the self-call frame (kernel-checked); the
    theorem applies, and the disjunct realized is success with `perm = true` and `SendPost`. -/
theorem nonvacuous_send_outcome :
    Iₛ.code = l1cdmRuntime ∧ Iₛ.calldata = sendMessageCd H T ∧ Iₛ.weiValue = ⟨0⟩ ∧
    ∃ σ' g' A', Ξ (world .ok) (world .ok) (UInt256.ofNat 3000000) default Iₛ =
        .ok (.success (σ', g', A') ByteArray.empty) ∧ Iₛ.perm = true ∧
      SendPost (world .ok) (world .ok) Iₛ H T σ' := by
  refine ⟨s_code, s_cd, s_val, ?_⟩
  obtain ⟨σ₁, g₁, A₁, o₁, hs⟩ := of_isSuccess native_send_success
  rcases send_outcome (σ := world .ok) (σ₀ := world .ok) (g := UInt256.ofNat 3000000)
      (A := default) s_code s_cd s_val with h | ⟨_, _, h⟩ | ⟨h, _⟩ | h
  · exact absurd (h.symm.trans hs) (by intro h; cases h)
  · exact absurd (h.symm.trans hs) (by intro h; cases h)
  · exact absurd (h.symm.trans hs) (by intro h; cases h)
  · exact h

/-- All hypotheses of `send_success`, including a successful run of the self-call frame, hold
    jointly; the theorem yields `perm = true` and `SendPost` (the deposit call with the exact
    envelope, then `++msgNonce`). -/
theorem nonvacuous_send_success :
    ∃ σ' g' A' o,
      Iₛ.code = l1cdmRuntime ∧ Iₛ.calldata = sendMessageCd H T ∧ Iₛ.weiValue = ⟨0⟩ ∧
      Ξ (world .ok) (world .ok) (UInt256.ofNat 3000000) default Iₛ =
        .ok (.success (σ', g', A') o) ∧
      Iₛ.perm = true ∧ SendPost (world .ok) (world .ok) Iₛ H T σ' ∧ o = ByteArray.empty := by
  obtain ⟨σ', g', A', o, hs⟩ := of_isSuccess native_send_success
  exact ⟨σ', g', A', o, s_code, s_cd, s_val, hs, send_success s_code s_cd s_val hs⟩

/-- All hypotheses of `relay_deposit` (the unproxied composition) hold jointly — code, selector,
    calldata bound, the seven summaries, `hself`, `hnp` (all kernel-checked) and a successful run —
    and the theorem yields the deposit call with the exact envelope, from this contract to its
    portal, followed by `++msgNonce`. -/
theorem nonvacuous_relay_deposit :
    ∃ σ' g' A' o,
      env.code = l1cdmRuntime ∧ selectorWord env = relaySelector ∧ env.calldata.size < 2 ^ 256 ∧
      Summaries (world .ok) (world .ok) env v₁ ∧
      ((world .ok).getD env.codeOwner default).code = l1cdmRuntime ∧ env.codeOwner ∉ π ∧
      run .ok = .ok (.success (σ', g', A') o) ∧
      RelayConds env v₁ ∧ env.perm = true ∧ o = ByteArray.empty ∧
      ∃ σc σp out, accountStorageStateEq (world .ok) σc ∧ accountCodeStateEq (world .ok) σc ∧
        DepositCall (world .ok) (selfCallEnv env (sendMessageCd (argHash env) (argTime env)) l1cdmRuntime)
          (depositCd (otherMessengerWord (world .ok) env.codeOwner)
            (versionedNonce (msgNonceWord (world .ok) env.codeOwner))
            (addrWord env.codeOwner) (argHash env) (argTime env)) σc σp true out ∧
        σ' = sstoreAccountMap env.codeOwner σp (UInt256.ofNat 205)
          (bumpNonceWord (msgNonceWord σp env.codeOwner)) := by
  obtain ⟨σ', g', A', o, hs⟩ := of_isSuccess native_relay_success
  exact ⟨σ', g', A', o, env_code, env_sel, env_cds, concrete_summaries, concrete_self.1,
    concrete_self.2, hs,
    relay_deposit env_code env_sel env_cds concrete_summaries concrete_self.1 concrete_self.2 hs⟩

end L1cdmEvm
