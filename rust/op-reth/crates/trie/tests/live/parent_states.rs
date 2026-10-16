//! Executes proofs blocks through the engine while reth's persistence moves their parent between
//! memory, disk and an abandoned fork.
//!
//! reth makes a block canonical in memory first and persists it later, so by the time the engine
//! executes a block its parent can be in memory, on disk and evicted from memory, or no longer
//! canonical at all. The cases here drive reth's canonical in-memory state and database with
//! reth's own persistence rule and execute the call and `BLOCKHASH` blocks of a short chain in
//! each of those parent states.

use super::{
    BlockSpec, TxSpec, chain_spec_with_address, create_block_from_spec, create_mdbx_proofs_storage,
    initialize_storage, public_key_to_address,
};
use alloy_consensus::BlockHeader;
use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, B256, Bytes, U256};
use parking_lot::Mutex;
use reth_chain_state::{CanonicalInMemoryState, ExecutedBlock, NewCanonicalChain};
use reth_chainspec::ChainSpec;
use reth_db_common::init::init_genesis;
use reth_ethereum_primitives::Block;
use reth_evm::{ConfigureEvm, execute::Executor};
use reth_evm_ethereum::EthEvmConfig;
use reth_optimism_trie::{
    MdbxProofsStorage, OpProofStoragePruner, OpProofsProviderRO, OpProofsStorageResult,
    OpProofsStore,
    engine::{EngineError, EngineHandle},
    provider::OpProofsStateProviderRef,
};
use reth_primitives_traits::RecoveredBlock;
use reth_provider::{
    HashedPostStateProvider, ProviderFactory, SaveBlocksInput, StateProvider, StateProviderFactory,
    StateRootProvider,
    noop::NoopProvider,
    providers::BlockchainProvider,
    test_utils::{MockNodeTypesWithDB, create_test_provider_factory_with_chain_spec},
};
use reth_revm::database::StateProviderDatabase;
use reth_trie_common::{ComputedTrieData, HashedPostState};
use rstest::rstest;
use secp256k1::{Keypair, Secp256k1};
use serial_test::serial;
use std::{fmt, sync::Arc};

/// reth's default `--engine.memory-block-buffer-target`.
const MEMORY_BLOCK_BUFFER_TARGET: u64 = 5;

/// Covers the contract's deployment and either of its calls.
const CONTRACT_GAS_LIMIT: u64 = 100_000;

/// Storage slot every call to the contract sets to one.
const CALLED_SLOT: B256 = B256::ZERO;

/// Storage slot a call with calldata sets to `BLOCKHASH(calldata)`.
const BLOCK_HASH_SLOT: B256 = B256::with_last_byte(1);

/// Runtime code of the contract: `slot 0 = 1`, then `slot 1 = BLOCKHASH(calldata[0..32])` unless
/// the call has no calldata.
///
/// ```text
/// PUSH1 1 PUSH1 0 SSTORE
/// CALLDATASIZE ISZERO PUSH1 0x11 JUMPI
/// PUSH1 0 CALLDATALOAD BLOCKHASH PUSH1 1 SSTORE
/// JUMPDEST STOP
/// ```
const CONTRACT_RUNTIME: [u8; 19] = [
    0x60, 0x01, 0x60, 0x00, 0x55, 0x36, 0x15, 0x60, 0x11, 0x57, 0x60, 0x00, 0x35, 0x40, 0x60, 0x01,
    0x55, 0x5b, 0x00,
];

/// Initcode returning [`CONTRACT_RUNTIME`], which follows the twelve initcode bytes:
/// `PUSH1 len PUSH1 12 PUSH1 0 CODECOPY PUSH1 len PUSH1 0 RETURN`.
fn contract_initcode() -> Bytes {
    let runtime_len = CONTRACT_RUNTIME.len() as u8;
    let mut initcode =
        vec![0x60, runtime_len, 0x60, 0x0c, 0x60, 0x00, 0x39, 0x60, runtime_len, 0x60, 0x00, 0xf3];
    initcode.extend_from_slice(&CONTRACT_RUNTIME);
    initcode.into()
}

fn block_number_calldata(number: u64) -> Bytes {
    Bytes::copy_from_slice(&U256::from(number).to_be_bytes::<32>())
}

