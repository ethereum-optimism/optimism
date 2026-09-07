# Withdrawal authorization

Work in progress: no end-to-end theft-prevention or arbitrary-history theorem is complete.
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
game statuses, eligibility flags, an arbitrary packed factory registration word (including other
nonzero game addresses), and `uint64` creation/resolution/proof/retirement/current times.
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

## Reproduction

Use pinned Foundry/Kontrol versions and a clean proof output directory. Run the component proofs with:

```sh
just build-go-ffi kontrol-summary-full test-withdrawal-authorization
```

The normal `test-kontrol-no-build` recipe runs the existing pausability proofs and then these
component proofs. Their logs, JUnit report and proof archive go into `test/kontrol/logs/withdrawal`
to preserve both suites' results. Separate selectors allow the witness to run alongside equivalence.
Strict mode limits proving to 60 minutes inside the container so the host can collect saved graphs
after a timeout. The branch currently runs the component alone in CI with a 90-minute overall limit;
restore the normal suite command before a PR. A timeout is incomplete, never a proof pass.

Strict mode returns Booster's symbolic branches directly and retains legacy fallback for stuck
or aborted execution. It retains post-execution simplification, including elimination of branches
the simplifier establishes as impossible. This avoids an expensive legacy reconfirmation of each
branch. The pinned backend checks branch applicability, definedness and remainder coverage before
returning branches.
An isolated comparison found equal parent states and equivalent branch states after collection
ordering, Boolean representation and explicit-equality substitution were accounted for. That
comparison is diagnostic evidence; acceptance still requires the fresh complete proof graphs.

Strict fixture mode uses the fresh deployment-state diff and `setUp`, enables stack checks,
uses CANCUN and abstracts gas. It omits `--assume-defined`. Existing pausability lemmas are imported
only into their original proof contract. Gas adequacy is not established by these obligations.
Kontrol's hash/storage-separation assumptions still apply. Record artifact/compiler identities
with results; the production source classes alone do not identify any particular live deployment.
Accept a result only when all selected proof graphs pass without pending, failing, or admitted
obligations and the acceptance witness passes. Compilation and JUnit alone are insufficient.
