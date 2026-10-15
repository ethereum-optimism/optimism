use crate::EngineClientResult;
use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;

/// Trait to be used by the Sequencer to interact with the engine, abstracting communication
/// mechanism.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait SequencerEngineClient: Debug + Send + Sync {
    /// Starts building a block with the provided attributes.
    ///
    /// Returns a `PayloadId` that can be used to seal the block later.
    async fn start_build_block(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId>;

    /// Seals a previously started block, without importing it.
    ///
    /// Takes a `PayloadId` from a previous `start_build_block` call and returns the sealed
    /// execution payload envelope.
    async fn seal_block(
        &self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope>;

    /// Imports a sealed block and makes it the unsafe head.
    async fn canonicalize_block(
        &self,
        payload: OpExecutionPayloadEnvelope,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<()>;

    /// Returns the current unsafe head [`L2BlockInfo`].
    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo>;
}