/// A fixed signer, so a failing case reproduces with the same addresses and trie shape.
fn signer() -> Keypair {
    Keypair::from_seckey_byte_array(&Secp256k1::new(), [0x11; 32]).expect("valid secret key")
}

/// Where reth holds a block.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
struct Location {
    canonical: bool,
    in_memory: bool,
    persisted: bool,
}

impl Location {
    const ABANDONED: Self = Self { canonical: false, in_memory: false, persisted: false };
    const IN_MEMORY: Self = Self { canonical: true, in_memory: true, persisted: false };
    const PERSISTED_AND_EVICTED: Self = Self { canonical: true, in_memory: false, persisted: true };
    const PERSISTED_AND_MASKED: Self = Self { canonical: true, in_memory: true, persisted: true };
}

/// Where reth holds the parent of the block under test when the engine executes the block.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum ParentState {
    /// Canonical and in memory, as every recent block is.
    InMemory,
    /// In memory while the deployment whose hash the block reads is persisted and evicted, so the
    /// hash comes from below reth's in-memory chain.
    InMemoryDeployEvicted,
    /// Persisted and evicted from memory because reth ran ahead of the engine.
    PersistedAndEvicted,
    /// Persisted but kept in memory by state masking.
    PersistedAndMasked,
    /// In memory when execution starts, persisted and evicted before execution reads the parent.
    /// reth writes a block to disk before evicting it, so this checks the transition, not that a
    /// dropped snapshot would be noticed.
    EvictedWhileExecuting,
    /// Reorged out of memory before execution, never persisted.
    AbandonedFork,
    /// In memory when execution starts, reorged out with its ancestry before execution reads a
    /// block hash.
    ReorgedWhileExecuting,
}

impl ParentState {
    /// Where reth holds the parent when the engine starts executing the block under test.
    const fn parent_before_execution(self, persistence: RethPersistence) -> Location {
        match self {
            Self::InMemory | Self::EvictedWhileExecuting | Self::ReorgedWhileExecuting => {
                Location::IN_MEMORY
            }
            Self::InMemoryDeployEvicted => Location {
                persisted: persistence.num_state_masking_blocks > 0,
                ..Location::IN_MEMORY
            },
            Self::PersistedAndEvicted => Location::PERSISTED_AND_EVICTED,
            Self::PersistedAndMasked => Location::PERSISTED_AND_MASKED,
            Self::AbandonedFork => Location::ABANDONED,
        }
    }

    /// Where reth holds the parent once the engine has executed the block under test.
    const fn parent_after_execution(self, persistence: RethPersistence) -> Location {
        match self {
            Self::EvictedWhileExecuting => Location::PERSISTED_AND_EVICTED,
            Self::ReorgedWhileExecuting => Location::ABANDONED,
            _ => self.parent_before_execution(persistence),
        }
    }

    /// Where reth holds the deployment after execution, for the states that move it away from
    /// the parent.
    const fn deploy_after_execution(self) -> Option<Location> {
        match self {
            Self::InMemoryDeployEvicted => Some(Location::PERSISTED_AND_EVICTED),
            Self::ReorgedWhileExecuting => Some(Location::ABANDONED),
            _ => None,
        }
    }
}

/// Which block of the deploy, call, `BLOCKHASH` chain the engine executes in the parent state.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum BlockContent {
    /// The block calling the contract its parent deployed.
    Call,
    /// The block storing `BLOCKHASH` of the deployment two blocks below it.
    BlockHash,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Outcome {
    /// The engine stores the block and the stored state hashes to the block's state root.
    Stored,
    /// The engine fails the state-root check and stores nothing.
    StateRootMismatch,
}

/// reth's persistence settings. When the canonical head runs more than `persistence_threshold`
/// blocks past the database tip, reth persists every block but the newest
/// `memory_block_buffer_target` and evicts the persisted blocks from memory, except the newest
/// `num_state_masking_blocks` of them, whose hashed state and trie it has not written yet.
#[derive(Clone, Copy, Debug)]
struct RethPersistence {
    memory_block_buffer_target: u64,
    num_state_masking_blocks: u64,
    persistence_threshold: u64,
}

