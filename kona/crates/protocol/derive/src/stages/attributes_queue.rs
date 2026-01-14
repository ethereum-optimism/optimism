//! Contains the logic for the `AttributesQueue` stage.

use crate::{
    errors::{PipelineError, ResetError},
    traits::{
        AttributesBuilder, AttributesProvider, NextAttributes, OriginAdvancer, OriginProvider,
        SignalReceiver,
    },
    types::{PipelineResult, Signal},
};
use alloc::{boxed::Box, sync::Arc};
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent, SingleBatch};
use op_alloy_rpc_types_engine::OpPayloadAttributes;

/// [`AttributesQueue`] accepts batches from the [`BatchQueue`] stage
/// and transforms them into [`OpPayloadAttributes`].
///
/// The outputted payload attributes cannot be buffered because each batch->attributes
/// transformation pulls in data about the current L2 safe head.
///
/// [`AttributesQueue`] also buffers batches that have been output because
/// multiple batches can be created at once.
///
/// This stage can be reset by clearing its batch buffer.
/// This stage does not need to retain any references to L1 blocks.
///
/// [`BatchQueue`]: crate::stages::BatchQueue
#[derive(Debug)]
pub struct AttributesQueue<P, AB>
where
    P: AttributesProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
    AB: AttributesBuilder + Debug,
{
    /// The rollup config.
    pub cfg: Arc<RollupConfig>,
    /// The previous stage of the derivation pipeline.
    pub prev: P,
    /// Whether the current batch is the last in its span.
    pub is_last_in_span: bool,
    /// The current batch being processed.
    pub batch: Option<SingleBatch>,
    /// The attributes builder.
    pub builder: AB,
}

impl<P, AB> AttributesQueue<P, AB>
where
    P: AttributesProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
    AB: AttributesBuilder + Debug,
{
    /// Create a new [`AttributesQueue`] stage.
    pub const fn new(cfg: Arc<RollupConfig>, prev: P, builder: AB) -> Self {
        Self { cfg, prev, is_last_in_span: false, batch: None, builder }
    }

    /// Loads a [`SingleBatch`] from the [`AttributesProvider`] if needed.
    pub async fn load_batch(&mut self, parent: L2BlockInfo) -> PipelineResult<SingleBatch> {
        if self.batch.is_none() {
            let batch = self.prev.next_batch(parent).await?;
            self.batch = Some(batch);
            self.is_last_in_span = self.prev.is_last_in_span();
        }
        self.batch.as_ref().cloned().ok_or(PipelineError::Eof.temp())
    }

    /// Returns the next [`OpAttributesWithParent`] from the current batch.
    pub async fn next_attributes(
        &mut self,
        parent: L2BlockInfo,
    ) -> PipelineResult<OpAttributesWithParent> {
        let batch = match self.load_batch(parent).await {
            Ok(batch) => batch,
            Err(e) => {
                return Err(e);
            }
        };

        // Construct the payload attributes from the loaded batch.
        #[cfg(feature = "metrics")]
        let start = std::time::Instant::now();
        let attributes = match self.create_next_attributes(batch, parent).await {
            Ok(attributes) => attributes,
            Err(e) => {
                return Err(e);
            }
        };
        let origin = self.origin().ok_or(PipelineError::MissingOrigin.crit())?;
        let populated_attributes =
            OpAttributesWithParent::new(attributes, parent, Some(origin), self.is_last_in_span);
        kona_macros::record!(
            histogram,
            crate::metrics::Metrics::PIPELINE_ATTRIBUTES_BUILD_DURATION,
            start.elapsed().as_secs_f64()
        );

        // Clear out the local state once payload attributes are prepared.
        self.batch = None;
        self.is_last_in_span = false;
        Ok(populated_attributes)
    }

    /// Creates the next attributes, transforming a [`SingleBatch`] into [`OpPayloadAttributes`].
    /// This sets `no_tx_pool` and appends the batched txs to the attributes tx list.
    pub async fn create_next_attributes(
        &mut self,
        batch: SingleBatch,
        parent: L2BlockInfo,
    ) -> PipelineResult<OpPayloadAttributes> {
        // Sanity check parent hash
        if batch.parent_hash != parent.block_info.hash {
            return Err(ResetError::BadParentHash(batch.parent_hash, parent.block_info.hash).into());
        }

        // Sanity check timestamp
        let actual = parent.block_info.timestamp + self.cfg.block_time;
        if actual != batch.timestamp {
            return Err(ResetError::BadTimestamp(batch.timestamp, actual).into());
        }

        // Prepare the payload attributes
        let tx_count = batch.transactions.len();
        let mut attributes = self.builder.prepare_payload_attributes(parent, batch.epoch()).await?;
        attributes.no_tx_pool = Some(true);
        match attributes.transactions {
            Some(ref mut txs) => txs.extend(batch.transactions),
            None => {
                if !batch.transactions.is_empty() {
                    attributes.transactions = Some(batch.transactions);
                }
            }
        }

        info!(
            target: "attributes_queue",
            txs = tx_count,
            timestamp = batch.timestamp,
            "generated attributes in payload queue",
        );

        Ok(attributes)
    }
}

