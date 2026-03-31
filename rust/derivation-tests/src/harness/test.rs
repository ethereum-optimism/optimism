//! The main `DerivationTest` entry point.

use alloy_eips::eip4844::{self, BlobTransactionSidecar};
use alloy_primitives::{B256, Bytes};
use kona_protocol::ChannelId;

use crate::{
    batch::{ChannelOut, CompressionAlgo, L1Origin, block_to_singular_batch, build_span_batch},
    config::DeterministicConfig,
    l1::{BatchSubmission, BlobWithCommitment, L1Block, L1ChainBuilder},
    l2::{L2BlockRef, L2ChainBuilder},
    roots::{compute_output_root_from_state, compute_single_chain_super_root},
    server::TestServers,
};

/// High-level test entry point that ties together L1 builder, L2 builder, and verification.
///
/// Provides both a low-level API (`self.l1`, `self.l2`) and a scenario-oriented DSL
/// (`advance_l1`, `derive_empty_l2_block`, `submit_batch`, `finalize`). The DSL methods
/// wrap the low-level API with sensible defaults — tests can freely mix both.
#[allow(missing_debug_implementations)]
pub struct DerivationTest {
    /// Deterministic config.
    pub config: DeterministicConfig,
    /// L1 chain builder.
    pub l1: L1ChainBuilder,
    /// L2 chain builder.
    pub l2: L2ChainBuilder,
    channel_counter: u16,
    // DSL state
    dsl_epoch_block: Option<L1Block>,
    pub(crate) pending_l2_blocks: Vec<L2BlockRef>,
    pub(crate) prefunded_nonce: u64,
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
        // Initialize the prefunded nonce from genesis state (may be > 0 due to
        // op-deployer using the account during deployment).
        let prefunded_nonce =
            l2.state().account(&crate::config::PREFUNDED_ACCOUNT).map_or(0, |a| a.nonce);
        Self {
            config,
            l1,
            l2,
            channel_counter: 0,
            dsl_epoch_block: None,
            pending_l2_blocks: Vec::new(),
            prefunded_nonce,
        }
    }

    /// Compute the expected output root for the current L2 head.
    pub fn expected_output_root(&self) -> B256 {
        let head = self.l2.head();
        let snapshot = self.l2.head_snapshot();
        let output_root = compute_output_root_from_state(head, snapshot);
        eprintln!("computed output root: {output_root:?}");
        output_root
    }

    /// Compute the output root at a specific L2 block.
    pub fn output_root_at(&self, block_ref: L2BlockRef) -> B256 {
        let block = self.l2.block(block_ref);
        let snapshot = self.l2.snapshot_at(block_ref);
        compute_output_root_from_state(block, snapshot)
    }

    /// Compute the super root at a specific L2 block.
    pub fn super_root_at(&self, block_ref: L2BlockRef) -> B256 {
        let output_root = self.output_root_at(block_ref);
        let timestamp = self.l2.block(block_ref).header.inner().timestamp;
        compute_single_chain_super_root(timestamp, self.config.l2_chain_id, output_root)
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
        let channel_id = self.next_channel_id();
        let rollup_config = self.config.rollup_config();
        let origin =
            L1Origin { number: l1_origin.header.inner().number, hash: l1_origin.header.hash() };

        let mut channel = ChannelOut::new(channel_id, CompressionAlgo::Zlib);
        for block_ref in block_refs {
            let block = self.l2.block(*block_ref);
            let batch = block_to_singular_batch(block, rollup_config, origin);
            channel.add_singular_batch(&batch).expect("add batch failed");
        }
        channel.close().expect("close channel failed");

        let calldata = channel.to_calldata(100_000);
        assert!(!calldata.is_empty(), "expected at least one frame");
        // For simplicity, return the first frame as calldata
        BatchSubmission::Calldata(calldata.into_iter().next().unwrap())
    }

    /// Encode L2 blocks as a span batch in a channel with zlib compression, returned as calldata.
    pub fn span_batch_calldata(
        &mut self,
        block_refs: &[L2BlockRef],
        l1_origin: &L1Block,
    ) -> BatchSubmission {
        let channel_id = self.next_channel_id();
        let rollup_config = self.config.rollup_config();
        let origin =
            L1Origin { number: l1_origin.header.inner().number, hash: l1_origin.header.hash() };

        let blocks: Vec<_> = block_refs.iter().map(|r| self.l2.block(*r)).collect();
        let span_batch = build_span_batch(&blocks, origin, rollup_config);

        let mut channel = ChannelOut::new(channel_id, CompressionAlgo::Zlib);
        channel.add_span_batch(&span_batch).expect("add span batch failed");
        channel.close().expect("close channel failed");

        let calldata = channel.to_calldata(100_000);
        assert!(!calldata.is_empty(), "expected at least one frame");
        BatchSubmission::Calldata(calldata.into_iter().next().unwrap())
    }

    /// Encode L2 blocks as a span batch in a channel with zlib compression, returned as a blob.
    pub fn blob_span_batch(
        &mut self,
        block_refs: &[L2BlockRef],
        l1_origin: &L1Block,
    ) -> BatchSubmission {
        let channel_id = self.next_channel_id();
        let rollup_config = self.config.rollup_config();
        let origin =
            L1Origin { number: l1_origin.header.inner().number, hash: l1_origin.header.hash() };

        let blocks: Vec<_> = block_refs.iter().map(|r| self.l2.block(*r)).collect();
        let span_batch = build_span_batch(&blocks, origin, rollup_config);

        let mut channel = ChannelOut::new(channel_id, CompressionAlgo::Zlib);
        channel.add_span_batch(&span_batch).expect("add span batch failed");
        channel.close().expect("close channel failed");

        let calldata = channel.to_calldata(100_000);
        assert!(!calldata.is_empty(), "expected at least one frame");
        let frame_data = calldata.into_iter().next().unwrap();

        let builder = eip4844::builder::SidecarBuilder::<eip4844::builder::SimpleCoder>::from_slice(
            &frame_data,
        );
        let sidecar: BlobTransactionSidecar =
            builder.build_4844().expect("blob sidecar construction failed");

        let blob = sidecar.blobs[0];
        let versioned_hash = eip4844::kzg_to_versioned_hash(sidecar.commitments[0].as_slice());

        BatchSubmission::Blob(BlobWithCommitment {
            blob: Box::new(blob),
            commitment: sidecar,
            versioned_hash,
        })
    }

    /// Create a batch submission from raw invalid data (for negative tests).
    pub const fn invalid_batch(data: Bytes) -> BatchSubmission {
        BatchSubmission::Calldata(data)
    }

    /// Start RPC servers for the built chains.
    pub async fn serve(&self) -> Result<TestServers, Box<dyn std::error::Error>> {
        TestServers::start(&self.config, &self.l1, &self.l2).await
    }

    // --- Scenario DSL ---------------------------------------------------------

    /// Emit `count` empty L1 blocks.
    pub fn advance_l1(&mut self, count: usize) {
        for _ in 0..count {
            self.l1.emit_empty_block();
        }
    }

    /// Advance the L2 epoch to the latest L1 block.
    ///
    /// Call this between `derive_*` calls to switch the L1 origin for
    /// subsequent L2 blocks. Automatically called by the first `derive_*`
    /// call (sets epoch to L1 genesis).
    pub fn advance_epoch(&mut self) {
        let head = self.l1.head().clone();
        self.l2.set_epoch(&head);
        self.dsl_epoch_block = Some(head);
    }

    /// Build one empty (deposit-only) L2 block.
    ///
    /// Auto-sets the epoch to L1 genesis on the first call.
    pub fn derive_empty_l2_block(&mut self) {
        self.ensure_epoch();
        let block_ref = self.l2.build_empty_block().expect("failed to build L2 block");
        self.pending_l2_blocks.push(block_ref);
    }

    /// Build `count` empty (deposit-only) L2 blocks.
    pub fn derive_empty_l2_blocks(&mut self, count: usize) {
        self.ensure_epoch();
        for _ in 0..count {
            let block_ref = self.l2.build_empty_block().expect("failed to build L2 block");
            self.pending_l2_blocks.push(block_ref);
        }
    }

    /// Start building an L2 block with user transactions.
    ///
    /// Returns a [`BlockBuilder`](super::BlockBuilder) that collects transactions and builds the
    /// block on `.build()`.
    pub fn derive_l2_block(&mut self) -> super::BlockBuilder<'_> {
        self.ensure_epoch();
        super::BlockBuilder::new(self)
    }

    /// Encode all pending L2 blocks as a batch and submit on L1.
    ///
    /// Uses the default batch config (span batch via blobs).
    pub fn submit_batch(&mut self) {
        self.submit_batch_with(super::BatchConfig::default());
    }

    /// Encode all pending L2 blocks as a batch and submit on L1.
    pub fn submit_batch_with(&mut self, config: super::BatchConfig) {
        use super::dsl::{BatchEncoding, BatchSubmissionType};

        let refs: Vec<L2BlockRef> = self.pending_l2_blocks.drain(..).collect();
        assert!(
            !refs.is_empty(),
            "no pending L2 blocks to submit — call derive_empty_l2_block() first"
        );

        let epoch =
            self.dsl_epoch_block.clone().expect("epoch must be set before submitting a batch");

        let batch = match (config.encoding, config.submission) {
            (BatchEncoding::SpanBatch, BatchSubmissionType::Blobs) => {
                self.blob_span_batch(&refs, &epoch)
            }
            (BatchEncoding::SpanBatch, BatchSubmissionType::Calldata) => {
                self.span_batch_calldata(&refs, &epoch)
            }
            (BatchEncoding::Singular, BatchSubmissionType::Calldata) => {
                self.singular_batch_calldata(&refs, &epoch)
            }
            (BatchEncoding::Singular, BatchSubmissionType::Blobs) => {
                panic!("singular batch via blobs is not supported");
            }
        };

        self.l1.emit_block_with_batches(vec![batch]);
    }

    /// Seal the L1 chain and compute the super root.
    ///
    /// Emits a trailing L1 empty block (needed for the derivation pipeline
    /// to advance its safe head) and returns the expected super root.
    pub fn finalize(&mut self) -> B256 {
        self.l1.emit_empty_block();
        self.expected_super_root()
    }

    /// Set epoch to L1 genesis on first call. No-op on subsequent calls.
    fn ensure_epoch(&mut self) {
        if self.dsl_epoch_block.is_none() {
            let genesis = self.l1.block_at(0).expect("L1 must have a genesis block").clone();
            self.l2.set_epoch(&genesis);
            // The L2 genesis block already consumed seq_num 0 for the genesis epoch,
            // so the first derived block starts at seq_num 1.
            self.l2.advance_seq_num();
            self.dsl_epoch_block = Some(genesis);
        }
    }

    const fn next_channel_id(&mut self) -> ChannelId {
        let mut id = [0u8; 16];
        id[0] = (self.channel_counter >> 8) as u8;
        id[1] = self.channel_counter as u8;
        self.channel_counter += 1;
        id
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn blob_encoding_roundtrip() {
        let mut test = DerivationTest::new();

        test.l1.emit_empty_block();
        test.l1.emit_empty_block();

        let l1_block = test.l1.block_at(1).unwrap().clone();
        test.l2.set_epoch(&l1_block);
        let block_ref = test.l2.build_empty_block().unwrap();

        let l1_origin = test.l1.block_at(1).unwrap().clone();

        // Encode as blob span batch — exercises SidecarBuilder internally
        let batch = test.blob_span_batch(&[block_ref], &l1_origin);

        match batch {
            BatchSubmission::Blob(blob_data) => {
                // Sidecar should have exactly one blob
                assert_eq!(blob_data.commitment.blobs.len(), 1, "expected one blob");
                assert_eq!(blob_data.commitment.commitments.len(), 1, "expected one commitment");
                assert_eq!(blob_data.commitment.proofs.len(), 1, "expected one proof");

                // Versioned hash should be non-zero
                assert_ne!(
                    blob_data.versioned_hash,
                    B256::ZERO,
                    "versioned hash should be non-zero"
                );

                // Versioned hash should start with 0x01 (VERSIONED_HASH_VERSION_KZG)
                assert_eq!(
                    blob_data.versioned_hash[0], 0x01,
                    "versioned hash should start with KZG version byte"
                );
            }
            BatchSubmission::Calldata(_) => panic!("expected blob batch, got calldata"),
        }
    }

    #[tokio::test]
    async fn run_config_fields() {
        let mut test = DerivationTest::new();

        // Build a minimal chain so we have non-genesis state
        test.l1.emit_empty_block();
        let l1_block = test.l1.block_at(1).unwrap().clone();
        test.l2.set_epoch(&l1_block);
        test.l2.build_empty_block().unwrap();

        let l1_origin = test.l1.block_at(1).unwrap().clone();
        let block_ref = crate::l2::L2BlockRef { index: 1 };
        let batch = test.singular_batch_calldata(&[block_ref], &l1_origin);
        test.l1.emit_block_with_batches(vec![batch]);

        let servers = test.serve().await.unwrap();
        let run_config = crate::harness::run_config_from_test(&test, &servers);

        // l2_head should be the genesis block hash (agreed starting point)
        let genesis_hash = test.l2.block(crate::l2::L2BlockRef { index: 0 }).header.hash();
        assert_eq!(run_config.l2_head, genesis_hash, "l2_head should be genesis block hash");

        // l2_output_root should be the output root at genesis
        let genesis_output_root = test.output_root_at(crate::l2::L2BlockRef { index: 0 });
        assert_eq!(
            run_config.l2_output_root, genesis_output_root,
            "l2_output_root should match genesis"
        );

        // l2_block_number should be the target (head) block number
        assert_eq!(
            run_config.l2_block_number,
            test.l2.head().header.inner().number,
            "l2_block_number should be the head block number"
        );
        assert!(run_config.l2_block_number > 0, "l2_block_number should be > 0");

        // l1_head should be the latest L1 block
        assert_eq!(
            run_config.l1_head,
            test.l1.head().header.hash(),
            "l1_head should be latest L1 block"
        );

        // expected_claim should be the output root at the head
        assert_eq!(
            run_config.expected_claim,
            test.expected_output_root(),
            "expected_claim should match output root"
        );

        assert_ne!(run_config.l1_head, B256::ZERO, "l1_head should be non-zero");
        assert_ne!(run_config.l2_head, B256::ZERO, "l2_head should be non-zero");
        assert_ne!(run_config.expected_claim, B256::ZERO, "expected_claim should be non-zero");
        assert_ne!(run_config.l2_output_root, B256::ZERO, "l2_output_root should be non-zero");
        assert!(!run_config.l1_rpc.is_empty());
        assert!(!run_config.l2_rpc.is_empty());
        assert!(!run_config.beacon_url.is_empty());

        servers.stop();
    }
}
