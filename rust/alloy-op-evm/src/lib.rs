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
    L1BlockInfo, OpBuilder, OpHaltReason, OpSpecId, OpTransaction, precompiles::OpPrecompiles,
};
use revm::{
    Context, ExecuteEvm, InspectEvm, Inspector, Journal, MainContext, SystemCallEvm,
    context::{BlockEnv, CfgEnv, TxEnv},
    context_interface::result::{EVMError, ResultAndState},
    handler::{PrecompileProvider, instructions::EthInstructions},
    inspector::NoOpInspector,
    interpreter::{InterpreterResult, interpreter::EthInterpreter},
};

pub mod tx;
pub use tx::OpTx;

pub mod block;
pub use block::{OpBlockExecutionCtx, OpBlockExecutor, OpBlockExecutorFactory};

/// The OP EVM context type.
pub type OpEvmContext<DB> = Context<BlockEnv, OpTx, CfgEnv<OpSpecId>, DB, Journal<DB>, L1BlockInfo>;

/// OP EVM implementation.
///
/// This is a wrapper type around the `revm` evm with optional [`Inspector`] (tracing)
/// support. [`Inspector`] support is configurable at runtime because it's part of the underlying
/// [`OpEvm`](op_revm::OpEvm) type.
///
/// The `Tx` type parameter controls the transaction environment type. By default it uses
/// [`OpTx`] which wraps [`OpTransaction<TxEnv>`] and implements the necessary foreign traits.
#[allow(missing_debug_implementations)] // missing revm::OpContext Debug impl
pub struct OpEvm<DB: Database, I, P = OpPrecompiles, Tx = OpTx> {
    inner:
        op_revm::OpEvm<OpEvmContext<DB>, I, EthInstructions<EthInterpreter, OpEvmContext<DB>>, P>,
    inspect: bool,
    _tx: PhantomData<Tx>,
}

impl<DB: Database, I, P, Tx> OpEvm<DB, I, P, Tx> {
    /// Consumes self and return the inner EVM instance.
    pub fn into_inner(
        self,
    ) -> op_revm::OpEvm<OpEvmContext<DB>, I, EthInstructions<EthInterpreter, OpEvmContext<DB>>, P>
    {
        self.inner
    }

    /// Provides a reference to the EVM context.
    pub const fn ctx(&self) -> &OpEvmContext<DB> {
        &self.inner.0.ctx
    }

    /// Provides a mutable reference to the EVM context.
    pub const fn ctx_mut(&mut self) -> &mut OpEvmContext<DB> {
        &mut self.inner.0.ctx
    }
}

impl<DB: Database, I, P, Tx> OpEvm<DB, I, P, Tx> {
    /// Creates a new OP EVM instance.
    ///
    /// The `inspect` argument determines whether the configured [`Inspector`] of the given
    /// [`OpEvm`](op_revm::OpEvm) should be invoked on [`Evm::transact`].
    pub const fn new(
        evm: op_revm::OpEvm<
            OpEvmContext<DB>,
            I,
            EthInstructions<EthInterpreter, OpEvmContext<DB>>,
            P,
        >,
        inspect: bool,
    ) -> Self {
        Self { inner: evm, inspect, _tx: PhantomData }
    }
}

impl<DB: Database, I, P, Tx> Deref for OpEvm<DB, I, P, Tx> {
    type Target = OpEvmContext<DB>;

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
    I: Inspector<OpEvmContext<DB>>,
    P: PrecompileProvider<OpEvmContext<DB>, Output = InterpreterResult>,
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

    fn cfg_env(&self) -> &CfgEnv<OpSpecId> {
        &self.cfg
    }

    fn chain_id(&self) -> u64 {
        self.cfg.chain_id
    }

