//! Safe and finalized chain derivation from canonical L1 data.

mod control;
pub use control::{SafeChainControlError, SafeChainHandle};

mod delegated;
pub use delegated::{
    DelegatedSafeChainService, DerivationDelegateClient, DerivationDelegateError,
    DerivationDelegateProvider,
};

mod finalizer;

mod pipeline;
pub use pipeline::ServicePipeline;

mod service;
pub use service::{DerivationPipeline, SafeChainService, SafeChainServiceError};

#[cfg(test)]
mod tests;
