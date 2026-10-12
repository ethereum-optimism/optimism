use crate::EngineClientResult;
use async_trait::async_trait;
use kona_engine::{ConsolidateInput, FinalizeBlockId};
use std::fmt::Debug;

/// Client to use to interact with the engine.
#[cfg_attr(test, mockall::automock(type SafeL2Signal = OpAttributesWithParent;))]
#[async_trait]
pub trait DerivationEngineClient: Debug + Send + Sync {
    /// Resets the engine's forkchoice.
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()>;

    /// Sends a request to finalize the L2 block identified by the provided [`FinalizeBlockId`].
    /// Note: This does not wait for the engine to process it.
    async fn send_finalized_l2_block(&self, block_id: FinalizeBlockId) -> EngineClientResult<()>;

    /// Sends a consolidation signal to the engine.
    ///
    /// This is the unified entry point for all consolidation-related inputs,
    /// including derived attributes and safe L2 block information, as represented
    /// by [`ConsolidateInput`].
    ///
    /// Note: This does not wait for the engine to process it.
    async fn send_safe_l2_signal(&self, signal: ConsolidateInput) -> EngineClientResult<()>;
}
