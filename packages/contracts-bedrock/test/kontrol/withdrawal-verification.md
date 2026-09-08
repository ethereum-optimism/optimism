# Withdrawal authorization

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
game addresses), and `uint64` creation/resolution/proof/retirement/current times. The registration
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

The eligibility/deletion fixture fixes game identity to type 0, root 0, and a 32-byte sequence number 1. The
game fixture also supplies the independent pause input. It models normally returning getters;
it does not establish real game behavior, arbitrary identity/extra-data handling, or pause access
control. The factory mapping and Portal proof record are installed directly at documented slots.
`prove_checkWithdrawal_eligible_succeeds` supplies a concrete acceptance witness. A successful
component result must not be labeled proof of authentic withdrawals.

The finalizer obligations under development connect this component to both finalization entry points.
`prove_finalizeWithdrawal_ineligible_reverts` uses the canonical hash of every withdrawal field,
an arbitrary caller and proof submitter, and a symbolic choice of entry point. Its rejection premise
uses the independent eligibility expression above; the finalizer executes its actual checks.
The separate equivalence obligation compares `checkWithdrawal` with that same expression, avoiding
a duplicate production eligibility call inside the general finalizer obligation.
The calldata bytes have symbolic length within Kontrol's compiler-compatible `uint64` length domain;
no fixed-length annotation is used. `prove_finalizeWithdrawal_eligible_succeeds` supplies an eligible
example for both entry points. These obligations retain the fixture's deployment and dependency
scope; they do not establish proof-record provenance or the later execution/accounting guarantees.

`prove_deleteProvenWithdrawal_equivalence`, also under development, checks deletion eligibility and
the selected record's removal or preservation. It seeds an arbitrary, distinct withdrawal/submitter
record and an independently chosen finalized flag, and asserts both are unchanged. The observed
withdrawal hash may equal the deleted record's hash when the submitters differ. A blacklisted-record
witness checks that deletion is possible. This covers deletion's local preservation obligation;
the authenticity of newly proven or re-proven records remains separate and unfinished.

`prove_proveAndFinalize_eligible_succeeds` is an acceptance witness that constructs a canonical
single-leaf storage proof and calls the actual proving method, checks the record it creates, then
advances the game reports and time and calls either finalizer. It includes legacy type 0 and super
type 4; the latter checks the configured chain ID and uses a single-chain Super Root v1 commitment,
asserted distinct from the per-chain output root so a wrong getter cannot satisfy the witness.
Both acceptance witnesses use 30 million concrete gas and a different caller from the proof submitter
for the external-proof entry point.
Game reports and factory registration remain fixture preconditions. The single-node witness is a
concrete acceptance example, not a bound on the required universal inclusion theorem or a proof
of protocol-wide history safety. This obligation has no proof result yet.

`prove_proveWithdrawal_recordTransition_succeeds` is the general record-update obligation under
the registered game-report fixture above. The withdrawal tuple, output tuple, candidate output claim,
submitter, prior record, full `uint256` proving time, `uint32` game type and encoded witness are symbolic.
The fixture independently classifies types 4, 5, 7, 9 and 10 as super games; other types use legacy roots.
A successful production proving
call must bind the entire output tuple to the candidate's per-chain claim and write that candidate's
address and the timestamp cast to `uint64` together. A reverted call must preserve the old record. Both outcomes
must preserve an arbitrary distinct withdrawal/submitter record and an observed finalized flag.
The prior and observed record words range over all 256 bits; the expected successful write retains
the prior slot's unused high 32 bits. No authenticity premise is attached to a seeded prior record.

The witness input is `bytes`, decoded by the fixture's `decodeProof` adapter into `bytes[]`; malformed
ABI encodings return before the proving call. This avoids Kontrol's one-element default for a direct
`bytes[]` parameter. It imposes no RLP, path, node-count or membership assumptions on the decoded
witness. The input-domain correspondence still requires auditing generated symbolic byte constraints
and the adapter's coverage of compiler-supported ABI encodings. The acceptance sequence also exercises
the adapter. The frame premise selects a distinct record, not a restricted subset of withdrawals.
This obligation is unproved and does not establish trie-verifier soundness: acceptance-to-membership
and composition across repeated proving/deletion transitions remain required separately.

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
Strict mode saves after each completed proof step and logs initialization stages. An interrupted
initialization or unfinished first step may still leave no graph; absence is not evidence of a pass.

