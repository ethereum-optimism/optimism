//! A downstream integration of Flashblocks.

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
pub use sequence::FlashBlockCompleteSequence;

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
