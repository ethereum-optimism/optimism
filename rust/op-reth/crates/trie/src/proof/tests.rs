//! Multiproofs over the proofs storage.
//!
//! The V2 multiproof of each proofs state provider must verify against the state root, show the
//! same accounts, storage values and proof nodes as its V1 multiproof, and carry the masks of the
//! stored branch nodes on its paths, whether the proven block is flushed to the store, buffered in
//! the engine, or passed as the caller's trie input, with or without its trie nodes, and whether
//! the block under it is flushed or buffered.
//!
//! The fixture is three blocks on an empty store. Block 1 creates [`CONTRACT`] with [`LIVE_SLOT`]
//! and [`ZEROED_SLOT`], [`DESTROYED`] with [`LIVE_SLOT`], the [`EOA`] and [`UNTOUCHED`] accounts,
//! and enough filler accounts and slots for the tries to hold branch nodes on the paths the later
//! blocks change. Block 2 writes [`EOA`] and the contract's [`ZEROED_SLOT`], so that the tip's
//! values show only when the tip is applied over block 2. The tip, block 3, destroys
//! [`DESTROYED`], zeroes [`ZEROED_SLOT`] and updates [`EOA`]. [`ABSENT`] and [`ABSENT_SLOT`]
//! never exist.

use super::DatabaseStateRoot;
use crate::{
    BlockStateDiff, InMemoryProofsStorage, OpProofsStore,
    engine::buffer::overlay::MemoryOverlayOpProofsStateProviderRef,
    provider::OpProofsStateProviderRef,
    test_utils::{
        create_storage,
        proofs_store::{
            INITIAL_ANCHOR, buffered_state_provider, hashed_slot, hashed_storage, plain_slot,
            seed_initial_storage, state_provider, store_block,
        },
    },
};
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_primitives::{
    Address, B256, Bytes, U256, keccak256,
    map::{B256Map, B256Set},
};
use reth_primitives_traits::Account;
use reth_provider::StateProofProvider;
use reth_trie::{
    AccountProof, BranchNodeCompact, BranchNodeMasks, DecodedMultiProofV2, EMPTY_ROOT_HASH,
    HashedPostState, MultiProofTargets, MultiProofTargetsV2, Nibbles, ProofTrieNodeV2,
    ProofV2Target, ProofV2TargetParent, StateRoot, TrieInput, TrieNodeV2,
    root::{state_root_unsorted, storage_root_unsorted},
    updates::TrieUpdatesSorted,
};
use rstest::rstest;
use std::{
    collections::{HashMap, HashSet},
    ops::Range,
    sync::LazyLock,
};

const CONTRACT: Address = Address::repeat_byte(0x01);
const DESTROYED: Address = Address::repeat_byte(0x02);
const EOA: Address = Address::repeat_byte(0x03);
const UNTOUCHED: Address = Address::repeat_byte(0x04);
const ABSENT: Address = Address::repeat_byte(0x05);

/// Plain slot of [`CONTRACT`] holding [`LIVE_VALUE`] at the tip.
const LIVE_SLOT: u64 = 1;
/// Plain slot of [`CONTRACT`] that block 2 rewrites and the tip sets to zero.
const ZEROED_SLOT: u64 = 2;
/// Plain slot no account ever writes.
const ABSENT_SLOT: u64 = 3;
const LIVE_VALUE: u64 = 11;

/// Plain slots of [`CONTRACT`] and [`DESTROYED`] that blocks 2 and 3 leave alone.
const FILLER_SLOTS: Range<u64> = 4..260;

const CONTRACT_TARGET: (Address, &[u64]) = (CONTRACT, &[LIVE_SLOT, ZEROED_SLOT, ABSENT_SLOT]);
const DESTROYED_TARGET: (Address, &[u64]) = (DESTROYED, &[LIVE_SLOT]);
const EOA_TARGET: (Address, &[u64]) = (EOA, &[ABSENT_SLOT]);
const UNTOUCHED_TARGET: (Address, &[u64]) = (UNTOUCHED, &[ABSENT_SLOT]);
const ABSENT_TARGET: (Address, &[u64]) = (ABSENT, &[LIVE_SLOT]);
const EVERY_TARGET: &[(Address, &[u64])] =
    &[CONTRACT_TARGET, DESTROYED_TARGET, EOA_TARGET, UNTOUCHED_TARGET, ABSENT_TARGET];

