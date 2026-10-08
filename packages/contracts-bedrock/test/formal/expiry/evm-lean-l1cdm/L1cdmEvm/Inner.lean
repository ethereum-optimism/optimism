import L1cdmEvm.InnerMid
import L1cdmEvm.Outer

/-!
# `sendMessage` on the self-call's calldata (EVM level)

The frame created by `relayUndeliveredMessage`'s self-call runs the same code with calldata
`sendMessageCd H t`, no value, and `msg.sender` = the contract itself. `send_success`: if that run
succeeds, it made exactly one external call, the deposit `portal.depositTransaction(...)` with
calldata `depositCd (otherMessenger) (versioned nonce) (msg.sender) H t`, which succeeded, and the
only other state change is `++msgNonce` on top of the deposit call's resulting account map.
-/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

theorem nonceTerm_eq (σ : AccountMap) (a : AccountAddress) :
    nonceTerm σ a = versionedNonce (msgNonceWord σ a) := by
  have e1 : (1766847064778384329583297500742918515827483896875618958121606201292619775 : ℕ) = 2 ^ 240 - 1 := by
    norm_num
  have e2 : (1766847064778384329583297500742918515827483896875618958121606201292619776 : ℕ) = 2 ^ 240 * 1 := by
    norm_num
  generalize hx : storageWord σ a (UInt256.ofNat 205) = x
  have hlt := Nat.mod_lt x.toNat (show 2 ^ 240 > 0 by norm_num)
  apply u_ext
  unfold nonceTerm versionedNonce msgNonceWord
  rw [hx, lor_toNat, uland_toNat, u_toNat_ofNat (by norm_num), u_toNat_ofNat (by norm_num), e1, e2,
    Nat.land_comm, Nat.and_two_pow_sub_one_eq_mod]
  show (2 ^ 240 * 1 ||| x.toNat % 2 ^ 240) % UInt256.size = _
  rw [← Nat.two_pow_add_eq_or_of_lt (Nat.mod_lt _ (by norm_num)), Words.size_eq,
    u_toNat_ofNat (by omega), Nat.mod_eq_of_lt (by omega)]
  ring

/-- What a successful `sendMessage` run on the self-call calldata did. -/
def SendPost (σ σ₀ : AccountMap) (I : ExecutionEnv) (H t : UInt256) (σ' : AccountMap) : Prop :=
  extCodeSizeWord σ (UInt256.land (storageWord σ I.codeOwner portalSlot) addrMask) ≠ ⟨0⟩ ∧
  ∃ σp o, DepositCall σ₀ I
      (depositCd (otherMessengerWord σ I.codeOwner) (versionedNonce (msgNonceWord σ I.codeOwner))
        (addrWord I.source) H t) σ σp true o ∧
    σ' = sstoreAccountMap I.codeOwner σp (UInt256.ofNat 205) (bumpNonceWord (msgNonceWord σp I.codeOwner))

theorem send_outcome {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256} {H t : UInt256}
    (hcode : I.code = l1cdmRuntime) (hcd : I.calldata = sendMessageCd H t) (hv : I.weiValue = ⟨0⟩) :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    (∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o)) ∨
    (Ξ σ σ₀ g A I = .error .StaticModeViolation ∧ I.perm = false) ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') ByteArray.empty) ∧ I.perm = true ∧
      SendPost σ σ₀ I H t σ') := by
  have hg : (Sat256.ofUInt256 g).toUInt256 = g := rfl
  obtain ⟨m, aw, k, C, hF, r⟩ := iseg_entry (σ := σ) (σ₀ := σ₀) (A := A) (g := Sat256.ofUInt256 g)
    (ienv σ I H t) hcode hcd
  rcases iseg_mid hcd hv hF r with hrev | ⟨hp, hstat⟩ | ⟨hp, hecs, σp, o, hcall, hret⟩
  · rcases RDrev.xiResult hcode hrev with hoog | ⟨g', o, hr⟩
    · left; rw [← hg]; exact hoog
    · right; left; exact ⟨g', o, by rw [← hg]; exact hr⟩
  · rcases RDstatic.xiResult hcode hstat with hoog | hs
    · left; rw [← hg]; exact hoog
    · right; right; left; exact ⟨by rw [← hg]; exact hs, hp⟩
  · rcases rdret_xi hcode hret with hoog | ⟨g', A', hr⟩
    · left; exact hoog
    · right; right; right
      refine ⟨_, g', A', hr, hp, hecs, σp, o, ?_, rfl⟩
      rw [← nonceTerm_eq]; exact hcall

/-- **Soundness of the self-call frame.** -/
theorem send_success {σ σ₀ σ' : AccountMap} {A A' : Substate} {I : ExecutionEnv} {g g' : UInt256}
    {o : ByteArray} {H t : UInt256}
    (hcode : I.code = l1cdmRuntime) (hcd : I.calldata = sendMessageCd H t) (hv : I.weiValue = ⟨0⟩)
    (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    I.perm = true ∧ SendPost σ σ₀ I H t σ' ∧ o = ByteArray.empty := by
  rcases send_outcome (g := g) (A := A) hcode hcd hv with h | ⟨_, _, h⟩ | ⟨h, _⟩ | ⟨σ'', g'', A'', h, hp, hpost⟩
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h; exact ⟨hp, hpost, rfl⟩

end L1cdmEvm
