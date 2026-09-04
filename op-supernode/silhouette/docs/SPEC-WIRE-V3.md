# Silhouette proof-batch wire format v3 (consolidation edition)

**Status: NORMATIVE. This is the version this branch implements**, in Go
(`op-supernode/proofbatch`), against the fixture corpus that is the format's real pin. It has not
been posted to L1: the cluster that ran v2 was decommissioned before the rotation to v3.

> References below to `PLAN.md` and `g-decisions.md` are to the Silhouette project's design record,
> which is not in this repository; `README.md` says where it and the shelf branches are. Everything
> normative is restated here.


**Lineage.** v1 → `keccak-cove/SPEC-PROOF-BATCH.md` (retained there). v2 → the same file's v2 header
block plus decisions D1–D11, shipped and live. v3 → this file, which is now the authoritative
document for the wire; the Cove spec stays as the pre-rotation record. Decisions D1–D11 carry over
**unchanged** unless restated below; D12–D22 are new.

**The fixtures are canonical, not any implementation.** The bytes are the contract: an implementation
is correct when it decodes and encodes the corpus byte-identically, and that is how the two sides
were kept honest about each other rather than by reading each other's code. On this branch the
implementation is **Go** (proven-node adapter + blob submitter, `op-supernode`); the prover-side
codec is checked against this same corpus on the shelf branch named in `README.md` §"What is
deliberately not here".

Fixture home: `op-supernode/proofbatch/testdata/proof-batch` (v3), with the frozen v2 tree retained
beside it at `testdata/proof-batch-v2` (see D22). The corpus lives with the Go codec because a wire
record is language-agnostic and has to survive either side being absent — which one now is.

---

## 1. What v3 changes, and why

**One field.** `BlockExport` gains `execMsgs[]` — the **executing messages** each block consumed,
as a deduplicated, canonically ordered **set** of `{Identifier, msgHash}` pairs.

The reason is G7's thesis (PLAN.md): before v3, P's cross-chain *imports* were invisible on the
wire, so the public network had to take P's own executing messages at face value (the retired
ruling-7 posture: "execMsgs empty / proof-trusted"). That made P a special case for the judge and
made the settlement superroot *composed* rather than *consolidated* (G6 D3). With `execMsgs[]` on
the wire, P's frontier view becomes exactly a driven chain's — the supervisor validates P's
dependencies with the stock checksum/hazard/expiry/fixpoint machinery, reading the import list from
the wire instead of from receipts it cannot have — and the settlement circuit can check P's imports
against A's derived receipts in memory.

The proof's claim strengthens from "P's STF is correct" to the **conditional-validity** form of §7.

Nothing else changes. No field is removed, no field's meaning moves, and the export-policy labels
keep their v2 spelling (D19).

---

## 2. Envelope

```
magic     = "KCPB"            (4 bytes, ASCII)
version   = 0x03              (u8)
pv_len    = u32 big-endian    (length of public_values)
public_values                 (ABI-encoded, see §3)
proof_len = u32 big-endian    (0 = an empty proof slot)
proof                         (proof bytes; empty under an attested proving system)
```

Magic is unchanged: `KCPB` is the object's identity across the Cove→Silhouette rename, and rotating
it would buy nothing but a fixture churn.

### D12 — the version gate stays symmetric, and now refuses 0x02 as well

Anything other than `0x03` MUST be refused outright, never best-effort decoded — backwards *or*
forwards. This is the same discipline v2 applied to v1, applied to v2:

* `0x01` — retired (v1 `BlockExport` is a different ABI type).
* `0x02` — retired. **A v2 envelope is not a truncated v3 envelope**: its `BlockExport` has six
  fields where v3 has seven, so ABI-decoding v2 bytes as v3 either fails or, worse, succeeds with
  `logs` read out of the wrong offset. Forwards-leniency is exactly how a field addition gets
  silently misread.
* `0x04` and above — refused as unknown. Refusing an unrecognised *higher* version is what makes a
  future v4 safe to introduce.

