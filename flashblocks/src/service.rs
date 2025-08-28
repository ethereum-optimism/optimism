use crate::{ExecutionPayloadBaseV1, FlashBlock};
use alloy_eips::{eip2718::WithEncoded, BlockNumberOrTag, Decodable2718};
use eyre::{eyre, OptionExt};
use futures_util::{FutureExt, Stream, StreamExt};
use reth_chain_state::{CanonStateNotifications, CanonStateSubscriptions, ExecutedBlock};
use reth_errors::RethError;
use reth_evm::{
    execute::{BlockBuilder, BlockBuilderOutcome},
    ConfigureEvm,
};
use reth_execution_types::ExecutionOutcome;
use reth_primitives_traits::{
    AlloyBlockHeader, BlockTy, HeaderTy, NodePrimitives, ReceiptTy, RecoveredBlock,
    SignedTransaction,
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
use tokio::pin;
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
    canon_receiver: CanonStateNotifications<N>,
}

impl<
        N: NodePrimitives,
        S,
        EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<ExecutionPayloadBaseV1> + Unpin>,
        Provider: StateProviderFactory
            + CanonStateSubscriptions<Primitives = N>
            + BlockReaderIdExt<
                Header = HeaderTy<N>,
                Block = BlockTy<N>,
                Transaction = N::SignedTx,
                Receipt = ReceiptTy<N>,
            >,
    > FlashBlockService<N, S, EvmConfig, Provider>
{
    /// Constructs a new `FlashBlockService` that receives [`FlashBlock`]s from `rx` stream.
    pub fn new(rx: S, evm_config: EvmConfig, provider: Provider) -> Self {
        Self {
            rx,
            current: None,
            blocks: Vec::new(),
            evm_config,
            canon_receiver: provider.subscribe_to_canonical_state(),
            provider,
        }
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

        if attrs.parent_hash != latest.hash() {
            return Err(eyre!("The base flashblock is old"));
        }

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

    /// Compares tip from the last notification of [`CanonStateSubscriptions`] with last computed
    /// pending block and verifies that the tip is the parent of the pending block.
    ///
    /// Returns:
    /// * `Ok(Some(true))` if tip == parent
    /// * `Ok(Some(false))` if tip != parent
    /// * `Ok(None)` if there weren't any new notifications or the pending block is not built
    /// * `Err` if the cannon state receiver returned an error
    fn verify_pending_block_integrity(
        &mut self,
        cx: &mut Context<'_>,
    ) -> eyre::Result<Option<bool>> {
        let mut tip = None;
        let fut = self.canon_receiver.recv();
        pin!(fut);

        while let Poll::Ready(result) = fut.poll_unpin(cx) {
            tip = result?.tip_checked().map(RecoveredBlock::hash);
        }

        Ok(tip
            .zip(self.current.as_ref().map(PendingBlock::parent_hash))
            .map(|(latest, parent)| latest == parent))
    }
}

impl<
        N: NodePrimitives,
        S: Stream<Item = eyre::Result<FlashBlock>> + Unpin,
        EvmConfig: ConfigureEvm<Primitives = N, NextBlockEnvCtx: From<ExecutionPayloadBaseV1> + Unpin>,
        Provider: StateProviderFactory
            + CanonStateSubscriptions<Primitives = N>
            + BlockReaderIdExt<
                Header = HeaderTy<N>,
                Block = BlockTy<N>,
                Transaction = N::SignedTx,
                Receipt = ReceiptTy<N>,
            > + Unpin,
    > Stream for FlashBlockService<N, S, EvmConfig, Provider>
{
    type Item = eyre::Result<Option<PendingBlock<N>>>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let this = self.get_mut();

        // Consume new flashblocks while they're ready
        while let Poll::Ready(Some(result)) = this.rx.poll_next_unpin(cx) {
            match result {
                Ok(flashblock) => this.add_flash_block(flashblock),
                Err(err) => return Poll::Ready(Some(Err(err))),
            }
        }

        // Execute block if there are flashblocks but no last pending block
        let changed = if this.current.is_none() && !this.blocks.is_empty() {
            match this.execute() {
                Ok(block) => this.current = Some(block),
                Err(err) => return Poll::Ready(Some(Err(err))),
            }

            true
        } else {
            false
        };

        // Verify that pending block is following up to the canonical state
        match this.verify_pending_block_integrity(cx) {
            // Integrity check failed: erase last block
            Ok(Some(false)) => Poll::Ready(Some(Ok(None))),
            // Integrity check is OK or skipped: output last block
            Ok(Some(true) | None) => {
                if changed {
                    Poll::Ready(Some(Ok(this.current.clone())))
                } else {
                    Poll::Pending
                }
            }
            // Cannot check integrity: error occurred
            Err(err) => Poll::Ready(Some(Err(err))),
        }
    }
}
