//! Builds one chain into three proofs stores, one indexed from reth's trie data, one re-executed
//! by the engine and one initialized at the tip and backfilled, and asserts that they serve the
//! same accounts, storage, trie nodes and proofs at every block.
//!
//! The index and execute paths are the ExEx's only cross-check of each other, and `proofs init`
//! plus backfill is the third way a store comes to hold a block, so the three must agree on every
//! kind of state change a block can make. The engine's buffer is not readable from outside, so
//! the flush cadence decides whether the execute path reads a parent from the buffer or from
//! disk, and the comparison runs over the persisted history once everything is flushed.

use super::{
    BlockSpec, TxSpec, block_timestamp, chain_spec_builder,
    contracts::{
        BLOCK_HASH_RECORDER_RUNTIME, BLOCK_HASH_SLOT, CREATE_AND_SELFDESTRUCT_SLOTS,
        SLOT_WRITER_SLOTS, block_number_calldata, create_and_selfdestruct_initcode,
        initcode_returning, slot_writer_calldata, slot_writer_runtime,
    },
    create_mdbx_proofs_storage, funded, initialize_storage, public_key_to_address,
    simulated_reth::{RethPersistence, SimulatedReth, fixed_signer},
};
use alloy_consensus::BlockHeader;
use alloy_eips::{
    BlockNumHash,
    eip1898::BlockWithParent,
    eip4788::{BEACON_ROOTS_ADDRESS, BEACON_ROOTS_CODE},
};
use alloy_genesis::GenesisAccount;
use alloy_primitives::{Address, B256, Bytes, U256, keccak256};
use reth_chainspec::ChainSpec;
use reth_ethereum_primitives::Block;
use reth_evm_ethereum::EthEvmConfig;
use reth_optimism_trie::{
    BackfillJob, InMemoryProofsStorage, OpProofStoragePruner, OpProofsProviderRO, OpProofsStore,
    ProofWindowRange, engine::EngineHandle, proof::DatabaseProof,
};
use reth_primitives_traits::{Account, RecoveredBlock};
use reth_provider::DatabaseProviderFactory;
use reth_trie::{hashed_cursor::HashedCursor, proof::Proof, trie_cursor::TrieCursor};
use reth_trie_common::{AccountProof, BranchNodeCompact, Nibbles, TrieInput};
use rstest::rstest;
use serial_test::serial;
use std::{
    collections::{BTreeMap, BTreeSet},
    fmt,
    sync::Arc,
};

/// Blocks in every chain: a multiple of both persistence thresholds, so the final flush leaves
/// nothing in the engine buffer.
const CHAIN_LENGTH: u64 = 10;

/// Empty blocks before the content, which places the content across the flush at block 5 under
/// the slower cadence.
const LEADING_FILLERS: u64 = 3;

/// Covers deploying either contract.
const DEPLOY_GAS_LIMIT: u64 = 200_000;

/// Covers eight first-time storage writes, or four writes and a self-destruct.
const WRITE_GAS_LIMIT: u64 = 400_000;

/// reth's persistence threshold: above the chain length, so reth keeps every block in memory
/// until the backfilled store needs them on disk.
const RETH_PERSISTENCE_THRESHOLD: u64 = 50;

/// Slots of the EIP-4788 ring buffer, which every block writes.
const BEACON_ROOTS_RING_LENGTH: u64 = 8191;

/// Where a self-destruct sends its balance.
const BENEFICIARY: Address = Address::repeat_byte(0xBE);

/// An address no transaction ever funds; touching it creates nothing.
const NEVER_FUNDED: Address = Address::repeat_byte(0xE2);

