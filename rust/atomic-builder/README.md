# Atomic demo: suspended execution

This experimental, unpublished crate lets a shared builder retain multiple OP
EVM transactions and advance them in serial. A registered local endpoint causes
the interpreter to yield before creating the child frame. The builder can run a
different chain, then resume the waiting call with a speculative response. No
transaction prefix is re-executed to resume a frame.

The integration test exercises A → B → A → B with one live transaction per chain.
B's second leg uses the state left by its first leg. Both chains pass one ordinary
final replay. A second case makes B revert and checks that A also reverts and both
application state changes roll back. A separate test keeps three EVMs suspended.

## Execution boundary

`discover` performs the normal OP transaction validation and pre-execution once.
It drives REVM's existing frame loop until a zero-value, non-static `CALL` reaches
a registered endpoint. `Paused` owns the EVM and the pending call. The program
counter, stack, shared memory, return data, remaining gas, storage journal, warm
accesses, transient storage and rollback checkpoints all remain live.

`resume` executes that endpoint normally. `resolve` skips its execution and inserts
an **untrusted discovery response**, using REVM's ordinary call-return machinery.
The supplied gas charge is for the *local endpoint*, not for execution on the
remote chain. Replies cannot mint gas or inject refunds, and call count and payload
limits bound discovery. Dropping a continuation abandons its isolated candidate.

Completion applies the normal OP post-execution once. `Candidate::verify` consumes
the candidate and performs one ordinary replay, with no discovery hooks. It checks
the complete result (including gas, status, output and logs) and journal state.
Any mismatch rejects the candidate; it does not restart discovery. This is a
deliberately strict local check. It does not establish cross-chain message
validity, authorize publication, or automatically commit state.

Only ordinary user envelopes are supported. Deposits, synthetic PostExec envelopes
and system calls require their own lifecycle and are rejected by this entry point.
Calls involving value, STATICCALL, DELEGATECALL and CREATE continue through normal
REVM execution; they are not replaced by speculative replies.

## No block padding

The EVM exposes the current call's remaining gas (`GAS`) and the block's total gas
limit (`GASLIMIT`). It does not expose a decreasing block gas counter. A preceding
transaction's gas use affects whether a later transaction fits; it does not reduce
that transaction's declared execution budget. The regression test runs the same
application after two prefixes with different gas usage and compares its GAS,
GASLIMIT, output and receipt gas.

The builder still has to pin the actual prefix state and block/transaction
environment for every chain. Earlier transactions can change storage and balances;
prefix log counts also determine message identifiers. Pausing must not allow another
transaction to modify the candidate state under an active frame. Each chain keeps
its own block gas limit and gas budget; different chains need not share them.

Warm accesses are scoped to a transaction. Retaining the journal preserves them
across suspension. An application cannot SLOAD another contract's storage, but it
can probe an inbox through calls, so warming discovered checksums is not by itself
a proof that all gas-sensitive behavior matches final replay. Every admitted inbox
call must be handled consistently, and final replay must enforce the real access
list and message checks. No new consensus restriction on SLOAD is proposed here.

## Router integration status

This is the continuation primitive and an executable scheduling experiment. It is
not yet connected to `op-test-sequencer/atomic`, the ERC-4337 adapter, the real inbox
or the op-reth payload service. The existing Go discovery path still restarts.
The tests use small bytecode fixtures with already-materialized response endpoints;
they do not demonstrate discovering or publishing an actual interop bundle.

The remaining adapter work is material:

- The current router copies the entire result witness tape before entering the
  application. An adapter must discover results without charging unknown future
  setup costs to the already-running application. An explicit fixed application
  call budget, with enough outer gas to avoid EIP-150 truncation, is a candidate
  solution. This crate does not implement that contract change.
- The destination router takes the entire batch in calldata. Streaming newly
  discovered legs into a live destination transaction needs a discovery driver
  whose application-visible state, caller context, local gas and logs match the
  canonical batch. Reusing separate top-level transactions would lose transient
  storage and warm-slot parity.
- Witnesses, access lists, ERC-4337 envelopes and log coordinates must be
  materialized before the existing message/whole-block acceptance checks. The
  strict whole-transaction comparator here is useful for equal-envelope tests;
  transformed envelopes need an explicit comparison of application execution plus
  independently validated wrapper fees and effects, not an arbitrary weakening of
  equality.

These are builder/adapter concerns; retaining frames requires no new EVM opcode,
padding transaction or gas-refund protocol. This crate makes no claim that the
current router can already execute discovery in a single pass.

## Validation

From `rust/`:

```sh
cargo nextest run -p op-atomic-builder
cargo clippy -p op-atomic-builder --all-targets --all-features -- -D warnings
cargo doc -p op-atomic-builder --no-deps
```

Related draft: https://github.com/ethereum-optimism/optimism/pull/22840.
