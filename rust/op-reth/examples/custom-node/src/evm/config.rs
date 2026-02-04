use crate::{
    chainspec::CustomChainSpec,
    engine::{CustomExecutionData, CustomPayloadBuilderAttributes},
    evm::{alloy::CustomEvmFactory, executor::CustomBlockExecutionCtx, CustomBlockAssembler},
    primitives::{Block, CustomHeader, CustomNodePrimitives, CustomTransaction},
};
use alloy_consensus::BlockHeader;
use alloy_eips::{eip2718::WithEncoded, Decodable2718};
use alloy_evm::EvmEnv;
use alloy_op_evm::OpBlockExecutionCtx;
use alloy_rpc_types_engine::PayloadError;
use op_alloy_rpc_types_engine::flashblock::OpFlashblockPayloadBase;
use op_revm::OpSpecId;
use reth_engine_primitives::ExecutableTxIterator;
use reth_ethereum::{
    chainspec::EthChainSpec,
    node::api::{BuildNextEnv, ConfigureEvm, PayloadBuilderError},
    primitives::{SealedBlock, SealedHeader},
};
use reth_node_builder::{ConfigureEngineEvm, NewPayloadError};
use reth_op::{
    chainspec::OpHardforks,
    evm::primitives::{EvmEnvFor, ExecutionCtxFor},
    node::{OpEvmConfig, OpNextBlockEnvAttributes, OpRethReceiptBuilder},
    primitives::SignedTransaction,
};
use reth_rpc_api::eth::helpers::pending_block::BuildPendingEnv;
use revm_primitives::Bytes;
use std::sync::Arc;

#[derive(Debug, Clone)]
pub struct CustomEvmConfig {
    pub(super) inner: OpEvmConfig,
    pub(super) block_assembler: CustomBlockAssembler,
    pub(super) custom_evm_factory: CustomEvmFactory,
}

impl CustomEvmConfig {
    pub fn new(chain_spec: Arc<CustomChainSpec>) -> Self {
        Self {
            inner: OpEvmConfig::new(
                Arc::new(chain_spec.inner().clone()),
                OpRethReceiptBuilder::default(),
            ),
            block_assembler: CustomBlockAssembler::new(chain_spec),
            custom_evm_factory: CustomEvmFactory::new(),
        }
    }
}

impl ConfigureEvm for CustomEvmConfig {
    type Primitives = CustomNodePrimitives;
    type Error = <OpEvmConfig as ConfigureEvm>::Error;
    type NextBlockEnvCtx = CustomNextBlockEnvAttributes;
    type BlockExecutorFactory = Self;
    type BlockAssembler = CustomBlockAssembler;

    fn block_executor_factory(&self) -> &Self::BlockExecutorFactory {
        self
    }

    fn block_assembler(&self) -> &Self::BlockAssembler {
        &self.block_assembler
    }

    fn evm_env(&self, header: &CustomHeader) -> Result<EvmEnv<OpSpecId>, Self::Error> {
        self.inner.evm_env(header)
    }

    fn next_evm_env(
        &self,
        parent: &CustomHeader,
        attributes: &CustomNextBlockEnvAttributes,
    ) -> Result<EvmEnv<OpSpecId>, Self::Error> {
        self.inner.next_evm_env(parent, &attributes.inner)
    }

    fn context_for_block(
        &self,
        block: &SealedBlock<Block>,
    ) -> Result<CustomBlockExecutionCtx, Self::Error> {
        Ok(CustomBlockExecutionCtx {
            inner: OpBlockExecutionCtx {
                parent_hash: block.header().parent_hash(),
                parent_beacon_block_root: block.header().parent_beacon_block_root(),
                extra_data: block.header().extra_data().clone(),
            },
            extension: block.extension,
        })
    }

    fn context_for_next_block(
        &self,
        parent: &SealedHeader<CustomHeader>,
        attributes: Self::NextBlockEnvCtx,
    ) -> Result<CustomBlockExecutionCtx, Self::Error> {
        Ok(CustomBlockExecutionCtx {
            inner: OpBlockExecutionCtx {
                parent_hash: parent.hash(),
                parent_beacon_block_root: attributes.inner.parent_beacon_block_root,
                extra_data: attributes.inner.extra_data,
            },
            extension: attributes.extension,
        })
    }
}

impl ConfigureEngineEvm<CustomExecutionData> for CustomEvmConfig {
    fn evm_env_for_payload(
        &self,
        payload: &CustomExecutionData,
    ) -> Result<EvmEnvFor<Self>, Self::Error> {
        self.inner.evm_env_for_payload(&payload.inner)
    }

    fn context_for_payload<'a>(
        &self,
        payload: &'a CustomExecutionData,
    ) -> Result<ExecutionCtxFor<'a, Self>, Self::Error> {
        Ok(CustomBlockExecutionCtx {
            inner: self.inner.context_for_payload(&payload.inner)?,
            extension: payload.extension,
        })
    }

    fn tx_iterator_for_payload(
        &self,
        payload: &CustomExecutionData,
    ) -> Result<impl ExecutableTxIterator<Self>, Self::Error> {
        let transactions = payload.inner.payload.transactions().clone();
        let convert = |encoded: Bytes| {
            let tx = CustomTransaction::decode_2718_exact(encoded.as_ref())
                .map_err(Into::into)
                .map_err(PayloadError::Decode)?;
            let signer = tx.try_recover().map_err(NewPayloadError::other)?;
            Ok::<_, NewPayloadError>(WithEncoded::new(encoded, tx.with_signer(signer)))
        };
        Ok((transactions, convert))
    }
}

/// Additional parameters required for executing next block of custom transactions.
#[derive(Debug, Clone)]
pub struct CustomNextBlockEnvAttributes {
    inner: OpNextBlockEnvAttributes,
    extension: u64,
}

impl From<OpFlashblockPayloadBase> for CustomNextBlockEnvAttributes {
    fn from(value: OpFlashblockPayloadBase) -> Self {
        Self { inner: value.into(), extension: 0 }
    }
}

impl BuildPendingEnv<CustomHeader> for CustomNextBlockEnvAttributes {
    fn build_pending_env(parent: &SealedHeader<CustomHeader>) -> Self {
        Self {
            inner: OpNextBlockEnvAttributes::build_pending_env(parent),
            extension: parent.extension,
        }
    }
}

impl<H, ChainSpec> BuildNextEnv<CustomPayloadBuilderAttributes, H, ChainSpec>
    for CustomNextBlockEnvAttributes
where
    H: BlockHeader,
    ChainSpec: EthChainSpec + OpHardforks,
{
    fn build_next_env(
        attributes: &CustomPayloadBuilderAttributes,
        parent: &SealedHeader<H>,
        chain_spec: &ChainSpec,
    ) -> Result<Self, PayloadBuilderError> {
        let inner =
            OpNextBlockEnvAttributes::build_next_env(&attributes.inner, parent, chain_spec)?;

        Ok(CustomNextBlockEnvAttributes { inner, extension: attributes.extension })
    }
}
