use super::{
    BuildToken, Consolidated, EngineClientError, EngineClientResult, EngineContext, EngineError,
    EngineHandle, Sealed,
    handle::{Message, Reply},
};
use crate::NodeActor;
use async_trait::async_trait;
use kona_engine::{
    self as engine, BuildError, ConsolidateError, EngineClient, EnginePhase, EngineState,
    EngineSyncStateUpdate, FinalizeError, ForkchoiceError, ImportedBlockSink, InsertError,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::{
    sync::{mpsc, watch},
    time::{self, Duration, Instant},
};

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
}

impl<C: EngineClient + 'static> EngineActor<C> {
    /// Constructs the sole engine owner and its bounded public handle.
    pub fn new(
        client: Arc<C>,
        config: Arc<RollupConfig>,
        sink: Arc<dyn ImportedBlockSink>,
    ) -> (Self, EngineHandle) {
        Self::with_capacities(client, config, sink, 32, 1024)
    }

    fn with_capacities(
        client: Arc<C>,
        config: Arc<RollupConfig>,
        sink: Arc<dyn ImportedBlockSink>,
        command_capacity: usize,
        unsafe_capacity: usize,
    ) -> (Self, EngineHandle) {
        let (commands, command_rx) = mpsc::channel(command_capacity);
        let (unsafe_payloads, unsafe_rx) = mpsc::channel(unsafe_capacity);
        let (resets, reset_rx) = mpsc::channel(1);
        let (state_tx, state) = watch::channel(EngineState::default());
        let (pending, _) = watch::channel(0);
        let handle =
            EngineHandle { commands, unsafe_payloads, resets, state, pending: pending.clone() };
        (
            Self {
                client,
                config,
                sink,
                state: EngineState::default(),
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

    fn begin_reset(&mut self) {
        if self.state.phase == EnginePhase::Resetting {
            return;
        }
        self.state.epoch = self.state.epoch.checked_add(1).expect("engine epoch exhausted");
        self.state.phase = EnginePhase::Resetting;
        if self.active_unsafe.take().is_some() {
            self.complete();
        }
        while self.unsafe_payloads.try_recv().is_ok() {
            self.complete();
        }
        self.retry_at = Instant::now();
        self.retry_delay = Duration::from_millis(100);
        self.publish();
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
            self.state.phase == EnginePhase::Resetting
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
        if self.state.phase != EnginePhase::Resetting && self.state.el_sync_finished {
            if self.state.sync_state.finalized_head() == L2BlockInfo::default() {
                self.begin_reset();
            } else {
                self.state.phase = EnginePhase::Ready;
            }
        }
        self.publish();
        (result, Ok(()))
    }

    async fn command(&mut self, message: Message) -> Result<(), EngineError> {
        let label = match &message {
            Message::Start(..) => engine::Metrics::BUILD_TASK_LABEL,
            Message::Seal(..) => engine::Metrics::SEAL_TASK_LABEL,
            Message::Canonicalize(..) => engine::Metrics::INSERT_TASK_LABEL,
            Message::Consolidate(..) | Message::FollowSafe(..) => {
                engine::Metrics::CONSOLIDATE_TASK_LABEL
            }
            Message::Finalize(..) => engine::Metrics::FINALIZE_TASK_LABEL,
        };
        macro_rules! reply {
            ($tx:expr, $result:expr) => {{
                let (result, fatal) = self.finish(label, $result);
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
                let (result, fatal) = self.finish(label, result);
                let result = result.map(|outcome| Consolidated { state: self.state, outcome });
                self.complete();
                let _ = tx.send(result);
                fatal
            }
            Message::FollowSafe(context, block, tx) => {
                let result = async {
                    self.guard(context, true)?;
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
                let (result, fatal) = self.finish(label, result);
                let result = result.map(|outcome| Consolidated { state: self.state, outcome });
                self.complete();
                let _ = tx.send(result);
                fatal
            }
            Message::Finalize(epoch, block, source, tx) => {
                let result = async {
                    if epoch != self.state.epoch || self.state.phase == EnginePhase::Resetting {
                        return Err(Failure::Stale);
                    }
                    engine::finalize(
                        self.client.as_ref(),
                        &self.config,
                        &mut self.state,
                        block,
                        source,
                    )
                    .await
                    .map_err(|error| match error {
                        FinalizeError::Forkchoice(e) => forkchoice_failure(e),
                        FinalizeError::L2Mismatch(_) | FinalizeError::L1Mismatch(_) => {
                            Failure::Reset(error.to_string())
                        }
                        FinalizeError::Rpc(_) | FinalizeError::BlockNotFound(_) => {
                            Failure::Retry(error.to_string())
                        }
                        FinalizeError::NotSafe => Failure::Stale,
                        FinalizeError::BlockInfo(_) => Failure::Fatal(error.to_string()),
                    })
                }
                .await;
                reply!(tx, result)
            }
        }
    }

    async fn reset(&mut self) -> Result<(), EngineError> {
        let result = async {
            let start =
                engine::find_starting_forkchoice(&self.config, self.client.as_ref()).await?;
            engine::synchronize(
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
            self.state.pending_safe_head = start.safe;
            Ok::<(), engine::ResetError>(())
        }
        .await;
        match result {
            Ok(()) => {
                self.state.phase = if self.state.el_sync_finished {
                    EnginePhase::Ready
                } else {
                    EnginePhase::Syncing
                };
                self.retry_delay = Duration::from_millis(100);
                kona_macros::inc!(counter, engine::Metrics::ENGINE_RESET_COUNT);
                self.publish();
                for waiter in self.reset_waiters.drain(..) {
                    self.pending.send_modify(|n| *n -= 1);
                    let _ = waiter.send(Ok(()));
                }
            }
            Err(error) if error.is_temporary() => {
                self.publish();
                self.defer_retry();
                warn!(target:"engine",?error,"Reset dependency unavailable; retaining reset");
            }
            Err(error) => {
                self.publish();
                return Err(EngineError::Invariant(error.to_string()));
            }
        }
        Ok(())
    }

    async fn unsafe_payload(&mut self) -> Result<(), EngineError> {
        let Some((epoch, payload)) = self.active_unsafe.take() else { return Ok(()) };
        if epoch != self.state.epoch ||
            payload.block_number() <= self.state.sync_state.safe_head().block_info.number
        {
            self.complete();
            return Ok(());
        }
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
        let (result, fatal) = self.finish(engine::Metrics::INSERT_TASK_LABEL, result);
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
        // The reset control lane never sits behind unsafe admission or acknowledged commands.
        if self.reset_waiters.len() < 32 &&
            let Ok(waiter) = self.resets.try_recv()
        {
            self.begin_reset();
            self.reset_waiters.push(waiter);
            return Ok(());
        }
        let resetting = self.state.phase == EnginePhase::Resetting;
        if resetting && Instant::now() >= self.retry_at {
            return self.reset().await;
        }
        if !resetting {
            if self.prefer_command &&
                let Ok(message) = self.commands.try_recv()
            {
                self.prefer_command = false;
                return self.command(message).await;
            }
            if self.active_unsafe.is_none() {
                self.active_unsafe = self.unsafe_payloads.try_recv().ok();
            }
            if self.active_unsafe.is_some() && Instant::now() >= self.retry_at {
                self.prefer_command = true;
                return self.unsafe_payload().await;
            }
            if let Ok(message) = self.commands.try_recv() {
                self.prefer_command = false;
                return self.command(message).await;
            }
        }
        tokio::select! {
            biased;
            waiter = self.resets.recv(), if self.reset_waiters.len() < 32 => {
                let waiter = waiter.ok_or(EngineError::ChannelClosed)?;
                self.begin_reset();
                self.reset_waiters.push(waiter);
            }
            message = self.commands.recv() => {
                self.prefer_command = false;
                return self.command(message.ok_or(EngineError::ChannelClosed)?).await;
            }
            payload = self.unsafe_payloads.recv(), if !resetting && self.active_unsafe.is_none() => {
                self.active_unsafe = Some(payload.ok_or(EngineError::ChannelClosed)?);
            }
            _ = time::sleep_until(self.retry_at), if resetting || self.active_unsafe.is_some() => {}
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

#[cfg(test)]
#[path = "tests.rs"]
mod tests;
