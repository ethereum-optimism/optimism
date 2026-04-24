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

//! Standalone crate for Optimism-specific Reth configuration and builder types.
//!
//! # features
//! - `js-tracer`: Enable the `JavaScript` tracer for the `debug_trace` endpoints

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

#[cfg(feature = "test-utils")]
use reth_db_api as _;

/// CLI argument parsing for the optimism node.
pub mod args;

/// Exports optimism-specific implementations of the [`EngineTypes`](reth_node_api::EngineTypes)
/// trait.
pub mod engine;
pub use engine::OpEngineTypes;

pub mod node;
pub use node::*;

pub mod rpc;
pub use rpc::OpEngineApiBuilder;

pub mod version;
pub use version::OP_NAME_CLIENT;

pub use reth_optimism_txpool as txpool;

pub mod proof_history;

/// Helpers for running test node instances.
#[cfg(feature = "test-utils")]
pub mod utils;

pub use reth_optimism_payload_builder::{
    self as payload, OpBuiltPayload, OpPayloadAttributes, OpPayloadBuilder,
    OpPayloadBuilderAttributes, OpPayloadPrimitives, OpPayloadTypes, config::OpDAConfig,
};

pub use reth_optimism_evm::*;

pub use reth_optimism_storage::OpStorage;

use op_revm as _;
use revm as _;

#[cfg(feature = "test-utils")]
use reth_tasks as _;
