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
- `CrossL2Inbox` exports a locally proven event through the standard L2 cross-domain messenger and message passer,
  imports an L1 certificate, and validates
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
3. `CrossL2Inbox` calls the standard `L2CrossDomainMessenger.sendMessage` with zero value, targeting
   `L1EventRegistry.registerEvent`. The messenger creates a normal message-passer withdrawal to its L1 counterpart.
4. Anyone proves and finalizes that withdrawal using the ordinary portal mechanism. The L1 messenger authenticates
   the portal delivery and calls the registry with `xDomainMessageSender() == CrossL2Inbox`.
5. `L1EventRegistry` accepts the certificate only when the messenger's portal is currently authorized by its immutable
   `ETHLockbox`, the portal uses that lockbox, and the portal's `SystemConfig.l1CrossDomainMessenger()` identifies the
   caller. It then checks the cross-domain sender and the chain ID derived from the portal's `SystemConfig`.
6. Anyone calls `relayEvent` or `relayMessage` with an authorized destination portal. The registry deposits a fixed
   call to the destination `CrossL2Inbox`; the registry becomes the aliased L2 caller.
7. The destination inbox authenticates the alias, stores the checksum, and either leaves generic consumption to the
   application or calls `L2ToL2CrossDomainMessenger.relayMessage` in the same deposit.

Certificates and imports are idempotent. Messenger replay protection remains the authority for whether a particular
interop message has executed successfully.

If L1 registration fails, the standard L1 messenger records a failed message that anyone can retry. The registry is
deliberately not the direct portal withdrawal target: a direct target failure would consume the finalized withdrawal
without providing a retry. Retrying the messenger delivery does not repeat the L2 log lookup, so its seven-day window
may expire in the meantime. Permanent removal of a source portal or replacement of its canonical messenger still
requires draining pending exports or an explicit migration policy; retry does not override revoked authorization.

## Consensus separation

Ordinary validation emits `ExecutingMessage`, which interop clients interpret as an executing-message dependency.
Certified validation instead emits `ExecutingCertifiedMessage`. Clients must not apply ordinary expiry, dependency-set,
or access-list hazard rules to that event: its safety derives from the L1 deposit and finalized certificate.

This distinction is consensus-critical. Reusing `ExecutingMessage` for the certified path would cause old messages to
be rejected by current supervisors even though the EVM call succeeded locally.

## Security properties

- Calldata cannot forge source identity. The registry binds the calling messenger to the authorized portal's
  `SystemConfig` before trusting its cross-domain sender. A fake messenger that merely reports a real portal is rejected.
- Certificates cannot cross ETHLockbox clusters unless a destination cluster explicitly trusts the registry.
- Destination imports accept only the configured registry's standard L1-to-L2 address alias.
- A failed one-shot message execution reverts the import deposit, but the L1 certificate remains registered and can be
  retried.
- The mechanism is censorship-resistant under the same L1 inclusion and portal force-inclusion assumptions as an
  ordinary deposit.

## Private projection proof path

An ordinary L1 deposit may target the reserved `ProjectionEventExporter` at
`0x4200000000000000000000000000000000000030`. Its proxy is active only in projection genesis and
uninitialized on the private chain. The latter still derives the deposit but reverts its call.
The exporter is the only caller accepted by `CrossL2Inbox.exportProvenEvent`. The inbox delegates
opaque proof verification to a governance-configured `IEventProofVerifier`, consumes the full event
identifier once, and sends the same standard messenger export described above. All state changes
roll back on rejection or failed export; consumption survives verifier replacement.

The initial `AttestedEventVerifier` checks a trusted ECDSA signer, binding the chain ID, verifier,
inbox, full identifier and payload hash. Its signer is responsible for canonical private history,
projected positions and execution validity. This is not an independently verified execution proof.
A TEE or ZK provider can implement the interface later. No local receipt oracle or seven-day lookup
limit applies to this proof path, but a user must already have a valid proof when its provider goes
offline.

The projection really executes the zero-value message-passer write. Receipt logs for the deposit
remain suppressed; a private receipt cannot supply the projection withdrawal hash. The superroot
still commits the projection output root, so only that outbox can settle through the source portal.
See [private integration](../../../op-private-interop/docs/EVENT-CERTIFICATES.md).

## Limitations and remaining work

The local-oracle path still needs deterministic consensus implementation across clients and fault
proofs. Its seven-day window applies to source lookup, not to later messenger retry. The proof path
is usable without this oracle; registry and verifier configuration remain governance operations.

A certificate establishes an event only under its configured proof policy. It does not establish
application-specific absence, non-execution or refund eligibility. Asset recovery requires separate
accounting with a single terminal outcome. Ordinary private asset withdrawals are outside this work.

Private raw and projected log indices can differ. Signers and future proof systems must authenticate
the projected identifier, and must not treat suppressed deposit logs as published initiating events.
Incoming certified execution on a private projection is unsupported: its replay messenger rejects
`relayMessage`.

Migrations must preserve or drain the old registry and canonical messenger routes. Export consumption
prevents reexporting an identifier to a replacement registry. Retries do not override revoked cluster
authorization. Reorg testing, production proof-provider integration, deployment/upgrade automation,
and batching remain required before production deployment.
