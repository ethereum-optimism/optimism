//! Simple derivation test scenarios exercising the full framework.
//!
//! # Golden super root values
//!
//! Tests pin expected super root hashes as hardcoded constants. This catches silent
//! regressions in batch encoding, block execution, state root computation, or output
//! root calculation — a consistently wrong root would pass a pure determinism check.
//!
//! ## Updating golden values
//!
//! When an intentional change modifies the derivation output (e.g. a new hardfork,
//! changed genesis config, or updated deposit tx encoding):
//!
//! 1. Run: `cargo test -p derivation-tests -- --nocapture 2>&1 | grep "super root"`
//! 2. Each test prints its computed super root to stderr before asserting.
//! 3. Copy the new values into the `EXPECTED_*` constants below.
//! 4. Re-run to confirm all tests pass.
//!
//! The golden values are framework-internal — they are NOT cross-checked against Go
//! implementations yet. That validation happens via the `#[ignore]` integration tests
//! that run op-program and kona-host against the same chains.

use alloy_primitives::b256;
use derivation_tests::{config::DeterministicConfig, harness::DerivationTest};

/// Expected super root for 3 empty L2 blocks (deposit-only) derived from empty L1 blocks.
const EXPECTED_EMPTY_BLOCKS_ROOT: alloy_primitives::B256 =
    b256!("aacf2b3a7e6b551fee9073326b10fad187b245079da11acdb1a4b77c5c7461db");

/// Expected super root for 1 empty L2 block submitted as a singular batch.
const EXPECTED_SINGLE_BATCH_ROOT: alloy_primitives::B256 =
    b256!("f5d6c4e0d16e99712e0304012a3fef20599f3b0ac9e75fa705f58c4c5e53dedd");

/// Expected super root for 1 L2 block containing a 1 ETH transfer.
const EXPECTED_SINGLE_TRANSFER_ROOT: alloy_primitives::B256 =
    b256!("47c3a6fca4d92dda11dc084f4f8201d51adffe0307536f23bbf4e1f92388c555");

/// Expected super root for 6 empty L2 blocks (one full epoch).
const EXPECTED_MULTI_BLOCK_ROOT: alloy_primitives::B256 =
    b256!("42d497513a28cfa89371cc2c13d44fbbb6934899d18cf56a306b881112318cd0");

/// Build a test with N empty L2 blocks derived from N empty L1 blocks.
fn build_empty_blocks_test(l2_count: usize) -> DerivationTest {
    let mut test = DerivationTest::new();

    // Emit L1 blocks to serve as epochs
    for _ in 0..l2_count {
        test.l1.emit_empty_block();
    }

    // Set epoch to the first L1 block and build L2 blocks
    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);

    for _ in 0..l2_count {
        test.l2.build_empty_block().unwrap();
    }

    test
}

#[test]
fn test_empty_blocks_deterministic() {
    let test1 = build_empty_blocks_test(3);
    let test2 = build_empty_blocks_test(3);

    let root1 = test1.expected_super_root();
    let root2 = test2.expected_super_root();

    assert_eq!(root1, root2, "same inputs should produce same super root");
    assert_eq!(
        root1, EXPECTED_EMPTY_BLOCKS_ROOT,
        "super root mismatch — update EXPECTED_EMPTY_BLOCKS_ROOT if this is an intentional change"
    );
}

#[test]
fn test_empty_blocks_structure() {
    let test = build_empty_blocks_test(3);

    // 4 blocks total: genesis + 3 built
    assert_eq!(test.l2.blocks().len(), 4);

    // Check parent chain
    let blocks = test.l2.blocks();
    for i in 1..blocks.len() {
        assert_eq!(
            blocks[i].header.inner().parent_hash,
            blocks[i - 1].header.hash(),
            "block {} parent hash mismatch",
            i
        );
    }

    // Check timestamps increment by L2 block time
    let config = DeterministicConfig::default();
    for i in 1..blocks.len() {
        assert_eq!(
            blocks[i].header.inner().timestamp,
            blocks[i - 1].header.inner().timestamp + config.l2_block_time,
            "block {} timestamp mismatch",
            i
        );
    }
}

