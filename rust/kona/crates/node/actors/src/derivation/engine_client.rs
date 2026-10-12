use crate::{EngineClientError, EngineClientResult, engine};
use async_trait::async_trait;
use derive_more::Constructor;
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

/// Client to use to send requests to the engine actor.
#[derive(Constructor, Debug)]
pub struct QueuedDerivationEngineClient {
    /// The engine actor's handle.
    pub engine: engine::Handle,
}

#[async_trait]
impl DerivationEngineClient for QueuedDerivationEngineClient {
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
        info!(target: "derivation", "Sending reset request to engine.");
        self.engine
            .reset()
            .await
            .inspect(|_| info!(target: "derivation", "Engine reset successfully."))
            .inspect_err(|err| {
                if matches!(err, EngineClientError::ResponseError(_)) {
                    error!(target: "derivation_engine_client", "Failed to receive built payload");
                }
            })
    }

    async fn send_finalized_l2_block(&self, block_id: FinalizeBlockId) -> EngineClientResult<()> {
        trace!(target: "derivation", ?block_id, "Sending finalized L2 block id to engine.");
        self.engine.finalize(block_id).await
    }

    async fn send_safe_l2_signal(&self, signal: ConsolidateInput) -> EngineClientResult<()> {
        trace!(target: "derivation", ?signal, "Sending safe L2 signal info to engine.");
        self.engine.consolidate(signal).await
    }
}
