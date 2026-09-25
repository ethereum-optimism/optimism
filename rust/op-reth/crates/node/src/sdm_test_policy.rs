//! Test-only SDM refund policy and payload-service configuration.
//!
//! The policy is always compiled into op-reth, whose normal binary exposes it only through hidden,
//! default-off controls. Keeping selection in the payload-service builder avoids per-transaction
//! dispatch and lets test nodes use the production node type and launcher.

use crate::node::{OpNodeTypes, OpPayloadBuilder};
use alloy_primitives::{Address, U256};
use op_alloy_consensus::OpTxEnvelope;
use reth_node_api::{BuildNextEnv, NodeTypes, node::FullNodeTypes};
use reth_node_builder::{
    BuilderContext,
    components::{BasicPayloadServiceBuilder, PayloadBuilderBuilder, PayloadServiceBuilder},
};
use reth_optimism_evm::{
    ConfigurePostExecEvm, OpEvmConfig, OpEvmFactory, OpRethReceiptBuilder, OpTx,
    PostExecEvmFactoryAdapter, PostExecExecutedTx, PostExecRefundInspector, PostExecTxContext,
    PostExecTxKind,
};
use reth_optimism_payload_builder::OpPayloadBuilderAttributes;
use reth_optimism_primitives::OpPrimitives;
use reth_payload_builder::PayloadBuilderHandle;
use reth_transaction_pool::TransactionPool;
use revm::{
    context_interface::ContextTr,
    inspector::JournalExt,
    interpreter::{CallInputs, CallOutcome, CreateInputs, CreateOutcome, Interpreter},
};
use std::sync::{Arc, OnceLock};

static EXCESSIVE_REFUND_TARGET: OnceLock<Option<Address>> = OnceLock::new();

/// Configures the optional call target used for excessive-refund fault injection.
///
/// [`OpEvmFactory`] constructs the policy through [`Default`], so the process-level test setting
/// must be installed before the payload service creates an EVM. A production op-reth process hosts
/// one node; accepting an identical second value also keeps in-process builder tests deterministic.
fn configure_excessive_refund_target(target: Option<Address>) -> eyre::Result<()> {
    if EXCESSIVE_REFUND_TARGET.set(target).is_ok() {
        return Ok(());
    }

    let configured = EXCESSIVE_REFUND_TARGET.get().copied().flatten();
    if configured == target {
        return Ok(());
    }

    eyre::bail!(
        "test SDM excessive-refund target already configured as {configured:?}, cannot change it to {target:?}"
    );
}

/// A deterministic fixture policy that refunds one gas per committed normal transaction.
///
/// Deposits and the synthetic post-exec transaction receive no refund. For acceptance-test fault
/// injection only, the configured excessive-refund target receives `u64::MAX`; this exercises
/// producer containment of a faulty policy.
#[derive(Debug, Clone, Copy, Default)]
pub struct FixedRefundPolicy {
    current_kind: Option<PostExecTxKind>,
    excessive_refund: bool,
}

impl PostExecRefundInspector for FixedRefundPolicy {
    type Snapshot = ();

    fn begin_tx(&mut self, ctx: PostExecTxContext) {
        self.current_kind = Some(ctx.kind);
        self.excessive_refund = false;
    }

    fn note_account_touch(&mut self, _address: Address) {}

    fn finish_tx(&mut self) -> PostExecExecutedTx {
        let refund_total = if self.current_kind.take() == Some(PostExecTxKind::Normal) {
            if self.excessive_refund { u64::MAX } else { 1 }
        } else {
            0
        };
        PostExecExecutedTx { refund_total, refund_events: Vec::new() }
    }

    fn inspect_step<CTX>(&mut self, _interp: &mut Interpreter, _context: &mut CTX)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_call<CTX>(&mut self, _context: &mut CTX, inputs: &mut CallInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
        if EXCESSIVE_REFUND_TARGET
            .get()
            .copied()
            .flatten()
            .is_some_and(|target| target == inputs.bytecode_address)
        {
            self.excessive_refund = true;
        }
    }

