# Bytecode-level soundness of `SuperchainETHBridge.refundETH` in Lean (EquiVM / EVMLean)

Status: **all claimed theorems proved.** `lake build BridgeEvm BridgeEvm.Axioms` passes. There is
no `sorry` or `admit`. Every headline theorem depends only on `propext`, `Classical.choice` and
`Quot.sound`: there are **0 `native_decide` axioms**, and `BridgeEvm/Axioms.lean` asserts this at
build time with `#assert_std_axioms`. Soundness and the outcome classification are claimed;
completeness is not (see "Not covered").

This is a sibling of `../evm-lean/` (the `expireMessage` proof). It uses the same toolchain pins and
the same method, plus three extensions this function needs:

* a list-based memory model for unaligned writes and the dynamic `bytes` hash (`Mem.lean`);
* a proved `RD` lemma for `CREATE` (`Create.lean`; the block generator stops at `CREATE`);
* memory above a symbolic free-memory pointer. The pointer moves by
  `roundUp32(returndatasize)` after the `expiredMessages` call.

## What is proved, in one paragraph

Take the **deployed runtime bytecode** of `SuperchainETHBridge` and run it with EVMLean's
code-execution function `Ξ` on calldata that selects
`refundETH(uint256,uint256,address,address,uint256)`. The theorems hold for every account map,
caller, value, depth, gas, permission and execution environment. If the run **succeeds**, all of
the following hold:

* The ABI conditions hold: no ETH was sent, the calldata is at least 164 bytes, and `from` and `to`
  are clean addresses.
* A `STATICCALL` went to `L2ToL2CrossDomainMessenger` (0x4200…0023) with calldata
  `expiredMessages(H)`. That call succeeded and returned at least 32 bytes, and the first word was
  `1`. Here `H = keccak256(refundPreimage)`, and the 352 preimage bytes are given below.
* `refunded[H]` was false in the pre-state.
* The bridge's storage became the pre-state's storage with `refunded[H]` set to true. Nothing else
  changed.
* ETHLiquidity (0x4200…0025) had code, and a `CALL` to it with value 0 and calldata `mint(amount)`
  succeeded.
* A `CREATE` ran with endowment `amount` and init code `SafeSend.creationCode ‖ abi.encode(from)`.
  It succeeded and pushed a nonzero address, and it produced the final account map.

No external call is assumed to succeed or to return any particular value. Each call is a
*conclusion*: the theorem names its exact input, and the output that made the code continue.

## Toolchain (all pinned; same as `../evm-lean`)

