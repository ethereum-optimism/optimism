use crate::{
    CancellableContext, DerivationActorRequest, DerivationEngineClient, NodeActor,
    actors::derivation::{DerivationDelegateClient, DerivationError},
};
use alloy_primitives::BlockHash;
use async_trait::async_trait;
use kona_protocol::{L2BlockInfo, SyncStatus};
use kona_providers_alloy::AlloyChainProvider;
use thiserror::Error;
use tokio::{select, sync::mpsc, time};
use tokio_util::sync::{CancellationToken, WaitForCancellationFuture};

/// The [NodeActor] for the delegate derivation sub-routine.
///
/// This actor is responsible for receiving messages from [NodeActor]s and polls
/// an external derivation delegation provider for derivation state. It validates
/// the canonicality of the L1 information associated with delegated derivation
/// results against the canonical L1 chain before forwarding updates.
///
/// Once validated, the actor sends the derived safe and finalized L2 info
/// to the [NodeActor] responsible for the execution sub-routine.
#[derive(Debug)]
pub struct DelegateDerivationActor<DerivationEngineClient_>
where
    DerivationEngineClient_: DerivationEngineClient,
{
    /// The cancellation token, shared between all tasks.
    cancellation_token: CancellationToken,
    /// The channel on which all inbound requests are received by the [`DelegateDerivationActor`].
    inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// Derivation delegate provider.
    derivation_delegate_provider: DerivationDelegateClient,
    /// L1 provider for validating L1 info for derivation delegation.
    l1_provider: AlloyChainProvider,

    /// The engine's L2 safe head, according to updates from the Engine.
    engine_l2_safe_head: L2BlockInfo,
    /// Whether the engine sync has completed. This will only ever go from false -> true.
    has_engine_sync_completed: bool,
}

impl<DerivationEngineClient_> CancellableContext
    for DelegateDerivationActor<DerivationEngineClient_>
where
    DerivationEngineClient_: DerivationEngineClient,
{
    fn cancelled(&self) -> WaitForCancellationFuture<'_> {
        self.cancellation_token.cancelled()
    }
}

impl<DerivationEngineClient_> DelegateDerivationActor<DerivationEngineClient_>
where
    DerivationEngineClient_: DerivationEngineClient,
{
    /// Creates a new instance of the [DelegateDerivationActor].
    pub fn new(
        engine_client: DerivationEngineClient_,
        cancellation_token: CancellationToken,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        derivation_delegate_provider: DerivationDelegateClient,
        l1_provider: AlloyChainProvider,
    ) -> Self {
        Self {
            cancellation_token,
            inbound_request_rx,
            engine_client,
            derivation_delegate_provider,
            l1_provider,
            engine_l2_safe_head: L2BlockInfo::default(),
            has_engine_sync_completed: false,
        }
    }
}

#[async_trait]
impl<DerivationEngineClient_> NodeActor for DelegateDerivationActor<DerivationEngineClient_>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
{
    type Error = DerivationError;
    type StartData = ();

    async fn start(mut self, _: Self::StartData) -> Result<(), Self::Error> {
        self.start_delegate_derivation().await
    }
}

impl<DerivationEngineClient_> DelegateDerivationActor<DerivationEngineClient_>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
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
        use kona_derive::ChainProvider;

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

        self.engine_client
            .send_safe_l2_signal(sync_status.safe_l2.into())
            .await
            .map_err(|e| DerivationError::Sender(Box::new(e)))?;

        self.engine_client
            .send_finalized_l2_block(sync_status.finalized_l2.block_info.number)
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

    async fn start_delegate_derivation(mut self) -> Result<(), DerivationError> {
        info!(target: "derivation", "Starting derivation with delegation");
        let mut delegated_derivation_ticker =
            time::interval(Self::DERIVATION_DELEGATE_POLL_INTERVAL);
        delegated_derivation_ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        loop {
            select! {
                biased;

                _ = self.cancellation_token.cancelled() => {
                    info!(
                        target: "derivation",
                        "Received shutdown signal. Exiting derivation task."
                    );
                    return Ok(());
                }
                req = self.inbound_request_rx.recv() => {
                    let Some(request_type) = req else {
                        error!(target: "derivation", "DerivationActor inbound request receiver closed unexpectedly");
                        self.cancellation_token.cancel();
                        return Err(DerivationError::RequestReceiveFailed);
                    };

                    self.handle_derivation_delegation_actor_request(request_type).await?;
                }
                _ = delegated_derivation_ticker.tick(),
                if self.has_engine_sync_completed => {
                    self.fetch_and_apply_delegate_safe_head().await?;
                }
            }
        }
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
            DerivationActorRequest::ProcessEngineSignalRequest(_) |
            DerivationActorRequest::ProcessFinalizedL1Block(_) |
            DerivationActorRequest::ProcessL1HeadUpdateRequest(_) => {
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
