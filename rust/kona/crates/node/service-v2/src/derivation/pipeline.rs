//! Online derivation pipeline backed by a direct, consumer-owned L1 reader.

use crate::l1::L1Reader;
use async_trait::async_trait;
use kona_derive::{
    DerivationPipeline, EthereumDataSource, OriginProvider, Pipeline, PipelineBuilder,
    PipelineErrorKind, PipelineResult, PolledAttributesQueueStage, Signal, SignalReceiver,
    StatefulAttributesBuilder, StepResult,
};
use kona_genesis::{L1ChainConfig, RollupConfig, SystemConfig};
use kona_interop::DependencySet;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use kona_providers_alloy::AlloyL2ChainProvider;
use std::sync::Arc;

type DirectDataProvider = EthereumDataSource<L1Reader, L1Reader>;
type DirectAttributesBuilder = StatefulAttributesBuilder<L1Reader, AlloyL2ChainProvider>;
type PolledPipeline = DerivationPipeline<
    PolledAttributesQueueStage<
        DirectDataProvider,
        L1Reader,
        AlloyL2ChainProvider,
        DirectAttributesBuilder,
    >,
    AlloyL2ChainProvider,
>;

/// Online polling derivation pipeline.
#[derive(Debug)]
pub struct ServicePipeline(PolledPipeline);

impl ServicePipeline {
    /// Creates an uninitialized polling pipeline.
    pub fn new(
        config: Arc<RollupConfig>,
        l1_config: Arc<L1ChainConfig>,
        l1: L1Reader,
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
        Self(
            PipelineBuilder::new()
                .rollup_config(config)
                .dap_source(data)
                .l2_chain_provider(l2)
                .chain_provider(l1)
                .builder(attributes)
                .origin(BlockInfo::default())
                .build_polled(),
        )
    }
}

#[async_trait]
impl SignalReceiver for ServicePipeline {
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        self.0.signal(signal).await
    }
}

impl OriginProvider for ServicePipeline {
    fn origin(&self) -> Option<BlockInfo> {
        self.0.origin()
    }
}

impl Iterator for ServicePipeline {
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        self.0.next()
    }
}

#[async_trait]
impl Pipeline for ServicePipeline {
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        self.0.peek()
    }

    async fn step(&mut self, cursor: L2BlockInfo) -> StepResult {
        self.0.step(cursor).await
    }

    fn rollup_config(&self) -> &RollupConfig {
        self.0.rollup_config()
    }

    async fn system_config_by_number(
        &mut self,
        number: u64,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        self.0.system_config_by_number(number).await
    }
}
