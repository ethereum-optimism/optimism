//! OP payload jobs.
//!
//! A derived (`noTxPool`) payload that carries a `0x7D` is built while its job is created. If its
//! transactions make it invalid, creating the job fails, `engine_forkchoiceUpdated` returns
//! `INVALID_PAYLOAD_ATTRIBUTES`, and the rollup node can replace the block with a deposits-only
//! one. Building the payload there, instead of checking it before a job builds it, executes the
//! block once. Every other payload is left to the wrapped generator.
//!
//! The build holds the payload-builder service until it finishes, so other payload commands wait
//! for it; the rollup node is waiting on the same `engine_forkchoiceUpdated` anyway.

use futures_util::future::{self, Either, Ready};
use op_alloy_consensus::POST_EXEC_TX_TYPE_ID;
use reth_basic_payload_builder::{BuildArguments, HeaderForPayload, PayloadBuilder, PayloadConfig};
use reth_chain_state::CanonStateNotification;
use reth_node_api::PayloadAttributes;
use reth_optimism_payload_builder::{OpPayloadAttrs, is_invalid_derived_payload_error};
use reth_payload_builder::{
    BuildNewPayload, KeepPayloadJobAlive, PayloadBuilderError, PayloadId, PayloadJob,
    PayloadJobGenerator, PayloadKind,
};
use reth_primitives_traits::NodePrimitives;
use reth_provider::HeaderProvider;
use reth_tracing::tracing::{error, warn};
use std::{
    future::Future,
    panic::{self, AssertUnwindSafe},
    pin::Pin,
    sync::Arc,
    task::{Context, Poll},
    time::Duration,
};
use tokio::time::Sleep;

/// Wraps a [`PayloadJobGenerator`] and builds derived payloads that carry a `0x7D` itself; see
/// the [module docs](self).
#[derive(Debug)]
pub struct OpPayloadJobGenerator<Inner, Client, Builder> {
    inner: Inner,
    client: Client,
    builder: Builder,
    deadline: Duration,
}

impl<Inner, Client, Builder> OpPayloadJobGenerator<Inner, Client, Builder> {
    /// Builds derived payloads that carry a `0x7D` with `builder`, on parents read from `client`,
    /// and leaves every other payload to `inner`.
    ///
    /// `builder` must build the same blocks as `inner`'s jobs. A payload it builds ends its job
    /// after `deadline` if it is never resolved; unlike a basic job's deadline, that doesn't add
    /// the time until the payload's timestamp, which is in the past for derived blocks. `inner`
    /// never sees those jobs.
    ///
    /// # Blocking
    ///
    /// [`PayloadJobGenerator::new_payload_job`] runs the build, so the
    /// [`PayloadBuilderService`](reth_payload_builder::PayloadBuilderService) must run on a
    /// dedicated thread, as [`OpBasicPayloadServiceBuilder`](crate::OpBasicPayloadServiceBuilder)
    /// runs it, not on an async worker.
    pub const fn new(inner: Inner, client: Client, builder: Builder, deadline: Duration) -> Self {
        Self { inner, client, builder, deadline }
    }
}

impl<Inner, Client, Builder> OpPayloadJobGenerator<Inner, Client, Builder>
where
    Client: HeaderProvider<Header = HeaderForPayload<Builder::BuiltPayload>>,
    Builder: PayloadBuilder<Attributes = OpPayloadAttrs>,
{
    /// Builds a derived payload. Returns `None` if the build failed for a node-local reason, so the
    /// payload is left to an inner job and the rollup node can retry the same attributes.
    fn build_derived(
        &self,
        input: &BuildNewPayload<OpPayloadAttrs>,
        id: PayloadId,
    ) -> Result<Option<Builder::BuiltPayload>, PayloadBuilderError> {
        // Left to the inner generator, as before this wrapper; the engine has just resolved this
        // head, so the parent is not expected to be missing.
        let Ok(Some(parent)) = self.client.sealed_header_by_hash(input.parent_hash) else {
            return Ok(None);
        };
        let args = BuildArguments {
            cached_reads: Default::default(),
            execution_cache: None,
            state_root_handle: None,
            config: PayloadConfig::new(Arc::new(parent), input.attributes.clone(), id),
            cancel: Default::default(),
            best_payload: None,
        };
        // The payload-builder service runs on a critical thread, where a panic stops the node. A
        // background build's panic only fails that build, so treat it the same way. Asserting
        // unwind safety is sound: the closure only borrows the builder, whose clones already run
        // background builds under `catch_unwind`.
        let Ok(result) = panic::catch_unwind(AssertUnwindSafe(|| self.builder.try_build(args)))
        else {
            error!(target: "payload_builder", %id, "Derived payload build panicked, leaving it to a payload job");
            return Ok(None);
        };
        match result {
            Ok(outcome) => Ok(outcome.into_payload()),
            Err(err) if is_invalid_derived_payload_error(&err) => {
                warn!(target: "payload_builder", %err, %id, "Rejecting derived payload attributes with invalid transactions");
                Err(err)
            }
            Err(err) => {
                warn!(target: "payload_builder", %err, %id, "Could not build derived payload, leaving it to a payload job");
                Ok(None)
            }
        }
    }
}