/// Three keys from `keys` whose hashes share the first nibble, the first two also the second, so
/// that a trie holding all three stores a branch node at the first nibble, which removing either
/// of the pair collapses.
///
/// reth stores no branch node for keys that all differ in their first nibble, so a small trie of
/// such keys has nothing to update or remove.
fn colliding_keys<K: Copy>(keys: impl IntoIterator<Item = K>, hash: impl Fn(K) -> B256) -> [K; 3] {
    let mut by_first_nibble: BTreeMap<u8, Vec<(u8, K)>> = BTreeMap::new();
    for key in keys {
        let hashed = hash(key);
        by_first_nibble.entry(hashed[0] >> 4).or_default().push((hashed[0] & 0x0f, key));
    }
    for group in by_first_nibble.values() {
        for (position, (second_nibble, key)) in group.iter().enumerate() {
            let sibling = group[position + 1..].iter().find(|(nibble, _)| nibble == second_nibble);
            let cousin = group.iter().find(|(nibble, _)| nibble != second_nibble);
            if let (Some((_, sibling)), Some((_, cousin))) = (sibling, cousin) {
                return [*key, *sibling, *cousin];
            }
        }
    }
    panic!("no three keys collide on their leading nibbles")
}

fn hashed_slot(slot: u8) -> B256 {
    keccak256(B256::with_last_byte(slot))
}

/// The slot writer's slots: three that collide on their leading nibbles, then five under distinct
/// first nibbles, so that filling them stores branch nodes and zeroing them removes those nodes.
fn storage_slots() -> [u8; SLOT_WRITER_SLOTS] {
    let colliding = colliding_keys(0..=u8::MAX, hashed_slot);
    let mut taken_nibbles = BTreeSet::from([hashed_slot(colliding[0])[0] >> 4]);
    let mut slots = colliding.to_vec();
    slots.extend((0..=u8::MAX).filter(|slot| taken_nibbles.insert(hashed_slot(*slot)[0] >> 4)));
    slots.truncate(SLOT_WRITER_SLOTS);
    slots.try_into().expect("eight slots")
}

/// Three genesis accounts colliding on their leading hashed nibbles: the empty one EIP-161
/// removes once a transaction touches it, its sibling and their cousin, both funded.
fn genesis_neighbors() -> [Address; 3] {
    colliding_keys((0x10..=u8::MAX).map(Address::with_last_byte), keccak256)
}

/// The account in the genesis allocation without balance, nonce or code.
fn empty_in_genesis() -> Address {
    genesis_neighbors()[0]
}

/// What the content blocks of the chain do.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum BlockContent {
    /// Deploys the slot writer, then fills its eight slots.
    DeployThenCall,
    /// Fills the slot writer's eight slots, zeroes half of them, then all but one.
    SstoreToZero,
    /// One transaction creates a contract that writes storage and self-destructs, which EIP-6780
    /// removes with its storage.
    CreateAndSelfdestructInOneTx,
    /// Self-destructs the slot writer after filling it; after Cancun the code and storage stay.
    SelfdestructPreExisting,
    /// Touches an empty account from genesis, which EIP-161 removes, and an address that never
    /// had anything.
    Eip161Touch,
    /// Stores `BLOCKHASH` of the first block.
    BlockHashRead,
}

impl BlockContent {
    /// The content blocks; the first transaction creates `contract` at the sender's nonce 0.
    fn blocks(self, contract: Address) -> Vec<BlockSpec> {
        let deploy_slot_writer = BlockSpec::new(vec![TxSpec::create(
            initcode_returning(&slot_writer_runtime(BENEFICIARY, storage_slots())),
            DEPLOY_GAS_LIMIT,
        )]);
        let write_slots = |values| {
            BlockSpec::new(vec![TxSpec::call(
                contract,
                slot_writer_calldata(values),
                WRITE_GAS_LIMIT,
            )])
        };
        match self {
            Self::DeployThenCall => vec![deploy_slot_writer, write_slots([1, 2, 3, 4, 5, 6, 7, 8])],
            // Zeroing the first colliding slot removes the stored branch the three share; keeping
            // only the cousin leaves a lone leaf.
            Self::SstoreToZero => vec![
                deploy_slot_writer,
                write_slots([7; SLOT_WRITER_SLOTS]),
                write_slots([0, 7, 7, 0, 0, 0, 7, 7]),
                write_slots([0, 0, 7, 0, 0, 0, 0, 0]),
            ],
            Self::CreateAndSelfdestructInOneTx => vec![BlockSpec::new(vec![
                TxSpec::create(create_and_selfdestruct_initcode(BENEFICIARY), WRITE_GAS_LIMIT)
                    .with_value(U256::from(1)),
            ])],
            Self::SelfdestructPreExisting => vec![
                deploy_slot_writer,
                write_slots([5; SLOT_WRITER_SLOTS]),
                BlockSpec::new(vec![
                    TxSpec::call(contract, Bytes::new(), WRITE_GAS_LIMIT).with_value(U256::from(1)),
                ]),
            ],
            Self::Eip161Touch => vec![BlockSpec::new(vec![
                TxSpec::transfer(empty_in_genesis(), U256::ZERO),
                TxSpec::transfer(NEVER_FUNDED, U256::ZERO),
            ])],
            Self::BlockHashRead => vec![
                BlockSpec::new(vec![TxSpec::create(
                    initcode_returning(&BLOCK_HASH_RECORDER_RUNTIME),
                    DEPLOY_GAS_LIMIT,
                )]),
                BlockSpec::new(vec![TxSpec::call(
                    contract,
                    block_number_calldata(1),
                    DEPLOY_GAS_LIMIT,
                )]),
            ],
        }
    }

