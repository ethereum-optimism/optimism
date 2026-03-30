//! In-memory state database backed by revm's `CacheDB`.

use alloy_genesis::GenesisAccount;
use alloy_primitives::{Address, B256, U256};
use revm::{
    DatabaseCommit,
    database::{CacheDB, EmptyDB},
    state::{AccountInfo, Bytecode},
};
use std::collections::BTreeMap;

use super::roots::{TrieNodeStore, compute_state_root, compute_storage_root};

/// Snapshot of the state at a particular block, used for historical queries and proof generation.
#[derive(Debug, Clone)]
pub struct StateSnapshot {
    /// All accounts with their balances, nonces, and code hashes.
    pub accounts: BTreeMap<Address, AccountState>,
    /// All storage slots per account.
    pub storage: BTreeMap<Address, BTreeMap<U256, U256>>,
    /// Contract bytecode keyed by code hash.
    pub code: BTreeMap<B256, Vec<u8>>,
    /// State root for this snapshot.
    pub state_root: B256,
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
    /// Accumulated accounts for state root computation (address → account state).
    accounts: BTreeMap<Address, AccountState>,
    /// Storage per account.
    storage: BTreeMap<Address, BTreeMap<U256, U256>>,
    /// Contract code keyed by code hash.
    code: BTreeMap<B256, Vec<u8>>,
}

impl Default for TestStateDb {
    fn default() -> Self {
        Self {
            db: CacheDB::new(EmptyDB::default()),
            accounts: BTreeMap::new(),
            storage: BTreeMap::new(),
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
        use alloy_primitives::keccak256;

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

            self.db.insert_account_info(*address, info);

            // Insert storage
            if let Some(ref storage) = account.storage {
                for (slot, value) in storage {
                    self.db
                        .insert_account_storage(*address, (*slot).into(), (*value).into())
                        .expect("storage insert should not fail");
                    self.storage
                        .entry(*address)
                        .or_default()
                        .insert((*slot).into(), (*value).into());
                }
            }

            self.accounts.insert(
                *address,
                AccountState {
                    nonce: account.nonce.unwrap_or(0),
                    balance: account.balance,
                    code_hash,
                },
            );

            if !code.is_empty() {
                self.code.insert(code_hash, code);
            }
        }
    }

    /// Apply execution results from revm's `BundleState` to the tracked state.
    pub fn apply_evm_result(&mut self, result: &revm::state::EvmState) {
        for (address, account) in result {
            if account.is_selfdestructed() {
                // Always clear storage for self-destructed accounts.
                self.storage.remove(address);

                let info = &account.info;
                // If the account was recreated after self-destruct (e.g. CREATE2 reuse
                // in the same tx under EIP-6780), preserve the new account state.
                if info.nonce == 0 &&
                    info.balance.is_zero() &&
                    info.code_hash == alloy_primitives::KECCAK256_EMPTY
                {
                    self.accounts.remove(address);
                } else {
                    self.accounts.insert(
                        *address,
                        AccountState {
                            nonce: info.nonce,
                            balance: info.balance,
                            code_hash: info.code_hash,
                        },
                    );

                    if let Some(ref code) = info.code {
                        let bytecode = code.original_bytes();
                        if !bytecode.is_empty() {
                            self.code.insert(info.code_hash, bytecode.to_vec());
                        }
                    }
                }
                continue;
            }

            let info = &account.info;
            let code_hash = info.code_hash;

            // EIP-161: Remove touched-but-empty accounts from state.
            // An account is empty if nonce == 0, balance == 0, and has no code.
            if account.is_touched() &&
                info.nonce == 0 &&
                info.balance.is_zero() &&
                code_hash == alloy_primitives::KECCAK256_EMPTY
            {
                self.accounts.remove(address);
                self.storage.remove(address);
                continue;
            }

            self.accounts.insert(
                *address,
                AccountState { nonce: info.nonce, balance: info.balance, code_hash },
            );

            // Store code if present
            if let Some(ref code) = info.code {
                let bytecode = code.original_bytes();
                if !bytecode.is_empty() {
                    self.code.insert(code_hash, bytecode.to_vec());
                }
            }

            // Apply storage changes
            for (slot, slot_value) in &account.storage {
                let value = slot_value.present_value;
                let storage = self.storage.entry(*address).or_default();
                if value.is_zero() {
                    storage.remove(slot);
                } else {
                    storage.insert(*slot, value);
                }
            }
        }

        // Also commit to the CacheDB
        self.db.commit(result.clone());
    }

