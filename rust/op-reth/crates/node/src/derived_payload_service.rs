//! A payload service that builds derived payloads as soon as it receives their attributes.
//!
//! The engine answers `engine_forkchoiceUpdated` with the payload id the payload service returns,
//! with `-38003` (invalid payload attributes) if the service rejects the attributes with
//! [`PayloadBuilderError::InvalidAttributes`], and with `-32603` if it fails to start the build in
//! any other way or drops the reply.

use alloy_consensus::BlockHeader;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadId;
use futures_util::{
    FutureExt,
    future::{BoxFuture, Shared},
};
use metrics::{Counter, Histogram};
use op_alloy_consensus::DEPOSIT_TX_TYPE_ID;
use reth_basic_payload_builder::{BuildArguments, HeaderForPayload, PayloadBuilder, PayloadConfig};
use reth_evm::ConfigureEvm;
use reth_metrics::Metrics;
use reth_node_api::{BuiltPayload, Events, NodeTypes, PayloadBuilderError, PayloadTypes};
use reth_node_builder::{BuilderContext, FullNodeTypes, components::PayloadServiceBuilder};
use reth_optimism_payload_builder::{
    OpPayloadAttrs, OpPayloadBuilder, is_invalid_derived_payload_error,
};
use reth_payload_builder::{
    BuildNewPayload, PayloadBuilderHandle, PayloadBuilderResources, PayloadKind,
    PayloadServiceCommand,
};
use reth_provider::HeaderProvider;
use reth_tasks::Runtime;
use reth_tracing::tracing::{Span, debug, warn};
use reth_transaction_pool::TransactionPool;
use std::{
    collections::{HashMap, VecDeque},
    pin::Pin,
    sync::{Arc, Mutex, MutexGuard},
    time::Instant,
};
use tokio::sync::{broadcast, mpsc, oneshot};

/// Number of built derived payloads kept for `engine_getPayload`.
const BUILT_PAYLOADS: usize = 4;

/// Capacity of the payload event channel, as in reth's payload service.
const PAYLOAD_EVENTS_BUFFER_SIZE: usize = 20;

/// Whether derived attributes that carry only deposits are built at `engine_forkchoiceUpdated`,
/// too.
///
/// Deposits cannot make a payload invalid, so the wrapped payload service can build these.
const BUILDS_DEPOSIT_ONLY_ATTRIBUTES: bool = false;

/// Returns whether the payload of `attributes` is built while `engine_forkchoiceUpdated` is
/// answered, instead of by the wrapped payload service.
///
/// These are derived (`noTxPool`) attributes, except those that carry only deposits (see
/// `BUILDS_DEPOSIT_ONLY_ATTRIBUTES`). Any other transaction can make the payload invalid, which
/// the engine must report in its `engine_forkchoiceUpdated` response.
fn builds_at_forkchoice_updated(attributes: &OpPayloadAttrs) -> bool {
    attributes.no_tx_pool == Some(true) &&
        (BUILDS_DEPOSIT_ONLY_ATTRIBUTES || has_non_deposit_transaction(attributes))
}

fn has_non_deposit_transaction(attributes: &OpPayloadAttrs) -> bool {
    attributes.transactions.iter().flatten().any(|tx| tx.first() != Some(&DEPOSIT_TX_TYPE_ID))
}

/// Wraps a [`PayloadServiceBuilder`] so that the payload service builds derived payloads with
/// op-reth's payload builder before it answers `engine_forkchoiceUpdated`.
///
/// Derived payloads are those of `noTxPool` attributes that carry a non-deposit transaction. The
/// wrapped payload service builds every other payload.
///
/// An OP Stack node must wrap its payload service in this. `engine_forkchoiceUpdated` has to reject
/// derived attributes whose transactions make the payload invalid, so that the rollup node
/// replaces the block with a deposits-only one; a payload service on its own only finds out while
/// it builds the payload, after the response.
///
/// The derived build uses the node's EVM configuration, the one `engine_newPayload` validates
/// with, whatever the wrapped payload service builds with.
#[derive(Debug, Clone)]
pub struct DerivedPayloadServiceBuilder<S> {
    inner: S,
}

impl<S> DerivedPayloadServiceBuilder<S> {
    /// Wraps `inner`.
    pub const fn new(inner: S) -> Self {
        Self { inner }
    }
}

