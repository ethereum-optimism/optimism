//! [`OpProofsProviderRw`] implementation for [`MdbxProofsProviderV2`].

use super::{MdbxProofsProviderV2, write::HistoryCollector};
use crate::{
    BlockStateDiff, OpProofsStorageError, OpProofsStorageResult,
    api::{OpProofsProviderRw, WriteCounts},
};
use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
use alloy_primitives::{B256, BlockNumber};
use reth_db::transaction::{DbTx, DbTxMut};
use std::fmt::Debug;

impl<TX: DbTxMut + DbTx + Send + Sync + Debug + 'static> OpProofsProviderRw
    for MdbxProofsProviderV2<TX>
{
    fn store_trie_updates(
        &self,
        block_ref: BlockWithParent,
        block_state_diff: BlockStateDiff,
    ) -> OpProofsStorageResult<WriteCounts> {
        let proof_window = self.get_proof_window_inner()?;
        if proof_window.latest.hash != block_ref.parent {
            return Err(OpProofsStorageError::OutOfOrder {
                block_number: block_ref.block.number,
                parent_block_hash: block_ref.parent,
                latest_block_hash: proof_window.latest.hash,
            });
        }

        let mut collector = HistoryCollector::default();
        let counts =
            self.store_block_updates(block_ref.block.number, block_state_diff, &mut collector)?;
        self.flush_collected_history(collector)?;

        self.set_latest_block_number_inner(block_ref.block.number, block_ref.block.hash)?;
        Ok(counts)
    }

    fn store_trie_updates_batch(
        &self,
        updates: Vec<(BlockWithParent, BlockStateDiff)>,
    ) -> OpProofsStorageResult<WriteCounts> {
        let mut total_counts = WriteCounts::default();
        let mut collector = HistoryCollector::default();

        // Track the latest hash in memory instead of reading/writing
        // V2ProofWindow per block (saves 2 cursor opens per block).
        let proof_window = self.get_proof_window_inner()?;
        let mut last_hash = proof_window.latest.hash;
        let mut last_written: Option<(BlockNumber, B256)> = None;

        for (block_ref, block_state_diff) in updates {
            let block_number = block_ref.block.number;

            if last_hash != block_ref.parent {
                return Err(OpProofsStorageError::OutOfOrder {
                    block_number,
                    parent_block_hash: block_ref.parent,
                    latest_block_hash: last_hash,
                });
            }

            let counts =
                self.store_block_updates(block_number, block_state_diff, &mut collector)?;

            last_hash = block_ref.block.hash;
            last_written = Some((block_number, block_ref.block.hash));
            total_counts += counts;
        }

        // Flush all history bitmap entries in one pass — each unique key is
        // seeked, decoded, and re-encoded exactly once regardless of how many
        // blocks in the batch touched it.
        self.flush_collected_history(collector)?;

        // Write V2ProofWindow once at the end instead of per-block.
        if let Some((number, hash)) = last_written {
            self.set_latest_block_number_inner(number, hash)?;
        }

        Ok(total_counts)
    }

    fn prune_earliest_state(
        &self,
        new_earliest_block_ref: BlockWithParent,
    ) -> OpProofsStorageResult<WriteCounts> {
        let target_block = new_earliest_block_ref.block.number;
        let proof_window = self.get_proof_window_inner()?;

        if proof_window.earliest.number >= target_block {
            return Err(OpProofsStorageError::PruneBeyondEarliest {
                target_block_number: target_block,
                earliest_block_number: proof_window.earliest.number,
            });
        }

        let range = (proof_window.earliest.number + 1)..=target_block;

        let counts = self.prune_changesets_and_history(&range)?;

        self.set_earliest_block_number_inner(target_block, new_earliest_block_ref.block.hash)?;

        Ok(counts)
    }

    fn unwind_history(&self, to: BlockWithParent) -> OpProofsStorageResult<()> {
        let proof_window = self.get_proof_window_inner()?;

        if to.block.number > proof_window.latest.number {
            return Ok(());
        }

        if to.block.number <= proof_window.earliest.number {
            return Err(OpProofsStorageError::UnwindBeyondEarliest {
                unwind_block_number: to.block.number,
                earliest_block_number: proof_window.earliest.number,
            });
        }

        let range = to.block.number..=proof_window.latest.number;

        self.unwind_changesets_and_history(&range)?;

        // Update latest block
        self.set_latest_block_number_inner(to.block.number.saturating_sub(1), to.parent)?;

        Ok(())
    }

    fn replace_updates(
        &self,
        latest_common_block: BlockNumHash,
        mut blocks_to_add: Vec<(BlockWithParent, BlockStateDiff)>,
    ) -> OpProofsStorageResult<()> {
        let proof_window = self.get_proof_window_inner()?;

        if latest_common_block.number <= proof_window.earliest.number ||
            latest_common_block.number > proof_window.latest.number
        {
            return Err(OpProofsStorageError::ReorgBaseOutOfWindow {
                block_number: latest_common_block.number,
                earliest_block_number: proof_window.earliest.number,
                latest_block_number: proof_window.latest.number,
            });
        }

        blocks_to_add.sort_unstable_by_key(|(bwp, _)| bwp.block.number);

        // Phase 1: unwind to the latest common block, which is the new base of the proof window.
        {
            let range = (latest_common_block.number + 1)..=proof_window.latest.number;
            self.unwind_changesets_and_history(&range)?;
        }

        // Phase 2: add new blocks on top of the latest common block.
        // Re-add blocks using a shared collector + cursors, same as the batch
        // path, so history bitmap appends are batched and cursors are reused.
        // Track block ordering in memory to avoid per-block V2ProofWindow I/O.
        let mut last_hash = latest_common_block.hash;
        let mut last_written = latest_common_block;
        let mut collector = HistoryCollector::default();

        for (block_ref, diff) in blocks_to_add {
            let block_number = block_ref.block.number;

            if last_hash != block_ref.parent {
                return Err(OpProofsStorageError::OutOfOrder {
                    block_number,
                    parent_block_hash: block_ref.parent,
                    latest_block_hash: last_hash,
                });
            }

            self.store_block_updates(block_number, diff, &mut collector)?;

            last_hash = block_ref.block.hash;
            last_written = NumHash::new(block_number, block_ref.block.hash);
        }

        self.flush_collected_history(collector)?;

        self.set_latest_block_number_inner(last_written.number, last_written.hash)?;

        Ok(())
    }

    fn set_earliest_block_number(
        &self,
        block_number: u64,
        hash: B256,
    ) -> OpProofsStorageResult<()> {
        self.set_earliest_block_number_inner(block_number, hash)
    }

    fn commit(self) -> OpProofsStorageResult<()> {
        self.tx.commit()?;
        Ok(())
    }
}
