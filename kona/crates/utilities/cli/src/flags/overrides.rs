//! Flags that allow overriding derived values.

use clap::Parser;
use kona_genesis::RollupConfig;

/// Override Flags.
#[derive(Parser, Debug, Clone, Copy, PartialEq, Eq)]
pub struct OverrideArgs {
    /// Manually specify the timestamp for the Canyon fork, overriding the bundled setting.
    #[arg(long, env = "KONA_OVERRIDE_CANYON")]
    pub canyon_override: Option<u64>,
    /// Manually specify the timestamp for the Delta fork, overriding the bundled setting.
    #[arg(long, env = "KONA_OVERRIDE_DELTA")]
    pub delta_override: Option<u64>,
    /// Manually specify the timestamp for the Ecotone fork, overriding the bundled setting.
    #[arg(long, env = "KONA_OVERRIDE_ECOTONE")]
    pub ecotone_override: Option<u64>,
    /// Manually specify the timestamp for the Fjord fork, overriding the bundled setting.
    #[arg(long, env = "KONA_OVERRIDE_FJORD")]
    pub fjord_override: Option<u64>,
    /// Manually specify the timestamp for the Granite fork, overriding the bundled setting.
    #[arg(long, env = "KONA_OVERRIDE_GRANITE")]
    pub granite_override: Option<u64>,
    /// Manually specify the timestamp for the Holocene fork, overriding the bundled setting.
    #[arg(long, env = "KONA_OVERRIDE_HOLOCENE")]
    pub holocene_override: Option<u64>,
    /// Manually specify the timestamp for the Isthmus fork, overriding the bundled setting.
    #[arg(long, env = "KONA_OVERRIDE_ISTHMUS")]
    pub isthmus_override: Option<u64>,
    /// Manually specify the timestamp for the Jovian fork, overriding the bundled setting.
    #[arg(long, env = "KONA_OVERRIDE_JOVIAN")]
    pub jovian_override: Option<u64>,
    /// Manually specify the timestamp for the pectra blob schedule, overriding the bundled
    /// setting.
    #[arg(long, env = "KONA_OVERRIDE_PECTRA_BLOB_SCHEDULE")]
    pub pectra_blob_schedule_override: Option<u64>,
    /// Manually specify the timestamp for the Interop fork, overriding the bundled setting.
    #[arg(long, env = "KONA_OVERRIDE_INTEROP")]
    pub interop_override: Option<u64>,
}

impl Default for OverrideArgs {
    fn default() -> Self {
        // Construct default values using the clap parser.
        // This works since none of the cli flags are required.
        Self::parse_from::<[_; 0], &str>([])
    }
}

impl OverrideArgs {
    /// Applies the override args to the given rollup config.
    pub fn apply(&self, config: RollupConfig) -> RollupConfig {
        let hardforks = kona_genesis::HardForkConfig {
            regolith_time: config.hardforks.regolith_time,
            canyon_time: self.canyon_override.map(Some).unwrap_or(config.hardforks.canyon_time),
            delta_time: self.delta_override.map(Some).unwrap_or(config.hardforks.delta_time),
            ecotone_time: self.ecotone_override.map(Some).unwrap_or(config.hardforks.ecotone_time),
            fjord_time: self.fjord_override.map(Some).unwrap_or(config.hardforks.fjord_time),
            granite_time: self.granite_override.map(Some).unwrap_or(config.hardforks.granite_time),
            holocene_time: self
                .holocene_override
                .map(Some)
                .unwrap_or(config.hardforks.holocene_time),
            pectra_blob_schedule_time: self
                .pectra_blob_schedule_override
                .map(Some)
                .unwrap_or(config.hardforks.pectra_blob_schedule_time),
            isthmus_time: self.isthmus_override.map(Some).unwrap_or(config.hardforks.isthmus_time),
            jovian_time: self.jovian_override.map(Some).unwrap_or(config.hardforks.jovian_time),
            interop_time: self.interop_override.map(Some).unwrap_or(config.hardforks.interop_time),
        };
        RollupConfig { hardforks, ..config }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// A mock command that uses the override args.
    #[derive(Parser, Debug, Clone)]
    #[command(about = "Mock command")]
    struct MockCommand {
        /// Override flags.
        #[clap(flatten)]
        pub override_flags: OverrideArgs,
    }

    #[test]
    fn test_apply_overrides() {
        let args = MockCommand::parse_from([
            "test",
            "--canyon-override",
            "1699981200",
            "--delta-override",
            "1703203200",
            "--ecotone-override",
            "1708534800",
            "--fjord-override",
            "1716998400",
            "--granite-override",
            "1723478400",
            "--holocene-override",
            "1732633200",
            "--pectra-blob-schedule-override",
            "1745000000",
            "--isthmus-override",
            "1740000000",
            "--jovian-override",
            "1745000001",
            "--interop-override",
            "1750000000",
        ]);
        let config = RollupConfig::default();
        let updated_config = args.override_flags.apply(config);
        assert_eq!(
            updated_config.hardforks,
            kona_genesis::HardForkConfig {
                regolith_time: Default::default(),
                canyon_time: Some(1699981200),
                delta_time: Some(1703203200),
                ecotone_time: Some(1708534800),
                fjord_time: Some(1716998400),
                granite_time: Some(1723478400),
                holocene_time: Some(1732633200),
                pectra_blob_schedule_time: Some(1745000000),
                isthmus_time: Some(1740000000),
                jovian_time: Some(1745000001),
                interop_time: Some(1750000000),
            }
        );
    }

    #[test]
    fn test_apply_default_overrides() {
        // Use OP Mainnet rollup config.
        let config = kona_registry::ROLLUP_CONFIGS
            .get(&10)
            .expect("No config found for chain ID 10")
            .clone();
        let init_forks = config.hardforks;
        let args = MockCommand::parse_from(["test"]);
        let updated_config = args.override_flags.apply(config);
        assert_eq!(updated_config.hardforks, init_forks);
    }

    #[test]
    fn test_default_override_flags() {
        let args = MockCommand::parse_from(["test"]);
        assert_eq!(
            args.override_flags,
            OverrideArgs {
                canyon_override: None,
                delta_override: None,
                ecotone_override: None,
                fjord_override: None,
                granite_override: None,
                holocene_override: None,
                pectra_blob_schedule_override: None,
                isthmus_override: None,
                jovian_override: None,
                interop_override: None,
            }
        );
        // Sanity check that the default impl matches the expected default values.
        assert_eq!(args.override_flags, OverrideArgs::default());
    }
}
