mod actor;
pub use actor::DelegateDerivationActor;

mod client;
pub use client::{
    DerivationDelegateClient, DerivationDelegateClientError, DerivationDelegateProvider,
};

#[cfg(test)]
mod tests;