Fixtures: `wrong-version-0x01`, `retired-version-0x02`, `unknown-version-0x04`, `bad-version`
(`0xff`). Implementations SHOULD name `0x01` and `0x02` as *retired* rather than merely wrong, so the
rejection reads as deliberate in a log.

The retained v2 fixture corpus doubles as a live test of this gate (D22).

---

## 3. public_values

Exactly what the proving program commits (the ABI encoding, committed whole). One tuple in
offset-prefixed `abi.encode` form (D5 — the first word is `0x20`).

```solidity
struct ProofBatch {
    bytes32 prevOutputRoot;      // output root this batch extends
    bytes32 newOutputRoot;       // output root after the last derived block in range
    bytes32 l1Head;              // L1 head the in-circuit derivation ran against
    bytes32 rollupConfigHash;    // hash of P's rollup config as parsed by the prover
    bytes32 depSetHash;          // commitment to the dependency set
    bytes32 exportPolicyHash;    // commitment to the log-export filter
    BlockExport[] blocks;        // EVERY block in range, in order, no gaps
}

struct BlockExport {
    uint64    blockNumber;
    uint64    timestamp;
    bytes32   blockHash;
    bytes32   stateRoot;
    bytes32   messagePasserStorageRoot;
    LogExport[] logs;            // exported initiating messages  (EXPORTS)
    ExecMsg[]   execMsgs;        // consumed executing messages   (IMPORTS)  ← NEW IN v3
}

struct LogExport {
    uint32  logIndex;
    bytes32 logHash;
    bytes   preimage;            // empty == absent (D9)
}

struct ExecMsg {
    Identifier id;
    bytes32    msgHash;          // the message PAYLOAD hash — see D15
}

struct Identifier {
    address origin;
    uint64  blockNumber;
    uint32  logIndex;
    uint64  timestamp;
    uint256 chainId;
}
```

`ExecMsg` is a **static** struct (six words, 192 bytes), so `ExecMsg[]` encodes as a length word
followed by the elements packed inline — no per-element offset words.

An **empty `execMsgs`** is a zero-length array, always present in the tuple, never elided: an offset
word plus a zero length word. `decode` → re-encode is byte-identical, and every fixture asserts it.
Same rule v2 fixed for an absent preimage.

### D13 — the field order is stock `Identifier`'s; the widths are the checksum's own

`Identifier`'s **field order** is verbatim the stock Solidity struct
(`packages/contracts-bedrock/src/L2/CrossL2Inbox.sol:13`): `origin, blockNumber, logIndex,
timestamp, chainId`. Do not reorder it to match the checksum's packing, which transposes `logIndex`
and `timestamp` (§6) — that transposition is a documented trap, not a convention to spread.

The **widths** are narrower than Solidity's (`uint256` for the middle three) and this is
provably lossless rather than a judgement call. `validateMessage` computes the checksum
unconditionally before emitting, and `calculateChecksum` reverts
`BlockNumberTooHigh` / `LogIndexTooHigh` / `TimestampTooHigh` above `2^64`, `2^32`, `2^64`
respectively (`CrossL2Inbox.sol:95-97`). **A message whose fields do not fit these widths cannot have
been validated, so no `ExecutingMessage` event carrying one can exist.** The wire therefore carries
the same widths the chain's own validation used. `chainId` stays `uint256`: nothing narrows it.

Three further reasons this is the right side of the trade:

1. ABI-encoding cost is identical either way — every field occupies a full word regardless.
2. It matches the Go domain type (`op-core/interop/messages.Identifier`: `common.Address`,
   `uint64`, `uint32`, `uint64`, `eth.ChainID`), so the Go mirror decodes into its own struct with no
   narrowing step of its own to get wrong.
3. It matches `LogExport.logIndex: uint32` and `BlockExport.blockNumber/timestamp: uint64`, so the
   wire is internally consistent about what a block coordinate is.

