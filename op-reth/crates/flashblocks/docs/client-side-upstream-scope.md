# Client-Side Flashblocks Upstream Scope

**Authors:** Upstreaming from node-reth
**Status:** Planning
**Date:** January 2026

## Executive Summary

This document scopes the work required to upstream the client-side flashblocks functionality from node-reth to reth. This enables any OP Stack follower node to receive flashblocks and serve flashblock-aware RPC endpoints.

## Current State

### reth (upstream) - Builder Side Only
```
op-reth/crates/flashblocks/
├── service.rs          # FlashBlockService (builds blocks)
├── worker.rs           # Block execution
├── cache.rs            # Sequence management
├── pending_state.rs    # Speculative building state
├── validation.rs       # Sequence validation, reorg detection
├── ws/                 # WebSocket stream for receiving
└── ...
```

### node-reth - Full Client Side
```
crates/client/flashblocks/
├── state.rs            # FlashblocksState (coordinator)
├── processor.rs        # StateProcessor (executes flashblocks)
├── pending_blocks.rs   # PendingBlocks (RPC-queryable state)
├── state_builder.rs    # PendingStateBuilder (tx execution)
├── subscription.rs     # WebSocket subscriber with reconnection
├── validation.rs       # Same as reth (already upstreamed)
├── traits.rs           # FlashblocksAPI, PendingBlocksAPI
├── extension.rs        # Node builder integration
├── rpc/
│   ├── eth.rs          # EthApiExt (flashblock-aware eth_*)
│   └── pubsub.rs       # EthPubSub (flashblock subscriptions)
└── ...
```

## Proposed Architecture

### Module Structure
```
op-reth/crates/flashblocks/
├── src/
│   ├── lib.rs
│   │
│   │   # Existing (builder-side)
│   ├── service.rs
│   ├── worker.rs
│   ├── cache.rs
│   ├── pending_state.rs
│   ├── validation.rs
│   ├── ws/
│   │
│   │   # New (client-side)
│   ├── client/
│   │   ├── mod.rs
│   │   ├── state.rs            # FlashblocksClientState
│   │   ├── processor.rs        # ClientStateProcessor
│   │   ├── pending_blocks.rs   # ClientPendingBlocks
│   │   └── state_builder.rs    # ClientStateBuilder
│   │
│   └── rpc/                    # New (optional feature)
│       ├── mod.rs
│       ├── eth_ext.rs          # EthApiFlashblocksExt
│       ├── pubsub.rs           # FlashblocksPubSub
│       └── types.rs            # Subscription types
```

### Feature Flags
```toml
[features]
default = []
client = []           # Client-side state management
rpc = ["client"]      # RPC extensions (requires client)
```

## Implementation Phases

### Phase 1: Core Client State Management

**Goal:** Enable follower nodes to receive and track flashblock state.

**Files to create:**

| File | Source | Changes Required |
|------|--------|------------------|
| `client/mod.rs` | New | Module exports |
| `client/state.rs` | `node-reth/state.rs` | Replace `base_flashtypes::Flashblock` with `FlashBlock` |
| `client/processor.rs` | `node-reth/processor.rs` | Same type replacements, generalize over chain spec |
| `client/pending_blocks.rs` | `node-reth/pending_blocks.rs` | Use reth RPC types |
| `client/state_builder.rs` | `node-reth/state_builder.rs` | Already uses reth types |

**New traits:**
```rust
/// Core API for accessing flashblock state.
pub trait FlashblocksClientAPI {
    /// Retrieves the pending blocks.
    fn pending_blocks(&self) -> Option<Arc<ClientPendingBlocks>>;

    /// Subscribes to flashblock updates.
    fn subscribe(&self) -> broadcast::Receiver<Arc<ClientPendingBlocks>>;
}

/// API for querying pending block data.
pub trait PendingBlocksAPI {
    fn get_block(&self, full: bool) -> Option<RpcBlock<Optimism>>;
    fn get_transaction_receipt(&self, hash: TxHash) -> Option<RpcReceipt<Optimism>>;
    fn get_balance(&self, address: Address) -> Option<U256>;
    fn get_transaction_count(&self, address: Address) -> U256;
    fn get_transaction_by_hash(&self, hash: TxHash) -> Option<RpcTransaction<Optimism>>;
    fn get_pending_logs(&self, filter: &Filter) -> Vec<Log>;
    fn get_state_overrides(&self) -> Option<StateOverride>;
}
```

**Dependencies:**
- `reth_rpc_eth_api` - RPC types
- `reth_optimism_primitives` - OP primitives
- `op_alloy_rpc_types` - OP RPC types
- Existing flashblocks crate dependencies

**Estimated scope:** ~1000 LOC

---

### Phase 2: RPC Extensions

**Goal:** Enable flashblock-aware RPC methods.

**Files to create:**

| File | Source | Changes Required |
|------|--------|------------------|
| `rpc/mod.rs` | New | Module exports |
| `rpc/eth_ext.rs` | `node-reth/rpc/eth.rs` | Adapt to reth RPC infrastructure |
| `rpc/pubsub.rs` | `node-reth/rpc/pubsub.rs` | Adapt to reth pubsub |
| `rpc/types.rs` | `node-reth/rpc/types.rs` | Subscription kind enums |

**RPC methods to support:**

