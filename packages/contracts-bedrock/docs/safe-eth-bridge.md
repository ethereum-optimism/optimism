# SafeETHBridge: atomic asynchronous native ETH PoC

This is an experimental replacement implementation for `SuperchainETHBridge`, named
`SafeETHBridge`. It uses **ordinary native ETH**, the existing `ETHLiquidity`, and
`L2ToL2CrossDomainMessenger`. It does not create a token, wrapped balance, independent
liquidity pool, synchronous execution mechanism, or custom relayer.

Deploy the implementation on each of two L2s with the same `(chainA, chainB)` arguments,
then install it behind the **existing** `SuperchainETHBridge` predeploy proxy on those
fresh test chains. This is necessary because `ETHLiquidity` only authorizes that address.
The acceptance test does this through the devnet proxy admin. No production deployment,
new predeploy address, network upgrade transaction, messenger, inbox, or production
bridge source is changed. Legacy send/relay entrypoints remain available, but a live
production migration is not validated by this PoC.

## Optional modes

- `sendETH(recipient, chainId)` retains ordinary immediate source locking and one-message
  delivery via `relayETH`. It has no timeout refund and supports the original chain routing.
- `initiateTransfer(recipient, deadline)` opts into reversible escrow and the three-message
  safe protocol below, between the configured pair. Both modes pay the same native ETH.

Ordinary destination relays can spend only **unreserved** ETHLiquidity capacity. If safe
transfers have reserved the remaining capacity, an ordinary relay reverts and must be
retried after funding or reservation release. The safe reservation always has priority;
ordinary bridging therefore retains its burn-first liveness limitations. Existing
ordinary message selectors and events are preserved, with messenger replay protection.

## Protocol

`initiateTransfer(recipient, deadline)` is payable; the destination is the immutable
other chain. The nonce and transfer ID bind both chain IDs, the bridge address, sender,
recipient, amount, nonce, and deadline. Incoming safe-protocol callbacks authenticate the local
messenger, its cross-domain sender, and its source chain, including before any no-op.

```mermaid
sequenceDiagram
    participant U as Sender
    participant A as SafeETHBridge A
    participant B as SafeETHBridge B
    participant L as ETHLiquidity B
    participant R as Recipient
    U->>A: initiateTransfer + ETH
    Note over A: PREPARED; ETH escrowed
    A-->>B: PREPARE
    B->>L: Check unreserved mint capacity (balance)
    Note over B: PREPARED; reserve capacity, mint nothing
    B-->>A: PREPARED_ACK
    Note over A: COMMITTED; burn/lock escrow into ETHLiquidity A
    A-->>B: COMMIT
    B->>L: mint(amount)
    L->>B: Native ETH
    B->>R: Force-send native ETH
    Note over B: COMMITTED; reservation consumed
```

| Side | Transition | Required condition | ETH effect |
|---|---|---|---|
| Source | NONE → PREPARED | Valid payable initiation, future deadline | Sender ETH escrowed in bridge |
| Source | PREPARED → COMMITTED | Authenticated ACK, source time < deadline | Escrow locked using `ETHLiquidity.burn` |
| Source | PREPARED → ABORTED | Sender request, source time ≥ deadline | Original escrow refunded |
| Destination | NONE → PREPARED | Authenticated PREPARE, matching ID, sufficient capacity | Reserve capacity; no mint |
| Destination | PREPARED → COMMITTED | Authenticated COMMIT | `ETHLiquidity.mint`, pay recipient once |
| Destination | NONE/PREPARED → ABORTED | Authenticated ABORT, matching ID | Tombstone; release any reservation |

## Why PREPARED guarantees payment

Let `L` be `ETHLiquidity.balance`, and `R` the sum of destination PREPARED amounts.
PREPARE requires `L >= R + amount` and increases `R`. COMMIT reduces both `L` and `R`
by its own amount. ABORT reduces only `R`. Source commits, ordinary source sends, and
permissionless funding only increase `L`. An ordinary destination relay also requires
`L >= R + amount`, then reduces only `L`. Therefore **`L >= R` is preserved**, including concurrent transfers.

This proof relies on replacing the sole authorized minting implementation: every mint
either consumes its own reservation or preserves all reservations. Source escrow lives
in the bridge, not `ETHLiquidity`, so refunding it cannot consume destination backing.
Mint authority and code must remain unchanged while reservations exist. In particular,
merely wrapping the old bridge and checking its balance would NOT prove this property.