const BLOCK_ONE: BlockWithParent =
    BlockWithParent::new(INITIAL_ANCHOR.hash, NumHash::new(1, B256::repeat_byte(0xb1)));
const BLOCK_TWO: BlockWithParent =
    BlockWithParent::new(BLOCK_ONE.block.hash, NumHash::new(2, B256::repeat_byte(0xb2)));
const TIP: BlockWithParent =
    BlockWithParent::new(BLOCK_TWO.block.hash, NumHash::new(3, B256::repeat_byte(0xb3)));

/// The proofs store backend a case runs against.
#[derive(Clone, Copy, Debug)]
enum StoreKind {
    InMemory,
    Mdbx,
}

/// Where the tip block and block 2 live when the proofs are requested.
#[derive(Clone, Copy, Debug)]
enum TipPlacement {
    /// Held in the engine's buffer, which the overlay state provider lays over the store holding
    /// block 2.
    Buffered,
    /// Held in the engine's buffer together with block 2, over the store holding block 1.
    BufferedWithParent,
    /// Flushed to the store.
    Flushed,
    /// Passed by the caller as the trie input's nodes and state over the store holding block 2.
    TrieInputBlocks,
    /// Passed by the caller as the trie input's nodes and state over the engine's buffer holding
    /// block 2.
    TrieInputOverBuffered,
    /// Passed by the caller as the trie input's state alone over the store holding block 2, so
    /// that only the prefix sets keep the stored branch nodes on the changed paths from being
    /// reused.
    TrieInputState,
}

/// What the V2 targets of a case request for each of its addresses.
#[derive(Clone, Copy, Debug)]
enum TargetShape {
    /// The account and its slots, each from its trie's root.
    AccountAndStorage,
    /// The account alone, with no storage targets for the address.
    AccountOnly,
    /// The account and its slots, each below its trie's already revealed root branch.
    BelowKnownRoot,
    /// The slots alone, with no account target for the address.
    StorageOnly,
}

impl TargetShape {
    const fn targets_account(self) -> bool {
        !matches!(self, Self::StorageOnly)
    }

    const fn targets_storage(self) -> bool {
        !matches!(self, Self::AccountOnly)
    }

    /// The slots of a target that `self` requests.
    fn targeted_slots(self, slots: &[u64]) -> &[u64] {
        if self.targets_storage() { slots } else { &[] }
    }
}

fn contract_account() -> Account {
    Account { balance: U256::from(100), bytecode_hash: Some(B256::repeat_byte(0xcc)), nonce: 1 }
}

fn destroyed_before_tip() -> Account {
    Account { balance: U256::from(200), bytecode_hash: Some(B256::repeat_byte(0xdd)), nonce: 1 }
}

fn eoa_account(nonce: u64, balance: u64) -> Account {
    Account { balance: U256::from(balance), bytecode_hash: None, nonce }
}

fn eoa_at_tip() -> Account {
    eoa_account(2, 250)
}

fn untouched_account() -> Account {
    eoa_account(5, 400)
}

/// Accounts the later blocks leave alone, one per last byte of the address, so that the account
/// trie holds branch nodes on the paths they change.
fn filler_accounts() -> impl Iterator<Item = (Address, Account)> {
    (0..=u8::MAX).map(|byte| (Address::with_last_byte(byte), eoa_account(0, 1)))
}

/// Plain storage the later blocks leave alone, so that a storage trie holding it has branch nodes
/// on the paths they change.
fn filler_slots() -> Vec<(u64, u64)> {
    FILLER_SLOTS.map(|slot| (slot, slot)).collect()
}

/// Plain storage of [`CONTRACT`] at the tip.
fn contract_storage_at_tip() -> Vec<(u64, u64)> {
    [vec![(LIVE_SLOT, LIVE_VALUE)], filler_slots()].concat()
}

