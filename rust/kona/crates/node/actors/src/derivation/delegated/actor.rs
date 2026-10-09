use crate::{
    DerivationActorRequest, DerivationEngineClient, DerivationStatus, NodeActor,
    derivation::{DerivationDelegateProvider, DerivationError},
};
use alloy_primitives::BlockHash;
use async_trait::async_trait;
use kona_derive::ChainProvider;
use kona_engine::FinalizeBlockId;
use kona_protocol::{L2BlockInfo, SyncStatus};
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
pub struct DelegateDerivationActor<DerivationEngineClient_, DelegateProvider, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient,
    DelegateProvider: DerivationDelegateProvider,
    L1Provider: ChainProvider,
{
    /// The channel on which all inbound requests are received by the [`DelegateDerivationActor`].
    inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// Validated progress published to RPC readers.
    status: watch::Sender<DerivationStatus>,

    /// Derivation delegate provider.
    derivation_delegate_provider: DelegateProvider,
    /// L1 provider for validating L1 info for derivation delegation.
    l1_provider: L1Provider,

    /// The engine's L2 safe head, according to updates from the Engine.
    engine_l2_safe_head: L2BlockInfo,
    /// Whether the engine sync has completed. This will only ever go from false -> true.
    has_engine_sync_completed: bool,
    /// Ticker driving periodic polls of the derivation delegate provider.
    delegated_derivation_ticker: time::Interval,
}

impl<DerivationEngineClient_, DelegateProvider, L1Provider>
    DelegateDerivationActor<DerivationEngineClient_, DelegateProvider, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + 'static,
{
    /// Creates a new instance of the [`DelegateDerivationActor`].
    pub fn new(
        engine_client: DerivationEngineClient_,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        derivation_delegate_provider: DelegateProvider,
        l1_provider: L1Provider,
    ) -> Self {
        let mut delegated_derivation_ticker =
            time::interval(Self::DERIVATION_DELEGATE_POLL_INTERVAL);
        delegated_derivation_ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        let (status, _) = watch::channel(DerivationStatus::default());
        Self {
            status,
            inbound_request_rx,
            engine_client,
            derivation_delegate_provider,
            l1_provider,
            engine_l2_safe_head: L2BlockInfo::default(),
            has_engine_sync_completed: false,
            delegated_derivation_ticker,
        }
    }

    /// Subscribes to the actor’s validated derivation progress.
    pub fn state_receiver(&self) -> watch::Receiver<DerivationStatus> {
        self.status.subscribe()
    }
}

#[async_trait]
impl<DerivationEngineClient_, DelegateProvider, L1Provider> NodeActor
    for DelegateDerivationActor<DerivationEngineClient_, DelegateProvider, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + Send + 'static,
{
    type Error = DerivationError;

    async fn step(&mut self) -> Result<(), Self::Error> {
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
                self.handle_derivation_delegation_actor_request(request).await
            }
            _ = self.delegated_derivation_ticker.tick(),
            if self.has_engine_sync_completed => {
                self.fetch_and_apply_delegate_safe_head().await
            }
        }
    }
}

