//! reth's database and canonical in-memory state, driven the way reth's engine tree and
//! persistence task drive them.

use super::{BlockSpec, create_block_from_spec};
use alloy_eips::BlockNumHash;
use reth_chain_state::{CanonicalInMemoryState, ExecutedBlock, NewCanonicalChain};
use reth_chainspec::ChainSpec;
use reth_db_common::init::init_genesis;
use reth_ethereum_primitives::Block;
use reth_evm::{ConfigureEvm, execute::Executor};
use reth_evm_ethereum::EthEvmConfig;
use reth_primitives_traits::RecoveredBlock;
use reth_provider::{
    HashedPostStateProvider, ProviderFactory, SaveBlocksInput, StateProviderFactory,
    StateRootProvider,
    providers::BlockchainProvider,
    test_utils::{MockNodeTypesWithDB, create_test_provider_factory_with_chain_spec},
};
use reth_revm::database::StateProviderDatabase;
use reth_trie_common::ComputedTrieData;
use secp256k1::{Keypair, Secp256k1};
use std::sync::Arc;

/// reth's default `--engine.memory-block-buffer-target`.
pub(super) const MEMORY_BLOCK_BUFFER_TARGET: u64 = 5;

/// A fixed signer, so a failing case reproduces with the same addresses and trie shape.
pub(super) fn fixed_signer() -> Keypair {
    Keypair::from_seckey_byte_array(&Secp256k1::new(), [0x11; 32]).expect("valid secret key")
}

/// Where reth holds a block.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) struct Location {
    pub(super) canonical: bool,
    pub(super) in_memory: bool,
    pub(super) persisted: bool,
}

impl Location {
    pub(super) const ABANDONED: Self =
        Self { canonical: false, in_memory: false, persisted: false };
    pub(super) const IN_MEMORY: Self = Self { canonical: true, in_memory: true, persisted: false };
    pub(super) const PERSISTED_AND_EVICTED: Self =
        Self { canonical: true, in_memory: false, persisted: true };
    pub(super) const PERSISTED_AND_MASKED: Self =
        Self { canonical: true, in_memory: true, persisted: true };
}

/// reth's persistence settings. When the canonical head runs more than `persistence_threshold`
/// blocks past the database tip, reth persists every block but the newest
/// `memory_block_buffer_target` and evicts the persisted blocks from memory, except the newest
/// `num_state_masking_blocks` of them, whose hashed state and trie it has not written yet.
#[derive(Clone, Copy, Debug)]
pub(super) struct RethPersistence {
    pub(super) memory_block_buffer_target: u64,
    pub(super) num_state_masking_blocks: u64,
    pub(super) persistence_threshold: u64,
}

impl RethPersistence {
    /// Settings with reth's default buffer target and no masking.
    pub(super) fn threshold(persistence_threshold: u64) -> Self {
        assert!(
            MEMORY_BLOCK_BUFFER_TARGET <= persistence_threshold,
            "reth keeps at most persistence_threshold blocks in memory"
        );
        Self {
            memory_block_buffer_target: MEMORY_BLOCK_BUFFER_TARGET,
            num_state_masking_blocks: 0,
            persistence_threshold,
        }
    }

    /// reth rejects masking that leaves no room between the buffer target and the threshold.
    pub(super) fn masking(self, num_state_masking_blocks: u64) -> Self {
        assert!(
            num_state_masking_blocks + self.memory_block_buffer_target < self.persistence_threshold,
            "reth rejects num_state_masking_blocks + memory_block_buffer_target >= persistence_threshold"
        );
        Self { num_state_masking_blocks, ..self }
    }

    /// The block the first persistence cycle evicts through: the cycle fires at head
    /// `persistence_threshold + 1`, persists through `memory_block_buffer_target` below the head
    /// and evicts through `num_state_masking_blocks` below that.
    pub(super) const fn first_eviction_boundary(self) -> u64 {
        self.persistence_threshold + 1 -
            self.memory_block_buffer_target -
            self.num_state_masking_blocks
    }
}

