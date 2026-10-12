use crate::{EngineClientResult, engine::FinalizeBlockId};
use async_trait::async_trait;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use std::fmt::Debug;

/// Client to use to interact with the engine.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait DerivationEngineClient: Debug + Send + Sync {
    /// Resets the engine's forkchoice.
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()>;

    /// Sends a request to finalize the L2 block identified by the provided [`FinalizeBlockId`].
    /// Note: This does not wait for the engine to process it.
    async fn send_finalized_l2_block(&self, block_id: FinalizeBlockId) -> EngineClientResult<()>;

    /// Sends attributes derived from L1 for the engine to consolidate the safe head with.
    /// Note: This does not wait for the engine to process them.
    async fn send_derived_attributes(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<()>;

    /// Sends a safe head supplied by the derivation delegate, for the engine to follow.
    /// Note: This does not wait for the engine to process it.
    async fn send_delegated_safe_head(&self, safe_head: L2BlockInfo) -> EngineClientResult<()>;
}