/// Plain storage of [`DESTROYED`] before the tip destroys it.
fn destroyed_storage_before_tip() -> Vec<(u64, u64)> {
    [vec![(LIVE_SLOT, 33)], filler_slots()].concat()
}

/// Block 1's state: every account the fixture holds, with the storage the later blocks change.
fn first_block_state() -> HashedPostState {
    let accounts = filler_accounts().chain([
        (CONTRACT, contract_account()),
        (DESTROYED, destroyed_before_tip()),
        (EOA, eoa_account(0, 300)),
        (UNTOUCHED, untouched_account()),
    ]);
    let contract_storage = [contract_storage_at_tip(), vec![(ZEROED_SLOT, 22)]].concat();
    let mut state = HashedPostState::default();
    state.accounts.extend(accounts.map(|(address, account)| (keccak256(address), Some(account))));
    state.storages.insert(keccak256(CONTRACT), hashed_storage(&contract_storage));
    state.storages.insert(keccak256(DESTROYED), hashed_storage(&destroyed_storage_before_tip()));
    state
}

/// Block 2's state: values of [`EOA`] and [`ZEROED_SLOT`] that the tip overwrites.
fn second_block_state() -> HashedPostState {
    let mut state = HashedPostState::default();
    state.accounts.insert(keccak256(EOA), Some(eoa_account(1, 275)));
    state.storages.insert(keccak256(CONTRACT), hashed_storage(&[(ZEROED_SLOT, 23)]));
    state
}

/// The tip's state, with every slot of the destroyed account written as an explicit zero so that
/// the storage prefix set covers them.
fn tip_state() -> HashedPostState {
    let wiped_storage: Vec<(u64, u64)> =
        destroyed_storage_before_tip().into_iter().map(|(slot, _)| (slot, 0)).collect();
    let mut state = HashedPostState::default();
    state.accounts.insert(keccak256(DESTROYED), None);
    state.accounts.insert(keccak256(EOA), Some(eoa_at_tip()));
    state.storages.insert(keccak256(CONTRACT), hashed_storage(&[(ZEROED_SLOT, 0)]));
    state.storages.insert(keccak256(DESTROYED), hashed_storage(&wiped_storage));
    state
}

/// Root of `address`'s storage trie among `storage_roots`, empty when it has none.
fn storage_root_of(storage_roots: &HashMap<Address, B256>, address: Address) -> B256 {
    storage_roots.get(&address).copied().unwrap_or(EMPTY_ROOT_HASH)
}

/// The tip's plain state and the roots computed from it, independently of the store.
struct PlainTip {
    accounts: HashMap<Address, Account>,
    state_root: B256,
    storage_roots: HashMap<Address, B256>,
    storages: HashMap<Address, HashMap<u64, u64>>,
}

impl PlainTip {
    fn compute() -> Self {
        let accounts: HashMap<Address, Account> = filler_accounts()
            .chain([
                (CONTRACT, contract_account()),
                (EOA, eoa_at_tip()),
                (UNTOUCHED, untouched_account()),
            ])
            .collect();
        let storages: HashMap<Address, HashMap<u64, u64>> =
            HashMap::from([(CONTRACT, contract_storage_at_tip().into_iter().collect())]);
        let storage_roots: HashMap<Address, B256> = storages
            .iter()
            .map(|(&address, storage)| {
                let hashed_storage =
                    storage.iter().map(|(&slot, &value)| (hashed_slot(slot), U256::from(value)));
                (address, storage_root_unsorted(hashed_storage))
            })
            .collect();
        let state_root = state_root_unsorted(accounts.iter().map(|(&address, account)| {
            let storage_root = storage_root_of(&storage_roots, address);
            (keccak256(address), account.into_trie_account(storage_root))
        }));
        Self { accounts, state_root, storage_roots, storages }
    }

    fn account(&self, address: Address) -> Option<Account> {
        self.accounts.get(&address).copied()
    }

    fn storage_root(&self, address: Address) -> B256 {
        storage_root_of(&self.storage_roots, address)
    }

