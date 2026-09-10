# Running Silhouette v1

**A private, interoperable L2 whose public footprint is signed proof-batch blobs, with no ZK
toolchain in the v1 build or run.**

This is the whole v1 system: what to build, what to configure, what to start, and the four traps that
have actually bitten. It assumes you have read `TRUST-MODEL.md` — v1 rests P's state on its
operator's signature, deliberately, and there is no point standing this up without knowing that.

This system has been run for real, on Sepolia, as a two-chain cluster. That cluster has since been
decommissioned and its run record is not on this branch (see `README.md`); everything below is
checkable locally, which is how §5 asks you to check it.

---

## 0. What the system is, in one picture

```
  P (private)                                          A (public)
  ┌──────────────────────────┐                         ┌──────────────────┐
  │ LightCL op-node + EL      │                        │ op-node + EL     │
  │ real blocks, real txs     │                        │ + op-batcher     │
  │ + ordinary op-batcher    ├──────────────┐         └────────┬─────────┘
  └───────────────────────────┘ proof + stock carrier blobs    │ batches
                                            │                  │
                                     ┌─────▼────────────────────▼──────┐
                                     │             L1                  │
                                     └─────┬───────────────────────────┘
                                           │
                  ┌──────────────────▼────────────────────┐
                  │ op-silhouette-el                      │
                  │ persistent proof-derived public view │
                  └──────────────────┬────────────────────┘
                                     │ Engine + interop-facts RPC
                            ┌────────▼───────────────────────────────┐
                            │ VERIFIER SUPERNODE                    │
                            │ A: ordinary EL                        │
                            │ P: op-silhouette-el in the EL slot    │
                            │ stock derivation + interop judge      │
                            └───────────────────────────────────────┘
```

Three things to notice, because they are the deliverable:

1. **P uses the ordinary op-batcher and ordinary batch inbox.** It reads real private blocks and
   their receipts. Its terminal encoder emits the proof envelope followed by a standard carrier
   channel for the same blocks with user transactions removed. Deposits remain in the carrier.
2. **The verifier swaps only P's EL component.** It points at `op-silhouette-el`, a separate,
   persistent Engine API that learns P only from signed blobs and delegates replacement construction
   to P's private EL. The supernode owns no proof walker or duplicate fact table.
3. **A is unmodified.** No fork, no plugin, no special casing. It is a stock chain that happens to
   have a silhouette chain in its dependency set.

---

## 1. Build (three runtime images)

```
docker buildx bake op-batcher op-supernode op-silhouette-el
```

The images are named `op-batcher`, `op-supernode`, and `op-silhouette-el`. `op-node` and the L1/L2
execution images are the same images used by a normal interop network. There is no separate
silhouette submitter image or binary.

---

## 2. Configure

Three artefacts. A validated, loadable example of the last two lives at
`op-supernode/silhouette/example/v1/` (a test loads it through the real manifest path, so it cannot
silently rot).

### 2a. P's rollup config — generated, not hand-written

```
silhouette-config --l2-chain-id … --l1-chain-id … --deposit-contract 0x… --seq-window … …
```

Generate it. A silhouette chain's rollup config differs from a stock chain's in three ways that a
hand-edited file loses silently: a **finite** sequencing window (the forced-extension convention
depends on it) and every fork active at genesis. `deposit_contract_address` is the ordinary deployed
OptimismPortal, so deposits are derived by the stock path. The generator runs the invariant checks
over its own output.

Its `rollupConfigHash` is what the wire binds and what the verifier requires. Compute it the way the
runbook describes — from the *parsed* config — never with a second implementation of the hash.

### 2b. The verifier config — one file per silhouette chain

```json
{
  "l1ChainID": 11155111,
  "submitter": "0x…",     ← THE ATTESTER. This key's signature is the proof.
  "inbox": "0x…",
  "rollupConfigHash": "0x…",
  "depSetHash": "0x…",
  "proofType": "attested",
  "wireVersion": 4,
  "anchor": { "outputRoot": "0x…", "blockNumber": 0, "blockHash": "0x…",
              "timestamp": …, "l1Origin": { "hash": "0x…", "number": … } }
}
```

Every field is either a binding the wire is checked against or an L1 coordinate. Notably absent:
anything about how P executes. A verifier of a silhouette chain holds no state, no genesis allocation
and no EVM.

Two fields carry the trust posture and both are logged at startup, once, so an operator can read them
without reasoning about behaviour:

