//! The actor that follows a derivation delegate.

use super::{DerivationDelegateProvider, Reported};
use crate::engine::Safe;
use alloy_primitives::BlockHash;
use kona_derive::ChainProvider;
use kona_protocol::{BlockInfo, SyncStatus};
use std::convert::Infallible;
use thiserror::Error;
use tokio::{sync::watch, time};

/// Polls an external derivation delegate for its sync status, validates the status's L1 blocks
/// against the canonical L1 chain, and publishes the validated safe and finalized L2 heads.
#[derive(Debug)]
pub(super) struct Actor<Delegate, L1Provider> {
    /// The derivation delegate.
    delegate: Delegate,
    /// L1 provider for validating L1 info for derivation delegation.
    l1_provider: L1Provider,
    /// Publishes the validated heads.
    reported: watch::Sender<Option<Reported>>,
    /// Publishes the L1 block the delegate has derived up to.
    current_l1: watch::Sender<BlockInfo>,
}

impl<Delegate, L1Provider> Actor<Delegate, L1Provider>
where
    Delegate: DerivationDelegateProvider,
    L1Provider: ChainProvider + Send,
{
    /// How often the actor polls the delegate.
    const POLL_INTERVAL: time::Duration = time::Duration::from_secs(4);

    /// Creates the actor.
    pub(super) const fn new(
        delegate: Delegate,
        l1_provider: L1Provider,
        reported: watch::Sender<Option<Reported>>,
        current_l1: watch::Sender<BlockInfo>,
    ) -> Self {
        Self { delegate, l1_provider, reported, current_l1 }
    }

    /// Polls the delegate for as long as the actor runs.
    pub(super) async fn run(mut self) -> Infallible {
        let mut ticker = time::interval(Self::POLL_INTERVAL);
        ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        loop {
            ticker.tick().await;
            self.poll().await;
        }
    }

    /// Fetches the delegate's sync status, validates it, and publishes it.
    async fn poll(&mut self) {
        let sync_status = match self.delegate.fetch_sync_status().await {
            Ok(status) => status,
            Err(_) => {
                warn!(target: "derivation", "Failed to fetch sync status from delegate");
                return;
            }
        };

        // Validation failures here are expected to be transient, so the actor skips this sync
        // status and keeps polling.
        if !self.validate_sync_status(&sync_status).await {
            return;
        }

        self.current_l1.send_replace(sync_status.current_l1);
        // The delegate reports no source for its safe head. The L1 block it has derived up to is
        // at or after the source.
        let reported = Reported {
            safe: Safe { head: sync_status.safe_l2, source: sync_status.current_l1 },
            finalized: sync_status.finalized_l2,
        };
        self.reported.send_if_modified(|published| published.replace(reported) != Some(reported));
        debug!(
            target: "derivation",
            safe_l2 = ?sync_status.safe_l2,
            finalized_l2 = ?sync_status.finalized_l2,
            "Processed sync status from delegate"
        );
    }

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
        DerivationDelegateClientError,
        trusted::{Builder, Handle},
    };
    use alloy_primitives::B256;
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
    fn build_does_not_require_a_runtime() {
        let task = Builder::new()
            .build(TestDelegate(Mutex::new(VecDeque::new())), TestChainProvider::default());
        drop(task);
    }

    /// Returns a delegate sync status whose L1 blocks are the given current L1 block and genesis,
    /// with the given safe head.
    fn sync_status(current: BlockInfo, safe: L2BlockInfo) -> SyncStatus {
        // Fill the delegate's unused sync fields with zero values.
        SyncStatus {
            current_l1: current,
            current_l1_finalized: BlockInfo::default(),
            head_l1: BlockInfo::default(),
            safe_l1: BlockInfo::default(),
            finalized_l1: BlockInfo::default(),
            unsafe_l2: L2BlockInfo::default(),
            local_safe_l2: L2BlockInfo::default(),
            safe_l2: safe,
            finalized_l2: L2BlockInfo::default(),
        }
    }

    /// The actor publishes only the delegate's progress that L1 confirms, with the L1 block the
    /// delegate derived up to standing in for its safe head's source.
    #[tokio::test]
    async fn publishes_only_validated_delegate_progress() {
        let current = BlockInfo { number: 7, hash: B256::repeat_byte(1), ..Default::default() };
        let safe = L2BlockInfo {
            block_info: BlockInfo { number: 3, ..Default::default() },
            ..Default::default()
        };
        let status = sync_status(current, safe);
        let mut invalid = status.clone();
        invalid.current_l1.hash = B256::repeat_byte(2);
        let delegate = TestDelegate(Mutex::new(VecDeque::from([invalid.clone(), status, invalid])));
        let mut provider = TestChainProvider::default();
        provider.insert_block(0, BlockInfo::default());
        provider.insert_block(current.number, current);
        let (reported_tx, reported) = watch::channel(None);
        let (current_l1_tx, current_l1) = watch::channel(BlockInfo::default());
        let mut actor = Actor::new(delegate, provider, reported_tx, current_l1_tx);

        actor.poll().await;
        assert_eq!(*current_l1.borrow(), BlockInfo::default());
        assert_eq!(*reported.borrow(), None);
        let expected = Reported {
            safe: Safe { head: safe, source: current },
            finalized: L2BlockInfo::default(),
        };
        actor.poll().await;
        assert_eq!(*current_l1.borrow(), current);
        assert_eq!(*reported.borrow(), Some(expected));
        actor.poll().await;
        assert_eq!(*current_l1.borrow(), current);
        assert_eq!(*reported.borrow(), Some(expected));
    }

    /// The handle returns the delegate's safe head once it is new, or differs from the engine's,
    /// such as after the engine reset, and its finalized head once it is new.
    #[tokio::test(start_paused = true)]
    async fn handle_returns_heads_the_engine_has_not_adopted() {
        let (reported_tx, reported) = watch::channel(None);
        let (_current_l1_tx, current_l1) = watch::channel(BlockInfo::default());
        let mut handle = Handle::new(reported, current_l1);
        let block = |number| L2BlockInfo {
            block_info: BlockInfo { number, ..Default::default() },
            ..Default::default()
        };
        let safe = Safe { head: block(2), source: BlockInfo::default() };
        let heads = Reported { safe, finalized: block(1) };
        let wait = time::Duration::from_secs(10);

        // Nothing reported yet.
        assert!(time::timeout(wait, handle.next_safe(block(0))).await.is_err());
        assert!(time::timeout(wait, handle.next_finalized()).await.is_err());
        reported_tx.send_replace(Some(heads));
        assert_eq!(handle.next_safe(block(0)).await.unwrap(), safe);
        assert_eq!(handle.next_finalized().await.unwrap(), block(1));
        // The engine adopted them.
        assert!(time::timeout(wait, handle.next_safe(block(2))).await.is_err());
        assert!(time::timeout(wait, handle.next_finalized()).await.is_err());
        // The engine reset onto another safe head.
        assert_eq!(handle.next_safe(block(1)).await.unwrap(), safe);
        // Only the finalized head moves.
        reported_tx.send_replace(Some(Reported { finalized: block(2), ..heads }));
        assert!(time::timeout(wait, handle.next_safe(block(2))).await.is_err());
        assert_eq!(handle.next_finalized().await.unwrap(), block(2));

        drop(reported_tx);
        assert!(handle.next_safe(block(2)).await.is_err());
        assert!(handle.next_finalized().await.is_err());
    }
}
