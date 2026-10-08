import ExporterEvm.TraceCall
import ExporterEvm.LogTrack

/-!
# Trace segment 5 (pc 710 → `RETURN`): the event and the return value

`emit UndeliveredMessageExported(H, _source, _sourceMessenger, block.timestamp)` (`LOG3` at
pc 796; a static frame raises `StaticModeViolation` there) and `return messageHash_` (pc 78/88):
the output is exactly the 32-byte word `H`.
-/

namespace ExporterEvm

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach exporterBlocks Mem

/-- The stack at pc 710. -/
abbrev logStack (I : ExecutionEnv) (H : UInt256) (fp : ℕ) : List UInt256 :=
  UInt256.isZero ⟨1⟩ :: UInt256.ofNat (fp + 328) :: UInt256.ofNat 1035673643 :: l2cdmWord :: H :: staticTail I

theorem len_take_fill (X : List UInt8) (c : ℕ) :
    (X.take c ++ List.replicate (c - X.length) (0 : UInt8)).length = c := by
  simp; omega

/-- pc 78 → 88: `mstore(p, H); return(p, 32)` with `p = fp + 100`. -/
theorem ret_tail {ee : ExecutionEnv} {g : Sat256} {s0 : State} {σ₂ : AccountMap} {aw : UInt256}
    {k C : ℕ} {H x : UInt256} {B Q : List UInt8} {fp : ℕ} {rd : ByteArray}
    (hB : FmpMem B fp) (hfp : fp < 2 ^ 140)
    (h : RD exporterRuntime ee g s0 (UInt256.ofNat 78) [H, x]
        (ofL (pre (setFmp B (fp + 100)) fp ++ Q)) aw rd σ₂ k C) :
    RDret exporterRuntime g s0 σ₂ (UInt256.toByteArray H) := by
  have h96 := hB.len96
  have hge := hB.ge
  obtain ⟨_, _, _, r1⟩ := exporter_block_78_packed (by simp) h
  simp only [exporter_block_78_stack, exporter_block_78_memory, memLoad_setFmp_pre B _ fp _ h96 hge] at r1
  rw [toNat_small (by omega), write_rel_pre] at r1
  have r2 := exporter_block_88 (by simp) r1
  rw [memLoad_setFmp_pre B _ fp _ h96 hge, Words.sub_add_self', toNat_small (by omega),
    toNat_small (n := 32) (by norm_num), read_rel_pre _ _ _ 100 32 (by norm_num) (by norm_num)
      (by rw [List.length_append, List.length_append, len_take_fill]; simp),
    List.append_assoc, List.drop_left' (len_take_fill _ _), List.take_left' (by simp),
    ← toByteArray_eq_ofL] at r2
  exact r2

/-- The 64 bytes `LOG3` reads after `mstore(p, a); mstore(p + 32, b)` at `p = fp + 100`. -/
theorem two_writes (Q : List UInt8) (hQ : 164 ≤ Q.length) (a b : UInt256) :
    (((Q.take 100 ++ List.replicate (100 - Q.length) (0 : UInt8) ++ wb a ++ Q.drop (100 + 32)).take 132 ++
      List.replicate (132 - (Q.take 100 ++ List.replicate (100 - Q.length) (0 : UInt8) ++ wb a ++
        Q.drop (100 + 32)).length) (0 : UInt8) ++ wb b ++
      (Q.take 100 ++ List.replicate (100 - Q.length) (0 : UInt8) ++ wb a ++ Q.drop (100 + 32)).drop (132 + 32)).drop
        100).take 64 =
    wb a ++ wb b := by
  have h1 : (Q.take 100).length = 100 := by simp; omega
  rw [show 100 - Q.length = 0 by omega, List.replicate_zero, List.append_nil]
  have hl : (Q.take 100 ++ wb a ++ Q.drop (100 + 32)).length = Q.length := by simp; omega
  have hX1 : (Q.take 100 ++ wb a ++ Q.drop (100 + 32)).take 132 = Q.take 100 ++ wb a :=
    List.take_left' (by simp; omega)
  have hl1 : (Q.take 100 ++ wb a).length = 132 := by simp; omega
  have hX2 : (Q.take 100 ++ wb a ++ Q.drop (100 + 32)).drop (132 + 32) = Q.drop 164 := by
    rw [List.drop_append, List.drop_eq_nil_of_le (by rw [hl1]; omega), List.nil_append, hl1, List.drop_drop]
  rw [hl, show 132 - Q.length = 0 by omega, List.replicate_zero, List.append_nil, hX1, hX2]
  simp only [List.append_assoc]
  rw [List.drop_left' h1, ← List.append_assoc, List.take_left' (by simp)]