impl RethPersistence {
    /// Settings with reth's default buffer target and no masking.
    fn threshold(persistence_threshold: u64) -> Self {
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
    fn masking(self, num_state_masking_blocks: u64) -> Self {
        assert!(
            num_state_masking_blocks + self.memory_block_buffer_target < self.persistence_threshold,
            "reth rejects num_state_masking_blocks + memory_block_buffer_target >= persistence_threshold"
        );
        Self { num_state_masking_blocks, ..self }
    }

    /// The block the first persistence cycle evicts through: the cycle fires at head
    /// `persistence_threshold + 1`, persists through `memory_block_buffer_target` below the head
    /// and evicts through `num_state_masking_blocks` below that.
    const fn first_eviction_boundary(self) -> u64 {
        self.persistence_threshold + 1 -
            self.memory_block_buffer_target -
            self.num_state_masking_blocks
    }
}

/// reth's database and canonical in-memory state, driven the way reth's engine tree and
/// persistence task drive them: a block is canonical in memory first and reaches the database
/// when the persistence rule fires.
struct SimulatedReth {
    /// The canonical chain above genesis, indexed by block number minus one.
    canonical: Vec<ExecutedBlock>,
    chain_spec: Arc<ChainSpec>,
    evm_config: EthEvmConfig,
    factory: ProviderFactory<MockNodeTypesWithDB>,
    in_memory: CanonicalInMemoryState,
    key_pair: Keypair,
    /// Newest block evicted from memory; the persisted blocks above it are the masking window.
    last_evicted: BlockNumHash,
    /// Newest block in the database.
    last_persisted: BlockNumHash,
    nonce: u64,
    persistence: RethPersistence,
    provider: BlockchainProvider<MockNodeTypesWithDB>,
}

impl SimulatedReth {
    fn new(
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

    fn num_hash(&self, number: u64) -> BlockNumHash {
        match number {
            0 => BlockNumHash::new(0, self.chain_spec.genesis_hash()),
            _ => self.block(number).num_hash(),
        }
    }

    fn head(&self) -> BlockNumHash {
        self.num_hash(self.canonical.len() as u64)
    }

    fn block(&self, number: u64) -> &RecoveredBlock<Block> {
        self.canonical[number as usize - 1].recovered_block()
    }

    fn locate(&self, block: BlockNumHash) -> Location {
        let canonical = block.number <= self.head().number && self.num_hash(block.number) == block;
        Location {
            canonical,
            in_memory: self.in_memory.state_by_hash(block.hash).is_some(),
            persisted: canonical && self.last_persisted.number >= block.number,
        }
    }

    /// Appends a block to the canonical chain and runs the persistence rule.
    fn commit(&mut self, spec: &BlockSpec) -> eyre::Result<RecoveredBlock<Block>> {
        let block = self.build_block(spec)?;
        let recovered = block.recovered_block().clone();
        self.make_canonical(NewCanonicalChain::Commit { new: vec![block] });
        self.advance_persistence()?;
        Ok(recovered)
    }

    /// Appends empty blocks until `reached` holds, which the persistence rule must bring about
    /// within one cycle.
    fn commit_fillers_until(&mut self, reached: impl Fn(&Self) -> bool) -> eyre::Result<()> {
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
    fn reorg_from(&mut self, number: u64) -> eyre::Result<()> {
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

type ReadHook = Box<dyn FnOnce() + Send>;

/// A proofs store that runs an action on reth right before the engine opens its next read view.
///
/// The engine's execute task snapshots reth's parent state and then opens the proofs read view it
/// executes against, so an action armed here lands between the two. The engine also opens a read
/// view to find its tip whenever its block buffer is empty; the cases keep the preceding blocks
/// buffered so that read never happens while an action is armed.
#[derive(Clone)]
struct StoreWithReadHook<S> {
    inner: S,
    on_next_read: Arc<Mutex<Option<ReadHook>>>,
}

impl<S> StoreWithReadHook<S> {
    fn new(inner: S) -> Self {
        Self { inner, on_next_read: Arc::default() }
    }

    fn on_next_read(&self, action: impl FnOnce() + Send + 'static) {
        *self.on_next_read.lock() = Some(Box::new(action));
    }

    fn is_armed(&self) -> bool {
        self.on_next_read.lock().is_some()
    }
}

impl<S: fmt::Debug> fmt::Debug for StoreWithReadHook<S> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("StoreWithReadHook").field("inner", &self.inner).finish_non_exhaustive()
    }
}

impl<S: OpProofsStore> OpProofsStore for StoreWithReadHook<S> {
    type ProviderRO<'a>
        = S::ProviderRO<'a>
    where
        Self: 'a;

    type ProviderRw<'a>
        = S::ProviderRw<'a>
    where
        Self: 'a;

    type Initializer<'a>
        = S::Initializer<'a>
    where
        Self: 'a;

    fn provider_ro<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRO<'a>> {
        let action = self.on_next_read.lock().take();
        if let Some(action) = action {
            action();
        }
        self.inner.provider_ro()
    }

    fn provider_rw<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRw<'a>> {
        self.inner.provider_rw()
    }

