use crate::{
    DerivationActorRequest, DerivationEngineClient, NodeActor,
    actors::derivation::{DerivationDelegateProvider, DerivationError},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::BlockHash;
use async_trait::async_trait;
use kona_derive::{ChainProvider, Signal};
use kona_engine::{
    DelegatedRejection, DelegatedSafeHeads, EngineState, FinalityViolation, FinalizeBlockId,
    InconsistentSafeHeadsError,
};
use kona_protocol::{BlockInfo, L2BlockInfo, SyncStatus};
use thiserror::Error;
use tokio::{
    select,
    sync::{mpsc, watch},
    time::{self, Instant},
};

/// Hardcoded poll interval for Derivation Delegation
pub(super) const DERIVATION_DELEGATE_POLL_INTERVAL: std::time::Duration =
    std::time::Duration::from_secs(4);

/// How long safe heads stay held back after the engine found the delegate's chain below its
/// finality, unless the engine's finality moves first. Each retry costs one rejected forkchoice
/// update and a restore if the delegate is still on that chain; without retries, a delegate that
/// returned to the canonical chain with its finalized head above the engine's frozen safe head
/// would stay held back until the next reset.
pub(super) const CHAIN_BELOW_FINALITY_RETRY_INTERVAL: std::time::Duration =
    std::time::Duration::from_secs(5 * 60);

/// The [`NodeActor`] for the delegate derivation sub-routine.
///
/// This actor is responsible for receiving messages from [`NodeActor`]s and polls
/// an external derivation delegation provider for derivation state. It validates
/// the canonicality of the L1 information associated with delegated derivation
/// results against the canonical L1 chain before forwarding updates.
///
/// Once validated, the actor sends the derived safe and finalized L2 info
/// to the [`NodeActor`] responsible for the execution sub-routine.
///
/// The delegate's local-safe and cross-safe heads are imported
/// separately: the local-safe head is the block the execution layer must have, or sync to, and
/// sets the engine's local-safe label; the cross-safe head sets the safe label that forkchoice
/// updates carry, and finalization never goes above it. Before interop both are the same block.
/// Under interop the delegate must verify cross-chain dependencies, for example op-node with a
/// supervisor, since this node checks none itself: its safe label is only as safe as the
/// delegate's cross-safe head.
#[derive(Debug)]
pub struct DelegateDerivationActor<DerivationEngineClient_, DelegateProvider, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient,
    DelegateProvider: DerivationDelegateProvider,
    L1Provider: ChainProvider,
{
    /// The channel on which all inbound requests are received by the [`DelegateDerivationActor`].
    inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// Derivation delegate provider.
    derivation_delegate_provider: DelegateProvider,
    /// L1 provider for validating L1 info for derivation delegation.
    l1_provider: L1Provider,
    /// Publishes the delegate's `current_l1`: the L1 block that its derivation has reached.
    derivation_origin_tx: watch::Sender<Option<BlockInfo>>,

    /// The engine's state as it publishes it. Delegate statuses are checked against the
    /// finalized head the engine applied, not the last one forwarded to it, which the engine may
    /// have skipped.
    engine_state_rx: watch::Receiver<EngineState>,
    /// The engine's L2 safe head, according to updates from the Engine.
    engine_l2_safe_head: L2BlockInfo,
    /// Whether the engine sync has completed. This will only ever go from false -> true.
    has_engine_sync_completed: bool,
    /// What the execution layer rejected and would reject again until the engine resets, so it is
    /// not sent again before then.
    rejected_by_engine: Option<RejectedByEngine>,
    /// Ticker driving periodic polls of the derivation delegate provider.
    delegated_derivation_ticker: time::Interval,

    /// Safe heads behind the engine's that were forwarded while the execution layer applied the
    /// engine's forkchoice, with the engine's local-safe and safe heads at the time. The engine
    /// checks such heads against its execution layer once; until either side changes, sending
    /// them again repeats that lookup to the same answer.
    forwarded_lagging_safe_heads: Option<(DelegatedSafeHeads, BlockNumHash, BlockNumHash)>,
    /// The last rejection of a delegate status, so a delegate stuck on one inconsistency is
    /// warned about once rather than on every poll.
    last_rejection: Option<String>,
}

impl<DerivationEngineClient_, DelegateProvider, L1Provider>
    DelegateDerivationActor<DerivationEngineClient_, DelegateProvider, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + 'static,
{
    /// Creates a new instance of the [`DelegateDerivationActor`].
    pub fn new(
        engine_client: DerivationEngineClient_,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        derivation_delegate_provider: DelegateProvider,
        l1_provider: L1Provider,
        derivation_origin_tx: watch::Sender<Option<BlockInfo>>,
        engine_state_rx: watch::Receiver<EngineState>,
    ) -> Self {
        let mut delegated_derivation_ticker = time::interval(DERIVATION_DELEGATE_POLL_INTERVAL);
        delegated_derivation_ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        Self {
            inbound_request_rx,
            engine_client,
            derivation_delegate_provider,
            l1_provider,
            derivation_origin_tx,
            engine_state_rx,
            engine_l2_safe_head: L2BlockInfo::default(),
            has_engine_sync_completed: false,
            rejected_by_engine: None,
            delegated_derivation_ticker,
            forwarded_lagging_safe_heads: None,
            last_rejection: None,
        }
    }
}

#[async_trait]
impl<DerivationEngineClient_, DelegateProvider, L1Provider> NodeActor
    for DelegateDerivationActor<DerivationEngineClient_, DelegateProvider, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + Send + 'static,
{
    type Error = DerivationError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        select! {
            biased;

            req = self.inbound_request_rx.recv() => {
                let request = req.ok_or_else(|| {
                    error!(
                        target: "derivation",
                        "DerivationActor inbound request receiver closed unexpectedly",
                    );
                    DerivationError::RequestReceiveFailed
                })?;
                self.handle_derivation_delegation_actor_request(request).await
            }
            _ = self.delegated_derivation_ticker.tick(),
            if self.has_engine_sync_completed => {
                self.fetch_and_apply_delegate_safe_head().await
            }
        }
    }
}

