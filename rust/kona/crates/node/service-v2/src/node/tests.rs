use super::*;
use crate::engine::{EngineDriver, EngineError, EngineResult, SafeChainUpdate};
use alloy_rpc_types_engine::ExecutionPayloadV1;
use async_trait::async_trait;
use kona_engine::EngineSyncState;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio_util::sync::CancellationToken;

#[derive(Debug)]
struct TestFollowerDriver {
    reject_import: bool,
}

#[async_trait]
impl EngineDriver for TestFollowerDriver {
    async fn build_unsafe(
        &mut self,
        _attributes: OpAttributesWithParent,
    ) -> EngineResult<OpExecutionPayloadEnvelope> {
        Err(EngineError::SequencingDisabled)
    }

    async fn import_unsafe(
        &mut self,
        _payload: OpExecutionPayloadEnvelope,
    ) -> EngineResult<L2BlockInfo> {
        if self.reject_import {
            Err(EngineError::Driver("test engine failure".into()))
        } else {
            Ok(L2BlockInfo::default())
        }
    }

    async fn update_safe(&mut self, _update: SafeChainUpdate) -> EngineResult<L2BlockInfo> {
        Ok(L2BlockInfo::default())
    }

    async fn update_finalized(&mut self, _block: L2BlockInfo) -> EngineResult<()> {
        Ok(())
    }

    fn state(&self) -> EngineSyncState {
        EngineSyncState::default()
    }
}

#[tokio::test]
async fn follower_node_shuts_down_in_dependency_order() {
    let (node, _engine, _ingress) = FollowerNode::new(TestFollowerDriver { reject_import: false });
    let shutdown = CancellationToken::new();
    let node_task = tokio::spawn(node.run(shutdown.clone()));

    shutdown.cancel();
    assert!(node_task.await.unwrap().is_ok());
}

#[tokio::test]
async fn follower_failure_stops_the_node() {
    let (node, _engine, ingress) = FollowerNode::new(TestFollowerDriver { reject_import: true });
    let node_task = tokio::spawn(node.run(CancellationToken::new()));

    ingress.send(test_payload()).await.unwrap();
    let error = node_task.await.unwrap().unwrap_err();
    assert!(matches!(
        error,
        NodeError::UnsafeChain(crate::unsafe_chain::UnsafeChainError::Engine(EngineError::Driver(
            _
        )))
    ));
}

fn test_payload() -> OpExecutionPayloadEnvelope {
    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1 {
        parent_hash: Default::default(),
        fee_recipient: Default::default(),
        state_root: Default::default(),
        receipts_root: Default::default(),
        logs_bloom: Default::default(),
        prev_randao: Default::default(),
        block_number: 0,
        gas_limit: 0,
        gas_used: 0,
        timestamp: 0,
        extra_data: Default::default(),
        base_fee_per_gas: Default::default(),
        block_hash: Default::default(),
        transactions: Vec::new(),
    })
}
