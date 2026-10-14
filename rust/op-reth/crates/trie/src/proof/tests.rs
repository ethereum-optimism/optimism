//! Multiproofs over the proofs storage.
//!
//! The V2 multiproof of each proofs state provider must verify against the state root and show the
//! same account and storage values as its V1 multiproof, whether the proven block is flushed to
//! the store, buffered in the engine, or passed as the caller's trie input.
//!
//! The fixture is two blocks on an empty store. Block 1 creates [`CONTRACT`] with [`LIVE_SLOT`]
//! and [`ZEROED_SLOT`], [`DESTROYED`] with [`LIVE_SLOT`], the [`EOA`] and [`UNTOUCHED`] accounts,
//! and enough filler accounts and slots for the tries to hold branch nodes on the paths the tip
//! changes. The tip, block 2, destroys [`DESTROYED`], zeroes the contract's [`ZEROED_SLOT`] and
//! updates [`EOA`]. [`ABSENT`] and [`ABSENT_SLOT`] never exist.

use super::DatabaseStateRoot;
use crate::{
    BlockStateDiff, InMemoryProofsStorage, OpProofsProviderRw, OpProofsStore,
    engine::buffer::state::TrieBufferState,
    provider::OpProofsStateProviderRef,
    test_utils::{
        create_storage,
        destroyed_accounts::{
            INITIAL_ANCHOR, hashed_slot, hashed_storage, plain_slot, seed_initial_storage,
        },
    },
};
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_primitives::{Address, B256, U256, keccak256, map::B256Set};
use reth_primitives_traits::Account;
use reth_provider::{StateProofProvider, noop::NoopProvider};
use reth_trie::{
    HashedPostState, MultiProofTargets, MultiProofTargetsV2, StateRoot, TrieInput,
    root::{state_root_unsorted, storage_root_unsorted},
};
use rstest::rstest;
use std::ops::Range;

const CONTRACT: Address = Address::repeat_byte(0x01);
const DESTROYED: Address = Address::repeat_byte(0x02);
const EOA: Address = Address::repeat_byte(0x03);
const UNTOUCHED: Address = Address::repeat_byte(0x04);
const ABSENT: Address = Address::repeat_byte(0x05);

/// Plain slot of [`CONTRACT`] holding [`LIVE_VALUE`] at the tip.
const LIVE_SLOT: u64 = 1;
/// Plain slot of [`CONTRACT`] that the tip sets to zero.
const ZEROED_SLOT: u64 = 2;
/// Plain slot no account ever writes.
const ABSENT_SLOT: u64 = 3;
const LIVE_VALUE: u64 = 11;

/// Plain slots of [`CONTRACT`] and [`DESTROYED`] that the tip leaves alone.
const FILLER_SLOTS: Range<u64> = 4..260;

const CONTRACT_TARGET: (Address, &[u64]) = (CONTRACT, &[LIVE_SLOT, ZEROED_SLOT, ABSENT_SLOT]);
const DESTROYED_TARGET: (Address, &[u64]) = (DESTROYED, &[LIVE_SLOT]);
const EOA_TARGET: (Address, &[u64]) = (EOA, &[ABSENT_SLOT]);
const UNTOUCHED_TARGET: (Address, &[u64]) = (UNTOUCHED, &[ABSENT_SLOT]);
const ABSENT_TARGET: (Address, &[u64]) = (ABSENT, &[LIVE_SLOT]);
const EVERY_TARGET: &[(Address, &[u64])] =
    &[CONTRACT_TARGET, DESTROYED_TARGET, EOA_TARGET, UNTOUCHED_TARGET, ABSENT_TARGET];

const BLOCK_ONE: NumHash = NumHash::new(1, B256::repeat_byte(0xb1));
const TIP: NumHash = NumHash::new(2, B256::repeat_byte(0xb2));

/// Proofs storage backend a case runs against.
#[derive(Clone, Copy, Debug)]
enum Backend {
    InMemory,
    Mdbx,
}

/// Where the tip block lives when the proofs are requested.
#[derive(Clone, Copy, Debug)]
enum TipPlacement {
    /// Flushed to the store.
    Flushed,
    /// Held in the engine's buffer, which the overlay state provider lays over the store.
    Buffered,
    /// Passed by the caller as the trie input over the store at the tip's parent.
    TrieInput,
}

fn contract_account() -> Account {
    Account { nonce: 1, balance: U256::from(100), bytecode_hash: Some(B256::repeat_byte(0xcc)) }
}

fn destroyed_before_tip() -> Account {
    Account { nonce: 1, balance: U256::from(200), bytecode_hash: Some(B256::repeat_byte(0xdd)) }
}

fn eoa_account(nonce: u64, balance: u64) -> Account {
    Account { nonce, balance: U256::from(balance), bytecode_hash: None }
}

