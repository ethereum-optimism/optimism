# Bytecode-level soundness of `UndeliveredMessageExporter.exportUndeliveredMessage` in Lean (EquiVM / EVMLean)

Status: **all claimed theorems proved.** `lake build ExporterEvm ExporterEvm.Axioms` passes. There
is no `sorry` or `admit`. Every headline theorem depends only on `propext`, `Classical.choice` and
`Quot.sound` (0 `native_decide` axioms), and `ExporterEvm/Axioms.lean` asserts this at build time
with `#assert_headline`, which also fails the build when a headline theorem has no `nonvacuous_`
partner or the partner's proof does not use the theorem. Soundness and the outcome classification are claimed; completeness is not (see "Not
covered").

This is a sibling of `../evm-lean-bridge/` (the `refundETH` proof) and `../evm-lean-l1cdm/` (the
L1 side of the same flow). It uses the same toolchain pins and machinery, plus three extensions
this function needs:

* `_message` has a **symbolic length**. solc's `copy_memory_to_memory` loop that ABI-encodes it is
  proved by induction over the iteration count (`Loop.lean`, `copy_loop`), not unrolled, and all
  memory above `0x80` is handled relative to symbolic free-memory pointers (`HashMem.lean`,
  `CallMem.lean`).
* **Revert outputs are recorded.** EquiVM's `RDrev` only says "reverts"; `RDrevP` (`Terminal.lean`)
  keeps the output, so the theorem lists every possible revert output.
* **The event is tracked.** `RDL` (`LogTrack.lean`) is EquiVM's `RD` invariant plus "the state's
  log series ends with entry `e`", from the `LOG3` to the `RETURN`.

## What is proved, in one paragraph

