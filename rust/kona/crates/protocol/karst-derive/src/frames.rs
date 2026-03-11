//! Frame queue: loading, parsing, and Holocene pruning of frames.

use alloy_primitives::Bytes;
use kona_derive::{
    AttributesBuilder, ChainProvider, DataAvailabilityProvider, L2ChainProvider, PipelineError,
    PipelineResult,
};
use kona_protocol::Frame;
use tracing::{debug, error};

use crate::Pipeline;

impl<CP, DAP, L2P, AB> Pipeline<CP, DAP, L2P, AB>
where
    CP: ChainProvider + Send,
    DAP: DataAvailabilityProvider + Send,
    L2P: L2ChainProvider + Send,
    AB: AttributesBuilder + Send,
{
    /// Loads frames from L1 data into the frame queue.
    ///
    /// Fetches batcher data from the DA provider, parses it into frames,
    /// and applies Holocene pruning rules.
    pub(crate) async fn load_frames(&mut self) -> PipelineResult<()> {
        // Skip if queue already has frames.
        if !self.frame_queue.is_empty() {
            return Ok(());
        }

        // Get the L1 block to retrieve data from.
        if self.retrieval_block.is_none() {
            let block = self.next_l1_block()?;
            self.retrieval_block = Some(block);
        }

        let block = self.retrieval_block.as_ref().expect("retrieval_block set above");
        let batcher_addr = self.system_config.batcher_address;

        let data = match self.da_provider.next(block, batcher_addr).await {
            Ok(data) => data,
            Err(e) => {
                // On EOF, clear retrieval state so next call fetches a new block.
                if e == PipelineError::Eof.temp() {
                    self.retrieval_block = None;
                    self.da_provider.clear();
                }
                debug!(target: "karst", "Failed to retrieve data: {:?}", e);
                return Err(e);
            }
        };

        let raw: Bytes = data.into();
        let Ok(frames) = Frame::parse_frames(&raw) else {
            error!(target: "karst", "Failed to parse frames from data.");
            return Ok(());
        };

        self.frame_queue.extend(frames);

        // Always apply Holocene pruning (karst only supports Holocene+).
        self.prune_frames();

        Ok(())
    }

    /// Holocene frame pruning.
    ///
    /// Enforces:
    /// - Sequential frame numbers within a channel
    /// - No frames after a closing frame within the same channel
    /// - New channels must start with frame 0
    /// - Incomplete channels are dropped when a new channel starts
    pub(crate) fn prune_frames(&mut self) {
        if self.frame_queue.len() <= 1 {
            return;
        }

        let mut i = 0;
        while i < self.frame_queue.len() - 1 {
            let prev_frame_id = self.frame_queue[i].id;
            let prev_frame_number = self.frame_queue[i].number;
            let prev_frame_is_last = self.frame_queue[i].is_last;

            let next_frame_id = self.frame_queue[i + 1].id;
            let next_frame_number = self.frame_queue[i + 1].number;

            let extends_channel = prev_frame_id == next_frame_id;

            // Sequential frame numbers within same channel.
            if extends_channel && prev_frame_number + 1 != next_frame_number {
                self.frame_queue.remove(i + 1);
                continue;
            }

            // No frames after closing frame in same channel.
            if extends_channel && prev_frame_is_last {
                self.frame_queue.remove(i + 1);
                continue;
            }

            // New channel must start with frame 0.
            if !extends_channel && next_frame_number != 0 {
                self.frame_queue.remove(i + 1);
                continue;
            }

            // Incomplete channel dropped when new channel starts.
            if !extends_channel && !prev_frame_is_last && next_frame_number == 0 {
                let first_frame = self
                    .frame_queue
                    .iter()
                    .position(|f| f.id == prev_frame_id)
                    .expect("infallible");
                let drained = self.frame_queue.drain(first_frame..=i);
                i = i.saturating_sub(drained.len());
                continue;
            }

            i += 1;
        }
    }

    /// Pops and returns the next frame from the queue, loading more if needed.
    pub(crate) async fn next_frame(&mut self) -> PipelineResult<Frame> {
        self.load_frames().await?;

        if self.frame_queue.is_empty() {
            return Err(PipelineError::NotEnoughData.temp());
        }

        Ok(self.frame_queue.pop_front().expect("checked non-empty"))
    }
}
