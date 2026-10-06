//! Import a sequenced payload after committing it to the conductor.

use crate::{
    EngineClient, EngineState, EngineTaskExt, ImportedBlockSink, SealTaskError,
    task_queue::insert_payload,
};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::sync::{mpsc, watch};

/// Canonicalizes a sequenced payload, checking that its build parent is still the unsafe head.
///
/// The conductor RPC runs outside the engine queue. Derivation or another unsafe payload may
/// advance the head while that RPC is pending, so this check and insertion must be atomic.
/// Unlike derived attributes, an invalid sequenced payload must not be replaced by a deposits-only
/// block here: that replacement would need its own conductor commit.
#[derive(Debug, Clone, Constructor)]
pub struct CanonicalizeTask {
    /// The engine API client.
    pub engine: Arc<EngineClient>,
    /// The rollup configuration.
    pub cfg: Arc<RollupConfig>,
    /// The payload to import.
    pub payload: OpExecutionPayloadEnvelope,
    /// The unsafe head on which the build started.
    pub parent: L2BlockInfo,
    /// The response channel. Errors are handled by the sequencer, which owns retry policy.
    pub result_tx: mpsc::Sender<Result<OpExecutionPayloadEnvelope, SealTaskError>>,
    /// Publish the new unsafe head before acknowledging canonicalization to the sequencer.
    pub unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,
    /// Where to hand the block after canonicalization.
    pub block_sink: Arc<dyn ImportedBlockSink>,
}

#[async_trait]
impl EngineTaskExt for CanonicalizeTask {
    type Output = ();
    type Error = SealTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), SealTaskError> {
        let head = state.sync_state.unsafe_head().block_info;
        let result = if head.hash != self.parent.block_info.hash ||
            head.number != self.parent.block_info.number
        {
            Err(SealTaskError::UnsafeHeadChangedSinceBuild)
        } else {
            insert_payload(
                self.engine.as_ref(),
                &self.cfg,
                state,
                self.payload.clone(),
                false,
                self.block_sink.as_ref(),
            )
            .await
            .map(|_| self.payload.clone())
            .map_err(|err| SealTaskError::PayloadInsertionFailed(Box::new(err)))
        };
        if result.is_ok() &&
            let Some(tx) = &self.unsafe_head_tx
        {
            tx.send_replace(state.sync_state.unsafe_head());
        }
        self.result_tx.send(result).await.map_err(|err| SealTaskError::MpscSend(Box::new(err)))
    }
}

#[cfg(test)]
#[path = "canonicalize_test.rs"]
pub(super) mod tests;
