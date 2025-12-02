//! Contains the hardfork configuration for the chain.

use alloc::string::{String, ToString};
use core::fmt::Display;

/// Hardfork configuration.
///
/// See: <https://github.com/ethereum-optimism/superchain-registry/blob/8ff62ada16e14dd59d0fb94ffb47761c7fa96e01/ops/internal/config/chain.go#L102-L110>
#[derive(Debug, Copy, Clone, Default, Hash, Eq, PartialEq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(deny_unknown_fields))]
pub struct HardForkConfig {
    /// `regolith_time` sets the activation time of the Regolith network-upgrade:
    /// a pre-mainnet Bedrock change that addresses findings of the Sherlock contest related to
    /// deposit attributes. "Regolith" is the loose deposited rock that sits on top of Bedrock.
    /// Active if regolith_time != None && L2 block timestamp >= Some(regolith_time), inactive
    /// otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub regolith_time: Option<u64>,
    /// `canyon_time` sets the activation time of the Canyon network upgrade.
    /// Active if `canyon_time` != None && L2 block timestamp >= Some(canyon_time), inactive
    /// otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub canyon_time: Option<u64>,
    /// `delta_time` sets the activation time of the Delta network upgrade.
    /// Active if `delta_time` != None && L2 block timestamp >= Some(delta_time), inactive
    /// otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub delta_time: Option<u64>,
    /// `ecotone_time` sets the activation time of the Ecotone network upgrade.
    /// Active if `ecotone_time` != None && L2 block timestamp >= Some(ecotone_time), inactive
    /// otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub ecotone_time: Option<u64>,
    /// `fjord_time` sets the activation time of the Fjord network upgrade.
    /// Active if `fjord_time` != None && L2 block timestamp >= Some(fjord_time), inactive
    /// otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub fjord_time: Option<u64>,
    /// `granite_time` sets the activation time for the Granite network upgrade.
    /// Active if `granite_time` != None && L2 block timestamp >= Some(granite_time), inactive
    /// otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub granite_time: Option<u64>,
    /// `holocene_time` sets the activation time for the Holocene network upgrade.
    /// Active if `holocene_time` != None && L2 block timestamp >= Some(holocene_time), inactive
    /// otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub holocene_time: Option<u64>,
    /// `pectra_blob_schedule_time` sets the activation time for the activation of the Pectra blob
    /// fee schedule for the L1 block info transaction. This is an optional fork, only present
    /// on OP Stack sepolia chains that observed the L1 Pectra network upgrade with `op-node`
    /// <=v1.11.1 sequencing the network.
    ///
    /// Active if `pectra_blob_schedule_time` != None && L2 block timestamp >=
    /// Some(pectra_blob_schedule_time), inactive otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub pectra_blob_schedule_time: Option<u64>,
    /// `isthmus_time` sets the activation time for the Isthmus network upgrade.
    /// Active if `isthmus_time` != None && L2 block timestamp >= Some(isthmus_time), inactive
    /// otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub isthmus_time: Option<u64>,
    /// `jovian_time` sets the activation time for the Jovian network upgrade.
    /// Active if `jovian_time` != None && L2 block timestamp >= Some(jovian_time), inactive
    /// otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub jovian_time: Option<u64>,
    /// `interop_time` sets the activation time for the Interop network upgrade.
    /// Active if `interop_time` != None && L2 block timestamp >= Some(interop_time), inactive
    /// otherwise.
    #[cfg_attr(feature = "serde", serde(skip_serializing_if = "Option::is_none"))]
    pub interop_time: Option<u64>,
}

impl Display for HardForkConfig {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        #[inline(always)]
        fn fmt_time(t: Option<u64>) -> String {
            t.map(|t| t.to_string()).unwrap_or_else(|| "Not scheduled".to_string())
        }

        writeln!(f, "🍴 Scheduled Hardforks:")?;
        for (name, time) in self.iter() {
            writeln!(f, "-> {} Activation Time: {}", name, fmt_time(time))?;
        }
        Ok(())
    }
}

impl HardForkConfig {
    /// Returns an iterator of hardfork names -> their activation times (if scheduled.)
    pub fn iter(&self) -> impl Iterator<Item = (&'static str, Option<u64>)> {
        [
            ("Regolith", self.regolith_time),
            ("Canyon", self.canyon_time),
            ("Delta", self.delta_time),
            ("Ecotone", self.ecotone_time),
            ("Fjord", self.fjord_time),
            ("Granite", self.granite_time),
            ("Holocene", self.holocene_time),
            ("Pectra Blob Schedule", self.pectra_blob_schedule_time),
            ("Isthmus", self.isthmus_time),
            ("Jovian", self.jovian_time),
            ("Interop", self.interop_time),
        ]
        .into_iter()
    }
}

#[cfg(test)]
#[cfg(feature = "serde")]
mod tests {
    use super::*;

