use crate::{
    CrossSafetyError,
    event::ChainEvent,
    safety_checker::{CrossSafetyChecker, traits::SafetyPromoter},
};
use alloy_primitives::ChainId;
use derive_more::Constructor;
use kona_interop::InteropValidator;
use kona_protocol::BlockInfo;
use kona_supervisor_storage::{CrossChainSafetyProvider, StorageError};
use op_alloy_consensus::interop::SafetyLevel;
use std::{sync::Arc, time::Duration};
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use tracing::{debug, error, info};

/// A background job that promotes blocks to a target safety level on a given chain.
///
/// It uses [`CrossChainSafetyProvider`] to fetch candidate blocks and the [`CrossSafetyChecker`]
/// to validate cross-chain message dependencies.
#[derive(Debug, Constructor)]
pub struct CrossSafetyCheckerJob<P, V, L> {
    chain_id: ChainId,
    provider: Arc<P>,
    cancel_token: CancellationToken,
    interval: Duration,
    promoter: L,
    event_tx: mpsc::Sender<ChainEvent>,
    validator: Arc<V>,
}

impl<P, V, L> CrossSafetyCheckerJob<P, V, L>
where
    P: CrossChainSafetyProvider + Send + Sync + 'static,
    V: InteropValidator + Send + Sync + 'static,
    L: SafetyPromoter,
{
    /// Runs the job loop until cancelled, promoting blocks by Promoter
    ///
    /// On each iteration:
    /// - Tries to promote the next eligible block
    /// - Waits for configured interval if promotion fails
    /// - Exits when [`CancellationToken`] is triggered
    pub async fn run(self) {
        let target_level = self.promoter.target_level();
        let chain_id = self.chain_id;

        info!(
            target: "supervisor::safety_checker",
            chain_id,
            %target_level,
            "Started safety checker");

        let checker =
            CrossSafetyChecker::new(chain_id, &*self.validator, &*self.provider, target_level);

        loop {
            tokio::select! {
                _ = self.cancel_token.cancelled() => {
                    info!(target: "supervisor::safety_checker", chain_id, %target_level, "Canceled safety checker");
                    break;
                }

                _ = async {
                    match self.promote_next_block(&checker) {
                        Ok(block_info) => {
                            debug!(
                                target: "supervisor::safety_checker",
                                chain_id,
                                %target_level,
                                %block_info,
                                "Promoted next candidate block"
                            );
                        }
                        Err(err) => {
                            match err {
                                // Expected / non-fatal errors:
                                //  - no candidate is ready right now
                                //  - validation failed (we already emitted invalidate event in promote_next_block for CrossSafe)
                                //  - dependency not yet safe on another chain
                                CrossSafetyError::NoBlockToPromote |
                                CrossSafetyError::ValidationError(_) |
                                CrossSafetyError::DependencyNotSafe { .. }   => {
                                    debug!(
                                        target: "supervisor::safety_checker",
                                        chain_id,
                                        %target_level,
                                        %err,
                                        "Error promoting next candidate block"
                                    );
                                },
                                _ => {
                                    error!(
                                        target: "supervisor::safety_checker",
                                        chain_id,
                                        %target_level,
                                        %err,
                                        "Unexpected error promoting next candidate block"
                                    );
                                }
                            }
                            tokio::time::sleep(self.interval).await;
                        }
                    }
                } => {}
            }
        }

        info!(target: "supervisor::safety_checker", chain_id = self.chain_id, %target_level, "Stopped safety checker");
    }

    // Attempts to promote the next block by the Promoter
    // after validating cross-chain dependencies.
    fn promote_next_block(
        &self,
        checker: &CrossSafetyChecker<'_, P, V>,
    ) -> Result<BlockInfo, CrossSafetyError> {
        let candidate = self.find_next_promotable_block()?;

        match checker.validate_block(candidate) {
            Ok(()) => {
                // Success: promote + emit
                let ev = self.promoter.update_and_emit_event(
                    &*self.provider,
                    self.chain_id,
                    &candidate,
                )?;
                self.broadcast_event(ev);
                Ok(candidate)
            }

            Err(err @ CrossSafetyError::ValidationError(_)) => {
                // Only invalidate if we are targeting CrossSafe
                if self.promoter.target_level() == SafetyLevel::CrossSafe {
                    info!(
                        target: "supervisor::safety_checker",
                        chain_id = self.chain_id,
                        target_level = %self.promoter.target_level(),
                        block_info = %candidate,
                        %err,
                        "Triggering block invalidation for the invalid block"
                    );
                    self.broadcast_event(ChainEvent::InvalidateBlock { block: candidate });
                }
                Err(err) // propagate the error for logging
            }
            Err(err) => Err(err),
        }
    }

    // Finds the next block that is eligible for promotion at the configured target level.
    fn find_next_promotable_block(&self) -> Result<BlockInfo, CrossSafetyError> {
        let current_head = self
            .provider
            .get_safety_head_ref(self.chain_id, self.promoter.target_level())
            .map_err(|err| {
                if matches!(err, StorageError::FutureData) {
                    CrossSafetyError::NoBlockToPromote
                } else {
                    err.into()
                }
            })?;

        let upper_head = self
            .provider
            .get_safety_head_ref(self.chain_id, self.promoter.lower_bound_level())
            .map_err(|err| {
                if matches!(err, StorageError::FutureData) {
                    CrossSafetyError::NoBlockToPromote
                } else {
                    err.into()
                }
            })?;

        if current_head.number >= upper_head.number {
            return Err(CrossSafetyError::NoBlockToPromote);
        }

        let candidate = self.provider.get_block(self.chain_id, current_head.number + 1)?;

        Ok(candidate)
    }

    fn broadcast_event(&self, event: ChainEvent) {
        if let Err(err) = self.event_tx.try_send(event) {
            error!(
                target: "supervisor::safety_checker",
                target_level = %self.promoter.target_level(),
                %err,
                "Failed to broadcast cross head update event",
            );
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::safety_checker::promoter::{CrossSafePromoter, CrossUnsafePromoter};
    use alloy_primitives::{B256, ChainId};
    use kona_interop::{DerivedRefPair, InteropValidationError};
    use kona_supervisor_storage::{CrossChainSafetyProvider, StorageError};
    use kona_supervisor_types::{ExecutingMessage, Log};
    use mockall::mock;
    use op_alloy_consensus::interop::SafetyLevel;

    mock! {
        #[derive(Debug)]
        pub Provider {}

        impl CrossChainSafetyProvider for Provider {
            fn get_block(&self, chain_id: ChainId, block_number: u64) -> Result<BlockInfo, StorageError>;
            fn get_log(&self, chain_id: ChainId, block_number: u64, log_index: u32) -> Result<Log, StorageError>;
            fn get_block_logs(&self, chain_id: ChainId, block_number: u64) -> Result<Vec<Log>, StorageError>;
            fn get_safety_head_ref(&self, chain_id: ChainId, level: SafetyLevel) -> Result<BlockInfo, StorageError>;
            fn update_current_cross_unsafe(&self, chain_id: ChainId, block: &BlockInfo) -> Result<(), StorageError>;
            fn update_current_cross_safe(&self, chain_id: ChainId, block: &BlockInfo) -> Result<DerivedRefPair, StorageError>;
        }
    }

    mock! (
        #[derive(Debug)]
        pub Validator {}

        impl InteropValidator for Validator {
            fn validate_interop_timestamps(
                &self,
                initiating_chain_id: ChainId,
                initiating_timestamp: u64,
                executing_chain_id: ChainId,
                executing_timestamp: u64,
                timeout: Option<u64>,
            ) -> Result<(), InteropValidationError>;

            fn is_post_interop(&self, chain_id: ChainId, timestamp: u64) -> bool;

            fn is_interop_activation_block(&self, chain_id: ChainId, block: BlockInfo) -> bool;
        }
    );

    fn b256(n: u64) -> B256 {
        let mut bytes = [0u8; 32];
        bytes[24..].copy_from_slice(&n.to_be_bytes());
        B256::from(bytes)
    }

    fn block(n: u64) -> BlockInfo {
        BlockInfo { number: n, hash: b256(n), parent_hash: b256(n - 1), timestamp: 0 }
    }

    #[tokio::test]
    async fn promotes_next_cross_unsafe_successfully() {
        let chain_id = 1;
        let mut mock = MockProvider::default();
        let mock_validator = MockValidator::default();
        let (event_tx, mut event_rx) = mpsc::channel::<ChainEvent>(10);

        mock.expect_get_safety_head_ref()
            .withf(move |cid, lvl| *cid == chain_id && *lvl == SafetyLevel::CrossUnsafe)
            .returning(|_, _| Ok(block(99)));

        mock.expect_get_safety_head_ref()
            .withf(move |cid, lvl| *cid == chain_id && *lvl == SafetyLevel::LocalUnsafe)
            .returning(|_, _| Ok(block(100)));

        mock.expect_get_block()
            .withf(move |cid, num| *cid == chain_id && *num == 100)
            .returning(|_, _| Ok(block(100)));

        mock.expect_get_block_logs()
            .withf(move |cid, num| *cid == chain_id && *num == 100)
            .returning(|_, _| Ok(vec![]));

        mock.expect_update_current_cross_unsafe()
            .withf(move |cid, blk| *cid == chain_id && blk.number == 100)
            .returning(|_, _| Ok(()));

        let job = CrossSafetyCheckerJob::new(
            chain_id,
            Arc::new(mock),
            CancellationToken::new(),
            Duration::from_secs(1),
            CrossUnsafePromoter,
            event_tx,
            Arc::new(mock_validator),
        );
        let checker = CrossSafetyChecker::new(
            job.chain_id,
            &*job.validator,
            &*job.provider,
            CrossUnsafePromoter.target_level(),
        );
        let result = job.promote_next_block(&checker);

        assert!(result.is_ok());
        assert_eq!(result.unwrap().number, 100);

        // Receive and assert the correct event
        let received_event = event_rx.recv().await.expect("expected event not received");

        assert_eq!(received_event, ChainEvent::CrossUnsafeUpdate { block: block(100) });
    }

    #[tokio::test]
    async fn promotes_next_cross_safe_successfully() {
        let chain_id = 1;
        let mut mock = MockProvider::default();
        let mock_validator = MockValidator::default();
        let (event_tx, mut event_rx) = mpsc::channel::<ChainEvent>(10);

        mock.expect_get_safety_head_ref()
            .withf(move |cid, lvl| *cid == chain_id && *lvl == SafetyLevel::CrossSafe)
            .returning(|_, _| Ok(block(99)));

        mock.expect_get_safety_head_ref()
            .withf(move |cid, lvl| *cid == chain_id && *lvl == SafetyLevel::LocalSafe)
            .returning(|_, _| Ok(block(100)));

        mock.expect_get_block()
            .withf(move |cid, num| *cid == chain_id && *num == 100)
            .returning(|_, _| Ok(block(100)));

        mock.expect_get_block_logs()
            .withf(move |cid, num| *cid == chain_id && *num == 100)
            .returning(|_, _| Ok(vec![]));

        mock.expect_update_current_cross_safe()
            .withf(move |cid, blk| *cid == chain_id && blk.number == 100)
            .returning(|_, _| Ok(DerivedRefPair { derived: block(100), source: block(1) }));

        let job = CrossSafetyCheckerJob::new(
            chain_id,
            Arc::new(mock),
            CancellationToken::new(),
            Duration::from_secs(1),
            CrossSafePromoter,
            event_tx,
            Arc::new(mock_validator),
        );

        let checker = CrossSafetyChecker::new(
            job.chain_id,
            &*job.validator,
            &*job.provider,
            CrossSafePromoter.target_level(),
        );
        let result = job.promote_next_block(&checker);

        assert!(result.is_ok());
        assert_eq!(result.unwrap().number, 100);

        // Receive and assert the correct event
        let received_event = event_rx.recv().await.expect("expected event not received");

        assert_eq!(
            received_event,
            ChainEvent::CrossSafeUpdate {
                derived_ref_pair: DerivedRefPair { derived: block(100), source: block(1) }
            }
        );
    }

    #[tokio::test]
    async fn promotes_next_cross_safe_triggers_block_invalidation() {
        let chain_id = 1;
        let mut mock = MockProvider::default();
        let mut mock_validator = MockValidator::default();
        let (event_tx, mut event_rx) = mpsc::channel::<ChainEvent>(10);

        let exec_msg = ExecutingMessage {
            chain_id: 2,
            block_number: 99,
            log_index: 0,
            timestamp: 195,
            hash: b256(99),
        };

        let exec_log = Log { index: 0, hash: b256(100), executing_message: Some(exec_msg) };

        mock.expect_get_safety_head_ref()
            .withf(move |cid, lvl| *cid == chain_id && *lvl == SafetyLevel::CrossSafe)
            .returning(|_, _| Ok(block(99)));

        mock.expect_get_safety_head_ref()
            .withf(move |cid, lvl| *cid == chain_id && *lvl == SafetyLevel::LocalSafe)
            .returning(|_, _| Ok(block(100)));

        mock.expect_get_block()
            .withf(move |cid, num| *cid == 2 && *num == 99)
            .returning(|_, _| Ok(block(99)));

        mock.expect_get_block()
            .withf(move |cid, num| *cid == chain_id && *num == 100)
            .returning(|_, _| Ok(block(100)));

        mock.expect_get_block_logs()
            .withf(move |cid, num| *cid == chain_id && *num == 100)
            .returning(move |_, _| Ok(vec![exec_log.clone()]));

        mock_validator.expect_validate_interop_timestamps().returning(move |_, _, _, _, _| {
            Err(InteropValidationError::InvalidTimestampInvariant { executing: 0, initiating: 0 })
        });

        let job = CrossSafetyCheckerJob::new(
            chain_id,
            Arc::new(mock),
            CancellationToken::new(),
            Duration::from_secs(1),
            CrossSafePromoter,
            event_tx,
            Arc::new(mock_validator),
        );

        let checker = CrossSafetyChecker::new(
            job.chain_id,
            &*job.validator,
            &*job.provider,
            CrossSafePromoter.target_level(),
        );
        let result = job.promote_next_block(&checker);

        assert!(result.is_err());
        assert!(
            matches!(result, Err(CrossSafetyError::ValidationError(_))),
            "Expected validation error"
        );

        // Receive and assert the correct event
        let received_event = event_rx.recv().await.expect("expected event not received");

        assert_eq!(received_event, ChainEvent::InvalidateBlock { block: block(100) });
    }

    #[test]
    fn promotes_next_cross_unsafe_failed_with_no_candidates() {
        let chain_id = 1;
        let mut mock = MockProvider::default();
        let mock_validator = MockValidator::default();
        let (event_tx, _) = mpsc::channel::<ChainEvent>(10);

        mock.expect_get_safety_head_ref()
            .withf(|_, lvl| *lvl == SafetyLevel::CrossSafe)
            .returning(|_, _| Ok(block(200)));

        mock.expect_get_safety_head_ref()
            .withf(|_, lvl| *lvl == SafetyLevel::LocalSafe)
            .returning(|_, _| Ok(block(200)));

        let job = CrossSafetyCheckerJob::new(
            chain_id,
            Arc::new(mock),
            CancellationToken::new(),
            Duration::from_secs(1),
            CrossSafePromoter,
            event_tx,
            Arc::new(mock_validator),
        );

        let checker = CrossSafetyChecker::new(
            job.chain_id,
            &*job.validator,
            &*job.provider,
            CrossSafePromoter.target_level(),
        );
        let result = job.promote_next_block(&checker);

        assert!(matches!(result, Err(CrossSafetyError::NoBlockToPromote)));
    }
}
