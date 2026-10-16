//! Builds one chain into four proofs stores, one indexed from reth's trie data, one re-executed
//! by the engine and two initialized at the tip and backfilled, and asserts that they serve the
//! same accounts, storage, trie nodes and proofs at every block.
//!
//! A store receives a block by indexing reth's trie data, by executing it, or through
//! `proofs init` plus backfill, so all of them must write the same state for every kind of state
//! change a block can make. The flush cadence decides whether the execute path reads a parent
//! from the engine buffer or from disk. The buffer is not readable from outside, so the
//! comparison runs over the persisted history once everything is flushed.

use super::{
    Backend, BlockSpec, FlushCadence, TxSpec, block_timestamp, chain_spec_builder,
    contracts::{
        BLOCK_HASH_RECORDER_RUNTIME, BLOCK_HASH_SLOT, CALLED_SLOT, CREATE_AND_SELFDESTRUCT_SLOTS,
        SLOT_WRITER_SLOTS, block_number_calldata, create_and_selfdestruct_initcode,
        initcode_returning, slot_writer_calldata, slot_writer_runtime,
    },
    create_mdbx_proofs_storage, funded, initialize_storage, public_key_to_address,
    simulated_reth::{RethPersistence, SimulatedReth, fixed_signer, index_executed_block},
    store_view::{BlockView, ProofTarget, assert_same_view},
};
use alloy_eips::{
    BlockNumHash,
    eip4788::{BEACON_ROOTS_ADDRESS, BEACON_ROOTS_CODE},
};
use alloy_genesis::GenesisAccount;
use alloy_primitives::{Address, B256, Bytes, U256, keccak256};
use reth_chainspec::ChainSpec;
use reth_ethereum_primitives::Block;
use reth_optimism_trie::{
    BackfillJob, InMemoryProofsStorage, MdbxProofsStorage, OpProofsProviderRO, OpProofsStore,
    ProofWindowRange, engine::EngineHandle,
};
use reth_provider::DatabaseProviderFactory;
use rstest::rstest;
use serial_test::serial;
use std::{collections::BTreeSet, sync::Arc};

/// Blocks in every chain.
const CHAIN_LENGTH: u64 = 10;

/// Empty blocks before the content, so that multi-block content straddles the flush at block 5
/// under the slower cadence.
const LEADING_FILLERS: u64 = 3;

/// Blocks per backfill transaction, so that a backfill commits more than once.
const BACKFILL_BATCH_SIZE: usize = 3;

/// Covers deploying either contract.
const DEPLOY_GAS_LIMIT: u64 = 200_000;

/// Covers eight first-time storage writes, or four writes and a self-destruct.
const WRITE_GAS_LIMIT: u64 = 400_000;

/// reth's persistence threshold, above the chain length so that reth's own persistence never
/// runs: where reth holds a parent is the parent-state cases' concern, and `persist_all` writes
/// the chain for the backfilled stores at the end.
const RETH_PERSISTENCE_THRESHOLD: u64 = CHAIN_LENGTH + 1;

/// Slots of the EIP-4788 ring buffer, which every block writes.
const BEACON_ROOTS_RING_LENGTH: u64 = 8191;

/// Where a self-destruct sends its balance.
const BENEFICIARY: Address = Address::repeat_byte(0xBE);

/// An address no transaction ever funds; touching it creates nothing.
const NEVER_FUNDED: Address = Address::repeat_byte(0xE2);

fn leading_nibble(hashed: B256) -> u8 {
    hashed[0] >> 4
}

/// Three keys from `keys` whose hashes share the first nibble, the first two also the second, so
/// that a trie holding all three stores a branch node at the first nibble, which removing either
/// of the pair collapses.
///
/// reth stores no branch node for keys that all differ in their first nibble, so a small trie of
/// such keys has nothing to update or remove.
fn colliding_keys<K: Copy>(keys: impl IntoIterator<Item = K>, hash: impl Fn(K) -> B256) -> [K; 3] {
    let keys: Vec<(u8, K)> = keys.into_iter().map(|key| (hash(key)[0], key)).collect();
    keys.iter()
        .enumerate()
        .find_map(|(index, &(leading_byte, key))| {
            let (_, sibling) =
                keys[index + 1..].iter().find(|(other, _)| *other == leading_byte)?;
            let (_, cousin) = keys
                .iter()
                .find(|(other, _)| other >> 4 == leading_byte >> 4 && *other != leading_byte)?;
            Some([key, *sibling, *cousin])
        })
        .expect("three keys collide on their leading nibbles")
}