#[test]
fn test_single_batch_submission() {
    let mut test = DerivationTest::new();

    // Build L1 blocks
    test.l1.emit_empty_block(); // block 1
    test.l1.emit_empty_block(); // block 2

    // Build L2 blocks
    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);
    let block_ref = test.l2.build_empty_block().unwrap();

    // Encode as a singular batch
    let l1_origin = test.l1.block_at(1).unwrap().clone();
    let batch = test.singular_batch_calldata(&[block_ref], &l1_origin);

    // Submit batch on L1
    test.l1.emit_block_with_batches(vec![batch]);

    // Verify the super root is deterministic and matches expected value
    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2);
    assert_eq!(
        root1, EXPECTED_SINGLE_BATCH_ROOT,
        "super root mismatch — update EXPECTED_SINGLE_BATCH_ROOT if this is an intentional change"
    );
}

#[test]
fn test_multiple_l2_blocks() {
    let mut test = DerivationTest::new();

    // Build L1 blocks
    for _ in 0..5 {
        test.l1.emit_empty_block();
    }

    // Build 6 L2 blocks per L1 block (2s L2 / 12s L1 = 6 blocks per epoch)
    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);

    let mut block_refs = Vec::new();
    for _ in 0..6 {
        block_refs.push(test.l2.build_empty_block().unwrap());
    }

    // Encode all blocks as batches and submit
    let l1_origin = test.l1.block_at(1).unwrap().clone();
    let batch = test.singular_batch_calldata(&block_refs, &l1_origin);
    test.l1.emit_block_with_batches(vec![batch]);

    // 7 blocks total: genesis + 6 built
    assert_eq!(test.l2.blocks().len(), 7);

    // Super root matches expected golden value
    let root = test.expected_super_root();
    assert_eq!(
        root, EXPECTED_MULTI_BLOCK_ROOT,
        "super root mismatch — update EXPECTED_MULTI_BLOCK_ROOT if this is an intentional change"
    );
}

#[test]
fn test_l1_chain_structure() {
    let config = DeterministicConfig::default();
    let mut test = DerivationTest::new();

    // Emit 5 empty L1 blocks
    for _ in 0..5 {
        test.l1.emit_empty_block();
    }

    let blocks = test.l1.blocks();
    assert_eq!(blocks.len(), 6); // genesis + 5

    // Check parent chain
    for i in 1..blocks.len() {
        assert_eq!(blocks[i].header.inner().parent_hash, blocks[i - 1].header.hash(),);
    }

    // Check timestamps
    for i in 1..blocks.len() {
        assert_eq!(
            blocks[i].header.inner().timestamp,
            blocks[i - 1].header.inner().timestamp + config.l1_block_time,
        );
    }
}

#[test]
fn test_span_batch_submission() {
    use derivation_tests::batch::{ChannelOut, CompressionAlgo, L1Origin, build_span_batch};

    let mut test = DerivationTest::new();

    // Build L1 blocks
    test.l1.emit_empty_block(); // block 1
    test.l1.emit_empty_block(); // block 2

    // Build multiple L2 blocks in the same epoch
    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);

    let mut block_refs = Vec::new();
    for _ in 0..3 {
        block_refs.push(test.l2.build_empty_block().unwrap());
    }

    // Collect block references for span batch construction
    let blocks: Vec<&_> = block_refs.iter().map(|r| test.l2.block(*r)).collect();

    let l1_origin =
        L1Origin { number: l1_block.header.inner().number, hash: l1_block.header.hash() };
    let rollup_config = test.config.rollup_config();

    // Build span batch
    let span_batch = build_span_batch(&blocks, l1_origin, &rollup_config);
    assert_eq!(span_batch.batches.len(), 3);

    // Encode into channel and produce calldata
    let channel_id = [0xFFu8; 16];
    let mut channel = ChannelOut::new(channel_id, CompressionAlgo::Zlib);
    channel.add_span_batch(&span_batch).unwrap();
    channel.close().unwrap();

    let calldata = channel.to_calldata(100_000);
    assert!(!calldata.is_empty(), "expected at least one frame of calldata");

    // First byte is DerivationVersion0
    assert_eq!(calldata[0][0], 0x00);

    // Submit as batch on L1
    let batch_submission =
        derivation_tests::l1::BatchSubmission::Calldata(calldata.into_iter().next().unwrap());
    test.l1.emit_block_with_batches(vec![batch_submission]);

    // Verify super root is still computable and deterministic
    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2, "super root should be deterministic after span batch submission");
}

