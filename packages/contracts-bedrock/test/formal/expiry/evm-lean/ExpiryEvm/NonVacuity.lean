import ExpiryEvm.Abstract
import ExpiryEvm.Concrete

/-!
# Non-vacuity witnesses, one per headline theorem

For every headline theorem `T` (the list in `Axioms.lean`), `nonvacuous_T` exhibits one concrete
instance on which **all** of `T`'s hypotheses hold **jointly**, applies `T` to it, and shows which
case of `T`'s conclusion the instance realizes. The instance is `Concrete.lean`'s world: the pinned
`L2ToL2CrossDomainMessenger` runtime at 0x4200..0023, a mock L2CrossDomainMessenger at 0x4200..0007
that returns the same address for both view calls, `sentMessageTimestamps[H] = 5`, and the call
`expireMessage(H, 5 + P_contract + 1)` from 0x4200..0007 with 10^6 gas.

**What the kernel checks.** Every hypothesis of every headline theorem on this instance: the code
(`rfl`), the selector and calldata bound (`decide +kernel`), both call summaries
(`Concrete.mock_returnsAddress`, an `RD` proof of the mock bytecode, kernel decodes), and, for
`expireMessage_revert_cause`, all seven fields of `ExpireConds` (`witness_conds`; `decide +kernel`
evaluates keccak256 for the storage slot). The keccak slot side conditions of `refines_expire`
(`sentAtSlot H ≠ expiredSlot H`, `expiredSlot H' ≠ expiredSlot H`) are kernel-checked too.

**What is compiled evaluation (`native_decide`).** Only facts of the form "`Ξ` on this concrete
input ends in success / out of gas / static violation": `Concrete.native_xi_success`,
`native_xi_oog`, `native_xi_static`. The kernel cannot evaluate `Ξ` (its well-founded recursion
does not reduce). They are needed only for the hypotheses about `Ξ`'s result (`hres`, `he`) and to
identify which disjunct of an outcome theorem is realized. `Axioms.lean` checks that every
non-standard axiom of a witness comes from a theorem named `native_*`, and that no headline theorem
depends on any of them.
-/

namespace ExpiryEvm

open Ethereum Ethereum.EVM ExpiryEvm.Concrete

namespace NV

/-- The witness environment: `expireMessage(H, 5 + P + 1)` from the L2CrossDomainMessenger. -/
abbrev I₁ : ExecutionEnv := env l2cdm tS

/-- The same call entered by `STATICCALL` (`perm = false`). -/
abbrev Iₛ : ExecutionEnv := { env l2cdm tS with perm := false }

def isOOG : Except ExecutionException (ExecutionResult (AccountMap × UInt256 × Substate)) → Bool
  | .error .OutOfGass => true
  | _ => false

def isStatic : Except ExecutionException (ExecutionResult (AccountMap × UInt256 × Substate)) → Bool
  | .error .StaticModeViolation => true
  | _ => false

/-- **NATIVE (compiled evaluation).** The witness call with 20000 gas runs out of gas. -/
theorem native_xi_oog : isOOG (Ξ σ σ (UInt256.ofNat 20000) default I₁) = true := by
  native_decide

/-- **NATIVE (compiled evaluation).** The witness call entered by `STATICCALL` halts with a
    static-mode violation (at the `SSTORE`). -/
theorem native_xi_static :
    isStatic (Ξ σ σ (UInt256.ofNat 1000000) default Iₛ) = true := by
  native_decide

theorem xi_oog : Ξ σ σ (UInt256.ofNat 20000) default I₁ = .error .OutOfGass := by
  have h := native_xi_oog
  revert h
  generalize Ξ σ σ (UInt256.ofNat 20000) default I₁ = r
  intro h
  unfold isOOG at h
  split at h <;> first | rfl | cases h

theorem xi_static : Ξ σ σ (UInt256.ofNat 1000000) default Iₛ = .error .StaticModeViolation := by
  have h := native_xi_static
  revert h
  generalize Ξ σ σ (UInt256.ofNat 1000000) default Iₛ = r
  intro h
  unfold isStatic at h
  split at h <;> first | rfl | cases h