The narrowing MUST still be **checked, never truncating** (D16): a reader that takes the event's
words as 256-bit values and narrows them MUST hard-error on any value that does not fit. On a stock
predeploy the check never fires; it is there because the alternative to a check is a silent
collision.

---

## 4. The extraction rule (normative, soundness-critical)

`execMsgs[b]` is a function of block `b`'s **own receipts**, and of nothing else. This section is
the definition of that function. It is soundness-critical in the *completeness* direction: the
conditional-validity claim of §7 says the set is **exactly** what the block assumed, so a rule that
can skip an entry is a hole.

### D14 — the filter

For a block `b`, walk `b`'s receipts in ascending transaction index, and within each receipt walk its
logs in order. A log is an **executing-message log** iff both:

1. `log.address == 0x4200000000000000000000000000000000000022` (the `CrossL2Inbox` predeploy —
   `Predeploys::CROSS_L2_INBOX`); **and**
2. `log.topics[0] == 0x5c37832d2e8d10e346e55ad62071a6a2f9fa5130614ef2ec6617555c6f467ba7`, i.e.
   `keccak256("ExecutingMessage(bytes32,(address,uint256,uint256,uint256,uint256))")`.

Every executing-message log contributes one `ExecMsg` (before dedup). Nothing else contributes.

**No receipt-status filter is applied, and none may be added.** By EVM construction a receipt
contains only the logs of execution that took effect — a reverted transaction's logs, and the logs
of a reverted sub-call inside a successful transaction, are discarded before the receipt exists. So
"present in a receipt" already means "was assumed". Special-casing status would be at best a no-op
and at worst a way to drop a real import.

### D15 — the decode is STRICT, and a malformed log is an ABORT, not a skip

Given an executing-message log, decode:

```
topics.len() == 2                                  else ABORT
data.len()   == 160                                else ABORT
msgHash      = topics[1]
w0..w4       = data as five 32-byte words
w0[0..12]  == 0  →  origin      = w0[12..32]       else ABORT
w1[0..24]  == 0  →  blockNumber = w1[24..32] (BE)  else ABORT
w2[0..28]  == 0  →  logIndex    = w2[28..32] (BE)  else ABORT
w3[0..24]  == 0  →  timestamp   = w3[24..32] (BE)  else ABORT
chainId    = w4 (full 256 bits)
```

Two normative points, both departures from what the nearest stock Rust helper does:

* **Strict, not lax.** `kona_interop::parse_log_to_executing_message`
  (`rust/kona/crates/protocol/interop/src/message.rs:158`) filters on address and
  `topics.len() == 2`, then uses the **non-validating** `decode_log_data`, reads the middle fields as
  `U256`, and lets callers `saturating_to()` them later. Go's
  `messages.Message.DecodeEvent` (`op-core/interop/messages/messages.go:73`) is strict: exact
  `32*5` length, explicit zero-padding checks, and an error on `logIndex > MaxUint32`. **v3 pins the
  Go semantics on both sides.** A lax producer paired with a strict verifier is a cross-language
  divergence with a proof on one side of it; and since Solidity's `emit` always produces clean
  padding and `calculateChecksum` bounds the fields, strictness rejects nothing a stock chain can
  produce.
* **Abort, not skip.** A log at the `CrossL2Inbox` address with the `ExecutingMessage` topic that
  does **not** decode strictly MUST make the producer fail — refuse to emit the batch — never be
  silently omitted from the set. Skipping is a completeness hole with teeth: a non-stock predeploy
  emitting a near-miss event shape would consume a cross-chain fact that never reaches the wire, and
  the resulting proof would claim conditional validity over an import list missing an entry.

**`msgHash` is the message PAYLOAD hash** — `keccak256(topic₀ ‖ … ‖ topicₙ ‖ data)` of the
*initiating* log, address-free and position-free. It is `kona-interop`'s `payloadHash`
(`RawMessagePayload`, `message.rs:50`), the value `MessageGraph` compares at `graph.rs:234`. It is
**not** the address-salted `logHash` and **not** the `0x03`-tagged checksum; §6 relates all three.

