//! Tasks to insert payloads into the execution engine.

use super::super::synchronize::CanonicalizeForkchoiceTask;
use crate::{
    EngineClient, EngineState, EngineTaskExt, ImportedBlockSink, InsertTaskError, SynchronizeTask,
    state::EngineSyncStateUpdate,
};
use alloy_eips::{BlockId, eip1898::RpcBlockHash};
use alloy_rpc_types_engine::{ExecutionPayloadInputV2, PayloadStatusEnum};
use async_trait::async_trait;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_consensus::OpBlock;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{sync::Arc, time::Instant};
use tokio::sync::mpsc;

/// The task to insert a payload into the execution engine.
#[derive(Debug, Clone)]
pub struct InsertTask<EngineClient_: EngineClient> {
    /// The engine client.
    client: Arc<EngineClient_>,
    /// The rollup config.
    rollup_config: Arc<RollupConfig>,
    /// The complete execution payload envelope.
    payload: OpExecutionPayloadEnvelope,
    /// If the payload is safe this is true.
    /// A payload is safe if it is derived from a safe block.
    is_payload_safe: bool,
    /// Where to hand the decoded block once the engine has canonicalized it.
    block_sink: Arc<dyn ImportedBlockSink>,
}

/// Canonicalizes a sealed, conductor-approved payload under sequencing safety constraints.
#[derive(Debug, Clone)]
pub struct CanonicalizeTask<EngineClient_: EngineClient> {
    /// Common payload insertion data and operations.
    insertion: InsertTask<EngineClient_>,
    /// Where to send the canonicalization result.
    result_tx: mpsc::Sender<Result<L2BlockInfo, InsertTaskError>>,
}

