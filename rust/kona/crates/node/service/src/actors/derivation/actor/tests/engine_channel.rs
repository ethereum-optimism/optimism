//! The engine's requests reach derivation on their own channel, which the engine never waits on.

use super::*;
use crate::{
    QueuedDerivationEngineClient,
    actors::engine::tests::{EngineWithFullInbox, engine_with_full_inbox, until_other_tasks_park},
};
use std::time::Duration;
use tokio::time;

/// Derivation is blocked sending attributes into the engine's full inbox, and the L1 watcher has
/// filled derivation's inbox. The engine still sends derivation its safe head and takes the next
/// request from its inbox, which lets derivation's send through.
#[tokio::test(start_paused = true)]
async fn the_engine_serves_its_inbox_while_derivation_waits_on_it() {
    let (engine_request_tx, engine_request_rx) = mpsc::unbounded_channel();
    let EngineWithFullInbox { actor: mut engine, requests, l1, l2 } =
        engine_with_full_inbox(engine_request_tx.clone()).await;
    let (l1_watcher, l1_watcher_rx) = mpsc::channel(1);
    let mut derivation = DerivationActor::new(
        QueuedDerivationEngineClient::new(requests.clone()),
        l1_watcher_rx,
        engine_request_rx,
        QueuedPipeline {
            attributes: VecDeque::from([derived_attributes()]),
            derived_from: l1_block(DERIVED_FROM_L1),
            rollup_config: Arc::new(RollupConfig::default()),
            signal_error: None,
        },
        None,
        l1_block(0).id(),
        watch::channel(None).0,
    );
    engine_request_tx
        .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(l2_block(
            RESUME_HEAD,
        ))))
        .unwrap();

    // Derivation sees the completed EL sync, derives, and blocks sending the attributes.
    let derivation_step = tokio::spawn(async move {
        derivation.step().await.unwrap();
        derivation
    });
    until_other_tasks_park().await;
    assert!(!derivation_step.is_finished(), "the engine inbox must be full");
    l1_watcher
        .try_send(DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::new(l1_block(
            DERIVED_FROM_L1 + 1,
        ))))
        .unwrap();

    time::timeout(Duration::from_secs(10), engine.step())
        .await
        .expect("the engine must not wait on derivation")
        .unwrap();

    let derivation = time::timeout(Duration::from_secs(10), derivation_step)
        .await
        .expect("the engine took a request, so the attributes fit")
        .unwrap();
    assert!(derivation.attributes_awaiting_confirmation.is_some());
    assert_eq!(requests.capacity(), 0, "the attributes are in the engine inbox");
    l1.assert_finished();
    l2.assert_finished();
}

/// A request on the engine's channel is handled like one on the inbox, and ahead of an earlier
/// one on the inbox.
#[tokio::test]
async fn a_request_on_the_engine_channel_is_handled_first() {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());
    node.request_tx
        .send(DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::new(l1_block(
            DERIVED_FROM_L1,
        ))))
        .await
        .unwrap();
    node.engine_request_tx
        .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(l2_block(
            RESUME_HEAD,
        ))))
        .unwrap();

    node.actor.step().await.unwrap();

    assert_eq!(
        node.actor.derivation_state_machine.current_state(),
        DerivationState::AwaitingSafeHeadConfirmation
    );
    assert!(node.actor.attributes_awaiting_confirmation.is_some());
    assert_eq!(node.request_tx.capacity(), 0, "the L1 head update is still queued");
}

/// A closed engine channel means the engine is gone, which stops the actor like a closed inbox.
#[tokio::test]
async fn a_closed_engine_channel_stops_the_actor() {
    let mut node = RecordingActor::new(MockDerivationEngineClient::new());

    drop(node.engine_request_tx);

    assert!(matches!(node.actor.step().await, Err(DerivationError::RequestReceiveFailed)));
}
