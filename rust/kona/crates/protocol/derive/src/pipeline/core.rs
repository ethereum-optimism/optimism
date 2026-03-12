//! Contains the core derivation pipeline.

use crate::{
    ActivationSignal, L2ChainProvider, NextAttributes, OriginAdvancer, OriginProvider, Pipeline,
    PipelineError, PipelineErrorKind, PipelineResult, ResetSignal, Signal, SignalReceiver,
};
use alloc::{boxed::Box, sync::Arc, vec::Vec};
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};

/// The derivation pipeline is responsible for deriving L2 inputs from L1 data.
#[derive(Debug)]
pub struct DerivationPipeline<S, P>
where
    S: NextAttributes + SignalReceiver + OriginProvider + OriginAdvancer + Debug + Send,
    P: L2ChainProvider + Send + Sync + Debug,
{
    /// A handle to the next attributes.
    pub attributes: S,
    /// The rollup config.
    pub rollup_config: Arc<RollupConfig>,
    /// The L2 Chain Provider used to fetch the system config on reset.
    pub l2_chain_provider: P,
}

impl<S, P> DerivationPipeline<S, P>
where
    S: NextAttributes + SignalReceiver + OriginProvider + OriginAdvancer + Debug + Send,
    P: L2ChainProvider + Send + Sync + Debug,
{
    /// Creates a new instance of the [`DerivationPipeline`].
    pub const fn new(
        attributes: S,
        rollup_config: Arc<RollupConfig>,
        l2_chain_provider: P,
    ) -> Self {
        Self { attributes, rollup_config, l2_chain_provider }
    }
}

impl<S, P> OriginProvider for DerivationPipeline<S, P>
where
    S: NextAttributes + SignalReceiver + OriginProvider + OriginAdvancer + Debug + Send,
    P: L2ChainProvider + Send + Sync + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.attributes.origin()
    }
}

#[async_trait]
impl<S, P> SignalReceiver for DerivationPipeline<S, P>
where
    S: NextAttributes + SignalReceiver + OriginProvider + OriginAdvancer + Debug + Send + Sync,
    P: L2ChainProvider + Send + Sync + Debug,
{
    /// Signals the pipeline by calling the [`SignalReceiver::signal`] method.
    ///
    /// During a [`Signal::Reset`], each stage is recursively called from the top-level
    /// [`crate::stages::AttributesQueue`] to the bottom [`crate::PollingTraversal`]
    /// with a head-recursion pattern. This effectively clears the internal state
    /// of each stage in the pipeline from bottom on up.
    ///
    /// [`Signal::Activation`] does a similar thing to the reset, with different
    /// holocene-specific reset rules.
    ///
    /// ### Parameters
    ///
    /// The `signal` is contains the signal variant with any necessary parameters.
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match signal {
            mut s @ (Signal::Reset(ResetSignal { l2_safe_head, .. }) |
            Signal::Activation(ActivationSignal { l2_safe_head, .. })) => {
                let system_config = self
                    .l2_chain_provider
                    .system_config_by_number(
                        l2_safe_head.block_info.number,
                        Arc::clone(&self.rollup_config),
                    )
                    .await
                    .map_err(Into::into)?;
                s = s.with_system_config(system_config);
                match self.attributes.signal(s).await {
                    Ok(()) => trace!(target: "pipeline", "Stages reset"),
                    Err(err) => {
                        if err == PipelineErrorKind::Temporary(PipelineError::Eof) {
                            trace!(target: "pipeline", "Stages reset with EOF");
                        } else {
                            error!(target: "pipeline", "Stage reset errored: {:?}", err);
                            return Err(err);
                        }
                    }
                }
            }
            Signal::FlushChannel | Signal::ProvideBlock(_) => {
                self.attributes.signal(signal).await?;
            }
        }
        kona_macros::inc!(
            gauge,
            crate::metrics::Metrics::PIPELINE_SIGNALS,
            "type" => signal.to_string(),
        );
        Ok(())
    }
}

