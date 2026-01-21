//! Contains the `[OpSpecId]` type and its implementation.
use core::str::FromStr;
use revm::primitives::hardfork::{name as eth_name, SpecId, UnknownHardfork};

/// Optimism spec id.
#[repr(u8)]
#[derive(Clone, Copy, Debug, Hash, PartialEq, Eq, PartialOrd, Ord, Default)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[allow(non_camel_case_types)]
pub enum OpSpecId {
    /// Bedrock spec id.
    BEDROCK = 100,
    /// Regolith spec id.
    REGOLITH,
    /// Canyon spec id.
    CANYON,
    /// Ecotone spec id.
    ECOTONE,
    /// Fjord spec id.
    FJORD,
    /// Granite spec id.
    GRANITE,
    /// Holocene spec id.
    HOLOCENE,
    /// Isthmus spec id.
    ISTHMUS,
    /// Jovian spec id.
    #[default]
    JOVIAN,
    /// Interop spec id.
    INTEROP,
    /// Osaka spec id.
    OSAKA,
}

impl OpSpecId {
    /// Converts the [`OpSpecId`] into a [`SpecId`].
    pub const fn into_eth_spec(self) -> SpecId {
        match self {
            Self::BEDROCK | Self::REGOLITH => SpecId::MERGE,
            Self::CANYON => SpecId::SHANGHAI,
            Self::ECOTONE | Self::FJORD | Self::GRANITE | Self::HOLOCENE => SpecId::CANCUN,
            Self::ISTHMUS | Self::JOVIAN | Self::INTEROP => SpecId::PRAGUE,
            Self::OSAKA => SpecId::OSAKA,
        }
    }

    /// Checks if the [`OpSpecId`] is enabled in the other [`OpSpecId`].
    pub const fn is_enabled_in(self, other: OpSpecId) -> bool {
        other as u8 <= self as u8
    }
}

impl From<OpSpecId> for SpecId {
    fn from(spec: OpSpecId) -> Self {
        spec.into_eth_spec()
    }
}

impl FromStr for OpSpecId {
    type Err = UnknownHardfork;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            name::BEDROCK => Ok(OpSpecId::BEDROCK),
            name::REGOLITH => Ok(OpSpecId::REGOLITH),
            name::CANYON => Ok(OpSpecId::CANYON),
            name::ECOTONE => Ok(OpSpecId::ECOTONE),
            name::FJORD => Ok(OpSpecId::FJORD),
            name::GRANITE => Ok(OpSpecId::GRANITE),
            name::HOLOCENE => Ok(OpSpecId::HOLOCENE),
            name::ISTHMUS => Ok(OpSpecId::ISTHMUS),
            name::JOVIAN => Ok(OpSpecId::JOVIAN),
            name::INTEROP => Ok(OpSpecId::INTEROP),
            eth_name::OSAKA => Ok(OpSpecId::OSAKA),
            _ => Err(UnknownHardfork),
        }
    }
}

impl From<OpSpecId> for &'static str {
    fn from(spec_id: OpSpecId) -> Self {
        match spec_id {
            OpSpecId::BEDROCK => name::BEDROCK,
            OpSpecId::REGOLITH => name::REGOLITH,
            OpSpecId::CANYON => name::CANYON,
            OpSpecId::ECOTONE => name::ECOTONE,
            OpSpecId::FJORD => name::FJORD,
            OpSpecId::GRANITE => name::GRANITE,
            OpSpecId::HOLOCENE => name::HOLOCENE,
            OpSpecId::ISTHMUS => name::ISTHMUS,
            OpSpecId::JOVIAN => name::JOVIAN,
            OpSpecId::INTEROP => name::INTEROP,
            OpSpecId::OSAKA => eth_name::OSAKA,
        }
    }
}

/// String identifiers for Optimism hardforks
pub mod name {
    /// Bedrock spec name.
    pub const BEDROCK: &str = "Bedrock";
    /// Regolith spec name.
    pub const REGOLITH: &str = "Regolith";
    /// Canyon spec name.
    pub const CANYON: &str = "Canyon";
    /// Ecotone spec name.
    pub const ECOTONE: &str = "Ecotone";
    /// Fjord spec name.
    pub const FJORD: &str = "Fjord";
    /// Granite spec name.
    pub const GRANITE: &str = "Granite";
    /// Holocene spec name.
    pub const HOLOCENE: &str = "Holocene";
    /// Isthmus spec name.
    pub const ISTHMUS: &str = "Isthmus";
    /// Jovian spec name.
    pub const JOVIAN: &str = "Jovian";
    /// Interop spec name.
    pub const INTEROP: &str = "Interop";
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::vec;

