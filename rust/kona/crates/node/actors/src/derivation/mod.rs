//! Derivation: what decides the engine's safe and finalized heads.
//!
//! The engine drives derivation through [`engine::Safety`](crate::engine::Safety): from its first
//! reset, it asks what to make safe next on top of its safe head, makes it safe, and asks again,
//! and it reports its resets. Derivation never asks anything of the engine.
//!
//! [`untrusted`] derives payload attributes from L1 with a derivation pipeline. [`trusted`] follows
//! the safe and finalized heads of an external derivation delegate, and also implements
//! [`engine::Finality`](crate::engine::Finality).

pub mod untrusted;
pub use untrusted::{L1, L1ChainProvider, L1ChainProviderError};

pub mod trusted;
pub use trusted::{
    DerivationDelegateClient, DerivationDelegateClientError, DerivationDelegateProvider,
};

use thiserror::Error;

/// A derivation actor stopped.
#[derive(Debug, Clone, Copy, Error)]
#[error("the derivation actor stopped")]
pub struct Stopped;