| Component | Version |
|---|---|
| Lean | `leanprover/lean4:v4.29.0` (`lean-toolchain`) |
| [EquiVM](https://github.com/argotorg/EquiVM) (MIT) | `b0e9d55a277bc081411fd11f1b69f2e11b38e478` |
| [EVMLean](https://github.com/lefterislazar/EVMLean) | `63f61339dd17a809961e34028cbc8c5fcb02af13` (Cancun semantics) |
| Mathlib | `v4.29.0` (`8a178386…`), via EquiVM |

The exact transitive pins are in `lake-manifest.json`. The kernel-decision machinery
(`KernelDecide.lean`, `KernelRun.lean`, `scripts/gen_bytecode.py`) is copied from `../evm-lean` and
works for any contract. See `../evm-lean-kernel/README.md`.

## The artifact proved against

| | |
|---|---|
| Source | `src/L2/SuperchainETHBridge.sol` at `5992028e08` (tip of `karl/message-expiry-refunds` when this was written). Re-checked at the later tip `52ff613e14`: `SuperchainETHBridge.sol`, `SafeSend.sol`, `Hashing.sol` and `foundry.toml` are unchanged, and the `semver-lock.json` init-code hash is identical. |
| Compiler | solc `0.8.15+commit.e14f2714` via forge 1.8.1, repository **default** profile |
| Settings | `{"evmVersion":"london","libraries":{},"metadata":{"bytecodeHash":"none"},"optimizer":{"enabled":true,"runs":999999}}` (solc 0.8.15 caps the profile's `cancun` at `london`) |
| Runtime | 3131 bytes, `keccak256 = 0x74d5d26355c633189db6e892ba12456b1481ade7a2446b295dbc1d42cc32d2a2` (`bytecode/SuperchainETHBridge.runtime.hex`) |
| Init code | `keccak256 = 0x7387ae889af66f29bcd2f95e32e06ddfab05be17bd9dd2671195815bd32b596a` = `initCodeHash` in `snapshots/semver-lock.json` |
| SafeSend creation code | 89 bytes, `keccak256 = 0xfd5b265533779bec7c59f23224f55fcae732beb46dc9a7ebe6b132230063c8e5` (`bytecode/SafeSend.creation.hex`). It is embedded in the bridge runtime at offset 3030, which is the `CODECOPY` source of `new SafeSend`. |
| Lean | `BridgeEvm/Bytecode.lean` (`ethbridgeRuntime`, one flat literal, generated). Its JUMPDEST table is checked by the kernel. |

To reproduce, run `scripts/regen.sh`. It forces the default profile and unsets every `FOUNDRY_*`
override, then builds into a temporary directory. It refuses to replace anything unless the solc
version and the full settings JSON match, the init-code hash equals `semver-lock.json`, and the
SafeSend creation code sits at offset 3030. It then regenerates `Bytecode.lean`, the block summaries
`Blocks/` (with `native_decide` rewritten to the kernel tactic `evm_kdecide`) and `AllBlocks.lean`.
The proofs refer to concrete pcs, so a layout change breaks them loudly.

## Statements

The main file is `BridgeEvm/Refund.lean`. The vocabulary is in `BridgeEvm/Spec.lean`. Every theorem
takes these hypotheses:

```lean
(hcode : I.code = ethbridgeRuntime)          -- the pinned artifact runs (any code owner: proxy frame)
(hsel  : selectorWord I = refundSelector)    -- CALLDATALOAD(0) >> 224 = 0xe17a776b
(hcds  : I.calldata.size < 2 ^ 256)          -- always true in the EVM
```

Each theorem is universally quantified over `σ σ₀ A g I`. No other hypothesis appears in
`refundETH_outcome`, `refundETH_success` or `refundETH_no_other_error`.

```lean
theorem refundETH_outcome … :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    (∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o)) ∨
    (Ξ σ σ₀ g A I = .error .StaticModeViolation ∧ I.perm = false) ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') ByteArray.empty) ∧ RefundRun σ σ₀ I σ')

theorem refundETH_success … (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    RefundRun σ σ₀ I σ' ∧ o = ByteArray.empty

theorem refundETH_no_other_error … (he : Ξ σ σ₀ g A I = .error e) :
    e = .OutOfGass ∨ (e = .StaticModeViolation ∧ I.perm = false)
```

`refundETH_trace` is the same statement at the level of EquiVM's `RD` invariant.

### `RefundRun σ σ₀ I σ'` (with `H = refundHash I`)

```lean
ArgsOk I ∧ I.perm = true ∧
∃ σ₁ oE, StaticCall σ₀ I l2l2 (expiredCalldata H) σ σ₁ true oE ∧   -- expiredMessages(H) succeeded
  32 ≤ oE.size ∧ returnWord oE = 1 ∧                              -- … and returned true
  accountStorageStateEq σ σ₁ ∧ accountCodeStateEq σ σ₁ ∧          -- (proved for static calls)
  land 0xff (refundedWord σ I H) = 0 ∧                            -- refunded[H] was false
  extCodeSizeWord (storedMap σ₁ I H) ethLiqWord ≠ 0 ∧             -- ETHLiquidity has code
  ∃ σ₃ oM, CallTo σ₀ I ethLiq (mintCalldata amount) (storedMap σ₁ I H) σ₃ true oM ∧  -- mint ok
  ∃ x rd', CreateStep I σ₀ σ₃ amount (safeSendDeploy from) x σ' rd' ∧ x ≠ 0           -- CREATE ok
```

* `ArgsOk I`: `I.weiValue = 0` (non-payable). The calldata size is at least 164 and below
  `2^255 + 4`, where solc's signed length check wraps. `from` and `to` are below `2^160`, because
  solc's address validator reverts otherwise.
* `argDest/argNonce/argFrom/argTo/argAmount I` are `CALLDATALOAD(4/36/68/100/132)`.
* `StaticCall` and `CallTo` are EVMLean's message-call function `Θ`, with the real arguments of
  the call site: sender = the executing account, value 0, the given calldata, depth + 1, `w = false`
  for the static call and `w = I.perm` for the call. The call gas and the input substate are
  existentially quantified, because the `RD` framework does not expose them.
* `expiredCalldata H = 0xd5a522fe ‖ H` (36 bytes). `mintCalldata a = 0xa0712d68 ‖ a` (36 bytes).
* `returnWord o` is the first 32 bytes of the return data as a word. solc accepts any return data
  of at least 32 bytes and requires the word to be a clean bool. The theorem says exactly that,
  and does not require exactly 32 bytes.
* `storedMap σ₁ I H = sstoreAccountMap I.codeOwner σ₁ (refundedSlot H) (setTrueWord (refundedWord σ₁ I H))`.
  Here `refundedSlot H = keccak256(H ‖ 0)` (storage layout: `refunded` at slot 0), and
  `setTrueWord old = (old & ~0xff) | 1` (Solidity's packed-bool write).
* `CreateStep I σ₀ σ value i x σ' rd'` is what EVMLean's `step_create` computes for the executing
  account with endowment `value` and init code `i`. It has three cases. If the creator's nonce is
  at least `2^64-1`, nothing happens and `x = 0`. If the balance is too low, the depth is 1024, or
  the init code is longer than 49152 bytes, nothing happens and `x = 0`. Otherwise `Lambda` (the
  contract-creation function) runs from `σ` with the creator's nonce incremented, `σ'` is its
  resulting map, and `x` is the new address on success and `0` on failure. `createStep_success`
  extracts the third case from `x ≠ 0`: `Lambda (σ with nonce+1) σ₀ A I.codeOwner I.sender gas …
  value i … = (a, σ', g', A', true, o)` and `x = a`.
* `safeSendDeploy from = safeSendInitcode ‖ toByteArray from` (121 bytes). `safeSendInitcode` is the
  89-byte literal, checked by the kernel to equal the bytes `CODECOPY` copies out of the runtime
  (`code_safeSend`).

### The exact preimage (`refundPreimage I`, 352 bytes; `refundPreimage_size`)

| bytes | content |
|---|---|
| 0..32 | `_destination` |
| 32..64 | `block.chainid` (EVMLean: the constant `Ethereum.chainId`; see notes) |
| 64..96 | `_nonce` |
| 96..128 | `address(this)` (the executing account, i.e. the proxy) |
| 128..160 | `address(this)` |
| 160..192 | `0xc0` (offset of the `bytes` tail) |
| 192..224 | `100` (length of the `bytes`) |
| 224..228 | `0x4f0edcc9` (`relayETH` selector) |
| 228..260 | `_from` |
| 260..292 | `_to` |
| 292..324 | `_amount` |
| 324..352 | 28 zero bytes |

`refundHash I = keccak256(refundPreimage I)` uses EVMLean's `KEC`. The proof shows that the word
the code puts on the stack after its `KECCAK256` is exactly `refundHash I`. To check this, the hash
is computed through `abi.encodeCall`, `abi.encode`, solc's unrolled 4-iteration copy loop, the
selector merge and the `KECCAK256` (`TraceHash.lean`, `seg_hash`).

### Post-state lemmas (`BridgeEvm/Post.lean`)

```lean
theorem storedMap_post (hst : accountStorageStateEq σ σ₁) (hcd : accountCodeStateEq σ σ₁)
    (hex : σ₁.get? I.codeOwner = some acc) :
    (storedMap σ₁ I H).getD I.codeOwner default |>.storage =
        (σ.getD I.codeOwner default).storage.insert (refundedSlot H) (setTrueWord (refundedWord σ I H)) ∧
    (∀ a ≠ I.codeOwner, storage of a in storedMap σ₁ I H = storage of a in σ) ∧
    accountCodeStateEq σ (storedMap σ₁ I H) ∧
    land 0xff (refundedWord (storedMap σ₁ I H) I H) = 1          -- refunded[H] now reads true

theorem refundETH_bridgeStorage (hrun : RefundRun σ σ₀ I σ') (hmint : MintFrame σ₀ I)
    (hcreate : CreateFrame σ₀ I) (hex : ∀ σ₁, accountStorageStateEq σ σ₁ → (σ₁.get? I.codeOwner).isSome) :
    (σ'.getD I.codeOwner default).storage =
      (σ.getD I.codeOwner default).storage.insert (refundedSlot H) (setTrueWord (refundedWord σ I H))
```

`storedMap_post` is the "only `refunded[H]` changes" statement for the store, and it is
unconditional apart from `hex`. `refundETH_bridgeStorage` carries it to the *final* map. For that it
needs the frame hypotheses listed below. The last conjunct of `storedMap_post` and
`setTrueWord_lowByte` show that a second `refundETH` for the same `H` from that state fails the
`AlreadyRefunded` check.

## Hypotheses, summaries, axioms (complete list)

1. `hcode`, `hsel`, `hcds`: the code, the selector, and a calldata bound that holds in any EVM.
2. **No call summaries** in the headline theorems. The three external operations appear in
   `RefundRun` as conclusions. Two things are **proved** from EVMLean, not assumed: static calls
   leave storage and code unchanged (`Theta_static_accountStorageStateEq`,
   `Theta_static_accountCodeStateEq`), and return data is below `2^138` bytes
   (`Theta_returnData_size_lt_2pow138_of_eq`).
3. **CREATE is extended, not summarized.** `RD.create` (`Create.lean`) is proved from EVMLean's
   `step_create` in the style of EquiVM's `RD.call`, with gas accounting. It assumes nothing and
   depends on the standard axioms only.
4. **Frame hypotheses, used only by `refundETH_bridgeStorage`** (`Post.lean`):
   * `MintFrame σ₀ I`: a successful `mint(amount)` call to ETHLiquidity leaves the bridge's storage
     unchanged. ETHLiquidity's code is not verified here.
   * `CreateFrame σ₀ I`: a successful SafeSend creation leaves the bridge's storage unchanged.
     SafeSend's constructor only `SELFDESTRUCT`s, but its execution inside `Lambda` is not proved
     symbolically.
   * `hex`: the executing account is present in every map with σ's storage. EVMLean's `SSTORE`
     (EquiVM's `sstoreAccountMap`) is a no-op on an absent account. That is a modelling quirk, and
     a deployed contract's account always exists.

   All three hold in the concrete success run (`Concrete.success_storage_exact`: afterwards the
   bridge's storage has exactly one entry, `refunded[H] = 1`).
5. **No collision or injectivity hypothesis about keccak is used.** `H` is defined as the keccak of
   the preimage, and the statements are about that word.
6. **Lean axioms** (`lake build BridgeEvm.Axioms`). `refundETH_trace`, `refundETH_outcome`,
   `refundETH_success`, `refundETH_no_other_error`, `createStep_success`, `storedMap_post`,
   `refundETH_bridgeStorage`, `RD.create` and `refundPreimage_size` all depend on exactly
   `[propext, Classical.choice, Quot.sound]`, and the build asserts it. **0 `native_decide`
   axioms.** Before the kernel switch there were 947. Instruction decodes, pc arithmetic, JUMPDEST
   membership, the JUMPDEST table, the SafeSend bytes and the selector bytes are all checked by the
   kernel (`decide +kernel`) through the proved `decode_eq_decodeList` and `D_J_eq_jumpdestScan`.
   The `Concrete.*` tests use `native_decide` (compiled evaluation of `Ξ`). They are tests, not
   dependencies.
7. **Trusted semantics.** The Lean kernel, EVMLean's definitions (`Ξ`, `Θ`, `Lambda`, gas, `KEC`)
   and EquiVM's `Reasoning` library (proved).

## Bounds and modelling notes

* **Nothing is bounded.** Account maps, storage, caller, value, calldata contents and length, gas,
  depth, `perm`, the call results, and the return-data size (and so the free-memory pointer) are
  all symbolic. The only loop on the path is solc's `bytes` copy loop. Its trip count is fixed by
  the constant 100-byte message (4 iterations, unrolled), so the unrolling loses no generality.
* **`block.chainid`.** EVMLean's `CHAINID` returns the constant `Ethereum.chainId` (= 1), not a
  field of the environment. The theorems therefore talk about that constant. The proof never
  inspects its value: it is carried as the opaque word `chainIdWord`.
* **Proxy.** On L2 the predeploy 0x4200…0024 is a `Proxy` that `DELEGATECALL`s this
  implementation. The theorems quantify over every `I` with `I.code = ethbridgeRuntime`, so they
  include the delegatecall frame (`address(this) = I.codeOwner` is the proxy). The proxy's own code
  is not verified.
* **Gas.** Out of gas is always a possible outcome. No gas bound is proved. The call gas and the
  `CREATE` gas are existential (63/64 rule inside).
* **Events and balances.** The `RefundETH` `LOG3` is executed and its memory and gas are accounted
  for, but its topics and data are not asserted. Balance changes appear only inside `Θ` and
  `Lambda`.
* **Beneficiary of the SafeSend.** The theorem pins the init code to
  `SafeSend.creationCode ‖ abi.encode(from)` with endowment `amount`. That the constructor sends
  the funds to `from` is the init code's behaviour. It is checked concretely
  (`Concrete.success_reachable`: `from`'s balance becomes `amount`), not symbolically.
* **Fork.** EVMLean implements Cancun. The bytecode is London-compiled and uses no opcode whose
  semantics changed (the `SELFDESTRUCT` in SafeSend runs in the same transaction as its creation).

## Non-vacuity and checks (`BridgeEvm/Concrete.lean`, executed with `Ξ`)

The setup is the bridge code at 0x4200…0024 with balance `amount`, a mock messenger returning
`H == Hgood`, and a mock ETHLiquidity `STOP`. The arguments are `(901, 7, 0xf0f0, 0x7070, 1e18)`.

| theorem | checks |
|---|---|
| `refundHash_matches_cast` | `refundHash` equals foundry's `cast keccak (cast abi-encode … (cast calldata relayETH …))` = `0x9fea071a…4d43`. This checks the statement's preimage against Solidity's `abi.encode`. |
| `success_reachable` | succeeds; `refunded[H] = 1`; `from` receives `amount` (SafeSend beneficiary). |
| `success_storage_exact` | after success the bridge's storage is exactly `{refunded[H] ↦ 1}` (frame hypotheses hold). |
| `dirty_high_bytes_success` | a slot `0x100` reads false (low byte) and becomes `0x101` (`setTrueWord`). |
| `wrong_preimage_reverts` | nonce 8 → different `H` → mock says not expired → revert. |
| `not_expired_reverts` | mock returns false for every `H` → revert (`MessageNotExpired`). |
| `already_refunded_reverts` | `refunded[H] = 1` → revert (`AlreadyRefunded`). |
| `no_liquidity_code_reverts` | ETHLiquidity has no code → revert (solc's `EXTCODESIZE` check). |
| `create_failure_reverts` | the bridge cannot fund `amount` → `CREATE` pushes 0 → revert. |
| `static_violation` | `perm = false` → `StaticModeViolation` at the `SSTORE`. |

Proof mutations, run once and then reverted:

* Changing the preimage's `0xc0` to `0xa0` in `refundPreimage` breaks `refundPreimage_eq`.
* Changing it consistently in both `refundPreimage` and `refundPreimageL` makes `seg_hash` fail:
  the code's hash is not the mutated one.

## Proof structure

The trace is split into segments. Each segment is a theorem whose conclusion is a disjunction of
terminals. Block names are `ethbridge_block_<pc>`, from EquiVM's generator.

| file | pcs | content |
|---|---|---|
| `TraceEntry.lean` | 0 → 1539 | dispatcher, non-payable, ABI length, address validation (`seg_entry`) |
| `TraceHash.lean` | 1539 → 1699 | `abi.encodeCall`, `abi.encode` with copy loop, `KECCAK256` = `refundHash I` (`seg_hash`) |
| `TraceExpired.lean` | 1699 → 1897 | `STATICCALL expiredMessages(H)`, return-data length, bool check, value 1 (`seg_expired`) |
| `TraceStore.lean` | 1897 → 2147 | `refunded[H]` check, `SSTORE` (static split), `EXTCODESIZE`, `CALL mint(amount)` (`seg_store`) |
| `TraceCreate.lean` | 2147 → STOP | `CODECOPY` SafeSend, `abi.encode(from)`, `CREATE`, zero check, `LOG3`, `STOP` (`seg_create`) |
| `Refund.lean` | | composition (`refundETH_trace`) and the `Ξ`-level theorems |

Supporting files:

* `Mem.lean`: memory as `ofL (L : List UInt8)`. It provides `write_wb`, `memLoad_ofL`,
  `keccakWord_ofL` and `read_ofL` for numeral offsets, normalized by the `msimp` tactic. It provides
  `pre`, `write_first`, `write_rel`, `write_rel_src`, `read_rel` and `memLoad_pre` for writes above a
  symbolic pointer. It also proves the selector merge (`wb_merge'`) and the rounding lemmas, all
  from EVMLean's definitions.
* `Words.lean`: word facts for the branch conditions.
* `Create.lean`: `RD.create`, `rd_after_create` and `CreateStep`.

## Build and timings

```sh
cd packages/contracts-bedrock/test/formal/expiry/evm-lean-bridge
lake exe cache get                  # Mathlib cache (or copy ../evm-lean/.lake/packages: same pins)
lake build BridgeEvm BridgeEvm.Axioms   # must print "Build completed successfully" and 9
                                        # "standard axioms only" lines
grep -rn "sorry\|admit" BridgeEvm/      # nothing
```

The machine was a shared 32-core Linux box under heavy load from other jobs, and the build ran
under a 12 GB memory cap. With dependencies built, a clean rebuild of this project took
**1 min 32 s** wall: 261 s CPU, 4.7 GB peak RSS. That covers the 9 kernel-checked block shards, all
proofs and the concrete runs. The slowest file was shard 005 (34 s). The proof files take 2–12 s
each.

## Not covered

* **Completeness.** No theorem says that `refundETH` succeeds when the conditions hold. The
  `RD` framework hides the call gas, so a failure clause like "one of the calls failed" would be
  satisfiable in every state. Only soundness and the outcome classification are claimed.
* **The callees.** The code of ETHLiquidity and of the messenger's `expiredMessages` getter is not
  verified. Neither is SafeSend's constructor, symbolically (see the frame hypotheses and the
  beneficiary note).
* **Revert causes.** The reverting cases are not characterized in the `Ξ`-level statement; the
  segment docstrings list them.
* The `RefundETH` event's topics and data.
* `sendETH` and `relayETH` (the latter shares the SafeSend pattern; the same lemmas apply).
* **The protocol-level link.** That `expiredMessages(H) = true` implies the message was never
  relayed and never can be is the job of the protocol model (`../lean`) and of `../evm-lean`. This
  development only proves that the bridge consults that flag for the right `H` and spends it at
  most once.

## Files

| file | content |
|---|---|
| `lakefile.toml`, `lean-toolchain`, `lake-manifest.json` | project, pinned |
| `bytecode/*.hex` | compiled runtime and SafeSend creation code |
| `scripts/regen.sh`, `scripts/gen_bytecode.py` | validated recompilation, generation of `Bytecode.lean`, `Blocks/`, `AllBlocks.lean` |
| `scripts/trace_paths.py` | concrete mini-EVM tracer listing calls, keccak inputs, `CREATE` and the block path of each branch |
| `BridgeEvm/Bytecode.lean`, `BridgeEvm/Blocks/`, `BridgeEvm/AllBlocks.lean` | generated: bytecode, kernel-checked `RD` summaries of every basic block |
| `BridgeEvm/KernelDecide.lean`, `BridgeEvm/KernelRun.lean` | kernel decision of bytecode facts (copied from `../evm-lean`) |
| `BridgeEvm/Spec.lean` | statement vocabulary (preimage, hash, calls, storage, SafeSend) |
| `BridgeEvm/Mem.lean`, `BridgeEvm/Words.lean`, `BridgeEvm/Create.lean` | libraries (see "Proof structure") |
| `BridgeEvm/Trace*.lean`, `BridgeEvm/Refund.lean`, `BridgeEvm/Post.lean` | proof |
| `BridgeEvm/Concrete.lean` | executable checks |
| `BridgeEvm/Axioms.lean` | `#assert_std_axioms` |

## Review log

* **Self-review, before hand-back.** These lessons from the review of `../evm-lean` were applied:
  * No completeness clause with an existential-gas failure disjunct is claimed.
  * The return-data condition is "at least 32 bytes and the first word", matching solc's decoder.
  * No per-key no-collision hypothesis is used.
  * `regen.sh` forces the default profile and validates against `semver-lock.json` before writing.
  * The frame hypotheses are shown to hold in the concrete success run.
* External reviews (Codex, Astra, Sol 6.1): *pending*.