/-- After the `LOG3` (pc 797 → `RETURN` at pc 96), tracking the last log entry: the jump back to
    the dispatcher's return block, `mstore(p, H)` and `return(p, 32)`. -/
theorem log_tail {ee : ExecutionEnv} {g : Sat256} {s0 : State} {σ₂ : AccountMap} {aw : UInt256}
    {k C : ℕ} {H x1 x2 x3 x4 x5 x6 x7 x8 x : UInt256} {B Q : List UInt8} {fp : ℕ} {rd : ByteArray}
    {e : LogEntry} (hB : FmpMem B fp) (hfp : fp < 2 ^ 140)
    (h : RDL exporterRuntime ee g s0 (UInt256.ofNat 797) [H, x1, x2, x3, x4, x5, x6, x7, x8, UInt256.ofNat 78, x]
        (ofL (pre (setFmp B (fp + 100)) fp ++ Q)) aw rd σ₂ e k C) :
    RDretL exporterRuntime g s0 σ₂ (UInt256.toByteArray H) e := by
  have h96 := hB.len96
  have hge := hB.ge
  have t1 := RDL.stepSwap h (fun _ hc hp hs => swap9_xstep hc hp (by evm_kdecide) hs (by simp))
  have t2 := RDL.stepSwap t1 (fun _ hc hp hs => swap8_xstep hc hp (by evm_kdecide) hs (by simp))
  have t3 := RDL.pop (RDL.pop (RDL.pop (RDL.pop t2 (by evm_kdecide) (by simp)) (by evm_kdecide) (by simp))
    (by evm_kdecide) (by simp)) (by evm_kdecide) (by simp)
  have t4 := RDL.pop (RDL.pop (RDL.pop (RDL.pop t3 (by evm_kdecide) (by simp)) (by evm_kdecide) (by simp))
    (by evm_kdecide) (by simp)) (by evm_kdecide) (by simp)
  have t5 := RDL.jump t4 (by evm_kdecide) (by kjump_dest) (by simp)
  have t6 := RDL.push1 (RDL.jumpdest t5 (by evm_kdecide) (by simp)) (UInt256.ofNat 64) (by evm_kdecide) (by simp)
  have t7 := RDL.genMload t6 (by evm_kdecide) (by simp)
  rw [memLoad_setFmp_pre B _ fp _ h96 hge] at t7
  have t8 := RDL.stepSwap t7 (fun _ hc hp hs => swap1_xstep hc hp (by evm_kdecide) hs (by simp))
  have t9 := RDL.stepSwap t8 (fun _ hc hp hs => dup2_xstep hc hp (by evm_kdecide) hs (by simp))
  have t10 := RDL.genMstore t9 (by evm_kdecide) (by simp)
  rw [toNat_small (by omega), write_rel_pre] at t10
  have t11 := RDL.push1 t10 (UInt256.ofNat 32) (by evm_kdecide) (by simp)
  have t12 := RDL.stepBinop t11 (fun _ hc hp hs => add_xstep hc hp (by evm_kdecide) hs (by simp))
  have t13 := RDL.push1 (RDL.jumpdest t12 (by evm_kdecide) (by simp)) (UInt256.ofNat 64) (by evm_kdecide)
    (by simp)
  have t14 := RDL.genMload t13 (by evm_kdecide) (by simp)
  rw [memLoad_setFmp_pre B _ fp _ h96 hge] at t14
  have t15 := RDL.dup1 t14 (by evm_kdecide) (by simp)
  have t16 := RDL.stepSwap t15 (fun _ hc hp hs => swap2_xstep hc hp (by evm_kdecide) hs (by simp))
  have t17 := RDL.stepBinop t16 (fun _ hc hp hs => sub_xstep hc hp (by evm_kdecide) hs (by simp))
  have t18 := RDL.stepSwap t17 (fun _ hc hp hs => swap1_xstep hc hp (by evm_kdecide) hs (by simp))
  have t19 := RDL.ret t18 (by evm_kdecide) (by simp)
  rw [Words.sub_add_self', toNat_small (by omega), toNat_small (n := 32) (by norm_num),
    read_rel_pre _ _ _ 100 32 (by norm_num) (by norm_num)
      (by rw [List.length_append, List.length_append, len_take_fill]; simp),
    List.append_assoc, List.drop_left' (len_take_fill _ _), List.take_left' (by simp),
    ← toByteArray_eq_ofL] at t19
  exact t19

