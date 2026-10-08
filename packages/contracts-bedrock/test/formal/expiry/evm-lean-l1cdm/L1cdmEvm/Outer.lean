import L1cdmEvm.OuterCall1
import L1cdmEvm.OuterCall2
import L1cdmEvm.OuterCall3
import L1cdmEvm.OuterCall4
import L1cdmEvm.OuterCall5
import L1cdmEvm.OuterFinal

/-!
# `relayUndeliveredMessage` (EVM level): soundness of the compiled code

`relay_trace` chains the trace segments; `relay_outcome`/`relay_success`/`relay_no_other_error`
state the result at the level of EVMLean's code-execution function `Ξ`.
-/

namespace L1cdmEvm

set_option maxRecDepth 20000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

/-- What a successful run did: the seven view calls (static, so storage and code are unchanged,
    `σc`), then exactly one state-changing operation, the self-call `this.sendMessage(...)` with
    calldata `sendMessageCd H t`, which succeeded; the final account map is that call's result. -/
def RelayPost (σ σ₀ : AccountMap) (I : ExecutionEnv) (σ' : AccountMap) : Prop :=
  ∃ σc o, accountStorageStateEq σ σc ∧ accountCodeStateEq σ σc ∧
    extCodeSizeWord σc (UInt256.ofNat I.codeOwner) ≠ ⟨0⟩ ∧
    SelfCall σ₀ I (sendMessageCd (argHash I) (argTime I)) σc σ' true o

theorem relay_trace {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256} {v : Views}
    (hcode : I.code = l1cdmRuntime) (hsel : selectorWord I = relaySelector)
    (hcds : I.calldata.size < 2 ^ 256) (hS : Summaries σ σ₀ I v) :
    RDrev l1cdmRuntime g (initState σ σ₀ g A I) ∨
    (RelayConds I v ∧ ∃ σ', RelayPost σ σ₀ I σ' ∧
      RDret l1cdmRuntime g (initState σ σ₀ g A I) σ' ByteArray.empty) := by
  have hcdok := Words.calldata_ok_iff _ hcds
  rcases seg_entry (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode hsel with
    hrev | ⟨hv, hlt, hcd, m0, aw0, k0, C0, hF0, r0⟩
  · exact Or.inl hrev
  have hlen := hcdok.mp ⟨hlt, hcd⟩
  have hB0 : (UInt256.ofNat 128).toNat = 128 := rfl
  rcases seg_feat (bound_of_returns hS.feat) (accountStorageStateEq_refl σ) (accountCodeStateEq_refl σ) hF0
      (by rw [hB0]; norm_num) (by rw [hB0]; norm_num) r0 with
    hrev | ⟨wF, hrF, hfeat, σ1, m1, B1, aw1, rd1, k1, C1, hs1, hc1, hF1, hB1, hB1', r1⟩
  · exact Or.inl hrev
  rw [hB0] at hB1'
  rw [returned_eq hrF hS.feat] at hfeat
  rcases seg_portal (bound_of_returns hS.callerPortal) hs1 hc1 hF1 hB1 (by omega) r1 with
    hrev | ⟨wP, hrP, hPc, σ2, m2, B2, aw2, rd2, k2, C2, hs2, hc2, hF2, hB2, hB2', r2⟩
  · exact Or.inl hrev
  obtain rfl := returned_eq hrP hS.callerPortal
  rcases seg_callerSC (bound_of_returns hS.callerSC) hs2 hc2 hF2 hB2 (by omega) r2 with
    hrev | ⟨wS, hrS, hSc, σ3, m3, B3, aw3, rd3, k3, C3, hs3, hc3, hF3, hB3, hB3', r3⟩
  · exact Or.inl hrev
  obtain rfl := returned_eq hrS hS.callerSC
  rcases seg_callerMsgr (bound_of_returns hS.callerMsgr) hs3 hc3 hF3 hB3 (by omega) r3 with
    hrev | ⟨wM, hrM, hMsgr, σ4, m4, B4, aw4, rd4, k4, C4, hs4, hc4, hF4, hB4, hB4', r4⟩
  · exact Or.inl hrev
  obtain rfl := returned_eq hrM hS.callerMsgr
  rcases seg_lockbox (bound_of_returns hS.lockbox) hs4 hc4 hF4 hB4 (by omega) r4 with
    hrev | ⟨wL, hrL, hLc, σ5, m5, B5, aw5, rd5, k5, C5, hs5, hc5, hF5, hB5, hB5', r5⟩
  · exact Or.inl hrev
  obtain rfl := returned_eq hrL hS.lockbox
  rcases seg_auth (bound_of_returns hS.auth) hPc hs5 hc5 hF5 hB5 (by omega) r5 with
    hrev | ⟨wA, hrA, hAuth, σ6, m6, B6, aw6, rd6, k6, C6, hs6, hc6, hF6, hB6, hB6', r6⟩
  · exact Or.inl hrev
  rw [returned_eq hrA hS.auth] at hAuth
  rcases seg_xsender (bound_of_returns hS.xSender) hs6 hc6 hF6 hB6 (by omega) r6 with
    hrev | ⟨wX, hrX, hX, σ7, m7, B7, aw7, rd7, k7, C7, hs7, hc7, hF7, hB7, hB7', r7⟩
  · exact Or.inl hrev
  rw [returned_eq hrX hS.xSender] at hX
  rcases seg_final hs7 hc7 hF7 hB7 (by omega) r7 with hrev | ⟨hecs, σ', o, hcall, hret⟩
  · exact Or.inl hrev
  exact Or.inr ⟨⟨hv, hlen, hfeat, hPc, hSc, hLc, hMsgr, hAuth, hX⟩, σ',
    ⟨σ7, o, hs7, hc7, hecs, hcall⟩, hret⟩

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

/-- **Outcome.** Every run of the compiled code on `relayUndeliveredMessage` calldata runs out of
    gas, reverts, or succeeds with empty output, all of `RelayConds`, and `RelayPost`. -/
theorem relay_outcome {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256} {v : Views}
    (hcode : I.code = l1cdmRuntime) (hsel : selectorWord I = relaySelector)
    (hcds : I.calldata.size < 2 ^ 256) (hS : Summaries σ σ₀ I v) :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    (∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o)) ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') ByteArray.empty) ∧
      RelayConds I v ∧ RelayPost σ σ₀ I σ') := by
  have hg : (Sat256.ofUInt256 g).toUInt256 = g := rfl
  rcases relay_trace (A := A) (g := Sat256.ofUInt256 g) hcode hsel hcds hS with
    hrev | ⟨hc, σ', hpost, hret⟩
  · rcases RDrev.xiResult hcode hrev with hoog | ⟨g', o, hr⟩
    · left; rw [← hg]; exact hoog
    · right; left; exact ⟨g', o, by rw [← hg]; exact hr⟩
  · rcases rdret_xi hcode hret with hoog | ⟨g', A', hr⟩
    · left; exact hoog
    · right; right; exact ⟨σ', g', A', hr, hc, hpost⟩

/-- **Soundness.** If the compiled code succeeds on `relayUndeliveredMessage(H, t)`, then the INTEROP
    feature is enabled, (a) the caller's portal's SystemConfig names the caller as its
    L1CrossDomainMessenger, (b) this chain's ETHLockbox authorizes the caller's portal, (c) the
    caller's `xDomainMessageSender()` is the UndeliveredMessageExporter, no ETH was attached, the
    calldata is well-formed, and the only state change is the result of one successful self-call
    `this.sendMessage` with calldata `sendMessageCd H t`. -/
theorem relay_success {σ σ₀ σ' : AccountMap} {A A' : Substate} {I : ExecutionEnv} {g g' : UInt256}
    {o : ByteArray} {v : Views}
    (hcode : I.code = l1cdmRuntime) (hsel : selectorWord I = relaySelector)
    (hcds : I.calldata.size < 2 ^ 256) (hS : Summaries σ σ₀ I v)
    (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    RelayConds I v ∧ RelayPost σ σ₀ I σ' ∧ o = ByteArray.empty := by
  rcases relay_outcome (g := g) (A := A) hcode hsel hcds hS with h | ⟨_, _, h⟩ | ⟨σ'', g'', A'', h, hc, hp⟩
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h; exact ⟨hc, hp, rfl⟩

/-- The compiled code never ends in an exceptional halt other than out of gas (no static-mode
    violation: the outer frame itself writes nothing; no invalid opcode, bad jump, stack error). -/
theorem relay_no_other_error {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256}
    {v : Views} {e : ExecutionException}
    (hcode : I.code = l1cdmRuntime) (hsel : selectorWord I = relaySelector)
    (hcds : I.calldata.size < 2 ^ 256) (hS : Summaries σ σ₀ I v)
    (he : Ξ σ σ₀ g A I = .error e) : e = .OutOfGass := by
  rcases relay_outcome (g := g) (A := A) hcode hsel hcds hS with h | ⟨_, _, h⟩ | ⟨_, _, _, h, _⟩
  · rw [he] at h; cases h; rfl
  · rw [he] at h; cases h
  · rw [he] at h; cases h

end L1cdmEvm
