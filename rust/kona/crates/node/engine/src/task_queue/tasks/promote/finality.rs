//! The finality of a node that follows a cross-chain verifier.

use crate::{FinalizeBlockId, VerifierHead};
use kona_genesis::RollupConfig;

/// The finality of a node that follows a cross-chain verifier: the verifier's finalized head,
/// bounded by the node's own L1 finality, as op-node's `FinalizedHead` resolves it under a super
/// authority.
///
/// - Before the verifier's activation no block has a cross-chain dependency, so the blocks the node
///   derived from finalized L1 data are final up to the last block before activation, whatever the
///   verifier answers, and also before it answers.
/// - [`VerifierHead::PreActivation`]: the node's L1 finality.
/// - [`VerifierHead::Anchor`]: the block at the anchor, bounded by the node's L1 finality.
/// - [`VerifierHead::Verified`]: the verified block by hash if the node's L1 finality covers it, or
///   else the node's L1-finalized block: every block below a verified one is verified.
///
/// Nothing is final before the node's L1 finality covers a block.
#[derive(Debug, Clone, Default)]
pub struct VerifierFinality {
    /// The highest L2 block the node derived from finalized L1 data.
    l1_finalized: Option<u64>,
    /// The verifier's latest finalized head.
    verifier: Option<VerifierHead>,
}

impl VerifierFinality {
    /// Records the highest L2 block the node derived from finalized L1 data. A lower one, as
    /// after a reset, leaves it.
    pub fn on_l1_finalized(&mut self, number: u64) {
        self.l1_finalized = Some(self.l1_finalized.map_or(number, |known| known.max(number)));
    }

    /// Records the verifier's latest finalized head.
    pub const fn on_verifier_finalized(&mut self, head: VerifierHead) {
        self.verifier = Some(head);
    }

    /// The block to finalize, if any.
    pub fn target(&self, cfg: &RollupConfig) -> Option<FinalizeBlockId> {
        let l1_finalized = self.l1_finalized?;
        let pre_activation = l1_finalized.min(last_block_before_activation(cfg));
        let verified = self.verifier.map(|head| match head {
            VerifierHead::Anchor { timestamp } => FinalizeBlockId::ByNumber(
                cfg.block_number_from_timestamp(timestamp).min(l1_finalized),
            ),
            VerifierHead::Verified { block, .. } if block.number <= l1_finalized => {
                FinalizeBlockId::ByHash(block)
            }
            VerifierHead::PreActivation | VerifierHead::Verified { .. } => {
                FinalizeBlockId::ByNumber(l1_finalized)
            }
        });
        Some(match verified {
            Some(target) if target.number() >= pre_activation => target,
            _ => FinalizeBlockId::ByNumber(pre_activation),
        })
    }
}

/// The verifier's first L2 timestamp for a chain: its interop activation, or its genesis if
/// interop activates at or before it. A chain without interop never activates a verifier.
pub fn verifier_activation_timestamp(cfg: &RollupConfig) -> u64 {
    cfg.hardforks.lagoon_time.map_or(u64::MAX, |time| time.max(cfg.genesis.l2_time))
}

/// The number of the last block before the verifier's activation, or genesis if it activates at
/// genesis.
fn last_block_before_activation(cfg: &RollupConfig) -> u64 {
    let activation = verifier_activation_timestamp(cfg);
    if activation <= cfg.genesis.l2_time {
        return cfg.genesis.l2.number;
    }
    cfg.block_number_from_timestamp(activation - 1)
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use kona_genesis::ChainGenesis;

    /// Interop activates at block 10's timestamp, so block 9 is the last before it.
    const ACTIVATION_BLOCK: u64 = 10;

    fn cfg(lagoon_block: Option<u64>) -> RollupConfig {
        let mut cfg = RollupConfig {
            block_time: 2,
            genesis: ChainGenesis { l2_time: 100, ..Default::default() },
            ..Default::default()
        };
        cfg.hardforks.lagoon_time = lagoon_block.map(|number| 100 + 2 * number);
        cfg
    }

    fn block(number: u64) -> BlockNumHash {
        BlockNumHash { number, hash: B256::with_last_byte(number as u8) }
    }

    fn finality(l1_finalized: Option<u64>, verifier: Option<VerifierHead>) -> VerifierFinality {
        let mut finality = VerifierFinality::default();
        if let Some(number) = l1_finalized {
            finality.on_l1_finalized(number);
        }
        if let Some(head) = verifier {
            finality.on_verifier_finalized(head);
        }
        finality
    }

    const ANCHOR: VerifierHead = VerifierHead::Anchor { timestamp: 100 + 2 * ACTIVATION_BLOCK - 1 };

    #[rstest::rstest]
    // Nothing is final before the node's own L1 finality covers it.
    #[case::no_l1_finality(None, Some(VerifierHead::Verified { block: block(12), timestamp: 124, l1: BlockNumHash::default() }), None)]
    // Before activation L1 finality alone finalizes, also without an answer from the verifier,
    // but never past the last block before activation.
    #[case::pre_activation_without_verifier(Some(7), None, Some(FinalizeBlockId::ByNumber(7)))]
    #[case::post_activation_without_verifier(Some(30), None, Some(FinalizeBlockId::ByNumber(9)))]
    // The anchor is bounded by L1 finality.
    #[case::anchor_beyond_l1_finality(Some(7), Some(ANCHOR), Some(FinalizeBlockId::ByNumber(7)))]
    #[case::anchor_within_l1_finality(Some(30), Some(ANCHOR), Some(FinalizeBlockId::ByNumber(9)))]
    // A verified head is final by hash once L1 finality covers it, and L1 finality bounds it.
    #[case::verified_within_l1_finality(
        Some(30),
        Some(VerifierHead::Verified { block: block(12), timestamp: 124, l1: BlockNumHash::default() }),
        Some(FinalizeBlockId::ByHash(block(12)))
    )]
    #[case::verified_beyond_l1_finality(
        Some(11),
        Some(VerifierHead::Verified { block: block(12), timestamp: 124, l1: BlockNumHash::default() }),
        Some(FinalizeBlockId::ByNumber(11))
    )]
    #[case::pre_activation_answer(
        Some(30),
        Some(VerifierHead::PreActivation),
        Some(FinalizeBlockId::ByNumber(30))
    )]
    fn target_is_the_verifier_head_bounded_by_l1_finality(
        #[case] l1_finalized: Option<u64>,
        #[case] verifier: Option<VerifierHead>,
        #[case] expected: Option<FinalizeBlockId>,
    ) {
        let finality = finality(l1_finalized, verifier);

        assert_eq!(finality.target(&cfg(Some(ACTIVATION_BLOCK))), expected);
    }

    /// With interop at genesis only genesis is before activation; without interop every block
    /// is.
    #[rstest::rstest]
    #[case::at_genesis(Some(0), FinalizeBlockId::ByNumber(0))]
    #[case::never(None, FinalizeBlockId::ByNumber(30))]
    fn pre_activation_finality_depends_on_the_activation(
        #[case] lagoon_block: Option<u64>,
        #[case] expected: FinalizeBlockId,
    ) {
        assert_eq!(finality(Some(30), None).target(&cfg(lagoon_block)), Some(expected));
    }

    /// L1 finality reported lower, as after a reset, does not lower the bound.
    #[test]
    fn l1_finality_never_moves_back() {
        let mut finality = finality(Some(30), Some(VerifierHead::PreActivation));
        finality.on_l1_finalized(20);

        assert_eq!(
            finality.target(&cfg(Some(ACTIVATION_BLOCK))),
            Some(FinalizeBlockId::ByNumber(30))
        );
    }
}
