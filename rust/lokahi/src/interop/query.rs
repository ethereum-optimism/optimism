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
use alloy_primitives::ChainId;
use lokahi_interop::{StoreError, VerifiedResult};
use tokio::sync::{mpsc, oneshot, watch};
use tracing::info;

/// How many queries may be waiting on the actor at once.
///
/// Small on purpose: these are answered between rounds, so a deep queue would only let callers
/// pile up behind a stale round rather than being told to retry.
const QUERY_QUEUE: usize = 64;

/// A question for the interop actor, or an instruction to it.
///
/// Almost all of these are reads. [`Self::SetPause`] is not, and it travels the same queue on
/// purpose: the pause it sets has to take effect between two of the actor's rounds, never inside
/// one, or a test that pauses at a timestamp could still see that timestamp committed by the round
/// that was already in flight.
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
    /// The verifier's test-visible progress.
    Status {
        /// Where the answer goes.
        sender: oneshot::Sender<InteropStatus>,
    },
    /// How far one chain's interop log store extends.
    SealedBlocks {
        /// The chain being asked about.
        chain_id: ChainId,
        /// Where the answer goes.
        sender: oneshot::Sender<Result<SealedBlocks, SealedBlocksError>>,
    },
    /// Stop the round loop at a timestamp, or clear an existing stop.
    SetPause {
        /// The timestamp to stop at, or [`None`] to resume.
        timestamp: Option<u64>,
        /// Acknowledgement, so the caller knows the pause is in force before it returns.
        sender: oneshot::Sender<()>,
    },
}

/// The verifier's test-visible progress, as one snapshot.
///
/// One snapshot rather than a field per call because that is how it is read: "has cold start
/// finished, and where did verification start" is one question, and a round trip per field would
/// let the answers disagree with each other. The Go counterpart is
/// `eth.SupernodeInteropStatus`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct InteropStatus {
    /// How many cold-start attempts have been made in this process.
    pub(crate) backfill_attempts: u32,
    /// Whether cold start has finished, by backfilling or by resuming off existing state.
    pub(crate) backfill_completed: bool,
    /// The timestamp the cluster activates interop at.
    pub(crate) activation_timestamp: u64,
    /// The timestamp the round loop began at, or zero while cold-starting.
    pub(crate) verification_start_timestamp: u64,
    /// The lowest timestamp the verifier covers, or zero while cold-starting.
    pub(crate) first_verifiable_timestamp: u64,
}

/// One block a chain's log store has sealed.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub(crate) struct SealedBlock {
    /// The block's number and hash.
    pub(crate) id: BlockNumHash,
    /// The block's L2 timestamp.
    pub(crate) timestamp: u64,
}

/// The extent of one chain's log store: its earliest and most recent sealed blocks.
///
/// Both ends arrive together because the assertion they serve is about the span between them —
/// that backfill reached back far enough and handed off far enough forward — and reading the ends
/// in separate calls would let the store move underneath the comparison. The Go counterpart is
/// `eth.SupernodeSealedBlocks`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub(crate) struct SealedBlocks {
    /// The earliest sealed block. Meaningful only when [`Self::has_blocks`].
    pub(crate) first: SealedBlock,
    /// The most recent sealed block. Meaningful only when [`Self::has_blocks`].
    pub(crate) latest: SealedBlock,
    /// Whether the store holds any sealed block at all.
    ///
    /// An empty store is not an error: a verifier that has backfilled nothing for a chain answers
    /// truthfully rather than failing, which is what lets a test distinguish "nothing sealed yet"
    /// from "this chain is not one I follow".
    pub(crate) has_blocks: bool,
}