fn hashed_slot(slot: u8) -> B256 {
    keccak256(B256::with_last_byte(slot))
}

/// The slot writer's slots: three that collide on their leading nibbles, so that filling them
/// stores a branch node and zeroing them removes it, then slots under distinct first nibbles to
/// fill the writer without a second collision.
fn storage_slots() -> [u8; SLOT_WRITER_SLOTS] {
    let colliding = colliding_keys(0..=u8::MAX, hashed_slot);
    let mut taken_nibbles = BTreeSet::from([leading_nibble(hashed_slot(colliding[0]))]);
    let spread =
        (0..=u8::MAX).filter(|&slot| taken_nibbles.insert(leading_nibble(hashed_slot(slot))));
    let slots: Vec<u8> = colliding.into_iter().chain(spread).take(SLOT_WRITER_SLOTS).collect();
    slots.try_into().expect("256 slots cover all sixteen leading nibbles")
}

/// Three genesis accounts, above the precompile addresses, colliding on their leading hashed
/// nibbles: the empty one EIP-161 removes once a transaction touches it, its sibling and their
/// cousin, both funded.
fn genesis_neighbors() -> [Address; 3] {
    colliding_keys((0x10..=u8::MAX).map(Address::with_last_byte), keccak256)
}

/// The account in the genesis allocation without balance, nonce or code.
fn empty_in_genesis() -> Address {
    genesis_neighbors()[0]
}

/// What the content blocks of the chain do.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum StateChange {
    /// Stores `BLOCKHASH` of the first block.
    BlockHashRead,
    /// One transaction creates a contract that writes storage and self-destructs, which EIP-6780
    /// removes with its storage.
    CreateAndSelfdestructInOneTx,
    /// Deploys the slot writer, then fills its eight slots.
    DeployThenCall,
    /// Touches an empty account from genesis, which EIP-161 removes, and an address that never
    /// had anything.
    Eip161Touch,
    /// Self-destructs the slot writer after filling it; after Cancun the code and storage stay.
    SelfdestructPreExisting,
    /// Fills the slot writer's eight slots, zeroes half of them, then all but one.
    SstoreToZero,
}

impl StateChange {
    /// The content blocks. `contract` is the address the sender's nonce-0 transaction creates.
    fn blocks(self, contract: Address) -> Vec<BlockSpec> {
        let deploy_slot_writer = || {
            BlockSpec::new(vec![TxSpec::create(
                initcode_returning(&slot_writer_runtime(BENEFICIARY, storage_slots())),
                DEPLOY_GAS_LIMIT,
            )])
        };
        let write_slots = |values| {
            BlockSpec::new(vec![TxSpec::call(
                contract,
                slot_writer_calldata(values),
                WRITE_GAS_LIMIT,
            )])
        };
        match self {
            Self::BlockHashRead => vec![
                BlockSpec::new(vec![TxSpec::create(
                    initcode_returning(&BLOCK_HASH_RECORDER_RUNTIME),
                    DEPLOY_GAS_LIMIT,
                )]),
                BlockSpec::new(vec![TxSpec::call(
                    contract,
                    block_number_calldata(1),
                    WRITE_GAS_LIMIT,
                )]),
            ],
            Self::CreateAndSelfdestructInOneTx => vec![BlockSpec::new(vec![
                TxSpec::create(create_and_selfdestruct_initcode(BENEFICIARY), WRITE_GAS_LIMIT)
                    .with_value(U256::from(1)),
            ])],
            Self::DeployThenCall => {
                vec![deploy_slot_writer(), write_slots([1, 2, 3, 4, 5, 6, 7, 8])]
            }
            Self::Eip161Touch => vec![BlockSpec::new(vec![
                TxSpec::transfer(empty_in_genesis(), U256::ZERO),
                TxSpec::transfer(NEVER_FUNDED, U256::ZERO),
            ])],
            Self::SelfdestructPreExisting => vec![
                deploy_slot_writer(),
                write_slots([5; SLOT_WRITER_SLOTS]),
                BlockSpec::new(vec![
                    TxSpec::call(contract, Bytes::new(), WRITE_GAS_LIMIT).with_value(U256::from(1)),
                ]),
            ],
            // Zeroing the first colliding slot removes the stored branch the three share; keeping
            // only the cousin leaves a lone leaf.
            Self::SstoreToZero => vec![
                deploy_slot_writer(),
                write_slots([7; SLOT_WRITER_SLOTS]),
                write_slots([0, 7, 7, 0, 0, 0, 7, 7]),
                write_slots([0, 0, 7, 0, 0, 0, 0, 0]),
            ],
        }
    }