    fn value(&self, address: Address, slot: u64) -> U256 {
        let value = self.storages.get(&address).and_then(|storage| storage.get(&slot));
        U256::from(value.copied().unwrap_or_default())
    }
}

static PLAIN_TIP: LazyLock<PlainTip> = LazyLock::new(PlainTip::compute);

/// The fixture's blocks, each computed on top of the previous one over a scratch in-memory
/// store.
struct FixtureBlocks {
    first: BlockStateDiff,
    second: BlockStateDiff,
    tip: BlockStateDiff,
    tip_state_root: B256,
}

impl FixtureBlocks {
    fn compute() -> Self {
        let store = InMemoryProofsStorage::new();
        seed_initial_storage(&store, &[]);
        let (first, _) = compute_block(&store, INITIAL_ANCHOR.number, first_block_state());
        store_block(&store, BLOCK_ONE, first.clone());
        let (second, _) = compute_block(&store, BLOCK_ONE.block.number, second_block_state());
        store_block(&store, BLOCK_TWO, second.clone());
        let (tip, tip_state_root) = compute_block(&store, BLOCK_TWO.block.number, tip_state());
        Self { first, second, tip, tip_state_root }
    }

    /// The blocks' trie updates, newest first.
    fn trie_updates_newest_first(&self) -> [&TrieUpdatesSorted; 3] {
        [&self.tip, &self.second, &self.first].map(|block| &block.sorted_trie_updates)
    }
}

static BLOCKS: LazyLock<FixtureBlocks> = LazyLock::new(FixtureBlocks::compute);

/// Computes `post_state` on top of block `parent_number` of `store`, returning the block's diff
/// and state root.
fn compute_block<S: OpProofsStore>(
    store: &S,
    parent_number: u64,
    post_state: HashedPostState,
) -> (BlockStateDiff, B256) {
    let provider = store.provider_ro().expect("provider ro");
    let (state_root, trie_updates) =
        StateRoot::overlay_root_with_updates(provider, parent_number, post_state.clone())
            .expect("state root");
    let diff = BlockStateDiff {
        sorted_post_state: post_state.into_sorted(),
        sorted_trie_updates: trie_updates.into_sorted(),
    };
    (diff, state_root)
}

fn trie_input(diff: &BlockStateDiff) -> TrieInput {
    TrieInput::from_blocks_sorted([(&diff.sorted_post_state, &diff.sorted_trie_updates)])
}

/// Hash of the root node among `nodes`, or `None` when the proof holds no root node.
fn root_hash(nodes: &[ProofTrieNodeV2]) -> Option<B256> {
    nodes
        .iter()
        .find(|node| node.path.is_empty())
        .map(|root| keccak256(alloy_rlp::encode(&root.node)))
}

/// `proofs` without the root node of each trie.
fn below_roots(proofs: &DecodedMultiProofV2) -> DecodedMultiProofV2 {
    let below_root = |nodes: &[ProofTrieNodeV2]| -> Vec<ProofTrieNodeV2> {
        nodes.iter().filter(|node| !node.path.is_empty()).cloned().collect()
    };
    DecodedMultiProofV2 {
        account_proofs: below_root(&proofs.account_proofs),
        storage_proofs: proofs
            .storage_proofs
            .iter()
            .map(|(&hashed_address, nodes)| (hashed_address, below_root(nodes)))
            .collect(),
    }
}

type NodeUpdates = [(Nibbles, Option<BranchNodeCompact>)];

fn storage_nodes(updates: &TrieUpdatesSorted, address: Address) -> &NodeUpdates {
    updates
        .storage_tries_ref()
        .get(&keccak256(address))
        .map(|trie| trie.storage_nodes_ref())
        .unwrap_or_default()
}

/// Whether `rewritten` updates or removes a branch node below the root on `key`'s path at which
/// `stored` holds one, so that the two decide between them what a proof of `key` reads there.
fn rewrites_stored_node_on_path(stored: &NodeUpdates, rewritten: &NodeUpdates, key: B256) -> bool {
    let key_path = Nibbles::unpack(key);
    let stored_paths: HashSet<Nibbles> =
        stored.iter().filter(|(_, node)| node.is_some()).map(|(path, _)| *path).collect();
    rewritten.iter().any(|(path, _)| {
        !path.is_empty() && key_path.starts_with(path) && stored_paths.contains(path)
    })
}

