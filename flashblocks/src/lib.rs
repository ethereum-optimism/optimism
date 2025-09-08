//! A downstream integration of Flashblocks.

pub use payload::{
    ExecutionPayloadBaseV1, ExecutionPayloadFlashblockDeltaV1, FlashBlock, Metadata,
};
use reth_rpc_eth_types::PendingBlock;
pub use service::FlashBlockService;
pub use ws::{WsConnect, WsFlashBlockStream};

mod payload;
mod sequence;
pub use sequence::FlashBlockCompleteSequence;
mod service;
mod worker;
mod ws;

/// Receiver of the most recent [`PendingBlock`] built out of [`FlashBlock`]s.
///
/// [`FlashBlock`]: crate::FlashBlock
pub type FlashBlockRx<N> = tokio::sync::watch::Receiver<Option<PendingBlock<N>>>;