    /// Checks that the chain did what the case says, on the state the stores agree on.
    fn assert_effect(
        self,
        genesis: &BlockView,
        tip: &BlockView,
        first_block_hash: B256,
        contract: Address,
    ) -> eyre::Result<()> {
        let hashed_contract = keccak256(contract);
        let contract_account = tip.accounts.get(&hashed_contract);
        let contract_storage = &tip.storages[&hashed_contract];
        let beneficiary_balance = tip.accounts.get(&keccak256(BENEFICIARY)).map(|a| a.balance);
        match self {
            Self::BlockHashRead => {
                let stored = contract_storage.slots.get(&keccak256(BLOCK_HASH_SLOT));
                eyre::ensure!(stored == Some(&U256::from_be_bytes(first_block_hash.0)));
            }
            Self::CreateAndSelfdestructInOneTx => {
                eyre::ensure!(contract_account.is_none());
                eyre::ensure!(contract_storage.slots.is_empty());
                eyre::ensure!(contract_storage.trie.is_empty());
                eyre::ensure!(beneficiary_balance == Some(U256::from(1)));
            }
            Self::DeployThenCall => {
                eyre::ensure!(contract_account.is_some_and(|a| a.bytecode_hash.is_some()));
                eyre::ensure!(contract_storage.slots.len() == SLOT_WRITER_SLOTS);
            }
            Self::Eip161Touch => {
                eyre::ensure!(genesis.accounts.contains_key(&keccak256(empty_in_genesis())));
                eyre::ensure!(!tip.accounts.contains_key(&keccak256(empty_in_genesis())));
                eyre::ensure!(!tip.accounts.contains_key(&keccak256(NEVER_FUNDED)));
            }
            Self::SelfdestructPreExisting => {
                eyre::ensure!(contract_account.is_some_and(|a| a.bytecode_hash.is_some()));
                eyre::ensure!(contract_storage.slots.len() == SLOT_WRITER_SLOTS);
                eyre::ensure!(beneficiary_balance == Some(U256::from(1)));
            }
            Self::SstoreToZero => {
                let cousin_slot = hashed_slot(storage_slots()[2]);
                let remaining = &contract_storage.slots;
                eyre::ensure!(remaining.keys().eq([&cousin_slot]), "{remaining:?}");
            }
        }
        Ok(())
    }
}

/// How a backfill reads the trie state it rewinds from.
#[derive(Clone, Copy, Debug)]
enum BackfillMode {
    /// Through the store's history, as `proofs init` does by default.
    History,
    /// Through a snapshot of the trie at the window's earliest block.
    Snapshot,
}

/// The stores under comparison; the indexed store is the reference the others are held to.
struct ParityStores<S> {
    backfilled: Arc<MdbxProofsStorage>,
    executed: S,
    indexed: S,
    snapshot_backfilled: Arc<MdbxProofsStorage>,
}

impl<S: OpProofsStore> ParityStores<S> {
    fn assert_proof_windows(&self, expected: ProofWindowRange) -> eyre::Result<()> {
        assert_proof_window(&self.indexed, "indexed", expected)?;
        assert_proof_window(&self.executed, "executed", expected)?;
        assert_proof_window(&self.backfilled, "backfilled", expected)?;
        assert_proof_window(&self.snapshot_backfilled, "snapshot-backfilled", expected)
    }

    /// Every store serves the indexed store's view at `number`, and that view's proofs hash to
    /// `state_root`.
    fn assert_agree_at(
        &self,
        number: u64,
        targets: &[ProofTarget],
        state_root: B256,
    ) -> eyre::Result<()> {
        let indexed = BlockView::read(&self.indexed, number, targets)?;
        let executed = BlockView::read(&self.executed, number, targets)?;
        let backfilled = BlockView::read(&self.backfilled, number, targets)?;
        let snapshot_backfilled = BlockView::read(&self.snapshot_backfilled, number, targets)?;
        assert_same_view(number, "executed", &indexed, &executed)?;
        assert_same_view(number, "backfilled", &indexed, &backfilled)?;
        assert_same_view(number, "snapshot-backfilled", &indexed, &snapshot_backfilled)?;
        indexed.verify_proofs(number, state_root)
    }
}

