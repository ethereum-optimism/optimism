#![allow(missing_docs, rustdoc::missing_crate_level_docs)]

use alloy_primitives::Address;
use reth_node_core::version::{RethCliVersionConsts, try_init_version_metadata};
use reth_optimism_chainspec::is_superchain_chain_id;
use reth_optimism_cli::{Cli, chainspec::OpChainSpecParser};
use reth_optimism_node::{OpNode, args::RollupArgs, launch_node};
use tracing::{info, warn};

use std::borrow::Cow;

#[global_allocator]
static ALLOC: reth_cli_util::allocator::Allocator = reth_cli_util::allocator::new_allocator();

#[cfg(all(feature = "jemalloc-prof", unix))]
#[unsafe(export_name = "_rjem_malloc_conf")]
static MALLOC_CONF: &[u8] = b"prof:true,prof_active:true,lg_prof_sample:19\0";

fn main() {
    reth_cli_util::sigsegv_handler::install();

    // Enable backtraces unless a RUST_BACKTRACE value has already been explicitly provided.
    if std::env::var_os("RUST_BACKTRACE").is_none() {
        unsafe {
            std::env::set_var("RUST_BACKTRACE", "1");
        }
    }

    // Install op-reth's own build metadata before clap reads it during parsing.
    const CLIENT_NAME: &str = "op-reth";
    let info = op_version::build_info!();
    let result = try_init_version_metadata(RethCliVersionConsts {
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
    });
    if result.is_err() {
        eprintln!("Error: build info is already embedded. This is a bug.")
    }

    if let Err(err) = Cli::<OpChainSpecParser, RollupArgs>::parse_with_denied_args().run(
        async move |builder, args| {
            if let Some(excessive_refund_target) = args.testing_sdm_fixed_policy {
                validate_testing_sdm_fixed_policy(
                    excessive_refund_target,
                    builder.config().chain.chain.id(),
                )?;
            }

            info!(target: "reth::cli", "Launching node");
            launch_node(builder, OpNode::new(args)).await
        },
    ) {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}

fn validate_testing_sdm_fixed_policy(
    excessive_refund_target: Option<Address>,
    chain_id: u64,
) -> eyre::Result<()> {
    // Check the resolved chain ID, not how the chain was selected: a custom genesis can
    // still configure a registered chain. Unregistered chains are not necessarily dev chains.
    if is_superchain_chain_id(chain_id) {
        eyre::bail!(
            "--testing.sdm-fixed-policy is not allowed for superchain registry chain ID {chain_id}"
        );
    }
    warn!(
        target: "reth::cli",
        ?excessive_refund_target,
        "TEST-ONLY SDM fixed-refund policy enabled; never use this policy in production"
    );
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_optimism_cli::commands::Commands;
    use std::io::Write;

    const TARGET: &str = "0x0000000000000000000000000000000000000001";
    const POLICY_ARGS: [&[&str]; 2] =
        [&["--testing.sdm-fixed-policy"], &["--testing.sdm-fixed-policy", TARGET]];

    fn validate_cli(chain: &str, policy_args: &[&str]) -> eyre::Result<()> {
        let cli = Cli::<OpChainSpecParser, RollupArgs>::try_parse_args_from(
            ["op-reth", "node", "--chain", chain].into_iter().chain(policy_args.iter().copied()),
        )?;
        let Commands::Node(node) = cli.command else {
            panic!("expected node command");
        };
        if let Some(excessive_refund_target) = node.ext.testing_sdm_fixed_policy {
            validate_testing_sdm_fixed_policy(excessive_refund_target, node.chain.chain.id())?;
        }
        Ok(())
    }

    fn custom_genesis(chain_id: u64) -> String {
        format!(
            r#"{{"config":{{"chainId":{chain_id}}},"gasLimit":"0x1c9c380","difficulty":"0x0","alloc":{{}}}}"#
        )
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
