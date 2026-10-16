//! Reading what the trusted source reports.

use super::Reported;
use crate::{
    derivation::Stopped,
    engine::{Finality, Safe},
};
use async_trait::async_trait;
use kona_protocol::{BlockInfo, L2BlockInfo};
use tokio::sync::watch;

/// Reads the safe and finalized heads the derivation delegate reports, and its progress.
///
/// Each clone remembers the heads it last returned, so the engine reads safe and finalized heads
/// through separate clones.
#[derive(Debug, Clone)]
pub struct Handle {
    reported: watch::Receiver<Option<Reported>>,
    current_l1: watch::Receiver<BlockInfo>,
    /// The safe head this handle last returned.
    returned_safe: Option<Safe>,
    /// The finalized head this handle last returned.
    returned_finalized: Option<L2BlockInfo>,
}

impl Handle {
    /// Creates a handle from the actor's published values.
    pub(super) const fn new(
        reported: watch::Receiver<Option<Reported>>,
        current_l1: watch::Receiver<BlockInfo>,
    ) -> Self {
        Self { reported, current_l1, returned_safe: None, returned_finalized: None }
    }

    /// Returns the L1 block the delegate has derived up to, as it last reported it. It is the
    /// default block until the delegate first reports.
    pub fn current_l1(&self) -> BlockInfo {
        *self.current_l1.borrow()
    }

    /// Waits for a safe head to adopt on top of `safe_head`: the delegate's latest, once it differs
    /// from the one last returned, or from `safe_head`, such as after the engine reset. The L1
    /// block the delegate derived up to stands in for its source. Cancelling the wait loses
    /// nothing.
    pub async fn next_safe(&mut self, safe_head: L2BlockInfo) -> Result<Safe, Stopped> {
        loop {
            let reported = *self.reported.borrow_and_update();
            if let Some(Reported { safe, .. }) = reported &&
                (safe.head != safe_head || self.returned_safe != Some(safe))
            {
                self.returned_safe = Some(safe);
                return Ok(safe);
            }
            self.reported.changed().await.map_err(|_| Stopped)?;
        }
    }

    /// Waits for the delegate's finalized head, once it differs from the one last returned.
    /// Cancelling the wait loses nothing.
    pub async fn next_finalized(&mut self) -> Result<L2BlockInfo, Stopped> {
        loop {
            let reported = *self.reported.borrow_and_update();
            if let Some(Reported { finalized, .. }) = reported &&
                self.returned_finalized != Some(finalized)
            {
                self.returned_finalized = Some(finalized);
                return Ok(finalized);
            }
            self.reported.changed().await.map_err(|_| Stopped)?;
        }
    }
}

#[async_trait]
impl Finality for Handle {
    type Error = Stopped;

    async fn next(&mut self, _safe: Safe) -> Result<L2BlockInfo, Stopped> {
        self.next_finalized().await
    }
}
