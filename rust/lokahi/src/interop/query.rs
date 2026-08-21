//! Reading the verifier from outside its own actor.
//!
//! The verifier owns its stores by value and is stepped by `&mut self`, so there is no second
//! handle to hand the RPC layer: rocksdb takes a directory lock, and opening the verified store
//! twice would fail even if the types allowed it. The RPC therefore *asks* the actor, over a
//! channel the actor drains around its rounds, and the actor answers from the one instance.
//!
//! That makes every answer a consistent snapshot for free. op-supernode reaches the same
//! guarantee with an `RWMutex` held across the `(currentL1, verifiedDB)` pair and documents why
//! it has to: a reader that samples the two separately can straddle a commit and report an L1 the
//! frontier it returned was not yet derived from. Here both come from one turn of a
//! single-threaded actor, so there is no window to close.
//!
//! The cost is latency. A query submitted while a round is in flight waits for that round, which
//! is one L1/EL round trip in the worst case; queries are otherwise served immediately, both
//! before a round starts and throughout the actor's backoff. That is a bound on freshness, not on
//! correctness, and the consumers of these RPCs poll.

use crate::interop::actor::InteropActor;
use alloy_eips::BlockNumHash;
use lokahi_interop::{StoreError, VerifiedResult};
use tokio::sync::{mpsc, oneshot, watch};

/// How many queries may be waiting on the actor at once.
///
/// Small on purpose: these are answered between rounds, so a deep queue would only let callers
/// pile up behind a stale round rather than being told to retry.
const QUERY_QUEUE: usize = 64;

/// A read-only question for the interop actor.
#[derive(Debug)]
pub(crate) enum InteropQuery {
    /// What the verifier can say about one L2 timestamp, with its L1 progress read in the same
    /// turn.
    VerifiedAt {
        /// The L2 timestamp being asked about.
        timestamp: u64,
        /// Where the answer goes.
        sender: oneshot::Sender<Result<VerifiedAt, StoreError>>,
    },
}

/// What the verifier can say about one L2 timestamp.
///
/// The variants are op-supernode's `VerifiedResultAtTimestamp` return classes, which the superroot
/// response shape depends on: two of them mean "the optimistic outputs are canonical, compose the
/// super root from them", one means "wait", and one means "here it is". Collapsing any of them
/// into an absent answer is how a consumer ends up treating a pre-activation timestamp as
/// unverifiable forever.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum Verdict {
    /// A frontier is committed at the timestamp. op-supernode: a `VerifiedResult` and no error.
    Verified(VerifiedResult),
    /// Interop covers the timestamp and the verifier will get to it, but has not yet.
    /// op-supernode: `ethereum.NotFound`.
    NotYet,
    /// The timestamp is before the interop activation, so pre-interop consensus covers it.
    /// op-supernode: `ErrNotActive`.
    NotActive,
    /// The timestamp is at or after activation but below the first timestamp this verifier
    /// covers, so the safe-head handoff covers it. op-supernode: `ErrBeforeVerifiedDB`.
    BeforeVerified,
    /// The verifier has not chosen where to start yet, so it can say nothing.
    /// op-supernode: `ErrNotStarted`.
    NotStarted,
}

/// A verdict about one timestamp together with the verifier's L1 progress.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) struct VerifiedAt {
    /// The L1 block the verifier has considered up to.
    ///
    /// Zero before its first advance, which is not a placeholder: op-supernode's `currentL1`
    /// starts zero and is folded into the aggregate as-is, so a supernode whose verifier has not
    /// advanced reports `current_l1` zero and every consumer gating on L1 progress waits. Keeping
    /// that means a lokahi supernode is not mistaken for one that is further along.
    pub(crate) current_l1: BlockNumHash,
    /// What the verifier can say about the timestamp.
    pub(crate) verdict: Verdict,
}