    fn transact_raw(
        &mut self,
        tx: Self::Tx,
    ) -> Result<ResultAndState<Self::HaltReason>, Self::Error> {
        let result = if self.inspect {
            self.inner.inspect_tx(OpTx(tx.into()))
        } else {
            self.inner.transact(OpTx(tx.into()))
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

    fn finish(self) -> (Self::DB, EvmEnv<Self::Spec, Self::BlockEnv>) {
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
/// By default it uses [`OpTx`] which wraps [`OpTransaction<TxEnv>`] and implements
/// the necessary foreign traits.
#[derive(Debug)]
pub struct OpEvmFactory<Tx = OpTx>(PhantomData<Tx>);

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
    type Evm<DB: Database, I: Inspector<OpEvmContext<DB>>> = OpEvm<DB, I, Self::Precompiles, Tx>;
    type Context<DB: Database> = OpEvmContext<DB>;
    type Tx = Tx;
    type Error<DBError: core::error::Error + Send + Sync + 'static> = EVMError<DBError, OpTxError>;
    type HaltReason = OpHaltReason;
    type Spec = OpSpecId;
    type BlockEnv = BlockEnv;
    type Precompiles = PrecompilesMap;

    fn create_evm<DB: Database>(
        &self,
        db: DB,
        input: EvmEnv<OpSpecId, BlockEnv>,
    ) -> Self::Evm<DB, NoOpInspector> {
        let spec_id = input.cfg_env.spec;
        OpEvm {
            inner: Context::mainnet()
                .with_tx(OpTx(OpTransaction::builder().build_fill()))
                .with_cfg(CfgEnv::new_with_spec(OpSpecId::BEDROCK))
                .with_chain(L1BlockInfo::default())
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
        input: EvmEnv<OpSpecId, BlockEnv>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        let spec_id = input.cfg_env.spec;
        OpEvm {
            inner: Context::mainnet()
                .with_tx(OpTx(OpTransaction::builder().build_fill()))
                .with_cfg(CfgEnv::new_with_spec(OpSpecId::BEDROCK))
                .with_chain(L1BlockInfo::default())
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
    use alloc::vec;
    use alloy_evm::{
        EvmInternals,
        precompiles::{Precompile, PrecompileInput},
    };
    use alloy_primitives::U256;
    use op_revm::precompiles::{bls12_381, bn254_pair};
    use revm::{context::CfgEnv, database::EmptyDB, precompile::PrecompileHalt};

    use super::*;

    #[test]
    fn test_precompiles_jovian_fail() {
        let mut evm = OpEvmFactory::<OpTx>::default().create_evm(
            EmptyDB::default(),
            EvmEnv::new(CfgEnv::new_with_spec(OpSpecId::JOVIAN), BlockEnv::default()),
        );

        let (precompiles, ctx) = (&mut evm.inner.0.precompiles, &mut evm.inner.0.ctx);

        let jovian_precompile = precompiles.get(bn254_pair::JOVIAN.address()).unwrap();
        let result = jovian_precompile
            .call(PrecompileInput {
                data: &vec![0; bn254_pair::JOVIAN_MAX_INPUT_SIZE + 1],
                gas: u64::MAX,
                reservoir: 0,
                caller: Address::ZERO,
                value: U256::ZERO,
                is_static: false,
                target_address: Address::ZERO,
                bytecode_address: Address::ZERO,
                internals: EvmInternals::from_context(ctx),
            })
            .unwrap();

        assert!(result.is_halt());
        assert!(matches!(result.halt_reason(), Some(&PrecompileHalt::Bn254PairLength)));

        let jovian_precompile = precompiles.get(bls12_381::JOVIAN_G1_MSM.address()).unwrap();
        let result = jovian_precompile
            .call(PrecompileInput {
                data: &vec![0; bls12_381::JOVIAN_G1_MSM_MAX_INPUT_SIZE + 1],
                gas: u64::MAX,
                reservoir: 0,
                caller: Address::ZERO,
                value: U256::ZERO,
                is_static: false,
                target_address: Address::ZERO,
                bytecode_address: Address::ZERO,
                internals: EvmInternals::from_context(ctx),
            })
            .unwrap();

        assert!(result.is_halt());
        assert!(matches!(
            result.halt_reason(),
            Some(PrecompileHalt::Other(msg)) if msg.contains("G1MSM input length too long")
        ));

        let jovian_precompile = precompiles.get(bls12_381::JOVIAN_G2_MSM.address()).unwrap();
        let result = jovian_precompile
            .call(PrecompileInput {
                data: &vec![0; bls12_381::JOVIAN_G2_MSM_MAX_INPUT_SIZE + 1],
                gas: u64::MAX,
                reservoir: 0,
                caller: Address::ZERO,
                value: U256::ZERO,
                is_static: false,
                target_address: Address::ZERO,
                bytecode_address: Address::ZERO,
                internals: EvmInternals::from_context(ctx),
            })
            .unwrap();

        assert!(result.is_halt());
        assert!(matches!(
            result.halt_reason(),
            Some(PrecompileHalt::Other(msg)) if msg.contains("G2MSM input length too long")
        ));

        let jovian_precompile = precompiles.get(bls12_381::JOVIAN_PAIRING.address()).unwrap();
        let result = jovian_precompile
            .call(PrecompileInput {
                data: &vec![0; bls12_381::JOVIAN_PAIRING_MAX_INPUT_SIZE + 1],
                gas: u64::MAX,
                reservoir: 0,
                caller: Address::ZERO,
                value: U256::ZERO,
                is_static: false,
                target_address: Address::ZERO,
                bytecode_address: Address::ZERO,
                internals: EvmInternals::from_context(ctx),
            })
            .unwrap();

        assert!(result.is_halt());
        assert!(matches!(
            result.halt_reason(),
            Some(PrecompileHalt::Other(msg)) if msg.contains("Pairing input length too long")
        ));
    }

    #[test]
    fn test_precompiles_jovian() {
        let mut evm = OpEvmFactory::<OpTx>::default().create_evm(
            EmptyDB::default(),
            EvmEnv::new(CfgEnv::new_with_spec(OpSpecId::JOVIAN), BlockEnv::default()),
        );
        let (precompiles, ctx) = (&mut evm.inner.0.precompiles, &mut evm.inner.0.ctx);
        let jovian_precompile = precompiles.get(bn254_pair::JOVIAN.address()).unwrap();
        let result = jovian_precompile.call(PrecompileInput {
            data: &vec![0; bn254_pair::JOVIAN_MAX_INPUT_SIZE],
            gas: u64::MAX,
            reservoir: 0,
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
            reservoir: 0,
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
            reservoir: 0,
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
            reservoir: 0,
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
