//! In-memory state database backed by revm's `CacheDB`.

use alloy_genesis::GenesisAccount;
use alloy_primitives::{Address, B256, U256, keccak256, map::B256Map};
use reth_primitives_traits::Account;
use reth_trie::{
    HashedPostState, HashedStorage, KeccakKeyHasher, MultiProofTargets, StateRoot,
    hashed_cursor::{HashedPostStateCursorFactory, noop::NoopHashedCursorFactory},
    proof::Proof,
    trie_cursor::noop::NoopTrieCursorFactory,
};
use revm::{
    database::{CacheDB, EmptyDB},
    state::{AccountInfo, Bytecode},
};
use std::collections::BTreeMap;

use super::roots::TrieNodeStore;

/// Snapshot of the state at a particular block, used for historical queries and proof generation.
#[derive(Debug, Clone)]
pub struct StateSnapshot {
    /// State root for this snapshot.
    pub state_root: B256,
    /// Accumulated hashed post-state for trie computation.
    pub hashed_state: HashedPostState,
    /// Contract bytecode keyed by code hash.
    pub code: BTreeMap<B256, Vec<u8>>,
    /// Trie node store accumulated during root computation.
    pub node_store: TrieNodeStore,
}

/// State of a single account.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccountState {
    /// Account nonce.
    pub nonce: u64,
    /// Account balance.
    pub balance: U256,
    /// Code hash (keccak256 of bytecode, or `KECCAK_EMPTY` for EOAs).
    pub code_hash: B256,
}

/// In-memory state database wrapping revm's `CacheDB`.
///
/// Tracks all state and can produce snapshots with computed state roots.
// CacheDB doesn't implement Debug
#[allow(missing_debug_implementations)]
pub struct TestStateDb {
    /// Inner revm cache database.
    pub db: CacheDB<EmptyDB>,
    /// Accumulated hashed state from genesis + all block bundle states.
    hashed_state: HashedPostState,
    /// Contract code keyed by code hash.
    code: BTreeMap<B256, Vec<u8>>,
}

impl Default for TestStateDb {
    fn default() -> Self {
        Self {
            db: CacheDB::new(EmptyDB::default()),
            hashed_state: HashedPostState::default(),
            code: BTreeMap::new(),
        }
    }
}

impl TestStateDb {
    /// Create a new empty state database.
    pub fn new() -> Self {
        Self::default()
    }

    /// Initialize state from genesis allocations.
    pub fn init_genesis(&mut self, allocs: &BTreeMap<Address, GenesisAccount>) {
        let mut hashed_accounts = B256Map::default();
        let mut hashed_storages = B256Map::default();

        for (address, account) in allocs {
            let code = account.code.as_ref().map(|c| c.to_vec()).unwrap_or_default();
            let code_hash =
                if code.is_empty() { alloy_primitives::KECCAK256_EMPTY } else { keccak256(&code) };

            let info = AccountInfo {
                balance: account.balance,
                nonce: account.nonce.unwrap_or(0),
                code_hash,
                account_id: None,
                code: if code.is_empty() {
                    None
                } else {
                    Some(Bytecode::new_raw(code.clone().into()))
                },
            };

            self.db.insert_account_info(*address, info.clone());

            // Insert storage into CacheDB
            if let Some(ref storage) = account.storage {
                for (slot, value) in storage {
                    self.db
                        .insert_account_storage(*address, (*slot).into(), (*value).into())
                        .expect("storage insert should not fail");
                }
            }

            // Build HashedPostState entry
            let hashed_addr = keccak256(address);
            let bytecode_hash =
                if code_hash == alloy_primitives::KECCAK256_EMPTY { None } else { Some(code_hash) };
            hashed_accounts.insert(
                hashed_addr,
                Some(Account { nonce: info.nonce, balance: info.balance, bytecode_hash }),
            );

            let storage_entries: Vec<_> = account
                .storage
                .as_ref()
                .unwrap_or(&BTreeMap::new())
                .iter()
                .filter(|(_, v)| !v.is_zero())
                .map(|(k, v)| (keccak256(B256::from(*k)), (*v).into()))
                .collect();
            if !storage_entries.is_empty() {
                hashed_storages
                    .insert(hashed_addr, HashedStorage::from_iter(false, storage_entries));
            }

            // Track code for debug_dbGet
            if !code.is_empty() {
                self.code.insert(code_hash, code);
            }
        }

        self.hashed_state =
            HashedPostState { accounts: hashed_accounts, storages: hashed_storages };
    }

