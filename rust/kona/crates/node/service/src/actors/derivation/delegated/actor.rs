use crate::{
    DerivationActorRequest, EngineClientError, EngineContext, EngineHandle, NodeActor,
    actors::derivation::{DerivationDelegateProvider, DerivationError},
};
use alloy_primitives::BlockHash;
use async_trait::async_trait;
use kona_derive::ChainProvider;
use kona_engine::{EnginePhase, EngineState};
use kona_protocol::{BlockInfo, SyncStatus};
use kona_rpc::DerivationStatus;
use thiserror::Error;
use tokio::{
    select,
    sync::{mpsc, watch},
    time,
};

/// The [`NodeActor`] for the delegate derivation sub-routine.
///
/// This actor is responsible for receiving messages from [`NodeActor`]s and polls
/// an external derivation delegation provider for derivation state. It validates
/// the canonicality of the L1 information associated with delegated derivation
/// results against the canonical L1 chain before forwarding updates.
///
/// Once validated, the actor sends the derived safe and finalized L2 info
/// to the [`NodeActor`] responsible for the execution sub-routine.
#[derive(Debug)]
pub struct DelegateDerivationActor<DelegateProvider, L1Provider>
where
    DelegateProvider: DerivationDelegateProvider,
    L1Provider: ChainProvider,
{
    /// The channel on which all inbound requests are received by the [`DelegateDerivationActor`].
    inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
    /// The Engine client used to interact with the engine.
    engine_client: EngineHandle,
    engine_state: watch::Receiver<EngineState>,
    status: watch::Sender<DerivationStatus>,
    observed_epoch: u64,

    /// Derivation delegate provider.
    derivation_delegate_provider: DelegateProvider,
    /// L1 provider for validating L1 info for derivation delegation.
    l1_provider: L1Provider,

    /// Ticker driving periodic polls of the derivation delegate provider.
    delegated_derivation_ticker: time::Interval,
}

impl<DelegateProvider, L1Provider> DelegateDerivationActor<DelegateProvider, L1Provider>
where
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + 'static,
{
    /// Creates a new instance of the [`DelegateDerivationActor`].
    pub fn new(
        engine_client: EngineHandle,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        derivation_delegate_provider: DelegateProvider,
        l1_provider: L1Provider,
        status: watch::Sender<DerivationStatus>,
    ) -> Self {
        let mut delegated_derivation_ticker =
            time::interval(Self::DERIVATION_DELEGATE_POLL_INTERVAL);
        delegated_derivation_ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        Self {
            inbound_request_rx,
            engine_state: engine_client.subscribe(),
            observed_epoch: engine_client.state().epoch,
            status,
            engine_client,
            derivation_delegate_provider,
            l1_provider,
            delegated_derivation_ticker,
        }
    }
}

#[async_trait]
impl<DelegateProvider, L1Provider> NodeActor
    for DelegateDerivationActor<DelegateProvider, L1Provider>
where
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + Send + 'static,
{
    type Error = DerivationError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        self.observe_engine();
        select! {
            biased;

            req = self.inbound_request_rx.recv() => {
                let request = req.ok_or_else(|| {
                    error!(
                        target: "derivation",
                        "DerivationActor inbound request receiver closed unexpectedly",
                    );
                    DerivationError::RequestReceiveFailed
                })?;
                let DerivationActorRequest::ProcessL1HeadUpdateRequest(_) = request;
                trace!(target: "derivation", "Delegation polls upstream state independently of L1 notifications");
                Ok(())
            }
            result = self.engine_state.changed() => {
                result.map_err(|_| EngineClientError::Closed)?;
                self.observe_engine();
                Ok(())
            }
            _ = self.delegated_derivation_ticker.tick(),
            if self.engine_client.state().phase == EnginePhase::Ready => {
                let state = self.engine_client.state();
                let epoch = state.epoch;
                let mut engine_state = self.engine_state.clone();
                select! {
                    biased;
                    result = super::super::actor::context_changed(&mut engine_state, epoch) => {
                        result?;
                        self.observe_engine();
                        Ok(())
                    },
                    result = self.fetch_and_apply_delegate_safe_head(state) => result,
                }
            }
        }
    }
}

