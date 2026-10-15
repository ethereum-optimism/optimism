//! Executes proofs blocks through the engine while reth's persistence moves their parent between
//! memory, disk and an abandoned fork.
//!
//! reth makes a block canonical in memory first and persists it later, so by the time the engine
//! executes a block its parent can be in memory, on disk and evicted from memory, or no longer
//! canonical at all. The cases here drive reth's canonical in-memory state and database with
//! reth's own persistence rule and execute the call and `BLOCKHASH` blocks of a short chain in
//! each of those parent states.

use super::{
    BlockSpec, TxSpec, chain_spec_with_address, commit_blocks_to_database, create_block_from_spec,
    create_mdbx_proofs_storage, public_key_to_address, reth_trie_layout,
};
use alloy_consensus::BlockHeader;
use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, B256, Bytes, U256};
use parking_lot::Mutex;
use reth_chain_state::{CanonicalInMemoryState, ExecutedBlock, NewCanonicalChain};
use reth_chainspec::ChainSpec;
use reth_db::Database;
use reth_db_common::init::init_genesis;
use reth_ethereum_primitives::Block;
use reth_evm::{ConfigureEvm, execute::Executor};
use reth_evm_ethereum::EthEvmConfig;
use reth_optimism_trie::{
    MdbxProofsStorage, OpProofStoragePruner, OpProofsProviderRO, OpProofsStorageResult,
    OpProofsStore,
    engine::{EngineError, EngineHandle},
    initialize::InitializationJob,
    provider::OpProofsStateProviderRef,
};
use reth_primitives_traits::RecoveredBlock;
use reth_provider::{
    HashedPostStateProvider, ProviderFactory, StateProvider, StateProviderFactory,
    StateRootProvider,
    noop::NoopProvider,
    providers::BlockchainProvider,
    test_utils::{MockNodeTypesWithDB, create_test_provider_factory_with_chain_spec},
};
use reth_revm::database::StateProviderDatabase;
use reth_trie_common::{ComputedTrieData, HashedPostState};
use rstest::rstest;
use secp256k1::{Keypair, Secp256k1, rand::rng};
use serial_test::serial;
use std::{fmt, sync::Arc, thread};

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

/// Where reth holds the parent of the block under test when the engine executes the block.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum ParentState {
    /// Canonical and in memory, as every recent block is.
    InMemory,
    /// Persisted and evicted from memory because reth ran ahead of the engine.
    PersistedAndEvicted,
    /// Persisted but kept in memory by state masking.
    PersistedAndMasked,
    /// In memory when execution starts, persisted and evicted before execution reads the parent.
    EvictedWhileExecuting,
    /// Reorged out of memory before execution, never persisted.
    AbandonedFork,
    /// In memory when execution starts, reorged out with its ancestry before execution reads a
    /// block hash.
    ReorgedWhileExecuting,
}

/// Which block of the deploy, call, `BLOCKHASH` chain the engine executes in the parent state.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum BlockContent {
    /// The block calling the contract its parent deployed.
    DeployAndCall,
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
    fn new(
        persistence_threshold: u64,
        num_state_masking_blocks: u64,
        memory_block_buffer_target: u64,
    ) -> Self {
        assert!(
            memory_block_buffer_target <= persistence_threshold,
            "reth keeps at most persistence_threshold blocks in memory"
        );
        assert!(
            num_state_masking_blocks == 0 ||
                num_state_masking_blocks + memory_block_buffer_target < persistence_threshold,
            "reth rejects num_state_masking_blocks + memory_block_buffer_target >= persistence_threshold"
        );
        Self { memory_block_buffer_target, num_state_masking_blocks, persistence_threshold }
    }
}

