# Atomic demo: suspended execution

This unpublished builder library discovers cross-chain calls and nested callbacks using one
live OP EVM transaction per chain. A can call B, consume its result, and call B
again; B retains the state, warm slots and transient storage from its first leg.
B can also call back into A, or into a different contract C on A's chain, while
A's original call is waiting. Callbacks share that transaction's pending writes,
transient storage, origin and rollback checkpoints. Multiple destinations and
multiple callbacks during one waiting call are supported. Each included chain
gets one final canonical execution after witnesses and access lists are known.
There is no discovery restart or fixed-point loop, and no padding transaction.

`router::Builder` executes the actual `AtomicCallRouter`, CREATE2 facades and
`CrossL2Inbox`. An `Envelope` callback prepares ordinary transactions or signed
ERC-4337 `handleOps` calls. The integration tests use the exact OP EntryPoint v0.7
preinstall bytecode, deployed upstream SimpleAccounts and `AtomicPaymaster`.
Accounts and facades must be provisioned before discovery. The callback is a trusted
adapter for these envelopes; arbitrary smart-account validation/execution policies
are outside this demo's guarantees. Signing happens for discovery and finalization,
so this is not yet an interactive, single-signature wallet flow.

## How discovery works

`discover_with` runs ordinary OP validation and pre-execution once, then drives
REVM's existing frame loop. Selected local CALL/STATICCALL endpoints yield before
their child frame is created. `Paused` owns the live interpreter, memory, stack,
gas, return data, journal, warm accesses and rollback checkpoints. Dropping it
abandons that chain's speculative work; this crate never calls `DatabaseCommit`.
Database implementations must provide isolated snapshots and side-effect-free reads.

The router preloads the canonical witness/remote-call tapes, then enters application
code with an explicit fixed gas budget. It checks EIP-150 headroom and calls the
existing memory buffer directly, so even large calldata cannot silently truncate
that budget. Legacy router entry points remain available to the original Go demo.

During discovery the tapes start empty. Self-only, read-only getters expose the
waiting request or destination cursor. The coordinator obtains a remote result by
advancing the destination EVM, then resumes the root at that getter. To charge the
exact local getter gas, it executes the real getter bytecode in an isolated scratch
tape snapshot with the slots warmed as canonical preload would warm them. This
repeats only the small getter, never the application transaction prefix. The scratch
storage layout is pinned by tests against the compiled Solidity artifact.

The actual inbox code executes during discovery. The coordinator warms each newly
known checksum immediately before its router-originated validation call. The inbox
address is warm from the start. Final execution has no substitution/warming hooks
and uses the real encoded access list. Direct application inbox probes are rejected
by this adapter. No new consensus restriction on
SLOAD or new EVM opcode is introduced.

## Nested callback dispatch

The `executeRootNested` and `executeRemoteNested` entry points preload callbacks
keyed to their waiting outbound sequence. `remoteCall` dispatches those callbacks
through ordinary CALLs before reading the final result witness. A callback can
itself make remote calls; the coordinator switches between the same live machines
instead of starting additional transactions. Reserved witness slots let sequence
N+1 resolve before sequence N. `callFinished` is a self-only read-only acknowledgement
that exposes each incoming result before its parent unwinds, including failure.

Incoming calls save and restore the authenticated caller context. A callback gets
half the gas remaining at its dispatch point, leaving gas for unwinding and result
verification. It spends gas from the existing local call tree. Top-level application
calls retain their explicit fixed budget. Discovery and canonical replay compare
nested application calls as well as their enclosing calls. Global call limits and
a maximum scheduling depth of 32 bound recursion; EVM gas/depth limits still apply.

Callback failure hints live in the preloaded tape above application rollback scopes.
A false hint forces the entire router operation to revert even if application code
catches the actual callback failure. Unused callback entries are rejected, and consumed
call identities prevent duplication. Successful tape cleanup permits later bundles.
These hints authorize rollback; they are not proof that the builder honestly tried
success or grounds for automatic user billing.

The existing Go and Kona message-order cycle verifiers accept the nested call order.
Regression tests also preserve rejection of a true cycle; no verifier algorithm changes.

## Final acceptance and rollback

The primitive `Candidate::verify` still checks exact whole-transaction result and
state equality for equal-envelope experiments. Materializing router tapes changes
outer calldata, preload cost and fee accounting, so `router::Builder` instead:

1. Requires the expected account to invoke the exact prepared router calldata once,
   and observes the router's actual success/revert independently of the outer
   ERC-4337 transaction status.
2. Executes each final envelope once, with no discovery endpoints. Compares each
   application call's input, supplied/consumed/refunded gas, result, logs, and state
   effects, including empty-account deletion. Preload, cleanup and envelope fees
   remain outside that application comparison and execute through real bytecode.
3. Compares router results and surviving protocol logs, then matches every executing
   message to the actual source log's chain, block, timestamp, index, origin and hash.

