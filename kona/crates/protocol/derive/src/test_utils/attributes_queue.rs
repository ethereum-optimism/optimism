//! Testing utilities for the attributes queue stage.

use crate::{
    errors::{BuilderError, PipelineError, PipelineErrorKind},
    traits::{
        AttributesBuilder, AttributesProvider, OriginAdvancer, OriginProvider, SignalReceiver,
    },
    types::{PipelineResult, Signal},
};
use alloc::{boxed::Box, string::ToString, vec::Vec};
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use kona_protocol::{BlockInfo, L2BlockInfo, SingleBatch};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use thiserror::Error;

/// An error returned by the [`TestAttributesBuilder`].
#[derive(Error, Debug, PartialEq, Eq)]
pub enum TestAttributesBuilderError {}

/// A mock implementation of the [`AttributesBuilder`] for testing.
#[derive(Debug, Default)]
pub struct TestAttributesBuilder {
    /// The attributes to return.
    pub attributes: Vec<Result<OpPayloadAttributes, TestAttributesBuilderError>>,
}

#[async_trait]
impl AttributesBuilder for TestAttributesBuilder {
    /// Prepares the [`OpPayloadAttributes`] for the next payload.
    async fn prepare_payload_attributes(
        &mut self,
        _l2_parent: L2BlockInfo,
        _epoch: BlockNumHash,
    ) -> PipelineResult<OpPayloadAttributes> {
        match self.attributes.pop() {
            Some(Ok(attrs)) => Ok(attrs),
            Some(Err(err)) => {
                Err(PipelineErrorKind::Temporary(BuilderError::Custom(err.to_string()).into()))
            }
            None => Err(PipelineErrorKind::Critical(BuilderError::AttributesUnavailable.into())),
        }
    }
}

/// A mock implementation of the [`AttributesProvider`] stage for testing.
#[derive(Debug, Default)]
pub struct TestAttributesProvider {
    /// The origin of the L1 block.
    origin: Option<BlockInfo>,
    /// A list of batches to return.
    batches: Vec<PipelineResult<SingleBatch>>,
    /// Tracks if the provider has been reset.
    pub reset: bool,
    /// Tracks if the provider has been flushed.
    pub flushed: bool,
}

impl OriginProvider for TestAttributesProvider {
    fn origin(&self) -> Option<BlockInfo> {
        self.origin
    }
}

#[async_trait]
impl OriginAdvancer for TestAttributesProvider {
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        Ok(())
    }
}

#[async_trait]
impl SignalReceiver for TestAttributesProvider {
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match signal {
            Signal::FlushChannel => self.flushed = true,
            Signal::Reset { .. } => self.reset = true,
            _ => {}
        }
        Ok(())
    }
}

#[async_trait]
impl AttributesProvider for TestAttributesProvider {
    async fn next_batch(&mut self, _parent: L2BlockInfo) -> PipelineResult<SingleBatch> {
        self.batches.pop().ok_or(PipelineError::Eof.temp())?
    }

    fn is_last_in_span(&self) -> bool {
        self.batches.is_empty()
    }
}

/// Creates a new [`TestAttributesProvider`] with the given origin and batches.
pub const fn new_test_attributes_provider(
    origin: Option<BlockInfo>,
    batches: Vec<PipelineResult<SingleBatch>>,
) -> TestAttributesProvider {
    TestAttributesProvider { origin, batches, reset: false, flushed: false }
}
