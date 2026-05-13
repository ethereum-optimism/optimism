#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "metrics"), no_std)]

extern crate alloc;

#[cfg(feature = "async")]
#[macro_use]
extern crate tracing;

mod errors;
pub use errors::{
    BatchDecompressionError, BlobDecodingError, BlobProviderError, BuilderError,
    PipelineEncodingError, PipelineError, PipelineErrorKind, ResetError,
};

mod types;
pub use types::{ActivationSignal, PipelineResult, ResetSignal, Signal, StepResult};

mod metrics;
pub use metrics::Metrics;

// Sync derivation building blocks. `pub(crate)` for now; phase 6b promotes
// the module to `pub` as the seam for a future `kona-core` extraction.
// `core::*` is consumed by the async stages today and by `pure::Deriver`
// in phase 3 — both build on the same IO-free primitives.
mod core;

// Async derivation surface, gated behind the `async` feature.
//
// Phase 1 of the pure-derivation migration: the existing async pipeline
// (`DerivationPipeline`, async stages, `Pipeline`/`Stage`/`SignalReceiver`
// traits, IO-bound `StatefulAttributesBuilder` impl, test mocks) sits behind
// this gate. The truly-sync surface — error types, signals, results,
// metrics — stays unconditional and is reused by the pure deriver in phase 3.
//
// The plan calls out `AttributesBuilder` and `StatefulAttributesBuilder`'s
// math as "sync items"; those become sync in phase 2's carve-out. Until then
// they live behind the `async` gate because their current shape is async.
//
// See plans/2026-05-06-refactor-kona-pure-derivation-plan.md.

#[cfg(feature = "async")]
mod attributes;
#[cfg(feature = "async")]
pub use attributes::StatefulAttributesBuilder;

#[cfg(feature = "async")]
mod pipeline;
#[cfg(feature = "async")]
pub use pipeline::{
    AttributesQueueStage, BatchProviderStage, BatchStreamStage, ChannelProviderStage,
    ChannelReaderStage, DerivationPipeline, FrameQueueStage, IndexedAttributesQueueStage,
    L1RetrievalStage, PipelineBuilder, PolledAttributesQueueStage,
};

#[cfg(feature = "async")]
mod sources;
#[cfg(feature = "async")]
pub use sources::{BlobData, BlobSource, CalldataSource, EthereumDataSource};

#[cfg(feature = "async")]
mod stages;
#[cfg(feature = "async")]
pub use stages::{
    AttributesQueue, BatchProvider, BatchQueue, BatchStream, BatchStreamProvider, BatchValidator,
    ChannelAssembler, ChannelBank, ChannelProvider, ChannelReader, ChannelReaderProvider,
    FrameQueue, FrameQueueProvider, IndexedTraversal, L1Retrieval, L1RetrievalProvider,
    NextBatchProvider, NextFrameProvider, PollingTraversal, TraversalStage,
};

#[cfg(feature = "async")]
mod traits;
#[cfg(feature = "async")]
pub use traits::{
    AttributesBuilder, AttributesProvider, BatchValidationProviderDerive, BlobProvider,
    ChainProvider, DataAvailabilityProvider, L2ChainProvider, NextAttributes, OriginAdvancer,
    OriginProvider, Pipeline, ResetProvider, SignalReceiver, Stage,
};

#[cfg(all(any(test, feature = "test-utils"), feature = "async"))]
pub mod test_utils;
