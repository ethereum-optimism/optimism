//! This module contains each stage of the derivation pipeline.
//!
//! It offers a high-level API to functionally apply each stage's output as an input to the next
//! stage, until finally arriving at the produced execution payloads.
//!
//! **Stages:**
//!
//! 1. L1 Traversal
//! 2. L1 Retrieval
//! 3. Frame Queue
//! 4. Channel Provider
//! 5. Channel Reader (Batch Decoding)
//! 6. Batch Stream (Introduced in the Holocene Hardfork)
//! 7. Batch Queue
//! 8. Payload Attributes Derivation
//! 9. (Omitted) Engine Queue

mod traversal;
pub use traversal::{IndexedTraversal, PollingTraversal, TraversalStage};

mod l1_retrieval;
pub use l1_retrieval::{L1Retrieval, L1RetrievalProvider};

mod frame_queue;
pub use frame_queue::{FrameQueue, FrameQueueProvider};

mod channel;
pub use channel::{
    ChannelAssembler, ChannelBank, ChannelProvider, ChannelReader, ChannelReaderProvider,
    NextFrameProvider,
};

mod batch;
pub use batch::{
    BatchProvider, BatchQueue, BatchStream, BatchStreamProvider, BatchValidator, NextBatchProvider,
};

mod attributes_queue;
pub use attributes_queue::AttributesQueue;