impl<DerivationEngineClient_, DelegateProvider, L1Provider>
    DelegateDerivationActor<DerivationEngineClient_, DelegateProvider, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    DelegateProvider: DerivationDelegateProvider + 'static,
    L1Provider: ChainProvider + 'static,
{
    /// Validates a single L1 block height and hash against the canonical L1 chain.
    async fn validate_l1_block(
        &mut self,
        context: &str,
        l1_block_number: u64,
        expected_hash: BlockHash,
    ) -> Result<(), DerivationDelegationError> {
        let block = self
            .l1_provider
            .block_info_by_number(l1_block_number)
            .await
            .map_err(|e| DerivationDelegationError::L1Provider(e.to_string()))?;

        if block.hash != expected_hash {
            return Err(DerivationDelegationError::L1ValidationFailed {
                context: context.to_string(),
                number: l1_block_number,
                expected: expected_hash,
                actual: block.hash,
            });
        }

        Ok(())
    }

    /// The delegate's local-safe head, or its safe head when it reports a zero local-safe head:
    /// `SyncStatus` zeroes a label the delegate has not derived, and a zero head carries nothing.
    fn reported_local_safe_head(status: &SyncStatus) -> L2BlockInfo {
        if status.local_safe_l2 == L2BlockInfo::default() {
            status.safe_l2
        } else {
            status.local_safe_l2
        }
    }

    /// Rejects a status whose labels are out of order: its finalized block ahead of its safe head,
    /// or its safe head ahead of its local safe head or a different block at its height. No
    /// consistent delegate reports any of these, and a finalized block ahead of the safe head would
    /// let a stale safe head sit below finality. Returns its safe heads otherwise.
    fn validate_label_order(
        status: &SyncStatus,
    ) -> Result<DelegatedSafeHeads, DerivationDelegationError> {
        let finalized = status.finalized_l2.block_info.number;
        let safe = status.safe_l2.block_info.number;
        if finalized > safe {
            return Err(DerivationDelegationError::FinalizedAheadOfSafe { finalized, safe });
        }
        Ok(DelegatedSafeHeads::try_new(Self::reported_local_safe_head(status), status.safe_l2)?)
    }

    /// Verifies that the labels reported by the derivation delegate are in order, that its
    /// finalized L2 block does not move back from the engine's, and that its L1 info is consistent
    /// with the canonical L1 chain. Returns its safe heads when it passes.
    async fn validate_sync_status(
        &mut self,
        v: &SyncStatus,
    ) -> Result<DelegatedSafeHeads, DerivationDelegationError> {
        let safe_heads = Self::validate_label_order(v)?;
        self.engine_state_rx
            .borrow()
            .sync_state
            .ensure_respects_finality(v.finalized_l2.block_info.id())?;

        let local_safe = safe_heads.local_safe_head();
        let checks = [
            ("L1 Origin of Local Safe L2", local_safe.l1_origin.number, local_safe.l1_origin.hash),
            ("L1 Origin of Safe L2", v.safe_l2.l1_origin.number, v.safe_l2.l1_origin.hash),
            (
                "L1 Origin of Finalized L2",
                v.finalized_l2.l1_origin.number,
                v.finalized_l2.l1_origin.hash,
            ),
            ("Current L1", v.current_l1.number, v.current_l1.hash),
        ];
        let mut validated_l1_blocks = Vec::with_capacity(checks.len());
        for (context, number, hash) in checks {
            // Labels often share an L1 origin, and one lookup proves it canonical for all of them.
            if validated_l1_blocks.contains(&(number, hash)) {
                continue;
            }
            self.validate_l1_block(context, number, hash).await?;
            validated_l1_blocks.push((number, hash));
        }
        Ok(safe_heads)
    }

    /// Logs a rejected delegate status at `warn!` when the reason changed, and at `debug!` while
    /// the delegate keeps reporting the same inconsistency.
    fn log_rejected_status(&mut self, err: &DerivationDelegationError) {
        let reason = err.to_string();
        if self.last_rejection.as_ref() == Some(&reason) {
            debug!(target: "derivation", error = %err, "Sync status from delegate rejected again");
        } else {
            warn!(target: "derivation", error = %err, "Rejected sync status from delegate");
            self.last_rejection = Some(reason);
        }
    }

    /// What to remember for safe heads behind the engine's consistent local-safe head while its
    /// execution layer applied the engine's forkchoice, or `None` for any other heads, which the
    /// engine must always see.
    fn lagging_safe_heads_to_remember(
        &self,
        safe_heads: DelegatedSafeHeads,
    ) -> Option<(DelegatedSafeHeads, BlockNumHash, BlockNumHash)> {
        let engine_state = self.engine_state_rx.borrow();
        let labels = &engine_state.sync_state;
        let engine_local_safe = labels.local_safe_head().block_info;
        let engine_safe = labels.safe_head().block_info;
        let are_labels_consistent = labels.unsafe_head().block_info.number >= engine_safe.number &&
            labels.unsafe_head().block_info.number >= engine_local_safe.number;
        (engine_state.el_applied_last_forkchoice &&
            are_labels_consistent &&
            safe_heads.local_safe_head().block_info.number < engine_local_safe.number)
            .then(|| (safe_heads, engine_local_safe.id(), engine_safe.id()))
    }

    /// Fetches, validates, and applies sync status from the derivation delegate.
    async fn fetch_and_apply_delegate_safe_head(&mut self) -> Result<(), DerivationError> {
        let sync_status = match self.derivation_delegate_provider.fetch_sync_status().await {
            Ok(status) => status,
            Err(_) => {
                warn!(target: "derivation", "Failed to fetch sync status from delegate");
                return Ok(());
            }
        };

        let safe_heads = match self.validate_sync_status(&sync_status).await {
            Ok(safe_heads) => safe_heads,
            Err(err) => {
                // Validation failures here are expected to be transient, so we skip processing
                // this sync status and continue delegating derivation instead of treating it as
                // fatal.
                self.log_rejected_status(&err);
                return Ok(());
            }
        };
        self.last_rejection = None;
        self.derivation_origin_tx.send_replace(Some(sync_status.current_l1));

        let lagging_safe_heads = self.lagging_safe_heads_to_remember(safe_heads);
        if self.is_rejected_by_engine(safe_heads) {
            debug!(target: "derivation", ?safe_heads, "Not sending safe heads the engine rejected before its next reset");
        } else if lagging_safe_heads.is_some() &&
            lagging_safe_heads == self.forwarded_lagging_safe_heads
        {
            debug!(
                target: "derivation",
                ?safe_heads,
                "Not forwarding lagging safe heads the engine already checked"
            );
        } else {
            self.forwarded_lagging_safe_heads = lagging_safe_heads;
            self.engine_client
                .send_safe_l2_signal(safe_heads.into())
                .await
                .map_err(|e| DerivationError::Sender(Box::new(e)))?;
        }

        // Delegated polling supplies `(number, hash)`. Carry the hash through so the engine
        // finalizes the specific block we were asked to, not whatever it happens to have at the
        // same height.
        self.engine_client
            .send_finalized_l2_block(FinalizeBlockId::ByHash(
                sync_status.finalized_l2.block_info.id(),
            ))
            .await
            .map_err(|e| DerivationError::Sender(Box::new(e)))?;

        debug!(
            target: "derivation",
            local_safe_l2 = ?sync_status.local_safe_l2,
            safe_l2 = ?sync_status.safe_l2,
            finalized_l2 = ?sync_status.finalized_l2,
            "Processed sync status from delegate"
        );

        Ok(())
    }

    /// What to hold back after the engine rejected `heads` for `rejection`.
    fn rejected_by_engine_for(
        &self,
        heads: DelegatedSafeHeads,
        rejection: DelegatedRejection,
    ) -> RejectedByEngine {
        match rejection {
            DelegatedRejection::ChainBelowFinality => RejectedByEngine::ChainBelowFinality {
                engine_finalized: self.engine_finalized_head(),
                retry_at: Instant::now() + CHAIN_BELOW_FINALITY_RETRY_INTERVAL,
            },
            DelegatedRejection::HeadLabels | DelegatedRejection::SafeLabel => {
                RejectedByEngine::SafeHeads(Box::new(heads))
            }
        }
    }

    /// The finalized head the engine applied, once it established finality.
    fn engine_finalized_head(&self) -> Option<BlockNumHash> {
        self.engine_state_rx.borrow().sync_state.established_finalized_head()
    }

    /// Whether `safe_heads` are held back because the engine rejected them, or the delegate's
    /// chain, since its last reset.
    ///
    /// A chain below finality is held back as a whole rather than by the blocks it reported: an
    /// advancing delegate reports new blocks on every poll, but none of its finalized targets is
    /// canonical, so the engine's finality cannot move while the delegate stays on that chain. The
    /// engine's finality moving therefore shows the delegate's finalized target is canonical, and
    /// lifts the hold; so does the retry interval passing.
    fn is_rejected_by_engine(&mut self, safe_heads: DelegatedSafeHeads) -> bool {
        match &self.rejected_by_engine {
            Some(RejectedByEngine::SafeHeads(rejected)) => **rejected == safe_heads,
            Some(RejectedByEngine::ChainBelowFinality { engine_finalized, retry_at }) => {
                let current_finalized = self.engine_finalized_head();
                if current_finalized != *engine_finalized {
                    info!(target: "derivation", ?current_finalized, "Engine finality moved since it rejected the delegate's chain; sending its safe heads again");
                } else if Instant::now() >= *retry_at {
                    info!(target: "derivation", ?safe_heads, "Retrying the safe heads of a delegate whose chain the engine found below its finality");
                } else {
                    return true;
                }
                self.rejected_by_engine = None;
                false
            }
            None => false,
        }
    }

    async fn handle_derivation_delegation_actor_request(
        &mut self,
        request_type: DerivationActorRequest,
    ) -> Result<(), DerivationError> {
        match request_type {
            DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(safe_head) => {
                debug!(target: "derivation", safe_head = ?*safe_head, "Received safe head from engine.");
                self.engine_l2_safe_head = *safe_head;
            }
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(sync_state) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                self.engine_l2_safe_head = sync_state.local_safe_head();
                self.has_engine_sync_completed = true;
            }
            DerivationActorRequest::ProcessRejectedDelegatedSafeHeads { heads, rejection } => {
                info!(target: "derivation", ?heads, %rejection, "Engine rejected delegated safe heads; not sending them again before its next reset");
                self.rejected_by_engine = Some(self.rejected_by_engine_for(*heads, rejection));
            }
            DerivationActorRequest::ProcessEngineSignalRequest(signal) => match *signal {
                // A reset rebuilds the engine's labels, after which rejected heads may apply.
                Signal::Reset(reset) => {
                    self.engine_l2_safe_head = reset.l2_safe_head;
                    self.rejected_by_engine = None;
                }
                signal => {
                    debug!(target: "derivation", ?signal, "Ignoring signal while derivation delegation");
                }
            },
            DerivationActorRequest::ProcessFinalizedL1Block(_) |
            DerivationActorRequest::ProcessL1HeadUpdateRequest(_) => {
                debug!(target: "derivation", "Ignoring request while derivation delegation: {:?}", request_type);
            }
        }
        Ok(())
    }
}