impl<Node, Pool, Evm, S> PayloadServiceBuilder<Node, Pool, Evm> for DerivedPayloadServiceBuilder<S>
where
    Node: FullNodeTypes<
            Types: NodeTypes<Payload: PayloadTypes<PayloadAttributes = OpPayloadAttrs>>,
            Provider: HeaderProvider<
                Header = HeaderForPayload<
                    <<Node::Types as NodeTypes>::Payload as PayloadTypes>::BuiltPayload,
                >,
            >,
        >,
    Pool: TransactionPool + Clone + 'static,
    Evm: ConfigureEvm + 'static,
    S: PayloadServiceBuilder<Node, Pool, Evm>,
    OpPayloadBuilder<Pool, Node::Provider, Evm>: PayloadBuilder<
            Attributes = OpPayloadAttrs,
            BuiltPayload = <<Node::Types as NodeTypes>::Payload as PayloadTypes>::BuiltPayload,
        > + 'static,
{
    async fn spawn_payload_builder_service(
        self,
        ctx: &BuilderContext<Node>,
        pool: Pool,
        evm_config: Evm,
    ) -> eyre::Result<PayloadBuilderHandle<<Node::Types as NodeTypes>::Payload>> {
        let wrapped =
            self.inner.spawn_payload_builder_service(ctx, pool.clone(), evm_config.clone()).await?;
        let builder = OpDerivedPayloadBuilder::new(
            ctx.provider().clone(),
            OpPayloadBuilder::new(pool, ctx.provider().clone(), evm_config),
        );
        Ok(DerivedPayloadService::spawn(wrapped, builder, ctx.task_executor().clone()).await?)
    }
}

/// Builds derived payloads for a [`DerivedPayloadService`].
trait DerivedPayloadBuilder<P>: Send + Sync + 'static {
    /// Builds the payload `id` for `attributes` on top of the block `parent_hash`, with the
    /// engine's `resources`.
    fn build(
        &self,
        parent_hash: B256,
        attributes: OpPayloadAttrs,
        id: PayloadId,
        resources: PayloadBuilderResources,
    ) -> Result<P, PayloadBuilderError>;
}

/// [`DerivedPayloadBuilder`] that builds with a [`PayloadBuilder`], such as op-reth's.
struct OpDerivedPayloadBuilder<Client, B> {
    client: Client,
    builder: B,
}

impl<Client, B> OpDerivedPayloadBuilder<Client, B> {
    /// Creates a new instance that looks parent headers up in `client` and builds with `builder`.
    const fn new(client: Client, builder: B) -> Self {
        Self { client, builder }
    }
}

impl<Client, B> DerivedPayloadBuilder<B::BuiltPayload> for OpDerivedPayloadBuilder<Client, B>
where
    Client: HeaderProvider<Header = HeaderForPayload<B::BuiltPayload>> + Send + Sync + 'static,
    B: PayloadBuilder<Attributes = OpPayloadAttrs> + 'static,
{
    fn build(
        &self,
        parent_hash: B256,
        attributes: OpPayloadAttrs,
        id: PayloadId,
        mut resources: PayloadBuilderResources,
    ) -> Result<B::BuiltPayload, PayloadBuilderError> {
        let parent = self
            .client
            .sealed_header_by_hash(parent_hash)?
            .ok_or(PayloadBuilderError::MissingParentHeader(parent_hash))?;
        let args = BuildArguments::new(
            Default::default(),
            resources.take_execution_cache(),
            resources.take_state_root_handle(),
            PayloadConfig::new(Arc::new(parent), attributes, id),
            Default::default(),
            None,
        );
        self.builder.try_build(args)?.into_payload().ok_or(PayloadBuilderError::MissingPayload)
    }
}

/// How building derived attributes ended.
#[derive(Debug, Clone)]
enum DerivedBuild {
    /// The payload was built and is kept for `engine_getPayload`.
    Built,
    /// The attributes' transactions make the payload invalid.
    Invalid(Arc<PayloadBuilderError>),
    /// The build failed for a reason local to this node, so it says nothing about the
    /// attributes.
    Failed(Arc<PayloadBuilderError>),
}

/// Metrics of derived payloads built by the [`DerivedPayloadService`].
#[derive(Metrics)]
#[metrics(scope = "optimism_payload_builder.derived")]
struct DerivedPayloadMetrics {
    /// Duration of a derived payload build, including the state root.
    build_duration: Histogram,
    /// Derived payloads built.
    built: Counter,
    /// Derived attributes rejected because their transactions make the payload invalid.
    invalid: Counter,
    /// Derived builds that failed for a node-local reason and went to the wrapped service.
    failed: Counter,
}

type SharedBuild = Shared<BoxFuture<'static, DerivedBuild>>;

/// The future a payload service answers `Resolve` with.
type PayloadFuture<P> = Pin<Box<dyn Future<Output = Result<P, PayloadBuilderError>> + Send>>;

/// A derived payload kept for `engine_getPayload`.
struct Built<P> {
    id: PayloadId,
    attributes: OpPayloadAttrs,
    payload: P,
    resolved: bool,
}

struct State<P> {
    building: HashMap<PayloadId, (OpPayloadAttrs, SharedBuild)>,
    built: VecDeque<Built<P>>,
}

