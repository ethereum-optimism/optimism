use std::collections::HashMap;

use alloy_eips::eip7928::{AccountChanges, BalanceChange};
use alloy_primitives::Address;
use base_access_lists::FlashblockAccessList;

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
