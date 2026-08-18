//! Proposer service for super-root ZK dispute games.
//!
//! Handles super-root claims sourced from a supernode, `parentIndex || superRootProof`
//! extra data, prestate-based ownership, two-phase `DelayedWETH` bond claims, and defense
//! of challenged games in the owned set. Witness collection and native output computation
//! live in [`proving`]; SP1 proof providers live in [`prover`].

/// Prefix for all proposer-owned environment variables.
pub const ENV_VAR_PREFIX: &str = "KONA_SP1_PROPOSER";

/// Builds a proposer-owned environment-variable name.
pub fn env_var(suffix: &str) -> String {
    kona_sp1_host_utils::prefixed_env_var(ENV_VAR_PREFIX, suffix)
}

pub mod config;
pub mod contract;
pub mod metrics;
pub mod proposer;
pub mod prover;
pub mod proving;
pub mod signer;
pub mod superroot;

mod adapters;
mod ports;

use alloy_eips::BlockId;
use alloy_primitives::U256;
use alloy_provider::{Provider, RootProvider};
use anyhow::{Context, Result};
use async_trait::async_trait;

use crate::contract::{DisputeGameFactory::DisputeGameFactoryInstance, GameStatus, ZKDisputeGame};

/// The L1 provider type used throughout the proposer (a plain alloy root provider).
pub type L1Provider = RootProvider;

/// The dispute game type this proposer plays. Game type 10 is reserved as the
/// ZK dispute game type for all OP Stack networks; not configurable to avoid
/// misconfiguration.
pub const ZK_GAME_TYPE: u32 = 10;

/// Read-only view of the `DisputeGameFactory` contract used during sync and game creation.
#[async_trait]
pub trait FactoryTrait<P>
where
    P: Provider + Clone,
{
    /// Fetches the bond required to create a game.
    async fn fetch_init_bond(&self, game_type: u32) -> Result<U256>;

    /// Fetches the latest game index.
    async fn fetch_latest_game_index(&self, block: BlockId) -> Result<Option<U256>>;

    /// Fetches the game address by index at the given block.
    async fn fetch_game_address_by_index(
        &self,
        game_index: U256,
        block: BlockId,
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
        block: BlockId,
    ) -> Result<alloy_primitives::Address> {
        let game = self.gameAtIndex(game_index).block(block).call().await?;
        Ok(game.proxy_)
    }
}

/// Returns whether the parent game resolved in the defender's favor,
/// making its children eligible for resolution.
///
/// A `u32::MAX` parent index denotes an anchor-rooted game; the contract's
/// `getParentGameStatus` treats it as `DefenderWins`, so it counts as
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
    let parent_address =
        factory.fetch_game_address_by_index(U256::from(parent_index), pinned_block).await?;
    let parent = ZKDisputeGame::new(parent_address, factory.provider().clone());
    let status = parent.status().block(pinned_block).call().await?;
    let status = GameStatus::try_from(status).context("invalid parent game status")?;
    Ok(status == GameStatus::DefenderWins)
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

#[cfg(test)]
mod tests {
    use super::{TX_REVERTED_PREFIX, TxErrorExt};

    /// `is_revert` matches on the OUTERMOST rendering. Context added above a
    /// bail site defeats the prefix check - pinned here so any refactor of
    /// `is_revert`'s matching (e.g. switching to `chain()` traversal) must
    /// revisit this contract. Typed error lands with #22019.
    #[test]
    fn revert_detection_pins_prefix_rendering() {
        let revert = anyhow::anyhow!("{TX_REVERTED_PREFIX} receipt");
        assert!(revert.is_revert());
        let wrapped = revert.context("submitting resolution");
        assert!(!wrapped.is_revert());
        assert!(!anyhow::anyhow!("other failure").is_revert());
    }
}
