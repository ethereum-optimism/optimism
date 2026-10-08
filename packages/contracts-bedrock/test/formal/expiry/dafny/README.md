# Dafny: the off-chain half of the expiry argument, on the supernode's own model

**What this proves.** The repo already has a Dafny model of op-supernode's interop validity
(`op-supernode/dafny-models/`). `ExpiryBridge.dfy` `include`s that model and proves, using the
model's own definitions:

> Let P be the contract's expiry period (`L2ToL2CrossDomainMessenger.EXPIRY_PERIOD`, 8 days =
> 691200 s) and W the protocol window (`messageExpiryWindow` / `MESSAGE_EXPIRY_WINDOW`), with
> W ≤ P. Suppose the destination exported the message as undelivered at block timestamp
> tExport > initTimestamp + P, which is exactly when `expireMessage` accepts the fact. Then no
> executing timestamp exec ≥ tExport makes the message valid. That is,
> `!ValidExecutingMessage(exec, …)`, and the model's imperative `VerifyExecutingMessage` returns
> `false`, from its `ErrMessageExpired` branch.

The definitions used are:
- `Interop.ValidExecutingMessage`: the declarative predicate.
- `Interop.VerifyExecutingMessage`: the imperative check.
- The `Interop.messageExpiryWindow` field and the `Types.MESSAGE_EXPIRY_WINDOW` constant, which
  `Interop.Valid()` ties together.

Nothing is restated. The file adds no axioms, no `{:axiom}`, no `assume` and no bodiless lemmas.

The on-chain half is proved in `../lean`, `../kontrol` and `../halmos`. It says `expireMessage`
accepts only `t > sentAt + P`, and that the fact's `t` is the destination's `block.timestamp` at
export, taken while the message was unrelayed there. This directory supplies the step "after that
moment the destination can never validly relay it", against the supernode model rather than our
own `t ≤ e + W` abstraction.

## Dafny version

