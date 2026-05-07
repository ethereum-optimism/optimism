//! Contains an online derivation pipeline.

use crate::{AlloyChainProvider, AlloyL2ChainProvider, OnlineBeaconClient, OnlineBlobProvider};
use async_trait::async_trait;
use core::fmt::Debug;
use kona_derive::{
    DataAvailabilityProvider, DerivationPipeline, EthereumDataSource, IndexedAttributesQueueStage,
    OriginProvider, Pipeline, PipelineBuilder, PipelineErrorKind, PipelineResult,
    PolledAttributesQueueStage, ResetSignal, Signal, SignalReceiver, StatefulAttributesBuilder,
    StepResult,
};
use kona_genesis::{L1ChainConfig, RollupConfig, SystemConfig};
use kona_interop::DependencySet;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use std::sync::Arc;

/// An online polled derivation pipeline. Generic over the
/// `DataAvailabilityProvider` so downstream binaries can substitute a custom
/// DA source (e.g. one that filters L1 batcher txs by SRA-set membership);
/// defaults to the RPC-backed [`OnlineDataProvider`].
type OnlinePolledDerivationPipeline<DAP = OnlineDataProvider> = DerivationPipeline<
    PolledAttributesQueueStage<DAP, AlloyChainProvider, AlloyL2ChainProvider, OnlineAttributesBuilder>,
    AlloyL2ChainProvider,
>;

/// An online managed derivation pipeline. See [`OnlinePolledDerivationPipeline`]
/// for the rationale behind the `DAP` parameter.
type OnlineManagedDerivationPipeline<DAP = OnlineDataProvider> = DerivationPipeline<
    IndexedAttributesQueueStage<
        DAP,
        AlloyChainProvider,
        AlloyL2ChainProvider,
        OnlineAttributesBuilder,
    >,
    AlloyL2ChainProvider,
>;

/// An RPC-backed Ethereum data source.
pub type OnlineDataProvider =
    EthereumDataSource<AlloyChainProvider, OnlineBlobProvider<OnlineBeaconClient>>;

/// An RPC-backed payload attributes builder for the `AttributesQueue` stage of the derivation
/// pipeline.
type OnlineAttributesBuilder = StatefulAttributesBuilder<AlloyChainProvider, AlloyL2ChainProvider>;

/// An online derivation pipeline.
///
/// Generic over `DAP: DataAvailabilityProvider` so binaries can plug a
/// custom DA source (e.g. PSO Chain's SRA-aware
/// [`PsoEthereumDataSource`](kona_derive::DataAvailabilityProvider)) in
/// place of the stock [`EthereumDataSource`]. Defaults to the
/// RPC-backed [`OnlineDataProvider`] so existing callers compile
/// unchanged.
#[derive(Debug)]
pub enum OnlinePipeline<DAP = OnlineDataProvider>
where
    // The `Send + Sync` bounds satisfy the inner FrameQueue/DerivationPipeline
    // chain, which transitively requires the DAP to be thread-safe so the
    // pipeline itself can be moved across actor boundaries. `Debug` is
    // needed for the `derive(Debug)` on the enum.
    DAP: DataAvailabilityProvider + Debug + Send + Sync,
{
    /// An online derivation pipeline that uses a polled traversal stage.
    Polled(OnlinePolledDerivationPipeline<DAP>),
    /// An online derivation pipeline that uses a managed traversal stage.
    Managed(OnlineManagedDerivationPipeline<DAP>),
}

// =========================================================================
// Default-DAP constructors — preserve the existing OnlinePipeline API so
// stock kona-node-service callers compile unchanged.
// =========================================================================