impl<DelegateProvider, L1Provider> DelegateDerivationActor<DelegateProvider, L1Provider>
where
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + 'static,
{
    fn observe_engine(&mut self) {
        let state = *self.engine_state.borrow_and_update();
        if state.epoch != self.observed_epoch || state.phase != EnginePhase::Ready {
            self.status.send_if_modified(|status| status.current_l1.take().is_some());
            self.observed_epoch = state.epoch;
        }
    }

    /// Hardcoded poll interval for Derivation Delegation
    const DERIVATION_DELEGATE_POLL_INTERVAL: std::time::Duration =
        std::time::Duration::from_secs(4);

    /// Validates a single L1 block height and hash against the canonical L1 chain.
    async fn validate_l1_block(
        &mut self,
        context: &str,
        l1_block_number: u64,
        expected_hash: BlockHash,
    ) -> Result<(), DerivationDelegationError> {
        let block = self
            .l1_provider
            .block_info_by_number(l1_block_number)
            .await
            .map_err(|e| DerivationDelegationError::L1Provider(e.to_string()))?;

        if block.hash != expected_hash {
            return Err(DerivationDelegationError::L1ValidationFailed {
                context: context.to_string(),
                number: l1_block_number,
                expected: expected_hash,
                actual: block.hash,
            });
        }

        Ok(())
    }

    /// Verifies that the L1 info reported by the derivation delegate
    /// are consistent with canonical L1 chain.
    async fn validate_sync_status(&mut self, v: &SyncStatus) -> bool {
        let current = (v.current_l1 != BlockInfo::default()).then_some(v.current_l1.id());
        let checks = [
            ("L1 Origin of Safe L2", Some(v.safe_l2.l1_origin)),
            ("L1 Origin of Finalized L2", Some(v.finalized_l2.l1_origin)),
            ("Current L1", current),
        ];
        for (context, block) in checks {
            let Some(block) = block else { continue };
            let number = block.number;
            let hash = block.hash;
            if let Err(err) = self.validate_l1_block(context, number, hash).await {
                warn!(
                    target: "derivation",
                    context = context,
                    error = %err,
                    "L1 inconsistency detected at sync status from delegate"
                );
                return false;
            }
        }
        true
    }

    /// Fetches, validates, and applies sync status from the derivation delegate.
    async fn fetch_and_apply_delegate_safe_head(
        &mut self,
        state: EngineState,
    ) -> Result<(), DerivationError> {
        let sync_status = match self.derivation_delegate_provider.fetch_sync_status().await {
            Ok(status) => status,
            Err(_) => {
                warn!(target: "derivation", "Failed to fetch sync status from delegate");
                return Ok(());
            }
        };

        if !self.validate_sync_status(&sync_status).await {
            // Validation failures here are expected to be transient, so we skip processing
            // this sync status and continue delegating derivation instead of treating it as
            // fatal.
            return Ok(());
        }

        let latest = self.engine_client.state();
        if latest.epoch != state.epoch || latest.phase != EnginePhase::Ready {
            self.observe_engine();
            return Ok(());
        }
        // Validated source progress can advance while local safe is unchanged or import retries.
        let current_l1 =
            (sync_status.current_l1 != BlockInfo::default()).then_some(sync_status.current_l1);
        self.status.send_if_modified(|status| {
            std::mem::replace(&mut status.current_l1, current_l1) != current_l1
        });

        let context = EngineContext { epoch: state.epoch, parent: state.sync_state.safe_head() };
        let result = self
            .engine_client
            .follow_safe(context, sync_status.safe_l2, sync_status.finalized_l2)
            .await;
        match result {
            Ok(_) => {}
            Err(
                EngineClientError::Temporary(_) |
                EngineClientError::Stale |
                EngineClientError::Rejected(_),
            ) => return Ok(()),
            Err(error) => return Err(error.into()),
        }
        debug!(
            target: "derivation",
            safe_l2 = ?sync_status.safe_l2,
            finalized_l2 = ?sync_status.finalized_l2,
            "Accepted delegated safe update and finality claim"
        );

        Ok(())
    }
}

