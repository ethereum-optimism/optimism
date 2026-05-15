//! Command that builds the OP proofs trie-state snapshot at a target block.

use clap::Parser;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::args::ProofsStorageVersion;
use reth_optimism_primitives::OpPrimitives;
use reth_optimism_trie::{SnapshotInitJob, db::MdbxProofsStorageV2};
use reth_provider::{DBProvider, DatabaseProviderFactory};
use std::{path::PathBuf, sync::Arc};
use tracing::info;

/// Builds the snapshot at `--target-block` (defaults to `latest`) and marks it `Ready`.
#[derive(Debug, Parser)]
pub struct SnapshotInitCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// The path to the storage DB for proofs history.
    #[arg(
        long = "proofs-history.storage-path",
        value_name = "PROOFS_HISTORY_STORAGE_PATH",
        required = true
    )]
    pub storage_path: PathBuf,

    /// Target block for the snapshot anchor. Must fall inside the proofs
    /// window `[earliest, latest]`. Defaults to `latest`.
    #[arg(long = "proofs-history.snapshot-target-block", value_name = "TARGET_BLOCK")]
    pub target_block: u64,

    /// Storage schema version. Snapshot is only supported on v2.
    #[arg(
        long = "proofs-history.storage-version",
        value_name = "PROOFS_HISTORY_STORAGE_VERSION",
        default_value = "v1"
    )]
    pub storage_version: ProofsStorageVersion,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> SnapshotInitCommand<C> {
    /// Execute [`SnapshotInitCommand`].
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", version_metadata().short_version);
        info!(target: "reth::cli", "Initializing OP proofs snapshot at: {:?}", self.storage_path);

        let Environment { provider_factory, .. } = self.env.init::<N>(AccessRights::RO, runtime)?;

        match self.storage_version {
            ProofsStorageVersion::V1 => Err(eyre::eyre!(
                "Snapshot is not supported for V1 proofs storage. \
                 Re-run with --proofs-history.storage-version v2."
            )),
            ProofsStorageVersion::V2 => {
                let storage: Arc<MdbxProofsStorageV2> = Arc::new(
                    MdbxProofsStorageV2::new(&self.storage_path)
                        .map_err(|e| eyre::eyre!("Failed to open MdbxProofsStorageV2: {e}"))?,
                );

                let provider = provider_factory
                    .database_provider_ro()
                    .map_err(|e| eyre::eyre!("Failed to open reth DB provider: {e}"))?
                    .disable_long_read_transaction_safety();

                let outcome = SnapshotInitJob::new(provider, storage).run(self.target_block)?;
                info!(
                    target: "reth::cli",
                    anchor = ?outcome.block,
                    status = ?outcome.status,
                    account_nodes_copied = outcome.account_nodes_copied,
                    storage_nodes_copied = outcome.storage_nodes_copied,
                    "Snapshot init complete"
                );
                Ok(())
            }
        }
    }
}

impl<C: ChainSpecParser> SnapshotInitCommand<C> {
    /// Returns the underlying chain being used to run this command.
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