/// A payload service that builds derived payloads itself and passes every other command to the
/// payload service it wraps.
///
/// A derived payload is built once, off the service loop, before the engine gets its payload id,
/// which only the forkchoiceUpdated response waits for. A request whose id and attributes match a
/// payload that is being built waits for that build, and one that matches a built payload is
/// answered at once, so a rollup node that retries never starts a second build. A build that fails
/// for a node-local reason goes to the wrapped service, like any other payload.
///
/// Payloads it built are served like a payload job's: their first resolution emits
/// [`Events::BuiltPayload`], so the engine inserts the executed block and `engine_newPayload` does
/// not execute it again. Subscribers receive the wrapped service's events, too.
struct DerivedPayloadService<T: PayloadTypes> {
    wrapped: PayloadBuilderHandle<T>,
    builder: Box<dyn DerivedPayloadBuilder<T::BuiltPayload>>,
    runtime: Runtime,
    events: broadcast::Sender<Events<T>>,
    state: Mutex<State<T::BuiltPayload>>,
    metrics: DerivedPayloadMetrics,
}

impl<T> DerivedPayloadService<T>
where
    T: PayloadTypes<PayloadAttributes = OpPayloadAttrs>,
{
    /// Spawns the service on `runtime` in front of the payload service behind `wrapped`, and
    /// returns its handle.
    async fn spawn(
        wrapped: PayloadBuilderHandle<T>,
        builder: impl DerivedPayloadBuilder<T::BuiltPayload>,
        runtime: Runtime,
    ) -> Result<PayloadBuilderHandle<T>, PayloadBuilderError> {
        let mut wrapped_events = wrapped.subscribe().await?.receiver;
        let (events, _) = broadcast::channel(PAYLOAD_EVENTS_BUFFER_SIZE);
        let forwarded = events.clone();
        runtime.spawn_task(async move {
            loop {
                match wrapped_events.recv().await {
                    Ok(event) => {
                        let _ = forwarded.send(event);
                    }
                    // A lost `BuiltPayload` makes `engine_newPayload` execute that block again.
                    Err(broadcast::error::RecvError::Lagged(skipped)) => {
                        warn!(target: "payload_builder", skipped, "Missed events of the wrapped payload service");
                    }
                    Err(broadcast::error::RecvError::Closed) => break,
                }
            }
        });

        let service = Arc::new(Self {
            wrapped,
            builder: Box::new(builder),
            runtime: runtime.clone(),
            events,
            state: Mutex::new(State {
                building: HashMap::new(),
                built: VecDeque::with_capacity(BUILT_PAYLOADS),
            }),
            metrics: DerivedPayloadMetrics::default(),
        });
        let (tx, mut commands) = mpsc::unbounded_channel();
        runtime.spawn_critical_task("derived payload service", async move {
            while let Some(command) = commands.recv().await {
                service.on_command(command);
            }
        });
        Ok(PayloadBuilderHandle::new(tx))
    }

    /// Handles `command` without waiting for any build or for the wrapped service.
    fn on_command(self: &Arc<Self>, command: PayloadServiceCommand<T>) {
        match command {
            PayloadServiceCommand::BuildNewPayload(input, span, tx)
                if builds_at_forkchoice_updated(&input.attributes) =>
            {
                self.build(*input, span, tx);
            }
            PayloadServiceCommand::BuildNewPayload(input, span, tx) => {
                let answer = self.build_in_wrapped_service(*input, &span);
                self.runtime.spawn_task(async move {
                    let _ = tx.send(answer.await);
                });
            }
            PayloadServiceCommand::BestPayload(id, tx) => {
                if let Some(payload) = self.built(id) {
                    let _ = tx.send(Some(Ok(payload)));
                } else {
                    let wrapped = self.wrapped.clone();
                    self.forward(async move {
                        let _ = tx.send(wrapped.best_payload(id).await);
                    });
                }
            }
            PayloadServiceCommand::PayloadTimestamp(id, tx) => {
                if let Some(payload) = self.built(id) {
                    let _ = tx.send(Some(Ok(payload.block().header().timestamp())));
                } else {
                    let wrapped = self.wrapped.clone();
                    self.forward(async move {
                        let _ = tx.send(wrapped.payload_timestamp(id).await);
                    });
                }
            }
            PayloadServiceCommand::Resolve(id, kind, tx) => {
                if let Some(payload) = self.resolve(id) {
                    let _ = tx.send(Some(Box::pin(std::future::ready(Ok(payload)))));
                } else {
                    self.resolve_in_wrapped_service(id, kind, tx);
                }
            }
            PayloadServiceCommand::Subscribe(tx) => {
                let _ = tx.send(self.events.subscribe());
            }
        }
    }

    /// Hands `input` to the wrapped service, within `span`, and returns its answer.
    fn build_in_wrapped_service(
        &self,
        input: BuildNewPayload<OpPayloadAttrs>,
        span: &Span,
    ) -> impl Future<Output = Result<PayloadId, PayloadBuilderError>> + Send + use<T> {
        let answer = {
            let _entered = span.enter();
            self.wrapped.send_new_payload(input)
        };
        async move { answer.await.unwrap_or_else(|closed| Err(closed.into())) }
    }

    /// Polls `forward` once, which sends its command to the wrapped service, and finishes it on a
    /// task. Commands thus reach the wrapped service in the order this service receives them.
    fn forward(&self, forward: impl Future<Output = ()> + Send + 'static) {
        let mut forward = Box::pin(forward);
        if (&mut forward).now_or_never().is_none() {
            self.runtime.spawn_task(forward);
        }
    }

    /// Resolves `id` in the wrapped service. Dropping the request cancels the resolution there,
    /// too.
    fn resolve_in_wrapped_service(
        &self,
        id: PayloadId,
        kind: PayloadKind,
        mut tx: oneshot::Sender<Option<PayloadFuture<T::BuiltPayload>>>,
    ) {
        let wrapped = self.wrapped.clone();
        self.forward(async move {
            let resolved = wrapped.resolve_kind(id, kind);
            let resolved = tokio::select! {
                resolved = resolved => resolved,
                () = tx.closed() => return,
            };
            let _ = tx.send(resolved.map(|payload| {
                Box::pin(std::future::ready(payload)) as PayloadFuture<T::BuiltPayload>
            }));
        });
    }

    /// Builds the derived payload of `input` unless it is built or being built already, and
    /// answers with its id once it is built. Transactions that make the payload invalid reject the
    /// attributes. A node-local failure hands `input` to the wrapped service.
    fn build(
        self: &Arc<Self>,
        input: BuildNewPayload<OpPayloadAttrs>,
        span: Span,
        tx: oneshot::Sender<Result<PayloadId, PayloadBuilderError>>,
    ) {
        let id = input.payload_id();
        let BuildNewPayload { attributes, parent_hash, mut resources } = input;
        // Held until the answer, so the engine keeps the in-memory state the build reads.
        let leases = resources.take_leases();
        let build = {
            let mut state = self.state();
            if state.built.iter().any(|built| built.id == id && built.attributes == attributes) {
                let _ = tx.send(Ok(id));
                return;
            }
            match state.building.get(&id) {
                Some((building, build)) if *building == attributes => build.clone(),
                _ => {
                    let build = self.spawn_build(
                        parent_hash,
                        attributes.clone(),
                        id,
                        resources,
                        span.clone(),
                    );
                    state.building.insert(id, (attributes.clone(), build.clone()));
                    let _ = self.events.send(Events::Attributes(attributes.clone()));
                    build
                }
            }
        };

        let service = self.clone();
        self.runtime.spawn_task(async move {
            let answer = match build.await {
                DerivedBuild::Built => Ok(id),
                DerivedBuild::Invalid(err) => {
                    warn!(target: "payload_builder", %err, %id, %parent_hash, "Derived attributes make the payload invalid");
                    Err(PayloadBuilderError::invalid_attributes(err))
                }
                DerivedBuild::Failed(err) => {
                    warn!(target: "payload_builder", %err, %id, %parent_hash, "Failed to build derived payload, building it in the wrapped payload service");
                    let resources = leases
                        .into_iter()
                        .fold(PayloadBuilderResources::default(), PayloadBuilderResources::with_lease);
                    let input = BuildNewPayload { attributes, parent_hash, resources };
                    // After a node-local failure, an error of the wrapped service must not reject
                    // the attributes either.
                    service
                        .build_in_wrapped_service(input, &span)
                        .await
                        .map_err(PayloadBuilderError::other)
                }
            };
            let _ = tx.send(answer);
        });
    }

    /// Starts building `id` on a blocking task. The returned build records its result.
    fn spawn_build(
        self: &Arc<Self>,
        parent_hash: B256,
        attributes: OpPayloadAttrs,
        id: PayloadId,
        resources: PayloadBuilderResources,
        span: Span,
    ) -> SharedBuild {
        let start = Instant::now();
        let built = attributes.clone();
        let service = self.clone();
        let build = self.runtime.spawn_blocking(move || {
            let _entered = span.enter();
            service.builder.build(parent_hash, attributes, id, resources)
        });
        let service = self.clone();
        async move {
            let result = build.await;
            service.metrics.build_duration.record(start.elapsed());
            service.finish(id, built, result)
        }
        .boxed()
        .shared()
    }

    fn state(&self) -> MutexGuard<'_, State<T::BuiltPayload>> {
        self.state.lock().unwrap_or_else(|e| e.into_inner())
    }

    /// Returns the derived payload built for `id`.
    fn built(&self, id: PayloadId) -> Option<T::BuiltPayload> {
        let state = self.state();
        state.built.iter().find(|built| built.id == id).map(|built| built.payload.clone())
    }

    /// Returns the derived payload built for `id`. Its first resolution emits
    /// [`Events::BuiltPayload`], as resolving a payload job does in reth's payload service.
    fn resolve(&self, id: PayloadId) -> Option<T::BuiltPayload> {
        let mut state = self.state();
        let built = state.built.iter_mut().find(|built| built.id == id)?;
        if !std::mem::replace(&mut built.resolved, true) {
            let _ = self.events.send(Events::BuiltPayload(built.payload.clone()));
        }
        Some(built.payload.clone())
    }

    /// Records the result of building `id` for `attributes`. A panic in the build counts as a
    /// node-local failure.
    fn finish(
        &self,
        id: PayloadId,
        attributes: OpPayloadAttrs,
        result: Result<Result<T::BuiltPayload, PayloadBuilderError>, tokio::task::JoinError>,
    ) -> DerivedBuild {
        let mut state = self.state();
        if state.building.get(&id).is_some_and(|(building, _)| *building == attributes) {
            state.building.remove(&id);
        }
        let err = match result {
            Ok(Ok(payload)) => {
                self.metrics.built.increment(1);
                debug!(target: "payload_builder", %id, "Built derived payload");
                state.built.retain(|built| built.id != id);
                if state.built.len() == BUILT_PAYLOADS {
                    state.built.pop_front();
                }
                state.built.push_back(Built { id, attributes, payload, resolved: false });
                return DerivedBuild::Built;
            }
            Ok(Err(err)) => err,
            Err(panic) => PayloadBuilderError::other(panic),
        };
        if is_invalid_derived_payload_error(&err) {
            self.metrics.invalid.increment(1);
            DerivedBuild::Invalid(Arc::new(err))
        } else {
            self.metrics.failed.increment(1);
            DerivedBuild::Failed(Arc::new(err))
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::OpEngineTypes;
    use alloy_primitives::{Bytes, U256};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;
    use reth_evm::execute::{BlockExecutionError, BlockValidationError};
    use reth_node_api::PayloadAttributes;
    use reth_optimism_payload_builder::OpBuiltPayload;
    use reth_optimism_primitives::OpBlock;
    use reth_payload_builder::PayloadBuilderLease;
    use reth_primitives_traits::SealedBlock;
    use reth_provider::ProviderError;
    use revm::context::result::InvalidTransaction;
    use std::{
        sync::atomic::{AtomicUsize, Ordering},
        time::Duration,
    };

    type Outcome = fn(PayloadId) -> Result<OpBuiltPayload, PayloadBuilderError>;

    /// Counts its builds, and holds the first one until the test releases it.
    struct TestBuilder {
        builds: Arc<AtomicUsize>,
        gate: Mutex<Option<std::sync::mpsc::Receiver<()>>>,
        outcome: Outcome,
    }

    impl DerivedPayloadBuilder<OpBuiltPayload> for TestBuilder {
        fn build(
            &self,
            _parent_hash: B256,
            _attributes: OpPayloadAttrs,
            id: PayloadId,
            _resources: PayloadBuilderResources,
        ) -> Result<OpBuiltPayload, PayloadBuilderError> {
            self.builds.fetch_add(1, Ordering::SeqCst);
            if let Some(gate) = self.gate.lock().unwrap().take() {
                gate.recv().unwrap();
            }
            (self.outcome)(id)
        }
    }

    fn payload(id: PayloadId) -> OpBuiltPayload {
        OpBuiltPayload::new(
            id,
            Arc::new(SealedBlock::seal_slow(OpBlock::default())),
            U256::ZERO,
            None,
        )
    }

    fn built(id: PayloadId) -> Result<OpBuiltPayload, PayloadBuilderError> {
        Ok(payload(id))
    }

    /// Derived attributes with a non-deposit transaction, told apart by their gas limit.
    fn derived(gas_limit: u64) -> OpPayloadAttrs {
        OpPayloadAttrs(OpPayloadAttributes {
            transactions: Some(vec![Bytes::from_static(&[0x02, 0x01])]),
            no_tx_pool: Some(true),
            gas_limit: Some(gas_limit),
            ..Default::default()
        })
    }

    /// A parent the wrapped service fails to build on, as reth's does for a missing header.
    const UNKNOWN_PARENT: B256 = B256::repeat_byte(0xff);
    /// A parent on which the wrapped service rejects the attributes.
    const REJECTED_PARENT: B256 = B256::repeat_byte(0xfe);
    /// A parent on which the wrapped service drops the request.
    const DROPPED_PARENT: B256 = B256::repeat_byte(0xfd);

    struct Fixture {
        service: PayloadBuilderHandle<OpEngineTypes>,
        builds: Arc<AtomicUsize>,
        release: std::sync::mpsc::Sender<()>,
        /// Builds the wrapped service started.
        wrapped: Arc<Mutex<Vec<PayloadId>>>,
        /// Commands the wrapped service received, except subscriptions, in order.
        wrapped_commands: Arc<Mutex<Vec<&'static str>>>,
        /// Leases the wrapped service holds for its builds.
        wrapped_leases: Arc<Mutex<Vec<PayloadBuilderLease>>>,
        wrapped_events: broadcast::Sender<Events<OpEngineTypes>>,
    }

    /// Spawns the service in front of a stand-in for the wrapped service, which starts every
    /// build it is asked for, serves those, and broadcasts `wrapped_events`.
    async fn fixture(gated: bool, outcome: Outcome) -> Fixture {
        let (to_wrapped, mut commands) = mpsc::unbounded_channel();
        let wrapped = Arc::new(Mutex::new(Vec::new()));
        let wrapped_leases = Arc::new(Mutex::new(Vec::new()));
        let wrapped_commands = Arc::new(Mutex::new(Vec::new()));
        let (wrapped_events, _) = broadcast::channel(16);
        let (jobs, leases, subscriptions, log) = (
            wrapped.clone(),
            wrapped_leases.clone(),
            wrapped_events.clone(),
            wrapped_commands.clone(),
        );
        tokio::spawn(async move {
            while let Some(command) = commands.recv().await {
                let known = |id| jobs.lock().unwrap().contains(&id);
                let name = match &command {
                    PayloadServiceCommand::BuildNewPayload(..) => Some("BuildNewPayload"),
                    PayloadServiceCommand::BestPayload(..) => Some("BestPayload"),
                    PayloadServiceCommand::PayloadTimestamp(..) => Some("PayloadTimestamp"),
                    PayloadServiceCommand::Resolve(..) => Some("Resolve"),
                    PayloadServiceCommand::Subscribe(_) => None,
                };
                log.lock().unwrap().extend(name);
                match command {
                    PayloadServiceCommand::BuildNewPayload(input, _, tx)
                        if input.parent_hash == UNKNOWN_PARENT =>
                    {
                        let _ =
                            tx.send(Err(PayloadBuilderError::MissingParentHeader(UNKNOWN_PARENT)));
                    }
                    PayloadServiceCommand::BuildNewPayload(input, _, tx)
                        if input.parent_hash == REJECTED_PARENT =>
                    {
                        let _ = tx.send(Err(PayloadBuilderError::invalid_attributes(
                            PayloadBuilderError::MissingPayload,
                        )));
                    }
                    PayloadServiceCommand::BuildNewPayload(input, _, tx)
                        if input.parent_hash == DROPPED_PARENT =>
                    {
                        drop(tx);
                    }
                    PayloadServiceCommand::BuildNewPayload(mut input, _, tx) => {
                        let id = input.payload_id();
                        jobs.lock().unwrap().push(id);
                        leases.lock().unwrap().extend(input.resources.take_leases());
                        let _ = tx.send(Ok(id));
                    }
                    PayloadServiceCommand::BestPayload(id, tx) => {
                        let _ = tx.send(known(id).then(|| built(id)));
                    }
                    PayloadServiceCommand::PayloadTimestamp(id, tx) => {
                        let _ = tx.send(known(id).then_some(Ok(0)));
                    }
                    PayloadServiceCommand::Resolve(id, _, tx) => {
                        let _ = tx.send(known(id).then(|| {
                            Box::pin(std::future::ready(built(id))) as PayloadFuture<OpBuiltPayload>
                        }));
                    }
                    PayloadServiceCommand::Subscribe(tx) => {
                        let _ = tx.send(subscriptions.subscribe());
                    }
                }
            }
        });

        let builds = Arc::new(AtomicUsize::new(0));
        let (release, gate) = std::sync::mpsc::channel();
        let builder = TestBuilder {
            builds: builds.clone(),
            gate: Mutex::new(gated.then_some(gate)),
            outcome,
        };
        let service = DerivedPayloadService::spawn(
            PayloadBuilderHandle::new(to_wrapped),
            builder,
            Runtime::test(),
        )
        .await
        .unwrap();
        Fixture {
            service,
            builds,
            release,
            wrapped,
            wrapped_commands,
            wrapped_leases,
            wrapped_events,
        }
    }

    fn request(
        service: &PayloadBuilderHandle<OpEngineTypes>,
        attributes: OpPayloadAttrs,
    ) -> oneshot::Receiver<Result<PayloadId, PayloadBuilderError>> {
        service.send_new_payload(BuildNewPayload {
            attributes,
            parent_hash: B256::ZERO,
            resources: Default::default(),
        })
    }

    async fn until(condition: impl Fn() -> bool) {
        tokio::time::timeout(Duration::from_secs(10), async {
            while !condition() {
                tokio::time::sleep(Duration::from_millis(1)).await;
            }
        })
        .await
        .unwrap();
    }

    #[tokio::test]
    async fn forwarded_commands_keep_their_order() {
        let f = fixture(false, built).await;
        let unknown = PayloadId::new([7; 8]);
        let mut timestamp = std::pin::pin!(f.service.payload_timestamp(unknown));
        assert!(futures::poll!(&mut timestamp).is_pending());
        let mut best = std::pin::pin!(f.service.best_payload(unknown));
        assert!(futures::poll!(&mut best).is_pending());
        let resolved = f.service.resolve_kind(unknown, PayloadKind::Earliest);
        let sequenced = OpPayloadAttrs(OpPayloadAttributes { no_tx_pool: None, ..derived(1).0 });
        request(&f.service, sequenced).await.unwrap().unwrap();

        assert!(timestamp.await.is_none());
        assert!(best.await.is_none());
        assert!(resolved.await.is_none());
        assert_eq!(
            *f.wrapped_commands.lock().unwrap(),
            ["PayloadTimestamp", "BestPayload", "Resolve", "BuildNewPayload"]
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn derived_payload_is_built_once_and_served_like_a_job() {
        let f = fixture(true, built).await;
        let mut events = f.service.subscribe().await.unwrap().into_built_payload_stream();
        let attributes = derived(1);

        let first = request(&f.service, attributes.clone());
        until(|| f.builds.load(Ordering::SeqCst) == 1).await;
        // The service keeps answering while the build runs.
        let sequenced = OpPayloadAttrs(OpPayloadAttributes { no_tx_pool: None, ..derived(2).0 });
        let sequenced_id = request(&f.service, sequenced).await.unwrap().unwrap();
        assert_eq!(*f.wrapped.lock().unwrap(), [sequenced_id]);
        let retry = request(&f.service, attributes.clone());
        // Commands are handled in order, so the retry has joined the build once this is answered.
        assert!(f.service.payload_timestamp(PayloadId::new([0; 8])).await.is_none());
        f.release.send(()).unwrap();

        let id = first.await.unwrap().unwrap();
        assert_eq!(retry.await.unwrap().unwrap(), id);
        assert_eq!(request(&f.service, attributes).await.unwrap().unwrap(), id);
        assert_eq!(f.builds.load(Ordering::SeqCst), 1);
        assert_eq!(
            *f.wrapped.lock().unwrap(),
            [sequenced_id],
            "the wrapped service never builds it"
        );

        assert_eq!(f.service.payload_timestamp(id).await.unwrap().unwrap(), 0);
        assert!(f.service.best_payload(id).await.unwrap().is_ok());
        let resolved = f.service.resolve_kind(id, PayloadKind::Earliest).await.unwrap().unwrap();
        let event =
            tokio::time::timeout(Duration::from_secs(10), futures::StreamExt::next(&mut events))
                .await
                .unwrap()
                .unwrap();
        assert_eq!(event.block().hash(), resolved.block().hash());
        assert!(f.service.resolve_kind(id, PayloadKind::Earliest).await.unwrap().is_ok());
        assert!(
            tokio::time::timeout(Duration::from_millis(100), futures::StreamExt::next(&mut events))
                .await
                .is_err(),
            "only the first resolution emits the payload"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn invalid_transactions_fail_the_request() {
        let invalid_tx: Outcome = |_| {
            Err(BlockExecutionError::Validation(BlockValidationError::InvalidTx {
                hash: B256::ZERO,
                error: Box::new(InvalidTransaction::NonceTooHigh { tx: 5, state: 0 }),
            })
            .into())
        };
        let f = fixture(false, invalid_tx).await;

        let err = request(&f.service, derived(1)).await.unwrap().unwrap_err();
        assert!(matches!(err, PayloadBuilderError::InvalidAttributes(_)), "got {err:?}");
        assert!(err.to_string().contains("nonce 5 too high"), "got {err}");
        assert!(f.wrapped.lock().unwrap().is_empty());
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn node_local_failures_go_to_the_wrapped_service() {
        let missing_state: Outcome =
            |_| Err(ProviderError::StateForHashNotFound(B256::ZERO).into());
        let panics: Outcome = |_| panic!("build panicked");

        for outcome in [missing_state, panics] {
            let f = fixture(false, outcome).await;
            let id = request(&f.service, derived(1)).await.unwrap().unwrap();
            assert_eq!(*f.wrapped.lock().unwrap(), [id]);
            assert!(f.service.resolve_kind(id, PayloadKind::Earliest).await.unwrap().is_ok());
        }
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn node_local_failures_are_never_invalid_attributes() {
        let missing_state: Outcome =
            |_| Err(ProviderError::StateForHashNotFound(B256::ZERO).into());
        let f = fixture(false, missing_state).await;

        // The wrapped service fails to start the build, too.
        for parent_hash in [UNKNOWN_PARENT, REJECTED_PARENT, DROPPED_PARENT] {
            let err = f
                .service
                .send_new_payload(BuildNewPayload {
                    attributes: derived(1),
                    parent_hash,
                    resources: Default::default(),
                })
                .await
                .unwrap()
                .unwrap_err();
            assert!(
                !matches!(err, PayloadBuilderError::InvalidAttributes(_)),
                "{parent_hash}: {err:?}"
            );
        }
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn engine_leases_are_held_until_the_answer() {
        let lease = Arc::new(());
        let request = |service: &PayloadBuilderHandle<OpEngineTypes>| {
            service.send_new_payload(BuildNewPayload {
                attributes: derived(1),
                parent_hash: B256::ZERO,
                resources: PayloadBuilderResources::default()
                    .with_lease(PayloadBuilderLease::new(lease.clone())),
            })
        };

        let f = fixture(true, built).await;
        let answer = request(&f.service);
        until(|| f.builds.load(Ordering::SeqCst) == 1).await;
        assert_eq!(Arc::strong_count(&lease), 2, "held while the payload is built");
        f.release.send(()).unwrap();
        answer.await.unwrap().unwrap();
        until(|| Arc::strong_count(&lease) == 1).await;

        let missing_state: Outcome =
            |_| Err(ProviderError::StateForHashNotFound(B256::ZERO).into());
        let f = fixture(false, missing_state).await;
        request(&f.service).await.unwrap().unwrap();
        assert_eq!(f.wrapped_leases.lock().unwrap().len(), 1, "the fallback build holds them");
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn wrapped_service_answers_for_its_own_payloads() {
        let f = fixture(false, built).await;
        let mut events = f.service.subscribe().await.unwrap().into_stream();

        let deposits_only = OpPayloadAttrs(OpPayloadAttributes {
            transactions: Some(vec![Bytes::from_static(&[DEPOSIT_TX_TYPE_ID, 0x01])]),
            ..derived(1).0
        });
        let id = request(&f.service, deposits_only).await.unwrap().unwrap();
        assert_eq!(f.builds.load(Ordering::SeqCst), 0);
        assert_eq!(*f.wrapped.lock().unwrap(), [id]);
        assert!(f.service.resolve_kind(id, PayloadKind::Earliest).await.unwrap().is_ok());
        let unknown = PayloadId::new([7; 8]);
        assert!(f.service.resolve_kind(unknown, PayloadKind::Earliest).await.is_none());
        assert!(f.service.payload_timestamp(unknown).await.is_none());

        f.wrapped_events.send(Events::BuiltPayload(payload(id))).unwrap();
        let event =
            tokio::time::timeout(Duration::from_secs(10), futures::StreamExt::next(&mut events))
                .await
                .unwrap()
                .unwrap()
                .unwrap();
        assert!(matches!(event, Events::BuiltPayload(_)));
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn dropped_request_does_not_cancel_its_build() {
        let f = fixture(true, built).await;
        let attributes = derived(1);

        drop(request(&f.service, attributes.clone()));
        until(|| f.builds.load(Ordering::SeqCst) == 1).await;
        f.release.send(()).unwrap();
        let id = attributes.payload_id(&B256::ZERO);
        tokio::time::timeout(Duration::from_secs(10), async {
            while f.service.payload_timestamp(id).await.is_none() {
                tokio::time::sleep(Duration::from_millis(1)).await;
            }
        })
        .await
        .unwrap();

        assert_eq!(request(&f.service, attributes).await.unwrap().unwrap(), id);
        assert_eq!(f.builds.load(Ordering::SeqCst), 1);
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn keeps_the_most_recent_payloads() {
        let f = fixture(false, built).await;
        let mut ids = Vec::new();
        for gas_limit in 0..=BUILT_PAYLOADS as u64 {
            ids.push(request(&f.service, derived(gas_limit)).await.unwrap().unwrap());
        }

        assert!(f.service.payload_timestamp(ids[0]).await.is_none());
        assert!(f.service.payload_timestamp(ids[BUILT_PAYLOADS]).await.is_some());
    }

    #[test]
    fn builds_derived_attributes_with_a_non_deposit_transaction() {
        let attributes = |no_tx_pool, transactions: &[&'static [u8]]| {
            OpPayloadAttrs(OpPayloadAttributes {
                transactions: Some(transactions.iter().map(|tx| Bytes::from_static(tx)).collect()),
                no_tx_pool,
                ..Default::default()
            })
        };
        let deposit: &[u8] = &[DEPOSIT_TX_TYPE_ID, 0x01];
        let eip1559: &[u8] = &[0x02, 0x01];
        let legacy: &[u8] = &[0xf8, 0x01];

        assert!(builds_at_forkchoice_updated(&attributes(Some(true), &[deposit, eip1559])));
        assert!(builds_at_forkchoice_updated(&attributes(Some(true), &[deposit, legacy])));
        assert!(builds_at_forkchoice_updated(&attributes(Some(true), &[&[]])));
        assert_eq!(
            builds_at_forkchoice_updated(&attributes(Some(true), &[deposit, deposit])),
            BUILDS_DEPOSIT_ONLY_ATTRIBUTES
        );
        assert!(!builds_at_forkchoice_updated(&attributes(Some(false), &[deposit, eip1559])));
        assert!(!builds_at_forkchoice_updated(&attributes(None, &[deposit, eip1559])));
    }
}
