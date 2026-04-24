//! Optimism's payload builder implementation.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![allow(clippy::useless_let_if_seq)]

extern crate alloc;

pub mod builder;
pub use builder::OpPayloadBuilder;
pub mod error;
pub mod payload;
use op_alloy_rpc_types_engine::OpExecutionData;
pub use payload::{
    OpBuiltPayload, OpExecData, OpPayloadAttributes, OpPayloadAttrs, OpPayloadBuilderAttributes,
    payload_id_optimism,
};
mod traits;
use reth_optimism_primitives::OpPrimitives;
use reth_payload_primitives::{BuiltPayload, PayloadTypes};
use reth_primitives_traits::{Block, NodePrimitives, SealedBlock};
pub use traits::*;
pub mod validator;
pub use validator::OpExecutionPayloadValidator;

pub mod config;

// Implement `ConfigureEngineEvm<OpExecData>` by delegating to the `OpExecutionData` implementation.
// This must live here because `OpExecData` is defined in this crate (orphan rules).
impl<ChainSpec, N, R> reth_evm::ConfigureEngineEvm<OpExecData>
    for reth_optimism_evm::OpEvmConfig<ChainSpec, N, R>
where
    N: NodePrimitives,
    R: Send + Sync + Unpin + Clone + 'static,
    ChainSpec: Send + Sync + Unpin + Clone + 'static,
    Self: reth_evm::ConfigureEngineEvm<OpExecutionData>,
{
    fn evm_env_for_payload(
        &self,
        payload: &OpExecData,
    ) -> Result<reth_evm::EvmEnvFor<Self>, <Self as reth_evm::ConfigureEvm>::Error> {
        reth_evm::ConfigureEngineEvm::<OpExecutionData>::evm_env_for_payload(self, &payload.0)
    }

    fn context_for_payload<'a>(
        &self,
        payload: &'a OpExecData,
    ) -> Result<reth_evm::ExecutionCtxFor<'a, Self>, <Self as reth_evm::ConfigureEvm>::Error> {
        reth_evm::ConfigureEngineEvm::<OpExecutionData>::context_for_payload(self, &payload.0)
    }

    fn tx_iterator_for_payload(
        &self,
        payload: &OpExecData,
    ) -> Result<impl reth_evm::ExecutableTxIterator<Self>, <Self as reth_evm::ConfigureEvm>::Error>
    {
        reth_evm::ConfigureEngineEvm::<OpExecutionData>::tx_iterator_for_payload(self, &payload.0)
    }
}

/// ZST that aggregates Optimism [`PayloadTypes`].
#[derive(Debug, Default, Clone, serde::Deserialize, serde::Serialize)]
#[non_exhaustive]
pub struct OpPayloadTypes<N: NodePrimitives = OpPrimitives>(core::marker::PhantomData<N>);

impl<N: NodePrimitives> PayloadTypes for OpPayloadTypes<N>
where
    OpBuiltPayload<N>: BuiltPayload,
{
    type ExecutionData = crate::payload::OpExecData;
    type BuiltPayload = OpBuiltPayload<N>;
    type PayloadAttributes = crate::payload::OpPayloadAttrs;

    fn block_to_payload(
        block: SealedBlock<
            <<Self::BuiltPayload as BuiltPayload>::Primitives as NodePrimitives>::Block,
        >,
    ) -> Self::ExecutionData {
        crate::payload::OpExecData::from(OpExecutionData::from_block_unchecked(
            block.hash(),
            &block.into_block().into_ethereum_block(),
        ))
    }
}