    /// Checks that the chain did what the case says, on the state the stores agree on.
    fn assert_effect(
        self,
        genesis: &BlockView,
        tip: &BlockView,
        first_block: &RecoveredBlock<Block>,
        contract: Address,
    ) -> eyre::Result<()> {
        let hashed_contract = keccak256(contract);
        let contract_account = tip.accounts.get(&hashed_contract);
        let contract_storage = &tip.storages[&hashed_contract];
        let beneficiary_balance = tip.accounts.get(&keccak256(BENEFICIARY)).map(|a| a.balance);
        match self {
            Self::DeployThenCall => {
                eyre::ensure!(contract_account.is_some_and(|a| a.bytecode_hash.is_some()));
                eyre::ensure!(contract_storage.len() == SLOT_WRITER_SLOTS);
            }
            Self::SstoreToZero => {
                let cousin_slot = hashed_slot(storage_slots()[2]);
                eyre::ensure!(contract_storage.keys().eq([&cousin_slot]), "{contract_storage:?}");
            }
            Self::CreateAndSelfdestructInOneTx => {
                eyre::ensure!(contract_account.is_none());
                eyre::ensure!(contract_storage.is_empty());
                eyre::ensure!(tip.storage_tries[&hashed_contract].is_empty());
                eyre::ensure!(beneficiary_balance == Some(U256::from(1)));
            }
            Self::SelfdestructPreExisting => {
                eyre::ensure!(contract_account.is_some_and(|a| a.bytecode_hash.is_some()));
                eyre::ensure!(contract_storage.len() == SLOT_WRITER_SLOTS);
                eyre::ensure!(beneficiary_balance == Some(U256::from(1)));
            }
            Self::Eip161Touch => {
                eyre::ensure!(genesis.accounts.contains_key(&keccak256(empty_in_genesis())));
                eyre::ensure!(!tip.accounts.contains_key(&keccak256(empty_in_genesis())));
                eyre::ensure!(!tip.accounts.contains_key(&keccak256(NEVER_FUNDED)));
            }
            Self::BlockHashRead => {
                let stored = contract_storage.get(&keccak256(BLOCK_HASH_SLOT));
                eyre::ensure!(stored == Some(&U256::from_be_bytes(first_block.hash().0)));
            }
        }
        Ok(())
    }
}

/// The engine's persistence and backpressure thresholds.
#[derive(Clone, Copy, Debug)]
struct FlushCadence {
    backpressure: u64,
    persistence: u64,
}

impl FlushCadence {
    const fn new(persistence: u64, backpressure: u64) -> Self {
        Self { backpressure, persistence }
    }
}

/// Which store implementation the indexed and executed stores use.
#[derive(Clone, Copy, Debug)]
enum Backend {
    InMemory,
    Mdbx,
}

/// An address whose proof, with these slots, every store must serve identically.
struct ProofTarget {
    address: Address,
    slots: Vec<B256>,
}

/// Everything a store serves at one block, read through its cursors and proof builder.
struct BlockView {
    accounts: BTreeMap<B256, Account>,
    account_trie: BTreeMap<Nibbles, BranchNodeCompact>,
    proofs: BTreeMap<Address, AccountProof>,
    storage_tries: BTreeMap<B256, BTreeMap<Nibbles, BranchNodeCompact>>,
    storages: BTreeMap<B256, BTreeMap<B256, U256>>,
}

