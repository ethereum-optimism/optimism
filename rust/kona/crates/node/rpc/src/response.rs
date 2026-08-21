//! Response to safe head request

use alloy_eips::BlockNumHash;
use alloy_primitives::ChainId;
use kona_genesis::{ChainDependency, DependencySet};
use std::collections::BTreeMap;

/// The safe head response.
///
/// <https://github.com/ethereum-optimism/optimism/blob/77c91d09eaa44d2c53bec60eb89c5c55737bc325/op-service/eth/output.go#L19-L22>
/// Note: the optimism "eth.BlockID" type is number,hash <https://github.com/ethereum-optimism/optimism/blob/77c91d09eaa44d2c53bec60eb89c5c55737bc325/op-service/eth/id.go#L10-L13>
#[derive(Debug, Clone, Copy, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SafeHeadResponse {
    /// The L1 block.
    pub l1_block: BlockNumHash,
    /// The safe head.
    pub safe_head: BlockNumHash,
}

/// The `optimism_dependencySet` response.
///
/// op-node serves this method from `nodeAPI.DependencySet` (`op-node/node/api.go:178`) and the Go
/// client decodes it into `depset.StaticConfigDependencySet`, whose JSON is written by
/// `minStaticConfigDependencySet` (`op-core/interop/depset/static_depset.go:52`): a map from the
/// chain id, rendered in decimal, to that chain's dependency entry, and an expiry-window override
/// that is *omitted* when it is zero rather than sent as null. That omission is the reason this
/// type exists instead of serializing [`DependencySet`] directly — kona models the override as an
/// `Option<u64>`, which would put `null` on the wire for the common case.
///
/// The entries themselves are empty objects on both sides: a chain either is in the set or is not,
/// and there is nothing else recorded per chain.
#[derive(Debug, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(clippy::zero_sized_map_values)]
pub struct DependencySetResponse {
    /// The dependency entry of every chain in the set, keyed by chain id.
    pub dependencies: BTreeMap<ChainId, ChainDependency>,
    /// The message expiry window this set overrides the protocol default with, when it sets one.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub override_message_expiry_window: Option<u64>,
}

impl From<&DependencySet> for DependencySetResponse {
    fn from(depset: &DependencySet) -> Self {
        Self {
            dependencies: depset.dependencies.clone(),
            // A zero override is no override: `StaticConfigDependencySet.MessageExpiryWindow`
            // reads zero as "use the protocol default", and `omitempty` means Go never puts a zero
            // on the wire in the first place. Normalising it here keeps a config that wrote 0
            // explicitly from producing a field op-node would not have sent.
            override_message_expiry_window: depset
                .override_message_expiry_window
                .filter(|window| *window > 0),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// The wire shape op-node puts on `optimism_dependencySet`, as the Go client decodes it.
    ///
    /// Written out as a literal rather than round-tripped, because the parity claim is about the
    /// field names and the decimal chain-id keys, and a round trip through this type would agree
    /// with itself whatever they were.
    #[test]
    fn dependency_set_response_matches_op_node() {
        let depset = DependencySet {
            dependencies: BTreeMap::from([(901, ChainDependency {}), (902, ChainDependency {})]),
            override_message_expiry_window: None,
        };

        assert_eq!(
            serde_json::to_value(DependencySetResponse::from(&depset)).unwrap(),
            serde_json::json!({"dependencies": {"901": {}, "902": {}}}),
        );
    }

    /// An override is sent when there is one, under the name Go reads.
    #[test]
    fn dependency_set_response_carries_an_expiry_override() {
        let depset = DependencySet {
            dependencies: BTreeMap::from([(901, ChainDependency {})]),
            override_message_expiry_window: Some(120),
        };

        assert_eq!(
            serde_json::to_value(DependencySetResponse::from(&depset)).unwrap(),
            serde_json::json!({
                "dependencies": {"901": {}},
                "overrideMessageExpiryWindow": 120,
            }),
        );
    }

    /// A zero override is not an override: Go's `omitempty` never puts one on the wire, and
    /// `MessageExpiryWindow` reads zero as the protocol default.
    #[test]
    fn a_zero_expiry_override_is_left_off_the_wire() {
        let depset = DependencySet {
            dependencies: BTreeMap::new(),
            override_message_expiry_window: Some(0),
        };

        assert_eq!(
            serde_json::to_value(DependencySetResponse::from(&depset)).unwrap(),
            serde_json::json!({"dependencies": {}}),
        );
    }

    // <https://github.com/alloy-rs/op-alloy/issues/155>
    #[test]
    fn test_safe_head_response() {
        let s = r#"{"l1Block":{"hash":"0x7de331305c2bb3e5642a2adcb9c003cc67cefc7b05a3da5a6a4b12cf3af15407","number":6834391},"safeHead":{"hash":"0xa5e5ec1ade7d6fef209f73861bf0080950cde74c4b0c07823983eb5225e282a8","number":18266679}}"#;
        let _response: SafeHeadResponse = serde_json::from_str(s).unwrap();
    }
}
