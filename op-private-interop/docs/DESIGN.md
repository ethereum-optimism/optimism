# Private Interop — the design

**Status:** RATIFIED (Karl, 2026-08-30 and 2026-08-31, in-conversation; written up by Fable).

This is the project's ONLY design document (Karl, 2026-08-31). The separate proven-mode note and
testing plan were folded in below; the work plan and the alt-DA option survey were deleted. Git
history holds all four. Older docs, branches and package names say "Silhouette" — the project's
former name; new code says private-interop.

Every mechanical claim marked *(spike)* was verified by a runnable test against this checkout on
2026-08-30; see "Spike inventory".

## Ratified decisions (Karl, 2026-08-30, in-conversation)

1. **No custom execution client, no supernode changes.** The public verifier stack is stock
   op-node + stock op-reth + a rollup.json. The judge machinery is stock because the private
   chain's cross-chain messages become ordinary transactions on the public chain.
2. **Messenger-only interop.** The private chain exports and imports exclusively through the
   L2ToL2CrossDomainMessenger. Accepted consequence: exported message content is public (
   previously unconsumed exports leaked only a hash).
3. **Synthesized batches, no public sequencer.** The operator's batching service constructs the
   public chain's batches directly from private-chain data: one public block per private block,
   same numbers and timestamps.
4. **No sidecar for public verifiers**, and nothing nonstandard on the batch transaction. (As
   originally ratified this read "envelope as a skipped blob"; the same day it was AMENDED to the
   leading claim TRANSACTION — see "L1 layout" and "The claim-chain framing" below. There is no
   skipped blob and no Alt-DA commitment anywhere in this design.)
5. **Private data availability is the operator's P2P network** (AMENDED, Karl 2026-08-31; as
   originally ratified this read "S3-compatible object store for the full private derivation data,
   content-addressed"). Private blocks reach the operator's own followers over the firewalled
   private p2p network, which is where every legitimate reader of them already lives. The claim
   carries only the CONTENT HASH of the range's full derivation input — a binding commitment,
   with no publication behind it. An off-chain archive, if an operator wants one, is an external
   sidecar reading its own private node; out of scope here.
6. **Same chain ID**: the public chain is the private chain's identity in the dependency set.
7. **Deposits prevented at the L1 portal** (`OptimismPortalNoDeposit` reverts), so derivation and
   attribute handling are never touched — no user deposit can exist for stock derivation to see.
   Note the batch layer independently enforces this: deposit-type transactions inside batch data
   are dropped by stock validation *(spike)*.

## The architecture in one paragraph

The public chain is a derived-only OP Stack chain. Its blocks contain the stock L1-attributes
deposit plus the operator's **replay transactions**: for each message the private chain exported,
a transaction re-emitting the identical `SentMessage` event through a replay implementation at the
standard messenger predeploy address; for each message the private chain imported, a transaction
executing `CrossL2Inbox.validateMessage` with the standard access list. Everything downstream is
therefore ordinary: receipts feed the message database, the cross-safety judge validates imports
and serves exports natively, expiry and same-timestamp cycle rules work unmodified (the executing
log has a real position again, so the old same-timestamp restriction is gone). Each cadence, the
operator posts one L1 blob transaction carrying nothing but stock channel frames. The range's
**claim** — its block range, the private chain's terminal hash, the content hash of the full
private derivation data, and an empty proof slot that a real ZK proof fills later — rides INSIDE
that batch, as the leading L2 transaction of the range's first block. The batch transaction's own
signature is the v1 attestation, checked by the stock inbox filter.

## L1 layout (AMENDED, Karl 2026-08-30: the claim is an L2 transaction, not a blob)

ONE blob transaction per cadence, submitter EOA → the normal batch inbox, containing ONLY stock
channel frames (derivation version 0x00) — byte-for-byte indistinguishable from any chain's
batcher output. There is no nonstandard blob, no skip-byte convention, and no second inbox: this
chain posts exactly one kind of L1 transaction. (The skip-byte spike remains valid knowledge; it
is not load-bearing for anything here.)

## Batch construction (the synthesized public chain)

Channel ID (deterministic, normative): first 16 bytes of
`keccak256(prevRangeTerminalRenderingHash ‖ uint64be(firstBlock))` — the stock batcher
randomizes channel IDs, and the builder's whole payload must be a pure function of its inputs.

One **span batch** per cadence (never singular batches: each singular batch carries a full L2
parent hash, which for intra-range blocks is unknowable before execution; a span batch carries one
20-byte parent check for the whole range, belonging to the previous, already-derived range).

Rules, each verified by the span spike against stock validation:

- **Shape**: ~300 blocks at 2 s (10-minute cadence), empty or with replay transactions — both
  accepted; 300 empty blocks encode to 385 bytes uncompressed, 300 blocks with 90 signed txs to
  ~58 KB; all size limits have ≥3 orders of magnitude headroom.
- **Parent check** = first 20 bytes of the previous range's last public block hash; a mismatch is
  a plain `BatchDrop` (loud stall attributable to the operator, never a divergence or reset).
- **Contiguity**: first block timestamp = safe head time + block time; start epoch ∈
  {parent's origin, parent's origin + 1}; emit **non-overlapping** ranges only (overlap must be
  byte-exact against the canonical chain or the whole batch drops).
- **Origins (AMENDED 2026-08-31: COPIED, not chosen)**: each rendering block reuses the private
  block's OWN L1 origin, read from its L1-info deposit — origins and sequence numbers equal by
  construction, validity transferring since the private sequencer already satisfied the identical
  stock rules at identical timestamps (pinned: a copied-origin range passes derive.CheckBatch).
  The builder has NO live-L1 input and no origin-selection machinery; a block arriving without an
  origin is refused, never defaulted.
- **Posting deadline**: the only lateness rule is the sequencing window in L1 blocks from the
  span's START epoch (~12 h at the default 3600) — L2 timestamps are never compared to the
  inclusion block's time. Channel-level: the carrying channel must open and close within 50 L1
  blocks (automatic for a single-transaction channel) and frames are ≤ 1 MB.
