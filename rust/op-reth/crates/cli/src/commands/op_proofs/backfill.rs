//! Command that backfills OP proofs storage to an older earliest block.

use clap::Parser;
use reth_chainspec::EthChainSpec;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_node::args::{
    ProofsHistoryBackfillArgs, ProofsHistoryStorageArgs, ProofsHistoryWindowArg,
};
use reth_optimism_trie::{
    BackfillJob, OpProofsBackfillStore, OpProofsProviderRO, db::MdbxProofsStorage,
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

    /// Shared backfill flags (batch size + snapshot toggle).
    #[command(flatten)]
    pub backfill_args: ProofsHistoryBackfillArgs,
}

impl<C: ChainSpecParser<ChainSpec: EthChainSpec>> BackfillCommand<C> {
    /// Execute [`BackfillCommand`].
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", version_metadata().short_version);

        let Environment { provider_factory, data_dir, .. } =
            self.env.init::<N>(AccessRights::RO, runtime)?;
        let storage_path = self.history.resolve_storage_path(data_dir.as_ref());
        info!(target: "reth::cli", "Backfilling OP proofs storage at: {:?}", storage_path);

        let storage: Arc<MdbxProofsStorage> = Arc::new(
            MdbxProofsStorage::new(&storage_path)
                .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
        );
        Self::run_backfill(
            &provider_factory,
            storage,
            self.proofs_history_window.window,
            self.backfill_args.use_snapshot,
            self.backfill_args.backfill_batch_size,
        )?;

        Ok(())
    }

    fn run_backfill<F, S>(
        provider_factory: &F,
        storage: S,
        window_blocks: u64,
        use_snapshot: bool,
        batch_size: usize,
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
            batch_size,
            "Starting backfill job"
        );

        let provider = provider_factory
            .database_provider_ro()
            .map_err(|e| eyre::eyre!("Failed to open reth DB provider: {e}"))?
            .disable_long_read_transaction_safety();

        let job = BackfillJob::new(provider, storage).with_batch_size(batch_size);
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
