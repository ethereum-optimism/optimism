use crate::{
    derivation::{
        ActorError, Builder, DerivationDelegateProvider, DerivationEngineClient, DerivationStatus,
        Message,
    },
    engine::FinalizeBlockId,
};
use alloy_primitives::BlockHash;
use kona_derive::ChainProvider;
use kona_protocol::SyncStatus;
use std::future::Future;
use thiserror::Error;
use tokio::{
    select,
    sync::{mpsc, watch},
    time,
};

impl Builder {
    /// Supplies the dependencies of the derivation actor that follows an external derivation
    /// delegate, and produces its lifetime future without spawning it.
    ///
    /// Runtime work begins when the future is polled. Dropping the future stops the actor.
    pub fn build_delegated<DerivationEngineClient_, DelegateProvider, L1Provider>(
        self,
        engine_client: DerivationEngineClient_,
        derivation_delegate_provider: DelegateProvider,
        l1_provider: L1Provider,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static
    where
        DerivationEngineClient_: DerivationEngineClient + 'static,
        DelegateProvider: DerivationDelegateProvider + 'static,
        L1Provider: ChainProvider + Send + 'static,
    {
        let Self { handle: _, messages_rx, status_tx } = self;
        async move {
            Actor::new(
                engine_client,
                messages_rx,
                status_tx,
                derivation_delegate_provider,
                l1_provider,
            )
            .run()
            .await
        }
    }
}

/// Polls an external derivation delegate for its sync status, validates the status's L1 blocks
/// against the canonical L1 chain, and sends the validated safe and finalized L2 heads to the
/// engine actor.
#[derive(Debug)]
struct Actor<DerivationEngineClient_, DelegateProvider, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient,
    DelegateProvider: DerivationDelegateProvider,
    L1Provider: ChainProvider,
{
    /// Receives messages from handles.
    inbound_request_rx: mpsc::Receiver<Message>,
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// Validated progress published to RPC readers.
    status: watch::Sender<DerivationStatus>,

    /// Derivation delegate provider.
    derivation_delegate_provider: DelegateProvider,
    /// L1 provider for validating L1 info for derivation delegation.
    l1_provider: L1Provider,

    /// Whether the engine sync has completed. This will only ever go from false -> true.
    has_engine_sync_completed: bool,
    /// Ticker driving periodic polls of the derivation delegate provider.
    delegated_derivation_ticker: time::Interval,
}

impl<DerivationEngineClient_, DelegateProvider, L1Provider>
    Actor<DerivationEngineClient_, DelegateProvider, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + Send + 'static,
{
    /// Creates the actor. Its poll timer needs a Tokio runtime.
    fn new(
        engine_client: DerivationEngineClient_,
        inbound_request_rx: mpsc::Receiver<Message>,
        status: watch::Sender<DerivationStatus>,
        derivation_delegate_provider: DelegateProvider,
        l1_provider: L1Provider,
    ) -> Self {
        let mut delegated_derivation_ticker =
            time::interval(Self::DERIVATION_DELEGATE_POLL_INTERVAL);
        delegated_derivation_ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        Self {
            status,
            inbound_request_rx,
            engine_client,
            derivation_delegate_provider,
            l1_provider,
            has_engine_sync_completed: false,
            delegated_derivation_ticker,
        }
    }

    async fn run(mut self) -> Result<(), ActorError> {
        loop {
            self.step().await?;
        }
    }

    /// Handles the next message, or polls the delegate once EL sync has completed.
    async fn step(&mut self) -> Result<(), ActorError> {
        select! {
            biased;

            req = self.inbound_request_rx.recv() => {
                let request = req.ok_or_else(|| {
                    error!(
                        target: "derivation",
                        "Derivation request receiver closed unexpectedly",
                    );
                    ActorError::RequestReceiveFailed
                })?;
                self.handle_message(request).await
            }
            _ = self.delegated_derivation_ticker.tick(),
            if self.has_engine_sync_completed => {
                self.fetch_and_apply_delegate_safe_head().await
            }
        }
    }
}

impl<DerivationEngineClient_, DelegateProvider, L1Provider>
    Actor<DerivationEngineClient_, DelegateProvider, L1Provider>
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
    async fn fetch_and_apply_delegate_safe_head(&mut self) -> Result<(), ActorError> {
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

        self.engine_client.send_delegated_safe_head(sync_status.safe_l2).await?;

        // Delegated polling supplies `(number, hash)`. Carry the hash through so the engine
        // finalizes the specific block we were asked to, not whatever it happens to have at the
        // same height.
        self.engine_client
            .send_finalized_l2_block(FinalizeBlockId::ByHash(
                sync_status.finalized_l2.block_info.id(),
            ))
            .await?;

        debug!(
            target: "derivation",
            safe_l2 = ?sync_status.safe_l2,
            finalized_l2 = ?sync_status.finalized_l2,
            "Processed sync status from delegate"
        );

        Ok(())
    }

    async fn handle_message(&mut self, message: Message) -> Result<(), ActorError> {
        match message {
            Message::SafeHead(safe_head) => {
                debug!(target: "derivation", safe_head = ?*safe_head, "Received safe head from engine.");
            }
            Message::ElSyncCompleted(_) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                self.has_engine_sync_completed = true;
            }
            Message::Signal(signal) => {
                debug!(target: "derivation", ?signal, "Ignoring signal while delegating derivation");
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
    use async_trait::async_trait;
    use kona_derive::test_utils::TestChainProvider;
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use std::{collections::VecDeque, sync::Mutex};

    struct TestDelegate(Mutex<VecDeque<SyncStatus>>);

    #[async_trait]
    impl DerivationDelegateProvider for TestDelegate {
        async fn fetch_sync_status(&self) -> Result<SyncStatus, DerivationDelegateClientError> {
            Ok(self.0.lock().unwrap().pop_front().unwrap())
        }
    }

    /// Building needs no runtime: the actor creates its poll timer when it starts running.
    #[test]
    fn build_delegated_does_not_require_a_runtime() {
        let builder = Builder::new(crate::derivation::Capacity::try_from(1).unwrap());
        let task = builder.build_delegated(
            MockDerivationEngineClient::new(),
            TestDelegate(Mutex::new(VecDeque::new())),
            TestChainProvider::default(),
        );
        drop(task);
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
        engine.expect_send_delegated_safe_head().times(1).returning(|_| Ok(()));
        engine.expect_send_finalized_l2_block().times(1).returning(|_| Ok(()));
        let (tx, rx) = mpsc::channel(1);
        let (status, progress) = watch::channel(DerivationStatus::default());
        let mut actor = Actor::new(engine, rx, status, delegate, provider);
        tx.send(Message::ElSyncCompleted(Box::default())).await.unwrap();
        actor.step().await.unwrap();
        actor.step().await.unwrap();
        assert_eq!(progress.borrow().current_l1, None);
        actor.step().await.unwrap();
        assert_eq!(progress.borrow().current_l1, Some(current));
        actor.step().await.unwrap();
        assert_eq!(progress.borrow().current_l1, Some(current));
    }
}
