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
