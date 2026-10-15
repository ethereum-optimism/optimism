//! Builds derived (`noTxPool`) payloads inside `engine_forkchoiceUpdated`.

use alloy_rpc_types_engine::PayloadId;
use futures::{
    FutureExt,
    future::{BoxFuture, Shared},
};
use metrics::{Counter, Histogram};
use op_alloy_consensus::DEPOSIT_TX_TYPE_ID;
use reth_basic_payload_builder::{BuildArguments, HeaderForPayload, PayloadBuilder, PayloadConfig};
use reth_metrics::Metrics;
use reth_node_api::{PayloadBuilderError, PayloadTypes};
use reth_optimism_payload_builder::{OpPayloadAttrs, is_invalid_derived_payload_error};
use reth_payload_builder::PayloadBuilderHandle;
use reth_primitives_traits::SealedHeader;
use reth_tasks::Runtime;
use std::{
    collections::{HashMap, VecDeque},
    fmt,
    sync::{Arc, Mutex, MutexGuard},
    time::Instant,
};
use tokio::sync::oneshot;

/// Number of built derived payloads kept for `engine_getPayload`.
const BUILT_PAYLOADS: usize = 4;

/// Whether `engine_forkchoiceUpdated` builds derived attributes that carry only deposits, too.
///
/// Deposits cannot make a payload invalid, so the payload service can build these.
const BUILDS_DEPOSIT_ONLY_ATTRIBUTES: bool = false;

/// Returns whether `engine_forkchoiceUpdated` builds the payload of `attributes` itself, instead
/// of leaving it to the payload service.
///
/// These are derived (`noTxPool`) attributes, except those that carry only deposits (see
/// `BUILDS_DEPOSIT_ONLY_ATTRIBUTES`). Any other transaction can make the payload invalid, which
/// the engine must report in its `engine_forkchoiceUpdated` response.
pub fn builds_at_forkchoice_updated(attributes: &OpPayloadAttrs) -> bool {
    attributes.no_tx_pool == Some(true) &&
        (BUILDS_DEPOSIT_ONLY_ATTRIBUTES || has_non_deposit_transaction(attributes))
}

fn has_non_deposit_transaction(attributes: &OpPayloadAttrs) -> bool {
    attributes.transactions.iter().flatten().any(|tx| tx.first() != Some(&DEPOSIT_TX_TYPE_ID))
}

/// Builds the payload of derived attributes on top of a parent block.
pub trait DerivedPayloadBuilder<T: PayloadTypes>: Send + Sync + 'static {
    /// Builds the payload `id` for `attributes` on top of `parent`.
    fn build_derived_payload(
        &self,
        parent: SealedHeader<HeaderForPayload<T::BuiltPayload>>,
        attributes: T::PayloadAttributes,
        id: PayloadId,
    ) -> Result<T::BuiltPayload, PayloadBuilderError>;
}

impl<T, B> DerivedPayloadBuilder<T> for B
where
    T: PayloadTypes,
    B: PayloadBuilder<Attributes = T::PayloadAttributes, BuiltPayload = T::BuiltPayload> + 'static,
{
    fn build_derived_payload(
        &self,
        parent: SealedHeader<HeaderForPayload<T::BuiltPayload>>,
        attributes: T::PayloadAttributes,
        id: PayloadId,
    ) -> Result<T::BuiltPayload, PayloadBuilderError> {
        let config = PayloadConfig::new(Arc::new(parent), attributes, id);
        let args =
            BuildArguments::new(Default::default(), None, None, config, Default::default(), None);
        self.try_build(args)?.into_payload().ok_or(PayloadBuilderError::MissingPayload)
    }
}

/// How building derived attributes ended.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DerivedBuild {
    /// The payload was built and is kept for `engine_getPayload`.
    Built,
    /// The attributes' transactions make the payload invalid.
    Invalid(Arc<str>),
    /// The build failed for a reason local to this node, so it says nothing about the
    /// attributes.
    Failed(Arc<str>),
}

/// Metrics of payloads built at `engine_forkchoiceUpdated`.
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_rpc.engine.derived_payloads")]
struct DerivedPayloadMetrics {
    /// Duration of a derived payload build, including the state root.
    build_duration: Histogram,
    /// Derived payloads built.
    built: Counter,
    /// Derived attributes rejected because their transactions make the payload invalid.
    invalid: Counter,
    /// Derived builds that failed for a node-local reason.
    failed: Counter,
}

