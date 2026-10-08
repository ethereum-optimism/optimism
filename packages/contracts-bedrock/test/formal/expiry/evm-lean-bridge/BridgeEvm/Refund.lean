import BridgeEvm.TraceCreate

/-!
# `refundETH` refinement theorems (EVM level)

The deployed runtime code `ethbridgeRuntime` of `SuperchainETHBridge`, run by EVMLean's
code-execution function `Ξ` on calldata that selects `refundETH(uint256,uint256,address,address,uint256)`,
from any account map, caller, value, depth, gas and permission:

* `refundETH_trace` — the whole symbolic execution, at the level of EquiVM's `RD` invariant.
* `refundETH_outcome` — every run ends in out-of-gas, a revert, a static-mode violation (only
  when entered by `STATICCALL`), or success with `RefundRun` (all success conditions, the exact
  external calls, and the final account map).
* `refundETH_success` — soundness: a successful run implies `RefundRun`.
* `refundETH_no_other_error` — no other exceptional halt.

No summary of any external call is assumed: `RefundRun` *states* that the three external
operations happened with exact inputs and the outputs that make the code proceed.
-/

namespace BridgeEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach Mem

/-- **What a successful `refundETH(destination, nonce, from, to, amount)` run implies**, with
    `H = refundHash I` and the pre-state `σ`, ending in account map `σ'`:
    1. the ABI conditions `ArgsOk` and a non-static frame;
    2. a `STATICCALL` to the L2ToL2CrossDomainMessenger 0x4200…0023 with calldata exactly
       `expiredMessages(H)` succeeded and returned ≥ 32 bytes whose first word is `1` (true);
       it left storage and code unchanged (σ₁);
    3. `refunded[H]` was false in `σ` (low byte of its slot zero);
    4. the bridge's storage then became `storedMap σ₁ I H` (`refunded[H] := true`);
    5. ETHLiquidity 0x4200…0025 had code in that state, and a `CALL` to it with value 0 and
       calldata exactly `mint(amount)` succeeded (σ₃);
    6. a `CREATE` with endowment `amount` and init code exactly
       `SafeSend.creationCode ‖ abi.encode(from)` ran from σ₃ and pushed a nonzero address,
       and its resulting account map is the final `σ'` (`CreateStep`; see `createStep_success`). -/
