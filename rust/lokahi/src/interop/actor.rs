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
use kona_protocol::L2BlockInfo;
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
        // The queue is taken out of `self` for the wait and put back after it. Answering needs
        // `&mut self` — a pause instruction reaches the verifier — and that cannot overlap a
        // borrow of the field the instruction arrived through.
        let Some(mut queries) = self.queries.take() else {
            tokio::time::sleep(wait).await;
            return;
        };

        let deadline = tokio::time::sleep(wait);
        tokio::pin!(deadline);
        loop {
            tokio::select! {
                () = &mut deadline => break,
                query = queries.recv() => match query {
                    Some(query) => self.answer(query),
                    // Every read handle is gone. Nothing more will arrive, so wait out the rest
                    // of the backoff plainly and leave the queue behind.
                    None => {
                        deadline.await;
                        return;
                    }
                },
            }
        }
        self.queries = Some(queries);
    }

    /// Promotes every chain's cross-safe head: to the verified frontier when one is committed,
    /// and through op-node's pre-activation and anchor fall-throughs when none is.
    ///
    /// Runs after every round rather than only after an advance, so a frontier committed by a
    /// previous process is promoted once on startup: the verified store is durable and the engine
    /// heads are not, so after a restart the two disagree until this reconciles them.
    ///
    /// The branch order per chain is op-node's `SafeL2Head` over op-supernode's
    /// `FullyVerifiedL2Head` (`op-node/rollup/engine/engine_controller.go:219-241`,
    /// `op-supernode/supernode/chain_container/super_authority.go:23-41`):
    ///
    /// 1. **Pre-activation** — the verifier is not active at the chain's local-safe timestamp, so
    ///    the cross-safe head follows the local-safe head. Without this branch, cross-safe — and
    ///    with it finalization, which kona's `FinalizeTask` clamps at cross-safe — would sit at
    ///    genesis until activation, however far away activation is.
    /// 2. **Verified** — a frontier is committed: promote the verified tip, exactly as before.
    /// 3. **Anchor** — active but nothing verified yet: promote the canonical block at `activation
    ///    - 1` (which `block_number_at_timestamp` floors to genesis when activation is at or before
    ///    it), bounded by the local-safe head — op-node's `resolveAnchorAsSafe`
    ///    (`engine_controller.go:273-290`).
    ///
    /// A chain whose head or block cannot be read right now is left for the next round.
    /// Promotions are independent per chain — the target is already decided, and holding back the
    /// chains that *can* be promoted would only widen the gap.
    async fn promote(&mut self) -> Result<(), InteropActorError> {
        let frontier = match self.verifier.verified().last_timestamp() {
            Some(timestamp) => Some(self.verifier.verified().get(timestamp)?),
            None => None,
        };
        let activation = self.verifier.activation_timestamp();

        for (&chain_id, route) in &self.routes {
            // Every branch starts from the chain's local-safe head: pre-activation it is the
            // target itself, and the anchor is bounded by it. A chain that cannot answer is
            // deferred to the next round, like every other transient here.
            let local_safe = match route.chain.local_safe_head().await {
                Ok(head) => head,
                Err(err) => {
                    debug!(
                        target: "lokahi_interop",
                        chain_id,
                        %err,
                        "Cannot read the local-safe head yet; deferring the chain's promotion"
                    );
                    continue;
                }
            };
            // Nothing derived or discovered yet: op-node's fall-throughs never see this state —
            // its engine controller starts from a found forkchoice — and promoting the zero
            // block would put a forkchoice update naming no real block on the wire.
            if local_safe == L2BlockInfo::default() {
                continue;
            }

            let (target, source) = if local_safe.block_info.timestamp < activation {
                // The verifier is not active at the local-safe timestamp: cross-safe follows
                // local-safe (`VerifierHeadPreActivation`). Checked before the frontier, in
                // op-supernode's order.
                (local_safe, "pre-activation local-safe")
            } else if let Some(frontier) = &frontier {
                let Some(&block) = frontier.l2_heads.get(&chain_id) else {
                    // The verifier's chain set and this actor's routes are built from one list,
                    // so this cannot happen; reporting it beats promoting the wrong chain.
                    warn!(
                        target: "lokahi_interop",
                        chain_id,
                        "Verified frontier names no block for this chain"
                    );
                    continue;
                };
                if self.promoted.get(&chain_id) == Some(&block) {
                    continue;
                }

                // The promoted head is a block, not a number: the engine records the cross-safe
                // head as a full `L2BlockInfo`, and the one authority on what that block is, is
                // the chain itself.
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
                    // The chain reorged below a block the verifier declared cross-safe. The
                    // verifier will see it too, on its own next round, and this must not hand
                    // the engine a block off the branch that was verified.
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
                (info, "verified frontier")
            } else {
                // Active but nothing verified yet: the anchor is the canonical block at
                // `activation - 1`, clamped to genesis (`VerifierHeadAnchor`;
                // `block_number_at_timestamp` floors a pre-genesis timestamp onto genesis, the
                // clamp op-supernode's `verifierContribution` applies).
                let number =
                    match route.chain.block_number_at_timestamp(activation.saturating_sub(1)).await
                    {
                        Ok(number) => number,
                        Err(err) => {
                            debug!(
                                target: "lokahi_interop",
                                chain_id,
                                %err,
                                "Cannot resolve the anchor height yet; deferring the promotion"
                            );
                            continue;
                        }
                    };
                if number > local_safe.block_info.number {
                    // Local-safe has not reached the anchor: hold at local-safe, op-node's
                    // `resolveAnchorAsSafe` bound. Unreachable when the head and its timestamp
                    // come from one snapshot, as they do here, but kept for parity.
                    (local_safe, "anchor above local-safe")
                } else {
                    match route.chain.block_at(number).await {
                        Ok(info) => (info, "activation anchor"),
                        Err(err) => {
                            debug!(
                                target: "lokahi_interop",
                                chain_id,
                                number,
                                %err,
                                "Cannot read the anchor block yet; deferring its promotion"
                            );
                            continue;
                        }
                    }
                }
            };

            let block = target.block_info.id();
            if self.promoted.get(&chain_id) == Some(&block) {
                continue;
            }

            route
                .requests
                .send(ChainControllerRequest::PromoteCrossSafe(Box::new(
                    route.promoter.promote(target),
                )))
                .await
                .map_err(|_| InteropActorError::ChainGone(chain_id))?;

            info!(
                target: "lokahi_interop",
                chain_id,
                source,
                number = block.number,
                "Promoted the cross-safe head"
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::interop::ChainInterop;
    use alloy_eips::BlockNumberOrTag;
    use alloy_primitives::B256;
    use alloy_provider::RootProvider;
    use kona_engine::{
        EngineQueries, EngineState, EngineSyncStateUpdate, LocalSafeHead, OpEngineClient,
    };
    use kona_genesis::{ChainGenesis, HardForkConfig, RollupConfig};
    use kona_protocol::{BlockInfo, L2BlockInfo, OutputRoot};
    use op_alloy_network::Optimism;

    /// The chain these tests promote.
    const CHAIN_ID: ChainId = 901;
    /// Genesis at `t = 1_000` with two-second blocks.
    const GENESIS_TIME: u64 = 1_000;

    /// The canonical block at `number` on the test chain: a distinct non-zero hash, and the
    /// timestamp the two-second spacing implies.
    fn block(number: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                number,
                hash: B256::with_last_byte(number as u8 + 1),
                parent_hash: B256::with_last_byte(number as u8),
                timestamp: GENESIS_TIME + number * 2,
            },
            ..Default::default()
        }
    }

    /// An engine state whose local-safe (and unsafe) head is `local_safe`.
    fn engine_state(local_safe: L2BlockInfo) -> EngineState {
        let state = EngineState { chain_id: CHAIN_ID, ..Default::default() };
        let sync_state = state.apply_sync_update(EngineSyncStateUpdate {
            unsafe_head: Some(local_safe),
            local_safe_head: Some(LocalSafeHead::unpaired(local_safe)),
            ..Default::default()
        });
        EngineState { sync_state, ..state }
    }

    /// Answers the chain controller queries `promote` asks: the engine state from the watch, and
    /// the canonical block at any asked height, shaped by [`block`].
    fn serve_engine(
        mut queries: mpsc::Receiver<kona_node_service::ChainControllerRpcRequest>,
        state: watch::Receiver<EngineState>,
    ) {
        tokio::spawn(async move {
            while let Some(request) = queries.recv().await {
                let current = *state.borrow();
                match *request.0 {
                    EngineQueries::State(sender) => {
                        let _ = sender.send(current);
                    }
                    EngineQueries::OutputAtBlock { block: asked, sender } => {
                        let BlockNumberOrTag::Number(number) = asked else { continue };
                        let info = block(number);
                        let output = OutputRoot {
                            state_root: B256::ZERO,
                            bridge_storage_root: B256::ZERO,
                            block_hash: info.block_info.hash,
                        };
                        let _ = sender.send((info, output, current));
                    }
                    _ => {}
                }
            }
        });
    }

    /// An actor over one chain, with the seams a test drives it through: the engine-state watch
    /// the responder serves from, and the controller request queue promotions land on.
    fn actor_with_one_chain(
        root: &std::path::Path,
        activation: u64,
    ) -> (InteropActor, watch::Sender<EngineState>, mpsc::Receiver<ChainControllerRequest>) {
        let datadir = root.join(CHAIN_ID.to_string());
        std::fs::create_dir_all(&datadir).expect("chain datadir");

        let (queries, queries_rx) = mpsc::channel(8);
        let (l1_queries, _l1_queries_rx) = mpsc::channel(8);
        let (requests, requests_rx) = mpsc::channel(8);
        let (state_tx, state_rx) = watch::channel(EngineState::default());
        serve_engine(queries_rx, state_rx);

        let chain = ChainInterop {
            chain_id: CHAIN_ID,
            safe_db: ChainInterop::open_safe_db(&datadir).expect("safe db"),
            archive: ChainInterop::open_archive(&datadir).expect("archive"),
            datadir,
            rollup_config: RollupConfig {
                block_time: 2,
                genesis: ChainGenesis {
                    l2: block(0).block_info.id(),
                    l2_time: GENESIS_TIME,
                    ..Default::default()
                },
                hardforks: HardForkConfig { lagoon_time: Some(activation), ..Default::default() },
                l2_chain_id: CHAIN_ID.into(),
                ..Default::default()
            },
            el: RootProvider::<Optimism>::new_http(url::Url::parse("http://127.0.0.1:1/").unwrap()),
            queries,
            l1_queries,
            requests,
            promoter: promoter(),
        };

        let l1 = url::Url::parse("http://127.0.0.1:1/").unwrap();
        let actor =
            InteropActor::build(Some(&root.join("interop")), &l1, activation, None, vec![chain])
                .expect("actor builds");
        (actor, state_tx, requests_rx)
    }

    /// A promoter, obtained the only way one can be: from an engine built to be fed externally.
    fn promoter() -> CrossSafePromoter {
        let (state_tx, _state_rx) = watch::channel(EngineState::default());
        let (len_tx, _len_rx) = watch::channel(0usize);
        kona_engine::Engine::<OpEngineClient<RootProvider, RootProvider<Optimism>>>::with_external_cross_safe(
            EngineState::default(),
            state_tx,
            len_tx,
        )
        .1
    }

    /// The block a received promotion names, or [`None`] when nothing was promoted.
    fn promoted_block(
        requests_rx: &mut mpsc::Receiver<ChainControllerRequest>,
    ) -> Option<L2BlockInfo> {
        match requests_rx.try_recv() {
            Ok(ChainControllerRequest::PromoteCrossSafe(promotion)) => Some(promotion.target()),
            Ok(other) => panic!("expected a cross-safe promotion, got {other:?}"),
            Err(_) => None,
        }
    }

    /// Before activation, the cross-safe head follows the local-safe head — op-node's
    /// `VerifierHeadPreActivation` fall-through (`engine_controller.go:232-233`). Without it,
    /// cross-safe (and the finalization kona clamps at it) sits at genesis until activation:
    /// `TestPreNoInbox` and `TestSupernodeResyncSchedulesAtActivation_PreActivation` assert
    /// exactly this advance.
    #[tokio::test]
    async fn pre_activation_the_cross_safe_head_follows_local_safe() {
        let root = tempfile::tempdir().expect("temp dir");
        // Activation is well in the future, as the pre-activation acceptance tests set it.
        let (mut actor, state_tx, mut requests_rx) =
            actor_with_one_chain(root.path(), GENESIS_TIME + 3_600);

        state_tx.send(engine_state(block(5))).expect("state");
        actor.promote().await.expect("promote");
        assert_eq!(promoted_block(&mut requests_rx), Some(block(5)));

        // The same head is not re-promoted: an idle cluster puts nothing on the wire.
        actor.promote().await.expect("promote");
        assert_eq!(promoted_block(&mut requests_rx), None);

        // A local-safe advance is re-promoted, so cross-safe keeps following it.
        state_tx.send(engine_state(block(6))).expect("state");
        actor.promote().await.expect("promote");
        assert_eq!(promoted_block(&mut requests_rx), Some(block(6)));
    }

    /// A chain that has not discovered any head yet is skipped rather than promoted to the zero
    /// block: there is nothing to promote until the engine has found its forkchoice.
    #[tokio::test]
    async fn a_defaulted_local_safe_head_is_not_promoted() {
        let root = tempfile::tempdir().expect("temp dir");
        let (mut actor, state_tx, mut requests_rx) =
            actor_with_one_chain(root.path(), GENESIS_TIME + 3_600);

        state_tx.send(EngineState::default()).expect("state");
        actor.promote().await.expect("promote");
        assert_eq!(promoted_block(&mut requests_rx), None);
    }

    /// Active with nothing verified yet: the anchor is the canonical block at `activation - 1`,
    /// clamped to genesis — op-node's `VerifierHeadAnchor` resolution
    /// (`engine_controller.go:273-290`, `super_authority.go:76-87`). With activation at genesis,
    /// the anchor is the genesis block itself.
    #[tokio::test]
    async fn an_unverified_active_chain_is_anchored_at_genesis_for_a_genesis_activation() {
        let root = tempfile::tempdir().expect("temp dir");
        let (mut actor, state_tx, mut requests_rx) =
            actor_with_one_chain(root.path(), GENESIS_TIME);

        state_tx.send(engine_state(block(5))).expect("state");
        actor.promote().await.expect("promote");
        assert_eq!(promoted_block(&mut requests_rx), Some(block(0)));
    }

    /// The anchor for a mid-chain activation is the last block before it, not genesis and not the
    /// local-safe head.
    #[tokio::test]
    async fn the_anchor_is_the_last_block_before_activation() {
        let root = tempfile::tempdir().expect("temp dir");
        // Activation at the timestamp of block 3; the anchor timestamp is activation - 1, which
        // floors onto block 2.
        let (mut actor, state_tx, mut requests_rx) =
            actor_with_one_chain(root.path(), GENESIS_TIME + 6);

        state_tx.send(engine_state(block(5))).expect("state");
        actor.promote().await.expect("promote");
        assert_eq!(promoted_block(&mut requests_rx), Some(block(2)));
    }
}