/// Masks of the branch node at `path` after the newest of `updates_newest_first`: `None` when no
/// block stores one there or the newest write there removes it.
fn stored_branch_masks(
    updates_newest_first: [&NodeUpdates; 3],
    path: &Nibbles,
) -> Option<BranchNodeMasks> {
    updates_newest_first
        .iter()
        .find_map(|updates| updates.iter().find(|(stored_path, _)| stored_path == path))
        .and_then(|(_, node)| node.as_ref())
        .map(|node| BranchNodeMasks { hash_mask: node.hash_mask, tree_mask: node.tree_mask })
}

/// Checks that each branch node among `nodes` carries the masks of the branch node the blocks
/// store at its path, and that the root branch carries none.
fn assert_branch_masks_stored(
    nodes: &[ProofTrieNodeV2],
    updates_newest_first: [&NodeUpdates; 3],
    trie: &str,
) {
    for proof_node in nodes {
        let TrieNodeV2::Branch(branch) = &proof_node.node else { continue };
        let branch_path = proof_node.path.join(&branch.key);
        let expected = if branch_path.is_empty() {
            None
        } else {
            stored_branch_masks(updates_newest_first, &branch_path)
        };
        assert_eq!(proof_node.masks, expected, "{trie} branch node masks at {branch_path:?}");
    }
}

/// Proofs state provider reading `store` at `block_number`.
fn state_provider_at<S: OpProofsStore>(
    store: &S,
    block_number: u64,
) -> OpProofsStateProviderRef<'static, S::ProviderRO<'_>> {
    state_provider(store.provider_ro().expect("provider ro"), block_number)
}

/// Overlay state provider at the newest of `blocks`, which are buffered over `store` at
/// `store_tip`.
fn buffered_provider_at<S: OpProofsStore>(
    store: &S,
    store_tip: u64,
    blocks: impl IntoIterator<Item = (BlockWithParent, BlockStateDiff)>,
) -> MemoryOverlayOpProofsStateProviderRef<'static, S::ProviderRO<'_>> {
    buffered_state_provider(state_provider_at(store, store_tip), blocks)
}

fn v1_targets(targets: &[(Address, &[u64])], shape: TargetShape) -> MultiProofTargets {
    targets
        .iter()
        .map(|&(address, slots)| {
            let slots: B256Set =
                shape.targeted_slots(slots).iter().copied().map(hashed_slot).collect();
            (keccak256(address), slots)
        })
        .collect()
}

/// V2 targets of `targets` shaped by `shape`, each with `parent` revealed, sorted as the proof
/// calculators expect.
fn v2_targets(
    targets: &[(Address, &[u64])],
    shape: TargetShape,
    parent: ProofV2TargetParent,
) -> MultiProofTargetsV2 {
    let sorted_targets = |keys: &mut dyn Iterator<Item = B256>| -> Vec<ProofV2Target> {
        let mut sorted: Vec<_> =
            keys.map(|key| ProofV2Target::new(key).with_parent(parent)).collect();
        sorted.sort_unstable_by_key(|target| target.key_nibbles);
        sorted
    };
    let account_targets = if shape.targets_account() {
        sorted_targets(&mut targets.iter().map(|&(address, _)| keccak256(address)))
    } else {
        Vec::new()
    };
    let storage_targets: B256Map<Vec<ProofV2Target>> = if shape.targets_storage() {
        targets
            .iter()
            .map(|&(address, slots)| {
                (keccak256(address), sorted_targets(&mut slots.iter().copied().map(hashed_slot)))
            })
            .collect()
    } else {
        B256Map::default()
    };
    MultiProofTargetsV2 { account_targets, storage_targets }
}

/// The proof nodes of each slot proof of `proof`, keyed by slot.
fn slot_proof_nodes(proof: &AccountProof) -> Vec<(B256, &[Bytes])> {
    proof.storage_proofs.iter().map(|slot| (slot.key, slot.proof.as_slice())).collect()
}