The repo pins no Dafny version. There is no CI job, no `mise.toml` entry and no justfile recipe,
and `dafny-spec-check.md` names no version. The PR that added the model (#22507) says "the Dafny
CLI is not installed in the local environment, so the model was not re-verified locally". We
therefore use the latest release:

- **Dafny 4.11.0** (`4.11.0+fcb2042d6d043a2634f0854338c08feeaaaf4ae2`), with the bundled
  **Z3 4.12.1**.
- Release zip `dafny-4.11.0-x64-ubuntu-22.04.zip` (sha256
  `a46a9ff7cdd720f7955854c78e95df13f4cfe6b80691b05f8654fe19e8267179`). It is self-contained, so
  no .NET install is needed. On hel1 it lives at `~/tools/dafny-4.11.0/dafny/dafny`.

```sh
curl -sSLo dafny.zip https://github.com/dafny-lang/dafny/releases/download/v4.11.0/dafny-4.11.0-x64-ubuntu-22.04.zip
unzip -q dafny.zip -d dafny-4.11.0
export DAFNY=$PWD/dafny-4.11.0/dafny/dafny
```

## How to run

```sh
DAFNY=/path/to/dafny ./run.sh            # ExpiryBridge.dfy must verify; ExpectFail.dfy must fail with exactly 2 errors
DAFNY=/path/to/dafny ./run.sh --model    # also re-verify the whole existing model first
```

On the shared hel1 box, run under a memory cap:
`systemd-run --user --scope -p MemoryMax=16G -p MemorySwapMax=0 ./run.sh --model`. Peak RSS for
the whole model was about 1.5 GB.

`run.sh` passes `--allow-warnings` because the included model has three `assume` statements
without `{:axiom}` (`Interop.dfy:1019`, `:1020`, `:1631`). Dafny warns on them and otherwise exits
2 even when verification succeeds. None of the three is in any definition this file uses (see the
trust base below). Dafny does not re-verify included files by default, so `run.sh` checks only our
files unless `--model` is given.

## Results (hel1, branch tip `5992028e08`, Dafny 4.11.0)

| Check | Result | Wall time |
|---|---|---|
| `ExpiryBridge.dfy` | `28 verified, 0 errors` | about 5 s (10 s for all of `run.sh`) |
| `ExpiryBridge.dfy` with `--warn-contradictory-assumptions` | `28 verified, 0 errors`, no contradictory-assumption warnings | about 5 s |
| `ExpectFail.dfy` (must fail) | `2 verified, 2 errors`: both lemmas rejected, as intended | about 4 s |
| Existing model, `Interop.dfy` with `--cores 8 --verification-time-limit 1200` (task 1) | **`7313 verified, 0 errors`**, exit 0 | 10 min 19 s wall, 1118 s CPU, peak RSS 1.5 GB |
| Existing model with default limits, `--cores 16`, first attempt at load average ~330 | Internal error "The operation has timed out" on `ApplyPendingTransition` (`Interop.dfy:1054`), plus Z3 pipe "Prover error" lines | 10 min 01 s |

hel1 was heavily loaded by other jobs during these runs (load average 35 to 330 on 32 cores), so
the times are upper bounds. **Task 1 verdict:** the existing model verifies as-is with Dafny 4.11.0.
The only messages are the three "assume statement has no {:axiom} annotation" warnings. The first
attempt failed only because of the time limit under extreme host load, not because of a proof
failure: the same file passes with a larger per-VC time limit. `run.sh --model` uses that limit.

## Exact statements (`ExpiryBridge.dfy`)

In every statement below, `i: Interop.Interop` and `msg: ExecutingMessage`. `msg.timestamp` is
the initiating timestamp, `exec` the executing block timestamp and `execChain` the destination.

**Main theorem, field form.** This needs only the two `requires` of `ValidExecutingMessage`
itself, and not `Valid()`:
```dafny
lemma ExpiredAtExportNeverValid(i, P, tExport, exec, execChain, msg)
  requires execChain in CHAIN_IDS && i.chains.Keys == CHAIN_IDS
  requires i.messageExpiryWindow <= P
  requires tExport > msg.timestamp + P
  requires exec >= tExport
  ensures !i.ValidExecutingMessage(exec, execChain, msg)
  ensures msg.timestamp + i.messageExpiryWindow < exec   // the ErrMessageExpired guard is true
```

**Main theorem, global form.** `ExpiredAtExportNeverValidGlobal` is the same statement under
`i.Valid()` and `MESSAGE_EXPIRY_WINDOW <= P`.

**Imperative form.** It calls the model's method, and the conclusion follows from that method's
own verified postcondition `valid ==> ValidExecutingMessage(...)`:
```dafny
method ExpiredAtExportRejected(i, view, P, tExport, exec, execChain, msg) returns (ok: bool)
  requires i.Valid() && execChain in CHAIN_IDS
  requires MESSAGE_EXPIRY_WINDOW <= P
  requires tExport > msg.timestamp + P
  requires exec >= tExport
  ensures !ok
{ ok := i.VerifyExecutingMessage(execChain, exec, msg, view); ... }
```
`ExpiredAtExportRejectedWithCap` instantiates `P := EXPIRY_PERIOD` (691200) and replaces the side
condition with the cap `MESSAGE_EXPIRY_WINDOW <= 604800`.

**Which branch fails.** `ExpiryIsTheFailingGuard` covers the case where the source chain is
registered, both activation checks pass and init ≤ exec. There, every guard before the expiry
check in `VerifyExecutingMessage`'s body evaluates to false, and the expiry guard
`msg.timestamp + messageExpiryWindow < exec` to true. So the method returns from the
`ErrMessageExpired` branch. The model's result is a `bool` with no error code, so it cannot state
this more directly. In the other cases the method returns `false` earlier, for another reason,
which is still a rejection.

**Timestamp binding.** `TimestampBoundToInitBlock` shows that when `VerifyExecutingMessage`
accepts, `msg.timestamp` equals the timestamp of the real initiating block: the sealed logsDB
block, or the frontier block if same-timestamp. So an executing message cannot stretch its window
by claiming a later initiating timestamp. This is what makes the identifier's timestamp equal the
contract's `sentAt`: the `SentMessage` log and `sentMessageTimestamps[H] = block.timestamp` are in
the same source block. The proof uses the model's `LogsDB.Contains` `{:axiom}` ensures (timestamp
equality). Go (`raftwallogdb/db.go` `Contains`: `rec.Timestamp != query.Timestamp` gives
`ErrConflict`) and kona (`graph.rs`: `remote_header.timestamp != initiating_timestamp` gives
`InvalidMessageTimestamp`) both implement this check.

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
  which ensures `Valid()`. So the `Valid()` hypotheses are satisfiable. On that instance it builds a
  concrete message that is valid at init + W, and runs `ExpiredAtExportRejectedWithCap` on it at
  `tExport = init + EXPIRY_PERIOD + 1`. Its hypotheses are three: `CHAIN_IDS` is non-empty (it is
  abstract), some `ChainContainer` object exists (the model's class has no constructor, so it is
  taken as a parameter), and the cap.

