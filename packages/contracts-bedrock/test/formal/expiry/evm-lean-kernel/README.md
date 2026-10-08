# Kernel-checking the `native_decide` facts of `evm-lean`

**Result: all 585 `native_decide` facts behind the headline theorems are now checked by the Lean
kernel.** `expireMessage_outcome`, `expireMessage_success`, `expireMessage_revert_cause`,
`expireMessage_no_other_error` and `Abstract.refines_expire` depend only on `propext`,
`Classical.choice` and `Quot.sound`. No `…native_decide.ax_*` axiom remains, and the compiler is
no longer trusted. `ExpiryEvm/Axioms.lean` now asserts this, so `lake build` fails if a
`native_decide` returns. The full rebuild costs about 45 s more wall time (1:07 → 1:52 on a 32-core Linux build host).

This directory records how the `../evm-lean/` proof was moved from `native_decide` to kernel checking. The change is applied in `../evm-lean/` (`ExpiryEvm/KernelDecide.lean`, `ExpiryEvm/KernelRun.lean`, `ExpiryEvm/Axioms.lean`, `scripts/gen_bytecode.py`); what remains here is the measurement harness and data.


## What the 585 facts were

They were dumped with `measure/ListAx.lean` (`Lean.collectAxioms` on the headline theorems) and
classified by the constants they mention. The list is in `measure/native_axioms.tsv`.

| class | count | example |
|---|---|---|
| instruction decode | 516 | `decode l2tol2Runtime (ofNat 1713 + ⟨1⟩ + ⟨1⟩ + ofNat 21) = some (.EQ, none)` |
| pc arithmetic / trivial word equality | 36 | `ofNat 2048 + ⟨1⟩ + ⟨1⟩ + ofNat 3 + ⟨1⟩ = ofNat 2054` |
| JUMPDEST membership (`jump_dest`) | 32 | `#[⟨87⟩, …].contains (ofNat 4592) = true` |
| JUMPDEST table | 1 | `D_J l2tol2Runtime 0 = #[⟨87⟩, …, ⟨5185⟩]` |

These match the old README's split: "529 from block summaries" = decodes + pc arithmetic in
`l2tol2Blocks.*`, "55 in `seg_*`" = 32 memberships + 23 decodes (the two `STATICCALL`s, the
`evm_run` steps in `TraceStore`, and the hand-written `RD.*` steps).

**Precompile / `implemented_by` path: none of the 585 facts touches it.** `measure/Reach.lean`
computes every constant reachable from the 585 statements through definition bodies (936
constants). The results:

* `totallySafePerformIO`, `unsafePerformIO`, `ffi.sha256`, `ffi.BLAKE2Compress`, `Ξ`, `Θ` and `X`
  are all unreachable.
* The only `@[implemented_by]` constants reached are core ones: `Array.anyM`, `Array.foldlM`,
  `Lean.Name.*` and `List.attachWith`.
* The only `@[extern]` constants reached are core `Nat`, `Array`, `ByteArray` and `UInt8`
  primitives (e.g. `lean_byte_array_copy_slice`).

So compiled evaluation used to trust those core C implementations and the compiler. It never
trusted EVMLean's precompile shims. With this patch neither matters: the kernel ignores
`implemented_by` and `extern`.

## Approach

Two lemmas, both **proved for every `ByteArray`** in `../evm-lean/ExpiryEvm/KernelDecide.lean`, with no
axioms:

```lean
theorem decode_eq_decodeList (code : ByteArray) (pc : UInt256) :
    decode code pc = decodeList (code.data.toList.drop pc.toNat)
theorem D_J_eq_jumpdestScan (code : ByteArray) (i : Nat) :
    D_J code i = jumpdestScan (code.data.toList.length + 1) (code.data.toList.drop i) i #[]
```

Here `decodeList` and `jumpdestScan` are structural recursions on the byte *list*. The kernel
reduces them quickly. EVMLean's own definitions are slow in the kernel for three reasons:

* `ByteArray.get?` and `extract'` go through `Array` primitives.
* The chunked `l2tol2RuntimeChunk0 ++ … ++ Chunk5` makes the kernel evaluate `ByteArray.append`
  in every decode. That costs O(n²) per declaration.
* `D_J_aux` is well-founded recursion, which the kernel unfolds through `Acc.rec`.

So the trust is **not** reduced to "the decoder agrees with a reference on these bytes": that
agreement is a theorem. What remains is the kernel and EVMLean's definitions, as for every other
step of the proof.

The patch changes these things:

1. **`gen_bytecode.py`** emits `l2tol2Runtime` as one flat `⟨#[…5231 bytes…]⟩` literal instead of
   six appended chunks. The bytes and the name are the same. It proves `l2tol2ValidJumps` by
   `rw [D_J_eq_jumpdestScan]; decide +kernel`. The Python-computed table is still written out,
   and is now checked by the kernel.
2. **New tactics** (`KernelDecide.lean`):
   * `evm_kdecide` does `rw [decode_eq_decodeList]; decide +kernel`, or plain `decide +kernel` for
     non-decode goals.
   * `kjump_dest` does `simp only [valid_jumps, List.contains_toArray]; decide +kernel`.
   * **`KernelRun.lean`** adds `kevm_run`, which is EquiVM's `evm_run` macro (copied from
     `Reasoning/Reach.lean` @ `b0e9d55a`) with `native_decide` replaced by `evm_kdecide`.
3. **Generated block shards:** `(by native_decide)` becomes `(by evm_kdecide)`. That is all 3379
   occurrences, not only the 529 used, so no shard keeps compiler trust. `regen.sh` gets one
   `perl` line after `generate_rd_blocks.py` so that regeneration keeps this.
4. **Trace segments** (`TraceEntry`, `TraceCall1`, `TraceCall2`, `TraceStore`): `jump_dest` →
   `kjump_dest`, `native_decide` → `evm_kdecide`, `evm_run` → `kevm_run`, and `TraceStore` gets
   `import ExpiryEvm.KernelRun`. No proof structure changes.
5. **`Axioms.lean`** gets `#assert_std_axioms` for the five headline theorems: an error unless the
   axioms are ⊆ {`propext`, `Classical.choice`, `Quot.sound`}. It still prints the `Concrete.*`
   footprint.

The `Concrete.lean` executable witnesses keep `native_decide`; they are tests, not dependencies
(see "Not done").

## Measurements (32-core Linux build host, Lean 4.29.0, shared box at load 20–90, so ±30% noise)

### Per fact, kernel only (`measure/Measure.lean`)

Each statement is submitted to `Environment.addDeclCore` with proof `Eq.refl true`, synchronously,
and timed. Raw data: `measure/flat_all.tsv` and `measure/list_all.tsv`.

| approach | decode (516) | JUMPDEST table (1) | membership (32) | pc arith (36) |
|---|---|---|---|---|
| EVMLean `decode` on the original chunked bytecode | 0.5–0.75 s each (samples) | no result: the run that began with it produced nothing in 9 min 52 s | – | – |
| EVMLean `decode` on a flat literal | avg 640 ms, max 13 s, Σ 330 s, all ok | – | avg 262 ms (`Array.contains`) | ~0 ms |
| **proved list decoder + flat literal** (adopted) | **avg 100 ms, max 330 ms, Σ 52 s, all ok** | **2.8–4.0 s, all at once (no chunking)** | ~0 ms with `List.contains` | ~0 ms |
| proved list decoder + chunked literal | ~460 ms (samples) | – | – | – |

Micro-benchmarks (`measure/Micro*.lean`):

* Walking the 5231-element literal costs 60–200 ms per declaration whatever the method (recursor,
  `List.length`, `drop`), so that is the floor per decode.
* `parseInstr` costs about 10 ms.
* One Keccak permutation in the kernel (`KEC` on 3 bytes) takes 7.3 s.

**Does the JUMPDEST table need chunking? No.** The proved `jumpdestScan` checks the whole
5231-byte table in about 3–4 s in one declaration (`Bytecode.lean` builds in 8–15 s).

### Full build (`lake build ExpiryEvm ExpiryEvm.Axioms` after deleting the project's own oleans; dependencies prebuilt; same box, about 2 min apart)

| | wall | CPU (user) | peak RSS |
|---|---|---|---|
| original (`native_decide`) | 1:07 | 274 s | 4.6 GB |
| **kernel (this patch)**, all 3379 block facts + 55 segment facts + table | **1:52** | **588 s** | 6.4 GB |

Slowest shard: 71 s (`RuntimeBlocks_015`; it was 23 s). About 90 ms CPU per kernel-checked fact on
average. Run under `systemd-run --user --scope -p MemoryMax=16G -p MemorySwapMax=0`.

### Non-vacuity checks (all on the patched copy, then restored)

