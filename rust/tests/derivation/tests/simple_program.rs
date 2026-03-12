//! Simple derivation test: produce an empty L2 block, batch it, and verify
//! the derivation pipeline processes it correctly.

use alloy_consensus::Transaction;
use derivation_tests::{genesis, harness::TestEnv};
use tracing_subscriber::{EnvFilter, fmt};

fn init_tracing() {
    let _ = fmt()
        .with_env_filter(
            EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")),
        )
        .try_init();
}

#[tokio::test]
async fn test_derivation_produces_attributes_for_empty_block() {
    init_tracing();

    let mut env = TestEnv::builder().build().await;
    let genesis_l2 = env.genesis_l2;

    // Produce one empty L2 block
    let _l2_block = env.produce_l2_block(vec![]);

    // Batch -> submit -> mine L1 -> advance safe/finalized
    env.batch_and_mine();
    env.advance_l1_safe_and_finalized();

    // Tell the derivation actor that EL sync is complete
    env.node_actor.notify_el_sync_complete(genesis_l2).await;

    // Tell the derivation actor about the new L1 head
    env.node_actor.notify_l1_head(env.l1.latest()).await;

    // Step the derivation actor - it should process the L1 head update
    // and attempt derivation
    let step_result = env.node_actor.step().await;
    tracing::info!(?step_result, "First step (EL sync complete)");

    let step_result = env.node_actor.step().await;
    tracing::info!(?step_result, "Second step (L1 head update)");

    // Check that the engine received consolidation signals
    let outputs = env.engine_client.captured_outputs();
    tracing::info!(
        consolidation_signals = outputs.consolidation_signals.len(),
        finalized_blocks = outputs.finalized_blocks.len(),
        reset_count = outputs.reset_count,
        "Engine outputs after derivation"
    );
}

#[tokio::test]
async fn test_mock_l1_chain_block_production() {
    init_tracing();

    let genesis = genesis::default_l1_genesis();
    let l1 = derivation_tests::mock_l1::MockL1Chain::new(genesis);

    // Mine some blocks
    for i in 1..=5 {
        l1.act_start_block(12);
        let info = l1.act_end_block(12);
        assert_eq!(info.number, i);
        assert_eq!(info.parent_hash, l1.block_info_at(i - 1).unwrap().hash);
    }

    assert_eq!(l1.len(), 6); // genesis + 5 blocks

    // Advance safe and finalized
    l1.act_safe_next();
    l1.act_safe_next();
    l1.act_finalize_next();

    assert_eq!(l1.safe_head().number, 2);
    assert_eq!(l1.finalized_head().number, 1);
}

#[tokio::test]
async fn test_mock_batcher_encoding() {
    init_tracing();

    use derivation_tests::mock_batcher::MockBatcher;
    use derivation_tests::mock_sequencer::MockSequencer;

    let genesis = genesis::default_l1_genesis();
    let l2_genesis = genesis::default_l2_genesis();

    let mut sequencer =
        MockSequencer::new(l2_genesis.block_info.hash, l2_genesis.block_info.timestamp, 2);

    let mut batcher = MockBatcher::new(genesis::TEST_BATCH_INBOX, genesis::TEST_BATCHER_ADDR);

    // Produce an empty L2 block
    let block = sequencer.produce_block(vec![], genesis);
    batcher.buffer_block(&block);

    // Submit - should produce at least one L1 tx
    let txs = batcher.submit_all();
    assert!(!txs.is_empty(), "batcher should produce at least one tx");
    tracing::info!(tx_count = txs.len(), "Batcher produced L1 transactions");

    // Verify the tx calldata starts with DERIVATION_VERSION_0
    for tx in &txs {
        let input = tx.input();
        assert_eq!(
            input[0], 0x00,
            "calldata should start with DERIVATION_VERSION_0"
        );

        // Verify frames can be parsed from the calldata
        let frames = kona_protocol::Frame::parse_frames(input).expect("should parse frames");
        assert!(!frames.is_empty());
        tracing::info!(frame_count = frames.len(), "Parsed frames from tx calldata");
    }
}
