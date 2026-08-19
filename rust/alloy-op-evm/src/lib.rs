//! OP EVM implementation.
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
use op_alloy::consensus::POST_EXEC_TX_TYPE_ID;
use op_revm::{
    L1BlockInfo, OpBuilder, OpHaltReason, OpSpecId, OpTransaction,
    constants::{BASE_FEE_RECIPIENT, L1_FEE_RECIPIENT, OPERATOR_FEE_RECIPIENT},
    precompiles::OpPrecompiles,
};
use revm::{
    Context, ExecuteEvm, InspectEvm, Inspector, Journal, MainContext, SystemCallEvm,
    context::{BlockEnv, CfgEnv, DBErrorMarker, TxEnv},
    context_interface::{
        Transaction,
        result::{EVMError, ResultAndState},
    },
    handler::{PrecompileProvider, instructions::EthInstructions},
    inspector::NoOpInspector,
    interpreter::{InterpreterResult, interpreter::EthInterpreter},
};

pub mod tx;
pub use tx::OpTx;

pub mod block;
pub use block::{
    OpBlockExecutionCtx, OpBlockExecutor, OpBlockExecutorFactory, PostExecMode, PreRefundGasUsed,
};

pub mod post_exec;

/// The OP EVM context type.
pub type OpEvmContext<DB> = Context<BlockEnv, OpTx, CfgEnv<OpSpecId>, DB, Journal<DB>, L1BlockInfo>;

type OpEvmInner<DB, I, P, R> = op_revm::OpEvm<
    OpEvmContext<DB>,
    post_exec::PostExecCompositeInspector<I, R>,
    EthInstructions<EthInterpreter, OpEvmContext<DB>>,
    P,
>;

/// OP EVM implementation.
///
/// This is a wrapper type around the `revm` evm with optional [`Inspector`] (tracing)
/// support. [`Inspector`] support is configurable at runtime because it's part of the underlying
/// [`OpEvm`](op_revm::OpEvm) type.
///
/// The `Tx` type parameter controls the transaction environment type. By default it uses
/// [`OpTx`] which wraps [`OpTransaction<TxEnv>`] and implements the necessary foreign traits.
///
/// The `R` type parameter is the post-exec refund inspector embedded alongside the user inspector
/// `I` (see [`post_exec::PostExecCompositeInspector`]). It is fixed by the EVM factory and defaults
/// to [`NullRefundPolicy`](post_exec::NullRefundPolicy).
#[allow(missing_debug_implementations)] // missing revm::OpContext Debug impl
pub struct OpEvm<DB: Database, I, P = OpPrecompiles, Tx = OpTx, R = post_exec::NullRefundPolicy> {
    inner: OpEvmInner<DB, I, P, R>,
    inspect: bool,
    post_exec_tracking_active: bool,
    last_tx_post_exec_result: post_exec::PostExecExecutedTx,
    _tx: PhantomData<Tx>,
}