    #[test]
    fn test_hardforks_deserialize_json() {
        let raw: &str = r#"
        {
            "canyon_time": 1699981200,
            "delta_time": 1703203200,
            "ecotone_time": 1708534800,
            "fjord_time": 1716998400,
            "granite_time": 1723478400,
            "holocene_time":1732633200
        }
        "#;

        let hardforks = HardForkConfig {
            regolith_time: None,
            canyon_time: Some(1699981200),
            delta_time: Some(1703203200),
            ecotone_time: Some(1708534800),
            fjord_time: Some(1716998400),
            granite_time: Some(1723478400),
            holocene_time: Some(1732633200),
            pectra_blob_schedule_time: None,
            isthmus_time: None,
            jovian_time: None,
            interop_time: None,
        };

        let deserialized: HardForkConfig = serde_json::from_str(raw).unwrap();
        assert_eq!(hardforks, deserialized);
    }

    #[test]
    fn test_hardforks_deserialize_new_field_fail_json() {
        let raw: &str = r#"
        {
            "canyon_time": 1704992401,
            "delta_time": 1708560000,
            "ecotone_time": 1710374401,
            "fjord_time": 1720627201,
            "granite_time": 1726070401,
            "holocene_time": 1736445601,
            "new_field": 0
        }
        "#;

        let err = serde_json::from_str::<HardForkConfig>(raw).unwrap_err();
        assert_eq!(err.classify(), serde_json::error::Category::Data);
    }

    #[test]
    fn test_hardforks_deserialize_toml() {
        let raw: &str = r#"
        canyon_time =  1699981200 # Tue 14 Nov 2023 17:00:00 UTC
        delta_time =   1703203200 # Fri 22 Dec 2023 00:00:00 UTC
        ecotone_time = 1708534800 # Wed 21 Feb 2024 17:00:00 UTC
        fjord_time =   1716998400 # Wed 29 May 2024 16:00:00 UTC
        granite_time = 1723478400 # Mon Aug 12 16:00:00 UTC 2024
        holocene_time = 1732633200 # Tue Nov 26 15:00:00 UTC 2024
        "#;

        let hardforks = HardForkConfig {
            regolith_time: None,
            canyon_time: Some(1699981200),
            delta_time: Some(1703203200),
            ecotone_time: Some(1708534800),
            fjord_time: Some(1716998400),
            granite_time: Some(1723478400),
            holocene_time: Some(1732633200),
            pectra_blob_schedule_time: None,
            isthmus_time: None,
            jovian_time: None,
            interop_time: None,
        };

        let deserialized: HardForkConfig = toml::from_str(raw).unwrap();
        assert_eq!(hardforks, deserialized);
    }

    #[test]
    fn test_hardforks_deserialize_new_field_fail_toml() {
        let raw: &str = r#"
        canyon_time =  1699981200 # Tue 14 Nov 2023 17:00:00 UTC
        delta_time =   1703203200 # Fri 22 Dec 2023 00:00:00 UTC
        ecotone_time = 1708534800 # Wed 21 Feb 2024 17:00:00 UTC
        fjord_time =   1716998400 # Wed 29 May 2024 16:00:00 UTC
        granite_time = 1723478400 # Mon Aug 12 16:00:00 UTC 2024
        holocene_time = 1732633200 # Tue Nov 26 15:00:00 UTC 2024
        new_field_time = 1732633200 # Tue Nov 26 15:00:00 UTC 2024
        "#;
        toml::from_str::<HardForkConfig>(raw).unwrap_err();
    }

    #[test]
    fn test_hardforks_iter() {
        let hardforks = HardForkConfig {
            regolith_time: Some(1),
            canyon_time: Some(2),
            delta_time: Some(3),
            ecotone_time: Some(4),
            fjord_time: Some(5),
            granite_time: Some(6),
            holocene_time: Some(7),
            pectra_blob_schedule_time: Some(8),
            isthmus_time: Some(9),
            jovian_time: Some(10),
            interop_time: Some(11),
        };

        let mut iter = hardforks.iter();
        assert_eq!(iter.next(), Some(("Regolith", Some(1))));
        assert_eq!(iter.next(), Some(("Canyon", Some(2))));
        assert_eq!(iter.next(), Some(("Delta", Some(3))));
        assert_eq!(iter.next(), Some(("Ecotone", Some(4))));
        assert_eq!(iter.next(), Some(("Fjord", Some(5))));
        assert_eq!(iter.next(), Some(("Granite", Some(6))));
        assert_eq!(iter.next(), Some(("Holocene", Some(7))));
        assert_eq!(iter.next(), Some(("Pectra Blob Schedule", Some(8))));
        assert_eq!(iter.next(), Some(("Isthmus", Some(9))));
        assert_eq!(iter.next(), Some(("Jovian", Some(10))));
        assert_eq!(iter.next(), Some(("Interop", Some(11))));
        assert_eq!(iter.next(), None);
    }
}
