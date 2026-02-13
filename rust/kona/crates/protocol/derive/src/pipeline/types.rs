//! Type aliases for the stages in the derivation pipeline.

use crate::{
    AttributesQueue, BatchProvider, BatchStream, ChannelProvider, ChannelReader, FrameQueue,
    IndexedTraversal, L1Retrieval, PollingTraversal,
};

/// Type alias for the [`L1Retrieval`] stage.
pub type L1RetrievalStage<DAP, T> = L1Retrieval<DAP, T>;

/// Type alias for the [`FrameQueue`] stage.
pub type FrameQueueStage<DAP, T> = FrameQueue<L1RetrievalStage<DAP, T>>;

/// Type alias for the [`ChannelProvider`] stage.
pub type ChannelProviderStage<DAP, T> = ChannelProvider<FrameQueueStage<DAP, T>>;

/// Type alias for the [`ChannelReader`] stage.
pub type ChannelReaderStage<DAP, T> = ChannelReader<ChannelProviderStage<DAP, T>>;

/// Type alias for the [`BatchStream`] stage.
pub type BatchStreamStage<DAP, T, F> = BatchStream<ChannelReaderStage<DAP, T>, F>;

/// Type alias for the [`BatchProvider`] stage.
pub type BatchProviderStage<DAP, T, F> = BatchProvider<BatchStreamStage<DAP, T, F>, F>;

/// Type alias for the [`AttributesQueue`] stage.
pub type AttributesQueueStage<DAP, T, F, B> = AttributesQueue<BatchProviderStage<DAP, T, F>, B>;

/// Type alias for the [`AttributesQueue`] stage that uses a [`PollingTraversal`] stage.
pub type PolledAttributesQueueStage<DAP, P, F, B> =
    AttributesQueueStage<DAP, PollingTraversal<P>, F, B>;

/// Type alias for the [`AttributesQueue`] stage that uses a [`IndexedTraversal`] stage.
pub type IndexedAttributesQueueStage<DAP, P, F, B> =
    AttributesQueueStage<DAP, IndexedTraversal<P>, F, B>;