impl<DB: Database, I, P, Tx, R> OpEvm<DB, I, P, Tx, R> {
    /// Consumes self and return the inner EVM instance.
    pub fn into_inner(
        self,
    ) -> op_revm::OpEvm<OpEvmContext<DB>, I, EthInstructions<EthInterpreter, OpEvmContext<DB>>, P>
    {
        let op_revm::OpEvm(revm::context::Evm {
            ctx,
            inspector,
            instruction,
            precompiles,
            frame_stack,
        }) = self.inner;

        op_revm::OpEvm(revm::context::Evm {
            ctx,
            inspector: inspector.into_inner(),
            instruction,
            precompiles,
            frame_stack,
        })
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

impl<DB: Database, I, P, Tx, R: Default> OpEvm<DB, I, P, Tx, R> {
    /// Creates a new OP EVM instance.
    ///
    /// The `inspect` argument determines whether the configured [`Inspector`] of the given
    /// [`OpEvm`](op_revm::OpEvm) should be invoked on [`Evm::transact`].
    pub fn new(
        evm: op_revm::OpEvm<
            OpEvmContext<DB>,
            I,
            EthInstructions<EthInterpreter, OpEvmContext<DB>>,
            P,
        >,
        inspect: bool,
    ) -> Self {
        let op_revm::OpEvm(revm::context::Evm {
            ctx,
            inspector,
            instruction,
            precompiles,
            frame_stack,
        }) = evm;

        Self {
            inner: op_revm::OpEvm(revm::context::Evm {
                ctx,
                inspector: post_exec::PostExecCompositeInspector::new(inspector),
                instruction,
                precompiles,
                frame_stack,
            }),
            inspect,
            post_exec_tracking_active: false,
            last_tx_post_exec_result: Default::default(),
            _tx: PhantomData,
        }
    }
}

impl<DB: Database, I, Tx, R: Default> OpEvm<DB, I, PrecompilesMap, Tx, R> {
    /// Creates an OP EVM with the standard OP context and precompiles.
    ///
    /// This is shared by factories that differ only in their fixed post-exec refund inspector.
    /// The `inspect` argument controls whether `inspector` is invoked during execution.
    pub fn from_env(
        db: DB,
        input: EvmEnv<OpSpecId, BlockEnv>,
        inspector: I,
        inspect: bool,
    ) -> Self {
        let spec_id = input.cfg_env.spec;
        let inner = Context::mainnet()
            .with_tx(OpTx(OpTransaction::builder().build_fill()))
            .with_cfg(CfgEnv::new_with_spec(OpSpecId::BEDROCK))
            .with_chain(L1BlockInfo::default())
            .with_db(db)
            .with_block(input.block_env)
            .with_cfg(input.cfg_env)
            .build_op_with_inspector(inspector)
            .with_precompiles(PrecompilesMap::from_static(
                OpPrecompiles::new_with_spec(spec_id).precompiles(),
            ));

        Self::new(inner, inspect)
    }
}

impl<DB: Database, I, P, Tx, R> OpEvm<DB, I, P, Tx, R>
where
    R: post_exec::PostExecRefundInspector,
{
    /// Begin post-exec tracking for the next transaction.
    pub fn begin_post_exec_tx(&mut self, ctx: post_exec::PostExecTxContext) {
        self.post_exec_tracking_active = true;
        self.inner.0.inspector.begin_post_exec_tx(ctx);
    }

    fn note_post_exec_account_touch(&mut self, address: Address) {
        self.inner.0.inspector.note_account_touch(address);
    }

    /// Take the extracted post-exec result for the most recently executed transaction.
    pub fn take_last_post_exec_tx_result(&mut self) -> post_exec::PostExecExecutedTx {
        core::mem::take(&mut self.last_tx_post_exec_result)
    }

    /// Snapshot refund state to carry across subblock executors.
    pub fn refund_snapshot(&self) -> R::Snapshot {
        self.inner.0.inspector.refund_snapshot()
    }

    /// Seed refund state captured from a prior subblock.
    pub fn seed_refund_snapshot(&mut self, state: R::Snapshot) {
        self.inner.0.inspector.seed_refund_snapshot(state);
    }
}

impl<DB: Database, I, P, Tx, R> post_exec::PostExecEvm for OpEvm<DB, I, P, Tx, R>
where
    Self: Evm,
    R: post_exec::PostExecRefundInspector,
{
    type Snapshot = R::Snapshot;

    fn begin_post_exec_tx(&mut self, ctx: post_exec::PostExecTxContext) {
        Self::begin_post_exec_tx(self, ctx);
    }

    fn take_last_post_exec_tx_result(&mut self) -> post_exec::PostExecExecutedTx {
        Self::take_last_post_exec_tx_result(self)
    }

    fn refund_snapshot(&self) -> Self::Snapshot {
        Self::refund_snapshot(self)
    }

    fn seed_refund_snapshot(&mut self, state: Self::Snapshot) {
        Self::seed_refund_snapshot(self, state);
    }
}

impl<Tx, R> post_exec::PostExecEvmFactoryHooks for OpEvmFactory<Tx, R>
where
    Tx: IntoTxEnv<Tx> + Into<OpTransaction<TxEnv>> + Default + Clone + Debug,
    R: Default + post_exec::PostExecRefundInspector,
{
    type Snapshot = R::Snapshot;

    fn begin_post_exec_tx<DB, I>(evm: &mut Self::Evm<DB, I>, ctx: post_exec::PostExecTxContext)
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>,
    {
        evm.begin_post_exec_tx(ctx);
    }

    fn take_last_post_exec_tx_result<DB, I>(
        evm: &mut Self::Evm<DB, I>,
    ) -> post_exec::PostExecExecutedTx
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>,
    {
        evm.take_last_post_exec_tx_result()
    }

    fn refund_snapshot<DB, I>(evm: &Self::Evm<DB, I>) -> Self::Snapshot
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>,
    {
        evm.refund_snapshot()
    }

    fn seed_refund_snapshot<DB, I>(evm: &mut Self::Evm<DB, I>, state: Self::Snapshot)
    where
        DB: Database,
        I: Inspector<Self::Context<DB>>,
    {
        evm.seed_refund_snapshot(state);
    }
}

impl<DB: Database, I, P, Tx, R> Deref for OpEvm<DB, I, P, Tx, R> {
    type Target = OpEvmContext<DB>;

    #[inline]
    fn deref(&self) -> &Self::Target {
        self.ctx()
    }
}

impl<DB: Database, I, P, Tx, R> DerefMut for OpEvm<DB, I, P, Tx, R> {
    #[inline]
    fn deref_mut(&mut self) -> &mut Self::Target {
        self.ctx_mut()
    }
}

impl<DB, I, P, Tx, R> Evm for OpEvm<DB, I, P, Tx, R>
where
    DB: Database,
    I: Inspector<OpEvmContext<DB>>,
    P: PrecompileProvider<OpEvmContext<DB>, Output = InterpreterResult>,
    Tx: IntoTxEnv<Tx> + Into<OpTransaction<TxEnv>>,
    R: post_exec::PostExecRefundInspector,
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
        self.last_tx_post_exec_result = post_exec::PostExecExecutedTx::default();

        let tx = OpTx(tx.into());

        // Post-exec transactions never execute: the block executor short-circuits them, and revm
        // would reject their tx env outright. Replay paths (e.g. RPC tracing) transact each
        // transaction directly, so synthesize the consensus-defined result here.
        if tx.tx_type() == POST_EXEC_TX_TYPE_ID {
            return Ok(post_exec::noop_post_exec_result());
        }

        // Deposits are force-included from L1 and are exempt from EIP-7825's per-transaction gas
        // limit cap: https://specs.optimism.io/protocol/karst/overview.html#execution-layer
        // Temporarily remove the cap so it cannot limit the deposit's execution, then restore it
        // so non-deposit transactions remain subject to it. Changing the cap itself, rather than
        // special-casing deposits at each place that reads it, means every reader sees the
        // exemption, including any added upstream later.
        //
        // The cap feeds `initial_gas_and_reservoir`, which splits the gas limit between the first
        // frame's budget and the EIP-8037 reservoir: removing it hands the frame the whole limit
        // and leaves the reservoir empty, which is what the exemption means while no OP fork
        // enables EIP-8037. The cap is shared across every transaction this EVM runs, and the RPC
        // call, estimate and simulate paths raise it deliberately, so the previous value is put
        // back rather than recomputed.
        let saved_tx_gas_limit_cap = (tx.tx_type() ==
            op_revm::transaction::deposit::DEPOSIT_TRANSACTION_TYPE)
            .then(|| self.inner.0.ctx.cfg.tx_gas_limit_cap.replace(u64::MAX));

        let track_post_exec = self.post_exec_tracking_active;
        let result = if self.inspect || track_post_exec {
            self.inner.inspect_tx(tx)
        } else {
            self.inner.transact(tx)
        };

        if let Some(cap) = saved_tx_gas_limit_cap {
            self.inner.0.ctx.cfg.tx_gas_limit_cap = cap;
        }

        if track_post_exec {
            if self.inner.0.ctx.tx.tx_type() !=
                op_revm::transaction::deposit::DEPOSIT_TRANSACTION_TYPE
            {
                self.note_post_exec_account_touch(L1_FEE_RECIPIENT);
                self.note_post_exec_account_touch(BASE_FEE_RECIPIENT);
                if self.inner.0.ctx.cfg.spec.is_enabled_in(OpSpecId::ISTHMUS) {
                    self.note_post_exec_account_touch(OPERATOR_FEE_RECIPIENT);
                }
            }

            self.last_tx_post_exec_result = self.inner.0.inspector.finish_post_exec_tx();
            self.post_exec_tracking_active = false;
        }

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
            self.inner.0.inspector.inner(),
            &self.inner.0.precompiles,
        )
    }

    fn components_mut(&mut self) -> (&mut Self::DB, &mut Self::Inspector, &mut Self::Precompiles) {
        (
            &mut self.inner.0.ctx.journaled_state.database,
            self.inner.0.inspector.inner_mut(),
            &mut self.inner.0.precompiles,
        )
    }
}

