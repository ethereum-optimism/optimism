# Sequencer batches and ordinary deposits

The public projection advances under normal OP derivation when the private operator
is offline. Its fallback blocks contain the required system and ordinary deposit
transactions, but no synthetic private message replay transactions or new private
range claims. Private commitments advance when the sequencer publishes another
accepted range. This does not provide independent private execution during an outage.

There is one ordinary portal deposit path. The special proof-carrying projection
exporter, verifier hooks and reserved exporter address are removed. The general L1
event oracle remains a separate contract feature; its ordinary export route uses
the standard cross-domain messengers so failed L1 registration can be retried.

The batcher skips projection positions that are already canonical and publishes a
claim at the beginning of each new range, followed by its synthetic message replay
transactions. A range may cover later projection positions whose private blocks have
already executed. It cannot commit unknown future L1 deposits. Claims publish private
terminal block and parent hashes, L1/configuration bindings and a private input hash.
There are no published write sets, key/value commitments or private-writes RPC.

The private input hash covers stock span-batch frames, which exclude deposit payloads.
Canonical L1 headers, receipts, configuration and block origins provide those deposits
through ordinary derivation. An authentic private terminal block hash commits the
executed deposits through transaction roots and ancestor headers. Today the registry
accepts an operator attestation: it does not verify private execution or deposit
completeness, and it rejects nonempty proof bytes. A future execution verifier must
bind the prior private checkpoint, the canonical deposit history, resulting private
state and published message outputs; the input hash alone is insufficient.

[Recovery](RECOVERY.md) remains in place for private reconciliation after fallback or
cross-chain invalidation. Removing write publication and the special exporter does
not remove that mechanism. Existing interop message validation also retains its normal
transaction access-list checks, which are separate from the removed private write sets.

## Development migration

Start a fresh deployment with matching contracts, Go services and op-reth. This rollback
restores claim wire version 1 and removes the exporter predeploy, changing projection
genesis. Existing experimental version-3 deployments cannot adopt this as a live upgrade.