type SharedBuild = Shared<BoxFuture<'static, DerivedBuild>>;

/// Builds derived payloads and keeps the most recent ones for `engine_getPayload`.
///
/// Builds run detached from the caller, and at most one runs per payload: a request for a
/// payload that is being built waits for that build, and one for a built payload returns at once.
/// A rollup node that retries after a timeout therefore never starts a second build. Payloads are
/// matched by id and attributes, so attributes whose ids collide are still built.
pub struct DerivedPayloads<T: PayloadTypes>(Arc<Inner<T>>);

struct Inner<T: PayloadTypes> {
    builder: Box<dyn DerivedPayloadBuilder<T>>,
    payload_service: PayloadBuilderHandle<T>,
    runtime: Runtime,
    state: Mutex<State<T>>,
    metrics: DerivedPayloadMetrics,
}

/// A derived payload kept for `engine_getPayload`.
struct Built<T: PayloadTypes> {
    id: PayloadId,
    attributes: T::PayloadAttributes,
    payload: T::BuiltPayload,
}

struct State<T: PayloadTypes> {
    building: HashMap<PayloadId, (T::PayloadAttributes, SharedBuild)>,
    built: VecDeque<Built<T>>,
}

impl<T> DerivedPayloads<T>
where
    T: PayloadTypes<PayloadAttributes: PartialEq>,
{
    /// Creates a new instance that builds with `builder` on `runtime`.
    ///
    /// A payload served by `engine_getPayload` is published to `payload_service`, whose
    /// subscribers include the engine: it inserts the executed block, so `engine_newPayload` of
    /// that block does not execute it again.
    pub fn new(
        builder: impl DerivedPayloadBuilder<T>,
        payload_service: PayloadBuilderHandle<T>,
        runtime: Runtime,
    ) -> Self {
        Self(Arc::new(Inner {
            builder: Box::new(builder),
            payload_service,
            runtime,
            state: Mutex::new(State {
                building: HashMap::new(),
                built: VecDeque::with_capacity(BUILT_PAYLOADS),
            }),
            metrics: DerivedPayloadMetrics::default(),
        }))
    }

    /// Builds the payload `id` for `attributes` on top of `parent`, unless it is built or being
    /// built already.
    pub async fn build(
        &self,
        parent: SealedHeader<HeaderForPayload<T::BuiltPayload>>,
        attributes: T::PayloadAttributes,
        id: PayloadId,
    ) -> DerivedBuild {
        let build = {
            let mut state = self.0.state();
            if state.built.iter().any(|built| built.id == id && built.attributes == attributes) {
                return DerivedBuild::Built;
            }
            match state.building.get(&id) {
                Some((building, build)) if *building == attributes => build.clone(),
                _ => {
                    let build = self.spawn_build(parent, attributes.clone(), id);
                    state.building.insert(id, (attributes, build.clone()));
                    build
                }
            }
        };
        build.await
    }

    fn spawn_build(
        &self,
        parent: SealedHeader<HeaderForPayload<T::BuiltPayload>>,
        attributes: T::PayloadAttributes,
        id: PayloadId,
    ) -> SharedBuild {
        let (tx, rx) = oneshot::channel();
        let inner = self.0.clone();
        self.0.runtime.spawn_task(async move {
            let start = Instant::now();
            let builder = inner.clone();
            let built = attributes.clone();
            let result = inner
                .runtime
                .spawn_blocking(move || {
                    builder.builder.build_derived_payload(parent, attributes, id)
                })
                .await;
            inner.metrics.build_duration.record(start.elapsed());
            let _ = tx.send(inner.finish(id, built, result));
        });
        rx.map(|outcome| {
            outcome.unwrap_or_else(|_| DerivedBuild::Failed("derived payload build dropped".into()))
        })
        .boxed()
        .shared()
    }

    /// Returns the derived payload built for `id`.
    pub fn get(&self, id: PayloadId) -> Option<T::BuiltPayload> {
        let state = self.0.state();
        state.built.iter().find(|built| built.id == id).map(|built| built.payload.clone())
    }

    /// Hands `payload` to the subscribers of the payload service, like a resolved payload job.
    pub fn publish(&self, payload: T::BuiltPayload) {
        self.0.payload_service.publish_built_payload(payload);
    }
}

