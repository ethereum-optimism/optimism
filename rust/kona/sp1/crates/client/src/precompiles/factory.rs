//! [`EvmFactory`] implementation for the EVM in the ZKVM environment.

use super::OpZkvmPrecompiles;
use alloy_evm::{Database, EvmEnv, EvmFactory};
use alloy_op_evm::{
    OpEvm, OpEvmContext, OpTx, OpTxError,
    post_exec::{PostExecEvmFactoryHooks, PostExecTxContext, WarmingState},
};
use op_revm::{L1BlockInfo, OpBuilder, OpHaltReason, OpSpecId, OpTransaction};
use revm::{
    Context, Inspector, MainContext,
    context::{BlockEnv, CfgEnv, DBErrorMarker, TxEnv, result::EVMError},
    inspector::NoOpInspector,
};

/// Factory producing [`OpEvm`]s with FPVM-accelerated precompile overrides enabled.
#[derive(Debug, Clone)]
pub struct ZkvmOpEvmFactory;

impl PostExecEvmFactoryHooks for ZkvmOpEvmFactory {
    type Snapshot = WarmingState;

    fn begin_post_exec_tx<DB, I>(evm: &mut Self::Evm<DB, I>, ctx: PostExecTxContext)
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>,
    {
        evm.begin_post_exec_tx(ctx);
    }

    fn take_last_post_exec_refund<DB, I>(evm: &mut Self::Evm<DB, I>) -> u64
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>,
    {
        evm.take_last_post_exec_refund()
    }

    fn warming_state<DB, I>(evm: &Self::Evm<DB, I>) -> Self::Snapshot
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>,
    {
        evm.warming_state()
    }

    fn seed_warming_state<DB, I>(evm: &mut Self::Evm<DB, I>, state: Self::Snapshot)
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>,
    {
        evm.seed_warming_state(state);
    }
}

impl EvmFactory for ZkvmOpEvmFactory {
    type Evm<DB: Database, I: Inspector<OpEvmContext<DB>>> = OpEvm<DB, I, OpZkvmPrecompiles, OpTx>;
    type Context<DB: Database> = OpEvmContext<DB>;
    type Tx = OpTx;
    type Error<DBError: DBErrorMarker> = EVMError<DBError, OpTxError>;
    type HaltReason = OpHaltReason;
    type Spec = OpSpecId;
    type Precompiles = OpZkvmPrecompiles;
    type BlockEnv = BlockEnv;

    fn create_evm<DB: Database>(
        &self,
        db: DB,
        input: EvmEnv<OpSpecId>,
    ) -> Self::Evm<DB, NoOpInspector> {
        let spec_id = *input.spec_id();
        let revm_evm = Context::mainnet()
            .with_tx(OpTx(OpTransaction::<TxEnv>::builder().build_fill()))
            .with_cfg(CfgEnv::new_with_spec(OpSpecId::BEDROCK))
            .with_chain(L1BlockInfo::default())
            .with_db(db)
            .with_block(input.block_env)
            .with_cfg(input.cfg_env)
            .build_op_with_inspector(NoOpInspector {})
            .with_precompiles(OpZkvmPrecompiles::new_with_spec(spec_id));

        OpEvm::new(revm_evm, false)
    }

    fn create_evm_with_inspector<DB: Database, I: Inspector<Self::Context<DB>>>(
        &self,
        db: DB,
        input: EvmEnv<OpSpecId>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        let spec_id = *input.spec_id();
        let revm_evm = Context::mainnet()
            .with_tx(OpTx(OpTransaction::<TxEnv>::builder().build_fill()))
            .with_cfg(CfgEnv::new_with_spec(OpSpecId::BEDROCK))
            .with_chain(L1BlockInfo::default())
            .with_db(db)
            .with_block(input.block_env)
            .with_cfg(input.cfg_env)
            .build_op_with_inspector(inspector)
            .with_precompiles(OpZkvmPrecompiles::new_with_spec(spec_id));

        OpEvm::new(revm_evm, true)
    }
}
