import L1cdmEvm.Inner

/-!
# Composition: `relayUndeliveredMessage` succeeds ⇒ exactly one deposit with the exact envelope

`relay_success` (outer frame) says the only state-changing operation is a successful self-call
`this.sendMessage(...)` (EVMLean `Θ`) with calldata `sendMessageCd H t`. When the contract's account
holds the L1CrossDomainMessenger code itself (the implementation is called directly, not through a
proxy) and is not a precompile address, `Θ` runs that code with `Ξ` in the frame `selfCallEnv`, and
`send_success` (inner frame) applies to it.

Behind a proxy the self-call first runs the proxy's code, which `DELEGATECALL`s the implementation
with the same calldata, caller and storage context; the proxy code is not verified here. In that
deployment `send_success` applies directly to the implementation frame (any `ExecutionEnv` with this
code, this calldata and no value), see README.
-/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach L1cdmEvm.SymMem

/-- The execution environment `Θ` builds for the self-call (`Θ`'s equations (132)–(141)). -/
def selfCallEnv (I : ExecutionEnv) (cd code : ByteArray) : ExecutionEnv :=
  { codeOwner := I.codeOwner, sender := I.sender, gasPrice := (UInt256.ofNat I.gasPrice).toNat,
    calldata := cd, source := I.codeOwner, weiValue := ⟨0⟩, depth := I.depth + 1, perm := I.perm,
    code := code, header := I.header, blobVersionedHashes := I.blobVersionedHashes, blocks := I.blocks }

theorem insert_same {σ : AccountMap} {a : AccountAddress} {acc : Account} (h : σ.get? a = some acc) :
    σ.insert a acc = σ := by
  apply Std.ExtTreeMap.ext_getElem?
  intro k
  rw [Std.ExtTreeMap.getElem?_insert]
  split
  · next hk =>
    have : a = k := Std.LawfulEqCmp.eq_of_compare hk
    subst this; rw [← Std.ExtTreeMap.get?_eq_getElem?, h]
  · rfl

theorem add_zero' (x : UInt256) : x + ⟨0⟩ = x := by
  apply u_ext; rw [Words.toNat_add]; show (x.toNat + 0) % _ = _; rw [Nat.add_zero, Nat.mod_eq_of_lt]
  exact (Words.size_eq ▸ x.val.isLt)

theorem sub_zero' (x : UInt256) : x - ⟨0⟩ = x := by
  cases x with
  | mk v => show UInt256.mk (v - 0) = UInt256.mk v; rw [sub_zero]

/-- A successful value-0 self-call into code `c` is a successful `Ξ` run of `c` in `selfCallEnv`
    on the unchanged account map. -/
theorem theta_self_success {σc σ₀ σ' : AccountMap} {A_in A' : Substate} {I : ExecutionEnv}
    {c cd out : ByteArray} {callGas g' : UInt256}
    (hacc : ∃ acc, σc.get? I.codeOwner = some acc)
    (hΘ : (σ', g', A', true, out) = Θ σc σ₀ A_in I.codeOwner I.sender I.codeOwner (ToExecute.Code c)
      callGas (UInt256.ofNat I.gasPrice) ⟨0⟩ ⟨0⟩ cd (I.depth + 1) I.header I.blobVersionedHashes I.blocks
      I.perm) :
    ∃ A'', Ξ σc σ₀ callGas A_in (selfCallEnv I cd c) = .ok (.success (σ', g', A'') out) := by
  obtain ⟨acc, hacc⟩ := hacc
  unfold Θ at hΘ
  simp only [hacc, add_zero'] at hΘ
  have e1 : σc.insert I.codeOwner { acc with balance := acc.balance } = σc := insert_same hacc
  simp only [e1, hacc, sub_zero'] at hΘ
  have hX0 : Ξ σc σ₀ callGas A_in (selfCallEnv I cd c) = Ξ σc σ₀ callGas A_in (selfCallEnv I cd c) := rfl
  generalize hR : Ξ σc σ₀ callGas A_in (selfCallEnv I cd c) = R at hX0
  simp only [selfCallEnv] at hR
  rw [hR] at hΘ
  rw [← hX0] at hR
  cases R with
  | error e => simp at hΘ
  | ok r =>
    cases r with
    | revert g2 o2 => simp at hΘ
    | success tr o2 =>
      obtain ⟨a, b, c'⟩ := tr
      by_cases ha : (a == ∅) = true
      · simp [ha] at hΘ
      · simp only [ha, if_false, Prod.mk.injEq] at hΘ
        obtain ⟨h1, h2, h3, _, h5⟩ := hΘ
        subst h1 h2 h5
        refine ⟨c', ?_⟩
        rw [← hX0]

theorem getD_code_get {σ : AccountMap} {a : AccountAddress} {code : ByteArray} (hne : code.size ≠ 0)
    (h : (σ.getD a default).code = code) : ∃ acc, σ.get? a = some acc ∧ acc.code = code := by
  cases hg : σ.get? a with
  | none =>
    exfalso
    have : σ[a]? = none := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact hg
    rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this] at h
    apply hne; rw [← h]; rfl
  | some acc =>
    refine ⟨acc, rfl, ?_⟩
    have : σ[a]? = some acc := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact hg
    rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this] at h
    exact h

theorem toExecute_code {σ : AccountMap} {a : AccountAddress} {code : ByteArray} (hne : code.size ≠ 0)
    (hnp : a ∉ π) (h : (σ.getD a default).code = code) : toExecute σ a = ToExecute.Code code := by
  obtain ⟨acc, hacc, hc⟩ := getD_code_get hne h
  unfold toExecute
  rw [if_neg hnp]
  simp only [Id.run, hacc, hc]

theorem getD_code_ne_empty_get {σ : AccountMap} {a : AccountAddress}
    (h : (σ.getD a default).code = l1cdmRuntime) : ∃ acc, σ.get? a = some acc ∧ acc.code = l1cdmRuntime := by
  cases hg : σ.get? a with
  | none =>
    exfalso
    have : σ[a]? = none := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact hg
    rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this] at h
    have hsz := congrArg ByteArray.size h
    simp only [Option.getD_none] at hsz
    revert hsz; decide +kernel
  | some acc =>
    refine ⟨acc, rfl, ?_⟩
    have : σ[a]? = some acc := by rw [← Std.ExtTreeMap.get?_eq_getElem?]; exact hg
    rw [Std.ExtTreeMap.getD_eq_getD_getElem?, this] at h
    exact h

theorem toExecute_self {σ : AccountMap} {a : AccountAddress} (hnp : a ∉ π)
    (h : (σ.getD a default).code = l1cdmRuntime) : toExecute σ a = ToExecute.Code l1cdmRuntime := by
  obtain ⟨acc, hacc, hc⟩ := getD_code_ne_empty_get h
  unfold toExecute
  rw [if_neg hnp]
  simp only [Id.run, hacc, hc]

/-- **Headline (composition).** If the compiled code succeeds on `relayUndeliveredMessage(H, t)` and
    the executing account holds this code (not a precompile address), then the conditions hold, the
    run was not static, and the final account map is: one successful deposit
    `portal.depositTransaction(otherMessenger, 0, 412835, false, relayMessage(nonce, this,
    0x4200..0023, 0, 100000, expireMessage(H, t)))` made from the self-call frame, from an account map
    with the original storage and code, followed by `++msgNonce`. -/
theorem relay_deposit {σ σ₀ σ' : AccountMap} {A A' : Substate} {I : ExecutionEnv} {g g' : UInt256}
    {o : ByteArray} {v : Views}
    (hcode : I.code = l1cdmRuntime) (hsel : selectorWord I = relaySelector)
    (hcds : I.calldata.size < 2 ^ 256) (hS : Summaries σ σ₀ I v)
    (hself : (σ.getD I.codeOwner default).code = l1cdmRuntime) (hnp : I.codeOwner ∉ π)
    (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    RelayConds I v ∧ I.perm = true ∧ o = ByteArray.empty ∧
    ∃ σc σp out, accountStorageStateEq σ σc ∧ accountCodeStateEq σ σc ∧
      DepositCall σ₀ (selfCallEnv I (sendMessageCd (argHash I) (argTime I)) l1cdmRuntime)
        (depositCd (otherMessengerWord σ I.codeOwner) (versionedNonce (msgNonceWord σ I.codeOwner))
          (addrWord I.codeOwner) (argHash I) (argTime I)) σc σp true out ∧
      σ' = sstoreAccountMap I.codeOwner σp (UInt256.ofNat 205) (bumpNonceWord (msgNonceWord σp I.codeOwner)) := by
  obtain ⟨hc, ⟨σc, out1, hst, hcd, _, A_in, callGas, g1, A1, hΘ⟩, ho⟩ :=
    relay_success hcode hsel hcds hS hres
  have hselfc : (σc.getD I.codeOwner default).code = l1cdmRuntime := by rw [← hcd I.codeOwner]; exact hself
  rw [toExecute_self hnp hselfc] at hΘ
  obtain ⟨acc, hacc, _⟩ := getD_code_ne_empty_get hselfc
  obtain ⟨A'', hX⟩ := theta_self_success ⟨acc, hacc⟩ hΘ
  obtain ⟨hp, ⟨_, σp, out, hdep, hfin⟩, _⟩ := send_success (I := selfCallEnv I _ l1cdmRuntime) rfl rfl rfl hX
  refine ⟨hc, hp, ho, σc, σp, out, hst, hcd, ?_, hfin⟩
  have e1 : otherMessengerWord σc I.codeOwner = otherMessengerWord σ I.codeOwner := by
    unfold otherMessengerWord; rw [storageWord_eq_of_storageEq hst]
  have e2 : msgNonceWord σc I.codeOwner = msgNonceWord σ I.codeOwner := by
    unfold msgNonceWord; rw [storageWord_eq_of_storageEq hst]
  have := hdep
  simp only [selfCallEnv] at this ⊢
  rw [e1, e2] at this
  exact this

end L1cdmEvm