A mismatch rejects the candidate; it never triggers another discovery pass. The
coordinator bounds remote operations, intercepted calls and endpoint bytes. Each
transaction also retains its EVM gas limit. These bounds do not implement billing
for off-chain simulation.

If B fails, its revert data propagates through the facade to A. A caught remote
failure still makes the root router revert during finalization. Successful suspended
destinations are discarded when A aborts; the root and a failed destination can be
replayed as reverted operations. EntryPoint can catch those reverts and settle the
paymaster's normal gas charge. This does not guarantee reimbursement of all builder
work or all outer transaction/L1 data fees.

**The returned bundle still requires the normal whole-block and interop protocol
verification gate before publication.** Local log matching is not dependency-set,
cycle, replacement-block or proof verification. Aborted access lists can reference
logs rolled back by the operations: the shared payload path must handle admission
consistently with the existing receipt-based verifier. This library does not change
the sequencer's interop filter or authorize bypassing the protocol gate.

## Integration boundary

`op-atomic-builder` is a private, one-bundle stdio worker. The Go adapter
`atomic.BuildSuspended` serves immutable account/storage/ancestor reads and signs
both envelopes locally. Rust recovers the sender from the exact signed EIP-1559
bytes, uses those bytes for OP fee accounting, and returns the identical bytes
from final replay. Keys remain in Go. Cancellation terminates and reaps the worker;
protocol messages, RPC requests and remote operations are bounded. No restart or
fallback to the original Go discovery algorithm occurs.

`SuspendedChain` requires the exact candidate environment and prefix snapshot.
`PrefixGas` means cumulative gas **before SDM refunds**, as used for block admission.
The current bridge accepts Lagoon EIP-1559 envelopes and OP's fixed blob base fee of
one. `RPCPrefixSnapshot` pins all reads by hash and anchors BLOCKHASH to that block's
ancestry; the host must retain its state throughout the build. It cannot turn a
parent snapshot into a correct next-block prefix by changing the timestamp.

The devstack scenarios pause sequencers and batchers, execute a system-only prefix
block on each chain, wait for its proof index, retain it through discovery and
replay, then replace it with a sibling containing the exact signed envelopes.
Batchers resume only after both final siblings are canonical. The fixture checks
for protocol-only transactions and
no prefix logs, and uses the prefix's base fee, gas limit, coinbase and randomness.
It then runs ordinary op-reth block execution and supernode interop verification,
checking application rollback, sponsored settlement, receipt gas and message logs.
The generic transaction plan detaches its signing dependencies to prevent later
nonce assignment from replacing the replayed envelope.

This is a **demo node integration**, not a production op-reth payload service.
A production adapter still needs a retained candidate-prefix provider and whole-block
admission/publication integration, including SDM accounting. The existing Go
`BuildSponsored` path remains available with its original discovery algorithm.

The EVM exposes current call gas (`GAS`) and the total block gas limit (`GASLIMIT`),
not a decreasing block-gas-left counter. Padding is unnecessary. Prefix state and
log counts still matter, and earlier transactions/system calls must be reflected
in the pinned snapshot. Gas-sensitive transactions that diverge after envelope
materialization are rejected, not repaired or retried.

This adapter supports uint64 chain IDs, ordinary user envelopes and zero-value facade
calls. It does not implement STATICCALL facades, ETH forwarding or distributed
caught-revert semantics. The legacy Go discovery adapter still supports only
root-driven calls; nested dispatch is connected to `BuildSuspended`.

## Validation

From `rust/`, after compiling contracts and the `AtomicCallRouter.t.sol` fixture:

```sh
cargo nextest run -p op-atomic-builder --run-ignored all
cargo clippy -p op-atomic-builder --all-targets --all-features -- -D warnings
cargo doc -p op-atomic-builder --no-deps
```

The artifact-dependent router tests are explicitly run by the contracts `just pr`
`atomic-suspension` check after its developer build. Ordinary Rust tests also cover
frame preservation, finite ping-pong, cancellation, strict replay rejection and the
Go message/access-list golden vector. The real-contract cases cover repeated and
three-chain destinations, root and remote reverts, caught failures, exact 4337
settlement, gas/transient-storage parity and malformed final envelopes/access lists.

This follow-up is stacked on the [atomic demo](https://github.com/ethereum-optimism/optimism/pull/22840).

Run the suspended worker against the two-chain op-reth/supernode devstack from the
repository root (with the normal acceptance-test prerequisites built):

```sh
RUST_JIT_BUILD=1 mise exec -- go test ./op-acceptance-tests/tests/interop/atomic \
  -run 'Test(Suspended|Nested)AtomicCalls' -count=1 -timeout=15m
```

The contracts `atomic-suspension` check also builds the worker and runs the Go bridge
suite, including rejected final access lists, exhausted prefix gas and cancellation.
