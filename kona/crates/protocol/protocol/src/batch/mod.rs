//! Batch types and processing for OP Stack L2 derivation.
//!
//! This module contains comprehensive batch handling functionality for the OP Stack
//! derivation pipeline. Batches are the fundamental unit of L2 transaction data
//! that are derived from L1 data and used to construct L2 blocks.
//!
//! # Batch Types
//!
//! ## Single Batches
//! Traditional batch format containing transactions for a single L2 block.
//! Simple, straightforward format used before span batch optimization.
//!
//! ## Span Batches  
//! Advanced batch format that can contain transactions for multiple L2 blocks,
//! providing significant compression and efficiency improvements. Introduced
//! to reduce L1 data costs for high-throughput L2 chains.
//!
//! # Processing Pipeline
//!
//! ```text
//! L1 Data → Frame → Channel → Batch → L2 Block Attributes
//! ```
//!
//! # Key Components
//!
//! - **Batch Types**: [`SingleBatch`], [`SpanBatch`] for different batch formats
//! - **Batch Reading**: [`BatchReader`] for decoding batch data from channels
//! - **Validation**: [`BatchValidationProvider`] for batch validity checking
//! - **Transaction Data**: Specialized transaction formats for span batches
//! - **Error Handling**: Comprehensive error types for batch processing failures

mod r#type;
pub use r#type::*;

mod reader;
pub use reader::{BatchReader, DecompressionError};

mod tx;
pub use tx::BatchTransaction;

mod core;
pub use core::Batch;

mod raw;
pub use raw::RawSpanBatch;

mod payload;
pub use payload::SpanBatchPayload;

mod prefix;
pub use prefix::SpanBatchPrefix;

mod inclusion;
pub use inclusion::BatchWithInclusionBlock;

mod errors;
pub use errors::{BatchDecodingError, BatchEncodingError, SpanBatchError, SpanDecodingError};

mod bits;
pub use bits::SpanBatchBits;

mod span;
pub use span::SpanBatch;

mod transactions;
pub use transactions::SpanBatchTransactions;

mod element;
pub use element::{MAX_SPAN_BATCH_ELEMENTS, SpanBatchElement};

mod validity;
pub use validity::{BatchDropReason, BatchValidity};

mod single;
pub use single::SingleBatch;

mod tx_data;
pub use tx_data::{
    SpanBatchEip1559TransactionData, SpanBatchEip2930TransactionData,
    SpanBatchEip7702TransactionData, SpanBatchLegacyTransactionData, SpanBatchTransactionData,
};

mod traits;
pub use traits::BatchValidationProvider;
