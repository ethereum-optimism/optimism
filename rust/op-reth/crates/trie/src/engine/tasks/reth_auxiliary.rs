//! Bytecode and block-hash reads without constructing reth's historical state provider.

use alloy_eips::{BlockHashOrNumber, BlockNumHash};
use alloy_primitives::{B256, BlockNumber};
use reth_chain_state::{BlockState, CanonicalInMemoryStateProvider};
use reth_db::{tables, transaction::DbTx};
use reth_primitives_traits::Bytecode;
use reth_provider::{
    BlockHashReader, BytecodeReader, DBProvider, DatabaseProviderFactory, ProviderResult,
};
use std::sync::Arc;

/// Reads the bytecode and block hashes needed to execute the child of `parent`.
///
/// reth's in-memory chain ending at `parent` is captured once, on construction. reth removes a
/// block from memory only after persisting it, so every block older than the snapshot was already
/// on disk when it was taken. Reads check the snapshot first and then the database. In-memory
/// reads stay on `parent`'s ancestry if reth reorgs after construction. If `parent` was already
/// off reth's canonical chain at construction, it has no snapshot, and code it deployed before
/// persistence is unavailable; execution then fails the state-root check rather than committing.
pub(super) struct RethAuxiliaryProvider<'a, P: CanonicalInMemoryStateProvider> {
    provider: &'a P,
    parent: BlockNumHash,
    /// reth's in-memory chain ending at `parent`, if `parent` was in memory.
    in_memory: Option<Arc<BlockState<P::Primitives>>>,
}

impl<'a, P: CanonicalInMemoryStateProvider> RethAuxiliaryProvider<'a, P> {
    pub(super) fn new(provider: &'a P, parent: BlockNumHash) -> Self {
        let in_memory = provider.canonical_in_memory_state().state_by_hash(parent.hash);
        Self { provider, parent, in_memory }
    }
}

impl<P> BlockHashReader for RethAuxiliaryProvider<'_, P>
where
    P: CanonicalInMemoryStateProvider + BlockHashReader,
{
    fn block_hash(&self, number: BlockNumber) -> ProviderResult<Option<B256>> {
        if number > self.parent.number {
            return Ok(None);
        }
        if number == self.parent.number {
            return Ok(Some(self.parent.hash));
        }
        if let Some(state) = self
            .in_memory
            .as_ref()
            .and_then(|state| state.block_on_chain(BlockHashOrNumber::Number(number)))
        {
            return Ok(Some(state.hash()));
        }
        // Below the snapshot, the block was persisted before the snapshot was taken, so reth's
        // canonical hash at this height is `parent`'s ancestor unless reth has since unwound
        // persisted blocks. A wrong hash can only change BLOCKHASH results, which the state-root
        // check catches.
        self.provider.block_hash(number)
    }

    fn canonical_hashes_range(
        &self,
        start: BlockNumber,
        end: BlockNumber,
    ) -> ProviderResult<Vec<B256>> {
        let end = end.min(self.parent.number.saturating_add(1));
        let mut hashes = Vec::with_capacity(end.saturating_sub(start) as usize);
        for number in start..end {
            let Some(hash) = self.block_hash(number)? else { break };
            hashes.push(hash);
        }
        Ok(hashes)
    }
}

