//! The actor that drives the verification round loop and routes its promotions.
//!
//! One actor for the whole process, because there is one verifier for the whole process: the
//! round it runs is a statement about every chain at one timestamp, and there is no per-chain
//! version of that question. It sits in the same actor set as the chains' own actors, so a
//! verifier that halts stops the supernode rather than leaving it serving chains whose cross-safe
//! heads have quietly stopped moving.
//!
//! ## What this does not yet do: finality
//!
//! L2 finality is still driven straight from L1 finality, by each chain's own finalize task,
//! without consulting the verified frontier. Under interop it should be gated on cross-safety —
//! a block cannot be irreversible before it has been verified — and it is not yet.
//!
//! The visible consequence is one-directional and bounded. If L1 finality ever overtakes the
//! verified frontier, the engine's promotion clamp holds the cross-safe head at the finalized
//! head instead of at the verified block, warning as it does so, and the chain reports a
//! `safe_l2` the verifier has not vouched for. In steady state the verifier runs seconds behind
//! the chains while L1 finality runs about two epochs behind, so the clamp does not fire; a cold
//! start that has to backfill a whole expiry window is where it can. Gating finality on the
//! frontier is the other half of this work and is deliberately left to a follow-up rather than
//! half-done here.

use crate::interop::{chain::NodeChain, query::InteropQuery};
use alloy_eips::BlockNumHash;
use alloy_primitives::ChainId;
use async_trait::async_trait;
use kona_engine::CrossSafePromoter;
use kona_node_service::{ChainControllerRequest, NodeActor};
use lokahi_interop::{Halted, InteropChain, Pace, RocksKv, StoreError, Verifier};
use std::{collections::BTreeMap, sync::Arc, time::Duration};
use tokio::sync::{mpsc, watch};
use tracing::{debug, info, warn};

/// How long to wait after a round that made no progress.
///
/// A no-op round means the chains have not reached the next timestamp yet, so the next useful
/// moment is a block time away. Shorter than a block time so the verifier is not the thing adding
/// latency, long enough that an idle cluster is not polled in a tight loop.
const IDLE_BACKOFF: Duration = Duration::from_millis(500);

/// How long to wait after a round that failed transiently.
///
/// Longer than the idle wait: the usual causes are an execution layer that is down or an L1 RPC
/// that timed out, and retrying those quickly helps nobody.
const RETRY_BACKOFF: Duration = Duration::from_secs(2);

/// The ways the interop actor can stop for good.
#[derive(Debug, thiserror::Error)]
pub(crate) enum InteropActorError {
    /// The verifier halted and will not resume.
    #[error(transparent)]
    Halted(#[from] Halted),
    /// The verified store could not be read.
    #[error("interop verified store: {0}")]
    Store(#[from] StoreError),
    /// A chain's controller is no longer accepting requests, so its cross-safe head can never be
    /// promoted again.
    #[error("chain {0}: the chain controller is gone, so its cross-safe head cannot be promoted")]
    ChainGone(ChainId),
}

/// One chain's promotion route: the seam to read it through, and the capability to move its
/// cross-safe head.
///
/// Holding the [`CrossSafePromoter`] here is what makes this actor that chain's only cross-safe
/// writer — there is exactly one promoter per engine and it is not [`Clone`], so the type system
/// carries the invariant rather than a comment.
pub(crate) struct ChainRoute {
    /// The chain, as the verifier reads it.
    pub(crate) chain: Arc<NodeChain>,
    /// The capability to mint this chain's cross-safe promotions.
    pub(crate) promoter: CrossSafePromoter,
    /// The chain controller's request channel, which applies them.
    pub(crate) requests: mpsc::Sender<ChainControllerRequest>,
}

/// Runs the verification round loop and applies each verified frontier to the chains.
pub(crate) struct InteropActor {
    /// The verifier, with the stores it owns.
    pub(super) verifier: Verifier<RocksKv>,
    /// Each chain's promotion route, by chain id.
    routes: BTreeMap<ChainId, ChainRoute>,
    /// Read-only questions from the RPC layer, drained around each round.
    ///
    /// [`None`] when nothing holds a read handle, which is the case for every use of this actor
    /// other than a supernode serving `superroot_atTimestamp`. See
    /// [`InteropActor::attach_queries`](crate::interop::InteropActor::attach_queries).
    pub(super) queries: Option<mpsc::Receiver<InteropQuery>>,
    /// Where the verifier's L1 progress is republished after each round, for readers that need
    /// that and nothing else. [`None`] alongside [`Self::queries`].
    pub(super) current_l1: Option<watch::Sender<BlockNumHash>>,
    /// The block each chain's cross-safe head was last promoted to.
    ///
    /// Timestamps advance one second at a time while blocks are a block time apart, so most
    /// rounds re-verify the block the previous one did. Without this, every round would put an
    /// identical forkchoice update on the wire.
    promoted: BTreeMap<ChainId, BlockNumHash>,
}

impl core::fmt::Debug for InteropActor {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("InteropActor")
            .field("chains", &self.routes.keys().collect::<Vec<_>>())
            .field("state", &self.verifier.state())
            .finish_non_exhaustive()
    }
}

impl InteropActor {
    /// Builds the actor over a verifier and the chains it promotes.
    ///
    /// The routes are keyed by chain id here rather than by the caller, so the key can only ever
    /// be the id the route's own chain reports.
    pub(super) fn new(verifier: Verifier<RocksKv>, routes: Vec<ChainRoute>) -> Self {
        let routes = routes.into_iter().map(|route| (route.chain.chain_id(), route)).collect();
        Self { verifier, routes, queries: None, current_l1: None, promoted: BTreeMap::new() }
    }

