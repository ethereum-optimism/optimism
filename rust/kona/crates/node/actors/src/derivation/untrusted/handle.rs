//! The engine's handle to derivation from L1.

use super::Message;
use crate::{
    derivation::Stopped,
    engine::{Derived, Safety},
};
use async_trait::async_trait;
use kona_protocol::{BlockInfo, L2BlockInfo};
use std::sync::Arc;
use tokio::sync::{mpsc, oneshot, watch};

/// Drives derivation from L1, and reads its progress.
#[derive(Debug, Clone)]
pub struct Handle {
    messages: mpsc::Sender<Message>,
    safe_head: Arc<watch::Sender<Option<L2BlockInfo>>>,
    derived: watch::Receiver<Option<(L2BlockInfo, Derived)>>,
    current_l1: watch::Receiver<BlockInfo>,
}

impl Handle {
    /// Creates a handle from the actor's request queue and published values.
    pub(super) fn new(
        messages: mpsc::Sender<Message>,
        safe_head: watch::Sender<Option<L2BlockInfo>>,
        derived: watch::Receiver<Option<(L2BlockInfo, Derived)>>,
        current_l1: watch::Receiver<BlockInfo>,
    ) -> Self {
        Self { messages, safe_head: Arc::new(safe_head), derived, current_l1 }
    }

    /// Returns the L1 block derivation has read up to: the pipeline's origin, which may not be
    /// fully derived yet. It is the default block until derivation starts.
    pub fn current_l1(&self) -> BlockInfo {
        *self.current_l1.borrow()
    }

    /// Waits for what to make safe next on top of `safe_head`: the unsafe chain's block at the
    /// next height, if it matches the attributes derived, and otherwise the attributes.
    ///
    /// The actor derives on the latest safe head given. Cancelling the wait loses nothing: what
    /// it derived stays published until a newer safe head replaces it.
    pub async fn derive(&mut self, safe_head: L2BlockInfo) -> Result<Derived, Stopped> {
        self.safe_head.send_if_modified(|current| current.replace(safe_head) != Some(safe_head));
        let derived = self
            .derived
            .wait_for(|derived| derived.as_ref().is_some_and(|(on, _)| *on == safe_head))
            .await
            .map_err(|_| Stopped)?;
        let (_, derived) = derived.clone().expect("derived on the safe head");
        Ok(derived)
    }

    /// Restarts derivation from `safe`, interrupting any derivation in progress: the attributes
    /// derived before are discarded, and the pipeline resets onto `safe` before it derives again.
    /// Returns once the actor has taken the reset; the pipeline resets in the actor's own time, so
    /// the caller never waits on L1 for it.
    pub async fn reset(&self, safe: L2BlockInfo) -> Result<(), Stopped> {
        let (result_tx, result_rx) = oneshot::channel();
        self.messages.send(Message::Reset { safe, result_tx }).await.map_err(|_| Stopped)?;
        result_rx.await.map_err(|_| Stopped)
    }

    /// Flushes the channel of the last block derived, and waits until it has.
    pub async fn flush(&self) -> Result<(), Stopped> {
        let (result_tx, result_rx) = oneshot::channel();
        self.messages.send(Message::Flush { result_tx }).await.map_err(|_| Stopped)?;
        result_rx.await.map_err(|_| Stopped)
    }
}

#[async_trait]
impl Safety for Handle {
    type Error = Stopped;

    async fn next(&mut self, safe_head: L2BlockInfo) -> Result<Derived, Stopped> {
        Self::derive(self, safe_head).await
    }

    async fn reset(&mut self, safe: L2BlockInfo) -> Result<(), Stopped> {
        Self::reset(self, safe).await
    }

    async fn flush(&mut self) -> Result<(), Stopped> {
        Self::flush(self).await
    }
}
