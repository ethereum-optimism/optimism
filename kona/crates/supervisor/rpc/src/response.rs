//! Supervisor RPC response types.

use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, Bytes, ChainId, map::HashMap};
use kona_protocol::BlockInfo;
use kona_supervisor_types::SuperHead;
use serde::{Deserialize, Serialize, Serializer};

/// Describes superchain sync status.
///
/// Specs: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#supervisorsyncstatus>.
#[derive(Debug, Default, Clone, PartialEq, Eq)]
#[cfg_attr(
    feature = "serde",
    derive(serde::Serialize, serde::Deserialize),
    serde(rename_all = "camelCase")
)]
pub struct SupervisorSyncStatus {
    /// [`BlockInfo`] of highest L1 block.
    pub min_synced_l1: BlockInfo,
    /// Timestamp of highest cross-safe block.
    ///
    /// NOTE: Some fault-proof releases may already depend on `safe`, so we keep JSON field name as
    /// `safe`.
    #[cfg_attr(feature = "serde", serde(rename = "safeTimestamp"))]
    pub cross_safe_timestamp: u64,
    /// Timestamp of highest finalized block.
    pub finalized_timestamp: u64,
    /// Map of all tracked chains and their individual [`SupervisorChainSyncStatus`].
    pub chains: HashMap<ChainId, SupervisorChainSyncStatus>,
}

/// Describes the sync status for a specific chain.
///
/// Specs: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#supervisorchainsyncstatus>
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(
    feature = "serde",
    derive(serde::Serialize, serde::Deserialize),
    serde(rename_all = "camelCase")
)]
pub struct SupervisorChainSyncStatus {
    /// Highest [`Unsafe`] head of chain.
    ///
    /// [`Unsafe`]: op_alloy_consensus::interop::SafetyLevel::LocalUnsafe
    pub local_unsafe: BlockInfo,
    /// Highest [`CrossUnsafe`] head of chain.
    ///
    /// [`CrossUnsafe`]: op_alloy_consensus::interop::SafetyLevel::CrossUnsafe
    pub cross_unsafe: BlockNumHash,
    /// Highest [`LocalSafe`] head of chain.
    ///
    /// [`LocalSafe`]: op_alloy_consensus::interop::SafetyLevel::LocalSafe
    pub local_safe: BlockNumHash,
    /// Highest [`Safe`] head of chain [`BlockNumHash`].
    ///
    /// NOTE: Some fault-proof releases may already depend on `safe`, so we keep JSON field name as
    /// `safe`.
    ///
    /// [`Safe`]: op_alloy_consensus::interop::SafetyLevel::CrossSafe
    #[cfg_attr(feature = "serde", serde(rename = "safe"))]
    pub cross_safe: BlockNumHash,
    /// Highest [`Finalized`] head of chain [`BlockNumHash`].
    ///
    /// [`Finalized`]: op_alloy_consensus::interop::SafetyLevel::Finalized
    pub finalized: BlockNumHash,
}

impl From<SuperHead> for SupervisorChainSyncStatus {
    fn from(super_head: SuperHead) -> Self {
        let SuperHead { local_unsafe, cross_unsafe, local_safe, cross_safe, finalized, .. } =
            super_head;

        let cross_unsafe = cross_unsafe.unwrap_or_else(BlockInfo::default);
        let local_safe = local_safe.unwrap_or_else(BlockInfo::default);
        let cross_safe = cross_safe.unwrap_or_else(BlockInfo::default);
        let finalized = finalized.unwrap_or_else(BlockInfo::default);

        Self {
            local_unsafe,
            local_safe: local_safe.id(),
            cross_unsafe: cross_unsafe.id(),
            cross_safe: cross_safe.id(),
            finalized: finalized.id(),
        }
    }
}

/// This is same as [`kona_interop::ChainRootInfo`] but with [`u64`] serializing as a valid hex
/// string.
///
/// Required by
/// [`super_root_at_timestamp`](crate::jsonrpsee::SupervisorApiServer::super_root_at_timestamp) RPC
/// for marshalling and unmarshalling in GO implementation. Required for e2e tests.
#[derive(Debug, Clone, Eq, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ChainRootInfoRpc {
    /// The chain ID.
    #[serde(rename = "chainID", with = "alloy_serde::quantity")]
    pub chain_id: ChainId,
    /// The canonical output root of the latest canonical block at a particular timestamp.
    pub canonical: B256,
    /// The pending output root.
    ///
    /// This is the output root preimage for the latest block at a particular timestamp prior to
    /// validation of executing messages. If the original block was valid, this will be the
    /// preimage of the output root from the `canonical` array. If it was invalid, it will be
    /// the output root preimage from the optimistic block deposited transaction added to the
    /// deposit-only block.
    pub pending: Bytes,
}

/// This is same as [`kona_interop::SuperRootOutput`] but with timestamp serializing as a valid hex
/// string. version is also serialized as an even length hex string.
#[derive(Debug, Clone, Eq, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SuperRootOutputRpc {
    /// The Highest L1 Block that is cross-safe among all chains.
    pub cross_safe_derived_from: BlockNumHash,
    /// The timestamp of the super root.
    #[serde(with = "alloy_serde::quantity")]
    pub timestamp: u64,
    /// The super root hash.
    pub super_root: B256,
    /// The version of the super root.
    #[serde(serialize_with = "serialize_u8_as_hex")]
    pub version: u8,
    /// The chain root info for each chain in the dependency set.
    /// It represents the state of the chain at or before the timestamp.
    pub chains: Vec<ChainRootInfoRpc>,
}

