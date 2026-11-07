//! Command that checks the proofs-history index for shard keys damaged by unwinds before
//! ethereum-optimism/optimism#23334.

use clap::Parser;
use reth_chainspec::EthChainSpec;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_node::args::ProofsHistoryStorageArgs;
use reth_optimism_trie::db::{HISTORY_LAYOUT_VERSION, HistoryLayoutVerdict, MdbxProofsStorage};
use std::sync::Arc;
use tracing::info;

/// Reports the proofs-history layout version and any history shards with stale keys.
///
/// A stale key makes `eth_getProof` silently serve chain-tip values for a band of historical
/// blocks. Starting this release on the storage migrates it once and re-keys those shards. This
/// command inspects the storage without migrating it, so it can run before an upgrade, and exits
/// with an error when it finds damage. `--repair` re-keys the shards regardless of the recorded
/// version, which repairs a storage damaged again by an older release. Run it with the node
/// stopped.
#[derive(Debug, Parser)]
pub struct CheckHistoryCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// Shared proofs-history storage flags (storage path + version).
    #[command(flatten)]
    pub history: ProofsHistoryStorageArgs,

    /// Re-key stale shards instead of only reporting them.
    #[arg(long)]
    pub repair: bool,
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

        if self.repair {
            let rekeyed = MdbxProofsStorage::repair_history_layout_at(&storage_path)?;
            info!(target: "reth::cli", total = rekeyed.total(), "Re-keyed proofs history shards");
            return Ok(());
        }

        let report = MdbxProofsStorage::check_history_layout_at(&storage_path)?;
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

        let total = misplaced.total();
        match report.verdict() {
            HistoryLayoutVerdict::Clean => Ok(()),
            HistoryLayoutVerdict::NeedsMigration => Err(eyre::eyre!(
                "{total} history shards have stale keys; starting this release on the storage \
                 migrates it and re-keys them"
            )),
            HistoryLayoutVerdict::DamagedAfterMigration => Err(eyre::eyre!(
                "{total} history shards have stale keys although the storage is already at layout \
                 version {}, so it was damaged after migrating, most likely by an older release; \
                 rerun with --repair",
                report.version
            )),
            HistoryLayoutVerdict::NewerLayout => Err(eyre::eyre!(
                "the storage records history layout version {}, newer than the {} this release \
                 supports; check it with a newer op-reth",
                report.version,
                HISTORY_LAYOUT_VERSION
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
