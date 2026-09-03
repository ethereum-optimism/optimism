//! The tokio-side handle to the circuit thread.

use tokio::sync::{mpsc, oneshot, watch};

use crate::{
    driver::ChainViewError,
    facts::Fact,
    snapshot::{ChainViewSnapshot, SafeHeadEntry},
};

/// Messages the circuit thread consumes.
#[derive(Debug)]
pub(crate) enum Msg {
    /// Push a fact (boxed: facts are far larger than queries).
    Fact(Box<Fact>),
    /// Answer a query from integrated state (after any facts queued before it).
    Query(ChainViewQuery),
    /// Stop after the current batch.
    Shutdown,
}

/// A point-in-time question answered from the driver's integrated state.
#[derive(Debug)]
pub enum ChainViewQuery {
    /// Answered once every message queued before it has been applied; a barrier.
    Sync {
        /// Where to send the acknowledgement.
        reply: oneshot::Sender<()>,
    },
    /// The safe L2 head after derivation consumed L1 block `number` (nearest at or before).
    SafeHeadAtL1 {
        /// The L1 block number.
        number: u64,
        /// Where to send the answer.
        reply: oneshot::Sender<Option<SafeHeadEntry>>,
    },
}

/// Pushes facts into the chain view and reads its snapshot.
#[derive(Debug, Clone)]
pub struct ChainViewClient {
    tx: mpsc::Sender<Msg>,
    snapshot: watch::Receiver<ChainViewSnapshot>,
}

impl ChainViewClient {
    pub(crate) const fn new(
        tx: mpsc::Sender<Msg>,
        snapshot: watch::Receiver<ChainViewSnapshot>,
    ) -> Self {
        Self { tx, snapshot }
    }

    /// Queues a fact; waits for channel capacity.
    pub async fn push(&self, fact: Fact) -> Result<(), ChainViewError> {
        self.tx.send(Msg::Fact(Box::new(fact))).await.map_err(|_| ChainViewError::Closed)
    }

    /// The latest published snapshot.
    pub fn snapshot(&self) -> ChainViewSnapshot {
        self.snapshot.borrow().clone()
    }

    /// A receiver that wakes on every published snapshot.
    pub fn subscribe(&self) -> watch::Receiver<ChainViewSnapshot> {
        self.snapshot.clone()
    }

    /// The safe L2 head recorded for the nearest derived-from L1 block at or below `number`.
    pub async fn safe_head_at_l1(
        &self,
        number: u64,
    ) -> Result<Option<SafeHeadEntry>, ChainViewError> {
        let (reply, rx) = oneshot::channel();
        self.tx
            .send(Msg::Query(ChainViewQuery::SafeHeadAtL1 { number, reply }))
            .await
            .map_err(|_| ChainViewError::Closed)?;
        rx.await.map_err(|_| ChainViewError::Closed)
    }

    /// Resolves once every fact queued before the call has been applied and stepped, so the
    /// snapshot read afterwards reflects them.
    pub async fn sync(&self) -> Result<(), ChainViewError> {
        let (reply, rx) = oneshot::channel();
        self.tx
            .send(Msg::Query(ChainViewQuery::Sync { reply }))
            .await
            .map_err(|_| ChainViewError::Closed)?;
        rx.await.map_err(|_| ChainViewError::Closed)
    }

    /// Asks the thread to stop after the current batch.
    pub fn try_shutdown(&self) -> Result<(), ChainViewError> {
        self.tx.try_send(Msg::Shutdown).map_err(|_| ChainViewError::Closed)
    }
}
