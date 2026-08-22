//! Engine query interface for external communication.
//!
//! Provides a channel-based API for querying engine state and configuration
//! from external actors. Uses oneshot channels for responses to maintain
//! clean async communication patterns.

use std::sync::Arc;

use alloy_eips::BlockNumberOrTag;
use alloy_transport::{RpcError, TransportErrorKind};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OutputRoot, Predeploys};
use tokio::sync::oneshot::Sender;

use crate::{EngineClientError, EngineRpcClient, EngineState, LocalSafeSnapshot};

/// Channel sender for submitting [`EngineQueries`] to the engine.
pub type EngineQuerySender = tokio::sync::mpsc::Sender<EngineQueries>;

/// Query types supported by the engine for external communication.
///
/// Each variant includes a oneshot sender for the response, enabling
/// async request-response patterns. The engine processes these queries
/// and sends responses back through the provided channels.
#[derive(Debug)]
pub enum EngineQueries {
    /// Request the current rollup configuration.
    Config(Sender<RollupConfig>),
    /// Request the current [`EngineState`] snapshot.
    State(Sender<EngineState>),
    /// Request the L2 output root for a specific block.
    ///
    /// Returns a tuple of block info, output root, and engine state at the requested block.
    OutputAtBlock {
        /// The block number or tag to retrieve the output for.
        block: BlockNumberOrTag,
        /// Response channel for (`block_info`, `output_root`, `engine_state`).
        sender: Sender<(L2BlockInfo, OutputRoot, EngineState)>,
    },
    /// Request the local-safe head at an L2 timestamp, the L1 block it was derived from, and the
    /// sync status all of it was read with, as one [`LocalSafeSnapshot`].
    ///
    /// Interop's cross-safety decision needs the L2 block and its L1 key to describe the same
    /// instant. op-supernode's `ChainContainer.OptimisticAt` assembles the same answer from two
    /// reads and documents the TOCTOU gap between them; this variant is answered from a single
    /// borrow of the state watch, so there is no gap to document.
    LocalSafeSnapshotAt {
        /// The L2 timestamp to answer for.
        timestamp: u64,
        /// Response channel for the snapshot.
        sender: Sender<LocalSafeSnapshot>,
    },
    /// Subscribe to engine state updates via a watch channel receiver.
    StateReceiver(Sender<tokio::sync::watch::Receiver<EngineState>>),
    /// Development API: Subscribe to task queue length updates.
    QueueLengthReceiver(Sender<tokio::sync::watch::Receiver<usize>>),
    /// Development API: Get the current number of pending tasks in the queue.
    TaskQueueLength(Sender<usize>),
}

