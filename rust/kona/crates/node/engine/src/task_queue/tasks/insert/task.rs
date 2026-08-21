//! A task to insert an unsafe payload into the execution engine.

use crate::{
    EngineClient, EngineState, EngineTaskExt, InsertTaskError, SynchronizeTask,
    state::EngineSyncStateUpdate,
};
use alloy_rpc_types_engine::{ExecutionPayloadInputV2, PayloadStatusEnum};
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{sync::Arc, time::Instant};

/// The task to insert a payload into the execution engine.
#[derive(Debug, Clone)]
pub struct InsertTask<EngineClient_: EngineClient> {
    /// The engine client.
    client: Arc<EngineClient_>,
    /// The rollup config.
    rollup_config: Arc<RollupConfig>,
    /// The complete execution payload envelope.
    payload: OpExecutionPayloadEnvelope,
    /// If the payload is local-safe this is true.
    /// A payload is local-safe if it was derived from L1, rather than received over gossip.
    is_payload_local_safe: bool,
}

impl<EngineClient_: EngineClient> InsertTask<EngineClient_> {
    /// Creates a new insert task.
    pub const fn new(
        client: Arc<EngineClient_>,
        rollup_config: Arc<RollupConfig>,
        payload: OpExecutionPayloadEnvelope,
        is_attributes_derived: bool,
    ) -> Self {
        Self { client, rollup_config, payload, is_payload_local_safe: is_attributes_derived }
    }

    /// Checks the response of the `engine_newPayload` call.
    const fn check_new_payload_status(&self, status: &PayloadStatusEnum) -> bool {
        matches!(status, PayloadStatusEnum::Valid | PayloadStatusEnum::Syncing)
    }

    /// The block number this payload claims.
    pub(crate) const fn payload_block_number(&self) -> u64 {
        self.payload.block_number()
    }

    /// Whether this payload may become the unsafe head, i.e. whether it descends from the
    /// local-safe head.
    ///
    /// A gossiped payload is untrusted input, and inserting one moves the unsafe head to it. The
    /// unsafe head is the top of the ordering `unsafe >= local-safe >= cross-safe >= finalized`, so
    /// a payload at or below the local-safe head would rewind the unsafe head under a head derived
    /// from L1 and the next [`crate::EngineSyncState::create_forkchoice_state`] would report
    /// `safeBlockHash` ahead of `headBlockHash` — the `INVALID_FORK_CHOICE_STATE` rejection that
    /// costs a full engine reset.
    ///
    /// [`crate::EngineSyncState::apply_update`] deliberately clamps only the cross-safe head and
    /// leaves the unsafe head alone, on the grounds that no writer produces an unsafe head below
    /// cross-safe. This check is what makes that true for the one writer fed from outside the node.
    /// It cannot live in the actor that enqueues the payload: the local-safe head can advance
    /// between enqueue and execution, so only a check against the state the write actually uses
    /// holds the ordering.
    ///
    /// The execution layer's own forkchoice-consistency check does not substitute for this, so the
    /// rejection is made here rather than inferred from the `engine_newPayload` response.
    ///
    /// Untrusted input is rejected, not clamped: the payload is dropped and the heads stay put.
    /// Two cases are decidable locally, and only those two:
    ///
    /// - **At or below local-safe.** Rejected. Either it is the local-safe block itself — already
    ///   canonical, nothing to insert — or it is a competitor on a fork L1-derived data has already
    ///   ruled out.
    /// - **Exactly one above local-safe.** Rejected unless its parent *is* the local-safe head.
    ///
    /// Anything further ahead is admitted: its ancestry back to the local-safe head is not on hand,
    /// and derivation settles it when local-safe catches up. A payload arriving before the engine
    /// has a local-safe head at all — the execution-layer sync bootstrap — is admitted for the same
    /// reason: there is nothing yet to descend from.
    pub(crate) fn descends_from_local_safe(&self, state: &EngineState) -> bool {
        // A derived payload is a local-safe write, not an unsafe one; it defines the head this
        // check compares against instead of being checked by it.
        if self.is_payload_local_safe {
            return true;
        }

        let local_safe = state.sync_state.local_safe_head();
        if local_safe == L2BlockInfo::default() {
            return true;
        }

        match self.payload.block_number().checked_sub(local_safe.block_info.number) {
            Some(0) | None => false,
            Some(1) => self.payload.parent_hash() == local_safe.block_info.hash,
            _ => true,
        }
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for InsertTask<EngineClient_> {
    type Output = L2BlockInfo;

    type Error = InsertTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<L2BlockInfo, InsertTaskError> {
        let time_start = Instant::now();

        // Insert the new payload.
        // Form the new unsafe block ref from the execution payload.
        let payload = self.payload.clone();
        let insert_time_start = Instant::now();
        let response = match payload.clone() {
            OpExecutionPayloadEnvelope::V1(payload) => self.client.new_payload_v1(payload).await,
            OpExecutionPayloadEnvelope::V2(payload) => {
                let payload_input = ExecutionPayloadInputV2 {
                    execution_payload: payload.payload_inner,
                    withdrawals: Some(payload.withdrawals),
                };
                self.client.new_payload_v2(payload_input).await
            }
            OpExecutionPayloadEnvelope::V3 { payload, parent_beacon_block_root } => {
                self.client.new_payload_v3(payload, parent_beacon_block_root).await
            }
            OpExecutionPayloadEnvelope::V4 { payload, parent_beacon_block_root } => {
                self.client.new_payload_v4(payload, parent_beacon_block_root).await
            }
        };

        // Check the `engine_newPayload` response.
        let response = match response {
            Ok(resp) => resp,
            Err(e) => {
                warn!(target: "engine", "Failed to insert new payload: {e}");
                return Err(InsertTaskError::InsertFailed(e));
            }
        };
        if !self.check_new_payload_status(&response.status) {
            return Err(InsertTaskError::UnexpectedPayloadStatus(response.status));
        }
        let insert_duration = insert_time_start.elapsed();

        let block: OpBlock = payload.try_into_block().map_err(InsertTaskError::FromBlockError)?;
        let new_unsafe_ref =
            L2BlockInfo::from_block_and_genesis(&block, &self.rollup_config.genesis)
                .map_err(InsertTaskError::L2BlockInfoConstruction)?;

        // Send a FCU to canonicalize the imported block.
        SynchronizeTask::new(
            Arc::clone(&self.client),
            self.rollup_config.clone(),
            EngineSyncStateUpdate {
                unsafe_head: Some(new_unsafe_ref),
                local_safe_head: self.is_payload_local_safe.then_some(new_unsafe_ref),
                ..Default::default()
            },
        )
        .execute(state)
        .await?;

        let total_duration = time_start.elapsed();

        info!(
            target: "engine",
            hash = %new_unsafe_ref.block_info.hash,
            number = new_unsafe_ref.block_info.number,
            total_duration = ?total_duration,
            insert_duration = ?insert_duration,
            "Inserted new unsafe block"
        );

        Ok(new_unsafe_ref)
    }
}
