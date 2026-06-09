//! Command that backfills OP proofs storage to an older earliest block.

use clap::Parser;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::args::ProofsStorageVersion;
use reth_optimism_primitives::OpPrimitives;
use reth_optimism_trie::{
    BackfillJob, OpProofsBackfillStore, OpProofsProviderRO, db::MdbxProofsStorageV2,
};
use reth_provider::{
    BlockHashReader, BlockNumReader, ChangeSetReader, DBProvider, DatabaseProviderFactory,
    HeaderProvider, StageCheckpointReader, StorageChangeSetReader, StorageSettingsCache,
};
use std::{path::PathBuf, sync::Arc};
use tracing::info;

/// Backfills the proofs storage to an older earliest block.
#[derive(Debug, Parser)]
pub struct BackfillCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// The path to the storage DB for proofs history.
    #[arg(
        long = "proofs-history.storage-path",
        value_name = "PROOFS_HISTORY_STORAGE_PATH",
        required = true
    )]
    pub storage_path: PathBuf,

    /// Target earliest block number after backfill.
    #[arg(long = "proofs-history.target-earliest-block", value_name = "TARGET_EARLIEST_BLOCK")]
    pub target_earliest_block: u64,

    /// Storage schema version. Must match the version used when starting the node.
    #[arg(
        long = "proofs-history.storage-version",
        value_name = "PROOFS_HISTORY_STORAGE_VERSION",
        default_value = "v1"
    )]
    pub storage_version: ProofsStorageVersion,

    /// Use the trie-state snapshot to accelerate per-block reads during
    /// backfill. If no snapshot exists, one is built at the current
    /// `earliest` before the backfill loop begins. Requires v2 storage.
    #[arg(long = "proofs-history.use-snapshot")]
    pub use_snapshot: bool,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> BackfillCommand<C> {
    /// Execute [`BackfillCommand`].
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", version_metadata().short_version);
        info!(target: "reth::cli", "Backfilling OP proofs storage at: {:?}", self.storage_path);

        let Environment { provider_factory, .. } = self.env.init::<N>(AccessRights::RO, runtime)?;

        match self.storage_version {
            ProofsStorageVersion::V1 => {
                return Err(eyre::eyre!(
                    "Backfill is not supported for V1 proofs storage. \
                     Re-run with --proofs-history.storage-version v2."
                ));
            }
            ProofsStorageVersion::V2 => {
                let storage: Arc<MdbxProofsStorageV2> = Arc::new(
                    MdbxProofsStorageV2::new(&self.storage_path)
                        .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorageV2: {e}"))?,
                );
                Self::run_backfill(
                    &provider_factory,
                    storage,
                    self.target_earliest_block,
                    self.use_snapshot,
                )?;
            }
        }

        Ok(())
    }

    fn run_backfill<F, S>(
        provider_factory: &F,
        storage: S,
        target_earliest_block: u64,
        use_snapshot: bool,
    ) -> eyre::Result<()>
    where
        F: DatabaseProviderFactory,
        F::Provider: DBProvider
            + StageCheckpointReader
            + ChangeSetReader
            + StorageChangeSetReader
            + BlockNumReader
            + BlockHashReader
            + HeaderProvider
            + StorageSettingsCache
            + Send
            + Sync,
        S: OpProofsBackfillStore + Clone + Send,
    {
        let window = storage.provider_ro()?.get_proof_window()?;
        info!(
            target: "reth::cli",
            earliest = ?window.earliest,
            latest = ?window.latest,
            target_earliest_block,
            use_snapshot,
            "Starting backfill job"
        );

        let provider = provider_factory
            .database_provider_ro()
            .map_err(|e| eyre::eyre!("Failed to open reth DB provider: {e}"))?
            .disable_long_read_transaction_safety();

        let job = BackfillJob::new(provider, storage);
        if use_snapshot {
            job.run_with_snapshot(target_earliest_block)?;
        } else {
            job.run(target_earliest_block)?;
        }
        Ok(())
    }
}

impl<C: ChainSpecParser> BackfillCommand<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
