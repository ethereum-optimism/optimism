use crate::{
    chainspec::CustomChainSpec,
    evm::CustomEvmConfig,
    primitives::{CustomHeader, CustomNodePrimitives, CustomTransaction},
    CustomNode,
};
use alloy_eips::eip2718::WithEncoded;
use alloy_primitives::Bytes;
use op_alloy_rpc_types_engine::{OpExecutionData, OpExecutionPayload};
use reth_engine_primitives::EngineApiValidator;
use reth_ethereum::{
    node::api::{
        validate_version_specific_fields, AddOnsContext, BuiltPayload, BuiltPayloadExecutedBlock,
        EngineApiMessageVersion, EngineObjectValidationError, ExecutionPayload, FullNodeComponents,
        NewPayloadError, NodePrimitives, PayloadAttributes, PayloadBuilderAttributes,
        PayloadOrAttributes, PayloadTypes, PayloadValidator,
    },
    primitives::SealedBlock,
    storage::StateProviderFactory,
    trie::{KeccakKeyHasher, KeyHasher},
};
use reth_node_builder::{rpc::PayloadValidatorBuilder, InvalidPayloadAttributesError};
use reth_op::node::{
    engine::OpEngineValidator, payload::OpAttributes, OpBuiltPayload, OpEngineTypes,
    OpPayloadAttributes, OpPayloadBuilderAttributes,
};
use revm_primitives::U256;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use thiserror::Error;

#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub struct CustomPayloadTypes;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CustomExecutionData {
    pub inner: OpExecutionData,
    pub extension: u64,
}

impl ExecutionPayload for CustomExecutionData {
    fn parent_hash(&self) -> revm_primitives::B256 {
        self.inner.parent_hash()
    }

    fn block_hash(&self) -> revm_primitives::B256 {
        self.inner.block_hash()
    }

    fn block_number(&self) -> u64 {
        self.inner.block_number()
    }

    fn withdrawals(&self) -> Option<&Vec<alloy_eips::eip4895::Withdrawal>> {
        None
    }

    fn block_access_list(&self) -> Option<&Bytes> {
        None
    }

    fn parent_beacon_block_root(&self) -> Option<revm_primitives::B256> {
        self.inner.parent_beacon_block_root()
    }

    fn timestamp(&self) -> u64 {
        self.inner.timestamp()
    }

    fn gas_used(&self) -> u64 {
        self.inner.gas_used()
    }

    fn transaction_count(&self) -> usize {
        self.inner.payload.as_v1().transactions.len()
    }
}

impl TryFrom<&reth_optimism_flashblocks::FlashBlockCompleteSequence> for CustomExecutionData {
    type Error = &'static str;