impl<EngineClient_: EngineClient> InsertTask<EngineClient_> {
    /// Creates a new insert task.
    pub const fn new(
        client: Arc<EngineClient_>,
        rollup_config: Arc<RollupConfig>,
        payload: OpExecutionPayloadEnvelope,
        is_attributes_derived: bool,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> Self {
        Self { client, rollup_config, payload, is_payload_safe: is_attributes_derived, block_sink }
    }

    /// Submits the payload to the execution layer and returns its status and request duration.
    async fn submit_payload(
        &self,
    ) -> Result<(PayloadStatusEnum, std::time::Duration), InsertTaskError> {
        let insert_time_start = Instant::now();
        let response = match self.payload.clone() {
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

        match response {
            Ok(response) => Ok((response.status, insert_time_start.elapsed())),
            Err(err) => {
                warn!(target: "engine", "Failed to insert new payload: {err}");
                Err(InsertTaskError::InsertFailed(err))
            }
        }
    }

    /// Decodes the payload and derives its L2 block information.
    fn decode_payload(&self) -> Result<(OpBlock, L2BlockInfo), InsertTaskError> {
        let block: OpBlock =
            self.payload.clone().try_into_block().map_err(InsertTaskError::FromBlockError)?;
        let block_info = L2BlockInfo::from_block_and_genesis(&block, &self.rollup_config.genesis)
            .map_err(InsertTaskError::L2BlockInfoConstruction)?;
        Ok((block, block_info))
    }

    /// Reports a successfully canonicalized block to downstream consumers and metrics.
    fn complete_import(
        &self,
        block: OpBlock,
        block_info: L2BlockInfo,
        time_start: Instant,
        insert_duration: std::time::Duration,
    ) {
        self.block_sink.block_imported(block, block_info);

        info!(
            target: "engine",
            hash = %block_info.block_info.hash,
            number = block_info.block_info.number,
            total_duration = ?time_start.elapsed(),
            insert_duration = ?insert_duration,
            "Inserted new unsafe block"
        );
    }
}

impl<EngineClient_: EngineClient> CanonicalizeTask<EngineClient_> {
    /// Creates a task for canonicalizing a conductor-approved payload.
    pub const fn new(
        client: Arc<EngineClient_>,
        rollup_config: Arc<RollupConfig>,
        payload: OpExecutionPayloadEnvelope,
        block_sink: Arc<dyn ImportedBlockSink>,
        result_tx: mpsc::Sender<Result<L2BlockInfo, InsertTaskError>>,
    ) -> Self {
        Self {
            insertion: InsertTask::new(client, rollup_config, payload, false, block_sink),
            result_tx,
        }
    }

    /// Executes canonicalization and sends its result to the waiting sequencer.
    pub async fn execute_and_send(&self, state: &mut EngineState) -> Result<(), InsertTaskError> {
        let result = self.execute(state).await;
        self.result_tx.send(result).await.map_err(|err| InsertTaskError::MpscSend(Box::new(err)))
    }

    /// Returns the current head when the retained payload is already on the unsafe chain.
    ///
    /// Conflicting payloads are rejected, while ancestry lookup failures remain retryable so the
    /// sequencer keeps the exact conductor-approved payload.
    async fn classify_retained_payload(
        &self,
        state: &mut EngineState,
    ) -> Result<Option<L2BlockInfo>, InsertTaskError> {
        let execution_payload = self.insertion.payload.clone().into_parts().0;
        let parent = execution_payload.parent_hash();
        let payload_hash = execution_payload.block_hash();
        let payload_number = execution_payload.block_number();
        let unsafe_head = state.sync_state.unsafe_head();

        if parent == unsafe_head.block_info.hash || payload_hash == unsafe_head.block_info.hash {
            return Ok(None);
        }

        if payload_number < unsafe_head.block_info.number {
            let mut ancestor_hash = unsafe_head.block_info.parent_hash;
            let mut ancestor_number = unsafe_head.block_info.number - 1;
            while ancestor_number > payload_number {
                let block = self
                    .insertion
                    .client
                    .get_l2_block(BlockId::Hash(RpcBlockHash::from_hash(
                        ancestor_hash,
                        Some(false),
                    )))
                    .await
                    .map_err(InsertTaskError::AncestryLookupFailed)?
                    .ok_or(InsertTaskError::AncestorBlockNotFound {
                        hash: ancestor_hash,
                        number: ancestor_number,
                    })?;
                ancestor_hash = block.header.inner.parent_hash;
                ancestor_number -= 1;
            }

            if ancestor_hash == payload_hash {
                CanonicalizeForkchoiceTask::new(
                    Arc::clone(&self.insertion.client),
                    self.insertion.rollup_config.clone(),
                    EngineSyncStateUpdate::default(),
                )
                .execute(state)
                .await?;
                info!(
                    target: "engine",
                    %payload_hash,
                    payload_number,
                    unsafe_head = %unsafe_head.block_info.hash,
                    "Retained sequencer payload is already an unsafe-chain ancestor"
                );
                return Ok(Some(unsafe_head));
            }
        }

        info!(
            target: "engine",
            %parent,
            unsafe_head = %unsafe_head.block_info.hash,
            "Dropping stale sequencer payload before insertion"
        );
        Err(InsertTaskError::StalePayload { parent, unsafe_head: unsafe_head.block_info.hash })
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for InsertTask<EngineClient_> {
    type Output = L2BlockInfo;
    type Error = InsertTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<Self::Output, Self::Error> {
        let time_start = Instant::now();
        let (status, insert_duration) = self.submit_payload().await?;
        if !matches!(status, PayloadStatusEnum::Valid | PayloadStatusEnum::Syncing) {
            return Err(InsertTaskError::UnexpectedPayloadStatus(status));
        }

        let (block, new_unsafe_ref) = self.decode_payload()?;
        SynchronizeTask::new(
            Arc::clone(&self.client),
            self.rollup_config.clone(),
            EngineSyncStateUpdate {
                unsafe_head: Some(new_unsafe_ref),
                local_safe_head: self.is_payload_safe.then_some(new_unsafe_ref),
                safe_head: self.is_payload_safe.then_some(new_unsafe_ref),
                ..Default::default()
            },
        )
        .execute(state)
        .await?;

        self.complete_import(block, new_unsafe_ref, time_start, insert_duration);
        Ok(new_unsafe_ref)
    }
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for CanonicalizeTask<EngineClient_> {
    type Output = L2BlockInfo;
    type Error = InsertTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<Self::Output, Self::Error> {
        let time_start = Instant::now();
        if let Some(unsafe_head) = self.classify_retained_payload(state).await? {
            return Ok(unsafe_head);
        }

        let (status, insert_duration) = self.insertion.submit_payload().await?;
        if !matches!(status, PayloadStatusEnum::Valid) {
            return Err(InsertTaskError::UnexpectedPayloadStatus(status));
        }

        let (block, new_unsafe_ref) = self.insertion.decode_payload()?;
        CanonicalizeForkchoiceTask::new(
            Arc::clone(&self.insertion.client),
            self.insertion.rollup_config.clone(),
            EngineSyncStateUpdate { unsafe_head: Some(new_unsafe_ref), ..Default::default() },
        )
        .execute(state)
        .await?;

        self.insertion.complete_import(block, new_unsafe_ref, time_start, insert_duration);
        Ok(new_unsafe_ref)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        EngineTaskExt, NoopBlockSink,
        test_utils::{TestEngineStateBuilder, test_block_info, test_engine_client_builder},
    };
    use alloy_primitives::{Address, B256, Bloom, Bytes, U256};
    use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadStatus, PayloadStatusEnum};
    use alloy_rpc_types_eth::{Block as RpcBlock, BlockTransactions, Header as RpcHeader};
    use op_alloy_rpc_types::Transaction;
    use rstest::rstest;

    fn test_payload(parent_hash: B256) -> OpExecutionPayloadEnvelope {
        OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1 {
            parent_hash,
            fee_recipient: Address::ZERO,
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            prev_randao: B256::ZERO,
            block_number: 1,
            gas_limit: 30_000_000,
            gas_used: 0,
            timestamp: 2,
            extra_data: Bytes::new(),
            base_fee_per_gas: U256::from(1),
            block_hash: B256::ZERO,
            transactions: Vec::new(),
        })
    }

    #[tokio::test]
    async fn canonicalization_revalidates_syncing_payload_already_at_unsafe_head() {
        let client = Arc::new(
            test_engine_client_builder()
                .with_new_payload_v1_response(PayloadStatus::from_status(
                    PayloadStatusEnum::Syncing,
                ))
                .build(),
        );
        let (result_tx, _result_rx) = mpsc::channel(1);
        let task = CanonicalizeTask::new(
            client,
            Arc::new(RollupConfig::default()),
            // The payload is already the default state's unsafe head even though its parent is
            // not.
            test_payload(B256::repeat_byte(1)),
            Arc::new(NoopBlockSink),
            result_tx,
        );

        let err = task.execute(&mut EngineState::default()).await.unwrap_err();
        assert!(matches!(
            err,
            InsertTaskError::UnexpectedPayloadStatus(PayloadStatusEnum::Syncing)
        ));
    }

    #[derive(Debug)]
    enum OlderPayloadOutcome {
        Complete,
        Conflict,
        Retry,
    }

    #[rstest]
    #[case::matching_ancestor(true, PayloadStatusEnum::Valid, OlderPayloadOutcome::Complete)]
    #[case::conflicting_block(false, PayloadStatusEnum::Valid, OlderPayloadOutcome::Conflict)]
    #[case::syncing_ancestor(true, PayloadStatusEnum::Syncing, OlderPayloadOutcome::Retry)]
    #[tokio::test]
    async fn canonicalization_classifies_older_payload_against_unsafe_chain(
        #[case] is_ancestor: bool,
        #[case] forkchoice_status: PayloadStatusEnum,
        #[case] expected: OlderPayloadOutcome,
    ) {
        let payload_hash = B256::repeat_byte(2);
        let intermediate_header = RpcHeader::new(alloy_consensus::Header {
            parent_hash: if is_ancestor { payload_hash } else { B256::repeat_byte(3) },
            number: 2,
            ..Default::default()
        });
        let intermediate_hash = intermediate_header.hash;
        let intermediate_block =
            RpcBlock::new(intermediate_header, BlockTransactions::Full(Vec::<Transaction>::new()));
        let payload = match test_payload(B256::repeat_byte(1)) {
            OpExecutionPayloadEnvelope::V1(mut payload) => {
                payload.block_hash = payload_hash;
                OpExecutionPayloadEnvelope::V1(payload)
            }
            _ => unreachable!("test payload is V1"),
        };
        let mut unsafe_head = test_block_info(3);
        unsafe_head.block_info.parent_hash = intermediate_hash;
        let mut state = TestEngineStateBuilder::new().with_unsafe_head(unsafe_head).build();
        let block_id = BlockId::Hash(RpcBlockHash::from_hash(intermediate_hash, Some(false)));
        let client = Arc::new(
            test_engine_client_builder()
                .with_l2_block(block_id, intermediate_block)
                .with_fork_choice_updated_v3_response(
                    alloy_rpc_types_engine::ForkchoiceUpdated::new(PayloadStatus::from_status(
                        forkchoice_status,
                    )),
                )
                .build(),
        );
        let (result_tx, _result_rx) = mpsc::channel(1);
        let task = CanonicalizeTask::new(
            client,
            Arc::new(RollupConfig::default()),
            payload,
            Arc::new(NoopBlockSink),
            result_tx,
        );

        let result = task.execute(&mut state).await;
        match expected {
            OlderPayloadOutcome::Complete => assert_eq!(result.unwrap(), unsafe_head),
            OlderPayloadOutcome::Conflict => {
                assert!(matches!(result, Err(InsertTaskError::StalePayload { .. })));
            }
            OlderPayloadOutcome::Retry => assert!(matches!(
                result,
                Err(InsertTaskError::ForkchoiceUpdateFailed(
                    crate::SynchronizeTaskError::UnexpectedPayloadStatus(
                        PayloadStatusEnum::Syncing
                    )
                ))
            )),
        }
    }

    #[tokio::test]
    async fn canonicalization_rejects_payload_that_no_longer_extends_unsafe_head() {
        let unsafe_head = test_block_info(0);
        let stale_parent = B256::ZERO;
        assert_ne!(stale_parent, unsafe_head.hash());
        let mut state = TestEngineStateBuilder::new().with_unsafe_head(unsafe_head).build();
        let (result_tx, _result_rx) = mpsc::channel(1);
        let task = CanonicalizeTask::new(
            Arc::new(test_engine_client_builder().build()),
            Arc::new(RollupConfig::default()),
            test_payload(stale_parent),
            Arc::new(NoopBlockSink),
            result_tx,
        );

        let err = task.execute(&mut state).await.unwrap_err();
        assert!(matches!(
            err,
            InsertTaskError::StalePayload { parent, unsafe_head: actual }
                if parent == stale_parent && actual == unsafe_head.hash()
        ));
    }
}
