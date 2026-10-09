# Dafny: the off-chain half of the expiry argument, on the supernode's own model

**What this proves.** The repo already has a Dafny model of op-supernode's interop validity
(`op-supernode/dafny-models/`). `ExpiryBridge.dfy` `include`s that model and proves the following,
using the model's own definitions:

> Let P be the contract's expiry period (`L2ToL2CrossDomainMessenger.EXPIRY_PERIOD`, an immutable
> set by the constructor; production deployments pass 8 days = 691200 s) and W the protocol window (`messageExpiryWindow` / `MESSAGE_EXPIRY_WINDOW`), with
> W ≤ P. Suppose the destination exported the message as undelivered at block timestamp
> tExport > initTimestamp + P, which is exactly when `expireMessage` accepts the fact. Then no
> executing timestamp exec ≥ tExport makes the message valid: `!ValidExecutingMessage(exec, …)`,
> and the model's imperative `VerifyExecutingMessage` returns `false`. When the source chain is
> registered and both activation checks pass, the rejection comes from the `ErrMessageExpired`
> guard; otherwise an earlier check rejects first. The branch claim rests on a hand-copied
> restatement of the guards (see "Which branch fails").

The definitions used:
- `Interop.ValidExecutingMessage`, the declarative predicate.
- `Interop.VerifyExecutingMessage`, the imperative check.
- The `Interop.messageExpiryWindow` field and the `Types.MESSAGE_EXPIRY_WINDOW` constant, which
  `Interop.Valid()` ties together.

The theorems refer to these definitions directly and restate none of them. Two auxiliary
conclusions do hand-copy guard expressions from `VerifyExecutingMessage`'s body, because Dafny
cannot refer to a method's internal guards:
- the second `ensures` of `ExpiredAtExportNeverValid`;
- `ExpiryIsTheFailingGuard`.

Both are labelled **RESTATEMENT** in the source. The file adds no axioms, no `{:axiom}`, no
`assume` and no bodiless lemmas.

The on-chain half is proved in `../lean`, `../kontrol` and `../halmos`. It says `expireMessage`
marks a message expired only if `t > sentAt + P` (a call for an already-expired message returns
early and changes nothing), where `t` is the destination's `block.timestamp` at export, taken
while the message was unrelayed there. This directory supplies the next step: after that moment
the destination can never validly relay the message. It proves this against the supernode model,
not against our own `t ≤ e + W` abstraction.

## Dafny version