- **No deposit-type transactions in batch data** (stock rule; aligns with the reverting portal).

## Canonical message positions (normative; Karl 2026-08-30)

The public rendering's log positions are THE identity of the private chain's messages, for every
component symmetrically: the counterparty judge, the interop filter, the operator's OWN supernode
(which judges the rendering, never the private receipts — feeding it private positions would make
it invalidate counterparties' valid executing messages), relayers, and tooling. The private
chain's real receipts are never an interop source of truth for anyone.

**The rendering transformation (normative primitive, Karl 2026-08-30; block-level packaging
ratified same day).** The primitive is exposed at two levels from ONE implementation:
`RenderedLogs` (below) and `RenderBlock(private block + receipts) → rendered block content` (the
replay transactions plus the rendered log sequence). Both callers are IN-PROCESS: the builder's
write path, and — for the one consumer needing canonical positions at unsafe-head latency, the
sequencer's interop filter — the transformation applied inside the filter's ingestion (see "The
filter's own-chain integration" below). There is no RPC service in front of it.
Boundary, stated plainly: a pure transformation cannot produce execution-derived fields (the
rendering's stateRoot, hence its blockHash), so a caller reporting on a private block gets
PRIVATE identity + RENDERED content; that is sound because message validity never touches block
hashes (positions and log hashes only), and the rendering's real identity comes from the
RENDERING NODE the builder reads anyway (component 3: it supplies the span's parent check and the
channel-ID seed). One pure function, defined once and used by every component that needs the
public view of a private block:

    RenderedLogs(block, emitterSet) = the block's logs restricted to emitterSet,
                                      in their original order, re-indexed 0..k-1

where emitterSet is a set of (address, topic0) FILTERS (refined 2026-08-30, builder-lane
finding): {(L2ToL2CrossDomainMessenger, SentMessage), (CrossL2Inbox, ExecutingMessage)} ∪ the
genesis-configured extra emitters (any topic). The pair's generated rollup config is the sole
runtime source of this set: the batcher, interop filter, resolver and devstack all read the same
value, and no process accepts an independent emitter override. Topic-filtered, not purely
address-based, because the set defines which logs are CLAIMS: the messenger also emits bookkeeping
events
(RelayedMessage on every import) which are not claims — and since stock interop treats ANY log
as a potential initiating message, rendering one at a replayer's address would be a publicly
consumable message at the wrong address, a broken claim rather than harmless noise. Within the
filtered set there is still no per-message selection: every SentMessage is exported, every
ExecutingMessage is public. A rendering block's log sequence IS `RenderedLogs` of the
corresponding private block; the k-th rendered log has rendering log index k (the "relative log
index"). Consumers: (1) the builder — it constructs the replay transactions in exactly this
order; (2) the OPERATOR-SIDE supernode and interop-filter backend — they seed the private
chain's message database by applying the transformation to real private blocks directly, giving
canonical positions at unsafe-head latency with no batch/derive round trip; (3) identifier
resolution in tooling and the devstack. The standing equivalence invariant, asserted live in
testing: for every block, `RenderedLogs(private block)` equals the derived rendering block's log
sequence exactly.

## Replay transactions

Ordinary signed EIP-1559 transactions from the chain's standard batcher EOA (nonce-sequential,
zero fee cap and zero tip). There is no second private-interop operator key: the L1 transaction
carrying the channel and the L2 transactions inside it use the same standard batcher signer, whose
rotation remains `SystemConfig.setBatcherHash`. The rendering genesis base fee and minimum base fee
are zero, so a newly rotated batcher needs no rendering-chain premine. Rendering genesis uses the
execution protocol's maximum gas limit (`2^63-1`). The builder reserves half of its EIP-1559 target
for deposits/upgrades and refuses synthetic transactions whose declared gas exceeds the other half;
consequently the base fee remains zero. This fee policy applies only to the
synthetic rendering transactions — the outer channel transaction still pays ordinary L1 fees.

**Shared-chain-ID replay caveat.** Because the private chain and rendering deliberately share a
chain ID, a signed transaction is not cryptographically domain-separated between the two ledgers.
This is true in both directions. In particular, a published rendering import transaction targets
the stock `CrossL2Inbox` that exists on the private chain too, so including that raw transaction on
the private chain could create a spurious `ExecutingMessage` and consume the batcher's nonce. The
private sequencer has no obligation to accept public transactions and MUST reject raw rendering
transactions; conversely, the derived-only rendering has no mempool and its batcher MUST include
only transactions synthesized by this builder. This is an operational boundary, not an on-chain
replay-protection guarantee. The two halves' private-interop address maps must not converge without
adding explicit transaction-domain separation.

The replay transactions have two kinds, in a **normative deterministic order** so the batch is a
pure function of private-chain data — `RenderedLogs` order, which is the private block's OWN log
order with exports and imports interleaved exactly as they occurred (an earlier draft said
"exports then imports"; that grouping contradicts the canonical-position invariant below and is
superseded) — one public block per private block at the same height and timestamp:

**Message admission and replay gas (normative).** A sequencer has
no obligation to include every submitted private-chain transaction and may reject an oversized
message before inclusion. Once a private block contains a selected `SentMessage`, however, its
export replay is mandatory: omitting it, or allowing its replay transaction to revert, removes a
rendering log and renumbers every later rendered log in that block. This is not the ordinary
retryable `relayMessage` failure case. Retrying delivery or calling `resendMessage` creates a later
log; neither repairs the missing log at its original canonical position.

Accordingly, every `SentMessage` admitted to the private chain MUST be renderable within the public
chain's configured block-gas policy. The private messenger MUST enforce a protocol-level maximum
message size so this property does not depend only on discretionary sequencer censorship (which
cannot reliably detect messenger calls nested inside arbitrary contracts). The rendering builder
MUST derive export-replay gas deterministically from the encoded message length and MUST retain a
larger hard ceiling as defence in depth. The enforced ceiling is 64 KiB and export gas is the
configured base plus 28 gas per message byte. These conservative policy values still require
measurement before a production deployment.

1. **Export replay**: calls the batch-authenticated replay implementation installed at the
   L2ToL2CrossDomainMessenger predeploy address in the public genesis; it emits a `SentMessage`
   event byte-identical to the private chain's (same topics, same payload). Emitter address and
   event shape therefore match what every stock consumer expects; the public position (block,
   log index, timestamp) IS the message's identity, since the public chain is the chain's
   identity.
2. **Import replay**: calls the stock `CrossL2Inbox.validateMessage` with the standard checksum
   access list, at the same timestamp the private chain consumed the message. The stock judge
   validates it against the counterparty chain's own message database — a fabricating operator's
   imports are still caught, exactly as before; expiry and same-timestamp cycle handling are
   stock because the executing log has a real position.

## The claim-chain framing and the range claim (REINSTATED, Karl 2026-08-30, final)

**The rendering is a chain of claims.** Every transaction on it is a metadata transaction about
the private chain: each replay transaction claims "the private chain exported this message" /
"imported that one," and the range claim claims "the range ended with this identity, backed by
data at this content hash." There is ONE class of content (claims) with ONE
authenticator per mode: in v1 the batch submitter's signature authenticates the whole claim-set
at once (it covers every blob, hence every transaction, hence every claim); in proven mode a
proof attests the SAME total claim-set through one commitment over the batch content, checked by
the data-source gate at derivation (see "Proven mode" below) — the messages and the range claim
inherit validity transitively, with no per-claim binding. Upgrading trust is swapping the
authenticator over an unchanged claim surface: same wire, same record, same transactions.

**The claim transaction (all modes; FINAL SHAPE, Karl 2026-08-30 late): the LEADING transaction
of a range describes, commits to, and (in proven mode) proves the series of claims that follows
it.** The FIRST transaction of range N's first block posts range N's OWN `RangeClaim` to the
ClaimRegistry predeploy:

```solidity
struct RangeClaim {
    uint8   version;                  // exactly 1
    uint64  firstBlock;               // this range, inclusive — the range this tx OPENS
    uint64  lastBlock;
    bytes32 privateTerminalBlockHash; // the PRIVATE chain's hash at lastBlock — a known past
                                      // fact at build time (the private range completes before
                                      // anything renders), so leading placement is non-circular;
                                      // restores the ZK-era prevRoot→newRoot chaining shape
    bytes32 privateTerminalParentHash;// that block's parent hash. Added 2026-08-31 so the public
                                      // supernode can serve a COMPLETE private L2BlockRef from
                                      // public data alone — see "Operator topology".
    bytes32 l1Head;                   // RATIFIED SEMANTICS (2026-08-31): the terminal block's own
                                      // L1 origin — DERIVED, never operator-supplied. With
                                      // origin-copy this is the newest L1 the range actually
                                      // consumed, needs no live-L1 access to produce, and is
                                      // VERIFIABLE: the rendering's terminal block carries the
                                      // same origin, so a claim cannot name L1 the range never
                                      // saw.
    bytes32 rollupConfigHash;
    bytes32 depSetHash;
    bytes32 privateDataHash;          // content address of this range's full private input — a
                                      // COMMITMENT, not a pointer: nothing publishes the object
    bytes   proof;                    // EMPTY in v1 (registry rejects non-empty); in proven mode
                                      // attests the claim series: public values = the ordered
                                      // claim-set hash + the private-hash chain + config hashes
}
```

What changed from the earlier (trailing) envelope and why:
- **Leading, describing its own range**: possible because nothing in the claim references the
  rendering's identity — the RENDERING hash is gone from the claim entirely (its only consumers,
  the span parent check and audit, live at the batch layer / in the chain itself). The one-range
  lag and the genesis edge case both die: range 0 opens with its own claim.
- **The registry emits NO event** (log-less: contiguity check + storage update only; the calldata
  is the record, read by scanning transactions to the registry). A leading rendering-only log
  would shift every message's log index in range-opening blocks and break the canonical-position
  rule; log-less placement keeps rendering indices equal to RenderedLogs ranks with zero
  exceptions.
- **A per-range PRIVATE terminal commitment is now published** (one 32-byte hash per range). This
  deliberately supersedes the earlier leak-minimization exclusion, which was about PER-BLOCK
  private roots with no consumer; the per-range hash has three consumers — the proof chain, the
  supernode follow module (which serves it as the claimed private head, and against which a
  diverged local chain raises a monitoring alert), and a recovery or audit replay's byte-check
  against its own derived head.
- **Proven-mode transport is settled**: the proof rides INSIDE the batch as this ordinary leading
  transaction — chain-durable, trivially in scope for the data-source gate (parse tx 0, verify
  natively, compare the batch's replay txs against the proven claim-set hash), FP-servable via
  blobs. No carry-on blob, no ProofPosted event, no separate L1 object. No circularity: the proof
  is over the REAL private data and the claim list, never over the rendered blocks that carry it.
- Registry rules otherwise unchanged: batch-authenticated, on-chain contiguity — AMENDED to
  `firstBlock > lastPostedLastBlock` (no overlap, no regression, FORWARD GAPS ALLOWED: a range
  whose opening block is invalidated-and-replaced never executes its claim, so a gap in the
  record is the self-documenting mark of a voided range, not an error) — and v1 rejects
  non-empty proofs.
  Codec rules unchanged: canonical ABI form, 64 KiB proof cap.

## Config hash convention (RATIFIED, 2026-08-31 — Karl delegated the pick)

The claim's two configuration commitments are computed from JSON, not from a bespoke binary
encoding:

    rollupConfigHash = keccak256( canonical JSON of the RENDERING's rollup config,
                                  as marshaled by op-node's rollup.Config JSON encoding )
    depSetHash       = keccak256( canonical JSON of the dependency set )

JSON rather than a new binary format because the marshaling already exists, is the form operators
already exchange (`rollup.json`), is cross-client readable, and needs no second spec to disagree
about. The rollup config hashed is the RENDERING's — the chain the claim speaks for and the chain
a public verifier holds — not the private chain's.

This closes the hole the Silhouette-era wire documented and never did ("the spec does not say WHAT
is hashed"). Both values are frozen configuration, so the batcher takes them as flags
(`--private-interop.rollup-config-hash`, `--private-interop.dep-set-hash`) rather than recomputing
them per range.

**Today the devstack injects the values directly**, computing exactly this convention inline at
pair construction (`keccak256(json.Marshal(...))` over the rendering's `rollup.Config` and over the
dependency set). A SHARED HELPER that both the devstack and an operator's tooling call — so that
"what a claim binds" has one implementation rather than one per caller — is a devnet-prep item, not
built here. Until it exists, an operator computing these by hand must reproduce the recipe above
byte for byte; a mismatch is a claim that commits to a config nobody else can name.

## Hardfork adoption on the rendering (constraint recorded 2026-08-31, genesis lane)

Two facts every future fork adoption must reckon with:

1. **Network-upgrade transactions are injected by stock derivation at fork-activation blocks** —
   deterministic on every verifier, never carried in batches, executed on the rendering like on
   any chain. The builder's byte-determinism is unaffected (batch bytes never contain them), and
   the private chain executes the same upgrade transactions at the same timestamp. BUT: (a) any
   upgrade transaction that emits logs would shift the absolute log indices of that block's
   replay logs — audit each fork's bundle for log emission before adoption, and state the
   canonical-position rule as "emitter-set-filtered equivalence" (upgrade-tx logs are never
   messenger/inbox logs, so the emitter-set view stays equal; absolute identifiers shift only if
   a bundle emits). **Finding (2026-08-31; unverified leads, re-derive at adoption time): newer
   NUT bundles DO emit — ~65-76 logs at Karst/Lagoon — and the count is HISTORY-DEPENDENT (the
   ConditionalDeployer's ImplementationExists branch skips constructor logs when the CREATE2
   target already has code; dep-set-shaped wrapper deposits; the CGT branch), so the pair's two
   halves can emit DIFFERENT counts at the same fork block. The rule that voids the class for
   identifiers: THE FORK-ACTIVATION BLOCK CARRIES NO MESSAGES — sequencer policy (no messenger or
   inbox transactions at the activation timestamp) enforced by a builder check; one message-free
   block per fork, and absolute-index correspondence never matters at the only blocks where the
   halves' log counts can differ;** (b) **a fork bundle that upgrades the L2ToL2CrossDomainMessenger predeploy
   would REPLACE the rendering's replay implementation with a stock messenger**, breaking every
   subsequent replay transaction. Adopting such a fork requires a design decision (there is no
   in-band way to re-install the replay impl; the rendering's three private-interop predeploys
   are deliberately outside the standard upgrade path already). Until then: the operator pins the
   fork schedule and audits bundles before adoption.
2. **Both op-deployer runs of a pair MUST pin the same `l1StartBlockHash`** — otherwise the two
   halves' genesis timestamps diverge and the builder can never emit a block at both the private
   chain's number and its timestamp (caught as a real 2-second mismatch by the rollup-config
   test, where the rule is recorded).

## Invalidation semantics (Karl question, 2026-08-30 — stated as a design rule)

Nothing ever invalidates the chain; validity failures pin the TRUST FRONTIER. The span batch's
only admission rule is the stock submitter-signature check — whatever the operator posts IS the
canonical rendering, and no verifier rejects a batch for semantic content (that would be a diff).
The claim carries trust, not derivation validity:

- **v1 (attested)**: the registry enforces structure at post time — version, EMPTY proof slot,
  and contiguity. Contiguity is load-bearing: range N+1's claim cannot post without range N's,
  so a missing or malformed claim is a visible, blocking on-chain gap, never something a
  consumer must detect by vigilance. The claim is an audit record, not a gate; cross-safety is
  unconditional under the attested trust model.
- **Proven mode**: direction ratified separately — batch admission gated IN DERIVATION by the
  data-source gate, which verifies the leading claim's proof natively and refuses to admit an
  unproven range's data at all. See "Proven mode (v2, unbuilt)" below, which owns this question
  and also records the shapes considered and set aside. Nothing in proven mode is on the v1 path.

## Operator topology (RE-RATIFIED, Karl 2026-08-31: claim-driven private safety — no alt-DA)

The leading claim transaction makes alt-DA mode on the private side redundant: the public batch
itself carries the private chain's safety information (`privateTerminalBlockHash` per range). The
private chain's SAFE label is CLAIM-DRIVEN: a range is safe once its claim has landed in an L1
batch the rendering has derived. Reorging the private chain below a landed claim would contradict
the operator's public commitment, which is exactly what a safe label must anchor.

What happens when the local chain DISAGREES with a landed claim was re-ratified the same day, and
the answer is snap-to-commitment, not fail-stop: the claim is the operator's binding public
statement, so the module serves it verbatim and a diverged sequencer's own stock consolidation
force-resets ONTO it — recovery, not a halt. "The chain diverged from its claims" is a MONITORING
alert, and a claim naming a block that exists nowhere stalls loudly in sync. See "The supernode
follow module" below; the earlier withhold-latch rule is superseded there.

**Components (NORMATIVE NUMBERING — code comments cite these numbers).** Four, and no new binary
in any of them: one LightCL, one batcher, one rendering node, one supernode.

1. **Private sequencer**: stock op-reth + STOCK LightCL, whose follow-source URL points at the
   supernode's claimed route (component 4) instead of a same-chain CL — "the slightly different
   thing it queries in private mode". No op-node changes.
2. **op-batcher with `--private-interop` flags** — the builder. It renders, computes the range's
   full private derivation input and commits to its keccak as the claim's `privateDataHash`, and
   posts ONE public batch tx per cadence. The object is hashed and the bytes dropped — nothing
   publishes them (ratified decision 5). **The private-data object format (RATIFIED): `0x00 ‖
   stock channel frames` over ONE span batch of the private blocks, deterministic channel ID** —
   consensus-relevant THROUGH THE HASH even though the batcher never emits it: it is what anyone
   holding the private blocks reproduces to check a claim, and it is derivation-shaped, so a
   replica or recovery driver replays it through stock machinery and verifies the executed result
   against the claim's terminal hash. There is no separate batching binary: everything the
   terminal seam needs is a flag on the stock service.
3. **Rendering node**: stock op-node + op-reth on the public config. It is the builder's
   parent-check follower (component 2 reads the previous range's terminal hash and the operator's
   nonce off it) and the follow module's source. It may be a dedicated node or the supernode's own
   public route — read-only public data either way, so no coupling is created by sharing it.
4. **The public supernode**: judges the dependency set (the sequencer's interop filter dials it
   for import gating — the private side's only cross-chain touchpoint), AND hosts the follow
   module that serves the claimed private view at `<base>/<chainID>/claimed`. It is pure public:
   it reads only its own derived data and holds no private credentials.

(A fifth component used to be listed here, an S3-compatible object store, and a sixth, a
standalone claim-follower sidecar. Both were deleted 2026-08-31 — see "Private data availability"
and "The supernode follow module" below. The numbering above is post-deletion and final.)

**Private followers (FINAL, Karl 2026-08-31): 100% stock LightCL + op-reth, and the claim stays
HASH-ONLY.** The follow endpoint must serve complete L2BlockRefs (verified first-hand:
followUpstream hash-checks each served ref's L1 origin against real L1, and consolidation is
full-struct equality). The claim publishes the two fields that cannot be derived —
`privateTerminalBlockHash` and `privateTerminalParentHash` — and origin-copy supplies the rest
from the rendering block at the same height, so the module completes every ref from PUBLIC data
alone (see "The supernode follow module"). Leak minimization stands: the private chain's block
BODIES are never published, and they do not need to be — every legitimate consumer of the private
chain's forkchoice refs runs INSIDE the private network by definition, outsiders follow the
rendering. A follower is a stock LightCL pointing its follow-source at the claimed route, plus a
stock op-reth EL-syncing block bodies from private peers (FCU to the claimed head, backfill,
execute — a follower gets exactly the claimed chain or nothing). There is no separate FCU-driver
replica mode: the stock LightCL is the replica driver. NOBODY EVER ATTACHES A PRIVATE EL TO ANY
SUPERNODE — the operator's supernode is identical to a stranger's. Followers advance at claim
cadence by design; the private p2p peer set IS the privacy boundary (static peers/allowlist).

**Ops rule: DISTINCT ROUTES.** The claimed view is served at its own route
(`<base>/<chainID>/claimed`), never by flipping the meaning of the existing per-chain route.
Flipping saves nothing (same code either way) and turns a mispointed consumer's failure from a
loud 404 into plausible-looking refs of the WRONG CHAIN — which a sequencing LightCL force-resets
onto. Consumer inventory, for what reads what: the interop filter reads the SEQUENCER NODE
directly and transforms positions in-process at ingestion (admission gating needs unsafe latency —
see "The filter's own-chain integration"); the LightCL reads the claimed route; the builder reads
the rendering EL. No private-side component consumes the supernode's PUBLIC view of this chain.

**Private data availability (FINAL, Karl 2026-08-31): the operator's P2P network, and the store
is DELETED.** The store had already demoted to disaster recovery and third-party audit; the
finding that closed it is that NOTHING EVER READ AN OBJECT BACK. The supernode follow module
serves refs from its own derivation, the interop filter reads the sequencer node in-process, and
private blocks reach every legitimate reader — the operator's own followers, all of whom live
inside the private network by definition — over the firewalled p2p network. An S3 bucket in the
batcher's critical path was a dependency that could stall the chain and served no reader, so the
package, the `--private-interop.store-*` flags and the upload-before-publish gate all go.

What survives is the COMMITMENT. `privateDataHash` stays in the claim: the builder encodes the
range's full derivation input in the ratified `0x00 ‖ stock frames` format, commits to its keccak,
and drops the bytes. So the recovery and audit stories are unchanged in kind and changed in
custody: anyone holding the private blocks — a follower, or an operator's own archive — replays
the derivation-shaped object in range order and checks each range's executed result against its
claim, and the claim is still the read-side authority for what the right object is. An operator
who wants an off-chain backup runs an EXTERNAL SIDECAR against its own private node; that sidecar
is out of scope for this design and for the batcher process.

**The supernode follow module (RATIFIED, Karl 2026-08-31, final round): the follower binary is
DELETED; the public supernode serves the follow endpoint from public data.** Three amendments
make it possible, all blessed: (1) **origin-copy** — the batcher's transformation reuses each
private block's OWN L1 origin (read from its L1-info deposit) as the rendering block's epoch,
instead of independently re-deriving origins; private and rendering origins (and therefore
sequence numbers) are equal by construction, and the builder's origin-selection/L1-view
machinery is deleted. (2) The claim gains **privateTerminalParentHash** (one field). With both,
every field of the six-field L2BlockRef the follow protocol demands is public: hash+parentHash
from the claim, number/timestamp/origin/seqNumber from the supernode's own rendering block.
(3) **Snap-to-commitment replaces the withhold latch**: the module serves claims verbatim; a
diverged sequencer's own stock consolidation force-resets it onto the publicly claimed chain —
the claim is the operator's binding statement, so automatic recovery TO it is correct — and a
claim naming a block that exists nowhere fail-stops as a loud unfindable-hash sync stall.
"Chain diverged from its claims" becomes a MONITORING alert, not a serving gate; the ratified
"divergence never self-clears" posture is superseded. The module lives in op-supernode behind
dormant flags at a distinct per-chain route; it reads only the supernode's own derived data and
holds no private credentials. The batcher framing, restated per the design owner: the "builder"
IS the stock batch submitter — stock lifecycle walking the private chain block-for-block — with
one per-block transformation: strip private txs, insert the claim tx (range-opening block), and
insert the replay txs; block progression is never altered.

**Sharp edge (recorded 2026-08-31, module lane): a DISABLED follow module's route is not a 404.**
The supernode's per-chain handler mounts its root JSON-RPC at "/", so with the module disabled,
`<base>/<chainID>/claimed` falls through and answers `optimism_syncStatus` FOR THE RENDERING —
plausible-looking refs of the wrong chain, the exact class the distinct-route ruling exists to
prevent, living in the unconfigured case. Ops rule until fixed: ENABLE THE MODULE BEFORE POINTING
ANY LIGHTCL AT THE ROUTE (the consumer-side backstop: a private LightCL fed rendering refs
force-resets toward a hash no private peer holds and stalls loudly in sync — bad day, not silent
corruption). Follow-up recorded: unregistered sibling sub-paths should 404 — plausibly an
upstream oprpc fix, the third entry on the upstream-pitch list.

**The filter's own-chain integration (FINAL, Karl 2026-08-31): in-process transformation.** The
sequencer's interop filter reads the RAW private node — a fully self-consistent real chain, all
verification ON — and applies `RenderedLogs` itself between fetch and LogsDB insertion: entries
stored at RENDERED indices under the block's REAL private hash ("the real L2 hash, but only the
right logs at the right indexes"). This supersedes the brief view-plus-trusted-source detour:
that shape existed to keep the filter untouched, but the filter is modified either way, and
in-process transformation needs no trust flag, no header rewriting, and gives the phantom-hash
class nothing to exist on. (The detour's finding stays recorded as an upstream-pitch candidate
alongside the relayETH CGT deny: stock `TrustRPC` gates verification but not identity derivation
— the by-number accessors derive the hash from header fields regardless of trust.) With the
detour dropped, the rendering-view RPC service it needed was deleted too: no component dials a
rendered view of the private EL, and `RenderedLogs` is called in-process by everything that
needs it.

**Position-preserving padding: REJECTED (Karl, 2026-08-31).** Filling index gaps with commitment
logs to unify the two index spaces was evaluated and declined — the transformation is the
accepted design; the rejection is recorded so the idea is not re-derived without new evidence
(its open dealbreaker when abandoned: same-chain references at padded positions diverge between
the operator's private-data view and the public judge's).

## Division of checks

| Check | Where | Stock? |
|---|---|---|
| Attestation (submitter signature) | op-node inbox filter | stock |
| Batch structure, parent check, origins, timestamps | op-node span-batch validation | stock |
| No deposits ever | L1 portal reverts + batch validation drops 0x7E | stock + 1 contract |
| Import validity, expiry, cycles | cross-safety judge over real receipts | stock |
| Export serving to counterparties | message DB from real receipts | stock |
| Public block execution, roots, hashes | op-reth | stock |
| Replay faithfulness to the private chain | v1: attested (unchecked, by design); v2: the proof | operator / shelf |
| Claim structure + contiguity | the ClaimRegistry, at post time | ours (small) |
| Claimed private terminal hash vs the local private chain | the supernode follow module serves the claim verbatim; divergence is a MONITORING alert, and a diverged sequencer snaps back to the claim | ours (small) |
| Full-input integrity | the claim's `privateDataHash` = keccak of the re-encoded range, checked by whoever holds the private blocks | ours (small) |

The v1 trust model is unchanged in substance: the operator can render a chain that lied, and the
counterparty-checked import path is what a lying operator still cannot fake. The
fabricated-export-is-accepted test must survive the retarget as a passing test.

## Proven mode (v2, unbuilt)

**Status:** direction RATIFIED (Karl, 2026-08-30). NOTHING HERE IS ON THE v1 PATH — in v1 the batch
submitter's signature is the verification, full stop. The exploratory survey behind the rulings
below is in git history.

A real proof fills the range claim's proof slot. The claim it attests: "the covered public blocks
are exactly the deterministic rendering of a valid private chain's messenger traffic" —
block-for-block correspondence keeps that simple. At settlement the superroot program derives the
public chain fully stock.

**The gate is an op-node DERIVATION RULE at the DATA SOURCE.** Single-chain validity belongs in
derivation: a supernode-only rule is invisible to a standalone op-node (which would derive
unproven ranges and fork from gated verifiers at the first void) and invisible to fault proofs.
The data source has the whole inbox transaction's blobs BEFORE anything derives, so it parses the
span, extracts the leading claim, verifies, checks that the range's replay transactions hash to
the proven claim-set, and on failure NEVER ADMITS the data. The prior art measured the seam at
~145 lines (karl/zk-validium's contract-inbox op-node changes patched exactly here). A MANDATORY
KONA TWIN comes with it: cross-client derivation parity is normative.

**Verification is NATIVE, not an eth_call.** The proof rides L1 with the batch and the gate
verifies it IN-PROCESS — the pattern this project already ran live (the pure-Go SP1 Groth16
verifier node-side, the sp1-verifier crate as the kona/circuit twin). A literal eth_call gate was
costed and set aside for three reasons: the fault-proof oracle has NO L1-state hint family at all
(headers/txs/receipts/blobs only, verified against kona's hint enum), so it would be
unreproducible by any proof program without new hint plumbing plus an L1-configured EVM
in-circuit; eth_call results are a function of an EVM the protocol does not pin; and any
state-pinned call drags in historical-state access. Native verification has none of those costs:
no gas, no on-chain call on the consensus path, determinism is the determinism of a pure function
over L1-committed bytes, and the FP program re-verifies natively (recursion, measured cheap on
this project's SP1 stack). The L1 CONTRACT still exists as the rule's socially-legible governance
home — the vkey registry whose rotations the node reads via receipts (publish-then-re-derive, the
FP-servable event pattern) — it is simply never CALLED.

**The proof bytes ride in the leading claim transaction** (settled with the claim's final shape).
Non-circular: the proof is over the real private data and the claim list, never over the rendered
blocks carrying it. Chain-durable, trivially in scope at admission, FP-servable via the blob hints
derivation already uses. Rejected transports, for the record: a carry-on blob under a non-0x00
version byte; a `ProofPosted` event from a recording contract; blob-tx calldata (gas-priced); the
permissioned inbox; anything storage-read (no FP hint family). Public values: the batch-content
commitment, the covered range, the terminal rendering block hash, and the config hashes — ONE
commitment over the batch content, so a valid proof cannot be replayed onto different bytes and
every claim inside inherits validity transitively, with no per-claim binding.

**Unproven past the window = the range publicly never happened**, and that is the design, not a
bug to engineer around. Two mechanical facts force it, both test-pinned: (1) a dropped or withheld
batch yields SILENCE until the sequencing window expires from the range's START epoch, at which
point the covered timestamps back-fill as deposit-only blocks (exactly one L1-attributes deposit,
zero messages) with NO re-post path afterward — a late re-post is refused by the lateness rule;
(2) Holocene derivation has no hold state — an "undecided" span is consumed and never revisited,
so an in-derivation gate is necessarily binary. "Wait for the proof inside derivation" is not a
shape stock derivation can express. Embracing the binary-ness re-implements at the rendering level
the FORCED-EXTENSION LIVENESS PHILOSOPHY this project ratified in its ZK era: a dead or slow
prover can degrade its OWN chain to empty blocks but can never stall the dependency set. The
window IS the proving deadline — at the ratified cadence the default 3600-L1-block window leaves
**~11.8 h per range** for proving plus posting. The private chain reorg-adopts the gap.

Atomicity falls out for free: one range = one span batch = one channel = one L1 transaction, so
the claim unit and the admission/invalidation unit coincide by construction. Registry consequence:
contiguity is `firstBlock > lastPostedLastBlock` — FORWARD GAPS ALLOWED — because a voided range's
claim never executes and a strict rule would wedge the next honest claim; a gap in the record IS
the mark of a voided range.

**Recorded, not chosen:**

- *The lossless fallback* — supernode-side frontier pinning (verdict behind `IsDenied`, chain
  derives, cross-safety waits) via the already-merged SuperAuthority `FullyVerifiedL2Head` seam:
  zero op-node/kona diff, same voided end-state, at the cost of the rule being node-external —
  not a derivation rule, unenforceable by a bare op-node, invisible to proof programs. Reach for
  it only if the data-loss cliff is ever unacceptable.
- *The free intermediate ("v1.5")* — withhold-until-proved. On a derived-only chain with no public
  sequencer, an operator simply NOT POSTING an unproven batch produces identical chain-level
  semantics with zero code anywhere, and the window enforces the same deadline. The gate's whole
  delta over operator scheduling is WHO ENFORCES: verifiers refuse unproven ranges even from a
  malicious operator. That is precisely the v1→v2 trust delta — which means nothing about v1 needs
  to anticipate the gate mechanically.

**Costs, stated honestly:** the op-node data-source diff, the kona twin, and the settlement program
reproducing the gate (cheap under SP1; heavy for a cannon-style FPVM — this design assumes the SP1
settlement path). The data-loss cliff is real: a prover slower than the window burns the range to
deposit-only and forces private-side gap adoption, so window sizing is a genesis decision (budget =
range duration + proving + posting + margin, against the reorg exposure of a large window). And a
permanent constraint worth restating: kona has no alt-DA derivation, config parsing only — alt-DA
was never available as the gate's transport and never will be for proving.

## Testing

**Ratified goal (Karl, 2026-08-30): PLUG-IN REPLACEMENT.** Existing devstack/acceptance suites run
UNCHANGED against a private-interop pair; single-chain suites run against the private chain
directly; a small private-interop-specific suite covers what is new or flips. "Unchanged" is
literal — a test that had an option added to its constructor is a test that was changed, and it
proves only that the option compiles.

**The identifier seam is the one thing tests cannot be naively oblivious to.** Cross-chain message
identifiers are RENDERING positions: same block number and timestamp as the private chain, but a
different log index (the rendering carries only emitter-set logs) and a different transaction hash.
A test deriving an identifier from the private receipt would build a checksum the judge must
reject. This is fixed in ONE central place, never per test: a position resolver registered through
`txintent.RegisterPositionResolver`, keyed by the private chain's ID, from the one place that holds
both halves of the pair. It is torn down with the test, so a suite that never builds a pair has no
resolver registered and every stock chain's identifiers are minted exactly as before. Tests call
the same helpers and never learn the difference — that is what makes the run-unchanged matrix run
unchanged. Production relayers do the same thing naturally: they read the rendering.

**Entry points.**

- In a test: `presets.WithPrivateInteropChain()` on a two-L2 preset — the pair's second L2 becomes
  a private chain plus its rendering.
- Ambient, for running STOCK suites against a pair without editing them:
  `DEVSTACK_PRIVATE_INTEROP=true go test ./tests/interop/contract/...`. It is honoured only by
  presets that can actually build a pair; a preset that cannot is left alone rather than silently
  building something else, and a preset asked for something a pair cannot provide (the in-process
  interop filter, which the pair's runtime does not wire) SKIPS WITH A REASON rather than running a
  weaker test under a name that promises more. `DEVSTACK_PRIVATE_INTEROP_CADENCE` overrides the
  range cadence in private blocks per range.

**E2E environment recipe** (the pair needs a real op-reth on both halves):

    DEVSTACK_L2EL_KIND=op-reth
    RUST_BINARY_PATH_OP_RETH=<abs path to a built op-reth>   # the path must exist; skips the build
    FOUNDRY_DENY=never                                        # quirk: leave it unset and the run
                                                              # trips foundry's deny path

**What the acceptance suite covers.** Two private-interop-specific tests today:

1. **The pair comes up** — the topology's own smoke test: both halves exist, are the two chains
   they are supposed to be (one chain ID, two chains — ratified decision 6, and the reason nothing
   is peered across the boundary), and the rendering starts advancing. Deliberately separate from
   the message round trip, because almost everything that can go wrong with a pair goes wrong
   before any message is sent — a genesis mismatch, a follow source that never answers, a builder
   that cannot resolve its parent check — and a failure here says WHICH.
2. **Messenger, both directions** — the gate. A messenger message crossing INTO the private chain
   and back OUT, through the whole pipeline. The asymmetry is the point: inbound is stock
   `interop/contract` with chain B swapped for a pair; outbound is the seam, written with the SAME
   stock helpers because the resolver centralises the position translation. The test additionally
   CHECKS the resolver by sending the outbound message at a position the private chain and the
   rendering deliberately disagree about, and asserting the identifier names the rendering's.

The standing invariant asserted live alongside them: for every block,
`RenderedLogs(private block)` equals the derived rendering block's log sequence exactly, and the
two chains correspond block-for-block in number and timestamp (hashes differ BY DESIGN and are
never compared).

**The one-command devnet gate (op-up).** Everything above is `go test`. There is also a
hand-driven gate, for the same reason op-up exists at all: one command that stands the whole thing
up and says whether it works.

    DEVSTACK_L2EL_KIND=op-reth RUST_BINARY_PATH_OP_RETH=<path to op-reth> \
      go run ./op-up --private-interop --smoke

`--private-interop` builds the ordinary two-L2 supernode devnet with `WithPrivateInteropChain`, so
chain B is a pair: L2B's RPC is the private chain and the endpoints printed under it are the
rendering, the private op-node's follow source, and the operator. `--smoke` then runs
`op-chain-ops/interopsmoke` IN THIS PROCESS against the nodes' own RPCs and exits with its result;
without `--private-interop` the same command runs the public control on two ordinary chains.

In-process is not a convenience. A message initiated on the private chain is named by its position
on the rendering, and that correction is made by the resolver described above, which the devstack
registers process-globally when it builds the pair. A smoke run from another process has no
resolver: it would quote raw private receipt positions, and the legs meant to prove the naming
works would fail or pass vacuously. The CLI therefore refuses the private-pair profile outright
rather than offering it.

What the profile does with each test, none of it silently: identity and transfer run unchanged
(transfer says which unit it moved, chain B being CGT); bridge is SKIPPED with reason —
SuperchainETHBridge is the closed path on a CGT chain, and NativeMintBridge, the sanctioned one,
is not exercised here; valid-message runs A→B as before PLUS the mirror leg B→A — a messenger
message sent on the private chain and executed on the counterparty, the one leg that makes the
resolver load-bearing and the heart of this gate (it goes through the messenger because the export
policy publishes the messenger's `SentMessage` and the inbox's `ExecutingMessage` and nothing
else: an EventLogger log has no public position at all, so a leg built on one would be a
fabricated-import test wearing a valid-message name); invalid-message is held to `b-to-a` — the
other directions are refused, since they land the invalid message on a chain whose blocks are
never replaced, and even `b-to-a` proves only that the counterparty rejects a message the
rendering does not carry, its init being an EventLogger log with no public position;
chained-invalid-message is refused: its cascade begins with chain B's block being replaced.

**Deliberately excluded, with reasons:**

| Suite | Why |
|---|---|
| `tests/base/deposit`, `tests/base/withdrawal`, `dsl/bridge.go` deposit paths | Deposits revert on the private chain by the resource-config gate (`maxResourceLimit=0`; the portal itself is stock — ratified). There is nothing on the private side for a deposit suite to assert. |
| `supernode/interop/eth_bridge` | The private half deliberately has no `SuperchainETHBridge` or `ETHLiquidity` implementation; its only ETH-denominated path is `NativeMintBridge`/`ETHLockVault`. |
| `interop/proofs*` (~25) | Fault-proof program fixtures; the rendering settles by a different (future) proof path. Out of v1 scope by ratified decision. |
| `interop/upgrade*` predeploy-introspection against the RENDERING | The messenger predeploy carries the replay implementation, so impl-slot equality assertions are wrong by design there. The private half is also specialized by removing the stock protocol ETH path. |
| Anything asserting the rendering's mempool or sequencer behaviour | The rendering has neither. |

**Traps that bite, carried forward and verified still present:**

- **P2P severance.** The private-side nodes and the rendering share a chain ID but are different
  chains in content. They must never gossip-peer, or a stock node re-gossips a conflicting history.
  `connectL2CLPeers`/`connectL2ELPeers` are explicit calls; the pair's runtime simply does not make
  them across the boundary, and that absence IS the severance.
- **Interop activation timestamp** is cluster-wide. If the rendering's verification start is not
  aligned with a block the message DB actually seals, the cross-safe frontier hangs behind a
  healthy-looking pipeline. Activation = anchor + blockTime.
- **Emitter-set drift.** A mismatch silently renumbers message positions. The filter and builder
  accept no independent emitter flags: both load the set committed by the genesis intent into the
  generated rendering rollup config. The private-pair devstack wires the filter with that same
  config so acceptance runs exercise the in-process transformation.
- **One rollup config per network handle.** The handle exposes the PUBLIC rollup.json; the private
  config exists only inside the private CL's construction. Tests reading `Escape().RollupConfig()`
  get public values — block time and genesis are identical by construction, and the preset asserts
  that once.
- **Startup cycles.** Anything reading the private chain over RPC at construction forces
  construct-last ordering; follow-source cycles need the stable `tcpproxy` indirection.

## Spike inventory (2026-08-30, all green, test files removed after capture)

Only #2 is load-bearing for the design as it now stands. The other three were run against shapes
this design subsequently dropped (the skipped-blob envelope; Alt-DA on the private side) and are
kept as verified knowledge about the stock stack, not as requirements.

1. **Skip bytes** — 7 tests / 22 subtests: unknown-byte blobs harmless in all configs; `0x00`
   unsafe; `0x01` context-dependent; calldata inbox txs still work post-Ecotone. *(No longer used:
   nothing nonstandard rides the batch transaction.)*
2. **Span batches** — 14 tests: 300-block empty and mixed spans accepted end-to-end (including
   the Holocene stage path, safe head advanced 300 blocks); parent check semantics and
   drop-not-reset confirmed; all limits measured. *(These are the batch-construction rules above.)*
3. **Alt-DA finality** — 6 tests: zero windows rejected at startup; window=1 floor; finality
   formula quirks mapped. *(No longer used: no component of this design runs Alt-DA.)*
4. **kona config parse** — 2 tests: the `alt_da` block parses and round-trips losslessly in the
   Rust tree; field spellings pinned. *(No longer used, same reason.)*
