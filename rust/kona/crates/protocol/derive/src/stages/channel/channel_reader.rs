//! This module contains the `ChannelReader` struct.

use crate::{
    BatchStreamProvider, OriginAdvancer, OriginProvider, PipelineError, PipelineResult, Signal,
    SignalReceiver,
};
use alloc::{boxed::Box, sync::Arc};
use alloy_primitives::Bytes;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_genesis::{
    MAX_RLP_BYTES_PER_CHANNEL_BEDROCK, MAX_RLP_BYTES_PER_CHANNEL_FJORD, RollupConfig,
};
use kona_protocol::{Batch, BatchReader, BlockInfo};
use tracing::{debug, warn};

/// The [`ChannelReader`] provider trait.
#[async_trait]
pub trait ChannelReaderProvider {
    /// Pulls the next piece of data from the channel bank. Note that it attempts to pull data out
    /// of the channel bank prior to loading data in (unlike most other stages). This is to
    /// ensure maintain consistency around channel bank pruning which depends upon the order
    /// of operations.
    async fn next_data(&mut self) -> PipelineResult<Option<Bytes>>;
}

/// [`ChannelReader`] is a stateful stage that reads [`Batch`]es from `Channel`s.
///
/// The [`ChannelReader`] pulls `Channel`s from the channel bank as raw data
/// and pipes it into a `BatchReader`. Since the raw data is compressed,
/// the `BatchReader` first decompresses the data using the first bytes as
/// a compression algorithm identifier.
///
/// Once the data is decompressed, it is decoded into a `Batch` and passed
/// to the next stage in the pipeline.
#[derive(Debug)]
pub struct ChannelReader<P>
where
    P: ChannelReaderProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// The previous stage of the derivation pipeline.
    pub prev: P,
    /// The batch reader.
    pub next_batch: Option<BatchReader>,
    /// The rollup configuration.
    pub cfg: Arc<RollupConfig>,
}

impl<P> ChannelReader<P>
where
    P: ChannelReaderProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    /// Create a new [`ChannelReader`] stage.
    pub const fn new(prev: P, cfg: Arc<RollupConfig>) -> Self {
        Self { prev, next_batch: None, cfg }
    }

    /// Creates the batch reader from available channel data.
    async fn set_batch_reader(&mut self) -> PipelineResult<()> {
        if self.next_batch.is_none() {
            let channel =
                self.prev.next_data().await?.ok_or(PipelineError::ChannelReaderEmpty.temp())?;

            let origin = self.prev.origin().ok_or(PipelineError::MissingOrigin.crit())?;
            let max_rlp_bytes_per_channel = if self.cfg.is_fjord_active(origin.timestamp) {
                MAX_RLP_BYTES_PER_CHANNEL_FJORD
            } else {
                MAX_RLP_BYTES_PER_CHANNEL_BEDROCK
            };

            self.next_batch =
                Some(BatchReader::new(&channel[..], max_rlp_bytes_per_channel as usize));
            kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_BATCH_READER_SET, 1);
        }
        Ok(())
    }

    /// Forces the read to continue with the next channel, resetting any
    /// decoding / decompression state to a fresh start.
    pub fn next_channel(&mut self) {
        self.next_batch = None;
        kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_BATCH_READER_SET, 0);
    }
}

#[async_trait]
impl<P> OriginAdvancer for ChannelReader<P>
where
    P: ChannelReaderProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    async fn advance_origin(&mut self) -> PipelineResult<()> {
        self.prev.advance_origin().await
    }
}

#[async_trait]
impl<P> BatchStreamProvider for ChannelReader<P>
where
    P: ChannelReaderProvider + OriginAdvancer + OriginProvider + SignalReceiver + Send + Debug,
{
    /// This method is called by the BatchStream if an invalid span batch is found.
    /// In the case of an invalid span batch, the associated channel must be flushed.
    ///
    /// See: <https://specs.optimism.io/protocol/holocene/derivation.html#span-batches>
    ///
    /// SAFETY: Only called post-holocene activation.
    fn flush(&mut self) {
        debug!(target: "channel_reader", "[POST-HOLOCENE] Flushing channel");
        self.next_channel();
    }

    async fn next_batch(&mut self) -> PipelineResult<Batch> {
        if let Err(e) = self.set_batch_reader().await {
            debug!(target: "channel_reader", "Failed to set batch reader: {:?}", e);
            self.next_channel();
            return Err(e);
        }

        // SAFETY: The batch reader must be set above.
        let next_batch = self.next_batch.as_mut().expect("Batch reader must be set");
        match next_batch.decompress() {
            Ok(()) => {
                // Record the decompressed size and type.
                let size = next_batch.decompressed.len() as f64;
                let ty = if next_batch.brotli_used {
                    BatchReader::CHANNEL_VERSION_BROTLI
                } else {
                    BatchReader::ZLIB_DEFLATE_COMPRESSION_METHOD
                };
                kona_macros::set!(
                    gauge,
                    crate::metrics::Metrics::PIPELINE_LATEST_DECOMPRESSED_BATCH_SIZE,
                    size
                );
                kona_macros::set!(
                    gauge,
                    crate::metrics::Metrics::PIPELINE_LATEST_DECOMPRESSED_BATCH_TYPE,
                    ty as f64
                );
            }
            Err(err) => {
                debug!(target: "channel_reader", ?err, "Failed to decompress batch");
                self.next_channel();
                return Err(PipelineError::NotEnoughData.temp());
            }
        }

        // Read the next batch from the reader's decompressed data
        match next_batch.next_batch(self.cfg.as_ref()).ok_or(PipelineError::NotEnoughData.temp()) {
            Ok(batch) => {
                kona_macros::inc!(
                    gauge,
                    crate::metrics::Metrics::PIPELINE_READ_BATCHES,
                    "type" => batch.to_string(),
                );
                Ok(batch)
            }
            Err(e) => {
                self.next_channel();
                Err(e)
            }
        }
    }
}

