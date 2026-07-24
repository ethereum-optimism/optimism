//! Proposer service for the super-root ZK dispute game.
//!
//! Derived from op-succinct's fault-proof proposer
//! (succinctlabs/op-succinct `fault-proof` crate @ 13716c2c), adapted for the
//! monorepo `ZKDisputeGame`: super-root claims sourced from a supernode,
//! `parentIndex || superRootProof` extraData, vkey-based identity, and
//! two-phase `DelayedWETH` bond claiming. Proving/defense is intentionally
//! absent here; it arrives with the defend path (#21463).

pub mod backup;
pub mod config;
pub mod contract;
pub mod metrics;
pub mod proposer;
pub mod signer;
pub mod supernode;

use alloy_eips::BlockId;
use alloy_primitives::U256;
use alloy_provider::{Provider, RootProvider};
use anyhow::{Context, Result};
use async_trait::async_trait;

use crate::contract::{DisputeGameFactory::DisputeGameFactoryInstance, GameStatus, ZKDisputeGame};

pub type L1Provider = RootProvider;

pub const NUM_CONFIRMATIONS: u64 = 3;
pub const TIMEOUT_SECONDS: u64 = 60;

#[async_trait]
pub trait FactoryTrait<P>
where
    P: Provider + Clone,
{
    /// Fetches the bond required to create a game.
    async fn fetch_init_bond(&self, game_type: u32) -> Result<U256>;

    /// Fetches the latest game index.
    async fn fetch_latest_game_index(&self, block: BlockId) -> Result<Option<U256>>;

    /// Fetches the game address by index.
    async fn fetch_game_address_by_index(
        &self,
        game_index: U256,
    ) -> Result<alloy_primitives::Address>;
}

#[async_trait]
impl<P> FactoryTrait<P> for DisputeGameFactoryInstance<P>
where
    P: Provider + Clone,
{
    async fn fetch_init_bond(&self, game_type: u32) -> Result<U256> {
        let init_bond = self.initBonds(game_type).call().await?;
        Ok(init_bond)
    }

    async fn fetch_latest_game_index(&self, block: BlockId) -> Result<Option<U256>> {
        let game_count = self.gameCount().block(block).call().await?;
        if game_count == U256::ZERO {
            return Ok(None);
        }
        Ok(Some(game_count - U256::from(1)))
    }

    async fn fetch_game_address_by_index(
        &self,
        game_index: U256,
    ) -> Result<alloy_primitives::Address> {
        let game = self.gameAtIndex(game_index).call().await?;
        Ok(game.proxy_)
    }
}

/// Returns whether the parent game is resolved (not `IN_PROGRESS`).
///
/// A `u32::MAX` parent index denotes an anchor-rooted game; the contract's
/// `getParentGameStatus` treats it as `DEFENDER_WINS`, so it counts as
/// resolved here too.
pub async fn is_parent_resolved<P>(
    parent_index: u32,
    factory: &DisputeGameFactoryInstance<P>,
    pinned_block: BlockId,
) -> Result<bool>
where
    P: Provider + Clone,
{
    if parent_index == u32::MAX {
        return Ok(true);
    }
    let parent_address = factory.fetch_game_address_by_index(U256::from(parent_index)).await?;
    let parent = ZKDisputeGame::new(parent_address, factory.provider().clone());
    let status = parent.status().block(pinned_block).call().await?;
    let status = GameStatus::try_from(status).context("invalid parent game status")?;
    Ok(status != GameStatus::IN_PROGRESS)
}

/// Prefix used for transaction revert errors.
pub const TX_REVERTED_PREFIX: &str = "transaction reverted:";

/// Extension trait for checking transaction error types.
pub trait TxErrorExt {
    /// Whether the error is a transaction revert.
    fn is_revert(&self) -> bool;
}

impl TxErrorExt for anyhow::Error {
    fn is_revert(&self) -> bool {
        self.to_string().starts_with(TX_REVERTED_PREFIX)
    }
}