#[async_trait]
impl<S, P> Pipeline for DerivationPipeline<S, P>
where
    S: NextAttributes + SignalReceiver + OriginProvider + OriginAdvancer + Debug + Send + Sync,
    P: L2ChainProvider + Send + Sync + Debug,
{
    /// Returns the rollup config.
    fn rollup_config(&self) -> &RollupConfig {
        &self.rollup_config
    }

    /// Returns the [`SystemConfig`] by L2 number.
    async fn system_config_by_number(
        &mut self,
        number: u64,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        self.l2_chain_provider
            .system_config_by_number(number, self.rollup_config.clone())
            .await
            .map_err(Into::into)
    }

    /// Attempts to progress the pipeline by one step.
    ///
    /// Calls `next_attributes` once. On success, returns a single-element vec.
    /// On EOF, advances the L1 origin and returns an empty vec.
    /// On any other error, returns the error for the caller to handle.
    async fn step(
        &mut self,
        cursor: L2BlockInfo,
    ) -> PipelineResult<Vec<OpAttributesWithParent>> {
        kona_macros::inc!(gauge, crate::metrics::Metrics::PIPELINE_STEPS);
        kona_macros::set!(
            gauge,
            crate::metrics::Metrics::PIPELINE_STEP_BLOCK,
            cursor.block_info.number as f64
        );

        match self.attributes.next_attributes(cursor).await {
            Ok(a) => {
                trace!(target: "pipeline", "Prepared L2 attributes: {:?}", a);
                kona_macros::inc!(
                    gauge,
                    crate::metrics::Metrics::PIPELINE_PAYLOAD_ATTRIBUTES_BUFFER
                );
                kona_macros::set!(
                    gauge,
                    crate::metrics::Metrics::PIPELINE_LATEST_PAYLOAD_TX_COUNT,
                    a.attributes.transactions.as_ref().map_or(0.0, |txs| txs.len() as f64)
                );
                if a.is_last_in_span {
                    kona_macros::set!(
                        gauge,
                        crate::metrics::Metrics::PIPELINE_DERIVED_SPAN_SIZE,
                        0
                    );
                } else {
                    kona_macros::inc!(
                        gauge,
                        crate::metrics::Metrics::PIPELINE_DERIVED_SPAN_SIZE
                    );
                }
                kona_macros::inc!(
                    gauge,
                    crate::metrics::Metrics::PIPELINE_PREPARED_ATTRIBUTES
                );
                Ok(alloc::vec![a])
            }
            Err(PipelineErrorKind::Temporary(PipelineError::Eof)) => {
                trace!(target: "pipeline", "Pipeline advancing origin");
                self.attributes.advance_origin().await?;
                Ok(Vec::new())
            }
            Err(err) => Err(err),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{DerivationPipeline, test_utils::*};
    use alloc::{string::ToString, sync::Arc};
    use alloy_rpc_types_engine::PayloadAttributes;
    use kona_genesis::{RollupConfig, SystemConfig};
    use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;

    fn default_test_payload_attributes() -> OpAttributesWithParent {
        OpAttributesWithParent {
            attributes: OpPayloadAttributes {
                payload_attributes: PayloadAttributes {
                    timestamp: 0,
                    prev_randao: Default::default(),
                    suggested_fee_recipient: Default::default(),
                    withdrawals: None,
                    parent_beacon_block_root: None,
                },
                transactions: None,
                no_tx_pool: None,
                gas_limit: None,
                eip_1559_params: None,
                min_base_fee: None,
            },
            parent: Default::default(),
            derived_from: Default::default(),
            is_last_in_span: false,
        }
    }

    #[tokio::test]
    async fn test_derivation_pipeline_missing_block() {
        let mut pipeline = new_test_pipeline();
        let cursor = L2BlockInfo::default();
        let result = pipeline.step(cursor).await;
        assert_eq!(
            result,
            Err(PipelineError::Provider("Block not found".to_string()).temp())
        );
    }

    #[tokio::test]
    async fn test_derivation_pipeline_prepared_attributes() {
        let rollup_config = Arc::new(RollupConfig::default());
        let l2_chain_provider = TestL2ChainProvider::default();
        let expected = default_test_payload_attributes();
        let attributes = TestNextAttributes { next_attributes: Some(expected.clone()) };
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        // Step on the pipeline and expect the result.
        let cursor = L2BlockInfo::default();
        let result = pipeline.step(cursor).await;
        // TestNextAttributes returns one attr then EOF; advance_origin on the default
        // test provider fails, so we get an error. But the attr was collected.
        // Since the test provider's advance_origin succeeds (returns Ok(())),
        // we should get Ok with the attrs.
        assert!(result.is_ok());
        let attrs = result.unwrap();
        assert_eq!(attrs.len(), 1);
        assert_eq!(attrs[0], expected);
    }

    #[tokio::test]
    async fn test_derivation_pipeline_advance_origin() {
        let rollup_config = Arc::new(RollupConfig::default());
        let l2_chain_provider = TestL2ChainProvider::default();
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        // Step on the pipeline and expect the result — no attrs, just origin advance.
        let cursor = L2BlockInfo::default();
        let result = pipeline.step(cursor).await;
        assert!(result.is_ok());
        assert!(result.unwrap().is_empty());
    }

    #[tokio::test]
    async fn test_derivation_pipeline_signal_activation() {
        let rollup_config = Arc::new(RollupConfig::default());
        let mut l2_chain_provider = TestL2ChainProvider::default();
        l2_chain_provider.system_configs.insert(0, SystemConfig::default());
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        // Signal the pipeline to reset.
        let result = pipeline.signal(ActivationSignal::default().signal()).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_derivation_pipeline_flush_channel() {
        let rollup_config = Arc::new(RollupConfig::default());
        let l2_chain_provider = TestL2ChainProvider::default();
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        // Signal the pipeline to reset.
        let result = pipeline.signal(Signal::FlushChannel).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_derivation_pipeline_signal_reset_missing_sys_config() {
        let rollup_config = Arc::new(RollupConfig::default());
        let l2_chain_provider = TestL2ChainProvider::default();
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        // Signal the pipeline to reset.
        let result = pipeline.signal(ResetSignal::default().signal()).await.unwrap_err();
        assert_eq!(result, PipelineError::Provider("System config not found".to_string()).temp());
    }

    #[tokio::test]
    async fn test_derivation_pipeline_signal_reset_ok() {
        let rollup_config = Arc::new(RollupConfig::default());
        let mut l2_chain_provider = TestL2ChainProvider::default();
        l2_chain_provider.system_configs.insert(0, SystemConfig::default());
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        // Signal the pipeline to reset.
        let result = pipeline.signal(ResetSignal::default().signal()).await;
        assert!(result.is_ok());
    }
}
