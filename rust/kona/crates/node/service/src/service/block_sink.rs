//! Feeds blocks the engine imports into the buffer that backs derivation's L2 lookups.

use async_trait::async_trait;
use kona_engine::ImportedBlockSink;
use kona_protocol::L2BlockInfo;
use kona_providers_local::BufferedL2Provider;
use op_alloy_consensus::OpBlock;

/// Records every block the engine imports in the node's local block buffer, so that the
/// sequencer and derivation can read the block they are building on top of without fetching it
/// back from the execution layer.
///
/// Entries are only ever addressed by hash, so one for a block that is later reorged out is never
/// returned — it just ages out.
#[derive(Debug, Clone, derive_more::Constructor)]
pub(crate) struct BufferImportedBlocks {
    /// The buffer shared with the derivation providers.
    buffer: BufferedL2Provider,
}

#[async_trait]
impl ImportedBlockSink for BufferImportedBlocks {
    async fn block_imported(&self, block: OpBlock, info: L2BlockInfo) {
        // A block that fails to buffer only costs a fetch later, so this must never be fatal.
        if let Err(err) = self.buffer.add_block(block, info).await {
            warn!(target: "engine", ?err, "Failed to buffer imported block");
        }
    }
}
