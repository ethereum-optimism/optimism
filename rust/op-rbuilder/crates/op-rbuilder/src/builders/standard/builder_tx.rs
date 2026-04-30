use alloy_evm::Database;
use core::fmt::Debug;
use reth_provider::StateProvider;
use reth_revm::State;
use revm::DatabaseRef;
use tracing::warn;

use crate::{
    builders::{
        BuilderTransactionCtx, BuilderTransactionError, BuilderTransactions,
        builder_tx::BuilderTxBase, context::OpPayloadBuilderCtx,
    },
    flashtestations::builder_tx::FlashtestationsBuilderTx,
    primitives::reth::ExecutionInfo,
    tx_signer::Signer,
};

// This will be the end of block transaction of a regular block
#[derive(Debug, Clone)]
pub(super) struct StandardBuilderTx {
    pub base_builder_tx: BuilderTxBase,
    pub flashtestations_builder_tx: Option<FlashtestationsBuilderTx>,
}

impl StandardBuilderTx {
    pub(super) fn new(
        signer: Option<Signer>,
        flashtestations_builder_tx: Option<FlashtestationsBuilderTx>,
    ) -> Self {
        let base_builder_tx = BuilderTxBase::new(signer);
        Self {
            base_builder_tx,
            flashtestations_builder_tx,
        }
    }
}

impl BuilderTransactions for StandardBuilderTx {
    fn simulate_builder_txs(
        &self,
        state_provider: impl StateProvider + Clone,
        info: &mut ExecutionInfo,
        ctx: &OpPayloadBuilderCtx,
        db: &mut State<impl Database + DatabaseRef>,
        top_of_block: bool,
    ) -> Result<Vec<BuilderTransactionCtx>, BuilderTransactionError> {
        let mut builder_txs = Vec::<BuilderTransactionCtx>::new();
        let standard_builder_tx = self.base_builder_tx.simulate_builder_tx(ctx, &mut *db)?;
        builder_txs.extend(standard_builder_tx.clone());
        if let Some(flashtestations_builder_tx) = &self.flashtestations_builder_tx {
            if let Some(builder_tx) = standard_builder_tx {
                self.commit_txs(vec![builder_tx.signed_tx], ctx, db)?;
            }
            match flashtestations_builder_tx.simulate_builder_txs(
                state_provider,
                info,
                ctx,
                db,
                top_of_block,
            ) {
                Ok(flashtestations_builder_txs) => builder_txs.extend(flashtestations_builder_txs),
                Err(e) => {
                    warn!(target: "flashtestations", error = ?e, "failed to add flashtestations builder tx")
                }
            }
        }
        Ok(builder_txs)
    }
}
