//! Command that drops the OP proofs trie-state snapshot.

use clap::Parser;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::args::ProofsStorageVersion;
use reth_optimism_primitives::OpPrimitives;
use reth_optimism_trie::{
    OpProofsBackfillProvider, OpProofsBackfillStore, db::MdbxProofsStorageV2,
};
use std::{path::PathBuf, sync::Arc};
use tracing::info;

/// Wipes the snapshot tables and meta row, returning status to `NotStarted`.
#[derive(Debug, Parser)]
pub struct SnapshotDropCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// The path to the storage DB for proofs history.
    #[arg(
        long = "proofs-history.storage-path",
        value_name = "PROOFS_HISTORY_STORAGE_PATH",
        required = true
    )]
    pub storage_path: PathBuf,

    /// Storage schema version. Snapshot is only supported on v2.
    #[arg(
        long = "proofs-history.storage-version",
        value_name = "PROOFS_HISTORY_STORAGE_VERSION",
        default_value = "v1"
    )]
    pub storage_version: ProofsStorageVersion,
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> SnapshotDropCommand<C> {
    /// Execute [`SnapshotDropCommand`].
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", version_metadata().short_version);
        info!(target: "reth::cli", "Dropping OP proofs snapshot at: {:?}", self.storage_path);

        // Initialize the reth environment for arg consistency with sibling
        // op-proofs commands; drop only touches the proofs storage itself.
        let _ = self.env.init::<N>(AccessRights::RO, runtime)?;

        match self.storage_version {
            ProofsStorageVersion::V1 => Err(eyre::eyre!(
                "Snapshot is not supported for V1 proofs storage. \
                 Re-run with --proofs-history.storage-version v2."
            )),
            ProofsStorageVersion::V2 => {
                let storage = MdbxProofsStorageV2::new(&self.storage_path)
                    .map_err(|e| eyre::eyre!("Failed to open MdbxProofsStorageV2: {e}"))?;

                let sp = storage.backfill_provider()?;
                sp.clear_snapshot()?;
                OpProofsBackfillProvider::commit(sp)?;

                info!(target: "reth::cli", "Snapshot dropped");
                Ok(())
            }
        }
    }
}

impl<C: ChainSpecParser> SnapshotDropCommand<C> {
    /// Returns the underlying chain being used to run this command.
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