    #[test]
    fn test_op_spec_id_eth_spec_compatibility() {
        // Define test cases: (OpSpecId, enabled in ETH specs, enabled in OP specs)
        let test_cases = [
            (
                OpSpecId::BEDROCK,
                vec![
                    (SpecId::MERGE, true),
                    (SpecId::SHANGHAI, false),
                    (SpecId::CANCUN, false),
                    (SpecId::default(), false),
                ],
                vec![(OpSpecId::BEDROCK, true), (OpSpecId::REGOLITH, false)],
            ),
            (
                OpSpecId::REGOLITH,
                vec![
                    (SpecId::MERGE, true),
                    (SpecId::SHANGHAI, false),
                    (SpecId::CANCUN, false),
                    (SpecId::default(), false),
                ],
                vec![(OpSpecId::BEDROCK, true), (OpSpecId::REGOLITH, true)],
            ),
            (
                OpSpecId::CANYON,
                vec![
                    (SpecId::MERGE, true),
                    (SpecId::SHANGHAI, true),
                    (SpecId::CANCUN, false),
                    (SpecId::default(), false),
                ],
                vec![
                    (OpSpecId::BEDROCK, true),
                    (OpSpecId::REGOLITH, true),
                    (OpSpecId::CANYON, true),
                ],
            ),
            (
                OpSpecId::ECOTONE,
                vec![
                    (SpecId::MERGE, true),
                    (SpecId::SHANGHAI, true),
                    (SpecId::CANCUN, true),
                    (SpecId::default(), false),
                ],
                vec![
                    (OpSpecId::BEDROCK, true),
                    (OpSpecId::REGOLITH, true),
                    (OpSpecId::CANYON, true),
                    (OpSpecId::ECOTONE, true),
                ],
            ),
            (
                OpSpecId::FJORD,
                vec![
                    (SpecId::MERGE, true),
                    (SpecId::SHANGHAI, true),
                    (SpecId::CANCUN, true),
                    (SpecId::default(), false),
                ],
                vec![
                    (OpSpecId::BEDROCK, true),
                    (OpSpecId::REGOLITH, true),
                    (OpSpecId::CANYON, true),
                    (OpSpecId::ECOTONE, true),
                    (OpSpecId::FJORD, true),
                ],
            ),
            (
                OpSpecId::JOVIAN,
                vec![
                    (SpecId::PRAGUE, true),
                    (SpecId::SHANGHAI, true),
                    (SpecId::CANCUN, true),
                    (SpecId::MERGE, true),
                ],
                vec![
                    (OpSpecId::BEDROCK, true),
                    (OpSpecId::REGOLITH, true),
                    (OpSpecId::CANYON, true),
                    (OpSpecId::ECOTONE, true),
                    (OpSpecId::FJORD, true),
                    (OpSpecId::HOLOCENE, true),
                    (OpSpecId::ISTHMUS, true),
                ],
            ),
        ];

        for (op_spec, eth_tests, op_tests) in test_cases {
            // Test ETH spec compatibility
            for (eth_spec, expected) in eth_tests {
                assert_eq!(
                    op_spec.into_eth_spec().is_enabled_in(eth_spec),
                    expected,
                    "{:?} should {} be enabled in ETH {:?}",
                    op_spec,
                    if expected { "" } else { "not " },
                    eth_spec
                );
            }

            // Test OP spec compatibility
            for (other_op_spec, expected) in op_tests {
                assert_eq!(
                    op_spec.is_enabled_in(other_op_spec),
                    expected,
                    "{:?} should {} be enabled in OP {:?}",
                    op_spec,
                    if expected { "" } else { "not " },
                    other_op_spec
                );
            }
        }
    }

    #[test]
    fn default_op_spec_id() {
        assert_eq!(OpSpecId::default(), OpSpecId::JOVIAN);
    }
}
