#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/alloy.jpg",
    html_favicon_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/favicon.ico"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(not(feature = "std"), no_std)]

extern crate alloc;

pub use alloy_rpc_types_engine::ForkchoiceUpdateVersion;

#[cfg(feature = "std")]
mod blocks;
#[cfg(feature = "std")]
pub use blocks::{
    BLOCK_VERSION_V1, BLOCK_VERSION_V2, BLOCK_VERSION_V3, BLOCK_VERSION_V4, BlocksWireError,
    decode_block_frame, encode_block_frame,
};

mod attributes;
pub use attributes::OpPayloadAttributes;

mod envelope;
pub use envelope::{
    OpExecutionData, OpExecutionPayloadEnvelope, OpNetworkPayloadEnvelope,
    PayloadEnvelopeEncodeError, PayloadEnvelopeError, PayloadHash,
};

mod sidecar;
pub use sidecar::OpExecutionPayloadSidecar;

pub mod payload;
pub use payload::{
    OpExecutionPayload,
    error::OpPayloadError,
    v3::OpExecutionPayloadEnvelopeV3,
    v4::{OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4},
};

pub mod flashblock;
pub use flashblock::{
    OpFlashblockError, OpFlashblockPayload, OpFlashblockPayloadBase, OpFlashblockPayloadDelta,
    OpFlashblockPayloadMetadata,
};
