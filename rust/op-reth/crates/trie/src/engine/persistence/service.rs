//! Background persistence service for the live trie engine.

#[cfg(feature = "metrics")]
use super::metrics::PersistenceMetrics;
use super::{error::PersistenceError, handle::PersistenceAction};
use crate::{BlockStateDiff, OpProofsStore, api::OpProofsProviderRw, prune::OpProofStoragePruner};
use alloy_eips::eip1898::BlockWithParent;
use crossbeam_channel::{Receiver, Sender};
use reth_provider::BlockHashReader;
use std::{sync::Arc, time::Instant};
use tracing::{debug, error, info, warn};

/// Service that runs in a background thread to persist trie updates.
#[derive(Debug)]
pub struct PersistenceService<H, S> {
    /// Pruner that also owns the storage backend and block hash reader.
    pruner: OpProofStoragePruner<S, H>,
    storage: S,
    incoming: Receiver<PersistenceAction>,

    #[cfg(feature = "metrics")]
    metrics: PersistenceMetrics,
}

impl<H: BlockHashReader, S: OpProofsStore> PersistenceService<H, S> {
    /// Create a new persistence service.
    pub fn new(
        pruner: OpProofStoragePruner<S, H>,
        storage: S,
        incoming: Receiver<PersistenceAction>,
    ) -> Self {
        Self {
            pruner,
            storage,
            incoming,

            #[cfg(feature = "metrics")]
            metrics: PersistenceMetrics::new_with_labels(&[] as &[(&str, &str)]),
        }
    }

    /// Main loop for the service.
    /// Listens for incoming actions and processes them sequentially.
    pub fn run(self) {
        debug!(target: "trie::engine::persistence", "Service started");

        loop {
            match self.incoming.recv() {
                Ok(action) => match action {
                    PersistenceAction::Unwind(to, reply_tx) => {
                        self.on_unwind(to, reply_tx);
                    }
                    PersistenceAction::SaveUpdates(updates, reply_tx) => {
                        self.on_save_updates(updates, reply_tx);
                    }
                },
                Err(e) => {
                    debug!(target: "trie::engine::persistence", ?e, "Service shutting down, channel closed");
                    return;
                }
            }
        }
    }

    fn on_save_updates(
        &self,
        arc_updates: Vec<Arc<(BlockWithParent, BlockStateDiff)>>,
        reply_tx: Sender<Result<Option<u64>, PersistenceError>>,
    ) {
        if arc_updates.is_empty() {
            if let Err(e) = reply_tx.send(Ok(None)) {
                warn!(target: "trie::engine::persistence", ?e, "Failed to send empty batch result, receiver dropped");
            }
            return;
        }

        let updates: Vec<(BlockWithParent, BlockStateDiff)> =
            arc_updates.into_iter().map(Arc::unwrap_or_clone).collect();

        if let Err(e) = reply_tx.send(self.try_save_updates(updates)) {
            warn!(target: "trie::engine::persistence", ?e, "Failed to send batch result, receiver dropped");
        }
    }

    fn try_save_updates(
        &self,
        updates: Vec<(BlockWithParent, BlockStateDiff)>,
    ) -> Result<Option<u64>, PersistenceError> {
        let start = Instant::now();
        let count = updates.len();
        let first = updates.first().map(|u| u.0.block.number);
        let last = updates.last().map(|u| u.0.block.number);
        debug!(target: "trie::engine::persistence", ?count, ?first, ?last, "Writing batch to storage");

        let (provider, open_tx_duration) = Self::timed(|| self.storage.provider_rw())?;
        let (write_counts, write_duration) =
            Self::timed(|| provider.store_trie_updates_batch(updates))?;
        let (_, prune_duration) = Self::timed(|| {
            self.pruner.prune_with_provider(&provider)
                .inspect_err(|e| error!(target: "trie::engine::persistence", ?e, "Pruning failed during save, aborting transaction"))
        })?;
        let (_, commit_duration) = Self::timed(|| provider.commit())?;

        let duration = start.elapsed();

        #[cfg(feature = "metrics")]
        self.metrics.record_metrics(
            &write_counts,
            open_tx_duration,
            write_duration,
            prune_duration,
            commit_duration,
        );

        info!(
            target: "trie::engine::persistence",
            ?last,
            ?duration,
            ?open_tx_duration,
            ?write_duration,
            ?prune_duration,
            ?commit_duration,
            ?write_counts,
            blocks_count = count,
            "Batch write complete"
        );

        Ok(last)
    }

    fn on_unwind(&self, to: BlockWithParent, reply_tx: Sender<Result<(), PersistenceError>>) {
        if let Err(e) = reply_tx.send(self.try_unwind(to)) {
            warn!(target: "trie::engine::persistence", ?e, "Failed to send unwind result, receiver dropped");
        }
    }

    fn try_unwind(&self, to: BlockWithParent) -> Result<(), PersistenceError> {
        debug!(target: "trie::engine::persistence", to_block = ?to.block.number, "Unwinding storage");
        let provider = self.storage.provider_rw()?;
        provider.unwind_history(to)?;
        provider.commit()?;
        debug!(target: "trie::engine::persistence", "Unwind successful");
        Ok(())
    }

    /// Execute a closure and return its result along with the time elapsed.
    fn timed<T, E, F: FnOnce() -> Result<T, E>>(f: F) -> Result<(T, std::time::Duration), E> {
        let start = Instant::now();
        let result = f()?;
        Ok((result, start.elapsed()))
    }
}
