//! The engine's handle to finality.

use super::Stopped;
use crate::engine::{Finality, Safe};
use async_trait::async_trait;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;
use tokio::sync::watch;

/// Gives the finality actor the engine's safe heads, and reads the blocks it decides to finalize.
///
/// Each clone remembers the block it last returned.
#[derive(Debug, Clone)]
pub struct Handle {
    safe: Arc<watch::Sender<Option<Safe>>>,
    finalized: watch::Receiver<Option<L2BlockInfo>>,
    /// The block this handle last returned.
    returned: Option<L2BlockInfo>,
}

impl Handle {
    /// Creates a handle from the actor's published values.
    pub(super) const fn new(
        safe: Arc<watch::Sender<Option<Safe>>>,
        finalized: watch::Receiver<Option<L2BlockInfo>>,
    ) -> Self {
        Self { safe, finalized, returned: None }
    }
}

#[async_trait]
impl Finality for Handle {
    type Error = Stopped;

    /// Gives the actor `safe`, and waits for a block to finalize other than the one last
    /// returned. Cancelling the wait loses nothing.
    async fn next(&mut self, safe: Safe) -> Result<L2BlockInfo, Stopped> {
        self.safe.send_if_modified(|current| current.replace(safe) != Some(safe));
        loop {
            let finalized = *self.finalized.borrow_and_update();
            if let Some(finalized) = finalized &&
                self.returned != Some(finalized)
            {
                self.returned = Some(finalized);
                return Ok(finalized);
            }
            self.finalized.changed().await.map_err(|_| Stopped)?;
        }
    }
}
