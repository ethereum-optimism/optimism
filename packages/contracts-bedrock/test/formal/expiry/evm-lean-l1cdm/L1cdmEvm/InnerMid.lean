import L1cdmEvm.InnerEntry
import L1cdmEvm.KernelRun

/-! # Inner trace, segment 2: `bytes` copy, `baseGas`, `messageNonce()`, the `relayMessage` and
`depositTransaction` encodings, the deposit `CALL`, the events and `++msgNonce` (pc 3152 → end) -/

namespace L1cdmEvm

set_option maxRecDepth 100000

open Ethereum Ethereum.EVM Reasoning.Theory Reasoning.Reach l1cdmBlocks L1cdmEvm.SymMem

/-- `messageNonce()` as the code computes it from slot 205. -/
abbrev nonceTerm (σ : AccountMap) (a : AccountAddress) : UInt256 :=
  UInt256.lor (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619776)
    (UInt256.land (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775)
      (storageWord σ a (UInt256.ofNat 205)))

/-- The descriptors of `depositCd` with `env = ienv σ I H t`. -/
def depositSyms : List Sym :=
  csel 0xe9e05c42 ++ vsyms 4 ++ csyms 0 ++ csyms 412835 ++ csyms 0 ++ csyms 0xa0 ++ csyms 324 ++
    (csel 0xd764ad0b ++ vsyms 3 ++ vsyms 2 ++ csyms 0x4200000000000000000000000000000000000023 ++ csyms 0 ++
      csyms 100000 ++ csyms 0xc0 ++ csyms 68 ++ csel 0x763a1cb7 ++ vsyms 0 ++ vsyms 1 ++ zeros 28) ++ zeros 28

/-- The environment of the inner frame's memory descriptors. -/
abbrev ienv (σ : AccountMap) (I : ExecutionEnv) (H t : UInt256) : List UInt256 :=
  [H, t, UInt256.ofNat I.source.val, nonceTerm σ I.codeOwner, otherMessengerWord σ I.codeOwner]

set_option maxHeartbeats 0 in
theorem depositSyms_eq (σ : AccountMap) (I : ExecutionEnv) (H t : UInt256) :
    Mem (ienv σ I H t) depositSyms = depositCd (otherMessengerWord σ I.codeOwner) (nonceTerm σ I.codeOwner)
      (UInt256.ofNat I.source.val) H t := by
  simp only [depositSyms, Mem_append, sel4_eq_Mem, Mem_csyms, Mem_vsyms, Mem_zeros, depositCd,
    relayMessageCd, expireMessageCd, ByteArray.append_assoc]
  rfl