/// Requests the V1 and V2 multiproofs of `targets` shaped by `shape` from `provider` over
/// `input` and checks that the V2 proof nodes recompute the tip's roots and carry the stored
/// masks, that each version verifies against the tip and shows its values, and that the two
/// agree on accounts, storage roots and proof nodes. Below a known root, the V2 proof must hold
/// the nodes of the proof from the root, less the root.
fn assert_multiproofs_agree<P: StateProofProvider>(
    provider: &P,
    input: TrieInput,
    targets: &[(Address, &[u64])],
    shape: TargetShape,
) {
    let v1 = provider.multiproof(input.clone(), v1_targets(targets, shape)).expect("multiproof");
    let v2 = provider
        .multiproof_v2(input.clone(), v2_targets(targets, shape, ProofV2TargetParent::NONE))
        .expect("multiproof v2");
    assert_v2_roots_and_masks(&v2, targets, shape);

    for &(address, slots) in targets {
        let slots = shape.targeted_slots(slots);
        let plain_slots: Vec<B256> = slots.iter().copied().map(plain_slot).collect();
        let v1_proof = v1.account_proof(address, &plain_slots).expect("v1 account proof");
        let v2_proof = v2.account_proof(address, &plain_slots).expect("v2 account proof");
        for (version, proof) in [("v1", &v1_proof), ("v2", &v2_proof)] {
            assert_proof_shows_tip(version, address, slots, shape, proof);
        }
        if shape.targets_account() {
            assert_account_proofs_agree(address, &v1_proof, &v2_proof);
        }
        assert_storage_proofs_agree(address, &v1_proof, &v2_proof);
    }

    if matches!(shape, TargetShape::BelowKnownRoot) {
        let below_known_root = provider
            .multiproof_v2(input, v2_targets(targets, shape, ProofV2TargetParent::new(0)))
            .expect("multiproof v2 below the known root");
        assert_eq!(below_known_root, below_roots(&v2), "v2 proofs below the known root");
    }
}

/// Checks that the V2 proof nodes recompute the tip's state root and the storage root of each
/// of `targets`, where `shape` targets them, and carry the masks of the stored branch nodes.
fn assert_v2_roots_and_masks(
    v2: &DecodedMultiProofV2,
    targets: &[(Address, &[u64])],
    shape: TargetShape,
) {
    let trie_updates = BLOCKS.trie_updates_newest_first();
    let expected_state_root = shape.targets_account().then_some(BLOCKS.tip_state_root);
    assert_eq!(root_hash(&v2.account_proofs), expected_state_root, "v2 account proofs root");
    assert_branch_masks_stored(
        &v2.account_proofs,
        trie_updates.map(|updates| updates.account_nodes_ref()),
        "account",
    );

    for &(address, _) in targets {
        let v2_storage_nodes = v2.storage_proofs.get(&keccak256(address)).map(Vec::as_slice);
        let expected_storage_root =
            shape.targets_storage().then(|| PLAIN_TIP.storage_root(address));
        assert_eq!(
            v2_storage_nodes.and_then(root_hash),
            expected_storage_root,
            "v2 storage proofs root of {address}"
        );
        assert_branch_masks_stored(
            v2_storage_nodes.unwrap_or_default(),
            trie_updates.map(|updates| storage_nodes(updates, address)),
            &format!("storage of {address}"),
        );
    }
}

/// Checks that the `version` proof of `address` and its `slots` verifies against the tip's roots
/// and shows the tip's storage values and, where `shape` targets it, the tip's account.
fn assert_proof_shows_tip(
    version: &str,
    address: Address,
    slots: &[u64],
    shape: TargetShape,
    proof: &AccountProof,
) {
    if shape.targets_account() {
        assert_eq!(proof.verify(BLOCKS.tip_state_root), Ok(()), "{version} proof of {address}");
        assert_eq!(proof.info, PLAIN_TIP.account(address), "{version} account of {address}");
    }
    for slot_proof in &proof.storage_proofs {
        assert_eq!(
            slot_proof.verify(PLAIN_TIP.storage_root(address)),
            Ok(()),
            "{version} proof of slot {} of {address}",
            slot_proof.key
        );
    }
    let values: Vec<U256> = proof.storage_proofs.iter().map(|slot| slot.value).collect();
    let expected_values: Vec<U256> =
        slots.iter().map(|&slot| PLAIN_TIP.value(address, slot)).collect();
    assert_eq!(values, expected_values, "{version} storage of {address}");
}

