//! Command that prunes the OP proofs storage.

use clap::Parser;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::args::{
    ProofsHistoryStorageArgs, ProofsHistoryWindowArg, ProofsStorageVersion,
};
use reth_optimism_primitives::OpPrimitives;
use reth_optimism_trie::{
    OpProofStoragePruner, OpProofsProviderRO, OpProofsStore,
    db::{MdbxProofsStorage, MdbxProofsStorageV2},
};
use std::sync::Arc;
use tracing::info;

/// Prunes the proofs storage by removing old proof history and state updates.
#[derive(Debug, Parser)]
pub struct PruneCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// Shared proofs-history storage flags (storage path + version).
    #[command(flatten)]
    pub history: ProofsHistoryStorageArgs,

    /// Shared proofs-history retention window (in blocks).
    #[command(flatten)]
    pub proofs_history_window: ProofsHistoryWindowArg,

    /// The batch size for pruning operations.
    #[arg(
        long = "proofs-history.prune-batch-size",
        default_value_t = 1000,
        value_name = "PROOFS_HISTORY_PRUNE_BATCH_SIZE"
    )]
    pub proofs_history_prune_batch_size: u64,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> PruneCommand<C> {
    /// Execute [`PruneCommand`].
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", version_metadata().short_version);

        // Initialize the environment with read-only access. We use `RoInconsistent` to skip the
        // static-file/database consistency check.
        let Environment { provider_factory, data_dir, .. } =
            self.env.init::<N>(AccessRights::RoInconsistent, runtime)?;
        let storage_path = self.history.resolve_storage_path(data_dir.as_ref());
        info!(target: "reth::cli", "Pruning OP proofs storage at: {:?}", storage_path);

        match self.history.storage_version {
            ProofsStorageVersion::V1 => {
                let storage: Arc<MdbxProofsStorage> = Arc::new(
                    MdbxProofsStorage::new(&storage_path)
                        .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
                );
                Self::run_prune(
                    storage,
                    provider_factory,
                    self.proofs_history_window.window,
                    self.proofs_history_prune_batch_size,
                )?;
            }
            ProofsStorageVersion::V2 => {
                let storage: Arc<MdbxProofsStorageV2> = Arc::new(
                    MdbxProofsStorageV2::new(&storage_path)
                        .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorageV2: {e}"))?,
                );
                Self::run_prune(
                    storage,
                    provider_factory,
                    self.proofs_history_window.window,
                    self.proofs_history_prune_batch_size,
                )?;
            }
        }

        Ok(())
    }

    /// Run the pruner against the given proofs storage.
    fn run_prune(
        storage: impl OpProofsStore,
        block_hash_reader: impl reth_provider::BlockHashReader,
        proofs_history_window: u64,
        prune_batch_size: u64,
    ) -> eyre::Result<()> {
        let window = storage.provider_ro()?.get_proof_window()?;
        info!(
            target: "reth::cli",
            earliest_block = ?window.earliest,
            latest_block = ?window.latest,
            "Current proofs storage block range"
        );

        let pruner = OpProofStoragePruner::new(storage, block_hash_reader, proofs_history_window)
            .with_batch_size(prune_batch_size);
        pruner.run();
        Ok(())
    }
}

impl<C: ChainSpecParser> PruneCommand<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
