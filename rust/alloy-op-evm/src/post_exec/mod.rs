//! Post-exec execution extensions.

mod inspector;

use alloc::vec::Vec;
use op_alloy::consensus::post_exec::SDMGasEntry;

pub use inspector::{
    PostExecCompositeInspector, PostExecExecutedTx, PostExecTxContext, PostExecTxKind,
    SDMWarmingInspector, WarmingRefundEvent, WarmingRefundKind,
};

use crate::{
    OpEvm,
    block::{OpBlockExecutor, receipt_builder::OpReceiptBuilder},
};

/// Extension trait for EVMs that expose post-exec warming results for the last executed
/// transaction.
pub trait PostExecEvmExt {
    /// Begin post-exec tracking for the next transaction.
    fn begin_post_exec_tx(&mut self, ctx: PostExecTxContext);

    /// Take the exact warming result for the most recently executed transaction.
    fn take_last_post_exec_tx_result(&mut self) -> PostExecExecutedTx;
}

impl<DB: alloy_evm::Database, I, P, Tx> PostExecEvmExt for OpEvm<DB, I, P, Tx> {
    fn begin_post_exec_tx(&mut self, ctx: PostExecTxContext) {
        Self::begin_post_exec_tx(self, ctx)
    }

    fn take_last_post_exec_tx_result(&mut self) -> PostExecExecutedTx {
        Self::take_last_post_exec_tx_result(self)
    }
}

/// Extension trait for block executors that collect post-exec payload entries.
pub trait PostExecExecutorExt {
    /// Take the accumulated post-exec entries for the current block.
    fn take_post_exec_entries(&mut self) -> Vec<SDMGasEntry>;

    /// Take the exact per-transaction warming refund attribution events aligned with receipts.
    fn take_warming_events_by_tx(&mut self) -> Vec<Vec<WarmingRefundEvent>>;
}

impl<E, R, Spec> PostExecExecutorExt for OpBlockExecutor<E, R, Spec>
where
    E: alloy_evm::Evm,
    R: OpReceiptBuilder,
    Spec: alloy_op_hardforks::OpHardforks + Clone,
{
    fn take_post_exec_entries(&mut self) -> Vec<SDMGasEntry> {
        Self::take_post_exec_entries(self)
    }

    fn take_warming_events_by_tx(&mut self) -> Vec<Vec<WarmingRefundEvent>> {
        Self::take_warming_events_by_tx(self)
    }
}
