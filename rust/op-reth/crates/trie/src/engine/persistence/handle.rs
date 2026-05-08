//! Handle and action enum for the persistence service.

use super::{error::PersistenceError, service::PersistenceService};
use crate::{BlockStateDiff, OpProofsStore, prune::OpProofStoragePruner};
use alloy_eips::eip1898::BlockWithParent;
use crossbeam_channel::Sender;
use reth_provider::BlockHashReader;
use std::{sync::Arc, thread};

/// Messages sent to the persistence service.
#[derive(Debug)]
pub enum PersistenceAction {
    /// Save a batch of trie updates to storage.
    ///
    /// Contains:
    /// 1. The list of blocks and their diffs (ordered Oldest -> Newest).
    /// 2. A response channel: `Ok(Some(n))` = persisted up to block n, `Ok(None)` = empty batch,
    ///    `Err` = write/prune failure.
    SaveUpdates(
        Vec<Arc<(BlockWithParent, BlockStateDiff)>>,
        Sender<Result<Option<u64>, PersistenceError>>,
    ),
    /// Unwind history to the specified block (inclusive).
    /// All history strictly after this block is removed.
    Unwind(BlockWithParent, Sender<Result<(), PersistenceError>>),
}

/// A handle to communicate with the Live Trie persistence service.
#[derive(Debug, Clone)]
pub struct PersistenceHandle {
    sender: Sender<PersistenceAction>,
}

impl PersistenceHandle {
    /// Create a new handle.
    pub const fn new(sender: Sender<PersistenceAction>) -> Self {
        Self { sender }
    }

    /// Spawn the service in a new thread and return a handle.
    pub fn spawn<H, S>(pruner: OpProofStoragePruner<S, H>, storage: S) -> Self
    where
        S: OpProofsStore + Clone + 'static,
        H: BlockHashReader + Send + Sync + 'static,
    {
        let (tx, rx) = crossbeam_channel::bounded(2);
        let service = PersistenceService::new(pruner, storage, rx);

        thread::Builder::new()
            .name("Live Trie Persistence".into())
            .spawn(move || service.run())
            .expect("failed to spawn live trie persistence thread");

        Self::new(tx)
    }

    /// Send a save request.
    ///
    /// Returns an error if the persistence service has stopped.
    pub fn save_updates(
        &self,
        updates: Vec<Arc<(BlockWithParent, BlockStateDiff)>>,
        response_tx: Sender<Result<Option<u64>, PersistenceError>>,
    ) -> Result<(), PersistenceError> {
        self.sender
            .send(PersistenceAction::SaveUpdates(updates, response_tx))
            .map_err(|_| PersistenceError::Disconnected)
    }

    /// Send an unwind request.
    ///
    /// Returns an error if the persistence service has stopped.
    pub fn unwind(
        &self,
        to: BlockWithParent,
        response_tx: Sender<Result<(), PersistenceError>>,
    ) -> Result<(), PersistenceError> {
        self.sender
            .send(PersistenceAction::Unwind(to, response_tx))
            .map_err(|_| PersistenceError::Disconnected)
    }
}
