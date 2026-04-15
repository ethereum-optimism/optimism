use std::{clone, collections::HashMap};

use alloy_eips::eip7928::{AccountChanges, BalanceChange, SlotChanges};
use alloy_primitives::{
    Address, B256, Bytes, KECCAK256_EMPTY, U256, keccak256, keccak256_uncached, map::U256Map,
};
use base_access_lists::FlashblockAccessList;
use reth_provider::{StateProvider, changeset_walker};
use reth_revm::{
    Database, DatabaseRef,
    cached::{self, CachedReads, CachedReadsDbMut},
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
}

impl<'a, DB: Database> FbalsDb<'a, DB> {
    pub(crate) fn new(
        cached_db: CachedReadsDbMut<'a, DB>,
        access_list: &[FlashblockAccessList],
        tx_index: u64,
    ) -> Self {
        let mut rv = Self { inner: cached_db };
        rv.inject_fbals(access_list, tx_index);
        rv
    }

    pub(crate) fn cached(&mut self) -> &mut CachedReads {
        self.inner.cached
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

impl<'a, DB: reth_revm::Database> FbalsDb<'a, DB> {
    fn inject_fbals(&mut self, fbals: &[FlashblockAccessList], tx_index: u64) {
        for fbal in fbals {
            for AccountChanges {
                address,
                storage_changes,
                storage_reads: _,
                balance_changes,
                nonce_changes,
                code_changes,
            } in fbal.account_changes.iter()
            {
                let mut balance = Default::default();
                while let Some(bc) = balance_changes.iter().next() &&
                    bc.block_access_index() < tx_index
                {
                    balance = bc.post_balance();
                }

                let mut nonce = Default::default();
                while let Some(nc) = nonce_changes.iter().next() &&
                    nc.block_access_index() < tx_index
                {
                    nonce = nc.new_nonce();
                }

                let mut code0 = None;
                while let Some(cc) = code_changes.iter().next() &&
                    cc.block_access_index() < tx_index
                {
                    code0 = Some(cc.new_code())
                }
                let code = code0.cloned().map(Bytecode::new_raw);
                let code_hash = code.clone().map_or(KECCAK256_EMPTY, |x| x.hash_slow());

                let info = AccountInfo { balance, nonce, code_hash, account_id: None, code };

                let mut storage = U256Map::<U256>::default();
                while let Some(SlotChanges { slot, changes }) = storage_changes.iter().next() {
                    while let Some(sc) = changes.iter().next() &&
                        sc.block_access_index < tx_index
                    {
                        storage.insert(*slot, sc.new_value);
                    }
                }

                self.cached().insert_account(*address, info, storage);
            }
        }
    }
}

pub fn split_fbal_into_transactions(fbal: &FlashblockAccessList) -> Vec<FlashblockAccessList> {
    let FlashblockAccessList { account_changes, min_tx_index, max_tx_index, fal_hash } = fbal;

    let capacity = max_tx_index - min_tx_index;
    let out = Vec::with_capacity(capacity.try_into().unwrap());
    todo!();
    out
}

/// Linear search for last code change before current tx.
pub fn search_fbals_code_changes(
    address: &Address,
    tx_index: u64,
    fbals: &[FlashblockAccessList],
) -> Option<Bytes> {
    let mut rv: Option<&Bytes> = None;

    for fbal in fbals {
        if fbal.max_tx_index <= tx_index {
            return rv.cloned();
        }

        for ac in fbal.account_changes.iter() {
            if *address == ac.address() {
                for nc in ac.code_changes() {
                    if tx_index < nc.block_access_index() {
                        return rv.cloned();
                    }
                    rv = Some(nc.new_code());
                }
            }
        }
    }
    None
}
