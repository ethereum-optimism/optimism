# Silhouette — private interoperable chains

**A private L2 that participates in a public interop dependency set. No data availability layer. No
deposits. Its entire public footprint is one signed blob per cadence.**

The public network verifies the private chain's *outline* — block numbers, timestamps, real block
hashes, real output roots, the messages it chooses to export and the messages it imports — and never
reads its interior. There is no DA layer to read it from.

## The mechanism, in one sentence

**A proven chain is a chain whose execution client is a verifier.**

A stock `op-node` derives the private chain through its completely standard pipeline. One injected
data source sits where DA plugins sit: it filters the L1 inbox transactions, checks the acceptance
rules, and transcodes the proven blocks into stock channel frames. Everything from `FrameQueue`
onward is unmodified derivation running for real. The execution client is a **shim** — a small
service speaking the Engine API and the `eth_` query surface that never executes anything, and
"builds" blocks by serving the proof-committed facts from the wire.

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
private chain told the truth. The L1 contracts here gate the portal shut and bound ETH flow; they do
not settle. The cross-chain consistency that would otherwise need settling is enforced **live, in the
node**, by the stock cross-safety judge, on every block. The proof-shaped upgrade path — `attested`
→ `proven`, same wire, same slot, same rules — is real and the seam is built for it; it lives on the
shelf branches, and §"What is deliberately not here" says where.

## The diff

Measured against the upstream base this branch was cut from, `9104d734a9`:

| Category | Files | + | − | What |
|---|---:|---:|---:|---|
| **The op-node seam** | **3** | **71** | **3** | `DataSourceOverride` + `ExtraAPIs`. **The entire consensus-layer change.** |
| The silhouette Go stack | 41 | 10,426 | 0 | Decode source, transcoder, fact store, shim EL, log sink, assembly, postures — and its tests |
| Wire codec (Go) | 4 | 2,641 | 0 | The v3 envelope, and the acceptance structure checked before anything else looks at it |
| Fixture corpus | 131 | 3,406 | 0 | 35 v3 fixtures + 25 frozen v2 envelopes — the bytes that pin the wire, in either language |
| Producer + ops tools | 6 | 1,816 | 0 | The submitter (the send is the attestation), `silhouette-config`, `proofbatch-inspect` |
| Interop plumbing | 16 | 1,902 | 22 | The judge flip, the third verdict, the capability seams — none behind a chain-kind branch |
| Devstack + acceptance | 14 | 2,714 | 29 | The multi-process two-chain system and the 11-test v1 gate |
| Contracts | 19 | 2,404 | 10 | The gated portal; the ETH net-flow solvency cap; the home-pinned bridge |
| kona-derive | 3 | 36 | 2 | The op-stack blob payload decoder, exported for any kona-based reader |
| Docs | 5 | 1,441 | 0 | This file, the trust model, the runbook, two specs |
| Images | 3 | 50 | 1 | A bake target for the submitter, and the two build recipes behind it. The supernode's image already existed upstream |
| Misc | 1 | 5 | 0 | `.gitignore` for locally-built binaries |
| **Total** | **246** | **26,912** | **67** | |

**The number to look at first is 71.** A private chain in a public dependency set costs the
consensus layer three files and two seams, both in the pattern of seams already there. Everything
else is a service beside `op-node`, not a change to it.

Sizes, for the record: the verifier core (`silhouette` + `proofbatch`, non-test) is **5,858 lines**
of Go; **7,045** including the submitter and the two tools; **8,020** lines of Go tests. There is no
ZK toolchain in the build or in the run.

## How to read it

1. `docs/TRUST-MODEL.md` — on what authority the public network believes any of this. Read first.
2. `op-node/` (71 lines) — the whole CL touch.
3. `source.go` — the decode source: acceptance rules, chaining, transcode to stock frames.
4. `facts.go` + `forced.go` — the fact store and the forced-extension convention (a dead prover must
   never stall the dependency set's frontier, so stock derivation force-generates empty blocks).
5. `shim_engine.go` + `shim_shim.go` — the execution client that verifies instead of executing.
6. `docs/RUNNING-V1.md` — build, configure, run, and the four traps that have actually bitten.
7. `docs/SPEC-WIRE-V3.md` — the cross-language wire contract.

To see all of it run, in one command, with no cluster:

```
go test ./op-acceptance-tests/tests/interop/silhouette/ -v -timeout 45m
```

Two chains, two supernodes, real execution clients as subprocesses, a real L1, real blob
transactions. 11 tests, ~90s.

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

Also absent by design, from the architecture rather than the branch: any DA layer for the private
chain, any deposit path into it (the portal is deployed and gated — an open portal would let a
stranger force a transaction into a private chain), and any batcher on it.

The project's design record — the plan, the decision log, and the proving-key lineage — lives with
the Silhouette project outside this repository, alongside the shelf branches above. None of it is
needed to read, build, or run what is here: nothing on this branch has a proving key, so this branch
records none.
