use super::{
    Conductor, ConductorError, OriginSelector, SequencingWorkflow, SequencingWorkflowFactory,
};
use crate::engine::{
    EngineRequest,
    api::{BuiltUnsafePayload, EngineInternalHandle},
    network::NetworkClient,
};
use alloy_consensus::Block;
use alloy_rpc_types_engine::ExecutionPayloadV1;
use alloy_transport::RpcError;
use async_trait::async_trait;
use kona_derive::test_utils::TestAttributesBuilder;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use op_alloy_consensus::OpTxEnvelope;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use std::{
    sync::{
        Arc, Mutex,
        atomic::{AtomicUsize, Ordering},
    },
    time::Duration,
};

#[derive(Debug)]
struct FixedOrigin;

#[async_trait]
impl OriginSelector for FixedOrigin {
    async fn next_l1_origin(
        &mut self,
        _unsafe_head: L2BlockInfo,
        _is_recovery_mode: bool,
    ) -> Result<BlockInfo, super::L1OriginSelectorError> {
        Ok(BlockInfo::default())
    }
}

#[derive(Debug)]
struct RecordingConductor {
    events: Arc<Mutex<Vec<&'static str>>>,
    failures_remaining: AtomicUsize,
}

#[async_trait]
impl Conductor for RecordingConductor {
    async fn commit_unsafe_payload(
        &self,
        _payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        self.events.lock().unwrap().push("conductor");
        if self
            .failures_remaining
            .fetch_update(Ordering::SeqCst, Ordering::SeqCst, |remaining| remaining.checked_sub(1))
            .is_ok()
        {
            return Err(ConductorError::Rpc(RpcError::local_usage_str("conductor unavailable")));
        }
        Ok(())
    }

    async fn override_leader(&self) -> Result<(), ConductorError> {
        Ok(())
    }
}

fn payload() -> OpExecutionPayloadEnvelope {
    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
        &Block::<OpTxEnvelope>::default(),
    ))
}

#[test]
fn each_local_production_start_constructs_fresh_workflow_state() {
    let creations = Arc::new(AtomicUsize::new(0));
    let observed_creations = creations.clone();
    let factory = SequencingWorkflowFactory::new(
        move || {
            observed_creations.fetch_add(1, Ordering::SeqCst);
            let (engine, _engine_rx) = EngineInternalHandle::test_pair(1);
            let (network, _network_rx) = NetworkClient::test_pair(1);
            SequencingWorkflow::new(
                Box::new(TestAttributesBuilder::default()),
                None,
                engine,
                network,
                Box::new(FixedOrigin),
                Arc::new(RollupConfig::default()),
            )
        },
        None,
    );

    let first = factory.create();
    drop(first);
    let second = factory.create();
    drop(second);

    assert_eq!(creations.load(Ordering::SeqCst), 2);
}

#[tokio::test]
async fn conductor_authorization_precedes_gossip_and_canonicalization() {
    let events = Arc::new(Mutex::new(Vec::new()));
    let expected_payload = payload();
    let canonical_head = L2BlockInfo {
        block_info: BlockInfo { number: 1, ..Default::default() },
        ..Default::default()
    };

    let (engine, mut engine_rx) = EngineInternalHandle::test_pair(8);
    let engine_events = events.clone();
    let engine_payload = expected_payload.clone();
    let engine_task = tokio::spawn(async move {
        while let Some(request) = engine_rx.recv().await {
            match request {
                EngineRequest::State { response } => {
                    let _ = response.send(Ok(Default::default()));
                }
                EngineRequest::BuildUnsafe { response, .. } => {
                    engine_events.lock().unwrap().push("build");
                    let _ = response.send(Ok(BuiltUnsafePayload::test_new(
                        engine_payload.clone(),
                        L2BlockInfo::default(),
                    )));
                }
                EngineRequest::CanonicalizeUnsafe { candidate, response } => {
                    assert_eq!(candidate.payload().payload_hash(), engine_payload.payload_hash());
                    engine_events.lock().unwrap().push("import");
                    let _ = response.send(Ok(canonical_head));
                    break;
                }
                request => panic!("unexpected engine request: {request:?}"),
            }
        }
    });

    let (network, mut publish_rx) = NetworkClient::test_pair(1);
    let network_events = events.clone();
    let network_payload = expected_payload.clone();
    let network_task = tokio::spawn(async move {
        let request = publish_rx.recv().await.expect("publication request");
        assert_eq!(request.payload.payload_hash(), network_payload.payload_hash());
        network_events.lock().unwrap().push("gossip");
        let _ = request.response.send(Ok(()));
    });

    let conductor =
        RecordingConductor { events: events.clone(), failures_remaining: AtomicUsize::new(1) };
    let attributes = TestAttributesBuilder { attributes: vec![Ok(OpPayloadAttributes::default())] };
    let mut workflow = SequencingWorkflow::new(
        Box::new(attributes),
        Some(Arc::new(conductor)),
        engine,
        network,
        Box::new(FixedOrigin),
        Arc::new(RollupConfig { block_time: 0, ..Default::default() }),
    );

    let block = tokio::time::timeout(Duration::from_secs(1), workflow.sequence_one(false))
        .await
        .expect("workflow timed out")
        .expect("workflow failed")
        .expect("workflow replanned");
    assert_eq!(block, canonical_head);

    engine_task.await.unwrap();
    network_task.await.unwrap();
    assert_eq!(*events.lock().unwrap(), ["build", "conductor", "conductor", "gossip", "import"]);
}

#[tokio::test]
async fn follower_quiesces_and_joins_through_engine_owned_lifecycle() {
    let (engine, _engine_rx) = EngineInternalHandle::test_pair(1);
    let (_payload_tx, payload_rx) = tokio::sync::mpsc::channel(1);
    let (service, _sequencer, lifecycle) = super::UnsafeChainService::follower(engine, payload_rx);
    let task = tokio::spawn(service.run());

    let (done, completed) = tokio::sync::oneshot::channel();
    lifecycle.send(super::control::UnsafeLifecycleCommand::Quiesce(done)).await.unwrap();
    completed.await.unwrap();

    let (done, completed) = tokio::sync::oneshot::channel();
    lifecycle.send(super::control::UnsafeLifecycleCommand::Shutdown(done)).await.unwrap();
    completed.await.unwrap();
    task.await.unwrap().unwrap();
}
