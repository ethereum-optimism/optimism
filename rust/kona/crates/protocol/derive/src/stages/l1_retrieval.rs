//! Contains the [L1Retrieval] stage of the derivation pipeline.

use crate::{
    ActivationSignal, DataAvailabilityProvider, FrameQueueProvider, OriginAdvancer, OriginProvider,
    PipelineError, PipelineErrorKind, PipelineResult, ResetSignal, Signal, SignalReceiver,
};
use alloc::boxed::Box;
use alloy_primitives::Address;
use async_trait::async_trait;
use kona_protocol::BlockInfo;

/// Provides L1 blocks for the [`L1Retrieval`] stage.
/// This is the previous stage in the pipeline.
#[async_trait]
pub trait L1RetrievalProvider {
    /// Returns the next L1 [`BlockInfo`] in the [`PollingTraversal`] stage, if the stage is not
    /// complete. This function can only be called once while the stage is in progress, and will
    /// return [`None`] on subsequent calls unless the stage is reset or complete. If the stage
    /// is complete and the [`BlockInfo`] has been consumed, an [PipelineError::Eof] error is
    /// returned.
    ///
    /// [`PollingTraversal`]: crate::PollingTraversal
    async fn next_l1_block(&mut self) -> PipelineResult<Option<BlockInfo>>;

    /// Returns the batcher [`Address`] from the [kona_genesis::SystemConfig].
    fn batcher_addr(&self) -> Address;
}

/// The [`L1Retrieval`] stage of the derivation pipeline.
///
/// For each L1 [`BlockInfo`] pulled from the [`PollingTraversal`] stage, [`L1Retrieval`] fetches
/// the associated data from a specified [`DataAvailabilityProvider`].
///
/// [`PollingTraversal`]: crate::PollingTraversal
#[derive(Debug)]
pub struct L1Retrieval<DAP, P>
where
    DAP: DataAvailabilityProvider,
    P: L1RetrievalProvider + OriginAdvancer + OriginProvider + SignalReceiver,
{
    /// The previous stage in the pipeline.
    pub prev: P,
    /// The data availability provider to use for the L1 retrieval stage.
    pub provider: DAP,
    /// The current block ref.
    pub next: Option<BlockInfo>,
}

impl<DAP, P> L1Retrieval<DAP, P>
where
    DAP: DataAvailabilityProvider,
    P: L1RetrievalProvider + OriginAdvancer + OriginProvider + SignalReceiver,
{
    /// Creates a new [`L1Retrieval`] stage with the previous [`PollingTraversal`] stage and given
    /// [`DataAvailabilityProvider`].
    ///
    /// [`PollingTraversal`]: crate::PollingTraversal
    pub const fn new(prev: P, provider: DAP) -> Self {
        Self { prev, provider, next: None }
    }
}

#[async_trait]
impl<DAP, P> OriginAdvancer for L1Retrieval<DAP, P>
where
    DAP: DataAvailabilityProvider + Send,
    P: L1RetrievalProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.prev.advance_origin().await
    }
}

#[async_trait]
impl<DAP, P> FrameQueueProvider for L1Retrieval<DAP, P>
where
    DAP: DataAvailabilityProvider + Send,
    P: L1RetrievalProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send,
{
    type Item = DAP::Item;

    async fn next_data(&mut self) -> PipelineResult<Self::Item> {
        if self.next.is_none() {
            self.next = Some(
                self.prev
                    .next_l1_block()
                    .await? // SAFETY: This question mark bubbles up the Eof error.
                    .ok_or(PipelineError::MissingL1Data.temp())?,
            );
        }
        // SAFETY: The above check ensures that `next` is not None.
        let next = self.next.as_ref().expect("infallible");

        match self.provider.next(next, self.prev.batcher_addr()).await {
            Ok(data) => Ok(data),
            Err(e) => {
                if let PipelineErrorKind::Temporary(PipelineError::Eof) = e {
                    self.next = None;
                    self.provider.clear();
                }
                Err(e)
            }
        }
    }
}

impl<DAP, P> OriginProvider for L1Retrieval<DAP, P>
where
    DAP: DataAvailabilityProvider,
    P: L1RetrievalProvider + OriginAdvancer + OriginProvider + SignalReceiver,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.prev.origin()
    }
}

