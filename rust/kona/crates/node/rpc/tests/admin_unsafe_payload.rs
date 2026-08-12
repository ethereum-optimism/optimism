//! Integration tests for `admin_postUnsafePayload` validation and routing.

use alloy_primitives::{Address, B256, Bytes, U256};
use alloy_rpc_types_engine::ExecutionPayloadV1;
use jsonrpsee::{
    RpcModule,
    types::{Response, ResponsePayload},
};
use kona_genesis::RollupConfig;
use kona_rpc::{AdminApiServer, AdminRpc, NetworkAdminQuery, SequencerAdminAPIClient};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::sync::mpsc;

#[derive(Debug)]
struct UnusedSequencerClient;

#[async_trait::async_trait]
impl SequencerAdminAPIClient for UnusedSequencerClient {
    async fn is_sequencer_active(&self) -> Result<bool, kona_rpc::SequencerAdminAPIError> {
        unreachable!()
    }

    async fn is_conductor_enabled(&self) -> Result<bool, kona_rpc::SequencerAdminAPIError> {
        unreachable!()
    }

    async fn is_recovery_mode(&self) -> Result<bool, kona_rpc::SequencerAdminAPIError> {
        unreachable!()
    }

    async fn start_sequencer(&self) -> Result<(), kona_rpc::SequencerAdminAPIError> {
        unreachable!()
    }

    async fn stop_sequencer(&self) -> Result<B256, kona_rpc::SequencerAdminAPIError> {
        unreachable!()
    }

    async fn set_recovery_mode(&self, _mode: bool) -> Result<(), kona_rpc::SequencerAdminAPIError> {
        unreachable!()
    }

    async fn override_leader(&self) -> Result<(), kona_rpc::SequencerAdminAPIError> {
        unreachable!()
    }

    async fn reset_derivation_pipeline(&self) -> Result<(), kona_rpc::SequencerAdminAPIError> {
        unreachable!()
    }
}

fn payload_with_bad_hash() -> OpExecutionPayloadEnvelope {
    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1 {
        parent_hash: B256::ZERO,
        fee_recipient: Address::ZERO,
        state_root: B256::ZERO,
        receipts_root: B256::ZERO,
        logs_bloom: Default::default(),
        prev_randao: B256::ZERO,
        block_number: 1,
        gas_limit: 30_000_000,
        gas_used: 0,
        timestamp: 2,
        extra_data: Bytes::new(),
        base_fee_per_gas: U256::from(1),
        block_hash: B256::repeat_byte(0x42),
        transactions: Vec::new(),
    })
}

#[tokio::test]
async fn admin_rejects_bad_block_hash_without_forwarding_payload() {
    let (network_sender, mut network_rx) = mpsc::channel(1);
    let rpc = AdminRpc::<UnusedSequencerClient>::new(
        None,
        network_sender,
        Arc::new(RollupConfig::default()),
    );
    let module: RpcModule<_> = rpc.into_rpc();
    let request = serde_json::json!({
        "jsonrpc": "2.0",
        "method": "admin_postUnsafePayload",
        "params": [payload_with_bad_hash()],
        "id": 1,
    })
    .to_string();

    let (response, _) = module.raw_json_request(&request, 1).await.expect("valid JSON-RPC request");
    let response: Response<'_, ()> = serde_json::from_str(response.get()).expect("valid response");
    let ResponsePayload::Error(error) = response.payload else {
        panic!("payload should be rejected")
    };

    assert_eq!(error.code(), -32602);
    assert!(error.message().contains("bad block hash"));
    assert!(network_rx.try_recv().is_err());
}

#[tokio::test]
async fn admin_forwards_payload_with_valid_block_hash() {
    let (network_sender, mut network_rx) = mpsc::channel(1);
    let rpc = AdminRpc::<UnusedSequencerClient>::new(
        None,
        network_sender,
        Arc::new(RollupConfig::default()),
    );
    let module: RpcModule<_> = rpc.into_rpc();
    let payload = OpExecutionPayloadEnvelope::from_block_slow(
        &payload_with_bad_hash()
            .try_into_block::<op_alloy_consensus::OpTxEnvelope>()
            .expect("decodable test payload"),
    )
    .expect("valid test payload");
    let request = serde_json::json!({
        "jsonrpc": "2.0",
        "method": "admin_postUnsafePayload",
        "params": [payload.clone()],
        "id": 1,
    })
    .to_string();

    let (response, _) = module.raw_json_request(&request, 1).await.expect("valid JSON-RPC request");
    let response: Response<'_, ()> = serde_json::from_str(response.get()).expect("valid response");
    assert!(matches!(response.payload, ResponsePayload::Success(_)));

    let NetworkAdminQuery::PostUnsafePayload { payload: forwarded } =
        network_rx.recv().await.expect("payload forwarded to network actor");
    assert_eq!(forwarded, payload);
}
