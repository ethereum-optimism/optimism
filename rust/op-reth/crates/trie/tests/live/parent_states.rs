//! Executes proofs blocks through the engine while reth's persistence moves their parent between
//! memory, disk and an abandoned fork.
//!
//! reth makes a block canonical in memory first and persists it later, so by the time the engine
//! executes a block its parent can be in memory, on disk and evicted from memory, or no longer
//! canonical at all. The cases here drive reth's canonical in-memory state and database with
//! reth's own persistence rule and execute the call and `BLOCKHASH` blocks of a short chain in
//! each of those parent states.

use super::{
    BlockSpec, TxSpec, chain_spec_with_address,
    contracts::{
        BLOCK_HASH_RECORDER_RUNTIME, BLOCK_HASH_SLOT, CALLED_SLOT, block_number_calldata,
        initcode_returning,
    },
    create_mdbx_proofs_storage, initialize_storage, public_key_to_address,
    simulated_reth::{Location, RethPersistence, SimulatedReth, fixed_signer},
};
use alloy_consensus::BlockHeader;
use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, Bytes, U256};
use parking_lot::Mutex;
use reth_ethereum_primitives::Block;
use reth_optimism_trie::{
    MdbxProofsStorage, OpProofsProviderRO, OpProofsStorageResult, OpProofsStore,
    engine::EngineError, provider::OpProofsStateProviderRef,
};
use reth_primitives_traits::RecoveredBlock;
use reth_provider::{StateProvider, StateRootProvider, noop::NoopProvider};
use reth_trie_common::HashedPostState;
use rstest::rstest;
use serial_test::serial;
use std::{fmt, sync::Arc};

/// Covers the contract's deployment and either of its calls.
const CONTRACT_GAS_LIMIT: u64 = 100_000;

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
            blocks.push(reth.commit(&BlockSpec::empty())?.recovered_block().clone());
        }
        let deploy_number = leading + 1;
        let specs = [
            BlockSpec::new(vec![TxSpec::create(
                initcode_returning(&BLOCK_HASH_RECORDER_RUNTIME),
                CONTRACT_GAS_LIMIT,
            )]),
            BlockSpec::new(vec![TxSpec::call(contract, Bytes::new(), CONTRACT_GAS_LIMIT)]),
            BlockSpec::new(vec![TxSpec::call(
                contract,
                block_number_calldata(deploy_number),
                CONTRACT_GAS_LIMIT,
            )]),
        ];
        for spec in &specs {
            blocks.push(reth.commit(spec)?.recovered_block().clone());
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
    let key_pair = fixed_signer();
    let sender = public_key_to_address(key_pair.public_key());
    let contract = sender.create(0);
    let chain_spec = chain_spec_with_address(sender);
    let store = create_mdbx_proofs_storage();
    let mut reth = SimulatedReth::new(chain_spec, key_pair, persistence)?;
    initialize_storage(store.clone(), &reth.factory, reth.head())?;
    let chain = ProofChain::commit(
        &mut reth,
        leading_blocks(persistence, parent_state),
        contract,
        content,
    )?;
    arrange_parent_state(&mut reth, parent_state, &chain)?;

    let store_with_hook = StoreWithReadHook::new(store.clone());
    // The engine writes its buffer to the store once it holds this many blocks, and `flush` only
    // waits for that write, so the threshold is the number of blocks the case stores. It also
    // keeps the preceding blocks buffered while the block under test executes, which the read
    // hook relies on.
    let stored_blocks = match outcome {
        Outcome::Stored => chain.preceding().len() + 1,
        Outcome::StateRootMismatch => chain.preceding().len(),
    } as u64;
    let engine = reth.spawn_engine(store_with_hook.clone(), stored_blocks, stored_blocks + 1);
    let reth = Arc::new(Mutex::new(reth));
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