def RefundRun (σ σ₀ : AccountMap) (I : ExecutionEnv) (σ' : AccountMap) : Prop :=
  ArgsOk I ∧ I.perm = true ∧
  ∃ σ₁ oE, StaticCall σ₀ I l2l2 (expiredCalldata (refundHash I)) σ σ₁ true oE ∧
    32 ≤ oE.size ∧ returnWord oE = UInt256.ofNat 1 ∧
    accountStorageStateEq σ σ₁ ∧ accountCodeStateEq σ σ₁ ∧
    UInt256.land (UInt256.ofNat 255) (refundedWord σ I (refundHash I)) = ⟨0⟩ ∧
    extCodeSizeWord (storedMap σ₁ I (refundHash I)) ethLiqWord ≠ ⟨0⟩ ∧
    ∃ σ₃ oM, CallTo σ₀ I ethLiq (mintCalldata (argAmount I)) (storedMap σ₁ I (refundHash I)) σ₃ true oM ∧
    ∃ x rd', CreateStep I σ₀ σ₃ (argAmount I) (safeSendDeploy (argFrom I)) x σ' rd' ∧
      x ≠ UInt256.ofNat 0

theorem storageWord_eq_of_storageEq {σ σ' : AccountMap} (h : accountStorageStateEq σ σ')
    (a : AccountAddress) (s : UInt256) : storageWord σ' a s = storageWord σ a s := by
  unfold storageWord; rw [(h a).1]

theorem pre_window (L : List UInt8) (fp : ℕ) (h1 : 96 ≤ fp) (h2 : 96 ≤ L.length) :
    ((pre L fp).drop 64).take 32 = (L.drop 64).take 32 := by
  unfold pre
  rw [List.drop_append_of_le_length (by simp; omega), List.take_append_of_le_length (by simp; omega),
    List.drop_take, List.take_take]
  congr 1; omega

theorem scrMem_window (I : ExecutionEnv) (H : UInt256) (R : List UInt8) (fp : ℕ) (hRl : R.length = 32)
    (hfp : 676 ≤ fp) :
    ((pre (scrMem I H R fp) fp).drop 64).take 32 = wb (UInt256.ofNat fp) := by
  have hT := expMem_tail_take I H R fp
  have hTl := expMem_tail_length I H R fp hRl
  show ((pre (wb H ++ (wb (UInt256.ofNat 0) ++ (expMem I H R fp).drop 64)) fp).drop 64).take 32 = _
  generalize (expMem I H R fp).drop 64 = T at hT hTl ⊢
  rw [pre_window _ _ (by omega) (by simp only [List.length_append, length_wb]; omega),
    ← List.append_assoc, List.drop_left' (by simp), hT]

set_option maxHeartbeats 4000000 in
theorem refundETH_trace {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    (hcode : I.code = ethbridgeRuntime) (hsel : selectorWord I = refundSelector)
    (hcds : I.calldata.size < 2 ^ 256) :
    RDrev ethbridgeRuntime g (initState σ σ₀ g A I) ∨
    (I.perm = false ∧ RDstatic ethbridgeRuntime g (initState σ σ₀ g A I)) ∨
    (∃ σ', RefundRun σ σ₀ I σ' ∧ RDret ethbridgeRuntime g (initState σ σ₀ g A I) σ' ByteArray.empty) := by
  rcases seg_entry (σ := σ) (σ₀ := σ₀) (A := A) (g := g) hcode hsel hcds with
    ⟨hrev, _⟩ | ⟨hargs, aw, k, C, r1⟩
  · exact Or.inl hrev
  obtain ⟨aw2, k2, C2, r2⟩ := seg_hash hargs r1
  rcases seg_expired r2 with hrev | ⟨σ₁, o, hcall, hlen, hob, hw, hst, hcd, aw3, k3, C3, r3⟩
  · exact Or.inl hrev
  have hRl : (o.data.toList.take 32).length = 32 := by simp; exact hlen
  have hfp : 676 ≤ 644 + (o.size + 31) / 32 * 32 := by
    have : 32 ≤ (o.size + 31) / 32 * 32 := by
      have := Nat.div_mul_le_self (o.size + 31) 32
      have h2 : 1 ≤ (o.size + 31) / 32 := (Nat.le_div_iff_mul_le (by norm_num)).mpr (by omega)
      omega
    omega
  have hfpb : 644 + (o.size + 31) / 32 * 32 < 2 ^ 200 := by
    have := Nat.div_mul_le_self (o.size + 31) 32
    have h138 : (2:ℕ) ^ 138 < 2 ^ 199 := Nat.pow_lt_pow_right (by norm_num) (by norm_num)
    have h199 : (2:ℕ) ^ 200 = 2 * 2 ^ 199 := by ring
    omega
  rcases seg_store hRl hfp hfpb r3 with hrev | ⟨hp, _, hstat⟩ | ⟨hp, hrf, hx, σ₃, oM, hmint, aw4, k4, C4, r4⟩
  · exact Or.inl hrev
  · exact Or.inr (Or.inl ⟨hp, hstat⟩)
  rcases seg_create hargs.fromClean hp (length_pre _ _) (scrMem_window I _ _ _ hRl hfp) hfp hfpb r4 with
    hrev | ⟨x, σ', rd', hcs, hx0, hret⟩
  · exact Or.inl hrev
  refine Or.inr (Or.inr ⟨σ', ⟨hargs, hp, σ₁, o, hcall, hlen, hw, hst, hcd, ?_, hx, σ₃, oM, hmint,
    x, rd', hcs, hx0⟩, hret⟩)
  unfold refundedWord at hrf ⊢
  rwa [storageWord_eq_of_storageEq hst] at hrf

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

/-- **Outcome theorem.** Every run of the compiled bridge on `refundETH` calldata ends in exactly
    one of: out of gas; a revert; a static-mode violation (only when entered by `STATICCALL`);
    or success with empty output and `RefundRun` relating the pre-state `σ` to the final account
    map `σ'`. -/
theorem refundETH_outcome {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : UInt256}
    (hcode : I.code = ethbridgeRuntime) (hsel : selectorWord I = refundSelector)
    (hcds : I.calldata.size < 2 ^ 256) :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    (∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o)) ∨
    (Ξ σ σ₀ g A I = .error .StaticModeViolation ∧ I.perm = false) ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') ByteArray.empty) ∧ RefundRun σ σ₀ I σ') := by
  have hg : (Sat256.ofUInt256 g).toUInt256 = g := rfl
  rcases refundETH_trace (σ := σ) (σ₀ := σ₀) (A := A) (g := Sat256.ofUInt256 g) hcode hsel hcds with
    hrev | ⟨hp, hstat⟩ | ⟨σ', hrun, hret⟩
  · rcases RDrev.xiResult hcode hrev with hoog | ⟨g', o, hr⟩
    · left; rw [← hg]; exact hoog
    · right; left; exact ⟨g', o, by rw [← hg]; exact hr⟩
  · rcases RDstatic.xiResult hcode hstat with hoog | hs
    · left; rw [← hg]; exact hoog
    · right; right; left; exact ⟨by rw [← hg]; exact hs, hp⟩
  · rcases rdret_xi hcode hret with hoog | ⟨g', A', hr⟩
    · left; exact hoog
    · right; right; right; exact ⟨σ', g', A', hr, hrun⟩

/-- **Soundness.** If the compiled bridge succeeds on `refundETH(destination, nonce, from, to,
    amount)` from pre-state `σ` with final account map `σ'`, then `RefundRun σ σ₀ I σ'` holds (in
    particular `expiredMessages(H)` returned true, `refunded[H]` was false, `refunded[H]` was
    set, `mint(amount)` was called successfully, and a SafeSend `CREATE` with value `amount` and
    beneficiary argument `from` succeeded and produced `σ'`), and the output is empty. -/
theorem refundETH_success {σ σ₀ σ' : AccountMap} {A A' : Substate} {I : ExecutionEnv}
    {g g' : UInt256} {o : ByteArray}
    (hcode : I.code = ethbridgeRuntime) (hsel : selectorWord I = refundSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    RefundRun σ σ₀ I σ' ∧ o = ByteArray.empty := by
  rcases refundETH_outcome (g := g) (A := A) hcode hsel hcds with
    h | ⟨_, _, h⟩ | ⟨h, _⟩ | ⟨σ'', g'', A'', h, hrun⟩
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h; cases h
  · rw [hres] at h
    cases h
    exact ⟨hrun, rfl⟩

/-- No exceptional halt other than out-of-gas or (when entered by `STATICCALL`) a static-mode
    violation: no `INVALID`, stack under/overflow, bad jump, etc. -/
theorem refundETH_no_other_error {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv}
    {g : UInt256} {e : ExecutionException}
    (hcode : I.code = ethbridgeRuntime) (hsel : selectorWord I = refundSelector)
    (hcds : I.calldata.size < 2 ^ 256)
    (he : Ξ σ σ₀ g A I = .error e) :
    e = .OutOfGass ∨ (e = .StaticModeViolation ∧ I.perm = false) := by
  rcases refundETH_outcome (g := g) (A := A) hcode hsel hcds with
    h | ⟨_, _, h⟩ | ⟨h, hp⟩ | ⟨_, _, _, h, _⟩
  · rw [he] at h; cases h; exact Or.inl rfl
  · rw [he] at h; cases h
  · rw [he] at h; cases h; exact Or.inr ⟨rfl, hp⟩
  · rw [he] at h; cases h

/-- A `CREATE` that pushed a nonzero word ran `Lambda` (with the creator's nonce incremented)
    on exactly the given init code and endowment, and `Lambda` reported success (`z = true`);
    `σ'` is `Lambda`'s resulting account map and `x` the created address. -/
theorem createStep_success {I : ExecutionEnv} {σ₀ σ σ' : AccountMap} {value : UInt256}
    {i : ByteArray} {x : UInt256} {rd' : ByteArray}
    (h : CreateStep I σ₀ σ value i x σ' rd') (hx : x ≠ UInt256.ofNat 0) :
    let σa : Account := σ.get? I.codeOwner |>.getD default
    σa.nonce.toNat < 2 ^ 64 - 1 ∧
    ∃ (A : Substate) (gas : UInt256)
      (hD : value ≤ (σ.get? I.codeOwner |>.option ⟨0⟩ (·.balance)) ∧ I.depth < 1024 ∧ i.size ≤ 49152)
      (a : AccountAddress) (g' : UInt256) (A' : Substate) (o : ByteArray),
      Lambda (σ.insert I.codeOwner {σa with nonce := σa.nonce + ⟨1⟩}) σ₀ A I.codeOwner I.sender
        gas (UInt256.ofNat I.gasPrice) value i ⟨I.depth.val + 1, Nat.succ_lt_succ hD.2.1⟩ none
        I.header I.blobVersionedHashes I.blocks I.perm = (a, σ', g', A', true, o) ∧
      x = UInt256.ofNat a ∧ rd' = .empty := by
  obtain ⟨A, gas, h | h | ⟨hn, hD, a, g', A', z, o, hL, hxz, hrd⟩⟩ := h
  · exact absurd h.2.1 hx
  · exact absurd h.2.2.1 hx
  · cases z with
    | false => simp at hxz; exact absurd hxz hx
    | true => exact ⟨hn, A, gas, hD, a, g', A', o, hL, by simpa using hxz, by simpa using hrd⟩

end BridgeEvm
