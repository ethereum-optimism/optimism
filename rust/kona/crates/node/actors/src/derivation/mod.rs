//! Derivation actors and handle.
//!
//! The [`Builder`] creates the [`Handle`] before any dependencies exist, then builds one of two
//! actors: [`Builder::build`] derives payload attributes from L1 with a derivation pipeline, and
//! [`Builder::build_delegated`] follows the safe head of an external derivation delegate.
//!
//! Derivation drives the engine: it starts by resetting the engine once the execution layer is
//! synced, and it learns every result from the engine's replies and published state. The engine
//! never sends derivation anything on its own. The engine counts its resets in its state, and
//! derivation starts over from the engine's safe head whenever the count moves.

mod actor;

mod delegated;
pub use delegated::{DerivationDelegateClientError, DerivationDelegateProvider};

mod engine_client;
pub use engine_client::DerivationEngineClient;

mod error;
pub use error::ActorError;

mod finalizer;
pub(crate) use finalizer::L2Finalizer;

mod handle;

mod state_machine;
pub use state_machine::{
    DerivationState, DerivationStateMachine, DerivationStateTransitionError, DerivationStateUpdate,
};

mod status;
pub use status::DerivationStatus;

use crate::{EngineClientError, engine::EngineState};
use tokio::{
    sync::watch,
    time::{self, Duration},
};

/// How long to wait before asking the engine to reset again while the execution layer syncs.
const ENGINE_RESET_RETRY_INTERVAL: Duration = Duration::from_secs(1);

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

/// Resets the engine and returns its state afterwards, asking again while the execution layer
/// syncs.
async fn reset_engine(
    engine_client: &impl DerivationEngineClient,
) -> Result<EngineState, ActorError> {
    loop {
        match engine_client.reset().await {
            Err(EngineClientError::ElSyncing) => {
                debug!(target: "derivation", "Waiting for the execution layer to sync");
                time::sleep(ENGINE_RESET_RETRY_INTERVAL).await;
            }
            result => {
                return result.map_err(|err| {
                    error!(target: "derivation", ?err, "Failed to reset the engine");
                    err.into()
                });
            }
        }
    }
}
