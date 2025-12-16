//! Command that unwinds the OP proofs storage to a specific block number.

use clap::Parser;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_primitives::OpPrimitives;
use reth_optimism_trie::{db::MdbxProofsStorage, OpProofsStorage, OpProofsStore};
use reth_provider::{BlockReader, TransactionVariant};
use std::{path::PathBuf, sync::Arc};
use tracing::{info, warn};

/// Unwinds the proofs storage to a specific block number.
///
/// This command removes all proof history and state updates after the target block number.
#[derive(Debug, Parser)]
pub struct UnwindOpProofsCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// The path to the storage DB for proofs history.
    #[arg(
        long = "proofs-history.storage-path",
        value_name = "PROOFS_HISTORY_STORAGE_PATH",
        required = true
    )]
    pub storage_path: PathBuf,

    /// The target block number to unwind to.
    ///
    /// All history *after* this block will be removed.
    #[arg(long, value_name = "TARGET_BLOCK")]
    pub target: u64,
}

impl<C: ChainSpecParser> UnwindOpProofsCommand<C> {
    /// Validates that the target block number is within a valid range for unwinding.
    async fn validate_unwind_range<Store: OpProofsStore>(
        &self,
        storage: &OpProofsStorage<Store>,
    ) -> eyre::Result<bool> {
        let (Some((earliest, _)), Some((latest, _))) =
            (storage.get_earliest_block_number().await?, storage.get_latest_block_number().await?)
        else {
            warn!(target: "reth::cli", "No blocks found in proofs storage. Nothing to unwind.");
            return Ok(false);
        };

        if self.target <= earliest {
            warn!(target: "reth::cli", unwind_target = ?self.target, ?earliest, "Target block is less than the earliest block in proofs storage. Nothing to unwind.");
            return Ok(false);
        }

        if self.target > latest {
            warn!(target: "reth::cli", unwind_target = ?self.target, ?latest, "Target block is not less than the latest block in proofs storage. Nothing to unwind.");
            return Ok(false);
        }

        Ok(true)
    }
}

impl<C: ChainSpecParser<ChainSpec = OpChainSpec>> UnwindOpProofsCommand<C> {
    /// Execute [`UnwindOpProofsCommand`].
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec, Primitives = OpPrimitives>>(
        self,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", version_metadata().short_version);
        info!(target: "reth::cli", "Unwinding OP proofs storage at: {:?}", self.storage_path);

        // Initialize the environment with read-only access
        let Environment { provider_factory, .. } = self.env.init::<N>(AccessRights::RO)?;

        // Create the proofs storage
        let storage: OpProofsStorage<Arc<MdbxProofsStorage>> = Arc::new(
            MdbxProofsStorage::new(&self.storage_path)
                .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
        )
        .into();

        // Validate that the target block is within a valid range for unwinding
        if !self.validate_unwind_range(&storage).await? {
            return Ok(());
        }

        // Get the target block from the main database
        let block = provider_factory
            .recovered_block(self.target.into(), TransactionVariant::NoHash)?
            .ok_or_else(|| {
                eyre::eyre!("Target block {} not found in the main database", self.target)
            })?;

        info!(target: "reth::cli", block_number = block.number, block_hash = %block.hash(), "Unwinding to target block");
        storage.unwind_history(block.block_with_parent()).await?;

        Ok(())
    }
}

impl<C: ChainSpecParser> UnwindOpProofsCommand<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
