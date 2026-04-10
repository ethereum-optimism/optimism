use std::collections::HashMap;

use alloy_eips::eip7928::{AccountChanges, BalanceChange};
use alloy_primitives::{Address, B256, U256, map::U256Map};
use base_access_lists::FlashblockAccessList;
use reth_provider::StateProvider;
use reth_revm::{
    Database, DatabaseRef,
    cached::{self, CachedReadsDbMut},
    database::{EvmStateProvider, StateProviderDatabase},
    primitives::{StorageKey, StorageValue},
    state::{AccountInfo, Bytecode},
};

// OPT: compute hashmaps before call, take ownership and only store writes
/// Compute the prefix sum of all Account changes for FlashBlock 1..n
pub fn prefix_sum(fbals: &[FlashblockAccessList]) -> Vec<FlashblockAccessList> {
    debug_assert!(1 <= fbals.len());

    let identity = FlashblockAccessList {
        account_changes: vec![],
        min_tx_index: 0,
        max_tx_index: 0,
        fal_hash: Default::default(),
    };

    fbals
        .iter()
        .scan(identity, |left: &mut FlashblockAccessList, right: &FlashblockAccessList| {
            Some(merge_fbals(&*left, right))
        })
        .collect()
}

/// Merge two Flashblock's access lists assuming they are in consecutive and in chronological order
pub fn merge_fbals(
    left: &FlashblockAccessList,
    right: &FlashblockAccessList,
) -> FlashblockAccessList {
    debug_assert_eq!(left.max_tx_index, right.min_tx_index);

    let account_changes = merge_account_changes(&left.account_changes, &right.account_changes);

    FlashblockAccessList::build(account_changes, left.min_tx_index, right.max_tx_index)
}

/// Merge access lists so the latest write to each address is maintained.
pub fn merge_account_changes(
    left: &[AccountChanges],
    right: &[AccountChanges],
) -> Vec<AccountChanges> {
    let mut address_map = HashMap::<Address, AccountChanges>::new();
    for ac in left {
        address_map.insert(ac.address, ac.clone());
    }

    for ac in right {
        address_map
            .entry(ac.address)
            .and_modify(|left| merge_address_changes(left, ac))
            .or_insert(ac.clone());
    }

    address_map.into_values().collect()
}

pub fn merge_address_changes(left: &mut AccountChanges, right: &AccountChanges) {
    debug_assert_eq!(left.address(), right.address());
    let address = left.address();

    let storage_changes =
        left.storage_changes().into_iter().chain(right.storage_changes()).cloned().collect();

    // we could ignore these.
    let storage_reads =
        left.storage_reads().into_iter().chain(right.storage_reads()).cloned().collect();

    let balance_changes =
        left.balance_changes().into_iter().chain(right.balance_changes()).cloned().collect();

    let nonce_changes =
        left.nonce_changes().into_iter().chain(right.nonce_changes()).cloned().collect();

    let code_changes =
        left.code_changes().into_iter().chain(right.code_changes()).cloned().collect();

    *left = AccountChanges {
        address,
        storage_changes,
        storage_reads,
        balance_changes,
        nonce_changes,
        code_changes,
    };
}

#[derive(Debug)]
pub enum FBalsValidationResult {
    AllValidated,
    OneOrMoreFailed,
}

#[derive(Debug)]
pub struct FbalsDb<'a, DB> {
    inner: CachedReadsDbMut<'a, DB>,
    access_lists: FlashblockAccessList,
}

impl<'a, DB: Database> FbalsDb<'a, DB> {
    pub(crate) fn new(
        cached_db: CachedReadsDbMut<'a, DB>,
        access_list: FlashblockAccessList,
    ) -> Self {
        Self { inner: cached_db, access_lists: access_list }
    }
}

impl<'a, DB: DatabaseRef> Database for FbalsDb<'a, DB> {
    #[doc = " The database error type."]
    type Error = <DB as DatabaseRef>::Error;

    #[doc = " Gets basic account information."]
    fn basic(&mut self, address: Address) -> Result<Option<AccountInfo>, Self::Error> {
        self.inner.basic(address)
    }

    #[doc = " Gets account code by its hash."]
    fn code_by_hash(&mut self, code_hash: B256) -> Result<Bytecode, Self::Error> {
        self.inner.code_by_hash(code_hash)
    }

    #[doc = " Gets storage value of address at index."]
    fn storage(
        &mut self,
        address: Address,
        index: StorageKey,
    ) -> Result<StorageValue, Self::Error> {
        self.inner.storage(address, index)
    }

    #[doc = " Gets block hash by block number."]
    fn block_hash(&mut self, number: u64) -> Result<B256, Self::Error> {
        self.inner.block_hash(number)
    }
}

impl<'a, DB> FbalsDb<'a, DB> {
    pub fn inject_fbals(&self, fbals: &FlashblockAccessList) {
        for AccountChanges {
            address,
            storage_changes,
            storage_reads,
            balance_changes,
            nonce_changes,
            code_changes,
        } in fbals.account_changes.iter()
        {
            let info = AccountInfo {
                balance: todo!(),
                nonce: todo!(),
                code_hash: todo!(),
                account_id: todo!(),
                code: todo!(),
            };
            let storage: U256Map<U256> = todo!();
            self.inner.cached.insert_account(*address, info, storage);
        }
    }
}

pub fn split_fbal_into_transactions(fbal: &FlashblockAccessList) -> Vec<FlashblockAccessList> {
    let FlashblockAccessList { account_changes, min_tx_index, max_tx_index, fal_hash } = fbal;

    let capacity = max_tx_index - min_tx_index;
    let out = Vec::with_capacity(capacity);
    out
}
