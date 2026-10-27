# `kona-providers-alloy`

Alloy-backed providers for `kona`.

With `trust_rpc = false`, the L1 and L2 providers recompute header hashes and
transaction roots before returning or caching blocks. They also check reported
transaction hashes against their EIP-2718 encodings, including deposit and
`PostExec` transactions. Number lookups verify the returned block number.

The L1 provider verifies both raw and JSON receipts against the requested header's
receipt root before discarding receipt types and blooms. Invalid raw receipts can
fall back to JSON receipts, which undergo the same verification. Verification
failures remain retryable and rejected responses are not cached.
