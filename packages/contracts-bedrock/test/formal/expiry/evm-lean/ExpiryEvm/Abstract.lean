import ExpiryEvm.ExpireMessage
import Reasoning.WordArithmetic

/-!
# Bridge to the abstract `expire` action

`../lean/Expiry/Model.lean` (the protocol model; a v2 is being written concurrently, so this file
does not import it — it is also on a different Lean toolchain) defines

```lean
| .expire f, s => s.deposits f ∧ s.sentAt f.toL1 f.hash ≠ 0 ∧ expiredBy cfg (s.sentAt f.toL1 f.hash) f.time
-- next:
| .expire f, s => { s with expired := fun c h => s.expired c h ∨ (c = f.toL1 ∧ h = f.hash) }
-- with expiredBy cfg sent t = sent + cfg.contractPeriod < t   (cfg.expireGe = false)
```

This file restates exactly that action for one chain `z = f.toL1` (`AbsView`, `absGuard`,
`absNext`) and proves that the compiled code refines it, through the abstraction `viewOf`
of the messenger's storage:

* `Model.State.sentAt z h`  ↦  `sentMessageTimestamps[h]` (as `ℕ`),
* `Model.State.expired z h` ↦  `expiredMessages[h]` (Solidity's bool read: low byte ≠ 0),
* `Model.State.deposits f`  ↦  "the call is the L2CrossDomainMessenger relaying an L1→L2
  message whose L1 sender is this chain's L1CrossDomainMessenger":
  `I.source = l2cdm ∧ xDomainMessageSender() = otherMessenger()`. That the L2CrossDomainMessenger
  only sets `xDomainMessageSender` to the L1 sender of a deposit it is relaying is a property of
  the L2CrossDomainMessenger (not verified here; it is the meaning of the `ReturnsAddress`
  summaries' values `vS`, `vO`).
* `cfg.contractPeriod` ↦ `P_contract` (the compiled constant), `f.hash` ↦ `argHash I`,
  `f.time` ↦ `(argTime I).toNat`.

The model works with ideal hashes; the code with `keccak256` storage slots. Instead of a global
non-collision assumption, the view equations are stated **per key**: for every key `H'` whose slot
differs from `expiredMessages[H]`'s, the abstract value at `H'` is preserved (resp. follows
`absNext`). Keys that do collide (if any exist) are simply not covered; no idealized hash
assumption is made.

What this is and is not:
* It is a **projection**: one chain, the two maps `expire` touches, the action's guard and
  effect. It is not a refinement of the whole model state, and it does not relate executions.
* `deposits f` is **not derived**: the model's `s.deposits f` is a history fact (an
  `expireMessage(H, t)` deposit from this chain's L1CrossDomainMessenger exists). Here it is read
  as the call-time condition `RelayFromOtherMessenger` that the code checks. That `RelayFromOtherMessenger` holds only for
  authentic deposits is supplied externally: by the L1CrossDomainMessenger's
  `relayUndeliveredMessage` checks and by the L2CrossDomainMessenger relaying deposits with
  `xDomainMessageSender` = their L1 sender (neither is verified here).
* The restated definitions were compared with `../lean/Expiry/Model.lean` by hand (different
  toolchains); a follow-up on a shared toolchain should replace them by imports.
-/

namespace ExpiryEvm.Abstract

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach Mem

/-- The part of the model state that `expire` reads and writes, for one chain. -/
structure AbsView where
  sentAt : UInt256 → ℕ
  expired : UInt256 → Prop

/-- `Model.expiredBy` with `expireGe = false`. -/
def expiredBy (P sent t : ℕ) : Prop := sent + P < t

/-- `Model.guard (.expire f)` for `f = ⟨z, h, t⟩`, with `deposits f` given as `deposit`. -/
def absGuard (P : ℕ) (deposit : Prop) (v : AbsView) (h : UInt256) (t : ℕ) : Prop :=
  deposit ∧ v.sentAt h ≠ 0 ∧ expiredBy P (v.sentAt h) t

/-- `Model.next (.expire f)`, restricted to chain `z`. -/
def absNext (v : AbsView) (h : UInt256) : AbsView :=
  { v with expired := fun h' => v.expired h' ∨ h' = h }

/-- The abstraction of the messenger's storage at address `a`. -/
def viewOf (σ : AccountMap) (a : AccountAddress) : AbsView where
  sentAt H := (storageWord σ a (sentAtSlot H)).toNat
  expired H := UInt256.land (storageWord σ a (expiredSlot H)) (UInt256.ofNat 0xff) ≠ ⟨0⟩

/-- The concrete meaning of `deposits ⟨z, H, t⟩` at this call. -/
def RelayFromOtherMessenger (I : ExecutionEnv) (vO vS : AccountAddress) : Prop :=
  I.source = l2cdm ∧ vS = vO

theorem setTrueWord_testBit0 (old : UInt256) : (setTrueWord old).toNat.testBit 0 = true := by
  unfold setTrueWord
  set y := UInt256.land (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639680) old
  have hlt : Nat.lor (UInt256.ofNat 1).toNat y.toNat < UInt256.size := by
    simpa [UInt256.size] using
      nat_lor_lt_two_pow (a := (UInt256.ofNat 1).toNat) (b := y.toNat) (k := 256)
        (by change (UInt256.ofNat 1).val.val < UInt256.size; exact (UInt256.ofNat 1).val.isLt)
        (by change y.val.val < UInt256.size; exact y.val.isLt)
  change (Nat.lor (UInt256.ofNat 1).toNat y.toNat % UInt256.size).testBit 0 = true
  rw [Nat.mod_eq_of_lt hlt, show (UInt256.ofNat 1).toNat = 1 by decide]
  change (1 ||| y.toNat).testBit 0 = true
  rw [Nat.testBit_lor]; simp

theorem land_setTrueWord_ff (old : UInt256) :
    UInt256.land (setTrueWord old) (UInt256.ofNat 0xff) ≠ ⟨0⟩ := by
  intro h
  have h1 := congrArg UInt256.toNat h
  change Nat.land (setTrueWord old).toNat (UInt256.ofNat 0xff).toNat % UInt256.size = 0 at h1
  rw [show (UInt256.ofNat 0xff).toNat = 255 by decide] at h1
  have hle : Nat.land (setTrueWord old).toNat 255 ≤ 255 := Nat.and_le_right
  rw [Nat.mod_eq_of_lt (by unfold UInt256.size; omega)] at h1
  have hb : (Nat.land (setTrueWord old).toNat 255).testBit 0 = true := by
    change ((setTrueWord old).toNat &&& 255).testBit 0 = true
    rw [Nat.testBit_and, setTrueWord_testBit0]; decide
  rw [h1] at hb
  simp at hb

theorem storageWord_insert_ne {st : Storage} {s s' v : UInt256} (h : s ≠ s') :
    (st.insert s' v).getD s ⟨0⟩ = st.getD s ⟨0⟩ := by
  rw [Std.ExtTreeMap.getD_insert, if_neg]
  intro hc; exact h (Std.LawfulEqCmp.eq_of_compare hc).symm

theorem storageWord_insert_self {st : Storage} {s v : UInt256} :
    (st.insert s v).getD s ⟨0⟩ = v := by
  rw [Std.ExtTreeMap.getD_insert, if_pos (Std.ReflCmp.compare_self)]

/-- The effect of a successful run on the abstraction, per key: `expired` becomes true at `H`,
    and every key whose slot differs from `expiredSlot H` keeps its `sentAt` / follows `absNext`. -/
theorem post_view {σ σ' : AccountMap} {I : ExecutionEnv} (hpost : ExpirePost σ σ' I) :
    (viewOf σ' I.codeOwner).expired (argHash I) ∧
    (∀ H, sentAtSlot H ≠ expiredSlot (argHash I) →
      (viewOf σ' I.codeOwner).sentAt H = (viewOf σ I.codeOwner).sentAt H) ∧
    (∀ H, expiredSlot H ≠ expiredSlot (argHash I) →
      ((viewOf σ' I.codeOwner).expired H ↔ (absNext (viewOf σ I.codeOwner) (argHash I)).expired H)) := by
  have hs : ∀ s, storageWord σ' I.codeOwner s =
      ((σ.getD I.codeOwner default).storage.insert (expiredSlot (argHash I))
        (setTrueWord (storageWord σ I.codeOwner (expiredSlot (argHash I))))).getD s ⟨0⟩ := by
    intro s; unfold storageWord; rw [hpost.self_storage]; rfl
  refine ⟨?_, fun H hH => ?_, fun H hH => ?_⟩
  · show UInt256.land (storageWord σ' I.codeOwner (expiredSlot (argHash I))) (UInt256.ofNat 0xff) ≠ ⟨0⟩
    rw [hs, storageWord_insert_self]
    exact land_setTrueWord_ff _
  · show (storageWord σ' I.codeOwner (sentAtSlot H)).toNat = (storageWord σ I.codeOwner (sentAtSlot H)).toNat
    rw [hs, storageWord_insert_ne hH]; rfl
  · have hne : H ≠ argHash I := fun h => hH (by rw [h])
    show UInt256.land (storageWord σ' I.codeOwner (expiredSlot H)) (UInt256.ofNat 0xff) ≠ ⟨0⟩ ↔
      (UInt256.land (storageWord σ I.codeOwner (expiredSlot H)) (UInt256.ofNat 0xff) ≠ ⟨0⟩ ∨
        H = argHash I)
    rw [hs, storageWord_insert_ne hH]
    exact ⟨fun h => Or.inl h, fun h => h.resolve_right hne⟩

/-- Every messenger storage slot other than `expiredSlot H` is unchanged by a successful run. -/
theorem slot_frame {σ σ' : AccountMap} {I : ExecutionEnv} (hpost : ExpirePost σ σ' I) :
    ∀ s, s ≠ expiredSlot (argHash I) → storageWord σ' I.codeOwner s = storageWord σ I.codeOwner s := by
  intro s hs
  unfold storageWord
  rw [hpost.self_storage, storageWord_insert_ne hs]

/-- Storage slot of `successfulMessages[H]` (slot 0). -/
def successfulSlot (H : UInt256) : UInt256 := solcMappingSlot (UInt256.ofNat 0) H
/-- Storage slot of `sentMessages[n]` (slot 2). -/
def sentMessagesSlot (n : UInt256) : UInt256 := solcMappingSlot (UInt256.ofNat 2) n
/-- Storage slot of `msgNonce` (slot 1). -/
def msgNonceSlot : UInt256 := UInt256.ofNat 1

/-- Per-key frame facts for the messenger's other state, each under its slot side condition. -/
theorem frame_other_maps {σ σ' : AccountMap} {I : ExecutionEnv} (hpost : ExpirePost σ σ' I) :
    (∀ H, successfulSlot H ≠ expiredSlot (argHash I) →
      storageWord σ' I.codeOwner (successfulSlot H) = storageWord σ I.codeOwner (successfulSlot H)) ∧
    (∀ n, sentMessagesSlot n ≠ expiredSlot (argHash I) →
      storageWord σ' I.codeOwner (sentMessagesSlot n) = storageWord σ I.codeOwner (sentMessagesSlot n)) ∧
    (msgNonceSlot ≠ expiredSlot (argHash I) →
      storageWord σ' I.codeOwner msgNonceSlot = storageWord σ I.codeOwner msgNonceSlot) :=
  ⟨fun H h => slot_frame hpost _ h, fun n h => slot_frame hpost _ h, fun h => slot_frame hpost _ h⟩

/-- **Conditional per-key correspondence with the `expire` step (soundness); deposit history
    supplied externally.** After a successful run of the compiled `expireMessage(H, t)`:
    the abstract guard held before, with the model's history fact `deposits f` *replaced* by the
    call-time authorization `RelayFromOtherMessenger` (independent of `H` and `t`; that it implies
    `deposits f` is not proved here); `expired` holds at `H`; every key whose slot differs from
    `expiredSlot H` keeps `sentAt` (this applies to `H` itself too: `sentAtSlot H ≠ expiredSlot H`
    is a side condition) and follows `absNext`; every other storage slot of the messenger
    (`successfulMessages`, `sentMessages`, `msgNonce`, …) is unchanged unless it *is*
    `expiredSlot H`; and no other account's storage changed. This is not a refinement of the
    model's full state: colliding keys are excluded and the model's deposit history is not
    related. -/
theorem refines_expire {σ σ₀ σ' : AccountMap} {A A' : Substate} {I : ExecutionEnv}
    {g g' : UInt256} {o : ByteArray} {vO vS : AccountAddress}
    (hcode : I.code = l2tol2Runtime) (hsel : selectorWord I = expireSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)
    (hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS)
    (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    absGuard P_contract (RelayFromOtherMessenger I vO vS) (viewOf σ I.codeOwner) (argHash I) (argTime I).toNat ∧
    (viewOf σ' I.codeOwner).expired (argHash I) ∧
    (∀ H, sentAtSlot H ≠ expiredSlot (argHash I) →
      (viewOf σ' I.codeOwner).sentAt H = (viewOf σ I.codeOwner).sentAt H) ∧
    (∀ H, expiredSlot H ≠ expiredSlot (argHash I) →
      ((viewOf σ' I.codeOwner).expired H ↔ (absNext (viewOf σ I.codeOwner) (argHash I)).expired H)) ∧
    (∀ s, s ≠ expiredSlot (argHash I) →
      storageWord σ' I.codeOwner s = storageWord σ I.codeOwner s) ∧
    (∀ a, a ≠ I.codeOwner → (σ'.getD a default).storage = (σ.getD a default).storage) := by
  obtain ⟨_, hc, hpost, _⟩ := expireMessage_success hcode hsel hcds hO hX hres
  refine ⟨⟨⟨hc.callerIsL2cdm, hc.senderIsOther⟩, ?_, hc.expired⟩, (post_view hpost).1,
    (post_view hpost).2.1, (post_view hpost).2.2, slot_frame hpost, hpost.other_storage⟩
  intro h0
  apply hc.wasSent
  exact Words.ext_iff.mpr h0

end ExpiryEvm.Abstract