| Method | Behavior |
|--------|----------|
| `eth_getBlockByNumber("pending")` | Returns current flashblock |
| `eth_getTransactionReceipt` | Checks flashblocks first |
| `eth_getBalance(addr, "pending")` | Returns flashblock balance |
| `eth_getTransactionCount(addr, "pending")` | Includes flashblock txs |
| `eth_getTransactionByHash` | Checks flashblocks first |
| `eth_call(tx, "pending")` | Executes against flashblock state |
| `eth_estimateGas(tx, "pending")` | Estimates with flashblock state |
| `eth_getLogs` | Includes pending logs |
| **`eth_sendRawTransactionSync`** | Wait for flashblock inclusion |

**New subscriptions:**

| Subscription | Description |
|--------------|-------------|
| `newFlashblocks` | Stream of new flashblocks |
| `pendingLogs` | Stream of logs from flashblocks |
| `newFlashblockTransactions` | Stream of pending transactions |

**Dependencies:**
- `jsonrpsee` - RPC server
- `reth_rpc` - Base RPC implementation
- `reth_rpc_eth_api` - Eth API traits

**Estimated scope:** ~800 LOC

---

### Phase 3: Node Builder Integration

**Goal:** Easy integration into OP Stack nodes.

**Files to create:**

| File | Description |
|------|-------------|
| `client/extension.rs` | Node builder extension trait |
| `client/config.rs` | Configuration types |

**Integration approach:**

```rust
/// Configuration for flashblocks client.
#[derive(Debug, Clone)]
pub struct FlashblocksClientConfig {
    /// WebSocket URL for flashblocks stream.
    pub ws_url: String,
    /// Maximum pending block depth.
    pub max_depth: u64,
    /// Enable RPC extensions.
    pub enable_rpc: bool,
}

/// Extension for adding flashblocks client to a node.
pub trait FlashblocksClientExt {
    /// Adds flashblocks client support to the node.
    fn with_flashblocks_client(self, config: FlashblocksClientConfig) -> Self;
}
```

**Integration with ExEx:**
```rust
// The client uses an ExEx to receive canonical block notifications
// for reconciliation with flashblock state.
impl FlashblocksClientExtension {
    fn install_exex(&self, ctx: &BuilderContext) {
        ctx.install_exex("flashblocks-reconciler", |ctx| {
            // Forward canonical blocks to state processor
        });
    }
}
```

**Estimated scope:** ~300 LOC

---

## Type Mappings

| node-reth | reth (upstream) |
|-----------|-----------------|
| `base_flashtypes::Flashblock` | `FlashBlock` (alias for `OpFlashblockPayload`) |
| `base_client_node::*` | reth node builder APIs |
| `PendingBlocks` | `ClientPendingBlocks` (renamed to avoid confusion) |
| `FlashblocksState` | `FlashblocksClientState` |
| `StateProcessor` | `ClientStateProcessor` |

## Key Decisions

### 1. Naming Convention
- Prefix client-side types with `Client` to distinguish from builder-side
- Use `client/` submodule for clear separation

### 2. Feature Organization
- `client` feature: Core state management (no RPC dependencies)
- `rpc` feature: RPC extensions (depends on `client`)
- Default: Neither enabled (builder-only, current behavior)

### 3. Generics
- Make types generic over chain spec where possible
- Use associated types for network-specific RPC types

### 4. Shared Code
- `validation.rs` - Already shared (upstreamed in Phase 1-2 of speculative building)
- `ws/` - Can be shared for WebSocket connection

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Type incompatibility with `base_flashtypes` | Use `op_alloy_rpc_types_engine::OpFlashblockPayload` (already aliased as `FlashBlock`) |
| RPC infrastructure differences | Adapt to reth's RPC patterns, may need some refactoring |
| Breaking changes to builder-side | Feature flags isolate client code |
| Performance regression | Benchmark critical paths, optimize state access |

## Testing Strategy

### Unit Tests
- State processor reconciliation logic
- Pending blocks query methods
- RPC method behavior with/without flashblock state

### Integration Tests
- Full flow: WS → State → RPC response
- Canonical block reconciliation
- Reorg handling

### Test Harness
- Extend existing `FlashBlockServiceTestHarness` for client testing
- Mock flashblock streams
- Simulated canonical chain updates

## Estimated Total Scope

| Phase | LOC | Complexity |
|-------|-----|------------|
| Phase 1: Core Client State | ~1000 | Medium |
| Phase 2: RPC Extensions | ~800 | Medium-High |
| Phase 3: Node Integration | ~300 | Low |
| Tests | ~500 | Medium |
| **Total** | **~2600** | |

## Open Questions

1. **Should `eth_sendRawTransactionSync` be included?**
   - Very useful for UX (wait for flashblock inclusion)
   - Adds complexity (timeout handling, dual listening)

2. **How to handle Base-specific subscription types?**
   - Option A: Generic subscription system
   - Option B: OP-specific types in `op_alloy`

3. **Integration with existing reth RPC modules?**
   - Replace vs extend existing eth_* implementations
   - node-reth uses `replace_configured` pattern

## Next Steps

1. [ ] Review and approve scope
2. [ ] Phase 1 implementation
3. [ ] Phase 1 PR and review
4. [ ] Phase 2 implementation
5. [ ] Phase 2 PR and review
6. [ ] Phase 3 implementation
7. [ ] Final integration testing
8. [ ] Documentation updates

## References

- node-reth flashblocks: `node-reth/crates/client/flashblocks/`
- reth flashblocks: `reth/op-reth/crates/flashblocks/`
- Speculative building design doc: `docs/speculative-building.md`