    /// Apply execution results from a `BundleState` (produced by `OpBlockExecutor`) to the
    /// tracked state.
    pub fn apply_bundle_state(
        &mut self,
        bundle: &revm_database::BundleState,
        mut db: revm::database::CacheDB<revm::database::EmptyDB>,
    ) {
        // Accumulate hashed state from this block's bundle.
        // from_bundle_state handles all AccountStatus variants correctly:
        // - Destroyed -> None (account removed from trie)
        // - DestroyedChanged -> Some (account recreated)
        // - LoadedEmptyEIP161 -> None (EIP-161 empty account removed)
        // - Changed/InMemoryChange -> Some (account updated)
        let bundle_hashed = HashedPostState::from_bundle_state::<KeccakKeyHasher>(&bundle.state);
        self.hashed_state.extend(bundle_hashed);

        // Track new code deployments for debug_dbGet
        for (hash, bytecode) in &bundle.contracts {
            let bytes = bytecode.original_bytes();
            if !bytes.is_empty() {
                self.code.insert(*hash, bytes.to_vec());
            }
        }

        // Merge the post-execution DB with accounts from the original DB that weren't
        // touched during execution. This preserves funded test accounts on L1 that
        // weren't accessed by any transaction.
        for (addr, cache_account) in &self.db.cache.accounts {
            if !db.cache.accounts.contains_key(addr) {
                db.insert_account_info(*addr, cache_account.info.clone());
                for (slot, value) in &cache_account.storage {
                    let _ = db.insert_account_storage(*addr, *slot, *value);
                }
            }
        }
        // Also preserve contracts not in the new DB
        for (hash, bytecode) in &self.db.cache.contracts {
            if !db.cache.contracts.contains_key(hash) {
                db.cache.contracts.insert(*hash, bytecode.clone());
            }
        }
        self.db = db;
    }

    /// Capture a snapshot of the current state with computed state root.
    pub fn snapshot(&self) -> StateSnapshot {
        let sorted = self.hashed_state.clone().into_sorted();
        let prefix_sets = self.hashed_state.construct_prefix_sets();

        // Compute state root via reth's StateRoot
        let state_root = StateRoot::new(
            NoopTrieCursorFactory::default(),
            HashedPostStateCursorFactory::new(NoopHashedCursorFactory::default(), &sorted),
        )
        .with_prefix_sets(prefix_sets.clone().freeze())
        .root()
        .expect("state root computation should succeed");

        // Capture all trie nodes for debug_dbGet via multiproof over entire state.
        let all_targets: MultiProofTargets = self
            .hashed_state
            .accounts
            .iter()
            .filter(|(_, acct)| acct.is_some())
            .map(|(addr_hash, _)| {
                let storage_keys = self
                    .hashed_state
                    .storages
                    .get(addr_hash)
                    .map(|s| s.storage.keys().copied().collect())
                    .unwrap_or_default();
                (*addr_hash, storage_keys)
            })
            .collect();

        let multiproof = Proof::new(
            NoopTrieCursorFactory::default(),
            HashedPostStateCursorFactory::new(NoopHashedCursorFactory::default(), &sorted),
        )
        .with_prefix_sets_mut(prefix_sets)
        .multiproof(all_targets)
        .expect("multiproof should succeed");

        // Build hash->node index from proof nodes
        let mut node_store = TrieNodeStore::new();
        for (_, node_bytes) in multiproof.account_subtree.nodes_sorted() {
            node_store.insert(keccak256(&node_bytes), node_bytes);
        }
        for storage_mp in multiproof.storages.values() {
            for (_, node_bytes) in storage_mp.subtree.nodes_sorted() {
                node_store.insert(keccak256(&node_bytes), node_bytes);
            }
        }

        StateSnapshot {
            state_root,
            hashed_state: self.hashed_state.clone(),
            code: self.code.clone(),
            node_store,
        }
    }

    /// Get the account state.
    pub fn account(&self, address: &Address) -> Option<AccountState> {
        self.db.cache.accounts.get(address).map(|db_account| AccountState {
            nonce: db_account.info.nonce,
            balance: db_account.info.balance,
            code_hash: db_account.info.code_hash,
        })
    }

    /// Get the accumulated hashed state.
    pub const fn hashed_state(&self) -> &HashedPostState {
        &self.hashed_state
    }

    /// Fund an account with ETH (creates it if it doesn't exist).
    pub fn fund_account(&mut self, address: Address, balance: U256) {
        let code_hash = alloy_primitives::KECCAK256_EMPTY;
        let info = AccountInfo { balance, nonce: 0, code_hash, account_id: None, code: None };
        self.db.insert_account_info(address, info);

        // Also insert into hashed_state so the funded account appears in trie computation
        let hashed_addr = keccak256(address);
        self.hashed_state
            .accounts
            .insert(hashed_addr, Some(Account { nonce: 0, balance, bytecode_hash: None }));
    }
}

/// Rebuild a plain `CacheDB<EmptyDB>` from a `State<CacheDB<EmptyDB>>` after execution.
///
/// The `State` wrapper stores all committed changes in its internal cache. This function
/// extracts those changes into a fresh `CacheDB` that can replace the pre-execution one.
pub fn rebuild_cache_db(state: &revm::database::State<CacheDB<EmptyDB>>) -> CacheDB<EmptyDB> {
    let mut db = CacheDB::new(EmptyDB::default());

    // Copy all cached accounts from the State wrapper's cache
    for (addr, cache_account) in &state.cache.accounts {
        if let Some(ref account) = cache_account.account {
            db.insert_account_info(*addr, account.info.clone());
            for (slot, value) in &account.storage {
                let _ = db.insert_account_storage(*addr, *slot, *value);
            }
        }
    }

    // Also copy contracts
    for (hash, bytecode) in &state.cache.contracts {
        db.cache.contracts.insert(*hash, bytecode.clone());
    }

    db
}