    /// Answers every query already waiting, without blocking on more.
    ///
    /// Run before each round so a caller that arrived during the previous backoff is not made to
    /// wait for a second round.
    fn answer_pending(&mut self) {
        let Some(queries) = self.queries.as_mut() else { return };
        // `try_recv` also reports a closed queue, which is not an error here: nothing holding a
        // read handle is the normal state, and the actor's job is unaffected.
        let mut pending = Vec::new();
        while let Ok(query) = queries.try_recv() {
            pending.push(query);
        }
        for query in pending {
            self.answer(query);
        }
    }

    /// Waits `wait`, answering queries as they arrive.
    ///
    /// The backoff is where this actor spends nearly all of its time, so serving queries through
    /// it is what keeps read latency at "the current round" rather than "the next idle moment".
    async fn wait_answering(&mut self, wait: Duration) {
        let deadline = tokio::time::sleep(wait);
        tokio::pin!(deadline);
        loop {
            // `recv` on a `None` queue would be a future that never completes, which is exactly
            // what the `else` branch below wants — but borrowing `self` twice inside `select!` is
            // not expressible, so the two cases are split.
            let Some(queries) = self.queries.as_mut() else {
                deadline.await;
                return;
            };
            tokio::select! {
                () = &mut deadline => return,
                query = queries.recv() => match query {
                    Some(query) => self.answer(query),
                    // Every read handle is gone. Nothing more will arrive, so wait out the rest
                    // of the backoff plainly.
                    None => {
                        self.queries = None;
                        deadline.await;
                        return;
                    }
                },
            }
        }
    }

    /// Promotes every chain's cross-safe head to the verified frontier, if there is one.
    ///
    /// Runs after every round rather than only after an advance, so a frontier committed by a
    /// previous process is promoted once on startup: the verified store is durable and the engine
    /// heads are not, so after a restart the two disagree until this reconciles them.
    ///
    /// A chain whose block cannot be read right now is left for the next round. Promotions are
    /// independent per chain — the frontier is already decided, and holding back the chains that
    /// *can* be promoted would only widen the gap.
    async fn promote(&mut self) -> Result<(), InteropActorError> {
        let Some(timestamp) = self.verifier.verified().last_timestamp() else { return Ok(()) };
        let frontier = self.verifier.verified().get(timestamp)?;

        for (&chain_id, &block) in &frontier.l2_heads {
            if self.promoted.get(&chain_id) == Some(&block) {
                continue;
            }
            let Some(route) = self.routes.get(&chain_id) else {
                // The verifier's chain set and this actor's routes are built from one list, so
                // this cannot happen; reporting it beats promoting the wrong chain.
                warn!(
                    target: "lokahi_interop",
                    chain_id,
                    "Verified frontier names a chain with no promotion route"
                );
                continue;
            };

            // The promoted head is a block, not a number: the engine records the cross-safe head
            // as a full `L2BlockInfo`, and the one authority on what that block is, is the chain
            // itself.
            let info = match route.chain.block_at(block.number).await {
                Ok(info) => info,
                Err(err) => {
                    debug!(
                        target: "lokahi_interop",
                        chain_id,
                        number = block.number,
                        %err,
                        "Cannot read the verified block yet; deferring its promotion"
                    );
                    continue;
                }
            };
            if info.block_info.hash != block.hash {
                // The chain reorged below a block the verifier declared cross-safe. The verifier
                // will see it too, on its own next round, and this must not hand the engine a
                // block off the branch that was verified.
                warn!(
                    target: "lokahi_interop",
                    chain_id,
                    number = block.number,
                    verified = %block.hash,
                    canonical = %info.block_info.hash,
                    "Canonical block at the verified height differs; withholding the promotion"
                );
                continue;
            }

            route
                .requests
                .send(ChainControllerRequest::PromoteCrossSafe(Box::new(
                    route.promoter.promote(info),
                )))
                .await
                .map_err(|_| InteropActorError::ChainGone(chain_id))?;

            info!(
                target: "lokahi_interop",
                chain_id,
                timestamp,
                number = block.number,
                "Promoted the cross-safe head to the verified frontier"
            );
            self.promoted.insert(chain_id, block);
        }

        Ok(())
    }
}

#[async_trait]
impl NodeActor for InteropActor {
    type Error = InteropActorError;

    /// Runs one round and applies its result.
    ///
    /// The pace the round asks for is honoured here rather than inside the verifier, because it
    /// is the only thing in this loop that is about scheduling: a round that advanced runs the
    /// next one immediately, a no-op round waits for the world to change, and a transient failure
    /// waits longer. Only a halt is fatal, and it is fatal on purpose — a supernode whose
    /// verifier has stopped is no longer answering the question it exists to answer.
    async fn step(&mut self) -> Result<(), Self::Error> {
        self.answer_pending();
        let pace = self.verifier.step().await?;
        self.publish_current_l1();
        self.promote().await?;

        match pace {
            // An immediate round is a catch-up round, and a run of them can be long. Queries are
            // answered at the top of the next step rather than here, which bounds a reader's wait
            // at one round instead of at the end of the catch-up.
            Pace::Immediate => {}
            Pace::Idle => self.wait_answering(IDLE_BACKOFF).await,
            Pace::Retry => self.wait_answering(RETRY_BACKOFF).await,
        }
        Ok(())
    }
}
