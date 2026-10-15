//! What a proofs store serves at one block, read through its public cursors and proof builder,
//! and the comparison of two such views that names the first differing table and key.

use alloy_primitives::{Address, B256, U256, keccak256};
use reth_optimism_trie::{
    OpProofsProviderRO, OpProofsStore, ProofWindowRange, proof::DatabaseProof,
};
use reth_primitives_traits::Account;
use reth_trie::{
    hashed_cursor::HashedCursor,
    proof::Proof,
    trie_cursor::{TrieCursor, TrieCursorIter},
};
use reth_trie_common::{AccountProof, BranchNodeCompact, Nibbles, TrieInput};
use std::{
    collections::{BTreeMap, BTreeSet},
    fmt,
};

/// The store's window: its earliest and latest persisted blocks.
pub(super) fn proof_window<S: OpProofsStore>(store: &S) -> eyre::Result<ProofWindowRange> {
    Ok(store.provider_ro()?.get_proof_window()?)
}

/// An address whose proof, with these slots, every store must serve identically.
pub(super) struct ProofTarget {
    pub(super) address: Address,
    pub(super) slots: Vec<B256>,
}

impl ProofTarget {
    /// Targets for `addresses`, none of which has storage.
    pub(super) fn accounts(addresses: impl IntoIterator<Item = Address>) -> Vec<Self> {
        addresses.into_iter().map(|address| Self { address, slots: Vec::new() }).collect()
    }
}

/// One account's storage: its slots and its storage trie nodes.
pub(super) struct AccountStorage {
    pub(super) slots: BTreeMap<B256, U256>,
    pub(super) trie: BTreeMap<Nibbles, BranchNodeCompact>,
}

/// Everything a store serves at one block.
pub(super) struct BlockView {
    pub(super) account_trie: BTreeMap<Nibbles, BranchNodeCompact>,
    pub(super) accounts: BTreeMap<B256, Account>,
    pub(super) proofs: BTreeMap<Address, AccountProof>,
    /// The storage of every account the store holds and of every proof target, by hashed address.
    pub(super) storages: BTreeMap<B256, AccountStorage>,
}

impl BlockView {
    pub(super) fn read<S: OpProofsStore>(
        store: &S,
        number: u64,
        targets: &[ProofTarget],
    ) -> eyre::Result<Self> {
        Self::read_from(&store.provider_ro()?, number, targets)
    }

    /// Reads the view through `provider`, so that several views and the window come from one
    /// read transaction while saves go on.
    pub(super) fn read_from<P: OpProofsProviderRO + Clone>(
        provider: &P,
        number: u64,
        targets: &[ProofTarget],
    ) -> eyre::Result<Self> {
        let accounts = collect_hashed(provider.account_hashed_cursor(number)?)?;
        let hashed_addresses: BTreeSet<B256> = accounts
            .keys()
            .copied()
            .chain(targets.iter().map(|target| keccak256(target.address)))
            .collect();
        let storages = hashed_addresses
            .into_iter()
            .map(|hashed_address| {
                let slots =
                    collect_hashed(provider.storage_hashed_cursor(hashed_address, number)?)?;
                let trie = collect_trie(provider.storage_trie_cursor(hashed_address, number)?)?;
                Ok((hashed_address, AccountStorage { slots, trie }))
            })
            .collect::<eyre::Result<_>>()?;
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
            account_trie: collect_trie(provider.account_trie_cursor(number)?)?,
            accounts,
            proofs,
            storages,
        })
    }

    /// Every proof must hash to `state_root`, the header's, which ties the store to reth.
    pub(super) fn verify_proofs(&self, number: u64, state_root: B256) -> eyre::Result<()> {
        for (address, proof) in &self.proofs {
            proof.verify(state_root).map_err(|err| {
                eyre::eyre!(
                    "the proof of {address} at block {number} does not verify against the \
                     header state root: {err}"
                )
            })?;
        }
        Ok(())
    }
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
    Ok(TrieCursorIter::new(&mut cursor).collect::<Result<_, _>>()?)
}

/// Fails on the first table and key at which the `store` store's `actual` view differs from the
/// indexed store's `expected` view.
pub(super) fn assert_same_view(
    number: u64,
    store: &str,
    expected: &BlockView,
    actual: &BlockView,
) -> eyre::Result<()> {
    assert_same_table(number, store, "HashedAccounts", &expected.accounts, &actual.accounts)?;
    assert_same_table(number, store, "AccountsTrie", &expected.account_trie, &actual.account_trie)?;
    // Equal account tables give both views the same storage keys.
    for (hashed_address, storage) in &expected.storages {
        let actual_storage = &actual.storages[hashed_address];
        let slots_table = format!("HashedStorages[{hashed_address}]");
        assert_same_table(number, store, &slots_table, &storage.slots, &actual_storage.slots)?;
        let trie_table = format!("StoragesTrie[{hashed_address}]");
        assert_same_table(number, store, &trie_table, &storage.trie, &actual_storage.trie)?;
    }
    assert_same_table(number, store, "AccountProof", &expected.proofs, &actual.proofs)
}

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
