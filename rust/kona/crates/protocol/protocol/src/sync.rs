//! Common sync types

use crate::{BlockInfo, L2BlockInfo};

/// The [`SyncStatus`][ss] of an Optimism Rollup Node.
///
/// The sync status is a snapshot of the current state of the node's sync process.
/// Values may not be derived yet and are zeroed out if they are not yet derived.
///
/// [ss]: https://github.com/ethereum-optimism/optimism/blob/develop/op-service/eth/sync_status.go#L5
#[derive(Clone, Debug, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "snake_case"))]
pub struct SyncStatus {
    /// The current L1 block.
    ///
    /// This is the L1 block that the derivation process is last idled at.
    /// This may not be fully derived into L2 data yet.
    /// The safe L2 blocks were produced/included fully from the L1 chain up to _but excluding_
    /// this L1 block. If the node is synced, this matches the `head_l1`, minus the verifier
    /// confirmation distance.
    pub current_l1: BlockInfo,
    /// The current L1 finalized block.
    ///
    /// This is a legacy sync-status attribute. This is deprecated.
    /// A previous version of the L1 finalization-signal was updated only after the block was
    /// retrieved by number. This attribute just matches `finalized_l1` now.
    pub current_l1_finalized: BlockInfo,
    /// The L1 head block ref.
    ///
    /// The head is not guaranteed to build on the other L1 sync status fields,
    /// as the node may be in progress of resetting to adapt to a L1 reorg.
    pub head_l1: BlockInfo,
    /// The L1 safe head block ref.
    pub safe_l1: BlockInfo,
    /// The finalized L1 block ref.
    pub finalized_l1: BlockInfo,
    /// The unsafe L2 block ref.
    ///
    /// This is the absolute tip of the L2 chain, pointing to block data that has not been
    /// submitted to L1 yet. The sequencer is building this, and verifiers may also be ahead of
    /// the safe L2 block if they sync blocks via p2p or other offchain sources.
    pub unsafe_l2: L2BlockInfo,
    /// The safe L2 block ref.
    ///
    /// This points to the L2 block that was derived from the L1 chain.
    /// This point may still reorg if the L1 chain reorgs.
    /// This is considered to be cross-safe post-interop, see `local_safe_l2` to ignore cross-L2
    /// guarantees.
    pub safe_l2: L2BlockInfo,
    /// The finalized L2 block ref.
    ///
    /// This points to the L2 block that was derived fully from finalized L1 information, thus
    /// irreversible.
    pub finalized_l2: L2BlockInfo,
    /// Local safe L2 block ref.
    ///
    /// This is an L2 block derived from L1, not yet verified to have valid cross-L2 dependencies.
    pub local_safe_l2: L2BlockInfo,
}

#[cfg(all(test, feature = "serde"))]
mod tests {
    use super::*;
    use alloy_primitives::b256;

    /// A CL that still reports `cross_unsafe_l2` must stay parseable. `DerivationDelegateClient`
    /// reads `optimism_syncStatus` from an arbitrary external node, and op-nodes predating the
    /// field's removal — plus third-party implementations — still send it. This only works
    /// because [`SyncStatus`] does not `deny_unknown_fields`.
    #[test]
    fn test_sync_status_ignores_cross_unsafe_from_an_external_node() {
        let l1_block = r#"{
            "hash": "0x1111111111111111111111111111111111111111111111111111111111111111",
            "number": 18123456,
            "parentHash": "0x2222222222222222222222222222222222222222222222222222222222222222",
            "timestamp": 1699123456
        }"#;
        let l2_block = |number: u64| {
            format!(
                r#"{{
                    "hash": "0x3333333333333333333333333333333333333333333333333333333333333333",
                    "number": {number},
                    "parentHash": "0x4444444444444444444444444444444444444444444444444444444444444444",
                    "timestamp": 1699123456,
                    "l1Origin": {{
                        "hash": "0x5555555555555555555555555555555555555555555555555555555555555555",
                        "number": 18123456
                    }},
                    "sequenceNumber": 42
                }}"#
            )
        };

        let payload = format!(
            r#"{{
                "current_l1": {l1_block},
                "current_l1_finalized": {l1_block},
                "head_l1": {l1_block},
                "safe_l1": {l1_block},
                "finalized_l1": {l1_block},
                "unsafe_l2": {unsafe_l2},
                "safe_l2": {safe_l2},
                "finalized_l2": {finalized_l2},
                "cross_unsafe_l2": {cross_unsafe_l2},
                "local_safe_l2": {local_safe_l2}
            }}"#,
            l1_block = l1_block,
            unsafe_l2 = l2_block(12350),
            safe_l2 = l2_block(12345),
            finalized_l2 = l2_block(12340),
            cross_unsafe_l2 = l2_block(12348),
            local_safe_l2 = l2_block(12346),
        );

        let status: SyncStatus = serde_json::from_str(&payload).unwrap();

        // The extra field is dropped, and the heads we do read are unaffected by it.
        assert_eq!(status.unsafe_l2.block_info.number, 12350);
        assert_eq!(status.safe_l2.block_info.number, 12345);
        assert_eq!(status.local_safe_l2.block_info.number, 12346);
        assert_eq!(
            status.unsafe_l2.block_info.hash,
            b256!("3333333333333333333333333333333333333333333333333333333333333333")
        );
        assert_eq!(status.unsafe_l2.l1_origin.number, 18123456);
        assert_eq!(status.unsafe_l2.seq_num, 42);
    }
}
