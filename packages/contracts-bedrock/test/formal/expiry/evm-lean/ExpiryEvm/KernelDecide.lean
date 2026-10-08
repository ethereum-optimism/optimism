import Ethereum.Semantics
import Reasoning.JumpDest

/-!
# Kernel-checked bytecode facts (replacing `native_decide`)

The block summaries and trace segments need closed facts about the concrete bytecode:
instruction decodes `decode code pc = some (op, arg)`, pc arithmetic, the JUMPDEST table
`D_J code 0 = #[…]` and JUMPDEST membership. They used to be closed by `native_decide`, which
trusts the Lean compiler and runtime (`Lean.ofReduceBool`). This file lets the **kernel** check
them instead (`decide +kernel`; no `Lean.ofReduceBool`, no new axiom), via two *proved*
reformulations of EVMLean's definitions:

* `decode_eq_decodeList` : `decode code pc = decodeList (code.data.toList.drop pc.toNat)`
* `D_J_eq_jumpdestScan`  : `D_J code i = jumpdestScan (len + 1) (code.data.toList.drop i) i #[]`

`decodeList` and `jumpdestScan` walk the byte *list* by structural recursion, which the kernel
reduces in tens of milliseconds per decode, and in a few seconds for the whole JUMPDEST scan.
Unfolding EVMLean's own definitions in the kernel is slower: `ByteArray.get?`/`extract'` go
through `Array` operations (and `++` of byte chunks is quadratic), and `D_J_aux` is
well-founded recursion. Both lemmas hold for every `ByteArray` and are proved below from the
definitions, so the trusted base is the kernel plus EVMLean's definitions, as for every other
proof here.
-/

namespace Ethereum.EVM.KernelDecide

open Ethereum Ethereum.EVM

/-! ## Byte-array lemmas -/

/-- `ByteArray.toList` (a reversing loop) is `data.toList`. (Same proof as EquiVM's
`Reasoning.Theory.byteArray_toList_eq`; restated to keep this file's imports light.) -/
theorem byteArray_toList_eq (b : ByteArray) : b.toList = b.data.toList := by
  show ByteArray.toList.loop b 0 [] = _
  suffices h : ∀ i r, ByteArray.toList.loop b i r = r.reverse ++ b.data.toList.drop i by
    simpa using h 0 []
  intro i r
  induction i, r using ByteArray.toList.loop.induct (bs := b) with
  | case1 i r hlt ih =>
    rw [ByteArray.toList.loop, if_pos hlt, ih, List.reverse_cons, List.append_assoc]
    congr 1
    have hi : i < b.data.size := hlt
    have hlen : i < b.data.toList.length := by rw [Array.length_toList]; exact hi
    have hget : b.get! i = b.data.toList[i]'hlen := by
      rw [Array.getElem_toList]; exact getElem!_pos b.data i hi
    rw [hget, List.singleton_append, List.getElem_cons_drop]
  | case2 i r hge =>
    rw [ByteArray.toList.loop, if_neg hge]
    have : b.data.toList.length ≤ i := by rw [Array.length_toList]; exact Nat.le_of_not_lt hge
    rw [List.drop_eq_nil_of_le this, List.append_nil]

theorem byteArray_get?_eq (a : ByteArray) (n : Nat) : a.get? n = a.data.toList[n]? := by
  rcases a with ⟨d⟩
  unfold ByteArray.get?
  by_cases h : n < d.size
  · rw [dif_pos (by simpa [ByteArray.size] using h)]
    simp [ByteArray.get, h]; rfl
  · rw [dif_neg (by simpa [ByteArray.size] using h)]
    simp [h]

theorem extract'_toList (a : ByteArray) (b e : Nat) :
    (a.extract' b e).data.toList = (a.data.toList.drop b).take (e - b) := by
  unfold ByteArray.extract'
  split
  · rw [ByteArray.data_extract, Array.toList_extract, List.extract_eq_take_drop]
  · simp [byteArray_toList_eq]

/-! ## Decoding -/

/-- `decode`, reading the instruction byte and its immediate from a byte list that starts at the
instruction. -/
def decodeList : List UInt8 → Option (Operation × Option (UInt256 × Nat))
  | [] => none
  | b :: rest =>
    match parseInstr b with
    | none => none
    | some instr =>
      let w := argOnNBytesOfInstr instr
      some (instr, if w == 0 then none
        else some (UInt256.ofNat (fromBytes' (rest.take w).reverse), w))

/-- EVMLean's `decode` agrees with `decodeList` on every byte array and pc. -/
theorem decode_eq_decodeList (code : ByteArray) (pc : UInt256) :
    decode code pc = decodeList (code.data.toList.drop pc.toNat) := by
  unfold decode
  rw [byteArray_get?_eq]
  cases hl : code.data.toList.drop pc.toNat with
  | nil =>
    have : code.data.toList[pc.toNat]? = none := by
      rw [← List.head?_drop, hl]; rfl
    simp [this, decodeList]
  | cons b rest =>
    have hb : code.data.toList[pc.toNat]? = some b := by
      rw [← List.head?_drop, hl]; rfl
    have hr : code.data.toList.drop pc.toNat.succ = rest := by
      rw [Nat.succ_eq_add_one, ← List.drop_drop, hl]; rfl
    simp only [hb, decodeList, Option.bind_eq_bind, Option.bind_some]
    cases parseInstr b with
    | none => rfl
    | some instr =>
      simp only [Option.bind_some, Option.some.injEq, Prod.mk.injEq, true_and]
      split
      · rfl
      · simp only [uInt256OfByteArray, extract'_toList, hr, Nat.add_sub_cancel_left]

/-! ## JUMPDEST analysis -/

/-- `D_J_aux` as a fuelled structural scan over the byte list starting at index `i`. -/
def jumpdestScan : Nat → List UInt8 → Nat → Array UInt256 → Array UInt256
  | 0, _, _, r => r
  | _ + 1, [], _, r => r
  | fuel + 1, b :: rest, i, r =>
    match parseInstr b with
    | none => r
    | some ci => jumpdestScan fuel (rest.drop (argOnNBytesOfInstr ci)) (N i ci)
        (if ci = .JUMPDEST then r.push (UInt256.ofNat i) else r)

theorem D_J_aux_eq_jumpdestScan (code : ByteArray) :
    ∀ fuel i r, code.data.toList.length - i < fuel →
      D_J_aux code i r = jumpdestScan fuel (code.data.toList.drop i) i r := by
  intro fuel
  induction fuel with
  | zero => intro i r h; omega
  | succ fuel ih =>
    intro i r hf
    unfold D_J_aux
    split
    · rename_i hget
      rw [byteArray_get?_eq] at hget
      cases hl : code.data.toList.drop i with
      | nil => rfl
      | cons b rest =>
        have hb : code.data.toList[i]? = some b := by rw [← List.head?_drop, hl]; rfl
        rw [hb] at hget
        simp only [jumpdestScan]
        simp only [Option.bind_eq_bind, Option.bind_some] at hget
        rw [hget]
    · rename_i ci hget
      rw [byteArray_get?_eq] at hget
      cases hl : code.data.toList.drop i with
      | nil =>
        have : code.data.toList[i]? = none := by rw [← List.head?_drop, hl]; rfl
        rw [this] at hget; simp at hget
      | cons b rest =>
        have hb : code.data.toList[i]? = some b := by rw [← List.head?_drop, hl]; rfl
        rw [hb] at hget
        simp only [Option.bind_eq_bind, Option.bind_some] at hget
        have hlt : i < code.data.toList.length := by
          by_contra hc; rw [List.drop_eq_nil_of_le (by omega)] at hl; cases hl
        simp only [jumpdestScan, hget]
        have hdrop : code.data.toList.drop (N i ci) = rest.drop (argOnNBytesOfInstr ci) := by
          have hN : N i ci = (i + 1) + argOnNBytesOfInstr ci := rfl
          rw [hN, ← List.drop_drop, ← List.drop_drop, hl]; simp
        rw [← hdrop]
        apply ih
        simp only [N]; omega

/-- EVMLean's `D_J` agrees with `jumpdestScan` on every byte array. -/
theorem D_J_eq_jumpdestScan (code : ByteArray) (i : Nat) :
    D_J code i = jumpdestScan (code.data.toList.length + 1) (code.data.toList.drop i) i #[] :=
  D_J_aux_eq_jumpdestScan code _ i #[] (by omega)

end Ethereum.EVM.KernelDecide

/-! ## Tactics -/

/-- Close a closed bytecode fact in the kernel: a `decode` goal through `decode_eq_decodeList`,
anything else (pc arithmetic, equalities of words) directly. Never uses `native_decide`. -/
macro "evm_kdecide" : tactic =>
  `(tactic| first
    | (rw [Ethereum.EVM.KernelDecide.decode_eq_decodeList]; decide +kernel)
    | decide +kernel)

/-- Kernel version of EquiVM's `jump_dest`: rewrite by the `@[valid_jumps]` table, turn the array
membership into list membership, decide in the kernel. -/
macro "kjump_dest" : tactic =>
  `(tactic| (simp only [valid_jumps, List.contains_toArray]; decide +kernel))