/-- The witness run succeeds (from `Concrete.native_xi_success`). -/
theorem xi_success :
    ∃ σ' g' A' o, Ξ σ σ (UInt256.ofNat 1000000) default I₁ = .ok (.success (σ', g', A') o) := by
  have hs := native_xi_success
  revert hs
  generalize Ξ σ σ (UInt256.ofNat 1000000) default I₁ = r
  intro hs
  cases r with
  | error e => cases hs
  | ok r =>
    cases r with
    | revert g' o => cases hs
    | success t o => obtain ⟨σ', g', A'⟩ := t; exact ⟨σ', g', A', o, rfl⟩

/-- **Kernel-checked**: every success condition holds on the witness, evaluated directly on the
    concrete state (keccak256 included), without running `Ξ`. -/
theorem witness_conds : ExpireConds σ I₁ vMock vMock where
  noValue := rfl
  calldataLen := by decide +kernel
  callerIsL2cdm := rfl
  senderIsOther := rfl
  wasSent := by decide +kernel
  noOverflow := by decide +kernel
  expired := by decide +kernel

theorem sel_static : selectorWord Iₛ = expireSelector := by decide +kernel
theorem cds_static : Iₛ.calldata.size < 2 ^ 256 := by decide +kernel

/-- `expiredMessages[H]` of the messenger before the run (kernel: keccak slot lookup). -/
theorem expired_before : storageWord σ messengerAddr (expiredSlot H) = ⟨0⟩ := by decide +kernel

theorem setTrue_zero : setTrueWord ⟨0⟩ = UInt256.ofNat 1 := by decide +kernel

end NV

open NV

/-- All hypotheses of `expireMessage_outcome` hold jointly on the witness (kernel-checked); the
    theorem applies, and the disjunct realized is success with `ExpireConds` and `ExpirePost`. -/
