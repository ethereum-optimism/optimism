//! Rust Optimism (op-reth) binary executable.
//!
//! [`main_with`] runs op-reth. A binary that embeds op-reth calls it with [`OpRethOverrides`] to
//! add CLI arguments, report its own version, and change the node before it launches.
//!
//! ## Feature Flags
//!
//! - `jemalloc`: Uses [jemallocator](https://github.com/tikv/jemallocator) as the global allocator.
//!   This is **not recommended on Windows**. See [here](https://rust-lang.github.io/rfcs/1974-global-allocators.html#jemalloc)
//!   for more info.
//! - `jemalloc-prof`: Enables [jemallocator's](https://github.com/tikv/jemallocator) heap profiling
//!   and leak detection functionality. See [jemalloc's opt.prof](https://jemalloc.net/jemalloc.3.html#opt.prof)
//!   documentation for usage details. This is **not recommended on Windows**. See [here](https://rust-lang.github.io/rfcs/1974-global-allocators.html#jemalloc)
//!   for more info.
//! - `asm-keccak`: replaces the default, pure-Rust implementation of Keccak256 with one implemented
//!   in assembly; see [the `keccak-asm` crate](https://github.com/DaniPopes/keccak-asm) for more
//!   details and supported targets
//! - `min-error-logs`: Disables all logs below `error` level.
//! - `min-warn-logs`: Disables all logs below `warn` level.
//! - `min-info-logs`: Disables all logs below `info` level. This can speed up the node, since fewer
//!   calls to the logging component are made.
//! - `min-debug-logs`: Disables all logs below `debug` level.
//! - `min-trace-logs`: Disables all logs below `trace` level.
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod overrides;
mod rollup_consensus;
pub use overrides::{OpRethArgs, OpRethOverrides};

use reth_node_core::{
    node_config::NodeConfig,
    version::{RethCliVersionConsts, try_init_version_metadata},
};
use reth_optimism_chainspec::{OpChainSpec, is_superchain_chain_id, superchain_block_time};
use reth_optimism_node::{OpLaunchHooks, OpNode, args::RollupArgs, launch_node};
use rollup_consensus::RollupConsensusArgs;
use std::{borrow::Cow, fmt};
use tracing::warn;

/// Re-exported from `reth_optimism_cli`.
pub mod cli {
    pub use reth_optimism_cli::*;
}

/// Re-exported from `reth_optimism_chainspec`.
pub mod chainspec {
    pub use reth_optimism_chainspec::*;
}

/// Re-exported from `reth_optimism_consensus`.
pub mod consensus {
    pub use reth_optimism_consensus::*;
}

/// Re-exported from `reth_optimism_evm`.
pub mod evm {
    pub use reth_optimism_evm::*;
}

/// Re-exported from `reth_optimism_forks`.
pub mod forks {
    pub use reth_optimism_forks::*;
}

/// Re-exported from `reth_optimism_node`.
pub mod node {
    pub use reth_optimism_node::*;
}

/// Re-exported from `reth_optimism_payload_builder`.
pub mod payload {
    pub use reth_optimism_payload_builder::*;
}

/// Re-exported from `reth_optimism_primitives`.
pub mod primitives {
    pub use reth_optimism_primitives::*;
}

/// Re-exported from `reth_optimism_rpc`.
pub mod rpc {
    pub use reth_optimism_rpc::*;
}

/// The name op-reth reports wherever its client identity is observable.
const CLIENT_NAME: &str = "op-reth";

#[derive(Debug, clap::Args)]
struct LaunchArgs<Ext: clap::Args> {
    #[command(flatten)]
    args: OpRethArgs<Ext>,
    #[command(flatten)]
    consensus: RollupConsensusArgs,
}

/// Runs op-reth.
pub fn main() -> eyre::Result<()> {
    main_with(OpRethOverrides::default())
}

