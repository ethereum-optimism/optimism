//! Derivation actors and handle.
//!
//! The [`Builder`] creates the [`Handle`] before any dependencies exist, then builds one of two
//! actors: [`Builder::build`] derives payload attributes from L1 with a derivation pipeline, and
//! [`Builder::build_delegated`] follows the safe head of an external derivation delegate.
//!
//! Derivation drives the engine: it starts from the engine's first reset, which the engine makes
//! once the execution layer is synced, and it learns every result from the engine's replies and
//! published resets. The engine only replies to requests and publishes its state and resets.
//! Whenever the engine resets, derivation resets its pipeline onto the engine's safe head.

mod actor;

mod delegated;
pub use delegated::{
    DerivationDelegateClient, DerivationDelegateClientError, DerivationDelegateProvider,
};

mod engine;
pub use engine::Engine;

mod l1;
pub use l1::L1;

mod chain_provider;
pub use chain_provider::{L1ChainProvider, L1ChainProviderError};

mod error;
pub use error::ActorError;

mod handle;

mod status;
pub use status::DerivationStatus;

use tokio::sync::watch;

/// Constructs the handle and one of the derivation actors.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    status_tx: watch::Sender<DerivationStatus>,
}

impl Builder {
    /// Creates the builder with no derivation progress.
    pub fn new() -> Self {
        let (status_tx, status_rx) = watch::channel(DerivationStatus::default());
        Self { handle: Handle { status: status_rx }, status_tx }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }
}

impl Default for Builder {
    fn default() -> Self {
        Self::new()
    }
}

/// The derivation actor's published progress.
#[derive(Debug, Clone)]
pub struct Handle {
    status: watch::Receiver<DerivationStatus>,
}

/// Resets the engine. The engine publishes the reset before it replies.
async fn reset_engine(engine: &impl Engine) -> Result<(), ActorError> {
    engine.reset().await.map(|_| ()).map_err(|err| {
        error!(target: "derivation", ?err, "Failed to reset the engine");
        err.into()
    })
}
