//! # Flashblock Support for Optimism
//!
//! This module implements support for [Flashblocks](https://docs.base.org/chain/flashblocks),
//! which provide real-time block-like structures for faster state insight.
//!
//! ## Overview
//!
//! Flashblocks enable real-time visibility into block construction on OP Stack L2,
//! allowing users to see transaction effects before blocks are finalized. Each flashblock
//! represents a snapshot of the block's evolving state during its construction.
//!
//! ## Structure
//!
//! A flashblock sequence consists of:
//!
//! - **Base payload** ([`OpFlashblockPayloadBase`]): Immutable block properties that remain
//!   constant throughout the block construction. Only present in the first flashblock (index 0).
//!
//! - **Delta payloads** ([`OpFlashblockPayloadDelta`]): Mutable/accumulating properties that change
//!   as transactions are added. Present in all flashblocks.
//!
//! - **Metadata** ([`OpFlashblockPayloadMetadata`]): Additional information useful for indexing and
//!   analysis.
//!
//! - **Complete payload** ([`OpFlashblockPayload`]): The envelope containing all of the above,
//!   identified by a payload ID and sequential index.
//!
//! ## Usage
//!
//! Convert a sequence of flashblocks to a full execution payload:
//!
//! ```rust,ignore
//! use op_alloy_rpc_types_engine::{OpExecutionData, OpFlashblockPayload};
//!
//! let flashblocks: Vec<OpFlashblockPayload> = vec![/* ... */];
//! let execution_data = OpExecutionData::from_flashblocks(flashblocks)?;
//! # Ok::<(), op_alloy_rpc_types_engine::OpFlashblockError>(())
//! ```
//!
//! ## Validation Rules
//!
//! The [`OpExecutionData::from_flashblocks`](crate::OpExecutionData::from_flashblocks) method
//! performs comprehensive validation:
//!
//! - Indices must be sequential starting from 0
//! - Only the first flashblock (index 0) can have a base payload
//! - All flashblocks must have delta payloads
//! - The sequence must contain at least one flashblock

mod base;
pub use base::OpFlashblockPayloadBase;

mod delta;
pub use delta::OpFlashblockPayloadDelta;

mod metadata;
pub use metadata::OpFlashblockPayloadMetadata;

mod payload;
pub use payload::OpFlashblockPayload;

mod error;
pub use error::OpFlashblockError;