/// Runs op-reth with the `overrides` of a binary that embeds it.
///
/// Parses the command line, with the extra arguments `Ext`, and runs the selected command. The
/// `node` command applies op-reth's checks and defaults to its arguments, builds an [`OpNode`],
/// and hands it to [`OpRethOverrides::map_node`] before launching it.
///
/// The global allocator is the embedding binary's: op-reth declares jemalloc in its own `main.rs`,
/// not here.
pub fn main_with<Ext>(overrides: OpRethOverrides<Ext>) -> eyre::Result<()>
where
    Ext: clap::Args + fmt::Debug + 'static,
{
    reth_cli_util::sigsegv_handler::install();

    // Enable backtraces unless a RUST_BACKTRACE value has already been explicitly provided.
    if std::env::var_os("RUST_BACKTRACE").is_none() {
        unsafe {
            std::env::set_var("RUST_BACKTRACE", "1");
        }
    }

    let OpRethOverrides { version, node_mapper } = overrides;
    // Install the build metadata before clap reads it during parsing.
    try_init_version_metadata(version.unwrap_or_else(op_reth_version))
        .map_err(|_| eyre::eyre!("build info is already embedded. This is a bug."))?;

    reth_optimism_cli::Cli::<LaunchArgs<Ext>>::parse_with_denied_args().run(
        async move |mut builder, LaunchArgs { args: OpRethArgs { rollup, ext }, consensus }| {
            let node = op_node(builder.config().chain.chain.id(), rollup)?;
            let configure_consensus = |config: &NodeConfig<OpChainSpec>, hooks: OpLaunchHooks| {
                if consensus.enabled() {
                    let chain = config.chain.chain;
                    let metrics = config.metrics.prometheus.is_some();
                    hooks.with_consensus(Box::pin(consensus.run(chain, metrics)))
                } else {
                    hooks
                }
            };
            if let Some(mapper) = node_mapper {
                let (node, hooks) = mapper(builder.config_mut(), ext, node).await?;
                let hooks = configure_consensus(builder.config(), hooks);
                node.launch(builder, hooks).await
            } else {
                // Keep the stock launch compiled in reth-optimism-node.
                let hooks = configure_consensus(builder.config(), OpLaunchHooks::default());
                launch_node(builder, node, hooks).await
            }
        },
    )
}

/// The [`OpNode`] for `rollup` on the chain with ID `chain_id`, the resolved chain ID rather than
/// how the chain was selected.
fn op_node(chain_id: u64, mut rollup: RollupArgs) -> eyre::Result<OpNode> {
    if rollup.testing_sdm_fixed_policy.is_some() {
        validate_testing_sdm_fixed_policy(chain_id)?;
    }
    // A custom genesis of a registry chain gets the registry's block time too.
    rollup.block_time = rollup.block_time.or_else(|| superchain_block_time(chain_id));
    Ok(OpNode::new(rollup))
}

fn validate_testing_sdm_fixed_policy(chain_id: u64) -> eyre::Result<()> {
    // Check the resolved chain ID, not how the chain was selected: a custom genesis can
    // still configure a registered chain. Unregistered chains are not necessarily dev chains.
    if is_superchain_chain_id(chain_id) {
        eyre::bail!(
            "--testing.sdm-fixed-policy is not allowed for superchain registry chain ID {chain_id}"
        );
    }
    warn!(
        target: "reth::cli",
        "TEST-ONLY SDM fixed-refund policy enabled; never use this policy in production"
    );
    Ok(())
}

