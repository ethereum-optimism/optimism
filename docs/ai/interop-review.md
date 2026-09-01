# Interop Review

Method and knowledge base for reviewing material changes to interop protocol code, Go or
Rust. Pairs with the `interop-reviewer` agent (`.claude/agents/interop-reviewer.md`);
under a harness without that agent, work through this guide directly.

Protocol source of truth: the [specs repo](https://github.com/ethereum-optimism/specs),
directory `specs/interop/`. Spec citations below are paths in that repo. Where the tree
and a spec disagree, that is a finding (spec drift), not something to reconcile silently.

## Invariants and design decisions registry

### Message validity

Spec: `specs/interop/messaging.md` (Messaging Invariants), `specs/interop/derivation.md`
(Invariants).

- **Timestamp invariant**: initiating timestamp `<=` executing block timestamp — equal
  timestamps are valid at consensus level (cycles handled separately). Spec also requires
  the initiating timestamp to be greater than the activation timestamp.
- **Activation invariant** (both sides): neither the executing nor the initiating block
  may fall in its chain's activation block; the activation block contains only
  deposit-type transactions and its logs are not valid initiating messages
  (`specs/interop/derivation.md`, Activation Block). Implementations enforce this as
  `timestamp >= activation + block_time` on *both* chains — `verifyExecutingMessage` in
  `op-supernode/supernode/activity/interop/algo.go` and `check_single_dependency` in
  `rust/kona/crates/protocol/interop/src/graph.rs` must agree. The formulations differ:
  Rust reads each chain's own config (`is_interop_active` plus `is_first_interop_block`,
  `rust/kona/crates/protocol/genesis/src/rollup.rs`), while Go uses a single global
  activation timestamp with per-chain block time (`algo.go`). They agree only while
  (a) every chain's Lagoon timestamp is equal — op-supernode refuses to start otherwise
  ("mismatched Lagoon activation timestamps", `supernode.go`) — and (b) activation is
  block-aligned. Latent divergence: per-chain dependency-set activation timestamps
  would silently diverge Go's global-field formulation.
- **Message expiry invariant**: invalid iff
  `init.timestamp + EXPIRY_TIME < executing_block.timestamp`, `EXPIRY_TIME = 604800`
  (7 days). Constants: Go `MessageExpiryTimeSecondsInterop`
  (`op-core/interop/depset/static_depset.go`), Rust `MESSAGE_EXPIRY_WINDOW`
  (`rust/kona/crates/protocol/genesis/src/interop/constants.rs`). These must stay equal.
  Known spec drift at this boundary, at two sites: the normative text is
  `specs/interop/messaging.md` § Message Expiry Invariant (invalid iff
  `id.timestamp + EXPIRY_TIME < executing_block.timestamp`); both messaging.md's own
  invariants-summary bullet ("MUST be lower than … + `EXPIRY_TIME`") and
  `specs/interop/derivation.md`'s bullet (`t > execution_timestamp - expiry_window`)
  restate it one second stricter. The normative section wins; both Go and Rust implement
  its boundary, accepting `executing_timestamp == init.timestamp + EXPIRY_TIME`.
  Tracked upstream as
  [ethereum-optimism/specs#931](https://github.com/ethereum-optimism/specs/issues/931).
- **ChainID invariant**: the initiating chain must be in the executing chain's
  dependency set (`specs/interop/dependency-set.md`).
- **Existence matching**: the executing message must match an actual initiating log.
  Mechanics differ by implementation — checksum-equality against an indexed logs store
  (Go verifier, via `ChecksumArgs`/`MessageChecksum` in `op-core/interop/messages`,
  type-3 access-list checksum per `specs/interop/predeploys.md`) versus full
  origin/payload-hash/timestamp equality against fetched receipts (proof side,
  `graph.rs`). Different mechanics, same accept-set — that equivalence is exactly what
  parity review protects.
- **Same-timestamp messages and cycles**: same-timestamp dependencies are valid but must
  not form a cycle (`specs/interop/messaging.md`, Intra-block messaging). Cycle
  detection is Kahn's algorithm over same-timestamp executing messages in both the Go
  verifier (`op-supernode/supernode/activity/interop/cycle.go`) and the proof side
  (`check_cycles` in `graph.rs`). Same-timestamp existence resolves against the current
  frontier view, older timestamps against accepted history.

### Verification and safety

Spec: `specs/interop/verifier.md`, `specs/interop/messaging.md` (Resolving cross-chain
safety).

- **Timestamp-lockstep verification** (design decision, both live verifiers): one global
  timestamp cursor advanced across all chains in lockstep, rather than per-chain
  frontiers with a transitive hazard closure. Older-timestamp dependencies are already
  verified by construction; same-timestamp dependencies use the frontier view plus the
  cycle check. Consequence: the slowest chain gates all chains; a change reintroducing
  per-chain frontiers must bring back a transitive-validity mechanism.
- **Safety ladder — concept boundary**: `cross-unsafe` in `specs/interop/verifier.md` is
  an *input safety level* concept. As a **chain-head label** it is not implemented —
  there is a single unsafe head; verifiers promote local-safe blocks directly to
  verified/cross-safe. As a **message-validation threshold** it is live: the tx-pool
  filter's `minSafety` parameter (`safety.Level` in `op-service/eth/safety`,
  `SafetyLevel` in `rust/op-alloy/crates/consensus/src/interop.rs`) accepts
  `local-unsafe` and `cross-unsafe`. Never conflate the two when sweeping or renaming.
