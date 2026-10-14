//! Blocks for the L1 actor's tests.

use alloy_primitives::B256;
use kona_protocol::BlockInfo;
use std::num::NonZeroUsize;

/// The most blocks the tests ask for at once.
pub(super) const MAX: NonZeroUsize = NonZeroUsize::new(16).unwrap();

/// Returns block `number` of the chain `fork`, whose parent is block `number - 1` of the chain
/// `parent_fork`.
pub(super) fn block(number: u64, fork: u8, parent_fork: u8) -> BlockInfo {
    let hash = |number: u64, fork: u8| {
        let mut hash = B256::repeat_byte(fork);
        hash[..8].copy_from_slice(&number.to_be_bytes());
        hash
    };
    let parent_hash = number.checked_sub(1).map_or(B256::ZERO, |parent| hash(parent, parent_fork));
    BlockInfo::new(hash(number, fork), number, parent_hash, number * 12)
}

/// Returns blocks `numbers` of the chain `fork`, all children of the same chain.
pub(super) fn chain(numbers: std::ops::RangeInclusive<u64>, fork: u8) -> Vec<BlockInfo> {
    numbers.map(|number| block(number, fork, fork)).collect()
}
