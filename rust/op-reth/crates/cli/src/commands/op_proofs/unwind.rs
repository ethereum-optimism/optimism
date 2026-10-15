//! Command that unwinds the OP proofs storage to a specific block number.

use alloy_consensus::{BlockHeader, EMPTY_ROOT_HASH};
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_primitives::B256;
use clap::Parser;
use reth_chainspec::EthChainSpec;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::common::{AccessRights, CliNodeTypes, Environment, EnvironmentArgs};
use reth_node_core::version::version_metadata;
use reth_optimism_node::args::ProofsHistoryStorageArgs;
use reth_optimism_trie::{
    OpProofsProviderRO, OpProofsProviderRw, OpProofsStorageError, OpProofsStore, ProofWindowRange,
    db::MdbxProofsStorage,
};
use reth_primitives_traits::BlockBody;
use reth_provider::{BlockReader, TransactionVariant};
use std::sync::Arc;
use tracing::{info, warn};

/// Unwinds the proofs storage to a specific block number.
///
/// This command removes all proof history and state updates after the target block number, so
/// the target becomes the latest block in the proofs storage. A target at or above the latest
/// block leaves the storage untouched. A target below the earliest block is refused: the earliest
/// block carries the state the storage was initialized or backfilled to, and nothing below it can
/// be reconstructed.
#[derive(Debug, Parser)]
pub struct UnwindCommand<C: ChainSpecParser> {
    #[command(flatten)]
    env: EnvironmentArgs<C>,

    /// Shared proofs-history storage flags (storage path + version).
    #[command(flatten)]
    pub history: ProofsHistoryStorageArgs,

    /// The target block number to unwind to.
    ///
    /// All history *after* this block will be removed; the target block stays and becomes the
    /// latest block in the proofs storage.
    #[arg(long, value_name = "TARGET_BLOCK")]
    pub target: u64,
}

impl<C: ChainSpecParser> UnwindCommand<C> {
    /// Returns the proof window when it holds blocks above the target, and `None` when there is
    /// nothing to unwind. A target below the earliest block is an error.
    fn window_to_unwind<Store: OpProofsStore>(
        &self,
        storage: Store,
    ) -> eyre::Result<Option<ProofWindowRange>> {
        let window = match storage.provider_ro()?.get_proof_window() {
            Ok(window) => window,
            Err(OpProofsStorageError::NoBlocksFound) => {
                warn!(target: "reth::cli", "No blocks found in proofs storage. Nothing to unwind.");
                return Ok(None);
            }
            Err(err) => return Err(err.into()),
        };

        if self.target < window.earliest.number {
            return Err(eyre::eyre!(
                "Target block {} is below the earliest block {} in proofs storage; the storage \
                 holds no state below the block it was initialized or backfilled to. Delete the \
                 storage and run `proofs init` to seed it again instead.",
                self.target,
                window.earliest.number
            ));
        }

        if self.target >= window.latest.number {
            warn!(target: "reth::cli", unwind_target = self.target, latest = window.latest.number, "Target block is not below the latest block in proofs storage. Nothing to unwind.");
            return Ok(None);
        }

        Ok(Some(window))
    }
}

impl<C: ChainSpecParser<ChainSpec: EthChainSpec>> UnwindCommand<C> {
    /// Execute [`UnwindCommand`].
    pub async fn execute<N: CliNodeTypes<ChainSpec = C::ChainSpec>>(
        self,
        runtime: reth_tasks::Runtime,
    ) -> eyre::Result<()> {
        info!(target: "reth::cli", "reth {} starting", version_metadata().short_version);

        // Initialize the environment with read-only access. We use `RoInconsistent` to skip the
        // static-file/database consistency check.
        let Environment { provider_factory, data_dir, .. } =
            self.env.init::<N>(AccessRights::RoInconsistent, runtime)?;
        let storage_path = self.history.resolve_storage_path(data_dir.as_ref());
        info!(target: "reth::cli", "Unwinding OP proofs storage at: {:?}", storage_path);

        let storage: Arc<MdbxProofsStorage> = Arc::new(
            MdbxProofsStorage::new(&storage_path)
                .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
        );
        self.run_unwind(storage, &provider_factory)?;

        Ok(())
    }

    /// Validate the target against the proof window and remove every block above it.
    fn run_unwind<S, F>(&self, storage: S, provider_factory: &F) -> eyre::Result<()>
    where
        S: OpProofsStore + Clone,
        F: BlockReader,
    {
        let Some(window) = self.window_to_unwind(storage.clone())? else {
            return Ok(());
        };

        // Get the target block from the main database
        let block = provider_factory
            .recovered_block(self.target.into(), TransactionVariant::NoHash)?
            .ok_or_else(|| {
                eyre::eyre!("Target block {} not found in the main database", self.target)
            })?;

        // Refuse to unwind when reth has pruned the target block's body. The unwind itself
        // would technically succeed (we only need the target's hash), but re-backfilling
        // afterward is impossible on the same node, so the operator would be left without a
        // way to resume from this point. Fail fast with a clear remediation instead.
        if block.header().transactions_root() != EMPTY_ROOT_HASH &&
            block.body().transactions().is_empty()
        {
            return Err(eyre::eyre!(
                "Target block {} body has been pruned by reth; refusing to unwind because \
                 re-backfilling from here is not possible until reth is re-synced without \
                 body pruning.",
                self.target
            ));
        }

        // The storage removes every block from the given block's number on and keeps the given
        // parent hash as its tip; the removed block's own hash is not part of that contract, and
        // reth does not hold it when the storage is ahead of reth's tip.
        let first_removed_block =
            BlockWithParent::new(block.hash(), NumHash::new(self.target + 1, B256::ZERO));

        info!(
            target: "reth::cli",
            target = self.target,
            target_hash = %block.hash(),
            latest = window.latest.number,
            "Unwinding proofs storage to target block"
        );
        let provider_rw = storage.provider_rw()?;
        provider_rw.unwind_history(first_removed_block)?;
        provider_rw.commit()?;
        info!(
            target: "reth::cli",
            earliest = window.earliest.number,
            latest = self.target,
            "Proofs storage unwound"
        );

        Ok(())
    }
}

impl<C: ChainSpecParser> UnwindCommand<C> {
    /// Returns the underlying chain being used to run this command
    pub const fn chain_spec(&self) -> Option<&Arc<C::ChainSpec>> {
        Some(&self.env.chain)
    }
}