/// Why one chain's sealed range could not be read.
#[derive(Debug, thiserror::Error)]
pub(crate) enum SealedBlocksError {
    /// The chain is not one this verifier follows, so there is no store to report on.
    #[error("chain {0} is not followed by this supernode's interop verifier")]
    UnknownChain(ChainId),
    /// The store is there and could not be read.
    #[error("interop log store of chain {chain_id}: {source}")]
    Store {
        /// The chain whose store failed.
        chain_id: ChainId,
        /// What the store said.
        #[source]
        source: StoreError,
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
    /// One chain's sealed range could not be read.
    #[error(transparent)]
    SealedBlocks(#[from] SealedBlocksError),
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

        self.ask(|sender| InteropQuery::VerifiedAt { timestamp, sender }).await?.map_err(Into::into)
    }

    /// Returns the verifier's test-visible progress.
    pub(crate) async fn status(&self) -> Result<InteropStatus, InteropQueryError> {
        self.ask(|sender| InteropQuery::Status { sender }).await
    }

    /// Returns how far one chain's interop log store extends.
    pub(crate) async fn sealed_blocks(
        &self,
        chain_id: ChainId,
    ) -> Result<SealedBlocks, InteropQueryError> {
        self.ask(|sender| InteropQuery::SealedBlocks { chain_id, sender }).await?.map_err(Into::into)
    }

    /// Stops the round loop at `timestamp`, or clears an existing stop with [`None`].
    ///
    /// Returns once the actor has applied it, so a caller that pauses and then reads knows it is
    /// reading a paused verifier rather than racing the round that was in flight.
    pub(crate) async fn set_pause(&self, timestamp: Option<u64>) -> Result<(), InteropQueryError> {
        self.ask(|sender| InteropQuery::SetPause { timestamp, sender }).await
    }

    /// Sends one query and waits for its answer.
    ///
    /// A closed queue and a dropped answer channel are the same condition — the actor is gone —
    /// and are reported as one, because a caller can do nothing different about either.
    async fn ask<T>(
        &self,
        query: impl FnOnce(oneshot::Sender<T>) -> InteropQuery,
    ) -> Result<T, InteropQueryError> {
        let (sender, receiver) = oneshot::channel();
        self.queries.send(query(sender)).await.map_err(|_| InteropQueryError::Unreachable)?;
        receiver.await.map_err(|_| InteropQueryError::Unreachable)
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

    /// Answers `query` from the verifier's current state, or applies it to the verifier.
    ///
    /// Every arm ignores a send failure: a dropped receiver means the caller gave up between
    /// asking and being answered, and there is nobody left to report that to.
    pub(super) fn answer(&mut self, query: InteropQuery) {
        match query {
            InteropQuery::VerifiedAt { timestamp, sender } => {
                let _ = sender.send(self.verified_at(timestamp));
            }
            InteropQuery::Status { sender } => {
                let _ = sender.send(self.status());
            }
            InteropQuery::SealedBlocks { chain_id, sender } => {
                let _ = sender.send(self.sealed_blocks(chain_id));
            }
            InteropQuery::SetPause { timestamp, sender } => {
                self.verifier.set_pause(timestamp);
                info!(
                    target: "lokahi_interop",
                    pause_at = timestamp,
                    "Interop verification pause updated by test control"
                );
                let _ = sender.send(());
            }
        }
    }

    /// The verifier's test-visible progress, read in a single turn.
    ///
    /// The zeroes are op-supernode's: `VerificationStartTimestamp` and `FirstVerifiableTimestamp`
    /// both report zero before cold start completes rather than an absent value, because the Go
    /// wire type they cross is a plain `uint64`.
    fn status(&self) -> InteropStatus {
        let start = self.verifier.verification_start();
        InteropStatus {
            backfill_attempts: self.verifier.backfill_attempts(),
            // Cold start is finished exactly when a start has been chosen — whether backfill ran
            // or the verifier resumed off existing state and skipped it. That is what
            // op-supernode's `backfillCompleted` means too.
            backfill_completed: start.is_some(),
            activation_timestamp: self.verifier.activation_timestamp(),
            verification_start_timestamp: start.unwrap_or_default(),
            first_verifiable_timestamp: self.verifier.first_verifiable_timestamp().unwrap_or_default(),
        }
    }

    /// How far one chain's log store extends, read in a single turn.
    ///
    /// An empty store is reported rather than raised: only an unknown chain is an error, which is
    /// the same split op-supernode's `InteropSealedBlocks` makes. The latest block is looked up
    /// first, because it is the one read that distinguishes the two cases without failing.
    fn sealed_blocks(&self, chain_id: ChainId) -> Result<SealedBlocks, SealedBlocksError> {
        let store = self
            .verifier
            .logs(chain_id)
            .ok_or(SealedBlocksError::UnknownChain(chain_id))?;
        let store_err = |source| SealedBlocksError::Store { chain_id, source };

        let Some(latest) = store.latest_sealed_block() else {
            return Ok(SealedBlocks::default());
        };
        let latest = store.find_sealed_block(latest.number).map_err(store_err)?;
        let first = store.first_sealed_block().map_err(store_err)?;
        Ok(SealedBlocks {
            first: SealedBlock { id: first.id(), timestamp: first.timestamp },
            latest: SealedBlock { id: latest.id(), timestamp: latest.timestamp },
            has_blocks: true,
        })
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
