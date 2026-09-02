//! Contains the core derivation pipeline.

use crate::{
    ActivationSignal, L2ChainProvider, NextAttributes, OriginAdvancer, OriginProvider, Pipeline,
    PipelineError, PipelineErrorKind, PipelineResult, ResetSignal, Signal, SignalReceiver, Stage,
    StepResult,
};
use alloc::{boxed::Box, collections::VecDeque, sync::Arc};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};

/// The derivation pipeline is responsible for deriving L2 inputs from L1 data.
#[derive(Debug)]
pub struct DerivationPipeline<S, P>
where
    S: NextAttributes + Stage + OriginProvider + OriginAdvancer + Debug + Send,
    P: L2ChainProvider + Send + Sync + Debug,
{
    /// A handle to the next attributes.
    pub attributes: S,
    /// Reset provider for the pipeline.
    /// A list of prepared [`OpAttributesWithParent`] to be used by the derivation pipeline
    /// consumer.
    pub prepared: VecDeque<OpAttributesWithParent>,
    /// The rollup config.
    pub rollup_config: Arc<RollupConfig>,
    /// The L2 Chain Provider used to fetch the system config on reset.
    pub l2_chain_provider: P,
}

impl<S, P> DerivationPipeline<S, P>
where
    S: NextAttributes + Stage + OriginProvider + OriginAdvancer + Debug + Send,
    P: L2ChainProvider + Send + Sync + Debug,
{
    /// Creates a new instance of the [`DerivationPipeline`].
    pub const fn new(
        attributes: S,
        rollup_config: Arc<RollupConfig>,
        l2_chain_provider: P,
    ) -> Self {
        Self { attributes, prepared: VecDeque::new(), rollup_config, l2_chain_provider }
    }

    /// Walks back the L2 chain to find the correct L1 origin for a pipeline reset.
    /// This mirrors op-node's `initialReset` algorithm.
    pub(crate) async fn initial_reset(
        &mut self,
        l2_safe_head: L2BlockInfo,
    ) -> Result<(BlockNumHash, SystemConfig), PipelineErrorKind> {
        let l1_origin_number = l2_safe_head.l1_origin.number;
        let channel_timeout = self.rollup_config.channel_timeout(l2_safe_head.block_info.timestamp);

        let mut current = l2_safe_head;
        loop {
            let before_l2_genesis =
                current.block_info.number <= self.rollup_config.genesis.l2.number;
            let before_l1_genesis =
                current.l1_origin.number <= self.rollup_config.genesis.l1.number;
            let before_channel_timeout =
                current.l1_origin.number + channel_timeout <= l1_origin_number;
            if before_l2_genesis || before_l1_genesis || before_channel_timeout {
                break;
            }

            current = self
                .l2_chain_provider
                .l2_block_info_by_hash(current.block_info.parent_hash)
                .await
                .map_err(Into::<PipelineErrorKind>::into)?;
        }

        let system_config = self
            .l2_chain_provider
            .system_config_by_l2_hash(current.block_info.hash, Arc::clone(&self.rollup_config))
            .await
            .map_err(|e| PipelineError::Provider(alloc::string::ToString::to_string(&e)).temp())?;

        Ok((current.l1_origin, system_config))
    }
}

impl<S, P> OriginProvider for DerivationPipeline<S, P>
where
    S: NextAttributes + Stage + OriginProvider + OriginAdvancer + Debug + Send,
    P: L2ChainProvider + Send + Sync + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.attributes.origin()
    }
}

impl<S, P> Iterator for DerivationPipeline<S, P>
where
    S: NextAttributes + Stage + OriginProvider + OriginAdvancer + Debug + Send + Sync,
    P: L2ChainProvider + Send + Sync + Debug,
{
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        kona_macros::set!(
            gauge,
            crate::metrics::Metrics::PIPELINE_PAYLOAD_ATTRIBUTES_BUFFER,
            self.prepared.len().saturating_sub(1) as f64
        );
        self.prepared.pop_front()
    }
}

