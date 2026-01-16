//! Test Utilities for the [`DerivationPipeline`]
//! as well as its stages and providers.

use crate::{
    BatchProvider, PipelineResult,
    test_utils::{TestChainProvider, TestL2ChainProvider},
};
use alloc::{boxed::Box, sync::Arc};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};

// Re-export these types used internally to the test pipeline.
use crate::{
    AttributesQueue, BatchStream, ChannelProvider, ChannelReader, DerivationPipeline, FrameQueue,
    L1Retrieval, NextAttributes, OriginAdvancer, OriginProvider, PipelineBuilder, PipelineError,
    PollingTraversal, Signal, SignalReceiver,
    test_utils::{TestAttributesBuilder, TestDAP},
};

/// A fully custom [`NextAttributes`].
#[derive(Default, Debug, Clone)]
pub struct TestNextAttributes {
    /// The next [`OpAttributesWithParent`] to return.
    pub next_attributes: Option<OpAttributesWithParent>,
}

#[async_trait::async_trait]
impl SignalReceiver for TestNextAttributes {
    /// Resets the derivation stage to its initial state.
    async fn signal(&mut self, _: Signal) -> PipelineResult<()> {
        Ok(())
    }
}

#[async_trait::async_trait]
impl OriginProvider for TestNextAttributes {
    /// Returns the current origin.
    fn origin(&self) -> Option<BlockInfo> {
        Some(BlockInfo::default())
    }
}

#[async_trait::async_trait]
impl OriginAdvancer for TestNextAttributes {
    /// Advances the origin to the given block.
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        Ok(())
    }
}

#[async_trait::async_trait]
impl NextAttributes for TestNextAttributes {
    /// Returns the next valid [`OpAttributesWithParent`].
    async fn next_attributes(&mut self, _: L2BlockInfo) -> PipelineResult<OpAttributesWithParent> {
        self.next_attributes.take().ok_or(PipelineError::Eof.temp())
    }
}

/// A [`PollingTraversal`] using test providers and sources.
pub type TestPollingTraversal = PollingTraversal<TestChainProvider>;

/// An [`L1Retrieval`] stage using test providers and sources.
pub type TestL1Retrieval = L1Retrieval<TestDAP, TestPollingTraversal>;

/// A [`FrameQueue`] using test providers and sources.
pub type TestFrameQueue = FrameQueue<TestL1Retrieval>;

/// A [`ChannelProvider`] using test providers and sources.
pub type TestChannelProvider = ChannelProvider<TestFrameQueue>;

/// A [`ChannelReader`] using test providers and sources.
pub type TestChannelReader = ChannelReader<TestChannelProvider>;

/// A [`BatchStream`] using test providers and sources.
pub type TestBatchStream = BatchStream<TestChannelReader, TestL2ChainProvider>;

/// A [`BatchProvider`] using test providers and sources.
pub type TestBatchProvider = BatchProvider<TestBatchStream, TestL2ChainProvider>;

/// An [`AttributesQueue`] using test providers and sources.
pub type TestAttributesQueue = AttributesQueue<TestBatchProvider, TestAttributesBuilder>;

/// A [`DerivationPipeline`] using test providers and sources.
pub type TestPipeline = DerivationPipeline<TestAttributesQueue, TestL2ChainProvider>;

/// Constructs a [`DerivationPipeline`] using test providers and sources.
pub fn new_test_pipeline() -> TestPipeline {
    PipelineBuilder::new()
        .rollup_config(Arc::new(RollupConfig::default()))
        .origin(BlockInfo::default())
        .dap_source(TestDAP::default())
        .builder(TestAttributesBuilder::default())
        .chain_provider(TestChainProvider::default())
        .l2_chain_provider(TestL2ChainProvider::default())
        .build_polled()
}