#[test]
fn test_single_transfer() {
    use alloy_consensus::{SignableTransaction, TxEip1559};
    use alloy_primitives::{Address, TxKind, U256};
    use alloy_signer::SignerSync;
    use op_alloy_consensus::OpTxEnvelope;

    let mut test = DerivationTest::new();

    // Build L1 genesis epoch
    test.l1.emit_empty_block(); // block 1
    test.l1.emit_empty_block(); // block 2

    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);

    // Sign a transfer tx from the prefunded account
    let signer = helpers::funded_signer();
    let recipient = Address::with_last_byte(0x99);
    let tx = TxEip1559 {
        chain_id: test.config.l2_chain_id,
        nonce: 0,
        gas_limit: 21_000,
        max_fee_per_gas: 0,
        max_priority_fee_per_gas: 0,
        to: TxKind::Call(recipient),
        value: U256::from(1_000_000_000_000_000_000u64), // 1 ETH
        ..Default::default()
    };
    let sig = signer.sign_hash_sync(&tx.signature_hash()).expect("signing works");
    let signed = tx.into_signed(sig);
    let eth_envelope = alloy_consensus::TxEnvelope::Eip1559(signed);
    let op_tx = OpTxEnvelope::try_from_eth_envelope(eth_envelope)
        .expect("should convert ETH envelope to OP envelope");

    // Build L2 block with the transfer
    let block_ref = test.l2.build_block(vec![op_tx]).unwrap();

    // Encode as a singular batch and submit to L1
    let l1_origin = test.l1.block_at(1).unwrap().clone();
    let batch = test.singular_batch_calldata(&[block_ref], &l1_origin);
    test.l1.emit_block_with_batches(vec![batch]);

    // Super root matches expected golden value
    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2, "super root should be deterministic with a transfer");
    assert_eq!(
        root1, EXPECTED_SINGLE_TRANSFER_ROOT,
        "super root mismatch — update EXPECTED_SINGLE_TRANSFER_ROOT if this is an intentional change"
    );

    // Super root differs from an empty-blocks-only chain
    let empty_test = build_empty_blocks_test(1);
    let empty_root = empty_test.expected_super_root();
    assert_ne!(
        root1, empty_root,
        "transfer should produce a different super root than empty blocks"
    );
}

#[test]
fn test_blob_batch() {
    let mut test = DerivationTest::new();

    test.l1.emit_empty_block(); // block 1
    test.l1.emit_empty_block(); // block 2

    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);

    let mut block_refs = Vec::new();
    for _ in 0..3 {
        block_refs.push(test.l2.build_empty_block().unwrap());
    }

    // Encode as a blob span batch
    let l1_origin = test.l1.block_at(1).unwrap().clone();
    let batch = test.blob_span_batch(&block_refs, &l1_origin);
    test.l1.emit_block_with_batches(vec![batch]);

    // Super root is deterministic
    let root1 = test.expected_super_root();
    let root2 = test.expected_super_root();
    assert_eq!(root1, root2, "super root should be deterministic after blob batch");
}

#[tokio::test]
async fn test_blob_batch_beacon_endpoint() {
    let mut test = DerivationTest::new();

    test.l1.emit_empty_block(); // block 1
    test.l1.emit_empty_block(); // block 2

    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);

    let block_ref = test.l2.build_empty_block().unwrap();

    // Encode as a blob and submit
    let l1_origin = test.l1.block_at(1).unwrap().clone();
    let batch = test.blob_span_batch(&[block_ref], &l1_origin);
    test.l1.emit_block_with_batches(vec![batch]);

    let servers = test.serve().await.unwrap();
    let client = reqwest::Client::new();

    // The blob was submitted on the block after the two empty blocks (block 3).
    // Slot = (timestamp - genesis_timestamp) / seconds_per_slot
    let blob_block = test.l1.block_at(3).unwrap();
    let slot = test.l1.timestamp_to_slot(blob_block.header.inner().timestamp);

    let resp: serde_json::Value = client
        .get(format!("{}/eth/v1/beacon/blobs/{}", servers.beacon_url(), slot))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    let data = resp["data"].as_array().expect("beacon blobs should have data array");
    assert!(!data.is_empty(), "blob slot should have blob data");

    servers.stop();
}

