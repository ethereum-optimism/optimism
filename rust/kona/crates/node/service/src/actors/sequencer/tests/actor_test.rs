use crate::{
    Conductor, ConductorError, EngineClientError, SequencerActor, SequencerActorError,
    SequencerEngineClient, UnsafePayloadGossipClient, UnsafePayloadGossipClientError,
    actors::{
        MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
    },
};
use alloy_consensus::Block;
use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadId};
use async_trait::async_trait;
use kona_derive::{BuilderError, PipelineErrorKind, test_utils::TestAttributesBuilder};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_consensus::OpTxEnvelope;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadAttributes};
use rstest::rstest;
use std::{
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, Ordering},
    },
    time::Duration,
};
use tokio::sync::Notify;
use tokio_util::sync::CancellationToken;

#[rstest]
#[case::temp(PipelineErrorKind::Temporary(BuilderError::Custom(String::new()).into()), false)]
#[case::reset(PipelineErrorKind::Reset(BuilderError::Custom(String::new()).into()), false)]
#[case::critical(PipelineErrorKind::Critical(BuilderError::Custom(String::new()).into()), true)]
#[tokio::test]
async fn test_build_payload_prepare_payload_attributes_error(
    #[case] forced_error: PipelineErrorKind,
    #[case] expect_err: bool,
) {
    let mut client = MockSequencerEngineClient::new();

    let unsafe_head = L2BlockInfo::default();
    client.expect_get_unsafe_head().times(1).return_once(move || Ok(unsafe_head));
    client.expect_start_build_block().times(0);
    if let PipelineErrorKind::Reset(_) = &forced_error {
        client.expect_reset_engine_forkchoice().times(1).return_once(move || Ok(()));
    }

    let l1_origin = BlockInfo::default();
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(move |_, _| Ok(l1_origin));

    let attributes_builder = TestAttributesBuilder { attributes: vec![Err(forced_error)] };
    let (mut actor, _handle) = SequencerActor::<_, MockConductor, _, _, _>::new(
        attributes_builder,
        None,
        client,
        true,
        false,
        origin_selector,
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        MockUnsafePayloadGossipClient::new(),
    );

    let result = actor.workflow.build_payload(false).await;
    if expect_err {
        assert!(matches!(
            result.unwrap_err(),
            SequencerActorError::AttributesBuilder(PipelineErrorKind::Critical(_))
        ));
    } else {
        assert!(result.is_ok());
    }
}

#[derive(Debug)]
struct BlockingSealEngine {
    canonicalized: Arc<AtomicBool>,
    head: Arc<Mutex<L2BlockInfo>>,
    payload: OpExecutionPayloadEnvelope,
    release_seal: Arc<Notify>,
    seal_started: Arc<Notify>,
}

#[async_trait]
impl SequencerEngineClient for BlockingSealEngine {
    async fn reset_engine_forkchoice(&self) -> Result<(), EngineClientError> {
        Ok(())
    }

    async fn start_build_block(
        &self,
        _attributes: OpAttributesWithParent,
    ) -> Result<PayloadId, EngineClientError> {
        Ok(PayloadId::default())
    }

    async fn seal_block(
        &self,
        _payload_id: PayloadId,
        _attributes: OpAttributesWithParent,
    ) -> Result<OpExecutionPayloadEnvelope, EngineClientError> {
        self.seal_started.notify_one();
        self.release_seal.notified().await;
        Ok(self.payload.clone())
    }

    async fn canonicalize_block(
        &self,
        _payload: OpExecutionPayloadEnvelope,
        _attributes: OpAttributesWithParent,
    ) -> Result<L2BlockInfo, EngineClientError> {
        let canonical_head = canonical_head();
        *self.head.lock().unwrap() = canonical_head;
        self.canonicalized.store(true, Ordering::SeqCst);
        Ok(canonical_head)
    }

    async fn get_unsafe_head(&self) -> Result<L2BlockInfo, EngineClientError> {
        Ok(*self.head.lock().unwrap())
    }
}

#[derive(Debug)]
struct BlockingConductor {
    commit_started: Arc<Notify>,
    release_commit: Arc<Notify>,
}

#[async_trait]
impl Conductor for BlockingConductor {
    async fn commit_unsafe_payload(
        &self,
        _payload: &OpExecutionPayloadEnvelope,
    ) -> Result<(), ConductorError> {
        self.commit_started.notify_one();
        self.release_commit.notified().await;
        Ok(())
    }

    async fn override_leader(&self) -> Result<(), ConductorError> {
        Ok(())
    }
}

#[derive(Debug)]
struct RecordingEngine {
    canonicalized: Arc<AtomicBool>,
    head: Arc<Mutex<L2BlockInfo>>,
    payload: OpExecutionPayloadEnvelope,
}

#[async_trait]
impl SequencerEngineClient for RecordingEngine {
    async fn reset_engine_forkchoice(&self) -> Result<(), EngineClientError> {
        Ok(())
    }