- **`proofType: attested`** — required, no default. The trust model is never assumed, and this
  build implements exactly one. `proofType: groth16` is recognised and refused with an error saying
  so; when it is implemented it is this same file plus the proving program's key, and nothing else
  changes.
- **`wireVersion: 4`** — the version carrying both the import list and the exact L1-origin/sequence
  metadata followed by LightCL. Exactly one version is accepted, refused at load if unreadable.

`anchor` is where this verifier's proven history begins. On a fresh chain that is P's genesis:
`blockNumber: 0`, P's real genesis hash, and the output root read off P at block 0.

### 2c. The manifest — which chains are silhouette chains

```json
{ "chains": [ { "chainID": 424247, "verifierConfig": "verifier-p.json" } ] }
```

Chain A is **absent**, and its absence is the two-chain story: a chain not in the manifest is an
ordinary driven chain and nothing about its construction changes. That is what makes one binary safe
to run over a mixed cluster.

There is one role: verifier. A `labels: "proven-head"` entry is rejected. Private sequencing belongs
to a LightCL-based sequencing node, not to `op-supernode`.

---

## 3. Run

### The Silhouette EL

```
op-silhouette-el --l1 $L1_RPC --l1.beacon $L1_BEACON --supernode $SUPERNODE_RPC \
  --rollup-config rollup-p.json --verifier-config verifier-p.json \
  --data-dir /var/lib/op-silhouette-el \
  --replacement-engine $PRIVATE_ENGINE_AUTH_RPC \
  --replacement-engine.jwt-secret jwt.txt \
  --rpc.addr 0.0.0.0 --rpc.port 9545
```

This is the proof-backed, persistent EL. It independently watches the authenticated proof stream and
serves Engine/eth RPC. It executes no EVM code. Its data directory atomically stores accepted facts,
batch/L1 provenance, public renderings, forkchoice labels, denial/replacement state, and the L1 walker
checkpoint, so a restart resumes without replaying the chain. The replacement engine is the private
chain's normal EL; it is used only when stock interop invalidation asks op-node to build a Holocene
deposits-only replacement. On accelerated devnets, pass the same
`--l1.beacon.slot-duration-override` to this process and `op-supernode`.

### The verifier supernode

```
op-supernode --chains … --dependency-set depset.json --silhouette manifest.json \
             --vn.<P>.l2 http://op-silhouette-el:9545 \
             --vn.<P>.interop.dependency-set depset.json …
```

Two invocation details that are not obvious and cost time if guessed:

- **`--vn.<P>.l2` is the real `op-silhouette-el` URL.** The supernode does not replace it and does
  not embed a second EL.
- **`--vn.<P>.interop.dependency-set` must be passed in addition to the supernode-level
  `--dependency-set`**, because op-node's own config check runs before the supernode applies the
  shared set. The "virtual node flag is ignored" warning is expected; the supernode-level value wins.

Read the EL and supernode startup lines and check the declared client and chain:

```
connected standalone silhouette EL chain=424247 client="op-silhouette-el/v1 …"
```

The full trust posture and wire version remain explicit in `verifier-p.json`; the EL refuses an
incompatible config before opening its RPC listener.

### The producer batcher

```
op-batcher <normal batcher flags> --data-availability-type blobs --silhouette \
  --max-pending-tx 1 --target-num-frames 6 \
  --silhouette.inbox 0x… --silhouette.wire-version 4 \
  --silhouette.rollup-config-hash 0x… --silhouette.dependency-set-hash 0x… \
  --silhouette.max-blocks 300
```

All ordinary flags retain their meanings: L1 RPC, rollup RPC, L2 RPC, batcher key, polling, channel
retry, tx manager, and blob submission. The silhouette flags augment the final channel encoding.
`--silhouette.inbox` must equal P's normal `batch_inbox_address`; there is no second inbox or custom
portal. One pending transaction is recommended so reorg replacement ranges are not obscured
by speculative overlapping channels; all frames for one envelope must fit in one blob transaction.
The send is the attestation: acceptance rule 1 binds the L1 transaction sender to the verifier
config's `submitter`, and attested mode requires the proof slot to be empty.

Each submission contains the following, in this order:

- envelope: `KCPB` magic, wire version, public-values length and bytes, then proof length and an
  empty proof in attested mode;
- batch: previous output root, new output root, L1 head hash, rollup-config hash, dependency-set
  hash, export-policy hash, and the block list;
- each block: number, timestamp, real block hash, state root, L2-to-L1-message-passer storage root,
  exported logs, imported messages, L1-origin hash and number, and L1 sequence number;
