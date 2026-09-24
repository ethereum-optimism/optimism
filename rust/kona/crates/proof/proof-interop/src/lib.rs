#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "arbitrary"), no_std)]

extern crate alloc;

// The fault-proof program admits `sp1-private-projection-v1` spans exactly as the nodes do:
// without the verifier every sp1 span would be dropped here and admitted by op-node and
// kona-node. Enabled by this crate's own dependency declaration, not by feature unification.
const _: () = assert!(
    kona_protocol::projection::SP1_PROJECTION_VERIFIER_COMPILED,
    "kona-protocol must be built with `sp1-projection-verifier`"
);

mod pre_state;
pub use pre_state::{
    INVALID_TRANSITION, INVALID_TRANSITION_HASH, OptimisticBlock, PreState,
    TRANSITION_STATE_MAX_STEPS, TransitionState,
};

mod hint;
pub use hint::HintType;

mod provider;
pub use provider::{ChainScopedHinter, OracleInteropProvider};

pub mod boot;
pub use boot::BootInfo;

mod consolidation;
pub use consolidation::{ConsolidationError, SuperchainConsolidator};
