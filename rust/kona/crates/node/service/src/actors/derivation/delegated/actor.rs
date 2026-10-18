use crate::{
    DerivationActorRequest, EngineClientError, EngineContext, EngineHandle, NodeActor,
    actors::derivation::{DerivationDelegateProvider, DerivationError},
};
use alloy_primitives::BlockHash;
use async_trait::async_trait;
use kona_derive::ChainProvider;
use kona_engine::{EnginePhase, EngineState, FinalizationSource};
use kona_protocol::SyncStatus;
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
    ) -> Self {
        let mut delegated_derivation_ticker =
            time::interval(Self::DERIVATION_DELEGATE_POLL_INTERVAL);
        delegated_derivation_ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        Self {
            inbound_request_rx,
            engine_state: engine_client.subscribe(),
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
                match request {
                    DerivationActorRequest::ProcessFinalizedL1Block(_) |
                    DerivationActorRequest::ProcessL1HeadUpdateRequest(_) => {
                        trace!(target: "derivation", "Delegation polls upstream state independently of L1 notifications");
                    }
                }
                Ok(())
            }
            result = self.engine_state.changed() => {
                result.map_err(|_| EngineClientError::Closed)?;
                Ok(())
            }
            _ = self.delegated_derivation_ticker.tick(),
            if self.engine_client.state().phase == EnginePhase::Ready => {
                let state = self.engine_client.state();
                let epoch = state.epoch;
                let mut engine_state = self.engine_state.clone();
                select! {
                    biased;
                    result = super::super::actor::context_changed(&mut engine_state, epoch) => result,
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

        let context = EngineContext { epoch: state.epoch, parent: state.sync_state.safe_head() };
        let result = self.engine_client.follow_safe(context, sync_status.safe_l2).await;
        match result {
            Ok(_) => {}
            Err(
                EngineClientError::Temporary(_) |
                EngineClientError::Stale |
                EngineClientError::Rejected(_),
            ) => return Ok(()),
            Err(error) => return Err(error.into()),
        }
        // Delegation retains its explicit upstream trust; no local inclusion source is invented.
        match self
            .engine_client
            .finalize(
                state.epoch,
                sync_status.finalized_l2.block_info.id(),
                FinalizationSource::Delegated,
            )
            .await
        {
            Ok(()) => {}
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
            "Processed sync status from delegate"
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