#[async_trait]
impl<P, AB> OriginAdvancer for AttributesQueue<P, AB>
where
    P: AttributesProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug + Send,
    AB: AttributesBuilder + Debug + Send,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.prev.advance_origin().await
    }
}

#[async_trait]
impl<P, AB> NextAttributes for AttributesQueue<P, AB>
where
    P: AttributesProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug + Send,
    AB: AttributesBuilder + Debug + Send,
{
    async fn next_attributes(
        &mut self,
        parent: L2BlockInfo,
    ) -> PipelineResult<OpAttributesWithParent> {
        self.next_attributes(parent).await
    }
}

impl<P, AB> OriginProvider for AttributesQueue<P, AB>
where
    P: AttributesProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
    AB: AttributesBuilder + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.prev.origin()
    }
}

#[async_trait]
impl<P, AB> SignalReceiver for AttributesQueue<P, AB>
where
    P: AttributesProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
    AB: AttributesBuilder + Send + Debug,
{
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match signal {
            s @ Signal::Reset(_) | s @ Signal::Activation(_) => {
                self.prev.signal(s).await?;
                self.batch = None;
                self.is_last_in_span = false;
            }
            s @ Signal::FlushChannel => {
                self.batch = None;
                self.prev.signal(s).await?;
            }
            s @ Signal::ProvideBlock(_) => {
                self.prev.signal(s).await?;
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        errors::{BuilderError, PipelineErrorKind},
        test_utils::{TestAttributesBuilder, TestAttributesProvider, new_test_attributes_provider},
        types::ResetSignal,
    };
    use alloc::{sync::Arc, vec, vec::Vec};
    use alloy_primitives::{Address, B256, Bytes, b256};
    use alloy_rpc_types_engine::PayloadAttributes;

    fn default_optimism_payload_attributes() -> OpPayloadAttributes {
        OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 0,
                suggested_fee_recipient: Address::default(),
                prev_randao: B256::default(),
                withdrawals: None,
                parent_beacon_block_root: None,
            },
            no_tx_pool: Some(false),
            transactions: None,
            gas_limit: None,
            eip_1559_params: None,
            min_base_fee: None,
        }
    }

    fn new_attributes_queue(
        cfg: Option<RollupConfig>,
        origin: Option<BlockInfo>,
        batches: Vec<PipelineResult<SingleBatch>>,
    ) -> AttributesQueue<TestAttributesProvider, TestAttributesBuilder> {
        let cfg = cfg.unwrap_or_default();
        let mock_batch_queue = new_test_attributes_provider(origin, batches);
        let mock_attributes_builder = TestAttributesBuilder::default();
        AttributesQueue::new(Arc::new(cfg), mock_batch_queue, mock_attributes_builder)
    }

    #[tokio::test]
    async fn test_attributes_queue_flush() {
        let mut attributes_queue = new_attributes_queue(None, None, vec![]);
        attributes_queue.batch = Some(SingleBatch::default());
        assert!(!attributes_queue.prev.flushed);
        attributes_queue.signal(Signal::FlushChannel).await.unwrap();
        assert!(attributes_queue.prev.flushed);
        assert!(attributes_queue.batch.is_none());
    }

    #[tokio::test]
    async fn test_attributes_queue_reset() {
        let cfg = RollupConfig::default();
        let mock = new_test_attributes_provider(None, vec![]);
        let mock_builder = TestAttributesBuilder::default();
        let mut aq = AttributesQueue::new(Arc::new(cfg), mock, mock_builder);
        aq.batch = Some(SingleBatch::default());
        assert!(!aq.prev.reset);
        aq.signal(ResetSignal::default().signal()).await.unwrap();
        assert!(aq.batch.is_none());
        assert!(aq.prev.reset);
    }

    #[tokio::test]
    async fn test_load_batch_eof() {
        let mut attributes_queue = new_attributes_queue(None, None, vec![]);
        let parent = L2BlockInfo::default();
        let result = attributes_queue.load_batch(parent).await.unwrap_err();
        assert_eq!(result, PipelineError::Eof.temp());
    }

    #[tokio::test]
    async fn test_load_batch_last_in_span() {
        let mut attributes_queue = new_attributes_queue(None, None, vec![Ok(Default::default())]);
        let parent = L2BlockInfo::default();
        let result = attributes_queue.load_batch(parent).await.unwrap();
        assert_eq!(result, Default::default());
        assert!(attributes_queue.is_last_in_span);
    }

    #[tokio::test]
    async fn test_create_next_attributes_bad_parent_hash() {
        let mut attributes_queue = new_attributes_queue(None, None, vec![]);
        let bad_hash = b256!("6666666666666666666666666666666666666666666666666666666666666666");
        let parent = L2BlockInfo {
            block_info: BlockInfo { hash: bad_hash, ..Default::default() },
            ..Default::default()
        };
        let batch = SingleBatch::default();
        let result = attributes_queue.create_next_attributes(batch, parent).await.unwrap_err();
        assert_eq!(
            result,
            PipelineErrorKind::Reset(ResetError::BadParentHash(Default::default(), bad_hash))
        );
    }

    #[tokio::test]
    async fn test_create_next_attributes_bad_timestamp() {
        let mut attributes_queue = new_attributes_queue(None, None, vec![]);
        let parent = L2BlockInfo::default();
        let batch = SingleBatch { timestamp: 1, ..Default::default() };
        let result = attributes_queue.create_next_attributes(batch, parent).await.unwrap_err();
        assert_eq!(result, PipelineErrorKind::Reset(ResetError::BadTimestamp(1, 0)));
    }

    #[tokio::test]
    async fn test_create_next_attributes_bad_parent_timestamp() {
        let mut attributes_queue = new_attributes_queue(None, None, vec![]);
        let parent = L2BlockInfo {
            block_info: BlockInfo { timestamp: 2, ..Default::default() },
            ..Default::default()
        };
        let batch = SingleBatch { timestamp: 1, ..Default::default() };
        let result = attributes_queue.create_next_attributes(batch, parent).await.unwrap_err();
        assert_eq!(result, PipelineErrorKind::Reset(ResetError::BadTimestamp(1, 2)));
    }

    #[tokio::test]
    async fn test_create_next_attributes_bad_config_timestamp() {
        let cfg = RollupConfig { block_time: 1, ..Default::default() };
        let mut attributes_queue = new_attributes_queue(Some(cfg), None, vec![]);
        let parent = L2BlockInfo {
            block_info: BlockInfo { timestamp: 1, ..Default::default() },
            ..Default::default()
        };
        let batch = SingleBatch { timestamp: 1, ..Default::default() };
        let result = attributes_queue.create_next_attributes(batch, parent).await.unwrap_err();
        assert_eq!(result, PipelineErrorKind::Reset(ResetError::BadTimestamp(1, 2)));
    }

    #[tokio::test]
    async fn test_create_next_attributes_preparation_fails() {
        let mut attributes_queue = new_attributes_queue(None, None, vec![]);
        let parent = L2BlockInfo::default();
        let batch = SingleBatch::default();
        let result = attributes_queue.create_next_attributes(batch, parent).await.unwrap_err();
        assert_eq!(
            result,
            PipelineError::AttributesBuilder(BuilderError::AttributesUnavailable).crit()
        );
    }

    #[tokio::test]
    async fn test_create_next_attributes_success() {
        let cfg = RollupConfig::default();
        let mock = new_test_attributes_provider(None, vec![]);
        let mut payload_attributes = default_optimism_payload_attributes();
        let mock_builder =
            TestAttributesBuilder { attributes: vec![Ok(payload_attributes.clone())] };
        let mut aq = AttributesQueue::new(Arc::new(cfg), mock, mock_builder);
        let parent = L2BlockInfo::default();
        let txs = vec![Bytes::default(), Bytes::default()];
        let batch = SingleBatch { transactions: txs.clone(), ..Default::default() };
        let attributes = aq.create_next_attributes(batch, parent).await.unwrap();
        // update the expected attributes
        payload_attributes.no_tx_pool = Some(true);
        match payload_attributes.transactions {
            Some(ref mut t) => t.extend(txs),
            None => payload_attributes.transactions = Some(txs),
        }
        assert_eq!(attributes, payload_attributes);
    }

    #[tokio::test]
    async fn test_next_attributes_load_batch_eof() {
        let mut attributes_queue = new_attributes_queue(None, None, vec![]);
        let parent = L2BlockInfo::default();
        let result = attributes_queue.next_attributes(parent).await.unwrap_err();
        assert_eq!(result, PipelineError::Eof.temp());
    }

    #[tokio::test]
    async fn test_next_attributes_load_batch_last_in_span() {
        let cfg = RollupConfig::default();
        let mock =
            new_test_attributes_provider(Some(Default::default()), vec![Ok(Default::default())]);
        let mut pa = default_optimism_payload_attributes();
        let mock_builder = TestAttributesBuilder { attributes: vec![Ok(pa.clone())] };
        let mut aq = AttributesQueue::new(Arc::new(cfg), mock, mock_builder);
        // If we load the batch, we should get the last in span.
        // But it won't take it so it will be available in the next_attributes call.
        let _ = aq.load_batch(L2BlockInfo::default()).await.unwrap();
        assert!(aq.is_last_in_span);
        assert!(aq.batch.is_some());
        // This should successfully construct the next payload attributes.
        // It should also reset the last in span flag and clear the batch.
        let attributes = aq.next_attributes(L2BlockInfo::default()).await.unwrap();
        pa.no_tx_pool = Some(true);
        let populated_attributes = OpAttributesWithParent {
            attributes: pa,
            parent: L2BlockInfo::default(),
            derived_from: Some(BlockInfo::default()),
            is_last_in_span: true,
        };
        assert_eq!(attributes, populated_attributes);
        assert!(!aq.is_last_in_span);
        assert!(aq.batch.is_none());
    }
}
