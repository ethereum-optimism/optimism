use super::{
    BuildToken, Consolidated, EngineClientError, EngineClientResult, EngineContext, EngineError,
    EngineHandle, Sealed,
    finalizer::L2Finalizer,
    handle::{Message, Reply},
};
use crate::NodeActor;
use alloy_eips::{BlockId, BlockNumHash};
use async_trait::async_trait;
use kona_engine::{
    self as engine, BuildError, ConsolidateError, EngineClient, EnginePhase, EngineState,
    EngineSyncStateUpdate, FinalizationSource, FinalizeError, ForkchoiceError, ForkchoiceStatus,
    ImportedBlockSink, InsertError, L2ForkchoiceState, SyncStartError, SyncStartPolicy,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use kona_rpc::L1State;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::{
    sync::{mpsc, watch},
    time::{self, Duration, Instant},
};

/// Bootstrap trust is available only until the first fully accepted initialization.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Lifecycle {
    Loading,
    Bootstrap(SyncStartPolicy),
    Ready,
    Reset(SyncStartPolicy),
    /// An explicit trusted upstream target is not a new bootstrap exception.
    Following {
        prior: SyncStartPolicy,
    },
}

/// Serializes execution mutations, publishes accepted state, and owns every engine reset.
#[derive(Debug)]
pub struct EngineActor<C: EngineClient> {
    client: Arc<C>,
    config: Arc<RollupConfig>,
    sink: Arc<dyn ImportedBlockSink>,
    state: EngineState,
    state_tx: watch::Sender<EngineState>,
    pending: watch::Sender<usize>,
    commands: mpsc::Receiver<Message>,
    unsafe_payloads: mpsc::Receiver<(u64, OpExecutionPayloadEnvelope)>,
    resets: mpsc::Receiver<Reply<()>>,
    reset_waiters: Vec<Reply<()>>,
    active_unsafe: Option<(u64, OpExecutionPayloadEnvelope)>,
    retry_at: Instant,
    retry_delay: Duration,
    prefer_command: bool,
    /// Give admitted work one opportunity between periodic dependency attempts.
    yield_to_work: bool,
    /// Alternate due canonicality and finality checks when dependencies are slow.
    prefer_finality: bool,
    l1: watch::Receiver<L1State>,
    lifecycle: Lifecycle,
    recovery_at: Instant,
    recovery_delay: Duration,
    check_at: Instant,
    /// Applied origin identities at the last check attempt; dirty retains an unconfirmed result.
    origin_check_inputs: Vec<BlockNumHash>,
    origins_dirty: bool,
    /// A real requested target; observing `eth_syncing` alone does not establish one.
    sync_target: Option<L2ForkchoiceState>,
    /// Actual completed-safe local blocks and their real L1 inclusion sources.
    finalizer: L2Finalizer,
    /// A validated delegate's explicit trust claim, independent of local inclusion records.
    delegated_finalized: Option<L2BlockInfo>,
    finality_retry_at: Option<Instant>,
}