impl<T> Inner<T>
where
    T: PayloadTypes<PayloadAttributes: PartialEq>,
{
    fn state(&self) -> MutexGuard<'_, State<T>> {
        self.state.lock().unwrap_or_else(|e| e.into_inner())
    }

    /// Records the result of building `id` for `attributes`. A panic in the build counts as a
    /// node-local failure.
    fn finish(
        &self,
        id: PayloadId,
        attributes: T::PayloadAttributes,
        result: Result<Result<T::BuiltPayload, PayloadBuilderError>, tokio::task::JoinError>,
    ) -> DerivedBuild {
        let mut state = self.state();
        if state.building.get(&id).is_some_and(|(building, _)| *building == attributes) {
            state.building.remove(&id);
        }
        match result {
            Ok(Ok(payload)) => {
                self.metrics.built.increment(1);
                state.built.retain(|built| built.id != id);
                if state.built.len() == BUILT_PAYLOADS {
                    state.built.pop_front();
                }
                state.built.push_back(Built { id, attributes, payload });
                DerivedBuild::Built
            }
            Ok(Err(err)) if is_invalid_derived_payload_error(&err) => {
                self.metrics.invalid.increment(1);
                DerivedBuild::Invalid(err.to_string().into())
            }
            Ok(Err(err)) => {
                self.metrics.failed.increment(1);
                DerivedBuild::Failed(err.to_string().into())
            }
            Err(err) => {
                self.metrics.failed.increment(1);
                DerivedBuild::Failed(err.to_string().into())
            }
        }
    }
}

impl<T: PayloadTypes> Clone for DerivedPayloads<T> {
    fn clone(&self) -> Self {
        Self(self.0.clone())
    }
}