- **Head-ordering invariant**: `unsafe >= local-safe >= cross-safe >= finalized`, and a
  block must be cross-safe before it can be finalized. Enforcement splits by provenance,
  all in `op-node/rollup/engine/engine_controller.go`: opinions from the outside
  authority are *clamped* to local knowledge (`resolveVerifiedAsSafe` and
  `resolveVerifiedAsFinalized` fall back to the local head with a warning;
  `resolveAnchorAsSafe` clamps the same way, silently by design), internal out-of-order
  writes are *rejected* (`promoteFinalized` refuses both finality rewinds and
  finalizing a block past the safe head), and the cross-safe fallback *floors* at
  finalized — never local-safe, never below finalized (`crossSafeFallback`). The
  supernode's authority layer (`super_authority.go`) deliberately does not enforce
  ordering: it supplies verifier heads and consults local-safe only for activation
  gating. The Rust-side design intent is the same invariant by construction: cross-safe
  moves only via promotion clamped `>= finalized`, and finalized is clamped
  `<= cross-safe`. Both symmetric mistakes are findings: adding ordering enforcement to
  the authority layer, or turning the reject path into a clamp.
- **Crash-safety model** (design decision, Go verifier; binding for lokahi): decisions
  are written to a WAL before any side effect; everything the apply step needs is
  captured *while the affected block is still canonical*, so recovery never consults the
  live EL; applies are idempotent under replay.

### Block replacement

Spec: `specs/interop/derivation.md` (Replacing Invalid Blocks), Holocene engine-queue
rules.

- A block with an invalid executing message must not become safe and is replaced by a
  **deposits-only block at the same height** — same attributes, transaction list trimmed
  to deposits. The replacement is therefore *fully determined by the invalidated block*.
- Trimming primitives: Go `AttributesWithParent.WithDepositsOnly`
  (`op-node/rollup/derive/attributes_queue.go`), Rust
  `OpAttributesWithParent::as_deposits_only`
  (`rust/kona/crates/protocol/protocol/src/attributes.rs`). These must produce
  equivalent blocks from equivalent inputs.
- Replacement cascades: blocks whose executing messages depended on a replaced block are
  themselves replaced (recursively) — `specs/interop/fault-proof.md`, Consolidation.
