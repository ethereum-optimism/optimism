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
pub use builder::{
    CommittedTxGas, OpPayloadBuilder, PayloadTransactionsWithCommitHook, RethPayloadTransactions,
    is_invalid_derived_payload_error,
};
pub mod error;
pub mod payload;
use alloy_rpc_types_engine::{ExecutionPayloadEnvelopeV2, ExecutionPayloadV1};
use op_alloy_rpc_types_engine::{
    OpExecutionData, OpExecutionPayloadEnvelope, OpExecutionPayloadEnvelopeV3,
    OpExecutionPayloadEnvelopeV4,
};
pub use payload::{
    OpBuiltPayload, OpExecData, OpPayloadAttributes, OpPayloadAttrs, OpPayloadBuilderAttributes,
    payload_id_optimism,
};
use reth_engine_primitives::EngineTypes;
use reth_optimism_primitives::OpBlock;
use reth_payload_primitives::PayloadTypes;
use reth_primitives_traits::{Block, SealedBlock};
pub mod validator;
pub use validator::OpExecutionPayloadValidator;

pub mod config;
mod sdm_metrics;

// Implement `ConfigureEngineEvm<OpExecData>` by delegating to the `OpExecutionData` implementation.
// This must live here because `OpExecData` is defined in this crate (orphan rules).
impl reth_evm::ConfigureEngineEvm<OpExecData> for reth_optimism_evm::OpEvmConfig {
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

/// The payload and engine API types of an OP node.
#[derive(Debug, Default, Clone, serde::Deserialize, serde::Serialize)]
pub struct OpEngineTypes;

impl PayloadTypes for OpEngineTypes {
    type ExecutionData = crate::payload::OpExecData;
    type BuiltPayload = OpBuiltPayload;
    type PayloadAttributes = crate::payload::OpPayloadAttrs;

    // `block_to_payload` and `From<OpBuiltPayload> for OpExecData` are
    // separate conversion paths to the same type. OP execution data does not
    // currently carry a BAL, so both paths intentionally discard it.
    /// UPSTREAM-MIRROR(copy): reth@op-v2.7.0-2
    /// `reth_ethereum_engine_primitives::EthPayloadTypes::block_to_payload`
    ///
    /// Mirrors the upstream conversion while constructing OP execution data.
    /// Re-check this path when upstream changes or OP execution data gains BAL support.
    fn block_to_payload(
        block: SealedBlock<OpBlock>,
        _bal: Option<alloy_primitives::Bytes>,
    ) -> Self::ExecutionData {
        crate::payload::OpExecData::from(OpExecutionData::from(
            OpExecutionPayloadEnvelope::from_block_unchecked(
                block.hash(),
                &block.into_block().into_ethereum_block(),
            )
            .expect("built OP blocks must normalize"),
        ))
    }
}

impl EngineTypes for OpEngineTypes {
    type ExecutionPayloadEnvelopeV1 = ExecutionPayloadV1;
    type ExecutionPayloadEnvelopeV2 = ExecutionPayloadEnvelopeV2;
    type ExecutionPayloadEnvelopeV3 = OpExecutionPayloadEnvelopeV3;
    type ExecutionPayloadEnvelopeV4 = OpExecutionPayloadEnvelopeV4;
    type ExecutionPayloadEnvelopeV5 = OpExecutionPayloadEnvelopeV4;
    type ExecutionPayloadEnvelopeV6 = OpExecutionPayloadEnvelopeV4;
}
