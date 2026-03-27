#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/alloy.jpg",
    html_favicon_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/favicon.ico"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]

extern crate alloc;

pub mod env;
#[cfg(feature = "engine")]
pub use env::evm_env_for_op_payload;
pub use env::{
    evm_env_for_op_block, evm_env_for_op_next_block, spec, spec_by_timestamp_after_bedrock,
};

pub mod error;
pub use error::{OpTxError, map_op_err};

use alloy_evm::{Database, Evm, EvmEnv, EvmFactory, IntoTxEnv, precompiles::PrecompilesMap};
use alloy_primitives::{Address, Bytes};
use core::{
    fmt::Debug,
    marker::PhantomData,
    ops::{Deref, DerefMut},
};
use op_revm::{
    DefaultOp, OpBuilder, OpContext, OpHaltReason, OpSpecId, OpTransaction,
    precompiles::OpPrecompiles,
};
use revm::{
    Context, ExecuteEvm, InspectEvm, Inspector, SystemCallEvm,
    context::{BlockEnv, TxEnv},
    context_interface::result::{EVMError, ResultAndState},
    handler::{PrecompileProvider, instructions::EthInstructions},
    inspector::NoOpInspector,
    interpreter::{InterpreterResult, interpreter::EthInterpreter},
};

pub mod block;
pub use block::{OpBlockExecutionCtx, OpBlockExecutor, OpBlockExecutorFactory};

/// OP EVM implementation.
///
/// This is a wrapper type around the `revm` evm with optional [`Inspector`] (tracing)
/// support. [`Inspector`] support is configurable at runtime because it's part of the underlying
/// [`OpEvm`](op_revm::OpEvm) type.
///
/// The `Tx` type parameter controls the transaction environment type. By default it uses
/// [`OpTransaction<TxEnv>`] directly, but consumers can provide a newtype wrapper to
/// satisfy additional trait bounds (e.g. `FromRecoveredTx`,
/// `TransactionEnv`).
#[allow(missing_debug_implementations)] // missing revm::OpContext Debug impl
pub struct OpEvm<DB: Database, I, P = OpPrecompiles, Tx = OpTransaction<TxEnv>> {
    inner: op_revm::OpEvm<OpContext<DB>, I, EthInstructions<EthInterpreter, OpContext<DB>>, P>,
    inspect: bool,
    _tx: PhantomData<Tx>,
}

impl<DB: Database, I, P, Tx> OpEvm<DB, I, P, Tx> {
    /// Consumes self and return the inner EVM instance.
    pub fn into_inner(
        self,
    ) -> op_revm::OpEvm<OpContext<DB>, I, EthInstructions<EthInterpreter, OpContext<DB>>, P> {
        self.inner
    }

    /// Provides a reference to the EVM context.
    pub const fn ctx(&self) -> &OpContext<DB> {
        &self.inner.0.ctx
    }

    /// Provides a mutable reference to the EVM context.
    pub const fn ctx_mut(&mut self) -> &mut OpContext<DB> {
        &mut self.inner.0.ctx
    }
}

impl<DB: Database, I, P, Tx> OpEvm<DB, I, P, Tx> {
    /// Creates a new OP EVM instance.
    ///
    /// The `inspect` argument determines whether the configured [`Inspector`] of the given
    /// [`OpEvm`](op_revm::OpEvm) should be invoked on [`Evm::transact`].
    pub const fn new(
        evm: op_revm::OpEvm<OpContext<DB>, I, EthInstructions<EthInterpreter, OpContext<DB>>, P>,
        inspect: bool,
    ) -> Self {
        Self { inner: evm, inspect, _tx: PhantomData }
    }
}

impl<DB: Database, I, P, Tx> Deref for OpEvm<DB, I, P, Tx> {
    type Target = OpContext<DB>;

    #[inline]
    fn deref(&self) -> &Self::Target {
        self.ctx()
    }
}

impl<DB: Database, I, P, Tx> DerefMut for OpEvm<DB, I, P, Tx> {
    #[inline]
    fn deref_mut(&mut self) -> &mut Self::Target {
        self.ctx_mut()
    }
}

