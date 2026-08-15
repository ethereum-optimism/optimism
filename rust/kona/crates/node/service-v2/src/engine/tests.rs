use super::*;
use alloy_rpc_types_engine::ExecutionPayloadV1;
use async_trait::async_trait;
use kona_engine::EngineSyncState;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::sync::{Arc, Mutex};
use tokio::sync::Notify;
use tokio_util::sync::CancellationToken;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum RecordedCall {
    BuildUnsafe,
    ImportUnsafe,
    UpdateSafeAttributes,
    UpdateSafeBlock,
    UpdateFinalized,
}

#[derive(Debug)]
struct RecordingDriver {
    calls: Arc<Mutex<Vec<RecordedCall>>>,
}

impl RecordingDriver {
    fn record(&self, call: RecordedCall) {
        self.calls.lock().expect("recording mutex poisoned").push(call);
    }
}

#[async_trait]
impl EngineDriver for RecordingDriver {
    async fn build_unsafe(
        &mut self,
        _attributes: OpAttributesWithParent,
    ) -> EngineResult<OpExecutionPayloadEnvelope> {
        self.record(RecordedCall::BuildUnsafe);
        Ok(test_payload())
    }

    async fn import_unsafe(
        &mut self,
        _payload: OpExecutionPayloadEnvelope,
    ) -> EngineResult<L2BlockInfo> {
        self.record(RecordedCall::ImportUnsafe);
        Ok(L2BlockInfo::default())
    }

    async fn update_safe(&mut self, update: SafeChainUpdate) -> EngineResult<L2BlockInfo> {
        self.record(match update {
            SafeChainUpdate::Attributes(_) => RecordedCall::UpdateSafeAttributes,
            SafeChainUpdate::Block(_) => RecordedCall::UpdateSafeBlock,
        });
        Ok(L2BlockInfo::default())
    }

    async fn update_finalized(&mut self, _block: L2BlockInfo) -> EngineResult<()> {
        self.record(RecordedCall::UpdateFinalized);
        Ok(())
    }

    fn state(&self) -> EngineSyncState {
        EngineSyncState::default()
    }
}

#[tokio::test]
async fn semantic_operations_are_serialized_by_one_driver() {
    let calls = Arc::new(Mutex::new(Vec::new()));
    let (service, client) = EngineService::new(RecordingDriver { calls: Arc::clone(&calls) });
    let shutdown = CancellationToken::new();
    let service_task = tokio::spawn(service.run(shutdown.clone()));

    let attributes = test_attributes();
    let payload = client.build_unsafe(attributes.clone()).await.unwrap();
    client.import_unsafe(payload).await.unwrap();
    client.update_safe(attributes).await.unwrap();
    client.update_safe(L2BlockInfo::default()).await.unwrap();
    client.update_finalized(L2BlockInfo::default()).await.unwrap();
    assert_eq!(client.state().await.unwrap(), EngineSyncState::default());

    shutdown.cancel();
    assert_eq!(service_task.await.unwrap(), Ok(()));
    assert_eq!(
        *calls.lock().expect("recording mutex poisoned"),
        vec![
            RecordedCall::BuildUnsafe,
            RecordedCall::ImportUnsafe,
            RecordedCall::UpdateSafeAttributes,
            RecordedCall::UpdateSafeBlock,
            RecordedCall::UpdateFinalized,
        ]
    );
}

#[derive(Debug)]
struct BlockingDriver {
    started: Arc<Notify>,
    release: Arc<Notify>,
}

#[async_trait]
impl EngineDriver for BlockingDriver {
    async fn build_unsafe(
        &mut self,
        _attributes: OpAttributesWithParent,
    ) -> EngineResult<OpExecutionPayloadEnvelope> {
        self.started.notify_one();
        self.release.notified().await;
        Ok(test_payload())
    }

    async fn import_unsafe(
        &mut self,
        _payload: OpExecutionPayloadEnvelope,
    ) -> EngineResult<L2BlockInfo> {
        unreachable!("test only issues a build")
    }

    async fn update_safe(&mut self, _update: SafeChainUpdate) -> EngineResult<L2BlockInfo> {
        unreachable!("test only issues a build")
    }

    async fn update_finalized(&mut self, _block: L2BlockInfo) -> EngineResult<()> {
        unreachable!("test only issues a build")
    }

    fn state(&self) -> EngineSyncState {
        EngineSyncState::default()
    }
}

#[tokio::test]
async fn shutdown_waits_for_an_active_engine_operation() {
    let started = Arc::new(Notify::new());
    let release = Arc::new(Notify::new());
    let (service, client) = EngineService::new(BlockingDriver {
        started: Arc::clone(&started),
        release: Arc::clone(&release),
    });
    let shutdown = CancellationToken::new();
    let service_task = tokio::spawn(service.run(shutdown.clone()));
    let build_task = tokio::spawn(async move { client.build_unsafe(test_attributes()).await });

    started.notified().await;
    shutdown.cancel();
    tokio::task::yield_now().await;
    assert!(!service_task.is_finished());

    release.notify_one();
    assert!(build_task.await.unwrap().is_ok());
    assert_eq!(service_task.await.unwrap(), Ok(()));
}

fn test_attributes() -> OpAttributesWithParent {
    OpAttributesWithParent::new(OpPayloadAttributes::default(), L2BlockInfo::default(), None, false)
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
