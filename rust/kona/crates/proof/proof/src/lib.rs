#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![no_std]

extern crate alloc;

// The fault-proof program admits `sp1-private-projection-v1` spans exactly as the nodes do:
// without the verifier every sp1 span would be dropped here and admitted by op-node and
// kona-node. Enabled by this crate's own dependency declaration, not by feature unification.
const _: () = assert!(
    kona_protocol::projection::SP1_PROJECTION_VERIFIER_COMPILED,
    "kona-protocol must be built with `sp1-projection-verifier`"
);

#[macro_use]
extern crate tracing;

pub mod l1;

pub mod l2;

pub mod sync;

pub mod errors;

pub mod executor;

mod hint;
pub use hint::{Hint, HintType};

pub mod boot;
pub use boot::BootInfo;

mod caching_oracle;
pub use caching_oracle::{CachingOracle, FlushableCache};

mod blocking_runtime;
pub use blocking_runtime::block_on;

mod eip2935;
pub use eip2935::eip_2935_history_lookup;
