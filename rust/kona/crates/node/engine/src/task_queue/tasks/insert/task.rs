//! A task to insert an unsafe payload into the execution engine.

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
    /// Optional sender for callers that need to await canonicalization.
    result_tx: Option<mpsc::Sender<Result<L2BlockInfo, InsertTaskError>>>,
    /// Whether the payload must still extend or equal the current unsafe head when this executes.
    require_current_unsafe_parent: bool,
    /// Whether insertion and forkchoice synchronization must both return `VALID`.
    require_valid_payload_status: bool,
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
        Self {
            client,
            rollup_config,
            payload,
            is_payload_safe: is_attributes_derived,
            block_sink,
            result_tx: None,
            require_current_unsafe_parent: false,
            require_valid_payload_status: false,
        }
    }

    /// Configures a response channel for callers that need to await canonicalization.
    pub fn with_result_sender(
        mut self,
        result_tx: mpsc::Sender<Result<L2BlockInfo, InsertTaskError>>,
    ) -> Self {
        self.result_tx = Some(result_tx);
        self
    }

    /// Requires the payload to extend or equal the current unsafe head when this task executes.
    pub const fn require_current_unsafe_parent(mut self) -> Self {
        self.require_current_unsafe_parent = true;
        self
    }

    /// Requires both engine calls to return `VALID` before reporting successful insertion.
    pub const fn require_valid_payload_status(mut self) -> Self {
        self.require_valid_payload_status = true;
        self
    }

    /// Returns whether this task reports its result to a caller.
    pub const fn has_result_sender(&self) -> bool {
        self.result_tx.is_some()
    }

    /// Executes the insertion and sends its result to the waiting caller.
    pub async fn execute_and_send(&self, state: &mut EngineState) -> Result<(), InsertTaskError> {
        let result = self.execute(state).await;
        self.result_tx
            .as_ref()
            .expect("result sender must be configured")
            .send(result)
            .await
            .map_err(|err| InsertTaskError::MpscSend(Box::new(err)))
    }

    /// Checks the response of the `engine_newPayload` call.
    const fn check_new_payload_status(&self, status: &PayloadStatusEnum) -> bool {
        matches!(status, PayloadStatusEnum::Valid) ||
            (!self.require_valid_payload_status && matches!(status, PayloadStatusEnum::Syncing))
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
        if self.require_current_unsafe_parent {
            let execution_payload = payload.clone().into_parts().0;
            let parent = execution_payload.parent_hash();
            let payload_hash = execution_payload.block_hash();
            let unsafe_head = state.sync_state.unsafe_head();
            if parent != unsafe_head.block_info.hash && payload_hash != unsafe_head.block_info.hash
            {
                let payload_number = execution_payload.block_number();
                if payload_number < unsafe_head.block_info.number {
                    let mut ancestor_hash = unsafe_head.block_info.parent_hash;
                    let mut ancestor_number = unsafe_head.block_info.number - 1;
                    while ancestor_number > payload_number {
                        let block = self
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
                        if self.require_valid_payload_status {
                            SynchronizeTask::new(
                                Arc::clone(&self.client),
                                self.rollup_config.clone(),
                                EngineSyncStateUpdate::default(),
                            )
                            .require_valid_payload_status()
                            .execute(state)
                            .await?;
                        }
                        info!(
                            target: "engine",
                            %payload_hash,
                            payload_number,
                            unsafe_head = %unsafe_head.block_info.hash,
                            "Retained sequencer payload is already an unsafe-chain ancestor"
                        );
                        return Ok(unsafe_head);
                    }
                }

                info!(
                    target: "engine",
                    %parent,
                    unsafe_head = %unsafe_head.block_info.hash,
                    "Dropping stale sequencer payload before insertion"
                );
                return Err(InsertTaskError::StalePayload {
                    parent,
                    unsafe_head: unsafe_head.block_info.hash,
                });
            }
        }

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
        let mut synchronize = SynchronizeTask::new(
            Arc::clone(&self.client),
            self.rollup_config.clone(),
            EngineSyncStateUpdate {
                unsafe_head: Some(new_unsafe_ref),
                local_safe_head: self.is_payload_safe.then_some(new_unsafe_ref),
                safe_head: self.is_payload_safe.then_some(new_unsafe_ref),
                ..Default::default()
            },
        );
        if self.require_valid_payload_status {
            synchronize = synchronize.require_valid_payload_status();
        }
        synchronize.execute(state).await?;

        // The block is now canonical, so anything reading the L2 chain locally can rely on it.
        self.block_sink.block_imported(block, new_unsafe_ref);

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
    async fn strict_insert_revalidates_syncing_payload_already_at_unsafe_head() {
        let client = Arc::new(
            test_engine_client_builder()
                .with_new_payload_v1_response(PayloadStatus::from_status(
                    PayloadStatusEnum::Syncing,
                ))
                .build(),
        );
        let task = InsertTask::new(
            client,
            Arc::new(RollupConfig::default()),
            // The payload is already the default state's unsafe head even though its parent is
            // not.
            test_payload(B256::repeat_byte(1)),
            false,
            Arc::new(NoopBlockSink),
        )
        .require_valid_payload_status();

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
    async fn checked_insert_classifies_older_payload_against_unsafe_chain(
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
        let task = InsertTask::new(
            client,
            Arc::new(RollupConfig::default()),
            payload,
            false,
            Arc::new(NoopBlockSink),
        )
        .require_current_unsafe_parent()
        .require_valid_payload_status();

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
    async fn checked_insert_rejects_payload_that_no_longer_extends_unsafe_head() {
        let unsafe_head = test_block_info(0);
        let stale_parent = B256::ZERO;
        assert_ne!(stale_parent, unsafe_head.hash());
        let mut state = TestEngineStateBuilder::new().with_unsafe_head(unsafe_head).build();
        let task = InsertTask::new(
            Arc::new(test_engine_client_builder().build()),
            Arc::new(RollupConfig::default()),
            test_payload(stale_parent),
            false,
            Arc::new(NoopBlockSink),
        )
        .require_current_unsafe_parent();

        let err = task.execute(&mut state).await.unwrap_err();
        assert!(matches!(
            err,
            InsertTaskError::StalePayload { parent, unsafe_head: actual }
                if parent == stale_parent && actual == unsafe_head.hash()
        ));
    }
}