    fn try_from(
        sequence: &reth_optimism_flashblocks::FlashBlockCompleteSequence,
    ) -> Result<Self, Self::Error> {
        let inner = OpExecutionData::try_from(sequence)?;
        Ok(Self { inner, extension: sequence.last().diff.gas_used })
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CustomPayloadAttributes {
    #[serde(flatten)]
    inner: OpPayloadAttributes,
    extension: u64,
}

impl PayloadAttributes for CustomPayloadAttributes {
    fn timestamp(&self) -> u64 {
        self.inner.timestamp()
    }

    fn withdrawals(&self) -> Option<&Vec<alloy_eips::eip4895::Withdrawal>> {
        self.inner.withdrawals()
    }

    fn parent_beacon_block_root(&self) -> Option<revm_primitives::B256> {
        self.inner.parent_beacon_block_root()
    }
}

#[derive(Debug, Clone)]
pub struct CustomPayloadBuilderAttributes {
    pub inner: OpPayloadBuilderAttributes<CustomTransaction>,
    pub extension: u64,
}

impl PayloadBuilderAttributes for CustomPayloadBuilderAttributes {
    type RpcPayloadAttributes = CustomPayloadAttributes;
    type Error = alloy_rlp::Error;

    fn try_new(
        parent: revm_primitives::B256,
        rpc_payload_attributes: Self::RpcPayloadAttributes,
        version: u8,
    ) -> Result<Self, Self::Error>
    where
        Self: Sized,
    {
        let CustomPayloadAttributes { inner, extension } = rpc_payload_attributes;

        Ok(Self { inner: OpPayloadBuilderAttributes::try_new(parent, inner, version)?, extension })
    }

    fn payload_id(&self) -> alloy_rpc_types_engine::PayloadId {
        self.inner.payload_id()
    }

    fn parent(&self) -> revm_primitives::B256 {
        self.inner.parent()
    }

    fn timestamp(&self) -> u64 {
        self.inner.timestamp()
    }

    fn parent_beacon_block_root(&self) -> Option<revm_primitives::B256> {
        self.inner.parent_beacon_block_root()
    }

    fn suggested_fee_recipient(&self) -> revm_primitives::Address {
        self.inner.suggested_fee_recipient()
    }

    fn prev_randao(&self) -> revm_primitives::B256 {
        self.inner.prev_randao()
    }

    fn withdrawals(&self) -> &alloy_eips::eip4895::Withdrawals {
        self.inner.withdrawals()
    }
}

impl OpAttributes for CustomPayloadBuilderAttributes {
    type Transaction = CustomTransaction;

    fn no_tx_pool(&self) -> bool {
        self.inner.no_tx_pool
    }

    fn sequencer_transactions(&self) -> &[WithEncoded<Self::Transaction>] {
        &self.inner.transactions
    }
}

#[derive(Debug, Clone)]
pub struct CustomBuiltPayload(pub OpBuiltPayload<CustomNodePrimitives>);

impl BuiltPayload for CustomBuiltPayload {
    type Primitives = CustomNodePrimitives;

    fn block(&self) -> &SealedBlock<<Self::Primitives as NodePrimitives>::Block> {
        self.0.block()
    }

    fn fees(&self) -> U256 {
        self.0.fees()
    }

    fn executed_block(&self) -> Option<BuiltPayloadExecutedBlock<Self::Primitives>> {
        self.0.executed_block()
    }

    fn requests(&self) -> Option<alloy_eips::eip7685::Requests> {
        self.0.requests()
    }
}

impl From<CustomBuiltPayload>
    for alloy_consensus::Block<<CustomNodePrimitives as NodePrimitives>::SignedTx>
{
    fn from(value: CustomBuiltPayload) -> Self {
        value.0.into_sealed_block().into_block().map_header(|header| header.inner)
    }
}

impl PayloadTypes for CustomPayloadTypes {
    type ExecutionData = CustomExecutionData;
    type BuiltPayload = OpBuiltPayload<CustomNodePrimitives>;
    type PayloadAttributes = CustomPayloadAttributes;
    type PayloadBuilderAttributes = CustomPayloadBuilderAttributes;

    fn block_to_payload(
        block: SealedBlock<
            <<Self::BuiltPayload as BuiltPayload>::Primitives as NodePrimitives>::Block,
        >,
    ) -> Self::ExecutionData {
        let extension = block.header().extension;
        let block_hash = block.hash();
        let block = block.into_block().map_header(|header| header.inner);
        let (payload, sidecar) = OpExecutionPayload::from_block_unchecked(block_hash, &block);
        CustomExecutionData { inner: OpExecutionData { payload, sidecar }, extension }
    }
}

/// Custom engine validator
#[derive(Debug, Clone)]
pub struct CustomEngineValidator<P> {
    inner: OpEngineValidator<P, CustomTransaction, CustomChainSpec>,
}

impl<P> CustomEngineValidator<P>
where
    P: Send + Sync + Unpin + 'static,
{
    /// Instantiates a new validator.
    pub fn new<KH: KeyHasher>(chain_spec: Arc<CustomChainSpec>, provider: P) -> Self {
        Self { inner: OpEngineValidator::new::<KH>(chain_spec, provider) }
    }

    /// Returns the chain spec used by the validator.
    #[inline]
    fn chain_spec(&self) -> &CustomChainSpec {
        self.inner.chain_spec()
    }
}

impl<P> PayloadValidator<CustomPayloadTypes> for CustomEngineValidator<P>
where
    P: StateProviderFactory + Send + Sync + Unpin + 'static,
{
    type Block = crate::primitives::block::Block;

    fn validate_payload_attributes_against_header(
        &self,
        _attr: &CustomPayloadAttributes,
        _header: &<Self::Block as reth_ethereum::primitives::Block>::Header,
    ) -> Result<(), InvalidPayloadAttributesError> {
        // skip default timestamp validation
        Ok(())
    }

    fn convert_payload_to_block(
        &self,
        payload: CustomExecutionData,
    ) -> Result<SealedBlock<Self::Block>, NewPayloadError> {
        let sealed_block = PayloadValidator::<OpEngineTypes>::convert_payload_to_block(
            &self.inner,
            payload.inner,
        )?;
        let (header, body) = sealed_block.split_sealed_header_body();
        let header = CustomHeader { inner: header.into_header(), extension: payload.extension };
        let body = body.map_ommers(|_| CustomHeader::default());
        Ok(SealedBlock::<Self::Block>::from_parts_unhashed(header, body))
    }
}

impl<P> EngineApiValidator<CustomPayloadTypes> for CustomEngineValidator<P>
where
    P: StateProviderFactory + Send + Sync + Unpin + 'static,
{
    fn validate_version_specific_fields(
        &self,
        version: EngineApiMessageVersion,
        payload_or_attrs: PayloadOrAttributes<'_, CustomExecutionData, CustomPayloadAttributes>,
    ) -> Result<(), EngineObjectValidationError> {
        validate_version_specific_fields(self.chain_spec(), version, payload_or_attrs)
    }

    fn ensure_well_formed_attributes(
        &self,
        version: EngineApiMessageVersion,
        attributes: &CustomPayloadAttributes,
    ) -> Result<(), EngineObjectValidationError> {
        validate_version_specific_fields(
            self.chain_spec(),
            version,
            PayloadOrAttributes::<CustomExecutionData, _>::PayloadAttributes(attributes),
        )?;

        // custom validation logic - ensure that the custom field is not zero
        // if attributes.extension == 0 {
        //     return Err(EngineObjectValidationError::invalid_params(
        //         CustomError::CustomFieldIsNotZero,
        //     ))
        // }

        Ok(())
    }
}

/// Custom error type used in payload attributes validation
#[derive(Debug, Error)]
pub enum CustomError {
    #[error("Custom field is not zero")]
    CustomFieldIsNotZero,
}

/// Custom engine validator builder
#[derive(Debug, Default, Clone, Copy)]
#[non_exhaustive]
pub struct CustomEngineValidatorBuilder;

impl<N> PayloadValidatorBuilder<N> for CustomEngineValidatorBuilder
where
    N: FullNodeComponents<Types = CustomNode, Evm = CustomEvmConfig>,
{
    type Validator = CustomEngineValidator<N::Provider>;

    async fn build(self, ctx: &AddOnsContext<'_, N>) -> eyre::Result<Self::Validator> {
        Ok(CustomEngineValidator::new::<KeccakKeyHasher>(
            ctx.config.chain.clone(),
            ctx.node.provider().clone(),
        ))
    }
}
