# Running Silhouette v1

**A private, interoperable L2 whose public footprint is one signed blob per cadence — in ~6k lines of
Go, with no ZK toolchain in the build or in the run.**

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
  │ op-node + real EL        │                         │ op-node + EL     │
  │ real blocks, real txs    │                         │ + batcher        │
  │ NO batcher               │                         └────────┬─────────┘
  └───────────┬──────────────┘                                  │ batches
              │ safe/unsafe head, output roots                  │
     ┌────────▼─────────┐                                       │
     │ proofbatch-      │  blob tx, SIGNED BY THE OPERATOR      │
     │ submitter        ├──────────────────┐                    │
     └──────────────────┘                  │                    │
                                     ┌─────▼────────────────────▼──────┐
                                     │             L1                  │
                                     └─────┬───────────────────────────┘
                                           │
                            ┌──────────────▼───────────────────────────┐
                            │ VERIFIER SUPERNODE                       │
                            │  A: ordinary chain, derived from L1      │
                            │  P: silhouette chain — NO execution      │
                            │     client at all. Proof-batch source    │
                            │     + shim EL serving proven facts.      │
                            │  stock interop judge over both.          │
                            └──────────────────────────────────────────┘
```

Three things to notice, because they are the deliverable:

1. **P has no batcher.** Nothing puts P's blocks on L1. The only public record of P is the batch
   stream.
2. **The verifier has no execution client for P.** Its entire knowledge of P is signed blobs. That
   absence is the product, not a limitation of the harness.
3. **A is unmodified.** No fork, no plugin, no special casing. It is a stock chain that happens to
   have a silhouette chain in its dependency set.

---

## 1. Build (four Go binaries, no toolchain beyond Go)

```
go build -o op-supernode           ./op-supernode/cmd
go build -o proofbatch-submitter   ./op-supernode/cmd/proofbatch-submitter
go build -o silhouette-config      ./op-supernode/cmd/silhouette-config
go build -o proofbatch-inspect     ./op-supernode/cmd/proofbatch-inspect
```

There is no `cargo`, no prover, and no proving artefact of any kind anywhere in this list — which is
the v1 pitch stated as a build. `TestAttestedChainIsRenderedWithoutAProvingToolchain` asserts the
configuration half of the same claim.

Size, for the record: the verifier core (`op-supernode/silhouette` + `op-supernode/proofbatch`,
non-test) is **5,858 lines** of Go; **7,045** including the submitter, the config generator and the
inspect tool; **8,020** lines of tests. Nothing in that build verifies a proof, so nothing in it
needs a toolchain that could.

---

## 2. Configure

Three artefacts. A validated, loadable example of the last two lives at
`op-supernode/silhouette/example/v1/` (a test loads it through the real manifest path, so it cannot
silently rot).

### 2a. P's rollup config — generated, not hand-written

```
silhouette-config --l2-chain-id … --l1-chain-id … --gated-portal 0x… --seq-window … …
```

Generate it. A silhouette chain's rollup config differs from a stock chain's in three ways that a
hand-edited file loses silently: a **finite** sequencing window (the forced-extension convention
depends on it), a `deposit_contract_address` pointing at the **gated** portal, and every fork active
at genesis. The generator runs the invariant checks over its own output.

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
  "wireVersion": 3,
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
- **`wireVersion: 3`** — the version carrying the import list, which is what puts P's dependencies
  under the stock cross-safety judge. Exactly one version is accepted, refused at load if unreadable.

`anchor` is where this verifier's proven history begins. On a fresh chain that is P's genesis:
`blockNumber: 0`, P's real genesis hash, and the output root read off P at block 0.

### 2c. The manifest — which chains are silhouette chains

```json
{ "chains": [ { "chainID": 424247, "verifierConfig": "verifier-p.json", "labels": "derivation" } ] }
```

Chain A is **absent**, and its absence is the two-chain story: a chain not in the manifest is an
ordinary driven chain and nothing about its construction changes. That is what makes one binary safe
to run over a mixed cluster.

`labels` selects the posture: `derivation` for a public verifier (it derives P itself), `proven-head`
for the sequencer-side node (it fronts P's real EL and must take its **public** labels from the
proven head — mandatory there, or a frozen P local-safe freezes A cluster-wide).

---

## 3. Run

### The verifier supernode

```
op-supernode --chains … --dependency-set depset.json --silhouette manifest.json \
             --vn.<P>.l2 http://127.0.0.1:1 --vn.<P>.interop.dependency-set depset.json …
