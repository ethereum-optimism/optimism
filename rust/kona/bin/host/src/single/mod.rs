//! This module contains the single-chain mode for the host.

mod cfg;
pub use cfg::{SingleChainHost, SingleChainHostError, SingleChainProviders};

mod local_kv;
pub use local_kv::SingleChainLocalInputs;

mod handler;
pub use handler::SingleChainHintHandler;