/// Checks that the V2 proof of `address` shows the account, storage root and proof nodes of its
/// V1 proof.
fn assert_account_proofs_agree(address: Address, v1: &AccountProof, v2: &AccountProof) {
    assert_eq!(
        (v2.info, v2.storage_root),
        (v1.info, v1.storage_root),
        "v2 account of {address} matches v1"
    );
    assert_eq!(v2.proof, v1.proof, "v2 account proof nodes of {address} match v1");
}

/// Checks that the V2 slot proofs of `address` hold the proof nodes of its V1 slot proofs.
fn assert_storage_proofs_agree(address: Address, v1: &AccountProof, v2: &AccountProof) {
    assert_eq!(
        slot_proof_nodes(v2),
        slot_proof_nodes(v1),
        "v2 storage proof nodes of {address} match v1"
    );
}

/// Seeds a fresh `store` with block 1, places block 2 and the tip as `placement` says, and checks
/// the multiproofs of `targets` shaped by `shape` at the tip.
fn run_case(
    store: StoreKind,
    placement: TipPlacement,
    targets: &[(Address, &[u64])],
    shape: TargetShape,
) {
    match store {
        StoreKind::InMemory => {
            run_case_on(&InMemoryProofsStorage::new(), placement, targets, shape)
        }
        StoreKind::Mdbx => run_case_on(&*create_storage(), placement, targets, shape),
    }
}

fn run_case_on<S: OpProofsStore>(
    store: &S,
    placement: TipPlacement,
    targets: &[(Address, &[u64])],
    shape: TargetShape,
) {
    let flush = |(block, diff)| store_block(store, block, diff);
    let second = || (BLOCK_TWO, BLOCKS.second.clone());
    let tip = || (TIP, BLOCKS.tip.clone());
    seed_initial_storage(store, &[]);
    flush((BLOCK_ONE, BLOCKS.first.clone()));

    match placement {
        TipPlacement::Buffered => {
            flush(second());
            let provider = buffered_provider_at(store, BLOCK_TWO.block.number, [tip()]);
            assert_multiproofs_agree(&provider, TrieInput::default(), targets, shape);
        }
        TipPlacement::BufferedWithParent => {
            let provider = buffered_provider_at(store, BLOCK_ONE.block.number, [second(), tip()]);
            assert_multiproofs_agree(&provider, TrieInput::default(), targets, shape);
        }
        TipPlacement::Flushed => {
            flush(second());
            flush(tip());
            let provider = state_provider_at(store, TIP.block.number);
            assert_multiproofs_agree(&provider, TrieInput::default(), targets, shape);
        }
        TipPlacement::TrieInputBlocks => {
            flush(second());
            let provider = state_provider_at(store, BLOCK_TWO.block.number);
            assert_multiproofs_agree(&provider, trie_input(&BLOCKS.tip), targets, shape);
        }
        TipPlacement::TrieInputOverBuffered => {
            let provider = buffered_provider_at(store, BLOCK_ONE.block.number, [second()]);
            assert_multiproofs_agree(&provider, trie_input(&BLOCKS.tip), targets, shape);
        }
        TipPlacement::TrieInputState => {
            flush(second());
            let provider = state_provider_at(store, BLOCK_TWO.block.number);
            let input = TrieInput::from_state(tip_state());
            assert_multiproofs_agree(&provider, input, targets, shape);
        }
    }
}

/// The tip root the store computes is the plain tip state's root, so the cases' expected root
/// does not depend on the store.
#[test]
fn fixture_tip_root_matches_plain_state() {
    assert_eq!(BLOCKS.tip_state_root, PLAIN_TIP.state_root);
}

