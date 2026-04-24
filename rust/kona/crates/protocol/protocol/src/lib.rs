#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]
// Pedantic/nursery lints allowed crate-wide. Each is either a style preference
// that would require hundreds of mechanical rewrites, or a lint inherent to the
// low-level nature of this crate (byte-offset casts, async trait futures, etc.).
#![allow(
    // Byte-offset / length arithmetic routinely crosses u64<->usize.
    clippy::cast_possible_truncation,
    clippy::cast_sign_loss,
    clippy::cast_lossless,
    clippy::cast_precision_loss,
    clippy::cast_possible_wrap,
    // Suggesting `#[must_use]` on every getter / ctor is noise; the public API
    // is stable and documented.
    clippy::must_use_candidate,
    clippy::return_self_not_must_use,
    // `# Errors` / `# Panics` sections would be added to hundreds of public fns.
    clippy::missing_errors_doc,
    clippy::missing_panics_doc,
    // `Default::default()` spread pattern is idiomatic here.
    clippy::default_trait_access,
    // Closures forwarding a single arg are used for readability.
    clippy::redundant_closure_for_method_calls,
    // `map(f).unwrap_or*` forms are used for clarity over `map_or`.
    clippy::option_if_let_else,
    // Async trait methods where `Send`ness depends on generic `C`.
    clippy::future_not_send,
    // Domain-specific field/var naming overlap.
    clippy::similar_names,
    clippy::struct_field_names,
    clippy::module_name_repetitions,
    // Cheap-Copy args consumed by async fns.
    clippy::needless_pass_by_value,
    // Enum wildcard matches in tests and fixtures.
    clippy::wildcard_enum_match_arm,
    // `pub use super::*;` style.
    clippy::wildcard_imports,
    // Long internal fns are kept inline for locality.
    clippy::too_many_lines,
    // Workspace allows inherited: these lints fire under `-W clippy::pedantic/nursery`
    // but are acceptable in this crate.
    clippy::redundant_pub_crate,
    clippy::significant_drop_tightening,
    clippy::too_long_first_doc_paragraph,
    clippy::doc_markdown,
    clippy::unreadable_literal,
    clippy::items_after_statements,
    clippy::missing_const_for_fn,
    clippy::map_unwrap_or,
    clippy::needless_collect,
    clippy::large_stack_arrays,
    clippy::large_stack_frames,
    clippy::large_futures,
    clippy::unused_self,
    clippy::unused_async,
    clippy::bool_to_int_with_if,
    clippy::single_match_else,
    clippy::manual_let_else,
    clippy::redundant_else,
    clippy::ignored_unit_patterns,
    clippy::unnecessary_debug_formatting,
    clippy::unnecessary_semicolon,
    clippy::from_over_into,
    clippy::iter_not_returning_iterator,
    clippy::unnecessary_wraps,
    clippy::redundant_closure,
    clippy::must_use_unit,
    clippy::stable_sort_primitive,
    clippy::match_wildcard_for_single_variants,
    clippy::ref_option,
    clippy::needless_raw_string_hashes,
    clippy::elidable_lifetime_names,
    clippy::inline_always,
    clippy::non_std_lazy_statics,
    clippy::ignore_without_reason,
    clippy::should_panic_without_expect,
    clippy::ptr_as_ptr,
    clippy::range_plus_one,
    clippy::used_underscore_binding,
    clippy::zero_sized_map_values,
    clippy::implicit_clone,
    clippy::ip_constant,
    clippy::needless_continue,
    clippy::or_fun_call,
    clippy::semicolon_if_nothing_returned,
    clippy::let_underscore_future,
    clippy::manual_assert,
)]

extern crate alloc;

mod batch;
pub use batch::{
    Batch, BatchDecodingError, BatchDropReason, BatchEncodingError, BatchReader, BatchTransaction,
    BatchType, BatchValidationProvider, BatchValidity, BatchWithInclusionBlock, DecompressionError,
    MAX_SPAN_BATCH_ELEMENTS, RawSpanBatch, SINGLE_BATCH_TYPE, SPAN_BATCH_TYPE, SingleBatch,
    SpanBatch, SpanBatchBits, SpanBatchEip1559TransactionData, SpanBatchEip2930TransactionData,
    SpanBatchEip7702TransactionData, SpanBatchElement, SpanBatchError,
    SpanBatchLegacyTransactionData, SpanBatchPayload, SpanBatchPrefix, SpanBatchTransactionData,
    SpanBatchTransactions, SpanDecodingError,
};

mod brotli;
pub use brotli::{BrotliDecompressionError, decompress_brotli};

mod sync;
pub use sync::SyncStatus;

mod attributes;
pub use attributes::OpAttributesWithParent;

mod errors;
pub use errors::OpBlockConversionError;

mod block;
pub use block::{BlockInfo, FromBlockError, L2BlockInfo};

mod frame;
pub use frame::{
    DERIVATION_VERSION_0, FRAME_OVERHEAD, Frame, FrameDecodingError, FrameParseError, MAX_FRAME_LEN,
};

mod utils;
pub use utils::{read_tx_data, to_system_config};

mod channel;
pub use channel::{
    CHANNEL_ID_LENGTH, Channel, ChannelError, ChannelId, FJORD_MAX_RLP_BYTES_PER_CHANNEL,
    MAX_RLP_BYTES_PER_CHANNEL,
};

mod ordered_channel;
pub use ordered_channel::{OrderedChannel, ReadError};

mod deposits;
pub use deposits::{
    DEPOSIT_EVENT_ABI, DEPOSIT_EVENT_ABI_HASH, DEPOSIT_EVENT_VERSION_0, DepositError,
    decode_deposit,
};

mod info;
pub use info::{
    BlockInfoError, DecodeError, L1BlockInfoBedrock, L1BlockInfoBedrockBaseFields,
    L1BlockInfoBedrockFields, L1BlockInfoBedrockOnlyFields, L1BlockInfoEcotone,
    L1BlockInfoEcotoneBaseFields, L1BlockInfoEcotoneFields, L1BlockInfoEcotoneOnlyFields,
    L1BlockInfoIsthmus, L1BlockInfoIsthmusBaseFields, L1BlockInfoIsthmusFields, L1BlockInfoJovian,
    L1BlockInfoJovianBaseFields, L1BlockInfoJovianFields, L1BlockInfoTx,
};

mod predeploys;
pub use predeploys::Predeploys;

mod output_root;
pub use output_root::OutputRoot;

#[cfg(any(test, feature = "test-utils"))]
pub mod test_utils;