fn eoa_at_tip() -> Account {
    eoa_account(1, 250)
}

fn untouched_account() -> Account {
    eoa_account(5, 400)
}

/// Accounts the tip leaves alone, one per last byte of the address, so that the account trie
/// holds branch nodes on the paths the tip changes.
fn filler_accounts() -> impl Iterator<Item = (Address, Account)> {
    (0..=u8::MAX).map(|byte| (Address::with_last_byte(byte), eoa_account(0, 1)))
}

/// Plain storage the tip leaves alone, so that a storage trie holding it has branch nodes on the
/// paths the tip changes.
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

/// Block 1's state: every account the fixture holds, with the storage the tip changes.
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

/// The tip's state, shaped as the proofs state providers' `hashed_post_state` shapes it: every
/// slot of the destroyed account is zeroed explicitly.
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

/// Every account alive at the tip.
fn accounts_at_tip() -> impl Iterator<Item = (Address, Account)> {
    filler_accounts().chain([
        (CONTRACT, contract_account()),
        (EOA, eoa_at_tip()),
        (UNTOUCHED, untouched_account()),
    ])
}

fn account_at_tip(address: Address) -> Option<Account> {
    accounts_at_tip().find(|&(candidate, _)| candidate == address).map(|(_, account)| account)
}

/// Plain storage of `address` at the tip.
fn storage_at_tip(address: Address) -> Vec<(u64, u64)> {
    if address == CONTRACT { contract_storage_at_tip() } else { Vec::new() }
}

fn value_at_tip(address: Address, slot: u64) -> U256 {
    let value = storage_at_tip(address)
        .into_iter()
        .find(|&(candidate, _)| candidate == slot)
        .map_or(0, |(_, value)| value);
    U256::from(value)
}

/// The tip's state root computed from its plain state, independently of the store.
fn tip_state_root() -> B256 {
    state_root_unsorted(accounts_at_tip().map(|(address, account)| {
        let storage = storage_at_tip(address)
            .into_iter()
            .map(|(slot, value)| (hashed_slot(slot), U256::from(value)));
        (keccak256(address), account.into_trie_account(storage_root_unsorted(storage)))
    }))
}

/// A block computed on top of its parent in the store.
struct ComputedBlock {
    diff: BlockStateDiff,
    state_root: B256,
}

impl ComputedBlock {
    /// Computes `post_state` on top of block `parent_number` of `store`.
    fn compute<S: OpProofsStore>(
        store: &S,
        parent_number: u64,
        post_state: HashedPostState,
    ) -> Self {
        let provider = store.provider_ro().expect("provider ro");
        let (state_root, trie_updates) =
            StateRoot::overlay_root_with_updates(provider, parent_number, post_state.clone())
                .expect("state root");
        let diff = BlockStateDiff {
            sorted_trie_updates: trie_updates.into_sorted(),
            sorted_post_state: post_state.into_sorted(),
        };
        Self { diff, state_root }
    }

    fn trie_input(&self) -> TrieInput {
        TrieInput::from_blocks_sorted([(
            &self.diff.sorted_post_state,
            &self.diff.sorted_trie_updates,
        )])
    }
}

fn store_block<S: OpProofsStore>(store: &S, block: BlockWithParent, diff: BlockStateDiff) {
    let provider = store.provider_rw().expect("provider rw");
    provider.store_trie_updates(block, diff).expect("store trie updates");
    provider.commit().expect("commit");
}

/// Seeds `store` with block 1 and computes the tip on top of it.
fn seed_and_compute_tip<S: OpProofsStore>(store: &S) -> ComputedBlock {
    seed_initial_storage(store, &[]);
    let first_block = ComputedBlock::compute(store, INITIAL_ANCHOR.number, first_block_state());
    store_block(store, BlockWithParent::new(INITIAL_ANCHOR.hash, BLOCK_ONE), first_block.diff);
    ComputedBlock::compute(store, BLOCK_ONE.number, tip_state())
}

fn state_provider<S: OpProofsStore>(
    store: &S,
    block_number: u64,
) -> OpProofsStateProviderRef<'static, S::ProviderRO<'_>> {
    let provider = store.provider_ro().expect("provider ro");
    OpProofsStateProviderRef::new(Box::new(NoopProvider::default()), provider, block_number)
}