impl<C: EngineClient + 'static> EngineActor<C> {
    /// Constructs the sole engine owner and its bounded public handle.
    pub fn new(
        client: Arc<C>,
        config: Arc<RollupConfig>,
        sink: Arc<dyn ImportedBlockSink>,
        l1: watch::Receiver<L1State>,
    ) -> (Self, EngineHandle) {
        Self::with_capacities(client, config, sink, l1, 32, 1024)
    }

    fn with_capacities(
        client: Arc<C>,
        config: Arc<RollupConfig>,
        sink: Arc<dyn ImportedBlockSink>,
        l1: watch::Receiver<L1State>,
        command_capacity: usize,
        unsafe_capacity: usize,
    ) -> (Self, EngineHandle) {
        let (commands, command_rx) = mpsc::channel(command_capacity);
        let (unsafe_payloads, unsafe_rx) = mpsc::channel(unsafe_capacity);
        let (resets, reset_rx) = mpsc::channel(1);
        let initial = EngineState { epoch: 1, ..Default::default() };
        let (state_tx, state) = watch::channel(initial);
        let (pending, _) = watch::channel(0);
        let handle =
            EngineHandle { commands, unsafe_payloads, resets, state, pending: pending.clone() };
        (
            Self {
                client,
                config,
                sink,
                state: initial,
                state_tx,
                pending,
                commands: command_rx,
                unsafe_payloads: unsafe_rx,
                resets: reset_rx,
                reset_waiters: Vec::new(),
                active_unsafe: None,
                retry_at: Instant::now(),
                retry_delay: Duration::from_millis(100),
                prefer_command: true,
                yield_to_work: false,
                prefer_finality: false,
                l1,
                lifecycle: Lifecycle::Loading,
                recovery_at: Instant::now(),
                recovery_delay: Duration::from_millis(100),
                check_at: Instant::now(),
                origin_check_inputs: Vec::new(),
                origins_dirty: false,
                sync_target: None,
                finalizer: L2Finalizer::default(),
                delegated_finalized: None,
                finality_retry_at: None,
            },
            handle,
        )
    }

    fn complete(&self) {
        self.pending.send_modify(|n| *n -= 1);
    }

    fn publish(&self) {
        self.state.record_metrics();
        self.state_tx.send_replace(self.state);
    }

    const fn checkpoint(&self) -> SyncStartPolicy {
        SyncStartPolicy::Checkpoint {
            safe: self.state.sync_state.safe_head(),
            finalized: self.state.sync_state.finalized_head(),
        }
    }

    fn invalidate(&mut self, phase: EnginePhase) {
        self.state.epoch = self.state.epoch.checked_add(1).expect("engine epoch exhausted");
        self.state.phase = phase;
        self.yield_to_work = false;
        self.prefer_finality = false;
        self.finalizer.clear();
        self.delegated_finalized = None;
        self.finality_retry_at = None;
        if self.active_unsafe.take().is_some() {
            self.complete();
        }
        while self.unsafe_payloads.try_recv().is_ok() {
            self.complete();
        }
        self.recovery_at = Instant::now();
        self.recovery_delay = Duration::from_millis(100);
        self.publish();
    }

    fn begin_reset(&mut self) {
        match self.lifecycle {
            Lifecycle::Ready => {
                self.sync_target = None;
                self.lifecycle = Lifecycle::Reset(self.checkpoint());
                self.invalidate(EnginePhase::Resetting);
            }
            Lifecycle::Following { prior } => {
                self.sync_target = None;
                self.lifecycle = Lifecycle::Reset(prior);
                self.invalidate(EnginePhase::Resetting);
            }
            Lifecycle::Loading | Lifecycle::Bootstrap(_) | Lifecycle::Reset(_) => {
                // Join the retained recovery. No new epoch or trust allowance on a retry.
                if self.state.phase != EnginePhase::Syncing {
                    self.state.phase = EnginePhase::Resetting;
                }
                self.publish();
            }
        }
    }

    fn defer_recovery(&mut self) {
        self.recovery_at = Instant::now() + self.recovery_delay;
        self.recovery_delay = (self.recovery_delay * 2).min(Duration::from_secs(5));
    }

    fn defer_retry(&mut self) {
        self.retry_at = Instant::now() + self.retry_delay;
        self.retry_delay = (self.retry_delay * 2).min(Duration::from_secs(5));
    }

    fn guard(&self, context: EngineContext, safe: bool) -> Result<(), Failure> {
        let parent =
            if safe { self.state.pending_safe_head } else { self.state.sync_state.unsafe_head() };
        if context.epoch != self.state.epoch ||
            context.parent != parent ||
            self.state.phase != EnginePhase::Ready
        {
            return Err(Failure::Stale);
        }
        Ok(())
    }

    // Complete an attempt even if its caller disappeared. Partial accepted mutations must be
    // visible before either the acknowledgement or reset invalidation can reach another producer.
    fn finish<T>(
        &mut self,
        label: &'static str,
        result: Result<T, Failure>,
        prior: SyncStartPolicy,
        trusted_follow: bool,
        requested_parent: Option<L2BlockInfo>,
    ) -> (EngineClientResult<T>, Result<(), EngineError>) {
        match &result {
            Ok(_) => {
                kona_macros::inc!(counter, engine::Metrics::ENGINE_TASK_SUCCESS, label);
            }
            Err(error) => {
                let severity = match error {
                    Failure::Reset(_) => "Reset",
                    Failure::Fatal(_) => "Critical",
                    _ => "Temporary",
                };
                kona_macros::inc!(counter,engine::Metrics::ENGINE_TASK_FAILURE,label => severity);
            }
        }
        let result = match result {
            Ok(value) => Ok(value),
            Err(Failure::Reset(error)) => {
                self.begin_reset();
                Err(EngineClientError::Temporary(error))
            }
            Err(Failure::Retry(error)) => Err(EngineClientError::Temporary(error)),
            Err(Failure::Rejected(error)) => Err(EngineClientError::Rejected(error)),
            Err(Failure::Expired) => Err(EngineClientError::Expired),
            Err(Failure::Stale) => Err(EngineClientError::Stale),
            Err(Failure::Fatal(error)) => {
                self.publish();
                return (
                    Err(EngineClientError::Fatal(error.clone())),
                    Err(EngineError::Invariant(error)),
                );
            }
        };
        if self.state.phase == EnginePhase::Syncing {
            if result.is_ok() || (self.lifecycle == Lifecycle::Ready && requested_parent.is_some())
            {
                let mut target = self.heads();
                if result.is_err() &&
                    let Some(parent) = requested_parent
                {
                    target.un_safe = parent;
                }
                self.sync_target = Some(target);
            }
            if self.lifecycle == Lifecycle::Ready {
                self.lifecycle = if trusted_follow {
                    Lifecycle::Following { prior }
                } else {
                    Lifecycle::Reset(prior)
                };
                self.invalidate(EnginePhase::Syncing);
            }
        }
        self.publish();
        (result, Ok(()))
    }

    async fn command(&mut self, message: Message) -> Result<(), EngineError> {
        let prior = self.checkpoint();
        let trusted_follow = matches!(&message, Message::FollowSafe(..));
        let requested_parent = match &message {
            Message::Start(context, ..) | Message::Consolidate(context, ..) => Some(context.parent),
            _ => None,
        };
        let label = match &message {
            Message::Start(..) => engine::Metrics::BUILD_TASK_LABEL,
            Message::Seal(..) => engine::Metrics::SEAL_TASK_LABEL,
            Message::Canonicalize(..) => engine::Metrics::INSERT_TASK_LABEL,
            Message::Consolidate(..) | Message::FollowSafe(..) => {
                engine::Metrics::CONSOLIDATE_TASK_LABEL
            }
        };
        macro_rules! reply {
            ($tx:expr, $result:expr) => {{
                let (result, fatal) =
                    self.finish(label, $result, prior, trusted_follow, requested_parent);
                self.complete();
                let _ = $tx.send(result);
                fatal
            }};
        }
        match message {
            Message::Start(context, attrs, tx) => {
                let result = async {
                    self.guard(context, false)?;
                    if attrs.parent != context.parent {
                        return Err(Failure::Stale);
                    }
                    let id = engine::start_build(
                        self.client.as_ref(),
                        &self.config,
                        &mut self.state,
                        &attrs,
                    )
                    .await
                    .map_err(build_failure)?;
                    Ok(BuildToken {
                        context,
                        id,
                        timestamp: attrs.attributes.payload_attributes.timestamp,
                    })
                }
                .await;
                reply!(tx, result)
            }
            Message::Seal(token, tx) => {
                let result = async {
                    self.guard(token.context, false)?;
                    let payload = engine::get_payload(
                        self.client.as_ref(),
                        &self.config,
                        token.id,
                        token.timestamp,
                    )
                    .await
                    .map_err(|error| {
                        if error.as_error_resp().is_some_and(|e| e.code == -38001) {
                            Failure::Expired
                        } else {
                            Failure::Retry(error.to_string())
                        }
                    })?;
                    if payload.parent_hash() != token.context.parent.hash() ||
                        payload.timestamp() != token.timestamp
                    {
                        return Err(Failure::Fatal("getPayload returned a different build".into()));
                    }
                    Ok(Sealed { context: token.context, payload })
                }
                .await;
                reply!(tx, result)
            }
            Message::Canonicalize(sealed, tx) => {
                let result = async {
                    self.guard(sealed.context, false)?;
                    engine::insert_payload(
                        self.client.as_ref(),
                        &self.config,
                        &mut self.state,
                        &sealed.payload,
                        engine::ImportMode::Unsafe,
                        self.sink.as_ref(),
                    )
                    .await
                    .map_err(insert_failure)?;
                    Ok(sealed.payload)
                }
                .await;
                reply!(tx, result)
            }
            Message::Consolidate(context, attrs, tx) => {
                let previous_safe = self.state.sync_state.safe_head();
                let inclusion_source = attrs.derived_from;
                let result = async {
                    self.guard(context, true)?;
                    if attrs.parent != context.parent {
                        return Err(Failure::Stale);
                    }
                    let source = attrs.derived_from.ok_or_else(|| {
                        Failure::Fatal("local derivation has no inclusion source".into())
                    })?;
                    engine::consolidate(
                        self.client.as_ref(),
                        &self.config,
                        &mut self.state,
                        &attrs,
                        source.timestamp,
                        self.sink.as_ref(),
                    )
                    .await
                    .map_err(consolidate_failure)
                }
                .await;
                if let (Some(source), Ok(outcome)) = (inclusion_source, &result) &&
                    *outcome != engine::Consolidation::Dropped &&
                    self.state.phase == EnginePhase::Ready &&
                    self.state.sync_state.safe_head().block_info.number >
                        previous_safe.block_info.number
                {
                    self.finalizer.record(self.state.sync_state.safe_head(), source);
                }
                let (result, fatal) =
                    self.finish(label, result, prior, trusted_follow, requested_parent);
                let result = result.map(|outcome| Consolidated { state: self.state, outcome });
                self.complete();
                let _ = tx.send(result);
                fatal
            }
            Message::FollowSafe(context, block, trusted_finalized, tx) => {
                let result = async {
                    self.guard(context, true)?;
                    if trusted_finalized.block_info.number > block.block_info.number {
                        return Err(Failure::Rejected(
                            "delegate finalized head is ahead of its safe head".into(),
                        ));
                    }
                    engine::follow_safe(
                        self.client.as_ref(),
                        &self.config,
                        &mut self.state,
                        block,
                        self.sink.as_ref(),
                    )
                    .await
                    .map_err(consolidate_failure)
                }
                .await;
                let (result, fatal) =
                    self.finish(label, result, prior, trusted_follow, requested_parent);
                // Following may invalidate the old epoch inside finish. Retain only this
                // accepted request's claim afterward; stale/unready requests cannot install it.
                if result.is_ok() {
                    let finalized = self.state.sync_state.finalized_head().block_info;
                    self.delegated_finalized = (trusted_finalized.block_info.number >=
                        finalized.number &&
                        trusted_finalized.block_info.id() != finalized.id())
                    .then_some(trusted_finalized);
                }
                let result = result.map(|outcome| Consolidated { state: self.state, outcome });
                self.complete();
                let _ = tx.send(result);
                fatal
            }
        }
    }

    async fn recover(&mut self) -> Result<(), EngineError> {
        match self.recover_attempt().await {
            Ok(true) => {
                self.state.pending_safe_head = self.state.sync_state.safe_head();
                self.state.phase = EnginePhase::Ready;
                self.lifecycle = Lifecycle::Ready;
                self.sync_target = None;
                self.recovery_delay = Duration::from_millis(100);
                self.origin_check_inputs = self.origins();
                self.origins_dirty = false;
                kona_macros::inc!(counter, engine::Metrics::ENGINE_RESET_COUNT);
                self.publish();
                for waiter in self.reset_waiters.drain(..) {
                    self.pending.send_modify(|n| *n -= 1);
                    let _ = waiter.send(Ok(()));
                }
            }
            Ok(false) => {
                self.publish();
                self.defer_recovery();
            }
            Err(error) if error.is_temporary() => {
                self.publish();
                self.defer_recovery();
                warn!(target:"engine",?error,"Recovery dependency unavailable; retaining recovery");
            }
            Err(error) => {
                self.publish();
                return Err(EngineError::Invariant(error.to_string()));
            }
        }
        Ok(())
    }

    async fn recover_attempt(&mut self) -> Result<bool, engine::ResetError> {
        if self.lifecycle == Lifecycle::Loading {
            let heads = L2ForkchoiceState::current(&self.config, self.client.as_ref()).await?;
            if heads.finalized.block_info.number < self.config.genesis.l2.number ||
                heads.finalized.block_info.number > heads.safe.block_info.number ||
                heads.safe.block_info.number > heads.un_safe.block_info.number
            {
                return Err(SyncStartError::InvalidHeadOrder.into());
            }
            let policy = if heads.safe.block_info.number > self.config.genesis.l2.number {
                SyncStartPolicy::Checkpoint { safe: heads.safe, finalized: heads.finalized }
            } else {
                SyncStartPolicy::TrustedResume { finalized: heads.finalized }
            };
            let unsafe_head = self.sync_target.map_or(heads.un_safe, |target| target.un_safe);
            self.state.sync_state = self.state.sync_state.apply_update(EngineSyncStateUpdate {
                unsafe_head: Some(unsafe_head),
                local_safe_head: Some(heads.safe),
                safe_head: Some(heads.safe),
                finalized_head: Some(heads.finalized),
            });
            self.state.pending_safe_head = heads.safe;
            if self.sync_target.is_some() {
                // An earlier unsafe payload may have initiated sync before safety labels existed.
                self.sync_target = Some(self.heads());
            }
            self.state.phase = EnginePhase::Syncing;
            self.lifecycle = Lifecycle::Bootstrap(policy);
        }

        if let alloy_rpc_types_eth::SyncStatus::Info(info) =
            self.client.syncing().await.map_err(SyncStartError::from)?
        {
            // A fresh EL can report syncing until its first FCU, without knowing any target.
            // Confirm the configured genesis to break that startup cycle. A real advertised
            // later target must still complete sync before recovery. Reconfirming an earlier
            // SYNCING response for this exact genesis cannot redirect a later target.
            let genesis = self.config.genesis.l2.number;
            let heads = self.heads();
            let initializing_genesis = matches!(self.lifecycle, Lifecycle::Bootstrap(_)) &&
                self.sync_target.is_none_or(|target| target == heads) &&
                heads.un_safe.block_info.id() == self.config.genesis.l2 &&
                heads.safe.block_info.id() == self.config.genesis.l2 &&
                heads.finalized.block_info.id() == self.config.genesis.l2 &&
                info.starting_block <= genesis &&
                info.current_block <= genesis &&
                info.highest_block <= genesis;
            if !initializing_genesis {
                self.state.phase = EnginePhase::Syncing;
                return Ok(false);
            }
        }
        // An EL's latest label can lag a target we actually requested. An eth_syncing
        // observation alone is insufficient: in that case walk the fresh EL latest instead.
        if let Some(target) = self.sync_target {
            let accepted = engine::synchronize(
                self.client.as_ref(),
                &mut self.state,
                EngineSyncStateUpdate {
                    unsafe_head: Some(target.un_safe),
                    safe_head: Some(target.safe),
                    local_safe_head: Some(target.safe),
                    finalized_head: Some(target.finalized),
                },
            )
            .await?;
            if accepted == ForkchoiceStatus::Syncing {
                return Ok(false);
            }
            // Following retains its exact target until its body and origins are usable too.
            if !matches!(self.lifecycle, Lifecycle::Following { .. }) {
                self.sync_target = None;
            }
        }
        if matches!(self.lifecycle, Lifecycle::Following { .. }) {
            let safe = self.state.sync_state.safe_head();
            let id: BlockId = safe.block_info.number.into();
            let block = self
                .client
                .get_l2_block(id)
                .full()
                .await
                .map_err(SyncStartError::from)?
                .ok_or(SyncStartError::BlockNotFound(id))?;
            let actual =
                L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &self.config.genesis)
                    .map_err(SyncStartError::from)?;
            if actual != safe || !self.origins_are_canonical().await? {
                self.begin_reset();
                return Ok(false);
            }
            return Ok(true);
        }
        let policy = match self.lifecycle {
            Lifecycle::Bootstrap(policy) | Lifecycle::Reset(policy) => policy,
            Lifecycle::Loading | Lifecycle::Ready | Lifecycle::Following { .. } => {
                unreachable!("recovery policy established above")
            }
        };
        let start =
            engine::find_starting_forkchoice(&self.config, self.client.as_ref(), policy).await?;
        let accepted = engine::synchronize(
            self.client.as_ref(),
            &mut self.state,
            EngineSyncStateUpdate {
                unsafe_head: Some(start.un_safe),
                local_safe_head: Some(start.safe),
                safe_head: Some(start.safe),
                finalized_head: Some(start.finalized),
            },
        )
        .await?;
        if accepted == ForkchoiceStatus::Syncing {
            self.sync_target = Some(start);
        }
        Ok(accepted == ForkchoiceStatus::Valid)
    }

    const fn heads(&self) -> L2ForkchoiceState {
        L2ForkchoiceState {
            un_safe: self.state.sync_state.unsafe_head(),
            safe: self.state.sync_state.safe_head(),
            finalized: self.state.sync_state.finalized_head(),
        }
    }

    fn origins(&self) -> Vec<BlockNumHash> {
        let mut origins = Vec::with_capacity(4);
        for block in [
            self.state.sync_state.unsafe_head(),
            self.state.pending_safe_head,
            self.state.sync_state.safe_head(),
            self.state.sync_state.finalized_head(),
        ] {
            if !origins.contains(&block.l1_origin) {
                origins.push(block.l1_origin);
            }
        }
        origins
    }

    async fn origins_are_canonical(&self) -> Result<bool, SyncStartError> {
        let observed = *self.l1.borrow();
        let visible = match observed.head {
            Some(head) => head.number,
            None => {
                self.client
                    .get_l1_block(BlockId::latest())
                    .await?
                    .ok_or(SyncStartError::BlockNotFound(BlockId::latest()))?
                    .header
                    .number
            }
        };
        for origin in self.origins() {
            if origin.number > visible {
                continue;
            }
            let id: BlockId = origin.number.into();
            let block =
                self.client.get_l1_block(id).await?.ok_or(SyncStartError::BlockNotFound(id))?;
            if block.header.number != origin.number {
                return Err(SyncStartError::InvalidL1Response(id));
            }
            if block.header.hash != origin.hash {
                return Ok(false);
            }
        }
        Ok(true)
    }

    async fn check_origins(&mut self) -> Result<(), EngineError> {
        self.origin_check_inputs = self.origins();
        match self.origins_are_canonical().await {
            Ok(true) => {
                self.origins_dirty = false;
            }
            Ok(false) => self.begin_reset(),
            Err(error @ SyncStartError::InvalidL1Response(_)) => {
                return Err(EngineError::Invariant(error.to_string()));
            }
            Err(error) => {
                // A hole is dependency loss, not proof of reorg. Retry without another event.
                warn!(target:"engine",?error,"Retrying canonical L1 origin check");
                self.check_at = Instant::now() + Duration::from_secs(1);
            }
        }
        Ok(())
    }

    fn has_finality_candidate(&self) -> bool {
        self.delegated_finalized.is_some() ||
            self.l1
                .borrow()
                .finalized
                .is_some_and(|signal| self.finalizer.candidate(signal.number).is_some())
    }

    /// Finality is owner-generated work: it has no mailbox admission to decrement.
    async fn finalize(&mut self) -> Result<(), EngineError> {
        let signal = self.l1.borrow().finalized;
        let record = if self.delegated_finalized.is_none() {
            signal.and_then(|signal| self.finalizer.candidate(signal.number))
        } else {
            None
        };
        let (block, source) = match (self.delegated_finalized, record) {
            (Some(block), _) => (block.block_info.id(), FinalizationSource::Delegated),
            (None, Some(record)) => {
                (record.block.block_info.id(), FinalizationSource::Local(record.source.id()))
            }
            (None, None) => return Ok(()),
        };
        let prior = self.checkpoint();
        let result = async {
            let finalized = self.state.sync_state.finalized_head().block_info;
            if block.number < finalized.number || block == finalized.id() {
                return Ok(());
            }
            if block.number > self.state.sync_state.safe_head().block_info.number {
                return Err(Failure::Reset(
                    "retained finality candidate is above completed safe".into(),
                ));
            }
            if record.is_some() {
                let signal =
                    signal.expect("local candidate requires an observed finalized L1 block");
                let id: BlockId = signal.number.into();
                let canonical = self
                    .client
                    .get_l1_block(id)
                    .await
                    .map_err(|error| Failure::Retry(error.to_string()))?
                    .ok_or_else(|| {
                        Failure::Retry(format!("finalized L1 block unavailable: {id}"))
                    })?;
                if canonical.header.number != signal.number {
                    return Err(Failure::Fatal(SyncStartError::InvalidL1Response(id).to_string()));
                }
                if canonical.header.hash != signal.hash {
                    return Err(Failure::Reset(
                        "observed finalized L1 block is no longer canonical".into(),
                    ));
                }
            }
            engine::finalize(self.client.as_ref(), &self.config, &mut self.state, block, source)
                .await
                .map_err(finalize_failure)
        }
        .await;
        let retry = matches!(result, Err(Failure::Retry(_)));
        if result.is_ok() {
            if let Some(record) = record {
                self.finalizer.acknowledge(record);
            } else {
                self.delegated_finalized = None;
            }
            self.finality_retry_at = None;
        }
        let (result, fatal) =
            self.finish(engine::Metrics::FINALIZE_TASK_LABEL, result, prior, false, None);
        if retry && self.lifecycle == Lifecycle::Ready {
            self.finality_retry_at = Some(Instant::now() + Duration::from_millis(100));
        }
        if let Err(error) = result {
            debug!(target: "engine", ?error, "Finality attempt did not apply");
        }
        fatal
    }

    async fn unsafe_payload(&mut self) -> Result<(), EngineError> {
        let Some((epoch, payload)) = self.active_unsafe.take() else { return Ok(()) };
        if epoch != self.state.epoch ||
            payload.block_number() <= self.state.sync_state.safe_head().block_info.number
        {
            self.complete();
            return Ok(());
        }
        let prior = self.checkpoint();
        let result = engine::insert_payload(
            self.client.as_ref(),
            &self.config,
            &mut self.state,
            &payload,
            engine::ImportMode::Unsafe,
            self.sink.as_ref(),
        )
        .await
        .map_err(insert_failure);
        let retry = matches!(result, Err(Failure::Retry(_)));
        let (result, fatal) =
            self.finish(engine::Metrics::INSERT_TASK_LABEL, result, prior, false, None);
        if retry && self.state.phase != EnginePhase::Resetting {
            self.active_unsafe = Some((epoch, payload));
            self.defer_retry();
        } else {
            self.complete();
            self.retry_delay = Duration::from_millis(100);
            self.retry_at = Instant::now();
        }
        if let Err(error) = result {
            debug!(target:"engine",?error,"Unsafe payload attempt did not apply");
        }
        fatal
    }
}