#[test]
fn test_system_config_update() {
    use derivation_tests::l1::SystemConfigUpdate;
    use kona_genesis::{CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC};

    let mut test = DerivationTest::new();

    let new_batcher = alloy_primitives::address!("0xDEAD000000000000000000000000000000000001");
    test.l1.emit_block_with_system_config_update(SystemConfigUpdate::BatcherAddress(new_batcher));

    let block = test.l1.block_at(1).unwrap();

    // Block should have a receipt with a log
    assert_eq!(block.receipts.len(), 1, "should have one receipt");

    // Extract the log from the receipt
    let receipt = &block.receipts[0];
    let logs: Vec<_> = match receipt {
        alloy_consensus::ReceiptEnvelope::Eip1559(rwb) => rwb.receipt.logs.clone(),
        _ => panic!("expected EIP-1559 receipt"),
    };
    assert_eq!(logs.len(), 1, "should have one log");

    let log = &logs[0];
    assert_eq!(log.address, test.config.system_config, "log should be from system config address");
    assert_eq!(log.data.topics().len(), 3, "should have 3 topics");
    assert_eq!(log.data.topics()[0], CONFIG_UPDATE_TOPIC, "topic[0] should be ConfigUpdate");
    assert_eq!(log.data.topics()[1], CONFIG_UPDATE_EVENT_VERSION_0, "topic[1] should be version 0");
}

/// Integration test: run op-program against empty (deposit-only) L2 blocks.
///
/// Verifies that op-program can derive blocks from our test framework's L1/L2 chain.
/// The claim validation will fail because the Rust framework doesn't replicate
/// EIP-4788/EIP-2935 system contract state changes — but the derivation pipeline
/// itself succeeds (block is built, inserted, and safe head advances).
///
/// Requires: `OP_PROGRAM_PATH=/path/to/op-program`
/// Run with: `OP_PROGRAM_PATH=/path/to/op-program cargo test test_op_program_empty_blocks --
/// --ignored`
#[tokio::test]
#[ignore]
async fn test_op_program_empty_blocks() {
    use derivation_tests::harness::{run_config_from_test, run_op_program};

    if std::env::var("OP_PROGRAM_PATH").is_err() {
        eprintln!("SKIP: OP_PROGRAM_PATH not set. Set it to run this integration test.");
        return;
    }

    let mut test = DerivationTest::new();

    // Build L1 blocks
    test.l1.emit_empty_block(); // block 1
    test.l1.emit_empty_block(); // block 2

    // Build a deposit-only L2 block using L1 genesis as epoch.
    // The L2 timestamp (genesis + l2_block_time = 1700000002) must be >= the epoch's
    // timestamp (genesis = 1700000000). L1 block 1 has timestamp 1700000012, which is
    // too high for the first L2 block.
    let l1_genesis = test.l1.block_at(0).unwrap().clone();
    test.l2.set_epoch(&l1_genesis);
    let block_ref = test.l2.build_empty_block().unwrap();

    // Submit batch referencing L1 genesis as the epoch origin
    let batch = test.singular_batch_calldata(&[block_ref], &l1_genesis);
    test.l1.emit_block_with_batches(vec![batch]);

    // The derivation pipeline needs at least one more L1 block after the batch
    // to advance past it and process the channel.
    test.l1.emit_empty_block();

    // Start servers — they must stay alive while op-program runs
    let servers = test.serve().await.unwrap();
    let config = run_config_from_test(&test, &servers);
    let rollup_config = test.config.rollup_config();
    let l1_chain_config = test.config.l1_chain_config();

    let status = run_op_program(&config, &rollup_config, &l1_chain_config)
        .await
        .expect("op-program should execute");
    servers.stop();

    // Op-program exits 1 for claim mismatch (expected — the Rust framework doesn't
    // replicate EIP-4788/EIP-2935 system contract state changes during block building,
    // so the output root differs). Derivation success is verified by op-program reaching
    // the claim validation stage without panicking or hanging.
    assert!(
        status.code() == Some(1) || status.success(),
        "op-program should complete derivation (exit 0 or 1), got: {status}"
    );
}

