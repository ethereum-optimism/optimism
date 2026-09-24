# Projection batch admission and private checkpoints

The public projection derives from ordinary L1 channel frames. Published spans carry
private output commitments and message records, not private application transactions
or write sets. Ordinary L1 deposits remain public. On the projection they follow its
existing deposit execution policy; the private chain executes them normally.

This is an experimental, fresh-deployment profile. Its production verifier is
**`sp1-private-projection-v1`, the sound proof profile**: derivation admits a span only if
an SP1 Groth16 proof verifies against a public statement that the verifying node computes
itself from its own derivation state and consensus configuration. A valid proof implies
that every published private output root follows from executing the pinned private chain,
that the claim's terminal and parent fields are the unique consequence of that execution,
and that the span renders every initiating and executing message of the executed private
blocks, in order, with nothing omitted, added or moved (see
[the statement](#pure-admission-and-the-proof-statement)).

What this change implements and what it defers:

| Item | Status |
|---|---|
| Consensus constants, statement, public values, envelope, Groth16 verifier (Go and Kona), mock-envelope gate | implemented |
| Admission binds `l1Head`, `rollupConfigHash`, `depSetHash` against the node's own view, in every verifier mode | implemented |
| Merklized `outputsRoot` (every per-block output) and `messagesRoot` (every rendered message) | implemented |
| Relation derives private attributes and forced inputs in-guest from L1, anchored at `l1Head` | implemented |
| Execution rule: a failed carrier transaction invalidates the projection block | implemented |
| Publication blocks until the proof exists; carrier pre-run; scaled proof timeout | implemented |
| End-to-end plumbing with SP1 **mock** envelopes (test-gated) | implemented, exercised in CI |
| Producing a real Groth16 proof of the relation (`--private-interop.sp1-prover=cpu\|network`) | code path exists; **not exercised in CI** |
| Chunked recovery proofs after long outages | **deferred** (see [long outages](#latency-and-the-sequencing-window)) |
| Rotation of `program_vkey`, `private_config_hash`, `dependency_set_hash` or the circuit | **deferred**: a projection hardfork, not specified |
| Unifying the op-reth and Kona projection-execution triggers | deferred (the rule itself is shared code) |

The legacy verifiers `insecure-stub-v1` and `execution-mock-v1` remain for tests only
([legacy, test-only](#legacy-test-only-verifiers)). Neither provides cryptographic
private-execution verification. Neither an output root nor a successful local recovery
proves private execution; only an admitted `sp1-private-projection-v1` Groth16 proof does.

## Submitted block layout

Each submitted span has exactly one leading `postClaim` transaction. Its first block
then contains `recordOutput(bytes32)`, followed by zero or more message replay calls.
Every subsequent block starts with exactly one `recordOutput`, followed by replays.
Thus an application-empty submitted block still has an output record. The first
block has both a claim and an output record. Missing, repeated, late or malformed
records reject the entire submitted span before its first new block is released.

The root is the standard private OP OutputV0 commitment:

```text
keccak256(versionZero || privateStateRoot || privateMessagePasserStorageRoot || privateBlockHash)
```

The publisher obtains the message-passer storage root from the private payload's
post-Isthmus `withdrawalsRoot`. The source configuration must activate Isthmus at
genesis. Projection configuration includes the independently computed private
genesis output root as the initial checkpoint. The public projection's own state
root is never used as a private commitment.

Both record methods emit no logs. `recordOutput` deliberately stores nothing: its
canonical admitted calldata is the durable record. Every carrier transaction
(`postClaim`, `recordOutput` and every replay) must execute successfully: a revert
invalidates the whole projection block (see [the execution rule](#execution-rule-a-failed-carrier-invalidates-the-block)).
`ClaimRegistry` 4.0.0 therefore no longer reverts on an overlapping range: after a
partial-span invalidation the next honest claim overlaps the last posted range, and an
overlap revert would invalidate its block forever. `postClaim` always advances
`rangeCount`, sets `lastPostedLastBlock` and extends `lastClaimHash`; an overlap in the
record marks a partially invalidated span. A claim signed by a key other than the current
batcher still reverts and invalidates its block. Legacy claim-follow mode retains its
receipt policy. Protocol-generated deposits/PostExec transactions cannot create
checkpoints, even if their destination or calldata impersonates a record method.

## Pure admission and the proof statement

### Configuration: the consensus constants

The projection rollup config opts into `private_projection`. Holocene and interop
must be active at genesis. Ordinary chains omit this config and retain their rules.
Go `projection.Config` and Kona `PrivateProjectionConfig` use the same JSON:

```json
"private_projection": {
  "verifier":            "sp1-private-projection-v1",
  "genesis_output_root": "0x…",
  "program_vkey":        "0x…",
  "private_config_hash": "0x…",
  "dependency_set_hash": "0x…",
  "allow_events":        false,
  "mock_proofs":         false
}
```

Three consensus constants pin what a proof may be about, plus the dependency set:

- `program_vkey`: the SP1 verification-key hash of the private-projection guest
  (`vk.bytes32()`, as `cargo prove vkey` or `kona-sp1-private-projection-executor --print-vkey`
  prints it). Nonzero and below the BN254 scalar modulus. The production value comes from
  the reproducible docker ELF build (`just build-elfs` writes it to `elf/vkeys.toml`).
- `private_config_hash`: the exact deployed bytes of the private chain's configuration,
  `keccak256("optimism.private-config.v1\0" ‖ keccak256(PRIVATE_ROLLUP_JSON) ‖ keccak256(L1_CHAIN_CONFIG_JSON))`.
  The L1 chain config is included because private L1-info deposits (blob base fee) depend
  on the L1 fork schedule. There is no canonicalisation: nothing re-serialises a config to
  recompute the hash, because Go and Kona serde disagree on fields. Operators distribute
  both files, byte for byte, to the batcher and the prover. This is a consensus constant
  and deliberately not a claim field, so the operator cannot choose a config per claim.
- `genesis_output_root`: the private genesis OutputV0, the base checkpoint.
- `dependency_set_hash`: `keccak256("optimism.private-dependency-set.v1\0" ‖ u64be(n) ‖ id_1 ‖ … ‖ id_n)`
  over the distinct chain IDs of the private chain's dependency set (including itself), as
  32-byte big-endian integers sorted ascending. No other dependency-set field is hashed.

The verifier ID also pins the circuit: `sp1-private-projection-v1` means the SP1 Groth16
circuit v6.1.0 shipped with sp1-sdk 6.8.0. A new circuit needs a new verifier ID.
`allow_events` must be `false`: the v1 relation does not render generic `EventReplayer`
emitters. `mock_proofs` accepts SP1 mock envelopes and passes `Check` only under the
[test gate](#the-test-gate).

The profile is genesis-only: fresh projection chains. Changing any constant is a
projection hardfork that this change does not specify. Projection activation blocks
cannot sit inside a span, so every rotation would force a sequencing-window outage plus a
proven recovery.

### What admission checks

After decompression and decoding, Go's `checkSpanBatchHolocene` and Kona's
`SpanBatch::check_batch_holocene_with_context` check range geometry, authenticate the
original span parent and overlap, resolve continuation context, and call
`projection.ValidateProjectionRange` / `validate_projection_range`. This happens
before decomposition releases any new singular batch. The producer uses the same
Go validator. Byte decoding alone cannot authenticate a canonical parent.

```text
resolveContext(canonicalProjectionHistory, originalSpanParent, genesis)
    -> Ready(context) | Unavailable | Invalid
validateSpan(config, context, completeDecodedSpan, pureVerifier)
    -> Valid(statement) | Invalid
```

The validator performs no RPC, clock, database or mutable-cache access. Its input
includes chain ID, block interval, genesis number, hash and time, full canonical public
parent hash, the L1 head (below) and an independently resolved continuation. It enforces:

- 1–65,536 aligned consecutive blocks, excluding genesis, with overflow-safe heights.
- One version-2 claim whose first/last heights cover exactly the entire span.
- The fixed per-block output placement described above, with nonzero roots.
- Canonical, validly signed EIP-1559 envelopes for this chain; zero value and fee caps;
  nonzero gas at most 16,777,216; no creation or operator-supplied deposit envelopes.
- Only `postClaim`, `recordOutput`, `replaySentMessage`, `validateMessage`, and enabled
  `replayEvent` at their designated predeploys. Native bridge replay is forbidden.
- Exact ABI encoding, at most 64 KiB proof bytes, 1 MiB message bytes and four event
  topics; no trailing data, dirty padding or alternate offsets.
- Empty access lists except the exact inbox checksum list derived from each import.
- Claim anchor height/root/recovery hash equal to the independently resolved context.
  Without a recovery interval, the claimed private parent output must equal the
  checkpoint at the public parent, and the recovery hash must be zero.
- **In every verifier mode**, three claim fields against the node's own view:
  - `claim.l1Head` equals the hash of the L1 block in the node's own L1 window whose
    number is the span's **last** epoch (the terminal block's L1 origin). The span prefix
    check already looks up that block. The batcher derives `l1Head` the same way.
  - `claim.rollupConfigHash` equals `ConfigHash` of the node's own projection config (below).
  - `claim.depSetHash` equals `private_projection.dependency_set_hash`.

`rollupConfigHash` is the canonical binary hash of the **projection** config, which also
commits the verifier ID, vkey, private config hash and dependency set hash:

```text
ConfigHash = keccak256("optimism.private-projection-config.v1\0"
    ‖ chainId (32 bytes) ‖ u64be(genesis.l2.number) ‖ genesis.l2.hash ‖ u64be(genesis.l2_time)
    ‖ u64be(block_time) ‖ keccak256(verifier) ‖ genesis_output_root ‖ program_vkey
    ‖ private_config_hash ‖ dependency_set_hash ‖ u8(allow_events) ‖ u8(mock_proofs))
```

This replaces the earlier convention of hashing a Go JSON marshal of the rollup config
and of the dependency set, which Kona could not reproduce and derivation never checked.
The claim layout and version (2) are unchanged.

In the same pass admission collects two commitment trees, with domain-separated roots:

- `outputsRoot`: one leaf per block, `keccak256(0x00 ‖ u64be(blockNumber) ‖ outputRoot)`,
  over the root decoded from each block's `recordOutput`. The last leaf is `terminalOutput`.
- `messagesRoot`: one leaf per replay,
  `keccak256(0x00 ‖ u64be(blockNumber) ‖ u32be(renderedIndex) ‖ u8(kind) ‖ messageHash)`, ordered
  by block then rendered index. `kind` is `0x01` for an export (`SentMessage`, hash =
  the standard interop payload hash `keccak256(topics ‖ data)` rebuilt from the replay
  calldata) and `0x02` for an import (`ExecutingMessage`, hash =
  `keccak256(identifier ‖ payloadHash)`, which is `keccak256(calldata[4:196])` of
  `validateMessage`). `renderedIndex` is the replay's ordinal among the block's
  transactions after the output record, which equals its log index on the projection.

Both trees duplicate the last node at odd levels (`H(0x01 ‖ left ‖ right)`) and finish
with `keccak256(domain ‖ u64be(n) ‖ top)`, where the empty tree's top is zero and the
domains are `optimism.private-outputs.v1\0` and `optimism.private-messages.v1\0`.
Inclusion proofs (`CommitmentProof` / `VerifyCommitmentProof`) are implemented and
vector-tested for later single-checkpoint verification.

The records root is unchanged. Each leaf is `H(0x00 || blockTranscript)`. The transcript
uses big-endian uint64 heights, timestamps, L1-origin numbers and transaction counts, then
each sender, nonce, gas, destination, calldata length and canonical calldata. Proof bytes
are normalized to empty; signature bytes are excluded to avoid circularity, while the
recovered sender remains bound. Internal nodes are `H(0x01 || left || right)`,
duplicating the last node at odd levels. The final root is
`H("optimism.private-projection.v2\0" || uint64be(leafCount) || treeRoot)`. The count
prevents duplicated leaves from aliasing spans of different lengths.

### Public values

The guest commits a fixed 672-byte `PublicValuesV1`: 21 big-endian 32-byte words (the
ABI encoding of a static tuple), written with `commit_slice`. Admission in Go and Kona
builds the same bytes from its own state, and the proof must commit exactly those bytes.

| Word | Field | Admission's source |
|---|---|---|
| 0 | magic | `keccak256("optimism.private-projection.public-values.v1")` |
| 1 | chainId | the node's chain ID |
| 2 | projectionConfigHash | `ConfigHash` of the node's projection config |
| 3 | privateConfigHash | consensus constant `private_config_hash` |
| 4 | depSetHash | consensus constant `dependency_set_hash` |
| 5 | parentHash | the span's canonical parent |
| 6–9 | anchorNumber, anchorHash, anchorOutputRoot, recoveryHash | the node's own `ContextCollector` |
| 10–11 | firstBlock, lastBlock | claim, checked against the span |
| 12 | parentOutputRoot | claim (checked in normal mode; bound by the relation in recovery mode) |
| 13–14 | privateTerminalBlockHash, privateTerminalParentHash | claim; the relation requires the executed values |
| 15 | l1Head | claim, checked against the node's L1 window |
| 16 | privateDataHash | claim; the relation requires `keccak(private_data)` |
| 17 | projectionHash | the records root |
| 18 | outputsRoot | the published `recordOutput` roots |
| 19 | messagesRoot | the published replays |
| 20 | terminalOutput | the last `recordOutput` |

Inside the guest the relation computes each word from what it executed and fails unless
the published data agrees: the configs it parsed hash to words 2–4, the anchor header and
storage witness match word 8, the executed outputs build word 18 and equal the published
records, and the `RenderedLogs` of the executed receipts build word 19 and equal the
published replays. Everything the prover supplies is untrusted. The only verified output
is the public values, and the verifier recomputes every word. In particular the private
payload attributes and forced deposits of every recovery and new block are derived
in-guest with Kona's attributes builder from L1 headers and receipts authenticated by
`l1Head` (word 15); the prover cannot choose deposits, fee parameters or L1-info fields.

### Envelope and verification

The claim's `proof` slot carries a strict envelope:

| Offset | Size | Field |
|---|---|---|
| 0 | 1 | version `0x01` |
| 1 | 1 | kind: `0x01` SP1 Groth16, `0x02` SP1 mock |
| 2 | 2 | proof length `L`, u16 big-endian |
| 4 | L | proof |
| 4+L | 672 | `PublicValuesV1` |

The total length must be exactly `676 + L`. A Groth16 proof is 356 bytes (4-byte circuit
prefix ‖ exit code ‖ vk root ‖ nonce ‖ 256-byte gnark proof, as `SP1ProofWithPublicValues::bytes()`
lays it out), so the envelope is 1,032 bytes. A mock proof is 160 bytes (five words
`[vkey, digest, exit, vkRoot, nonce]`), so the envelope is 836 bytes.

`SP1Verifier.Verify(statement, bytes)` decodes strictly, requires the envelope's public
values to equal `PublicValues(statement)` byte for byte, then:

- kind `0x01`: verifies the Groth16 proof against the pinned circuit, with public inputs
  `[program_vkey, sha256(publicValues) with the top three bits cleared, exit, vkRoot, nonce]`.
  The prefix must match the circuit key, the exit code must be zero and the vk root must
  equal the circuit's. Go uses gnark-crypto BN254 directly (`sp1groth16.Verify`); Kona uses
  `sp1-verifier` (`verify_groth16`, feature `sp1-projection-verifier`). The blake3 digest
  variant is not accepted. A Kona build without the feature fails closed.
- kind `0x02`: requires the [test gate](#the-test-gate) and `mock_proofs`, then requires
  the first word to equal `program_vkey`, the second to equal the masked sha256 of the
  public values, and the rest to be zero. **A mock envelope proves nothing**: anyone can
  build one. It exists to exercise the statement, public-values and envelope plumbing end
  to end.
- anything else: reject.

Both verifiers are tested against a vendored real Groth16 fixture (SP1's Fibonacci
program on the v6.0.0 circuit) and its negatives, and pin the v6.1.0 production circuit
key and root.

### Composition with the super root

Private → projection is verified at admission during derivation. Projection → super
root is covered by the super dispute game: the super-range program re-runs Kona
derivation, including this admission check and its Groth16 verification (compiled into
the zkVM ELF with `sp1-projection-verifier`, without the test gate). A super-root proof
over the projection therefore implies its spans were admitted with valid proofs.

### Scheduling

A final preflight checks ordinary singular scheduling rules over the unexecuted
suffix, including L1-origin timestamps, drift and upgrade blocks. For drift only,
strict checkpoint metadata counts as application-empty; any replay still counts as
nonempty. All empty-batch next-origin timing checks remain enforced. Fork activation
blocks still prohibit submitted transactions: this fresh interop-at-genesis profile
must not assume a future upgrade can be crossed with a published span. Such spans
are rejected and require expiry/recovery, or a separately specified upgrade policy.
Structural/proof failure drops the span and flushes its channel. Submitted singular wire
batches cannot bypass this gate. Invalid interop dependencies retain their ordinary handling.

### The test gate

A verifier mode is test-gated if it is `insecure-stub-v1`, `execution-mock-v1`, or
`sp1-private-projection-v1` with `mock_proofs = true`. A test-gated config passes `Check`
only if the gate is compiled in (Go: build tag `private_interop_test_verifiers` or a
`go test` binary; Kona: Cargo feature `private-projection-test-verifiers`) **and** the
projection chain ID is 901 or 902 (the devstack chains). The gate is enforced both when
the rollup config is checked and inside the validator, so it fails closed: an ungated
binary rejects every span of a test-gated chain. Production zkVM ELFs are built without
the gate, so mock envelopes also fail inside a super-range proof.

## Execution rule: a failed carrier invalidates the block

On a public-projection chain, every transaction in a block that is not a deposit (0x7E)
and not a post-exec (0x7D) transaction must execute with a successful receipt. A revert,
out-of-gas or other EVM failure makes the whole block invalid, both when it is imported
and when it is built from derived attributes. Admission already restricts non-deposit
transactions to the four carrier kinds, so the rule is exactly "all carriers fail
closed". It is implemented once in `alloy-op-evm` (`require_sequencer_tx_success`,
error `ProjectionSequencerTxFailed`) and switched on by both op-reth and the Kona
executor for projection chains.

A carrier the EVM rejects as an *invalid transaction* rather than executing it (a nonce gap
or reuse, a gas limit below its intrinsic gas or the EIP-7623 calldata floor) is covered too:
on projection chains the executor reports it as `ProjectionSequencerTxInvalid` instead of the
generic `InvalidTx`. That matters because payload builders skip `InvalidTx` transactions of
derived attributes: before, op-reth's payload job and its FCU pre-check dropped such a carrier
and built the block without it (a hidden executing message, and a split from Kona, which fails
the block), while now the job fails and the FCU pre-check answers `INVALID_PAYLOAD_ATTRIBUTES`,
as op-geth and Kona do. Admission additionally rejects any span transaction whose gas is below
`max(intrinsic, floor)` (`projection.MinTxGas` in Go, `projection::min_tx_gas` in Kona). It
cannot check nonces, because it is pure and has no state; a nonce fault is caught by the
execution rule, and the batcher's pre-run checks that its carriers' nonces are contiguous
from its nonce at the span parent.

This closes the replay-revert hole: previously gas was only range-checked, so an
under-gassed replay could revert, drop its log and renumber every later message in the
block while the span was still admitted. Now each admitted replay emits its log, and
`messagesRoot` binds which logs there are. The two layers together give completeness.

Under Holocene an invalid payload built from a batch is replaced by a deposit-only
block and the rest of the span is dropped. The records of the prefix already executed
stay canonical, and the next span continues in recovery mode from the last canonical
record.

**How op-reth reports it.** Derivation hands each block to the execution client as payload
attributes in `engine_forkchoiceUpdated`. op-reth normally answers VALID with a payload ID
and builds the block asynchronously, so a carrier failure inside that job only surfaces as an
unknown payload, which the consensus client would retry forever. On projection chains op-reth
therefore **pre-executes** the attributes' transactions on the parent state inside
`engine_forkchoiceUpdated` (`reth_optimism_payload_builder::projection::first_failing_sequencer_tx`,
installed by `OpEngineValidatorBuilder`) and answers `INVALID_PAYLOAD_ATTRIBUTES` when the
execution rule rejects them; derivation then substitutes the deposit-only block. This mirrors
op-geth, which builds attributes-only payloads synchronously. Two costs follow: every derived
projection block is executed twice (once in the check, once in the payload job), and the check
gives no verdict when the parent state is unavailable, in which case the payload job decides
as before. The Kona stateless executor needs no such step: it executes the block itself and
reports the failure directly.

Claim-follow applies the same rule when it maps projection safety onto private safety: a
claim whose carrier survives but whose later blocks were replaced by deposit-only blocks on the
same branch (no reorg, nothing denied) is clipped to the last block before the first
record-less block of its range. The replaced suffix is recovered, never served as private-safe
([RECOVERY.md](RECOVERY.md#canonical-recovery-intervals)).

## Continuation after invalidation or sequencing-window expiry

For a span 1–5 followed by replacement of 3–5 with 3′–5′, the next span 6–10 extends
canonical public 5′. Its surviving private checkpoint is the output record at 2,
not the old submitted endpoint at 5. The claim carries:

- `anchorBlock` and `anchorOutputRoot`: the surviving checkpoint.
- `recoveryHash`: a commitment to the canonical replacement inputs from the
  publication parent back to, but excluding, the anchor.
- `parentOutputRoot`: the private output after executing that recovery interval.

The resolver walks hash-linked canonical projection history backward from the
**parent of the original full span**, including when part of the span overlaps safe
history. It stops at the nearest admitted output record, or at the configured
private genesis output. A block containing sequencer transactions but no valid
output record is invalid context, never silently classified as fallback.

Recovery hashes fold blocks in descending height order, beginning with zero:

```text
H("optimism.private-recovery.v1\0" || previousHash || publicBlockHash || publicParentHash
  || uint64be(number) || uint64be(timestamp) || uint64be(transactionCount)
  || each(uint64be(transactionLength) || rawTransaction))
```

The public block hash additionally binds the complete header. Under
`sp1-private-projection-v1` the relation executes from the surviving output (word 8)
through these canonical replacement inputs (word 9), deriving their private attributes
from L1, and requires the result to equal `parentOutputRoot` (word 12); then it executes
the submitted range and binds every output and message. In normal mode admission itself
checks `parentOutputRoot` against the canonical parent record. In recovery mode admission
only requires it to be nonzero, and the proof is the binding: under the legacy
`execution-mock-v1` and `insecure-stub-v1` verifiers a recovery-mode `parentOutputRoot`
is **unconstrained**.

Protocol replacements need no publisher proof or private RPC to be derived. Their
private-state correspondence is established later by the recovery segment of the
next span proof. Deposits replayed during recovery can emit private messages;
those messages are not retroactively exported into deposit-only public replacements
and are not part of `messagesRoot`. Requiring a proven replacement root before that
recovery proof would deadlock.

## Missing context, overlap and resets

The stateful context collector is outside the pure validator. Each attempt pins the
canonical parent hash, then performs at most 128 backward reads. One constant-sized
cursor persists across attempts; there is no consensus maximum outage length.
Unavailable committed data or unfinished scanning retains one candidate and its
original L1 inclusion block, returning a temporary error. Every decomposed block
retains that inclusion block for its sequencing-window checks. The pipeline does not
flush, emit a prefix, substitute unsafe history, or read a new candidate while waiting.

A changed target parent restarts collection. Reset/flush discards both candidate and
cursor. Malformed context rejects the candidate. Ordinary missing L1 scheduling
information retains existing Holocene handling so waiting cannot prevent L1 traversal.
Valid overlapping public records are checked against canonical history and only the
new suffix is emitted. An overlap matching abandoned history is rejected.

[LightCL recovery](RECOVERY.md) uses a surviving root to identify a matching local
private checkpoint, preserves ancestry/finality and restart safeguards, executes the
canonical replacement interval, and resumes sequencing after the reserved range.

## Publication

The batcher builds the candidate span, runs the structural preflight, then:

1. **Carrier pre-run.** A static check that each carrier's gas covers the intrinsic and
   EIP-7623 floor cost (the admission bound), a check that the carriers' nonces are
   contiguous from the batcher's nonce at the span parent (`eth_getTransactionCount` by
   block hash; `eth_call` ignores nonces), then an `eth_call` of each carrier, in order, on
   the projection at the span parent. Any revert or out-of-gas is a fatal error naming the block, index and
   revert data. The span is not published and not silently retried. The carriers are
   stateless or read only parent state, so the simulation is exact, except for a batcher
   rotation in the first block's epoch. Export and claim gas limits now also cover the
   calldata floor.
2. **Prove.** The producer (`--private-interop.proof-command`, the
   `kona-sp1-private-projection-executor --publication-request` host) receives a version-2
   request with the three config byte strings, the span, the private data, `l1_head` and the
   expected public values. It collects the private witness and the L1 headers and receipts,
   runs the native relation, requires the expected public values, and emits the envelope.
3. **Verify locally** with the same `VerifierFor(config, chainID)` that nodes use, rebuild
   the claim with the proof, and run admission again.

The batcher never chooses the verifier. It reads `private_projection` from the
**deployed** projection rollup config over RPC and refuses to start unless its own
projected config matches, the `--private-interop.private-rollup-config` and
`--private-interop.l1-chain-config` files hash to `private_config_hash`, and the rollup
node's dependency set hashes to `dependency_set_hash`. `--private-interop.sp1-prover`
selects `network` (default), `cpu`, `mock` or `native-mock`; the two mock provers are
refused unless the deployed config sets `mock_proofs`, and the two real provers are refused
unless `--private-interop.proof-timeout` is given explicitly.

## Latency and the sequencing window

**Publication blocks until the proof exists.** There is no fallback verifier and no
unproven publication. The proof timeout is
`--private-interop.proof-timeout` plus `--private-interop.proof-timeout-per-block`
(default 100ms) times the number of blocks from the anchor to the span end, so it scales
with the recovery interval. The base timeout defaults to 2m only for the mock provers and
`execution-mock-v1`. With `--private-interop.sp1-prover=cpu|network` it has no default and the
batcher refuses to start without it: a real Groth16 proof never finishes in 2m, so each span
would time out, be retried and never publish until its window expired, and the next span would
have a longer recovery interval to prove (the long-outage spiral below). Size it from measured
proving time for a full span plus the longest recovery interval you intend to survive (the
devstack uses 60m for real proving).

A span must be included within `SeqWindowSize` L1 blocks of its first epoch. If proving
time plus L1 inclusion exceeds the remaining window, the window expires, derivation fills
deposit-only blocks, the private chain recovers, and the next span must prove the
**entire** recovery interval. `native` and `native-mock` fit the default timeout for normal
spans; SP1 `mock` with the ELF needs a longer timeout (the devstack uses 20m).

Long outages, what this change fixes and what it defers:

- The 128 MiB request cap is no longer outage-dependent: the request carries only
  span-bounded data, recovery blocks and witnesses are fetched by the host over RPC, and
  the batcher checks the size before spawning the host.
- The timeout scales with the interval (above), and the host fetches witnesses with
  bounded concurrency.
- **Deferred: chunked recovery proofs.** With real proving, one SP1 proof over a long
  recovery interval exceeds practical cycle and memory limits. The fix is a recovery-segment
  statement chained through an intermediate public output, which needs a second statement
  type, admission support for several proofs per span and a claim format change. It only
  matters once real proving exists. **Until then the sound profile inherits this liveness
  risk for real proving:** a long enough outage can make the next span unprovable.

Other liveness limits, not soundness gaps: an unrenderable private log (bridge
sender/target, oversize, malformed) makes the span unprovable and stalls publication.
The renderer's conservative gas formula limits export messages to 581,329 bytes, below
the structural 1 MiB wire bound.

## Verification and deployment limits

Go generates and Kona consumes shared vectors: `ranges.json` and `proofs.json` (schema 2,
with a per-vector `context`, including recovery-mode cases and one flipped-word case per
public-values word), `commitments.json` (commitment roots and proofs, output and message
leaves from both calldata and log, `ConfigHash`, `PrivateConfigHash`, `DependencySetHash`,
one full public-values sample) and `execution.json` (the execution rule in op-reth and
Kona). Regenerate them with `go test ./op-private-interop/projection -update-projection-vectors`
and `go test ./op-private-interop/projection -run TestExecutionVectors -update-execution-vectors`
([DEVNET.md](DEVNET.md#local-verification)). `projection/soundness_test.go` flips every public-values word of every accepted
vector in both envelope kinds, and applies completeness mutations to the published span,
expecting rejection. `private_projection/tests_negative.rs` runs the relation's negative
series against expected public values computed independently by admission. Tests also
cover missing context, long incremental recovery, reorgs, reset, original inclusion
retention, malformed records and changed intermediate roots.

Node and fault-proof derivation use the same Kona gate. Kona's stateless executor and
op-reth share the authenticated-system/user-deposit classifier and the carrier execution
rule: public user deposits are inert, while private execution applies them normally.
Both attribute builders keep projection publication fee-free after L1 SystemConfig
updates (zero fee scalars, operator fees and minimum base fee; maximum gas limit).
Batcher authorization continues to follow L1 updates.

The private super-root lifecycle test (`op-acceptance-tests/tests/proofs/zk`) is an
honest-proposer lifecycle test under a **mock on-chain verifier**, covering only the public
projection. It is not a cryptographic super-root proof.

Use a fresh deployment with matching contracts, rollup config, Go services and op-reth.
Claim wire version 2, ClaimRegistry 4.0.0, the execution rule and the sound-profile
config change projection genesis and admission rules. This is not a live upgrade for
existing private devnets.

## Legacy, test-only verifiers

Two earlier verifiers remain for CI and are [test-gated](#the-test-gate). Both still get
the `l1Head`, `rollupConfigHash` and `depSetHash` checks, the output and message roots
and the execution rule; neither verifies private execution.

- `insecure-stub-v1` accepts any proof bytes; its claims carry an empty proof and no producer
  runs. It is retired from production.
- `execution-mock-v1` requires `optimism.private-execution.mock.v1\0` followed by the
  32-byte admission digest: Keccak-256 over `optimism.private-admission.v1\0`, chain ID,
  public parent hash, records root, anchor height (8-byte big-endian), anchor public hash,
  anchor private output root and recovery-input hash. The honest producer runs the host
  in `native` mode, which executes the relation before publishing, but **anyone can forge
  the envelope**. `program_vkey` and
  `private_config_hash` must be zero in this mode, and the host excludes word 3 from its
  comparison.

These bytes are not proofs. They let acceptance tests exercise publication, recovery
and messaging quickly, and let the execution rule be tested without a prover.
