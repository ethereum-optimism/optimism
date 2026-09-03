//! The engine's imported blocks, held by the chain view and served back to derivation.

use std::sync::Arc;

use alloy_primitives::B256;
use async_trait::async_trait;
use kona_chainview::{ChainViewClient, Fact, ImportedL2Block};
use kona_engine::ImportedBlockSink;
use kona_protocol::L2BlockInfo;
use kona_providers_alloy::ImportedL2Blocks;
use op_alloy_consensus::OpBlock;

/// Hands every block the engine imports to the chain view, and answers derivation's hash-keyed
/// lookups from what the view still holds (the newest blocks at or above the engine's finalized
/// head), so the block being built on is not fetched back from the execution layer.
///
/// Entries are only ever addressed by hash, so one for a block that is later reorged out is
/// never returned; it is dropped once it falls below finalized or out of the newest set.
#[derive(Debug, Clone, derive_more::Constructor)]
pub(crate) struct ChainViewL2Blocks {
    /// The chain view holding the blocks.
    chainview: ChainViewClient,
}

impl ImportedBlockSink for ChainViewL2Blocks {
    fn block_imported(&self, block: OpBlock, info: L2BlockInfo) {
        // Runs inline with import, so it never waits; a dropped block only costs a fetch later.
        let fact = Fact::L2Imported(ImportedL2Block { info, block: Arc::new(block) });
        if let Err(err) = self.chainview.try_push(fact) {
            warn!(
                target: "chainview",
                %err,
                number = info.block_info.number,
                "Dropping imported block; derivation will fetch it from the L2 RPC"
            );
        }
    }
}

#[async_trait]
impl ImportedL2Blocks for ChainViewL2Blocks {
    async fn imported_l2_block(&self, hash: B256) -> Option<(L2BlockInfo, Arc<OpBlock>)> {
        match self.chainview.imported_l2_block(hash).await {
            Ok(held) => held.map(|held| (held.info, held.block)),
            Err(err) => {
                warn!(target: "chainview", %err, "Imported block lookup failed; using the L2 RPC");
                None
            }
        }
    }
}