Take the **deployed runtime bytecode** of `UndeliveredMessageExporter` and run it with EVMLean's
code-execution function `Ξ` on calldata that selects
`exportUndeliveredMessage(address,uint256,uint256,address,address,bytes,uint32)` and is shorter
than `2^63` bytes. The theorems hold for every account map, caller, value, depth, gas, permission,
block header and code owner (so they cover the predeploy proxy's `DELEGATECALL` frame). If the run
**succeeds**, all of the following hold, with `H = keccak256(abi.encode(block.chainid, _source,
_nonce, _sender, _target, _message))` (EVMLean fixes `block.chainid` to 1; other chain ids are
outside the model, see "Bounds and modelling notes"):

* The ABI conditions hold (no ETH, well-formed calldata, clean addresses and `uint32`).
* A `STATICCALL` went to `L2ToL2CrossDomainMessenger` (0x4200…0023) with calldata exactly
  `successfulMessages(H)` (`0xb1b1b209 ‖ H`). It succeeded, returned at least 32 bytes, and the
  first word was `0` (false). It changed no storage and no code.
* `L2CrossDomainMessenger` (0x4200…0007) had code, and a `CALL` with value 0 went to it with
  calldata exactly `sendMessage(_sourceMessenger, relayUndeliveredMessage(H, block.timestamp),
  _minGasLimit)` (228 bytes, below). It succeeded, and **its resulting account map is the final
  account map**.
* The output is exactly `abi.encode(H)` (the 32-byte word `H`).
* The last entry of the final log series is `UndeliveredMessageExported(H, _source,
  _sourceMessenger, block.timestamp)` emitted by the executing account.

If the run **reverts**, the output is one of: empty; the bubbled return data of a failed
`successfulMessages(H)` call; the 4 bytes `0xccc3f3b0` (`UndeliveredMessageExporter_MessageRelayed()`),
and then `successfulMessages(H)` returned true; or the bubbled return data of a failed `sendMessage`
call, after `successfulMessages(H)` returned false. This is a necessary condition on the output,
not a unique cause: the alternatives overlap (a callee can itself revert with empty data or with
`0xccc3f3b0`, which the exporter bubbles). The only other outcomes are out of gas and, only in a
static frame (`I.perm = false`, e.g. entered by `STATICCALL`, also through the proxy's
`DELEGATECALL`), a static-mode violation.

No external call is assumed to succeed or to return any particular value. Each call is a
*conclusion*: the theorem names its exact input and the output that made the code continue. The
call gas and the input substate of each call are existential (see `StaticCall`/`CallTo` below):
the statement says that `Θ` on exactly that input, from that account map, under *some* gas and
substate, succeeds with that output and account map, and that the run continued with exactly that
account map. The proof derives these from the run's own call step, but the statement does not
expose the link to the run's actual gas and substate.

## Toolchain (all pinned; same as `../evm-lean`, `../evm-lean-bridge`)

| Component | Version |
|---|---|
| Lean | `leanprover/lean4:v4.29.0` (`lean-toolchain`) |
| [EquiVM](https://github.com/argotorg/EquiVM) (MIT) | `b0e9d55a277bc081411fd11f1b69f2e11b38e478` |
| [EVMLean](https://github.com/lefterislazar/EVMLean) | `63f61339dd17a809961e34028cbc8c5fcb02af13` (Cancun semantics) |
| Mathlib | `v4.29.0` (`8a178386…`), via EquiVM |

The exact transitive pins are in `lake-manifest.json` (a copy of the bridge's, package renamed).
`KernelDecide.lean`, `KernelRun.lean`, `Mem.lean` and `Words.lean` are copied from
`../evm-lean-bridge` with the namespace renamed.

## The artifact proved against

| | |
|---|---|
| Source | `src/L2/UndeliveredMessageExporter.sol` at `c7c51d79e2` (errors prefixed `UndeliveredMessageExporter_`, the `UndeliveredMessageExported` event, predeploy `0x4200…0030`). The file is unchanged at the formal branch's merge `a449f55d9b`. |
| Compiler | solc `0.8.15+commit.e14f2714` via forge 1.8.1, repository **default** profile |
| Settings | `{"evmVersion":"london","libraries":{},"metadata":{"bytecodeHash":"none"},"optimizer":{"enabled":true,"runs":999999}}` (solc 0.8.15 caps the profile's `cancun` at `london`) |
| Runtime | 1468 bytes, `keccak256 = 0x7f211b443a2038a83879ba5692e509cdbf9382046f101cdd07a95058ae4115e7` (`bytecode/UndeliveredMessageExporter.runtime.hex`); no immutables, so this is the deployed code |
| Init code | `keccak256 = 0xc4e2d1f2343ded5ed74a38f67c79bbf896817e39adeb1605f8677537319e4fd2` = `initCodeHash` of `src/L2/UndeliveredMessageExporter.sol:UndeliveredMessageExporter` in `snapshots/semver-lock.json` |
| Lean | `ExporterEvm/Bytecode.lean` (`exporterRuntime`, one flat literal, generated). Its JUMPDEST table is checked by the kernel. |

To reproduce, run `scripts/regen.sh`. It forces the default profile and unsets every `FOUNDRY_*`
override, then builds into a temporary directory. It refuses to replace anything unless the solc
version and the full settings JSON match, the init-code hash equals `semver-lock.json`, the
artifact has no immutables, and eight operands sit at the pcs the proof uses (the two selectors,
both predeploy addresses, the error selector, the `sendMessage` and `relayUndeliveredMessage`
selectors, the event topic). It then regenerates `Bytecode.lean`, the block summaries `Blocks/` (with
`native_decide` rewritten to the kernel tactic `evm_kdecide`) and `AllBlocks.lean`. The proofs
refer to concrete pcs, so a layout change breaks them loudly.

## Statements

The main file is `ExporterEvm/Export.lean`; the vocabulary is in `ExporterEvm/Spec.lean`. Every
theorem takes these hypotheses and is universally quantified over `σ σ₀ A g I` (and the result
variables):

```lean
(hcode : I.code = exporterRuntime)                -- the pinned artifact runs (any code owner)
(hsel  : selectorWord I = exportSelector)         -- CALLDATALOAD(0) >> 224 = 0x186e7328
(hcds  : I.calldata.size < 2 ^ 63)                -- see "Hypotheses" item 1
```

```lean
theorem export_outcome … :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    (∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o) ∧ ExportRevert σ σ₀ I o) ∨
    (Ξ σ σ₀ g A I = .error .StaticModeViolation ∧ I.perm = false) ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') (UInt256.toByteArray (exportHash I))) ∧
      ExportRun σ σ₀ I σ' ∧ ∃ L : LogSeries, A'.logSeries = L.push (exportedLog I (exportHash I)))

theorem export_success … (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    ExportRun σ σ₀ I σ' ∧ o = UInt256.toByteArray (exportHash I) ∧
      ∃ L : LogSeries, A'.logSeries = L.push (exportedLog I (exportHash I))

theorem export_revert … (hres : Ξ σ σ₀ g A I = .ok (.revert g' o)) : ExportRevert σ σ₀ I o

theorem export_no_other_error … (he : Ξ σ σ₀ g A I = .error e) :
    e = .OutOfGass ∨ (e = .StaticModeViolation ∧ I.perm = false)

theorem exportPreimage_size (I : ExecutionEnv) (hargs : ArgsOk I) :
    (exportPreimage I).size = 224 + roundUp32 (msgLen I)
```

`export_trace` is the same statement at the level of EquiVM's `RD` invariant (`RDrevP ∨ RDstatic ∨
RDretL`).

### `ExportRun σ σ₀ I σ'` (with `H = exportHash I`)

```lean
ArgsOk I ∧ I.perm = true ∧
∃ σ₁ oS, StaticCall σ₀ I l2l2 (successfulCalldata H) σ σ₁ true oS ∧   -- successfulMessages(H) succeeded
  32 ≤ oS.size ∧ returnWord oS = UInt256.ofNat 0 ∧                     -- … and returned false
  accountStorageStateEq σ σ₁ ∧ accountCodeStateEq σ σ₁ ∧               -- (proved for static calls)
  extCodeSizeWord σ₁ l2cdmWord ≠ ⟨0⟩ ∧                                 -- 0x4200…0007 has code
  ∃ oC, CallTo σ₀ I l2cdm
    (sendMessageCd (argSrcMessenger I) H (tsWord I) (argMinGas I)) σ₁ σ' true oC  -- the call; σ' is its result
```

* `StaticCall` and `CallTo` are EVMLean's message-call function `Θ` with the real arguments of the
  call site: sender = the executing account, value 0, the given calldata, depth + 1, `w = false`
  for the static call and `w = I.perm` for the call. The call gas and the input substate are
  existentially quantified, because the `RD` framework does not expose them (EVMLean forwards
  all but 1/64 of the remaining gas, as `GAS` is the gas argument at both call sites).
* `l2l2 = 0x4200…0023` and `l2cdm = 0x4200…0007` are the `PUSH20` operands at pcs 289 and 456
  (`regen.sh` checks them). The source calls `ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER)
  .sendMessage(...)`, i.e. the L2CrossDomainMessenger, not the L2ToL2 one.
* `successfulCalldata H = 0xb1b1b209 ‖ H` (36 bytes).
* `returnWord o` is the first 32 bytes of the return data as a word. solc accepts any return data
  of at least 32 bytes whose first word is a clean bool; the theorem says exactly that.
* `tsWord I = UInt256.ofNat I.header.timestamp` (`block.timestamp`).
* `argSrcMessenger/argSource/argNonce/argSender/argTarget/argMsgOffset/argMinGas I` are
  `CALLDATALOAD(4/36/68/100/132/164/196)`; `msgPos I = 4 + offset`, `argMsgLen I =
  CALLDATALOAD(msgPos I)`, `msgLen I` its value, and `argMessage I` the `msgLen I` calldata bytes
  after the length word.
* **Nothing after the call changes the account map.** The final `σ'` *is* the `CALL`'s result: the
  `LOG3` and the `RETURN` touch only the substate and memory.

`sendMessageCd target H t gasLimit` (228 bytes; `w(x)` = 32-byte big-endian word):

```
0x3dbb202b ‖ w(target) ‖ w(0x60) ‖ w(gasLimit) ‖ w(0x44)        sendMessage(address,bytes,uint32) head
‖ 0x372293c3 ‖ w(H) ‖ w(t)                                       relayUndeliveredMessage(H, t), 68 bytes
‖ 0^28                                                           padding
```

### `ArgsOk I` (the ABI decoder's checks, each proved to revert with empty output when it fails)

`I.weiValue = 0`; `228 ≤ calldatasize`; `_sourceMessenger`, `_sender`, `_target` below `2^160`;
`offset ≤ 2^64 - 1`; `msgPos + 31 < calldatasize`; `msgLen ≤ 2^64 - 1`;
`msgPos + 32 + msgLen ≤ calldatasize`; `_minGasLimit < 2^32`. Under `hcds` solc's signed
comparisons coincide with these. Non-canonical encodings (any offset) are covered.

### The exact preimage (`exportPreimage I`, `224 + roundUp32(len)` bytes)

| bytes | content |
|---|---|
| 0..32 | `block.chainid` (EVMLean: the constant `Ethereum.chainId`; see notes) |
| 32..64 | `_source` |
| 64..96 | `_nonce` |
| 96..128 | `_sender` |
| 128..160 | `_target` |
| 160..192 | `0xc0` (offset of the `bytes` tail) |
| 192..224 | `len = _message.length` |
| 224..224+len | `_message` |
| ..224+roundUp32(len) | zero padding |

This is `abi.encode(block.chainid, _source, _nonce, _sender, _target, _message)`, i.e. the input of
`Hashing.hashL2toL2CrossDomainMessage` with the destination set to this chain.
`exportHash I = keccak256(exportPreimage I)` (EVMLean's `KEC`). The proof shows that the word the
code leaves after its `KECCAK256` (pc 859) is exactly `exportHash I`, through the copy of
`_message` into memory (`CALLDATACOPY` of symbolic length in the block at pc 170), the six head words, the
`copy_memory_to_memory` loop (⌈len/32⌉ iterations, by induction), its zero-word tail, and the
length word (`TraceHash.lean`, `seg_hash`). On the witness this definition equals foundry's
`cast keccak $(cast abi-encode "f(uint256,uint256,uint256,address,address,bytes)" …)`
(`Concrete.exportHash_matches_cast`, kernel-checked). `L2ToL2CrossDomainMessenger.relayMessage`
keys `successfulMessages` with the same function and requires `destination == block.chainid`
(source-level; the messenger's bytecode is not verified here).

### `ExportRevert σ σ₀ I o` (all possible revert outputs)

```lean
o = ByteArray.empty ∨
(ArgsOk I ∧ ∃ σ₁ rd, StaticCall σ₀ I l2l2 (successfulCalldata H) σ σ₁ false rd ∧ o = bubble rd) ∨
(ArgsOk I ∧ o = messageRelayedError ∧ ∃ σ₁ oS,
    StaticCall σ₀ I l2l2 (successfulCalldata H) σ σ₁ true oS ∧ 32 ≤ oS.size ∧ returnWord oS = 1) ∨
(ArgsOk I ∧ ∃ σ₁ oS, StaticCall σ₀ I l2l2 (successfulCalldata H) σ σ₁ true oS ∧
    32 ≤ oS.size ∧ returnWord oS = 0 ∧
    accountStorageStateEq σ σ₁ ∧ accountCodeStateEq σ σ₁ ∧ extCodeSizeWord σ₁ l2cdmWord ≠ ⟨0⟩ ∧
    ∃ σ₂ rd, CallTo σ₀ I l2cdm (sendMessageCd (argSrcMessenger I) H (tsWord I) (argMinGas I)) σ₁ σ₂ false rd ∧
      o = bubble rd)
```

* Empty output (`revert(0, 0)`) comes from: ETH attached, a failed ABI check, call depth 1024 (the
  call pushes 0 with empty return data), return data shorter than 32 bytes or not a clean bool, or
  no code at 0x4200…0007. The theorem does not say which, and the empty disjunct is
  unconditional, so it also covers a callee that fails with empty return data.
* The alternatives overlap: a callee can revert with exactly `0xccc3f3b0`, which the exporter
  bubbles, so those bytes alone do not establish that `successfulMessages(H)` returned true.
* `messageRelayedError = 0xccc3f3b0` (`cast sig "UndeliveredMessageExporter_MessageRelayed()"`),
  exactly 4 bytes. It is the only custom error in the contract.
* `bubble rd = if rd.size < 2^64 then rd else ByteArray.empty` is what solc's
  `RETURNDATACOPY(0, 0, n); REVERT(0, n)` produces in EVMLean (its memory read panics, i.e. returns
  the empty array, for a length ≥ 2^64; no real gas limit reaches that).

### The event (`exportedLog I H`)

```lean
⟨I.codeOwner, #[exportedTopic, H, argSource I],
  UInt256.toByteArray (argSrcMessenger I) ++ UInt256.toByteArray (tsWord I)⟩
```

`exportedTopic = 0x1f8c424a…fc6d = keccak256("UndeliveredMessageExported(bytes32,uint256,address,uint256)")`;
topics `[topic0, messageHash, source]` (the two `indexed` parameters), data
`abi.encode(sourceMessenger, undeliveredAt)` with `undeliveredAt = block.timestamp`. The success
conclusion says the final substate's log series is `L.push (exportedLog I H)` for some `L`: the
event is the **last** log entry. Earlier entries (`L`, e.g. the messenger's own `SentMessage` events)
are not described.

## Hypotheses, summaries, axioms (complete list)

1. `hcode`, `hsel`, and **`hcds : calldatasize < 2^63`**. The last is a modelling bound, not an EVM
   fact. It is used three ways: (a) EVMLean's `ByteArray.readWithPadding` panics (returns the
   empty array) for a length ≥ 2^64, so a `KECCAK256` over ≥ 2^64 bytes would hash the empty
   string in EVMLean; the encoding hashed here has `224 + roundUp32(len)` bytes, and `hcds` keeps
   it below 2^64; (b) it makes solc's signed `SLT` checks in the decoder coincide with the
   unsigned `ArgsOk` conditions; (c) it keeps the free-memory pointers below the bounds the memory
   lemmas use (`endPtr < 2^139`, then `< 2^140` after the return data, `Theta`'s return data
   being `< 2^138`). Real calldata is bounded by the block gas limit (a few MB). The bridge proof
   needed only `< 2^256` because its preimage has a fixed 352 bytes.
2. **No call summaries.** Both external calls appear in `ExportRun` / `ExportRevert` as
   conclusions. Proved from EVMLean, not assumed: static calls leave storage and code unchanged
   (`Theta_static_accountStorageStateEq`, `Theta_static_accountCodeStateEq`), and return data is
   below `2^138` bytes (`Theta_returnData_size_lt_2pow138_of_eq`, used to bound the free-memory
   pointer after the static call).
3. **No frame hypothesis, no keccak collision hypothesis.** `H` is defined as the keccak of the
   preimage; the statements are about that word.
4. **Lean axioms** (`lake build ExporterEvm.Axioms`): `export_trace`, `export_outcome`,
   `export_success`, `export_revert`, `export_no_other_error`, `exportPreimage_size`,
   `Concrete.exportHash_matches_cast` and `Concrete.sendMessageCd_matches_cast` depend on exactly
   `[propext, Classical.choice, Quot.sound]`, and the build asserts it. **0 `native_decide`
   axioms** in them. Instruction decodes, pc arithmetic, JUMPDEST membership, the JUMPDEST table
   and the selector/topic literals are checked by the kernel (`decide +kernel`) through the proved
   `decode_eq_decodeList` and `D_J_eq_jumpdestScan`.
5. **Trusted semantics.** The Lean 4.29 kernel, EVMLean's definitions (`Ξ`, `Θ`, gas, `KEC`) and
   EquiVM's `Reasoning` library (proved). The `RDL`/`RDrevP` combinators in this directory are
   proved from EVMLean's step function (copies of EquiVM's `RD` proofs with one more conjunct).

## Bounds and modelling notes

* **Nothing else is bounded.** Account maps, storage, caller, value, calldata contents and length
  (below `2^63`), the message offset and length, gas, depth, `perm`, the timestamp, the call
  results and the return-data size are all symbolic. The `_message` copy loop runs ⌈len/32⌉ times
  for a symbolic `len` and is proved by induction; the `relayUndeliveredMessage` copy loop
  (68 bytes, 3 iterations) uses the same lemma.
* **`block.chainid`.** EVMLean's `CHAINID` returns the constant `Ethereum.chainId` (= 1), not a
  field of the environment, so the preimage's first word is that constant (`chainIdWord`). The
  proof never uses its value, but this is an observation about the proof script, not a theorem
  about other chain ids.
* **Proxy.** The predeploy 0x4200…0030 is a `Proxy` that `DELEGATECALL`s this implementation. The
  theorems quantify over every `I` with `I.code = exporterRuntime`, so they include the
  delegatecall frame (`address(this) = I.codeOwner` is the proxy, which is also the event's
  address). The proxy's own code is not verified.
* **Gas.** Out of gas is always a possible outcome. No gas bound is proved; the call gas is
  existential.
* **Static frames.** "Static" means `I.perm = false` (entered by `STATICCALL`, or inheriting it,
  e.g. through the proxy's `DELEGATECALL`). A `CALL` with value 0 is allowed in a static frame; the callee then runs
  static. The code reaches its `LOG3` only after that call succeeds, and the `LOG3` raises
  `StaticModeViolation` (proved, `seg_log`).
* **Fork.** EVMLean implements Cancun. The bytecode is London-compiled and uses no opcode whose
  semantics changed.

## Non-vacuity (automated, `ExporterEvm/NonVacuous.lean`)

Every headline theorem has a partner `nonvacuous_<name>` that exhibits concrete values satisfying
all of its hypotheses jointly, instantiates the theorem on them, and derives its conclusion.
`Axioms.lean` runs `#assert_headline` on each, which asserts the standard-axiom footprint **and**
fails the build if the partner is missing or its proof term does not use the headline theorem.
That the instantiation discharges every hypothesis with a concrete witness is a property of the
partner's statement (listed below), not something the guard can check.

| theorem | witness |
|---|---|
| `export_trace` | `Concrete.env0` on `Concrete.σ0`; `w_code` by `rfl`, `w_sel`, `w_cds` by `decide +kernel`; the realized disjunct is the success one (the others contradict the successful run) |
| `export_outcome` | the same; the success disjunct is derived |
| `export_success` | the successful run (`Ξ … = .ok (.success …)`) gives `ExportRun`, the output `abi.encode(Hcast)` and the event, with `exportHash env0 = Hcast` (kernel) |
| `export_revert` | the "relayed" world `σR` (mock returns true for `Hcast`): the run reverts, and the theorem classifies the output |
| `export_no_other_error` | the statically entered run `envS` on `σS` (0x4200…0007 code `STOP`) errors with `StaticModeViolation`; the theorem gives `perm = false` |
| `exportPreimage_size` | `ArgsOk env0` (each field by `decide +kernel`); the preimage has 288 bytes |

**Trust.** The partners are kernel-checked except three facts that need `Ξ` evaluated on the
concrete bytecode: `native_run_success`, `native_run_relayed`, `native_run_static`. These are
named `native_decide` lemmas used only by the partners; `#print axioms` shows each success/revert/
static partner depends on exactly one `…native_decide.ax_1_1`, and the `#assert_std_axioms` checks
show no headline theorem does.

**Mutation checks** (each applied on a copy, `lake build ExporterEvm.Axioms` run, then reverted;
all eleven fail the build):

| mutation | first error |
|---|---|
| rename `nonvacuous_export_revert` | `Axioms.lean`: `export_revert has no non-vacuity partner` |
| replace `nonvacuous_export_revert` by `True := trivial` | `Axioms.lean`: `nonvacuous_export_revert does not use export_revert` |
| `successfulSelector` last byte `0x09 → 0x0a` | `TraceStatic.lean` (`successfulCalldata_eq`) |
| preimage offset word `0xc0 → 0xa0` | `HashMem.lean` (`exportPreimage_eq`) |
| `relaySelector` last byte | `TraceCall.lean` (`sendMessageCd_eq`) |
| `sendMessageSelector` last byte | `TraceCall.lean` (`sendMessageCd_eq`) |
| `exportedTopic` last digit | `TraceLog.lean` (the `PUSH32` decode) |
| `messageRelayedError` last byte | `TraceStatic.lean` (`relayed_out`) |
| swap `gasLimit` and `0x44` in `sendMessageCd` | `TraceCall.lean` |
| `l2cdmWord := 0x4200…0016` | `TraceCall.lean` |
| bytecode: the `PUSH4 0x3dbb202b` operand → `0x3dbb202c` | `Blocks/RuntimeBlocks_002.lean` (decode) |

## Concrete runs (`ExporterEvm/Concrete.lean`, executed with `Ξ`)

The setup is the exporter code at 0x4200…0030, a mock L2ToL2CrossDomainMessenger returning
`H == Hrel`, and a mock L2CrossDomainMessenger that stores `keccak256(calldata)` in its slot 0. The
arguments are `(0x1111, 901, 7, 0xaaaa, 0xbbbb, 0x0102…25 (37 bytes), 200000)` and
`block.timestamp = 1700000000`.

| theorem | checks |
|---|---|
| `exportHash_matches_cast` | `exportHash env0` = foundry's `cast keccak $(cast abi-encode "f(uint256,uint256,uint256,address,address,bytes)" 1 901 7 0x…aaaa 0x…bbbb 0x0102…25)` = `0xe8834c9a…ff2c` (kernel) |
| `sendMessageCd_matches_cast` | `keccak256(sendMessageCd 0x1111 Hcast 1700000000 200000)` = `cast keccak $(cast calldata "sendMessage(address,bytes,uint32)" 0x…1111 $(cast calldata "relayUndeliveredMessage(bytes32,uint256)" Hcast 1700000000) 200000)` = `0x7b18faca…08c7` (kernel) |
| `success_reachable` | succeeds; output `abi.encode(Hcast)`; the mock messenger received calldata with keccak `0x7b18faca…08c7`; the last log entry is `exportedLog env0 Hcast` |
| `relayed_reverts` | mock returns true for `Hcast` → revert output exactly `0xccc3f3b0` |
| `no_code_reverts` | no code at 0x4200…0007 → empty revert |
| `callee_revert_bubbles` | 0x4200…0007 reverts with `0xdeadbeef` → the run reverts with `0xdeadbeef` |
| `dirty_sender_reverts` | `_sender` with bit 160 set → empty revert |
| `value_reverts` | 1 wei attached → empty revert |
| `static_violation` | `perm = false`, 0x4200…0007 code `STOP` → `StaticModeViolation` |

## Proof structure

The trace is split into segments; each is a theorem whose conclusion is a disjunction of
terminals. Block names are `exporter_block_<pc>`, from EquiVM's generator.

| file | pcs | content |
|---|---|---|
| `TraceEntry.lean` | 0 → 170 | non-payable check, dispatcher, ABI decoder with the dynamic `bytes` (`seg_entry`) |
| `TraceHash.lean` | 170 → 240 | `bytes memory m = _message`, `abi.encode(…)`, the copy loop, `KECCAK256` = `exportHash I` (`seg_hash`) |
| `TraceStatic.lean` | 240 → 439 | `STATICCALL successfulMessages(H)`, bool decoding, the custom error (`seg_static`) |
| `TraceCall.lean` | 439 → 710 | `abi.encodeCall(relayUndeliveredMessage, …)`, `sendMessage` encoding (3-iteration loop), `EXTCODESIZE`, `CALL` (`seg_call`) |
| `TraceLog.lean` | 710 → `RETURN` | `LOG3` (static split), `return messageHash_` (`seg_log`, `log_tail`) |
| `Export.lean` | | composition (`export_trace`) and the `Ξ`-level theorems |

Supporting files: `Spec.lean` (vocabulary), `Terminal.lean` (`RDrevP`, `RD.revP`, `RD.rev00`),
`Words2.lean` (decoder and loop branch conditions), `Loop.lean` (`copy_loop` by induction,
`copy_tail`), `HashMem.lean` and `CallMem.lean` (list-memory lemmas relative to symbolic
free-memory pointers), `LogTrack.lean` (`RDL`, `RDretL`).

## Build and timings

```sh
cd packages/contracts-bedrock/test/formal/expiry/evm-lean-exporter
lake exe cache get                          # Mathlib cache (or copy ../evm-lean-bridge/.lake/packages: same pins)
lake build ExporterEvm ExporterEvm.Axioms   # "Build completed successfully"; 6 "standard axioms only"
                                            # + 6 "partner … present" lines, 2 more axiom lines
grep -rn "sorry\|admit" ExporterEvm/        # nothing
```

On a shared 32-core Linux host under a 16 GB memory cap, with dependencies built, a clean rebuild
of this project took **1 min 52 s** wall (187 s user CPU, 8.8 GB peak RSS). The slowest file is
`Concrete.lean` (36 s: two kernel-evaluated keccak hashes and seven compiled `Ξ` runs); the five
generated block shards take 9–17 s each and the proof files 2–8 s each.

## Not covered

* **Completeness.** No theorem says that the call succeeds when the conditions hold (EquiVM's `RD`
  is partial correctness modulo out of gas, and the forwarded call gas is existential).
* **The callees.** The code of the L2ToL2CrossDomainMessenger's `successfulMessages` getter and of
  `L2CrossDomainMessenger.sendMessage` is not verified. What `sendMessage` does with the message
  (a withdrawal whose L2 sender is the exporter), and the L1 side, are covered elsewhere
  (`../evm-lean-l1cdm` for `relayUndeliveredMessage`, `../lean` for the protocol).
* **Chain ids other than 1.** EVMLean's `CHAINID` is the constant 1, so the preimage's first word
  is that constant; the proof does not use its value, but no theorem covers other chain ids.
* **The protocol-level meaning.** That `successfulMessages(H) = false` means the message was not
  relayed is the messenger's property; that the destination is this chain is by construction of
  `H` (the first preimage word is `block.chainid`, as EVMLean models it).
* **Revert causes** of the empty output are not distinguished.
* **The rest of the substate** (earlier log entries, access sets, refunds) is not described.
* `version()`.
* **The proxy in the concrete runs.** The concrete runs call the implementation directly; the
  theorems cover the proxy frame.

## Files

| file | content |
|---|---|
| `lakefile.toml`, `lean-toolchain`, `lake-manifest.json` | project, pinned |
| `bytecode/UndeliveredMessageExporter.runtime.hex` | compiled runtime |
| `scripts/regen.sh`, `scripts/gen_bytecode.py` | validated recompilation, generation of `Bytecode.lean`, `Blocks/`, `AllBlocks.lean` |
| `ExporterEvm/Bytecode.lean`, `ExporterEvm/Blocks/`, `ExporterEvm/AllBlocks.lean` | generated: bytecode, kernel-checked `RD` summaries of every basic block |
| `ExporterEvm/KernelDecide.lean`, `KernelRun.lean`, `Mem.lean`, `Words.lean` | copied from `../evm-lean-bridge` |
| `ExporterEvm/Spec.lean` | statement vocabulary |
| `ExporterEvm/Terminal.lean`, `Words2.lean`, `Loop.lean`, `HashMem.lean`, `CallMem.lean`, `LogTrack.lean` | libraries |
| `ExporterEvm/Trace*.lean`, `ExporterEvm/Export.lean` | proof |
| `ExporterEvm/Concrete.lean` | executable checks |
| `ExporterEvm/NonVacuous.lean` | one `nonvacuous_` partner per headline theorem |
| `ExporterEvm/Axioms.lean` | `#assert_headline`: standard axioms + partner present |

## Review log

* **Self-review, before hand-back.** Lessons from the reviews of `../evm-lean-bridge` and
  `../evm-lean-l1cdm` applied: no call summaries or universally quantified frame hypotheses; the
  return-data condition is "at least 32 bytes and the first word"; no completeness clause with an
  existential-gas failure disjunct; `regen.sh` validates the full settings and the semver lock
  before writing; every headline theorem has a build-enforced non-vacuity partner.
* **Round 1: R1 (fresh-context reviewer), R2 and R3 (independent model-based reviewers).** None
  found a critical or high issue or a bug in the contract. R2 and R3 independently recompiled the
  source and matched the runtime byte for byte and the init-code hash with the semver lock; R1
  matched every selector, the topic, both cast reference hashes, all README pcs and each `ArgsOk`
  field against the compiled decoder, and checked `RDL`/`RDrevP` against EquiVM's `RD`. Findings and
  resolutions:
  1. *Medium (R1).* The headline paragraph called the preimage's first word `block.chainid`
     without saying EVMLean fixes it to 1. **Fixed:** stated in the headline paragraph and under
     "Not covered".
  2. *Medium (R1), low (R2, R3).* `#assert_nonvacuous` only checked that a constant with the
     partner's name existed. **Fixed:** it now also requires the partner's proof term to use the
     headline theorem (mutation-tested with a partner proving `True`); the README states that the
     adequacy of the witness itself is reviewed by hand.
  3. *Medium (R2).* "A `STATICCALL`/`CALL` went …" reads as trace evidence, while `StaticCall`/
     `CallTo` quantify gas and substate existentially. **Fixed:** wording qualified in "What is
     proved" (the account-map chain is tied to the run; the gas and substate are not exposed).
  4. *Low (R1, R2, R3).* The revert alternatives overlap (a callee can bubble empty data or
     `0xccc3f3b0`); "calldata shorter than 4 bytes" is excluded by `hsel`. **Fixed:** README and
     `ExportRevert` docstring state that the classification is a necessary condition, not a unique
     cause; the stray cause removed.
  5. *Low (R2, R3).* "Only when entered by `STATICCALL`" is too narrow (a proxy's `DELEGATECALL`
     inherits a static frame). **Fixed:** "in a static frame (`I.perm = false`)" throughout.
  6. *Low (R1).* The bubbled-`sendMessage` revert disjunct dropped facts already proved on that
     path. **Fixed:** it now carries the static call's storage/code preservation and
     `extcodesize(0x4200…0007) ≠ 0`.
  7. *Low (R1).* `regen.sh` did not check the event-topic `PUSH32` (pc 754). **Fixed:** added.
  8. *Low (R1).* The three uses of `hcds` were not listed together. **Fixed:** Hypotheses item 1.
