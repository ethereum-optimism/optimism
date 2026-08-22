//! The JSON shapes of `supernode_syncStatus` and `superroot_atTimestamp`.
//!
//! These types exist to be byte-comparable with what op-supernode serves. Their Go counterparts
//! are `eth.SuperNodeSyncStatusResponse` (`op-service/eth/supernode_status.go`) and
//! `eth.SuperRootAtTimestampResponse` (`op-service/eth/superroot_at_timestamp.go`), and the
//! consumers on the other side of the wire — op-proposer, op-challenger, op-interop-mon,
//! `kona-sp1-proposer` — decode exactly those. A field that is spelled differently, or numeric
//! where Go is hexadecimal, is a consumer that fails to deserialize or, worse, one that reads a
//! zero where a real value was meant.
//!
//! Three encodings here are easy to get wrong and are therefore spelled out rather than derived:
//!
//! - A chain id is a **decimal string**. Go's `eth.ChainID` is a `uint256` with a `MarshalText`
//!   that renders decimal, so it is a JSON string both as a map key and as an array element —
//!   `"chain_ids": ["901", "902"]`, not `[901, 902]`.
//! - `super.timestamp` is **hexadecimal**, because `eth.SuperV1` marshals it through
//!   `hexutil.Uint64`. Every *other* timestamp in these responses is a plain JSON number, because
//!   those fields are plain `uint64` in Go.
//! - `super.chains` entries carry the field names `ChainID` and `Output`: `eth.ChainIDAndOutput`
//!   has no struct tags, so Go marshals its Go field names verbatim.
//!
//! Nothing here computes anything. The super root hash is [`kona_interop::SuperRoot`]'s, and the
//! per-chain output root is [`kona_protocol::OutputRoot`]'s, so the commitments this API publishes
//! are the ones the rest of the stack already agrees on.

use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, ChainId};
use kona_protocol::{OutputRoot, SyncStatus};
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use std::collections::BTreeMap;

/// A chain id as it appears on this wire: a decimal string.
///
/// A newtype rather than a `serialize_with` on every field, because it is also a **map key**, and
/// a key has no attribute to hang the conversion on — `serde_json` asks the key type to serialize
/// itself as a string. Ordering is the numeric ordering of the underlying id, so a
/// [`BTreeMap`] keyed by this iterates in the ascending chain-id order op-supernode sorts
/// `chain_ids` into.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) struct WireChainId(pub(crate) ChainId);

impl Serialize for WireChainId {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.collect_str(&self.0)
    }
}

/// A block's number and hash, as `eth.BlockID`.
///
/// `number` is a plain JSON number: `eth.BlockID.Number` is a bare `uint64` with no
/// `hexutil` wrapper.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default, Serialize)]
pub(crate) struct WireBlockId {
    /// The block hash.
    pub(crate) hash: B256,
    /// The block number.
    pub(crate) number: u64,
}

impl From<BlockNumHash> for WireBlockId {
    fn from(id: BlockNumHash) -> Self {
        Self { hash: id.hash, number: id.number }
    }
}

/// The V0 output-root preimage, as `eth.OutputV0`.
///
/// The version word is not a field: it is implied by the type, exactly as it is in Go, and it is
/// [`OutputRoot::encode`] that puts it into the hashed image.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
#[serde(rename_all = "camelCase")]
pub(crate) struct WireOutputV0 {
    /// The L2 state root.
    pub(crate) state_root: B256,
    /// The storage root of the `L2ToL1MessagePasser` predeploy.
    pub(crate) message_passer_storage_root: B256,
    /// The L2 block hash.
    pub(crate) block_hash: B256,
}

impl From<OutputRoot> for WireOutputV0 {
    fn from(output: OutputRoot) -> Self {
        Self {
            state_root: output.state_root,
            message_passer_storage_root: output.bridge_storage_root,
            block_hash: output.block_hash,
        }
    }
}

/// One chain's optimistic output and the L1 block it needs, as `eth.OutputWithRequiredL1`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
pub(crate) struct WireOutputWithRequiredL1 {
    /// The output-root preimage.
    ///
    /// Always present here. It is `*eth.OutputV0` in Go, so the field is nullable on the wire and
    /// `kona-sp1-super-range-executor` decodes it as such; op-supernode never sends a null either.
    pub(crate) output: WireOutputV0,
    /// The hash of [`Self::output`].
    pub(crate) output_root: B256,
    /// The lowest L1 block from which this output can be derived.
    pub(crate) required_l1: WireBlockId,
}

/// One chain's contribution to a super root, as `eth.ChainIDAndOutput`.
///
/// The field names are capitalised because the Go struct carries no JSON tags.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
pub(crate) struct WireChainIdAndOutput {
    /// The chain id.
    #[serde(rename = "ChainID")]
    pub(crate) chain_id: WireChainId,
    /// That chain's output root at the super root's timestamp.
    #[serde(rename = "Output")]
    pub(crate) output: B256,
}

/// A V1 super root's preimage, as `eth.SuperV1`.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub(crate) struct WireSuperV1 {
    /// The super root's timestamp, hexadecimal — see the module documentation.
    #[serde(serialize_with = "hex_u64")]
    pub(crate) timestamp: u64,
    /// The per-chain output roots, ascending by chain id.
    pub(crate) chains: Vec<WireChainIdAndOutput>,
}