impl<T: PayloadTypes> fmt::Debug for DerivedPayloads<T> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("DerivedPayloads").finish_non_exhaustive()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use alloy_primitives::{B256, Bytes, U256};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;
    use reth_evm::execute::{BlockExecutionError, BlockValidationError};
    use reth_optimism_payload_builder::{OpBuiltPayload, OpPayloadTypes};
    use reth_optimism_primitives::OpBlock;
    use reth_primitives_traits::SealedBlock;
    use reth_storage_api::errors::ProviderError;
    use revm::context::result::InvalidTransaction;
    use std::{
        sync::{
            atomic::{AtomicUsize, Ordering},
            mpsc,
        },
        time::Duration,
    };

    type Outcome = fn(PayloadId) -> Result<OpBuiltPayload, PayloadBuilderError>;

    /// Counts its builds, and holds the first one until the test releases it.
    struct TestBuilder {
        builds: Arc<AtomicUsize>,
        gate: Mutex<Option<mpsc::Receiver<()>>>,
        outcome: Outcome,
    }

    impl DerivedPayloadBuilder<OpPayloadTypes> for TestBuilder {
        fn build_derived_payload(
            &self,
            _parent: SealedHeader,
            _attributes: OpPayloadAttrs,
            id: PayloadId,
        ) -> Result<OpBuiltPayload, PayloadBuilderError> {
            self.builds.fetch_add(1, Ordering::SeqCst);
            if let Some(gate) = self.gate.lock().unwrap().take() {
                gate.recv().unwrap();
            }
            (self.outcome)(id)
        }
    }

    fn built(id: PayloadId) -> Result<OpBuiltPayload, PayloadBuilderError> {
        Ok(OpBuiltPayload::new(
            id,
            Arc::new(SealedBlock::seal_slow(OpBlock::default())),
            U256::ZERO,
            None,
        ))
    }

    /// Returns the payloads, the number of builds so far, and the sender releasing the first
    /// build, if `gated`.
    fn derived_payloads(
        gated: bool,
        outcome: Outcome,
    ) -> (DerivedPayloads<OpPayloadTypes>, Arc<AtomicUsize>, Option<mpsc::Sender<()>>) {
        let builds = Arc::new(AtomicUsize::new(0));
        let (release, gate) = mpsc::channel();
        let builder = TestBuilder {
            builds: builds.clone(),
            gate: Mutex::new(gated.then_some(gate)),
            outcome,
        };
        let payloads = DerivedPayloads::new(builder, PayloadBuilderHandle::noop(), Runtime::test());
        (payloads, builds, gated.then_some(release))
    }

    async fn build(payloads: &DerivedPayloads<OpPayloadTypes>, id: PayloadId) -> DerivedBuild {
        let attributes = OpPayloadAttrs(Default::default());
        payloads.build(SealedHeader::seal_slow(Header::default()), attributes, id).await
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

    #[tokio::test(flavor = "multi_thread")]
    async fn requests_for_one_payload_share_its_build() {
        let (payloads, builds, release) = derived_payloads(true, built);
        let id = PayloadId::new([1; 8]);

        let first = tokio::spawn({
            let payloads = payloads.clone();
            async move { build(&payloads, id).await }
        });
        until(|| builds.load(Ordering::SeqCst) == 1).await;
        // Polled once while the first build is held, so it joins that build.
        let mut retry = Box::pin(build(&payloads, id));
        assert!(futures::poll!(&mut retry).is_pending());
        release.unwrap().send(()).unwrap();

        assert_eq!(first.await.unwrap(), DerivedBuild::Built);
        assert_eq!(retry.await, DerivedBuild::Built);
        assert_eq!(build(&payloads, id).await, DerivedBuild::Built);
        assert_eq!(builds.load(Ordering::SeqCst), 1);
        assert!(payloads.get(id).is_some());
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn dropped_request_does_not_cancel_its_build() {
        let (payloads, builds, release) = derived_payloads(true, built);
        let id = PayloadId::new([1; 8]);

        tokio::time::timeout(Duration::from_millis(10), build(&payloads, id)).await.unwrap_err();
        release.unwrap().send(()).unwrap();
        until(|| payloads.get(id).is_some()).await;

        assert_eq!(build(&payloads, id).await, DerivedBuild::Built);
        assert_eq!(builds.load(Ordering::SeqCst), 1);
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn only_invalid_transactions_reject_the_attributes() {
        let missing_state: Outcome =
            |_| Err(ProviderError::StateForHashNotFound(B256::ZERO).into());
        let panics: Outcome = |_| panic!("build panicked");
        let invalid_tx: Outcome = |_| {
            Err(BlockExecutionError::Validation(BlockValidationError::InvalidTx {
                hash: B256::ZERO,
                error: Box::new(InvalidTransaction::NonceTooHigh { tx: 5, state: 0 }),
            })
            .into())
        };
        let id = PayloadId::new([1; 8]);

        for outcome in [missing_state, panics] {
            let (payloads, builds, _) = derived_payloads(false, outcome);
            assert!(matches!(build(&payloads, id).await, DerivedBuild::Failed(_)));
            // A failed build is not kept, so the next request builds again.
            assert!(matches!(build(&payloads, id).await, DerivedBuild::Failed(_)));
            assert_eq!(builds.load(Ordering::SeqCst), 2);
            assert!(payloads.get(id).is_none());
        }
        let (payloads, _, _) = derived_payloads(false, invalid_tx);
        assert!(matches!(build(&payloads, id).await, DerivedBuild::Invalid(_)));
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn other_attributes_with_the_same_id_are_built() {
        let (payloads, builds, _) = derived_payloads(false, built);
        let id = PayloadId::new([1; 8]);
        let other =
            OpPayloadAttrs(OpPayloadAttributes { gas_limit: Some(1), ..Default::default() });

        assert_eq!(build(&payloads, id).await, DerivedBuild::Built);
        let parent = SealedHeader::seal_slow(Header::default());
        assert_eq!(payloads.build(parent, other, id).await, DerivedBuild::Built);
        assert_eq!(builds.load(Ordering::SeqCst), 2);
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn keeps_the_most_recent_payloads() {
        let (payloads, _, _) = derived_payloads(false, built);
        for i in 0..=BUILT_PAYLOADS as u8 {
            assert_eq!(build(&payloads, PayloadId::new([i; 8])).await, DerivedBuild::Built);
        }
        assert!(payloads.get(PayloadId::new([0; 8])).is_none());
        assert!(payloads.get(PayloadId::new([BUILT_PAYLOADS as u8; 8])).is_some());
    }

    fn attributes(no_tx_pool: Option<bool>, transactions: &[&[u8]]) -> OpPayloadAttrs {
        OpPayloadAttrs(OpPayloadAttributes {
            transactions: Some(transactions.iter().map(|tx| Bytes::copy_from_slice(tx)).collect()),
            no_tx_pool,
            ..Default::default()
        })
    }

    #[test]
    fn builds_derived_attributes_with_a_non_deposit_transaction() {
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
        assert_eq!(
            builds_at_forkchoice_updated(&attributes(Some(true), &[])),
            BUILDS_DEPOSIT_ONLY_ATTRIBUTES
        );
        assert!(!builds_at_forkchoice_updated(&attributes(Some(false), &[deposit, eip1559])));
        assert!(!builds_at_forkchoice_updated(&attributes(None, &[deposit, eip1559])));
    }
}
