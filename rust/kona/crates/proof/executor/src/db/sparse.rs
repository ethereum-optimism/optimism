//! An experimental alternative to [`TrieDB`](crate::TrieDB) backed by reth's sparse trie
//! ([`reth_trie_sparse`]) instead of `kona-mpt`.
//!
//! The sparse trie does not fetch missing nodes itself. A read reports the first blinded node on
//! its path, and a batch of leaf updates reports the keys it could not apply. This module fetches
//! those nodes from the [`TrieDBProvider`] by hash, reveals them, and retries.
//!
//! The sparse trie stores an extension node merged into its child branch, so it cannot reveal an
//! extension without the branch below it. Where `kona-mpt` stops at an extension (a key that
//! diverges inside the extension's path, or a branch collapse onto an extension sibling), this
//! backend fetches one more node. [`SparseTrieDB::stats`] counts those fetches.

use crate::{
    TrieDBProvider,
    errors::{TrieDBError, TrieDBResult},
};
use alloc::{format, string::ToString, vec, vec::Vec};
use alloy_consensus::{EMPTY_ROOT_HASH, Header, Sealed};
use alloy_primitives::{Address, B256, U256, keccak256, map::B256Map};
use alloy_rlp::Decodable;
use alloy_trie::{TrieAccount, nodes::TrieNode};
use core::fmt::Display;
use kona_mpt::TrieHinter;
use reth_trie_common::{Nibbles, ProofTrieNodeV2};
use reth_trie_sparse::{
    LeafLookupError, LeafUpdate, RevealableSparseTrie, SparseStateTrie, SparseTrie, TrieNodeEpoch,
};
use revm::{
    Database,
    database::BundleState,
    primitives::{BLOCK_HASH_HISTORY, HashMap},
    state::{AccountInfo, Bytecode},
};

/// The maximum number of reveal rounds for one lookup or one batch of updates. A trie has at most
/// 64 levels, so a well-formed trie never needs more.
const MAX_REVEAL_ROUNDS: usize = 128;

/// Counters for the trie nodes this backend fetched.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub struct SparseTrieStats {
    /// Trie node preimages fetched from the provider.
    pub node_fetches: u64,
    /// Extension child branches fetched only because the sparse trie cannot reveal an extension
    /// without its child. `kona-mpt` does not fetch these.
    pub extension_child_fetches: u64,
    /// Reveal rounds needed to apply leaf updates.
    pub update_rounds: u64,
    /// Reads answered "absent" from an extension node whose path the key leaves. The extension
    /// stays blinded, because revealing it would need its child branch.
    pub absent_at_extension: u64,
}

/// A [`revm`] database over reth's [`SparseStateTrie`], fetching trie nodes on demand.
///
/// It exposes the same interface as [`TrieDB`](crate::TrieDB) to the block builder.
#[derive(Debug)]
pub struct SparseTrieDB<F, H>
where
    F: TrieDBProvider,
    H: TrieHinter,
{
    /// The account trie and the storage tries.
    trie: SparseStateTrie,
    /// The storage root of each loaded account, used to reveal its storage trie on first access.
    storage_roots: HashMap<Address, B256>,
    /// The epoch of the next root computation. Incremented once per block.
    epoch: u64,
    /// The parent block header of the current block.
    parent_block_header: Sealed<Header>,
    /// Fetch counters.
    stats: SparseTrieStats,
    /// The [`TrieDBProvider`]
    pub fetcher: F,
    /// The [`TrieHinter`]
    pub hinter: H,
}

