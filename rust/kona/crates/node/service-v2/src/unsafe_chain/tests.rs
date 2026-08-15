use super::*;
use crate::engine::{EngineDriver, EngineError, EngineResult, EngineService, SafeChainUpdate};
use alloy_rpc_types_engine::ExecutionPayloadV1;
use async_trait::async_trait;
use kona_engine::EngineSyncState;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::{
    Arc,
    atomic::{AtomicUsize, Ordering},
};
use tokio::sync::Notify;
use tokio_util::sync::CancellationToken;

#[derive(Debug)]
struct InvalidThenValidDriver {
    imports: Arc<AtomicUsize>,
    imported: Arc<Notify>,
}

#[async_trait]
impl EngineDriver for InvalidThenValidDriver {
    async fn build_unsafe(
        &mut self,
        _attributes: OpAttributesWithParent,
    ) -> EngineResult<OpExecutionPayloadEnvelope> {
        unreachable!("follower test does not build payloads")
    }

    async fn import_unsafe(
        &mut self,
        _payload: OpExecutionPayloadEnvelope,
    ) -> EngineResult<L2BlockInfo> {
        let import = self.imports.fetch_add(1, Ordering::SeqCst);
        self.imported.notify_one();
        if import == 0 {
            Err(EngineError::InvalidUnsafePayload("test rejection".into()))
        } else {
            Ok(L2BlockInfo::default())
        }
    }

    async fn update_safe(&mut self, _update: SafeChainUpdate) -> EngineResult<L2BlockInfo> {
        unreachable!("follower test does not update the safe chain")
    }

    async fn update_finalized(&mut self, _block: L2BlockInfo) -> EngineResult<()> {
        unreachable!("follower test does not update finality")
    }

    fn state(&self) -> EngineSyncState {
        EngineSyncState::default()
    }
}

#[tokio::test]
async fn follower_drops_invalid_payload_and_continues() {
    let imports = Arc::new(AtomicUsize::new(0));
    let imported = Arc::new(Notify::new());
    let (engine_service, engine) = EngineService::new(InvalidThenValidDriver {
        imports: Arc::clone(&imports),
        imported: Arc::clone(&imported),
    });
    let engine_shutdown = CancellationToken::new();
    let engine_task = tokio::spawn(engine_service.run(engine_shutdown.clone()));
    let _engine_guard = engine.clone();

    let (follower, ingress) = FollowerService::new(engine);
    let follower_shutdown = CancellationToken::new();
    let follower_task = tokio::spawn(follower.run(follower_shutdown.clone()));

    ingress.send(test_payload()).await.unwrap();
    ingress.send(test_payload()).await.unwrap();
    while imports.load(Ordering::SeqCst) < 2 {
        imported.notified().await;
    }

    follower_shutdown.cancel();
    assert_eq!(follower_task.await.unwrap(), Ok(()));
    engine_shutdown.cancel();
    assert_eq!(engine_task.await.unwrap(), Ok(()));
    assert_eq!(imports.load(Ordering::SeqCst), 2);
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
