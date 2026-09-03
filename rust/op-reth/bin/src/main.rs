#![allow(missing_docs, rustdoc::missing_crate_level_docs)]

use reth_node_core::version::{RethCliVersionConsts, try_init_version_metadata};
use reth_optimism_cli::{Cli, chainspec::OpChainSpecParser};
use reth_optimism_node::{args::RollupArgs, proof_history};
use tracing::info;

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
            info!(target: "reth::cli", "Launching node");
            proof_history::launch_node(builder, args).await
        },
    ) {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}
