# Silhouette v1 — the trust model

**Status: RATIFIED (Karl, 2026-08-26). Implemented as `proofType: attested`, the only proving system
this build has.**

A silhouette chain is a private L2 whose only public footprint is a batch posted to an L1 inbox. The
public network verifies its *outline* — block numbers, timestamps, real block hashes, real output
roots, the messages it exports and the messages it imports — and never reads its interior.

The question this document answers is the only one that matters about such a system: **on what
authority does the public network believe the outline?**

In v1 the answer is: **the operator's signature.** Not a proof. This document says exactly what that
buys, exactly what it does not, and what changes when it is replaced.

---

## The mechanism, in one paragraph

The proof of the chain is the operator's **attestation**, and the attestation is the L1 batch
transaction's own signature. There is no signature inside the envelope and no new cryptography,
because none is needed: acceptance rule 1 already requires that a batch arrive in a blob transaction
sent **to the configured inbox** and **from the configured submitter EOA**. A batch that reaches a
verifier at all was therefore signed by the operator's key over the blob hashes the envelope was read
from. Attester and submitter are the same key by construction, not by convention.

The proof slot on the wire stays, and stays **empty**. A verifier in attested mode requires
`proof_len == 0` and refuses a batch that carries proof bytes — even genuine ones. That refusal is a
rule, not an oversight: a node that accepts bytes it cannot check, while its operator believes those
bytes were checked, is in a worse state than a node that rejected the batch.

---

## What v1 GIVES

These are real properties, enforced in code, and none of them depends on trusting the operator about
anything beyond their own chain's interior.

**1. Authenticity.** Every accepted batch is bound to the operator's L1 key. Nobody else can advance
P's proven history, and nothing about P's public history is deniable by its operator: the claim is
signed, on L1, with the operator's own funds paying for it. The trust shape is exactly a batcher
inbox's — and every op-stack chain's public history already rests on it.

**2. Structure.** The wire object is checked, hard, against things the operator does not control:

- **chaining** — a batch must extend the verifier's current proven head's output root, so history
  cannot be forked, replayed, reordered or gapped;
- **internal consistency** — the last block's three committed roots must derive the claimed
  `newOutputRoot`, block numbers must be contiguous, timestamps must be spaced by exactly the chain's
  block time, log indices must ascend;
- **config binding** — `rollupConfigHash`, `depSetHash` and `exportPolicyHash` must be the values
  *this* verifier holds, so a valid batch about a *different* chain is not accepted as this one's;
- **L1 view** — the claimed `l1Head` must be canonical on the verifier's own L1 and within a bounded
  depth of the block that carried the batch, so a batch cannot be derived against an L1 history the
  verifier has passed or never saw;
- **wire version** — exactly one accepted version, refused at config load rather than at the first
  blob, because the version decides whether imports are checked at all.

**3. Cross-chain import consistency — genuinely verified.** This is the part that surprises people,
and it is the reason wire v3 and the G7 judge flip stay switched ON in v1.

P's batches declare, per block, the executing messages that block consumed (the *import list*). The
**stock** cross-safety judge validates that list against chain A's real, independently derived
message database: the checksum, the hazard set, expiry, the fixpoint. A lying attester's claimed
imports are still checked.

So: **an attester can invent what its own chain said. It cannot invent what someone else's chain
said.** A batch declaring an import that chain A never emitted does not become cross-safe — the
frontier pins below it, and the block is never replaced (`TestSilhouetteImportThatIsFalseIsRefused`).

The reason this survives the loss of the proof is structural, not lucky: the import list is **wire
data**. It is decoded from the envelope, recorded against the wire version, and handed to the stock
judge. The verifier's answer about the proof is one acceptance rule beside four others and
contributes no input to any of it. There is a test that asserts exactly this — every recorded fact is
byte-identical across an attested batch and a proof-verified one
(`TestTheJudgeReadsTheWireNotTheProof`).

**4. Liveness independent of the operator.** If the operator stops posting, P's L1 origin eventually
advances a full sequencing window past the last proven block and stock derivation force-generates
empty blocks. The dependency set's cross-safe frontier never stalls on a silent operator. A dead
attester costs P its own progress and nothing else.

---

## What v1 does NOT give

Stated without hedging, because the whole value of writing it down is that it is not softened.

**1. State validity.** The operator can attest to an invalid state transition. Nobody checks P's
execution. A verifier of a silhouette chain has no execution client for it, no state, no EVM and no
way to know what P's blocks really did — its entire knowledge of P is a signed blob. Concretely, an
operator who chooses to can:

- **fabricate exports** — declare an exported log that P never emitted, at a real block and a real
  index, with an invented hash. It is accepted, sealed into the verifier's interop log database, and
  cross-safed; from that moment it is referenceable by peers exactly like a genuine message. There is
  a passing end-to-end test that does this, deliberately
  (`TestAttestedFabricatedExportIsAccepted`).
- **attest to an invalid STF** — post roots that no correct execution of P's transactions produces.
- **inflate** — nothing prevents P's interior from minting value it did not receive. The
  no-deposit soundness rule that closes this belongs to a *proving* implementation, and v1 proves
  nothing. What v1 does close is the L1 half of it: the portal is deployed and gated, so no stranger
  can force a deposit in, and the ETH net-flow cap bounds what a lie can drain (see
  `SPEC-ETH-NETFLOW-CAP.md`).

