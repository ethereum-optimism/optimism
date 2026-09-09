# Withdrawal eligibility component

These component proofs do not establish end-to-end theft prevention or arbitrary-history safety.
The current obligations exercise `OptimismPortal2`, `AnchorStateRegistry`, and `DisputeGameFactory`
behind their proxies in the freshly generated fault-proof deployment. The normal deployment build
supplies bytecode through the repository's existing state-diff mechanism. Importing only interfaces
avoids recompiling dependencies with the fixture's optimizer settings. No implementation is
subclassed or given additional setters. External storage setup establishes component preconditions,
not reachable-state provenance or correct initialization.

## Trust boundary

The requirements are the [Portal specification](https://specs.optimism.io/fault-proof/stage-one/optimism-portal.html)
and [registry specification](https://specs.optimism.io/fault-proof/stage-one/anchor-state-registry.html).
The detailed baseline inventory uses `ethereum-optimism/specs@8be1cc9f9e833b34c8c3a1f81592e31d2440d234`.

| Link in withdrawal authorization | Current treatment |
| --- | --- |
| L2 execution to correct output commitment | Upstream obligation; not established here. |
| Game reports and factory registration reflect actual game history | Game reports and registration storage are seeded; creation/resolution are not proved. |
| Incorrectly resolved game is invalidated in time | Official assumptions aOP-003/aASR-003; never inferred from an accepted registry response. |
| Registry applies eligibility and invalidation checks | Actual registry and factory lookup code in the component obligation. |
| Withdrawal fields and inclusion witness match the candidate commitment | Next obligation; no current inclusion proof or record-provenance result. |
| Portal accepts only an eligible, mature, unconsumed record | `prove_checkWithdrawal_equivalence`, together with registry checks. |
| Authorized execution, accounting, and preservation across histories | Later obligations; not established by checking a seeded record. |

An incorrectly resolved, otherwise eligible game can authorize a false commitment if it is not
invalidated in time. A correct Portal does not independently reexecute L2. Establishing actual
L2 authenticity therefore requires the upstream correctness/invalidation assumptions in addition
to inclusion verification. These assumptions must remain explicit in any composed result.

## Component scope

`prove_checkWithdrawal_equivalence` quantifies over withdrawal hash, submitter, all three valid
game statuses, eligibility flags, every packed factory registration word (including other nonzero
game addresses), `uint64` stored creation/resolution/proof/retirement times and `uint256` current time. The registration
word is assembled from independent 32-bit type, 64-bit timestamp and 160-bit address inputs. Their
disjoint bit ranges cover all 256 bits, so this representation includes every possible word.
Future timestamps and equality boundaries remain in the domain. Delay parameters are read from
the deployment's immutable getters. Both source checks use strict `>` boundaries.
At proof-age equality, the implementation rejects, while the pinned Portal specification's
`Finalized Withdrawal` definition says "at least" the maturity delay and `checkWithdrawal` requires
rejection below it. This theorem uses the implementation's stricter `>` boundary: it covers the
required rejection below the delay but does not claim equality with that prose at the boundary.
The finalized and blacklisted inputs are integers constrained to 0 or 1, equivalent to the
boolean domain, so seeding storage does not introduce conditional conversions. The independent
eligibility expression is evaluated after the production call to reduce duplicated exploration.

The game identity is fixed to type 0, root 0, and a 32-byte encoding of sequence number 1. The
game fixture also supplies the independent pause input. It models normally returning getters;
it does not establish real game behavior, arbitrary identity/extra-data handling, or pause access
control. The factory mapping and Portal proof record are installed directly at documented slots.
`prove_checkWithdrawal_eligible_succeeds` supplies a concrete acceptance witness. A successful
component result must not be labeled proof of authentic withdrawals.

This candidate extends the historical fixture's current-time input from `uint64` to `uint256`.
Earlier passing artifacts therefore do not certify this candidate; fresh proof results and
their initialized domains must be inspected. Actual-finalizer linkage is a separate pending
milestone. This component neither executes either finalizer nor establishes proof-record provenance.

## Reproduction

Use pinned Foundry/Kontrol versions and a clean proof output directory. From `packages/contracts-bedrock`, run:

```sh
mise x -- just build-go-ffi kontrol-summary-full test-withdrawal-authorization
```

The normal `test-kontrol-no-build` recipe runs the existing pausability proofs and then these
component proofs. Their logs, JUnit report and proof archive go into `test/kontrol/logs/withdrawal`
to preserve both suites' results. Separate selectors allow the witness to run alongside equivalence.
Strict mode limits proving to 60 minutes inside the container so the host can collect saved graphs
after a timeout. A timeout is incomplete, never a proof pass.

Strict mode returns Booster's symbolic branches directly and retains legacy fallback for stuck
or aborted execution. It retains post-execution simplification, including elimination of branches
the simplifier establishes as impossible. This avoids an expensive legacy reconfirmation of each
branch. The pinned backend checks branch applicability, definedness and remainder coverage before
returning branches.
Strict fixture mode uses the fresh deployment-state diff and `setUp`, enables stack checks,
uses CANCUN and abstracts gas. It omits `--assume-defined` and builds without the repository's
pausability lemmas. The pinned compiler flattens contract-qualified imports into a shared main
module, so each suite is rebuilt separately; only the existing suite imports those lemmas.
Gas adequacy is not established by these obligations.
Kontrol's hash/storage-separation assumptions still apply. Record artifact/compiler identities
with results; the production source classes alone do not identify any particular live deployment.
Accept a result only when all selected proof graphs pass without pending, failing, or admitted
obligations and the acceptance witness passes. Compilation and JUnit alone are insufficient.

## Bounded finalizer experiment (not a completed proof)

`test-finalizer-smt` is a separate CI-only feasibility job using production Solc 0.8.15 and
Z3 from the pinned Kontrol image. It instruments an ephemeral copy of `OptimismPortal2` with
an assertion immediately before the actual finalization write: the selected hash is unconsumed
and its record for the selected submitter has a nonzero timestamp. Both original finalizer
bodies, hash computation and `checkWithdrawal` body remain. Reversing all instrumenting edits
must restore the original source exactly. No production file is edited.

An internal function with the compiler's documented `abstract-function-nondet` annotation
introduces arbitrary starting storage before the finalizer guards. The unrelated proving
method is also abstracted; no trie or record-provenance property follows from this experiment.
The assertions concern source-level control flow under SMTChecker's abstraction of Solidity;
compiler/encoder/solver correctness is assumed. They do not establish ABI correctness, gas
adequacy, committed transfers, arbitrary-history authenticity or correctness of game resolution.

Solc exports its actual SMT-LIB queries, Z3 executes them in CI, and the unmodified responses
are fed back to Solc through its supported `auxiliaryInput.smtlib2responses` interface. The
source-located target must first be reported unresolved and then close with no unanswered
queries or unknown solver responses. Input, output, queries, responses and versions are artifacts.
Large compiler outputs and queries are gzip compressed without changing their contents. Solc
0.8.15 can reorder conjunctions between processes and request a different query hash. The runner
solves each newly requested query verbatim, retaining the actual response for that exact hash.
It permits at most eight response rounds within the same total budget; exhaustion is failure.
An old response is never reassigned to a new hash, even for apparently equivalent queries.
Solc's CHC SMT-LIB adapter uses `sat` for safety; the runner delegates interpretation to Solc.
Controls omit the check or change the submitter. A third diagnostic asserts `false` at the
authorization point to detect vacuity in the abstract model; it is **not** a concrete EVM success
witness. A concrete admissible success case and complete guard/entry-point linkage are still
required before advertising the finalizer milestone.

Each experiment has a 30-minute total budget and two-minute solver queries. At most two focused
CI attempts are planned before reassessment. `check-finalizer-smt` only compiles the four
instrumented variants with the SMT engine disabled; its success is not proof evidence.
