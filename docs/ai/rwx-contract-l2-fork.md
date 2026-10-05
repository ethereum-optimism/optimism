# OP Mainnet L2 fork shadow

The optional `optimism-contract-l2-fork-shadow` implements the complete original
`contracts-bedrock-tests-l2-fork op-mainnet` workload. Hosted same-SHA parity is
pending; it adds zero verified coverage so far.

The definition follows shared Contracts routing. It retains profile `ci`, main
features, every `test/L2/fork/**` signature and compiler artifact, original
working directories and recursive source submodules. The authoritative NUT
bundle check runs immediately before the original `just test-l2-fork-upgrade`
verdict. Failure retains its first JUnit and console evidence, with the original
`--rerun -vvvv` command captured separately. Diagnostic success cannot replace an
initial failure.

This shadow uses the official public `https://mainnet.optimism.io` endpoint.
Preflight runs before expensive preparation and verifies chain 10, the chosen
block hash, predeploy code, implementation storage and a version call. Complete
original RPC requests, response bodies and attempt metadata remain retained.
Circle's optional isolated replay uses that same endpoint and the exact native
block. Normal Circle jobs retain their existing private RPC selection.

The live read-only preflight verified block 157803345,
`0xdfa269ad424586197ac2059c1cba15cf08c07f13e4fd3c00406f65626c0031ed`,
including nonempty code/state. The public endpoint is documented in this source
tree and the pinned superchain registry. It supplies no new credential and makes
no transaction submissions. Full hosted execution still needs validation of
provider availability and rate limits throughout the workload.

Go modules, Go compiler outputs, Foundry outputs and pinned solc installations
use isolated reusable producers. Compilation runs on 16 CPUs / 32 GiB and fresh
verdicts initially use the same resources. Runtime Go compilation has its own
cache; reusable runtime outputs contain only that compiler cache and Foundry RPC
state. Test results, failure caches and logs are excluded. Protected `develop`
warming targets compilation and executes zero tests or NUT checks.

`c-l2_fork_parity_replay` defaults to false. Pilot-only API replay requires
`main_dispatch=false` and a numeric `c-l2_fork_parity_block`. It runs the original
module preparation and original L2 job through the shared evidence adapter.
Routing fixtures pin the exact enabled workflow set and preserve normal pushes,
main dispatches and other branches. Neither provider's parity checklist nor the
Contracts aggregate receives credit until complete original reports agree.


The first automatic native run `b2bc1ecdff844e2bbe7367cf321f3002` at `63844aed`
completed compilation in 657 seconds, then rejected an incomplete preflight mount
before executing tests. The consumer received only `block.json`; validation
correctly required its complete sealed RPC originals. The corrected consumer
mounts the entire preflight report, and a regression rejects an unsealed block-only
mount before any NUT or test command. The first failure remains retained.

For the final pilot batch, `ops/ci/pilot-l2-fork-block.txt` pins block 157804208,
`0xdef17f84f90e7d1bbf71a5ff53e9cb92201f9a31f8b7e63b3d84b55c82008673`.
This is a reproducible comparison input, bound to the complete source SHA, rather
than two independently selected heads. Both providers verify the entire public
block and state again before execution. Explicit numeric CLI blocks remain
available; `develop` continues to discover the latest head. A normal pilot push
executes the same shared original L2 runner in Circle's existing Contracts
workflow and the native coordinator, so the L2 workload and its genuine gate can
be verified alongside the selector replay in one batch. Other Circle refs retain
their original RPC selection and test path. Complete preparation/runtime consoles
are archived as files to avoid provider console truncation; success never hides
an earlier failed command or diagnostic rerun.

The corrected native coordinator at `936ef20e` ran all seven selected cases
successfully in 77 seconds of fresh verdict execution. Its compilation reused
the compiler cache and executed in 23 seconds. These are task observations,
not a provider speed comparison. Circle pipeline 135628/job 5638523 was killed
after exactly ten minutes without console output during full preparation.
The complete partial preparation stream and all untruncated API logs remain
retained in the [first combined batch](rwx-pr-gates-evidence/contracts-first-batch.json).
The preparation step now allows 30 minutes without output while retaining its
full original console archive. The failed Circle run adds no parity credit.

The next combined batch at `a3a09108` completed preparation successfully.
Circle job 5638688's initial verdict reported five failed `setUp()` cases, each
with the same original public RPC HTTP 429 storage-fetch failure. No failing
assertion was replaced: the separate diagnostic rerun passed all seven selected
cases and the shared runner still exited 1. Native fresh execution passed all
seven cases in 59 seconds. Complete initial and diagnostic JUnit, request frames,
console streams and all untruncated Circle API logs are retained in the
[second combined batch](rwx-pr-gates-evidence/contracts-second-batch.json).

Both providers now run the complete verdict with `--threads 1` and
`--compute-units-per-second 100`. The pinned Forge 1.8.3 binary and source support
these native controls. Only L2 suite concurrency and RPC throughput change;
the complete selection, profile, fuzz/invariant settings, fork height, original
NUT check and ten transport retries remain intact. The shared runtime settings
are recorded and compared, and changed/omitted limits fail verification.
The diagnostic rerun uses the same controls and cannot make an initial failure
pass. Twenty-one helper/comparison fixtures cover these boundaries and the
existing failure, source, compiler, block and report checks. They add no coverage.

The backoff-only combined batch at `cb341ad0` still hit public RPC HTTP 429
on both providers. Native's initial verdict retained four passing cases and
two failed suite setups; its separate diagnostic rerun passed all seven and
correctly kept the job and Contracts gate failed. Both complete first verdicts
remain retained. Forge's assumed compute-unit budget adjusts retry backoff;
it does not proactively pace each request.

The shared runner now sends fork traffic through a loopback transport with one
upstream request in flight and a global budget of two requests per second.
Request bytes, IDs, response bodies, timestamps and every transport attempt are
retained. Only HTTP 429/500/502/503/504 and connection failures receive bounded
backoff; permanent denials and RPC execution errors pass through immediately.
The relay caches no state and uses the same public endpoint. Test assertion
failures still retain the original failing verdict and separate diagnostics.
Complete transport validation checks its source-bound policy, every frame/hash,
response IDs, pacing and retry bounds. Common stable archive results must agree
across providers; different request inventories from Foundry's existing RPC
cache remain retained. Live head metadata is identified separately from pinned
archive state.

An isolated Linux probe ran the pinned Forge 1.8.3 binary and Solc 0.8.15 through
the actual relay against the real pinned OP Mainnet block. Both tests verified
chain, block and nonempty implementation storage and passed. All 19 requests
and attempts are retained, with zero 429s. Five unsupported `anvil_nodeInfo` or
`eth_getAccountInfo` probes returned original HTTP 403 method-denial responses;
Forge's normal fallback succeeded and the relay did not retry those denials.
This integration probe and 30 helper/comparison fixtures add zero coverage.
The [transport preflight](rwx-pr-gates-evidence/contracts-pacing-preflight.json)
binds those complete originals and the prior full-workload failures.
