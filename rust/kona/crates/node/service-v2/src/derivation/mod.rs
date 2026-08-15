//! Safe-chain derivation, finality mapping, L1 reorg recovery, and pipeline reset.

mod control;
pub use control::{DerivationAdminAdapter, DerivationControlError};

mod delegated;
pub use delegated::{
    DelegatedDerivationService, DerivationDelegateClient, DerivationDelegateError,
    DerivationDelegateProvider,
};

mod finalizer;

mod pipeline;
pub use pipeline::ServicePipeline;

mod service;
pub use service::{DerivationPipeline, DerivationService, DerivationServiceError};

#[cfg(test)]
mod tests;