theorem nonvacuous_expireMessage_outcome :
    I₁.code = l2tol2Runtime ∧ selectorWord I₁ = expireSelector ∧ I₁.calldata.size < 2 ^ 256 ∧
    ReturnsAddress σ σ I₁ otherMessengerCalldata vMock ∧
    ReturnsAddress σ σ I₁ xDomainMessageSenderCalldata vMock ∧
    ∃ σ' g' A', Ξ σ σ (UInt256.ofNat 1000000) default I₁ =
        .ok (.success (σ', g', A') ByteArray.empty) ∧
      I₁.perm = true ∧ ExpireConds σ I₁ vMock vMock ∧ ExpirePost σ σ' I₁ := by
  have hO := mock_returnsAddress σ I₁ otherMessengerCalldata
  have hX := mock_returnsAddress σ I₁ xDomainMessageSenderCalldata
  refine ⟨rfl, env_sel, env_cds, hO, hX, ?_⟩
  obtain ⟨σ₁, g₁, A₁, o₁, hs⟩ := xi_success
  rcases expireMessage_outcome (g := UInt256.ofNat 1000000) (A := default) rfl env_sel
      env_cds hO hX with h | ⟨_, _, h⟩ | ⟨h, _⟩ | h
  · rw [hs] at h; cases h
  · rw [hs] at h; cases h
  · rw [hs] at h; cases h
  · exact h

/-- All hypotheses of `expireMessage_success`, including a successful run, hold jointly; the
    theorem yields the conditions and the post-state, and from `ExpirePost` (not from the
    evaluation) `expiredMessages[H]` is 1 afterwards. -/
theorem nonvacuous_expireMessage_success :
    ∃ σ' g' A' o,
      I₁.code = l2tol2Runtime ∧ selectorWord I₁ = expireSelector ∧ I₁.calldata.size < 2 ^ 256 ∧
      ReturnsAddress σ σ I₁ otherMessengerCalldata vMock ∧
      ReturnsAddress σ σ I₁ xDomainMessageSenderCalldata vMock ∧
      Ξ σ σ (UInt256.ofNat 1000000) default I₁ = .ok (.success (σ', g', A') o) ∧
      I₁.perm = true ∧ ExpireConds σ I₁ vMock vMock ∧ ExpirePost σ σ' I₁ ∧ o = ByteArray.empty ∧
      storageWord σ' messengerAddr (expiredSlot H) = UInt256.ofNat 1 := by
  have hO := mock_returnsAddress σ I₁ otherMessengerCalldata
  have hX := mock_returnsAddress σ I₁ xDomainMessageSenderCalldata
  obtain ⟨σ', g', A', o, hs⟩ := xi_success
  obtain ⟨hp, hc, hpost, ho⟩ := expireMessage_success rfl env_sel env_cds hO hX hs
  refine ⟨σ', g', A', o, rfl, env_sel, env_cds, hO, hX, hs, hp, hc, hpost, ho, ?_⟩
  have hself := hpost.self_storage
  unfold storageWord
  rw [show messengerAddr = I₁.codeOwner from rfl, hself,
    show argHash I₁ = H by decide +kernel, Abstract.storageWord_insert_self]
  rw [show I₁.codeOwner = messengerAddr from rfl, expired_before, setTrue_zero]

/-- All hypotheses of `expireMessage_revert_cause` — including `ExpireConds` and `perm = true` —
    hold jointly and are **all kernel-checked** (no `Ξ` evaluation); the theorem applies, and the
    disjunct realized is success with `ExpirePost`. -/
theorem nonvacuous_expireMessage_revert_cause :
    I₁.code = l2tol2Runtime ∧ selectorWord I₁ = expireSelector ∧ I₁.calldata.size < 2 ^ 256 ∧
    ReturnsAddress σ σ I₁ otherMessengerCalldata vMock ∧
    ReturnsAddress σ σ I₁ xDomainMessageSenderCalldata vMock ∧
    ExpireConds σ I₁ vMock vMock ∧ I₁.perm = true ∧
    (Ξ σ σ (UInt256.ofNat 1000000) default I₁ = .error .OutOfGass ∨
      (∃ σ' g' A', Ξ σ σ (UInt256.ofNat 1000000) default I₁ =
          .ok (.success (σ', g', A') ByteArray.empty) ∧ ExpirePost σ σ' I₁) ∨
      ((∃ g' o, Ξ σ σ (UInt256.ofNat 1000000) default I₁ = .ok (.revert g' o)) ∧
        CallFailed σ σ I₁)) ∧
    ∃ σ' g' A', Ξ σ σ (UInt256.ofNat 1000000) default I₁ =
        .ok (.success (σ', g', A') ByteArray.empty) ∧ ExpirePost σ σ' I₁ := by
  have hO := mock_returnsAddress σ I₁ otherMessengerCalldata
  have hX := mock_returnsAddress σ I₁ xDomainMessageSenderCalldata
  have hT := expireMessage_revert_cause (g := UInt256.ofNat 1000000) (A := default) rfl
    env_sel env_cds hO hX witness_conds rfl
  refine ⟨rfl, env_sel, env_cds, hO, hX, witness_conds, rfl, hT, ?_⟩
  obtain ⟨σ₁, g₁, A₁, o₁, hs⟩ := xi_success
  rcases hT with h | h | ⟨⟨_, _, h⟩, _⟩
  · rw [hs] at h; cases h
  · exact h
  · rw [hs] at h; cases h

/-- All hypotheses of `expireMessage_no_other_error` hold jointly for two runs that do end in an
    exceptional halt: with 20000 gas (out of gas) and entered by `STATICCALL` (static violation);
    the theorem applies to both, and for the second it yields `perm = false`. -/
theorem nonvacuous_expireMessage_no_other_error :
    (I₁.code = l2tol2Runtime ∧ selectorWord I₁ = expireSelector ∧ I₁.calldata.size < 2 ^ 256 ∧
      ReturnsAddress σ σ I₁ otherMessengerCalldata vMock ∧
      ReturnsAddress σ σ I₁ xDomainMessageSenderCalldata vMock ∧
      Ξ σ σ (UInt256.ofNat 20000) default I₁ = .error .OutOfGass ∧
      (ExecutionException.OutOfGass = .OutOfGass ∨
        (ExecutionException.OutOfGass = .StaticModeViolation ∧ I₁.perm = false))) ∧
    (Iₛ.code = l2tol2Runtime ∧ selectorWord Iₛ = expireSelector ∧ Iₛ.calldata.size < 2 ^ 256 ∧
      ReturnsAddress σ σ Iₛ otherMessengerCalldata vMock ∧
      ReturnsAddress σ σ Iₛ xDomainMessageSenderCalldata vMock ∧
      Ξ σ σ (UInt256.ofNat 1000000) default Iₛ = .error .StaticModeViolation ∧
      Iₛ.perm = false) := by
  have hO := mock_returnsAddress σ I₁ otherMessengerCalldata
  have hX := mock_returnsAddress σ I₁ xDomainMessageSenderCalldata
  have hOs := mock_returnsAddress σ Iₛ otherMessengerCalldata
  have hXs := mock_returnsAddress σ Iₛ xDomainMessageSenderCalldata
  refine ⟨⟨rfl, env_sel, env_cds, hO, hX, xi_oog,
    expireMessage_no_other_error rfl env_sel env_cds hO hX xi_oog⟩,
    rfl, sel_static, cds_static, hOs, hXs, xi_static, ?_⟩
  rcases expireMessage_no_other_error rfl sel_static cds_static hOs hXs xi_static with h | ⟨_, h⟩
  · cases h
  · exact h

namespace Abstract

/-- Another key, whose `expiredMessages` slot differs from `H`'s (kernel: keccak256). -/
def H' : UInt256 := UInt256.ofNat 0x1235

/-! Kernel-checked facts of the witness (each its own declaration: `decide +kernel` evaluates
keccak256 several times). -/

theorem argHash_I₁ : argHash I₁ = H := by decide +kernel
theorem slot_ne_sent : sentAtSlot H ≠ expiredSlot (argHash I₁) := by decide +kernel
theorem slot_ne_other : expiredSlot H' ≠ expiredSlot (argHash I₁) := by decide +kernel
theorem sentAt_before : (viewOf σ I₁.codeOwner).sentAt H = 5 := by decide +kernel
theorem other_not_expired_before :
    ¬ (UInt256.land (storageWord σ I₁.codeOwner (expiredSlot H')) (UInt256.ofNat 0xff) ≠ ⟨0⟩) := by
  decide +kernel
theorem other_ne : H' ≠ argHash I₁ := by decide +kernel

/-- All hypotheses of `refines_expire`, including a successful run, hold jointly; the theorem
    applies; its two per-key side conditions are met at the witness keys (kernel-checked keccak
    inequalities), so its conditional clauses fire: `sentAt H` stays 5, `expired H` becomes true,
    and `expired H'` is unchanged (false). -/
theorem nonvacuous_refines_expire :
    argHash I₁ = H ∧ I₁.codeOwner = messengerAddr ∧
    sentAtSlot H ≠ expiredSlot (argHash I₁) ∧ expiredSlot H' ≠ expiredSlot (argHash I₁) ∧
    ∃ σ' g' A' o,
      I₁.code = l2tol2Runtime ∧ selectorWord I₁ = expireSelector ∧ I₁.calldata.size < 2 ^ 256 ∧
      ReturnsAddress σ σ I₁ otherMessengerCalldata vMock ∧
      ReturnsAddress σ σ I₁ xDomainMessageSenderCalldata vMock ∧
      Ξ σ σ (UInt256.ofNat 1000000) default I₁ = .ok (.success (σ', g', A') o) ∧
      absGuard P_contract (RelayFromOtherMessenger I₁ vMock vMock) (viewOf σ I₁.codeOwner)
        (argHash I₁) (argTime I₁).toNat ∧
      (viewOf σ' I₁.codeOwner).expired (argHash I₁) ∧
      (viewOf σ' I₁.codeOwner).sentAt H = 5 ∧
      ¬ (viewOf σ' I₁.codeOwner).expired H' := by
  have hO := mock_returnsAddress σ I₁ otherMessengerCalldata
  have hX := mock_returnsAddress σ I₁ xDomainMessageSenderCalldata
  have hH := argHash_I₁
  have hs1 := slot_ne_sent
  have hs2 := slot_ne_other
  have hsent := sentAt_before
  have hexp' := other_not_expired_before
  have hne := other_ne
  obtain ⟨σ', g', A', o, hs⟩ := xi_success
  obtain ⟨hg, he, hsa, hex, _, _⟩ := refines_expire rfl env_sel env_cds hO hX hs
  refine ⟨hH, rfl, hs1, hs2, σ', g', A', o, rfl, env_sel, env_cds, hO, hX, hs, hg, he,
    (hsa H hs1).trans hsent, fun h => ?_⟩
  rcases (hex H' hs2).mp h with h1 | h1
  · dsimp only [viewOf] at h1
    exact hexp' h1
  · exact hne h1

end Abstract

end ExpiryEvm
