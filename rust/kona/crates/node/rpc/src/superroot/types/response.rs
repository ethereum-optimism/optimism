//! The wire types of `superroot_atTimestamp`, in op-node's shape.

use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, U64};
use kona_interop::SuperRoot;
use kona_protocol::OutputRoot;
use serde::{Deserialize, Deserializer, Serialize, Serializer, de::Error as _};
use std::collections::BTreeMap;

/// A chain ID in the form op-node serializes it: a decimal string. Accepts hex on the way in.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub struct DecimalChainId(pub u64);

impl Serialize for DecimalChainId {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.collect_str(&self.0)
    }
}

impl<'de> Deserialize<'de> for DecimalChainId {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let value: U64 = String::deserialize(deserializer)?.parse().map_err(D::Error::custom)?;
        Ok(Self(value.to()))
    }
}

/// The response of `superroot_atTimestamp`.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperRootAtTimestampResponse {
    /// The L1 block derivation is at. Every L1 block strictly below it is fully processed.
    pub current_l1: BlockNumHash,
    /// The timestamp of the safe head.
    #[serde(rename = "safe_timestamp")]
    pub current_safe_timestamp: u64,
    /// The timestamp of the local-safe head.
    #[serde(rename = "local_safe_timestamp")]
    pub current_local_safe_timestamp: u64,
    /// The timestamp of the finalized head.
    #[serde(rename = "finalized_timestamp")]
    pub current_finalized_timestamp: u64,
    /// The output at the requested timestamp and the L1 block needed to derive it. Empty when the
    /// block at the timestamp is not yet local-safe.
    pub optimistic_at_timestamp: BTreeMap<DecimalChainId, OutputWithRequiredL1>,
    /// The chains in the dependency set: only this one.
    pub chain_ids: Vec<DecimalChainId>,
    /// The super root at the requested timestamp. Absent when the block is not yet local-safe.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub data: Option<SuperRootResponseData>,
}

/// An output root with the L1 block that first made its block safe.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct OutputWithRequiredL1 {
    /// The output root preimage.
    pub output: OutputV0Response,
    /// The output root.
    pub output_root: B256,
    /// The L1 block that first made the block safe.
    pub required_l1: BlockNumHash,
}

/// A version 0 output root preimage.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct OutputV0Response {
    /// The state root.
    pub state_root: B256,
    /// The storage root of the `L2ToL1MessagePasser` predeploy.
    pub message_passer_storage_root: B256,
    /// The block hash.
    pub block_hash: B256,
}

impl From<&OutputRoot> for OutputV0Response {
    fn from(output: &OutputRoot) -> Self {
        Self {
            state_root: output.state_root,
            message_passer_storage_root: output.bridge_storage_root,
            block_hash: output.block_hash,
        }
    }
}

/// The super root section of the response.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperRootResponseData {
    /// The L1 block that first made every chain's block safe.
    pub verified_required_l1: BlockNumHash,
    /// The super root preimage.
    #[serde(rename = "super")]
    pub super_v1: SuperV1Response,
    /// The super root.
    pub super_root: B256,
}

/// A version 1 super root preimage, whose timestamp is a hex quantity unlike the plain integers
/// elsewhere in the response.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct SuperV1Response {
    /// The timestamp of the super root.
    pub timestamp: U64,
    /// The output root of every chain, sorted by chain ID.
    pub chains: Vec<ChainIdAndOutput>,
}

impl From<&SuperRoot> for SuperV1Response {
    fn from(super_root: &SuperRoot) -> Self {
        Self {
            timestamp: U64::from(super_root.timestamp),
            chains: super_root
                .output_roots
                .iter()
                .map(|root| ChainIdAndOutput {
                    chain_id: DecimalChainId(root.chain_id),
                    output: root.output_root,
                })
                .collect(),
        }
    }
}

/// One chain's output root inside a super root preimage.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ChainIdAndOutput {
    /// The chain ID.
    #[serde(rename = "ChainID")]
    pub chain_id: DecimalChainId,
    /// The output root.
    #[serde(rename = "Output")]
    pub output: B256,
}