### D16 — the completeness assumption, stated

Extraction is complete only if P's `CrossL2Inbox` is the stock predeploy: a modified one could
validate a message and emit nothing. That is a property of P's genesis state and its upgrade
history, bound by P's state root — which chains back through `prevOutputRoot` to the anchor — and
**not** by anything on the wire. The spec states it rather than implying it.

This is the *same* assumption a driven chain's supervisor makes when it derives executing messages
from receipts with the same filter. That parity is the point: after v3, P's import surface rests on
exactly the assumptions a public chain's does, which is what "a standard chain to the judge" means.

Corollary worth recording: `validateMessage` reverts `CrossL2Inbox_NoExecutingDeposits` when
`block.basefee > 0 && tx.gasprice == 0`, so a deposit-type transaction cannot execute a message.
This is independent of, and consistent with, the no-deposit soundness rule (DR-2).

---

## 5. The canonical set: ordering and dedup

### D17 — `execMsgs[]` is a set, ordered by its own content, and the codec enforces it

Let the **key** of an `ExecMsg` be the 192-byte ABI encoding of its six fields in declaration
order — each field in one 32-byte big-endian word, `origin` left-padded:

```
key = origin(32, left-padded) ‖ blockNumber(32) ‖ logIndex(32) ‖ timestamp(32) ‖ chainId(32) ‖ msgHash(32)
```

Then, within each block:

* **Order:** `execMsgs[]` is **strictly increasing** in `key`, compared as unsigned lexicographic
  bytes.
* **Dedup:** because the order is *strict*, at most one entry per distinct key can appear. Producing
  the array is therefore exactly: extract → sort by key → drop adjacent equals.

Three properties make this the right rule:

1. **It is byte-identical across implementations by construction.** Lexicographic order over the
   ABI words coincides with field-by-field big-endian unsigned comparison in declaration order, so
   "sort the bytes" and "sort the fields" are the same instruction. There is no second list of
   sort columns to keep in sync, and the key is a serialization both languages already have.
2. **It is enforceable at decode**, like D8's log indices. "Deduplicated ordered set" is a
   *structural* property of the encoding, not a promise about the producer. A verifier rejects a
   duplicated or misordered array without needing the proof.
3. **It carries no executing-transaction information.** Appearance order in the receipts would leak
   the relative order of the executing transactions and, with multiplicity, how many times each
   message was consumed. Content-derived order and full dedup destroy both. See §8.

**Dedup is on the FULL key, never on `(chainId, blockNumber, logIndex)`.** Two entries agreeing on
the coordinate triple but differing in `origin`, `timestamp` or `msgHash` are two *contradictory*
claims about the same log, and at most one of them can validate. Both were assumed, so both must be
committed: a partial-key dedup would silently drop the assumption a consolidator exists to reject.
Keeping them is what makes the contradiction detectable, and the consolidation of §7 aborts on it.

Fixture cases: `exec-msgs-empty`, `exec-msgs-single`, `exec-msgs-multi`, `exec-msgs-dedup`,
`exec-msgs-ordering`, `exec-msgs-unsorted` (refusal), `exec-msgs-duplicate` (refusal),
`exec-msgs-same-identifier-different-msghash` (valid — two entries, both kept).

---

## 6. The three hashes, and why the wire already had the one that matters

The single most useful fact about `CrossL2Inbox.calculateChecksum` is that it computes wire v2's
`LogExport.logHash` as an intermediate value. The ladder:

```
payload     = topic₀ ‖ topic₁ ‖ … ‖ data              (32 bytes per topic, data verbatim)
msgHash     = keccak256(payload)                       ← ExecMsg.msgHash  (payloadHash)
logHash     = keccak256(origin(20) ‖ msgHash(32))      ← LogExport.logHash  (LogToLogHash)
idPacked    = uint96(0) ‖ uint64 blockNumber ‖ uint64 timestamp ‖ uint32 logIndex
checksum    = ( keccak256( keccak256(logHash ‖ idPacked) ‖ uint256 chainId ) & ~(0xff<<248) ) | (0x03<<248)
```

