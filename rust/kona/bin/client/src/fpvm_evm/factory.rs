//! [`EvmFactory`] implementation for the EVM in the FPVM environment.

use super::precompiles::OpFpvmPrecompiles;
use alloy_evm::{Database, EvmEnv, EvmFactory};
use alloy_op_evm::OpEvm;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use op_revm::{
    DefaultOp, OpContext, OpEvm as RevmOpEvm, OpHaltReason, OpSpecId, OpTransaction,
    OpTransactionError,
};
use revm::{
    Context, Inspector,
    context::{BlockEnv, Evm as RevmEvm, FrameStack, TxEnv, result::EVMError},
    handler::instructions::EthInstructions,
    inspector::NoOpInspector,
};

/// Factory producing [`OpEvm`]s with FPVM-accelerated precompile overrides enabled.
#[derive(Debug, Clone)]
pub struct FpvmOpEvmFactory<H, O> {
    /// The hint writer.
    hint_writer: H,
    /// The oracle reader.
    oracle_reader: O,
}

impl<H, O> FpvmOpEvmFactory<H, O>
where
    H: HintWriterClient + Clone + Send + Sync,
    O: PreimageOracleClient + Clone + Send + Sync,
{
    /// Creates a new [`FpvmOpEvmFactory`].
    pub fn new(hint_writer: H, oracle_reader: O) -> Self {
        Self { hint_writer, oracle_reader }
    }

    /// Returns a reference to the inner [`HintWriterClient`].
    pub fn hint_writer(&self) -> &H {
        &self.hint_writer
    }

    /// Returns a reference to the inner [`PreimageOracleClient`].
    pub fn oracle_reader(&self) -> &O {
        &self.oracle_reader
    }
}

impl<H, O> EvmFactory for FpvmOpEvmFactory<H, O>
where
    H: HintWriterClient + Clone + Send + Sync + 'static,
    O: PreimageOracleClient + Clone + Send + Sync + 'static,
{
    type Evm<DB: Database, I: Inspector<OpContext<DB>>> = OpEvm<DB, I, OpFpvmPrecompiles<H, O>>;
    type Context<DB: Database> = OpContext<DB>;
    type Tx = OpTransaction<TxEnv>;
    type Error<DBError: core::error::Error + Send + Sync + 'static> =
        EVMError<DBError, OpTransactionError>;
    type HaltReason = OpHaltReason;
    type Spec = OpSpecId;
    type Precompiles = OpFpvmPrecompiles<H, O>;
    type BlockEnv = BlockEnv;

    fn create_evm<DB: Database>(
        &self,
        db: DB,
        input: EvmEnv<OpSpecId>,
    ) -> Self::Evm<DB, NoOpInspector> {
        let spec_id = *input.spec_id();
        let ctx = Context::op().with_db(db).with_block(input.block_env).with_cfg(input.cfg_env);
        let revm_evm = RevmOpEvm(RevmEvm {
            ctx,
            inspector: NoOpInspector {},
            instruction: EthInstructions::new_mainnet(),
            precompiles: OpFpvmPrecompiles::new_with_spec(
                spec_id,
                self.hint_writer.clone(),
                self.oracle_reader.clone(),
            ),
            frame_stack: FrameStack::new(),
        });

        OpEvm::new(revm_evm, false)
    }

    fn create_evm_with_inspector<DB: Database, I: Inspector<Self::Context<DB>>>(
        &self,
        db: DB,
        input: EvmEnv<OpSpecId>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        let spec_id = *input.spec_id();
        let ctx = Context::op().with_db(db).with_block(input.block_env).with_cfg(input.cfg_env);
        let revm_evm = RevmOpEvm(RevmEvm {
            ctx,
            inspector,
            instruction: EthInstructions::new_mainnet(),
            precompiles: OpFpvmPrecompiles::new_with_spec(
                spec_id,
                self.hint_writer.clone(),
                self.oracle_reader.clone(),
            ),
            frame_stack: FrameStack::new(),
        });

        OpEvm::new(revm_evm, true)
    }
}
