//! Command that initializes the OP proofs storage with the current state of the chain.

use clap::Parser;
use reth_chainspec::ChainInfo;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_primitives::OpPrimitives;
use reth_optimism_trie::{
    db::MdbxProofsStorage, InitializationJob, OpProofsStorage, OpProofsStore,
};
use reth_provider::{BlockNumReader, DBProvider, DatabaseProviderFactory};
use std::{path::PathBuf, sync::Arc};
use tracing::info;

/// Initializes the proofs storage with the current state of the chain.
///
/// This command must be run before starting the node with proofs history enabled.
/// It backfills the proofs storage with trie nodes from the current chain state.
#[derive(Debug, Parser)]
pub struct InitCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// The path to the storage DB for proofs history.
    ///
    /// This should match the path used when starting the node with
    /// `--proofs-history.storage-path`.
    #[arg(
        long = "proofs-history.storage-path",
        value_name = "PROOFS_HISTORY_STORAGE_PATH",
        required = true
    )]
    pub storage_path: PathBuf,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> InitCommand<C> {
    /// Execute `initialize-op-proofs` command
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", version_metadata().short_version);
        info!(target: "reth::cli", "Initializing OP proofs storage at: {:?}", self.storage_path);

        // Initialize the environment with read-only access
        let Environment { provider_factory, .. } = self.env.init::<N>(AccessRights::RO)?;

        // Create the proofs storage
        let storage: OpProofsStorage<Arc<MdbxProofsStorage>> = Arc::new(
            MdbxProofsStorage::new(&self.storage_path)
                .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
        )
        .into();

        // Check if already initialized
        if let Some((block_number, block_hash)) = storage.get_earliest_block_number().await? {
            info!(
                target: "reth::cli",
                block_number = block_number,
                block_hash = ?block_hash,
                "Proofs storage already initialized"
            );
            return Ok(());
        }

        // Get the current chain state
        let ChainInfo { best_number, best_hash, .. } = provider_factory.chain_info()?;

        info!(
            target: "reth::cli",
            best_number = best_number,
            best_hash = ?best_hash,
            "Starting backfill job for current chain state"
        );

        // Run the backfill job
        {
            let db_provider =
                provider_factory.database_provider_ro()?.disable_long_read_transaction_safety();
            let db_tx = db_provider.into_tx();

            InitializationJob::new(storage.clone(), db_tx).run(best_number, best_hash).await?;
        }

        info!(
            target: "reth::cli",
            best_number = best_number,
            best_hash = ?best_hash,
            "Proofs storage initialized successfully"
        );

        Ok(())
    }
}

impl<C: ChainSpecParser> InitCommand<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