Sources: `CrossL2Inbox.sol:94-118` (Solidity), `op-core/interop/messages/messages.go:125`
(`ChecksumArgs.Checksum`), `op-core/interop/messages/logs.go:53` (`LogToLogHash`), and
`crates/proof-batch/src/log_hash.rs` (the Rust mirror, whose cross-language pin is
`0x4e1dc08f…1b73f`).

**Trap, recorded because it is easy to get wrong in exactly one of two implementations:** `idPacked`
orders the fields `blockNumber, timestamp, logIndex` — transposing the last two relative to the
`Identifier` struct order used everywhere else.

### D18 — `logHash` is the conjunction of the origin check and the payload-hash check

Because `logHash = keccak256(origin ‖ msgHash)`, comparing one 32-byte value discharges *both*
stock per-message checks at once: `InvalidMessageOrigin` (`graph.rs:226`) and `InvalidMessageHash`
(`graph.rs:236`). Given an `ExecMsg`, `keccak256(id.origin ‖ msgHash)` is directly comparable to the
initiating chain's exported `LogExport.logHash` at that coordinate.

**Consequence: full in-circuit consolidation needs no preimages and no MPT.** The default
`AllHashes` export policy is sufficient in both directions (§7). This is why v3 adds one field
rather than two — the hash that binds origin to payload was already on the wire, put there by v2 for
an unrelated reason (LogsDB sealing).

### Live cross-language vector (verified against the deployed predeploy)

From the bridge demo's chain A → chain P deposit, reproduced from the real
`ExecutingMessage` log in chain P block 14467 (tx index 1, log index 0):

| field | value |
|---|---|
| `origin` | `0x4200000000000000000000000000000000000023` |
| `blockNumber` | `201796` |
| `logIndex` | `1` |
| `timestamp` | `1787665760` |
| `chainId` | `424246` |
| `msgHash` | `0x1c8f5a3f7a34098df1428809b8b23aeeb5d141c14498627fa3307652031341bb` |
| → `logHash` | `0xb8e1afa43e9a5d425320338e0cf04d18002f9cda1d33804af367919b4bc3935e` |
| → `checksum` | `0x03629159a5a565da57ee9593fe23dee3c2ed041b50d72cbe79c1566d18b04922` |

The checksum matches the value the demo obtained from `CrossL2Inbox.calculateChecksum` **called on
chain A**, so this vector pins the Rust implementation against the live contract, not against
another copy of the same arithmetic. It is a fixture (`exec-msg-checksum-vectors.json`).

---

## 7. What the proof now claims

### D19 — the conditional-validity statement

For every block `b` in `blocks[]`:

> P's state transition for `b` is correct, **and** the only cross-chain facts `b` assumed are
> exactly the elements of `execMsgs[b]` — each treated as an *assumption* that the named log exists
> on the named chain at the named coordinate with the named payload hash.

Both halves are load-bearing, in opposite directions:

* **Soundness** of each element: an element that does not correspond to a real log makes the
  block's validity condition false, and the consumer (judge or settlement circuit) must reject.
* **Completeness** of the set: "exactly" is what makes the claim conditional rather than merely
  consistent. It rests on D14's filter being total over the block's receipts and on D15's
  abort-not-skip, plus D16's stock-predeploy assumption.

The `exportPolicyHash` commitment covers the **export** filter (`logs[]`) only. The **import**
extraction has no policy and no commitment: it is not a choice. There is exactly one legal
`execMsgs[]` for a given block, defined by D14/D15/D17, and the proving program is what binds it. Adding a
policy knob here would be adding a way to export fewer imports than were assumed — the one thing
the field exists to prevent. The v2 policy labels are therefore unchanged (`cove-export-v2:*`),
which also keeps the configured `exportPolicyHash` stable across the rotation.