- each exported log: log index, log hash, and optional preimage. The active `all-hashes` policy sends
  the preimage as empty, so the address/topics/data are committed by the hash but not disclosed;
- each imported message: origin address, source block number, source log index, source timestamp,
  source chain ID, and message payload hash.
- standard derivation carrier blobs: ordinary channel/frame/compression encoding containing a
  singular batch per proof-committed block, with the real parent hash, epoch and timestamp, an empty
  user-transaction list, and stock-derived deposits (including the L1-info transaction).

Both parts are sent by the same normal batcher transaction to the normal batch inbox. A stock
`op-node` rejects the `KCPB` blob as an unknown derivation-format byte, then consumes the following
standard carrier frames. `op-silhouette-el` independently reads the proof envelope and maps those
otherwise ordinary payload attributes to the committed block hashes and roots.

No transaction bytes, transaction hashes, transaction senders, calldata, receipts, contract state,
or full log contents are included. As with any L1 blob transaction, the transaction envelope itself
also exposes its sender, destination inbox, nonce, fees, blob commitments, and L1 inclusion block.

---

## 4. Operational differences from a normal interop network

Each of these has actually happened. They are here rather than in a postmortem because each one
presents as a healthy system.

1. The private chain must be sequenced by a LightCL op-node. The supernode is verifier-only
   and rejects attempts to sequence the silhouette chain.
2. The LightCL follows the verifier supernode's chain-scoped rollup route. Its own EL remains the
   private, full execution engine; this is also where stock Holocene replacement blocks are built.
3. The normal op-batcher points at that LightCL and private EL, receives the silhouette flags above,
   and still submits to the rollup's normal batch inbox. It starts at the verifier's safe head and
   follows ordinary batcher reorg handling.
4. A verifier accepts exactly one wire version, and a rotation is two verifiers, not one lenient
   one. A node that accepted both would silently apply the weaker posture—dependencies not
   checked—to a chain whose operator believes it runs the stronger one.
5. `op-silhouette-el` owns its data directory exclusively. Give it persistent storage and restart
   that component in place; do not share the directory with `op-supernode` or another EL replica.

---

## 5. Checking it works

**Locally, with no cluster at all.** The multi-process two-chain harness is the real system: two
chains, two supernodes, real ELs as subprocesses, a real L1, real blob transactions.

```
go test ./op-acceptance-tests/tests/interop/silhouette/ -v -timeout 45m
```

It covers both cross-chain directions, proof-slot refusal, the fabricated-export honesty test, and
invalid-import replacement. If you are evaluating this system, this command is the shortest path to
seeing all of it run.

```
go test ./op-supernode/...          # the unit suites, including the wire's cross-language fixtures
```

For an interactive two-chain network with funded accounts and RPCs on `8545`/`8546:

```
go run ./op-up --silhouette
go run ./op-up smoke-interop all
go run ./op-up smoke-interop chained-invalid-message --require-cascade --reorg-timeout 5m
```

Run `go run ./op-up --interop` instead to compare the same smoke tools against the normal sysgo
interop topology.

**Against a live stream.** `proofbatch-inspect` reads envelope FILES (not L1) and reports the wire
object in words, including:

```
proof   0 bytes — ATTESTED: no proof was posted. A verifier configured with proofType: attested
        requires exactly this, and rests the batch on the submitter's L1 signature; one configured
        for a proving system will reject it
```

**Health, per chain.** The verifier's `proofType` and `dependenciesVerified` at startup; the proven
head advancing on the silhouette route; `supernode_interop_invalidations_total` staying at zero while
`timestamps verified` is positive (a zero invalidation count from an idle verifier says nothing).

---

## 6. Upgrading to proven state

Flip `proofType` to `groth16`, name the proving program's key, and post batches whose proof slot is
filled. Same wire, same envelope version, same acceptance rules 1–4, same judge, same import list.

Today this build refuses that value, and the error is the useful part: it says the proving system is
one this binary does not have rather than one you misspelled, and it points here. The machinery that
produces the proofs is built and working but deliberately absent from this branch — `README.md`
§"What is deliberately not here" says where it is and why.

Note what this upgrade does *not* add: an on-chain settlement path. v1 has none, and turning proofs
on does not by itself create one. Cross-chain consistency is checked live by the judge either way;
the dispute game is a separate piece of the shelf, with its own deployment story.

There is no migration and no re-anchoring: the two modes are two answers to one acceptance rule.