    fn initialization_provider<'a>(&'a self) -> OpProofsStorageResult<Self::Initializer<'a>> {
        self.inner.initialization_provider()
    }
}

/// The chain reth holds before the engine runs: `leading` empty blocks, the deployment, the call
/// and the `BLOCKHASH` read, with the indices of the deployment and of the block under test.
struct ProofChain {
    blocks: Vec<RecoveredBlock<Block>>,
    deploy: usize,
    under_test: usize,
}

impl ProofChain {
    fn commit(
        reth: &mut SimulatedReth,
        leading: u64,
        contract: Address,
        content: BlockContent,
    ) -> eyre::Result<Self> {
        let mut blocks = Vec::new();
        for _ in 0..leading {
            blocks.push(reth.commit(&BlockSpec::new(vec![]))?);
        }
        let deploy_number = leading + 1;
        let specs = [
            BlockSpec::new(vec![TxSpec::create(contract_initcode(), CONTRACT_GAS_LIMIT)]),
            BlockSpec::new(vec![TxSpec::call(contract, Bytes::new(), CONTRACT_GAS_LIMIT)]),
            BlockSpec::new(vec![TxSpec::call(
                contract,
                block_number_calldata(deploy_number),
                CONTRACT_GAS_LIMIT,
            )]),
        ];
        for spec in &specs {
            blocks.push(reth.commit(spec)?);
        }
        let deploy = leading as usize;
        let under_test = match content {
            BlockContent::Call => deploy + 1,
            BlockContent::BlockHash => deploy + 2,
        };
        Ok(Self { blocks, deploy, under_test })
    }

    fn deploy(&self) -> &RecoveredBlock<Block> {
        &self.blocks[self.deploy]
    }

    fn under_test(&self) -> &RecoveredBlock<Block> {
        &self.blocks[self.under_test]
    }

    fn parent(&self) -> BlockNumHash {
        self.blocks[self.under_test - 1].num_hash()
    }

    /// The blocks the engine executes before the block under test.
    fn preceding(&self) -> &[RecoveredBlock<Block>] {
        &self.blocks[..self.under_test]
    }
}

/// How many empty blocks precede the deployment so that the chain straddles the first eviction
/// boundary where `parent_state` needs it to: a masked parent is the first block above the
/// boundary, an evicted deployment is the boundary itself.
const fn leading_blocks(persistence: RethPersistence, parent_state: ParentState) -> u64 {
    match parent_state {
        ParentState::PersistedAndMasked => persistence.first_eviction_boundary(),
        ParentState::InMemoryDeployEvicted => persistence.first_eviction_boundary() - 1,
        _ => 0,
    }
}

/// Puts the parent into the states reth reaches before the engine executes the block.
fn arrange_parent_state(
    reth: &mut SimulatedReth,
    parent_state: ParentState,
    chain: &ProofChain,
) -> eyre::Result<()> {
    let parent = chain.parent();
    let deploy = chain.deploy().num_hash();
    match parent_state {
        ParentState::PersistedAndEvicted => reth
            .commit_fillers_until(|reth| reth.locate(parent) == Location::PERSISTED_AND_EVICTED)?,
        ParentState::PersistedAndMasked => {
            reth.commit_fillers_until(|reth| reth.locate(parent).persisted)?;
        }
        ParentState::InMemoryDeployEvicted => reth
            .commit_fillers_until(|reth| reth.locate(deploy) == Location::PERSISTED_AND_EVICTED)?,
        ParentState::AbandonedFork => reth.reorg_from(parent.number)?,
        ParentState::InMemory |
        ParentState::EvictedWhileExecuting |
        ParentState::ReorgedWhileExecuting => {}
    }
    assert_eq!(
        reth.locate(parent),
        parent_state.parent_before_execution(reth.persistence),
        "parent before execution"
    );
    Ok(())
}