#[async_trait]
impl<DAP, P> SignalReceiver for L1Retrieval<DAP, P>
where
    DAP: DataAvailabilityProvider + Send,
    P: L1RetrievalProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send,
{
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        self.prev.signal(signal).await?;
        match signal {
            Signal::Reset(ResetSignal { l1_origin, .. }) |
            Signal::Activation(ActivationSignal { l1_origin, .. }) => {
                self.next = Some(l1_origin);
            }
            _ => {}
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::{TestDAP, TraversalTestHelper};
    use alloc::vec;
    use alloy_primitives::Bytes;

    #[tokio::test]
    async fn test_l1_retrieval_flush_channel() {
        let traversal = TraversalTestHelper::new_populated();
        let dap = TestDAP { results: vec![] };
        let mut retrieval = L1Retrieval::new(traversal, dap);
        retrieval.prev.block = None;
        assert!(retrieval.prev.block.is_none());
        retrieval.next = None;
        retrieval.signal(Signal::FlushChannel).await.unwrap();
        assert!(retrieval.next.is_none());
        assert!(retrieval.prev.block.is_none());
    }

    #[tokio::test]
    async fn test_l1_retrieval_activation_signal() {
        let traversal = TraversalTestHelper::new_populated();
        let dap = TestDAP { results: vec![] };
        let mut retrieval = L1Retrieval::new(traversal, dap);
        retrieval.prev.block = None;
        assert!(retrieval.prev.block.is_none());
        retrieval.next = None;
        retrieval
            .signal(
                ActivationSignal { system_config: Some(Default::default()), ..Default::default() }
                    .signal(),
            )
            .await
            .unwrap();
        assert!(retrieval.next.is_some());
        assert_eq!(retrieval.prev.block, Some(BlockInfo::default()));
    }

    #[tokio::test]
    async fn test_l1_retrieval_reset_signal() {
        let traversal = TraversalTestHelper::new_populated();
        let dap = TestDAP { results: vec![] };
        let mut retrieval = L1Retrieval::new(traversal, dap);
        retrieval.prev.block = None;
        assert!(retrieval.prev.block.is_none());
        retrieval.next = None;
        retrieval
            .signal(
                ResetSignal { system_config: Some(Default::default()), ..Default::default() }
                    .signal(),
            )
            .await
            .unwrap();
        assert!(retrieval.next.is_some());
        assert_eq!(retrieval.prev.block, Some(BlockInfo::default()));
    }

    #[tokio::test]
    async fn test_l1_retrieval_origin() {
        let traversal = TraversalTestHelper::new_populated();
        let dap = TestDAP { results: vec![] };
        let retrieval = L1Retrieval::new(traversal, dap);
        let expected = BlockInfo::default();
        assert_eq!(retrieval.origin(), Some(expected));
    }

    #[tokio::test]
    async fn test_l1_retrieval_next_data() {
        let traversal = TraversalTestHelper::new_populated();
        let results = vec![Err(PipelineError::Eof.temp()), Ok(Bytes::default())];
        let dap = TestDAP { results };
        let mut retrieval = L1Retrieval::new(traversal, dap);
        assert_eq!(retrieval.next, None);
        let data = retrieval.next_data().await.unwrap();
        assert_eq!(data, Bytes::default());
    }

    #[tokio::test]
    async fn test_l1_retrieval_next_data_respect_next() {
        let mut traversal = TraversalTestHelper::new_populated();
        traversal.done = true;
        let results = vec![Err(PipelineError::Eof.temp()), Ok(Bytes::default())];
        let dap = TestDAP { results };
        let mut retrieval = L1Retrieval::new(traversal, dap);
        retrieval.next = Some(BlockInfo::default());
        let data = retrieval.next_data().await.unwrap();
        assert_eq!(data, Bytes::default());
        let err = retrieval.next_data().await.unwrap_err();
        assert_eq!(err, PipelineError::Eof.temp());
        assert!(retrieval.next.is_none());
    }

    #[tokio::test]
    async fn test_l1_retrieval_next_data_l1_block_errors() {
        let mut traversal = TraversalTestHelper::new_populated();
        traversal.done = true;
        let results = vec![Err(PipelineError::Eof.temp()), Ok(Bytes::default())];
        let dap = TestDAP { results };
        let mut retrieval = L1Retrieval::new(traversal, dap);
        assert_eq!(retrieval.next, None);
        let err = retrieval.next_data().await.unwrap_err();
        assert_eq!(err, PipelineError::Eof.temp());
        assert!(retrieval.next.is_none());
    }

    #[tokio::test]
    async fn test_l1_retrieval_existing_data_errors() {
        let traversal = TraversalTestHelper::new_populated();
        let dap = TestDAP { results: vec![Err(PipelineError::Eof.temp())] };
        let mut retrieval =
            L1Retrieval { prev: traversal, provider: dap, next: Some(BlockInfo::default()) };
        let data = retrieval.next_data().await.unwrap_err();
        assert_eq!(data, PipelineError::Eof.temp());
        assert!(retrieval.next.is_none());
    }
}
