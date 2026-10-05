# OP Mainnet L2 fork shadow

The optional `optimism-contract-l2-fork-shadow` implements the complete original
`contracts-bedrock-tests-l2-fork op-mainnet` workload. Complete hosted same-SHA
original parity passes at `f821983dd56cbd7e488ab903d1ac386a330de7f6`. Both
providers execute all seven initial cases and retain all 407 runtime relay
requests, with zero diagnostic reruns, transport retries or HTTP 429s.
The [complete comparison](rwx-contract-l2-fork-evidence/parity.json) retains
source, commands, compiler, selection, settings, block, RPC and original results.
This closes the L2 fork occurrence in the 86-job baseline.

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
Circle's pilot branch and optional isolated replay use that same endpoint and
the exact native block. Circle jobs on other refs use their existing private RPC
selection.

The live read-only preflight verified block 157803345,
`0xdfa269ad424586197ac2059c1cba15cf08c07f13e4fd3c00406f65626c0031ed`,
including nonempty code/state. The public endpoint is documented in this source
tree and the pinned superchain registry. It supplies no new credential and makes
no transaction submissions. Public archive availability and rate limits remain
runtime constraints. The completed hosted comparison below retains the actual
requests and transport outcomes.

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

The combined automatic batch at `3be585cb` passes the original result comparison;
its effective Circle runtime transport was subsequently found unverified.
Circle pipeline 135642, original [job 5639658](https://circleci.com/gh/ethereum-optimism/optimism/5639658),
and native [run d287cd4e](https://cloud.rwx.com/optimism/runs/d287cd4ef59b4c02b11cfbbf03d0e7f5)
both pass all seven initial cases, with zero skips or diagnostic reruns. All
selection, compiler methods and bytecode, effective settings, submodules,
commands, original outcomes and pinned-block frames agree. Circle retains all
58 report files and complete untruncated API logs; native retains all 1,686
report files, including 407 upstream requests and their complete response frames.
Native observed zero HTTP 429s and zero transport retries. Thirteen original
HTTP 403 capability denials passed through once; Forge's normal fallback succeeded.

Circle's relay report contains zero requests. The next batch at `30a54d78`
failed its initial verdict on HTTP 429, again with zero relay requests. A real
shell reproduction establishes that Circle's `BASH_ENV` reinitializes nested
Bash shells and overwrites the runner's loopback URL with the public endpoint.
The earlier attribution to warm Foundry state was incorrect. The Python runner
now removes `BASH_ENV` after the caller initializes the job environment and
before applying its runtime overrides. Exported job values remain inherited.

All 32 L2 helper/comparison tests pass, including a real nested Bash request
through the relay. A pinned Forge 1.8.3 / Solc 0.8.15 probe runs through two
nested Bash shells against the original pinned archive block: two tests pass,
all 19 requests use the relay, and no request receives HTTP 429. Five original
capability-denial HTTP 403 responses pass through once. Complete frames and
implementation hashes remain in the [corrective preflight](rwx-pr-gates-evidence/contracts-shell-preflight.json).
These local checks add zero hosted workload coverage.

The reader still accepts sealed zero-request reports: a fully warm RPC cache
can validly need no upstream calls. That condition must not be inferred from a
zero count alone. The complete fresh verdict and pinned-block checks still apply;
all 15 comparison fixtures pass without changing any original report.

Circle's full contract build took 649.3 seconds and its fresh verdict took 58.9
seconds. Native compilation executed in 19 seconds and its fresh verdict task
executed in 240 seconds on 16 CPUs / 32 GiB. These observations have different
runtime transport paths and do not establish a provider speed comparison. The
prior Contracts aggregate comparison also retains all 21 exact prerequisites
and its fresh final status. The corrected combined verification below supersedes
that runtime-path evidence.

The corrected combined batch at `f821983d` proves the actual nested-shell fix.
Circle pipeline 135650, original [job 5639942](https://circleci.com/gh/ethereum-optimism/optimism/5639942),
and native [run 7a0f17fd](https://cloud.rwx.com/optimism/runs/7a0f17fd80244703bc3b2a25c941c2da)
both pass all seven initial cases and preserve the same complete settings,
compiler artifacts, commands and pinned block. Each report contains all 407
relay requests and attempts: 394 HTTP 200 responses and thirteen original HTTP
403 capability denials. Forge's existing fallback succeeds; the relay retries
neither denial. Both providers record zero HTTP 429s and zero transport retries.
Complete original collections and sealed reports pass the strict reader. Both
genuine Contracts gates also pass their 21 prerequisites. The historical
[3be5 comparison](rwx-contract-l2-fork-evidence/3be5-batch-parity.json) and complete
first failures remain retained; no original failed verdict was replaced.
