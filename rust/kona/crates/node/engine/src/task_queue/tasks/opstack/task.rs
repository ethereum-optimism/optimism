//! A task for a block-building step of the `opstack` RPC namespace.

use super::OpenBlockError;
use crate::{
    EngineClient, EngineState, ImportedBlockSink, InsertTaskError, OpStackRequest, SealTaskError,
    task_queue::{get_payload, insert_payload, start_build},
};
use alloy_eips::{BlockId, BlockNumHash};
use alloy_rpc_types_engine::PayloadId;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::sync::Arc;
use tokio::sync::watch;

/// Runs an [`OpStackRequest`], a step of block building by a sequencer outside the node, and
/// sends its outcome to the caller.
///
/// As with op-node's `opstack` API, the caller owns its retries, so the task never fails the
/// queue: a rejected request neither resets nor stalls the engine. A committed block is an unsafe
/// block; it moves neither the local-safe head nor the safe label.
#[derive(Debug, Clone, Constructor)]
pub struct OpStackTask {
    /// The engine API client.
    pub client: Arc<EngineClient>,
    /// The rollup configuration.
    pub cfg: Arc<RollupConfig>,
    /// The step to run.
    pub request: OpStackRequest,
    /// Receives the unsafe head a commit moved to before the caller hears of the commit, so the
    /// node's sequencer can be started on it right away.
    pub unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,
    /// Where to hand a committed block once it is canonical.
    pub block_sink: Arc<dyn ImportedBlockSink>,
}

impl OpStackTask {
    /// Runs the step and sends its outcome. A caller that stopped waiting does not undo the step.
    pub async fn execute(&self, state: &mut EngineState) {
        match &self.request {
            OpStackRequest::Open { parent, attributes, result_tx } => {
                let result = self.open(state, *parent, attributes.as_ref().clone()).await;
                let _ = result_tx.send(result).await;
            }
            OpStackRequest::GetPayload { payload_id, timestamp, result_tx } => {
                let result = self.payload(*payload_id, *timestamp).await;
                let _ = result_tx.send(result).await;
            }
            OpStackRequest::Commit { payload, result_tx } => {
                let result = self.commit(state, payload.as_ref().clone()).await;
                let _ = result_tx.send(result).await;
            }
        }
    }

    /// Starts building a block with `attributes` on `parent`, which can be any block the execution
    /// layer has, not only the unsafe head. Like op-node, it finds the parent by hash.
    async fn open(
        &self,
        state: &EngineState,
        parent: BlockNumHash,
        attributes: OpPayloadAttributes,
    ) -> Result<PayloadId, OpenBlockError> {
        let block = self
            .client
            .get_l2_block(BlockId::hash(parent.hash))
            .full()
            .await?
            .ok_or(OpenBlockError::ParentNotFound(parent.hash))?;
        let parent =
            L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &self.cfg.genesis)?;
        let attributes = OpAttributesWithParent::new(attributes, parent, None, false);
        Ok(start_build(self.client.as_ref(), &self.cfg, state, attributes).await?)
    }

    /// Fetches the payload of a build without importing it.
    async fn payload(
        &self,
        payload_id: PayloadId,
        timestamp: u64,
    ) -> Result<OpExecutionPayloadEnvelope, SealTaskError> {
        get_payload(self.client.as_ref(), &self.cfg, payload_id, timestamp).await
    }

    /// Imports `payload` and makes it the unsafe head.
    async fn commit(
        &self,
        state: &mut EngineState,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), InsertTaskError> {
        let head = insert_payload(
            self.client.as_ref(),
            &self.cfg,
            state,
            payload,
            false,
            self.block_sink.as_ref(),
        )
        .await?;
        if let Some(unsafe_head_tx) = &self.unsafe_head_tx {
            unsafe_head_tx.send_replace(state.sync_state.unsafe_head());
        }
        info!(target: "engine", number = head.block_info.number, hash = %head.block_info.hash, "Committed a block built over the opstack API");
        Ok(())
    }
}