/// reth's database and canonical in-memory state, driven the way reth's engine tree and
/// persistence task drive them: a block is canonical in memory first and reaches the database
/// when the persistence rule fires.
pub(super) struct SimulatedReth {
    /// The canonical chain above genesis, indexed by block number minus one.
    canonical: Vec<ExecutedBlock>,
    chain_spec: Arc<ChainSpec>,
    evm_config: EthEvmConfig,
    pub(super) factory: ProviderFactory<MockNodeTypesWithDB>,
    in_memory: CanonicalInMemoryState,
    key_pair: Keypair,
    /// Newest block evicted from memory; the persisted blocks above it are the masking window.
    last_evicted: BlockNumHash,
    /// Newest block in the database.
    last_persisted: BlockNumHash,
    nonce: u64,
    pub(super) persistence: RethPersistence,
    pub(super) provider: BlockchainProvider<MockNodeTypesWithDB>,
}

impl SimulatedReth {
    pub(super) fn new(
        chain_spec: Arc<ChainSpec>,
        key_pair: Keypair,
        persistence: RethPersistence,
    ) -> eyre::Result<Self> {
        let factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
        init_genesis(&factory)?;
        let provider = BlockchainProvider::new(factory.clone())?;
        let genesis = BlockNumHash::new(0, chain_spec.genesis_hash());
        Ok(Self {
            canonical: Vec::new(),
            evm_config: EthEvmConfig::ethereum(chain_spec.clone()),
            chain_spec,
            in_memory: provider.canonical_in_memory_state(),
            factory,
            key_pair,
            last_evicted: genesis,
            last_persisted: genesis,
            nonce: 0,
            persistence,
            provider,
        })
    }

    pub(super) fn num_hash(&self, number: u64) -> BlockNumHash {
        match number {
            0 => BlockNumHash::new(0, self.chain_spec.genesis_hash()),
            _ => self.block(number).num_hash(),
        }
    }

    pub(super) fn head(&self) -> BlockNumHash {
        self.num_hash(self.canonical.len() as u64)
    }

    pub(super) fn block(&self, number: u64) -> &RecoveredBlock<Block> {
        self.canonical[number as usize - 1].recovered_block()
    }

    pub(super) fn locate(&self, block: BlockNumHash) -> Location {
        let canonical = block.number <= self.head().number && self.num_hash(block.number) == block;
        Location {
            canonical,
            in_memory: self.in_memory.state_by_hash(block.hash).is_some(),
            persisted: canonical && self.last_persisted.number >= block.number,
        }
    }

    /// Appends a block to the canonical chain and runs the persistence rule.
    pub(super) fn commit(&mut self, spec: &BlockSpec) -> eyre::Result<ExecutedBlock> {
        let block = self.build_block(spec)?;
        self.make_canonical(NewCanonicalChain::Commit { new: vec![block.clone()] });
        self.advance_persistence()?;
        Ok(block)
    }

    /// Appends empty blocks until `reached` holds, which the persistence rule must bring about
    /// within one cycle.
    pub(super) fn commit_fillers_until(
        &mut self,
        reached: impl Fn(&Self) -> bool,
    ) -> eyre::Result<()> {
        let limit = self.head().number + self.persistence.persistence_threshold + 1;
        while !reached(self) {
            assert!(
                self.head().number < limit,
                "the persistence rule did not reach the requested parent state within one cycle"
            );
            self.commit(&BlockSpec::new(vec![]))?;
        }
        Ok(())
    }

    /// Replaces the canonical blocks from `number` up with empty blocks on the same ancestor, the
    /// way a reorg to a sibling fork does.
    pub(super) fn reorg_from(&mut self, number: u64) -> eyre::Result<()> {
        assert!(number > self.last_persisted.number, "only in-memory blocks can be reorged");
        let old = self.canonical.split_off(number as usize - 1);
        let fork = self.build_block(&BlockSpec::new(vec![]))?;
        assert_ne!(
            fork.recovered_block().hash(),
            old[0].recovered_block().hash(),
            "the fork must differ from the chain it replaces"
        );
        let depth = old.len();
        self.make_canonical(NewCanonicalChain::Reorg { new: vec![fork], old });
        for _ in 1..depth {
            self.commit(&BlockSpec::new(vec![]))?;
        }
        Ok(())
    }

