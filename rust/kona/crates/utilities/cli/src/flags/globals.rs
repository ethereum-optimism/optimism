//! Global arguments for the CLI.

use alloy_chains::Chain;
use alloy_primitives::Address;
use clap::Parser;
use kona_genesis::RollupConfig;
use kona_registry::OPCHAINS;

use crate::{CliError, CliResult, LogArgs, MetricsArgs, OverrideArgs};

/// Global arguments for the CLI.
#[derive(Parser, Default, Clone, Debug)]
pub struct GlobalArgs {
    /// Logging arguments.
    #[command(flatten)]
    pub log_args: LogArgs,
    /// The L2 chain ID to use.
    #[arg(
        long = "chain",
        alias = "l2-chain-id",
        short = 'c',
        global = true,
        default_value = "10",
        env = "KONA_L2_CHAIN_ID",
        help = "The L2 chain ID to use"
    )]
    pub l2_chain_id: Chain,
    /// Embed the override flags globally to provide override values adjacent to the configs.
    #[command(flatten)]
    pub override_args: OverrideArgs,
    /// Prometheus CLI arguments.
    #[command(flatten)]
    pub metrics: MetricsArgs,
}

impl GlobalArgs {
    /// Applies the specified overrides to the given rollup config.
    ///
    /// Transforms the rollup config and returns the updated config with the overrides applied.
    pub fn apply_overrides(&self, config: RollupConfig) -> RollupConfig {
        self.override_args.apply(config)
    }

    /// Returns the signer [`Address`] from the rollup config for the given l2 chain id.
    pub fn genesis_signer(&self) -> CliResult<Address> {
        let id = self.l2_chain_id;
        OPCHAINS
            .get(&id.id())
            .ok_or(CliError::ChainConfigNotFound(id.id()))?
            .roles
            .as_ref()
            .ok_or(CliError::RolesNotFound(id.id()))?
            .unsafe_block_signer
            .ok_or(CliError::UnsafeBlockSignerNotFound(id.id()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::Parser;
    use rstest::rstest;

    #[rstest]
    #[case::numeric_optimism("10", 10)]
    #[case::numeric_ethereum("1", 1)]
    #[case::numeric_base("8453", 8453)]
    #[case::numeric_unknown("999999", 999999)]
    #[case::string_optimism("optimism", 10)]
    #[case::string_mainnet("mainnet", 1)]
    #[case::string_base("base", 8453)]
    fn test_l2_chain_id_parse_valid(#[case] value: &str, #[case] expected_id: u64) {
        let args = GlobalArgs::try_parse_from(["test", "--l2-chain-id", value]).unwrap();
        assert_eq!(args.l2_chain_id.id(), expected_id);
    }

    #[rstest]
    #[case::invalid_string("invalid_chain")]
    fn test_l2_chain_id_parse_invalid(#[case] invalid_value: &str) {
        let result = GlobalArgs::try_parse_from(["test", "--l2-chain-id", invalid_value]);
        assert!(result.is_err());

        // The error should be related to parsing
        let err = result.unwrap_err();
        assert!(err.to_string().to_lowercase().contains("invalid"));
    }

    #[rstest]
    #[case::numeric("10", 10)]
    #[case::string("optimism", 10)]
    fn test_l2_chain_id_short_flag(#[case] value: &str, #[case] expected_id: u64) {
        let args = GlobalArgs::try_parse_from(["test", "-c", value]).unwrap();
        assert_eq!(args.l2_chain_id.id(), expected_id);
    }

    #[rstest]
    #[case::numeric("10", 10)]
    #[case::string("optimism", 10)]
    fn test_l2_chain_id_env_var(#[case] env_value: &str, #[case] expected_id: u64) {
        unsafe {
            std::env::set_var("KONA_NODE_L2_CHAIN_ID", env_value);
        }
        let args = GlobalArgs::try_parse_from(["test"]).unwrap();
        assert_eq!(args.l2_chain_id.id(), expected_id);
        unsafe {
            std::env::remove_var("KONA_NODE_L2_CHAIN_ID");
        }
    }

    #[test]
    fn test_l2_chain_id_default() {
        // Test that the default value is chain ID 10 (Optimism)
        let args = GlobalArgs::try_parse_from(["test"]).unwrap();
        assert_eq!(args.l2_chain_id.id(), 10);
    }
}