```

Two invocation details that are not obvious and cost time if guessed:

- **`--vn.<P>.l2` must be present and is never dialled.** The assembly replaces the L2 endpoint with
  an in-process client to the shim before the container is built; op-node's config builder only
  requires the flag to exist. A placeholder is correct and is not a lie waiting to be discovered —
  nothing connects to it.
- **`--vn.<P>.interop.dependency-set` must be passed in addition to the supernode-level
  `--dependency-set`**, because op-node's own config check runs before the supernode applies the
  shared set. The "virtual node flag is ignored" warning is expected; the supernode-level value wins.

Read the startup line and check two words:

```
assembled silhouette chain  chain=424247 wireVersion=3 dependenciesVerified=true proofType=attested …
```

These are the only two properties of a silhouette verifier invisible from the outside. Every proving
system and both wire versions derive the chain, serve the same roots and report the same heads. Only
these say what an accepted batch **meant**.

### The submitter — v1's whole producer side

```
proofbatch-submitter --l1-eth-rpc … --private-key … --inbox 0x… --wire-version 3 \
  --attested.rollup-rpc http://p-node:9545 --attested.l2-rpc http://p-el:8545 \
  --attested.rollup-config-hash 0x… --attested.dep-set-hash 0x… \
  --attested.interval 10m --attested.max-blocks 300 \
  --attested.cursor /var/lib/silhouette/cursor.json
```

**The send IS the attestation.** The transaction is signed by `--private-key`, acceptance rule 1 binds
every accepted batch to that key, and the proof slot goes out empty. `--wire-version` is required and
has no default on purpose: the version a submitter posts decides whether every verifier in the
dependency set checks P's declared imports or trusts them, and inheriting it from whichever codec the
binary was built against is how that changes without anyone deciding it.

`--envelope <file>` is the other mode: post one pre-built envelope and exit.

---

## 4. The four traps

Each of these has actually happened. They are here rather than in a postmortem because each one
presents as a healthy system.

**1. SEED THE CURSOR BEFORE THE SUBMITTER STARTS.** Write it yourself from the verifier config's
anchor:

```json
{ "lastBlock": 0, "outputRoot": "<the verifier config's anchor.outputRoot>" }
```

Left to itself the submitter anchors on P's **current head**, and P's sequencer starts sealing
2-second blocks the moment its EL comes up. Its first batch would then start *above* the verifier's
anchor, acceptance rule 3 requires a batch to extend the anchor, and **every batch would be rejected
forever** — from two artefacts that were each individually correct.

**2. `--attested.cursor` MUST BE AN ABSOLUTE PATH.** It defaults to CWD-relative, so a unit with a
different `WorkingDirectory` silently ignores the seeded file and re-anchors on every restart. Never
delete this file; restarting the submitter without it re-anchors on the current head, which is
trap 1.

**3. `--attested.head` defaults to `unsafe`, and must.** On a chain with no batcher the safe head is
only ever what this tool last proved, so batching on the safe head deadlocks.

**4. A verifier accepts exactly ONE wire version, and a rotation is two verifiers, not one lenient
one.** A node that accepted both would silently apply the weaker posture — dependencies not checked
— to a chain whose operator believes it runs the stronger one. The failure mode is not "a batch was
rejected", it is "nothing was checked and everything looked fine".

---

## 5. Checking it works

**Locally, with no cluster at all.** The multi-process two-chain harness is the real system: two
chains, two supernodes, real ELs as subprocesses, a real L1, real blob transactions.

```
go test ./op-acceptance-tests/tests/interop/silhouette/ -v -timeout 45m
```

11 tests, ~90s. It covers both cross-chain directions, both postures, the proof-slot refusal, and the
fabricated-export honesty test. If you are evaluating this system, this command is the shortest path
to seeing all of it run.

```
go test ./op-supernode/...          # the unit suites, including the wire's cross-language fixtures
```

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