The repo pins no Dafny version. There is no CI job, no `mise.toml` entry and no justfile recipe,
and `dafny-spec-check.md` names no version. The PR that added the model (#22507) says "the Dafny
CLI is not installed in the local environment, so the model was not re-verified locally". We
therefore use the latest release:

- **Dafny 4.11.0** (`4.11.0+fcb2042d6d043a2634f0854338c08feeaaaf4ae2`), with the bundled
  **Z3 4.12.1**.
- The release zip is `dafny-4.11.0-x64-ubuntu-22.04.zip`, sha256
  `a46a9ff7cdd720f7955854c78e95df13f4cfe6b80691b05f8654fe19e8267179`. It is self-contained, so no
  .NET install is needed. Any Dafny 4.11.0 install works.

```sh
curl -sSLo dafny.zip https://github.com/dafny-lang/dafny/releases/download/v4.11.0/dafny-4.11.0-x64-ubuntu-22.04.zip
unzip -q dafny.zip -d dafny-4.11.0
export DAFNY=$PWD/dafny-4.11.0/dafny/dafny
```

## How to run

```sh
DAFNY=/path/to/dafny ./run.sh            # ExpiryBridge.dfy must verify; ExpectFail.dfy must fail exactly as expected;
                                         # every lemma has a NonVacuity.dfy witness; witnesses verify; vacuity probe fails
DAFNY=/path/to/dafny ./run.sh --model    # first re-verify the whole existing model (all six files)
```

On a shared build host, run it under a memory cap:
`systemd-run --user --scope -p MemoryMax=16G -p MemorySwapMax=0 ./run.sh --model`.

**Flags.**
- `run.sh` passes `--allow-warnings`. The included model has three `assume` statements without
  `{:axiom}` (`Interop.dfy:1019`, `:1020`, `:1631`). Dafny warns on them, and without the flag it
  exits 2 even when verification succeeds. None of the three is on a path this file uses (see the
  trust base below).
- Dafny does not verify included files by default, so plain `run.sh` checks only our files.
- `--model` adds `--verify-included-files`. It then checks `Interop.dfy` together with
  `VerifiedDB.dfy`, `ChainContainer.dfy`, `LogsDB.dfy`, `Types.dfy` and `Utils.dfy`.
- `--model` also uses `--verification-time-limit 1200`, because the default limit timed out
  under heavy load (see Results).

**The expected-fail check.** `ExpectFail.dfy` passes only if all of these hold:
- Dafny exits with status 4 (verification errors, not a parse or resolution error);
- the summary reads `2 verified, 2 errors`;
- both errors are "a postcondition could not be proved";
- their related locations are exactly the two lines marked `// EXPECT-FAIL`, one in
  `NoPeriodAssumption` and one in `ExclusiveBoundary`.

Two mutants of `ExpectFail.dfy` were run against this check, and both were caught:
- `requires false` added to `ExclusiveBoundary`;
- `ensures false` added to `NoPeriodAssumption`.

## Non-vacuity (`NonVacuity.dfy`, checked by `run.sh`)

`NonVacuity.dfy` has one method `Nonvacuous_<Name>` per lemma/method of `ExpiryBridge.dfy`.
Each builds a `Valid()` instance with the model's constructor (`new I.Interop(map k | k in
CHAIN_IDS :: cc)`), picks a concrete message (`ExecutingMessage(c, 0, 0, act + 2·BlockTime, 0)`),
and calls the lemma, so Dafny must prove all of the lemma's `requires` **jointly** at the call; the
witness then asserts the lemma's conclusion on that instance. For a negative conclusion ("not
valid", "rejected"), the witness also shows the same message **is** valid at the boundary
(`BoundaryStillValid`), so the conclusion is not true merely because nothing is valid.

`run.sh` enforces, and fails otherwise:
1. **Coverage:** every `lemma`/`method` declared in `ExpiryBridge.dfy` has a `Nonvacuous_<Name>`
   method that calls `B.<Name>(`.
2. **The witnesses verify** with 0 errors.
3. **Vacuity probe:** a copy with `assert false;` inserted at the end of every witness must fail at
   exactly the probe lines, one per witness. A witness whose context (its own `requires` plus the constructor's
   postconditions) were contradictory would prove `false`. This is a smoke test (Dafny failing to
   prove `false` is evidence, not proof, that the context is consistent); a manual run with
   `assert false` in three witnesses failed exactly there.

**Residual hypotheses** of the witnesses. They are about constants and functions the model leaves
abstract, so they can only enter as `requires`; each is satisfiable by choosing that constant:
- `c in CHAIN_IDS` (`CHAIN_IDS` non-empty) and a `ChainContainer` object `cc` (no constructor
  outside its module): all witnesses.
- `MESSAGE_EXPIRY_WINDOW <= PROTOCOL_WINDOW_CAP`: only the witnesses of the lemmas that themselves
  assume the cap (`ExpiredAtExportRejectedWithCap`, `EndToEndWitness`, `CapImpliesWindowWithinPeriod`).
- `MESSAGE_EXPIRY_WINDOW > 0`: only `ShortPeriodCounterexample` (`P < W` needs `W ≥ 1`).
- `cc.BlockInfo(blk).Some?`: only `TimestampBoundToInitBlock`. Its `IsCorrectFrontierView`
  hypothesis is supplied by the model's own `ResolveFrontierVerificationView`, whose
  `ensures {:axiom} IsCorrectFrontierView(...)` is how the model provides it. The *relevant*
  execution for this lemma (the imperative check accepting, `ok == true`) is **not** exhibited:
  acceptance depends on abstract logs-DB / frontier contents. Only joint satisfiability of its
  hypotheses is shown.

No hypothesis of any lemma was found unsatisfiable. `run.sh` total, including the probe: 18 s.

## Results (Dafny 4.11.0)

Re-run on the expiry PR stack (formal branch at `2c85209dc9`): `run.sh --model` gives the same
counts as below: the existing model `7355 verified, 0 errors`, `ExpiryBridge.dfy` `28 verified,
0 errors`, `ExpectFail.dfy` rejected at exactly its two EXPECT-FAIL postconditions,
`NonVacuity.dfy` `16 verified, 0 errors`, and `assert false` unprovable in all 14 witness
contexts. The table records the first run, at `5992028e08`.

| Check | Result | Wall time |
|---|---|---|
| `ExpiryBridge.dfy` (v2) | `28 verified, 0 errors` | about 5 s (about 6 to 10 s for all of `run.sh`) |
| `ExpiryBridge.dfy` (v1) with `--warn-contradictory-assumptions` | `28 verified, 0 errors`, no contradictory-assumption warnings | about 5 s |
| `ExpectFail.dfy` (must fail) | exit 4, `2 verified, 2 errors`, postcondition failures at the `EXPECT-FAIL` lines 21 and 30 | about 4 s |
| Existing model, all six files (`run.sh --model`, with `--verify-included-files`) (task 1) | **`7355 verified, 0 errors`**, exit 0. That is 42 VCs more than the Interop.dfy-only run, coming from VerifiedDB, Utils, Types and the other included files | 6 min 36 s wall for all of `run.sh --model`, 844 s CPU, peak RSS 1.4 GB (16 GB memory cap) |
| Existing model, `Interop.dfy` only (v1 run, without `--verify-included-files`) | `7313 verified, 0 errors`, exit 0, but this covers only `Interop.dfy`'s own declarations | 10 min 19 s wall, 1118 s CPU, peak RSS 1.5 GB |
| Existing model, first attempt (default time limit, `--cores 16`, load average about 330) | Internal error "The operation has timed out" on `ApplyPendingTransition` (`Interop.dfy:1054`), plus Z3 pipe "Prover error" lines | 10 min 01 s |

Other jobs loaded the build host heavily during these runs (load average 35 to 330 on 32 cores), so the
times are upper bounds.

**Task 1 verdict.** The existing model, all six files, verifies as-is with Dafny 4.11.0. The only
other output is the three "assume statement has no {:axiom} annotation" warnings. The first
attempt failed only on the time limit under extreme load, not on a proof.

## Exact statements (`ExpiryBridge.dfy`)

In every statement below:
- `i: Interop.Interop` and `msg: ExecutingMessage`;
- `msg.timestamp` is the initiating timestamp;
- `exec` is the executing block timestamp;
- `execChain` is the destination.

**Main theorem, field form.** It needs only the two `requires` of `ValidExecutingMessage` itself,
not `Valid()`.
```dafny
lemma ExpiredAtExportNeverValid(i, P, tExport, exec, execChain, msg)
  requires execChain in CHAIN_IDS && i.chains.Keys == CHAIN_IDS
  requires i.messageExpiryWindow <= P
  requires tExport > msg.timestamp + P
  requires exec >= tExport
  ensures !i.ValidExecutingMessage(exec, execChain, msg)
  ensures msg.timestamp + i.messageExpiryWindow < exec   // RESTATEMENT: hand-copied ErrMessageExpired guard (Interop.dfy:1748)
```

**Main theorem, global form.** `ExpiredAtExportNeverValidGlobal` is the same statement under
`i.Valid()` and `MESSAGE_EXPIRY_WINDOW <= P`.

**Imperative form.** It calls the model's method. The conclusion follows from that method's own
verified postcondition, `valid ==> ValidExecutingMessage(...)`. No guard is restated.
```dafny
method ExpiredAtExportRejected(i, view, P, tExport, exec, execChain, msg) returns (ok: bool)
  requires i.Valid() && execChain in CHAIN_IDS
  requires MESSAGE_EXPIRY_WINDOW <= P
  requires tExport > msg.timestamp + P
  requires exec >= tExport
  ensures !ok
{ ok := i.VerifyExecutingMessage(execChain, exec, msg, view); ... }
```
`ExpiredAtExportRejectedWithCap` sets `P := EXPIRY_PERIOD` (691200) and replaces the side
condition with the cap `MESSAGE_EXPIRY_WINDOW <= 604800`.

**Which branch fails (RESTATEMENT).** `ExpiryIsTheFailingGuard` covers the case where the source
chain is registered, both activation checks pass and init ≤ exec. Its conclusions are the guard
expressions of `VerifyExecutingMessage`, hand-copied from `Interop.dfy:1722`, `:1729`, `:1736`,
`:1742` and `:1748`:
- every guard before the expiry check is false;
- the expiry guard `msg.timestamp + messageExpiryWindow < exec` is true.

Dafny cannot refer to a method's internal guards, and the model returns a `bool` with no error
code. So "returns from the `ErrMessageExpired` branch" depends on the copy being faithful, which
was checked by reading the source, not by Dafny. The verified, non-restated result is
`ExpiredAtExportRejected`: the method returns `false`. When the activation hypotheses fail, an
earlier guard rejects, which is still a rejection.

**Timestamp binding.** `TimestampBoundToInitBlock` (v2) has three hypotheses:
- `i.Valid()`;
- `blocksAtTS.Keys == CHAIN_IDS`;
- **`i.IsCorrectFrontierView(view, blocksAtTS)`**.

It shows that when `VerifyExecutingMessage` accepts, two things hold. First, `msg.timestamp`
equals the timestamp that the model's chain data (`chains[msg.chainID].BlockInfo(...)`) gives the
initiating block. Second, that block's number is `msg.blockNum`. The proof splits on the path:
- **logsDB path** (`msg.timestamp < exec`): the sealed block `FindSealedBlock(msg.blockNum)`. It
  uses the `LogsDB.Contains` `{:axiom}` (timestamp equality), the `LogsDB.FindSealedBlock`
  `{:axiom}` (`id.number == number`), and the model's proved invariant
  `AllLogsDBsConsistentWithChainData`, which is part of `Valid()`.
- **frontier path** (`msg.timestamp == exec`): the block `blocksAtTS[msg.chainID]`. It uses the
  body of `FrontierView.Contains` and the hypothesis `IsCorrectFrontierView`. Inside the model,
  that property comes from the `{:axiom}` ensures of `ResolveFrontierVerificationView`
  (`Interop.dfy:1889`). Go builds the view from fetched block data.

So an executing message cannot stretch its window by claiming a later initiating timestamp. Go and
kona implement the same binding:
- Go: `Contains` in `raftwallogdb/db.go` returns `ErrConflict` when `rec.Timestamp != query.Timestamp`.
- kona: `graph.rs` returns `InvalidMessageTimestamp` when `remote_header.timestamp != initiating_timestamp`.

**The step to the contract's `sentAt` is outside the model.** That step needs "this executing
message references H's `SentMessage` log, whose block also wrote
`sentMessageTimestamps[H] = block.timestamp`". This is an argument about the real checksum: Go and
kona hash the log, block number, log index, timestamp and chain id. In the model,
`ExecutingMessage.checksum` and `Log.checksum` are abstract `nat`s, so the model cannot prove which
message a log is.

**Boundary and non-vacuity.**
```dafny
lemma BoundaryStillValid(i, execChain, msg)
  requires i.Valid() && execChain in CHAIN_IDS && msg.chainID in CHAIN_IDS
  requires i.activationTimestamp + i.chains[msg.chainID].BlockTime() <= msg.timestamp
  requires i.activationTimestamp + i.chains[execChain].BlockTime() <= msg.timestamp + MESSAGE_EXPIRY_WINDOW
  ensures  i.ValidExecutingMessage(msg.timestamp + MESSAGE_EXPIRY_WINDOW, execChain, msg)      // still valid at init + W
  ensures !i.ValidExecutingMessage(msg.timestamp + MESSAGE_EXPIRY_WINDOW + 1, execChain, msg)  // invalid one second later
```
- `BoundaryHypothesesSatisfiable` shows that every `Valid()` instance and every pair of registered
  chains admit a message that is valid at the boundary.
- `EndToEndWitness` builds an instance with the model's own constructor, `new Interop(chains)`,
  whose postcondition is `Valid()`, so the `Valid()` hypotheses are satisfiable. On that instance it
  builds a concrete message that is valid at init + W, then runs `ExpiredAtExportRejectedWithCap`
  on it at `tExport = init + EXPIRY_PERIOD + 1`.
- **Non-vacuity is conditional.** `EndToEndWitness` has three hypotheses: `CHAIN_IDS` is non-empty,
  some `ChainContainer` object exists, and the cap. Dafny cannot discharge the first two. `CHAIN_IDS`
  is an abstract const, and `ChainContainer` cannot be allocated from outside its module, so it is
  taken as a parameter.
- **The witness is declarative only.** It shows the predicate holding at the boundary. It starts
  with empty logsDBs, so it does not show `VerifyExecutingMessage` accepting at the boundary; that
  would need a populated logsDB.

**Counterexample to the timing implication when P < W.**
```dafny
lemma ShortPeriodCounterexample(i, P, execChain, msg) returns (tExport: nat, exec: nat)
  requires (same activation hypotheses as BoundaryStillValid) && P < MESSAGE_EXPIRY_WINDOW
  ensures tExport > msg.timestamp + P                     // expireMessage accepts the fact
  ensures exec >= tExport                                 // the relay comes after the export
  ensures i.ValidExecutingMessage(exec, execChain, msg)   // ...and still passes the validity predicate
```
The witnesses are `tExport = init + P + 1` and `exec = init + W`. This shows **temporal
eligibility** only. It does not establish:
- that the initiating log is present;
- that the imperative check accepts;
- the export, expiry and refund steps.

So on its own it is not a double spend. Lean's `cex_periodBelowWindow`
(`../lean/Expiry/Counterexamples.lean:257`) gives the full execution: export at time 8 > 1 + 6,
relay at 8 ≤ 1 + 7, then refund.

**Expected failures (`ExpectFail.dfy`).** `run.sh` requires exactly these two postcondition
errors, at the marked lines:
- `NoPeriodAssumption`: the main theorem without `W <= P`. It fails.
- `ExclusiveBoundary`: claims the message is invalid at exec == init + W. It fails.

**The cap.** `CapImpliesWindowWithinPeriod` proves
`MESSAGE_EXPIRY_WINDOW <= 604800 ==> MESSAGE_EXPIRY_WINDOW <= 691200 && MESSAGE_EXPIRY_WINDOW + 86400 <= 691200`.

**Link to the Lean and Quint relay rule.**
- `DafnyValidImpliesLeanWithinWindow`: `ValidExecutingMessage(exec, …)` implies
  `exec <= init + messageExpiryWindow`. This is the **numerical** half of the inclusion.
- The full relay-set inclusion also needs two correspondences that Dafny does not encode:
  - **Events.** Lean's `events z h e` and Quint's `e ∈ evts` must match the Dafny message's
    initiating log, with `e = msg.timestamp`.
  - **Windows.** Lean's `protocolWindow x` and Quint's `PROTOCOL_WINDOW` must equal
    `MESSAGE_EXPIRY_WINDOW`. Quint counts in **days** (`expiry.qnt:48`; `PROTOCOL_WINDOW = 7`,
    `CONTRACT_PERIOD = 8` at `:379-380`), so seconds = 86400 × days.
- `ValidIffWindow` states both directions. When both activation checks pass, Dafny validity is
  exactly `init <= exec <= init + W`.
- `LeanAcceptsFutureInitDafnyRejects` shows the containment is strict: exec = init − 1 passes
  `t ≤ e + W` but is invalid in Dafny.

The theorem holds even with W == P, so the boundaries line up exactly. The one-day margin
(P = W + 1 day) is extra slack.

## The 7-day cap and per-depset overrides (task 3)

**`MESSAGE_EXPIRY_WINDOW` is abstract.** `Types.dfy:124` declares `const MESSAGE_EXPIRY_WINDOW: nat`
with no value. Its comment says "Corresponds to defaultMessageExpiryWindow in algo.go", and the
constructor copies it into the `messageExpiryWindow` field. Every theorem about the model therefore
holds for every W, which includes every override value.

**The cap is a hypothesis, not an axiom.** It appears as
`requires MESSAGE_EXPIRY_WINDOW <= PROTOCOL_WINDOW_CAP` in `ExpiredAtExportRejectedWithCap`,
`CapImpliesWindowWithinPeriod` and `EndToEndWitness`. The model cannot derive the cap, because the
cap lives in config parsing, which the model does not describe. Go and kona enforce it, so the
effective W is in [1, 604800]:
- **Go:** `op-core/interop/depset/static_depset.go`.
  - `hydrate` (`:141-142`) rejects `overrideMessageExpiryWindow > MessageExpiryTimeSecondsInterop`
    (604800).
  - Every constructor and decoder calls `hydrate`: `NewStaticConfigDependencySet` (`:33`),
    `NewStaticConfigDependencySetWithMessageExpiryOverride` (`:43`), the JSON decoder (`:87-88`)
    and the TOML decoder (`:132-133`).
  - `MessageExpiryWindow()` (`:172-176`) maps an override of 0 to 604800.
  - The supernode copies this value into `Interop.messageExpiryWindow` (`interop.go:280-282`). With
    no depset it uses `defaultMessageExpiryWindow = 604800`.
- **kona:** `rust/kona/crates/protocol/genesis/src/interop/depset.rs`.
  - `override_message_expiry_window` is an `Option<MessageExpiryOverride>`. The newtype's
    `TryFrom<u64>` (`:47-56`) rejects values above `MESSAGE_EXPIRY_WINDOW` (604800), and serde
    deserializes through it (`try_from = "u64"`), so a larger value can be neither constructed nor
    parsed.
  - `get_message_expiry_window` (`:87-92`) returns the override when it is above 0; `None` and
    `Some(0)` map to 604800.
  - Both depset sources in the fault proof go through serde: the embedded registry
    `DEPENDENCY_SETS` uses `serde_json::from_str`, and the preimage-oracle fallback in
    `boot.rs:264-286` uses `serde_json::from_slice`. The result reaches `MessageRules` through
    `consolidation.rs:137`.
  - The field is `pub` and the `arbitrary` derive is on, so a `DependencySet` built in memory can
    hold a value above the cap. Since `d36e37862b` the getter falls back to 604800 for it, so such
    a value never becomes the effective window.

**Per-depset overrides are not modelled as such.** The model has one global window: a single
depset and a single cluster. `Valid()` pins `messageExpiryWindow == MESSAGE_EXPIRY_WINDOW`, and the
constant is abstract, so an override is just one particular value of it. "Override 0 means default"
and the parse-time cap are outside the model. The Lean model is more general, with a
per-destination `protocolWindow x`.

## What the existing model covers, and what it does not

**It covers:**
- `ValidExecutingMessage`: chain registration; activation for one full block on both chains (one
  global `activationTimestamp` plus each chain's `BlockTime()`); `init ≤ exec`; and
  `exec ≤ init + W`.
- `VerifyExecutingMessage`: the same checks in Go's order (unknown chain, ExecutedTooEarly,
  InitiatedTooEarly, TimestampViolation, MessageExpired), then the frontier-view or logsDB lookup.
- The full supernode loop: verification, cross-validity of committed timestamps, rewinds and
  pending transitions. Our lemmas use none of this.

**It does not cover:**
- Go's `ErrChainNotInDependencySet` checks. These only add rejections.
- Per-chain activation times. Kona uses them (`lagoon_time` per rollup config). So does Go
  `op-core/interop/depset/links.go`, through `IsInterop(chainID, ts)` and
  `IsInteropActivationBlock(chainID, ts)`. The model and the Go supernode use one global
  `activationTimestamp`.
- The `saturating_sub` genesis edge in kona's `is_first_interop_block`.
- The config-parse cap and override defaulting.
- What the checksum identifies. Checksums are abstract `nat`s in the model.
- The contracts.
- Time monotonicity of the destination chain. We take `exec >= tExport` as a hypothesis. It stands
  for the overall argument's assumption that block timestamps are monotone per chain.

**Trust base of our lemmas.**
- **Main theorems** (`ExpiredAtExportNeverValid`, `…Global`, `ExpiredAtExportRejected`,
  `…WithCap`). They use three things:
  - `ValidExecutingMessage`, a plain ghost predicate;
  - `Valid()`, only for `messageExpiryWindow == MESSAGE_EXPIRY_WINDOW` and
    `chains.Keys == CHAIN_IDS`;
  - the postcondition `valid ==> ValidExecutingMessage(...)` of `VerifyExecutingMessage`, which the
    model verifies against the method body.

  Our proofs of these lemmas use no `{:axiom}`. The model's own proof of `VerifyExecutingMessage`
  runs in a context that contains the model's axioms, but this postcondition follows from the
  sequence of guards alone.
- **`TimestampBoundToInitBlock`** uses:
  - the `LogsDB.Contains` timestamp `{:axiom}`;
  - the `LogsDB.FindSealedBlock` `id.number` `{:axiom}`;
  - the `ChainContainer.BlockInfo` `{:axiom}` (`BlockInfo(b).value.id == b`);
  - the invariant `AllLogsDBsConsistentWithChainData`, which the model proves;
  - the **hypothesis** `IsCorrectFrontierView`. Inside the model it comes from the `{:axiom}`
    ensures of `ResolveFrontierVerificationView` (`Interop.dfy:1889`).
- **`EndToEndWitness`** uses the verified postcondition of the `Interop` constructor. The
  constructor's own proof relies on two axioms and one verified method:
  - the bodiless `LogsDB` constructor, whose `{:axiom}` ensures `LatestSealedBlock() == None`;
  - the `{:axiom}` on `LatestSealedBlock` saying that an empty DB has `FindSealedBlock(n) == None`;
  - `Types.Enumerate`, which the model verifies.
- **The model's three bare `assume`s** (`CheckChainsReady`, `:1019-1020`; `PersistFrontierLogs`,
  `:1631`) are not on any path we use. In reviewer mutation testing, deleting the `LogsDB.Contains`
  axiom broke only `TimestampBoundToInitBlock`.

## Agreement of boundary semantics (task 4)

Here `e` is the initiating timestamp, `t` the executing (relay) timestamp, `W` the protocol window
and `P` the contract period.

| Where | Accepts (valid / relayable) iff | Boundary at t = e + W | init ≤ exec | Activation | Window source |
|---|---|---|---|---|---|
| Dafny `ValidExecutingMessage` (`Interop.dfy:517-529`) | `act+bt(exec) ≤ t ∧ act+bt(init) ≤ e ∧ e ≤ t ≤ e + W` | valid | yes | yes (global act + block time) | abstract `MESSAGE_EXPIRY_WINDOW` |
| Dafny `VerifyExecutingMessage` (`:1748`) | rejects when `e + W < t` | valid | yes (`:1742`) | yes (`:1729`, `:1736`) | same, via field |
| Go supernode `algo.go:218` | rejects when `t − e > W` (after the `e > t` check at `:212`) | valid | yes | yes (`:197`, `:206`), global act | depset, capped |
| Go `op-core/interop/depset/links.go:73` | rejects when `t − e > W` (after `e > t` at `:70`) | valid | yes | yes, per chain (`IsInterop` / `IsInteropActivationBlock`, `:56-69`) | depset, capped |
| kona `rules.rs:111-119` (`check_message_expiry`) | accepts when `t − e ≤ W` (after `check_message_ordering`, `graph.rs:193-195`) | valid | yes | yes, per chain (`lagoon_time`) | depset, capped |
| Lean `Model.lean:211-213` `withinWindow` | `∃ e, events z h e ∧ t ≤ e + protocolWindow x` | valid | **no** | **no** | per-destination `protocolWindow` |
| Quint `expiry.qnt:97` `withinWindow` | `∃ e ∈ evts, t <= e + PROTOCOL_WINDOW` | valid | **no** | **no** | `PROTOCOL_WINDOW`, in days |
| Contract `L2ToL2CrossDomainMessenger.sol:273` | expires iff `t > sentAt + EXPIRY_PERIOD` (reverts on `<=`) | n/a | n/a | n/a | `EXPIRY_PERIOD = 8 days` |

**Verdict.** The boundary agrees exactly everywhere: `t = e + W` is valid and `t = e + W + 1` is
invalid. The contract's strict `>` is the exact complement, so W == P would already be safe. Go
and kona subtract only after the ordering check, so the subtraction cannot underflow, and Dafny
works over `nat`. **There is no discrepancy between the Dafny model and the Go/Rust expiry check,
and no bug candidate.**

**Lean and Quint versus the code.** Lean and Quint omit `init ≤ exec` and the activation checks,
as `../README.md` already documents. Both omissions only add relays.
`DafnyValidImpliesLeanWithinWindow` proves the numerical part of the inclusion against the Dafny
model. Safety proved in Lean or Quint carries over to the code only under the event
correspondence and window identification described above, including Quint's day-to-second
scaling. That correspondence is argued, not proved as a refinement.

**Activation, outside the expiry rule.** Dafny and the Go supernode use one global
`activationTimestamp`. Kona (`lagoon_time`) and Go `links.go` use per-chain activation. This does
not affect the expiry argument, because the theorem needs no activation hypothesis.

## Review log

- **v1** (2026-10-07): the bridge lemmas, the expected-fail file and the agreement table.
- **v1 review** by R1 (fresh-context reviewer) and R2, R3 (independent model-based reviewers).
  - Verdict: sound, no axioms added, not vacuous, boundary agrees everywhere, no critical or high
    findings.
  - R1's mutants were all rejected by Dafny: dropping `exec >= tExport`; a non-strict export
    check; dropping activation; `ensures false`; `assert false` after `new Interop`; the imperative
    form without `W <= P`.
  - Deleting the `LogsDB.Contains` axiom breaks only `TimestampBoundToInitBlock`.
- **v3** (2026-10-08, non-vacuity audit of all layers): `NonVacuity.dfy` (one witness per
  lemma/method), and `run.sh` checks coverage, verifies the witnesses and runs the
  `assert false` vacuity probe. No hypothesis found unsatisfiable; residual hypotheses listed
  under "Non-vacuity". `TimestampBoundToInitBlock`'s accepting case is not exhibited.
- **v2** (2026-10-08) addresses the review:
  1. *Medium, all three reviewers: frontier timestamp binding.* The binding held only against an
     arbitrary view. `TimestampBoundToInitBlock` now requires
     `IsCorrectFrontierView(view, blocksAtTS)`, and both paths conclude equality with chain data
     (`chains[..].BlockInfo(..)`). The trust base lists the frontier axiom (`Interop.dfy:1889`),
     the FindSealedBlock and BlockInfo axioms, and `AllLogsDBsConsistentWithChainData`. The README
     now says the checksum is abstract, so the step to `sentAt` is outside the model.
  2. *Medium, R1: hand-copied guards.* The second `ensures` of `ExpiredAtExportNeverValid` and
     `ExpiryIsTheFailingGuard` are labelled RESTATEMENT, in the source and here. The claim
     "nothing restated" is softened, and the headline's `ErrMessageExpired` claim is qualified by
     the activation hypotheses.
  3. *Medium, R2: model coverage.* `run.sh --model` now passes `--verify-included-files`, so
     all six model files are verified; the new VC count is in Results. The trust base names the
     `LogsDB` constructor and `LatestSealedBlock` axioms.
  4. *Low, all three reviewers:*
     - `ShortPeriodCounterexample` is reworded as temporal eligibility, with a pointer to Lean's
       `cex_periodBelowWindow`.
     - The Lean and Quint carry-over is qualified: it needs event correspondence and window
       identification, and Quint counts in days.
     - `run.sh` now checks the exit status, the error kind and the exact `EXPECT-FAIL` lines, and
       two mutants confirmed it catches mistakes.
     - Non-vacuity is stated as conditional on a non-empty `CHAIN_IDS` and an existing
       `ChainContainer`.
     - Line references are fixed: `Types.dfy:124`; `Interop.dfy:517-529`, `:1742`, `:1748`;
       `algo.go:197`, `:206`, `:212`.
     - `links.go` activation is now described as per-chain.