### D20 — the consolidation kernel, and the check order

Both discharge sites — the node-side supervisor and the settlement circuit — reduce to one
operation, applied per `ExecMsg`:

> **Resolve** `(chainId, blockNumber, logIndex)` to an initiating log; require
> `keccak256(origin ‖ msgHash)` to equal that log's `logHash`, and the initiating block's timestamp
> to equal `id.timestamp`.

with the *source* of the initiating log differing by direction:

| direction | executing side | initiating side, resolved from |
|---|---|---|
| **P imports from A** | `execMsgs[]` on the wire | A's derived receipts, in memory |
| **A imports from P** | A's derived receipts | P's wire-exported `logs[]` (`logHash` at `logIndex`) |
| **P imports from P** | `execMsgs[]` on the wire | P's own wire-exported `logs[]` |

`logIndex` is the **block-global** log index — the log's position among every log the block emitted,
in `(transaction index, log index)` order. Same convention as `LogExport.logIndex` (D8) and as
`MessageGraph`'s `global_log_index` (`graph.rs:66-86`), which increments on every log, matching, not
just executing ones.

The invariants around it are stock and MUST be applied in stock order, mirroring
`MessageGraph::check_single_dependency` (`graph.rs:175-252`) and reusing
`kona_interop::MessageRules` rather than re-deriving it:

1. executing chain in the dependency set → else `ChainNotInDependencySet`
2. initiating chain in the dependency set → else `ChainNotInDependencySet`
3. `check_executing_activation` — interop active for a full block at the executing timestamp
4. `check_message_ordering` — `initiating_timestamp <= executing_timestamp`, **inclusive**; else
   `MessageInFuture`
5. `check_initiating_activation`
6. `check_message_expiry` — `executing_timestamp - initiating_timestamp <= window`, **inclusive**;
   else `MessageExpired`. MUST run after step 4, which is what makes the subtraction safe. Window is
   `DependencySet::get_message_expiry_window()` — 7 days unless overridden, and an override of `0`
   means *unset*, not a zero window.
7. resolve the coordinate → else `RemoteMessageNotFound`
8. the `logHash` comparison of D18 → else `InvalidMessageOrigin` / `InvalidMessageHash`
9. initiating block timestamp equality → else `InvalidMessageTimestamp`

Whole-graph cycle detection (`MessageRules::check_no_cycles`, for same-timestamp edges) runs before
the per-message loop, as in `MessageGraph::resolve`.

### D21 — failing consolidation ABORTS; it never commits

**A settlement proof that cannot consolidate is not produced.** Any failure of any step above makes
the superroot program *fail* — no public values are committed, no proof exists, nothing is posted.
There is deliberately **no** "consolidated: false" output and no replacement claim. The settlement
program either proves a valid history or proves nothing; it does not authorize a verifier to invent
a replacement block.

If A later reorgs away a consumed message, the sequencing supernode replaces the affected P block
through the stock deposits-only path and P **re-proves** from the last valid point. Verifiers accept
that corrected proof as a supersession of the denied suffix. The burned proving cost is wasted work
by design. A proof that could say "I failed to consolidate" would still be a second, weaker kind of
settlement claim, and the depset has no rule for reading one.

The scoping boundary this creates — a P import whose initiating block falls outside the settlement
window cannot be resolved in memory, so it aborts too — is a real operational constraint with a
liveness edge. It is written up as decision **G7R D6** in `g-decisions.md` with options and a
recommendation, because it is a choice about window sizing rather than a property of the wire.

---

## 8. What v3 discloses, in proportion

The disclosure is the **consumption edge** at block granularity: "P block *b* consumed A's message
at (block, logIndex)". Stated exhaustively:

**Newly public**
* the set of distinct (initiating log, consuming P block) edges, per block;
* hence that block *b* contained at least one executing transaction, and the cardinality of its
  distinct import set.

