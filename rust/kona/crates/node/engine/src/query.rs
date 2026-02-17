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

use crate::{EngineClient, EngineClientError, EngineState};

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
    pub async fn handle<EngineClient_: EngineClient>(
        self,
        state_recv: &tokio::sync::watch::Receiver<EngineState>,
        queue_length_recv: &tokio::sync::watch::Receiver<usize>,
        client: &Arc<EngineClient_>,
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
                        let l2_to_l1_message_passer = client
                            .get_proof(Predeploys::L2_TO_L1_MESSAGE_PASSER, Default::default())
                            .block_id(block.into())
                            .await?;

                        l2_to_l1_message_passer.storage_hash
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