impl<F, H> SparseTrieDB<F, H>
where
    F: TrieDBProvider,
    H: TrieHinter,
{
    /// Creates a new [`SparseTrieDB`] on top of the parent block's state root.
    pub fn new(parent_block_header: Sealed<Header>, fetcher: F, hinter: H) -> Self {
        Self {
            trie: SparseStateTrie::new(),
            storage_roots: Default::default(),
            epoch: 0,
            parent_block_header,
            stats: SparseTrieStats::default(),
            fetcher,
            hinter,
        }
    }

    /// Returns the fetch counters.
    pub const fn stats(&self) -> SparseTrieStats {
        self.stats
    }

    /// Returns a reference to the current parent block header of the trie DB.
    pub const fn parent_block_header(&self) -> &Sealed<Header> {
        &self.parent_block_header
    }

    /// Sets the parent block header of the trie DB. Should be called after a block has been
    /// executed and the Header has been created.
    pub fn set_parent_block_header(&mut self, parent_block_header: Sealed<Header>) {
        self.parent_block_header = parent_block_header;
    }

    /// Applies a [`BundleState`] changeset to the tries and returns the new state root.
    // Experiment: kept out of line so callgrind attributes trie cost to this function.
    #[inline(never)]
    pub fn state_root(&mut self, bundle: &BundleState) -> TrieDBResult<B256> {
        self.epoch += 1;
        let epoch = TrieNodeEpoch::new(self.epoch);

        let mut sorted_state =
            bundle.state().iter().map(|(k, v)| (keccak256(*k), k, v)).collect::<Vec<_>>();
        sorted_state.sort_by_key(|(hashed_address, _, _)| *hashed_address);

        let mut account_updates = B256Map::default();
        for (hashed_address, address, bundle_account) in sorted_state {
            if bundle_account.status.is_not_modified() {
                continue;
            }

            // A destroyed account loses its storage. If it was re-created, its new state is
            // inserted below on top of an empty storage trie.
            if bundle_account.was_destroyed() {
                self.trie
                    .insert_storage_trie(hashed_address, RevealableSparseTrie::revealed_empty());
                self.storage_roots.insert(*address, EMPTY_ROOT_HASH);
                if bundle_account.account_info().is_none() {
                    account_updates.insert(hashed_address, LeafUpdate::Changed(Vec::new()));
                    continue;
                }
            }

            let account_info =
                bundle_account.account_info().ok_or(TrieDBError::MissingAccountInfo)?;
            let parent_storage_root =
                self.storage_roots.get(address).copied().unwrap_or(EMPTY_ROOT_HASH);

            let storage_updates = bundle_account
                .storage
                .iter()
                .filter(|(_, slot)| slot.is_changed())
                .map(|(key, slot)| {
                    let value = if slot.present_value.is_zero() {
                        Vec::new()
                    } else {
                        alloy_rlp::encode(slot.present_value)
                    };
                    (keccak256(key.to_be_bytes::<32>()), LeafUpdate::Changed(value))
                })
                .collect::<B256Map<_>>();

            let storage_root = if storage_updates.is_empty() {
                self.trie
                    .storage_tries_mut()
                    .get_mut(&hashed_address)
                    .and_then(|trie| trie.root(epoch))
                    .unwrap_or(parent_storage_root)
            } else {
                let trie = self.trie.get_or_create_storage_trie_mut(hashed_address);
                apply_updates(
                    trie,
                    parent_storage_root,
                    storage_updates,
                    &self.fetcher,
                    &mut self.stats,
                )?;
                trie.root(epoch).ok_or_else(|| sparse_err("storage trie is blind after update"))?
            };
            self.storage_roots.insert(*address, storage_root);

            let trie_account = TrieAccount {
                nonce: account_info.nonce,
                balance: account_info.balance,
                storage_root,
                code_hash: account_info.code_hash,
            };
            account_updates
                .insert(hashed_address, LeafUpdate::Changed(alloy_rlp::encode(trie_account)));
        }

        apply_updates(
            self.trie.trie_mut(),
            self.parent_block_header.state_root,
            account_updates,
            &self.fetcher,
            &mut self.stats,
        )?;
        let root = self.trie.root(epoch).map_err(sparse_err)?;
        debug!(target: "client_executor", "Recomputed state root (sparse trie): {root}");
        Ok(root)
    }

    /// Fetches the [`TrieAccount`] of an account, revealing its path in the account trie.
    pub fn get_trie_account(&mut self, address: &Address) -> TrieDBResult<Option<TrieAccount>> {
        self.hinter
            .hint_account_proof(*address, self.parent_block_header.hash())
            .map_err(|e| TrieDBError::Provider(e.to_string()))?;

        let hashed_address = keccak256(address);
        reveal_until_resolved(
            self.trie.trie_mut(),
            self.parent_block_header.state_root,
            &Nibbles::unpack(hashed_address),
            &self.fetcher,
            &mut self.stats,
        )?;

        self.trie
            .get_account_value(&hashed_address)
            .map(|rlp| TrieAccount::decode(&mut rlp.as_slice()).map_err(sparse_err))
            .transpose()
    }

    /// Returns the current storage root of an account.
    pub fn account_storage_root(&mut self, address: &Address) -> TrieDBResult<B256> {
        let epoch = TrieNodeEpoch::new(self.epoch);
        if let Some(root) =
            self.trie.storage_tries_mut().get_mut(&keccak256(address)).and_then(|t| t.root(epoch))
        {
            return Ok(root);
        }
        Ok(self.get_trie_account(address)?.ok_or(TrieDBError::MissingAccountInfo)?.storage_root)
    }
}

