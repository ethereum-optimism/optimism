#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]

extern crate alloc;

mod graph;
pub use graph::MessageGraph;

mod event;
pub use event::ManagedEvent;

mod control;
pub use control::ControlEvent;

mod replacement;
pub use replacement::BlockReplacement;

mod traits;
pub use traits::{InteropProvider, InteropValidator};

mod safety;
pub use safety::SafetyLevelParseError;

mod errors;
pub use errors::{
    InteropValidationError, MessageGraphError, MessageGraphResult, SuperRootError, SuperRootResult,
};

mod root;
pub use root::{ChainRootInfo, OutputRootWithChain, SuperRoot, SuperRootOutput};

mod message;
pub use message::{
    EnrichedExecutingMessage, ExecutingDescriptor, ExecutingMessage, MessageIdentifier,
    RawMessagePayload, extract_executing_messages, parse_log_to_executing_message,
    parse_logs_to_executing_msgs,
};

mod depset;
pub use depset::{ChainDependency, DependencySet};

pub use op_alloy_consensus::interop::SafetyLevel;

mod access_list;
pub use access_list::{
    parse_access_list_item_to_inbox_entries, parse_access_list_items_to_inbox_entries,
};
mod derived;
pub use derived::{DerivedIdPair, DerivedRefPair};

mod constants;
pub use constants::{MESSAGE_EXPIRY_WINDOW, SUPER_ROOT_VERSION};

#[cfg(any(test, feature = "test-utils"))]
mod test_util;
#[cfg(any(test, feature = "test-utils"))]
pub use test_util::{
    ChainBuilder, ExecutingMessageBuilder, InteropProviderError, MockInteropProvider,
    SuperchainBuilder,
};
