use crate::{
    BuildRequest, ChainControllerClientError, ChainControllerDerivationClient,
    ChainControllerError, CommitRequest, NodeActor, ResetRequest, RewindRequest, SealRequest,
};
use alloy_eips::{BlockNumHash, BlockNumberOrTag};
use async_trait::async_trait;
use kona_derive::{ResetSignal, Signal};
use kona_engine::{
    BuildSealCoupling, BuildTask, CommitTask, ConsolidateInput, ConsolidateTask,
    CrossSafePromotion, Engine, EngineClient, EngineTask, EngineTaskError, EngineTaskErrorSeverity,
    FinalizeBlockId, FinalizeTask, ImportedBlockSink, InsertTask, L2ForkchoiceState, LocalSafeHead,
    PromoteCrossSafeTask, SealTask, SharedDenyList,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use kona_safedb::SharedSafeDb;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::sync::{mpsc, watch};

/// A request handled by the [`ChainController`].
#[derive(Debug)]
pub enum ChainControllerRequest {
    /// Request to start building a block.
    Build(Box<BuildRequest>),
    /// Request to process a local-safe signal, which can be derived attributes or delegated block
    /// info.
    ProcessLocalSafeL2Signal(ConsolidateInput),
    /// Request to process the finalized L2 block identified by the provided [`FinalizeBlockId`].
    ProcessFinalizedL2Block(Box<FinalizeBlockId>),
    /// Request to process a received unsafe L2 block.
    ProcessUnsafeL2Block(Box<OpExecutionPayloadEnvelope>),
    /// Request to commit an externally built payload as the unsafe head, answering the caller:
    /// `opstack_commitBlockV1`'s write. The gossip import above, with a result channel.
    CommitBlock(Box<CommitRequest>),
    /// Request to promote the cross-safe head to an externally verified block.
    ///
    /// The [`CrossSafePromotion`] cannot be forged: only the holder of this engine's unique
    /// [`CrossSafePromoter`] mints one, so the mere existence of this request is the proof that
    /// the cross-chain verifier — and not some other actor — decided the block is cross-safe.
    ///
    /// [`CrossSafePromoter`]: kona_engine::CrossSafePromoter
    PromoteCrossSafe(Box<CrossSafePromotion>),
    /// Request to reset the forkchoice.
    Reset(Box<ResetRequest>),
    /// Request to rewind the chain onto the parent of an invalidated block.
    ///
    /// The interop verifier sends this while applying an invalidation, after the block is durably
    /// on the deny list: the rewind takes the chain off the invalid block, and derivation's
    /// rebuild of the height then hits the deny list and becomes the deposits-only replacement.
    Rewind(Box<RewindRequest>),
    /// Request to seal a block.
    Seal(Box<SealRequest>),
}

/// Owns this chain's head state: the sole writer of the [`Engine`]'s heads and the only actor that
/// drives the execution layer's Engine API.
///
/// Every state-mutating input — derived attributes from derivation, unsafe payloads from gossip,
/// finalization, resets, and sequencer block building — arrives on one inbound request channel as a
/// [`ChainControllerRequest`] and is ordered through the [`Engine`] task queue by the [`Ord`]
/// implementation of [`EngineTask`]. In the other direction the controller mediates every
/// derivation-bound signal (reset, channel flush, sync-completed, and the local-safe lockstep
/// confirmation) and owns reset initiation, so derivation never reaches the execution layer itself.
///
/// Read-only queries are served by its peer [`ChainControllerRpcActor`], which shares a watch over
/// the engine state and queue length but holds a read-only client.
///
/// [`ChainControllerRpcActor`]: crate::ChainControllerRpcActor
#[derive(Debug)]
pub struct ChainController<EngineClient_, DerivationClient>
where
    EngineClient_: EngineClient,
    DerivationClient: ChainControllerDerivationClient,
{
    /// The client used to send messages to the [`crate::DerivationActor`].
    derivation_client: DerivationClient,
    /// Whether the EL sync is complete. This should only ever go from false to true.
    el_sync_complete: bool,
    /// The last local-safe head update sent to the derivation actor.
    last_local_safe_head_sent: L2BlockInfo,
    /// The safe-head database this controller records local-safe advances into.
    safe_db: SharedSafeDb,
    /// Where to hand every imported block, so the derivation providers can read it locally
    /// instead of fetching it back from the execution layer.
    block_sink: Arc<dyn ImportedBlockSink>,
    /// The last pairing written to [`Self::safe_db`], so an unchanged head is not rewritten.
    ///
    /// Cleared by a reset: the rewind that follows re-opens L1 block numbers this controller has
    /// already recorded once, and the ascending-order contract is about what is *written*, not
    /// about what was once written and then removed.
    last_recorded: Option<LocalSafeHead>,
    /// A channel to use to relay the current unsafe head.
    /// ## Note
    /// This is `Some` when the node is in sequencer mode, and `None` when the node is in validator
    /// mode.
    unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,

    /// Whether the startup engine reset has been attempted, so the first step performs it exactly
    /// once. See [`Self::startup_reset`].
    startup_reset_attempted: bool,

    /// The [`RollupConfig`] used to build tasks.
    rollup: Arc<RollupConfig>,
    /// An [`EngineClient`] used for creating engine tasks.
    client: Arc<EngineClient_>,
    /// The [`Engine`] task queue.
    engine: Engine<EngineClient_>,
    /// The inbound request channel.
    inbound_request_rx: mpsc::Receiver<ChainControllerRequest>,
    /// The super-authority deny list, when the node runs under one.
    ///
    /// Threaded into the consolidate and seal tasks — where a denied block is refused adoption and
    /// replaced deposits-only — and consulted here to gate unsafe ingestion while an invalidation
    /// is being recovered from.
    deny: Option<SharedDenyList>,
    /// Whether unsafe ingestion is currently gated by the deny list, kept so the gate logs its
    /// edges rather than every dropped payload.
    unsafe_deny_gated: bool,
}

impl<EngineClient_, DerivationClient> ChainController<EngineClient_, DerivationClient>
where
    EngineClient_: EngineClient + 'static,
    DerivationClient: ChainControllerDerivationClient + 'static,
{
    /// Constructs a new [`ChainController`] from the params.
    #[allow(clippy::too_many_arguments)] // the constructor takes one argument per field
    pub fn new(
        client: Arc<EngineClient_>,
        config: Arc<RollupConfig>,
        derivation_client: DerivationClient,
        engine: Engine<EngineClient_>,
        unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,
        inbound_request_rx: mpsc::Receiver<ChainControllerRequest>,
        safe_db: SharedSafeDb,
        deny: Option<SharedDenyList>,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> Self {
        Self {
            client,
            derivation_client,
            el_sync_complete: false,
            engine,
            last_local_safe_head_sent: L2BlockInfo::default(),
            rollup: config,
            unsafe_head_tx,
            inbound_request_rx,
            safe_db,
            block_sink,
            last_recorded: None,
            deny,
            unsafe_deny_gated: false,
            startup_reset_attempted: false,
        }
    }

    /// The startup engine reset: discover the execution layer's actual heads by walkback and put
    /// the initial forkchoice update on the wire, before anything else drives the engine.
    ///
    /// op-node does this in every sync mode — its derivation pipeline runs `initialReset` at
    /// startup and the engine controller answers with a `FindL2Heads` walkback and a force reset
    /// (`op-node/rollup/derive/pipeline.go:181-190, 224-…`,
    /// `op-node/rollup/engine/engine_controller.go:1423-1432`). The forkchoice update it sends
    /// doubles as the EL-sync probe: a `VALID` answer marks EL sync finished, so derivation starts
    /// against an execution layer that already has a consistent chain (op-node's CL-sync steady
    /// state) instead of waiting for a gossiped payload to say so — a wait that never ends against
    /// an execution layer that cannot advance on its own, which is exactly the sync-tester
    /// verifier restarted against a reset session while the sequencer is far ahead. A `SYNCING`
    /// answer marks nothing, and the gossip-driven EL-sync bootstrap stays in charge exactly as
    /// before (op-node's EL-sync regime).
    ///
    /// The reset only runs against an execution layer that reports a finalized block. An EL
    /// without one has never been driven past its genesis state — its initial EL sync is still
    /// pending, and probing it would answer `VALID` for its own genesis, wrongly marking EL sync
    /// finished and cutting off the gossip bootstrap that is supposed to point it at the tip.
    /// The discriminator is op-node's own: `initializeUnknowns` reads a missing finalized block
    /// as "no finalized block yet — EL sync has not completed"
    /// (`op-node/rollup/engine/engine_controller.go:597-624`), and EL-sync mode is only entered
    /// "if there is no finalized block" (`engine_controller.go:29`).
    ///
    /// Best-effort: an execution layer this reset cannot complete against — a walkback the EL
    /// cannot serve mid-sync, a forkchoice update it refuses — is not a startup failure. The
    /// engine falls back to the gossip bootstrap, and whatever condition blocked the reset
    /// surfaces again on the path that hits it next. (RPC-level unreachability is retried, both
    /// in the finalized-block read here and inside [`Engine::reset`] itself, so a merely-late EL
    /// delays this reset rather than skipping it.)
    async fn startup_reset(&mut self) -> Result<(), ChainControllerError> {
        if !self.el_reports_finalized_block().await {
            info!(
                target: "engine",
                "The execution layer reports no finalized block yet; skipping the startup engine \
                 reset and awaiting EL sync via unsafe gossip"
            );
            return Ok(());
        }

        info!(target: "engine", "Performing the startup engine reset");
        match self.reset().await {
            Err(ChainControllerError::EngineReset(err)) => {
                warn!(
                    target: "engine",
                    ?err,
                    "The startup engine reset did not complete; falling back to the unsafe-gossip \
                     EL-sync bootstrap"
                );
                Ok(())
            }
            result => result,
        }
    }

    /// Whether the execution layer reports a finalized block, retrying — with an exponential
    /// backoff — while it cannot answer at all.
    ///
    /// A missing finalized block is a definitive answer (`false`), whether it arrives as an empty
    /// result or as the non-standard not-found error some execution layers return for the
    /// finalized label before anything is finalized — the same shapes op-node folds together
    /// (`op-node/rollup/engine/engine_controller.go:611-624`,
    /// `op-service/eth/errors.go:10-27`). A transport failure is no answer, and startup must not
    /// guess: a merely-late execution layer is waited for, the way op-node's startup loops its
    /// engine reads.
    async fn el_reports_finalized_block(&self) -> bool {
        /// The delay before the first retry, doubled per attempt up to
        /// [`STARTUP_PROBE_RETRY_MAX_DELAY`].
        const STARTUP_PROBE_RETRY_BASE_DELAY: std::time::Duration =
            std::time::Duration::from_millis(250);
        /// The ceiling for the retry backoff.
        const STARTUP_PROBE_RETRY_MAX_DELAY: std::time::Duration =
            std::time::Duration::from_secs(10);

        let mut attempts: u32 = 0;
        loop {
            match self.client.l2_block_by_label(BlockNumberOrTag::Finalized).await {
                Ok(Some(_)) => return true,
                Ok(None) => return false,
                Err(err) => {
                    let msg = err.to_string().to_lowercase();
                    if msg.contains("block not found") ||
                        msg.contains("header not found") ||
                        msg.contains("unknown block")
                    {
                        return false;
                    }
                    let delay = STARTUP_PROBE_RETRY_BASE_DELAY
                        .saturating_mul(1 << attempts.min(7))
                        .min(STARTUP_PROBE_RETRY_MAX_DELAY);
                    attempts += 1;
                    warn!(
                        target: "engine",
                        ?err,
                        ?delay,
                        attempts,
                        "The execution layer could not answer the finalized-block read; retrying"
                    );
                    tokio::time::sleep(delay).await;
                }
            }
        }
    }

    /// Resets the inner [`Engine`] and propagates the reset to the derivation actor.
    async fn reset(&mut self) -> Result<(), ChainControllerError> {
        // Reset the engine. Resets re-derive local safety only; the cross-safe head is not
        // touched.
        let l2_safe_head = self.engine.reset(self.client.clone(), self.rollup.clone()).await?;

        // Before derivation is told anything: the records above the reset point describe blocks
        // this node has just disowned, and a reader that saw them between the reset and the
        // rewind would pair a live timestamp with an L1 block on the abandoned branch.
        self.rewind_safe_db(l2_safe_head);
        self.seed_genesis_safe_head(l2_safe_head).await;

        // Signal the derivation actor to reset.
        let signal = Signal::Reset(ResetSignal { l2_safe_head });
        match self.derivation_client.send_signal(signal).await {
            Ok(_) => info!(target: "engine", "Sent reset signal to derivation actor"),
            Err(err) => {
                error!(target: "engine", ?err, "Failed to send reset signal to the derivation actor");
                return Err(ChainControllerError::ChannelClosed);
            }
        }

        self.send_derivation_actor_local_safe_head_if_updated().await?;

        Ok(())
    }

    /// Rewinds the engine onto `parent` and propagates the reset to the derivation actor.
    ///
    /// The targeted counterpart of [`Self::reset`], for the one caller that already knows where
    /// the chain must land: the interop verifier applying an invalidation, whose target is the
    /// parent of the invalidated block. The walkback would re-*discover* a landing point and could
    /// pick a different block than the one the invalidation means, so [`Engine::reset_to`] applies
    /// the caller's target verbatim. Everything after the engine move is the same as a reset: the
    /// safe-head records above the target describe blocks this node has just disowned, and
    /// derivation restarts from the target to rebuild the invalidated height — where the deny
    /// list turns the rebuild into a deposits-only replacement.
    async fn rewind_to(&mut self, parent: L2BlockInfo) -> Result<(), ChainControllerError> {
        let target = L2ForkchoiceState {
            un_safe: parent,
            local_safe: parent,
            finalized: self.engine.state().sync_state.finalized_head(),
        };
        let l2_safe_head =
            self.engine.reset_to(self.client.clone(), self.rollup.clone(), target).await?;

        // Before derivation is told anything, for the same reason `reset` does it there.
        self.rewind_safe_db(l2_safe_head);
        self.seed_genesis_safe_head(l2_safe_head).await;

        let signal = Signal::Reset(ResetSignal { l2_safe_head });
        match self.derivation_client.send_signal(signal).await {
            Ok(_) => info!(target: "engine", "Sent reset signal to derivation actor after rewind"),
            Err(err) => {
                error!(target: "engine", ?err, "Failed to send reset signal to the derivation actor");
                return Err(ChainControllerError::ChannelClosed);
            }
        }

        self.send_derivation_actor_local_safe_head_if_updated().await?;

        Ok(())
    }

    /// Whether unsafe-payload ingestion is blocked by the deny list: from the moment a block is
    /// denied until the finalized head passes the highest denied height, so unsafe sync cannot
    /// re-adopt the invalidated branch.
    ///
    /// The mirror of op-node's `unsafeDenyGatingActive`
    /// (`op-node/rollup/engine/engine_controller.go:776-799`), with the same posture: a deny-list
    /// read error fails open — a wedged unsafe pipeline is worse than looping invalidation until
    /// the store heals — and it is always logged.
    fn unsafe_deny_gating_active(&mut self) -> bool {
        let Some(deny) = &self.deny else { return false };
        let max_denied = match deny.max_denied_height() {
            Ok(max_denied) => max_denied,
            Err(err) => {
                error!(
                    target: "engine",
                    %err,
                    "Failed to read max denied height, allowing unsafe ingestion"
                );
                return false;
            }
        };
        let finalized = self.engine.state().sync_state.finalized_head().block_info.number;
        let active = max_denied.is_some_and(|max_denied| max_denied > finalized);
        if active != self.unsafe_deny_gated {
            self.unsafe_deny_gated = active;
            if active {
                warn!(
                    target: "engine",
                    max_denied_height = max_denied,
                    finalized,
                    "Gating unsafe ingestion during invalidation recovery"
                );
            } else {
                warn!(
                    target: "engine",
                    max_denied_height = max_denied,
                    finalized,
                    "Resuming unsafe ingestion, finality passed the invalidation"
                );
            }
        }
        active
    }

    /// Drains the inner [`Engine`] task queue and attempts to update the safe head.
    async fn drain(&mut self) -> Result<(), ChainControllerError> {
        match self.engine.drain().await {
            Ok(_) => {
                trace!(target: "engine", "[ENGINE] tasks drained");
            }
            Err(err) => {
                match err.severity() {
                    EngineTaskErrorSeverity::Critical => {
                        error!(target: "engine", ?err, "Critical error draining engine tasks");
                        return Err(err.into());
                    }
                    EngineTaskErrorSeverity::Reset => {
                        warn!(target: "engine", ?err, "Received reset request");
                        self.reset().await?;
                    }
                    EngineTaskErrorSeverity::Flush => {
                        // This error is encountered when the payload is marked INVALID
                        // by the engine api. Post-holocene, the payload is replaced by
                        // a "deposits-only" block and re-executed. At the same time,
                        // the channel and any remaining buffered batches are flushed.
                        warn!(target: "engine", ?err, "Invalid payload, Flushing derivation pipeline.");
                        match self.derivation_client.send_signal(Signal::FlushChannel).await {
                            Ok(_) => {
                                debug!(target: "engine", "Sent flush signal to derivation actor")
                            }
                            Err(err) => {
                                error!(target: "engine", ?err, "Failed to send flush signal to the derivation actor.");
                                return Err(ChainControllerError::ChannelClosed);
                            }
                        }
                    }
                    EngineTaskErrorSeverity::Temporary => {
                        trace!(target: "engine", ?err, "Temporary error draining engine tasks");
                    }
                }
            }
        }

        self.record_local_safe_head(self.engine.state().sync_state.local_safe());
        self.send_derivation_actor_local_safe_head_if_updated().await?;

        if !self.el_sync_complete && self.engine.state().el_sync_finished {
            self.mark_el_sync_complete_and_notify_derivation_actor().await?;
        }

        Ok(())
    }

    async fn mark_el_sync_complete_and_notify_derivation_actor(
        &mut self,
    ) -> Result<(), ChainControllerError> {
        self.el_sync_complete = true;

        // Reset the engine if the sync state does not already know about a finalized block.
        if self.engine.state().sync_state.finalized_head() == L2BlockInfo::default() {
            // If the sync status is finished, we can reset the engine and start derivation.
            info!(target: "engine", "Performing initial engine reset");
            self.reset().await?;
        } else {
            info!(target: "engine", "finalized head is not default, so not resetting");
        }

        self.derivation_client
            .notify_sync_completed(self.engine.state().sync_state.local_safe_head())
            .await
            .map(|_| Ok(()))
            .map_err(|e| {
                error!(target: "engine", ?e, "Failed to notify sync completed");
                ChainControllerError::ChannelClosed
            })?
    }

    /// Records `local_safe` in the safe-head database, if it advanced and carries an L1 origin.
    ///
    /// This is the only writer of that database, which is what keeps the ascending-L1 contract
    /// [`SafeDb::safe_head_updated`] states satisfiable at all: the pairing is read out of the
    /// engine state this controller exclusively owns, so no other actor can interleave a record.
    /// It is taken as an argument rather than read here so that what gets recorded is a function
    /// of one observed pairing, and the caller decides which observation that is.
    ///
    /// An unpaired head is skipped rather than recorded against a defaulted L1 block. A reset
    /// walkback and the derivation-delegation path both write heads whose L1 origin they never
    /// knew, and recording those as "derived from block 0" would answer a later history query
    /// with a pairing that was never true.
    ///
    /// The granularity is one record per drain rather than one per L2 block, because the drained
    /// engine state is what this controller can observe. A reader asking which L1 block made an
    /// L2 block safe therefore gets an L1 block that is canonical and at or after the true one,
    /// never before it — the same conservative direction op-supernode's `safeDBAtL2` answers in,
    /// and safe for a consumer that only ever treats it as "safe by this L1 block".
    ///
    /// A failed write is logged and not propagated: derivation is not wrong because a record of
    /// it could not be stored, and the gap costs a later-than-necessary L1 answer rather than an
    /// incorrect one.
    ///
    /// [`SafeDb::safe_head_updated`]: kona_safedb::SafeDb::safe_head_updated
    pub(super) fn record_local_safe_head(&mut self, local_safe: LocalSafeHead) {
        if !self.safe_db.enabled() {
            return;
        }

        let Some(l1) = local_safe.derived_from_l1() else { return };
        if self.last_recorded == Some(local_safe) {
            return;
        }

        // A pairing below the last recorded L1 block is not an advance: it descends from a reset
        // whose rewind has already removed the records above it and re-recorded the boundary
        // itself, so writing it back would undo that. The *same* L1 block is an advance — a later
        // L2 block derived from it is the safe head as of that block, and supersedes the earlier
        // record under the same key.
        if let Some(last) = self.last_recorded &&
            let Some(last_l1) = last.derived_from_l1() &&
            l1.number < last_l1.number
        {
            return;
        }

        // Synchronous and fsynced, on the controller's own task. One small write per drain
        // against an L2 block time is not worth moving off the actor, and doing it here is what
        // makes the record ordered with respect to the head that produced it.
        match self.safe_db.safe_head_updated(local_safe.head, l1.id()) {
            Ok(()) => self.last_recorded = Some(local_safe),
            Err(err) => error!(
                target: "engine",
                ?err,
                l2_number = local_safe.head.block_info.number,
                l1_number = l1.number,
                "Failed to record the local-safe head; history queries below it will answer with \
                 a later L1 block"
            ),
        }
    }

    /// Rewinds the safe-head database so `safe_head` is its tip again.
    ///
    /// Paired with [`Self::record_local_safe_head`] across a reset: the engine has just disowned
    /// everything above `safe_head`, and the records for those blocks have to go with them.
    pub(super) fn rewind_safe_db(&mut self, safe_head: L2BlockInfo) {
        if !self.safe_db.enabled() {
            return;
        }

        if let Err(err) = self.safe_db.safe_head_reset(safe_head) {
            error!(
                target: "engine",
                ?err,
                l2_number = safe_head.block_info.number,
                "Failed to rewind the safe-head database; it may still hold records of blocks \
                 this node has disowned"
            );
        }
        // Cleared either way: the records above the reset point are gone, or the rewind failed
        // and this controller's idea of what is stored is no longer trustworthy. Both mean the
        // next advance should be written rather than suppressed as unchanged.
        self.last_recorded = None;
    }

    /// Records L2 genesis as safe from L1 block 0, when a reset lands the local-safe head there.
    ///
    /// This is op-node's behaviour on an engine-reset confirmation
    /// (`op-node/rollup/driver/sync_deriver.go:204-220`): the rollup genesis block is safe by
    /// definition, and a pipeline that resets this far back will observe every safe-head update
    /// from here on, so genesis can be recorded as safe from L1 genesis. op-node deliberately
    /// keys the record at L1 block 0 rather than `cfg.Genesis.L1` — the dispute contracts may
    /// predate the L2 genesis's own L1 origin, so games can anchor at an earlier L1 head.
    ///
    /// The entry is what holds the safe-head database's floor at genesis. Without it, the
    /// earliest recorded entry is the first post-reset advance — which, under one L1 key per
    /// recorded transition, can name an L2 block well above 1 — and `l1_at_safe_head` for any
    /// height below that floor answers `L1AtSafeHeadUnavailable`, permanently. op-supernode's
    /// superroot API fails the whole `superroot_atTimestamp` call on that error, so a supernode
    /// missing this record cannot answer for any timestamp its first recorded entry does not
    /// cover, even though it derived those blocks itself.
    ///
    /// A failed L1 read or write is logged rather than propagated, the posture
    /// [`Self::rewind_safe_db`] already takes for this database: the reset is not wrong because
    /// a record of it could not be stored. (op-node instead withholds the pipeline-reset
    /// confirmation so the reset re-runs; this controller has no such re-trigger, and the cost
    /// here is history queries below the first recorded advance failing as they did before the
    /// seed existed.)
    pub(super) async fn seed_genesis_safe_head(&self, l2_safe_head: L2BlockInfo) {
        if !self.safe_db.enabled() || l2_safe_head.block_info.id() != self.rollup.genesis.l2 {
            return;
        }
        let l1_genesis = match self.client.get_l1_block(BlockNumberOrTag::Number(0).into()).await {
            Ok(Some(block)) => {
                BlockNumHash { number: block.header.number, hash: block.header.hash }
            }
            Ok(None) => {
                error!(
                    target: "engine",
                    "The L1 has no genesis block; cannot record L2 genesis as safe"
                );
                return;
            }
            Err(err) => {
                error!(
                    target: "engine",
                    ?err,
                    "Failed to read the L1 genesis block; cannot record L2 genesis as safe"
                );
                return;
            }
        };
        if let Err(err) = self.safe_db.safe_head_updated(l2_safe_head, l1_genesis) {
            error!(
                target: "engine",
                ?err,
                "Failed to record L2 genesis as safe; history queries below the first recorded \
                 advance will answer as unavailable"
            );
        }
    }

    /// Attempts to send the [`crate::DerivationActor`] the local-safe head if updated.
    ///
    /// This is the depth-1 lockstep confirmation that unblocks derivation's next set of payload
    /// attributes, so it must be driven by local-safe. Driving it from cross-safe deadlocks under
    /// interop: derivation would wait on a promotion that waits on every chain's derivation.
    async fn send_derivation_actor_local_safe_head_if_updated(
        &mut self,
    ) -> Result<(), ChainControllerError> {
        let engine_local_safe_head = self.engine.state().sync_state.local_safe_head();
        if engine_local_safe_head == self.last_local_safe_head_sent {
            info!(target: "engine", local_safe_head = ?engine_local_safe_head, "Local-safe head unchanged");
            // This was already sent, so do not send it.
            return Ok(());
        }

        self.derivation_client
            .send_new_engine_local_safe_head(engine_local_safe_head)
            .await
            .map_err(|e| {
                error!(target: "engine", ?e, "Failed to send new engine local-safe head");
                ChainControllerError::ChannelClosed
            })?;

        info!(target: "engine", local_safe_head = ?engine_local_safe_head, "Attempted L2 local-safe head update");
        self.last_local_safe_head_sent = engine_local_safe_head;

        Ok(())
    }
}

#[async_trait]
impl<EngineClient_, DerivationClient> NodeActor for ChainController<EngineClient_, DerivationClient>
where
    EngineClient_: EngineClient + 'static,
    DerivationClient: ChainControllerDerivationClient + 'static,
{
    type Error = ChainControllerError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        // The first step performs the startup engine reset, before any request is taken: the
        // walkback discovers the execution layer's heads, and its forkchoice update doubles as
        // the EL-sync probe. See `Self::startup_reset`.
        if !self.startup_reset_attempted {
            self.startup_reset_attempted = true;
            self.startup_reset().await?;
        }

        // Attempt to drain all outstanding tasks from the engine queue before adding new ones.
        self.drain()
            .await
            .inspect_err(|err| error!(target: "engine", ?err, "Failed to drain engine tasks"))?;

        // If the unsafe head has updated, propagate it to the outbound channels.
        if let Some(unsafe_head_tx) = self.unsafe_head_tx.as_ref() {
            unsafe_head_tx.send_if_modified(|val| {
                let new_head = self.engine.state().sync_state.unsafe_head();
                (*val != new_head).then(|| *val = new_head).is_some()
            });
        }

        // Wait for the next processing request.
        let request = self.inbound_request_rx.recv().await.ok_or_else(|| {
            error!(target: "engine", "Engine processing request receiver closed unexpectedly");
            ChainControllerError::ChannelClosed
        })?;

        match request {
            ChainControllerRequest::Build(build_request) => {
                let BuildRequest { attributes, result_tx } = *build_request;
                let task = EngineTask::Build(Box::new(BuildTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    attributes,
                    Some(result_tx),
                )));
                self.engine.enqueue(task);
            }
            ChainControllerRequest::ProcessLocalSafeL2Signal(local_safe_signal) => {
                let task = EngineTask::Consolidate(Box::new(ConsolidateTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    local_safe_signal,
                    self.deny.clone(),
                    Arc::clone(&self.block_sink),
                )));
                self.engine.enqueue(task);
            }
            ChainControllerRequest::ProcessFinalizedL2Block(finalized_l2_block_id) => {
                // Finalize the L2 block identified by the provided [`FinalizeBlockId`].
                let task = EngineTask::Finalize(Box::new(FinalizeTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    *finalized_l2_block_id,
                )));
                self.engine.enqueue(task);
            }
            ChainControllerRequest::ProcessUnsafeL2Block(envelope) => {
                // Kept out of the queue rather than failed inside it: a queued task that can
                // never succeed would be retried forever, and op-node's gate likewise drops the
                // payload at ingestion (`AddUnsafePayload`). The gate covers the whole
                // invalidation-recovery window, not just the denied hash, so a descendant of the
                // denied block cannot re-adopt the branch either.
                if self.unsafe_deny_gating_active() {
                    warn!(
                        target: "engine",
                        block_number = envelope.block_number(),
                        block_hash = %envelope.block_hash(),
                        "Dropping unsafe payload: ingestion is gated during invalidation recovery"
                    );
                    return Ok(());
                }
                let task = EngineTask::Insert(Box::new(InsertTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    // The payload is not derived in this case. This is an unsafe block, so it
                    // moves no local-safe head and has no L1 origin to pair with one.
                    *envelope,
                    None,
                    Arc::clone(&self.block_sink),
                )));
                self.engine.enqueue(task);
            }
            ChainControllerRequest::CommitBlock(commit_request) => {
                let CommitRequest { envelope, result_tx } = *commit_request;
                let task = EngineTask::Commit(Box::new(CommitTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    envelope,
                    result_tx,
                    self.deny.clone(),
                    Arc::clone(&self.block_sink),
                )));
                self.engine.enqueue(task);
            }
            ChainControllerRequest::PromoteCrossSafe(promotion) => {
                let task = EngineTask::PromoteCrossSafe(Box::new(PromoteCrossSafeTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    *promotion,
                )));
                self.engine.enqueue(task);
            }
            ChainControllerRequest::Reset(reset_request) => {
                warn!(target: "engine", "Received reset request");

                let reset_res = self.reset().await;

                // Send the result.
                let response_payload = reset_res
                    .as_ref()
                    .map(|_| ())
                    .map_err(|e| ChainControllerClientError::ResetForkchoiceError(e.to_string()));
                if reset_request.result_tx.send(response_payload).await.is_err() {
                    warn!(target: "engine", "Sending reset response failed");
                    // If there was an error and we couldn't notify the caller to handle it,
                    // return the error.
                    reset_res?;
                }
            }
            ChainControllerRequest::Rewind(rewind_request) => {
                let RewindRequest { parent, result_tx } = *rewind_request;
                warn!(
                    target: "engine",
                    parent_number = parent.block_info.number,
                    parent_hash = %parent.block_info.hash,
                    "Received rewind request: taking the chain off an invalidated block"
                );

                let rewind_res = self.rewind_to(parent).await;

                let response_payload = rewind_res
                    .as_ref()
                    .map(|_| ())
                    .map_err(|e| ChainControllerClientError::ResetForkchoiceError(e.to_string()));
                if result_tx.send(response_payload).await.is_err() {
                    warn!(target: "engine", "Sending rewind response failed");
                    // If there was an error and we couldn't notify the caller to handle it,
                    // return the error.
                    rewind_res?;
                }
            }
            ChainControllerRequest::Seal(seal_request) => {
                let SealRequest { payload_id, attributes, result_tx } = *seal_request;
                let task = EngineTask::Seal(Box::new(SealTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    payload_id,
                    attributes,
                    // The payload is not derived in this case.
                    false,
                    // The sequencer seals in a separate request from the build that started the
                    // job, so the unsafe-head staleness check applies.
                    BuildSealCoupling::Detached,
                    Some(result_tx),
                    self.deny.clone(),
                    Arc::clone(&self.block_sink),
                )));
                self.engine.enqueue(task);
            }
        }

        Ok(())
    }
}