/// Factory producing [`OpEvm`]s.
///
/// The `Tx` type parameter controls the transaction type used by the created EVMs.
/// By default it uses [`OpTx`] which wraps [`OpTransaction<TxEnv>`] and implements
/// the necessary foreign traits.
///
/// The `R` type parameter fixes the post-exec refund inspector and its block-scoped snapshot.
/// It defaults to [`NullRefundPolicy`](post_exec::NullRefundPolicy), so released public binaries
/// cannot produce a non-empty post-exec payload.
#[derive(Debug)]
pub struct OpEvmFactory<Tx = OpTx, R = post_exec::NullRefundPolicy>(PhantomData<(Tx, R)>);

impl<Tx, R> Clone for OpEvmFactory<Tx, R> {
    fn clone(&self) -> Self {
        *self
    }
}

impl<Tx, R> Copy for OpEvmFactory<Tx, R> {}

impl<Tx, R> Default for OpEvmFactory<Tx, R> {
    fn default() -> Self {
        Self(PhantomData)
    }
}

impl<Tx, R> EvmFactory for OpEvmFactory<Tx, R>
where
    Tx: IntoTxEnv<Tx> + Into<OpTransaction<TxEnv>> + Default + Clone + Debug,
    R: Default + post_exec::PostExecRefundInspector,
{
    type Evm<DB: Database, I: Inspector<OpEvmContext<DB>>> = OpEvm<DB, I, Self::Precompiles, Tx, R>;
    type Context<DB: Database> = OpEvmContext<DB>;
    type Tx = Tx;
    type Error<DBError: DBErrorMarker> = EVMError<DBError, OpTxError>;
    type HaltReason = OpHaltReason;
    type Spec = OpSpecId;
    type BlockEnv = BlockEnv;
    type Precompiles = PrecompilesMap;

    fn create_evm<DB: Database>(
        &self,
        db: DB,
        input: EvmEnv<OpSpecId, BlockEnv>,
    ) -> Self::Evm<DB, NoOpInspector> {
        OpEvm::from_env(db, input, NoOpInspector {}, false)
    }

    fn create_evm_with_inspector<DB: Database, I: Inspector<Self::Context<DB>>>(
        &self,
        db: DB,
        input: EvmEnv<OpSpecId, BlockEnv>,
        inspector: I,
    ) -> Self::Evm<DB, I> {
        OpEvm::from_env(db, input, inspector, true)
    }
}

#[cfg(test)]
mod tests;
