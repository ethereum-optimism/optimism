//! Command that checks the proofs-history index for shard keys damaged by unwinds before
//! ethereum-optimism/optimism#23334.

use clap::Parser;
use reth_chainspec::EthChainSpec;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_node::args::ProofsHistoryStorageArgs;
use reth_optimism_trie::db::{HISTORY_LAYOUT_VERSION, MdbxProofsStorage};
use std::sync::Arc;
use tracing::info;

/// Reports the proofs-history layout version and any history shards with stale keys.
///
/// A stale key makes `eth_getProof` silently serve chain-tip values for a band of historical
/// blocks. Opening the storage with this release migrates it and re-keys those shards; this
/// command inspects the storage without migrating it, so it can be run before an upgrade. It
/// exits with an error when it finds damage. Run it with the node stopped: the scan holds the
/// storage's write lock.
#[derive(Debug, Parser)]
pub struct CheckHistoryCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// Shared proofs-history storage flags (storage path + version).
    #[command(flatten)]
    pub history: ProofsHistoryStorageArgs,
}

impl<C: ChainSpecParser<ChainSpec: EthChainSpec>> CheckHistoryCommand<C> {
    /// Execute [`CheckHistoryCommand`].
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", version_metadata().short_version);

        // Only the data dir is needed; `RoInconsistent` skips the static-file consistency check.
        let Environment { data_dir, .. } =
            self.env.init::<N>(AccessRights::RoInconsistent, runtime)?;
        let storage_path = self.history.resolve_storage_path(data_dir.as_ref());
        info!(target: "reth::cli", "Checking OP proofs history shard keys at: {:?}", storage_path);

        let storage = MdbxProofsStorage::open_unmigrated(&storage_path)
            .map_err(|e| eyre::eyre!("Failed to open MdbxProofsStorage: {e}"))?;
        let report = storage.check_history_layout()?;
        let misplaced = report.misplaced;
        info!(
            target: "reth::cli",
            version = report.version,
            current_version = HISTORY_LAYOUT_VERSION,
            account_trie = misplaced.account_trie,
            storage_trie = misplaced.storage_trie,
            hashed_accounts = misplaced.hashed_accounts,
            hashed_storages = misplaced.hashed_storages,
            total = misplaced.total(),
            "Checked proofs history layout"
        );

        match (misplaced.total(), report.version < HISTORY_LAYOUT_VERSION) {
            (0, _) => Ok(()),
            (total, true) => Err(eyre::eyre!(
                "{total} history shards have stale keys; starting this release on the storage \
                 migrates it and re-keys them"
            )),
            (total, false) => Err(eyre::eyre!(
                "{total} history shards have stale keys although the storage is already at layout \
                 version {}, so it was damaged after migrating, most likely by an older release; \
                 rebuild the proofs storage with `op-reth proofs init`",
                report.version
            )),
        }
    }
}

impl<C: ChainSpecParser> CheckHistoryCommand<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