**Counterexample when P < W.**
```dafny
lemma ShortPeriodCounterexample(i, P, execChain, msg) returns (tExport: nat, exec: nat)
  requires (same activation hypotheses as BoundaryStillValid) && P < MESSAGE_EXPIRY_WINDOW
  ensures tExport > msg.timestamp + P            // expireMessage accepts the fact
  ensures exec >= tExport                        // the relay comes after the export
  ensures i.ValidExecutingMessage(exec, execChain, msg)   // ...and is still valid: double spend
```
The witnesses are `tExport = init + P + 1` and `exec = init + W`.

**Expected failures (`ExpectFail.dfy`).** `run.sh` requires exactly these two errors:
- `NoPeriodAssumption` is the main theorem without `W <= P`. It fails.
- `ExclusiveBoundary` claims the message is invalid at exec == init + W. It fails.

**The cap.** `CapImpliesWindowWithinPeriod` proves
`MESSAGE_EXPIRY_WINDOW <= 604800 ==> MESSAGE_EXPIRY_WINDOW <= 691200 && MESSAGE_EXPIRY_WINDOW + 86400 <= 691200`.

**Link to the Lean and Quint relay rule.**
- `DafnyValidImpliesLeanWithinWindow`: `ValidExecutingMessage(exec, …)` implies
  `exec <= init + messageExpiryWindow`. So the Dafny valid set is contained in the Lean and Quint
  `t ≤ e + W` set.
- `ValidIffWindow` states both directions. When both activation checks pass, Dafny validity is
  exactly `init <= exec <= init + W`.
- `LeanAcceptsFutureInitDafnyRejects` shows the containment is strict: exec = init − 1 passes
  `t ≤ e + W` but is invalid in Dafny.

The theorem holds with W == P, so the boundaries line up exactly. The 1-day margin (P = W + 1 day)
is slack on top.

## The 7-day cap and per-depset overrides (task 3)

- **`MESSAGE_EXPIRY_WINDOW` is abstract.** `Types.dfy:121` declares `const MESSAGE_EXPIRY_WINDOW: nat`
  with no value. Its comment says "Corresponds to defaultMessageExpiryWindow in algo.go", and the
  constructor copies it into the `messageExpiryWindow` field. Every theorem about the model
  therefore holds for every W, which includes every override value.
- **The cap is a hypothesis, not an axiom.** It appears as `requires MESSAGE_EXPIRY_WINDOW <= PROTOCOL_WINDOW_CAP`
  in `ExpiredAtExportRejectedWithCap`, `CapImpliesWindowWithinPeriod` and `EndToEndWitness`. The
  model cannot derive it, because the cap lives in config parsing, which the model does not
  describe. The Go and kona code enforce it, giving an effective W in [1, 604800]:
  - **Go:** `op-core/interop/depset/static_depset.go`. `hydrate` (`:141-142`) rejects
    `overrideMessageExpiryWindow > MessageExpiryTimeSecondsInterop (604800)`. Every constructor
    and decoder calls `hydrate`: `NewStaticConfigDependencySet` (`:33`),
    `NewStaticConfigDependencySetWithMessageExpiryOverride` (`:43`), the JSON decoder (`:87-88`)
    and the TOML decoder (`:132-133`).
    `MessageExpiryWindow()` (`:171-175`) maps an override of 0 to 604800. The supernode then copies
    this value into `Interop.messageExpiryWindow` (`interop.go:280-282`), or uses
    `defaultMessageExpiryWindow = 604800` when there is no depset.
  - **kona:** `rust/kona/crates/protocol/genesis/src/interop/depset.rs`.
    `deserialize_override_window` (`:44-45`) rejects `> MESSAGE_EXPIRY_WINDOW (604800)`, and
    `get_message_expiry_window` (`:52-56`) maps `None` or `Some(0)` to 604800. Both depset sources
    in the fault proof go through serde: the embedded registry `DEPENDENCY_SETS` uses
    `serde_json::from_str`, and the preimage-oracle fallback in `boot.rs:264-286` uses
    `serde_json::from_slice`. The result reaches `MessageRules` through `consolidation.rs:137`.
    Note that the field is `pub` and the `arbitrary` derive is enabled, so Rust code that builds a
    `DependencySet` in memory bypasses the cap. Only tests and fuzzing do that; no production
    loading path does.
- **Per-depset overrides are not modelled as such.** The model has one global window (a single
  depset and a single cluster). `Valid()` pins `messageExpiryWindow == MESSAGE_EXPIRY_WINDOW`, and
  the constant is abstract, so an override is just one particular value of it. "Override 0 means
  default" and the parse-time cap are outside the model. The Lean model is more general, with a
  per-destination `protocolWindow x`.

## What the existing model covers, and what it does not

