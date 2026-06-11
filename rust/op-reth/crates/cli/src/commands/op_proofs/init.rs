//! Command that initializes the OP proofs storage with the current state of the chain.

use clap::Parser;
use reth_chainspec::ChainInfo;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::args::{
    ProofsHistoryStorageArgs, ProofsHistoryWindowArg, ProofsStorageVersion,
};
use reth_optimism_primitives::OpPrimitives;
use reth_optimism_trie::{
    BackfillJob, InitializationJob, OpProofsProviderRO, OpProofsStorageError, OpProofsStore,
    RethTrieStorageLayout,
    db::{MdbxProofsStorage, MdbxProofsStorageV2},
};
use reth_provider::{BlockNumReader, DBProvider, DatabaseProviderFactory, StorageSettingsCache};
use std::sync::Arc;
use tracing::{debug, info};

/// Initializes the proofs storage with the current state of the chain.
///
/// This command must be run before starting the node with proofs history enabled.
/// It backfills the proofs storage with trie nodes from the current chain state.
#[derive(Debug, Parser)]
pub struct InitCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// Shared proofs-history storage flags (storage path + version).
    #[command(flatten)]
    pub history: ProofsHistoryStorageArgs,

    /// Skip the post-init backward backfill. By default, after the snapshot of
    /// the current chain state is captured the proof window is extended back
    /// by `--proofs-history.window` blocks using the snapshot-accelerated
    /// path. Set this flag to leave the window at `[latest, latest]` and run
    /// `op-proofs backfill` later instead. No effect on V1 storage (V1 does
    /// not support backfill).
    #[arg(long = "proofs-history.skip-backfill")]
    pub skip_backfill: bool,

    /// Retention window in blocks. Backfill extends the proof window backward
    /// until `earliest <= latest - window`.
    #[command(flatten)]
    pub proofs_history_window: ProofsHistoryWindowArg,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> InitCommand<C> {
    /// Execute `initialize-op-proofs` command
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
        info!(target: "reth::cli", "Initializing OP proofs storage at: {:?}", storage_path);

        // Create the proofs storage without the metrics wrapper.
        // During initialization we write billions of entries; the metrics layer's
        // `AtomicBucket::push` (used by `Histogram::record_many`) is append-only and
        // would accumulate ~19 bytes per observation, causing OOM on large chains.
        match self.history.storage_version {
            ProofsStorageVersion::V1 => {
                if !self.skip_backfill {
                    return Err(eyre::eyre!(
                        "V1 proofs storage does not support backfill. \
                         Re-run with --proofs-history.storage-version v2, \
                         or with --proofs-history.skip-backfill to initialize without backfilling."
                    ));
                }

                let storage: Arc<MdbxProofsStorage> = Arc::new(
                    MdbxProofsStorage::new(&storage_path)
                        .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
                );
                Self::run_init(&provider_factory, storage)?;
            }
            ProofsStorageVersion::V2 => {
                let storage: Arc<MdbxProofsStorageV2> = Arc::new(
                    MdbxProofsStorageV2::new(&storage_path)
                        .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorageV2: {e}"))?,
                );
                Self::run_init(&provider_factory, storage.clone())?;

                if !self.skip_backfill {
                    let proof_window = storage.provider_ro()?.get_proof_window()?;
                    let window_blocks = self.proofs_history_window.window;
                    let target_earliest_block =
                        proof_window.earliest.number.saturating_sub(window_blocks);
                    info!(
                        target: "reth::cli",
                        latest = ?proof_window.latest,
                        window_blocks,
                        target_earliest_block,
                        "Running snapshot-accelerated backfill"
                    );
                    let provider = provider_factory
                        .database_provider_ro()
                        .map_err(|e| eyre::eyre!("Failed to open reth DB provider: {e}"))?
                        .disable_long_read_transaction_safety();
                    BackfillJob::new(provider, storage).run_with_snapshot(target_earliest_block)?;
                    info!(target: "reth::cli", "Backfill complete");
                }
            }
        }

        Ok(())
    }

    /// Run the initialization against the given proofs storage.
    ///
    /// If the storage is already initialized this is a no-op.
    fn run_init<F>(provider_factory: &F, storage: impl OpProofsStore) -> eyre::Result<()>
    where
        F: DatabaseProviderFactory + BlockNumReader + StorageSettingsCache,
        F::Provider: DBProvider,
    {
        // Check if already initialized
        match storage.provider_ro()?.get_earliest_block() {
            Ok(anchor) => {
                info!(
                    target: "reth::cli",
                    block_number = anchor.number,
                    block_hash = ?anchor.hash,
                    "Proofs storage already initialized"
                );
                return Ok(());
            }
            Err(OpProofsStorageError::NoBlocksFound) => {
                debug!(target: "reth::cli", "Proofs storage is empty; starting initialization");
            }
            Err(err) => return Err(err.into()),
        }

        // Get the current chain state
        let ChainInfo { best_number, best_hash, .. } = provider_factory.chain_info()?;

        info!(
            target: "reth::cli",
            best_number = best_number,
            best_hash = ?best_hash,
            "Starting backfill job for current chain state"
        );

        {
            let trie_layout = if provider_factory.cached_storage_settings().is_v2() {
                RethTrieStorageLayout::Packed
            } else {
                RethTrieStorageLayout::Legacy
            };
            let db_provider =
                provider_factory.database_provider_ro()?.disable_long_read_transaction_safety();
            let db_tx = db_provider.into_tx();

            InitializationJob::new(storage, db_tx, trie_layout).run(best_number, best_hash)?;
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
