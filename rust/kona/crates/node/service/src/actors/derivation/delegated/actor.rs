use crate::{
    DerivationActorRequest, DerivationEngineClient, NodeActor, actors::derivation::DerivationError,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, BlockHash};
use async_trait::async_trait;
use jsonrpsee::core::ClientError;
use kona_derive::{ChainProvider, Signal};
use kona_engine::{
    DelegatedSafeHeads, EngineState, FinalityViolation, FinalizeBlockId, InconsistentSafeHeadsError,
};
use kona_protocol::{BlockInfo, L2BlockInfo, SyncStatus};
use kona_rpc::RollupNodeApiClient;
use thiserror::Error;
use tokio::{
    select,
    sync::{mpsc, watch},
    time,
};

/// How often the derivation delegate is polled for its sync status.
pub(super) const DERIVATION_DELEGATE_POLL_INTERVAL: time::Duration = time::Duration::from_secs(4);

/// Supplies the sync status, and the L2 block hashes, of the external OP Stack CL node that
/// derivation is delegated to.
#[async_trait]
pub trait SyncStatusSource: Send + Sync {
    /// The error returned when the delegate cannot be queried.
    type Error: core::fmt::Display + Send;

    /// Fetches the delegate's current sync status.
    async fn sync_status(&self) -> Result<SyncStatus, Self::Error>;

    /// Fetches the hash of the delegate's L2 block at `number`.
    async fn block_hash_at(&self, number: u64) -> Result<B256, Self::Error>;
}

/// Any jsonrpsee client can serve the delegate's `optimism_syncStatus`, and its block hashes
/// through `optimism_outputAtBlock`.
#[async_trait]
impl<C: RollupNodeApiClient + Send + Sync> SyncStatusSource for C {
    type Error = ClientError;

    async fn sync_status(&self) -> Result<SyncStatus, ClientError> {
        self.op_sync_status().await
    }

    async fn block_hash_at(&self, number: u64) -> Result<B256, ClientError> {
        Ok(self.op_output_at_block(number.into()).await?.block_ref.block_info.hash)
    }
}

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
pub struct DelegateDerivationActor<DerivationEngineClient_, Delegate, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient,
    Delegate: SyncStatusSource,
    L1Provider: ChainProvider,
{
    /// The channel on which all inbound requests are received by the [`DelegateDerivationActor`].
    inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// Source of the sync status of the node that derivation is delegated to.
    delegate: Delegate,
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
    /// Ticker driving periodic polls of the derivation delegate.
    delegated_derivation_ticker: time::Interval,

    /// Safe heads behind the engine's that were forwarded while the execution layer applied the
    /// engine's forkchoice, with the engine's local-safe and safe heads at the time. The engine
    /// checks such heads against its execution layer once; until either side changes, sending
    /// them again repeats that lookup to the same answer.
    forwarded_lagging_safe_heads: Option<(DelegatedSafeHeads, BlockNumHash, BlockNumHash)>,
    /// The engine's finalized head and the hash of a delegate's finalized block above it whose
    /// chain was checked to contain that head. A block's ancestors never change, so the delegate
    /// is asked once for each pair.
    checked_finality: Option<(BlockNumHash, B256)>,
    /// Whether the delegate failed to serve its block at the engine's finalized height when last
    /// asked, so a delegate that cannot serve it is warned about once rather than on every poll.
    has_finality_lookup_failed: bool,
    /// The last rejection of a delegate status, so a delegate stuck on one inconsistency is
    /// warned about once rather than on every poll.
    last_rejection: Option<String>,
}

impl<DerivationEngineClient_, Delegate, L1Provider>
    DelegateDerivationActor<DerivationEngineClient_, Delegate, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    Delegate: SyncStatusSource + 'static,
    L1Provider: ChainProvider + 'static,
{
    /// Creates a new instance of the [`DelegateDerivationActor`].
    pub fn new(
        engine_client: DerivationEngineClient_,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        delegate: Delegate,
        l1_provider: L1Provider,
        derivation_origin_tx: watch::Sender<Option<BlockInfo>>,
        engine_state_rx: watch::Receiver<EngineState>,
    ) -> Self {
        let mut delegated_derivation_ticker = time::interval(DERIVATION_DELEGATE_POLL_INTERVAL);
        delegated_derivation_ticker.set_missed_tick_behavior(time::MissedTickBehavior::Skip);
        Self {
            inbound_request_rx,
            engine_client,
            delegate,
            l1_provider,
            derivation_origin_tx,
            engine_state_rx,
            engine_l2_safe_head: L2BlockInfo::default(),
            has_engine_sync_completed: false,
            delegated_derivation_ticker,
            forwarded_lagging_safe_heads: None,
            checked_finality: None,
            has_finality_lookup_failed: false,
            last_rejection: None,
        }
    }
}

