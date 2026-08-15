//! Derivation pipeline wired through the shared L1 service.

use crate::l1::L1Client;
use async_trait::async_trait;
use kona_derive::{
    DerivationPipeline, EthereumDataSource, IndexedAttributesQueueStage, OriginProvider, Pipeline,
    PipelineBuilder, PipelineErrorKind, PipelineResult, PolledAttributesQueueStage, Signal,
    SignalReceiver, StatefulAttributesBuilder, StepResult,
};
use kona_genesis::{L1ChainConfig, RollupConfig, SystemConfig};
use kona_interop::DependencySet;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use kona_providers_alloy::AlloyL2ChainProvider;
use std::sync::Arc;

type SharedDataProvider = EthereumDataSource<L1Client, L1Client>;
type SharedAttributesBuilder = StatefulAttributesBuilder<L1Client, AlloyL2ChainProvider>;
type PolledPipeline = DerivationPipeline<
    PolledAttributesQueueStage<
        SharedDataProvider,
        L1Client,
        AlloyL2ChainProvider,
        SharedAttributesBuilder,
    >,
    AlloyL2ChainProvider,
>;
type IndexedPipeline = DerivationPipeline<
    IndexedAttributesQueueStage<
        SharedDataProvider,
        L1Client,
        AlloyL2ChainProvider,
        SharedAttributesBuilder,
    >,
    AlloyL2ChainProvider,
>;

/// Online derivation pipeline whose L1 and blob reads are serialized by [`crate::l1::L1Service`].
#[derive(Debug)]
pub enum ServicePipeline {
    /// Polling traversal mode.
    Polled(PolledPipeline),
    /// Indexed traversal mode.
    Indexed(IndexedPipeline),
}

impl ServicePipeline {
    /// Creates an uninitialized polling pipeline.
    pub fn new_polled(
        config: Arc<RollupConfig>,
        l1_config: Arc<L1ChainConfig>,
        l1: L1Client,
        l2: AlloyL2ChainProvider,
        dependency_set: Option<Arc<DependencySet>>,
    ) -> Self {
        let attributes = StatefulAttributesBuilder::new(
            config.clone(),
            l1_config,
            l2.clone(),
            l1.clone(),
            dependency_set,
        );
        let data = EthereumDataSource::new_from_parts(l1.clone(), l1.clone(), &config);
        let pipeline = PipelineBuilder::new()
            .rollup_config(config)
            .dap_source(data)
            .l2_chain_provider(l2)
            .chain_provider(l1)
            .builder(attributes)
            .origin(BlockInfo::default())
            .build_polled();
        Self::Polled(pipeline)
    }

    /// Creates an uninitialized indexed pipeline.
    pub fn new_indexed(
        config: Arc<RollupConfig>,
        l1_config: Arc<L1ChainConfig>,
        l1: L1Client,
        l2: AlloyL2ChainProvider,
        dependency_set: Option<Arc<DependencySet>>,
    ) -> Self {
        let attributes = StatefulAttributesBuilder::new(
            config.clone(),
            l1_config,
            l2.clone(),
            l1.clone(),
            dependency_set,
        );
        let data = EthereumDataSource::new_from_parts(l1.clone(), l1.clone(), &config);
        let pipeline = PipelineBuilder::new()
            .rollup_config(config)
            .dap_source(data)
            .l2_chain_provider(l2)
            .chain_provider(l1)
            .builder(attributes)
            .origin(BlockInfo::default())
            .build_indexed();
        Self::Indexed(pipeline)
    }
}

#[async_trait]
impl SignalReceiver for ServicePipeline {
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match self {
            Self::Polled(pipeline) => pipeline.signal(signal).await,
            Self::Indexed(pipeline) => pipeline.signal(signal).await,
        }
    }
}

impl OriginProvider for ServicePipeline {
    fn origin(&self) -> Option<BlockInfo> {
        match self {
            Self::Polled(pipeline) => pipeline.origin(),
            Self::Indexed(pipeline) => pipeline.origin(),
        }
    }
}

impl Iterator for ServicePipeline {
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        match self {
            Self::Polled(pipeline) => pipeline.next(),
            Self::Indexed(pipeline) => pipeline.next(),
        }
    }
}

#[async_trait]
impl Pipeline for ServicePipeline {
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        match self {
            Self::Polled(pipeline) => pipeline.peek(),
            Self::Indexed(pipeline) => pipeline.peek(),
        }
    }

    async fn step(&mut self, cursor: L2BlockInfo) -> StepResult {
        match self {
            Self::Polled(pipeline) => pipeline.step(cursor).await,
            Self::Indexed(pipeline) => pipeline.step(cursor).await,
        }
    }

    fn rollup_config(&self) -> &RollupConfig {
        match self {
            Self::Polled(pipeline) => pipeline.rollup_config(),
            Self::Indexed(pipeline) => pipeline.rollup_config(),
        }
    }

    async fn system_config_by_number(
        &mut self,
        number: u64,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        match self {
            Self::Polled(pipeline) => pipeline.system_config_by_number(number).await,
            Self::Indexed(pipeline) => pipeline.system_config_by_number(number).await,
        }
    }
}