set_option maxHeartbeats 0 in
theorem iseg_mid {σ σ₀ : AccountMap} {A : Substate} {I : ExecutionEnv} {g : Sat256} {H t : UInt256}
    {m : ByteArray} {aw : UInt256} {k C : ℕ}
    (hcd : I.calldata = sendMessageCd H t) (hv : I.weiValue = ⟨0⟩)
    (hF : Frame (ienv σ I H t) m 0 (swriteU (csyms 128) [] 64))
    (h : RD l1cdmRuntime I g (initState σ σ₀ g A I) (UInt256.ofNat 3152)
        [UInt256.ofNat 100000, UInt256.ofNat 68, UInt256.ofNat 132,
          UInt256.ofNat 376793390874373408599387495934666716005045108771, UInt256.ofNat 766,
          UInt256.ofNat 1035673643] m aw ByteArray.empty σ k C) :
    RDrev l1cdmRuntime g (initState σ σ₀ g A I) ∨
    (I.perm = false ∧ RDstatic l1cdmRuntime g (initState σ σ₀ g A I)) ∨
    (I.perm = true ∧ extCodeSizeWord σ (UInt256.land (storageWord σ I.codeOwner portalSlot) addrMask) ≠ ⟨0⟩ ∧
      ∃ σp o, DepositCall σ₀ I (depositCd (otherMessengerWord σ I.codeOwner) (nonceTerm σ I.codeOwner)
          (UInt256.ofNat I.source.val) H t) σ σp true o ∧
        RDret l1cdmRuntime g (initState σ σ₀ g A I)
          (sstoreAccountMap I.codeOwner σp (UInt256.ofNat 205) (bumpNonceWord (msgNonceWord σp I.codeOwner)))
          ByteArray.empty) := by
  have hM : I.calldata = Mem (ienv σ I H t) sendMessageSyms := by rw [hcd, sendMessageSyms_eq']
  obtain ⟨k1, C1, r1⟩ := l1cdm_block_3152 (by simp) (by kjump_dest) h
  simp only [l1cdm_block_3152_stack, l1cdm_block_3152_memory,
    frame_memLoad_c hF (UInt256.ofNat 64) 64 rfl 128 (by decide +kernel),
    show UInt256.ofNat 32 + (UInt256.ofNat 128 + UInt256.mul (UInt256.ofNat 32)
      (UInt256.div (UInt256.ofNat 68 + UInt256.ofNat 31) (UInt256.ofNat 32))) = UInt256.ofNat 256 by decide,
    show (UInt256.ofNat 128 + UInt256.ofNat 32).toNat = 160 by decide,
    show (UInt256.ofNat 128 + UInt256.ofNat 32 + UInt256.ofNat 68).toNat = 228 by decide,
    show (UInt256.ofNat 132).toNat = 132 by decide, show (UInt256.ofNat 68).toNat = 68 by decide,
    show (UInt256.ofNat 128).toNat = 128 by decide, optWord_eq, hM] at r1
  have hF1 := fr_const (fr_copy (fr_const (fr_const hF 256 _ rfl (UInt256.ofNat 64).toNat 64 rfl) 68 _ rfl 128 128 rfl)
    sendMessageSyms 132 68 (by norm_num) (by decide) 160 160 rfl) 0 _ rfl 228 228 rfl
  generalize (UInt256.toByteArray (UInt256.ofNat 0)).write 0
      ((Mem (ienv σ I H t) sendMessageSyms).write 132
        ((UInt256.toByteArray (UInt256.ofNat 68)).write 0
          ((UInt256.toByteArray (UInt256.ofNat 256)).write 0 m (UInt256.ofNat 64).toNat 32) 128 32)
        160 68) 228 32 = m1 at r1 hF1
  have q0 := l1cdm_block_5315 (by simp) (by kjump_dest) r1
  try simp only [l1cdm_block_5315_stack] at q0
  have q1 := l1cdm_block_10634_taken (by simp) (by decide) (by kjump_dest) q0
  try simp only [l1cdm_block_10634_taken_stack] at q1
  have q2 := l1cdm_block_10673 (by simp) (by kjump_dest) q1
  try simp only [l1cdm_block_10673_stack] at q2
  have q3 := l1cdm_block_5337 (by simp) (by kjump_dest) q2
  try simp only [l1cdm_block_5337_stack] at q3
  have q4 := l1cdm_block_10682_taken (by simp) (by decide) (by kjump_dest) q3
  try simp only [l1cdm_block_10682_taken_stack] at q4
  have q5 := l1cdm_block_10748 (by simp) (by kjump_dest) q4
  try simp only [l1cdm_block_10748_stack] at q5
  have q6 := l1cdm_block_5347 (by simp) (by kjump_dest) q5
  try simp only [l1cdm_block_5347_stack] at q6
  have q7 := l1cdm_block_10760_taken (by simp) (by decide) (by kjump_dest) q6
  try simp only [l1cdm_block_10760_taken_stack] at q7
  have q8 := l1cdm_block_10795 (by simp) (by kjump_dest) q7
  try simp only [l1cdm_block_10795_stack] at q8
  have q9 := l1cdm_block_5366 (by simp) (by kjump_dest) q8
  try simp only [l1cdm_block_5366_stack] at q9
  have q10 := l1cdm_block_10760_taken (by simp) (by decide) (by kjump_dest) q9
  try simp only [l1cdm_block_10760_taken_stack] at q10
  have q11 := l1cdm_block_10795 (by simp) (by kjump_dest) q10
  try simp only [l1cdm_block_10795_stack] at q11
  have q12 := l1cdm_block_5376 (by simp) (by kjump_dest) q11
  try simp only [l1cdm_block_5376_stack] at q12
  have q13 := l1cdm_block_10760_taken (by simp) (by decide) (by kjump_dest) q12
  try simp only [l1cdm_block_10760_taken_stack] at q13
  have q14 := l1cdm_block_10795 (by simp) (by kjump_dest) q13
  try simp only [l1cdm_block_10795_stack] at q14
  have q15 := l1cdm_block_5386 (by simp) (by kjump_dest) q14
  try simp only [l1cdm_block_5386_stack] at q15
  have q16 := l1cdm_block_10760_taken (by simp) (by decide) (by kjump_dest) q15
  try simp only [l1cdm_block_10760_taken_stack] at q16
  have q17 := l1cdm_block_10795 (by simp) (by kjump_dest) q16
  try simp only [l1cdm_block_10795_stack] at q17
  have q18 := l1cdm_block_5396 (by simp) (by kjump_dest) q17
  try simp only [l1cdm_block_5396_stack, frame_memLoad_c hF1 (UInt256.ofNat 128) 128 rfl 68 (by decide +kernel)] at q18
  have q19 := l1cdm_block_10804_taken (by simp) (by decide) (by kjump_dest) q18
  try simp only [l1cdm_block_10804_taken_stack] at q19
  have q20 := l1cdm_block_10823 (by simp) (by kjump_dest) q19
  try simp only [l1cdm_block_10823_stack] at q20
  have q21 := l1cdm_block_5425 (by simp) (by kjump_dest) q20
  try simp only [l1cdm_block_5425_stack] at q21
  have q22 := l1cdm_block_10634_taken (by simp) (by decide) (by kjump_dest) q21
  try simp only [l1cdm_block_10634_taken_stack] at q22
  have q23 := l1cdm_block_10673 (by simp) (by kjump_dest) q22
  try simp only [l1cdm_block_10673_stack] at q23
  have q24 := l1cdm_block_5441 (by simp) (by kjump_dest) q23
  try simp only [l1cdm_block_5441_stack] at q24
  have q25 := l1cdm_block_10760_taken (by simp) (by decide) (by kjump_dest) q24
  try simp only [l1cdm_block_10760_taken_stack] at q25
  have q26 := l1cdm_block_10795 (by simp) (by kjump_dest) q25
  try simp only [l1cdm_block_10795_stack] at q26
  have q27 := l1cdm_block_5451 (by simp) (by kjump_dest) q26
  try simp only [l1cdm_block_5451_stack] at q27
  have q28 := l1cdm_block_10634_taken (by simp) (by decide) (by kjump_dest) q27
  try simp only [l1cdm_block_10634_taken_stack] at q28
  have q29 := l1cdm_block_10673 (by simp) (by kjump_dest) q28
  try simp only [l1cdm_block_10673_stack] at q29
  have q30 := l1cdm_block_5472 (by simp) (by kjump_dest) q29
  try simp only [l1cdm_block_5472_stack] at q30
  have q31 := l1cdm_block_8516_taken (by simp) (by decide) (by kjump_dest) q30
  try simp only [l1cdm_block_8516_taken_stack] at q31
  have q32 := l1cdm_block_8532 (by simp) q31
  try simp only [l1cdm_block_8532_stack] at q32
  have q33 := l1cdm_block_8534 (by simp) (by kjump_dest) q32
  try simp only [l1cdm_block_8534_stack] at q33
  have q34 := l1cdm_block_5487 (by simp) (by kjump_dest) q33
  try simp only [l1cdm_block_5487_stack] at q34
  have q35 := l1cdm_block_10760_taken (by simp) (by decide) (by kjump_dest) q34
  try simp only [l1cdm_block_10760_taken_stack] at q35
  have q36 := l1cdm_block_10795 (by simp) (by kjump_dest) q35
  try simp only [l1cdm_block_10795_stack] at q36
  have q37 := l1cdm_block_5499 (by simp) (by kjump_dest) q36
  try simp only [l1cdm_block_5499_stack] at q37
  -- baseGas(message, 100000) = 412835 (concrete: the message length and gas limit are fixed)
  have q38 := RD_head_eq (b := UInt256.ofNat depositGasLimit) (by decide) q37
  clear q37
  -- messageNonce(), and abi.encodeWithSelector(relayMessage.selector, ...)
  obtain ⟨k2, C2, s1⟩ := l1cdm_block_3246 (by simp) (by kjump_dest) q38
  simp only [l1cdm_block_3246_stack, optWord_eq, hv] at s1
  have s2 := l1cdm_block_3354 (by simp) (by kjump_dest) s1
  simp only [l1cdm_block_3354_stack, hv, frame_memLoad_c hF1 (UInt256.ofNat 64) 64 rfl 256 (by decide +kernel)]
    at s2
  have s3 := l1cdm_block_10353 (by simp) (by kjump_dest) s2
  simp only [l1cdm_block_10353_stack, l1cdm_block_10353_memory,
    land_clean_mask (source_clean I),
    show UInt256.land (UInt256.ofNat 376793390874373408599387495934666716005045108771)
      (UInt256.ofNat 1461501637330902918203684832716283019655932542975) =
      UInt256.ofNat 376793390874373408599387495934666716005045108771 by decide,
    show UInt256.land (UInt256.ofNat 100000) (UInt256.ofNat 4294967295) = UInt256.ofNat 100000 by decide,
    show UInt256.ofNat 36 + UInt256.ofNat 256 = UInt256.ofNat 292 by decide,
    show UInt256.ofNat 292 + UInt256.ofNat 192 = UInt256.ofNat 484 by decide] at s3
  have hF2 := fr_const (fr_const (fr_const (fr_const (fr_var (fr_var hF1 3 (UInt256.lor
        (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619776)
        (UInt256.land (UInt256.ofNat 1766847064778384329583297500742918515827483896875618958121606201292619775)
          (storageWord σ I.codeOwner (UInt256.ofNat 205)))) rfl
      (UInt256.ofNat 292).toNat 292 rfl) 2 (UInt256.ofNat I.source.val) rfl (UInt256.ofNat 292 + UInt256.ofNat 32).toNat 324 (by decide))
      376793390874373408599387495934666716005045108771 _ rfl (UInt256.ofNat 292 + UInt256.ofNat 64).toNat 356
        (by decide))
      0 ⟨0⟩ rfl (UInt256.ofNat 292 + UInt256.ofNat 96).toNat 388 (by decide))
      100000 _ rfl (UInt256.ofNat 292 + UInt256.ofNat 128).toNat 420 (by decide))
      192 _ rfl (UInt256.ofNat 292 + UInt256.ofNat 160).toNat 452 (by decide)
  generalize (UInt256.toByteArray (UInt256.ofNat 192)).write 0 _ (UInt256.ofNat 292 + UInt256.ofNat 160).toNat 32
    = m2 at s3 hF2
  -- the `bytes calldata` message: length, CALLDATACOPY, zero padding
  have s4 := l1cdm_block_10280 (by simp) (by kjump_dest) s3
  simp only [l1cdm_block_10280_stack, l1cdm_block_10280_memory, hM] at s4
  have hF3 := fr_const (fr_copy (fr_const hF2 68 _ rfl (UInt256.ofNat 484).toNat 484 rfl)
    sendMessageSyms (UInt256.ofNat 132).toNat (UInt256.ofNat 68).toNat (by decide) (by decide)
      (UInt256.ofNat 484 + UInt256.ofNat 32).toNat 516 (by decide)) 0 _ rfl (UInt256.ofNat 484 + UInt256.ofNat 68 + UInt256.ofNat 32).toNat 584 (by decide)
  generalize (UInt256.toByteArray (UInt256.ofNat 0)).write 0 _
    (UInt256.ofNat 484 + UInt256.ofNat 68 + UInt256.ofNat 32).toNat 32 = m3 at s4 hF3
  have s5 := l1cdm_block_10435 (by simp) (by kjump_dest) s4
  simp only [l1cdm_block_10435_stack] at s5
  have s6 := l1cdm_block_3382 (by simp) (by kjump_dest) s5
  simp only [l1cdm_block_3382_stack, l1cdm_block_3382_memory,
    frame_memLoad_c hF3 (UInt256.ofNat 64) 64 rfl 256 (by decide +kernel),
    show UInt256.ofNat 484 + UInt256.land (UInt256.ofNat 68 + UInt256.ofNat 31) (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639904) + UInt256.ofNat 32
      = UInt256.ofNat 612 by decide,
    show UInt256.sub (UInt256.ofNat 612) (UInt256.ofNat 256) + UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639904 = UInt256.ofNat 324
      by decide,
    show UInt256.land (UInt256.ofNat 97425141450557520039415288145065866845346673525582403475645706348105484468224)
      (UInt256.ofNat 115792089210356248756420345214020892766250353992003419616917011526809519390720) =
      UInt256.ofNat (0xd764ad0b * 2 ^ 224) by decide,
    show UInt256.ofNat 26959946667150639794667015087019630673637144422540572481103610249215 =
      UInt256.ofNat (2 ^ 224 - 1) by norm_num] at s6
  have hF4 := fr_selmask (fr_const (fr_const hF3 324 _ rfl (UInt256.ofNat 256).toNat 256 rfl)
      612 _ rfl (UInt256.ofNat 64).toNat 64 rfl) 0xd764ad0b (by norm_num) (UInt256.ofNat 256 + UInt256.ofNat 32)
    288 (by decide) (UInt256.ofNat 256 + UInt256.ofNat 32).toNat 288 (by decide)
  generalize (UInt256.toByteArray _).write 0 _ (UInt256.ofNat 256 + UInt256.ofNat 32).toNat 32 = m4 at s6 hF4
  -- depositTransaction: selector at the new free pointer 612, then the head at 616
  obtain ⟨k3, C3, s7⟩ := l1cdm_block_7916 (by simp) (by kjump_dest) s6
  simp only [l1cdm_block_7916_stack, l1cdm_block_7916_memory, optWord_eq,
    frame_memLoad_c hF4 (UInt256.ofNat 64) 64 rfl 612 (by decide +kernel)] at s7
  have hF5 := fr_const hF4 (0xe9e05c42 * 2 ^ 224)
    (UInt256.ofNat 105785304202431811344621858410042342609260365835814029823897665188858597212160) (by norm_num)
    (UInt256.ofNat 612).toNat 612 rfl
  generalize (UInt256.toByteArray _).write 0 m4 (UInt256.ofNat 612).toNat 32 = m5 at s7 hF5
  have s8 := l1cdm_block_10898 (by simp) (by kjump_dest) s7
  simp only [l1cdm_block_10898_stack, l1cdm_block_10898_memory,
    land_clean_mask (clean_land_mask _),
    show UInt256.land (UInt256.ofNat depositGasLimit) (UInt256.ofNat 18446744073709551615) =
      UInt256.ofNat depositGasLimit by decide,
    show UInt256.isZero (UInt256.isZero (UInt256.ofNat 0)) = UInt256.ofNat 0 by decide,
    show UInt256.ofNat 4 + UInt256.ofNat 612 = UInt256.ofNat 616 by decide,
    show UInt256.ofNat 616 + UInt256.ofNat 160 = UInt256.ofNat 776 by decide] at s8
  have hF6 := fr_const (fr_const (fr_const (fr_const (fr_var hF5 4
      (UInt256.land (UInt256.ofNat 1461501637330902918203684832716283019655932542975)
        (storageWord σ I.codeOwner (UInt256.ofNat 207))) rfl
      (UInt256.ofNat 616).toNat 616 rfl) 0 ⟨0⟩ rfl (UInt256.ofNat 616 + UInt256.ofNat 32).toNat 648 (by decide))
      depositGasLimit _ rfl (UInt256.ofNat 616 + UInt256.ofNat 64).toNat 680 (by decide))
      0 _ rfl (UInt256.ofNat 616 + UInt256.ofNat 96).toNat 712 (by decide))
      160 _ rfl (UInt256.ofNat 616 + UInt256.ofNat 128).toNat 744 (by decide)
  generalize (UInt256.toByteArray (UInt256.ofNat 160)).write 0 _ (UInt256.ofNat 616 + UInt256.ofNat 128).toNat 32
    = m6 at s8 hF6
  -- abi_encode_bytes of the relayMessage calldata (324 bytes at 256) into the deposit's `_data`
  have s9 := l1cdm_block_9592 (by simp) s8
  simp only [l1cdm_block_9592_stack, l1cdm_block_9592_memory,
    frame_memLoad_c hF6 (UInt256.ofNat 256) 256 rfl 324 (by decide +kernel)] at s9
  have hF7 := fr_const hF6 324 _ rfl (UInt256.ofNat 776).toNat 776 rfl
  generalize (UInt256.toByteArray (UInt256.ofNat 324)).write 0 m6 (UInt256.ofNat 776).toNat 32 = m7 at s9 hF7
  have l0a := l1cdm_block_9602_fallthrough (by simp) (by decide) s9
  have l0b := l1cdm_block_9611 (by simp) (by kjump_dest) l0a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 0 = UInt256.ofNat 32 by decide] at l0b
  have hL0 := fr_load hF7 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 0)) 288 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 0 + UInt256.ofNat 776)).toNat 808 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 0)) m7)).write 0
    m7 (UInt256.ofNat 32 + (UInt256.ofNat 0 + UInt256.ofNat 776)).toNat 32 = mL0 at l0b hL0
  have l1a := l1cdm_block_9602_fallthrough (by simp) (by decide) l0b
  have l1b := l1cdm_block_9611 (by simp) (by kjump_dest) l1a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 32 = UInt256.ofNat 64 by decide] at l1b
  have hL1 := fr_load hL0 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 32)) 320 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 32 + UInt256.ofNat 776)).toNat 840 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 32)) mL0)).write 0
    mL0 (UInt256.ofNat 32 + (UInt256.ofNat 32 + UInt256.ofNat 776)).toNat 32 = mL1 at l1b hL1
  have l2a := l1cdm_block_9602_fallthrough (by simp) (by decide) l1b
  have l2b := l1cdm_block_9611 (by simp) (by kjump_dest) l2a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 64 = UInt256.ofNat 96 by decide] at l2b
  have hL2 := fr_load hL1 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 64)) 352 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 64 + UInt256.ofNat 776)).toNat 872 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 64)) mL1)).write 0
    mL1 (UInt256.ofNat 32 + (UInt256.ofNat 64 + UInt256.ofNat 776)).toNat 32 = mL2 at l2b hL2
  have l3a := l1cdm_block_9602_fallthrough (by simp) (by decide) l2b
  have l3b := l1cdm_block_9611 (by simp) (by kjump_dest) l3a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 96 = UInt256.ofNat 128 by decide] at l3b
  have hL3 := fr_load hL2 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 96)) 384 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 96 + UInt256.ofNat 776)).toNat 904 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 96)) mL2)).write 0
    mL2 (UInt256.ofNat 32 + (UInt256.ofNat 96 + UInt256.ofNat 776)).toNat 32 = mL3 at l3b hL3
  have l4a := l1cdm_block_9602_fallthrough (by simp) (by decide) l3b
  have l4b := l1cdm_block_9611 (by simp) (by kjump_dest) l4a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 128 = UInt256.ofNat 160 by decide] at l4b
  have hL4 := fr_load hL3 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 128)) 416 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 128 + UInt256.ofNat 776)).toNat 936 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 128)) mL3)).write 0
    mL3 (UInt256.ofNat 32 + (UInt256.ofNat 128 + UInt256.ofNat 776)).toNat 32 = mL4 at l4b hL4
  have l5a := l1cdm_block_9602_fallthrough (by simp) (by decide) l4b
  have l5b := l1cdm_block_9611 (by simp) (by kjump_dest) l5a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 160 = UInt256.ofNat 192 by decide] at l5b
  have hL5 := fr_load hL4 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 160)) 448 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 160 + UInt256.ofNat 776)).toNat 968 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 160)) mL4)).write 0
    mL4 (UInt256.ofNat 32 + (UInt256.ofNat 160 + UInt256.ofNat 776)).toNat 32 = mL5 at l5b hL5
  have l6a := l1cdm_block_9602_fallthrough (by simp) (by decide) l5b
  have l6b := l1cdm_block_9611 (by simp) (by kjump_dest) l6a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 192 = UInt256.ofNat 224 by decide] at l6b
  have hL6 := fr_load hL5 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 192)) 480 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 192 + UInt256.ofNat 776)).toNat 1000 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 192)) mL5)).write 0
    mL5 (UInt256.ofNat 32 + (UInt256.ofNat 192 + UInt256.ofNat 776)).toNat 32 = mL6 at l6b hL6
  have l7a := l1cdm_block_9602_fallthrough (by simp) (by decide) l6b
  have l7b := l1cdm_block_9611 (by simp) (by kjump_dest) l7a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 224 = UInt256.ofNat 256 by decide] at l7b
  have hL7 := fr_load hL6 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 224)) 512 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 224 + UInt256.ofNat 776)).toNat 1032 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 224)) mL6)).write 0
    mL6 (UInt256.ofNat 32 + (UInt256.ofNat 224 + UInt256.ofNat 776)).toNat 32 = mL7 at l7b hL7
  have l8a := l1cdm_block_9602_fallthrough (by simp) (by decide) l7b
  have l8b := l1cdm_block_9611 (by simp) (by kjump_dest) l8a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 256 = UInt256.ofNat 288 by decide] at l8b
  have hL8 := fr_load hL7 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 256)) 544 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 776)).toNat 1064 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 256)) mL7)).write 0
    mL7 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 776)).toNat 32 = mL8 at l8b hL8
  have l9a := l1cdm_block_9602_fallthrough (by simp) (by decide) l8b
  have l9b := l1cdm_block_9611 (by simp) (by kjump_dest) l9a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 288 = UInt256.ofNat 320 by decide] at l9b
  have hL9 := fr_load hL8 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 288)) 576 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 288 + UInt256.ofNat 776)).toNat 1096 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 288)) mL8)).write 0
    mL8 (UInt256.ofNat 32 + (UInt256.ofNat 288 + UInt256.ofNat 776)).toNat 32 = mL9 at l9b hL9
  have l10a := l1cdm_block_9602_fallthrough (by simp) (by decide) l9b
  have l10b := l1cdm_block_9611 (by simp) (by kjump_dest) l10a
  simp only [l1cdm_block_9611_stack, l1cdm_block_9611_memory,
    show UInt256.ofNat 32 + UInt256.ofNat 320 = UInt256.ofNat 352 by decide] at l10b
  have hL10 := fr_load hL9 (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 320)) 608 (by decide)
    (UInt256.ofNat 32 + (UInt256.ofNat 320 + UInt256.ofNat 776)).toNat 1128 (by decide)
  generalize (UInt256.toByteArray (memLoad (UInt256.ofNat 32 + (UInt256.ofNat 256 + UInt256.ofNat 320)) mL9)).write 0
    mL9 (UInt256.ofNat 32 + (UInt256.ofNat 320 + UInt256.ofNat 776)).toNat 32 = mL10 at l10b hL10
  -- loop exit (352 > 324) and zero padding after the 324 bytes
  have s10 := l1cdm_block_9602_taken (by simp) (by decide) (by kjump_dest) l10b
  have s11 := l1cdm_block_9630_fallthrough (by simp) (by decide) s10
  have s12 := l1cdm_block_9639 (by simp) s11
  simp only [l1cdm_block_9639_memory] at s12
  have hF8 := fr_const hL10 0 _ rfl (UInt256.ofNat 776 + UInt256.ofNat 324 + UInt256.ofNat 32).toNat 1132
    (by decide)
  generalize (UInt256.toByteArray (UInt256.ofNat 0)).write 0 mL10
    (UInt256.ofNat 776 + UInt256.ofNat 324 + UInt256.ofNat 32).toNat 32 = m8 at s12 hF8
  have s13 := l1cdm_block_9648 (by simp) (by kjump_dest) s12
  simp only [l1cdm_block_9648_stack] at s13
  have s14 := l1cdm_block_10975 (by simp) (by kjump_dest) s13
  simp only [l1cdm_block_10975_stack] at s14
  have hin := frame_read_eq hF8 612 548 (by norm_num) depositSyms (by decide +kernel) (by decide +kernel)
  rw [depositSyms_eq] at hin
  -- extcodesize(portal) != 0, then the CALL
  by_cases hcode : UInt256.isZero (UInt256.isZero (extCodeSizeWord σ
      (UInt256.land (storageWord σ I.codeOwner (UInt256.ofNat 252)) addrMask))) = UInt256.ofNat 0
  · left
    obtain ⟨k4, C4, s15⟩ := l1cdm_block_8013_fallthrough (by simp) hcode s14
    exact l1cdm_block_8034 (by simp [l1cdm_block_8013_fallthrough_stack]) s15
  obtain ⟨k4, C4, s15⟩ := l1cdm_block_8013_taken (by simp) hcode (by kjump_dest) s14
  simp only [l1cdm_block_8013_taken_stack, frame_memLoad_c hF8 (UInt256.ofNat 64) 64 rfl 612 (by decide +kernel),
    show UInt256.ofNat 32 + (UInt256.land (UInt256.ofNat
      115792089237316195423570985008687907853269984665640564039457584007913129639904)
      (UInt256.ofNat 31 + UInt256.ofNat 324) + UInt256.ofNat 776) = UInt256.ofNat 1160 by decide,
    show UInt256.sub (UInt256.ofNat 1160) (UInt256.ofNat 612) = UInt256.ofNat 548 by decide] at s15
  have s16 := l1cdm_block_8038 (by simp) s15
  simp only [l1cdm_block_8038_stack] at s16
  have hecs : extCodeSizeWord σ (UInt256.land (storageWord σ I.codeOwner portalSlot) addrMask) ≠ ⟨0⟩ := by
    intro h0; apply hcode; rw [show portalSlot = UInt256.ofNat 252 from rfl] at h0; rw [h0]; rfl
  have hdec : decode l1cdmRuntime (UInt256.ofNat 8041) = some (.CALL, .none) := by evm_kdecide
  by_cases hd : I.depth.val < 1024
  swap
  · have hd' : I.depth = 1024 := by
      apply Fin.ext; have := I.depth.isLt; omega
    obtain ⟨k5, C5, s17⟩ := RD.callDepthLimit s16 hdec hd' (by simp)
    left
    exact l1cdm_block_8049 (by simp [l1cdm_block_8042_fallthrough_stack])
      (l1cdm_block_8042_fallthrough (by simp) (by decide) s17)
  obtain ⟨σp, z, o, A_in, callGas, k5, C5, ⟨g'', A', hΘ⟩, s17, _⟩ := RD.call s16 hdec hd (by simp)
  rw [show (UInt256.ofNat 612).toNat = 0 + 612 from rfl, show (UInt256.ofNat 548).toNat = 548 from rfl, hin,
    AccountAddress.ofUInt256_ofNat I.codeOwner, ofUInt256_land_mask] at hΘ
  cases z with
  | false =>
    left
    simp only [Bool.false_eq_true, if_false] at s17
    exact l1cdm_block_8049 (by simp [l1cdm_block_8042_fallthrough_stack])
      (l1cdm_block_8042_fallthrough (by simp) (by decide) s17)
  | true =>
    simp only [if_true] at s17
    have s18 := l1cdm_block_8042_taken (by simp) (by decide) (by kjump_dest) s17
    simp only [l1cdm_block_8042_taken_stack] at s18
    have s19 := l1cdm_block_8058 (by simp) (by kjump_dest) s18
    simp only [l1cdm_block_8058_stack] at s19
    obtain ⟨k6, C6, s20⟩ := l1cdm_block_3512 (by simp) (by kjump_dest) s19
    simp only [l1cdm_block_3512_stack] at s20
    have s21 := l1cdm_block_3645 (by simp) (by kjump_dest) s20
    simp only [l1cdm_block_3645_stack] at s21
    have s22 := l1cdm_block_10448 (by simp) (by kjump_dest) s21
    simp only [l1cdm_block_10448_stack] at s22
    have s23 := l1cdm_block_10280 (by simp) (by kjump_dest) s22
    simp only [l1cdm_block_10280_stack] at s23
    have s24 := l1cdm_block_10496 (by simp) (by kjump_dest) s23
    simp only [l1cdm_block_10496_stack] at s24
    cases hp : I.perm with
    | false =>
      right; left
      refine ⟨rfl, ?_⟩
      have q := kevm_run s24 with [jumpdest, push1 (UInt256.ofNat 64)]
      have q2 := RD.genMload q (by evm_kdecide) (by evm_ov)
      have q3 := kevm_run q2 with [dup1, swap2, sub, swap1]
      exact RD.log2Static q3 hp (by evm_kdecide) (by evm_ov)
    | true =>
      right; right
      refine ⟨rfl, hecs, σp, o, ⟨A_in, callGas, g'', A', ?_⟩, ?_⟩
      · rw [hp] at hΘ; rw [hp]; exact hΘ
      obtain ⟨k7, C7, s25⟩ := l1cdm_block_3663 (by simp) hp (by kjump_dest) s24
      simp only [l1cdm_block_3663_stack, optWord_eq] at s25
      exact l1cdm_block_766 (by simp) s25

end L1cdmEvm
