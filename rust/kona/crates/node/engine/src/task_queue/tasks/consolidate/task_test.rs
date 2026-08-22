//! Tests for the L1 origin a [`ConsolidateTask`] pairs with the local-safe head it writes.
//!
//! [`ConsolidateTask`]: super::ConsolidateTask

use crate::{
    LocalSafeOrigin, task_queue::tasks::consolidate::ConsolidateInput,
    test_utils::TestAttributesBuilder,
};
use kona_protocol::{BlockInfo, L2BlockInfo};

fn l1(number: u64) -> BlockInfo {
    BlockInfo { number, ..Default::default() }
}

/// The ordinary derivation path: the attributes name their L1 origin and it travels with them, so
/// the head this input writes is paired with the L1 block it was actually derived from.
#[test]
fn derived_attributes_pair_with_their_own_l1_origin() {
    let input =
        ConsolidateInput::from(TestAttributesBuilder::new().with_derived_from(l1(7)).build());

    assert_eq!(input.local_safe_origin(), LocalSafeOrigin::DerivedFrom(l1(7)));
}

/// Derivation delegation injects a bare block with no attributes, so there is no L1 origin to pair
/// with it. It has to be recorded as unpaired rather than inheriting whatever origin the previous
/// local-safe head carried.
#[test]
fn a_delegated_block_is_unpaired() {
    let input = ConsolidateInput::from(L2BlockInfo::default());

    assert_eq!(input.local_safe_origin(), LocalSafeOrigin::Unpaired);
}

/// `derived_from` is itself optional on the attributes, so even the derivation path can arrive
/// without an origin. That is the absent case too, not a defaulted L1 block.
#[test]
fn attributes_without_an_origin_are_unpaired() {
    let input = ConsolidateInput::from(TestAttributesBuilder::new().build());

    assert_eq!(input.local_safe_origin(), LocalSafeOrigin::Unpaired);
    assert_ne!(input.local_safe_origin(), LocalSafeOrigin::DerivedFrom(BlockInfo::default()));
}
