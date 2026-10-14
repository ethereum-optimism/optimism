use crate::{
    derivation::{ActorError, Builder, DerivationDelegateProvider, DerivationStatus, Engine},
    engine::FinalizeBlockId,
};
use alloy_primitives::BlockHash;
use kona_derive::ChainProvider;
use kona_protocol::SyncStatus;
use std::future::Future;
use thiserror::Error;
use tokio::{sync::watch, time};

impl Builder {
    /// Supplies the dependencies of the derivation actor that follows an external derivation
    /// delegate, and produces its lifetime future without spawning it.
    ///
    /// Runtime work begins when the future is polled. Dropping the future stops the actor.
    pub fn build_delegated<Engine_, DelegateProvider, L1Provider>(
        self,
        engine: Engine_,
        derivation_delegate_provider: DelegateProvider,
        l1_provider: L1Provider,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static
    where
        Engine_: Engine + 'static,
        DelegateProvider: DerivationDelegateProvider + 'static,
        L1Provider: ChainProvider + Send + 'static,
    {
        let Self { handle: _, status_tx } = self;
        async move {
            Actor::new(engine, status_tx, derivation_delegate_provider, l1_provider).run().await
        }
    }
}

/// Polls an external derivation delegate for its sync status, validates the status's L1 blocks
/// against the canonical L1 chain, and sends the validated safe and finalized L2 heads to the
/// engine actor.
#[derive(Debug)]
struct Actor<Engine_, DelegateProvider, L1Provider>
where
    Engine_: Engine,
    DelegateProvider: DerivationDelegateProvider,
    L1Provider: ChainProvider,
{
    /// The Engine client used to interact with the engine.
    engine: Engine_,

    /// Validated progress published to RPC readers.
    status: watch::Sender<DerivationStatus>,

    /// Derivation delegate provider.
    derivation_delegate_provider: DelegateProvider,
    /// L1 provider for validating L1 info for derivation delegation.
    l1_provider: L1Provider,

    /// Ticker driving periodic polls of the derivation delegate provider.
    delegated_derivation_ticker: time::Interval,
}

impl<Engine_, DelegateProvider, L1Provider> Actor<Engine_, DelegateProvider, L1Provider>
where
    Engine_: Engine + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + Send + 'static,
{
    /// Creates the actor. Its poll timer needs a Tokio runtime.
    fn new(
        engine: Engine_,
        status: watch::Sender<DerivationStatus>,
        derivation_delegate_provider: DelegateProvider,
        l1_provider: L1Provider,
    ) -> Self {
        let mut delegated_derivation_ticker =
            time::interval(Self::DERIVATION_DELEGATE_POLL_INTERVAL);
        delegated_derivation_ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        Self {
            status,
            engine,
            derivation_delegate_provider,
            l1_provider,
            delegated_derivation_ticker,
        }
    }

    /// Waits for the engine's first reset, which it makes once the execution layer is synced, then
    /// follows the delegate.
    async fn run(mut self) -> Result<(), ActorError> {
        self.engine.next_reset().await?;
        info!(target: "derivation", "Engine reset; following the derivation delegate");
        loop {
            self.delegated_derivation_ticker.tick().await;
            self.fetch_and_apply_delegate_safe_head().await?;
        }
    }
}

impl<Engine_, DelegateProvider, L1Provider> Actor<Engine_, DelegateProvider, L1Provider>
where
    Engine_: Engine + 'static,
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

        self.engine.follow_safe_head(sync_status.safe_l2).await?;

        // Delegated polling supplies `(number, hash)`. Carry the hash through so the engine
        // finalizes the specific block we were asked to, not whatever it happens to have at the
        // same height.
        self.engine
            .finalize(FinalizeBlockId::ByHash(sync_status.finalized_l2.block_info.id()))
            .await?;

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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        derivation::{
            DerivationDelegateClientError,
            engine::{Request, TestEngine},
        },
        engine::EngineState,
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
        let builder = Builder::new();
        let task = builder.build_delegated(
            TestEngine::new([]),
            TestDelegate(Mutex::new(VecDeque::new())),
            TestChainProvider::default(),
        );
        drop(task);
    }

    /// Returns a delegate sync status whose L1 blocks are the given current L1 block and genesis.
    fn sync_status(current: BlockInfo) -> SyncStatus {
        // Fill the delegate's unused sync fields with zero values.
        SyncStatus {
            current_l1: current,
            current_l1_finalized: BlockInfo::default(),
            head_l1: BlockInfo::default(),
            safe_l1: BlockInfo::default(),
            finalized_l1: BlockInfo::default(),
            unsafe_l2: L2BlockInfo::default(),
            local_safe_l2: L2BlockInfo::default(),
            safe_l2: L2BlockInfo::default(),
            finalized_l2: L2BlockInfo::default(),
        }
    }

    /// The actor waits for the engine's first reset before it follows the delegate.
    #[tokio::test(start_paused = true)]
    async fn follows_the_delegate_from_the_engines_first_reset() {
        let current = BlockInfo { number: 7, ..Default::default() };
        let delegate = TestDelegate(Mutex::new(VecDeque::from([sync_status(current)])));
        let mut provider = TestChainProvider::default();
        provider.insert_block(0, BlockInfo::default());
        provider.insert_block(current.number, current);
        let engine = TestEngine::new([Ok(EngineState::default())]);
        let (status, mut progress) = watch::channel(DerivationStatus::default());
        let actor = tokio::spawn(Actor::new(engine.clone(), status, delegate, provider).run());
        time::sleep(time::Duration::from_secs(10)).await;
        assert!(engine.requests().is_empty());
        assert_eq!(progress.borrow().current_l1, None);

        engine.start();
        time::timeout(
            time::Duration::from_secs(10),
            progress.wait_for(|progress| progress.current_l1 == Some(current)),
        )
        .await
        .unwrap()
        .unwrap();
        actor.abort();
        assert!(matches!(
            engine.requests()[..],
            [Request::FollowSafeHead(_), Request::Finalize(_)]
        ));
    }

    #[tokio::test(start_paused = true)]
    async fn publishes_only_validated_delegate_progress() {
        let current = BlockInfo {
            number: 7,
            hash: alloy_primitives::B256::repeat_byte(1),
            ..Default::default()
        };
        let status = sync_status(current);
        let mut invalid = status.clone();
        invalid.current_l1.hash = alloy_primitives::B256::repeat_byte(2);
        let delegate = TestDelegate(Mutex::new(VecDeque::from([invalid.clone(), status, invalid])));
        let mut provider = TestChainProvider::default();
        provider.insert_block(0, BlockInfo::default());
        provider.insert_block(current.number, current);
        let engine = TestEngine::new([]);
        let (status, progress) = watch::channel(DerivationStatus::default());
        let mut actor = Actor::new(engine.clone(), status, delegate, provider);
        actor.fetch_and_apply_delegate_safe_head().await.unwrap();
        assert_eq!(progress.borrow().current_l1, None);
        actor.fetch_and_apply_delegate_safe_head().await.unwrap();
        assert_eq!(progress.borrow().current_l1, Some(current));
        actor.fetch_and_apply_delegate_safe_head().await.unwrap();
        assert_eq!(progress.borrow().current_l1, Some(current));
        // Only the validated progress reaches the engine.
        assert!(matches!(
            engine.requests()[..],
            [Request::FollowSafeHead(_), Request::Finalize(_)]
        ));
    }
}