impl<F, H> Database for SparseTrieDB<F, H>
where
    F: TrieDBProvider,
    H: TrieHinter,
{
    type Error = TrieDBError;

    // Experiment: kept out of line so callgrind attributes trie cost to this function.
    #[inline(never)]
    fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
        let Some(trie_account) = self.get_trie_account(&address)? else {
            return Ok(None);
        };
        self.storage_roots.insert(address, trie_account.storage_root);

        Ok(Some(AccountInfo {
            balance: trie_account.balance,
            nonce: trie_account.nonce,
            code_hash: trie_account.code_hash,
            code: None,
            account_id: None,
        }))
    }

    fn code_by_hash(&mut self, code_hash: B256) -> Result<Bytecode, Self::Error> {
        self.fetcher
            .bytecode_by_hash(code_hash)
            .map(Bytecode::new_raw)
            .map_err(|e| TrieDBError::Provider(e.to_string()))
    }

    // Experiment: kept out of line so callgrind attributes trie cost to this function.
    #[inline(never)]
    fn storage(&mut self, address: Address, index: U256) -> Result<U256, Self::Error> {
        self.hinter
            .hint_storage_proof(address, index, self.parent_block_header.hash())
            .map_err(|e| TrieDBError::Provider(e.to_string()))?;

        // An account that was never loaded does not exist, so its storage is empty.
        let Some(&storage_root) = self.storage_roots.get(&address) else {
            return Ok(U256::ZERO);
        };

        let hashed_address = keccak256(address);
        let hashed_slot = keccak256(index.to_be_bytes::<32>());
        reveal_until_resolved(
            self.trie.get_or_create_storage_trie_mut(hashed_address),
            storage_root,
            &Nibbles::unpack(hashed_slot),
            &self.fetcher,
            &mut self.stats,
        )?;

        self.trie
            .get_storage_slot_value(&hashed_address, &hashed_slot)
            .map_or(Ok(U256::ZERO), |rlp| U256::decode(&mut rlp.as_slice()).map_err(sparse_err))
    }

    fn block_hash(&mut self, block_number: u64) -> Result<B256, Self::Error> {
        let mut header = self.parent_block_header.inner().clone();
        if block_number > header.number ||
            header.number.saturating_sub(block_number) > BLOCK_HASH_HISTORY
        {
            return Ok(B256::default());
        }
        while header.number > block_number {
            header = self
                .fetcher
                .header_by_hash(header.parent_hash)
                .map_err(|e| TrieDBError::Provider(e.to_string()))?;
        }
        Ok(header.hash_slow())
    }
}

fn sparse_err(e: impl Display) -> TrieDBError {
    TrieDBError::Provider(format!("sparse trie: {e}"))
}

/// Applies `updates` to `trie`, revealing the nodes the trie asks for until every update applies.
///
/// `root` is the trie's root hash, used if the trie is still blind.
fn apply_updates<F: TrieDBProvider>(
    trie: &mut RevealableSparseTrie,
    root: B256,
    mut updates: B256Map<LeafUpdate>,
    fetcher: &F,
    stats: &mut SparseTrieStats,
) -> TrieDBResult<()> {
    let update_keys = updates.keys().copied().collect::<Vec<_>>();
    for _ in 0..MAX_REVEAL_ROUNDS {
        let mut targets = Vec::new();
        trie.update_leaves(&mut updates, |key, _| targets.push(key)).map_err(sparse_err)?;
        if updates.is_empty() {
            return Ok(());
        }
        stats.update_rounds += 1;

        targets.sort_unstable();
        targets.dedup();
        for key in targets {
            // A target that is one of the updated keys needs its whole path revealed. Any other
            // target is the blinded sibling a removal collapses onto, which needs only that node.
            let walk = if update_keys.contains(&key) { Walk::Update } else { Walk::Node };
            reveal(trie, root, &Nibbles::unpack(key), walk, fetcher, stats)?;
        }
    }
    Err(sparse_err("leaf updates did not converge"))
}

/// Reveals nodes on the path to `key` until the trie can tell whether the leaf exists.
fn reveal_until_resolved<F: TrieDBProvider>(
    trie: &mut RevealableSparseTrie,
    root: B256,
    key: &Nibbles,
    fetcher: &F,
    stats: &mut SparseTrieStats,
) -> TrieDBResult<()> {
    for _ in 0..MAX_REVEAL_ROUNDS {
        match reveal(trie, root, key, Walk::Read, fetcher, stats)? {
            Revealed::Nothing | Revealed::AbsentAtExtension => return Ok(()),
            Revealed::Nodes => {}
        }
    }
    Err(sparse_err("lookup did not converge"))
}

/// How far [`reveal`] walks from the first blinded node.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Walk {
    /// Down to the leaf at `key` for a read. An extension the key leaves proves the key absent.
    Read,
    /// Down to the leaf at `key` for an update. An extension the key leaves is revealed with its
    /// child branch, because the update splits it.
    Update,
    /// Only the blinded node itself (plus an extension's child branch).
    Node,
}

/// What [`reveal`] did.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Revealed {
    /// Nothing on the path to `key` was blinded.
    Nothing,
    /// Nodes were revealed.
    Nodes,
    /// The key leaves the path of a blinded extension, so it is absent. Nodes above the extension
    /// may have been revealed.
    AbsentAtExtension,
}

