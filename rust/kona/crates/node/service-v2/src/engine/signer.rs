//! Canonical reconciliation of the unsafe-block signer used by Engine-owned gossip.

use crate::l1::{L1CursorStatus, L1Reader, L1Snapshot};
use alloy_primitives::Address;
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;
use std::sync::Arc;
use tokio::sync::{mpsc, oneshot, watch};

/// Maintains an Engine-owned L1 cursor and reads signer state at canonical snapshot targets.
///
/// Reading `SystemConfig` storage at the target hash makes coalesced watcher updates and branch
/// replacement equivalent to replaying every intermediate signer log, without depending on the
/// watch channel as an event log.
#[derive(Debug)]
pub(super) struct SignerTracker {
    config: Arc<RollupConfig>,
    reader: L1Reader,
    snapshots: watch::Receiver<L1Snapshot>,
    signer_tx: mpsc::Sender<Address>,
    signer: Address,
    cursor: Option<BlockInfo>,
}

impl SignerTracker {
    pub(super) const fn new(
        config: Arc<RollupConfig>,
        reader: L1Reader,
        snapshots: watch::Receiver<L1Snapshot>,
        signer_tx: mpsc::Sender<Address>,
        initial_signer: Address,
    ) -> Self {
        Self { config, reader, snapshots, signer_tx, signer: initial_signer, cursor: None }
    }

    pub(super) async fn run(
        mut self,
        mut shutdown: oneshot::Receiver<()>,
    ) -> Result<(), SignerTrackerError> {
        self.sync_snapshot().await;
        loop {
            tokio::select! {
                biased;
                _ = &mut shutdown => return Ok(()),
                changed = self.snapshots.changed() => {
                    changed.map_err(|_| SignerTrackerError::WatcherStopped)?;
                    self.sync_snapshot().await;
                }
            }
        }
    }

    async fn sync_snapshot(&mut self) {
        let Some(target) = self.snapshots.borrow_and_update().head else {
            return;
        };
        if let Err(error) = self.sync_to(target).await {
            warn!(target: "engine::signer", ?error, "Canonical signer reconciliation yielded; retrying at the next L1 snapshot");
        }
    }

    async fn sync_to(&mut self, target: BlockInfo) -> Result<(), SignerTrackerError> {
        if let Some(cursor) = self.cursor &&
            let L1CursorStatus::Reorg { common_ancestor, .. } =
                self.reader.validate_cursor(cursor).await?
        {
            warn!(target: "engine::signer", ?cursor, ?common_ancestor, "Engine signer cursor was replaced");
        }

        let canonical = self
            .reader
            .block_by_number(target.number)
            .await?
            .ok_or(SignerTrackerError::MissingCanonicalBlock(target.number))?;
        if canonical.hash != target.hash {
            return Err(SignerTrackerError::SnapshotReplaced { target, canonical });
        }

        let signer = self
            .reader
            .unsafe_block_signer_at(self.config.l1_system_config_address, canonical.hash)
            .await?;
        if signer != self.signer {
            self.signer_tx.send(signer).await.map_err(|_| SignerTrackerError::NetworkStopped)?;
            info!(target: "engine::signer", %signer, l1 = canonical.number, "Unsafe block signer updated");
            self.signer = signer;
        }
        self.cursor = Some(canonical);
        Ok(())
    }
}

/// Terminal signer reconciliation failure.
#[derive(Debug, thiserror::Error)]
pub(super) enum SignerTrackerError {
    /// Direct L1 access failed.
    #[error("L1 signer reconciliation failed: {0}")]
    L1(String),
    /// A canonical target unexpectedly contained a gap.
    #[error("canonical L1 block {0} was unavailable during signer reconciliation")]
    MissingCanonicalBlock(u64),
    /// The watched target was replaced before it could be processed.
    #[error("L1 snapshot target {target:?} was replaced by {canonical:?}")]
    SnapshotReplaced {
        /// Target published by the watcher.
        target: BlockInfo,
        /// Canonical block now present at the same height.
        canonical: BlockInfo,
    },
    /// The L1 watcher stopped publishing snapshots.
    #[error("L1 watcher stopped")]
    WatcherStopped,
    /// Engine's network child stopped accepting signer updates.
    #[error("Engine network stopped accepting signer updates")]
    NetworkStopped,
}

impl From<kona_derive::PipelineErrorKind> for SignerTrackerError {
    fn from(error: kona_derive::PipelineErrorKind) -> Self {
        Self::L1(error.to_string())
    }
}