/// Arms the reth transitions that happen while the engine executes the block.
fn arm_in_flight_transition(
    store: &StoreWithReadHook<Arc<MdbxProofsStorage>>,
    reth: &Arc<Mutex<SimulatedReth>>,
    parent_state: ParentState,
    chain: &ProofChain,
) {
    let parent = chain.parent();
    let deploy_number = chain.deploy().number();
    let reth = Arc::clone(reth);
    match parent_state {
        ParentState::EvictedWhileExecuting => store.on_next_read(move || {
            reth.lock()
                .commit_fillers_until(|reth| reth.locate(parent) == Location::PERSISTED_AND_EVICTED)
                .expect("reth persists and evicts the parent");
        }),
        ParentState::ReorgedWhileExecuting => store.on_next_read(move || {
            reth.lock().reorg_from(deploy_number).expect("reth reorgs below the block");
        }),
        _ => {}
    }
}

fn assert_locations_after_execution(
    reth: &SimulatedReth,
    parent_state: ParentState,
    chain: &ProofChain,
) {
    assert_eq!(
        reth.locate(chain.parent()),
        parent_state.parent_after_execution(reth.persistence),
        "parent after execution"
    );
    if let Some(expected) = parent_state.deploy_after_execution() {
        assert_eq!(reth.locate(chain.deploy().num_hash()), expected, "deployment after execution");
    }
}

/// Checks the stored block against reth's header: the proofs state hashes to the header's state
/// root, the call wrote its slot and a `BLOCKHASH` read saw the deployment on the parent's
/// ancestry.
fn assert_stored(
    store: &Arc<MdbxProofsStorage>,
    chain: &ProofChain,
    content: BlockContent,
    contract: Address,
) -> eyre::Result<()> {
    let under_test = chain.under_test();
    let proofs = store.provider_ro()?;
    assert_eq!(proofs.get_latest_block()?, under_test.num_hash());
    let state = OpProofsStateProviderRef::new(
        Box::new(NoopProvider::default()),
        proofs,
        under_test.number(),
    );
    assert_eq!(
        state.state_root(HashedPostState::default())?,
        under_test.state_root(),
        "the stored state must hash to the header's state root"
    );
    assert_eq!(state.storage(contract, CALLED_SLOT)?, Some(U256::from(1)));
    if content == BlockContent::BlockHash {
        assert_eq!(
            state.storage(contract, BLOCK_HASH_SLOT)?,
            Some(U256::from_be_bytes(chain.deploy().hash().0)),
            "BLOCKHASH must come from the parent's ancestry"
        );
    }
    Ok(())
}

fn run_parent_state_case(
    persistence: RethPersistence,
    content: BlockContent,
    parent_state: ParentState,
    outcome: Outcome,
) -> eyre::Result<()> {
    let key_pair = signer();
    let sender = public_key_to_address(key_pair.public_key());
    let contract = sender.create(0);
    let chain_spec = chain_spec_with_address(sender);
    let store = create_mdbx_proofs_storage();
    let mut reth = SimulatedReth::new(chain_spec.clone(), key_pair, persistence)?;
    initialize_storage(store.clone(), &reth.factory, reth.head())?;
    let chain = ProofChain::commit(
        &mut reth,
        leading_blocks(persistence, parent_state),
        contract,
        content,
    )?;
    arrange_parent_state(&mut reth, parent_state, &chain)?;

    let provider = reth.provider.clone();
    let reth = Arc::new(Mutex::new(reth));
    let store_with_hook = StoreWithReadHook::new(store.clone());
    let pruner = OpProofStoragePruner::new(store_with_hook.clone(), provider.clone(), 1000);
    // The engine writes its buffer to the store once it holds this many blocks, and `flush` only
    // waits for that write, so the threshold is the number of blocks the case stores. It also
    // keeps the preceding blocks buffered while the block under test executes, which the read
    // hook relies on.
    let stored_blocks = match outcome {
        Outcome::Stored => chain.preceding().len() + 1,
        Outcome::StateRootMismatch => chain.preceding().len(),
    } as u64;
    let engine = EngineHandle::spawn_with_thresholds(
        EthEvmConfig::ethereum(chain_spec),
        provider,
        store_with_hook.clone(),
        pruner,
        stored_blocks,
        stored_blocks + 1,
    );
    for block in chain.preceding() {
        engine.execute_block(block)?;
    }

    arm_in_flight_transition(&store_with_hook, &reth, parent_state, &chain);
    let result = engine.execute_block(chain.under_test());
    assert!(
        !store_with_hook.is_armed(),
        "the engine never opened a proofs read view, so the in-flight transition never ran"
    );
    engine.flush();
    assert_locations_after_execution(&reth.lock(), parent_state, &chain);

    match outcome {
        Outcome::Stored => {
            result?;
            assert_stored(&store, &chain, content, contract)
        }
        Outcome::StateRootMismatch => {
            let err = result.expect_err("an abandoned parent must fail the state-root check");
            let EngineError::StateRootMismatch { block_number, expected_state_hash, .. } = err
            else {
                panic!("expected a state-root mismatch, got {err:?}");
            };
            assert_eq!(block_number, chain.under_test().number());
            assert_eq!(expected_state_hash, chain.under_test().state_root());
            assert_eq!(store.provider_ro()?.get_latest_block()?, chain.parent());
            Ok(())
        }
    }
}

