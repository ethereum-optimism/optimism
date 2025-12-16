//! Command that prunes the OP proofs storage.

use clap::Parser;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_primitives::OpPrimitives;
use reth_optimism_trie::{
    db::MdbxProofsStorage, OpProofStoragePruner, OpProofsStorage, OpProofsStore,
};
use std::{path::PathBuf, sync::Arc};
use tracing::info;

/// Prunes the proofs storage by removing old proof history and state updates.
#[derive(Debug, Parser)]
pub struct PruneOpProofsCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// The path to the storage DB for proofs history.
    #[arg(
        long = "proofs-history.storage-path",
        value_name = "PROOFS_HISTORY_STORAGE_PATH",
        required = true
    )]
    pub storage_path: PathBuf,

    /// The window to span blocks for proofs history. Value is the number of blocks.
    /// Default is 1 month of blocks based on 2 seconds block time.
    /// 30 * 24 * 60 * 60 / 2 = `1_296_000`
    #[arg(
        long = "proofs-history.window",
        default_value_t = 1_296_000,
        value_name = "PROOFS_HISTORY_WINDOW"
    )]
    pub proofs_history_window: u64,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> PruneOpProofsCommand<C> {
    /// Execute [`PruneOpProofsCommand`].
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", version_metadata().short_version);
        info!(target: "reth::cli", "Pruning OP proofs storage at: {:?}", self.storage_path);

        // Initialize the environment with read-only access
        let Environment { provider_factory, .. } = self.env.init::<N>(AccessRights::RO)?;

        let storage: OpProofsStorage<Arc<MdbxProofsStorage>> = Arc::new(
            MdbxProofsStorage::new(&self.storage_path)
                .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
        )
        .into();

        let earliest_block = storage.get_earliest_block_number().await?;
        let latest_block = storage.get_latest_block_number().await?;
        info!(
            target: "reth::cli",
            ?earliest_block,
            ?latest_block,
            "Current proofs storage block range"
        );

        let pruner =
            OpProofStoragePruner::new(storage, provider_factory, self.proofs_history_window);
        pruner.run().await;
        Ok(())
    }
}

impl<C: ChainSpecParser> PruneOpProofsCommand<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
