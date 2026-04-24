// Pedantic/nursery lints from workspace-level clippy config are largely stylistic;
// reth upstream maintains its own curated lint set. Allow the cosmetic/architectural-cost
// categories here so real issues stay visible.
#![allow(clippy::cast_lossless)]
#![allow(clippy::cast_possible_truncation)]
#![allow(clippy::cast_possible_wrap)]
#![allow(clippy::cast_precision_loss)]
#![allow(clippy::cast_sign_loss)]
#![allow(clippy::default_trait_access)]
#![allow(clippy::doc_markdown)]
#![allow(clippy::elidable_lifetime_names)]
#![allow(clippy::fallible_impl_from)]
#![allow(clippy::float_cmp)]
#![allow(clippy::future_not_send)]
#![allow(clippy::ignore_without_reason)]
#![allow(clippy::ignored_unit_patterns)]
#![allow(clippy::inconsistent_struct_constructor)]
#![allow(clippy::inline_always)]
#![allow(clippy::items_after_statements)]
#![allow(clippy::large_futures)]
#![allow(clippy::large_stack_arrays)]
#![allow(clippy::large_stack_frames)]
#![allow(clippy::manual_let_else)]
#![allow(clippy::map_unwrap_or)]
#![allow(clippy::match_wildcard_for_single_variants)]
#![allow(clippy::mismatching_type_param_order)]
#![allow(clippy::missing_const_for_fn)]
#![allow(clippy::missing_errors_doc)]
#![allow(clippy::missing_fields_in_debug)]
#![allow(clippy::missing_panics_doc)]
#![allow(clippy::must_use_candidate)]
#![allow(clippy::needless_pass_by_value)]
#![allow(clippy::needless_raw_string_hashes)]
#![allow(clippy::non_std_lazy_statics)]
#![allow(clippy::redundant_closure_for_method_calls)]
#![allow(clippy::redundant_pub_crate)]
#![allow(clippy::ref_option)]
#![allow(clippy::return_self_not_must_use)]
#![allow(clippy::semicolon_if_nothing_returned)]
#![allow(clippy::significant_drop_tightening)]
#![allow(clippy::similar_names)]
#![allow(clippy::single_match_else)]
#![allow(clippy::struct_excessive_bools)]
#![allow(clippy::struct_field_names)]
#![allow(clippy::too_long_first_doc_paragraph)]
#![allow(clippy::too_many_lines)]
#![allow(clippy::unchecked_time_subtraction)]
#![allow(clippy::uninlined_format_args)]
#![allow(clippy::unnecessary_semicolon)]
#![allow(clippy::unnecessary_wraps)]
#![allow(clippy::unreadable_literal)]
#![allow(clippy::unused_async)]
#![allow(clippy::unused_self)]
#![allow(clippy::use_self)]
#![allow(clippy::used_underscore_binding)]
#![allow(clippy::wildcard_imports)]

//! Optimism meta crate that provides access to commonly used reth dependencies.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]
#![allow(unused_crate_dependencies)]

/// Re-exported optimism types
#[doc(inline)]
pub use reth_optimism_primitives::*;

/// Re-exported reth primitives
pub mod primitives {
    #[doc(inline)]
    pub use reth_primitives_traits::*;
}

/// Re-exported cli types
#[cfg(feature = "cli")]
pub mod cli {
    #[doc(inline)]
    pub use reth_cli_util::{
        allocator, get_secret_key, hash_or_num_value_parser, load_secret_key,
        parse_duration_from_secs, parse_duration_from_secs_or_ms, parse_ether_value,
        parse_socket_address, sigsegv_handler,
    };
    #[doc(inline)]
    pub use reth_optimism_cli::*;
}

/// Re-exported pool types
#[cfg(feature = "pool")]
pub use reth_transaction_pool as pool;

/// Re-exported consensus types
#[cfg(feature = "consensus")]
pub mod consensus {
    #[doc(inline)]
    pub use reth_consensus::*;
    /// Consensus rule checks.
    pub mod validation {
        #[doc(inline)]
        pub use reth_consensus_common::validation::*;
        #[doc(inline)]
        pub use reth_optimism_consensus::validation::*;
    }
}

/// Re-exported from `reth_chainspec`
#[allow(ambiguous_glob_reexports)]
pub mod chainspec {
    #[doc(inline)]
    pub use reth_chainspec::*;
    #[doc(inline)]
    pub use reth_optimism_chainspec::*;
}

/// Re-exported evm types
#[cfg(feature = "evm")]
pub mod evm {
    #[doc(inline)]
    pub use reth_optimism_evm::*;

    #[doc(inline)]
    pub use reth_evm as primitives;

    #[doc(inline)]
    pub use reth_revm as revm;
}

/// Re-exported exex types
#[cfg(feature = "exex")]
pub use reth_exex as exex;

/// Re-exported from `tasks`.
#[cfg(feature = "tasks")]
pub mod tasks {
    pub use reth_tasks::*;
}

/// Re-exported reth network types
#[cfg(feature = "network")]
pub mod network {
    #[doc(inline)]
    pub use reth_eth_wire as eth_wire;
    #[doc(inline)]
    pub use reth_network::*;
    #[doc(inline)]
    pub use reth_network_api as api;
}

/// Re-exported reth provider types
#[cfg(feature = "provider")]
pub mod provider {
    #[doc(inline)]
    pub use reth_provider::*;

    #[doc(inline)]
    pub use reth_db as db;
}

/// Re-exported codec crate
#[cfg(feature = "provider")]
pub use reth_codecs as codec;

/// Re-exported reth storage api types
#[cfg(feature = "storage-api")]
pub mod storage {
    #[doc(inline)]
    pub use reth_storage_api::*;
}

/// Re-exported optimism node
#[cfg(feature = "node-api")]
pub mod node {
    #[doc(inline)]
    pub use reth_node_api as api;
    #[cfg(feature = "node")]
    pub use reth_node_builder as builder;
    #[doc(inline)]
    pub use reth_node_core as core;
    #[cfg(feature = "node")]
    pub use reth_optimism_node::*;
}

/// Re-exported  engine types
#[cfg(feature = "node")]
pub mod engine {
    #[doc(inline)]
    pub use reth_engine_local as local;
    #[doc(inline)]
    pub use reth_optimism_node::engine::*;
}

/// Re-exported reth trie types
#[cfg(feature = "trie")]
pub mod trie {
    #[doc(inline)]
    pub use reth_trie::*;

    #[cfg(feature = "trie-db")]
    #[doc(inline)]
    pub use reth_trie_db::*;
}

/// Re-exported rpc types
#[cfg(feature = "rpc")]
pub mod rpc {
    #[doc(inline)]
    pub use reth_optimism_rpc::*;
    #[doc(inline)]
    pub use reth_rpc::*;

    #[doc(inline)]
    pub use reth_rpc_api as api;
    #[doc(inline)]
    pub use reth_rpc_builder as builder;
    #[doc(inline)]
    pub use reth_rpc_eth_types as eth;
}