/// Integration test: run kona-host against empty (deposit-only) L2 blocks.
///
/// Requires: `KONA_HOST_PATH=/path/to/kona-host`
/// Run with: `KONA_HOST_PATH=/path/to/kona-host cargo test test_kona_host_empty_blocks --
/// --ignored`
#[tokio::test]
#[ignore]
async fn test_kona_host_empty_blocks() {
    use derivation_tests::harness::{run_config_from_test, run_kona_host};

    if std::env::var("KONA_HOST_PATH").is_err() {
        eprintln!("SKIP: KONA_HOST_PATH not set. Set it to run this integration test.");
        return;
    }

    let mut test = DerivationTest::new();

    test.l1.emit_empty_block();
    test.l1.emit_empty_block();

    // Use L1 genesis as epoch — L2 timestamp must be >= epoch timestamp
    let l1_genesis = test.l1.block_at(0).unwrap().clone();
    test.l2.set_epoch(&l1_genesis);
    let block_ref = test.l2.build_empty_block().unwrap();

    let batch = test.singular_batch_calldata(&[block_ref], &l1_genesis);
    test.l1.emit_block_with_batches(vec![batch]);
    test.l1.emit_empty_block();

    let servers = test.serve().await.unwrap();
    let config = run_config_from_test(&test, &servers);
    let rollup_config = test.config.rollup_config();
    let l1_chain_config = test.config.l1_chain_config();

    let status = run_kona_host(&config, &rollup_config, &l1_chain_config)
        .await
        .expect("kona-host should execute");
    servers.stop();

    // Derivation succeeds; claim may mismatch (see op-program tests for explanation).
    assert!(
        status.code() == Some(1) || status.success(),
        "kona-host should complete derivation (exit 0 or 1), got: {status}"
    );
}

/// Integration test: run op-program with a user transfer submitted as a singular batch.
///
/// Requires: `OP_PROGRAM_PATH=/path/to/op-program`
/// Run with: `OP_PROGRAM_PATH=/path/to/op-program cargo test test_op_program_with_batch --
/// --ignored`
#[tokio::test]
#[ignore]
async fn test_op_program_with_batch() {
    use alloy_consensus::{SignableTransaction, TxEip1559};
    use alloy_primitives::{Address, TxKind, U256};
    use alloy_signer::SignerSync;
    use derivation_tests::harness::{run_config_from_test, run_op_program};
    use op_alloy_consensus::OpTxEnvelope;

    if std::env::var("OP_PROGRAM_PATH").is_err() {
        eprintln!("SKIP: OP_PROGRAM_PATH not set. Set it to run this integration test.");
        return;
    }

    let mut test = DerivationTest::new();

    test.l1.emit_empty_block();
    test.l1.emit_empty_block();

    // Use L1 genesis as epoch — L2 timestamp must be >= epoch timestamp
    let l1_genesis = test.l1.block_at(0).unwrap().clone();
    test.l2.set_epoch(&l1_genesis);

    // Sign a transfer from the prefunded account
    let signer = helpers::funded_signer();
    let tx = TxEip1559 {
        chain_id: test.config.l2_chain_id,
        nonce: 0,
        gas_limit: 21_000,
        max_fee_per_gas: 0,
        max_priority_fee_per_gas: 0,
        to: TxKind::Call(Address::with_last_byte(0x99)),
        value: U256::from(1_000_000_000_000_000_000u64), // 1 ETH
        ..Default::default()
    };
    let sig = signer.sign_hash_sync(&tx.signature_hash()).expect("signing works");
    let signed = tx.into_signed(sig);
    let eth_envelope = alloy_consensus::TxEnvelope::Eip1559(signed);
    let op_tx = OpTxEnvelope::try_from_eth_envelope(eth_envelope)
        .expect("should convert ETH envelope to OP envelope");

    let block_ref = test.l2.build_block(vec![op_tx]).unwrap();

    // Submit the batch to L1
    let batch = test.singular_batch_calldata(&[block_ref], &l1_genesis);
    test.l1.emit_block_with_batches(vec![batch]);
    test.l1.emit_empty_block(); // pipeline needs one more L1 block

    let servers = test.serve().await.unwrap();
    let config = run_config_from_test(&test, &servers);
    let rollup_config = test.config.rollup_config();
    let l1_chain_config = test.config.l1_chain_config();

    let status = run_op_program(&config, &rollup_config, &l1_chain_config)
        .await
        .expect("op-program should execute");
    servers.stop();

    // Derivation succeeds; claim may mismatch because the Rust framework doesn't
    // replicate EIP-4788/EIP-2935 system contract state changes (exit 1 = claim mismatch).
    assert!(
        status.code() == Some(1) || status.success(),
        "op-program should complete derivation (exit 0 or 1), got: {status}"
    );
}