    async fn start_build_block(
        &self,
        _attributes: OpAttributesWithParent,
    ) -> Result<PayloadId, EngineClientError> {
        Ok(PayloadId::default())
    }

    async fn seal_block(
        &self,
        _payload_id: PayloadId,
        _attributes: OpAttributesWithParent,
    ) -> Result<OpExecutionPayloadEnvelope, EngineClientError> {
        Ok(self.payload.clone())
    }

    async fn canonicalize_block(
        &self,
        _payload: OpExecutionPayloadEnvelope,
        _attributes: OpAttributesWithParent,
    ) -> Result<L2BlockInfo, EngineClientError> {
        let canonical_head = canonical_head();
        *self.head.lock().unwrap() = canonical_head;
        self.canonicalized.store(true, Ordering::SeqCst);
        Ok(canonical_head)
    }

    async fn get_unsafe_head(&self) -> Result<L2BlockInfo, EngineClientError> {
        Ok(*self.head.lock().unwrap())
    }
}

#[derive(Debug)]
struct RecordingGossip(Arc<AtomicBool>);

#[async_trait]
impl UnsafePayloadGossipClient for RecordingGossip {
    async fn schedule_execution_payload_gossip(
        &self,
        _payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafePayloadGossipClientError> {
        self.0.store(true, Ordering::SeqCst);
        Ok(())
    }
}

fn test_payload() -> OpExecutionPayloadEnvelope {
    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
        &Block::<OpTxEnvelope>::default(),
    ))
}

fn canonical_head() -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            number: 1,
            hash: alloy_primitives::B256::repeat_byte(1),
            ..Default::default()
        },
        ..Default::default()
    }
}

#[tokio::test]
async fn sequencer_handle_encapsulates_stopped_admin_operations() {
    let unsafe_head = canonical_head();
    let mut engine = MockSequencerEngineClient::new();
    engine.expect_reset_engine_forkchoice().times(2).returning(|| Ok(()));
    engine.expect_get_unsafe_head().times(1).return_once(move || Ok(unsafe_head));

    let mut conductor = MockConductor::new();
    conductor.expect_override_leader().times(1).return_once(|| Ok(()));

    let (actor, control) = SequencerActor::new(
        TestAttributesBuilder { attributes: Vec::new() },
        Some(conductor),
        engine,
        false,
        false,
        MockOriginSelector::new(),
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        MockUnsafePayloadGossipClient::new(),
    );
    let shutdown = CancellationToken::new();
    let actor_handle = tokio::spawn(actor.run(shutdown.clone()));

    assert!(!control.is_active().await.unwrap());
    assert!(control.conductor_enabled().await.unwrap());
    assert!(!control.recovery_mode().await.unwrap());
    control.set_recovery_mode(true).await.unwrap();
    assert!(control.recovery_mode().await.unwrap());
    control.override_leader().await.unwrap();
    control.reset_derivation_pipeline().await.unwrap();
    assert_eq!(control.stop().await.unwrap(), canonical_head().hash());

    shutdown.cancel();
    tokio::time::timeout(Duration::from_secs(1), actor_handle)
        .await
        .expect("sequencer should shut down")
        .unwrap()
        .unwrap();
}

#[tokio::test]
async fn admin_stop_waits_for_the_current_block() {
    let canonicalized = Arc::new(AtomicBool::new(false));
    let gossiped = Arc::new(AtomicBool::new(false));
    let release_seal = Arc::new(Notify::new());
    let seal_started = Arc::new(Notify::new());
    let engine = BlockingSealEngine {
        canonicalized: canonicalized.clone(),
        head: Arc::new(Mutex::new(L2BlockInfo::default())),
        payload: test_payload(),
        release_seal: release_seal.clone(),
        seal_started: seal_started.clone(),
    };
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(|_, _| Ok(BlockInfo::default()));
    let (actor, control) = SequencerActor::<_, MockConductor, _, _, _>::new(
        TestAttributesBuilder { attributes: vec![Ok(OpPayloadAttributes::default())] },
        None,
        engine,
        false,
        false,
        origin_selector,
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        RecordingGossip(gossiped.clone()),
    );
    let shutdown = CancellationToken::new();
    let actor_handle = tokio::spawn(actor.run(shutdown.clone()));

    assert!(!control.is_active().await.unwrap());
    assert!(!control.conductor_enabled().await.unwrap());
    assert!(!control.recovery_mode().await.unwrap());
    control.set_recovery_mode(true).await.unwrap();
    assert!(control.recovery_mode().await.unwrap());
    control.start().await.unwrap();

    tokio::time::timeout(Duration::from_secs(1), seal_started.notified())
        .await
        .expect("seal should start");
    let stop_control = control.clone();
    let mut stop_handle = tokio::spawn(async move { stop_control.stop().await });
    assert!(
        tokio::time::timeout(Duration::from_millis(20), &mut stop_handle).await.is_err(),
        "stop should remain queued while the block is being sequenced"
    );
    assert!(!gossiped.load(Ordering::SeqCst));
    assert!(!canonicalized.load(Ordering::SeqCst));

    release_seal.notify_one();
    let stopped_head = tokio::time::timeout(Duration::from_secs(1), stop_handle)
        .await
        .expect("stop should complete at the next block boundary")
        .unwrap()
        .unwrap();
    assert_eq!(stopped_head, canonical_head().hash());
    assert!(gossiped.load(Ordering::SeqCst));
    assert!(canonicalized.load(Ordering::SeqCst));
    assert!(!control.is_active().await.unwrap());
    assert_eq!(control.stop().await.unwrap(), canonical_head().hash());

    shutdown.cancel();
    tokio::time::timeout(Duration::from_secs(1), actor_handle)
        .await
        .expect("sequencer should shut down")
        .unwrap()
        .unwrap();
}

