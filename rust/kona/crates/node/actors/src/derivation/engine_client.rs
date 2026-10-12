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
pub trait DerivationEngineClient: Debug + Send + Sync {
    /// Resets the engine onto a starting point found from the execution layer's heads, and returns
    /// the engine state afterwards.
    ///
    /// Fails with [`EngineClientError::ElSyncing`] until the execution layer has validated the
    /// latest forkchoice update.
    ///
    /// [`EngineClientError::ElSyncing`]: crate::EngineClientError::ElSyncing
    async fn reset(&self) -> EngineClientResult<EngineState>;

    /// Consolidates the safe head with attributes derived from L1, and returns the engine state
    /// afterwards.
    async fn consolidate(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<Consolidation>;

    /// Sends a request to finalize the L2 block identified by the provided [`FinalizeBlockId`].
    /// Note: This does not wait for the engine to process it.
    async fn send_finalized_l2_block(&self, block_id: FinalizeBlockId) -> EngineClientResult<()>;

    /// Sends a safe head supplied by the derivation delegate, for the engine to follow.
    /// Note: This does not wait for the engine to process it.
    async fn send_delegated_safe_head(&self, safe_head: L2BlockInfo) -> EngineClientResult<()>;
}
