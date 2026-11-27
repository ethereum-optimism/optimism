//! What a cross-chain verifier reports for a chain.

use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use serde::{Deserialize, Serialize};

/// The safe or finalized head a cross-chain verifier reports for one chain.
///
/// On the wire it is `{ block: { hash, number }, timestamp, source, l1 }`, `source` being
/// `pre-activation`, `anchor` or `verified`, with the block, timestamp and L1 block zero where the
/// variant carries none.
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
        /// The latest L1 block the verification depended on: it holds on an L1 chain that
        /// includes this block.
        l1: BlockNumHash,
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
    #[serde(default)]
    l1: BlockNumHash,
}

impl From<VerifierHead> for WireVerifierHead {
    fn from(head: VerifierHead) -> Self {
        let none = BlockNumHash::default();
        match head {
            VerifierHead::PreActivation => {
                Self { block: none, timestamp: 0, source: Source::PreActivation, l1: none }
            }
            VerifierHead::Anchor { timestamp } => {
                Self { block: none, timestamp, source: Source::Anchor, l1: none }
            }
            VerifierHead::Verified { block, timestamp, l1 } => {
                Self { block, timestamp, source: Source::Verified, l1 }
            }
        }
    }
}

impl From<WireVerifierHead> for VerifierHead {
    fn from(wire: WireVerifierHead) -> Self {
        match wire.source {
            Source::PreActivation => Self::PreActivation,
            Source::Anchor => Self::Anchor { timestamp: wire.timestamp },
            Source::Verified => {
                Self::Verified { block: wire.block, timestamp: wire.timestamp, l1: wire.l1 }
            }
        }
    }
}

/// Everything a cross-chain verifier reports for one chain, from one snapshot of its state.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct VerifierStatus {
    /// The verifier's epoch. It changes whenever the verifier deletes a denial, and with a new
    /// store: a node that applied another epoch may hold blocks the verifier no longer stands
    /// behind, and derives again from its finalized head before it follows this answer.
    pub epoch: B256,
    /// The verified safe head, or `None` while the verifier cannot answer it yet, as until it
    /// has caught up with its sources after a start.
    pub verified: Option<VerifierHead>,
    /// The verified head whose L1 inclusion is finalized, or `None` like `verified`.
    pub finalized: Option<VerifierHead>,
    /// The blocks the verifier denied, at or above the requested height, ascending.
    pub denied: Vec<BlockNumHash>,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn verified_head_has_the_flat_wire_form() {
        let head = VerifierHead::Verified {
            block: BlockNumHash { number: 7, hash: B256::repeat_byte(1) },
            timestamp: 14,
            l1: BlockNumHash { number: 3, hash: B256::repeat_byte(2) },
        };

        let json = serde_json::to_value(head).unwrap();

        assert_eq!(
            json,
            serde_json::json!({
                "block": { "number": 7, "hash": B256::repeat_byte(1) },
                "timestamp": 14,
                "source": "verified",
                "l1": { "number": 3, "hash": B256::repeat_byte(2) },
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

    #[test]
    fn a_status_without_heads_round_trips() {
        let status = VerifierStatus {
            epoch: B256::repeat_byte(9),
            verified: None,
            finalized: None,
            denied: vec![BlockNumHash { number: 4, hash: B256::repeat_byte(4) }],
        };

        let json = serde_json::to_value(&status).unwrap();

        assert_eq!(json["verified"], serde_json::Value::Null);
        assert_eq!(serde_json::from_value::<VerifierStatus>(json).unwrap(), status);
    }
}
