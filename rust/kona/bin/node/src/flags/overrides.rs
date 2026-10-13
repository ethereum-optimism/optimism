//! Flags that allow overriding derived values.

use clap::Parser;
use kona_genesis::RollupConfig;

/// Flags that override the rollup config's hardfork schedule.
#[derive(Parser, Debug, Clone, Copy, PartialEq, Eq)]
#[command(next_help_heading = "Hardfork Overrides")]
pub struct OverrideArgs {
    /// Manually specify the timestamp for the Canyon fork, overriding the bundled setting.
    #[arg(
        long = "chain.hardfork.canyon",
        env = "KONA_NODE_CHAIN_HARDFORK_CANYON",
        value_name = "TIMESTAMP"
    )]
    pub canyon: Option<u64>,
    /// Manually specify the timestamp for the Delta fork, overriding the bundled setting.
    #[arg(
        long = "chain.hardfork.delta",
        env = "KONA_NODE_CHAIN_HARDFORK_DELTA",
        value_name = "TIMESTAMP"
    )]
    pub delta: Option<u64>,
    /// Manually specify the timestamp for the Ecotone fork, overriding the bundled setting.
    #[arg(
        long = "chain.hardfork.ecotone",
        env = "KONA_NODE_CHAIN_HARDFORK_ECOTONE",
        value_name = "TIMESTAMP"
    )]
    pub ecotone: Option<u64>,
    /// Manually specify the timestamp for the Fjord fork, overriding the bundled setting.
    #[arg(
        long = "chain.hardfork.fjord",
        env = "KONA_NODE_CHAIN_HARDFORK_FJORD",
        value_name = "TIMESTAMP"
    )]
    pub fjord: Option<u64>,
    /// Manually specify the timestamp for the Granite fork, overriding the bundled setting.
    #[arg(
        long = "chain.hardfork.granite",
        env = "KONA_NODE_CHAIN_HARDFORK_GRANITE",
        value_name = "TIMESTAMP"
    )]
    pub granite: Option<u64>,
    /// Manually specify the timestamp for the Holocene fork, overriding the bundled setting.
    #[arg(
        long = "chain.hardfork.holocene",
        env = "KONA_NODE_CHAIN_HARDFORK_HOLOCENE",
        value_name = "TIMESTAMP"
    )]
    pub holocene: Option<u64>,
    /// Manually specify the timestamp for the Isthmus fork, overriding the bundled setting.
    #[arg(
        long = "chain.hardfork.isthmus",
        env = "KONA_NODE_CHAIN_HARDFORK_ISTHMUS",
        value_name = "TIMESTAMP"
    )]
    pub isthmus: Option<u64>,
    /// Manually specify the timestamp for the Jovian fork, overriding the bundled setting.
    #[arg(
        long = "chain.hardfork.jovian",
        env = "KONA_NODE_CHAIN_HARDFORK_JOVIAN",
        value_name = "TIMESTAMP"
    )]
    pub jovian: Option<u64>,
    /// Manually specify the timestamp for the Karst fork, overriding the bundled setting.
    #[arg(
        long = "chain.hardfork.karst",
        env = "KONA_NODE_CHAIN_HARDFORK_KARST",
        value_name = "TIMESTAMP"
    )]
    pub karst: Option<u64>,
    /// Manually specify the timestamp for the pectra blob schedule, overriding the bundled
    /// setting.
    #[arg(
        long = "chain.hardfork.pectra-blob-schedule",
        env = "KONA_NODE_CHAIN_HARDFORK_PECTRA_BLOB_SCHEDULE",
        value_name = "TIMESTAMP"
    )]
    pub pectra_blob_schedule: Option<u64>,
    /// Manually specify the timestamp for the Lagoon fork, overriding the bundled setting.
    #[arg(
        long = "chain.hardfork.lagoon",
        env = "KONA_NODE_CHAIN_HARDFORK_LAGOON",
        value_name = "TIMESTAMP"
    )]
    pub lagoon: Option<u64>,
    /// Manually set `keep_karst_upgrade_gas`, overriding the bundled setting. When true, the Karst
    /// activation block's one-time upgrade gas is kept on every later block (for chains that
    /// activated Karst with the leak baked into their history).
    #[arg(
        long = "chain.keep-karst-upgrade-gas",
        env = "KONA_NODE_CHAIN_KEEP_KARST_UPGRADE_GAS",
        value_name = "BOOL"
    )]
    pub keep_karst_upgrade_gas: Option<bool>,
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
            canyon_time: self.canyon.map(Some).unwrap_or(config.hardforks.canyon_time),
            delta_time: self.delta.map(Some).unwrap_or(config.hardforks.delta_time),
            ecotone_time: self.ecotone.map(Some).unwrap_or(config.hardforks.ecotone_time),
            fjord_time: self.fjord.map(Some).unwrap_or(config.hardforks.fjord_time),
            granite_time: self.granite.map(Some).unwrap_or(config.hardforks.granite_time),
            holocene_time: self.holocene.map(Some).unwrap_or(config.hardforks.holocene_time),
            pectra_blob_schedule_time: self
                .pectra_blob_schedule
                .map(Some)
                .unwrap_or(config.hardforks.pectra_blob_schedule_time),
            isthmus_time: self.isthmus.map(Some).unwrap_or(config.hardforks.isthmus_time),
            jovian_time: self.jovian.map(Some).unwrap_or(config.hardforks.jovian_time),
            karst_time: self.karst.map(Some).unwrap_or(config.hardforks.karst_time),
            keep_karst_upgrade_gas: self
                .keep_karst_upgrade_gas
                .unwrap_or(config.hardforks.keep_karst_upgrade_gas),
            lagoon_time: self.lagoon.map(Some).unwrap_or(config.hardforks.lagoon_time),
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
            "--chain.hardfork.canyon",
            "1699981200",
            "--chain.hardfork.delta",
            "1703203200",
            "--chain.hardfork.ecotone",
            "1708534800",
            "--chain.hardfork.fjord",
            "1716998400",
            "--chain.hardfork.granite",
            "1723478400",
            "--chain.hardfork.holocene",
            "1732633200",
            "--chain.hardfork.pectra-blob-schedule",
            "1745000000",
            "--chain.hardfork.isthmus",
            "1740000000",
            "--chain.hardfork.jovian",
            "1745000001",
            "--chain.hardfork.karst",
            "1750000000",
            "--chain.hardfork.lagoon",
            "1755000000",
            "--chain.keep-karst-upgrade-gas",
            "true",
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
                karst_time: Some(1750000000),
                keep_karst_upgrade_gas: true,
                lagoon_time: Some(1755000000),
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
                canyon: None,
                delta: None,
                ecotone: None,
                fjord: None,
                granite: None,
                holocene: None,
                pectra_blob_schedule: None,
                isthmus: None,
                jovian: None,
                karst: None,
                lagoon: None,
                keep_karst_upgrade_gas: None,
            }
        );
        // Sanity check that the default impl matches the expected default values.
        assert_eq!(args.override_flags, OverrideArgs::default());
    }
}