#[derive(Error, Debug)]
enum DerivationDelegationError {
    /// The L1 provider returned an error (network, RPC, etc.)
    #[error("L1 provider error: {0}")]
    L1Provider(String),

    /// The hash provided by the derivation delegation does not match the canonical chain.
    #[error("L1 inconsistency in {context} at block {number}: expected {expected}, got {actual}")]
    L1ValidationFailed { context: String, number: u64, expected: BlockHash, actual: BlockHash },
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::actors::{derivation::DerivationDelegateClientError, engine::handle::Message};
    use alloy_primitives::B256;
    use kona_derive::test_utils::TestChainProvider;
    use kona_protocol::L2BlockInfo;
    use std::{
        collections::VecDeque,
        sync::{Arc, Mutex},
    };
    use tokio::sync::Notify;

    #[derive(Debug)]
    struct Statuses {
        values: Mutex<VecDeque<SyncStatus>>,
        blocked: Option<Arc<Notify>>,
    }
    #[async_trait]
    impl DerivationDelegateProvider for Statuses {
        async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateClientError> {
            if let Some(started) = &self.blocked {
                started.notify_one();
                std::future::pending::<()>().await;
            }
            Ok(self.values.lock().unwrap().pop_front().expect("unexpected delegate poll"))
        }
    }

    fn block(number: u64) -> BlockInfo {
        BlockInfo { number, hash: B256::repeat_byte(number as u8 + 1), ..Default::default() }
    }
    fn status(current_l1: BlockInfo) -> SyncStatus {
        let safe_l2 = L2BlockInfo { l1_origin: block(1).id(), ..Default::default() };
        SyncStatus {
            current_l1,
            current_l1_finalized: block(1),
            head_l1: block(3),
            safe_l1: block(1),
            finalized_l1: block(1),
            unsafe_l2: safe_l2,
            safe_l2,
            finalized_l2: safe_l2,
            local_safe_l2: safe_l2,
        }
    }
    fn chain() -> TestChainProvider {
        let mut chain = TestChainProvider::default();
        for number in 0..=3 {
            chain.insert_block(number, block(number));
        }
        chain
    }

    #[tokio::test]
    async fn every_validated_poll_publishes_progress_before_an_unchanged_safe_import_retries() {
        let state = EngineState { phase: EnginePhase::Ready, ..Default::default() };
        let (engine, mut owner) = EngineHandle::test_channel(state);
        let (_l1, requests) = mpsc::channel(1);
        let (published, progress) = watch::channel(DerivationStatus::default());
        let provider = Statuses {
            values: Mutex::new(VecDeque::from([
                status(block(2)),
                status(block(0)),
                status(block(3)),
            ])),
            blocked: None,
        };
        let mut actor =
            DelegateDerivationActor::new(engine, requests, provider, chain(), published);
        for number in [2, 0, 3] {
            let (result, ()) =
                tokio::join!(actor.fetch_and_apply_delegate_safe_head(state), async {
                    let Message::FollowSafe(_, safe, finalized, reply) =
                        owner.commands.recv().await.unwrap()
                    else {
                        panic!("expected follow safe")
                    };
                    assert_eq!(safe, status(block(number)).safe_l2);
                    assert_eq!(finalized, status(block(number)).finalized_l2);
                    assert_eq!(progress.borrow().current_l1, Some(block(number)));
                    reply.send(Err(EngineClientError::Temporary("EL unavailable".into()))).unwrap();
                });
            result.unwrap();
        }
        assert_eq!(owner.state.borrow().sync_state.safe_head(), L2BlockInfo::default());
    }

