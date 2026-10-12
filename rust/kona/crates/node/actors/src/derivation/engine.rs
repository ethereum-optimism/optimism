use crate::{
    EngineClientResult,
    engine::{Consolidation, EngineState, FinalizeBlockId},
};
use async_trait::async_trait;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use std::fmt::Debug;

/// The engine, as the derivation actors use it.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait Engine: Debug + Send + Sync {
    /// Resets the engine onto a starting point found from the execution layer's heads, and returns
    /// the engine state afterwards.
    ///
    /// Fails with [`EngineClientError::ElSyncing`] while the execution layer may be syncing.
    ///
    /// [`EngineClientError::ElSyncing`]: crate::EngineClientError::ElSyncing
    async fn reset(&self) -> EngineClientResult<EngineState>;

    /// Consolidates the safe head with attributes derived from L1, and returns the engine state
    /// afterwards.
    async fn consolidate(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<Consolidation>;

    /// Asks the engine to finalize the L2 block identified by the provided [`FinalizeBlockId`],
    /// without waiting for the engine to do so.
    async fn finalize(&self, block_id: FinalizeBlockId) -> EngineClientResult<()>;

    /// Asks the engine to follow a safe head supplied by the derivation delegate, without waiting
    /// for the engine to do so.
    async fn follow_safe_head(&self, safe_head: L2BlockInfo) -> EngineClientResult<()>;
}