impl<P> OriginProvider for ChannelReader<P>
where
    P: ChannelReaderProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug,
{
    fn origin(&self) -> Option<BlockInfo> {
        self.prev.origin()
    }
}

#[async_trait]
impl<P> SignalReceiver for ChannelReader<P>
where
    P: ChannelReaderProvider + OriginAdvancer + OriginProvider + SignalReceiver + Debug + Send,
{
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        match signal {
            Signal::FlushChannel => {
                // Drop the current in-progress channel.
                warn!(target: "channel_reader", "Flushed channel");
                self.next_batch = None;
                kona_macros::set!(gauge, crate::metrics::Metrics::PIPELINE_BATCH_READER_SET, 0);
            }
            s => {
                self.prev.signal(s).await?;
                self.next_channel();
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::{
        errors::PipelineErrorKind, test_utils::TestChannelReaderProvider, types::ResetSignal,
    };
    use alloc::vec;
    use kona_genesis::HardForkConfig;

    fn new_compressed_batch_data() -> Bytes {
        let file_contents =
            alloc::string::String::from_utf8_lossy(include_bytes!("../../../testdata/batch.hex"));
        let file_contents = &(&*file_contents)[..file_contents.len() - 1];
        let data = alloy_primitives::hex::decode(file_contents).unwrap();
        data.into()
    }

    #[tokio::test]
    async fn test_flush_channel_reader() {
        let mock = TestChannelReaderProvider::new(vec![Ok(Some(new_compressed_batch_data()))]);
        let mut reader = ChannelReader::new(mock, Arc::new(RollupConfig::default()));
        reader.next_batch = Some(BatchReader::new(
            new_compressed_batch_data(),
            MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize,
        ));
        reader.signal(Signal::FlushChannel).await.unwrap();
        assert!(reader.next_batch.is_none());
    }

    #[tokio::test]
    async fn test_reset_channel_reader() {
        let mock = TestChannelReaderProvider::new(vec![Ok(None)]);
        let mut reader = ChannelReader::new(mock, Arc::new(RollupConfig::default()));
        reader.next_batch = Some(BatchReader::new(
            vec![0x00, 0x01, 0x02],
            MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize,
        ));
        assert!(!reader.prev.reset);
        reader.signal(ResetSignal::default().signal()).await.unwrap();
        assert!(reader.next_batch.is_none());
        assert!(reader.prev.reset);
    }

    #[tokio::test]
    async fn test_next_batch_batch_reader_set_fails() {
        let mock = TestChannelReaderProvider::new(vec![Err(PipelineError::Eof.temp())]);
        let mut reader = ChannelReader::new(mock, Arc::new(RollupConfig::default()));
        assert_eq!(reader.next_batch().await, Err(PipelineError::Eof.temp()));
        assert!(reader.next_batch.is_none());
    }

    #[tokio::test]
    async fn test_next_batch_batch_reader_no_data() {
        let mock = TestChannelReaderProvider::new(vec![Ok(None)]);
        let mut reader = ChannelReader::new(mock, Arc::new(RollupConfig::default()));
        assert!(matches!(
            reader.next_batch().await.unwrap_err(),
            PipelineErrorKind::Temporary(PipelineError::ChannelReaderEmpty)
        ));
        assert!(reader.next_batch.is_none());
    }

    #[tokio::test]
    async fn test_next_batch_batch_reader_not_enough_data() {
        let mut first = new_compressed_batch_data();
        let second = first.split_to(first.len() / 2);
        let mock = TestChannelReaderProvider::new(vec![Ok(Some(first)), Ok(Some(second))]);
        let mut reader = ChannelReader::new(mock, Arc::new(RollupConfig::default()));
        assert_eq!(reader.next_batch().await, Err(PipelineError::NotEnoughData.temp()));
        assert!(reader.next_batch.is_none());
    }

    #[tokio::test]
    async fn test_next_batch_succeeds() {
        let raw = new_compressed_batch_data();
        let mock = TestChannelReaderProvider::new(vec![Ok(Some(raw))]);
        let mut reader = ChannelReader::new(mock, Arc::new(RollupConfig::default()));
        let res = reader.next_batch().await.unwrap();
        matches!(res, Batch::Span(_));
        assert!(reader.next_batch.is_some());
    }

    #[tokio::test]
    async fn test_flush_post_holocene() {
        let raw = new_compressed_batch_data();
        let config = Arc::new(RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        });
        let mock = TestChannelReaderProvider::new(vec![Ok(Some(raw))]);
        let mut reader = ChannelReader::new(mock, config);
        let res = reader.next_batch().await.unwrap();
        matches!(res, Batch::Span(_));
        assert!(reader.next_batch.is_some());
        reader.flush();
        assert!(reader.next_batch.is_none());
    }
}