    #[tokio::test]
    async fn unchanged_safe_transports_each_validated_finality_claim_without_waiting_for_finalization()
     {
        use crate::Consolidated;
        use kona_engine::{Consolidation, EngineSyncStateUpdate};
        let first = L2BlockInfo { block_info: block(0), l1_origin: block(0).id(), seq_num: 0 };
        let second = L2BlockInfo { block_info: block(1), l1_origin: block(1).id(), seq_num: 0 };
        let safe = L2BlockInfo { block_info: block(2), l1_origin: block(1).id(), seq_num: 1 };
        let mut state = EngineState {
            epoch: 3,
            phase: EnginePhase::Ready,
            pending_safe_head: safe,
            ..Default::default()
        };
        state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(safe),
            safe_head: Some(safe),
            local_safe_head: Some(safe),
            finalized_head: Some(first),
        });
        let (engine, mut owner) = EngineHandle::test_channel(state);
        let (_l1, requests) = mpsc::channel(1);
        let provider = Statuses {
            values: Mutex::new(
                [first, second]
                    .map(|finalized_l2| SyncStatus {
                        safe_l2: safe,
                        finalized_l2,
                        ..status(block(3))
                    })
                    .into(),
            ),
            blocked: None,
        };
        let mut actor = DelegateDerivationActor::new(
            engine,
            requests,
            provider,
            chain(),
            watch::channel(DerivationStatus::default()).0,
        );
        for claim in [first, second] {
            let (result, ()) =
                tokio::join!(actor.fetch_and_apply_delegate_safe_head(state), async {
                    let Message::FollowSafe(context, received_safe, finalized, reply) =
                        owner.commands.recv().await.unwrap()
                    else {
                        panic!("expected trusted safe/finality pair")
                    };
                    assert_eq!(context, EngineContext { epoch: 3, parent: safe });
                    assert_eq!(received_safe, safe);
                    assert_eq!(finalized, claim);
                    reply
                        .send(Ok(Consolidated { state, outcome: Consolidation::Applied }))
                        .unwrap();
                });
            result.unwrap();
            assert_eq!(owner.state.borrow().sync_state.finalized_head(), first);
            assert!(owner.commands.try_recv().is_err());
        }
    }

    #[tokio::test]
    async fn unknown_progress_stays_unknown_and_invalid_progress_never_publishes() {
        let state = EngineState { phase: EnginePhase::Ready, ..Default::default() };
        let (engine, mut owner) = EngineHandle::test_channel(state);
        let (_l1, requests) = mpsc::channel(1);
        let (published, progress) = watch::channel(DerivationStatus { current_l1: Some(block(2)) });
        let invalid = BlockInfo { hash: B256::repeat_byte(9), ..block(3) };
        let provider = Statuses {
            values: Mutex::new(VecDeque::from([status(BlockInfo::default()), status(invalid)])),
            blocked: None,
        };
        let mut actor =
            DelegateDerivationActor::new(engine, requests, provider, chain(), published);
        let (result, ()) = tokio::join!(actor.fetch_and_apply_delegate_safe_head(state), async {
            let Message::FollowSafe(_, _, _, reply) = owner.commands.recv().await.unwrap() else {
                panic!("expected follow safe")
            };
            reply.send(Err(EngineClientError::Temporary("retry".into()))).unwrap();
        });
        result.unwrap();
        assert_eq!(progress.borrow().current_l1, None); // No fake default genesis lookup.
        actor.status.send_replace(DerivationStatus { current_l1: Some(block(2)) });
        actor.fetch_and_apply_delegate_safe_head(state).await.unwrap();
        assert_eq!(progress.borrow().current_l1, Some(block(2)));
        assert!(owner.commands.try_recv().is_err());
    }

    #[tokio::test]
    async fn a_reset_cancels_an_old_delegate_poll_and_clears_its_progress() {
        let state = EngineState { phase: EnginePhase::Ready, ..Default::default() };
        let (engine, owner) = EngineHandle::test_channel(state);
        let (_l1, requests) = mpsc::channel(1);
        let (published, progress) = watch::channel(DerivationStatus { current_l1: Some(block(2)) });
        let started = Arc::new(Notify::new());
        let provider =
            Statuses { values: Mutex::new(VecDeque::new()), blocked: Some(started.clone()) };
        let mut actor =
            DelegateDerivationActor::new(engine, requests, provider, chain(), published);
        let (result, ()) = tokio::join!(actor.step(), async {
            started.notified().await;
            owner.state.send_replace(EngineState {
                epoch: 1,
                phase: EnginePhase::Resetting,
                ..state
            });
        });
        result.unwrap();
        assert_eq!(progress.borrow().current_l1, None);
    }
}
