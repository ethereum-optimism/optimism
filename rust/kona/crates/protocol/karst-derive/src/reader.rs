//! Channel reader: decompressing channel data into batches.

use kona_derive::{
    AttributesBuilder, ChainProvider, DataAvailabilityProvider, L2ChainProvider, PipelineError,
    PipelineResult,
};
use kona_genesis::MAX_RLP_BYTES_PER_CHANNEL_FJORD;
use kona_protocol::{Batch, BatchReader};
use tracing::debug;

use crate::Pipeline;

impl<CP, DAP, L2P, AB> Pipeline<CP, DAP, L2P, AB>
where
    CP: ChainProvider + Send,
    DAP: DataAvailabilityProvider + Send,
    L2P: L2ChainProvider + Send,
    AB: AttributesBuilder + Send,
{
    /// Ensures a `BatchReader` is available by assembling a channel if needed.
    pub(crate) async fn ensure_batch_reader(&mut self) -> PipelineResult<()> {
        if self.batch_reader.is_none() {
            let channel_data = self
                .assemble_next_channel()
                .await?
                .ok_or(PipelineError::ChannelReaderEmpty.temp())?;

            // Always use Fjord max size (Holocene+ only).
            self.batch_reader = Some(BatchReader::new(
                &channel_data[..],
                MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize,
            ));
        }
        Ok(())
    }

    /// Reads the next batch from the channel reader.
    ///
    /// Decompresses and decodes a batch from the current `BatchReader`.
    /// If decompression or decoding fails, the reader is discarded.
    pub(crate) async fn next_batch_from_reader(&mut self) -> PipelineResult<Batch> {
        if let Err(e) = self.ensure_batch_reader().await {
            debug!(target: "karst", "Failed to set batch reader: {:?}", e);
            self.batch_reader = None;
            return Err(e);
        }

        let reader = self.batch_reader.as_mut().expect("batch reader set above");

        // Decompress.
        if let Err(err) = reader.decompress() {
            debug!(target: "karst", ?err, "Failed to decompress batch");
            self.batch_reader = None;
            return Err(PipelineError::NotEnoughData.temp());
        }

        // Decode the next batch.
        match reader.next_batch(self.cfg.as_ref()) {
            Some(batch) => Ok(batch),
            None => {
                self.batch_reader = None;
                Err(PipelineError::NotEnoughData.temp())
            }
        }
    }
}
