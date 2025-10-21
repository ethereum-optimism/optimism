//! A downstream integration of Flashblocks.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

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