impl BlockView {
    /// Reads the store at `number`: every account and trie node, the storage and storage trie
    /// of each of `hashed_addresses`, and a proof for each target.
    fn read<S: OpProofsStore>(
        store: &S,
        number: u64,
        hashed_addresses: &BTreeSet<B256>,
        targets: &[ProofTarget],
    ) -> eyre::Result<Self> {
        let provider = store.provider_ro()?;
        let mut storages = BTreeMap::new();
        let mut storage_tries = BTreeMap::new();
        for hashed_address in hashed_addresses {
            let storage = collect_hashed(provider.storage_hashed_cursor(*hashed_address, number)?)?;
            let trie = collect_trie(provider.storage_trie_cursor(*hashed_address, number)?)?;
            storages.insert(*hashed_address, storage);
            storage_tries.insert(*hashed_address, trie);
        }
        let proofs = targets
            .iter()
            .map(|target| {
                let proof = Proof::overlay_account_proof(
                    provider.clone(),
                    number,
                    TrieInput::default(),
                    target.address,
                    &target.slots,
                )?;
                Ok((target.address, proof))
            })
            .collect::<eyre::Result<_>>()?;
        Ok(Self {
            accounts: collect_hashed(provider.account_hashed_cursor(number)?)?,
            account_trie: collect_trie(provider.account_trie_cursor(number)?)?,
            proofs,
            storage_tries,
            storages,
        })
    }

    /// Every proof must hash to the header's state root; this ties the stores to reth, so that
    /// three stores agreeing on the wrong state still fail.
    fn verify_proofs(&self, number: u64, store: &str, state_root: B256) -> eyre::Result<()> {
        for (address, proof) in &self.proofs {
            proof.verify(state_root).map_err(|err| {
                eyre::eyre!(
                    "the {store} store's proof of {address} at block {number} does not verify \
                     against the header state root: {err}"
                )
            })?;
        }
        Ok(())
    }
}

/// The hashed addresses every store must serve at `number`: the proof targets and every account
/// any store holds, so that storage a store keeps for an account it dropped is still compared.
fn hashed_addresses_at<A: OpProofsStore, B: OpProofsStore, C: OpProofsStore>(
    stores: (&A, &B, &C),
    number: u64,
    targets: &[ProofTarget],
) -> eyre::Result<BTreeSet<B256>> {
    let mut hashed_addresses: BTreeSet<B256> =
        targets.iter().map(|target| keccak256(target.address)).collect();
    hashed_addresses.extend(
        collect_hashed(stores.0.provider_ro()?.account_hashed_cursor(number)?)?.into_keys(),
    );
    hashed_addresses.extend(
        collect_hashed(stores.1.provider_ro()?.account_hashed_cursor(number)?)?.into_keys(),
    );
    hashed_addresses.extend(
        collect_hashed(stores.2.provider_ro()?.account_hashed_cursor(number)?)?.into_keys(),
    );
    Ok(hashed_addresses)
}

/// Every entry of a hashed cursor, from the lowest key up.
fn collect_hashed<C: HashedCursor>(mut cursor: C) -> eyre::Result<BTreeMap<B256, C::Value>> {
    let mut entries = BTreeMap::new();
    let mut entry = cursor.seek(B256::ZERO)?;
    while let Some((key, value)) = entry {
        entries.insert(key, value);
        entry = cursor.next()?;
    }
    Ok(entries)
}

/// Every node of a trie cursor, from the root path up.
fn collect_trie<C: TrieCursor>(
    mut cursor: C,
) -> eyre::Result<BTreeMap<Nibbles, BranchNodeCompact>> {
    let mut nodes = BTreeMap::new();
    let mut entry = cursor.seek(Nibbles::default())?;
    while let Some((path, node)) = entry {
        nodes.insert(path, node);
        entry = cursor.next()?;
    }
    Ok(nodes)
}

