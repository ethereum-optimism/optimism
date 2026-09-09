# Atomic demo design

The selected design keeps the existing CrossL2Inbox, access lists, EVM rules,
and log-order cycle verifier. New opt-in application helpers live in
`packages/contracts-bedrock`; no existing protocol contract is changed.
This is an experimental coordinator, not a production sequencer or a transparent
replacement for arbitrary EVM calls.

## Execution

`op-test-sequencer/atomic.Builder` constructs one root transaction and one
transaction per destination chain. It supports sequential root calls to N
chains, including repeated calls to a previously visited destination.

The coordinator executes serially and restarts speculative transactions as it
learns results. It needs no suspended Reth call frames:

1. Pin each chain's parent/prefix state, block number, timestamp, sender, and
   first block-global log index.
2. Run the root with its known witness tape. The first missing result produces
   a structured revert identifying the next remote call.
3. Replay that destination's accumulated operations from its pinned state.
   Earlier writes remain visible to later operations in the batch. Record the
   new return value and result log position.
4. Append that result and restart the root. Continue until completion or failure.
5. On success, resolve root request/completion positions and construct ordinary inbox
   access lists for all transactions.
6. Replay with real bytecode and final access lists. Reject changed output or
   application log layout, missing sources, or mismatched payloads.

When a remote target reverts, the destination batch reverts and the router wraps
its sequence and revert bytes. The builder supplies those bytes as a failed
result hint, and A's proxy bubbles the original revert into A. Final replay must
reproduce both failures with no surviving logs. The returned `Plan.Reverted`
contains only A and the failed destination; earlier successful destinations are
omitted. If A's own final profitability check rejects successful remote results,
only reverted A is included. Including failed B spends its normal gas/nonce; it
is useful for the demo's explicit two-receipt assertion, not needed for rollback.

Failed hints are **unauthenticated abort hints**, not proven return values. Failed
EVM calls leave no initiating result log. The router therefore forces its entire
transaction to revert if any supplied witness is false, even when the application
catches the error or never consumes the hint. No successful transaction can commit
using a false hint. Successful witnesses keep normal message authentication.

The caller signs the finalized transactions, builds complete blocks through the
engines, and runs ordinary interop validation. The builder cannot rewrite a
pre-signed transaction's calldata or access list. A separately signed intent
and relayer authorization mechanism is a future feature.

Repeated replay is intentionally simple and can cost quadratically in call
count. Call count, gas, RPC discovery passes, and cancellation bound the work.

## Application facade and dependencies

`AtomicCallRouter.proxyFor(chainId, target)` creates a local
`AtomicRemoteProxy`. Applications call that proxy using the target's ordinary
ABI. `AtomicCallExample` demonstrates two typed `add()` calls to a remote
`AtomicCounter`, where the second call uses the first result.

The root emits each request before validating its result. The destination
validates the request, runs the target, and emits a result commitment.
Every destination also validates root completion after its final result:

| Log | A | B |
| --- | --- | --- |
| 0 | request 1 | validate A request 1 |
| 1 | validate B result 1 | result 1 |
| 2 | request 2 | validate A request 2 |
| 3 | validate B result 2 | result 2 |
| 4 | root completed | validate A root completed |

The transactions depend on each other, while their executing-message graph is
acyclic under the existing log-order rule. Independent application commitments
avoid a circular hash construction; identifiers do not contain transaction hashes.

As a separate adversarial case, if A reverts, its request and completion logs
disappear. A builder might still include previously successful B work. B may execute
locally, but ordinary cross-chain validation rejects its missing dependencies
and replaces its block. Unsafe execution alone is not an atomicity guarantee.
A can remain a valid reverted transaction with ordinary nonce and fee effects;
the atomicity guarantee concerns application state.

## Adapters and limits

`EVMExecutor` copies a pinned op-geth StateDB. During discovery only, a
call-entry hook warms requested checksum slots before the real inbox executes.
The inbox's code and emitted logs are preserved. Final execution has no hook.

`RPCExecutor` uses `debug_traceCall`, `callTracer`, and block overrides.
It learns cold checksum slots from failed calls and warms them in subsequent
discovery attempts. Final replay uses only the finalized access list. It
reconstructs log order across subcalls and excludes reverted scopes.

The RPC adapter simulates an empty next-block prefix. Applications depending on
next-block system writes, deposits, preceding transactions, or other changed
block environment fields need a full candidate-prefix adapter. The devstack
counter cases avoid these inputs and then submit real payloads for validation.

Other constraints:

- Nested remote calls and callbacks into the active root are rejected.
- All chains must use the same router implementation/address; the application
  chooses and trusts that deployment domain.
- Targets see the router as `msg.sender`. `crossChainContext()` exposes the
  authenticated source chain and caller. Native caller semantics are not preserved.
- Native ETH forwarding, STATICCALL/view facades, and distributed caught-revert
  behavior are outside this version. A remote failure always aborts the root,
  including when application code tries to catch it.
- Nonces and consumed-call identities prevent replay. The proxy deliberately
  has no public `version()` getter, preserving forwarding for that selector.
- Candidate message checks complement full block/dependency-set/cycle validation;
  they do not replace it.
- Fresh atomic pairs use controlled shared block construction. No general
  transaction-pool/filter exemption is added.

## Validation and review split

Local EVM tests cover two round trips, three chains, propagated revert and rollback,
omission of earlier successful destinations, caught/unused failure hints, access-list
requirements, source/result tampering, replay protection, and session/bound checks.
The cycle regression includes the final root-completion dependency.

The devstack test under `op-acceptance-tests/tests/interop/atomic` builds real
blocks and checks supernode acceptance/replacement and application storage.
The corresponding `interop/proofs/serial` test additionally invokes Kona and
the challenger. Commands and runtime status are in
`op-test-sequencer/atomic/README.md`.

Review capability/regression coverage first, then the helper library and builder
integration. The draft keeps these together in separate commits to demonstrate
the complete flow without changing existing consensus rules.