/// Serializes a [u8] as a hex string. Ensure that the hex string has an even length.
///
/// This is used to serialize the [`SuperRootOutputRpc`]'s version field as a hex string.
fn serialize_u8_as_hex<S>(value: &u8, serializer: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    let hex_string = format!("0x{value:02x}");
    serializer.serialize_str(&hex_string)
}

#[cfg(test)]
mod test {
    use super::*;
    use alloy_primitives::b256;
    use kona_interop::SUPER_ROOT_VERSION;

    const CHAIN_STATUS: &str = r#"
    {
        "localUnsafe": {
            "number": 100,
            "hash": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
            "timestamp": 40044440000,
            "parentHash": "0x111def1234567890abcdef1234567890abcdef1234500000abcdef123456aaaa"
        },
        "crossUnsafe": {
            "number": 90,
            "hash": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
        },
        "localSafe": {
            "number": 80,
            "hash": "0x34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef13"
        },
        "safe": {
            "number": 70,
            "hash": "0x567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234"
        },
        "finalized": {
            "number": 60,
            "hash": "0x34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12"
        }
    }"#;

    const STATUS: &str = r#"
    {
        "minSyncedL1": {
            "number": 100,
            "hash": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
            "timestamp": 40044440000,
            "parentHash": "0x111def1234567890abcdef1234567890abcdef1234500000abcdef123456aaaa"
        },
        "safeTimestamp": 40044450000,
        "finalizedTimestamp": 40044460000,
        "chains" : {
            "1": {
                "localUnsafe": {
                    "number": 100,
                    "hash": "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
                    "timestamp": 40044440000,
                    "parentHash": "0x111def1234567890abcdef1234567890abcdef1234500000abcdef123456aaaa"
                },
                "crossUnsafe": {
                    "number": 90,
                    "hash": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
                },
                "localSafe": {
                    "number": 80,
                    "hash": "0x34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef13"
                },
                "safe": {
                    "number": 70,
                    "hash": "0x567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234"
                },
                "finalized": {
                    "number": 60,
                    "hash": "0x34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12"
                }
            }
        }
    }"#;

    #[cfg(feature = "serde")]
    #[test]
    fn test_serialize_supervisor_chain_sync_status() {
        assert_eq!(
            serde_json::from_str::<SupervisorChainSyncStatus>(CHAIN_STATUS)
                .expect("should deserialize"),
            SupervisorChainSyncStatus {
                local_unsafe: BlockInfo {
                    number: 100,
                    hash: b256!(
                        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
                    ),
                    timestamp: 40044440000,
                    parent_hash: b256!(
                        "0x111def1234567890abcdef1234567890abcdef1234500000abcdef123456aaaa"
                    ),
                },
                cross_unsafe: BlockNumHash::new(
                    90,
                    b256!("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
                ),
                local_safe: BlockNumHash::new(
                    80,
                    b256!("0x34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef13")
                ),
                cross_safe: BlockNumHash::new(
                    70,
                    b256!("0x567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234")
                ),
                finalized: BlockNumHash::new(
                    60,
                    b256!("0x34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12")
                ),
            }
        )
    }

    #[cfg(feature = "serde")]
    #[test]
    fn test_serialize_supervisor_sync_status() {
        let mut chains = HashMap::default();

        chains.insert(
            1,
            SupervisorChainSyncStatus {
                local_unsafe: BlockInfo {
                    number: 100,
                    hash: b256!(
                        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
                    ),
                    timestamp: 40044440000,
                    parent_hash: b256!(
                        "0x111def1234567890abcdef1234567890abcdef1234500000abcdef123456aaaa"
                    ),
                },
                cross_unsafe: BlockNumHash::new(
                    90,
                    b256!("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
                ),
                local_safe: BlockNumHash::new(
                    80,
                    b256!("0x34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef13"),
                ),
                cross_safe: BlockNumHash::new(
                    70,
                    b256!("0x567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234"),
                ),
                finalized: BlockNumHash::new(
                    60,
                    b256!("0x34567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12"),
                ),
            },
        );

        assert_eq!(
            serde_json::from_str::<SupervisorSyncStatus>(STATUS).expect("should deserialize"),
            SupervisorSyncStatus {
                min_synced_l1: BlockInfo {
                    number: 100,
                    hash: b256!(
                        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
                    ),
                    timestamp: 40044440000,
                    parent_hash: b256!(
                        "0x111def1234567890abcdef1234567890abcdef1234500000abcdef123456aaaa"
                    ),
                },
                cross_safe_timestamp: 40044450000,
                finalized_timestamp: 40044460000,
                chains,
            }
        )
    }

    #[test]
    fn test_super_root_version_even_length_hex() {
        let root = SuperRootOutputRpc {
            cross_safe_derived_from: BlockNumHash::default(),
            timestamp: 0,
            super_root: B256::default(),
            version: SUPER_ROOT_VERSION,
            chains: vec![],
        };
        let json = serde_json::to_string(&root).expect("should serialize");
        let v: serde_json::Value = serde_json::from_str(&json).expect("valid json");
        let version_field =
            v.get("version").expect("version field present").as_str().expect("version is string");
        let hex_part = &version_field[2..]; // remove 0x
        assert_eq!(hex_part.len() % 2, 0, "Hex string should have even length");
        // For SUPER_ROOT_VERSION = 1, should be 0x01
        assert_eq!(version_field, "0x01");
    }
}