/// Reveals the first blinded node on the path to `key`, and the nodes below it as `walk` says.
fn reveal<F: TrieDBProvider>(
    trie: &mut RevealableSparseTrie,
    root: B256,
    key: &Nibbles,
    walk: Walk,
    fetcher: &F,
    stats: &mut SparseTrieStats,
) -> TrieDBResult<Revealed> {
    let (hash, path) = match trie.as_revealed_ref() {
        None if root == EMPTY_ROOT_HASH => {
            let mut nodes = vec![ProofTrieNodeV2::empty()];
            trie.reveal_v2_proof_nodes(&mut nodes, false).map_err(sparse_err)?;
            return Ok(Revealed::Nodes);
        }
        None => (root, Nibbles::default()),
        Some(revealed) => match revealed.find_leaf(key, None) {
            Ok(_) => return Ok(Revealed::Nothing),
            Err(LeafLookupError::BlindedNode { path, hash }) => (hash, path),
            Err(e) => return Err(sparse_err(format!("{e:?}"))),
        },
    };

    let (mut nodes, absent) = fetch_proof_nodes(fetcher, hash, path, key, walk, stats)?;
    if !nodes.is_empty() {
        trie.reveal_v2_proof_nodes(&mut nodes, false).map_err(sparse_err)?;
    }
    if absent {
        stats.absent_at_extension += 1;
        return Ok(Revealed::AbsentAtExtension);
    }
    Ok(Revealed::Nodes)
}

/// Fetches the node `hash` at `path` and the nodes below it towards `key`, as `walk` says. An
/// extension is fetched with its child branch, because the sparse trie stores the two as one node,
/// except on a read whose key leaves the extension's path: then the extension is left out and the
/// second return value is `true`, meaning the key is absent.
fn fetch_proof_nodes<F: TrieDBProvider>(
    fetcher: &F,
    mut hash: B256,
    mut path: Nibbles,
    key: &Nibbles,
    walk: Walk,
    stats: &mut SparseTrieStats,
) -> TrieDBResult<(Vec<ProofTrieNodeV2>, bool)> {
    let whole_path = walk != Walk::Node;
    let mut absent = false;
    let mut nodes = Vec::new();
    loop {
        let node = fetch_node(fetcher, hash, stats)?;
        // `next` continues the walk; `merge_child` is an extension child fetched only to merge.
        let (next, merge_child) = match &node {
            TrieNode::Branch(branch) if whole_path && path.len() < key.len() => {
                let nibble = key.get_unchecked(path.len());
                let next = branch.as_ref().children().find(|(i, _)| *i == nibble).and_then(
                    |(_, child)| {
                        child.and_then(|c| c.as_hash()).map(|h| {
                            let mut child_path = path;
                            child_path.push_unchecked(nibble);
                            (h, child_path)
                        })
                    },
                );
                (next, None)
            }
            TrieNode::Extension(ext) => {
                let child = ext
                    .child
                    .as_hash()
                    .ok_or_else(|| sparse_err("extension with an inline child"))?;
                let child_path = path.join(&ext.key);
                let on_key_path = key.len() >= child_path.len() &&
                    key.slice(path.len()..child_path.len()) == ext.key;
                if whole_path && on_key_path {
                    (Some((child, child_path)), None)
                } else if walk == Walk::Read {
                    absent = true;
                    break;
                } else {
                    (None, Some((child, child_path)))
                }
            }
            _ => (None, None),
        };
        nodes.push((path, node, None));

        if let Some((child, child_path)) = merge_child {
            stats.extension_child_fetches += 1;
            let branch = fetch_node(fetcher, child, stats).map_err(|e| {
                sparse_err(format!("extension child at {child_path:?} ({walk:?}): {e}"))
            })?;
            nodes.push((child_path, branch, None));
            break;
        }
        let Some((next_hash, next_path)) = next else { break };
        hash = next_hash;
        path = next_path;
    }

    // `from_sorted_trie_nodes` expects children before parents.
    nodes.sort_by(|(a, _, _), (b, _, _)| b.cmp(a));
    Ok((ProofTrieNodeV2::from_sorted_trie_nodes(nodes), absent))
}

fn fetch_node<F: TrieDBProvider>(
    fetcher: &F,
    hash: B256,
    stats: &mut SparseTrieStats,
) -> TrieDBResult<TrieNode> {
    stats.node_fetches += 1;
    let rlp = fetcher
        .trie_node_rlp_by_hash(hash)
        .map_err(|e| TrieDBError::Provider(format!("trie node {hash}: {e}")))?;
    TrieNode::decode(&mut rlp.as_ref()).map_err(sparse_err)
}
