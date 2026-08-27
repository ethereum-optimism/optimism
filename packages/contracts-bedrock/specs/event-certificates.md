# L1-Certified L2 Events

## Status

Draft. The contracts in this change define and test the protocol boundary. The execution-client, fault-proof, genesis,
upgrade, and deployment changes listed under [Required follow-up work](#required-follow-up-work) are not implemented.

## Motivation

Interop messages currently depend on destination-chain inclusion before their initiating event expires. A sequencer can
therefore censor message execution until the event is no longer eligible. Users need an L1-mediated slow path that can
be force-included, takes no dependency on a sequencer, and remains usable after the ordinary interop expiry window.

The slow path turns a recent local L2 event into a durable L1 certificate. The event must be exported during the
seven-day lookup window, but its certificate can be relayed and consumed later. A user can force-include both the
source export and destination execution through the relevant portals.

## Components

- `LocalLogOracle` is a consensus-critical precompile at `0x4200000000000000000000000000000000000026`. It only proves
  logs from previous blocks of the chain on which it is called and only while they are at most seven days old.
- `CrossL2Inbox` exports a locally proven event through `L2ToL1MessagePasser`, imports an L1 certificate, and validates
  cached certificates without an EIP-2930 access-list entry.
- `L1EventRegistry` authenticates finalized exports and stores durable certificates. Its immutable `ETHLockbox`
  identifies the portals in one interop cluster.
- `OptimismPortal2` supplies the existing L2-to-L1 withdrawal proof path and L1-to-L2 force-inclusion path.

## Certificate

An event is identified by the existing `Identifier` fields:

```text
(origin, blockNumber, logIndex, timestamp, chainId)
```

The certificate is:

```text
keccak256(abi.encode(identifier, payloadHash))
```

`payloadHash` is the hash of the complete encoded log payload. For an interop message it is
`keccak256(sentMessagePayload)`, matching the existing `L2ToL2CrossDomainMessenger.relayMessage` validation.

## Protocol flow

1. Before the seven-day lookup window closes, a normal transaction or forced deposit calls
   `CrossL2Inbox.exportEvent(identifier, payloadHash)` on the source L2.
2. `CrossL2Inbox` requires the source chain ID, a previous block, and an in-window timestamp, then asks
   `LocalLogOracle.containsLog` to verify the exact local log.
3. `CrossL2Inbox` creates a zero-value withdrawal targeting `L1EventRegistry.registerEvent`.
4. Anyone proves and finalizes that withdrawal using the ordinary portal mechanism.
5. `L1EventRegistry` accepts the certificate only when the calling portal is currently authorized by its immutable
   `ETHLockbox`, the portal uses that lockbox, `portal.l2Sender()` is the canonical `CrossL2Inbox`, and the chain ID
   derived from `portal.systemConfig()` matches the identifier.
6. Anyone calls `relayEvent` or `relayMessage` with an authorized destination portal. The registry deposits a fixed
   call to the destination `CrossL2Inbox`; the registry becomes the aliased L2 caller.
7. The destination inbox authenticates the alias, stores the checksum, and either leaves generic consumption to the
   application or calls `L2ToL2CrossDomainMessenger.relayMessage` in the same deposit.

Certificates and imports are idempotent. Messenger replay protection remains the authority for whether a particular
interop message has executed successfully.

## Consensus separation

Ordinary validation emits `ExecutingMessage`, which interop clients interpret as an executing-message dependency.
Certified validation instead emits `ExecutingCertifiedMessage`. Clients must not apply ordinary expiry, dependency-set,
or access-list hazard rules to that event: its safety derives from the L1 deposit and finalized certificate.

This distinction is consensus-critical. Reusing `ExecutingMessage` for the certified path would cause old messages to
be rejected by current supervisors even though the EVM call succeeded locally.

## Security properties

- Calldata cannot forge source identity. L1 derives it from the calling portal and `portal.l2Sender()`.
- Certificates cannot cross ETHLockbox clusters unless a destination cluster explicitly trusts the registry.
- Destination imports accept only the configured registry's standard L1-to-L2 address alias.
- A failed one-shot message execution reverts the import deposit, but the L1 certificate remains registered and can be
  retried.
- The mechanism is censorship-resistant under the same L1 inclusion and portal force-inclusion assumptions as an
  ordinary deposit.

## Limitations

The mechanism does not recover an event that was never exported before its seven-day local lookup window closed.
Protocols requiring unconditional eventual recovery should export eagerly, allow a keeper to export, or batch recent
event certificates. Withdrawal finality can then complete after ordinary interop expiry without losing the event.

The registry establishes that a log existed. It does not establish application-specific absence, non-execution, or
refund eligibility. An ETH bridge using certificates for recovery must still bind a unique transfer identifier and
enforce a single terminal outcome across relay and refund paths.

## Required follow-up work

1. Specify `LocalLogOracle` gas, receipt encoding, pruning behavior, and exact boundary behavior.
2. Make historical receipt access deterministic in op-geth, op-reth, Kona, and fault-proof re-execution. Reading an
   execution client's optional archival database is insufficient. The final design needs either a state-committed
   rolling receipt-root/history structure or explicit receipt and ancestry witnesses anchored to consensus state.
3. Teach interop clients to recognize `ExecutingCertifiedMessage` as L1-certified rather than as an ordinary
   executing-message dependency.
4. Deploy `L1EventRegistry`, configure it in each participating `CrossL2Inbox`, and add genesis/NUT wiring and standard
   validation checks.
5. Add end-to-end tests for forced source export, withdrawal proving/finalization, destination deposit derivation,
   one-shot execution, retries, reorgs, and the exact seven-day boundary.
6. Define batching before production use; one withdrawal and one deposit per event is deliberately the simplest base
   case, not the most economical form.
