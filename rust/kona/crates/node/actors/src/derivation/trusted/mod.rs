//! Delegated derivation: the actor follows the safe and finalized heads of an external
//! derivation delegate.

mod actor;

mod client;
pub use client::{
    DerivationDelegateClient, DerivationDelegateClientError, DerivationDelegateProvider,
};