/// What the execution layer rejected and would reject again until the engine resets.
#[derive(Debug, Clone, PartialEq, Eq)]
enum RejectedByEngine {
    /// The delegate's chain, which does not contain the engine's finalized block.
    ChainBelowFinality {
        /// The engine's finalized head when it rejected the chain.
        engine_finalized: Option<BlockNumHash>,
        /// When the delegate's safe heads are sent once more.
        retry_at: Instant,
    },
    /// These exact safe heads.
    SafeHeads(Box<DelegatedSafeHeads>),
}

#[derive(Error, Debug)]
enum DerivationDelegationError {
    /// The delegate's finalized L2 block is ahead of its own safe head.
    #[error("Finalized L2 {finalized} is ahead of safe L2 {safe}")]
    FinalizedAheadOfSafe { finalized: u64, safe: u64 },

    /// The delegate's finalized L2 block is behind the engine's, or a different block at its
    /// height.
    #[error("Finalized L2 moves the engine's finality back: {0}")]
    FinalizedL2BreaksFinality(#[from] FinalityViolation),

    /// The delegate's safe heads are inconsistent with each other.
    #[error(transparent)]
    InconsistentSafeHeads(#[from] InconsistentSafeHeadsError),

    /// The L1 provider returned an error (network, RPC, etc.)
    #[error("L1 provider error: {0}")]
    L1Provider(String),

    /// The hash provided by the derivation delegation does not match the canonical chain.
    #[error("L1 inconsistency in {context} at block {number}: expected {expected}, got {actual}")]
    L1ValidationFailed { context: String, number: u64, expected: BlockHash, actual: BlockHash },
}