**It covers:**
- `ValidExecutingMessage`: chain registration, activation for one full block on both chains
  (one global `activationTimestamp` plus each chain's `BlockTime()`), `init ≤ exec`, and
  `exec ≤ init + W`.
- `VerifyExecutingMessage`: the same checks in Go's order (unknown chain → ExecutedTooEarly →
  InitiatedTooEarly → TimestampViolation → MessageExpired), then frontier-view or logsDB lookup.
- The full supernode loop: verify, cross-validity of committed timestamps, rewinds and pending
  transitions. Our lemmas use none of this.

**It does not cover:**
- Go's `ErrChainNotInDependencySet` checks. These only add rejections.
- Per-chain activation times, which kona uses (`lagoon_time` per rollup config) and the model does
  not.
- Kona's `saturating_sub` genesis edge in `is_first_interop_block`.
- The config-parse cap and override defaulting.
- The contracts.
- Time monotonicity of the destination chain. We take `exec >= tExport` as a hypothesis, which is
  the "block timestamps are monotone per chain" assumption of the overall argument.

**Trust base of our lemmas.** The main theorems use only these definitions:
- `ValidExecutingMessage`, a plain ghost predicate.
- `Valid()`, used only for `messageExpiryWindow == MESSAGE_EXPIRY_WINDOW` and
  `chains.Keys == CHAIN_IDS`.
- `VerifyExecutingMessage`'s postcondition, which the model verifies against the method body. The
  body calls no method with `{:axiom}` contracts on the path to `valid := false`.

`TimestampBoundToInitBlock` additionally uses the `LogsDB.Contains` timestamp `{:axiom}`.
`EndToEndWitness` uses the `Interop` constructor's verified postcondition. The model's three bare
`assume`s, in `CheckChainsReady` (`:1019-1020`) and `PersistFrontierLogs` (`:1631`), are not on any path we use.

## Agreement of boundary semantics (task 4)

Here `e` is the initiating timestamp, `t` the executing (relay) timestamp, `W` the protocol window
and `P` the contract period.

| Where | Accepts (valid / relayable) iff | Boundary at t = e + W | init ≤ exec | Activation | Window source |
|---|---|---|---|---|---|
| Dafny `ValidExecutingMessage` (`Interop.dfy:516-529`) | `act+bt(exec) ≤ t ∧ act+bt(init) ≤ e ∧ e ≤ t ≤ e + W` | valid | yes | yes (global act + block time) | abstract `MESSAGE_EXPIRY_WINDOW` |
| Dafny `VerifyExecutingMessage` (`:1747`) | rejects when `e + W < t` | valid | yes (`:1741`) | yes | same, via field |
| Go supernode `algo.go:218` | rejects when `t − e > W` (after `e > t` check at `:211`) | valid | yes | yes (`:196`, `:205`) | depset, capped |
| Go `op-core/interop/depset/links.go:73` | rejects when `t − e > W` (after `e > t` at `:70`) | valid | yes | yes (IsInterop / not activation block) | depset, capped |
| kona `rules.rs:111-119` (`check_message_expiry`) | accepts when `t − e ≤ W` (after `check_message_ordering`, `graph.rs:193-195`) | valid | yes | yes (per-chain `lagoon_time`) | depset, capped |
| Lean `Model.lean:211-213` `withinWindow` | `∃ e, events z h e ∧ t ≤ e + protocolWindow x` | valid | **no** | **no** | per-destination `protocolWindow` |
| Quint `expiry.qnt:97` `withinWindow` | `∃ e ∈ evts, t <= e + PROTOCOL_WINDOW` | valid | **no** | **no** | `PROTOCOL_WINDOW` |
| Contract `L2ToL2CrossDomainMessenger.sol:273` | expires iff `t > sentAt + EXPIRY_PERIOD` (reverts on `<=`) | n/a | n/a | n/a | `EXPIRY_PERIOD = 8 days` |

**Verdict.** The boundary agrees exactly everywhere: `t = e + W` is valid and `t = e + W + 1` is
invalid. The contract's strict `>` is its exact complement, so W == P would already be safe.
Go's and kona's subtraction runs only after the ordering check, so it cannot underflow, and Dafny
works over `nat`. **No discrepancy between the Dafny model and the Go/Rust expiry check. No bug
candidate.**

Lean and Quint differ from the code in two ways, as already documented in `../README.md`. They
omit `init ≤ exec` and the activation checks. Both omissions only add relays, so the Lean and Quint
relay set over-approximates the code's. `DafnyValidImpliesLeanWithinWindow` proves this inclusion
against the Dafny model, so safety proved there carries over.

The one modelling difference is outside the expiry rule. Activation is a global
`activationTimestamp` in Dafny and Go supernode, but per-chain `lagoon_time` in kona. It does not
affect the expiry argument, whose theorem needs no activation hypothesis.
