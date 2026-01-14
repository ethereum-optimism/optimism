//! The driver of the kona derivation pipeline.

use crate::{DriverError, DriverPipeline, DriverResult, Executor, PipelineCursor, TipCursor};
use alloc::{sync::Arc, vec::Vec};
use alloy_consensus::BlockBody;
use alloy_primitives::{B256, Bytes};
use alloy_rlp::Decodable;
use core::fmt::Debug;
use kona_derive::{Pipeline, PipelineError, PipelineErrorKind, Signal, SignalReceiver};
use kona_executor::BlockBuildingOutcome;
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_consensus::{OpBlock, OpTxEnvelope, OpTxType};
use spin::RwLock;

/// The Rollup Driver entrypoint.
///
/// The [`Driver`] is the main coordination component for the rollup derivation and execution
/// process. It manages the interaction between the derivation pipeline and block executor
/// to produce verified L2 blocks from L1 data.
///
/// ## Architecture
/// The driver operates with three main components:
/// - **Pipeline**: Derives L2 block attributes from L1 data
/// - **Executor**: Builds and executes L2 blocks from attributes
/// - **Cursor**: Tracks the current state of derivation progress
///
/// ## Usage Pattern
/// ```text
/// 1. Initialize driver with cursor, executor, and pipeline
/// 2. Call wait_for_executor() to ensure readiness
/// 3. Call advance_to_target() to derive blocks up to target
/// 4. Driver coordinates pipeline stepping and block execution
/// 5. Updates cursor with progress and maintains safe head artifacts
/// ```
///
/// ## Error Handling
/// The driver handles various error scenarios:
/// - Pipeline derivation failures (temporary, reset, critical)
/// - Block execution failures (with Holocene deposit-only retry)
/// - L1 data exhaustion (graceful halt)
/// - Interop mode considerations
#[derive(Debug)]
pub struct Driver<E, DP, P>
where
    E: Executor + Send + Sync + Debug,
    DP: DriverPipeline<P> + Send + Sync + Debug,
    P: Pipeline + SignalReceiver + Send + Sync + Debug,
{
    /// Marker for the pipeline type parameter.
    ///
    /// This phantom data ensures type safety while allowing the driver
    /// to work with different pipeline implementations.
    _marker: core::marker::PhantomData<P>,
    /// Cursor tracking the current L2 derivation state and safe head.
    ///
    /// The cursor maintains the current position in the derivation process,
    /// including the L2 safe head, output root, and L1 origin. It's wrapped
    /// in an `Arc<RwLock<_>>` for thread-safe shared access.
    pub cursor: Arc<RwLock<PipelineCursor>>,
    /// The block executor responsible for building and executing L2 blocks.
    ///
    /// The executor takes payload attributes from the pipeline and produces
    /// complete blocks with execution results and state changes.
    pub executor: E,
    /// The derivation pipeline that produces block attributes from L1 data.
    ///
    /// The pipeline abstracts the complex derivation logic and provides
    /// a high-level interface for producing sequential block attributes.
    pub pipeline: DP,
    /// Cached execution artifacts and transactions from the most recent safe head.
    ///
    /// This cache contains the [`BlockBuildingOutcome`] and raw transaction data
    /// from the last successfully executed block. It's used for efficiency and
    /// debugging purposes. `None` when no block has been executed yet.
    pub safe_head_artifacts: Option<(BlockBuildingOutcome, Vec<Bytes>)>,
}

