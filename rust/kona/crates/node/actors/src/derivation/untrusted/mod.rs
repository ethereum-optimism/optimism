//! Derivation from L1.
//!
//! The actor steps a derivation pipeline on the safe head the engine gives it, and compares the
//! payload attributes it derives with the unsafe chain's block at their height. It publishes the
//! block, if it matches them, for the engine to make safe, and otherwise the attributes, for the
//! engine to build the next safe block from. The engine drives
//! it through the [`Handle`], which implements [`engine::Safety`](crate::engine::Safety).

mod actor;

mod attributes;

mod handle;
pub use handle::Handle;

mod l1;
pub use l1::L1;

mod l2;
pub use l2::L2;

mod chain_provider;
pub use chain_provider::{L1ChainProvider, L1ChainProviderError};

use crate::engine::Derived;
use kona_derive::{Pipeline, PipelineErrorKind, ResetError, SignalReceiver};
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::future::Future;
use thiserror::Error;
use tokio::sync::{mpsc, oneshot, watch};

/// A request to the actor.
#[derive(Debug)]
enum Message {
    /// Restart derivation from the safe head, and reply once the actor has taken the reset.
    Reset { safe: L2BlockInfo, result_tx: oneshot::Sender<()> },
    /// Flush the channel of the last block derived, and reply once it has.
    Flush { result_tx: oneshot::Sender<()> },
}

/// An error that stops the actor.
#[derive(Error, Debug)]
pub enum ActorError {
    /// The derivation pipeline failed critically.
    #[error(transparent)]
    Pipeline(#[from] PipelineErrorKind),
    /// The derivation pipeline is inconsistent with the safe head it derives on, which only a bug
    /// can cause.
    #[error("the pipeline is inconsistent with the safe head: {0}")]
    Inconsistent(ResetError),
    /// The L1 actor stopped.
    #[error("the L1 actor stopped")]
    L1Stopped,
    /// Every handle was dropped.
    #[error("every handle was dropped")]
    HandlesDropped,
}

/// Constructs the handle and the actor.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    messages_rx: mpsc::Receiver<Message>,
    safe_head_rx: watch::Receiver<Option<L2BlockInfo>>,
    derived_tx: watch::Sender<Option<(L2BlockInfo, Derived)>>,
    current_l1_tx: watch::Sender<BlockInfo>,
}

impl Builder {
    /// Creates the builder. The request queue holds one request: each waits for its reply.
    pub fn new() -> Self {
        let (messages_tx, messages_rx) = mpsc::channel(1);
        let (safe_head_tx, safe_head_rx) = watch::channel(None);
        let (derived_tx, derived_rx) = watch::channel(None);
        let (current_l1_tx, current_l1_rx) = watch::channel(BlockInfo::default());
        let handle = Handle::new(messages_tx, safe_head_tx, derived_rx, current_l1_rx);
        Self { handle, messages_rx, safe_head_rx, derived_tx, current_l1_tx }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies the dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// The actor derives nothing until the engine first resets it. Whenever the pipeline fails
    /// temporarily and L1 has no child of the pipeline's origin, the actor waits on `l1` for one,
    /// and resets the pipeline if the origin is reorged out instead. The pipeline should read L1
    /// heights from the same L1 actor, through an [`L1ChainProvider`]. The actor reads the unsafe
    /// chain's blocks from `l2`. Runtime work begins when the future is polled. Dropping the future
    /// stops the actor.
    pub fn build<Pipeline_, L1_, L2_>(
        self,
        pipeline: Pipeline_,
        l1: L1_,
        l2: L2_,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static
    where
        Pipeline_: Pipeline + SignalReceiver + Send + Sync + 'static,
        L1_: L1 + 'static,
        L2_: L2 + 'static,
    {
        let Self { handle: _, messages_rx, safe_head_rx, derived_tx, current_l1_tx } = self;
        let channels = actor::Channels { messages_rx, safe_head_rx, derived_tx, current_l1_tx };
        let mut actor = actor::Actor::new(pipeline, l1, l2, channels);
        async move { actor.run().await }
    }
}

impl Default for Builder {
    fn default() -> Self {
        Self::new()
    }
}
