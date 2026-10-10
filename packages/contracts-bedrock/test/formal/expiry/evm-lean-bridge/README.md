# Bytecode-level soundness of `SuperchainETHBridge.refundETH` in Lean (EquiVM / EVMLean)

Status: **all claimed theorems proved.** `lake build BridgeEvm BridgeEvm.Axioms` passes. There is
no `sorry` or `admit`. Every headline theorem depends only on `propext`, `Classical.choice` and
`Quot.sound`: there are **0 `native_decide` axioms**, and `BridgeEvm/Axioms.lean` asserts this at
build time with `#assert_headline`, which also requires a non-vacuity partner per theorem. Soundness and the outcome classification are claimed;
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
* Right after the `SSTORE` (before the `mint` call and the `CREATE`), the bridge's storage is the
  pre-state's with exactly `refunded[H]` set to true; other accounts' storage and all code are
  unchanged (`refundETH_store`, given that the executing account has code). What the `mint` call
  and the creation do afterwards to any storage, including the bridge's, is not constrained.
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
| Source | `src/L2/SuperchainETHBridge.sol` at `010881014f` (semver 2.0.0; apart from the version string the source is unchanged since `c7c51d79e2`, the guideline pass that renamed the errors to `SuperchainETHBridge_MessageNotExpired` / `SuperchainETHBridge_AlreadyRefunded`). Relative to the earlier target `5992028e08`, the runtime differs only in the two revert-selector `PUSH32` operands (pcs 1852 and 1925). The proof rebuilt unchanged; only the concrete selector checks were updated. |
| Compiler | solc `0.8.15+commit.e14f2714` via forge 1.8.1, repository **default** profile |
| Settings | `{"evmVersion":"london","libraries":{},"metadata":{"bytecodeHash":"none"},"optimizer":{"enabled":true,"runs":999999}}` (solc 0.8.15 caps the profile's `cancun` at `london`) |
| Runtime | 3131 bytes, `keccak256 = 0x75ab659187cc99eac81e04e34dbbf918a7dc3a05a062e8da1e13493056a0ce4d` (`bytecode/SuperchainETHBridge.runtime.hex`) |
| Init code | `keccak256 = 0xf0789b4de18efc9ee1107d62420c3196543f152892ed0ac43aaf49136d6ffd20` = `initCodeHash` in `snapshots/semver-lock.json` |
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
    (hcode : (σ.getD I.codeOwner default).code.size ≠ 0) :
    (storedMap σ₁ I H).getD I.codeOwner default |>.storage =
        (σ.getD I.codeOwner default).storage.insert (refundedSlot H) (setTrueWord (refundedWord σ I H)) ∧
    (∀ a ≠ I.codeOwner, storage of a in storedMap σ₁ I H = storage of a in σ) ∧
    accountCodeStateEq σ (storedMap σ₁ I H) ∧
    land 0xff (refundedWord (storedMap σ₁ I H) I H) = 1          -- refunded[H] now reads true

theorem refundETH_store (hrun : RefundRun σ σ₀ I σ') (hcode : (σ.getD I.codeOwner default).code.size ≠ 0) :
    ∃ σ₂ σ₃,
      σ₂'s bridge storage = σ's with only refunded[H] := setTrueWord old ∧
      other accounts' storage in σ₂ = σ's ∧ accountCodeStateEq σ σ₂ ∧ refunded[H] reads true in σ₂ ∧
      (∃ oM, CallTo σ₀ I ethLiq (mintCalldata amount) σ₂ σ₃ true oM) ∧
      (∃ x rd', CreateStep I σ₀ σ₃ amount (safeSendDeploy from) x σ' rd' ∧ x ≠ 0)
```

`storedMap_post` is the "only `refunded[H]` changes" statement for the store. Its one extra
hypothesis is that the executing account has non-empty code in the pre-state (a deployed contract;
for the predeploy, the proxy). Presence of the account in the post-`STATICCALL` map is *derived*
from it (`present_of_code`; static calls preserve code, proved). Presence matters because EVMLean's
`SSTORE` (EquiVM's `sstoreAccountMap`) does nothing on an account absent from the map.
`refundETH_store` exposes the state chain of a successful run, `σ → σ₂ (store) → σ₃ (mint) → σ'
(CREATE)`. **The final storage is not constrained:** the `mint` callee and the SafeSend init code
run arbitrary code as far as this proof is concerned, so a claim about `σ'` would need verified (or
assumed) callee code. No frame assumption is made.

**At most once, within this proof's scope:** in `σ₂`, `refunded[H]` reads true
(`setTrueWord_lowByte`), so a later `refundETH` for the same `H` reverts with `SuperchainETHBridge_AlreadyRefunded`
*provided the slot still holds that value when it runs*. Whether it does depends on what the later
callees and transactions do to the bridge's storage and code. That is not proved here.

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
4. **No frame hypothesis.** The post-state lemmas assume only that the executing account has
   non-empty code in the pre-state (`storedMap_post`, `refundETH_store`). This holds in the
   concrete pre-state (`Concrete.bridge_has_code`). An earlier version stated final-storage
   preservation under universal frame hypotheses on the `mint` call and the creation. The reviews
   showed those hypotheses were false for general maps, so that theorem was removed (see "Review
   log").
5. **No collision or injectivity hypothesis about keccak is used.** `H` is defined as the keccak of
   the preimage, and the statements are about that word.
6. **Lean axioms** (`lake build BridgeEvm.Axioms`). `refundETH_trace`, `refundETH_outcome`,
   `refundETH_success`, `refundETH_no_other_error`, `createStep_success`, `storedMap_post`,
   `refundETH_store`, `RD.create` and `refundPreimage_size` all depend on exactly
   `[propext, Classical.choice, Quot.sound]`, and the build asserts it. **0 `native_decide`
   axioms.** Before the kernel switch there were 947. Instruction decodes, pc arithmetic, JUMPDEST
   membership, the JUMPDEST table, the SafeSend bytes and the selector bytes are all checked by the
   kernel (`decide +kernel`) through the proved `decode_eq_decodeList` and `D_J_eq_jumpdestScan`.
   The `Concrete.*` tests and the `native_*` lemmas of `NonVacuous.lean` use `native_decide`
   (compiled evaluation of `Ξ`). They are not dependencies of any headline theorem (checked).
7. **Trusted semantics.** The Lean kernel, EVMLean's definitions (`Ξ`, `Θ`, `Lambda`, gas, `KEC`)
   and EquiVM's `Reasoning` library (proved).

## Bounds and modelling notes

* **Nothing is bounded.** Account maps, storage, caller, value, calldata contents and length, gas,
  depth, `perm`, the call results, and the return-data size (and so the free-memory pointer) are
  all symbolic. The only loop on the path is solc's `bytes` copy loop. Its trip count is fixed by
  the constant 100-byte message (4 iterations, unrolled), so the unrolling loses no generality.
* **`block.chainid`.** EVMLean's `CHAINID` returns the constant `Ethereum.chainId` (= 1), not a
  field of the environment. The theorems are therefore stated for that constant, not for an
  arbitrary chain id. The proof never unfolds the constant (it appears as the word `chainIdWord`),
  but this is an observation about the proof script, not a theorem about other chain ids.
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
* **Known deviation of the pinned `Lambda` (contract creation).** EVMLean's `Lambda` (pinned
  `Semantics.lean`, the EIP-7610 check) makes a creation fail if the target address has a nonzero
  nonce, non-empty code, *or non-empty storage*. The Cancun execution specification lets a creation
  proceed over an address with empty code, zero nonce and non-empty storage. For `refundETH` this
  only matters if the SafeSend's freshly derived address already has storage. Our statements are
  phrased through `Lambda`'s own result (`CreateStep`, `createStep_success`), so they are exact for
  EVMLean. In that corner case they describe EVMLean's outcome (failure, hence a revert), not
  Cancun's.

## Non-vacuity (automated, `BridgeEvm/NonVacuous.lean`)

Every headline theorem has a partner `nonvacuous_<name>`. The partner exhibits concrete values that
satisfy all of the theorem's hypotheses jointly, applies the theorem to them, and states which case
of its conclusion is realized. For the theorems about success it uses the concrete *successful*
run. `Axioms.lean` runs `#assert_headline` on each theorem, and the build fails unless:
1. the theorem depends only on `propext`, `Classical.choice`, `Quot.sound`;
2. the partner exists and its proof term applies the theorem;
3. every non-standard axiom of the partner comes from a theorem named `native_*` (the traversal
   stops at those and lists them; each may carry only its own `native_decide` axiom).

Mutation tests, each run once on a copy and reverted, all fail the build: renaming a partner ("…
has no non-vacuity partner …"); proving `nonvacuous_refundPreimage_size` by `decide +kernel` instead
of applying the theorem ("… does not apply …"); renaming `native_run_static` to a non-`native_`
name ("… uses non-standard axioms outside `native_*` lemmas …").

| theorem | hypotheses (all jointly) | conclusion instantiated |
|---|---|---|
| `refundETH_trace` | `Concrete.env 7` on `Concrete.σ 0`: `w_code` (`rfl`), `w_sel`, `w_cds` (`decide +kernel`) | the success disjunct (`RefundRun` and `RDret`): the other two would make the concrete run end in out of gas, a revert or a static violation, contradicting the successful run |
| `refundETH_outcome` | the same | the success disjunct with `RefundRun` |
| `refundETH_success` | the same + the successful run (`hres`) | `RefundRun`, empty output; consistent with the kernel-checked pre-state `refundHash (env 7) = Hgood`, `refunded[Hgood] = 0` |
| `refundETH_no_other_error` | the same on the statically entered run (`perm := false`) + `he = StaticModeViolation` | `perm = false` (the second disjunct) |
| `createStep_success` | `CreateStep … x σ' rd'` and `x ≠ 0` from the concrete run; endowment `amount` and beneficiary `0xf0f0` (kernel) | the nonce bound, `x` = the created address, empty return data |
| `storedMap_post` | `σ₁`, `hst`, `hcd` from the concrete run; `hcode` (`bridge_has_code`, kernel) | at `H = Hgood`, with the kernel-checked empty pre-state storage: the bridge's storage after the store is exactly `{refunded[Hgood] ↦ 1}`, and `refunded[Hgood]` reads true |
| `refundETH_store` | the concrete `RefundRun` and `bridge_has_code` | `refunded[Hgood]` true after the store; the `mint(amount)` call; the SafeSend creation with `x ≠ 0` |
| `RD.create` | a one-byte program `CREATE` with stack `[0,0,0]`; the `RD` cursor by `RD.start`, the decode by `evm_kdecide`, `perm`, the size bound (fully kernel-checked) | the `CreateStep` relation and the `RD` cursor after it |
| `refundPreimage_size` | (no hypotheses) instantiated at `env 7` | size 352 |

`RD.create` is a general opcode lemma, so its partner uses a synthetic program; the bridge's own
`CREATE` is exercised through `createStep_success` / `refundETH_store` on the concrete run.

**Trust.** Everything is kernel-checked except two facts that need `Ξ` evaluated on the concrete
bytecode: `native_run_success` (the concrete run succeeds) and `native_run_static` (the static run
raises `StaticModeViolation`). The kernel cannot evaluate `Ξ` (well-founded recursion). They are used
only by the partners and are outside every headline theorem's axiom cone (checked). The concrete
hash `Concrete.refundHash_matches_cast` (keccak256 of the 352-byte preimage, ≈ 26 s) and the slot
facts (`w_unrefunded`, `w_storage_empty`) are kernel-checked.

**Hypotheses quantified over all states or callees.** No headline theorem has one any more.
`StaticCall`, `CallTo` and `CreateStep` are existential facts about the actual run. The former
universal `MintFrame`/`CreateFrame` (round 1: false in practice) are removed. Every remaining
hypothesis is discharged by the witnesses above.

## Non-vacuity and checks (`BridgeEvm/Concrete.lean`, executed with `Ξ`)

The setup is the bridge code at 0x4200…0024 with balance `amount`, a mock messenger returning
`H == Hgood`, and a mock ETHLiquidity `STOP`. The arguments are `(901, 7, 0xf0f0, 0x7070, 1e18)`.

| theorem | checks |
|---|---|
| `refundHash_matches_cast` (kernel-checked) | `refundHash` equals foundry's `cast keccak (cast abi-encode … (cast calldata relayETH …))` = `0x9fea071a…4d43`. This checks the statement's preimage against Solidity's `abi.encode`. |
| `success_reachable` | succeeds; `refunded[H] = 1`; `from` receives `amount` (SafeSend beneficiary). |
| `success_storage_exact` | after success the bridge's storage is exactly `{refunded[H] ↦ 1}`. This is an observation about this run, not a frame theorem. |
| `bridge_has_code` | the hypothesis of `storedMap_post`/`refundETH_store` holds in the pre-state. |
| `dirty_high_bytes_success` | a slot `0x100` reads false (low byte) and becomes `0x101` (`setTrueWord`). |
| `wrong_preimage_reverts` | nonce 8 → different `H` → mock says not expired → revert with selector `SuperchainETHBridge_MessageNotExpired()` = `0x0978275c`. |
| `not_expired_reverts` | mock returns false for every `H` → revert with selector `0x0978275c`. |
| `already_refunded_reverts` | `refunded[H] = 1` → revert with selector `SuperchainETHBridge_AlreadyRefunded()` = `0x2b792286`. |
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
lake build BridgeEvm BridgeEvm.Axioms   # must print "Build completed successfully" and one
                                        # "standard axioms only. Partner … applies it" line per theorem
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
  verified. Neither is SafeSend's constructor, symbolically (see the beneficiary note).
* **The final state.** The bridge's storage (and everything else) after the `mint` call and the
  SafeSend creation is whatever those executions produce. Only the state right after the `SSTORE`
  is described (`refundETH_store`).
* **Revert causes.** The reverting cases are not characterized in the `Ξ`-level statement; the
  segment docstrings list them.
* The `RefundETH` event's topics and data.
* `sendETH` and `relayETH` (the latter shares the SafeSend pattern; the same lemmas apply).
* **The protocol-level link.** That `expiredMessages(H) = true` implies the message was never
  relayed and never can be is the job of the protocol model (`../lean`) and of `../evm-lean`. This
  development proves that the bridge consults that flag for the right `H` and sets `refunded[H]`
  before calling out. "Spent at most once" across later calls additionally needs the bridge's
  storage and code to be preserved by the callees and later transactions; that is not proved here.
* **The proxy in the concrete runs.** The concrete runs call the implementation directly and do not
  exercise the `Proxy`'s `DELEGATECALL`; the theorems cover that frame.

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
| `BridgeEvm/NonVacuous.lean` | one `nonvacuous_` partner per headline theorem |
| `BridgeEvm/Axioms.lean` | `#assert_headline`: standard axioms + partner applies the theorem + compiled evaluation only in `native_*` lemmas |

## Review log

* **Self-review, before hand-back.** These lessons from the review of `../evm-lean` were applied:
  * No completeness clause with an existential-gas failure disjunct is claimed.
  * The return-data condition is "at least 32 bytes and the first word", matching solc's decoder.
  * No per-key no-collision hypothesis is used.
  * `regen.sh` forces the default profile and validates against `semver-lock.json` before writing.
* **Round 1: R1 (fresh-context reviewer), R2 and R3 (independent model-based reviewers).** All
  three found the core soundness chain faithful and kernel-checked (`refundETH_trace`, `_outcome`,
  `_success`, `_no_other_error`), and the artifacts reproducible. R1's bytecode mutations each
  broke the proof: mask `0xff→0xfe`, the mint selector, and SafeSend's `SELFDESTRUCT`. Findings and
  resolutions:
  1. *High (all).* `MintFrame`/`CreateFrame` quantified over every account map with unconstrained
     callee and bridge code, so they were false in practice. R1 proved their negation for the
     concrete environment, which made `refundETH_bridgeStorage` vacuous. **Resolved:** removed the
     frame hypotheses and the theorem. The final storage is now explicitly not constrained, and
     `refundETH_store` states the store and the call chain instead.
  2. *High (all).* The account-presence hypothesis "present in every storage-equivalent map" was
     unsatisfiable (e.g. erase the account from an empty-storage map). **Resolved:** the hypothesis
     is now non-empty code in the pre-state. Presence in the actual post-`STATICCALL` map is derived
     from it (`present_of_code`, using the proved code preservation of static calls). It is checked
     concretely (`bridge_has_code`).
  3. *Medium (all).* The README overstated final-storage and "at most once" guarantees.
     **Resolved:** the claims are now restricted to the state right after the `SSTORE`; see
     "Post-state lemmas" and "Not covered".
  4. *Medium (R3).* The pinned `Lambda` rejects creation over an address with non-empty storage,
     unlike the Cancun specification. **Resolved:** documented under "Bounds and modelling notes".
  5. *Low.* Fixes: `regen.sh` now validates the staged artifacts (including the SafeSend offset)
     before replacing tracked files. The chain-id wording is corrected. The concrete revert tests
     assert the error selector. The concrete runs' not exercising the proxy is noted.
* **Round 2: automated non-vacuity.** Added `NonVacuous.lean`, with one partner per headline
  theorem that satisfies every hypothesis jointly and derives the conclusion. Added the
  `#assert_headline` build check, which fails on a missing partner (mutation-tested). Scanned for
  hypotheses quantified over all states or callees and found none left.
* **Retarget to `c7c51d79e2`** (errors renamed with the `SuperchainETHBridge_` prefix).
  `regen.sh` validated the new artifact against `semver-lock.json` (`0xa9040c1c…`). Only the two
  selector operands changed. The proof and the non-vacuity partners rebuilt unchanged; the
  concrete selector checks were updated.
* **Retarget to `010881014f`** (semver 2.0.0). Only the version string's `PUSH32` operand changed;
  `regen.sh` validated the artifact against `semver-lock.json`, and the proof rebuilt unchanged.
* **Round 3: stronger non-vacuity check.** `#assert_headline` now also requires that each partner's
  proof term applies its theorem and that compiled evaluation occurs only inside `native_*`
  lemmas (`run_success_native`/`run_static_native` renamed `native_run_success`/
  `native_run_static`); three mutations confirm each check fails the build. Partners strengthened:
  `refundETH_trace` now shows the success disjunct is realized; `refundETH_success`,
  `_no_other_error`, `createStep_success` and `storedMap_post` state their hypotheses jointly with
  the instantiated conclusion; `storedMap_post` and `refundETH_store` are instantiated at
  `H = Hgood` with kernel-checked pre-state facts; `refundHash_matches_cast` moved from
  `native_decide` to the kernel. No hypothesis found unsatisfiable. Incremental build 21–47 s.
