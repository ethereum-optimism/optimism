# Bytecode-level verification of `expireMessage` in Lean (EquiVM / EVMLean)

Status: **draft. Soundness is proved; a converse (liveness) is not.** `lake build` passes; there
is no `sorry`/`admit` and no project `axiom`; the axiom footprint is below. Kontrol remains the
primary bytecode tool; this is an independent second track in Lean.

This directory proves facts about the **deployed runtime bytecode** of `L2ToL2CrossDomainMessenger`
at `448d31ad19` (tip of the PR #23259 branch: exporter design, `EXPIRY_PERIOD = 8 days`),
executed by an executable Lean model of the EVM (`Ξ`). The theorems quantify over every account
map, caller, value, calldata that selects `expireMessage(bytes32 H, uint256 t)`, gas, call depth
and static flag:

* **Soundness** (`expireMessage_success`): the run succeeds only if every one of these holds:
  * the call is not static (entered by `CALL`, not `STATICCALL`);
  * no ETH is attached;
  * the calldata length passes solc's ABI check: `68 ≤ calldatasize < 2^255 + 4`;
  * `msg.sender` is the L2CrossDomainMessenger;
  * `xDomainMessageSender() == otherMessenger()`, taking each value as the first word the code
    decodes from that call's return data;
  * `sentMessageTimestamps[H] ≠ 0`;
  * `sentAt + EXPIRY_PERIOD` does not overflow 256 bits;
  * `t > sentAt + EXPIRY_PERIOD`.

  On success the only state change is `expiredMessages[H] := true` (with Solidity's packed-bool
  write). That covers the storage of every account, all transient storage and all code; the
  substate (logs, access sets, refunds), balances and nonces are not asserted.
* **Outcome classification** (`expireMessage_outcome`, `expireMessage_no_other_error`): every run
  ends in one of four ways: out of gas; a revert; a static-mode violation (only on a static call
  that satisfies all the conditions); or a success as above. No other exceptional halt occurs.
* **Non-vacuity** (`NonVacuity.lean`, checked at build time): for **every** headline theorem `T`
  there is a `nonvacuous_T` that satisfies all of `T`'s hypotheses jointly on one concrete run and
  applies `T` there. Everything except "`Ξ` ends in success / out of gas / static violation on this
  input" is kernel-checked; see "Non-vacuity" below.
* **No converse.** I tried to prove "the conditions imply success unless out-of-gas or a callee
  failure", and that statement turned out to be **vacuous**. The framework does not expose how
  much gas a call forwards, so "a callee failure" can be satisfied in any state by a 0-gas call;
  `Concrete.callFailed_in_success_state` proves this. The lemma survives only as the auxiliary
  `expireMessage_revert_cause`, labelled as not a completeness result. Liveness is demonstrated by
  executable witnesses only (`Concrete.lean`).
* **Bridge** (`Abstract.refines_expire`): a **conditional per-key correspondence** with the
  protocol model's `expire` step, with the deposit history supplied externally. The model's
  `deposits f` is replaced by the call-time authorization `RelayFromOtherMessenger`, and each key
  is covered only under a slot-distinctness side condition; see "Connection to `../lean/`".

## Toolchain (all pinned)

| Component | Version | Role |
|---|---|---|
| Lean | `leanprover/lean4:v4.29.0` (`lean-toolchain`) | prover |
| [EquiVM](https://github.com/argotorg/EquiVM) (MIT) | `b0e9d55a277bc081411fd11f1b69f2e11b38e478` | refinement framework of arXiv 2607.26306: `RD` symbolic-execution invariant, opcode lemmas, static-call lemmas, block-summary generator |
| [EVMLean](https://github.com/lefterislazar/EVMLean) | `63f61339dd17a809961e34028cbc8c5fcb02af13` | the EVM semantics (`Ξ`, `Θ`, `X`): EquiVM's port of Nethermind's EVMYulLean to Lean 4.29; passes the Ethereum conformance tests (per its README); Cancun opcodes incl. `TLOAD`/`TSTORE`/`MCOPY` |
| Mathlib | `v4.29.0` (`8a178386…`) | via EquiVM; prebuilt cache with `lake exe cache get` |

Exact transitive pins are in `lake-manifest.json`.

Why EquiVM rather than raw EVMYulLean: Nethermind's EVMYulLean (`047f630`, Lean 4.22) builds
fine (2 min with the Mathlib cache on a 32-core Linux build host), but has no symbolic-execution library; EquiVM
(published, MIT) is built on a port of it and adds exactly what a dispatcher of this size needs:
a reached-or-out-of-gas invariant `RD` with one lemma per opcode, `STATICCALL`/`CALL` lemmas
that keep the callee as an opaque `Θ` result, and a generator that proves an `RD` summary for
every basic block of a bytecode. Clear (Yul level) was not needed. EquiVM's spec language Sol⁻
and its `runtimeRefinement` relation are not used: they target whole-contract refinement, and the
statements here are stated directly on `Ξ` (see HOWTO.md for using `runtimeRefinementFor` with a
one-transition Sol⁻ spec instead).


## The artifact proved against

| | |
|---|---|
| Source | `src/L2/L2ToL2CrossDomainMessenger.sol` at `448d31ad19` (`EXPIRY_PERIOD = 8 days`; semver 2.0.0) |
| Compiler | solc `0.8.25+commit.b61c2a91` via forge 1.8.1, repository **default** profile |
| Settings | optimizer on, 999999 runs, `evm_version = cancun`, `bytecode_hash = none` (CBOR trailer `a164736f6c6343000819000a`) |
| Runtime | 5231 bytes, `keccak256 = 0x2598d1f09fc5e29faaba7452988e684dc9c023480bb1fea3937a84511913d5d2` (`bytecode/…runtime.hex`) |
| Init code | `keccak256 = 0x804d3303278dcc3942f9c693fdfca5a09e3ccea9cb0855331f785a4ada89cf64` = `initCodeHash` in `snapshots/semver-lock.json` at `448d31ad19` |
| Lean | `ExpiryEvm/Bytecode.lean` (`l2tol2Runtime`), generated from the hex by `scripts/gen_bytecode.py` |

To reproduce, run `scripts/regen.sh`. It:
* unsets every `FOUNDRY_*` override and forces `FOUNDRY_PROFILE=default`;
* builds into a temporary directory;
* refuses to continue unless all three match: solc is `0.8.25+commit.b61c2a91`; the *complete*
  compiler settings minus remappings equal the expected JSON (so any viaIR, optimizer detail or
  library setting fails the check); and the init-code hash equals the `semver-lock.json` entry;
* only then writes `bytecode/*.hex` (creating the directory) and regenerates
  `ExpiryEvm/Bytecode.lean`, the block summaries `ExpiryEvm/Blocks/` and their import point
  `ExpiryEvm/AllBlocks.lean`;
* sets `P_contract` in `Spec.lean` from the compiled `PUSH3` operands, failing if they disagree.

The pinned hex files live in `bytecode/`, which is not gitignored (check: `git check-ignore bytecode/*`). The earlier `artifacts/` directory was ignored by
`packages/contracts-bedrock/.gitignore`, so those files were never committed.

**Constant.** `P_contract` in `ExpiryEvm/Spec.lean` is the compiled `EXPIRY_PERIOD`:
`PUSH3 0x0a8c00` at pc 2179, i.e. 691200 s = 8 days. The proofs use only the name.

**Retargeting history.** The first version proved the same theorems against `37b44c48c7`
(7 days; runtime keccak `0x84897ff9…0eaa`). Between that and the current tip the messenger
changed: the exporter function moved out, `EXPIRY_PERIOD` was renamed and set to 8 days, and the
target rule was extended. `expireMessage`'s code shape did not change, so retargeting needed two
things: renaming the basic-block pcs (one mechanical map) and setting `P_contract := 691200`.
From `5992028e08` to `c7c51d79e2` (custom errors renamed with the `L2ToL2CrossDomainMessenger_`
prefix, version string `"2.0.0"` → `"1.4.0"`, natspec) the runtime keeps its size and layout: only
five `PUSH32` operands changed (the version string at pc 411 and the error selectors at pcs 1296,
2057, 2198, 3038). `regen.sh` regenerated the bytecode and summaries; no proof file changed (the
proofs do not mention those operands); full rebuild 280 s. From `c7c51d79e2` to `448d31ad19`
(the version string back to `"2.0.0"`; semver-lock regenerated) only the version `PUSH32` at pc 411
changed (`0x312e342e30…` → `0x322e302e30…`); `regen.sh` validated the init-code hash against the new
semver-lock entry, no proof file changed, and the full rebuild took 201 s. The runtime is
byte-identical to `../hevm/current.runtime.hex` at the same commit.

## What is proved

Main file `ExpiryEvm/ExpireMessage.lean`; vocabulary in `ExpiryEvm/Spec.lean`. Every theorem takes

```lean
(hcode : I.code = l2tol2Runtime)                       -- the pinned artifact runs
(hsel  : selectorWord I = expireSelector)              -- CALLDATALOAD(0) >> 224 = 0x763a1cb7
(hcds  : I.calldata.size < 2 ^ 256)                    -- always true in the EVM
(hO : ReturnsAddress σ σ₀ I otherMessengerCalldata vO)        -- call summary (assumption)
(hX : ReturnsAddress σ σ₀ I xDomainMessageSenderCalldata vS)  -- call summary (assumption)
```

and is universally quantified over the account map `σ`, `σ₀`, substate `A`, gas `g`, and the
whole execution environment `I`: caller `I.source`, value `I.weiValue`, calldata, depth, `perm`,
code owner, block header and the rest. `H = argHash I = CALLDATALOAD(4)` and
`t = argTime I = CALLDATALOAD(36)`.

```lean
theorem expireMessage_outcome … :
    Ξ σ σ₀ g A I = .error .OutOfGass ∨
    (∃ g' o, Ξ σ σ₀ g A I = .ok (.revert g' o)) ∨
    (Ξ σ σ₀ g A I = .error .StaticModeViolation ∧ I.perm = false ∧ ExpireConds σ I vO vS) ∨
    (∃ σ' g' A', Ξ σ σ₀ g A I = .ok (.success (σ', g', A') ByteArray.empty) ∧
      I.perm = true ∧ ExpireConds σ I vO vS ∧ ExpirePost σ σ' I)

theorem expireMessage_success … (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    I.perm = true ∧ ExpireConds σ I vO vS ∧ ExpirePost σ σ' I ∧ o = ByteArray.empty

theorem expireMessage_no_other_error … (he : Ξ σ σ₀ g A I = .error e) :
    e = .OutOfGass ∨ (e = .StaticModeViolation ∧ I.perm = false)

-- Auxiliary, NOT a completeness theorem (see below; `expireMessage_outcome_aux` likewise
-- annotates the revert case with `¬ ExpireConds ∨ CallFailed`):
theorem expireMessage_revert_cause … (hc : ExpireConds σ I vO vS) (hperm : I.perm = true) :
    Ξ … = .error .OutOfGass ∨ (∃ σ' g' A', Ξ … = .ok (.success (σ', g', A') ByteArray.empty) ∧
      ExpirePost σ σ' I) ∨ ((∃ g' o, Ξ … = .ok (.revert g' o)) ∧ CallFailed σ σ₀ I)
```

`ExpireConds σ I vO vS` (all must hold; `sentAt σ I = sentMessageTimestamps[H]` read from `σ`):

| field | meaning |
|---|---|
| `noValue` | `I.weiValue = 0` (non-payable) |
| `calldataLen` | `68 ≤ calldatasize < 2^255 + 4` (solc's signed ABI length check) |
| `callerIsL2cdm` | `I.source = 0x4200…0007` (`msg.sender`) |
| `senderIsOther` | `vS = vO`, i.e. `xDomainMessageSender() == otherMessenger()` |
| `wasSent` | `sentAt ≠ 0` |
| `noOverflow` | `sentAt.toNat + P_contract < 2^256` (checked add; overflow → `Panic(0x11)` revert) |
| `expired` | `sentAt.toNat + P_contract < t.toNat` (strict) |

`ExpirePost σ σ' I` says three things:
* The code owner's storage is `old.insert (expiredSlot H) (setTrueWord old[expiredSlot H])`, where
  `setTrueWord w = 1 | (w & ~0xff)`. That is exactly what the code does: Solidity's packed-bool
  write keeps the other 31 bytes, and for a slot only ever written as a bool the result is `1`.
* Every other account's storage is unchanged.
* Every account's transient storage and code are unchanged.

Slots: `sentAtSlot H = keccak256(H ‖ 3)`, `expiredSlot H = keccak256(H ‖ 4)` (storage layout:
slot 3 is `sentMessageTimestamps`, slot 4 is `expiredMessages`).

A revert leaves no state change, and EVMLean guarantees that by construction:
`ExecutionResult.revert` carries only gas and output, and `Θ` (message call) returns the caller's
original account map when the callee's `Ξ` reverts or errors.

### Why there is no completeness theorem

EquiVM's invariant `RD` means "reached this point, or the run ran out of gas", so everything it
proves is partial correctness modulo out-of-gas. Two more things hide gas:
* `RD.solcStaticcall` hands back the callee's `Θ` result with the forwarded call gas and the
  substate existentially quantified.
* The block summaries for warm/cold accesses keep their gas counters existential.

`CallFailed` (`Spec.lean`) is as tight as the trace allows: call 1 from exactly `σ`, call 2 from a
state produced by a successful call 1, code and storage tracked. Even so it is satisfiable whenever
*some* forwarded gas makes the callee fail, which a 0-gas call always does.
`Concrete.callFailed_in_success_state` proves `CallFailed` in the very state where
`success_reachable` shows success. So `expireMessage_revert_cause` does not exclude any revert.
A real liveness theorem with a gas bound needs a gas-tracking ("reached and not out of gas")
layer that EquiVM does not provide.

What is shown:
* `Concrete.lean` executes the real bytecode and reaches success, the boundary revert, a revert
  from another caller, out-of-gas, a static violation, and a revert when the callee fails.
* At the code level, inspection of the trace files shows that under `ExpireConds` the only
  remaining revert blocks are the two bubbled-up call failures (pcs 1829 and 1980,
  `RETURNDATACOPY; REVERT`). This is not a stated theorem.

### Bridge (`ExpiryEvm/Abstract.lean`)

```lean
theorem refines_expire … (hres : Ξ σ σ₀ g A I = .ok (.success (σ', g', A') o)) :
    absGuard P_contract (RelayFromOtherMessenger I vO vS) (viewOf σ I.codeOwner) (argHash I) (argTime I).toNat ∧
    (viewOf σ' I.codeOwner).expired (argHash I) ∧
    (∀ H, sentAtSlot H ≠ expiredSlot (argHash I) →
      (viewOf σ' I.codeOwner).sentAt H = (viewOf σ I.codeOwner).sentAt H) ∧
    (∀ H, expiredSlot H ≠ expiredSlot (argHash I) →
      ((viewOf σ' I.codeOwner).expired H ↔ (absNext (viewOf σ I.codeOwner) (argHash I)).expired H)) ∧
    (∀ s, s ≠ expiredSlot (argHash I) → storageWord σ' I.codeOwner s = storageWord σ I.codeOwner s) ∧
    (∀ a, a ≠ I.codeOwner → (σ'.getD a default).storage = (σ.getD a default).storage)
```

Notes on reading it:
* Even `sentMessageTimestamps[H]` for the expired key `H` itself is preserved only under the side
  condition `sentAtSlot H ≠ expiredSlot H`.
* The slot frame (`∀ s ≠ expiredSlot H`) covers the messenger's other state, with per-key side
  conditions. `Abstract.frame_other_maps` names the instances: `successfulMessages[H']`
  (`keccak(H'‖0)`), `sentMessages[n]` (`keccak(n‖2)`) and `msgNonce` (slot 1).

### Executable witnesses (`ExpiryEvm/Concrete.lean`)

These run the real bytecode under `Ξ` (compiled evaluation, `native_decide`) in a fixed state:
* 0x..07 holds a mock L2CrossDomainMessenger that returns one fixed address for both calls;
* `sentMessageTimestamps[H] = 5`.

| theorem | run | result |
|---|---|---|
| `success_reachable` | `t = 5 + P + 1`, caller 0x..07, 10^6 gas | success, `expiredMessages[H] = 1` |
| `boundary_reverts` | `t = 5 + P` (the `≥` mutation would accept) | revert |
| `other_caller_reverts` | caller 0x99 | revert |
| `oog_reachable` | 20000 gas | out of gas |
| `static_reachable` | `perm = false` | `StaticModeViolation` |
| `callee_failure_reverts` | 0x..07 code `PUSH0 PUSH0 REVERT` | revert |
| `callFailed_in_success_state` | `CallFailed` holds in the success state (0-gas call) | shows `CallFailed` is weak |

These two are proved theorems, not executions:
* `mock_returnsAddress`: for every calldata, environment and `σ₀`, `ReturnsAddress σ σ₀ I cd vMock`
  holds in the concrete state. The proof goes from any storage- and code-equal account map, to
  `toExecute` running the mock, to `theta_code_success` (a successful `Θ` is a successful `Ξ` of
  the callee code), to an `RD` trace of the mock (`mock_xi`: out of gas, or exactly 32 bytes of
  `0xbeef`).
* `success_instance`: instantiates `expireMessage_success` on the concrete successful run with
  every hypothesis discharged (code `rfl`, selector and calldata bound by `decide +kernel`, both
  summaries by `mock_returnsAddress`), yielding `ExpireConds` and `ExpirePost` there. Its only
  compiled-evaluation fact is `native_xi_success` (the run succeeds).

`mock_returnsAddress` and its pieces (`mock_xi`, `toExecute_mock`) are now kernel-checked
(`kevm_run`/`evm_kdecide`/`decide +kernel` instead of `native_decide`).

## Non-vacuity

`ExpiryEvm/NonVacuity.lean` has one witness per headline theorem; `ExpiryEvm/Axioms.lean`
(`#assert_headline`) fails `lake build` unless, for each headline theorem `T`: `T` uses only the
three standard axioms; `nonvacuous_T` exists and its proof term applies `T`; and every non-standard
axiom of `nonvacuous_T` comes from a theorem named `native_*` (the traversal stops at those and
lists them). The witness world is `Concrete.lean`'s (`σ`, `env l2cdm (5 + P + 1)`, 10^6 gas).

| headline theorem | witness | hypotheses (all jointly) | conclusion instantiated | compiled evaluation |
|---|---|---|---|---|
| `expireMessage_outcome` | `nonvacuous_expireMessage_outcome` | code, selector, calldata bound, both summaries: kernel | the success disjunct, with `ExpireConds` and `ExpirePost` | `native_xi_success` (to rule out the other disjuncts) |
| `expireMessage_success` | `nonvacuous_expireMessage_success` | as above + `hres` (success) | `perm`, `ExpireConds`, `ExpirePost`, empty output; and from `ExpirePost` (kernel) `expiredMessages[H] = 1` | `native_xi_success` (`hres`) |
| `expireMessage_revert_cause` | `nonvacuous_expireMessage_revert_cause` | as for outcome + `ExpireConds` (`NV.witness_conds`, every field evaluated by the kernel, keccak slot included) + `perm = true`: **all kernel** | the theorem's disjunction; the realized disjunct is success with `ExpirePost` | `native_xi_success` (only to identify the disjunct) |
| `expireMessage_no_other_error` | `nonvacuous_expireMessage_no_other_error` | as for outcome + `he` for two runs: 20000 gas (out of gas) and `perm = false` (static violation) | both disjuncts: `OutOfGass`, and `StaticModeViolation ∧ perm = false` | `NV.native_xi_oog`, `NV.native_xi_static` (`he`) |
| `Abstract.refines_expire` | `Abstract.nonvacuous_refines_expire` | as for success | `absGuard` with the deposit condition, `expired H`; the per-key side conditions `sentAtSlot H ≠ expiredSlot H` and `expiredSlot H' ≠ expiredSlot H` (kernel keccak) hold, so `sentAt H = 5` is kept and `expired H'` stays false | `native_xi_success` (`hres`) |

**Trust, precisely.** The headline theorems' axiom cones contain no `native_*` lemma (checked).
The witnesses additionally trust Lean's compiler and EVMLean's executable code for exactly these
closed facts, each a `native_decide` on "`Ξ` on this concrete input ends in success / out of gas /
static violation": `Concrete.native_xi_success`, `NV.native_xi_oog`, `NV.native_xi_static`. The
kernel cannot evaluate `Ξ` (its well-founded recursion does not reduce). Everything else in the
witnesses, including keccak256 of the concrete slots, is evaluated by the kernel.

**Hypotheses that quantify over all states (the vacuity-prone class), checked.** Both
`ReturnsAddress` summaries quantify over every account map with the same storage/code, every call
gas and substate; `mock_returnsAddress` proves them for the witness state (any gas, any calldata,
any environment), so they are satisfiable jointly with a successful run. No other hypothesis
quantifies over states, targets, calldata or gas. `CallFailed` is a conclusion-side disjunct, known
weak (see above), not a hypothesis of a headline theorem.

## Hypotheses, summaries, axioms (complete list)

1. `hcode`, `hsel`, `hcds`: the code, the selector, and a calldata bound that holds in any EVM.
2. **Call summaries** (`ReturnsAddress`, `Spec.lean`), one per external call. From any account map
   with the same storage, transient storage and code as `σ`, a *successful* static call from the
   messenger to 0x..07 with calldata `otherMessenger()` (resp. `xDomainMessageSender()`) returns
   **at least 32 bytes** whose first word is the ABI encoding of `vO` (resp. `vS`). Extra bytes
   are allowed: the compiled decoder ignores them, and the proofs handle a symbolic return-data
   length, which shifts the free-memory pointer by `32·⌈n/32⌉`.
   * A failed call is unconstrained.
   * `vO` and `vS` are exactly the addresses the code decodes.
   * The L2CrossDomainMessenger bytecode itself is *not* verified.
   * That static calls change no storage or code is **proved** (EVMLean
     `Theta_static_accountStorageStateEq` and `Theta_static_accountCodeStateEq`), not assumed.
   * **Satisfiability:** proved for the concrete mock state (`Concrete.mock_returnsAddress`), and
     shown jointly satisfiable with success (`Concrete.success_instance`). For the real
     L2CrossDomainMessenger, whose two functions are Solidity `address` getters, it is argued,
     not proved.
3. No hash assumption. The bridge states its view equations per key `H'`, with the side condition
   that `H'`'s slot differs from `expiredSlot H`. Keys whose slot collides, if any exist, are not
   covered. This replaces the earlier global `NoSlotCollision`, which reviewers rightly called an
   idealized assumption.
4. **Lean axioms and trust base.** Every headline theorem is checked by the Lean 4.29.0 kernel and
   depends on `propext`, `Classical.choice` and `Quot.sound` only. `ExpiryEvm/Axioms.lean` asserts
   this with `#assert_headline`, so `lake build` fails if any other axiom appears. There is no
   `sorryAx`, no project `axiom` and no `native_decide` in any headline theorem.
   * **How the bytecode facts are kernel-checked.** The 585 closed facts about the concrete bytecode
     are instruction decodes at concrete pcs, pc arithmetic, JUMPDEST membership and the JUMPDEST
     table. They are proved via two axiom-free lemmas, in `ExpiryEvm/KernelDecide.lean`, that restate
     EVMLean's `decode` and `D_J` as list recursions the kernel evaluates quickly. The bytecode is one
     flat literal. Details, timings and mutation checks are in `../evm-lean-kernel/README.md`.
   * **The trust base:**
     - the kernel, including its built-in acceleration of closed `Nat` operations;
     - EVMLean's definitions;
     - EquiVM's proved `Reasoning` library.

     No compiled code, `@[implemented_by]`, `@[extern]` or `@[csimp]` is trusted.
   * **The `Concrete.lean` executable tests and the `native_*` lemmas of the non-vacuity
     witnesses** use `native_decide`, which evaluates `Ξ` by compiled code; kernel reduction gets
     stuck on its well-founded recursion. They are not dependencies of any headline theorem. Caveat: EVMLean implements some precompiles with
     `@[implemented_by]`, but none of these runs calls a precompile.
5. **Trusted semantics:** EVMLean's `Ξ`/`Θ`/gas model (Cancun; conformance-tested upstream) and
   EquiVM's `Reasoning` library (all proved; no axioms beyond the above).
6. **Context, not used by these proofs:** the branch's named governance assumption that each
   cluster chain's L2 governance can upgrade its own exporter. `expireMessage` trusts the
   L2CrossDomainMessenger and its `otherMessenger`, not the exporter. The exporter enters only
   through the L1 relay that produces the deposit, which is outside this proof.

## Bounds and modelling notes

* Nothing is bounded. Account maps, storage, caller, value, calldata (contents and length), gas,
  depth, `perm` and the return-data lengths of both calls are symbolic. There are no loops.
* Gas: out of gas is always a possible outcome. No liveness is claimed (see above).
* Not asserted after success:
  * Events: `LOG2` runs (its gas and memory are in the trace; static frames are handled), but its
    topics and data are not checked.
  * The substate: logs, accessed addresses and storage keys, refund counter.
  * Balances and nonces.
* Order of external calls: legacy solc evaluates the right operand of `!=` first, so the code calls
  `otherMessenger()` and then `xDomainMessageSender()`. Both happen only if `msg.sender` is the
  L2CrossDomainMessenger (short-circuit `||`).
* Proxy: on L2 the predeploy 0x..23 is a `Proxy` that `DELEGATECALL`s this implementation. The
  theorems quantify over every `I` with `I.code = l2tol2Runtime`, which includes the delegatecall
  frame (code owner = proxy, `I.source` = the original caller). The Proxy's own code is not
  verified.
* `I.source` is `msg.sender` (`CALLER`); `I.sender` is `tx.origin`.

## Connection to `../lean/` (protocol model)

`../lean/Expiry/Model.lean` uses Lean 4.34.1 and is being rewritten as a v2, so it is not imported.
`Abstract.lean` restates its `expire` action for one chain, **compared with the model by hand**:

| Model | here |
|---|---|
| `guard (.expire f) s = s.deposits f ∧ s.sentAt f.toL1 f.hash ≠ 0 ∧ expiredBy cfg (s.sentAt …) f.time` | `absGuard P deposit v h t` |
| `expiredBy cfg sent t = sent + cfg.contractPeriod < t` (`expireGe = false`) | `expiredBy P sent t` |
| `next (.expire f) s = { s with expired := … ∨ (c = f.toL1 ∧ h = f.hash) }` | `absNext v h` |
| `State.sentAt z h` | `(viewOf σ a).sentAt h = sentMessageTimestamps[h].toNat` |
| `State.expired z h` | `(viewOf σ a).expired h = (expiredMessages slot & 0xff ≠ 0)` |
| `State.deposits ⟨z, h, t⟩` | `RelayFromOtherMessenger I vO vS` (see below) |
| `cfg.contractPeriod` | `P_contract` (compiled constant) |
| `f.hash`, `f.time` | `argHash I`, `(argTime I).toNat` |

Limits of the bridge:
* It is a **projection** onto one chain and the two maps that `expire` touches. It is not a
  refinement of the whole model state.
* **Deposit authenticity is supplied externally.** The model's `s.deposits f` is a history fact.
  Here it is replaced by the call-time check the code performs, `RelayFromOtherMessenger` ("the
  L2CrossDomainMessenger relays an L1→L2 message whose L1 sender is `otherMessenger`"). That only
  authentic `expireMessage` deposits satisfy it rests on two things not verified here:
  `L1CrossDomainMessenger.relayUndeliveredMessage`'s checks, and the L2CrossDomainMessenger
  setting `xDomainMessageSender` to the deposit's L1 sender.

## Files

| File | Content |
|---|---|
| `lakefile.toml`, `lean-toolchain`, `lake-manifest.json` | project, pinned |
| `bytecode/*.hex` | compiled runtime and init code (tracked) |
| `scripts/regen.sh`, `scripts/gen_bytecode.py` | validated recompilation; regenerate Lean bytecode, summaries, `AllBlocks.lean`, `P_contract` |
| `scripts/trace_paths.py` | concrete mini-EVM tracer listing the block path of each branch |
| `ExpiryEvm/Bytecode.lean` | generated: runtime bytes + JUMPDEST table |
| `ExpiryEvm/Blocks/RuntimeBlocks_0NN.lean`, `RuntimeBlocks.index`, `ExpiryEvm/AllBlocks.lean` | generated by EquiVM's `generate_rd_blocks.py`: proved `RD` summary of every basic block (all functions) |
| `ExpiryEvm/Spec.lean` | statement vocabulary |
| `ExpiryEvm/Words.lean`, `ExpiryEvm/Mem.lean` | word facts for branch conditions; word-aligned memory (`wordsMem`, `memList`) |
| `ExpiryEvm/TraceEntry.lean` | pc 0 → 1713: dispatcher, non-payable, ABI length |
| `ExpiryEvm/TraceCall1.lean` | pc 1713 → 1872: caller check, `otherMessenger()` call, decode (symbolic return length) |
| `ExpiryEvm/TraceCall2.lean` | pc 1872 → 2103: `xDomainMessageSender()` call, decode, equality |
| `ExpiryEvm/TraceStore.lean` | pc 2103 → end: `sentAt` read, checked add, expiry check, `SSTORE` (static split), `LOG2`, `STOP` |
| `ExpiryEvm/Post.lean` | final account map satisfies `ExpirePost` |
| `ExpiryEvm/ExpireMessage.lean` | headline theorems |
| `ExpiryEvm/Abstract.lean` | projection bridge to the protocol model's `expire` |
| `ExpiryEvm/Concrete.lean` | executable witnesses |
| `ExpiryEvm/NonVacuity.lean` | one `nonvacuous_*` witness per headline theorem |
| `ExpiryEvm/Axioms.lean` | `#assert_headline`: axiom footprint + witness check (fails the build) |
| `HOWTO.md` | adding the next function |

## Build and timings

```sh
cd packages/contracts-bedrock/test/formal/expiry/evm-lean
lake exe cache get        # Mathlib cache
lake build                # everything; must report "Build completed successfully"
lake env lean ExpiryEvm/Axioms.lean
```

All timings are on a shared 32-core Linux host (, load 35–180 during this work).
* **Fresh build:** at `37b44c48c7`, a copy of this directory with no `.lake` built completely with
  `lake exe cache get && lake build` in 5 min 16 s wall, dependencies included.
* **Incremental:** with dependencies built, rebuilding everything in this directory takes about
  1 min (18 summary shards + proofs + concrete runs).
* **Retarget:** the full rebuild after moving to `5992028e08` took 66 s; to `c7c51d79e2`, 280 s;
  to `448d31ad19`, 201 s
  (load ~30; `NonVacuity.lean` alone ≈ 130–150 s, mostly kernel keccak evaluations).

## Review log

**Round 1 (3-way review: R1, R2, R3) of the `37b44c48c7` version, and the
fixes in this version:**
1. **HIGH, all three reviewers: completeness was vacuous.** `CallFailed` did not require code
   equality and quantified gas existentially, so it was satisfiable everywhere.
   * The theorem is renamed `expireMessage_revert_cause` and documented as *not* completeness.
   * `CallFailed` is tightened to the actual call order and states.
   * `Concrete.callFailed_in_success_state` proves the weakness in Lean.
   * The intro and the "Why there is no completeness theorem" section state that soundness is the
     claimed result.
   * `Abstract.complete_expire` is removed.
2. **MEDIUM: `ReturnsAddress` required exactly 32 bytes of return data.** It now requires
   "≥ 32 bytes, first word = the address", and the call segments were re-proved with a symbolic
   return-data length. That meant symbolic free-pointer rounding and a zero gap of symbolic length
   in memory (`Mem.wordsMem_write_gap`, `Mem.memList`). The summaries' satisfiability is argued,
   not proved (see list item 2 of the hypotheses).
3. **MEDIUM: `NoSlotCollision` excluded collisions against every key.** It is removed; the bridge
   is now per-key (hypotheses item 3).
4. **MEDIUM: the bridge substitutes authorization.** It is now documented as a projection, with
   deposit authenticity supplied externally.
5. **LOW items:**
   * The intro now lists every soundness condition.
   * `regen.sh` forces the default profile, pins and validates solc and the settings, and checks
     the init-code hash against `semver-lock.json` before replacing artifacts.
   * Out-of-gas, static and callee-failure witnesses were added.
   * The README now states that the substate is not asserted, records the precompile
     `implemented_by` caveat, and notes that the restated bridge was compared by hand.

**Round 2 (R1, R2, R3) of the `5992028e08` version.** No critical or high
findings; all round-1 fixes were verified (mutations of P, of the tightened check, of the slot,
and a 31-byte summary all break the build). Fixes:
1. **MEDIUM: the hex artifacts were gitignored.** `artifacts/` matched
   `packages/contracts-bedrock/.gitignore:2`, so regeneration failed in a clean checkout. The
   directory is renamed `bytecode/` and tracked, and `regen.sh` creates it.
2. **MEDIUM: bridge wording and frame.**
   * `DepositCall` is renamed `RelayFromOtherMessenger`.
   * The bridge is now described as a conditional per-key correspondence, with the deposit
     history supplied externally.
   * `refines_expire` now exports the slot frame for every other messenger slot, and
     `frame_other_maps` names the instances for `successfulMessages`, `sentMessages` and
     `msgNonce`.
3. **MEDIUM: non-vacuity of the headline hypotheses.** Both summaries are proved for the
   concrete mock (`mock_returnsAddress`), and `expireMessage_success` is instantiated on the
   concrete successful run (`success_instance`).
4. **LOW items:**
   * HOWTO no longer recommends `_complete`.
   * `CallFailed` is dropped from the headline `expireMessage_outcome`; the annotated version is
     `_outcome_aux`, and `_revert_cause` is auxiliary.
   * The README notes that even `sentAt[H]`'s preservation needs a slot side condition.
   * `regen.sh` compares the complete settings and automates `P_contract`.
   * The `gen_bytecode.py` docstring is fixed.

**Round 3 (non-vacuity audit, all headline theorems).** Prompted by an unsatisfiable-hypothesis
finding in a sibling project (`../evm-lean-l1cdm/`, `BoundedCalls`). No hypothesis of this project
was found unsatisfiable. Changes: one kernel-checked (except the named `Ξ` evaluations)
`nonvacuous_*` witness per headline theorem, a build-time check that every headline theorem has
one, `mock_returnsAddress` moved from `native_decide` to the kernel, and the retarget to
`c7c51d79e2`.

Round 1 reviewers also confirmed, with no change needed: soundness is bytecode-faithful and
mutation-sensitive (mutating the `PUSH3` window or `GT`→`LT` breaks the build), and the
artifacts and hashes check out.

## What's left

* **Liveness:** a gas-tracking layer (reached and not out of gas, with forwarded-gas bounds) and a
  gas-parametric callee summary.
* Discharge the two call summaries by proving the L2CrossDomainMessenger's `otherMessenger()` and
  `xDomainMessageSender()` bytecode with the same infrastructure, and prove that its
  `relayMessage` sets `xDomainMessageSender` (that is what makes `RelayFromOtherMessenger` the model's
  `deposits`).
* Assert the `MessageExpired` event (topics, data); the `LOG2` arguments are already in the trace.
* A `Θ`-level statement (message call into the proxy, value transfer, rollback) instead of `Ξ`.
* Import the protocol model directly once it and this project share a toolchain.
* The other functions: see HOWTO.md.
