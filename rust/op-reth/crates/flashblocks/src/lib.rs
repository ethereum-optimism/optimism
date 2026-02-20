//! A downstream integration of Flashblocks.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use reth_primitives_traits::NodePrimitives;
use std::sync::Arc;

// Included to enable serde feature for OpReceipt type used transitively
use reth_optimism_primitives as _;

mod consensus;
pub use consensus::FlashBlockConsensusClient;

mod payload;
pub use payload::{FlashBlock, PendingFlashBlock};

mod sequence;
pub use sequence::{FlashBlockCompleteSequence, FlashBlockPendingSequence};

mod service;
pub use service::{
    CanonicalBlockNotification, FlashBlockBuildInfo, FlashBlockService,
    create_canonical_block_channel,
};

mod worker;

mod cache;

mod pending_state;
pub use pending_state::{PendingBlockState, PendingStateRegistry};

#[cfg(test)]
mod test_utils;
pub mod validation;

mod ws;
pub use ws::{FlashBlockDecoder, WsConnect, WsFlashBlockStream};

/// Receiver of the most recent [`PendingFlashBlock`] built out of [`FlashBlock`]s.
///
/// [`FlashBlock`]: crate::FlashBlock
pub type PendingBlockRx<N> = tokio::sync::watch::Receiver<Option<PendingFlashBlock<N>>>;

/// Receiver of the sequences of [`FlashBlock`]s built.
///
/// [`FlashBlock`]: crate::FlashBlock
pub type FlashBlockCompleteSequenceRx =
    tokio::sync::broadcast::Receiver<FlashBlockCompleteSequence>;

/// Receiver of received [`FlashBlock`]s from the (websocket) subscription.
///
/// [`FlashBlock`]: crate::FlashBlock
pub type FlashBlockRx = tokio::sync::broadcast::Receiver<Arc<FlashBlock>>;

/// Receiver that signals whether a [`FlashBlock`] is currently being built.
pub type InProgressFlashBlockRx = tokio::sync::watch::Receiver<Option<FlashBlockBuildInfo>>;

/// Container for all flashblocks-related listeners.
///
/// Groups together the channels for flashblock-related updates.
#[derive(Debug)]
pub struct FlashblocksListeners<N: NodePrimitives> {
    /// Receiver of the most recent executed [`PendingFlashBlock`] built out of [`FlashBlock`]s.
    pub pending_block_rx: PendingBlockRx<N>,
    /// Subscription channel of the complete sequences of [`FlashBlock`]s built.
    pub flashblocks_sequence: tokio::sync::broadcast::Sender<FlashBlockCompleteSequence>,
    /// Receiver that signals whether a [`FlashBlock`] is currently being built.
    pub in_progress_rx: InProgressFlashBlockRx,
    /// Subscription channel for received flashblocks from the (websocket) connection.
    pub received_flashblocks: tokio::sync::broadcast::Sender<Arc<FlashBlock>>,
}

impl<N: NodePrimitives> FlashblocksListeners<N> {
    /// Creates a new [`FlashblocksListeners`] with the given channels.
    pub const fn new(
        pending_block_rx: PendingBlockRx<N>,
        flashblocks_sequence: tokio::sync::broadcast::Sender<FlashBlockCompleteSequence>,
        in_progress_rx: InProgressFlashBlockRx,
        received_flashblocks: tokio::sync::broadcast::Sender<Arc<FlashBlock>>,
    ) -> Self {
        Self { pending_block_rx, flashblocks_sequence, in_progress_rx, received_flashblocks }
    }
}
