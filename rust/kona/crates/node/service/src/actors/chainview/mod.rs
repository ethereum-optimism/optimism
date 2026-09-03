//! The chain view: the actor that feeds the circuit in `kona-chainview` what the derivation
//! actor does not, and the L1 reads both need.

mod actor;
pub use actor::{ChainViewActor, ChainViewActorError};

mod fetch;
#[cfg(test)]
pub(crate) use fetch::MockL1Fetcher;
pub use fetch::{L1Fetcher, ProviderL1Fetcher, UNSAFE_BLOCK_SIGNER_SLOT};