    /// Writes every canonical block to the database without evicting any from memory, the state
    /// a `proofs init` run finds on a node that has caught up.
    pub(super) fn persist_all(&mut self) -> eyre::Result<()> {
        let head = self.head().number;
        if head > self.last_persisted.number {
            self.persist(head)?;
        }
        Ok(())
    }

    fn make_canonical(&mut self, update: NewCanonicalChain) {
        let tip = update.tip().clone_sealed_header();
        match &update {
            NewCanonicalChain::Commit { new } | NewCanonicalChain::Reorg { new, .. } => {
                self.canonical.extend(new.iter().cloned());
            }
        }
        self.in_memory.update_chain(update);
        self.in_memory.set_canonical_head(tip);
    }

    /// Builds `spec` on the head and executes it, giving it the state root and trie data reth
    /// holds for a block it validated.
    fn build_block(&mut self, spec: &BlockSpec) -> eyre::Result<ExecutedBlock> {
        let parent = self.head();
        let mut block = create_block_from_spec(
            spec,
            parent.number + 1,
            parent.hash,
            &self.chain_spec,
            self.key_pair,
            &mut self.nonce,
        );
        let parent_state = self.provider.state_by_block_hash(parent.hash)?;
        let output = self
            .evm_config
            .batch_executor(StateProviderDatabase::new(&parent_state))
            .execute(&block)?;
        let hashed_state = parent_state.hashed_post_state(&output.state)?;
        let (state_root, trie_updates) =
            parent_state.state_root_with_updates(hashed_state.clone())?;
        block.set_state_root(state_root);
        let trie_data = ComputedTrieData::new(
            Arc::new(hashed_state.into_sorted()),
            Arc::new(trie_updates.into_sorted()),
        );
        Ok(ExecutedBlock::new(Arc::new(block), Arc::new(output), trie_data))
    }

    /// reth's persistence rule, run after every new canonical block.
    fn advance_persistence(&mut self) -> eyre::Result<()> {
        let head = self.head().number;
        if head - self.last_persisted.number <= self.persistence.persistence_threshold {
            return Ok(());
        }
        let new_db_tip = head.saturating_sub(self.persistence.memory_block_buffer_target);
        if new_db_tip <= self.last_persisted.number {
            return Ok(());
        }
        let new_evicted = new_db_tip
            .saturating_sub(self.persistence.num_state_masking_blocks)
            .max(self.last_evicted.number);
        self.persist(new_db_tip)?;
        self.evict_through(new_evicted);
        Ok(())
    }

    /// Writes the canonical blocks above the database tip through `new_db_tip` with reth's own
    /// writer. reth's `partial-persistence` feature is off on develop, so the writer cannot hold
    /// back the hashed state and trie of masked blocks; it gets both frontiers at the database
    /// tip, and masking only decides which persisted blocks stay in memory.
    fn persist(&mut self, new_db_tip: u64) -> eyre::Result<()> {
        let prev_db_tip = self.last_persisted.number;
        let input = SaveBlocksInput::new(
            self.canonical[prev_db_tip as usize..new_db_tip as usize].to_vec(),
            prev_db_tip,
            prev_db_tip,
            new_db_tip,
            new_db_tip,
        );
        let provider_rw = self.factory.provider_rw()?;
        provider_rw.save_blocks(&input)?;
        provider_rw.commit()?;
        self.last_persisted = self.num_hash(new_db_tip);
        Ok(())
    }

    /// Drops the persisted blocks through `number` from memory.
    fn evict_through(&mut self, number: u64) {
        assert!(number <= self.last_persisted.number, "only persisted blocks leave memory");
        self.in_memory.remove_persisted_blocks_until(self.last_persisted, number);
        self.last_evicted = self.num_hash(number);
    }
}