fn assert_proof_window<S: OpProofsStore>(
    store: &S,
    name: &str,
    expected: ProofWindowRange,
) -> eyre::Result<()> {
    let window = store.provider_ro()?.get_proof_window()?;
    eyre::ensure!(
        window == expected,
        "the {name} store's ProofWindow is {window:?}, not {expected:?}"
    );
    Ok(())
}

/// The chain: leading fillers, the content, and fillers up to [`CHAIN_LENGTH`].
fn chain_blocks(change: StateChange, contract: Address) -> Vec<BlockSpec> {
    let mut blocks: Vec<BlockSpec> = (0..LEADING_FILLERS).map(|_| BlockSpec::empty()).collect();
    blocks.extend(change.blocks(contract));
    assert!(blocks.len() as u64 <= CHAIN_LENGTH, "the content must fit the chain");
    blocks.resize_with(CHAIN_LENGTH as usize, BlockSpec::empty);
    blocks
}

/// A Cancun chain funding `sender`, holding the empty account EIP-161 removes with its two
/// neighbors, and the EIP-4788 contract every block writes to.
fn cancun_chain_spec(sender: Address) -> Arc<ChainSpec> {
    let [empty, sibling, cousin] = genesis_neighbors();
    let one_wei = GenesisAccount::default().with_balance(U256::from(1));
    let alloc = [
        funded(sender),
        (empty, GenesisAccount::default()),
        (sibling, one_wei.clone()),
        (cousin, one_wei),
        (
            BEACON_ROOTS_ADDRESS,
            GenesisAccount::default().with_code(Some(BEACON_ROOTS_CODE.clone())),
        ),
    ];
    Arc::new(chain_spec_builder(alloc).cancun_activated().build())
}

/// Every address the chain can touch, with the slots the contracts write; the beacon roots
/// contract writes the timestamp and root slots of each block.
fn proof_targets(sender: Address, contract: Address) -> Vec<ProofTarget> {
    let contract_slots: Vec<B256> = storage_slots()
        .into_iter()
        .chain(0..CREATE_AND_SELFDESTRUCT_SLOTS)
        .map(B256::with_last_byte)
        .chain([CALLED_SLOT, BLOCK_HASH_SLOT])
        .collect();
    let beacon_roots_slots = (1..=CHAIN_LENGTH)
        .flat_map(|number| {
            let timestamp_slot = block_timestamp(number) % BEACON_ROOTS_RING_LENGTH;
            [timestamp_slot, timestamp_slot + BEACON_ROOTS_RING_LENGTH]
        })
        .map(|slot| B256::from(U256::from(slot)))
        .collect();
    let [empty, sibling, cousin] = genesis_neighbors();
    let mut targets: Vec<ProofTarget> =
        [sender, Address::ZERO, BENEFICIARY, empty, sibling, cousin, NEVER_FUNDED, contract]
            .into_iter()
            .map(|address| ProofTarget { address, slots: contract_slots.clone() })
            .collect();
    targets.push(ProofTarget { address: BEACON_ROOTS_ADDRESS, slots: beacon_roots_slots });
    targets
}

/// `flush` only waits for the save in flight and the engine starts the next save after it
/// replies, so a buffer above the threshold drains over several flushes.
fn flush_until_persisted<S: OpProofsStore>(
    engine: &EngineHandle<Block>,
    store: &S,
    tip: BlockNumHash,
) -> eyre::Result<()> {
    for _ in 0..=CHAIN_LENGTH {
        engine.flush();
        if store.provider_ro()?.get_latest_block()? == tip {
            return Ok(());
        }
    }
    eyre::bail!("the engine did not persist through {tip:?}")
}