Before these methods, strict mode separately attempts a word-copy loop claim for the exact fixture
runtime, with a 15-minute timeout. The generator requires the complete opcode loop and its matching
jump destinations. The claim covers a symbolic prefix copied into disjoint memory, tracks memory
expansion, and stops before tail clearing. It uses symbolic infinite gas and leaves the final gas
formula unspecified, so it supplies no gas bound and cannot apply to the concrete-gas witnesses.
It explicitly enables stack checks, matching the strict caller model and the EVM stack limit;
the proof does not quantify over Kontrol's optional stack-check disabling configuration.
The helper states stack space at prefix sizes 3 through 6 explicitly: KEVM's symbolic
stack-count accumulator does not automatically relate those counts to the tail-length bound.
Prefix 3 is the tail after the final `ADD` consumes its operands, as checked by KEVM's optimized rule.
Callers must establish these conditions from their actual stack; no withdrawal-input assumption
or unchecked stack configuration is permitted to force applicability.
The endpoint is the unique multiple of 32 in `[LENGTH, LENGTH + 32)`, equivalently
`32 * ceil(LENGTH / 32)`. Stating its range and alignment keeps the endpoint symbolic
instead of substituting a quotient throughout the memory and stack postconditions.
The target binds the observed final index and requires it to equal this endpoint through
both inequalities. The exact byte-copy and memory-expansion postconditions use that same index.
Its specification and native graph are archived under `kout-proofs/copy-loop`.
The runner also stops after basic-block bookkeeping, exposing the claim's plain `#execute`
state at loop entry and exit; jump-only cut points stop before that bookkeeping finishes.
CI attempts both this claim and the Solidity methods; either failure fails the job.
This claim is under development and is not imported as an execution summary. Any later composition
must audit the native guarded circularity: the induction hypothesis becomes available only after
execution progress, and the loop decreases the nonnegative remaining byte count by 32 per iteration.
It is not a trusted claim. Composition
must first verify its completed graph and establish all its memory, stack and arithmetic premises
at the caller; those premises must not become restrictions on the parent theorem's inputs.

Strict mode returns Booster's symbolic branches directly and retains legacy fallback for stuck
or aborted execution. It retains post-execution simplification, including elimination of branches
the simplifier establishes as impossible. This avoids an expensive legacy reconfirmation of each
branch. The pinned backend checks branch applicability, definedness and remainder coverage before
returning branches.
Strict fixture mode uses the fresh deployment-state diff and `setUp`, enables stack checks,
uses CANCUN with gas tracking enabled. General methods retain Kontrol's symbolic infinite-gas model,
which suppresses implicit out-of-gas halts while retaining explicit `GAS` observations; the acceptance
witnesses set concrete gas. `--no-gas` is unsuitable for finalizer proofs because it initializes gas
to zero while `SafeCall` still checks `gasleft()`, making rejection uninformative. The earlier
gas-disabled eligibility results remain component evidence, not finalizer evidence.
Strict mode omits `--assume-defined` and builds without the repository's
pausability lemmas. The pinned compiler flattens contract-qualified imports into a shared main
module, so each suite is rebuilt separately; only the existing suite imports those lemmas.
Gas adequacy is not established by these obligations.
Kontrol's hash/storage-separation assumptions still apply. Record artifact/compiler identities
with results; the production source classes alone do not identify any particular live deployment.
Accept a result only when all selected proof graphs pass without pending, failing, or admitted
obligations and the acceptance witness passes. Compilation and JUnit alone are insufficient.

The current branch temporarily dispatches a copy-loop-only CI diagnostic with
`KONTROL_COPY_ONLY=true` to publish its graph before the longer Solidity phase. This mode
requires strict mode and preserves the helper's failure status. It is not full-suite evidence.
The current diagnostic stops after 12 proof iterations to retain the first exit, then asks the
backend why that saved state does not imply the target. This query runs in CI, has a two-minute
limit, and never changes the saved proof or its exit status. Its JSON report is diagnostic evidence,
not a proof certificate. Restore the normal helper iteration limit before assessing loop completion;
the normal integrated path retains 10,000 iterations.
The temporary diagnostic also retains the backend's simplified implication and unification-failure
log, because the higher-level failure report can be empty even when implication fails.
Before PR readiness, restore the CI command to `test-kontrol-no-build` without that variable
and verify the original suite, every withdrawal obligation, and the independent helper.
