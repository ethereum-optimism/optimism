//! The main `DerivationTest` entry point.

use alloy_primitives::B256;
use kona_protocol::ChannelId;

use crate::{
    batch::{ChannelOut, CompressionAlgo, L1Origin, block_to_singular_batch},
    config::DeterministicConfig,
    l1::{BatchSubmission, L1ChainBuilder, L1Block},
    l2::{L2BlockRef, L2ChainBuilder},
    roots::{compute_output_root_from_state, compute_single_chain_super_root},
    server::TestServers,
};

/// High-level test entry point that ties together L1 builder, L2 builder, and verification.
#[allow(missing_debug_implementations)]
pub struct DerivationTest {
    /// Deterministic config.
    pub config: DeterministicConfig,
    /// L1 chain builder.
    pub l1: L1ChainBuilder,
    /// L2 chain builder.
    pub l2: L2ChainBuilder,
    channel_counter: u16,
}

impl Default for DerivationTest {
    fn default() -> Self {
        Self::with_config(DeterministicConfig::default())
    }
}

impl DerivationTest {
    /// Create a new test with the default deterministic config.
    pub fn new() -> Self {
        Self::default()
    }

    /// Create a new test with a specific config.
    pub fn with_config(config: DeterministicConfig) -> Self {
        let l1 = L1ChainBuilder::new(&config);
        let l2 = L2ChainBuilder::new(&config);
        Self { config, l1, l2, channel_counter: 0 }
    }

    /// Compute the expected output root for the current L2 head.
    pub fn expected_output_root(&self) -> B256 {
        let head = self.l2.head();
        let snapshot = self.l2.head_snapshot();
        let output_root = compute_output_root_from_state(head, snapshot);
        eprintln!("computed output root: {output_root:?}");
        output_root
    }

    /// Compute the expected super root for the current L2 head.
    pub fn expected_super_root(&self) -> B256 {
        let output_root = self.expected_output_root();
        let timestamp = self.l2.head().header.inner().timestamp;
        let super_root =
            compute_single_chain_super_root(timestamp, self.config.l2_chain_id, output_root);
        eprintln!("computed super root: {super_root:?}");
        super_root
    }

    /// Encode L2 blocks as a singular batch in calldata.
    ///
    /// `l1_origin` specifies the L1 block that these L2 blocks were derived from.
    pub fn singular_batch_calldata(
        &mut self,
        block_refs: &[L2BlockRef],
        l1_origin: &L1Block,
    ) -> BatchSubmission {
        let rollup_config = self.config.rollup_config();
        let channel_id = self.next_channel_id();
        let origin = L1Origin {
            number: l1_origin.header.inner().number,
            hash: l1_origin.header.hash(),
        };

        let mut channel = ChannelOut::new(channel_id, CompressionAlgo::Zlib);
        for block_ref in block_refs {
            let block = self.l2.block(*block_ref);
            let batch = block_to_singular_batch(block, &rollup_config, origin);
            channel.add_singular_batch(&batch).expect("add batch failed");
        }
        channel.close().expect("close channel failed");

        let calldata = channel.to_calldata(100_000);
        assert!(!calldata.is_empty(), "expected at least one frame");
        // For simplicity, return the first frame as calldata
        BatchSubmission::Calldata(calldata.into_iter().next().unwrap())
    }

    /// Start RPC servers for the built chains.
    pub async fn serve(&self) -> Result<TestServers, Box<dyn std::error::Error>> {
        TestServers::start(&self.config, &self.l1, &self.l2).await
    }

    const fn next_channel_id(&mut self) -> ChannelId {
        let mut id = [0u8; 16];
        id[0] = (self.channel_counter >> 8) as u8;
        id[1] = self.channel_counter as u8;
        self.channel_counter += 1;
        id
    }
}
