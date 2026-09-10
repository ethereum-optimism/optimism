# Atomic demo

This package discovers sequential cross-chain calls and materializes ordinary
transactions with result witnesses and CrossL2Inbox access lists. It does not
change the inbox, consensus verification, or the execution clients.

Applications opt into the new `AtomicCallRouter` / `AtomicRemoteProxy` helpers
in `packages/contracts-bedrock`. The root can call B, use the returned value in
another call to B or C, and reject the operation after seeing the results. A remote revert propagates
through the proxy into A without application-specific error handling.
Nested callbacks, ETH forwarding, view-call facades, and committing after a
caught remote failure are outside this prototype. Targets must route cross-chain dependencies through
the facade. See [the design](DESIGN.md) for the execution,
caller-identity, signing, and block-prefix constraints.

`Builder.Build` returns unsigned candidate transactions after discovery and
replay. `Plan.Reverted` marks an abort: A and the failing destination batch
both revert; all other successful destination transactions are omitted. If
A alone rejects the final result, only its reverted transaction is included. It never commits its speculative state. The caller signs the finalized
transactions, builds the complete block set, and applies ordinary interop
verification before treating it as accepted. Local unsafe execution alone is
insufficient: deliberately including successful remote work without root
completion causes that remote block to be replaced. Ordinary propagated failure
does not need block replacement: both included transactions simply revert.

`BuildSponsored` wraps the same flow in signed ERC-4337 UserOperations using
the existing EntryPoint v0.7 preinstall. A bundler submits `handleOps`; an opt-in
`AtomicPaymaster` funds gas for approved CREATE2 SimpleAccounts. The accounts
need no ETH. Application failure still rolls back application state, while the
outer EntryPoint transaction succeeds, records `UserOperationEvent.success =
false`, and charges the paymaster. Sponsorship is not a user billing system.

The router is also the remote facade factory: `predictProxy(chainId, target)`
returns its CREATE2 address and `proxyFor(chainId, target)` deploys it
idempotently. Application code calls the deployed address using the target's
normal interface, for example `ICounter(remoteAddress).add(amount)`. No special
cross-chain library is needed in that application. Setup still specifies the
destination chain and contract; the facade need not share the target's address.

Discovery is bounded and restarts transactions to learn results. Afterwards,
each included chain receives exactly one canonical replay using its final signed
UserOperation and access list. A changed result or message layout rejects the
candidate. This prototype does not resume saved call frames or charge for
discarded off-chain simulations. See the design for signing and fee constraints.

## Run the tests

From the repository root, build the contracts and upstream execution client:

```sh
cd packages/contracts-bedrock
mise exec -- just forge-build-dev src scripts interfaces
cd ../../rust
mise exec -- cargo build --locked -p op-reth --bin op-reth
cd ..
```

Run isolated EVM integration, Go verification, and Kona rule tests:

```sh
mise exec -- go test -tags atomic_integration ./op-test-sequencer/atomic ./op-supernode/supernode/activity/interop ./op-core/interop/messages ./op-interop-filter/filter -count=1
cd rust
mise exec -- cargo test --locked -p kona-interop test_detect_cycles --lib
cd ..
```

Run the actual two-chain op-reth/supernode tests:

```sh
mise exec -- go test -v ./op-acceptance-tests/tests/interop/atomic -run '^TestAtomicSynchronousCalls$' -count=1 -timeout=5m
RUST_JIT_BUILD=1 mise exec -- go test -v ./op-acceptance-tests/tests/interop/atomic -run '^TestSponsoredAtomicCalls$' -count=1 -timeout=7m
```

The separate Kona/challenger test additionally needs the repository's Cannon
binary, `kona-host`, and normal/interop Kona prestate artifacts:

```sh
mise exec -- just cannon-prestates
cd rust
mise exec -- cargo build --locked -p kona-host --bin kona-host
cd ..
mise exec -- go test -v ./op-acceptance-tests/tests/interop/proofs/serial -run '^TestAtomicSynchronousCalls$' -count=1 -timeout=15m
```

Prestate generation requires either the MIPS64 cross toolchain or Docker.

## Validation record

Checked against develop `7e167ae2e15b7bf7dfaf238e514155e4bc904ed6`:

- Local EVM integration and the affected Go packages pass.
- Kona's ten cycle-rule tests, including the atomic round-trip regression, pass.
- New contract/interface ABI checks and Solidity/Rust formatting checks pass.
- Repository Go lint and module-tidiness checks pass.
- The node scenarios check success, propagated remote failure (both receipts
  reverted), and adversarial inclusion of an orphaned successful remote leg.
  Final run results are recorded in the PR description.
- Both sponsored real-node scenarios pass: success and remote failure preserve
  the expected application state, emit the correct UserOperation outcome, and
  debit the paymaster deposit by the emitted gas cost. Smart accounts remain
  unfunded. Local regressions require one canonical replay per chain and pin
  the outer fee parameters used in simulation and submission.
- The sponsored two-round-trip test also checks identical gas supplied and
  consumed at the application and both proxy calls in final discovery and replay.
- The full Kona/challenger test has **not passed**: its runtime setup stops at
  missing `rust/kona/prestate-artifacts-cannon/prestate-proof.json`. Neither a
  MIPS64 cross linker nor Docker is installed in this environment. Native Kona
  rule tests are not a substitute for that full proof execution.

This is a draft prototype, not a deployed library or a production sequencer.
The PR description tracks remaining validation and production-readiness work.

The follow-up [`op-atomic-builder`](../../rust/atomic-builder/README.md) implements
suspended discovery through the router's fixed-gas entry points and self-only tape
getters. `BuildSuspended` connects that coordinator to Go over a private stdio
worker, keeping signing keys in Go and returning the exact signed envelopes used
in final replay. The devstack's `TestSuspendedAtomicCalls` exercises this path;
`BuildSponsored` retains the original restart-based discovery algorithm.

The suspended bridge requires a retained candidate-prefix snapshot, exact block
environment and cumulative gas before SDM refunds. Its RPC snapshot reads a pinned
block by hash; a parent block alone is insufficient. The system-only preview block
used in devstack must remain canonical until the worker finishes because op-reth
may discard state on unwind. Node inclusion still runs full block execution and
the normal interop verifier. A production op-reth payload service is not included.