impl OnlinePipeline<OnlineDataProvider> {
    /// Constructs a new polled derivation pipeline that is initialized.
    ///
    /// `dependency_set` must be `Some` when the rollup config schedules the
    /// Interop hardfork. The inner [`StatefulAttributesBuilder`] constructor
    /// panics otherwise; turning a silent state-divergence bug into a
    /// startup crash.
    #[allow(clippy::too_many_arguments)]
    pub async fn new(
        cfg: Arc<RollupConfig>,
        l1_cfg: Arc<L1ChainConfig>,
        l2_safe_head: L2BlockInfo,
        _l1_origin: BlockInfo,
        blob_provider: OnlineBlobProvider<OnlineBeaconClient>,
        chain_provider: AlloyChainProvider,
        l2_chain_provider: AlloyL2ChainProvider,
        dependency_set: Option<Arc<DependencySet>>,
    ) -> PipelineResult<Self> {
        let mut pipeline = Self::new_polled(
            cfg.clone(),
            l1_cfg.clone(),
            blob_provider,
            chain_provider,
            l2_chain_provider.clone(),
            dependency_set,
        );

        // Reset the pipeline to populate the initial L1/L2 cursor and system configuration in L1
        // Traversal.
        pipeline.signal(Signal::Reset(ResetSignal { l2_safe_head })).await?;

        Ok(pipeline)
    }

    /// Constructs a new polled derivation pipeline that is uninitialized.
    ///
    /// Uses online providers as specified by the arguments.
    ///
    /// Before using the returned pipeline, a [`ResetSignal`] must be sent to
    /// instantiate the pipeline state. [`Self::new`] is a convenience method that
    /// constructs a new online pipeline and sends the reset signal.
    pub fn new_polled(
        cfg: Arc<RollupConfig>,
        l1_cfg: Arc<L1ChainConfig>,
        blob_provider: OnlineBlobProvider<OnlineBeaconClient>,
        chain_provider: AlloyChainProvider,
        l2_chain_provider: AlloyL2ChainProvider,
        dependency_set: Option<Arc<DependencySet>>,
    ) -> Self {
        let attributes = StatefulAttributesBuilder::new(
            cfg.clone(),
            l1_cfg,
            l2_chain_provider.clone(),
            chain_provider.clone(),
            dependency_set,
        );
        let dap = EthereumDataSource::new_from_parts(chain_provider.clone(), blob_provider, &cfg);

        let pipeline = PipelineBuilder::new()
            .rollup_config(cfg)
            .dap_source(dap)
            .l2_chain_provider(l2_chain_provider)
            .chain_provider(chain_provider)
            .builder(attributes)
            .origin(BlockInfo::default())
            .build_polled();

        Self::Polled(pipeline)
    }

    /// Constructs a new indexed derivation pipeline that is uninitialized.
    ///
    /// Uses online providers as specified by the arguments.
    ///
    /// Before using the returned pipeline, a [`ResetSignal`] must be sent to
    /// instantiate the pipeline state. [`Self::new`] is a convenience method that
    /// constructs a new online pipeline and sends the reset signal.
    pub fn new_indexed(
        cfg: Arc<RollupConfig>,
        l1_cfg: Arc<L1ChainConfig>,
        blob_provider: OnlineBlobProvider<OnlineBeaconClient>,
        chain_provider: AlloyChainProvider,
        l2_chain_provider: AlloyL2ChainProvider,
        dependency_set: Option<Arc<DependencySet>>,
    ) -> Self {
        let attributes = StatefulAttributesBuilder::new(
            cfg.clone(),
            l1_cfg,
            l2_chain_provider.clone(),
            chain_provider.clone(),
            dependency_set,
        );
        let dap = EthereumDataSource::new_from_parts(chain_provider.clone(), blob_provider, &cfg);

        let pipeline = PipelineBuilder::new()
            .rollup_config(cfg)
            .dap_source(dap)
            .l2_chain_provider(l2_chain_provider)
            .chain_provider(chain_provider)
            .builder(attributes)
            .origin(BlockInfo::default())
            .build_indexed();

        Self::Managed(pipeline)
    }
}

// =========================================================================
// Custom-DAP constructors — the override seam used by binaries that need a
// non-default DA source (e.g. PSO Chain's SRA-aware data provider).
// =========================================================================