/// op-reth's own build metadata.
fn op_reth_version() -> RethCliVersionConsts {
    let info = op_version::build_info!();
    RethCliVersionConsts {
        name_client: Cow::Borrowed(CLIENT_NAME),
        cargo_pkg_version: Cow::Owned(info.version().to_string()),
        vergen_git_sha_long: Cow::Owned(info.commit_sha().to_string()),
        vergen_git_sha: Cow::Owned(info.short_sha().to_string()),
        vergen_build_timestamp: Cow::Owned(info.build_timestamp().to_string()),
        vergen_cargo_target_triple: Cow::Owned(info.target_triple().to_string()),
        vergen_cargo_features: Cow::Owned(info.cargo_features().to_string()),
        short_version: Cow::Owned(info.short_version()),
        long_version: Cow::Owned(info.long_version()),
        build_profile_name: Cow::Owned(info.build_profile().to_string()),
        p2p_client_version: Cow::Owned(format!(
            "{CLIENT_NAME}/v{}-{}/{}",
            info.version(),
            info.short_sha(),
            info.target_triple()
        )),
        extra_data: Cow::Owned(String::new()), // Governed by the protocol on L2.
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_cli_commands::node::NoArgs;
    use reth_optimism_cli::{Cli, commands::Commands};
    use std::io::Write;

    const TARGET: &str = "0x0000000000000000000000000000000000000001";
    const POLICY_ARGS: [&[&str]; 2] =
        [&["--testing.sdm-fixed-policy"], &["--testing.sdm-fixed-policy", TARGET]];

    fn validate_cli(chain: &str, policy_args: &[&str]) -> eyre::Result<()> {
        let cli = Cli::<LaunchArgs<NoArgs>>::try_parse_with_denied_args_from(
            ["op-reth", "node", "--chain", chain].into_iter().chain(policy_args.iter().copied()),
        )?;
        let Commands::Node(node) = cli.command else {
            panic!("expected node command");
        };
        if node.ext.args.rollup.testing_sdm_fixed_policy.is_some() {
            validate_testing_sdm_fixed_policy(node.chain.chain.id())?;
        }
        Ok(())
    }

    fn custom_genesis(chain_id: u64) -> String {
        format!(
            r#"{{"config":{{"chainId":{chain_id}}},"gasLimit":"0x1c9c380","difficulty":"0x0","alloc":{{}}}}"#
        )
    }

    #[test]
    fn cli_is_valid() {
        use clap::CommandFactory;
        Cli::<LaunchArgs<NoArgs>>::command().debug_assert();
    }

    /// Stock op-reth keeps the help text pinned by the `Cli<RollupArgs>` snapshot.
    #[test]
    fn op_reth_args_keep_the_rollup_args_command_line() {
        fn node_help<Ext: clap::Args + std::fmt::Debug>() -> String {
            let mut cmd = Cli::<Ext>::command_with_denied_args_hidden();
            cmd.build();
            cmd.find_subcommand_mut("node").unwrap().render_long_help().to_string()
        }
        assert_eq!(node_help::<OpRethArgs>(), node_help::<RollupArgs>());
        assert_eq!(node_help::<LaunchArgs<NoArgs>>(), node_help::<RollupArgs>());
    }

    #[test]
    fn rejects_registered_chains_with_either_policy_form() {
        for chain in ["op-mainnet", "optimism-sepolia", "unichain", "unichain-sepolia"] {
            for policy_args in POLICY_ARGS {
                let err = validate_cli(chain, policy_args).unwrap_err();
                assert!(
                    err.to_string().contains("is not allowed for superchain registry chain ID")
                );
            }
        }
    }

    #[test]
    fn rejects_registered_chain_ids_in_custom_genesis() {
        for chain_id in [10, 130, 11155420, 1301] {
            for policy_args in POLICY_ARGS {
                let err = validate_cli(&custom_genesis(chain_id), policy_args).unwrap_err();
                assert_eq!(
                    err.to_string(),
                    format!(
                        "--testing.sdm-fixed-policy is not allowed for superchain registry chain ID {chain_id}"
                    )
                );
            }
        }
    }

    #[test]
    fn rejects_registered_chain_id_in_genesis_file() {
        let mut genesis = tempfile::NamedTempFile::new().unwrap();
        write!(genesis, "{}", custom_genesis(10)).unwrap();
        for policy_args in POLICY_ARGS {
            let err = validate_cli(genesis.path().to_str().unwrap(), policy_args).unwrap_err();
            assert_eq!(
                err.to_string(),
                "--testing.sdm-fixed-policy is not allowed for superchain registry chain ID 10"
            );
        }
    }

    #[test]
    fn allows_dev_and_unregistered_custom_chains() {
        for chain in
            ["dev".to_string(), custom_genesis(901), custom_genesis(902), custom_genesis(903)]
        {
            for policy_args in POLICY_ARGS {
                validate_cli(&chain, policy_args).unwrap();
            }
        }
    }

    #[test]
    fn allows_registered_chains_without_test_policy() {
        for chain in ["op-mainnet".to_string(), "unichain-sepolia".to_string(), custom_genesis(10)]
        {
            validate_cli(&chain, &[]).unwrap();
        }
    }
}