/// Block 1 stores branch nodes of the account trie on the EOA's path and of both contracts'
/// storage tries on the paths of the slots the tip changes, and the tip rewrites or removes
/// them, so the proofs over it read stored and overlaid branch nodes, not leaves alone.
#[test]
fn fixture_tip_rewrites_stored_branch_nodes() {
    let stored = &BLOCKS.first.sorted_trie_updates;
    let rewritten = &BLOCKS.tip.sorted_trie_updates;

    assert!(
        rewrites_stored_node_on_path(
            stored.account_nodes_ref(),
            rewritten.account_nodes_ref(),
            keccak256(EOA)
        ),
        "account branch node on the EOA's path rewritten"
    );
    assert!(
        rewrites_stored_node_on_path(
            storage_nodes(stored, CONTRACT),
            storage_nodes(rewritten, CONTRACT),
            hashed_slot(ZEROED_SLOT)
        ),
        "contract storage branch node on the zeroed slot's path rewritten"
    );
    assert!(
        rewrites_stored_node_on_path(
            storage_nodes(stored, DESTROYED),
            storage_nodes(rewritten, DESTROYED),
            hashed_slot(LIVE_SLOT)
        ),
        "destroyed account's storage branch node on the live slot's path rewritten"
    );
    assert!(
        storage_nodes(rewritten, DESTROYED).iter().all(|(_, node)| node.is_none()),
        "destroyed account's storage branch nodes removed"
    );
}

/// Block 2 and the tip both write a branch node below the root on the EOA's path in the account
/// trie and on the zeroed slot's path in the contract's storage trie, so the order the two are
/// applied in decides the nodes a proof of either reads.
#[test]
fn fixture_tip_rewrites_second_block_branch_nodes() {
    let second = &BLOCKS.second.sorted_trie_updates;
    let tip = &BLOCKS.tip.sorted_trie_updates;

    assert!(
        rewrites_stored_node_on_path(
            second.account_nodes_ref(),
            tip.account_nodes_ref(),
            keccak256(EOA)
        ),
        "account branch node on the EOA's path rewritten"
    );
    assert!(
        rewrites_stored_node_on_path(
            storage_nodes(second, CONTRACT),
            storage_nodes(tip, CONTRACT),
            hashed_slot(ZEROED_SLOT)
        ),
        "contract storage branch node on the zeroed slot's path rewritten"
    );
}

/// The V2 multiproof of the proofs state providers verifies against the tip's state root, shows
/// the same accounts, storage values and proof nodes as the V1 multiproof, and carries the masks
/// of the stored branch nodes.
#[rstest]
#[case::contract_slots(&[CONTRACT_TARGET], TargetShape::AccountAndStorage)]
#[case::destroyed_account(&[DESTROYED_TARGET], TargetShape::AccountAndStorage)]
#[case::updated_eoa(&[EOA_TARGET], TargetShape::AccountAndStorage)]
#[case::untouched_eoa(&[UNTOUCHED_TARGET], TargetShape::AccountAndStorage)]
#[case::absent_account(&[ABSENT_TARGET], TargetShape::AccountAndStorage)]
#[case::every_account(EVERY_TARGET, TargetShape::AccountAndStorage)]
#[case::contract_account_only(&[CONTRACT_TARGET], TargetShape::AccountOnly)]
#[case::contract_storage_only(&[CONTRACT_TARGET], TargetShape::StorageOnly)]
#[case::every_account_below_known_root(EVERY_TARGET, TargetShape::BelowKnownRoot)]
fn multiproof_v2_matches_multiproof(
    #[case] targets: &[(Address, &[u64])],
    #[case] shape: TargetShape,
    #[values(
        TipPlacement::Buffered,
        TipPlacement::BufferedWithParent,
        TipPlacement::Flushed,
        TipPlacement::TrieInputBlocks,
        TipPlacement::TrieInputOverBuffered,
        TipPlacement::TrieInputState
    )]
    placement: TipPlacement,
    #[values(StoreKind::InMemory, StoreKind::Mdbx)] store: StoreKind,
) {
    run_case(store, placement, targets, shape);
}
