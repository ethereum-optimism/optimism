//! An executor constructor.

use alloc::boxed::Box;
use alloy_consensus::{Header, Sealed};
use alloy_eips::Encodable2718;
use alloy_evm::{
    EvmFactory, FromRecoveredTx, FromTxWithEncoded, block::BlockExecutorFactory,
    revm::context::BlockEnv,
};
use alloy_op_evm::{
    OpBlockExecutionCtx, OpBlockExecutorFactory,
    block::{OpTxEnv, receipt_builder::OpReceiptBuilder},
};
use alloy_primitives::B256;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_driver::Executor;
use kona_executor::{BlockBuildingOutcome, StatelessL2Builder, TrieDBProvider};
use kona_genesis::RollupConfig;
use kona_mpt::TrieHinter;
use op_alloy_consensus::OpTxEnvelope;
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use op_revm::OpSpecId;

/// An executor wrapper type.
///
/// `R` selects the [`OpReceiptBuilder`] used to assemble per-tx receipts. OP Stack callers pass
/// `OpAlloyReceiptBuilder`; chains with different receipt envelopes (e.g. Celo's CIP-64 receipt)
/// pass their own.
#[derive(Debug)]
pub struct KonaExecutor<'a, P, H, Evm, R>
where
    P: TrieDBProvider + Send + Sync + Clone,
    H: TrieHinter + Send + Sync + Clone,
    Evm: EvmFactory + Send + Sync + Clone,
{
    /// The rollup config for the executor.
    rollup_config: &'a RollupConfig,
    /// The trie provider for the executor.
    trie_provider: P,
    /// The trie hinter for the executor.
    trie_hinter: H,
    /// The evm factory for the executor.
    evm_factory: Evm,
    /// The receipt builder. Cloned into each stateless builder spun up at `update_safe_head`.
    receipt_builder: R,
    /// The executor.
    inner: Option<StatelessL2Builder<'a, P, H, Evm, R>>,
}

impl<'a, P, H, Evm, R> KonaExecutor<'a, P, H, Evm, R>
where
    P: TrieDBProvider + Send + Sync + Clone,
    H: TrieHinter + Send + Sync + Clone,
    Evm: EvmFactory + Send + Sync + Clone,
{
    /// Creates a new executor.
    pub const fn new(
        rollup_config: &'a RollupConfig,
        trie_provider: P,
        trie_hinter: H,
        evm_factory: Evm,
        receipt_builder: R,
        inner: Option<StatelessL2Builder<'a, P, H, Evm, R>>,
    ) -> Self {
        Self { rollup_config, trie_provider, trie_hinter, evm_factory, receipt_builder, inner }
    }
}

#[async_trait]
impl<P, H, Evm, R> Executor for KonaExecutor<'_, P, H, Evm, R>
where
    P: TrieDBProvider + Debug + Send + Sync + Clone,
    H: TrieHinter + Debug + Send + Sync + Clone,
    Evm: EvmFactory<Spec = OpSpecId, BlockEnv = BlockEnv> + Send + Sync + Clone + 'static,
    <Evm as EvmFactory>::Tx:
        FromTxWithEncoded<OpTxEnvelope> + FromRecoveredTx<OpTxEnvelope> + OpTxEnv,
    R: OpReceiptBuilder<Transaction = OpTxEnvelope> + Clone + Send + Sync + 'static,
    R::Receipt: Encodable2718 + Send + Sync + 'static,
    OpBlockExecutorFactory<R, RollupConfig, Evm>: for<'b> BlockExecutorFactory<
            EvmFactory = Evm,
            ExecutionCtx<'b> = OpBlockExecutionCtx,
            Transaction = OpTxEnvelope,
            Receipt = R::Receipt,
        >,
{
    type Error = kona_executor::ExecutorError;
    type Receipt = R::Receipt;

    /// Waits for the executor to be ready.
    async fn wait_until_ready(&mut self) {
        /* no-op for the kona executor */
        /* This is used when an engine api is used instead of a stateless block executor */
    }

    /// Updates the safe header.
    ///
    /// Since the L2 block executor is stateless, on an update to the safe head,
    /// a new executor is created with the updated header.
    fn update_safe_head(&mut self, header: Sealed<Header>) {
        self.inner = Some(StatelessL2Builder::new(
            self.rollup_config,
            self.evm_factory.clone(),
            self.receipt_builder.clone(),
            self.trie_provider.clone(),
            self.trie_hinter.clone(),
            header,
        ));
    }

    /// Execute the given payload attributes.
    async fn execute_payload(
        &mut self,
        attributes: OpPayloadAttributes,
    ) -> Result<BlockBuildingOutcome<R::Receipt>, Self::Error> {
        self.inner.as_mut().map_or_else(
            || Err(kona_executor::ExecutorError::MissingExecutor),
            |e| e.build_block(attributes),
        )
    }

    /// Computes the output root.
    fn compute_output_root(&mut self) -> Result<B256, Self::Error> {
        self.inner.as_mut().map_or_else(
            || Err(kona_executor::ExecutorError::MissingExecutor),
            |e| e.compute_output_root(),
        )
    }
}