**Still private**
* every executing transaction's hash, index, sender, target, calldata, gas and value;
* how many executing transactions a block contained;
* the order in which the block consumed its messages;
* the multiplicity of consumption (dedup destroys it: consumed once and consumed nine times are the
  same wire bytes);
* the entire interior of the block — every non-interop transaction, every unexported log, all state.

The information content of an element is bounded above by A's own public data: `origin`,
`blockNumber`, `logIndex`, `timestamp` and `msgHash` are all facts about a log on a **public** chain,
already readable by anyone. The genuinely new bit is that P consumed it.

**Import policy v1 is therefore: all consumed messages are public.** A future private-import tier
needs no architecture change and no wire change — validate the import in-circuit against an
L1-anchored **settled** superroot (witnessed from L1 state) by walking output root → blockHash →
header → receiptsRoot → MPT → log. That buys private consumption at settlement-cadence latency plus
one MPT proof per message, and it is a strictly additive second discharge site.

---

## 9. Size

Per-batch fixed overhead is unchanged (13-byte envelope + the `ProofBatch` head). Per item,
**measured** by `tests::the_v3_size_formula_holds`:

| item | v2 | v3 |
|---|---|---|
| per block | 256 B | **320 B** (+64: one offset word + one length word for `execMsgs`) |
| per exported log | 160 B | 160 B |
| per distinct executing message | — | **192 B** (six words, packed inline; `ExecMsg` is static) |

Two generative fixtures measure the two ends of the range, both at a 300-block (10-minute) cadence:

| case | blocks | logs | imports | bytes | blobs |
|---|---|---|---|---|---|
| `idle-300-blocks` — **the operating point** | 300 | 0 | 0 | **96 301** | 1 |
| `max-realistic-300-blocks` — the heavy tail | 300 | 300 | 150 | **173 101** | **2** |

The operating point grows from v2's 77 101 B to 96 301 B (300 × 64) and stays comfortably inside one
blob (`MaxBlobDataSize` = 130 044, headroom 33 743). Chain P is private: the overwhelming majority
of its blocks touch nothing cross-chain, so this is the size the submitter actually sends.

**The finding, recorded because it must not be discovered in production:** the heavy tail now needs
**two** blobs where its v2 equivalent needed one. 300 blocks at 320 B = 96 000; adding 300 exported
logs reaches 144 301 B, which already exceeds one blob *before any imports*; adding 150 imports
reaches 173 101 B. This is not a v3 limitation — the envelope has always permitted multi-blob
transactions ("blobs concatenate in sidecar index order before decoding") — but a submitter that
assumed one blob per envelope must now count. Both generative fixtures assert their blob count, and
one asserts that the two straddle the boundary, so the crossing cannot drift back into an assumption.

Measured sizes for every fixture are in `testdata/proof-batch/index.json`.

---

## 10. Verifier acceptance rules

v2 rules 1–6 (`SPEC-PROOF-BATCH.md` §"Verifier acceptance rules") apply unchanged, with rule 2's
decode now including D12's version gate and D17's ordering/dedup enforcement, and rule 6 extended:

> 6. On accept: proven head ← `newOutputRoot`; per-block `(number, timestamp, blockHash, logs)` feed
>    the LogsDB ingestion seam as in v2 — **and** per-block `execMsgs[]` feed the executing-message
>    side, so P's dependencies enter the supervisor's cross-safety fixpoint exactly as a driven
>    chain's do (checksum, hazards, expiry, cycles). P's frontier view no longer treats P as
>    import-free.

The LogsDB rule (PLAN.md, binding) is unchanged and still one-directional: LogsDB is fed **from the
wire**, with explicit indices and poison gaps; rendering-device receipts are display-only and never
an ingestion source. `execMsgs[]` is the wire's import channel and is subject to the same rule — the
shim's receipts must never be a source for it either.

---

## 11. Fixtures