/// UPSTREAM-MIRROR(delegate): reth@rev:0fbe428 `reth_payload_builder::PayloadJobGenerator`
///
/// Forwards everything to the inner generator except new jobs for derived payloads that carry a
/// `0x7D`. Re-check newly defaulted methods, which this wrapper would not forward.
impl<Inner, Client, Builder> PayloadJobGenerator for OpPayloadJobGenerator<Inner, Client, Builder>
where
    Inner: PayloadJobGenerator<
        Job: PayloadJob<
            PayloadAttributes = OpPayloadAttrs,
            BuiltPayload = Builder::BuiltPayload,
        > + Unpin,
    >,
    Client: HeaderProvider<Header = HeaderForPayload<Builder::BuiltPayload>>,
    Builder: PayloadBuilder<Attributes = OpPayloadAttrs, BuiltPayload: 'static>,
{
    type Job = OpPayloadJob<Inner::Job>;

    fn new_payload_job(
        &self,
        input: BuildNewPayload<OpPayloadAttrs>,
        id: PayloadId,
    ) -> Result<Self::Job, PayloadBuilderError> {
        if is_derived_with_post_exec(&input.attributes) &&
            let Some(payload) = self.build_derived(&input, id)?
        {
            return Ok(OpPayloadJob::Frozen(Box::new(FrozenPayloadJob::new(
                input.attributes,
                payload,
                self.deadline,
            ))));
        }
        self.inner.new_payload_job(input, id).map(OpPayloadJob::Inner)
    }

    fn on_new_state<N: NodePrimitives>(&mut self, new_state: CanonStateNotification<N>) {
        self.inner.on_new_state(new_state);
    }
}

/// Whether `attributes` are derived and carry a `0x7D`.
fn is_derived_with_post_exec(attributes: &OpPayloadAttrs) -> bool {
    attributes.0.no_tx_pool == Some(true) &&
        attributes
            .0
            .transactions
            .as_ref()
            .is_some_and(|txs| txs.iter().any(|tx| tx.first() == Some(&POST_EXEC_TX_TYPE_ID)))
}

/// A payload job created by [`OpPayloadJobGenerator`].
#[derive(Debug)]
pub enum OpPayloadJob<Job: PayloadJob> {
    /// A job created by the wrapped generator.
    Inner(Job),
    /// A derived payload, built when the job was created.
    Frozen(Box<FrozenPayloadJob<Job::BuiltPayload>>),
}

/// A payload job whose payload is already built.
#[derive(Debug)]
pub struct FrozenPayloadJob<Payload> {
    attributes: OpPayloadAttrs,
    payload: Payload,
    /// Ends the job if the payload is never resolved.
    deadline: Pin<Box<Sleep>>,
}

impl<Payload> FrozenPayloadJob<Payload> {
    fn new(attributes: OpPayloadAttrs, payload: Payload, deadline: Duration) -> Self {
        Self { attributes, payload, deadline: Box::pin(tokio::time::sleep(deadline)) }
    }
}

/// UPSTREAM-MIRROR(delegate): reth@rev:0fbe428 `reth_payload_builder::PayloadJob`
///
/// Forwards every method to an inner job. Re-check newly defaulted methods, which this wrapper
/// would not forward, and give a frozen job its own answer for each.
impl<Job> PayloadJob for OpPayloadJob<Job>
where
    Job: PayloadJob<PayloadAttributes = OpPayloadAttrs, BuiltPayload: 'static> + Unpin,
{
    type PayloadAttributes = OpPayloadAttrs;
    type ResolvePayloadFuture =
        Either<Job::ResolvePayloadFuture, Ready<Result<Job::BuiltPayload, PayloadBuilderError>>>;
    type BuiltPayload = Job::BuiltPayload;

    fn best_payload(&self) -> Result<Self::BuiltPayload, PayloadBuilderError> {
        match self {
            Self::Inner(job) => job.best_payload(),
            Self::Frozen(job) => Ok(job.payload.clone()),
        }
    }

    fn payload_attributes(&self) -> Result<Self::PayloadAttributes, PayloadBuilderError> {
        match self {
            Self::Inner(job) => job.payload_attributes(),
            Self::Frozen(job) => Ok(job.attributes.clone()),
        }
    }

    fn payload_timestamp(&self) -> Result<u64, PayloadBuilderError> {
        match self {
            Self::Inner(job) => job.payload_timestamp(),
            Self::Frozen(job) => Ok(job.attributes.timestamp()),
        }
    }

    fn resolve_kind(
        &mut self,
        kind: PayloadKind,
    ) -> (Self::ResolvePayloadFuture, KeepPayloadJobAlive) {
        match self {
            Self::Inner(job) => {
                let (payload, keep_alive) = job.resolve_kind(kind);
                (Either::Left(payload), keep_alive)
            }
            Self::Frozen(job) => {
                (Either::Right(future::ready(Ok(job.payload.clone()))), KeepPayloadJobAlive::No)
            }
        }
    }

    fn resolve(&mut self) -> (Self::ResolvePayloadFuture, KeepPayloadJobAlive) {
        if let Self::Inner(job) = self {
            let (payload, keep_alive) = job.resolve();
            return (Either::Left(payload), keep_alive);
        }
        self.resolve_kind(PayloadKind::Earliest)
    }
}

