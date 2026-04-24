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

//! A downstream integration of Flashblocks.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use reth_primitives_traits::NodePrimitives;
use std::sync::Arc;

// Included to enable serde feature for OpReceipt type used transitively
use reth_optimism_primitives as _;

// Used by downstream crates that depend on this crate
use alloy_rpc_types as _;

mod consensus;
pub use consensus::FlashBlockConsensusClient;

mod payload;
pub use payload::{FlashBlock, PendingFlashBlock};

mod sequence;
pub use sequence::{
    FlashBlockCompleteSequence, FlashBlockPendingSequence, SequenceExecutionOutcome,
};

mod service;
pub use service::{
    CanonicalBlockNotification, FlashBlockBuildInfo, FlashBlockService,
    create_canonical_block_channel,
};

mod worker;
pub use worker::FlashblockCachedReceipt;

mod cache;

mod pending_state;
pub use pending_state::{PendingBlockState, PendingStateRegistry};

pub mod validation;

mod tx_cache;
pub use tx_cache::TransactionCache;
#[cfg(test)]
mod test_utils;

mod ws;
pub use ws::{FlashBlockDecoder, WsConnect, WsFlashBlockStream};

/// Receiver of the most recent [`PendingFlashBlock`] built out of [`FlashBlock`]s.
///
/// [`FlashBlock`]: crate::FlashBlock
pub type PendingBlockRx<N> = tokio::sync::watch::Receiver<Option<PendingFlashBlock<N>>>;

/// Receiver of the sequences of [`FlashBlock`]s built.
///
/// [`FlashBlock`]: crate::FlashBlock
pub type FlashBlockCompleteSequenceRx =
    tokio::sync::broadcast::Receiver<FlashBlockCompleteSequence>;

/// Receiver of received [`FlashBlock`]s from the (websocket) subscription.
///
/// [`FlashBlock`]: crate::FlashBlock
pub type FlashBlockRx = tokio::sync::broadcast::Receiver<Arc<FlashBlock>>;

/// Receiver that signals whether a [`FlashBlock`] is currently being built.
pub type InProgressFlashBlockRx = tokio::sync::watch::Receiver<Option<FlashBlockBuildInfo>>;

/// Container for all flashblocks-related listeners.
///
/// Groups together the channels for flashblock-related updates.
#[derive(Debug)]
pub struct FlashblocksListeners<N: NodePrimitives> {
    /// Receiver of the most recent executed [`PendingFlashBlock`] built out of [`FlashBlock`]s.
    pub pending_block_rx: PendingBlockRx<N>,
    /// Subscription channel of the complete sequences of [`FlashBlock`]s built.
    pub flashblocks_sequence: tokio::sync::broadcast::Sender<FlashBlockCompleteSequence>,
    /// Receiver that signals whether a [`FlashBlock`] is currently being built.
    pub in_progress_rx: InProgressFlashBlockRx,
    /// Subscription channel for received flashblocks from the (websocket) connection.
    pub received_flashblocks: tokio::sync::broadcast::Sender<Arc<FlashBlock>>,
}

impl<N: NodePrimitives> FlashblocksListeners<N> {
    /// Creates a new [`FlashblocksListeners`] with the given channels.
    pub const fn new(
        pending_block_rx: PendingBlockRx<N>,
        flashblocks_sequence: tokio::sync::broadcast::Sender<FlashBlockCompleteSequence>,
        in_progress_rx: InProgressFlashBlockRx,
        received_flashblocks: tokio::sync::broadcast::Sender<Arc<FlashBlock>>,
    ) -> Self {
        Self { pending_block_rx, flashblocks_sequence, in_progress_rx, received_flashblocks }
    }
}