| mutation | expected | observed |
|---|---|---|
| T1: one `kjump_dest` in `TraceEntry` back to `jump_dest` (native) | `Axioms.lean` fails | `expireMessage_outcome depends on non-standard axioms (1): [ExpiryEvm.seg_entry._native.native_decide.ax_1_2]` (×5 theorems), build failed |
| T2: `EXPIRY_PERIOD` operand at pc 2179 `0a8c00` → `0a8c01` in the hex, regenerated | the decode in block 2175 fails in the kernel | `RuntimeBlocks_008.lean:339` error, build failed. Same for the other `PUSH3` copy at block 271 (`RuntimeBlocks_003.lean:239`) |
| T3: drop `⟨1713⟩` from the JUMPDEST table | `l2tol2ValidJumps` fails | "`decide` proved that the proposition … is false", build failed |
| unmutated | builds; standard axioms only | `Build completed successfully (3504 jobs)`; the five `standard axioms only ([propext, Quot.sound, Classical.choice])` lines |

## Remaining trust (replaces item 4 of `../evm-lean/README.md`)

The headline theorems are checked by the Lean 4.29.0 kernel and depend on `propext`,
`Classical.choice` and `Quot.sound` only. The trust base is:

* The kernel, including its built-in GMP acceleration of closed `Nat` operations
  (`Nat.add/sub/mul/div/mod/beq/ble/land/lor/xor/shiftLeft/shiftRight/pow/gcd/log2`).
* EVMLean's definitions (`decode`, `D_J`, `parseInstr`, `Ξ`, …).
* EquiVM's proved `Reasoning` library.

No compiled code, C runtime primitive, `@[implemented_by]`, `@[extern]` or `@[csimp]` lemma is
trusted by these theorems. The `Concrete.*` witnesses still use `native_decide` (compiled
evaluation of `Ξ`). They are executable tests, not dependencies.

## Not done / limits

* **`Concrete.lean` stays native.** For example, `decide +kernel` on
  `reverted 0x99 (5 + P_contract + 1) = true` fails immediately: kernel reduction gets stuck. `Ξ`,
  `Θ` and `X` are mutual well-founded recursion on a lexicographic measure. Kernel-running them
  would need a fuelled re-statement of the interpreter proved equal to `Ξ`, plus about 7 s of
  kernel time per Keccak permutation. They are still the reachability witnesses and are not
  dependencies of any headline theorem.
* **EquiVM's own library** contains `native_decide` (e.g. `Reasoning/EVMWord.lean`, `ABI.lean`,
  `Memory.lean:239`). None is in the dependency cone of the headline theorems: the axiom
  assertion proves it.
* **Sibling projects.** When these measurements were taken, `../evm-lean-bridge` (2136
  `native_decide` occurrences) and `../evm-lean-l1cdm` (6453) had not been converted. They have
  been since, with the same steps (flat literal, sed of the shards, tactic renames, axiom
  assertion), as has `../evm-lean-exporter`: each now asserts at build time that its headline
  theorems use no `native_decide` (see their READMEs). The per-fact timings here are for
  `../evm-lean` only.
* Timings come from a heavily shared box (other workers' forge/halmos/kontrol jobs). Ratios are
  more reliable than absolute numbers.

## Applying

Already applied in `../evm-lean/` (commit "kernel-check every bytecode fact in the expireMessage proof"). `../evm-lean/scripts/regen.sh` keeps it applied when the block summaries are regenerated (it rewrites `native_decide` to `evm_kdecide`).

## Files

| path | content |
|---|---|
| `../evm-lean/ExpiryEvm/KernelDecide.lean` | the two proved reformulations, `evm_kdecide`, `kjump_dest` |
| `measure/ListAx.lean`, `native_axioms.tsv` | dump and list of the 585 facts |
| `measure/Measure.lean`, `KTime.lean`, `KD.lean`, `gen_flat.py` | per-fact kernel timing harness (variants `flat`, `list`, `listchunk`; `KD.lean` is an earlier copy of `KernelDecide.lean` in namespace `KD`; it needs a `Kernel` lean_lib with `Kernel.Flat`, `Kernel.KTime`, `Kernel.KD` in the lakefile) |
| `measure/flat_all.tsv`, `list_all.tsv` | per-fact results (index, class, axiom, ok/FAIL, ms) |
| `measure/Reach.lean` | `implemented_by`/`extern` reachability of the 585 statements |
| `measure/Micro*.lean` | micro-benchmarks quoted above |

## Review

This directory holds measurement data and timing harnesses only, no proofs, so it has no review
log. The kernel-decision lemmas it describes live in `../evm-lean/ExpiryEvm/KernelDecide.lean` and
its siblings' copies, and are covered by those layers' builds and reviews.
