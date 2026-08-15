//! Semantic engine operation driver.

use crate::engine::{EngineResult, SafeChainUpdate};
use async_trait::async_trait;
use kona_engine::EngineSyncState;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

/// Applies semantic rollup operations to an execution engine.
///
/// Implementations own raw Engine API translation and mutable forkchoice state. The surrounding
/// engine service guarantees that these methods are never invoked concurrently.
#[async_trait]
pub trait EngineDriver: core::fmt::Debug + Send + 'static {
    /// Builds and retrieves an unsafe payload without importing or canonicalizing it.
    async fn build_unsafe(
        &mut self,
        attributes: OpAttributesWithParent,
    ) -> EngineResult<OpExecutionPayloadEnvelope>;

    /// Imports a complete payload and advances the unsafe head.
    async fn import_unsafe(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
    ) -> EngineResult<L2BlockInfo>;

    /// Reconciles the safe chain with the current unsafe chain.
    async fn update_safe(&mut self, update: SafeChainUpdate) -> EngineResult<L2BlockInfo>;

    /// Advances the finalized head to an existing safe block.
    async fn update_finalized(&mut self, block: L2BlockInfo) -> EngineResult<()>;

    /// Returns the current engine synchronization state.
    fn state(&self) -> EngineSyncState;
}