fn reth(persistence_threshold: u64, num_state_masking_blocks: u64) -> RethPersistence {
    RethPersistence::new(
        persistence_threshold,
        num_state_masking_blocks,
        MEMORY_BLOCK_BUFFER_TARGET,
    )
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
    /// Newest block in the database.
    last_persisted: BlockNumHash,
    /// Newest block whose hashed state and trie are in the database; persisted blocks above it
    /// stay in memory.
    last_state_trie_persisted: BlockNumHash,
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
            last_persisted: genesis,
            last_state_trie_persisted: genesis,
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

    const fn is_persisted(&self, block: BlockNumHash) -> bool {
        self.last_persisted.number >= block.number
    }

    fn is_in_memory(&self, block: BlockNumHash) -> bool {
        self.in_memory.state_by_hash(block.hash).is_some()
    }

    /// Appends a block to the canonical chain and runs the persistence rule.
    fn commit(&mut self, spec: &BlockSpec) -> eyre::Result<RecoveredBlock<Block>> {
        let block = self.build_block(spec, self.head())?;
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
        let fork = self.build_block(&BlockSpec::new(vec![]), self.head())?;
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

    /// Builds `spec` on `parent` and executes it, giving it the state root and trie data reth
    /// holds for a block it validated.
    fn build_block(
        &mut self,
        spec: &BlockSpec,
        parent: BlockNumHash,
    ) -> eyre::Result<ExecutedBlock> {
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
        let new_state_trie_tip = new_db_tip
            .saturating_sub(self.persistence.num_state_masking_blocks)
            .max(self.last_state_trie_persisted.number);
        self.persist_through(new_db_tip)?;
        self.evict_through(new_state_trie_tip);
        Ok(())
    }

    /// Writes the canonical blocks above the database tip through `number` to the database.
    fn persist_through(&mut self, number: u64) -> eyre::Result<()> {
        let first = self.last_persisted.number as usize;
        let blocks = &self.canonical[first..number as usize];
        commit_blocks_to_database(
            blocks.iter().map(|block| (block.recovered_block(), block.execution_outcome())),
            &self.factory,
        )?;
        self.last_persisted = self.num_hash(number);
        Ok(())
    }

    /// Drops the persisted blocks through `number` from memory.
    fn evict_through(&mut self, number: u64) {
        assert!(number <= self.last_persisted.number, "only persisted blocks leave memory");
        self.in_memory.remove_persisted_blocks_until(self.last_persisted, number);
        self.last_state_trie_persisted = self.num_hash(number);
    }
}

type ReadHook = Box<dyn FnOnce() + Send>;

/// A proofs store that runs an action on reth right before the engine opens its next read view.
///
/// The engine's execute task snapshots reth's parent state before it opens the proofs read view
/// it executes against, so an action armed here lands after the snapshot and before execution.
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
            // reth's persistence runs on its own thread; so does the action, which keeps the
            // engine thread's open database transactions out of its way.
            thread::scope(|scope| scope.spawn(action).join()).expect("the read hook panicked");
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
/// and the `BLOCKHASH` read, with the index of the block under test.
struct ProofChain {
    blocks: Vec<RecoveredBlock<Block>>,
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
        let under_test = match content {
            BlockContent::DeployAndCall => leading + 1,
            BlockContent::BlockHash => leading + 2,
        } as usize;
        Ok(Self { blocks, under_test })
    }

    fn deploy(&self) -> &RecoveredBlock<Block> {
        &self.blocks[self.blocks.len() - 3]
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

/// How many empty blocks precede the deployment so that the parent ends up in `parent_state`.
///
/// The first persistence cycle fires at head `persistence_threshold + 1` and evicts through
/// `persistence_threshold + 1 - memory_block_buffer_target - num_state_masking_blocks`; a
/// deployment right above that boundary is persisted but masked, so it stays in memory.
const fn leading_blocks(persistence: RethPersistence, parent_state: ParentState) -> u64 {
    match parent_state {
        ParentState::PersistedAndMasked => {
            persistence.persistence_threshold + 1 -
                persistence.memory_block_buffer_target -
                persistence.num_state_masking_blocks
        }
        _ => 0,
    }
}

fn parent_persisted_and_evicted(reth: &SimulatedReth, parent: BlockNumHash) -> bool {
    reth.is_persisted(parent) && !reth.is_in_memory(parent)
}

/// Puts the parent into the states reth reaches before the engine executes the block.
fn arrange_parent_state(
    reth: &mut SimulatedReth,
    parent_state: ParentState,
    chain: &ProofChain,
) -> eyre::Result<()> {
    let parent = chain.parent();
    match parent_state {
        ParentState::PersistedAndEvicted => {
            reth.commit_fillers_until(|reth| parent_persisted_and_evicted(reth, parent))?;
        }
        ParentState::PersistedAndMasked => {
            reth.commit_fillers_until(|reth| reth.is_persisted(parent))?;
            assert!(reth.is_in_memory(parent), "masking must keep the persisted parent in memory");
        }
        ParentState::AbandonedFork => reth.reorg_from(parent.number)?,
        ParentState::InMemory |
        ParentState::EvictedWhileExecuting |
        ParentState::ReorgedWhileExecuting => {}
    }
    let expected_in_memory =
        !matches!(parent_state, ParentState::PersistedAndEvicted | ParentState::AbandonedFork);
    assert_eq!(reth.is_in_memory(parent), expected_in_memory, "parent state before execution");
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
                .commit_fillers_until(|reth| parent_persisted_and_evicted(reth, parent))
                .expect("reth persists and evicts the parent");
        }),
        ParentState::ReorgedWhileExecuting => store.on_next_read(move || {
            reth.lock().reorg_from(deploy_number).expect("reth reorgs below the block");
        }),
        _ => {}
    }
}

