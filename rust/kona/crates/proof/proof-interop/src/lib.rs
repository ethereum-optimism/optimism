#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "arbitrary"), no_std)]

extern crate alloc;

mod pre_state;
pub use pre_state::{
    INVALID_TRANSITION, INVALID_TRANSITION_HASH, OptimisticBlock, PreState,
    TRANSITION_STATE_MAX_STEPS, TransitionState,
};

mod hint;
pub use hint::HintType;

mod provider;
pub use provider::OracleInteropProvider;

pub mod boot;
pub use boot::BootInfo;

mod consolidation;
pub use consolidation::{ConsolidationError, SuperchainConsolidator};
