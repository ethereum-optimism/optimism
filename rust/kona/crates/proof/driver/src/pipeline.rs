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
    SignalReceiver,
};

/// High-level abstraction for the driver's derivation pipeline.
///
/// The [`DriverPipeline`] trait extends the base [`Pipeline`] functionality with
/// driver-specific operations needed for block production. It handles the complex
/// logic of stepping through derivation stages, managing resets and reorgs, and
/// producing payload attributes for block building.
#[async_trait]
pub trait DriverPipeline<P>: Pipeline + SignalReceiver
where
    P: Pipeline + SignalReceiver,
{
    /// Flushes any cached data due to a reorganization.
    fn flush(&mut self);

    /// Produces payload attributes for the next block after the given L2 safe head.
    ///
    /// This method advances the derivation pipeline to produce the next set of
    /// [`OpAttributesWithParent`] that can be used for block building.
    async fn produce_payload(
        &mut self,
        l2_safe_head: L2BlockInfo,
    ) -> Result<OpAttributesWithParent, PipelineErrorKind> {
        loop {
            match self.step(l2_safe_head).await {
                Ok(mut attrs) if !attrs.is_empty() => {
                    info!(target: "client_derivation_driver", "Stepped derivation pipeline");
                    return Ok(attrs.swap_remove(0));
                }
                Ok(_) => {
                    info!(
                        target: "client_derivation_driver",
                        l1_block_number = self.origin().map(|o| o.number).ok_or(PipelineError::MissingOrigin.crit())?,
                        "Advanced origin"
                    );
                }
                Err(PipelineErrorKind::Temporary(_)) => {
                    trace!(target: "client_derivation_driver", "Failed to step derivation pipeline temporarily");
                    continue;
                }
                Err(PipelineErrorKind::Reset(e)) => {
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
                        if matches!(e, ResetError::ReorgDetected(_, _)) {
                            self.flush();
                        }

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
                Err(e @ PipelineErrorKind::Critical(_)) => {
                    warn!(target: "client_derivation_driver", "Failed to step derivation pipeline: {:?}", e);
                    return Err(e);
                }
            }
        }
    }
}
