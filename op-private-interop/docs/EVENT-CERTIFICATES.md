# Private event certificates

This branch imports `codex/l1-certified-event-relay` (`ef837d687c`), fixes its settlement boundary,
and adds a projection-only export path with a pluggable verifier. The initial verifier accepts a
trusted signer's ECDSA attestation. It is not a ZK execution proof or a TEE attestation checker.

## Deposit execution and settlement

`ProjectionEventExporter` reserves `0x4200000000000000000000000000000000000030`. Stock genesis
already places an uninitialized proxy there. Private genesis leaves it inactive; Go and Rust
projection genesis activate its fixed implementation. Ordinary contract creation cannot replace
an occupied proxy. This is a genesis protocol change, not a live upgrade for existing projections.

A user submits an ordinary, zero-value L1 deposit targeting that address with an event identifier,
payload hash and opaque proof. There is no new transaction type. On the private chain, the deposit
is still derived, but its call reverts at the inactive proxy. On the projection, the exporter calls
`CrossL2Inbox.exportProvenEvent`. The inbox restricts that method to the reserved exporter and calls
the configured `IEventProofVerifier`. It records each accepted identifier once, independently of
verifier replacement. Failure of verification or export rolls back all consumption.

The inbox uses the standard L2 cross-domain messenger to send `registerEvent` to `L1EventRegistry`.
This creates a real zero-value entry in the projection's `L2ToL1MessagePasser.sentMessages` mapping.
Projection deposit receipt logs remain suppressed. Clients must reconstruct the withdrawal from
projection execution/nonces, not from a private receipt. The superroot commits the projection output
root under the private chain ID; ordinary portal proofs and finalization settle that outbox.

The L1 registry authenticates the source's canonical L1 messenger, its cross-domain sender, chain ID,
and current shared-lockbox membership. Messenger delivery remains retryable if registration fails
after portal finalization. Registry delivery to a public destination deposits an authenticated
certificate; execution emits `ExecutingCertifiedMessage`, separate from ordinary interop hazards.
The original event identifier stays in the certificate; the export does not backdate an emitted log.

## Proof policy and activation

Governance must configure the inbox's registry and verifier. Unconfigured proof exports fail closed.
`AttestedEventVerifier` binds signatures to chain, verifier, inbox, full projected event identifier,
and payload hash. The signer must establish canonicality, execution validity and any required
application accounting. A signature proves only that this signer attested those facts. Holding a
valid attestation allows export with the private operator offline; obtaining one still requires the
signer to be available. The transport does not recover missing private state or establish a debit.
A future TEE or ZK verifier can implement the same interface without changing deposit encoding.

The original receipt-oracle export API remains separate and unimplemented at the consensus layer;
this proof path does not depend on it or on its seven-day lookup window. Private raw log positions
must be translated to canonical projected positions before signing.

Only the private-source to public-destination event-certificate route is supported here. This does
not enable ordinary asset withdrawals or execution of incoming certified messages by the private
projection's replay messenger. Migration must preserve or drain old registry delivery routes:
once-only exports cannot be reissued merely because the registry or verifier was replaced.

## Validation

Contract tests cover real messenger/outbox insertion, source and destination authentication,
L1 messenger failure/retry, signature binding, duplicate consumption, downstream failure rollback,
and destination application retry. The devstack outage test exercises a real private event,
force-included proof export, super-game withdrawal settlement, destination execution, and private
recovery with a reverted export deposit. See test results accompanying this change for run status;
contract tests alone are not end-to-end validation. Reorg coverage and a production proof provider
remain future work.

The devstack fixture starts with ordinary portals whose lockbox feature is off. It initializes a
real shared lockbox using devnet governance, preserves packed portal storage, and restores portal
implementations atomically through the standard StorageSetter upgrade utility. This setup tests
transport under configured cluster authorization; it does not validate a production cluster migration.

To regenerate the exporter bytecode, build contracts with the default Foundry profile, then write
`forge inspect ProjectionEventExporter deployedBytecode` from `packages/contracts-bedrock` into
`op-private-interop/genesis/bytecode/ProjectionEventExporter.hex`. Regenerate both Go and Rust genesis
golden vectors together and run both suites; the embedded bytecode is shared protocol data.
