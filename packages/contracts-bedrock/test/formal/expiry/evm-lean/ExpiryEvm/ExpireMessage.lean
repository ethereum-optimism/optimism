import ExpiryEvm.Post

/-!
# `expireMessage` refinement theorems (EVM level)

The deployed runtime code `l2tol2Runtime` of `L2ToL2CrossDomainMessenger`, run by EVMLean's
code-execution function `Ξ` on calldata that selects `expireMessage(bytes32,uint256)`, from any
account map `σ`, caller, value, depth, gas and permission, refines the abstract action:

* `expireMessage_trace` — the whole symbolic execution, at the level of EquiVM's `RD` invariant.
* `expireMessage_outcome` — the same at the level of `Ξ`'s result.
* `expireMessage_success` — soundness: a successful run implies the conditions and the post-state.
Auxiliary (not headline): `expireMessage_outcome_aux` and `expireMessage_revert_cause` carry
the weak `CallFailed` (satisfiable in essentially every state); they are **not** completeness or
liveness results. Liveness evidence is the concrete run `Concrete.success_reachable`, and
`Concrete.success_instance` instantiates `expireMessage_success` with all hypotheses proved.
* `expireMessage_no_other_error` — the run never ends in any other exceptional halt.

Hypotheses common to all of them: `hcode` (the code is the pinned artifact), `hsel` (the
selector), `hcds` (calldata shorter than 2^256 bytes, which the EVM guarantees), and the two
call summaries `hO`, `hX` (`ReturnsAddress`, see `Spec.lean`). Reverts and exceptional halts
carry no state in EVMLean (`ExecutionResult.revert` has only gas and output; `Θ` restores the
caller's account map on revert or error), so "no state change on revert" is by construction.
-/

namespace ExpiryEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach Mem

theorem expireMessage_trace {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {vO vS : AccountAddress}
    (hcode : I.code = l2tol2Runtime) (hsel : selectorWord I = expireSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS) :
    (RDrev l2tol2Runtime g (initState σ σ₀ g A I) ∧
        (¬ ExpireConds σ I vO vS ∨ CallFailed σ σ₀ I)) ∨
    (ExpireConds σ I vO vS ∧ I.perm = false ∧ RDstatic l2tol2Runtime g (initState σ σ₀ g A I)) ∨
    (ExpireConds σ I vO vS ∧ I.perm = true ∧ ∃ σ', ExpirePost σ σ' I ∧
      RDret l2tol2Runtime g (initState σ σ₀ g A I) σ' ByteArray.empty) := by
  have hcdok := Words.calldata_ok_iff _ hcds
  rcases seg_entry (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode hsel with
    ⟨hrev, hnot⟩ | ⟨hlt, hv, hcd, aw, k, C, r⟩
  · left
    refine ⟨hrev, Or.inl fun hc => hnot ⟨(hcdok.mpr hc.calldataLen).1, hc.noValue,
      (hcdok.mpr hc.calldataLen).2⟩⟩
  have hlen := hcdok.mp ⟨hlt, hcd⟩
  rcases seg_call1 hO r with ⟨hrev, hwhy⟩ |
    ⟨hsrc, σ₁, o₁, j, aw1, k1, C1, hst1, hcd1, hcall1, hj5, hjb, r1⟩
  · left
    refine ⟨hrev, ?_⟩
    rcases hwhy with hne | hf
    · exact Or.inl fun hc => hne hc.callerIsL2cdm
    · exact Or.inr hf
  rcases seg_call2 hX hst1 hcd1 hcall1 hj5 hjb r1 with ⟨hrev, hwhy⟩ |
    ⟨hSO, σ₂, o₂, rest, aw2, k2, C2, hst2, hcd2, r2⟩
  · left
    refine ⟨hrev, ?_⟩
    rcases hwhy with hne | hf
    · exact Or.inl fun hc => hne hc.senderIsOther
    · exact Or.inr hf
  rcases seg_store hst2 r2 with ⟨hrev, hnot⟩ | ⟨hc, hp, hstat⟩ | ⟨hc, hp, hret⟩
  · left
    exact ⟨hrev, Or.inl fun hc => hnot ⟨hc.wasSent, hc.noOverflow, hc.expired⟩⟩
  · right; left
    exact ⟨⟨hv, hlen, hsrc, hSO, hc.1, hc.2.1, hc.2.2⟩, hp, hstat⟩
  · right; right
    exact ⟨⟨hv, hlen, hsrc, hSO, hc.1, hc.2.1, hc.2.2⟩, hp, finalMap σ₂ I,
      finalMap_post hst2 hcd2 hc.1, hret⟩

/-- `RDret.xiResult` with a final account map different from the initial one. -/
theorem rdret_xi {σ σ₀ acc : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256}
    {code o : ByteArray} (hcode : I.code = code)
    (h : RDret code (Sat256.ofUInt256 g) (initState σ σ₀ (Sat256.ofUInt256 g) A I) acc o) :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    ∃ (g' : UInt256) (A' : Substate), Ξ σ σ₀ g A I = .ok (.success (acc, g', A') o) := by
  rcases h with hoog | ⟨s, hX, hacc⟩
  · left
    exact Xi_error_of_X (by rw [hcode]; exact hoog)
  · right
    have hxi := Xi_success_of_X (σ := σ) (σ₀ := σ₀) (A := A) (I := I) (g := g)
      (by rw [hcode]; exact hX)
    rw [hacc] at hxi
    exact ⟨_, _, hxi⟩

/-- Auxiliary outcome theorem (revert case annotated with the weak `CallFailed`). Every run ends
    in exactly one of: out of gas; a revert, and then one of the success conditions fails or a call
    to the L2CrossDomainMessenger can fail; a static-mode violation (only when entered by
    `STATICCALL`, at the `SSTORE`); or success with empty output, all success conditions, and the
    post-state `ExpirePost`. -/
theorem expireMessage_outcome_aux {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256}
    {vO vS : AccountAddress}
    (hcode : I.code = l2tol2Runtime) (hsel : selectorWord I = expireSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS) :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    ((∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o)) ∧
      (¬ ExpireConds σ I vO vS ∨ CallFailed σ σ₀ I)) ∨
    (Ξ σ σ₀ g A I = .error .StaticModeViolation ∧ I.perm = false ∧ ExpireConds σ I vO vS) ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') ByteArray.empty) ∧
      I.perm = true ∧ ExpireConds σ I vO vS ∧ ExpirePost σ σ' I) := by
  have hg : (Sat256.ofUInt256 g).toUInt256 = g := rfl
  rcases expireMessage_trace (A := A) (g := Sat256.ofUInt256 g) hcode hsel hcds hO hX with
    ⟨hrev, hwhy⟩ | ⟨hc, hp, hstat⟩ | ⟨hc, hp, σ', hpost, hret⟩
  · rcases RDrev.xiResult hcode hrev with hoog | ⟨g', o, hr⟩
    · left; rw [← hg]; exact hoog
    · right; left; exact ⟨⟨g', o, by rw [← hg]; exact hr⟩, hwhy⟩
  · rcases RDstatic.xiResult hcode hstat with hoog | hs
    · left; rw [← hg]; exact hoog
    · right; right; left; exact ⟨by rw [← hg]; exact hs, hp, hc⟩
  · rcases rdret_xi hcode hret with hoog | ⟨g', A', hr⟩
    · left; exact hoog
    · right; right; right; exact ⟨σ', g', A', hr, hp, hc, hpost⟩

/-- **Outcome theorem.** Every run of the compiled code on `expireMessage` calldata ends in
    exactly one of: out of gas; a revert; a static-mode violation (only when entered by
    `STATICCALL`, with all conditions met, at the `SSTORE`); or success with empty output, all
    conditions, and the post-state `ExpirePost`. (The auxiliary `expireMessage_outcome_aux` also
    attaches `¬ ExpireConds ∨ CallFailed` to the revert case; since `CallFailed` is weak, that
    clause is uninformative and left out of the headline.) -/
theorem expireMessage_outcome {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256}
    {vO vS : AccountAddress}
    (hcode : I.code = l2tol2Runtime) (hsel : selectorWord I = expireSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS) :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    (∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o)) ∨
    (Ξ σ σ₀ g A I = .error .StaticModeViolation ∧ I.perm = false ∧ ExpireConds σ I vO vS) ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') ByteArray.empty) ∧
      I.perm = true ∧ ExpireConds σ I vO vS ∧ ExpirePost σ σ' I) := by
  rcases expireMessage_outcome_aux (g := g) (A := A) hcode hsel hcds hO hX with
    h | ⟨h, _⟩ | h | h
  · exact Or.inl h
  · exact Or.inr (Or.inl h)
  · exact Or.inr (Or.inr (Or.inl h))
  · exact Or.inr (Or.inr (Or.inr h))

/-- **Soundness.** If the compiled code succeeds on `expireMessage(H, t)`, then the caller is the
    L2CrossDomainMessenger, `xDomainMessageSender() == otherMessenger()`, no ETH was attached, the
    calldata is well-formed, `sentMessageTimestamps[H] ≠ 0`, `sentAt + P` does not overflow,
    `t > sentAt + P`, and the only storage change is `expiredMessages[H] := true`. -/
theorem expireMessage_success {σ σ₀ σ' : AccountMap} {A A' : Substate} {I : ExecutionEnv}
    {g g' : UInt256} {o : ByteArray} {vO vS : AccountAddress}
    (hcode : I.code = l2tol2Runtime) (hsel : selectorWord I = expireSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS)
    (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    I.perm = true ∧ ExpireConds σ I vO vS ∧ ExpirePost σ σ' I ∧ o = ByteArray.empty := by
  rcases expireMessage_outcome (g := g) (A := A) hcode hsel hcds hO hX with
    h | ⟨_, _, h⟩ | ⟨h, _⟩ | ⟨σ'', g'', A'', h, hp, hc, hpost⟩
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h
    cases h
    exact ⟨hp, hc, hpost, rfl⟩

/-- **Not a completeness theorem.** If all conditions hold and the run is not static, the run
    ends in out-of-gas, success with `ExpirePost`, or a revert together with `CallFailed`.
    `CallFailed` is satisfiable in essentially every state (existentially quantified call gas;
    see `Concrete.callFailed_in_success_state`), so this does **not** exclude reverts; it only
    records the outcome classification under the conditions. Liveness is out of reach of EquiVM's
    reached-or-out-of-gas invariant; `Concrete.success_reachable` is the evidence that the success
    branch is taken. -/
theorem expireMessage_revert_cause {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv}
    {g : UInt256} {vO vS : AccountAddress}
    (hcode : I.code = l2tol2Runtime) (hsel : selectorWord I = expireSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS)
    (hc : ExpireConds σ I vO vS) (hperm : I.perm = true) :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') ByteArray.empty) ∧
      ExpirePost σ σ' I) ∨
    ((∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o)) ∧ CallFailed σ σ₀ I) := by
  rcases expireMessage_outcome_aux (g := g) (A := A) hcode hsel hcds hO hX with
    h | ⟨hr, hwhy⟩ | ⟨_, hp, _⟩ | ⟨σ', g', A', h, _, _, hpost⟩
  · exact Or.inl h
  · rcases hwhy with hn | hf
    · exact absurd hc hn
    · exact Or.inr (Or.inr ⟨hr, hf⟩)
  · rw [hperm] at hp; cases hp
  · exact Or.inr (Or.inl ⟨σ', g', A', h, hpost⟩)

/-- The compiled code never ends in an exceptional halt other than out-of-gas or (when entered
    by `STATICCALL`) a static-mode violation: no `INVALID`, stack under/overflow, bad jump, etc. -/
theorem expireMessage_no_other_error {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv}
    {g : UInt256} {vO vS : AccountAddress} {e : ExecutionException}
    (hcode : I.code = l2tol2Runtime) (hsel : selectorWord I = expireSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS)
    (he : Ξ σ σ₀ g A I = .error e) :
    e = .OutOfGass ∨ (e = .StaticModeViolation ∧ I.perm = false) := by
  rcases expireMessage_outcome (g := g) (A := A) hcode hsel hcds hO hX with
    h | ⟨_, _, h⟩ | ⟨h, hp, _⟩ | ⟨_, _, _, h, _⟩
  · rw [he] at h; cases h; exact Or.inl rfl
  · rw [he] at h; cases h
  · rw [he] at h; cases h; exact Or.inr ⟨rfl, hp⟩
  · rw [he] at h; cases h

end ExpiryEvm
