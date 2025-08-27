use crate::{ExecutionPayloadBaseV1, FlashBlock};
use alloy_eips::{eip2718::WithEncoded, BlockNumberOrTag, Decodable2718};
use eyre::OptionExt;
use futures_util::{Stream, StreamExt};
use reth_chain_state::ExecutedBlock;
use reth_errors::RethError;
use reth_evm::{
    execute::{BlockBuilder, BlockBuilderOutcome},
    ConfigureEvm,
};
use reth_execution_types::ExecutionOutcome;
use reth_primitives_traits::{
    AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy, SignedTransaction,
};
use reth_revm::{database::StateProviderDatabase, db::State};
use reth_rpc_eth_types::{EthApiError, PendingBlock};
use reth_storage_api::{BlockReaderIdExt, StateProviderFactory};
use std::{
    pin::Pin,
    sync::Arc,
    task::{Context, Poll},
    time::{Duration, Instant},
};
use tracing::{debug, error, info};

/// The `FlashBlockService` maintains an in-memory [`PendingBlock`] built out of a sequence of
/// [`FlashBlock`]s.
#[derive(Debug)]
pub struct FlashBlockService<
    N: NodePrimitives,
    S,
    EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: Unpin>,
    Provider,
> {
    rx: S,
    current: Option<PendingBlock<N>>,
    blocks: Vec<FlashBlock>,
    evm_config: EvmConfig,
    provider: Provider,
}

impl<
        N: NodePrimitives,
        S,
        EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<ExecutionPayloadBaseV1> + Unpin>,
        Provider: StateProviderFactory
            + BlockReaderIdExt<
                Header = HeaderTy<N>,
                Block = BlockTy<N>,
                Transaction = N::SignedTx,
                Receipt = ReceiptTy<N>,
            >,
    > FlashBlockService<N, S, EvmConfig, Provider>
{
    /// Constructs a new `FlashBlockService` that receives [`FlashBlock`]s from `rx` stream.
    pub const fn new(rx: S, evm_config: EvmConfig, provider: Provider) -> Self {
        Self { rx, current: None, blocks: Vec::new(), evm_config, provider }
    }

    /// Adds the `block` into the collection.
    ///
    /// Depending on its index and associated block number, it may:
    /// * Be added to all the flashblocks received prior using this function.
    /// * Cause a reset of the flashblocks and become the sole member of the collection.
    /// * Be ignored.
    pub fn add_flash_block(&mut self, flashblock: FlashBlock) {
        // Flash block at index zero resets the whole state
        if flashblock.index == 0 {
            self.blocks = vec![flashblock];
            self.current.take();
        }
        // Flash block at the following index adds to the collection and invalidates built block
        else if flashblock.index == self.blocks.last().map(|last| last.index + 1).unwrap_or(0) {
            self.blocks.push(flashblock);
            self.current.take();
        }
        // Flash block at a different index is ignored
        else if let Some(pending_block) = self.current.as_ref() {
            // Delete built block if it corresponds to a different height
            if pending_block.block().header().number() == flashblock.metadata.block_number {
                info!(
                    message = "None sequential Flashblocks, keeping cache",
                    curr_block = %pending_block.block().header().number(),
                    new_block = %flashblock.metadata.block_number,
                );
            } else {
                error!(
                    message = "Received Flashblock for new block, zeroing Flashblocks until we receive a base Flashblock",
                    curr_block = %pending_block.block().header().number(),
                    new_block = %flashblock.metadata.block_number,
                );

                self.blocks.clear();
                self.current.take();
            }
        } else {
            debug!("ignoring {flashblock:?}");
        }
    }

    /// Returns the [`ExecutedBlock`] made purely out of [`FlashBlock`]s that were received using
    /// [`Self::add_flash_block`].
    /// Builds a pending block using the configured provider and pool.
    ///
    /// If the origin is the actual pending block, the block is built with withdrawals.
    ///
    /// After Cancun, if the origin is the actual pending block, the block includes the EIP-4788 pre
    /// block contract call using the parent beacon block root received from the CL.
    pub fn execute(&mut self) -> eyre::Result<PendingBlock<N>> {
        let latest = self
            .provider
            .latest_header()?
            .ok_or(EthApiError::HeaderNotFound(BlockNumberOrTag::Latest.into()))?;

        let attrs = self
            .blocks
            .iter()
            .find_map(|v| v.base.clone())
            .ok_or_eyre("Missing base flashblock")?;

        let state_provider = self.provider.history_by_block_hash(latest.hash())?;
        let state = StateProviderDatabase::new(&state_provider);
        let mut db = State::builder().with_database(state).with_bundle_update().build();

        let mut builder = self
            .evm_config
            .builder_for_next_block(&mut db, &latest, attrs.into())
            .map_err(RethError::other)?;

        builder.apply_pre_execution_changes()?;

        let transactions = self.blocks.iter().flat_map(|v| v.diff.transactions.clone());

        for encoded in transactions {
            let tx = N::SignedTx::decode_2718_exact(encoded.as_ref())?;
            let signer = tx.try_recover()?;
            let tx = WithEncoded::new(encoded, tx.with_signer(signer));
            let _gas_used = builder.execute_transaction(tx)?;
        }

        let BlockBuilderOutcome { execution_result, block, hashed_state, .. } =
            builder.finish(&state_provider)?;

        let execution_outcome = ExecutionOutcome::new(
            db.take_bundle(),
            vec![execution_result.receipts],
            block.number(),
            vec![execution_result.requests],
        );

        Ok(PendingBlock::with_executed_block(
            Instant::now() + Duration::from_secs(1),
            ExecutedBlock {
                recovered_block: block.into(),
                execution_output: Arc::new(execution_outcome),
                hashed_state: Arc::new(hashed_state),
            },
        ))
    }
}

impl<
        N: NodePrimitives,
        S: Stream<Item = eyre::Result<FlashBlock>> + Unpin,
        EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<ExecutionPayloadBaseV1> + Unpin>,
        Provider: StateProviderFactory
            + BlockReaderIdExt<
                Header = HeaderTy<N>,
                Block = BlockTy<N>,
                Transaction = N::SignedTx,
                Receipt = ReceiptTy<N>,
            > + Unpin,
    > Stream for FlashBlockService<N, S, EvmConfig, Provider>
{
    type Item = eyre::Result<PendingBlock<N>>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let this = self.get_mut();
        loop {
            match this.rx.poll_next_unpin(cx) {
                Poll::Ready(Some(Ok(flashblock))) => this.add_flash_block(flashblock),
                Poll::Ready(Some(Err(err))) => return Poll::Ready(Some(Err(err))),
                Poll::Ready(None) => return Poll::Ready(None),
                Poll::Pending => return Poll::Ready(Some(this.execute())),
            }
        }
    }
}