#[async_trait]
impl<DerivationEngineClient_, Delegate, L1Provider> NodeActor
    for DelegateDerivationActor<DerivationEngineClient_, Delegate, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    Delegate: SyncStatusSource + 'static,
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

impl<DerivationEngineClient_, Delegate, L1Provider>
    DelegateDerivationActor<DerivationEngineClient_, Delegate, L1Provider>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    Delegate: SyncStatusSource + 'static,
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

    /// Rejects a delegate whose finalized block is above the engine's on a chain that does not
    /// contain the engine's finalized block, by asking the delegate for its block at that height.
    /// The execution layer makes a forkchoice update's head canonical before it rejects the
    /// update, so heads on such a chain must not reach it.
    ///
    /// A failed lookup does not hold the status back, since a delegate that cannot serve the
    /// block would otherwise stall the node: the execution layer then judges the heads, and the
    /// engine resets if it rejects them.
    async fn ensure_on_engine_finalized_chain(
        &mut self,
        delegate_finalized: BlockNumHash,
    ) -> Result<(), DerivationDelegationError> {
        let engine_finalized =
            self.engine_state_rx.borrow().sync_state.established_finalized_head();
        let Some(engine_finalized) = engine_finalized else { return Ok(()) };
        let checked = (engine_finalized, delegate_finalized.hash);
        if delegate_finalized.number <= engine_finalized.number ||
            self.checked_finality == Some(checked)
        {
            return Ok(());
        }

        let delegate_hash = match self.delegate.block_hash_at(engine_finalized.number).await {
            Ok(hash) => hash,
            Err(err) => {
                let number = engine_finalized.number;
                if self.has_finality_lookup_failed {
                    debug!(target: "derivation", error = %err, number, "Failed again to fetch the delegate's block at the engine's finalized height");
                } else {
                    warn!(target: "derivation", error = %err, number, "Failed to fetch the delegate's block at the engine's finalized height; forwarding its status unchecked");
                    self.has_finality_lookup_failed = true;
                }
                return Ok(());
            }
        };
        self.has_finality_lookup_failed = false;
        if delegate_hash != engine_finalized.hash {
            return Err(DerivationDelegationError::OffFinalizedChain {
                finalized: engine_finalized,
                delegate_hash,
            });
        }
        self.checked_finality = Some(checked);
        Ok(())
    }

    /// Verifies that the labels reported by the derivation delegate are in order, that its
    /// finalized L2 block neither moves back from the engine's nor sits on a chain without it, and
    /// that its L1 info is consistent with the canonical L1 chain. Returns its safe heads when it
    /// passes.
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
        // Last, so the delegate is not asked about a status that is rejected anyway.
        self.ensure_on_engine_finalized_chain(v.finalized_l2.block_info.id()).await?;
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
        let sync_status = match self.delegate.sync_status().await {
            Ok(status) => status,
            Err(err) => {
                warn!(
                    target: "derivation",
                    error = %err,
                    "Failed to fetch sync status from delegate"
                );
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
        if lagging_safe_heads.is_some() && lagging_safe_heads == self.forwarded_lagging_safe_heads {
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

    async fn handle_derivation_delegation_actor_request(
        &mut self,
        request_type: DerivationActorRequest,
    ) -> Result<(), DerivationError> {
        match request_type {
            DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(safe_head) => {
                debug!(target: "derivation", safe_head = ?*safe_head, "Received safe head from engine.");
                self.engine_l2_safe_head = *safe_head;
            }
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(safe_head) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                self.engine_l2_safe_head = *safe_head;
                self.has_engine_sync_completed = true;
            }
            DerivationActorRequest::ProcessEngineSignalRequest(signal) => match *signal {
                Signal::Reset(reset) => self.engine_l2_safe_head = reset.l2_safe_head,
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

    /// The delegate's chain has another block at the height of the engine's finalized block.
    #[error(
        "Chain does not contain the engine's finalized L2 {finalized:?}: it has {delegate_hash} at that height"
    )]
    OffFinalizedChain { finalized: BlockNumHash, delegate_hash: B256 },

    /// The L1 provider returned an error (network, RPC, etc.)
    #[error("L1 provider error: {0}")]
    L1Provider(String),

    /// The hash provided by the derivation delegation does not match the canonical chain.
    #[error("L1 inconsistency in {context} at block {number}: expected {expected}, got {actual}")]
    L1ValidationFailed { context: String, number: u64, expected: BlockHash, actual: BlockHash },
}