impl<P> BytecodeReader for RethAuxiliaryProvider<'_, P>
where
    P: CanonicalInMemoryStateProvider + DatabaseProviderFactory,
{
    fn bytecode_by_hash(&self, code_hash: &B256) -> ProviderResult<Option<Bytecode>> {
        // Code deployed by a block in the snapshot may not be on disk yet.
        if let Some(code) =
            self.in_memory.iter().flat_map(|state| state.chain()).find_map(|state| {
                state.block_ref().execution_outcome().state.contracts.get(code_hash).cloned()
            })
        {
            return Ok(Some(Bytecode(code)));
        }
        // Bytecode is persisted with execution data, independently of the trie frontier.
        let provider = self.provider.database_provider_ro()?;
        Ok(provider.tx_ref().get::<tables::Bytecodes>(*code_hash)?)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use alloy_primitives::{Bytes, keccak256};
    use reth_chain_state::{CanonicalInMemoryState, ExecutedBlock, NewCanonicalChain};
    use reth_db::transaction::DbTxMut;
    use reth_ethereum_primitives::{Block, EthPrimitives, Receipt};
    use reth_primitives_traits::{RecoveredBlock, SealedBlock};
    use reth_provider::{
        BlockExecutionOutput, NodePrimitivesProvider, test_utils::create_test_provider_factory,
    };

    /// Stands in for reth's `BlockchainProvider`. Block hashes come from reth's current in-memory
    /// chain, then from the database, which reports `persisted_hash` at every height.
    struct Reader<F> {
        factory: F,
        state: CanonicalInMemoryState,
        persisted_hash: B256,
    }

    impl<F> Reader<F> {
        fn new(factory: F) -> Self {
            Self {
                factory,
                state: CanonicalInMemoryState::empty(),
                persisted_hash: B256::repeat_byte(0xcc),
            }
        }
    }

    impl<F> NodePrimitivesProvider for Reader<F> {
        type Primitives = EthPrimitives;
    }

    impl<F: Send + Sync> CanonicalInMemoryStateProvider for Reader<F> {
        fn canonical_in_memory_state(&self) -> CanonicalInMemoryState {
            self.state.clone()
        }
    }

    impl<F: DatabaseProviderFactory> DatabaseProviderFactory for Reader<F> {
        type DB = F::DB;
        type Provider = F::Provider;
        type ProviderRW = F::ProviderRW;

        fn database_provider_ro(&self) -> ProviderResult<Self::Provider> {
            self.factory.database_provider_ro()
        }

        fn database_provider_rw(&self) -> ProviderResult<Self::ProviderRW> {
            self.factory.database_provider_rw()
        }
    }

    impl<F> BlockHashReader for Reader<F> {
        fn block_hash(&self, number: BlockNumber) -> ProviderResult<Option<B256>> {
            Ok(Some(self.state.hash_by_number(number).unwrap_or(self.persisted_hash)))
        }

        fn canonical_hashes_range(
            &self,
            start: BlockNumber,
            end: BlockNumber,
        ) -> ProviderResult<Vec<B256>> {
            (start..end).map(|number| Ok(self.block_hash(number)?.unwrap())).collect()
        }
    }

    fn contract() -> (B256, Bytecode) {
        let code = Bytecode::new_raw(Bytes::from_static(&[0x60, 0x01, 0x60, 0x00, 0x55, 0x00]));
        (keccak256(code.original_bytes()), code)
    }

    fn executed_block(
        number: BlockNumber,
        parent_hash: B256,
        contracts: impl IntoIterator<Item = (B256, Bytecode)>,
    ) -> ExecutedBlock {
        let header = Header { number, parent_hash, ..Default::default() };
        let block = SealedBlock::seal_slow(Block { header, body: Default::default() });
        let mut output = BlockExecutionOutput::<Receipt> {
            result: Default::default(),
            state: Default::default(),
        };
        output.state.contracts.extend(contracts.into_iter().map(|(hash, code)| (hash, code.0)));
        ExecutedBlock::new(
            Arc::new(RecoveredBlock::new_sealed(block, Vec::new())),
            Arc::new(output),
            Default::default(),
        )
    }

    /// Commits blocks `1..=3` to reth's in-memory state, deploying `contract()` in block 2.
    fn commit_in_memory_chain(state: &CanonicalInMemoryState) -> Vec<ExecutedBlock> {
        let first = executed_block(1, B256::repeat_byte(0xaa), []);
        let second = executed_block(2, first.recovered_block().hash(), [contract()]);
        let third = executed_block(3, second.recovered_block().hash(), []);
        let blocks = vec![first, second, third];
        state.update_chain(NewCanonicalChain::Commit { new: blocks.clone() });
        blocks
    }

    #[test]
    fn block_hashes_follow_parent_ancestry_across_reorg() {
        let reader = Reader::new(create_test_provider_factory());
        let blocks = commit_in_memory_chain(&reader.state);
        let hashes: Vec<_> = blocks.iter().map(|block| block.recovered_block().hash()).collect();
        let auxiliary = RethAuxiliaryProvider::new(&reader, blocks[2].recovered_block().num_hash());

        // reth reorgs blocks 2 and 3 onto another fork after the snapshot is taken.
        let fork_second = executed_block(2, hashes[0], []);
        let fork_third = executed_block(3, fork_second.recovered_block().hash(), []);
        let fork_hashes =
            [fork_second.recovered_block().hash(), fork_third.recovered_block().hash()];
        reader.state.update_chain(NewCanonicalChain::Reorg {
            new: vec![fork_second, fork_third],
            old: blocks[1..].to_vec(),
        });
        assert_eq!(reader.block_hash(2).unwrap(), Some(fork_hashes[0]));

        // In-memory heights still come from the parent's chain, not reth's new canonical chain.
        assert_eq!(auxiliary.block_hash(1).unwrap(), Some(hashes[0]));
        assert_eq!(auxiliary.block_hash(2).unwrap(), Some(hashes[1]));
        assert_eq!(auxiliary.block_hash(3).unwrap(), Some(hashes[2]));
        assert_eq!(auxiliary.block_hash(4).unwrap(), None);
        // Below the snapshot, reth's persisted canonical chain is the parent's ancestry.
        assert_eq!(auxiliary.block_hash(0).unwrap(), Some(reader.persisted_hash));
        assert_eq!(
            auxiliary.canonical_hashes_range(0, u64::MAX).unwrap(),
            [reader.persisted_hash, hashes[0], hashes[1], hashes[2]]
        );
    }

    #[test]
    fn in_memory_parent_reads_persisted_bytecode() {
        let reader = Reader::new(create_test_provider_factory());
        let blocks = commit_in_memory_chain(&reader.state);
        let auxiliary = RethAuxiliaryProvider::new(&reader, blocks[2].recovered_block().num_hash());

        // Code deployed below the in-memory window is only on disk.
        let code = Bytecode::new_raw(Bytes::from_static(&[0x00]));
        let hash = keccak256(code.original_bytes());
        let provider = reader.database_provider_rw().unwrap();
        provider.tx_ref().put::<tables::Bytecodes>(hash, code.clone()).unwrap();
        provider.commit().unwrap();

        assert_eq!(auxiliary.bytecode_by_hash(&hash).unwrap(), Some(code));
        let (in_memory_hash, in_memory_code) = contract();
        assert_eq!(auxiliary.bytecode_by_hash(&in_memory_hash).unwrap(), Some(in_memory_code));
    }

    #[test]
    fn snapshot_outlives_eviction() {
        let reader = Reader::new(create_test_provider_factory());
        let blocks = commit_in_memory_chain(&reader.state);
        let parent = blocks[2].recovered_block().num_hash();
        let auxiliary = RethAuxiliaryProvider::new(&reader, parent);

        // reth persists and evicts the chain after the snapshot is taken. The test database never
        // receives the bytecode, so it can only come from the snapshot.
        reader.state.remove_persisted_blocks(parent);
        assert!(reader.state.state_by_hash(parent.hash).is_none());

        let (hash, code) = contract();
        assert_eq!(auxiliary.bytecode_by_hash(&hash).unwrap(), Some(code));
        assert_eq!(auxiliary.block_hash(2).unwrap(), Some(blocks[1].recovered_block().hash()));
    }

    #[test]
    fn persisted_parent_reads_database() {
        let reader = Reader::new(create_test_provider_factory());
        let (hash, code) = contract();
        let parent = BlockNumHash::new(10, B256::repeat_byte(0x10));
        let auxiliary = RethAuxiliaryProvider::new(&reader, parent);
        assert_eq!(auxiliary.bytecode_by_hash(&hash).unwrap(), None);

        let provider = reader.database_provider_rw().unwrap();
        provider.tx_ref().put::<tables::Bytecodes>(hash, code.clone()).unwrap();
        provider.commit().unwrap();

        assert_eq!(auxiliary.bytecode_by_hash(&hash).unwrap(), Some(code));
        assert_eq!(auxiliary.block_hash(10).unwrap(), Some(parent.hash));
        assert_eq!(auxiliary.block_hash(9).unwrap(), Some(reader.persisted_hash));
    }
}