/// An error that can occur when querying the engine.
#[derive(Debug, thiserror::Error)]
pub enum EngineQueriesError {
    /// The output channel was closed unexpectedly. Impossible to send query response.
    #[error("Output channel closed unexpectedly. Impossible to send query response")]
    OutputChannelClosed,
    /// Failed to retrieve the L2 block by label.
    #[error("Failed to retrieve L2 block by label: {0}")]
    BlockRetrievalFailed(#[from] EngineClientError),
    /// No block withdrawals root while Isthmus is active.
    #[error("No block withdrawals root while Isthmus is active")]
    NoWithdrawalsRoot,
    /// No L2 block found for block number or tag.
    #[error("No L2 block found for block number or tag: {0}")]
    NoL2BlockFound(BlockNumberOrTag),
    /// Impossible to retrieve L2 withdrawals root from state.
    #[error("Impossible to retrieve L2 withdrawals root from state. {0}")]
    FailedToRetrieveWithdrawalsRoot(#[from] RpcError<TransportErrorKind>),
}

impl EngineQueries {
    /// Handles the engine query request.
    ///
    /// The engine state is borrowed from the watch exactly once, at the top, and every
    /// state-derived answer below is computed from that one [`EngineState`] value. That is what
    /// makes [`Self::LocalSafeSnapshotAt`] atomic: a second borrow could observe a local-safe
    /// head that moved, or was reset, since the first.
    pub async fn handle<EngineRpcClient_: EngineRpcClient + ?Sized>(
        self,
        state_recv: &tokio::sync::watch::Receiver<EngineState>,
        queue_length_recv: &tokio::sync::watch::Receiver<usize>,
        client: &Arc<EngineRpcClient_>,
        rollup_config: &Arc<RollupConfig>,
    ) -> Result<(), EngineQueriesError> {
        let state = *state_recv.borrow();

        match self {
            Self::Config(sender) => sender
                .send((**rollup_config).clone())
                .map_err(|_| EngineQueriesError::OutputChannelClosed),
            Self::State(sender) => {
                sender.send(state).map_err(|_| EngineQueriesError::OutputChannelClosed)
            }
            Self::OutputAtBlock { block, sender } => {
                let output_block = client.l2_block_by_label(block).await?;
                let output_block = output_block.ok_or(EngineQueriesError::NoL2BlockFound(block))?;
                // Cloning the l2 block below is cheaper than sending a network request to get the
                // l2 block info. Querying the `L2BlockInfo` from the client ends up
                // fetching the full l2 block again.
                let consensus_block = output_block.clone().into_consensus();
                let output_block_info =
                    L2BlockInfo::from_block_and_genesis::<op_alloy_consensus::OpTxEnvelope>(
                        &consensus_block.map_transactions(|tx| tx.inner.inner.into_inner()),
                        &rollup_config.genesis,
                    )
                    .map_err(|_| EngineQueriesError::NoL2BlockFound(block))?;

                let state_root = output_block.header.state_root;

                let message_passer_storage_root =
                    if rollup_config.is_isthmus_active(output_block.header.timestamp) {
                        output_block
                            .header
                            .withdrawals_root
                            .ok_or(EngineQueriesError::NoWithdrawalsRoot)?
                    } else {
                        // Fetch the storage root for the L2 head block.
                        client
                            .get_storage_hash(Predeploys::L2_TO_L1_MESSAGE_PASSER, block.into())
                            .await?
                    };

                let output_response_v0 = OutputRoot::from_parts(
                    state_root,
                    message_passer_storage_root,
                    output_block.header.hash,
                );

                sender
                    .send((output_block_info, output_response_v0, state))
                    .map_err(|_| EngineQueriesError::OutputChannelClosed)
            }
            Self::LocalSafeSnapshotAt { timestamp, sender } => sender
                // `state` above is the one and only read of the watch, and every field of the
                // snapshot is computed from that value, so the pairing and the sync status it is
                // reported beside cannot come from different instants.
                .send(state.local_safe_snapshot_at(rollup_config, timestamp))
                .map_err(|_| EngineQueriesError::OutputChannelClosed),
            Self::StateReceiver(subscription) => subscription
                .send(state_recv.clone())
                .map_err(|_| EngineQueriesError::OutputChannelClosed),
            Self::QueueLengthReceiver(subscription) => subscription
                .send(queue_length_recv.clone())
                .map_err(|_| EngineQueriesError::OutputChannelClosed),
            Self::TaskQueueLength(sender) => {
                let queue_length = *queue_length_recv.borrow();
                if sender.send(queue_length).is_err() {
                    warn!(target: "engine", "Failed to send task queue length response");
                }
                Ok(())
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        EngineSyncStateUpdate, LocalSafeAtTimestamp, LocalSafeHead, LocalSafeOrigin,
        test_utils::{TestEngineStateBuilder, test_engine_client_builder},
    };
    use kona_genesis::{ChainGenesis, RollupConfig};
    use kona_protocol::{BlockInfo, L2BlockInfo};

    fn rollup() -> Arc<RollupConfig> {
        Arc::new(RollupConfig {
            block_time: 2,
            genesis: ChainGenesis { l2_time: 90, ..Default::default() },
            ..Default::default()
        })
    }

    fn l1(number: u64) -> BlockInfo {
        BlockInfo { number, ..Default::default() }
    }

    fn l2(number: u64, timestamp: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { number, timestamp, ..Default::default() },
            ..Default::default()
        }
    }

    /// Issues a [`EngineQueries::LocalSafeSnapshotAt`] against the given state watch and returns
    /// the answer.
    async fn query(
        state_recv: &tokio::sync::watch::Receiver<EngineState>,
        timestamp: u64,
    ) -> crate::LocalSafeSnapshot {
        let (_, queue_length_recv) = tokio::sync::watch::channel(0usize);
        let client = Arc::new(test_engine_client_builder().build());
        let (sender, receiver) = tokio::sync::oneshot::channel();

        EngineQueries::LocalSafeSnapshotAt { timestamp, sender }
            .handle(state_recv, &queue_length_recv, &client, &rollup())
            .await
            .expect("the query is answered from the state watch alone");

        receiver.await.expect("the handler answered")
    }

    /// The handler serves the pairing and the sync status it was read with in one response.
    #[tokio::test]
    async fn the_handler_serves_the_pairing_and_the_sync_status() {
        let state = TestEngineStateBuilder::new()
            .with_unsafe_head(l2(12, 104))
            .with_local_safe_head(l2(10, 100))
            .with_local_safe_origin(LocalSafeOrigin::DerivedFrom(l1(5)))
            .with_finalized_head(l2(8, 96))
            .build();
        let (_state_tx, state_recv) = tokio::sync::watch::channel(state);

        let snapshot = query(&state_recv, 100).await;

        assert_eq!(
            snapshot.local_safe_at,
            LocalSafeAtTimestamp::Head(LocalSafeHead::derived_from(l2(10, 100), l1(5)))
        );
        assert_eq!(snapshot.sync_state, state.sync_state);
    }

    /// The head the answer describes and the L1 key it carries always come from the same update: a
    /// query issued after the head advances reports the new pairing, never the new head beside the
    /// old origin.
    #[tokio::test]
    async fn the_answer_follows_the_head_and_its_origin_together() {
        let state = TestEngineStateBuilder::new()
            .with_unsafe_head(l2(11, 102))
            .with_local_safe_head(l2(10, 100))
            .with_local_safe_origin(LocalSafeOrigin::DerivedFrom(l1(5)))
            .with_finalized_head(l2(8, 96))
            .build();
        let (state_tx, state_recv) = tokio::sync::watch::channel(state);

        assert_eq!(
            query(&state_recv, 102).await.local_safe_at,
            LocalSafeAtTimestamp::NotLocalSafeYet
        );

        let mut advanced = state;
        advanced.sync_state = advanced.sync_state.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(LocalSafeHead::derived_from(l2(11, 102), l1(6))),
            ..EngineSyncStateUpdate::NONE
        });
        state_tx.send(advanced).expect("the receiver is alive");

        let snapshot = query(&state_recv, 102).await;
        assert_eq!(
            snapshot.local_safe_at,
            LocalSafeAtTimestamp::Head(LocalSafeHead::derived_from(l2(11, 102), l1(6)))
        );
        assert_eq!(
            snapshot.local_safe_at.head().unwrap().head,
            snapshot.sync_state.local_safe_head()
        );
        // The timestamp that used to be the head's is now behind it, and the head's L1 key is not
        // offered for it.
        assert_eq!(query(&state_recv, 100).await.local_safe_at, LocalSafeAtTimestamp::BehindHead);
    }

    /// An unpaired head survives the trip through the handler as unpaired.
    #[tokio::test]
    async fn the_handler_serves_an_unpaired_head_as_unpaired() {
        let state = TestEngineStateBuilder::new()
            .with_unsafe_head(l2(10, 100))
            .with_local_safe_head(l2(10, 100))
            .with_local_safe_origin(LocalSafeOrigin::Unpaired)
            .with_finalized_head(l2(8, 96))
            .build();
        let (_state_tx, state_recv) = tokio::sync::watch::channel(state);

        let head = query(&state_recv, 100).await.local_safe_at.head().expect("the head answered");

        assert_eq!(head.origin, LocalSafeOrigin::Unpaired);
        assert_eq!(head.derived_from_l1(), None);
    }
}
