//! Deterministic SDM policy used by hidden, default-off testing controls.

use alloy_primitives::{Address, U256};
use reth_optimism_evm::{
    PostExecExecutedTx, PostExecRefundInspector, PostExecRefundPolicyFactory, PostExecTxContext,
};
use revm::{
    context_interface::ContextTr,
    inspector::JournalExt,
    interpreter::{CallInputs, CallOutcome, CreateInputs, CreateOutcome, Interpreter},
};

/// A deterministic fixture policy that refunds one gas per committed normal transaction.
///
/// Deposits and the synthetic post-exec transaction receive no refund. For acceptance-test fault
/// injection only, the configured excessive-refund target receives `u64::MAX`; this exercises
/// producer containment of a faulty policy.
#[derive(Debug, Clone, Copy, Default)]
pub(crate) struct FixedRefundPolicy {
    excessive_refund_target: Option<Address>,
    current_refund: u64,
}

impl FixedRefundPolicy {
    const fn new(excessive_refund_target: Option<Address>) -> Self {
        Self { excessive_refund_target, current_refund: 0 }
    }
}

/// Creates independent fixed-refund policies with the configured fault-injection target.
#[derive(Debug, Clone, Copy)]
pub(crate) struct FixedRefundPolicyFactory {
    excessive_refund_target: Option<Address>,
}

impl FixedRefundPolicyFactory {
    /// Creates a fixed-refund policy factory.
    pub(crate) const fn new(excessive_refund_target: Option<Address>) -> Self {
        Self { excessive_refund_target }
    }
}

impl PostExecRefundPolicyFactory for FixedRefundPolicyFactory {
    type Policy = FixedRefundPolicy;

    fn create(&self) -> Self::Policy {
        FixedRefundPolicy::new(self.excessive_refund_target)
    }
}

impl PostExecRefundInspector for FixedRefundPolicy {
    type Snapshot = ();

    fn begin_tx(&mut self, ctx: PostExecTxContext) {
        self.current_refund = u64::from(ctx.kind.claims_refunds());
    }

    fn note_account_touch(&mut self, _address: Address) {}

    fn finish_tx(&mut self) -> PostExecExecutedTx {
        PostExecExecutedTx {
            refund_total: core::mem::take(&mut self.current_refund),
            refund_events: Vec::new(),
        }
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
        if self.current_refund != 0 &&
            self.excessive_refund_target.is_some_and(|target| target == inputs.bytecode_address)
        {
            self.current_refund = u64::MAX;
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
    use reth_optimism_evm::{OpEvmFactory, OpTx, PostExecTxKind};

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
        type Factory = OpEvmFactory<OpTx, FixedRefundPolicyFactory>;
        assert_unit_snapshot::<Factory>();
        let _ = Factory::new(FixedRefundPolicyFactory::new(None));
    }

    #[test]
    fn fixed_policy_factories_hold_independent_targets() {
        let first_target = Address::with_last_byte(1);
        let second_target = Address::with_last_byte(2);

        let first = FixedRefundPolicyFactory::new(Some(first_target)).create();
        let second = FixedRefundPolicyFactory::new(Some(second_target)).create();

        assert_eq!(first.excessive_refund_target, Some(first_target));
        assert_eq!(second.excessive_refund_target, Some(second_target));
    }
}
