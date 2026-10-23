use super::{TEST_START, building::step_time, test_actor, test_actor_with_config};
use crate::actors::{MockOriginSelector, MockSequencerEngineClient};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadId};
use kona_derive::{BuilderError, PipelineErrorKind, test_utils::TestAttributesBuilder};
use kona_protocol::{BlockInfo, L2BlockInfo};
use kona_rpc::SequencerAdminCommand;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use rstest::rstest;
use std::sync::{
    Arc,
    atomic::{AtomicU64, Ordering},
};
use tokio::{sync::oneshot, time::Duration};

fn head(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            number,
            hash: B256::repeat_byte(1),
            timestamp: TEST_START,
            ..Default::default()
        },
        ..Default::default()
    }
}

fn payload(number: u64) -> OpExecutionPayloadEnvelope {
    let mut block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
    block.header.number = number;
    OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(&block))
}

#[rstest]
#[case::disabled(0, u64::MAX, 0, false)]
#[case::below_limit(3, 100, 98, false)]
#[case::at_limit(3, 100, 97, true)]
#[case::above_limit(3, 100, 96, true)]
#[case::safe_ahead(3, 100, 101, false)]
#[case::no_addition_overflow(u64::MAX, u64::MAX, 1, false)]
#[case::maximum_gap(u64::MAX, u64::MAX, 0, true)]
#[tokio::test(start_paused = true)]
async fn safe_lag_bounds_new_builds(
    #[case] limit: u64,
    #[case] unsafe_number: u64,
    #[case] safe_number: u64,
    #[case] stalled: bool,
) {
    let mut actor = test_actor();
    actor.started = true;
    actor.max_safe_lag = limit;
    let mut engine = MockSequencerEngineClient::new();
    let steps = if stalled { 2 } else { 1 };
    engine.expect_get_unsafe_head().times(steps).returning(move || Ok(head(unsafe_number)));
    engine
        .expect_get_safe_head()
        .times(if limit == 0 { 0 } else { steps })
        .returning(move || Ok(head(safe_number)));
    engine.expect_start_build_block().times(0);
    actor.engine_client = engine;
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);

    let mut origins = MockOriginSelector::new();
    origins
        .expect_next_l1_origin()
        .times(usize::from(!stalled))
        .returning(|_, _| Ok(BlockInfo::default()));
    actor.origin_selector = origins;
    actor.attributes_builder = TestAttributesBuilder {
        attributes: vec![Err(PipelineErrorKind::Temporary(
            BuilderError::Custom("retry".into()).into(),
        ))],
    };

    assert_eq!(step_time(&mut actor).await, Duration::ZERO);
    assert!(actor.state().active, "lag must not administratively stop the sequencer");
    assert!(actor.in_flight.is_none());
    if stalled {
        assert_eq!(
            step_time(&mut actor).await,
            Duration::from_secs(2),
            "paused ticks must not spin"
        );
    }
}

#[tokio::test(start_paused = true)]
async fn safe_head_catchup_resumes_and_each_published_block_rechecks_the_limit() {
    let mut actor = test_actor();
    actor.started = true;
    actor.max_safe_lag = 2;
    let unsafe_number = Arc::new(AtomicU64::new(10));
    let safe_number = Arc::new(AtomicU64::new(8));
    let mut engine = MockSequencerEngineClient::new();
    let unsafe_read = unsafe_number.clone();
    engine
        .expect_get_unsafe_head()
        .times(5)
        .returning(move || Ok(head(unsafe_read.load(Ordering::SeqCst))));
    let safe_read = safe_number.clone();
    engine
        .expect_get_safe_head()
        .times(5)
        .returning(move || Ok(head(safe_read.load(Ordering::SeqCst))));
    engine.expect_start_build_block().times(2).returning(|_| Ok(PayloadId::new([1; 8])));
    engine.expect_seal_block().times(1).return_once(|_, _| Ok(payload(11)));
    engine.expect_canonicalize_block().times(1).return_once(move |_, parent| {
        unsafe_number.store(parent.block_info.number + 1, Ordering::SeqCst);
        Ok(())
    });
    actor.engine_client = engine;
    actor
        .origin_selector
        .expect_next_l1_origin()
        .times(2)
        .returning(|_, _| Ok(BlockInfo::default()));
    actor.attributes_builder =
        TestAttributesBuilder { attributes: (0..2).map(|_| Ok(Default::default())).collect() };
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .times(1)
        .returning(|_| Ok(()));

    step_time(&mut actor).await;
    assert!(actor.in_flight.is_none());
    safe_number.store(9, Ordering::SeqCst);
    step_time(&mut actor).await;
    assert!(actor.in_flight.is_some(), "safe-head advance must release the gate");
    step_time(&mut actor).await;
    assert!(actor.in_flight.is_none(), "publishing block 11 must stall at gap 2");
    assert_eq!(step_time(&mut actor).await, Duration::from_secs(2));
    safe_number.store(10, Ordering::SeqCst);
    step_time(&mut actor).await;
    assert!(actor.in_flight.is_some(), "catch-up must release a second stall too");
    assert!(actor.state().active);
}