impl<DerivationEngineClient_, DelegateProvider, L1Provider>
    DelegateDerivationActor<DerivationEngineClient_, DelegateProvider, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + 'static,
{
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
        let checks = [
            ("L1 Origin of Safe L2", v.safe_l2.l1_origin.number, v.safe_l2.l1_origin.hash),
            (
                "L1 Origin of Finalized L2",
                v.finalized_l2.l1_origin.number,
                v.finalized_l2.l1_origin.hash,
            ),
            ("Current L1", v.current_l1.number, v.current_l1.hash),
        ];
        for (context, number, hash) in checks {
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
    async fn fetch_and_apply_delegate_safe_head(&mut self) -> Result<(), DerivationError> {
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

        self.status.send_replace(DerivationStatus { current_l1: Some(sync_status.current_l1) });

        self.engine_client
            .send_safe_l2_signal(sync_status.safe_l2.into())
            .await
            .map_err(|e| DerivationError::Sender(Box::new(e)))?;

        // Delegated polling supplies `(number, hash)`. Carry the hash through so the engine
        // finalizes the specific block we were asked to, not whatever it happens to have at the
        // same height.
        self.engine_client
            .send_finalized_l2_block(FinalizeBlockId::ByHash(
                sync_status.finalized_l2.block_info.id(),
            ))
            .await
            .map_err(|e| DerivationError::Sender(Box::new(e)))?;

        debug!(
            target: "derivation",
            safe_l2 = ?sync_status.safe_l2,
            finalized_l2 = ?sync_status.finalized_l2,
            "Processed sync status from delegate"
        );

        Ok(())
    }

    async fn handle_derivation_delegation_actor_request(
        &mut self,
        request_type: DerivationActorRequest,
    ) -> Result<(), DerivationError> {
        match request_type {
            DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(safe_head) => {
                debug!(target: "derivation", safe_head = ?*safe_head, "Received safe head from engine.");
                self.engine_l2_safe_head = *safe_head;
            }
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(safe_head) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                self.engine_l2_safe_head = *safe_head;
                self.has_engine_sync_completed = true;
            }
            DerivationActorRequest::ProcessEngineSignalRequest(_) => {
                debug!(target: "derivation", "Ignoring request while derivation delegation: {:?}", request_type);
            }
        }
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
    use crate::derivation::{
        DerivationDelegateClientError, engine_client::MockDerivationEngineClient,
    };
    use kona_derive::test_utils::TestChainProvider;
    use kona_protocol::BlockInfo;
    use std::{collections::VecDeque, sync::Mutex};

    struct TestDelegate(Mutex<VecDeque<SyncStatus>>);

    #[async_trait]
    impl DerivationDelegateProvider for TestDelegate {
        async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateClientError> {
            Ok(self.0.lock().unwrap().pop_front().unwrap())
        }
    }

    #[tokio::test(start_paused = true)]
    async fn publishes_only_validated_delegate_progress() {
        let current = BlockInfo {
            number: 7,
            hash: alloy_primitives::B256::repeat_byte(1),
            ..Default::default()
        };
        // Fill the delegate's unused sync fields with zero values.
        let status = SyncStatus {
            current_l1: current,
            current_l1_finalized: BlockInfo::default(),
            head_l1: BlockInfo::default(),
            safe_l1: BlockInfo::default(),
            finalized_l1: BlockInfo::default(),
            unsafe_l2: L2BlockInfo::default(),
            local_safe_l2: L2BlockInfo::default(),
            safe_l2: L2BlockInfo::default(),
            finalized_l2: L2BlockInfo::default(),
        };
        let mut invalid = status.clone();
        invalid.current_l1.hash = alloy_primitives::B256::repeat_byte(2);
        let delegate = TestDelegate(Mutex::new(VecDeque::from([invalid.clone(), status, invalid])));
        let mut provider = TestChainProvider::default();
        provider.insert_block(0, BlockInfo::default());
        provider.insert_block(current.number, current);
        let mut engine = MockDerivationEngineClient::new();
        engine.expect_send_safe_l2_signal().times(1).returning(|_| Ok(()));
        engine.expect_send_finalized_l2_block().times(1).returning(|_| Ok(()));
        let (tx, rx) = mpsc::channel(1);
        let mut actor = DelegateDerivationActor::new(engine, rx, delegate, provider);
        let progress = actor.state_receiver();
        tx.send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::default()))
            .await
            .unwrap();
        actor.step().await.unwrap();
        actor.step().await.unwrap();
        assert_eq!(progress.borrow().current_l1, None);
        actor.step().await.unwrap();
        assert_eq!(progress.borrow().current_l1, Some(current));
        actor.step().await.unwrap();
        assert_eq!(progress.borrow().current_l1, Some(current));
    }
}
