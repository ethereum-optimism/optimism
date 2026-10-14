//! Contains an online derivation pipeline.

use crate::{
    AlloyChainProvider, BufferedAlloyL2ChainProvider, OnlineBeaconClient, OnlineBlobProvider,
};
use alloy_primitives::B256;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_derive::{
    ChainProvider, DerivationPipeline, EthereumDataSource, OriginProvider, Pipeline,
    PipelineBuilder, PipelineErrorKind, PipelineResult, PolledAttributesQueueStage, ResetSignal,
    Signal, SignalReceiver, StatefulAttributesBuilder, StepResult,
};
use kona_genesis::{L1ChainConfig, RollupConfig, SystemConfig};
use kona_interop::DependencySet;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use std::sync::Arc;

/// An online polled derivation pipeline that reads L1 from `L1`.
type OnlinePolledDerivationPipeline<L1> = DerivationPipeline<
    PolledAttributesQueueStage<
        OnlineDataProvider<L1>,
        L1,
        BufferedAlloyL2ChainProvider,
        OnlineAttributesBuilder<L1>,
    >,
    BufferedAlloyL2ChainProvider,
>;

/// An RPC-backed Ethereum data source.
type OnlineDataProvider<L1> = EthereumDataSource<L1, OnlineBlobProvider<OnlineBeaconClient>>;

/// An RPC-backed payload attributes builder for the `AttributesQueue` stage of the derivation
/// pipeline.
type OnlineAttributesBuilder<L1> = StatefulAttributesBuilder<L1, BufferedAlloyL2ChainProvider>;

/// An online derivation pipeline, which reads L1 from `L1`.
#[derive(Debug)]
pub struct OnlinePipeline<L1 = AlloyChainProvider>(OnlinePolledDerivationPipeline<L1>)
where
    L1: ChainProvider + Clone + Debug + Send + Sync + 'static;

impl<L1> OnlinePipeline<L1>
where
    L1: ChainProvider + Clone + Debug + Send + Sync + 'static,
{
    /// Constructs a new derivation pipeline that is initialized.
    ///
    /// `dependency_set` must be `Some` when the rollup config schedules the
    /// Lagoon hardfork. The inner [`StatefulAttributesBuilder`] constructor
    /// panics otherwise; turning a silent state-divergence bug into a
    /// startup crash.
    #[allow(clippy::too_many_arguments)]
    pub async fn new(
        cfg: Arc<RollupConfig>,
        l1_cfg: Arc<L1ChainConfig>,
        l2_safe_head: L2BlockInfo,
        _l1_origin: BlockInfo,
        blob_provider: OnlineBlobProvider<OnlineBeaconClient>,
        chain_provider: L1,
        l2_chain_provider: BufferedAlloyL2ChainProvider,
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

    /// Constructs a new derivation pipeline that is uninitialized.
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
        chain_provider: L1,
        l2_chain_provider: BufferedAlloyL2ChainProvider,
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

        Self(pipeline)
    }
}

#[async_trait]
impl<L1> SignalReceiver for OnlinePipeline<L1>
where
    L1: ChainProvider + Clone + Debug + Send + Sync + 'static,
{
    /// Receives a signal from the driver.
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        self.0.signal(signal).await
    }
}

impl<L1> OriginProvider for OnlinePipeline<L1>
where
    L1: ChainProvider + Clone + Debug + Send + Sync + 'static,
{
    /// Returns the optional L1 [`BlockInfo`] origin.
    fn origin(&self) -> Option<BlockInfo> {
        self.0.origin()
    }
}

impl<L1> Iterator for OnlinePipeline<L1>
where
    L1: ChainProvider + Clone + Debug + Send + Sync + 'static,
{
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        self.0.next()
    }
}

#[async_trait]
impl<L1> Pipeline for OnlinePipeline<L1>
where
    L1: ChainProvider + Clone + Debug + Send + Sync + 'static,
{
    /// Peeks at the next [`OpAttributesWithParent`] from the pipeline.
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        self.0.peek()
    }

    /// Attempts to progress the pipeline.
    async fn step(&mut self, cursor: L2BlockInfo) -> StepResult {
        self.0.step(cursor).await
    }

    /// Returns the rollup config.
    fn rollup_config(&self) -> &RollupConfig {
        self.0.rollup_config()
    }

    /// Returns the [`SystemConfig`] by L2 number.
    async fn system_config_by_l2_hash(
        &mut self,
        hash: B256,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        self.0.system_config_by_l2_hash(hash).await
    }
}