/// Requests the V1 and V2 multiproofs of `targets` from `provider` over `input`, and checks that
/// each verifies against `state_root` and shows the tip's account and storage values.
fn assert_multiproofs_agree<P: StateProofProvider>(
    provider: &P,
    input: TrieInput,
    state_root: B256,
    targets: &[(Address, &[u64])],
) {
    let v1_targets: MultiProofTargets = targets
        .iter()
        .map(|&(address, slots)| {
            (keccak256(address), slots.iter().copied().map(hashed_slot).collect::<B256Set>())
        })
        .collect();
    let v2_targets = MultiProofTargetsV2 {
        account_targets: v1_targets.keys().copied().map(Into::into).collect(),
        storage_targets: v1_targets
            .iter()
            .map(|(&hashed_address, slots)| {
                (hashed_address, slots.iter().copied().map(Into::into).collect())
            })
            .collect(),
    };

    let v1 = provider.multiproof(input.clone(), v1_targets).expect("multiproof");
    let v2 = provider.multiproof_v2(input, v2_targets).expect("multiproof v2");

    for &(address, slots) in targets {
        let plain_slots: Vec<B256> = slots.iter().copied().map(plain_slot).collect();
        let v1_proof = v1.account_proof(address, &plain_slots).expect("v1 account proof");
        let v2_proof = v2.account_proof(address, &plain_slots).expect("v2 account proof");
        let expected_values: Vec<U256> =
            slots.iter().map(|&slot| value_at_tip(address, slot)).collect();

        for (version, proof) in [("v1", v1_proof), ("v2", v2_proof)] {
            assert_eq!(proof.verify(state_root), Ok(()), "{version} proof of {address}");
            assert_eq!(proof.info, account_at_tip(address), "{version} account of {address}");
            let values: Vec<U256> = proof.storage_proofs.iter().map(|slot| slot.value).collect();
            assert_eq!(values, expected_values, "{version} storage of {address}");
        }
    }
}

/// Seeds `store`, places the tip as `placement` says, and checks the multiproofs of `targets` at
/// the tip.
fn run_case<S: OpProofsStore>(store: &S, placement: TipPlacement, targets: &[(Address, &[u64])]) {
    let tip = seed_and_compute_tip(store);
    assert_eq!(
        tip.state_root,
        tip_state_root(),
        "fixture: the store's tip root differs from the plain tip state's"
    );

    match placement {
        TipPlacement::Flushed => {
            store_block(store, BlockWithParent::new(BLOCK_ONE.hash, TIP), tip.diff);
            let provider = state_provider(store, TIP.number);
            assert_multiproofs_agree(&provider, TrieInput::default(), tip.state_root, targets);
        }
        TipPlacement::Buffered => {
            let buffer = TrieBufferState::new();
            buffer.insert(BlockWithParent::new(BLOCK_ONE.hash, TIP), tip.diff);
            let provider = buffer.state_provider(TIP.hash, state_provider(store, BLOCK_ONE.number));
            assert_multiproofs_agree(&provider, TrieInput::default(), tip.state_root, targets);
        }
        TipPlacement::TrieInput => {
            let provider = state_provider(store, BLOCK_ONE.number);
            assert_multiproofs_agree(&provider, tip.trie_input(), tip.state_root, targets);
        }
    }
}

/// The tip changes paths under stored branch nodes of the account trie and of both contracts'
/// storage tries, so the proofs over it read stored and overlaid branch nodes, not leaves alone.
#[rstest]
fn fixture_tip_rewrites_stored_branch_nodes(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
) {
    fn check<S: OpProofsStore>(store: &S) {
        let updates = seed_and_compute_tip(store).diff.sorted_trie_updates;
        let storage_nodes = |address: Address| {
            updates
                .storage_tries_ref()
                .get(&keccak256(address))
                .map(|trie| trie.storage_nodes_ref())
                .unwrap_or_default()
        };

        assert!(!updates.account_nodes_ref().is_empty(), "account branch nodes rewritten");
        assert!(
            storage_nodes(CONTRACT).iter().any(|(_, node)| node.is_some()),
            "contract storage branch nodes rewritten"
        );
        assert!(
            storage_nodes(DESTROYED).iter().any(|(_, node)| node.is_none()),
            "destroyed account's storage branch nodes removed"
        );
    }
    match backend {
        Backend::InMemory => check(&InMemoryProofsStorage::new()),
        Backend::Mdbx => check(&*create_storage()),
    }
}

/// The V2 multiproof of the proofs state providers verifies against the tip's state root and
/// shows the same account and storage values as the V1 multiproof.
#[rstest]
#[case::contract_slots(&[CONTRACT_TARGET])]
#[case::destroyed_account(&[DESTROYED_TARGET])]
#[case::updated_eoa(&[EOA_TARGET])]
#[case::untouched_eoa(&[UNTOUCHED_TARGET])]
#[case::absent_account(&[ABSENT_TARGET])]
#[case::every_account(EVERY_TARGET)]
fn multiproof_v2_matches_multiproof(
    #[case] targets: &[(Address, &[u64])],
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(TipPlacement::Flushed, TipPlacement::Buffered, TipPlacement::TrieInput)]
    placement: TipPlacement,
) {
    match backend {
        Backend::InMemory => run_case(&InMemoryProofsStorage::new(), placement, targets),
        Backend::Mdbx => run_case(&*create_storage(), placement, targets),
    }
}
