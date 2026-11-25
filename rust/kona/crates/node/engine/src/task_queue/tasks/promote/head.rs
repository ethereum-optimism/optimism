//! The head a cross-chain verifier reports for a chain.

use alloy_eips::BlockNumHash;
use serde::{Deserialize, Serialize};

/// The safe or finalized head a cross-chain verifier reports for one chain.
///
/// On the wire it is `{ block: { hash, number }, timestamp, source }`, `source` being
/// `pre-activation`, `anchor` or `verified`, with the block and timestamp zero where the variant
/// carries none.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(into = "WireVerifierHead", from = "WireVerifierHead")]
pub enum VerifierHead {
    /// The verifier is not active for the chain's local-safe head, so the label follows the
    /// node's own local view.
    PreActivation,
    /// The verifier is active but has verified nothing for the chain yet. The label is the
    /// canonical block at `timestamp`, the last timestamp before activation.
    Anchor {
        /// The L2 timestamp whose block anchors the label.
        timestamp: u64,
    },
    /// The verifier's head for the chain.
    Verified {
        /// The verified block.
        block: BlockNumHash,
        /// The L2 timestamp the verifier verified up to.
        timestamp: u64,
    },
}

/// The source of a [`WireVerifierHead`].
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
enum Source {
    PreActivation,
    Anchor,
    Verified,
}

/// The flat wire form of a [`VerifierHead`].
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
struct WireVerifierHead {
    block: BlockNumHash,
    timestamp: u64,
    source: Source,
}

impl From<VerifierHead> for WireVerifierHead {
    fn from(head: VerifierHead) -> Self {
        match head {
            VerifierHead::PreActivation => {
                Self { block: BlockNumHash::default(), timestamp: 0, source: Source::PreActivation }
            }
            VerifierHead::Anchor { timestamp } => {
                Self { block: BlockNumHash::default(), timestamp, source: Source::Anchor }
            }
            VerifierHead::Verified { block, timestamp } => {
                Self { block, timestamp, source: Source::Verified }
            }
        }
    }
}

impl From<WireVerifierHead> for VerifierHead {
    fn from(wire: WireVerifierHead) -> Self {
        match wire.source {
            Source::PreActivation => Self::PreActivation,
            Source::Anchor => Self::Anchor { timestamp: wire.timestamp },
            Source::Verified => Self::Verified { block: wire.block, timestamp: wire.timestamp },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;

    #[test]
    fn verified_head_has_the_flat_wire_form() {
        let head = VerifierHead::Verified {
            block: BlockNumHash { number: 7, hash: B256::repeat_byte(1) },
            timestamp: 14,
        };

        let json = serde_json::to_value(head).unwrap();

        assert_eq!(
            json,
            serde_json::json!({
                "block": { "number": 7, "hash": B256::repeat_byte(1) },
                "timestamp": 14,
                "source": "verified",
            })
        );
        assert_eq!(serde_json::from_value::<VerifierHead>(json).unwrap(), head);
    }

    #[rstest::rstest]
    #[case::pre_activation(VerifierHead::PreActivation, "pre-activation")]
    #[case::anchor(VerifierHead::Anchor { timestamp: 3 }, "anchor")]
    fn heads_without_a_block_round_trip(#[case] head: VerifierHead, #[case] source: &str) {
        let json = serde_json::to_value(head).unwrap();

        assert_eq!(json["source"], source);
        assert_eq!(serde_json::from_value::<VerifierHead>(json).unwrap(), head);
    }
}
