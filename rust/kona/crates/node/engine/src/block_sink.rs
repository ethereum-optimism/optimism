//! A hand-off point for blocks the engine has imported.

use async_trait::async_trait;
use kona_protocol::L2BlockInfo;
use op_alloy_consensus::OpBlock;
use std::fmt::Debug;

/// Receives every L2 block the engine successfully imports and canonicalizes.
///
/// The engine decodes each block anyway, so handing it over costs nothing beyond retaining it,
/// and spares consumers from re-fetching a block the node just processed.
///
/// Implementations run inline with block import and must not block.
#[async_trait]
pub trait ImportedBlockSink: Debug + Send + Sync {
    /// Records a block that the execution layer has accepted and the engine has canonicalized.
    async fn block_imported(&self, block: OpBlock, info: L2BlockInfo);
}

/// An [`ImportedBlockSink`] that discards what it is given, for consumers with nothing to record.
#[derive(Debug, Clone, Copy, Default)]
pub struct NoopBlockSink;

#[async_trait]
impl ImportedBlockSink for NoopBlockSink {
    async fn block_imported(&self, _: OpBlock, _: L2BlockInfo) {}
}
