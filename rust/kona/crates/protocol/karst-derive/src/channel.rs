//! Channel assembly: accumulating frames into a complete channel.

use alloy_primitives::{Bytes, hex};
use kona_derive::{
    AttributesBuilder, ChainProvider, DataAvailabilityProvider, L2ChainProvider, PipelineError,
    PipelineResult,
};
use kona_genesis::MAX_RLP_BYTES_PER_CHANNEL_FJORD;
use kona_protocol::Channel;
use tracing::{debug, error, info, warn};

use crate::Pipeline;

impl<CP, DAP, L2P, AB> Pipeline<CP, DAP, L2P, AB>
where
    CP: ChainProvider + Send,
    DAP: DataAvailabilityProvider + Send,
    L2P: L2ChainProvider + Send,
    AB: AttributesBuilder + Send,
{
    /// Assembles frames into a complete channel and returns the compressed channel bytes.
    ///
    /// This is the Holocene-only channel assembler. It always uses
    /// `MAX_RLP_BYTES_PER_CHANNEL_FJORD` for the size limit.
    pub(crate) async fn assemble_next_channel(&mut self) -> PipelineResult<Option<Bytes>> {
        let origin = self.l1_origin.ok_or(PipelineError::MissingOrigin.crit())?;

        // Time out the channel if it has exceeded the timeout.
        if let Some(channel) = self.channel.as_ref() {
            let timed_out = channel.open_block_number()
                + self.cfg.channel_timeout(origin.timestamp)
                < origin.number;
            if timed_out {
                warn!(
                    target: "karst",
                    "Channel (ID: {}) timed out at L1 origin #{}, open block #{}. Discarding.",
                    hex::encode(channel.id()),
                    origin.number,
                    channel.open_block_number()
                );
                self.channel = None;
            }
        }

        // Get the next frame.
        let next_frame = self.next_frame().await?;

        // Start a new channel if frame number is 0.
        if next_frame.number == 0 {
            info!(
                target: "karst",
                "Starting new channel (ID: {}) at L1 origin #{}",
                hex::encode(next_frame.id),
                origin.number
            );
            self.channel = Some(Channel::new(next_frame.id, origin));
        }

        if let Some(channel) = self.channel.as_mut() {
            debug!(
                target: "karst",
                "Adding frame #{} to channel (ID: {}) at L1 origin #{}",
                next_frame.number,
                hex::encode(channel.id()),
                origin.number
            );

            if channel.add_frame(next_frame, origin).is_err() {
                error!(
                    target: "karst",
                    "Failed to add frame to channel (ID: {}) at L1 origin #{}",
                    hex::encode(channel.id()),
                    origin.number
                );
                return Err(PipelineError::NotEnoughData.temp());
            }

            // Always use Fjord size limit (Holocene+ only).
            if channel.size() > MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize {
                warn!(
                    target: "karst",
                    "Channel size exceeded max RLP bytes, dropping channel (ID: {}) with {} bytes",
                    hex::encode(channel.id()),
                    channel.size()
                );
                self.channel = None;
                return Err(PipelineError::NotEnoughData.temp());
            }

            // If the channel is complete, return the compressed bytes.
            if channel.is_ready() {
                let channel_bytes = channel
                    .frame_data()
                    .ok_or(PipelineError::ChannelNotFound.crit())?;

                info!(
                    target: "karst",
                    "Channel (ID: {}) ready for decompression.",
                    hex::encode(channel.id()),
                );

                self.channel = None;
                return Ok(Some(channel_bytes));
            }
        }

        Err(PipelineError::NotEnoughData.temp())
    }
}