Canonical: `op-supernode/proofbatch/testdata/proof-batch`, read by `fixtures_test.go` package-relative.
The generator that produced these bytes is prover-side and lives on the shelf branch (`README.md`
§"What is deliberately not here"); regenerating them is therefore a shelf operation, and on this
branch the corpus is a frozen record rather than a build output. That is the point of a fixture
corpus: it outlives either implementation.

Every v2 case is regenerated under v3 (every valid one gains an `execMsgs` field, empty unless the
case is about imports), and every block's transcript now carries `execMsgs[]` with each entry's
derived `logHash`, `checksum` and 192-byte `key` — so the Go mirror pins the arithmetic, not only the
field order. **35 cases** (v2 had 25), plus two generative cases. New:

| case | kind | pins |
|---|---|---|
| `exec-msgs-empty` | valid | a block that exports but imports nothing; the zero-length array is present, not elided |
| `exec-msgs-single` | valid | one import, all five identifier fields non-trivial |
| `exec-msgs-multi` | valid | import counts 3/0/1 across three blocks, including a **self-import** (`chainId` = P) |
| `exec-msgs-dedup` | valid | the *output* of deduplicating an extraction that saw one message 4× and another 2× |
| `exec-msgs-ordering` | valid | key order differs from the authored order and from any coordinate-major order — it groups by **origin address** |
| `exec-msgs-max-widths` | valid | `blockNumber`/`timestamp` at `u64::MAX`, `logIndex` at `u32::MAX`, `chainId` at `u256::MAX` |
| `exec-msgs-same-identifier-different-msghash` | valid | contradictory pair, both retained (D17) |
| `exec-msgs-unsorted` | invalid | strictly-increasing key violated by a reversal |
| `exec-msgs-duplicate` | invalid | strictly-increasing key violated by a repeat |
| `retired-version-0x02` | invalid | D12, backwards — the version the live cluster runs |
| `unknown-version-0x04` | invalid | D12, forwards |
| `idle-300-blocks` | generative | the operating point: 96 301 B, one blob |
| `exec-msg-checksum-vectors.json` | vectors | §6's ladder + the canonical key + an ordering/dedup vector, incl. the live-verified demo vector and a transposition pair that must not collide |

Retired: `unknown-version-0x03`, whose byte is now valid. Its v2-era file survives in the frozen
corpus, where it does a better job than it used to (D22).

### D22 — the v2 corpus is retained, and given a job

The complete v2 fixture tree is frozen at `testdata/proof-batch-v2/` as the pre-rotation record
(the live cluster runs v2 until the rotation). It is not inert: `tests/retired_v2_corpus.rs` walks
every `*.bin` whose first five bytes are `KCPB\x02` — 21 of them — and asserts the **v3** codec
refuses each with `BadVersion(0x02)`, at both the `Envelope::decode` layer and the full `decode`
layer. So the retained record continuously proves D12's backwards gate against **real v2 envelopes,
bodies and all**, rather than against a v3 body wearing a flipped version byte, and the two
directories cannot drift into agreeing. It also asserts the frozen `index.json` still says
`version: 2`, which is what catches the v3 generator ever being pointed at the record by mistake.

One retained file earns a dedicated test. The v2 corpus contains `unknown-version-0x03.bin`, written
when `0x03` was the *next* version: its version byte is now **valid** and its body is still v2. That
is exactly the hazard D12 describes and the one case the version gate cannot catch, because the gate
passes. What must catch it is the ABI layer refusing a six-field `BlockExport` where seven are
required — verified: it fails with `AbiDecode`. Had it decoded, a v2 batch would read as a v3 batch
with every block's import list silently empty, which is the worst available failure mode for this
field.

---

## 12. Rotation

v3 rotates the key of each proving program it affects, recorded with the shelf and NOT DEPLOYED. The
configured `rollupConfigHash`, `depSetHash` and `exportPolicyHash` are unchanged — v3 touches
neither the rollup config, the dependency set, nor the export filter. Deploying v3 is a rotation
event (PLAN.md DR-5), and it is deliberately **not** part of building it.
