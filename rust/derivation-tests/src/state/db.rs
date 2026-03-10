//! In-memory state database backed by revm's `CacheDB`.

use alloy_genesis::GenesisAccount;
use alloy_primitives::{Address, B256, U256};
use revm::DatabaseCommit;
use revm::database::{CacheDB, EmptyDB};
use revm::state::{AccountInfo, Bytecode};
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
#[derive(Debug, Clone)]
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
                self.accounts.remove(address);
                self.storage.remove(address);
                continue;
            }

            let info = &account.info;
            let code_hash = info.code_hash;

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
}
