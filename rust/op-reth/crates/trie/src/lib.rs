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

//! Standalone crate for Optimism Trie Node storage.
//!
//! External storage for intermediary trie nodes that are otherwise discarded by pipeline and
//! live sync upon successful state root update. Storing these intermediary trie nodes enables
//! efficient retrieval of inputs to proof computation for duration of OP fault proof window.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

// Used only for feature propagation (serde-bincode-compat workaround).
#[cfg(feature = "serde-bincode-compat")]
use reth_ethereum_primitives as _;

pub mod api;
pub use api::{
    BlockStateDiff, OpProofsInitProvider, OpProofsProviderRO, OpProofsProviderRw, OpProofsStore,
};

pub mod initialize;
pub use initialize::InitializationJob;

pub mod in_memory;
pub use in_memory::{
    InMemoryAccountCursor, InMemoryProofsStorage, InMemoryStorageCursor, InMemoryTrieCursor,
};

pub mod db;
pub use db::{MdbxAccountCursor, MdbxProofsStorage, MdbxStorageCursor, MdbxTrieCursor};

#[cfg(feature = "metrics")]
pub mod metrics;
#[cfg(feature = "metrics")]
pub use metrics::{
    OpProofsHashedAccountCursor, OpProofsHashedStorageCursor, OpProofsStorage, OpProofsTrieCursor,
    StorageMetrics,
};

#[cfg(not(feature = "metrics"))]
/// Alias for [`OpProofsStore`] type without metrics (`metrics` feature is disabled).
pub type OpProofsStorage<S> = S;

pub mod proof;

pub mod provider;

pub mod live;

pub mod cursor;
#[cfg(not(feature = "metrics"))]
pub use cursor::{OpProofsHashedAccountCursor, OpProofsHashedStorageCursor, OpProofsTrieCursor};

pub mod cursor_factory;
pub use cursor_factory::{OpProofsHashedAccountCursorFactory, OpProofsTrieCursorFactory};

pub mod error;
pub use error::{OpProofsStorageError, OpProofsStorageResult};

mod prune;
pub use prune::{
    OpProofStoragePruner, OpProofStoragePrunerResult, OpProofStoragePrunerTask, PrunerError,
    PrunerOutput,
};
