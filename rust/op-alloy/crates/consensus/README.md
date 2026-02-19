## `op-alloy-consensus`

Optimism consensus interface.

This crate contains constants, types, and functions for implementing Optimism EL consensus and communication. This
includes an extended `OpTxEnvelope` type with [deposit transactions][deposit], and receipts containing OP Stack
specific fields (`deposit_nonce` + `deposit_receipt_version`).

In general a type belongs in this crate if it exists in the `alloy-consensus` crate, but was modified from the base Ethereum protocol in the OP Stack.
For consensus types that are not modified by the OP Stack, the `alloy-consensus` types should be used instead.

[deposit]: https://specs.optimism.io/protocol/deposits.html

### Provenance

Much of this code was ported from [reth-primitives] as part of ongoing alloy migrations.

[reth-primitives]: https://github.com/paradigmxyz/reth/tree/main/crates/primitives
