//! Module containing the derivation pipeline.

mod builder;
pub use builder::PipelineBuilder;

mod core;
pub use core::DerivationPipeline;

mod types;
pub use types::{
    AttributesQueueStage, BatchProviderStage, BatchStreamStage, ChannelProviderStage,
    ChannelReaderStage, FrameQueueStage, IndexedAttributesQueueStage, L1RetrievalStage,
    PolledAttributesQueueStage,
};