impl<E, DP, P> Driver<E, DP, P>
where
    E: Executor + Send + Sync + Debug,
    DP: DriverPipeline<P> + Send + Sync + Debug,
    P: Pipeline + SignalReceiver + Send + Sync + Debug,
{
    /// Creates a new [`Driver`] instance.
    ///
    /// Initializes the driver with the provided cursor, executor, and pipeline components.
    /// The driver starts with no cached safe head artifacts.
    ///
    /// # Arguments
    /// * `cursor` - Shared cursor for tracking derivation state
    /// * `executor` - Block executor for building and executing L2 blocks
    /// * `pipeline` - Derivation pipeline for producing block attributes
    ///
    /// # Returns
    /// A new [`Driver`] instance ready for operation after calling [`Self::wait_for_executor`].
    ///
    /// # Usage
    /// ```rust,ignore
    /// let driver = Driver::new(cursor, executor, pipeline);
    /// driver.wait_for_executor().await;
    /// let result = driver.advance_to_target(&config, Some(target_block)).await;
    /// ```
    pub const fn new(cursor: Arc<RwLock<PipelineCursor>>, executor: E, pipeline: DP) -> Self {
        Self {
            _marker: core::marker::PhantomData,
            cursor,
            executor,
            pipeline,
            safe_head_artifacts: None,
        }
    }

    /// Waits until the executor is ready for block processing.
    ///
    /// This method blocks until the underlying executor has completed any necessary
    /// initialization or synchronization required before it can begin processing
    /// payload attributes and executing blocks.
    ///
    /// # Usage
    /// Must be called after creating the driver and before calling [`Self::advance_to_target`].
    /// This ensures the executor is in a valid state for block execution.
    ///
    /// # Example
    /// ```rust,ignore
    /// let mut driver = Driver::new(cursor, executor, pipeline);
    /// driver.wait_for_executor().await;  // Required before derivation
    /// ```
    pub async fn wait_for_executor(&mut self) {
        self.executor.wait_until_ready().await;
    }

    /// Advances the derivation pipeline to the target block number.
    ///
    /// This is the main driver method that coordinates the derivation pipeline and block
    /// executor to produce L2 blocks up to the specified target. It handles the complete
    /// lifecycle of block derivation including pipeline stepping, block execution, error
    /// recovery, and state updates.
    ///
    /// # Arguments
    /// * `cfg` - The rollup configuration containing chain parameters and activation heights
    /// * `target` - Optional target block number. If `None`, derives indefinitely until data source
    ///   is exhausted or an error occurs
    ///
    /// # Returns
    /// * `Ok((l2_safe_head, output_root))` - Tuple containing the final [`L2BlockInfo`] and output
    ///   root hash when target is reached or derivation completes
    /// * `Err(DriverError)` - Various error conditions that prevent further derivation
    ///
    /// # Errors
    /// This method can fail with several error types:
    ///
    /// ## Pipeline Errors
    /// - **EndOfSource (Critical)**: L1 data source exhausted
    ///   - In interop mode: Returns error immediately for caller handling
    ///   - In normal mode: Adjusts target to current safe head and halts gracefully
    /// - **Temporary**: Insufficient data, automatically retried
    /// - **Reset**: Reorg detected, pipeline reset and derivation continues
    /// - **Other Critical**: Fatal pipeline errors that stop derivation
    ///
    /// ## Execution Errors  
    /// - **Pre-Holocene**: Block execution failures cause block to be discarded
    /// - **Holocene+**: Failed blocks are retried as deposit-only blocks
    ///   - Strips non-deposit transactions and flushes invalidated channel
    ///   - If deposit-only block also fails, returns critical error
    ///
    /// ## Other Errors
    /// - **MissingOrigin**: Pipeline origin not available when expected
    /// - **BlockConversion**: Failed to convert block format
    /// - **RLP**: Failed to decode transaction data
    ///
    /// # Behavior Details
    ///
    /// ## Main Loop
    /// The method operates in a continuous loop:
    /// 1. Check if target block number reached (if specified)
    /// 2. Produce payload attributes from pipeline
    /// 3. Execute payload with executor
    /// 4. Handle execution failures with retry logic
    /// 5. Construct complete block and update cursor
    /// 6. Cache artifacts and continue
    ///
    /// ## Target Handling
    /// - If `target` is `Some(n)`: Stops when safe head reaches block `n`
    /// - If `target` is `None`: Continues until data exhausted or critical error
    /// - Target can be dynamically adjusted if data source is exhausted
    ///
    /// ## State Updates
    /// Each successful block updates:
    /// - Pipeline cursor with new L1 origin and L2 safe head
    /// - Executor safe head for next block building
    /// - Cached artifacts for the most recent block
    /// - Output root computation for verification
    ///
    /// # Usage Pattern
    /// ```rust,ignore
    /// // Derive to specific block
    /// let (safe_head, output_root) = driver
    ///     .advance_to_target(&rollup_config, Some(100))
    ///     .await?;
    ///
    /// // Derive until data exhausted  
    /// let (final_head, output_root) = driver
    ///     .advance_to_target(&rollup_config, None)
    ///     .await?;
    /// ```
    ///
    /// # Panics
    /// This method does not explicitly panic, but may propagate panics from:
    /// - RwLock poisoning (if another thread panicked while holding the cursor lock)
    /// - Executor or pipeline implementation panics
    /// - Arithmetic overflow in block number operations (highly unlikely)
    pub async fn advance_to_target(
        &mut self,
        cfg: &RollupConfig,
        mut target: Option<u64>,
    ) -> DriverResult<(L2BlockInfo, B256), E::Error> {
        loop {
            // Check if we have reached the target block number.
            let pipeline_cursor = self.cursor.read();
            let tip_cursor = pipeline_cursor.tip();
            if let Some(tb) = target {
                if tip_cursor.l2_safe_head.block_info.number >= tb {
                    info!(target: "client", "Derivation complete, reached L2 safe head.");
                    return Ok((tip_cursor.l2_safe_head, tip_cursor.l2_safe_head_output_root));
                }
            }

            let mut attributes = match self.pipeline.produce_payload(tip_cursor.l2_safe_head).await
            {
                Ok(attrs) => attrs.take_inner(),
                Err(PipelineErrorKind::Critical(PipelineError::EndOfSource)) => {
                    warn!(target: "client", "Exhausted data source; Halting derivation and using current safe head.");

                    // Adjust the target block number to the current safe head, as no more blocks
                    // can be produced.
                    if target.is_some() {
                        target = Some(tip_cursor.l2_safe_head.block_info.number);
                    };

                    // If we are in interop mode, this error must be handled by the caller.
                    // Otherwise, we continue the loop to halt derivation on the next iteration.
                    if cfg.is_interop_active(self.cursor.read().l2_safe_head().block_info.number) {
                        return Err(PipelineError::EndOfSource.crit().into());
                    } else {
                        continue;
                    }
                }
                Err(e) => {
                    error!(target: "client", "Failed to produce payload: {:?}", e);
                    return Err(DriverError::Pipeline(e));
                }
            };

            self.executor.update_safe_head(tip_cursor.l2_safe_head_header.clone());
            let outcome = match self.executor.execute_payload(attributes.clone()).await {
                Ok(outcome) => outcome,
                Err(e) => {
                    error!(target: "client", "Failed to execute L2 block: {}", e);

                    if cfg.is_holocene_active(attributes.payload_attributes.timestamp) {
                        // Retry with a deposit-only block.
                        warn!(target: "client", "Flushing current channel and retrying deposit only block");

                        // Flush the current batch and channel - if a block was replaced with a
                        // deposit-only block due to execution failure, the
                        // batch and channel it is contained in is forwards
                        // invalidated.
                        self.pipeline.signal(Signal::FlushChannel).await?;

                        // Strip out all transactions that are not deposits.
                        attributes.transactions = attributes.transactions.map(|txs| {
                            txs.into_iter()
                                .filter(|tx| !tx.is_empty() && tx[0] == OpTxType::Deposit as u8)
                                .collect::<Vec<_>>()
                        });

                        // Retry the execution.
                        self.executor.update_safe_head(tip_cursor.l2_safe_head_header.clone());
                        match self.executor.execute_payload(attributes.clone()).await {
                            Ok(header) => header,
                            Err(e) => {
                                error!(
                                    target: "client",
                                    "Critical - Failed to execute deposit-only block: {e}",
                                );
                                return Err(DriverError::Executor(e));
                            }
                        }
                    } else {
                        // Pre-Holocene, discard the block if execution fails.
                        continue;
                    }
                }
            };

            // Construct the block.
            let block = OpBlock {
                header: outcome.header.inner().clone(),
                body: BlockBody {
                    transactions: attributes
                        .transactions
                        .as_ref()
                        .unwrap_or(&Vec::new())
                        .iter()
                        .map(|tx| OpTxEnvelope::decode(&mut tx.as_ref()).map_err(DriverError::Rlp))
                        .collect::<DriverResult<Vec<OpTxEnvelope>, E::Error>>()?,
                    ommers: Vec::new(),
                    withdrawals: None,
                },
            };

            // Get the pipeline origin and update the tip cursor.
            let origin = self.pipeline.origin().ok_or(PipelineError::MissingOrigin.crit())?;
            let l2_info = L2BlockInfo::from_block_and_genesis(
                &block,
                &self.pipeline.rollup_config().genesis,
            )?;
            let tip_cursor = TipCursor::new(
                l2_info,
                outcome.header.clone(),
                self.executor.compute_output_root().map_err(DriverError::Executor)?,
            );

            // Advance the derivation pipeline cursor
            drop(pipeline_cursor);
            self.cursor.write().advance(origin, tip_cursor);

            // Update the latest safe head artifacts.
            self.safe_head_artifacts = Some((outcome, attributes.transactions.unwrap_or_default()));
        }
    }
}
