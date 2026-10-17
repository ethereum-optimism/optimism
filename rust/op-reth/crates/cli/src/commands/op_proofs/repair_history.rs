//! Command that repairs history shard keys damaged by unwinds before
//! ethereum-optimism/optimism#23334.

use clap::Parser;
use reth_chainspec::EthChainSpec;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_node::args::ProofsHistoryStorageArgs;
use reth_optimism_trie::db::MdbxProofsStorage;
use std::sync::Arc;
use tracing::info;

/// Repairs proofs-history shard keys left stale by reorgs and unwinds before the unwind fix.
///
/// A stale key makes `eth_getProof` silently serve chain-tip values for a band of historical
/// blocks. Only shard keys are rewritten, in a single transaction; run this, including
/// `--dry-run`, with the node stopped.
#[derive(Debug, Parser)]
pub struct RepairHistoryCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// Shared proofs-history storage flags (storage path + version).
    #[command(flatten)]
    pub history: ProofsHistoryStorageArgs,

    /// Only report how many shards need repair; do not modify the storage.
    ///
    /// The scan still holds the storage's write lock, so the node must be stopped for this too.
    #[arg(long)]
    pub dry_run: bool,
}

impl<C: ChainSpecParser<ChainSpec: EthChainSpec>> RepairHistoryCommand<C> {
    /// Execute [`RepairHistoryCommand`].
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

        let storage = MdbxProofsStorage::new(&storage_path)
            .map_err(|e| eyre::eyre!("Failed to open MdbxProofsStorage: {e}"))?;
        let counts = if self.dry_run {
            storage.check_history_shard_keys()?
        } else {
            storage.repair_history_shard_keys()?
        };

        info!(
            target: "reth::cli",
            dry_run = self.dry_run,
            account_trie = counts.account_trie,
            storage_trie = counts.storage_trie,
            hashed_accounts = counts.hashed_accounts,
            hashed_storages = counts.hashed_storages,
            total = counts.total(),
            "{}",
            match (self.dry_run, counts.total()) {
                (_, 0) => "History shard keys are well formed",
                (true, _) => "History shards need re-keying; rerun without --dry-run to repair",
                (false, _) => "Re-keyed history shards",
            }
        );
        Ok(())
    }
}

impl<C: ChainSpecParser> RepairHistoryCommand<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