impl<DAP> OnlinePipeline<DAP>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync + Clone + 'static,
{
    /// Constructs a polled derivation pipeline using a caller-supplied
    /// `DataAvailabilityProvider` instead of the default
    /// [`OnlineDataProvider`]. Mirrors [`Self::new_polled`] otherwise.
    pub fn new_polled_with_dap(
        cfg: Arc<RollupConfig>,
        l1_cfg: Arc<L1ChainConfig>,
        chain_provider: AlloyChainProvider,
        l2_chain_provider: AlloyL2ChainProvider,
        dap: DAP,
        dependency_set: Option<Arc<DependencySet>>,
    ) -> Self {
        let attributes = StatefulAttributesBuilder::new(
            cfg.clone(),
            l1_cfg,
            l2_chain_provider.clone(),
            chain_provider.clone(),
            dependency_set,
        );

        let pipeline = PipelineBuilder::new()
            .rollup_config(cfg)
            .dap_source(dap)
            .l2_chain_provider(l2_chain_provider)
            .chain_provider(chain_provider)
            .builder(attributes)
            .origin(BlockInfo::default())
            .build_polled();

        Self::Polled(pipeline)
    }

    /// Constructs an indexed (managed) derivation pipeline using a
    /// caller-supplied `DataAvailabilityProvider`. Mirrors
    /// [`Self::new_indexed`] otherwise.
    pub fn new_indexed_with_dap(
        cfg: Arc<RollupConfig>,
        l1_cfg: Arc<L1ChainConfig>,
        chain_provider: AlloyChainProvider,
        l2_chain_provider: AlloyL2ChainProvider,
        dap: DAP,
        dependency_set: Option<Arc<DependencySet>>,
    ) -> Self {
        let attributes = StatefulAttributesBuilder::new(
            cfg.clone(),
            l1_cfg,
            l2_chain_provider.clone(),
            chain_provider.clone(),
            dependency_set,
        );

        let pipeline = PipelineBuilder::new()
            .rollup_config(cfg)
            .dap_source(dap)
            .l2_chain_provider(l2_chain_provider)
            .chain_provider(chain_provider)
            .builder(attributes)
            .origin(BlockInfo::default())
            .build_indexed();

        Self::Managed(pipeline)
    }
}

#[async_trait]
impl<DAP> SignalReceiver for OnlinePipeline<DAP>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync,
{
    /// Receives a signal from the driver.
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match self {
            Self::Polled(pipeline) => pipeline.signal(signal).await,
            Self::Managed(pipeline) => pipeline.signal(signal).await,
        }
    }
}

impl<DAP> OriginProvider for OnlinePipeline<DAP>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync,
{
    /// Returns the optional L1 [`BlockInfo`] origin.
    fn origin(&self) -> Option<BlockInfo> {
        match self {
            Self::Polled(pipeline) => pipeline.origin(),
            Self::Managed(pipeline) => pipeline.origin(),
        }
    }
}

impl<DAP> Iterator for OnlinePipeline<DAP>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync,
{
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        match self {
            Self::Polled(pipeline) => pipeline.next(),
            Self::Managed(pipeline) => pipeline.next(),
        }
    }
}

#[async_trait]
impl<DAP> Pipeline for OnlinePipeline<DAP>
where
    DAP: DataAvailabilityProvider + Debug + Send + Sync,
{
    /// Peeks at the next [`OpAttributesWithParent`] from the pipeline.
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        match self {
            Self::Polled(pipeline) => pipeline.peek(),
            Self::Managed(pipeline) => pipeline.peek(),
        }
    }

    /// Attempts to progress the pipeline.
    async fn step(&mut self, cursor: L2BlockInfo) -> StepResult {
        match self {
            Self::Polled(pipeline) => pipeline.step(cursor).await,
            Self::Managed(pipeline) => pipeline.step(cursor).await,
        }
    }

    /// Returns the rollup config.
    fn rollup_config(&self) -> &RollupConfig {
        match self {
            Self::Polled(pipeline) => pipeline.rollup_config(),
            Self::Managed(pipeline) => pipeline.rollup_config(),
        }
    }

    /// Returns the [`SystemConfig`] by L2 number.
    async fn system_config_by_number(
        &mut self,
        number: u64,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        match self {
            Self::Polled(pipeline) => pipeline.system_config_by_number(number).await,
            Self::Managed(pipeline) => pipeline.system_config_by_number(number).await,
        }
    }
}