/// The parent states reth reaches on its own, over reth's persistence settings and both blocks.
///
/// reth rejects `num_state_masking_blocks + memory_block_buffer_target >=
/// persistence_threshold`, so threshold 7 only runs unmasked. The in-memory and the two fork
/// rows never reach a persistence cycle, so the settings only exercise the simulation there.
#[rstest]
#[case::in_memory(ParentState::InMemory, Outcome::Stored)]
#[case::persisted_and_evicted(ParentState::PersistedAndEvicted, Outcome::Stored)]
#[case::evicted_while_executing(ParentState::EvictedWhileExecuting, Outcome::Stored)]
#[ignore = "passes on #23046; develop skips a parent reth no longer holds (StateForHashNotFound) and stores nothing"]
#[case::abandoned_fork(ParentState::AbandonedFork, Outcome::StateRootMismatch)]
#[serial]
fn execute_block_across_reth_parent_states(
    #[values(
        RethPersistence::threshold(7),
        RethPersistence::threshold(50),
        RethPersistence::threshold(50).masking(30)
    )]
    persistence: RethPersistence,
    #[values(BlockContent::Call, BlockContent::BlockHash)] content: BlockContent,
    #[case] parent_state: ParentState,
    #[case] outcome: Outcome,
) -> eyre::Result<()> {
    run_parent_state_case(persistence, content, parent_state, outcome)
}

/// State masking keeps persisted blocks in memory, so the parent is on disk and in memory at once.
#[rstest]
#[serial]
fn execute_block_with_persisted_and_masked_parent(
    #[values(BlockContent::Call, BlockContent::BlockHash)] content: BlockContent,
) -> eyre::Result<()> {
    run_parent_state_case(
        RethPersistence::threshold(50).masking(30),
        content,
        ParentState::PersistedAndMasked,
        Outcome::Stored,
    )
}

/// `BLOCKHASH` of a deployment reth has evicted comes from disk while the parent is in memory.
#[rstest]
#[serial]
fn execute_block_hash_read_below_reth_memory(
    #[values(
        RethPersistence::threshold(7),
        RethPersistence::threshold(50),
        RethPersistence::threshold(50).masking(30)
    )]
    persistence: RethPersistence,
) -> eyre::Result<()> {
    run_parent_state_case(
        persistence,
        BlockContent::BlockHash,
        ParentState::InMemoryDeployEvicted,
        Outcome::Stored,
    )
}

/// `BLOCKHASH` reads the parent's ancestry even after reth reorged it out mid-execution.
#[rstest]
#[serial]
fn execute_block_hash_read_while_reth_reorgs_below_it(
    #[values(
        RethPersistence::threshold(7),
        RethPersistence::threshold(50),
        RethPersistence::threshold(50).masking(30)
    )]
    persistence: RethPersistence,
) -> eyre::Result<()> {
    run_parent_state_case(
        persistence,
        BlockContent::BlockHash,
        ParentState::ReorgedWhileExecuting,
        Outcome::Stored,
    )
}
