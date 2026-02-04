use crate::{
    chainspec::CustomChainSpec,
    evm::executor::CustomBlockExecutionCtx,
    primitives::{Block, CustomHeader, CustomTransaction},
};
use alloy_evm::block::{BlockExecutionError, BlockExecutorFactory};
use reth_ethereum::{
    evm::primitives::execute::{BlockAssembler, BlockAssemblerInput},
    primitives::Receipt,
};
use reth_op::{node::OpBlockAssembler, DepositReceipt};
use std::sync::Arc;

#[derive(Clone, Debug)]
pub struct CustomBlockAssembler {
    block_assembler: OpBlockAssembler<CustomChainSpec>,
}

impl CustomBlockAssembler {
    pub const fn new(chain_spec: Arc<CustomChainSpec>) -> Self {
        Self { block_assembler: OpBlockAssembler::new(chain_spec) }
    }
}

impl<F> BlockAssembler<F> for CustomBlockAssembler
where
    F: for<'a> BlockExecutorFactory<
        ExecutionCtx<'a> = CustomBlockExecutionCtx,
        Transaction = CustomTransaction,
        Receipt: Receipt + DepositReceipt,
    >,
{
    type Block = Block;

    fn assemble_block(
        &self,
        input: BlockAssemblerInput<'_, '_, F, CustomHeader>,
    ) -> Result<Self::Block, BlockExecutionError> {
        Ok(self.block_assembler.assemble_block(input)?.map_header(From::from))
    }
}