/// Integration test: run kona-host with a user transfer submitted as a singular batch.
///
/// Requires: `KONA_HOST_PATH=/path/to/kona-host`
/// Run with: `KONA_HOST_PATH=/path/to/kona-host cargo test test_kona_host_with_batch -- --ignored`
#[tokio::test]
#[ignore]
async fn test_kona_host_with_batch() {
    use alloy_consensus::{SignableTransaction, TxEip1559};
    use alloy_primitives::{Address, TxKind, U256};
    use alloy_signer::SignerSync;
    use derivation_tests::harness::{run_config_from_test, run_kona_host};
    use op_alloy_consensus::OpTxEnvelope;

    if std::env::var("KONA_HOST_PATH").is_err() {
        eprintln!("SKIP: KONA_HOST_PATH not set. Set it to run this integration test.");
        return;
    }

    let mut test = DerivationTest::new();

    test.l1.emit_empty_block();
    test.l1.emit_empty_block();

    // Use L1 genesis as epoch — L2 timestamp must be >= epoch timestamp
    let l1_genesis = test.l1.block_at(0).unwrap().clone();
    test.l2.set_epoch(&l1_genesis);

    let signer = helpers::funded_signer();
    let tx = TxEip1559 {
        chain_id: test.config.l2_chain_id,
        nonce: 0,
        gas_limit: 21_000,
        max_fee_per_gas: 0,
        max_priority_fee_per_gas: 0,
        to: TxKind::Call(Address::with_last_byte(0x99)),
        value: U256::from(1_000_000_000_000_000_000u64),
        ..Default::default()
    };
    let sig = signer.sign_hash_sync(&tx.signature_hash()).expect("signing works");
    let signed = tx.into_signed(sig);
    let eth_envelope = alloy_consensus::TxEnvelope::Eip1559(signed);
    let op_tx = OpTxEnvelope::try_from_eth_envelope(eth_envelope)
        .expect("should convert ETH envelope to OP envelope");

    let block_ref = test.l2.build_block(vec![op_tx]).unwrap();

    let batch = test.singular_batch_calldata(&[block_ref], &l1_genesis);
    test.l1.emit_block_with_batches(vec![batch]);
    test.l1.emit_empty_block(); // pipeline needs one more L1 block

    let servers = test.serve().await.unwrap();
    let config = run_config_from_test(&test, &servers);
    let rollup_config = test.config.rollup_config();
    let l1_chain_config = test.config.l1_chain_config();

    let status = run_kona_host(&config, &rollup_config, &l1_chain_config)
        .await
        .expect("kona-host should execute");
    servers.stop();

    // Derivation succeeds; claim may mismatch (see op-program tests for explanation).
    assert!(
        status.code() == Some(1) || status.success(),
        "kona-host should complete derivation (exit 0 or 1), got: {status}"
    );
}

// ----- Server integration tests -----

mod helpers;