impl<DB, I, P, Tx> Evm for OpEvm<DB, I, P, Tx>
where
    DB: Database,
    I: Inspector<OpContext<DB>>,
    P: PrecompileProvider<OpContext<DB>, Output = InterpreterResult>,
    Tx: IntoTxEnv<Tx> + Into<OpTransaction<TxEnv>>,
{
    type DB = DB;
    type Tx = Tx;
    type Error = EVMError<DB::Error, OpTxError>;
    type HaltReason = OpHaltReason;
    type Spec = OpSpecId;
    type BlockEnv = BlockEnv;
    type Precompiles = P;
    type Inspector = I;

    fn block(&self) -> &BlockEnv {
        &self.block
    }

    fn chain_id(&self) -> u64 {
        self.cfg.chain_id
    }

    fn transact_raw(
        &mut self,
        tx: Self::Tx,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        let inner_tx: OpTransaction<TxEnv> = tx.into();
        let result = if self.inspect {
            self.inner.inspect_tx(inner_tx)
        } else {
            self.inner.transact(inner_tx)
        };
        result.map_err(map_op_err)
    }

    fn transact_system_call(
        &mut self,
        caller: Address,
        contract: Address,
        data: Bytes,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        self.inner.system_call_with_caller(caller, contract, data).map_err(map_op_err)
    }

    fn finish(self) -> (Self::DB, EvmEnv<Self::Spec>) {
        let Context { block: block_env, cfg: cfg_env, journaled_state, .. } = self.inner.0.ctx;

        (journaled_state.database, EvmEnv { block_env, cfg_env })
    }

    fn set_inspector_enabled(&mut self, enabled: bool) {
        self.inspect = enabled;
    }

    fn components(&self) -> (&Self::DB, &Self::Inspector, &Self::Precompiles) {
        (
            &self.inner.0.ctx.journaled_state.database,
            &self.inner.0.inspector,
            &self.inner.0.precompiles,
        )
    }

    fn components_mut(&mut self) -> (&mut Self::DB, &mut Self::Inspector, &mut Self::Precompiles) {
        (
            &mut self.inner.0.ctx.journaled_state.database,
            &mut self.inner.0.inspector,
            &mut self.inner.0.precompiles,
        )
    }
}

/// Factory producing [`OpEvm`]s.
///
/// The `Tx` type parameter controls the transaction type used by the created EVMs.
/// By default it uses [`OpTransaction<TxEnv>`] directly, but consumers can specify a newtype
/// wrapper to satisfy additional trait bounds.
#[derive(Debug)]
pub struct OpEvmFactory<Tx = OpTransaction<TxEnv>>(PhantomData<Tx>);

impl<Tx> Clone for OpEvmFactory<Tx> {
    fn clone(&self) -> Self {
        *self
    }
}

impl<Tx> Copy for OpEvmFactory<Tx> {}

impl<Tx> Default for OpEvmFactory<Tx> {
    fn default() -> Self {
        Self(PhantomData)
    }
}

impl<Tx> EvmFactory for OpEvmFactory<Tx>
where
    Tx: IntoTxEnv<Tx> + Into<OpTransaction<TxEnv>> + Default + Clone + Debug,
{
    type Evm<DB: Database, I: Inspector<OpContext<DB>>> = OpEvm<DB, I, Self::Precompiles, Tx>;
    type Context<DB: Database> = OpContext<DB>;
    type Tx = Tx;
    type Error<DBError: core::error::Error + Send + Sync + 'static> = EVMError<DBError, OpTxError>;
    type HaltReason = OpHaltReason;
    type Spec = OpSpecId;
    type BlockEnv = BlockEnv;
    type Precompiles = PrecompilesMap;

    fn create_evm<DB: Database>(
        &self,
        db: DB,
        input: EvmEnv<OpSpecId>,
    ) -> Self::Evm<DB, NoOpInspector> {
        let spec_id = input.cfg_env.spec;
        OpEvm {
            inner: Context::op()
                .with_db(db)
                .with_block(input.block_env)
                .with_cfg(input.cfg_env)
                .build_op_with_inspector(NoOpInspector {})
                .with_precompiles(PrecompilesMap::from_static(
                    OpPrecompiles::new_with_spec(spec_id).precompiles(),
                )),
            inspect: false,
            _tx: PhantomData,
        }
    }

    fn create_evm_with_inspector<DB: Database, I: Inspector<Self::Context<DB>>>(
        &self,
        db: DB,
        input: EvmEnv<OpSpecId>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        let spec_id = input.cfg_env.spec;
        OpEvm {
            inner: Context::op()
                .with_db(db)
                .with_block(input.block_env)
                .with_cfg(input.cfg_env)
                .build_op_with_inspector(inspector)
                .with_precompiles(PrecompilesMap::from_static(
                    OpPrecompiles::new_with_spec(spec_id).precompiles(),
                )),
            inspect: true,
            _tx: PhantomData,
        }
    }
}