impl<Job> Future for OpPayloadJob<Job>
where
    Job: PayloadJob + Unpin,
{
    type Output = Result<(), PayloadBuilderError>;

    fn poll(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Self::Output> {
        match self.get_mut() {
            Self::Inner(job) => Pin::new(job).poll(cx),
            Self::Frozen(job) => job.deadline.as_mut().poll(cx).map(Ok),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use alloy_primitives::{B256, Bytes, U256};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;
    use reth_basic_payload_builder::{BasicPayloadJob, BuildOutcome};
    use reth_optimism_payload_builder::OpBuiltPayload;
    use reth_optimism_primitives::OpBlock;
    use reth_primitives_traits::SealedBlock;
    use reth_provider::{ProviderError, test_utils::MockEthProvider};
    use std::sync::atomic::{AtomicUsize, Ordering};

    const PARENT: B256 = B256::repeat_byte(1);

    type BuildResult = Result<BuildOutcome<OpBuiltPayload>, PayloadBuilderError>;

    #[derive(Clone)]
    struct StubBuilder(Arc<dyn Fn() -> BuildResult + Send + Sync>);

    impl PayloadBuilder for StubBuilder {
        type Attributes = OpPayloadAttrs;
        type BuiltPayload = OpBuiltPayload;

        fn try_build(
            &self,
            _args: BuildArguments<Self::Attributes, Self::BuiltPayload>,
        ) -> BuildResult {
            (self.0)()
        }

        fn build_empty_payload(
            &self,
            _config: PayloadConfig<Self::Attributes, Header>,
        ) -> Result<Self::BuiltPayload, PayloadBuilderError> {
            unreachable!("derived payloads are never built empty")
        }
    }

    /// Counts the payloads left to it and fails each with [`PayloadBuilderError::MissingPayload`].
    #[derive(Default)]
    struct StubInner(AtomicUsize);

    impl PayloadJobGenerator for StubInner {
        type Job = BasicPayloadJob<StubBuilder>;

        fn new_payload_job(
            &self,
            _input: BuildNewPayload<OpPayloadAttrs>,
            _id: PayloadId,
        ) -> Result<Self::Job, PayloadBuilderError> {
            self.0.fetch_add(1, Ordering::Relaxed);
            Err(PayloadBuilderError::MissingPayload)
        }
    }

    type TestGenerator = OpPayloadJobGenerator<StubInner, MockEthProvider, StubBuilder>;

    /// Returns a generator whose builder runs `build`, and a count of its builds.
    fn setup(
        has_parent: bool,
        build: impl Fn() -> BuildResult + Send + Sync + 'static,
    ) -> (TestGenerator, Arc<AtomicUsize>) {
        let client = MockEthProvider::default();
        if has_parent {
            client.add_header(PARENT, Header::default());
        }
        let builds = Arc::new(AtomicUsize::new(0));
        let counter = Arc::clone(&builds);
        let builder = StubBuilder(Arc::new(move || {
            counter.fetch_add(1, Ordering::Relaxed);
            build()
        }));
        (OpPayloadJobGenerator::new(StubInner::default(), client, builder, Duration::ZERO), builds)
    }

    fn payload() -> OpBuiltPayload {
        OpBuiltPayload::new(
            PayloadId::default(),
            Arc::new(SealedBlock::seal_slow(OpBlock::default())),
            U256::ZERO,
            None,
        )
    }

    fn attributes(no_tx_pool: Option<bool>, transactions: &[&[u8]]) -> OpPayloadAttrs {
        OpPayloadAttrs(OpPayloadAttributes {
            no_tx_pool,
            transactions: Some(transactions.iter().map(|tx| Bytes::copy_from_slice(tx)).collect()),
            ..Default::default()
        })
    }

    fn derived() -> BuildNewPayload<OpPayloadAttrs> {
        BuildNewPayload {
            attributes: attributes(Some(true), &[&[POST_EXEC_TX_TYPE_ID]]),
            parent_hash: PARENT,
            resources: Default::default(),
        }
    }

    fn invalid() -> BuildResult {
        Err(PayloadBuilderError::other(alloy_rlp::Error::UnexpectedLength))
    }

    fn node_local() -> BuildResult {
        Err(ProviderError::StateForHashNotFound(B256::ZERO).into())
    }

    #[test]
    fn only_derived_attributes_with_post_exec_are_built_at_job_creation() {
        let post_exec: &[u8] = &[POST_EXEC_TX_TYPE_ID, 0xc0];
        let eip1559: &[u8] = &[0x02, 0xc0];
        assert!(is_derived_with_post_exec(&attributes(Some(true), &[eip1559, post_exec])));
        assert!(!is_derived_with_post_exec(&attributes(Some(false), &[post_exec])));
        assert!(!is_derived_with_post_exec(&attributes(None, &[post_exec])));
        assert!(!is_derived_with_post_exec(&attributes(Some(true), &[eip1559, &[]])));
    }

    fn build_derived(
        build: impl Fn() -> BuildResult + Send + Sync + 'static,
    ) -> Result<Option<OpBuiltPayload>, PayloadBuilderError> {
        setup(true, build).0.build_derived(&derived(), PayloadId::default())
    }

    #[test]
    fn derived_build_rejects_only_invalid_transactions() {
        let built = build_derived(|| Ok(BuildOutcome::Freeze(payload())));
        assert!(matches!(built, Ok(Some(_))), "{built:?}");
        let rejected = build_derived(invalid);
        assert!(rejected.is_err(), "{rejected:?}");

        // Node-local failures must stay retryable, so an inner job takes over.
        let state = build_derived(node_local);
        assert!(matches!(state, Ok(None)), "{state:?}");
        let panicked = build_derived(|| panic!("builder bug"));
        assert!(matches!(panicked, Ok(None)), "{panicked:?}");

        let (generator, builds) = setup(false, || Ok(BuildOutcome::Freeze(payload())));
        let no_parent = generator.build_derived(&derived(), PayloadId::default());
        assert!(matches!(no_parent, Ok(None)), "{no_parent:?}");
        assert_eq!(builds.load(Ordering::Relaxed), 0);
    }

    // `new_payload_job` builds a frozen job's payload, so the inner generator must not build it
    // again, and must still get everything else.
    #[tokio::test]
    async fn derived_post_exec_payloads_are_built_once_at_job_creation() {
        let id = PayloadId::default();
        let inner_jobs = |generator: &TestGenerator| generator.inner.0.load(Ordering::Relaxed);

        let (generator, builds) = setup(true, || Ok(BuildOutcome::Freeze(payload())));
        let job = generator.new_payload_job(derived(), id);
        assert!(matches!(job, Ok(OpPayloadJob::Frozen(_))));
        assert_eq!((builds.load(Ordering::Relaxed), inner_jobs(&generator)), (1, 0));

        let (generator, _) = setup(true, invalid);
        let job = generator.new_payload_job(derived(), id);
        assert!(matches!(job, Err(PayloadBuilderError::Other(_))));
        assert_eq!(inner_jobs(&generator), 0);

        let (generator, _) = setup(true, node_local);
        let job = generator.new_payload_job(derived(), id);
        assert!(matches!(job, Err(PayloadBuilderError::MissingPayload)));
        assert_eq!(inner_jobs(&generator), 1);

        let (generator, builds) = setup(true, || Ok(BuildOutcome::Freeze(payload())));
        let sequenced = BuildNewPayload {
            attributes: attributes(Some(false), &[&[POST_EXEC_TX_TYPE_ID]]),
            ..derived()
        };
        let job = generator.new_payload_job(sequenced, id);
        assert!(matches!(job, Err(PayloadBuilderError::MissingPayload)));
        assert_eq!((builds.load(Ordering::Relaxed), inner_jobs(&generator)), (0, 1));
    }

    #[tokio::test]
    async fn frozen_job_resolves_at_once_and_ends_at_its_deadline() {
        let payload = payload();
        let mut job =
            OpPayloadJob::<BasicPayloadJob<StubBuilder>>::Frozen(Box::new(FrozenPayloadJob::new(
                attributes(Some(true), &[]),
                payload.clone(),
                Duration::from_millis(10),
            )));

        let (resolved, keep_alive) = job.resolve_kind(PayloadKind::WaitForPending);
        assert_eq!(keep_alive, KeepPayloadJobAlive::No);
        assert_eq!(resolved.await.unwrap().block().hash(), payload.block().hash());

        tokio::time::timeout(Duration::from_secs(5), job).await.unwrap().unwrap();
    }
}
