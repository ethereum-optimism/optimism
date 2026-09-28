//! Deterministic SDM policy used by hidden, default-off testing controls.

use alloy_primitives::{Address, U256};
#[cfg(test)]
use reth_optimism_evm::{OpEvmFactory, OpTx};
use reth_optimism_evm::{
    PostExecExecutedTx, PostExecRefundInspector, PostExecTxContext, PostExecTxKind,
};
use revm::{
    context_interface::ContextTr,
    inspector::JournalExt,
    interpreter::{CallInputs, CallOutcome, CreateInputs, CreateOutcome, Interpreter},
};
use std::sync::OnceLock;

static EXCESSIVE_REFUND_TARGET: OnceLock<Option<Address>> = OnceLock::new();

/// Configures the optional call target used for excessive-refund fault injection.
///
/// [`reth_optimism_evm::OpEvmFactory`] constructs the policy through [`Default`], so the
/// process-level test setting must be installed before the payload service creates an EVM. A
/// production op-reth process hosts one node; accepting an identical second value also keeps
/// in-process builder tests deterministic.
pub(crate) fn configure_excessive_refund_target(target: Option<Address>) -> eyre::Result<()> {
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
pub(crate) struct FixedRefundPolicy {
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
