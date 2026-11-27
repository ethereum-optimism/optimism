//! Import a sequenced payload after committing it to the conductor.

use crate::{
    DeniedBlocks, EngineClient, EngineState, EngineTaskError, ExecutableEngineTask,
    ImportedBlockSink, InsertTaskError,
    task_queue::{insert_payload, tasks::task::EngineTaskErrorSeverity},
};
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::{mpsc, watch};

/// An error that occurs when running the [`CanonicalizeTask`].
#[derive(Debug, Error)]
pub enum CanonicalizeTaskError {
    /// The unsafe head changed after the payload was built, so the payload no longer extends it.
    #[error("Unsafe head changed between build and canonicalization")]
    UnsafeHeadChangedSinceBuild,
    /// A cross-chain verifier denied the payload's block.
    #[error("The cross-chain verifier denied block {0:?}")]
    Denied(BlockNumHash),
    /// Impossible to insert the payload into the engine.
    #[error(transparent)]
    PayloadInsertionFailed(#[from] Box<InsertTaskError>),
    /// Error sending the canonicalization result.
    #[error(transparent)]
    MpscSend(#[from] Box<mpsc::error::SendError<Result<(), Self>>>),
}

impl EngineTaskError for CanonicalizeTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::PayloadInsertionFailed(inner) => inner.severity(),
            Self::UnsafeHeadChangedSinceBuild | Self::Denied(_) | Self::MpscSend(_) => {
                EngineTaskErrorSeverity::Critical
            }
        }
    }
}

/// Canonicalizes a sequenced payload, checking that its build parent is still the unsafe head.
///
/// The conductor RPC runs outside the engine queue. Derivation or another unsafe payload may
/// advance the head while that RPC is pending, so this check and insertion must be atomic.
/// Unlike derived attributes, an invalid sequenced payload must not be replaced by a deposits-only
/// block here: that replacement would need its own conductor commit. As in op-node, this also
/// applies when the rejected payload contains only deposits: the sequencer drops it and rebuilds.
/// This differs from rejecting deposits-only attributes while starting a build.
///
/// As op-node checks its own sealed payloads against its super authority's deny list, a payload
/// whose block a cross-chain verifier denied is not imported.
#[derive(Debug, Clone)]
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
    pub result_tx: mpsc::Sender<Result<(), CanonicalizeTaskError>>,
    /// Publish the new unsafe head before acknowledging canonicalization to the sequencer.
    pub unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,
    /// Where to hand the block after canonicalization.
    pub block_sink: Arc<dyn ImportedBlockSink>,
    /// The blocks a cross-chain verifier denied.
    pub denied: DeniedBlocks,
}

impl CanonicalizeTask {
    /// Creates a task that imports `payload`, built on `parent`, and answers on `result_tx`.
    pub fn new(
        engine: Arc<EngineClient>,
        cfg: Arc<RollupConfig>,
        payload: OpExecutionPayloadEnvelope,
        parent: L2BlockInfo,
        result_tx: mpsc::Sender<Result<(), CanonicalizeTaskError>>,
        unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> Self {
        Self {
            engine,
            cfg,
            payload,
            parent,
            result_tx,
            unsafe_head_tx,
            block_sink,
            denied: DeniedBlocks::none(),
        }
    }

    /// Refuses a payload whose block `denied` holds. A verifier that has not answered yet
    /// refuses nothing.
    pub fn with_denied_blocks(self, denied: DeniedBlocks) -> Self {
        Self { denied, ..self }
    }

    /// The payload's block, if the verifier denied it.
    fn denied_block(&self) -> Option<BlockNumHash> {
        let block =
            BlockNumHash { number: self.payload.block_number(), hash: self.payload.block_hash() };
        self.denied.contains(block, self.payload.timestamp()).unwrap_or(false).then_some(block)
    }
}

#[async_trait]
impl ExecutableEngineTask for CanonicalizeTask {
    type Output = ();
    type Error = CanonicalizeTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), CanonicalizeTaskError> {
        let head = state.sync_state.unsafe_head().block_info;
        let result = if head.hash != self.parent.block_info.hash ||
            head.number != self.parent.block_info.number
        {
            Err(CanonicalizeTaskError::UnsafeHeadChangedSinceBuild)
        } else if let Some(denied) = self.denied_block() {
            warn!(target: "engine", ?denied, "Refusing a sequenced payload the cross-chain verifier denied");
            Err(CanonicalizeTaskError::Denied(denied))
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
            .map(|_| ())
            .map_err(|err| CanonicalizeTaskError::PayloadInsertionFailed(Box::new(err)))
        };
        if result.is_ok() &&
            let Some(tx) = &self.unsafe_head_tx
        {
            tx.send_replace(state.sync_state.unsafe_head());
        }
        self.result_tx
            .send(result)
            .await
            .map_err(|err| CanonicalizeTaskError::MpscSend(Box::new(err)))
    }
}

#[cfg(test)]
#[path = "canonicalize_test.rs"]
pub(super) mod tests;