    /// Apply execution results from a `BundleState` (produced by `OpBlockExecutor`) to the
    /// tracked state.
    pub fn apply_bundle_state(
        &mut self,
        bundle: &revm_database::BundleState,
        mut db: revm::database::CacheDB<revm::database::EmptyDB>,
    ) {
        use revm_database::AccountStatus;

        for (address, account) in &bundle.state {
            let is_destroyed = matches!(
                account.status,
                AccountStatus::Destroyed |
                    AccountStatus::DestroyedChanged |
                    AccountStatus::DestroyedAgain
            );

            if is_destroyed {
                self.storage.remove(address);

                match &account.info {
                    Some(info)
                        if info.nonce > 0 ||
                            !info.balance.is_zero() ||
                            info.code_hash != alloy_primitives::KECCAK256_EMPTY =>
                    {
                        // Account was recreated after destruction
                        self.accounts.insert(
                            *address,
                            AccountState {
                                nonce: info.nonce,
                                balance: info.balance,
                                code_hash: info.code_hash,
                            },
                        );
                        if let Some(ref code) = info.code {
                            let bytecode = code.original_bytes();
                            if !bytecode.is_empty() {
                                self.code.insert(info.code_hash, bytecode.to_vec());
                            }
                        }
                        // Apply new storage for recreated account
                        for (slot, slot_value) in &account.storage {
                            let value = slot_value.present_value;
                            let storage = self.storage.entry(*address).or_default();
                            if value.is_zero() {
                                storage.remove(slot);
                            } else {
                                storage.insert(*slot, value);
                            }
                        }
                    }
                    _ => {
                        self.accounts.remove(address);
                    }
                }
                continue;
            }

            let Some(info) = &account.info else {
                // Account was loaded but doesn't exist; skip.
                continue;
            };

            // EIP-161: Remove touched-but-empty accounts from state.
            let is_empty = info.nonce == 0 &&
                info.balance.is_zero() &&
                info.code_hash == alloy_primitives::KECCAK256_EMPTY;
            let was_changed = matches!(
                account.status,
                AccountStatus::Changed |
                    AccountStatus::InMemoryChange |
                    AccountStatus::LoadedEmptyEIP161
            );

            if is_empty && was_changed {
                self.accounts.remove(address);
                self.storage.remove(address);
                continue;
            }

            // Only update accounts that were actually changed
            if was_changed {
                self.accounts.insert(
                    *address,
                    AccountState {
                        nonce: info.nonce,
                        balance: info.balance,
                        code_hash: info.code_hash,
                    },
                );

                if let Some(ref code) = info.code {
                    let bytecode = code.original_bytes();
                    if !bytecode.is_empty() {
                        self.code.insert(info.code_hash, bytecode.to_vec());
                    }
                }
            }

            // Apply storage changes (only slots that actually changed)
            for (slot, slot_value) in &account.storage {
                if slot_value.present_value != slot_value.previous_or_original_value {
                    let value = slot_value.present_value;
                    let storage = self.storage.entry(*address).or_default();
                    if value.is_zero() {
                        storage.remove(slot);
                    } else {
                        storage.insert(*slot, value);
                    }
                }
            }
        }

        // Also store any new contract code from the bundle
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
        let mut node_store = TrieNodeStore::new();

        // Compute storage roots for all accounts and collect them
        let mut storage_roots: BTreeMap<Address, B256> = BTreeMap::new();
        for (address, storage) in &self.storage {
            if !storage.is_empty() {
                let root = compute_storage_root(storage, &mut node_store);
                storage_roots.insert(*address, root);
            }
        }

        let state_root =
            compute_state_root(&self.accounts, &storage_roots, &self.code, &mut node_store);

        StateSnapshot {
            accounts: self.accounts.clone(),
            storage: self.storage.clone(),
            code: self.code.clone(),
            state_root,
            node_store,
        }
    }

    /// Get the storage for a specific account.
    pub fn account_storage(&self, address: &Address) -> Option<&BTreeMap<U256, U256>> {
        self.storage.get(address)
    }

    /// Get the account state.
    pub fn account(&self, address: &Address) -> Option<&AccountState> {
        self.accounts.get(address)
    }

    /// Fund an account with ETH (creates it if it doesn't exist).
    pub fn fund_account(&mut self, address: Address, balance: U256) {
        let code_hash = alloy_primitives::KECCAK256_EMPTY;
        let info = AccountInfo { balance, nonce: 0, code_hash, account_id: None, code: None };
        self.db.insert_account_info(address, info);
        self.accounts.insert(address, AccountState { nonce: 0, balance, code_hash });
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
