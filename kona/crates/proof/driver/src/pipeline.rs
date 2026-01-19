//! Abstracts the derivation pipeline from the driver.
//!
//! This module provides the [`DriverPipeline`] trait which serves as a high-level
//! abstraction for the driver's derivation pipeline. The pipeline is responsible
//! for deriving L2 blocks from L1 data and producing payload attributes for execution.

use alloc::boxed::Box;
use async_trait::async_trait;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};

use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, ResetError, ResetSignal,
    SignalReceiver, StepResult,
};

/// High-level abstraction for the driver's derivation pipeline.
///
/// The [`DriverPipeline`] trait extends the base [`Pipeline`] functionality with
/// driver-specific operations needed for block production. It handles the complex
/// logic of stepping through derivation stages, managing resets and reorgs, and
/// producing payload attributes for block building.
///
/// ## Key Responsibilities
/// - Stepping through derivation pipeline stages
/// - Handling L1 origin advancement
/// - Managing pipeline resets due to reorgs or activation signals
/// - Producing payload attributes for disputed blocks
/// - Caching and cache invalidation
///
/// ## Error Handling
/// The pipeline can encounter several types of errors:
/// - **Temporary**: Retryable errors (e.g., missing data)
/// - **Reset**: Errors requiring pipeline reset (e.g., reorgs, activations)
/// - **Critical**: Fatal errors that stop derivation
#[async_trait]
pub trait DriverPipeline<P>: Pipeline + SignalReceiver
where
    P: Pipeline + SignalReceiver,
{
    /// Flushes any cached data due to a reorganization.
    ///
    /// This method clears internal caches that may contain stale data
    /// when a reorganization is detected on the L1 chain. It ensures
    /// that subsequent derivation operations work with fresh data.
    ///
    /// # Usage
    /// Called automatically when a reorg is detected during pipeline
    /// stepping, but can also be called manually if needed.
    fn flush(&mut self);

    /// Produces payload attributes for the next block after the given L2 safe head.
    ///
    /// This method advances the derivation pipeline to produce the next set of
    /// [`OpAttributesWithParent`] that can be used for block building. It handles
    /// the complex stepping logic including error recovery, resets, and reorgs.
    ///
    /// # Arguments
    /// * `l2_safe_head` - The current L2 safe head block info to build upon
    ///
    /// # Returns
    /// * `Ok(OpAttributesWithParent)` - Successfully produced payload attributes
    /// * `Err(PipelineErrorKind)` - Pipeline error preventing payload production
    ///
    /// # Errors
    /// This method can fail with various error types:
    /// - **Temporary errors**: Insufficient data, retries automatically
    /// - **Reset errors**: Reorg detected or activation needed, triggers pipeline reset
    /// - **Critical errors**: Fatal issues that require external intervention
    ///
    /// # Behavior
    /// The method operates in a loop, continuously stepping the pipeline until:
    /// 1. Payload attributes are successfully produced
    /// 2. A critical error occurs
    /// 3. The pipeline signals completion
    ///
    /// ## Reset Handling
    /// When reset errors occur:
    /// - **Reorg detected**: Flushes cache and resets to safe head
    /// - **Holocene activation**: Sends activation signal
    /// - **Other resets**: Standard reset to safe head with system config
    ///
    /// ## Step Results
    /// The pipeline can return different step results:
    /// - **PreparedAttributes**: Attributes ready for the next block
    /// - **AdvancedOrigin**: L1 origin moved forward
    /// - **OriginAdvanceErr/StepFailed**: Various error conditions
    async fn produce_payload(
        &mut self,
        l2_safe_head: L2BlockInfo,
    ) -> Result<OpAttributesWithParent, PipelineErrorKind> {
        // As we start the safe head at the disputed block's parent, we step the pipeline until the
        // first attributes are produced. All batches at and before the safe head will be
        // dropped, so the first payload will always be the disputed one.
        loop {
            match self.step(l2_safe_head).await {
                StepResult::PreparedAttributes => {
                    info!(target: "client_derivation_driver", "Stepped derivation pipeline")
                }
                StepResult::AdvancedOrigin => {
                    info!(
                        target: "client_derivation_driver",
                        l1_block_number = self.origin().map(|o| o.number).ok_or(PipelineError::MissingOrigin.crit())?,
                        "Advanced origin"
                    )
                }
                StepResult::OriginAdvanceErr(e) | StepResult::StepFailed(e) => {
                    // Break the loop unless the error signifies that there is not enough data to
                    // complete the current step. In this case, we retry the step to see if other
                    // stages can make progress.
                    match e {
                        PipelineErrorKind::Temporary(_) => {
                            trace!(target: "client_derivation_driver", "Failed to step derivation pipeline temporarily: {:?}", e);
                            continue;
                        }
                        PipelineErrorKind::Reset(e) => {
                            warn!(target: "client_derivation_driver", "Failed to step derivation pipeline due to reset: {:?}", e);
                            let system_config = self
                                .system_config_by_number(l2_safe_head.block_info.number)
                                .await?;

                            if matches!(e, ResetError::HoloceneActivation) {
                                let l1_origin =
                                    self.origin().ok_or(PipelineError::MissingOrigin.crit())?;
                                self.signal(
                                    ActivationSignal {
                                        l2_safe_head,
                                        l1_origin,
                                        system_config: Some(system_config),
                                    }
                                    .signal(),
                                )
                                .await?;
                            } else {
                                // Flushes cache if a reorg is detected.
                                if matches!(e, ResetError::ReorgDetected(_, _)) {
                                    self.flush();
                                }

                                // Reset the pipeline to the initial L2 safe head and L1 origin,
                                // and try again.
                                let l1_origin =
                                    self.origin().ok_or(PipelineError::MissingOrigin.crit())?;
                                self.signal(
                                    ResetSignal {
                                        l2_safe_head,
                                        l1_origin,
                                        system_config: Some(system_config),
                                    }
                                    .signal(),
                                )
                                .await?;
                            }
                        }
                        PipelineErrorKind::Critical(_) => {
                            warn!(target: "client_derivation_driver", "Failed to step derivation pipeline: {:?}", e);
                            return Err(e);
                        }
                    }
                }
            }

            if let Some(attrs) = self.next() {
                return Ok(attrs);
            }
        }
    }
}