#[cfg(test)]
mod tests {
    use alloc::{string::ToString, vec};
    use alloy_evm::{
        EvmInternals,
        precompiles::{Precompile, PrecompileInput},
    };
    use alloy_primitives::U256;
    use op_revm::precompiles::{bls12_381, bn254_pair};
    use revm::{context::CfgEnv, database::EmptyDB, precompile::PrecompileError};

    use super::*;

    #[test]
    fn test_precompiles_jovian_fail() {
        let mut evm = OpEvmFactory::<OpTransaction<TxEnv>>::default().create_evm(
            EmptyDB::default(),
            EvmEnv::new(CfgEnv::new_with_spec(OpSpecId::JOVIAN), BlockEnv::default()),
        );

        let (precompiles, ctx) = (&mut evm.inner.0.precompiles, &mut evm.inner.0.ctx);

        let jovian_precompile = precompiles.get(bn254_pair::JOVIAN.address()).unwrap();
        let result = jovian_precompile.call(PrecompileInput {
            data: &vec![0; bn254_pair::JOVIAN_MAX_INPUT_SIZE + 1],
            gas: u64::MAX,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        });

        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), PrecompileError::Bn254PairLength));

        let jovian_precompile = precompiles.get(bls12_381::JOVIAN_G1_MSM.address()).unwrap();
        let result = jovian_precompile.call(PrecompileInput {
            data: &vec![0; bls12_381::JOVIAN_G1_MSM_MAX_INPUT_SIZE + 1],
            gas: u64::MAX,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        });

        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("G1MSM input length too long"));

        let jovian_precompile = precompiles.get(bls12_381::JOVIAN_G2_MSM.address()).unwrap();
        let result = jovian_precompile.call(PrecompileInput {
            data: &vec![0; bls12_381::JOVIAN_G2_MSM_MAX_INPUT_SIZE + 1],
            gas: u64::MAX,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        });

        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("G2MSM input length too long"));

        let jovian_precompile = precompiles.get(bls12_381::JOVIAN_PAIRING.address()).unwrap();
        let result = jovian_precompile.call(PrecompileInput {
            data: &vec![0; bls12_381::JOVIAN_PAIRING_MAX_INPUT_SIZE + 1],
            gas: u64::MAX,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        });

        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("Pairing input length too long"));
    }

    #[test]
    fn test_precompiles_jovian() {
        let mut evm = OpEvmFactory::<OpTransaction<TxEnv>>::default().create_evm(
            EmptyDB::default(),
            EvmEnv::new(CfgEnv::new_with_spec(OpSpecId::JOVIAN), BlockEnv::default()),
        );
        let (precompiles, ctx) = (&mut evm.inner.0.precompiles, &mut evm.inner.0.ctx);
        let jovian_precompile = precompiles.get(bn254_pair::JOVIAN.address()).unwrap();
        let result = jovian_precompile.call(PrecompileInput {
            data: &vec![0; bn254_pair::JOVIAN_MAX_INPUT_SIZE],
            gas: u64::MAX,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        });

        assert!(result.is_ok());

        let jovian_precompile = precompiles.get(bls12_381::JOVIAN_G1_MSM.address()).unwrap();
        let result = jovian_precompile.call(PrecompileInput {
            data: &vec![0; bls12_381::JOVIAN_G1_MSM_MAX_INPUT_SIZE],
            gas: u64::MAX,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        });

        assert!(result.is_ok());

        let jovian_precompile = precompiles.get(bls12_381::JOVIAN_G2_MSM.address()).unwrap();
        let result = jovian_precompile.call(PrecompileInput {
            data: &vec![0; bls12_381::JOVIAN_G2_MSM_MAX_INPUT_SIZE],
            gas: u64::MAX,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        });

        assert!(result.is_ok());

        let jovian_precompile = precompiles.get(bls12_381::JOVIAN_PAIRING.address()).unwrap();
        let result = jovian_precompile.call(PrecompileInput {
            data: &vec![0; bls12_381::JOVIAN_PAIRING_MAX_INPUT_SIZE],
            gas: u64::MAX,
            caller: Address::ZERO,
            value: U256::ZERO,
            is_static: false,
            target_address: Address::ZERO,
            bytecode_address: Address::ZERO,
            internals: EvmInternals::from_context(ctx),
        });

        assert!(result.is_ok());
    }
}