#[async_trait]
impl<C: EngineClient + 'static> NodeActor for EngineActor<C> {
    type Error = EngineError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        // Reset admission and invalidation always precede bounded dependency work.
        if self.reset_waiters.len() < 32 &&
            let Ok(waiter) = self.resets.try_recv()
        {
            self.begin_reset();
            self.reset_waiters.push(waiter);
            return Ok(());
        }
        if self.l1.has_changed().map_err(|_| EngineError::ChannelClosed)? {
            self.l1.borrow_and_update();
            self.origins_dirty = true;
            self.check_at = Instant::now();
        }
        let recovering = self.lifecycle != Lifecycle::Ready;
        if recovering && Instant::now() >= self.recovery_at {
            return self.recover().await;
        }
        let inputs_changed = !recovering && self.origins() != self.origin_check_inputs;
        if inputs_changed {
            self.origins_dirty = true;
            self.check_at = Instant::now();
            self.yield_to_work = false;
            self.prefer_finality = false;
        }
        // Bootstrap/SYNCING still accepts unsafe targets, while producer commands require Ready.
        let unsafe_enabled = !matches!(self.lifecycle, Lifecycle::Following { .. }) &&
            (matches!(self.lifecycle, Lifecycle::Loading | Lifecycle::Bootstrap(_)) ||
                self.state.phase != EnginePhase::Resetting);
        let unsafe_pending = self.active_unsafe.is_some() || !self.unsafe_payloads.is_empty();
        let unsafe_ready = unsafe_enabled && Instant::now() >= self.retry_at && unsafe_pending;
        let yield_to_work = std::mem::take(&mut self.yield_to_work) &&
            (!self.commands.is_empty() || self.commands.is_closed() || unsafe_ready);
        let canonicality_due = !recovering && self.origins_dirty && Instant::now() >= self.check_at;
        let finality_due = !recovering &&
            self.state.phase == EnginePhase::Ready &&
            self.has_finality_candidate() &&
            self.finality_retry_at.is_none_or(|deadline| deadline <= Instant::now());
        if !yield_to_work && canonicality_due && (!self.prefer_finality || !finality_due) {
            // New inputs need their first safety check before another producer changes them.
            // Let due finality run next; stable-input retries still yield to admitted work.
            self.yield_to_work = !inputs_changed;
            self.prefer_finality = true;
            return self.check_origins().await;
        }
        if !yield_to_work && finality_due {
            self.yield_to_work = true;
            self.prefer_finality = false;
            return self.finalize().await;
        }
        if self.prefer_command &&
            let Ok(message) = self.commands.try_recv()
        {
            self.prefer_command = false;
            return self.command(message).await;
        }
        if unsafe_enabled && self.active_unsafe.is_none() {
            self.active_unsafe = self.unsafe_payloads.try_recv().ok();
        }
        if unsafe_enabled && self.active_unsafe.is_some() && Instant::now() >= self.retry_at {
            self.prefer_command = true;
            return self.unsafe_payload().await;
        }
        if let Ok(message) = self.commands.try_recv() {
            self.prefer_command = false;
            return self.command(message).await;
        }
        let finality_deadline = self.finality_retry_at;
        let wait_for_finality = !recovering && self.has_finality_candidate();
        tokio::select! {
            biased;
            waiter = self.resets.recv(), if self.reset_waiters.len() < 32 => {
                let waiter = waiter.ok_or(EngineError::ChannelClosed)?;
                self.begin_reset();
                self.reset_waiters.push(waiter);
            }
            result = self.l1.changed() => {
                result.map_err(|_| EngineError::ChannelClosed)?;
                self.l1.borrow_and_update();
                self.origins_dirty = true;
                self.check_at = Instant::now();
            }
            _ = time::sleep_until(self.recovery_at), if recovering => {}
            _ = time::sleep_until(self.check_at), if !recovering && self.origins_dirty => {}
            _ = async {
                if let Some(deadline) = finality_deadline { time::sleep_until(deadline).await; }
                else { std::future::pending::<()>().await; }
            }, if wait_for_finality => {}
            message = self.commands.recv() => {
                self.prefer_command = false;
                return self.command(message.ok_or(EngineError::ChannelClosed)?).await;
            }
            payload = self.unsafe_payloads.recv(), if unsafe_enabled && self.active_unsafe.is_none() => {
                self.active_unsafe = Some(payload.ok_or(EngineError::ChannelClosed)?);
            }
            _ = time::sleep_until(self.retry_at), if unsafe_enabled && self.active_unsafe.is_some() => {}
        }
        Ok(())
    }
}

