#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![cfg_attr(not(test), no_std)]

extern crate alloc;

mod errors;
pub use errors::{OrderedListWalkerError, OrderedListWalkerResult, TrieNodeError, TrieNodeResult};

mod traits;
pub use traits::{TrieHinter, TrieProvider};

mod node;
pub use node::TrieNode;

mod list_walker;
pub use list_walker::OrderedListWalker;

mod noop;
pub use noop::{NoopTrieHinter, NoopTrieProvider};

mod util;
pub use util::ordered_trie_with_encoder;

// Re-export [alloy_trie::Nibbles].
pub use alloy_trie::Nibbles;

#[cfg(test)]
mod test_util;