/// The ways an interop read can fail.
#[derive(Debug, thiserror::Error)]
pub(crate) enum InteropQueryError {
    /// The actor is not answering: it has stopped, or has not started stepping yet.
    #[error("the interop verifier is not answering queries")]
    Unreachable,
    /// The verified store could not be read.
    #[error("interop verified store: {0}")]
    Store(#[from] StoreError),
}

/// The read handle the RPC layer holds on the interop verifier.
///
/// Two of the three things it can answer need no round trip. The activation timestamp is
/// immutable configuration, so a pre-activation query is answered here — as op-supernode does,
/// checking activation before taking any lock — and cannot be delayed by a round or fail because
/// the actor stopped. The verifier's L1 progress is published on a watch channel after every
/// round, so `supernode_syncStatus`, which needs it and nothing else from the verifier, asks the
/// actor for nothing at all.
///
/// Only reading the frontier at a timestamp goes through the queue, because only that needs the
/// store the actor owns.
#[derive(Debug, Clone)]
pub(crate) struct InteropReader {
    /// The timestamp the cluster activates interop at.
    activation: u64,
    /// The L1 block the verifier has considered up to, republished after each round.
    current_l1: watch::Receiver<BlockNumHash>,
    /// The actor's query queue.
    queries: mpsc::Sender<InteropQuery>,
}

impl InteropReader {
    /// Returns the L1 block the verifier has considered up to.
    ///
    /// Zero before its first advance. op-supernode folds exactly this value into the per-chain
    /// minimum for every chain the verifier is registered on, so a supernode whose verifier has
    /// not advanced reports `current_l1` zero — see [`VerifiedAt::current_l1`].
    pub(crate) fn current_l1(&self) -> BlockNumHash {
        *self.current_l1.borrow()
    }

    /// Returns the verifier's verdict on `timestamp`.
    pub(crate) async fn verified_at(
        &self,
        timestamp: u64,
    ) -> Result<VerifiedAt, InteropQueryError> {
        // Before activation there is nothing for the verifier to have verified, and no reason to
        // wait on it to say so.
        if timestamp < self.activation {
            return Ok(VerifiedAt { current_l1: self.current_l1(), verdict: Verdict::NotActive });
        }

        let (sender, receiver) = oneshot::channel();
        self.queries
            .send(InteropQuery::VerifiedAt { timestamp, sender })
            .await
            .map_err(|_| InteropQueryError::Unreachable)?;
        receiver.await.map_err(|_| InteropQueryError::Unreachable)?.map_err(Into::into)
    }
}

impl InteropActor {
    /// Creates the actor's query queue, returning the read handle for the RPC layer.
    ///
    /// Called once, when the actor is built: the sender is what the RPC layer holds and the
    /// receiver is what the actor drains, so an actor that was never given a queue cannot be
    /// queried and one that has one cannot be queried through a second.
    pub(crate) fn attach_queries(&mut self, activation: u64) -> InteropReader {
        let (queries, receiver) = mpsc::channel(QUERY_QUEUE);
        let (current_l1, current_l1_rx) =
            watch::channel(self.verifier.current_l1().unwrap_or_default());
        self.queries = Some(receiver);
        self.current_l1 = Some(current_l1);
        InteropReader { activation, current_l1: current_l1_rx, queries }
    }

    /// Republishes the verifier's L1 progress for readers that need only that.
    pub(super) fn publish_current_l1(&self) {
        if let Some(current_l1) = &self.current_l1 {
            // `send_replace` rather than `send`: there may be no reader, and that is not a reason
            // to stop republishing.
            current_l1.send_replace(self.verifier.current_l1().unwrap_or_default());
        }
    }

    /// Answers `query` from the verifier's current state.
    pub(super) fn answer(&self, query: InteropQuery) {
        match query {
            InteropQuery::VerifiedAt { timestamp, sender } => {
                // A dropped receiver means the caller gave up; nothing to report.
                let _ = sender.send(self.verified_at(timestamp));
            }
        }
    }

    /// The verifier's verdict on one timestamp, read in a single turn.
    ///
    /// The order of the checks is op-supernode's `VerifiedResultAtTimestamp`: whether the verifier
    /// has started, then whether the timestamp is below the range it covers, then the store. The
    /// activation check that precedes all of them lives in [`InteropReader::verified_at`], which
    /// never reaches this far.
    fn verified_at(&self, timestamp: u64) -> Result<VerifiedAt, StoreError> {
        let current_l1 = self.verifier.current_l1().unwrap_or_default();
        let verdict = match self.verifier.first_verifiable_timestamp() {
            None => Verdict::NotStarted,
            Some(first) if timestamp < first => Verdict::BeforeVerified,
            Some(_) => match self.verifier.verified().get(timestamp) {
                Ok(result) => Verdict::Verified(result),
                Err(StoreError::NotFound) => Verdict::NotYet,
                Err(err) => return Err(err),
            },
        };
        Ok(VerifiedAt { current_l1, verdict })
    }
}