- The invalidated block's output (state root, message-passer storage root) must remain
  reconstructable after replacement — the superroot's *optimistic* branch depends on it
  (`specs/interop/superroot.md`). Implementation: the deny record in
  `op-supernode/supernode/chain_container/invalidation.go` (`DenyRecord` carries payload
  hash, decision timestamp, state root, message-passer storage root; served via
  `LastDeniedOutputV0`/`GetOutputV0`). Failure mode to watch: a missing archive record
  is indistinguishable from "never denied" — `OptimisticOutputAtTimestamp`
  (`chain_container.go`) falls through with no error and no log and returns the
  *replacement* block's output as the optimistic root. Changes near this path must not
  widen that silent window.

### Superroot

Spec: `specs/interop/superroot.md`. `SUPER_ROOT_VERSION = 1`; chain outputs ordered by
chain ID. The canonical Go encoder is `op-service/eth/super_root.go`
(`SuperRootVersionV1`, `SuperV1.Marshal`, `NewSuperV1` — which sorts by chain ID);
`op-service/eth/superroot_at_timestamp.go` is only the JSON RPC response schema, not
the encoding. Rust: `SuperRoot` in `rust/kona/crates/protocol/interop/src/root.rs`
(constructor sorts by chain ID); a Go/Rust wire-schema mirror pair also exists in
`rust/kona/sp1/` (super-range-executor / proposer). Serving side: Go
`op-supernode/supernode/activity/superroot/`. Encoding and ordering must agree
byte-for-byte — output roots feed fault proofs.

### Sequencer / tx-pool admission

Spec: `specs/interop/sequencer.md`, `specs/interop/tx-pool.md`. Admission filtering is
**policy, not consensus**: the filter may be stricter than the consensus accept-set
(rejecting messages that a verifier would accept), never looser in a way that admits
consensus-invalid blocks without the invalidation path as backstop. The EL-side filter
is `op-interop-filter` (queried by op-geth and by op-reth's tx-pool filter client,
`rust/op-reth/crates/txpool/src/interop_filter/`), monitored by `op-interop-mon`.

## Cross-implementation parity map

Assumes the post-cleanup tree (op-supervisor-era types and the cross-unsafe chain-head
notion removed). Verify rows against the tree at review time; update this table when
lokahi phases land a Rust node-side counterpart.

| Rule | Go node-side | Rust node-side | Proof side | Tx-pool filter |
| --- | --- | --- | --- | --- |
| Message validity (timestamp/activation/expiry/existence) | `op-supernode/.../interop/algo.go` `verifyExecutingMessage` | pending (lokahi; rules to be shared via kona-interop) | `graph.rs` `check_single_dependency` | `op-interop-filter/filter/lockstep_cross_validator.go` `validateMessageTiming` + log-existence check |
| Same-timestamp cycle detection | `op-supernode/.../interop/cycle.go` (Kahn) | pending (lokahi) | `graph.rs` `check_cycles` (Kahn) | none — intentional (see registry) |
| Expiry constant (604800s) | `op-core/interop/depset/static_depset.go` | `rust/kona/crates/protocol/genesis/src/interop/constants.rs` | same Rust constant | `messageExpiryWindow` config, fed from depset |
| Executing-message extraction from receipts | `op-core/interop/messages` (`DecodeExecutingMessageLog`) | `kona-interop` `message.rs` (`extract_executing_messages`) | same Rust functions | Go extraction via op-core |
| Access-list entries / type-3 checksum | `op-core/interop/messages` (`ChecksumArgs`, `Access`) | parsing in `kona-interop` `access_list.rs`; checksum *computation* has no verified Rust counterpart — check at review time | n/a (validates via receipts) | consumes Go checksum path |
| Deposits-only trimming | `WithDepositsOnly` (`op-node/rollup/derive/attributes_queue.go`) | `as_deposits_only` (`kona-protocol` `attributes.rs`) | proof-interop consolidation applies the same trim | n/a |
| Replacement trigger / invalidation flow | op-node engine paths (`op-node/rollup/engine/payload_process.go`, `rollup/iface.go` SuperAuthority) driven by op-supernode | pending (lokahi invalidation protocol) | `proof-interop/src/consolidation.rs` | n/a |
| Superroot encoding | `op-service/eth/super_root.go` (encoder; `superroot_at_timestamp.go` is the RPC schema only), served by `op-supernode/.../superroot/` | `kona-interop` `root.rs` (`SuperRoot`) | same Rust type | n/a |
| Dependency set | `op-core/interop/depset` | `kona-genesis` `interop/depset.rs` | same | via config |

