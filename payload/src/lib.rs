//! Optimism's payload builder implementation.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![allow(clippy::useless_let_if_seq)]

extern crate alloc;

pub mod builder;
pub use builder::OpPayloadBuilder;
pub mod error;
pub mod payload;
use op_alloy_rpc_types_engine::OpExecutionData;
pub use payload::{OpBuiltPayload, OpPayloadAttributes, OpPayloadBuilderAttributes};
mod traits;
use reth_optimism_primitives::{OpBlock, OpPrimitives};
use reth_payload_primitives::{BuiltPayload, PayloadTypes};
use reth_primitives_traits::{NodePrimitives, SealedBlock};
pub use traits::*;
pub mod validator;
pub use validator::OpExecutionPayloadValidator;

pub mod config;

/// ZST that aggregates Optimism [`PayloadTypes`].
#[derive(Debug, Default, Clone, serde::Deserialize, serde::Serialize)]
#[non_exhaustive]
pub struct OpPayloadTypes<N: NodePrimitives = OpPrimitives>(core::marker::PhantomData<N>);

impl<N: NodePrimitives> PayloadTypes for OpPayloadTypes<N>
where
    OpBuiltPayload<N>: BuiltPayload<Primitives: NodePrimitives<Block = OpBlock>>,
{
    type ExecutionData = OpExecutionData;
    type BuiltPayload = OpBuiltPayload<N>;
    type PayloadAttributes = OpPayloadAttributes;
    type PayloadBuilderAttributes = OpPayloadBuilderAttributes<N::SignedTx>;

    fn block_to_payload(
        block: SealedBlock<
            <<Self::BuiltPayload as BuiltPayload>::Primitives as NodePrimitives>::Block,
        >,
    ) -> Self::ExecutionData {
        OpExecutionData::from_block_unchecked(block.hash(), &block.into_block())
    }
}
