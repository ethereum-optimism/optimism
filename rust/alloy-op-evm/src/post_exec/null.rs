//! Public no-op post-exec refund policy.

use alloy_primitives::{Address, U256};
use revm::{
    context_interface::ContextTr,
    inspector::JournalExt,
    interpreter::{CallInputs, CallOutcome, CreateInputs, CreateOutcome, Interpreter},
};

use super::{PostExecExecutedTx, PostExecRefundInspector, PostExecTxContext};

/// The public production post-exec refund policy.
///
/// This policy observes nothing and refunds nothing. Produce mode therefore accumulates no
/// [`SDMGasEntry`](op_alloy::consensus::post_exec::SDMGasEntry) values and payload assembly appends
/// no `0x7D` transaction. Verification of an embedded payload is independent of this policy.
#[derive(Debug, Clone, Copy, Default)]
pub struct NullRefundPolicy;

impl PostExecRefundInspector for NullRefundPolicy {
    type Snapshot = ();

    fn begin_tx(&mut self, _ctx: PostExecTxContext) {}

    fn note_account_touch(&mut self, _address: Address) {}

    fn finish_tx(&mut self) -> PostExecExecutedTx {
        PostExecExecutedTx::default()
    }

    fn inspect_step<CTX>(&mut self, _interp: &mut Interpreter, _context: &mut CTX)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_call<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CallInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
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
    use crate::post_exec::PostExecTxKind;

    #[test]
    fn null_policy_refunds_no_transaction_kind() {
        let mut policy = NullRefundPolicy;
        for kind in [PostExecTxKind::Normal, PostExecTxKind::Deposit, PostExecTxKind::PostExec] {
            policy.begin_tx(PostExecTxContext { tx_index: 1, kind });
            policy.note_account_touch(Address::ZERO);
            assert_eq!(policy.finish_tx(), PostExecExecutedTx::default());
            policy.restore(());
        }
    }
}
