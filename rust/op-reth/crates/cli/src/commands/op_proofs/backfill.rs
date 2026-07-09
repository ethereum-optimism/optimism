//! Command that backfills OP proofs storage to an older earliest block.

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
    BackfillJob, OpProofsBackfillStore, OpProofsProviderRO, db::MdbxProofsStorageV2,
};
use reth_provider::{
    BlockHashReader, BlockNumReader, ChangeSetReader, DBProvider, DatabaseProviderFactory,
    HeaderProvider, StageCheckpointReader, StorageChangeSetReader, StorageSettingsCache,
};
use std::sync::Arc;
use tracing::info;

/// Backfills the proofs storage to an older earliest block.
#[derive(Debug, Parser)]
pub struct BackfillCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// Shared proofs-history storage flags (storage path + version).
    #[command(flatten)]
    pub history: ProofsHistoryStorageArgs,

    /// Retention window in blocks. Backfill extends the proof window backward
    /// until `earliest <= latest - window`.
    #[command(flatten)]
    pub proofs_history_window: ProofsHistoryWindowArg,

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

        let Environment { provider_factory, data_dir, .. } =
            self.env.init::<N>(AccessRights::RO, runtime)?;
        let storage_path = self.history.resolve_storage_path(data_dir.as_ref());
        info!(target: "reth::cli", "Backfilling OP proofs storage at: {:?}", storage_path);

        match self.history.storage_version {
            ProofsStorageVersion::V1 => {
                return Err(eyre::eyre!(
                    "Backfill is not supported for V1 proofs storage. \
                     Re-run with --proofs-history.storage-version v2."
                ));
            }
            ProofsStorageVersion::V2 => {
                let storage: Arc<MdbxProofsStorageV2> = Arc::new(
                    MdbxProofsStorageV2::new(&storage_path)
                        .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorageV2: {e}"))?,
                );
                Self::run_backfill(
                    &provider_factory,
                    storage,
                    self.proofs_history_window.window,
                    self.use_snapshot,
                )?;
            }
        }

        Ok(())
    }

    fn run_backfill<F, S>(
        provider_factory: &F,
        storage: S,
        window_blocks: u64,
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
        let proof_window = storage.provider_ro()?.get_proof_window()?;
        // Mirror prune's semantics: target `earliest = latest - window`,
        // clamped to 0 if the chain is shorter than the requested window.
        let target_earliest_block = proof_window.latest.number.saturating_sub(window_blocks);
        info!(
            target: "reth::cli",
            earliest = ?proof_window.earliest,
            latest = ?proof_window.latest,
            window_blocks,
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