/// The verified super root at a timestamp, as `eth.SuperRootResponseData`.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub(crate) struct WireSuperRootData {
    /// The lowest L1 block that includes everything needed to verify this super root.
    pub(crate) verified_required_l1: WireBlockId,
    /// The super root's preimage.
    #[serde(rename = "super")]
    pub(crate) super_v1: WireSuperV1,
    /// The super root itself.
    pub(crate) super_root: B256,
}

/// The response to `superroot_atTimestamp`, as `eth.SuperRootAtTimestampResponse`.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub(crate) struct WireSuperRootAtTimestamp {
    /// The L1 block the slowest L1 processor in the supernode is on. Every L1 block strictly
    /// below this one has been fully processed by every chain.
    pub(crate) current_l1: WireBlockId,
    /// The highest cross-safe L2 timestamp across the chain set.
    pub(crate) safe_timestamp: u64,
    /// The highest local-safe L2 timestamp across the chain set.
    pub(crate) local_safe_timestamp: u64,
    /// The highest finalized L2 timestamp across the chain set.
    pub(crate) finalized_timestamp: u64,
    /// Per-chain outputs that would hold if verification succeeded. A chain that has not derived
    /// the requested timestamp yet is absent.
    pub(crate) optimistic_at_timestamp: BTreeMap<WireChainId, WireOutputWithRequiredL1>,
    /// The chain set, ascending.
    pub(crate) chain_ids: Vec<WireChainId>,
    /// The super root at the requested timestamp, when one can be stated. Absent — the Go field
    /// is `omitempty` — while the timestamp is neither verified nor covered by the handoff.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub(crate) data: Option<WireSuperRootData>,
}

/// The response to `supernode_syncStatus`, as `eth.SuperNodeSyncStatusResponse`.
///
/// The per-chain values are [`kona_protocol::SyncStatus`], the same type this process's chains
/// serve from their own `optimism_syncStatus`. It carries two fields fewer than Go's
/// `eth.SyncStatus` — `pending_safe_l2` and `cross_unsafe_l2`, which kona does not track — and
/// those decode as their zero values on the Go side, exactly as they do when a Go consumer reads
/// a kona-node's own sync status.
#[derive(Debug, Clone, PartialEq, Eq, Serialize)]
pub(crate) struct WireSyncStatus {
    /// Each chain's own sync status.
    pub(crate) chains: BTreeMap<WireChainId, SyncStatus>,
    /// The chain set, ascending.
    pub(crate) chain_ids: Vec<WireChainId>,
    /// The L1 block the slowest L1 processor in the supernode is on.
    pub(crate) current_l1: WireBlockId,
    /// The highest cross-safe L2 timestamp across the chain set.
    pub(crate) safe_timestamp: u64,
    /// The highest local-safe L2 timestamp across the chain set.
    pub(crate) local_safe_timestamp: u64,
    /// The highest finalized L2 timestamp across the chain set.
    pub(crate) finalized_timestamp: u64,
}

/// Serializes a `u64` the way `hexutil.Uint64` does: `0x`-prefixed, lower case, no leading zeros.
fn hex_u64<S: Serializer>(value: &u64, serializer: S) -> Result<S::Ok, S::Error> {
    serializer.collect_str(&format_args!("{value:#x}"))
}

/// A `u64` request parameter, accepted as either a hexadecimal string or a JSON number.
///
/// `superroot_atTimestamp` takes `hexutil.Uint64` in Go, so every Go consumer — op-proposer,
/// op-challenger, op-interop-mon, and `op-service/sources.SuperNodeClient`, which the devstack
/// uses — sends `"0x…"`. A plain `u64` parameter would reject all of them. Numbers are accepted
/// too, because a human with `curl` and the `kona-sp1` mirror's own decoder both write them, and
/// refusing one spelling of the same integer buys nothing.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct WireU64(pub(crate) u64);

impl<'de> Deserialize<'de> for WireU64 {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        deserializer.deserialize_any(WireU64Visitor)
    }
}

/// Accepts the two spellings [`WireU64`] documents.
struct WireU64Visitor;

impl serde::de::Visitor<'_> for WireU64Visitor {
    type Value = WireU64;

    fn expecting(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter.write_str("a u64, as a number or as a 0x-prefixed hexadecimal string")
    }

    fn visit_u64<E: serde::de::Error>(self, value: u64) -> Result<Self::Value, E> {
        Ok(WireU64(value))
    }

    fn visit_str<E: serde::de::Error>(self, value: &str) -> Result<Self::Value, E> {
        let parsed = value.strip_prefix("0x").or_else(|| value.strip_prefix("0X")).map_or_else(
            // Go's `hexutil.Uint64` requires the prefix, but a decimal string is unambiguous and
            // some hand-written clients send one.
            || value.parse(),
            |digits| u64::from_str_radix(digits, 16),
        );
        parsed.map(WireU64).map_err(|err| E::custom(format!("invalid u64 {value:?}: {err}")))
    }
}
