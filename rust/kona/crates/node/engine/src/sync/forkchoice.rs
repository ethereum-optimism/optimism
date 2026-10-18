//! Contains the forkchoice state for the L2.

use super::{EngineSyncSource, SyncSource, read_l2};
use crate::{EngineClient, SyncStartError};
use alloy_eips::BlockNumberOrTag;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::fmt::Display;

/// An unsafe, safe, and finalized [`L2BlockInfo`] returned by the
/// [`crate::find_starting_forkchoice`] function.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct L2ForkchoiceState {
    /// The unsafe L2 block.
    pub un_safe: L2BlockInfo,
    /// The safe L2 block.
    pub safe: L2BlockInfo,
    /// The finalized L2 block.
    pub finalized: L2BlockInfo,
}

impl Display for L2ForkchoiceState {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "FINALIZED: {} (#{}) | SAFE: {} (#{}) | UNSAFE: {} (#{})",
            self.finalized.block_info.hash,
            self.finalized.block_info.number,
            self.safe.block_info.hash,
            self.safe.block_info.number,
            self.un_safe.block_info.hash,
            self.un_safe.block_info.number,
        )
    }
}

impl L2ForkchoiceState {
    /// Fetches the current forkchoice state of the L2 execution layer.
    ///
    /// - The finalized block may not always be available. If it is not, we fall back to genesis.
    /// - The safe block may not always be available. If it is not, we fall back to the finalized
    ///   block.
    /// - The unsafe block is always assumed to be available.
    pub async fn current<EngineClient_: EngineClient>(
        cfg: &RollupConfig,
        engine_client: &EngineClient_,
    ) -> Result<Self, SyncStartError> {
        Self::load(cfg, &EngineSyncSource { client: engine_client, cfg }).await
    }

    pub(super) async fn load(
        cfg: &RollupConfig,
        source: &impl SyncSource,
    ) -> Result<Self, SyncStartError> {
        let finalized = match read_l2(source, BlockNumberOrTag::Finalized.into()).await? {
            Some(block) => block,
            None => read_l2(source, cfg.genesis.l2.number.into())
                .await?
                .ok_or(SyncStartError::BlockNotFound(cfg.genesis.l2.number.into()))?,
        };
        let safe = read_l2(source, BlockNumberOrTag::Safe.into()).await?.unwrap_or(finalized);
        let un_safe = read_l2(source, BlockNumberOrTag::Latest.into())
            .await?
            .ok_or(SyncStartError::BlockNotFound(BlockNumberOrTag::Latest.into()))?;
        Ok(Self { un_safe, safe, finalized })
    }
}
