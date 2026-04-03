use alloy_eips::eip7928::account_changes;
use base_access_lists::FlashblockAccessList;
use reth_revm::bytecode::bitvec::vec;

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
    // WIP: This needs to handle non-disjoint account changes.
    let account_changes = [left.account_changes.clone(), right.account_changes.clone()].concat();

    FlashblockAccessList::build(account_changes, left.min_tx_index, right.max_tx_index)
}