/// Commits the chain to reth while feeding every block to an indexing engine over one store and
/// an executing engine over another, and returns the two stores once both hold the whole chain.
fn index_and_execute_chain<S>(
    reth: &mut SimulatedReth,
    make_store: impl Fn() -> S,
    cadence: FlushCadence,
    blocks: Vec<BlockSpec>,
) -> eyre::Result<(S, S)>
where
    S: OpProofsStore + Clone + 'static,
{
    let indexed = make_store();
    let executed = make_store();
    initialize_storage(indexed.clone(), &reth.factory, reth.head())?;
    initialize_storage(executed.clone(), &reth.factory, reth.head())?;
    let indexed_engine =
        reth.spawn_engine(indexed.clone(), cadence.persistence, cadence.backpressure);
    let executed_engine =
        reth.spawn_engine(executed.clone(), cadence.persistence, cadence.backpressure);
    for spec in &blocks {
        let reth_block = reth.commit(spec)?;
        index_executed_block(&indexed_engine, &reth_block)?;
        executed_engine.execute_block(reth_block.recovered_block())?;
    }
    flush_until_persisted(&indexed_engine, &indexed, reth.head())?;
    flush_until_persisted(&executed_engine, &executed, reth.head())?;
    Ok((indexed, executed))
}

/// Builds a store the way `proofs init` does: initialized at reth's database tip, then backfilled
/// to genesis.
fn backfill_store(
    reth: &SimulatedReth,
    mode: BackfillMode,
) -> eyre::Result<Arc<MdbxProofsStorage>> {
    let store = create_mdbx_proofs_storage();
    initialize_storage(store.clone(), &reth.factory, reth.head())?;
    let job = BackfillJob::new(reth.factory.database_provider_ro()?, store.clone())
        .with_batch_size(BACKFILL_BATCH_SIZE);
    match mode {
        BackfillMode::History => job.run(0)?,
        BackfillMode::Snapshot => job.run_with_snapshot(0)?,
    }
    Ok(store)
}

fn run_parity_case<S>(
    make_store: impl Fn() -> S,
    cadence: FlushCadence,
    change: StateChange,
) -> eyre::Result<()>
where
    S: OpProofsStore + Clone + 'static,
{
    assert_eq!(
        CHAIN_LENGTH % cadence.persistence,
        0,
        "the final flush must leave nothing buffered"
    );
    let key_pair = fixed_signer();
    let sender = public_key_to_address(key_pair.public_key());
    let contract = sender.create(0);
    let persistence = RethPersistence::threshold(RETH_PERSISTENCE_THRESHOLD);
    let mut reth = SimulatedReth::new(cancun_chain_spec(sender), key_pair, persistence)?;
    let genesis = reth.head();

    let blocks = chain_blocks(change, contract);
    let (indexed, executed) = index_and_execute_chain(&mut reth, make_store, cadence, blocks)?;
    let tip = reth.head();
    reth.persist_all()?;
    let stores = ParityStores {
        backfilled: backfill_store(&reth, BackfillMode::History)?,
        executed,
        indexed,
        snapshot_backfilled: backfill_store(&reth, BackfillMode::Snapshot)?,
    };

    let targets = proof_targets(sender, contract);
    stores.assert_proof_windows(ProofWindowRange { earliest: genesis, latest: tip })?;
    for number in genesis.number..=tip.number {
        stores.assert_agree_at(number, &targets, reth.state_root(number))?;
    }
    let genesis_view = BlockView::read(&stores.indexed, genesis.number, &targets)?;
    let tip_view = BlockView::read(&stores.indexed, tip.number, &targets)?;
    change.assert_effect(&genesis_view, &tip_view, reth.block(1).hash(), contract)
}

/// The indexed, executed and both backfilled stores serve the same state at every block, whatever
/// the blocks did, whether the engine flushed every block or every fifth, and on either backend.
/// The backfilled stores are always MDBX, the only backend that supports backfill.
#[rstest]
#[case::block_hash_read(StateChange::BlockHashRead)]
#[case::create_and_selfdestruct_in_one_tx(StateChange::CreateAndSelfdestructInOneTx)]
#[case::deploy_then_call(StateChange::DeployThenCall)]
#[case::eip161_touch(StateChange::Eip161Touch)]
#[case::selfdestruct_pre_existing(StateChange::SelfdestructPreExisting)]
#[case::sstore_to_zero(StateChange::SstoreToZero)]
#[serial]
fn index_execute_and_backfill_stores_agree(
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[case] change: StateChange,
) -> eyre::Result<()> {
    over_backend!(backend, run_parity_case(cadence, change))
}