/// Fails on the first key at which `actual` differs from the indexed store's `expected`,
/// naming the block, the table and the key.
fn assert_same_table<K: fmt::Debug + Ord, V: fmt::Debug + PartialEq>(
    number: u64,
    store: &str,
    table: &str,
    expected: &BTreeMap<K, V>,
    actual: &BTreeMap<K, V>,
) -> eyre::Result<()> {
    for key in expected.keys().chain(actual.keys()).collect::<BTreeSet<_>>() {
        let (want, got) = (expected.get(key), actual.get(key));
        eyre::ensure!(
            want == got,
            "the {store} store differs from the indexed store at block {number} in {table} at \
             key {key:?}: indexed {want:?}, {store} {got:?}"
        );
    }
    Ok(())
}

fn assert_same_view(
    number: u64,
    store: &str,
    expected: &BlockView,
    actual: &BlockView,
) -> eyre::Result<()> {
    assert_same_table(number, store, "HashedAccounts", &expected.accounts, &actual.accounts)?;
    assert_same_table(number, store, "AccountsTrie", &expected.account_trie, &actual.account_trie)?;
    for (hashed_address, storage) in &expected.storages {
        let table = format!("HashedStorages[{hashed_address}]");
        assert_same_table(number, store, &table, storage, &actual.storages[hashed_address])?;
    }
    for (hashed_address, trie) in &expected.storage_tries {
        let table = format!("StoragesTrie[{hashed_address}]");
        assert_same_table(number, store, &table, trie, &actual.storage_tries[hashed_address])?;
    }
    assert_same_table(number, store, "AccountProof", &expected.proofs, &actual.proofs)
}

/// The chain: leading fillers, the content, and fillers up to [`CHAIN_LENGTH`].
fn chain_blocks(content: BlockContent, contract: Address) -> Vec<BlockSpec> {
    let mut blocks: Vec<BlockSpec> = (0..LEADING_FILLERS).map(|_| BlockSpec::new(vec![])).collect();
    blocks.extend(content.blocks(contract));
    assert!(blocks.len() as u64 <= CHAIN_LENGTH, "the content must fit the chain");
    blocks.resize_with(CHAIN_LENGTH as usize, || BlockSpec::new(vec![]));
    blocks
}

