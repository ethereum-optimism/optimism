//! A downstream integration of Flashblocks.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use reth_primitives_traits::NodePrimitives;

pub use payload::{
    ExecutionPayloadBaseV1, ExecutionPayloadFlashblockDeltaV1, FlashBlock, FlashBlockDecoder,
    Metadata,
};
pub use service::{FlashBlockBuildInfo, FlashBlockService};
pub use ws::{WsConnect, WsFlashBlockStream};

mod consensus;
pub use consensus::FlashBlockConsensusClient;
mod payload;
pub use payload::PendingFlashBlock;
mod sequence;
pub use sequence::{FlashBlockCompleteSequence, FlashBlockPendingSequence};

mod service;
mod worker;
mod ws;

/// Receiver of the most recent [`PendingFlashBlock`] built out of [`FlashBlock`]s.
///
/// [`FlashBlock`]: crate::FlashBlock
pub type PendingBlockRx<N> = tokio::sync::watch::Receiver<Option<PendingFlashBlock<N>>>;

/// Receiver of the sequences of [`FlashBlock`]s built.
///
/// [`FlashBlock`]: crate::FlashBlock
pub type FlashBlockCompleteSequenceRx =
    tokio::sync::broadcast::Receiver<FlashBlockCompleteSequence>;

/// Receiver that signals whether a [`FlashBlock`] is currently being built.
pub type InProgressFlashBlockRx = tokio::sync::watch::Receiver<Option<FlashBlockBuildInfo>>;

/// Container for all flashblocks-related listeners.
///
/// Groups together the three receivers that provide flashblock-related updates.
#[derive(Debug)]
pub struct FlashblocksListeners<N: NodePrimitives> {
    /// Receiver of the most recent [`PendingFlashBlock`] built out of [`FlashBlock`]s.
    pub pending_block_rx: PendingBlockRx<N>,
    /// Receiver of the sequences of [`FlashBlock`]s built.
    pub flashblock_rx: FlashBlockCompleteSequenceRx,
    /// Receiver that signals whether a [`FlashBlock`] is currently being built.
    pub in_progress_rx: InProgressFlashBlockRx,
}

impl<N: NodePrimitives> FlashblocksListeners<N> {
    /// Creates a new [`FlashblocksListeners`] with the given receivers.
    pub const fn new(
        pending_block_rx: PendingBlockRx<N>,
        flashblock_rx: FlashBlockCompleteSequenceRx,
        in_progress_rx: InProgressFlashBlockRx,
    ) -> Self {
        Self { pending_block_rx, flashblock_rx, in_progress_rx }
    }
}

impl<N: NodePrimitives> Clone for FlashblocksListeners<N> {
    fn clone(&self) -> Self {
        Self {
            pending_block_rx: self.pending_block_rx.clone(),
            flashblock_rx: self.flashblock_rx.resubscribe(),
            in_progress_rx: self.in_progress_rx.clone(),
        }
    }
}
