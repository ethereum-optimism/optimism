//! Contains an online derivation pipeline.

use crate::{AlloyChainProvider, AlloyL2ChainProvider, OnlineBeaconClient, OnlineBlobProvider};
use async_trait::async_trait;
use core::fmt::Debug;
use kona_derive::{
    DerivationPipeline, EthereumDataSource, IndexedAttributesQueueStage, L2ChainProvider,
    OriginProvider, Pipeline, PipelineBuilder, PipelineErrorKind, PipelineResult,
    PolledAttributesQueueStage, ResetSignal, Signal, SignalReceiver, StatefulAttributesBuilder,
    StepResult,
};
use kona_genesis::{L1ChainConfig, RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use std::sync::Arc;

/// An online polled derivation pipeline.
type OnlinePolledDerivationPipeline = DerivationPipeline<
    PolledAttributesQueueStage<
        OnlineDataProvider,
        AlloyChainProvider,
        AlloyL2ChainProvider,
        OnlineAttributesBuilder,
    >,
    AlloyL2ChainProvider,
>;

/// An online managed derivation pipeline.
type OnlineManagedDerivationPipeline = DerivationPipeline<
    IndexedAttributesQueueStage<
        OnlineDataProvider,
        AlloyChainProvider,
        AlloyL2ChainProvider,
        OnlineAttributesBuilder,
    >,
    AlloyL2ChainProvider,
>;

/// An RPC-backed Ethereum data source.
type OnlineDataProvider =
    EthereumDataSource<AlloyChainProvider, OnlineBlobProvider<OnlineBeaconClient>>;

/// An RPC-backed payload attributes builder for the `AttributesQueue` stage of the derivation
/// pipeline.
type OnlineAttributesBuilder = StatefulAttributesBuilder<AlloyChainProvider, AlloyL2ChainProvider>;

/// An online derivation pipeline.
#[derive(Debug)]
pub enum OnlinePipeline {
    /// An online derivation pipeline that uses a polled traversal stage.
    Polled(OnlinePolledDerivationPipeline),
    /// An online derivation pipeline that uses a managed traversal stage.
    Managed(OnlineManagedDerivationPipeline),
}

impl OnlinePipeline {
    /// Constructs a new polled derivation pipeline that is initialized.
    pub async fn new(
        cfg: Arc<RollupConfig>,
        l1_cfg: Arc<L1ChainConfig>,
        l2_safe_head: L2BlockInfo,
        l1_origin: BlockInfo,
        blob_provider: OnlineBlobProvider<OnlineBeaconClient>,
        chain_provider: AlloyChainProvider,
        mut l2_chain_provider: AlloyL2ChainProvider,
    ) -> PipelineResult<Self> {
        let mut pipeline = Self::new_polled(
            cfg.clone(),
            l1_cfg.clone(),
            blob_provider,
            chain_provider,
            l2_chain_provider.clone(),
        );

        // Reset the pipeline to populate the initial L1/L2 cursor and system configuration in L1
        // Traversal.
        pipeline
            .signal(
                ResetSignal {
                    l2_safe_head,
                    l1_origin,
                    system_config: l2_chain_provider
                        .system_config_by_number(l2_safe_head.block_info.number, cfg.clone())
                        .await
                        .ok(),
                }
                .signal(),
            )
            .await?;

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
    ) -> Self {
        let attributes = StatefulAttributesBuilder::new(
            cfg.clone(),
            l1_cfg,
            l2_chain_provider.clone(),
            chain_provider.clone(),
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
    ) -> Self {
        let attributes = StatefulAttributesBuilder::new(
            cfg.clone(),
            l1_cfg,
            l2_chain_provider.clone(),
            chain_provider.clone(),
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

#[async_trait]
impl SignalReceiver for OnlinePipeline {
    /// Receives a signal from the driver.
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match self {
            Self::Polled(pipeline) => pipeline.signal(signal).await,
            Self::Managed(pipeline) => pipeline.signal(signal).await,
        }
    }
}

impl OriginProvider for OnlinePipeline {
    /// Returns the optional L1 [BlockInfo] origin.
    fn origin(&self) -> Option<BlockInfo> {
        match self {
            Self::Polled(pipeline) => pipeline.origin(),
            Self::Managed(pipeline) => pipeline.origin(),
        }
    }
}

impl Iterator for OnlinePipeline {
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        match self {
            Self::Polled(pipeline) => pipeline.next(),
            Self::Managed(pipeline) => pipeline.next(),
        }
    }
}

#[async_trait]
impl Pipeline for OnlinePipeline {
    /// Peeks at the next [OpAttributesWithParent] from the pipeline.
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

    /// Returns the [SystemConfig] by L2 number.
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