/// A Cancun chain funding `sender`, holding the empty account EIP-161 removes with its two
/// neighbors, and the EIP-4788 contract every block writes to.
fn cancun_chain_spec(sender: Address) -> Arc<ChainSpec> {
    let [empty, sibling, cousin] = genesis_neighbors();
    let one_wei = GenesisAccount { balance: U256::from(1), ..Default::default() };
    let beacon_roots =
        GenesisAccount { code: Some(BEACON_ROOTS_CODE.clone()), ..Default::default() };
    let alloc = [
        funded(sender),
        (empty, GenesisAccount::default()),
        (sibling, one_wei.clone()),
        (cousin, one_wei),
        (BEACON_ROOTS_ADDRESS, beacon_roots),
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

fn spawn_engine<S>(
    chain_spec: &Arc<ChainSpec>,
    reth: &SimulatedReth,
    store: S,
    cadence: FlushCadence,
) -> EngineHandle<Block>
where
    S: OpProofsStore + Clone + 'static,
{
    let pruner = OpProofStoragePruner::new(store.clone(), reth.provider.clone(), 1000);
    EngineHandle::spawn_with_thresholds(
        EthEvmConfig::ethereum(chain_spec.clone()),
        reth.provider.clone(),
        store,
        pruner,
        cadence.persistence,
        cadence.backpressure,
    )
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

fn block_with_parent(block: &RecoveredBlock<Block>) -> BlockWithParent {
    BlockWithParent::new(block.parent_hash(), block.num_hash())
}

fn run_parity_case<S>(
    make_store: impl Fn() -> S,
    cadence: FlushCadence,
    content: BlockContent,
) -> eyre::Result<()>
where
    S: OpProofsStore + Clone + Send + Sync + fmt::Debug + 'static,
{
    let key_pair = fixed_signer();
    let sender = public_key_to_address(key_pair.public_key());
    let contract = sender.create(0);
    let chain_spec = cancun_chain_spec(sender);
    let persistence = RethPersistence::threshold(RETH_PERSISTENCE_THRESHOLD);
    let mut reth = SimulatedReth::new(chain_spec.clone(), key_pair, persistence)?;
    let genesis = reth.head();

    let indexed_store = make_store();
    let executed_store = make_store();
    initialize_storage(indexed_store.clone(), &reth.factory, genesis)?;
    initialize_storage(executed_store.clone(), &reth.factory, genesis)?;
    let indexed_engine = spawn_engine(&chain_spec, &reth, indexed_store.clone(), cadence);
    let executed_engine = spawn_engine(&chain_spec, &reth, executed_store.clone(), cadence);

    let mut blocks = Vec::new();
    for spec in chain_blocks(content, contract) {
        let executed = reth.commit(&spec)?;
        let block = executed.recovered_block();
        indexed_engine.index_block(
            block_with_parent(block),
            (*executed.trie_updates()).clone(),
            (*executed.hashed_state()).clone(),
        )?;
        executed_engine.execute_block(block)?;
        blocks.push(block.clone());
    }
    let tip = reth.head();
    flush_until_persisted(&indexed_engine, &indexed_store, tip)?;
    flush_until_persisted(&executed_engine, &executed_store, tip)?;

    reth.persist_all()?;
    let backfilled_store = create_mdbx_proofs_storage();
    initialize_storage(backfilled_store.clone(), &reth.factory, tip)?;
    BackfillJob::new(reth.factory.database_provider_ro()?, backfilled_store.clone())
        .run(genesis.number)?;

    let targets = proof_targets(sender, contract);
    let window = ProofWindowRange { earliest: genesis, latest: tip };
    assert_eq!(indexed_store.provider_ro()?.get_proof_window()?, window, "indexed ProofWindow");
    assert_eq!(executed_store.provider_ro()?.get_proof_window()?, window, "executed ProofWindow");
    assert_eq!(
        backfilled_store.provider_ro()?.get_proof_window()?,
        window,
        "backfilled ProofWindow"
    );

    let mut genesis_view = None;
    let mut tip_view = None;
    for number in genesis.number..=tip.number {
        let stores = (&indexed_store, &executed_store, &backfilled_store);
        let hashed_addresses = hashed_addresses_at(stores, number, &targets)?;
        let indexed = BlockView::read(&indexed_store, number, &hashed_addresses, &targets)?;
        let executed = BlockView::read(&executed_store, number, &hashed_addresses, &targets)?;
        let backfilled = BlockView::read(&backfilled_store, number, &hashed_addresses, &targets)?;
        assert_same_view(number, "executed", &indexed, &executed)?;
        assert_same_view(number, "backfilled", &indexed, &backfilled)?;

        let state_root = match number {
            0 => chain_spec.genesis_header().state_root,
            _ => blocks[number as usize - 1].state_root(),
        };
        indexed.verify_proofs(number, "indexed", state_root)?;
        executed.verify_proofs(number, "executed", state_root)?;
        backfilled.verify_proofs(number, "backfilled", state_root)?;

        if number == genesis.number {
            genesis_view = Some(indexed);
        } else if number == tip.number {
            tip_view = Some(indexed);
        }
    }
    let (genesis_view, tip_view) =
        (genesis_view.expect("genesis read"), tip_view.expect("tip read"));
    content.assert_effect(&genesis_view, &tip_view, &blocks[0], contract)
}

/// The indexed, executed and backfilled stores serve the same state at every block, whatever the
/// blocks did, whether the engine flushed every block or every fifth, and on either backend. The
/// backfilled store is always MDBX, the only backend that supports backfill.
#[rstest]
#[case::deploy_then_call(BlockContent::DeployThenCall)]
#[case::sstore_to_zero(BlockContent::SstoreToZero)]
#[case::create_and_selfdestruct_in_one_tx(BlockContent::CreateAndSelfdestructInOneTx)]
#[case::selfdestruct_pre_existing(BlockContent::SelfdestructPreExisting)]
#[case::eip161_touch(BlockContent::Eip161Touch)]
#[case::block_hash_read(BlockContent::BlockHashRead)]
#[serial]
fn index_execute_and_backfill_stores_agree(
    #[values(FlushCadence::new(1, 2), FlushCadence::new(5, 10))] cadence: FlushCadence,
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[case] content: BlockContent,
) -> eyre::Result<()> {
    match backend {
        Backend::InMemory => run_parity_case(InMemoryProofsStorage::new, cadence, content),
        Backend::Mdbx => run_parity_case(create_mdbx_proofs_storage, cadence, content),
    }
}
