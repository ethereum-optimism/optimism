# `kona-providers-alloy`

Alloy-backed providers for `kona`.

The L1 and L2 derivation adapters share three composable source layers:

- `RpcSource<N>` fetches consensus data using Alloy's Ethereum or Optimism network
  types. Its Ethereum `ReceiptSource` implementation handles raw receipt decoding
  and JSON receipt fallback, with method support shared across clones.
- `Verifying<S>` authenticates headers, transactions, and Ethereum receipt
  envelopes. It recomputes header hashes and trie roots, checks reported transaction
  hashes against their EIP-2718 encodings (including deposit and `PostExec`
  transactions), and checks the block number on number-based lookups.
- `Cached<S>` caches successful results by hash. Number-based lookups always reach
  the source so they follow reorgs.

The node interprets both `trust_rpc` flags once at startup: it passes either the
RPC source directly or `Verifying::new(rpc)` to generic startup code, then wraps
each source in `Cached`. The pipeline, buffered L2 provider, and derivation adapters
retain these concrete source types. Neither fetching, verification, nor caching
receives a trust flag or requires a runtime mode wrapper.

Receipt envelopes retain transaction types and blooms through verification and
caching; the L1 derivation adapter converts them into plain receipts afterward.
Receipt-root verification failures return directly without trying another RPC
method. Method fallback remains inside `RpcSource`, and rejected verification
results are never cached. Verification failures remain retryable by derivation.

Empty raw receipt responses are checked against an existing header's transaction
root to distinguish empty blocks from missing blocks. Receipt verification obtains
its own header from the inner source; it does not reuse the outer header cache.

`AlloyChainProvider::new` and `AlloyL2ChainProvider::new` accept composed sources
instead of RPC clients and cache sizes. The `new_with_trust` constructors have been
removed; callers choose the wrapper explicitly. The HTTP convenience constructors
retain their trusting behavior.

The unused `ReceiptsConversion` error variant is removed. Receipt decoding errors
retain their typed source through `SourceError` (also exported as
`AlloyChainProviderError`).
