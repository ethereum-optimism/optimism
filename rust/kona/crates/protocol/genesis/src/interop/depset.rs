//! Dependency set primitives shared by `kona-interop` and `kona-registry`.

use super::MESSAGE_EXPIRY_WINDOW;
use alloc::collections::BTreeMap;
use alloy_primitives::ChainId;

/// Configuration for a dependency of a chain
#[derive(Debug, Clone, PartialEq, Eq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct ChainDependency {}

/// Configuration for the dependency set
#[derive(Debug, Clone, PartialEq, Eq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
#[allow(clippy::zero_sized_map_values)]
pub struct DependencySet {
    /// Dependencies information per chain.
    pub dependencies: BTreeMap<ChainId, ChainDependency>,

    /// Override message expiry window to use for this dependency set.
    pub override_message_expiry_window: Option<u64>,
}

impl DependencySet {
    /// Returns the message expiry window associated with this dependency set.
    pub const fn get_message_expiry_window(&self) -> u64 {
        match self.override_message_expiry_window {
            Some(window) if window > 0 => window,
            _ => MESSAGE_EXPIRY_WINDOW,
        }
    }
}

#[cfg(test)]
#[allow(clippy::zero_sized_map_values)]
mod tests {
    use super::*;
    use alloc::collections::BTreeMap;
    use alloy_primitives::ChainId;

    const fn create_dependency_set(
        dependencies: BTreeMap<ChainId, ChainDependency>,
        override_expiry: u64,
    ) -> DependencySet {
        DependencySet { dependencies, override_message_expiry_window: Some(override_expiry) }
    }

    #[test]
    fn test_get_message_expiry_window_default() {
        let deps = BTreeMap::default();
        // override_message_expiry_window is 0, so default should be used
        let ds = create_dependency_set(deps, 0);
        assert_eq!(
            ds.get_message_expiry_window(),
            MESSAGE_EXPIRY_WINDOW,
            "Should return default expiry window when override is 0"
        );
    }

    #[test]
    fn test_get_message_expiry_window_override() {
        let deps = BTreeMap::default();
        let override_value = 12345;
        let ds = create_dependency_set(deps, override_value);
        assert_eq!(
            ds.get_message_expiry_window(),
            override_value,
            "Should return override expiry window when it's non-zero"
        );
    }

    #[test]
    #[cfg(feature = "serde")]
    fn depset_json_round_trip() {
        let json = include_str!("../../tests/fixtures/dependency_set.json");
        let parsed: DependencySet = serde_json::from_str(json).unwrap();
        let reserialized = serde_json::to_string(&parsed).unwrap();
        let parsed_again: DependencySet = serde_json::from_str(&reserialized).unwrap();
        assert_eq!(parsed, parsed_again);
    }
}