    fn inspect_call_end<CTX>(
        &mut self,
        _context: &mut CTX,
        _inputs: &CallInputs,
        _outcome: &CallOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_create<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CreateInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_create_end<CTX>(
        &mut self,
        _context: &mut CTX,
        _inputs: &CreateInputs,
        _outcome: &CreateOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_selfdestruct(&mut self, _contract: Address, _target: Address, _value: U256) {}

    fn snapshot(&self) -> Self::Snapshot {}

    fn restore(&mut self, _snapshot: Self::Snapshot) {}
}

/// EVM configuration used only by the fixed SDM test policy.
type FixedPolicyOpEvmConfig<ChainSpec, N> = OpEvmConfig<
    ChainSpec,
    N,
    OpRethReceiptBuilder,
    PostExecEvmFactoryAdapter<OpEvmFactory<OpTx, FixedRefundPolicy>>,
>;

fn fixed_policy_evm_config<ChainSpec, N>(
    chain_spec: Arc<ChainSpec>,
) -> FixedPolicyOpEvmConfig<ChainSpec, N>
where
    N: reth_node_api::NodePrimitives,
{
    OpEvmConfig::new_with_evm_factory(
        chain_spec,
        OpRethReceiptBuilder::default(),
        PostExecEvmFactoryAdapter::new(OpEvmFactory::default()),
    )
}

/// Payload-service builder that selects either stock production or the fixed test policy.
///
/// Keeping the choice inside one payload-service type lets both modes use the same concrete
/// [`crate::OpNode`] and launcher while resolving the policy before transaction execution begins.
#[derive(Debug, Clone)]
pub struct TestSdmPayloadServiceBuilder {
    inner: OpPayloadBuilder,
    fixed_refund: bool,
    excessive_refund_target: Option<Address>,
}

impl TestSdmPayloadServiceBuilder {
    /// Creates a payload service using the stock null refund policy.
    pub const fn standard(inner: OpPayloadBuilder) -> Self {
        Self { inner, fixed_refund: false, excessive_refund_target: None }
    }

    /// Creates a payload service using the deterministic fixed-refund test policy.
    pub const fn fixed_refund(
        inner: OpPayloadBuilder,
        excessive_refund_target: Option<Address>,
    ) -> Self {
        Self { inner, fixed_refund: true, excessive_refund_target }
    }
}

impl<Node, Pool, EvmConfig> PayloadServiceBuilder<Node, Pool, EvmConfig>
    for TestSdmPayloadServiceBuilder
where
    Node: FullNodeTypes<Types: OpNodeTypes>,
    Pool: TransactionPool + Clone + Send + Sync + Unpin + 'static,
    EvmConfig: Send,
    FixedPolicyOpEvmConfig<<Node::Types as NodeTypes>::ChainSpec, OpPrimitives>:
        ConfigurePostExecEvm<
                Primitives = OpPrimitives,
                NextBlockEnvCtx: BuildNextEnv<
                    OpPayloadBuilderAttributes<OpTxEnvelope>,
                    alloy_consensus::Header,
                    <Node::Types as NodeTypes>::ChainSpec,
                >,
            > + Clone
            + Send
            + Sync
            + Unpin
            + 'static,
    OpPayloadBuilder: PayloadBuilderBuilder<Node, Pool, EvmConfig>
        + PayloadBuilderBuilder<
            Node,
            Pool,
            FixedPolicyOpEvmConfig<<Node::Types as NodeTypes>::ChainSpec, OpPrimitives>,
        >,
{
    async fn spawn_payload_builder_service(
        self,
        ctx: &BuilderContext<Node>,
        pool: Pool,
        evm_config: EvmConfig,
    ) -> eyre::Result<PayloadBuilderHandle<<Node::Types as NodeTypes>::Payload>> {
        let Self { inner, fixed_refund, excessive_refund_target } = self;
        if fixed_refund {
            configure_excessive_refund_target(excessive_refund_target)?;
            let evm_config = fixed_policy_evm_config::<
                <Node::Types as NodeTypes>::ChainSpec,
                OpPrimitives,
            >(ctx.chain_spec());
            BasicPayloadServiceBuilder::new(inner)
                .spawn_payload_builder_service(ctx, pool, evm_config)
                .await
        } else {
            BasicPayloadServiceBuilder::new(inner)
                .spawn_payload_builder_service(ctx, pool, evm_config)
                .await
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fixed_policy_refunds_only_normal_transactions() {
        let mut policy = FixedRefundPolicy::default();
        for (kind, expected) in [
            (PostExecTxKind::Normal, 1),
            (PostExecTxKind::Deposit, 0),
            (PostExecTxKind::PostExec, 0),
        ] {
            policy.begin_tx(PostExecTxContext { tx_index: 3, kind });
            policy.note_account_touch(Address::ZERO);
            assert_eq!(policy.finish_tx().refund_total, expected);
        }
    }

    #[test]
    fn fixed_policy_factory_uses_unit_snapshots() {
        fn assert_unit_snapshot<
            F: alloy_op_evm::post_exec::PostExecEvmFactoryHooks<Snapshot = ()>,
        >() {
        }
        assert_unit_snapshot::<OpEvmFactory<OpTx, FixedRefundPolicy>>();
        let _ = OpEvmFactory::<OpTx, FixedRefundPolicy>::default();
    }
}