fn assert_parent_state_after_execution(
    reth: &SimulatedReth,
    parent_state: ParentState,
    chain: &ProofChain,
) {
    let parent = chain.parent();
    match parent_state {
        ParentState::InMemory => assert!(reth.is_in_memory(parent) && !reth.is_persisted(parent)),
        ParentState::PersistedAndMasked => {
            assert!(reth.is_in_memory(parent) && reth.is_persisted(parent));
        }
        ParentState::PersistedAndEvicted | ParentState::EvictedWhileExecuting => {
            assert!(parent_persisted_and_evicted(reth, parent));
        }
        ParentState::AbandonedFork | ParentState::ReorgedWhileExecuting => {
            assert!(!reth.is_in_memory(parent) && !reth.is_persisted(parent));
            assert_ne!(reth.block(parent.number).hash(), parent.hash, "a fork replaced the parent");
            if parent_state == ParentState::ReorgedWhileExecuting {
                let deploy = chain.deploy();
                assert_ne!(reth.block(deploy.number()).hash(), deploy.hash());
            }
        }
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
    let key_pair = Keypair::new(&Secp256k1::new(), &mut rng());
    let sender = public_key_to_address(key_pair.public_key());
    let contract = sender.create(0);
    let chain_spec = chain_spec_with_address(sender);
    let store = create_mdbx_proofs_storage();
    let mut reth = SimulatedReth::new(chain_spec.clone(), key_pair, persistence)?;
    {
        let tx = reth.factory.db_ref().tx()?;
        InitializationJob::new(store.clone(), tx, reth_trie_layout(&reth.factory))
            .run(0, chain_spec.genesis_hash())?;
    }
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
    // waits for that write, so the threshold is the number of blocks the case stores.
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
    engine.flush();
    assert!(
        !store_with_hook.is_armed(),
        "the engine must open its proofs read view after snapshotting reth's parent state"
    );
    assert_parent_state_after_execution(&reth.lock(), parent_state, &chain);

    match outcome {
        Outcome::Stored => {
            result?;
            assert_stored(&store, &chain, content, contract)
        }
        Outcome::StateRootMismatch => {
            let err = result.expect_err("an abandoned parent must fail the state-root check");
            assert!(
                matches!(
                    err,
                    EngineError::StateRootMismatch { block_number, .. }
                        if block_number == chain.under_test().number()
                ),
                "{err:?}"
            );
            assert_eq!(store.provider_ro()?.get_latest_block()?, chain.parent());
            Ok(())
        }
    }
}

/// The parent states reth reaches on its own, over reth's persistence settings and both blocks.
///
/// reth rejects `num_state_masking_blocks + memory_block_buffer_target >=
/// persistence_threshold`, so threshold 7 only runs unmasked.
#[rstest]
#[case::in_memory(ParentState::InMemory, Outcome::Stored)]
#[case::persisted_and_evicted(ParentState::PersistedAndEvicted, Outcome::Stored)]
#[case::evicted_while_executing(ParentState::EvictedWhileExecuting, Outcome::Stored)]
#[ignore = "passes on #23046; develop skips a parent reth no longer holds (StateForHashNotFound) and stores nothing"]
#[case::abandoned_fork(ParentState::AbandonedFork, Outcome::StateRootMismatch)]
#[serial]
fn execute_block_across_reth_parent_states(
    #[values(reth(7, 0), reth(50, 0), reth(50, 30))] persistence: RethPersistence,
    #[values(BlockContent::DeployAndCall, BlockContent::BlockHash)] content: BlockContent,
    #[case] parent_state: ParentState,
    #[case] outcome: Outcome,
) -> eyre::Result<()> {
    run_parent_state_case(persistence, content, parent_state, outcome)
}

/// State masking keeps persisted blocks in memory, so the parent is on disk and in memory at once.
#[rstest]
#[serial]
fn execute_block_with_persisted_and_masked_parent(
    #[values(BlockContent::DeployAndCall, BlockContent::BlockHash)] content: BlockContent,
) -> eyre::Result<()> {
    run_parent_state_case(reth(50, 30), content, ParentState::PersistedAndMasked, Outcome::Stored)
}

/// `BLOCKHASH` reads the parent's ancestry even after reth reorged it out mid-execution.
#[rstest]
#[serial]
fn execute_block_hash_read_while_reth_reorgs_below_it(
    #[values(reth(7, 0), reth(50, 0), reth(50, 30))] persistence: RethPersistence,
) -> eyre::Result<()> {
    run_parent_state_case(
        persistence,
        BlockContent::BlockHash,
        ParentState::ReorgedWhileExecuting,
        Outcome::Stored,
    )
}
