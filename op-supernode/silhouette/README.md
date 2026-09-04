# Silhouette — private interoperable chains

**A private L2 that participates in a public interop dependency set. Its private transactions are
not published; its public footprint is a signed proof envelope plus a transaction-stripped standard
derivation carrier per cadence.**

The public network verifies the private chain's *outline* — block numbers, timestamps, real block
hashes, real output roots, the messages it chooses to export and the messages it imports — and never
reads its interior. There is no DA layer to read it from.

## The mechanism, in one sentence

**A proven chain is a chain whose execution client is a verifier.**

A stock `op-node` derives the private chain through its completely standard pipeline and its normal
batch inbox. The ordinary `op-batcher` appends standard carrier frames, with user transactions
removed, after the proof envelope. Stock derivation ignores the proof blob's unknown format byte and
consumes the carrier. Independently, `op-silhouette-el` checks the envelope and stores its facts.
That execution client is a separate, persistent service speaking the Engine API and the `eth_`
query surface that never executes anything, and "builds" blocks by serving proof-committed facts.
The supernode consumes its message view over RPC; it does not run a second proof observer.

Because `op-node` treats the engine as the hash authority and re-hashes no payload on the
derivation path, the public network operates on the private chain's **real identity**: stock code
computes its real output roots from these headers — byte-identical to the values a settlement claim
would carry.

## What v1 rests on, stated up front

In v1 the proof of the chain is **the operator's signature** — specifically the L1 batch
transaction's own signature, since acceptance rule 1 already requires `tx.from == submitterEOA`.
There is no in-envelope signature and no new cryptography. The operator can attest to an invalid
state, and nothing here prevents that.

What is *genuinely verified*, and is not weakened by that: the wire's structure and chaining, the
config binding, the L1 view — and **the private chain's cross-chain imports**, which the stock
cross-safety judge validates against the public chain's own independently derived message database.
So: **an attester can invent what its own chain said; it cannot invent what someone else's chain
said.**

`docs/TRUST-MODEL.md` states this in full, without softening, and every claim in it is a named test.
The most important of those tests asserts a **weakness**: `TestAttestedFabricatedExportIsAccepted`
posts an export the private chain never emitted and asserts it is accepted and cross-safed. It
passes. A limitation documented only in prose is a limitation the next reader will not find.

One thing to be clear about early, because it shapes the whole diff: **v1 has no on-chain settlement
path.** No proposer, no dispute game, no output-root claim, nothing on L1 adjudicating whether the
private chain told the truth. The portal is the stock OptimismPortal; deposits are read by stock
derivation. The L1 contracts here bound ETH flow but do not settle. Cross-chain consistency is enforced **live, in the
node**, by the stock cross-safety judge, on every block. The proof-shaped upgrade path — `attested`
→ `proven`, same wire, same slot, same rules — is real and the seam is built for it; it lives on the
shelf branches, and §"What is deliberately not here" says where.

## The implementation shape

There are **no changes under `op-node/`**. The verifier is a stock op-node assembled by
`op-supernode`; its configured L2 endpoint is `op-silhouette-el`, which serves proof-committed facts
and delegates stock replacement-block construction to the private chain's authenticated Engine API.
Its `--data-dir` retains the public chain view and L1 cursor across restarts.

There is no separate batch submitter or custom inbox. The normal `op-batcher` loads real blocks,
output roots and receipts, follows its ordinary reorg/channel/txmgr lifecycle, and drops private
transaction data only at its terminal encoding seam. The three runtime images are `op-batcher`, `op-supernode`, and
`op-silhouette-el`. There is no ZK toolchain in the v1 build or run.

## How to read it

1. `docs/TRUST-MODEL.md` — on what authority the public network believes any of this. Read first.
2. `source.go` — the EL-owned proof observer: acceptance rules, chaining and fact ingestion.
3. `facts.go` + `forced.go` — the fact store and the forced-extension convention (a dead prover must
   never stall the dependency set's frontier, so stock derivation force-generates empty blocks).
4. `shim_engine.go` + `shim_shim.go` — the execution client that verifies instead of executing.
5. `op-batcher/batcher/proof_batch.go` — the normal batcher's terminal proof-batch encoder.
6. `docs/RUNNING-V1.md` — build, configure, run, and the operational differences.
7. `docs/SPEC-WIRE-V4.md` — the current cross-language wire contract (`SPEC-WIRE-V3.md` is its
   import-list predecessor).

To see all of it run, in one command, with no cluster:

```
go test ./op-acceptance-tests/tests/interop/silhouette/ -v -timeout 45m
```

Two chains, one verifier supernode, real execution clients as subprocesses, a real L1, real blob
transactions, including invalid-dependency replacement through the stock interop path.

## What is deliberately not here

This branch is a **minimal system**, not an archive. Four bodies of real, working, gated code were
left off it on purpose, because a diff that carried them would argue for something this system does
not do:

- **The inner ZK proving stack** — the guest that proves the private chain's own state transition,
  its witness codec and host, and the pure-Go SP1 Groth16 verifier that has verified a real proof
  from the prover network. v1 verifies no such proof, so it ships none of the machinery. What it
  keeps is the **shape of the seam**: the proof slot is on the wire, acceptance rule 5 is the rung
  that reads it, and `proofType: groth16` is recognised by name and refused with an error saying the
  config is ahead of the binary rather than malformed. Branches `karl/silhouette` (Go verifier) and
  `karl/silhouette-guest` (the guest).
- **The settlement guest** — one circuit, two chains: it derives the public chain for real,
  authenticates the private chain from its blob, and consolidates the cross-chain edges between them
  in-circuit. It is off this branch for a sharper reason than size. **v1 has no on-chain settlement
  path**, so there is nothing for a settlement proof to settle *to*; and the consistency the guest
  checks is already checked — live, node-side, by the cross-safety judge, on every block, against the
  public chain's own independently derived message database. Proving a check that already runs is not
  a smaller system, it is a second answer to a question this one already answers. The one piece of it
  that *is* in-tree is the hook: kona-derive's blob payload decoder, exported so that a kona-based
  reader takes these blobs off L1 through the same decoder. Branch `karl/silhouette-guest`.
- **The game-10 settlement deployment tooling** — deploy/register/propose/prove scripts for the
  on-chain dispute game, which is the piece that would give v2 a settlement path at all. Branch
  `karl/silhouette-guest`.
- **The decommissioned demo cluster's run record** — the executed rotation, its genesis and rollup
  artifacts, and its evidence files. This system *has* run for real on Sepolia; that record is
  operational history rather than code, and its frozen configs use a config spelling this binary
  now refuses. Branch `karl/silhouette`.

Also absent by design is any publication of the private transaction list. The portal and deposit
derivation are stock, and the producer is the ordinary op-batcher with a proof-batch terminal
encoder; neither requires a custom contract.

The project's design record — the plan, the decision log, and the proving-key lineage — lives with
the Silhouette project outside this repository, alongside the shelf branches above. None of it is
needed to read, build, or run what is here: nothing on this branch has a proving key, so this branch
records none.
