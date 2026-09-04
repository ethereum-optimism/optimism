//! Contains the forkchoice state for the L2.

use crate::{EngineClient, SyncStartError};
use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_provider::Network;
use alloy_transport::TransportResult;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_network::Optimism;
use std::fmt::Display;

/// An unsafe, local-safe, and finalized [`L2BlockInfo`] returned by the
/// [`crate::find_starting_forkchoice`] function.
///
/// The walkback only ever establishes local-safety: it re-derives nothing cross-chain, so a reset
/// must not use it to move the cross-safe head.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct L2ForkchoiceState {
    /// The unsafe L2 block.
    pub un_safe: L2BlockInfo,
    /// The local-safe L2 block.
    pub local_safe: L2BlockInfo,
    /// The finalized L2 block.
    pub finalized: L2BlockInfo,
}

impl Display for L2ForkchoiceState {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(
            f,
            "FINALIZED: {} (#{}) | LOCAL-SAFE: {} (#{}) | UNSAFE: {} (#{})",
            self.finalized.block_info.hash,
            self.finalized.block_info.number,
            self.local_safe.block_info.hash,
            self.local_safe.block_info.number,
            self.un_safe.block_info.hash,
            self.un_safe.block_info.number,
        )
    }
}

impl L2ForkchoiceState {
    /// Fetches the current forkchoice state of the L2 execution layer.
    ///
    /// - The finalized block may not always be available. If it is not, we fall back to genesis.
    /// - The local-safe block may not always be available. If it is not, we fall back to the
    ///   finalized block.
    /// - The unsafe block is always assumed to be available.
    pub async fn current<EngineClient_: EngineClient>(
        cfg: &RollupConfig,
        engine_client: &EngineClient_,
    ) -> Result<Self, SyncStartError> {
        let finalized = {
            let rpc_block =
                match get_block_compat(engine_client, BlockNumberOrTag::Finalized.into()).await {
                    Ok(Some(block)) => block,
                    Ok(None) => engine_client
                        .get_l2_block(cfg.genesis.l2.number.into())
                        .full()
                        .await?
                        .ok_or(SyncStartError::BlockNotFound(cfg.genesis.l2.number.into()))?,
                    Err(e) => return Err(e.into()),
                }
                .into_consensus();

            L2BlockInfo::from_block_and_genesis(&rpc_block, &cfg.genesis)?
        };
        let local_safe = match get_block_compat(engine_client, BlockNumberOrTag::Safe.into()).await
        {
            Ok(Some(block)) => {
                L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &cfg.genesis)?
            }
            Ok(None) => finalized,
            Err(e) => return Err(e.into()),
        };
        let un_safe = {
            let rpc_block = get_block_compat(engine_client, BlockNumberOrTag::Latest.into())
                .await?
                .ok_or(SyncStartError::BlockNotFound(BlockNumberOrTag::Latest.into()))?;
            L2BlockInfo::from_block_and_genesis(&rpc_block.into_consensus(), &cfg.genesis)?
        };

        Ok(Self { un_safe, local_safe, finalized })
    }
}

/// Wrapper function around [`EngineClient::get_l2_block`] to handle compatibility issues with geth
/// and erigon. When serving a block-by-number request, these clients will return non-standard
/// errors for the safe and finalized heads when the chain has just started and nothing is marked as
/// safe or finalized yet.
async fn get_block_compat<EngineClient_: EngineClient>(
    engine_client: &EngineClient_,
    block_id: BlockId,
) -> TransportResult<Option<<Optimism as Network>::BlockResponse>> {
    match engine_client.get_l2_block(block_id).full().await {
        Err(e) => {
            let err_str = e.to_string();
            if err_str.contains("block not found") || err_str.contains("Unknown block") {
                Ok(None)
            } else {
                Err(e)
            }
        }
        r => r,
    }
}
