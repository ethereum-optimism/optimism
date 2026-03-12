//! Test derivation pipeline backed by mock providers.
//!
//! Constructs a real [`DerivationPipeline`] using [`MockL1Chain`] as the
//! chain provider and blob provider, and [`TestL2Provider`] as the L2 provider.

use crate::mock_l1::MockL1Chain;
use crate::node_actor::test_l2_provider::TestL2Provider;
use async_trait::async_trait;
use kona_derive::{
    DerivationPipeline, EthereumDataSource, OriginProvider, Pipeline,
    PolledAttributesQueueStage, PipelineBuilder, PipelineErrorKind, Signal,
    SignalReceiver, StatefulAttributesBuilder, StepResult,
};
use kona_genesis::{L1ChainConfig, RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use std::sync::Arc;

/// Type alias for the data availability provider using mock L1.
type TestDataProvider = EthereumDataSource<MockL1Chain, MockL1Chain>;

/// Type alias for the attributes builder.
type TestAttributesBuilder = StatefulAttributesBuilder<MockL1Chain, TestL2Provider>;

/// Type alias for the inner polled pipeline.
type InnerPipeline =
    DerivationPipeline<PolledAttributesQueueStage<TestDataProvider, MockL1Chain, TestL2Provider, TestAttributesBuilder>, TestL2Provider>;

/// A test derivation pipeline wrapping a real [`DerivationPipeline`].
///
/// This enum wrapper allows implementing [`Pipeline`] + [`SignalReceiver`]
/// while using concrete mock types.
#[derive(Debug)]
pub enum TestPipeline {
    /// A polled derivation pipeline.
    Polled(InnerPipeline),
}

impl TestPipeline {
    /// Constructs a new [`TestPipeline`] from the given components.
    pub fn new(
        l1_chain: MockL1Chain,
        l2_provider: TestL2Provider,
        rollup_config: Arc<RollupConfig>,
        l1_config: Arc<L1ChainConfig>,
        origin: BlockInfo,
    ) -> Self {
        let dap_source =
            EthereumDataSource::new_from_parts(l1_chain.clone(), l1_chain.clone(), &rollup_config);

        let attributes_builder = StatefulAttributesBuilder::new(
            rollup_config.clone(),
            l1_config,
            l2_provider.clone(),
            l1_chain.clone(),
        );

        let pipeline = PipelineBuilder::new()
            .rollup_config(rollup_config)
            .origin(origin)
            .dap_source(dap_source)
            .l2_chain_provider(l2_provider)
            .chain_provider(l1_chain)
            .builder(attributes_builder)
            .build_polled();

        Self::Polled(pipeline)
    }
}

impl OriginProvider for TestPipeline {
    fn origin(&self) -> Option<BlockInfo> {
        match self {
            Self::Polled(p) => p.origin(),
        }
    }
}

impl Iterator for TestPipeline {
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        match self {
            Self::Polled(p) => p.next(),
        }
    }
}

#[async_trait]
impl Pipeline for TestPipeline {
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        match self {
            Self::Polled(p) => p.peek(),
        }
    }

    async fn step(&mut self, cursor: L2BlockInfo) -> StepResult {
        match self {
            Self::Polled(p) => p.step(cursor).await,
        }
    }

    fn rollup_config(&self) -> &RollupConfig {
        match self {
            Self::Polled(p) => p.rollup_config(),
        }
    }

    async fn system_config_by_number(
        &mut self,
        number: u64,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        match self {
            Self::Polled(p) => p.system_config_by_number(number).await,
        }
    }
}

#[async_trait]
impl SignalReceiver for TestPipeline {
    async fn signal(
        &mut self,
        signal: Signal,
    ) -> Result<(), PipelineErrorKind> {
        match self {
            Self::Polled(p) => p.signal(signal).await,
        }
    }
}
