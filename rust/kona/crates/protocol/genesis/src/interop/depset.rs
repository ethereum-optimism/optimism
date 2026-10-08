//! Dependency set primitives shared by `kona-interop` and `kona-registry`.

use super::MESSAGE_EXPIRY_WINDOW;
use alloc::collections::BTreeMap;
use alloy_primitives::ChainId;
use core::fmt;

/// Configuration for a dependency of a chain
#[derive(Debug, Clone, PartialEq, Eq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
pub struct ChainDependency {}

/// An override of the message expiry window, in seconds. It may only shorten the window:
/// `L2ToL2CrossDomainMessenger` marks a message expired, and apps refund it, a day after
/// [`MESSAGE_EXPIRY_WINDOW`] has passed, so a longer window could let an expired message still be
/// relayed. A value above [`MESSAGE_EXPIRY_WINDOW`] cannot be constructed or deserialized, as in
/// op-core.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(try_from = "u64", into = "u64"))]
pub struct MessageExpiryOverride(u64);

impl MessageExpiryOverride {
    /// Returns the override in seconds.
    pub const fn get(self) -> u64 {
        self.0
    }
}

/// Error returned for a message expiry window override above [`MESSAGE_EXPIRY_WINDOW`].
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct MessageExpiryOverrideTooLong(pub u64);

impl fmt::Display for MessageExpiryOverrideTooLong {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "message expiry window override {}s exceeds protocol window {}s",
            self.0, MESSAGE_EXPIRY_WINDOW
        )
    }
}

impl core::error::Error for MessageExpiryOverrideTooLong {}

impl TryFrom<u64> for MessageExpiryOverride {
    type Error = MessageExpiryOverrideTooLong;

    fn try_from(window: u64) -> Result<Self, Self::Error> {
        if window > MESSAGE_EXPIRY_WINDOW {
            return Err(MessageExpiryOverrideTooLong(window));
        }
        Ok(Self(window))
    }
}

impl From<MessageExpiryOverride> for u64 {
    fn from(window: MessageExpiryOverride) -> Self {
        window.0
    }
}

#[cfg(feature = "arbitrary")]
impl<'a> arbitrary::Arbitrary<'a> for MessageExpiryOverride {
    fn arbitrary(u: &mut arbitrary::Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(Self(u.int_in_range(0..=MESSAGE_EXPIRY_WINDOW)?))
    }
}

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
    pub override_message_expiry_window: Option<MessageExpiryOverride>,
}

impl DependencySet {
    /// Returns the message expiry window associated with this dependency set.
    pub const fn get_message_expiry_window(&self) -> u64 {
        match self.override_message_expiry_window {
            Some(MessageExpiryOverride(window)) if window > 0 => window,
            _ => MESSAGE_EXPIRY_WINDOW,
        }
    }
}

#[cfg(test)]
#[allow(clippy::zero_sized_map_values)]
mod tests {
    use super::*;
    use alloc::{collections::BTreeMap, string::ToString};
    use alloy_primitives::ChainId;

    fn create_dependency_set(
        dependencies: BTreeMap<ChainId, ChainDependency>,
        override_expiry: u64,
    ) -> DependencySet {
        DependencySet {
            dependencies,
            override_message_expiry_window: Some(override_expiry.try_into().unwrap()),
        }
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
    fn test_message_expiry_override_rejects_above_protocol_window() {
        let err = MessageExpiryOverride::try_from(MESSAGE_EXPIRY_WINDOW + 1).unwrap_err();
        assert_eq!(
            err.to_string(),
            "message expiry window override 604801s exceeds protocol window 604800s"
        );
        assert_eq!(
            MessageExpiryOverride::try_from(MESSAGE_EXPIRY_WINDOW).unwrap().get(),
            MESSAGE_EXPIRY_WINDOW
        );
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_depset_json_rejects_override_above_protocol_window() {
        let parse = |window: u64| {
            serde_json::from_str::<DependencySet>(&alloc::format!(
                r#"{{"dependencies":{{}},"overrideMessageExpiryWindow":{window}}}"#
            ))
        };
        let err = parse(MESSAGE_EXPIRY_WINDOW + 1).unwrap_err();
        assert!(
            err.to_string()
                .contains("message expiry window override 604801s exceeds protocol window 604800s"),
            "unexpected error: {err}"
        );
        assert_eq!(
            parse(MESSAGE_EXPIRY_WINDOW).unwrap().get_message_expiry_window(),
            MESSAGE_EXPIRY_WINDOW
        );
        let missing: DependencySet = serde_json::from_str(r#"{"dependencies":{}}"#).unwrap();
        assert_eq!(missing.override_message_expiry_window, None);
    }

    /// op-core pins the same 7 days (`MessageExpiryTimeSecondsInterop`).
    #[test]
    fn test_message_expiry_window_is_seven_days() {
        assert_eq!(MESSAGE_EXPIRY_WINDOW, 604_800);
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
