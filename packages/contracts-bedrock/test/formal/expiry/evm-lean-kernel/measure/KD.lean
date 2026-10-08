import Ethereum.Semantics
import Reasoning.Memory
open Ethereum Ethereum.EVM

namespace KD

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
  · simp [Reasoning.Theory.byteArray_toList_eq]

def decodeList : List UInt8 → Option (Operation × Option (UInt256 × Nat))
  | [] => none
  | b :: rest =>
    match parseInstr b with
    | none => none
    | some instr =>
      let w := argOnNBytesOfInstr instr
      some (instr, if w == 0 then none
        else some (UInt256.ofNat (fromBytes' (rest.take w).reverse), w))

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

def jumpdestScan : Nat → List UInt8 → Nat → Array UInt256 → Array UInt256
  | 0, _, _, r => r
  | _ + 1, [], _, r => r
  | fuel + 1, b :: rest, i, r =>
    match parseInstr b with
    | none => r
    | some ci => jumpdestScan fuel (rest.drop (argOnNBytesOfInstr ci)) (N i ci)
        (if ci = .JUMPDEST then r.push (UInt256.ofNat i) else r)

theorem D_J_aux_eq_scan (code : ByteArray) :
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

theorem D_J_eq_scan (code : ByteArray) (i : Nat) :
    D_J code i = jumpdestScan (code.data.toList.length + 1) (code.data.toList.drop i) i #[] :=
  D_J_aux_eq_scan code _ i #[] (by omega)
