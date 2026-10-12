//! Derivation actors and handle.
//!
//! The [`Builder`] creates the [`Handle`] before any dependencies exist, then builds one of two
//! actors: [`Builder::build`] derives payload attributes from L1 with a derivation pipeline, and
//! [`Builder::build_delegated`] follows the safe head of an external derivation delegate.

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
pub use crate::capacity::{Capacity, InvalidCapacity};
pub use handle::{DerivationClientError, DerivationClientResult};

mod state_machine;
pub use state_machine::{
    DerivationState, DerivationStateMachine, DerivationStateTransitionError, DerivationStateUpdate,
};

mod status;
pub use status::DerivationStatus;

use kona_derive::Signal;
use kona_protocol::L2BlockInfo;
use tokio::sync::{mpsc, watch};

/// Constructs the handle and one of the derivation actors.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    messages_rx: mpsc::Receiver<Message>,
    status_tx: watch::Sender<DerivationStatus>,
}

impl Builder {
    /// Creates the builder with no derivation progress.
    pub fn new(capacity: Capacity) -> Self {
        let (messages_tx, messages_rx) = mpsc::channel(capacity.get());
        let (status_tx, status_rx) = watch::channel(DerivationStatus::default());
        Self { handle: Handle { messages: messages_tx, status: status_rx }, messages_rx, status_tx }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }
}

/// Derivation progress and the derivation actor's request queue.
#[derive(Debug, Clone)]
pub struct Handle {
    messages: mpsc::Sender<Message>,
    status: watch::Receiver<DerivationStatus>,
}

/// Work for the derivation actor, all from the engine actor.
#[derive(Debug)]
enum Message {
    /// The execution layer finished syncing; derivation starts from the safe head.
    ElSyncCompleted(Box<L2BlockInfo>),
    /// The engine's safe head changed.
    SafeHead(Box<L2BlockInfo>),
    /// A signal for the derivation pipeline, such as a reset or a flush.
    Signal(Box<Signal>),
}