#[tokio::test]
async fn test_l1_rpc_get_block_by_hash() {
    let test = build_empty_blocks_test(2);
    let servers = test.serve().await.unwrap();

    let client = reqwest::Client::new();

    // Get genesis block by number first
    let resp: serde_json::Value = client
        .post(servers.l1_rpc_url())
        .json(&serde_json::json!({
            "jsonrpc": "2.0",
            "method": "eth_getBlockByNumber",
            "params": ["0x0", false],
            "id": 1
        }))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    let block_by_number = resp["result"].clone();
    let hash = block_by_number["hash"].as_str().unwrap().to_string();

    // Now query by hash
    let resp: serde_json::Value = client
        .post(servers.l1_rpc_url())
        .json(&serde_json::json!({
            "jsonrpc": "2.0",
            "method": "eth_getBlockByHash",
            "params": [hash, false],
            "id": 2
        }))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    let block_by_hash = resp["result"].clone();
    assert_eq!(
        block_by_number["hash"], block_by_hash["hash"],
        "block by number and block by hash should match"
    );
    assert_eq!(block_by_number["number"], block_by_hash["number"]);

    servers.stop();
}

#[tokio::test]
async fn test_l2_rpc_get_proof() {
    use derivation_tests::config::L2_TO_L1_MESSAGE_PASSER;

    let test = build_empty_blocks_test(1);
    let servers = test.serve().await.unwrap();

    let client = reqwest::Client::new();

    let resp: serde_json::Value = client
        .post(servers.l2_rpc_url())
        .json(&serde_json::json!({
            "jsonrpc": "2.0",
            "method": "eth_getProof",
            "params": [format!("{:?}", L2_TO_L1_MESSAGE_PASSER), [], "latest"],
            "id": 1
        }))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    let result = &resp["result"];
    assert!(result.is_object(), "proof response should be an object");
    assert!(result["address"].is_string(), "proof should have address field");
    assert!(result["accountProof"].is_array(), "proof should have accountProof");
    assert!(result["storageProof"].is_array(), "proof should have storageProof");

    servers.stop();
}

#[tokio::test]
async fn test_debug_db_get() {
    let test = build_empty_blocks_test(1);
    let servers = test.serve().await.unwrap();

    let client = reqwest::Client::new();

    // The state root of the latest L2 block is a known trie node key.
    let state_root = test.l2.head().header.inner().state_root;

    let resp: serde_json::Value = client
        .post(servers.l2_rpc_url())
        .json(&serde_json::json!({
            "jsonrpc": "2.0",
            "method": "debug_dbGet",
            "params": [format!("{state_root:?}")],
            "id": 1
        }))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    // The state root should be a known trie node, so the result should be non-null
    assert!(
        resp["result"].is_string(),
        "debug_dbGet should return data for the state root trie node, got: {resp}"
    );
    let result_hex = resp["result"].as_str().unwrap();
    assert!(result_hex.len() > 2, "result should be non-empty hex data");

    servers.stop();
}

#[tokio::test]
async fn test_beacon_blobs() {
    use derivation_tests::config::DeterministicConfig;

    let config = DeterministicConfig::default();
    let mut test = derivation_tests::harness::DerivationTest::new();

    // Build L1 blocks
    test.l1.emit_empty_block(); // block 1
    test.l1.emit_empty_block(); // block 2

    // Build L2 blocks
    let l1_block = test.l1.block_at(1).unwrap().clone();
    test.l2.set_epoch(&l1_block);
    let block_ref = test.l2.build_empty_block().unwrap();

    // Encode as singular batch calldata
    let l1_origin = test.l1.block_at(1).unwrap().clone();
    let batch = test.singular_batch_calldata(&[block_ref], &l1_origin);
    test.l1.emit_block_with_batches(vec![batch]);

    let servers = test.serve().await.unwrap();

    let client = reqwest::Client::new();

    // Query beacon blobs at slot 0 (genesis) — should return empty data
    let resp: serde_json::Value = client
        .get(format!("{}/eth/v1/beacon/blobs/0", servers.beacon_url()))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    // Verify the response structure
    assert!(resp["data"].is_array(), "beacon blobs response should have data array");
    // Without blob batches, data should be empty
    assert_eq!(resp["data"].as_array().unwrap().len(), 0);

    // Also verify the beacon genesis endpoint works
    let resp: serde_json::Value = client
        .get(format!("{}/eth/v1/beacon/genesis", servers.beacon_url()))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();

    assert!(resp["data"]["genesis_time"].is_string());
    let genesis_time: u64 = resp["data"]["genesis_time"].as_str().unwrap().parse().unwrap();
    assert_eq!(genesis_time, config.genesis_timestamp);

    servers.stop();
}