`mint` returns ETH using the existing `SafeSend` helper; the new bridge likewise uses
`SafeSend` for destination payment and source refunds. Recipient code is not invoked,
so a reverting recipient cannot permanently reject payment. Execution still needs gas.
“Mint/burn” here means the existing ETHLiquidity unlock/lock operations, not a new EVM
supply opcode. No provisional ETH is minted at PREPARE and no extra supply is created.

## Atomicity, ordering, and recovery

The source has exactly one terminal decision: COMMITTED XOR ABORTED. It is the only
place an ACK/timeout race is serialized. Source COMMITTED means no refund and a backed
destination payment; source ABORTED means an immediate refund and no valid COMMIT can
be emitted. There is a period of zero user-spendable value while committed ETH is in
transit, but there is no valid refund-plus-payment execution. This is asynchronous
atomic settlement, **not synchronous cross-chain calls or atomic arbitrary DeFi calls**.

Duplicate safe-protocol callbacks have no further effect. An authenticated late ACK after abort or
at/after expiry is consumed without committing. ABORT before PREPARE stores a tombstone;
a later PREPARE cannot reserve or acknowledge it. COMMIT for an unknown destination
record reverts: a valid COMMIT causally requires that destination's earlier PREPARE/ACK.
ABORT after destination COMMITTED is inconsistent and reverts. Payload alterations fail
ID validation; source/destination records are separate. Hash collision resistance is assumed.

No destination reservation expires autonomously. Doing that could destroy backing after
the source commits but before COMMIT is delivered. The source sender can abort from the
deadline onward, even if destination PREPARE succeeded but its ACK has not arrived. An
expired ACK leaves the source refundable; it does not itself perform the refund.

Relaying is permissionless through the normal messenger. Failed relay transactions
revert and can be retried, including the ETH mint/payment and reservation update. For
messages outside Interop's execution window, anyone can use the messenger's existing
`resendMessage` with the original nonce/sender/target/payload. Its fresh log preserves
the same replay-protected message hash. This also recovers a delayed ACK or ABORT;
there is no reliance on duplicate PREPARE producing a new ACK. Source and destination
must retain their state and emitted message data must remain available.

Safety assumes the messenger/inbox/Interop derivation authenticate canonical histories
and handle reorg dependencies correctly. It does not assume FIFO delivery or honest
relayers. A forged ACK or COMMIT becomes possible if those authentication assumptions
fail. Chain/proxy administrators must not change the bridge or ETHLiquidity underneath
active transfers; matching addresses alone do not prove identical code on another L2.

**Safety takes precedence over unconditional liveness.** If a chain halts forever,
post-commit ETH can wait forever. Refunding after commit would allow the destination to
later pay as well. Before commit, refund requires the sender to submit an abort on a
functioning source chain. After commit, completion requires the destination to resume,
sufficient relay gas, and eventual delivery (potentially source availability to resend).
No asynchronous protocol can promise both unconditional refunds and no double payment
when another chain can disappear forever.

Safety needs no upper bound on message delay. A completion-time bound additionally needs
bounded delivery/inclusion, available chains, adequate gas, and a deadline that allows
destination PREPARE and the subsequent source ACK before expiry. Under those assumptions, the
remaining COMMIT leg completes within the delivery bound. No synchronized clocks or
destination-side timeout decide whether an already committed transfer is refundable.

## Validation and remaining work

Foundry tests cover escrow, reservation solvency, exact lock/mint accounting, duplicate
messages, forged context, transfer tampering, abort tombstones, deadline boundaries,
reverting recipients, message-send/mint rollback, and fuzzed concurrent decisions/reservations.
Mixed-mode tests prove ordinary relays cannot consume reservations, can spend excess
capacity, and can retry after an abort or funding; ordinary sends preserve safe escrow.
The two-L2 supernode acceptance tests install the experimental implementation via the
real proxy admin, relay all three actual messenger logs, and check balances/states after
every stage. They also relay a late ACK after refund and then the destination ABORT,
and complete an ordinary transfer while a safe reservation is pending.

The existing bridge immediately locks source ETH and emits one message that later mints
on the destination. That mode remains available. The optional safe path adds reversible escrow, destination capacity
reservation, and an explicit source decision before that irreversible lock.

Before production: independently audit/formally verify the protocol; design a migration
for legacy messages and existing storage; review upgrade governance and reservation
preservation; add stateful multi-chain/reorg/fault-proof tests and message-expiry recovery
tests; evaluate reservation griefing, fees, gas costs, permanent tombstone growth, and
multi-chain configuration. Constructor immutables deliberately pin one chain pair in
this minimal PoC; there is no generic initialization or migration framework.
