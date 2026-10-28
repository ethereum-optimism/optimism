# `kona-providers-alloy`

Alloy-backed providers for `kona`.

L1 and L2 derivation adapters share `BlockSource`, with `ReceiptSource` adding
Ethereum receipts. `RpcSource<N>` fetches consensus data for either network and
handles Ethereum raw receipt decoding and JSON receipt fallback. Method support
is shared across clones. Its `new_with_trust` constructor preserves the existing
optional header-hash checks on hash-based lookups; it does not add transaction or
receipt-root verification.

`Cached<S>` caches successful results by hash. Number-based lookups always reach
the source so they follow reorgs. Receipt envelopes retain their types and blooms
through fetching and caching; the L1 adapter converts them into plain receipts.
The pipeline and buffered L2 provider retain concrete source types.

`AlloyChainProvider::new` and `AlloyL2ChainProvider::new` now accept composed sources
instead of RPC clients and cache sizes. For example, use
`AlloyChainProvider::new(Cached::new(RpcSource::new_with_trust(rpc, trust_rpc), size))`.
The adapters' public RPC-client and trust fields and `new_with_trust` constructors
are removed; configure `RpcSource` instead. HTTP convenience constructors retain
their trusting behavior.

The unused `ReceiptsConversion` error variant is removed. Receipt decoding errors
retain their typed source through `SourceError` (also exported as
`AlloyChainProviderError`).