set_option maxHeartbeats 4000000 in
theorem seg_log {σ σ₀ σ₂ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256}
    {aw : UInt256} {k C : ℕ} {H : UInt256} {B Q : List UInt8} {fp : ℕ} {rd : ByteArray}
    (hargs : ArgsOk I) (hB : FmpMem B fp) (hfp : fp < 2 ^ 140) (hQ : 164 ≤ Q.length)
    (h : RD exporterRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 710) (logStack I H fp)
        (ofL (pre (setFmp B (fp + 100)) fp ++ Q)) aw rd σ₂ k C) :
    (I.perm = true ∧
      RDretL exporterRuntime g (initState σ σ₀ g A I) σ₂ (UInt256.toByteArray H) (exportedLog I H)) ∨
    (I.perm = false ∧ RDstatic exporterRuntime g (initState σ σ₀ g A I)) := by
  have h96 := hB.len96
  have hge := hB.ge
  cases hp : I.perm with
  | false =>
    right
    refine ⟨rfl, ?_⟩
    have q1 := kevm_run h with [jumpdest, pop, pop, push1 (UInt256.ofNat 64), dup1]
    have q2 := RD.genMload q1 (by evm_kdecide) (by stk_ov)
    have q3 := q2.pushConst (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (width := 20)
      (op := .PUSH20) (by decide) (by evm_kdecide) (by stk_ov)
    have q4 := kevm_run q3 with [dup14, and, dup2]
    have q5 := RD.genMstore q4 (by evm_kdecide) (by stk_ov)
    have q6 := kevm_run q5 with [timestamp, push1 (UInt256.ofNat 32), dup3, add]
    have q7 := RD.genMstore q6 (by evm_kdecide) (by stk_ov)
    have q8 := kevm_run q7 with [dup12, swap4, pop, dup5, swap3, pop]
    have q9 := q8.pushConst exportedTopic (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide)
      (by stk_ov)
    have q10 := kevm_run q9 with [swap2, add, push1 (UInt256.ofNat 64)]
    have q11 := RD.genMload q10 (by evm_kdecide) (by stk_ov)
    have q12 := kevm_run q11 with [dup1, swap2, sub, swap1]
    exact RD.log3Static q12 hp (by evm_kdecide) (by stk_ov)
  | true =>
    left
    refine ⟨rfl, ?_⟩
    have hS := hargs.srcMessengerClean
    have q1 := kevm_run h with [jumpdest, pop, pop, push1 (UInt256.ofNat 64), dup1]
    have q2 := RD.genMload q1 (by evm_kdecide) (by stk_ov)
    rw [memLoad_setFmp_pre B _ fp _ h96 hge] at q2
    have q3 := q2.pushConst (UInt256.ofNat 1461501637330902918203684832716283019655932542975) (width := 20)
      (op := .PUSH20) (by decide) (by evm_kdecide) (by stk_ov)
    have q4 := kevm_run q3 with [dup14, and, dup2]
    have q5 := RD.genMstore q4 (by evm_kdecide) (by stk_ov)
    rw [toNat_small (by omega), write_rel_pre, Words.land_mask_of_lt hS] at q5
    have q6 := kevm_run q5 with [timestamp, push1 (UInt256.ofNat 32), dup3, add]
    have q7 := RD.genMstore q6 (by evm_kdecide) (by stk_ov)
    rw [ofNat_add_ofNat, toNat_small (by omega), show fp + 100 + 32 = fp + 132 by omega, write_rel_pre] at q7
    have q8 := kevm_run q7 with [dup12, swap4, pop, dup5, swap3, pop]
    have q9 := q8.pushConst exportedTopic (width := 32) (op := .PUSH32) (by decide) (by evm_kdecide)
      (by stk_ov)
    have q10 := kevm_run q9 with [swap2, add, push1 (UInt256.ofNat 64)]
    have q11 := RD.genMload q10 (by evm_kdecide) (by stk_ov)
    rw [memLoad_setFmp_pre B _ fp _ h96 hge] at q11
    have q12 := kevm_run q11 with [dup1, swap2, sub, swap1]
    rw [Words.sub_add_self'] at q12
    obtain ⟨k13, C13, q13⟩ := RDL.log3 q12 (by evm_kdecide) hp (by stk_ov)
    rw [toNat_small (by omega), toNat_small (n := 64) (by norm_num),
      read_rel_pre _ _ _ 100 64 (by norm_num) (by norm_num) (by simp; omega), two_writes Q hQ,
      ← append_ofL (wb (argSrcMessenger I)), ← toByteArray_eq_ofL, ← toByteArray_eq_ofL] at q13
    have q14 := RDL.normalizePC (pc' := UInt256.ofNat 797) q13 (by decide)
    exact log_tail hB hfp q14

end ExporterEvm
