use alloy_consensus::{BlockBody, EMPTY_ROOT_HASH, Header, Sealed};
use alloy_eips::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, bytes::BufMut};
use alloy_rpc_types_engine::{
    ExecutionPayloadEnvelopeV2, ExecutionPayloadFieldV2, ForkchoiceUpdated, PayloadStatusEnum,
};
use async_trait::async_trait;
use kona_derive::Signal;
use kona_engine::{
    Engine, EngineState,
    test_utils::{MockEngineClient, MockNewPayloadResponse},
};
use kona_genesis::RollupConfig;
use kona_node_service::{
    DerivationClientResult, EngineActor, EngineActorRequest, EngineDerivationClient,
};
use kona_protocol::{L2BlockInfo, test_utils::RAW_BEDROCK_INFO_TX};
use op_alloy_consensus::{OpBlock, OpTxEnvelope, TxDeposit};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::sync::{Mutex, mpsc, watch};

#[derive(Debug, Clone, Default)]
pub(super) struct RecordingDerivationClient {
    signals: Arc<Mutex<Vec<Signal>>>,
}

impl RecordingDerivationClient {
    pub(super) async fn signals(&self) -> Vec<Signal> {
        self.signals.lock().await.clone()
    }
}

#[async_trait]
impl EngineDerivationClient for RecordingDerivationClient {
    async fn notify_sync_completed(&self, _safe_head: L2BlockInfo) -> DerivationClientResult<()> {
        Ok(())
    }

    async fn send_new_engine_safe_head(
        &self,
        _safe_head: L2BlockInfo,
    ) -> DerivationClientResult<()> {
        Ok(())
    }

    async fn send_signal(&self, signal: Signal) -> DerivationClientResult<()> {
        self.signals.lock().await.push(signal);
        Ok(())
    }
}

pub(super) struct EngineHarness {
    pub(super) client: MockEngineClient,
    pub(super) derivation: RecordingDerivationClient,
    pub(super) requests: mpsc::Sender<EngineActorRequest>,
    pub(super) unsafe_head: watch::Receiver<L2BlockInfo>,
    pub(super) queue_length: watch::Receiver<usize>,
    pub(super) actor: EngineActor<MockEngineClient, RecordingDerivationClient>,
}

impl EngineHarness {
    pub(super) fn new(
        config: Arc<RollupConfig>,
        responses: impl IntoIterator<Item = MockNewPayloadResponse>,
        initial_head: L2BlockInfo,
    ) -> Self {
        let client = MockEngineClient::builder()
            .with_config(config.clone())
            .with_new_payload_responses(responses)
            .with_fork_choice_updated_v3_response(ForkchoiceUpdated::from_status(
                PayloadStatusEnum::Valid,
            ))
            .build();
        let derivation = RecordingDerivationClient::default();
        let sync_state =
            EngineState::default().sync_state.apply_update(kona_engine::EngineSyncStateUpdate {
                unsafe_head: Some(initial_head),
                cross_unsafe_head: Some(initial_head),
                local_safe_head: Some(initial_head),
                safe_head: Some(initial_head),
                finalized_head: Some(initial_head),
            });
        let state = EngineState {
            sync_state,
            el_sync_finished: false,
            need_fcu_call_backup_unsafe_reorg: false,
        };
        let (state_tx, _) = watch::channel(state);
        let (queue_length_tx, queue_length) = watch::channel(0);
        let engine = Engine::new(state, state_tx, queue_length_tx);
        let (unsafe_head_tx, unsafe_head) = watch::channel(initial_head);
        let (requests, request_rx) = mpsc::channel(16);
        let actor = EngineActor::new(
            Arc::new(client.clone()),
            config,
            derivation.clone(),
            engine,
            Some(unsafe_head_tx),
            request_rx,
        );
        Self { client, derivation, requests, unsafe_head, queue_length, actor }
    }

    pub(super) async fn set_payload_to_get(&self, payload: OpExecutionPayloadEnvelope) {
        let execution_payload = match payload {
            OpExecutionPayloadEnvelope::V1(payload) => ExecutionPayloadFieldV2::V1(payload),
            OpExecutionPayloadEnvelope::V2(payload) => ExecutionPayloadFieldV2::V2(payload),
            _ => panic!("test harness only supports V1/V2 getPayload responses"),
        };
        self.client
            .set_execution_payload_v2(ExecutionPayloadEnvelopeV2 {
                execution_payload,
                block_value: U256::ZERO,
            })
            .await;
    }
}

pub(super) fn genesis_head() -> L2BlockInfo {
    L2BlockInfo::default()
}

pub(super) fn valid_payload(parent: L2BlockInfo, number: u64) -> OpExecutionPayloadEnvelope {
    let deposit = OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
        to: TxKind::Call(Address::ZERO),
        gas_limit: 1_000_000,
        input: Bytes::from_static(&RAW_BEDROCK_INFO_TX),
        ..Default::default()
    }));
    let encoded = Bytes::from(deposit.encoded_2718());
    let transactions_root =
        alloy_consensus::proofs::ordered_trie_root_with_encoder(&[encoded], |item, buf| {
            buf.put_slice(item)
        });
    let header = Header {
        parent_hash: parent.block_info.hash,
        transactions_root,
        receipts_root: EMPTY_ROOT_HASH,
        number,
        gas_limit: 30_000_000,
        timestamp: number.saturating_mul(2),
        base_fee_per_gas: Some(1),
        ..Default::default()
    };
    let block = OpBlock {
        header,
        body: BlockBody { transactions: vec![deposit], ommers: Vec::new(), withdrawals: None },
    };
    OpExecutionPayloadEnvelope::from_block_slow(&block).expect("valid test payload")
}

pub(super) fn payload_info(payload: &OpExecutionPayloadEnvelope) -> L2BlockInfo {
    let block: OpBlock = payload.clone().try_into_block().expect("decodable payload");
    L2BlockInfo::from_block_and_genesis(&block, &Default::default()).expect("L2 block info")
}

pub(super) fn malformed_payload(number: u64) -> OpExecutionPayloadEnvelope {
    use alloy_rpc_types_engine::ExecutionPayloadV1;

    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1 {
        parent_hash: B256::ZERO,
        fee_recipient: Address::ZERO,
        state_root: B256::ZERO,
        receipts_root: B256::ZERO,
        logs_bloom: Default::default(),
        prev_randao: B256::ZERO,
        block_number: number,
        gas_limit: 30_000_000,
        gas_used: 0,
        timestamp: number.saturating_mul(2),
        extra_data: Bytes::new(),
        base_fee_per_gas: U256::from(1),
        block_hash: B256::ZERO,
        transactions: vec![Bytes::from_static(&[0xff])],
    })
}