**2. Anything about P's interior at all.** Not that P's blocks are well-formed, not that its
transactions are valid, not that its state root follows from its parent. The outline is checked for
*consistency*; the interior is not checked for *anything*.

**3. Accountability beyond attribution.** The signature makes a lie **attributable**, not
**preventable** and not **automatically punishable**. There is no bond, no challenge game and no
slashing in v1. What exists is a signed, timestamped, on-L1 record of exactly what the operator
claimed — which is what makes the lie provable after the fact, to a human.

### Who is exposed, and to what

The exposure is not symmetric, and the asymmetry is the reason v1 is shippable:

- **P's own users** trust P's operator with P's state. They already did — it is a private chain run by
  that operator.
- **Peer chains in the dependency set** are exposed through **messages they import from P**. A
  fabricated export can be executed on chain A as though it were real. This is the real
  cross-chain risk of v1 and the reason the dependency set is a deliberate, curated object.
- **Peer chains are NOT exposed through P's imports.** Those are checked against their own chains.

---

## The upgrade path

**v1 has no on-chain settlement path.** There is no dispute game for P, no proposer, no output-root
claim and no verifier contract that adjudicates anything about it. The L1 contracts on this branch
exist for other reasons — to *gate* the portal shut and to *bound* the blast radius of ETH flow —
not to settle P's state. Nothing on L1 is asked to decide whether P told the truth.

The consistency that would otherwise need settling is enforced **live, in the node**: the stock
cross-safety judge validates every message P imports against the public chain's own independently
derived message database, and pins the frontier when one does not check out. That is not a degraded
stand-in for an on-chain check — it is where the check belongs, it runs on every block, and it is
what actually protects peer chains in v1.

What v1 keeps of the proven future is **the shape of the seam.** The proof slot is on the wire,
acceptance rule 5 is the rung that reads it, `proofType` is the field that selects it, and
`proofType: groth16` is recognised by name and refused with an error saying the config is ahead of
the binary rather than malformed. One hook goes further and is in-tree: kona-derive's blob payload
decoder is exported (`decode_blob`), so a future kona-based reader takes the proof-batch blobs off L1
through the *same* decoder rather than a second copy of a consensus-critical decoding.

The proving machinery itself — the settlement circuit that consolidates cross-chain edges in-circuit,
the inner state-transition proof, and a pure-Go Groth16 verifier that has verified a real proof from
the prover network — is **built, working, and deliberately off this branch.** Shipping the machinery
of a proving system nobody runs would make the diff argue for something this system does not do. See
`README.md` §"What is deliberately not here" for where each piece lives.

What changes when it comes off the shelf:

| | v1 | v2+ |
|---|---|---|
| config | `proofType: attested` | `proofType: groth16` + one program verifying key (recognised and refused today) |
| wire | v3 envelope, proof slot **empty** | v3 envelope, **same** proof slot, filled |
| acceptance rules 1–4 | enforced | enforced, identically |
| judge / import list | validated live, in the node | validated live, identically |
| settlement on L1 | none | the dispute game, off the shelf |
| state validity | operator's word | proven |
| build & run | Go only, no proving toolchain | requires the prover |

**Same wire. Same proof slot. Same acceptance rules. One config value.** That is the claim, and it is
the reason attested mode is a *proving system* in the code rather than a bypass: it occupies the same
seam, returns the same interface, and is selected by the same field.

`proofType` has **no default**. A config that does not state the trust model is refused at load.
Both plausible defaults are wrong in a direction an operator cannot see: defaulting to `attested`
would silently stop verifying proofs on a node meant to verify them, and defaulting to `groth16`
would reject every batch of a correctly running v1 chain.

---

## How to check any of this yourself

Everything above is a test, not a promise.

| Claim | Test |
|---|---|
| the proof slot must be empty | `TestAttestedVerifier` |
| a proof-carrying batch does not move the proven head, end to end | `TestAttestedRefusesABatchCarryingProofBytes` |
| the trust model is always stated | `TestProofTypeIsNeverADefault` |
| a config ahead of the binary is told so, not called malformed | `TestFutureProofTypeIsRecognisedAndRefused` |
| the retired `mockProofs` spelling is refused with instructions | `TestMockProofsIsRetired` |
| imports are checked, not trusted | `TestSilhouetteImportsAMessageAndTheDependencyIsVerified` |
| a **false** import is refused and the frontier pins | `TestSilhouetteImportThatIsFalseIsRefused` |
| the judge's inputs never depended on the proof | `TestTheJudgeReadsTheWireNotTheProof` |
| **a fabricated export IS accepted** | `TestAttestedFabricatedExportIsAccepted` |
| a v1 deployment holds no proving artefact | `TestAttestedChainIsRenderedWithoutAProvingToolchain` |

That last-but-one row is the one to read twice. It is a **passing** test asserting a **weakness**,
and it is there because a limitation documented only in prose is a limitation that will be forgotten
by whoever reads the code six months from now. If it ever starts failing, someone has changed the
trust model — which is a thing to know deliberately.

See also: `README.md` (what this system is), `RUNNING-V1.md` (how to stand it up),
`SPEC-WIRE-V3.md` (the wire), `SPEC-ETH-NETFLOW-CAP.md` (the L1-side blast-radius bound).