#[tokio::test(start_paused = true)]
async fn safe_lag_pause_keeps_stop_start_responsive_and_does_not_bypass_the_limit() {
    let (mut actor, commands) = test_actor_with_config(true, false, None);
    actor.started = true;
    actor.max_safe_lag = 2;
    let safe_number = Arc::new(AtomicU64::new(8));
    actor.engine_client.expect_get_unsafe_head().times(5).returning(|| Ok(head(10)));
    let safe_read = safe_number.clone();
    actor
        .engine_client
        .expect_get_safe_head()
        .times(3)
        .returning(move || Ok(head(safe_read.load(Ordering::SeqCst))));
    actor
        .engine_client
        .expect_start_build_block()
        .times(1)
        .return_once(|_| Ok(PayloadId::new([1; 8])));
    actor
        .origin_selector
        .expect_next_l1_origin()
        .times(1)
        .return_once(|_, _| Ok(BlockInfo::default()));
    actor.attributes_builder = TestAttributesBuilder { attributes: vec![Ok(Default::default())] };
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);

    step_time(&mut actor).await;
    let (tx, rx) = oneshot::channel();
    commands.send(SequencerAdminCommand::StopSequencer(tx)).await.unwrap();
    assert_eq!(step_time(&mut actor).await, Duration::ZERO);
    assert_eq!(rx.await.unwrap().unwrap(), head(10).hash());
    assert!(!actor.state().active);
    let (tx, rx) = oneshot::channel();
    commands.send(SequencerAdminCommand::StartSequencer(head(10).hash(), tx)).await.unwrap();
    assert_eq!(step_time(&mut actor).await, Duration::ZERO);
    rx.await.unwrap().unwrap();
    step_time(&mut actor).await;
    assert!(actor.in_flight.is_none(), "start must not override the lag bound");
    safe_number.store(10, Ordering::SeqCst);
    step_time(&mut actor).await;
    assert!(actor.in_flight.is_some());
}

#[tokio::test(start_paused = true)]
async fn safe_head_rewind_reapplies_the_limit_in_recovery_mode() {
    let mut actor = test_actor();
    actor.started = true;
    actor.max_safe_lag = 2;
    actor.set_recovery_mode(true);
    let safe_number = Arc::new(AtomicU64::new(9));
    actor.engine_client.expect_get_unsafe_head().times(3).returning(|| Ok(head(10)));
    let safe_read = safe_number.clone();
    actor
        .engine_client
        .expect_get_safe_head()
        .times(3)
        .returning(move || Ok(head(safe_read.load(Ordering::SeqCst))));
    actor.engine_client.expect_start_build_block().times(0);
    actor
        .origin_selector
        .expect_next_l1_origin()
        .times(2)
        .returning(|_, _| Ok(BlockInfo::default()));
    actor.attributes_builder = TestAttributesBuilder {
        attributes: (0..2)
            .map(|_| Err(PipelineErrorKind::Temporary(BuilderError::Custom("retry".into()).into())))
            .collect(),
    };
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);

    step_time(&mut actor).await;
    safe_number.store(8, Ordering::SeqCst);
    step_time(&mut actor).await;
    safe_number.store(9, Ordering::SeqCst);
    assert_eq!(step_time(&mut actor).await, Duration::from_secs(2));
    assert!(actor.state().active);
    assert!(actor.state().recovery_mode);
}

#[tokio::test(start_paused = true)]
async fn acknowledged_block_settles_before_the_lag_gate() {
    use super::super::{ConductorCommittedBlock, InFlightBlock};

    let mut actor = test_actor();
    actor.started = true;
    actor.max_safe_lag = 2;
    actor.in_flight = Some(InFlightBlock::Committed(ConductorCommittedBlock {
        envelope: payload(11),
        parent: head(10),
        tx_count: 0,
    }));
    actor.engine_client.expect_canonicalize_block().times(1).return_once(|_, _| Ok(()));
    actor.engine_client.expect_get_unsafe_head().times(1).return_once(|| Ok(head(11)));
    actor.engine_client.expect_get_safe_head().times(1).return_once(|| Ok(head(8)));
    actor.engine_client.expect_start_build_block().times(0);
    actor.origin_selector.expect_next_l1_origin().times(0);
    actor.unsafe_payload_gossip_client.expect_has_capacity().return_const(true);
    actor
        .unsafe_payload_gossip_client
        .expect_schedule_execution_payload_gossip()
        .times(1)
        .return_once(|_| Ok(()));

    step_time(&mut actor).await;
    assert!(actor.in_flight.is_none());
    assert!(actor.state().active);
}