## Method

1. **Identify** which registry invariants the diff touches; fetch the cited spec files
   and re-read the relevant sections — do not review from memory of the spec. A
   protocol-relevant behavior with no registry entry is itself a finding: either this
   guide needs a row, or the change invents un-specced protocol behavior.
2. **Locate** every counterpart via the parity map. No counterpart → spec-only review,
   plus a note that parity review activates when the pending implementation lands.
3. **Compare accept-sets**: for each rule, trace both implementations on the boundary
   inputs — activation boundary (`activation`, `activation + block_time`), expiry
   boundary (`init + window == exec` vs `+1`), same-timestamp messages, empty/absent
   data. Strictness of each comparison operator is the usual divergence site.
   Implementations may be structured completely differently and still agree — structure
   is irrelevant, the accept-set is everything.
4. **Demand vectors** for any accept-set change: tests pinning both sides of each
   affected boundary, mirrored across implementations (the cross-implementation
   golden-vector convention from rust-dev.md, extended to interop rules). "The logic
   looks equivalent" is not evidence.
5. **Consult the intentional-divergence registry** before flagging; unregistered
   divergence is always a finding — fix it or register it, never ignore it.

## Intentional-divergence registry

Verified against the tree; re-verify entries when reviewing changes to these files.

- **Tx-pool filter rejects same-timestamp messages**: `validateMessageTiming` requires
  `initTimestamp < inclusionTimestamp` (strict), and the lockstep validator documents
  "no cycle detection: same-block executing messages are not supported". Consensus
  accepts equal timestamps; the filter is deliberately stricter (admission policy).
- **Tx-pool filter supports `ExecutingDescriptor.Timeout`** (forward-validity: the
  message must not expire before `exec + timeout`): admission-only concept, no verifier
  counterpart.
- **Tx-pool filter has no explicit activation invariant check**: it bounds validation by
  its ingestion window (`startTimestamp` / min-ingested timestamp) rather than the
  per-chain `activation + block_time` rule the verifiers enforce.
- **Existence-check mechanics differ** between the indexed-checksum verifiers and the
  receipt-equality proof side (see registry) — same intended accept-set, different data
  access.

## Traps

- **Overloaded names**: "cross-unsafe" chain-head vs message-safety-level (see registry);
  a grep hit on a shared name is a question, not a finding.
- **The `safe` wire alias**: `safety.CrossSafe` serialises as `"safe"` while local-safe
  is `"local-safe"` (`op-service/eth/safety/safety.go`; mirrored by the serde rename in
  `rust/op-alloy/crates/consensus/src/interop.rs`). A bare `"safe"` in a diff or wire
  schema means **cross**-safe and must never be read as local-safe.
- **Policy vs consensus**: stricter admission filtering is legitimate; a "parity fix"
  that loosens the filter to match consensus, or tightens a verifier to match the
  filter, changes the wrong side.
- **Spec drift is a finding class**: when an implementation and `specs/interop/`
  disagree, report it as such — do not adjust either to match the other without
  surfacing the disagreement.
- **Boundary asymmetry**: the activation invariant applies on *both* chains and the two
  checks can drift independently; always exercise both sides.
- **Constant duplication**: the expiry window exists in Go and Rust; a change to one is
  incomplete by definition.