#[derive(Debug)]
enum Failure {
    Retry(String),
    Reset(String),
    Rejected(String),
    Expired,
    Stale,
    Fatal(String),
}

fn forkchoice_failure(error: ForkchoiceError) -> Failure {
    match error {
        ForkchoiceError::Rpc(_) |
        ForkchoiceError::InvalidState |
        ForkchoiceError::UnexpectedStatus(_) => Failure::Reset(error.to_string()),
        ForkchoiceError::FinalizedAheadOfUnsafe(_, _) => Failure::Fatal(error.to_string()),
    }
}

fn build_failure(error: BuildError) -> Failure {
    match error {
        BuildError::Forkchoice(e) => forkchoice_failure(e),
        BuildError::Rejected(_) | BuildError::InvalidAttributes(_) => {
            Failure::Reset(error.to_string())
        }
        BuildError::Syncing | BuildError::MissingPayloadId => Failure::Retry(error.to_string()),
        BuildError::UnexpectedStatus(_) => Failure::Reset(error.to_string()),
    }
}

fn insert_failure(error: InsertError) -> Failure {
    match error {
        InsertError::Forkchoice(e) => forkchoice_failure(e),
        InsertError::Rpc(_) => Failure::Retry(error.to_string()),
        InsertError::Rejected(ref status)
            if matches!(
                status,
                alloy_rpc_types_engine::PayloadStatusEnum::Syncing |
                    alloy_rpc_types_engine::PayloadStatusEnum::Accepted
            ) =>
        {
            Failure::Retry(error.to_string())
        }
        InsertError::Rejected(_) => Failure::Rejected(error.to_string()),
        InsertError::Payload(_) | InsertError::BlockInfo(_) => Failure::Fatal(error.to_string()),
    }
}

fn consolidate_failure(error: ConsolidateError) -> Failure {
    match error {
        ConsolidateError::Build(e) => build_failure(e),
        ConsolidateError::Insert(e) => insert_failure(e),
        ConsolidateError::Forkchoice(e) => forkchoice_failure(e),
        ConsolidateError::Read(_) | ConsolidateError::GetPayload(_) => {
            Failure::Retry(error.to_string())
        }
        ConsolidateError::DepositsOnlyRejected(_) => Failure::Fatal(error.to_string()),
    }
}

fn finalize_failure(error: FinalizeError) -> Failure {
    match error {
        FinalizeError::Forkchoice(error) => forkchoice_failure(error),
        FinalizeError::L2Mismatch(_) |
        FinalizeError::L1Mismatch(_) |
        FinalizeError::NotSafe |
        FinalizeError::Syncing => Failure::Reset(error.to_string()),
        FinalizeError::Rpc(_) | FinalizeError::BlockNotFound(_) => {
            Failure::Retry(error.to_string())
        }
        FinalizeError::BlockInfo(_) => Failure::Fatal(error.to_string()),
    }
}

#[cfg(test)]
#[path = "tests.rs"]
mod tests;