#[async_trait]
impl<S, P> SignalReceiver for DerivationPipeline<S, P>
where
    S: NextAttributes + Stage + OriginProvider + OriginAdvancer + Debug + Send + Sync,
    P: L2ChainProvider + Send + Sync + Debug,
{
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match signal {
            Signal::Reset(ResetSignal { l2_safe_head }) => {
                let (l1_origin, system_config) = self.initial_reset(l2_safe_head).await?;
                match self.attributes.reset(l1_origin, system_config).await {
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
            Signal::Activation(ActivationSignal { .. }) => {
                // Activation is a soft reset for hardfork boundaries. It clears
                // buffered data but preserves derivation state. No walkback needed.
                match self.attributes.activate().await {
                    Ok(()) => trace!(target: "pipeline", "Stages activated"),
                    Err(err) => {
                        if err == PipelineErrorKind::Temporary(PipelineError::Eof) {
                            trace!(target: "pipeline", "Stages activated with EOF");
                        } else {
                            error!(target: "pipeline", "Stage activation errored: {:?}", err);
                            return Err(err);
                        }
                    }
                }
            }
            Signal::FlushChannel => {
                self.attributes.flush_channel().await?;
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
    S: NextAttributes + Stage + OriginProvider + OriginAdvancer + Debug + Send + Sync,
    P: L2ChainProvider + Send + Sync + Debug,
{
    /// Peeks at the next prepared [`OpAttributesWithParent`] from the pipeline.
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        self.prepared.front()
    }

    /// Returns the rollup config.
    fn rollup_config(&self) -> &RollupConfig {
        &self.rollup_config
    }

    /// Returns the [`SystemConfig`] for the L2 block with the given hash.
    async fn system_config_by_l2_hash(
        &mut self,
        hash: B256,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        self.l2_chain_provider
            .system_config_by_l2_hash(hash, self.rollup_config.clone())
            .await
            .map_err(Into::into)
    }

    /// Attempts to progress the pipeline.
    ///
    /// ## Returns
    ///
    /// A [`PipelineError::Eof`] is returned if the pipeline is blocked by waiting for new L1 data.
    /// Any other error is critical and the derivation pipeline should be reset.
    /// An error is expected when the underlying source closes.
    ///
    /// When [`DerivationPipeline::step`] returns [Ok(())], it should be called again, to continue
    /// the derivation process.
    ///
    /// [`PipelineError`]: crate::errors::PipelineError
    async fn step(&mut self, cursor: L2BlockInfo) -> StepResult {
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
                    kona_macros::inc!(gauge, crate::metrics::Metrics::PIPELINE_DERIVED_SPAN_SIZE);
                }
                self.prepared.push_back(a);
                kona_macros::inc!(gauge, crate::metrics::Metrics::PIPELINE_PREPARED_ATTRIBUTES);
                StepResult::PreparedAttributes
            }
            Err(err) => match err {
                PipelineErrorKind::Temporary(PipelineError::Eof) => {
                    trace!(target: "pipeline", "Pipeline advancing origin");
                    if let Err(e) = self.attributes.advance_origin().await {
                        return StepResult::OriginAdvanceErr(e);
                    }
                    StepResult::AdvancedOrigin
                }
                PipelineErrorKind::Temporary(_) => {
                    trace!(target: "pipeline", "Attributes queue step failed due to temporary error: {:?}", err);
                    StepResult::StepFailed(err)
                }
                _ => {
                    warn!(target: "pipeline", "Attributes queue step failed: {:?}", err);
                    StepResult::StepFailed(err)
                }
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{DerivationPipeline, PipelineBuilder, ResetError, test_utils::*};
    use alloc::{string::ToString, sync::Arc, vec, vec::Vec};
    use alloy_rpc_types_engine::PayloadAttributes;
    use kona_genesis::{RollupConfig, SystemConfig};
    use kona_protocol::{Batch, L2BlockInfo, OpAttributesWithParent, SingleBatch, SpanBatch};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;

    /// A distinct hash per block number, so hash-keyed lookups address one specific block.
    fn test_block_hash(number: u64) -> B256 {
        B256::from(alloy_primitives::U256::from(number))
    }

    fn default_test_payload_attributes() -> OpAttributesWithParent {
        OpAttributesWithParent {
            attributes: OpPayloadAttributes {
                payload_attributes: PayloadAttributes {
                    timestamp: 0,
                    prev_randao: Default::default(),
                    suggested_fee_recipient: Default::default(),
                    withdrawals: None,
                    parent_beacon_block_root: None,
                    slot_number: None,
                    target_gas_limit: None,
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

    /// The L2 chain the multi-block pipeline test derives: one block at 1002, a group of three
    /// at 1004 and a group of two at 1006, all on L1 origin 99.
    const MULTI_BLOCK_TIMESTAMPS: [u64; 7] = [1000, 1002, 1004, 1004, 1004, 1006, 1006];

    /// The L1 block the multi-block chain is sequenced against.
    const MULTI_BLOCK_ORIGIN: u64 = 99;

    /// The L1 block the batcher includes the channel in, two blocks after the origin so that
    /// derivation has moved past that origin by the time it reads the batches.
    const MULTI_BLOCK_INCLUSION: u64 = 101;

    fn multi_block_rollup_config() -> Arc<RollupConfig> {
        Arc::new(RollupConfig {
            block_time: 2,
            seq_window_size: 100,
            max_sequencer_drift: 1000,
            genesis: kona_genesis::ChainGenesis {
                l1: BlockNumHash { number: 98, hash: test_block_hash(98) },
                l2: BlockNumHash { number: 0, hash: test_block_hash(1000) },
                l2_time: 1000,
                ..Default::default()
            },
            hardforks: kona_genesis::HardForkConfig {
                delta_time: Some(0),
                holocene_time: Some(0),
                karst_time: Some(0),
                ..Default::default()
            },
            multi_block_time: Some(1000),
            max_multi_blocks: Some(3),
            ..Default::default()
        })
    }

    fn multi_block_l1_block(number: u64) -> BlockInfo {
        BlockInfo {
            number,
            timestamp: 800 + number * 2,
            hash: test_block_hash(number),
            parent_hash: test_block_hash(number - 1),
        }
    }

    fn multi_block_l2_block(number: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                number,
                timestamp: MULTI_BLOCK_TIMESTAMPS[number as usize],
                hash: test_block_hash(1000 + number),
                parent_hash: test_block_hash(1000 + number.saturating_sub(1)),
            },
            l1_origin: BlockNumHash {
                number: MULTI_BLOCK_ORIGIN,
                hash: test_block_hash(MULTI_BLOCK_ORIGIN),
            },
            seq_num: number,
        }
    }

    /// Builds a span batch v2 over the L2 blocks `numbers`, taking each element's sibling flag
    /// from the chain the test derives.
    fn multi_block_span(numbers: &[u64]) -> SpanBatch {
        let mut span = SpanBatch {
            version: kona_protocol::BatchType::SpanV2,
            genesis_timestamp: 1000,
            ..Default::default()
        };
        for number in numbers {
            let parent = multi_block_l2_block(number - 1);
            span.append_singular_batch(
                SingleBatch {
                    parent_hash: parent.block_info.hash,
                    epoch_num: MULTI_BLOCK_ORIGIN,
                    epoch_hash: test_block_hash(MULTI_BLOCK_ORIGIN),
                    timestamp: MULTI_BLOCK_TIMESTAMPS[*number as usize],
                    transactions: vec![],
                },
                *number,
                parent.block_info.timestamp,
            )
            .unwrap();
        }
        span
    }

    /// Encodes the spans into a single channel, framed as one batcher transaction payload.
    fn multi_block_channel(spans: [SpanBatch; 2]) -> alloy_primitives::Bytes {
        let mut channel = Vec::new();
        for span in spans {
            let mut encoded = Vec::new();
            Batch::Span(span).encode(&mut encoded).unwrap();
            alloy_rlp::Encodable::encode(&alloy_primitives::Bytes::from(encoded), &mut channel);
        }
        let compressed = miniz_oxide::deflate::compress_to_vec_zlib(
            &channel,
            miniz_oxide::deflate::CompressionLevel::BestSpeed as u8,
        );
        let frame =
            kona_protocol::Frame { id: [0xAB; 16], number: 0, data: compressed, is_last: true };
        let mut payload = vec![kona_protocol::DERIVATION_VERSION_0];
        payload.extend_from_slice(&frame.encode());
        payload.into()
    }

    /// A data source holding one batcher payload, served at a single L1 block.
    #[derive(Debug, Clone)]
    struct TestBlockDAP {
        number: u64,
        payload: Option<alloy_primitives::Bytes>,
    }

    #[async_trait]
    impl crate::DataAvailabilityProvider for TestBlockDAP {
        type Item = alloy_primitives::Bytes;

        async fn next(
            &mut self,
            block: &BlockInfo,
            _: alloy_primitives::Address,
        ) -> PipelineResult<Self::Item> {
            if block.number != self.number {
                return Err(PipelineError::Eof.temp());
            }
            self.payload.take().ok_or(PipelineError::Eof.temp())
        }

        fn clear(&mut self) {}
    }

    /// Six blocks in three groups, cut into two spans in the middle of the group of three, derive
    /// to six sets of attributes with the group's timestamps.
    #[tokio::test]
    async fn test_derivation_pipeline_multi_block_groups() {
        let mut chain_provider = TestChainProvider::default();
        for number in 98..=120 {
            let block = multi_block_l1_block(number);
            chain_provider.insert_block(number, block);
            chain_provider.insert_receipts(block.hash, Vec::new());
        }

        let mut l2_chain_provider = TestL2ChainProvider::default();
        for number in 0..=6 {
            l2_chain_provider.blocks.push(multi_block_l2_block(number));
        }

        // The group of three at timestamp 1004 straddles the two spans.
        let dap = TestBlockDAP {
            number: MULTI_BLOCK_INCLUSION,
            payload: Some(multi_block_channel([
                multi_block_span(&[1, 2, 3]),
                multi_block_span(&[4, 5, 6]),
            ])),
        };

        let mut pipeline = PipelineBuilder::new()
            .rollup_config(multi_block_rollup_config())
            .origin(multi_block_l1_block(MULTI_BLOCK_ORIGIN))
            .dap_source(dap)
            .builder(TestAttributesBuilder {
                attributes: (0..6).map(|_| Ok(Default::default())).collect(),
                ..Default::default()
            })
            .chain_provider(chain_provider)
            .l2_chain_provider(l2_chain_provider)
            .build_polled();

        let mut cursor = multi_block_l2_block(0);
        let mut derived = Vec::new();
        for _ in 0..64 {
            if pipeline.step(cursor).await == StepResult::PreparedAttributes {
                pipeline.next().expect("prepared attributes are queued");
                derived.push(cursor.block_info.number + 1);
                cursor = multi_block_l2_block(cursor.block_info.number + 1);
            }
            if derived.len() == 6 {
                break;
            }
        }

        assert_eq!(derived, vec![1, 2, 3, 4, 5, 6]);
        assert_eq!(
            pipeline.attributes.builder.timestamps,
            vec![1002, 1004, 1004, 1004, 1006, 1006]
        );
    }

    #[test]
    fn test_pipeline_next_attributes_empty() {
        let mut pipeline = new_test_pipeline();
        let result = pipeline.next();
        assert_eq!(result, None);
    }

    #[test]
    fn test_pipeline_next_attributes_with_peek() {
        let mut pipeline = new_test_pipeline();
        let expected = default_test_payload_attributes();
        pipeline.prepared.push_back(expected.clone());

        let result = pipeline.peek();
        assert_eq!(result, Some(&expected));

        let result = pipeline.next();
        assert_eq!(result, Some(expected));
    }

    #[tokio::test]
    async fn test_derivation_pipeline_missing_block() {
        let mut pipeline = new_test_pipeline();
        let cursor = L2BlockInfo::default();
        let result = pipeline.step(cursor).await;
        assert_eq!(
            result,
            StepResult::OriginAdvanceErr(
                PipelineError::Provider("Block not found".to_string()).temp()
            )
        );
    }

    #[tokio::test]
    async fn test_derivation_pipeline_prepared_attributes() {
        let rollup_config = Arc::new(RollupConfig::default());
        let l2_chain_provider = TestL2ChainProvider::default();
        let expected = default_test_payload_attributes();
        let attributes = TestNextAttributes { next_attributes: Some(expected) };
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        // Step on the pipeline and expect the result.
        let cursor = L2BlockInfo::default();
        let result = pipeline.step(cursor).await;
        assert_eq!(result, StepResult::PreparedAttributes);
    }

    #[tokio::test]
    async fn test_derivation_pipeline_advance_origin() {
        let rollup_config = Arc::new(RollupConfig::default());
        let l2_chain_provider = TestL2ChainProvider::default();
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        // Step on the pipeline and expect the result.
        let cursor = L2BlockInfo::default();
        let result = pipeline.step(cursor).await;
        assert_eq!(result, StepResult::AdvancedOrigin);
    }

    #[tokio::test]
    async fn test_derivation_pipeline_signal_activation() {
        let rollup_config = Arc::new(RollupConfig::default());
        let mut l2_chain_provider = TestL2ChainProvider::default();
        l2_chain_provider.system_configs.insert(B256::ZERO, SystemConfig::default());
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        let result = pipeline.signal(Signal::Activation(ActivationSignal::default())).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_derivation_pipeline_flush_channel() {
        let rollup_config = Arc::new(RollupConfig::default());
        let l2_chain_provider = TestL2ChainProvider::default();
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        let result = pipeline.signal(Signal::FlushChannel).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_derivation_pipeline_signal_reset_missing_sys_config() {
        let rollup_config = Arc::new(RollupConfig::default());
        let l2_chain_provider = TestL2ChainProvider::default();
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        let result = pipeline.signal(Signal::Reset(ResetSignal::default())).await.unwrap_err();
        assert_eq!(result, PipelineError::Provider("System config not found".to_string()).temp());
    }

    #[tokio::test]
    async fn test_derivation_pipeline_signal_reset_ok() {
        let rollup_config = Arc::new(RollupConfig::default());
        let mut l2_chain_provider = TestL2ChainProvider::default();
        l2_chain_provider.system_configs.insert(B256::ZERO, SystemConfig::default());
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        let result = pipeline.signal(Signal::Reset(ResetSignal::default())).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_initial_reset_walks_back_system_config() {
        use alloy_primitives::address;

        let rollup_config = RollupConfig { channel_timeout: 10, ..Default::default() };

        let mut l2_chain_provider = TestL2ChainProvider::default();
        // L2 blocks 89..=100: block N has L1 origin (N - 50).
        // Safe head at block 100, L1 origin 50.
        // Walkback: block 90 has L1 origin 40. 40 + 10 = 50, NOT > 50, so walkback stops.
        for n in 89u64..=100 {
            l2_chain_provider.blocks.push(L2BlockInfo {
                block_info: BlockInfo {
                    hash: test_block_hash(n),
                    number: n,
                    parent_hash: test_block_hash(n - 1),
                    ..Default::default()
                },
                l1_origin: BlockNumHash { number: n - 50, ..Default::default() },
                seq_num: 0,
            });
        }

        // Old batcher at walked-back block 90.
        l2_chain_provider.system_configs.insert(
            test_block_hash(90),
            SystemConfig {
                batcher_address: address!("000000000000000000000000000000000000aaaa"),
                ..Default::default()
            },
        );
        // New batcher at safe head block 100.
        l2_chain_provider.system_configs.insert(
            test_block_hash(100),
            SystemConfig {
                batcher_address: address!("000000000000000000000000000000000000bbbb"),
                ..Default::default()
            },
        );

        let rollup_config = Arc::new(rollup_config);
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                hash: test_block_hash(100),
                number: 100,
                parent_hash: test_block_hash(99),
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 50, ..Default::default() },
            seq_num: 0,
        };

        let (l1_origin, sys_cfg) = pipeline.initial_reset(l2_safe_head).await.unwrap();
        assert_eq!(l1_origin.number, 40);
        assert_eq!(
            sys_cfg.batcher_address,
            address!("000000000000000000000000000000000000aaaa"),
            "Expected old batcher from walked-back block 90"
        );
    }

    #[tokio::test]
    async fn test_initial_reset_preserves_parent_lookup_error_kind() {
        let rollup_config = Arc::new(RollupConfig { channel_timeout: 1, ..Default::default() });
        let l2_chain_provider = TestL2ChainProvider::default();
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        let parent_hash = test_block_hash(1);
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                hash: test_block_hash(2),
                number: 2,
                parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 2, ..Default::default() },
            seq_num: 0,
        };

        let err = pipeline.initial_reset(l2_safe_head).await.unwrap_err();
        assert_eq!(err, ResetError::BlockNotFound(parent_hash.into()).reset());
    }

    #[tokio::test]
    async fn test_initial_reset_respects_genesis() {
        let rollup_config = RollupConfig {
            channel_timeout: 100,
            genesis: kona_genesis::ChainGenesis {
                l2: alloy_eips::BlockNumHash { number: 5, ..Default::default() },
                l1: alloy_eips::BlockNumHash { number: 3, ..Default::default() },
                ..Default::default()
            },
            ..Default::default()
        };

        let mut l2_chain_provider = TestL2ChainProvider::default();
        l2_chain_provider.blocks.push(L2BlockInfo {
            block_info: BlockInfo {
                hash: test_block_hash(5),
                number: 5,
                parent_hash: test_block_hash(4),
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 3, ..Default::default() },
            seq_num: 0,
        });
        l2_chain_provider.blocks.push(L2BlockInfo {
            block_info: BlockInfo {
                hash: test_block_hash(6),
                number: 6,
                parent_hash: test_block_hash(5),
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 4, ..Default::default() },
            seq_num: 0,
        });
        l2_chain_provider.system_configs.insert(test_block_hash(5), SystemConfig::default());

        let rollup_config = Arc::new(rollup_config);
        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                hash: test_block_hash(6),
                number: 6,
                parent_hash: test_block_hash(5),
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 4, ..Default::default() },
            seq_num: 0,
        };

        let (l1_origin, _) = pipeline.initial_reset(l2_safe_head).await.unwrap();
        assert_eq!(l1_origin.number, 3, "Should stop at genesis L1 origin");
    }

    #[tokio::test]
    async fn test_initial_reset_no_walkback_zero_timeout() {
        let rollup_config = Arc::new(RollupConfig::default());
        let mut l2_chain_provider = TestL2ChainProvider::default();
        l2_chain_provider.system_configs.insert(B256::ZERO, SystemConfig::default());

        let attributes = TestNextAttributes::default();
        let mut pipeline = DerivationPipeline::new(attributes, rollup_config, l2_chain_provider);

        let (l1_origin, sys_cfg) = pipeline.initial_reset(L2BlockInfo::default()).await.unwrap();
        assert_eq!(l1_origin.number, 0);
        assert_eq!(sys_cfg, SystemConfig::default());
    }
}