#[tokio::test]
async fn admin_stop_does_not_cancel_protected_distribution() {
    let commit_started = Arc::new(Notify::new());
    let release_commit = Arc::new(Notify::new());
    let conductor = BlockingConductor {
        commit_started: commit_started.clone(),
        release_commit: release_commit.clone(),
    };
    let canonicalized = Arc::new(AtomicBool::new(false));
    let gossiped = Arc::new(AtomicBool::new(false));
    let engine = RecordingEngine {
        canonicalized: canonicalized.clone(),
        head: Arc::new(Mutex::new(L2BlockInfo::default())),
        payload: test_payload(),
    };
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(|_, _| Ok(BlockInfo::default()));
    let (actor, control) = SequencerActor::new(
        TestAttributesBuilder { attributes: vec![Ok(OpPayloadAttributes::default())] },
        Some(conductor),
        engine,
        true,
        false,
        origin_selector,
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        RecordingGossip(gossiped.clone()),
    );
    let shutdown = CancellationToken::new();
    let actor_handle = tokio::spawn(actor.run(shutdown.clone()));

    tokio::time::timeout(Duration::from_secs(1), commit_started.notified())
        .await
        .expect("conductor commit should start");
    let mut stop_handle = tokio::spawn(async move { control.stop().await });
    assert!(
        tokio::time::timeout(Duration::from_millis(20), &mut stop_handle).await.is_err(),
        "stop must wait for the current block"
    );
    assert!(!gossiped.load(Ordering::SeqCst));
    assert!(!canonicalized.load(Ordering::SeqCst));

    release_commit.notify_one();
    let stopped_head = tokio::time::timeout(Duration::from_secs(1), stop_handle)
        .await
        .expect("stop should complete after distribution")
        .unwrap()
        .unwrap();
    assert_eq!(stopped_head, alloy_primitives::B256::repeat_byte(1));
    assert!(gossiped.load(Ordering::SeqCst));
    assert!(canonicalized.load(Ordering::SeqCst));

    shutdown.cancel();
    tokio::time::timeout(Duration::from_secs(1), actor_handle)
        .await
        .expect("sequencer should shut down")
        .unwrap()
        .unwrap();
}

#[tokio::test]
async fn node_shutdown_drains_protected_distribution() {
    let commit_started = Arc::new(Notify::new());
    let release_commit = Arc::new(Notify::new());
    let conductor = BlockingConductor {
        commit_started: commit_started.clone(),
        release_commit: release_commit.clone(),
    };
    let canonicalized = Arc::new(AtomicBool::new(false));
    let gossiped = Arc::new(AtomicBool::new(false));
    let engine = RecordingEngine {
        canonicalized: canonicalized.clone(),
        head: Arc::new(Mutex::new(L2BlockInfo::default())),
        payload: test_payload(),
    };
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(|_, _| Ok(BlockInfo::default()));
    let (actor, _control) = SequencerActor::new(
        TestAttributesBuilder { attributes: vec![Ok(OpPayloadAttributes::default())] },
        Some(conductor),
        engine,
        true,
        false,
        origin_selector,
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        RecordingGossip(gossiped.clone()),
    );
    let shutdown = CancellationToken::new();
    let mut handle = tokio::spawn(actor.run(shutdown.clone()));

    tokio::time::timeout(Duration::from_secs(1), commit_started.notified())
        .await
        .expect("conductor commit should start");
    shutdown.cancel();
    assert!(
        tokio::time::timeout(Duration::from_millis(20), &mut handle).await.is_err(),
        "shutdown must retain protected distribution"
    );
    assert!(!gossiped.load(Ordering::SeqCst));
    assert!(!canonicalized.load(Ordering::SeqCst));

    release_commit.notify_one();
    tokio::time::timeout(Duration::from_secs(1), handle)
        .await
        .expect("sequencer should exit after protected distribution")
        .unwrap()
        .unwrap();
    assert!(gossiped.load(Ordering::SeqCst));
    assert!(canonicalized.load(Ordering::SeqCst));
}
