//! Integration tests that run op-program and kona-host against test chains.
//!
//! These tests require external binaries and are marked `#[ignore]`.
//! Run with: `just test-derivation-integration` (from rust/)

mod helpers;

use derivation_tests::harness::{
    DerivationTest, run_config_from_test, run_kona_host, run_op_program,
};

/// Integration test: run op-program against empty (deposit-only) L2 blocks.
///
/// Requires: `OP_PROGRAM_PATH`
#[tokio::test]
#[ignore]
async fn test_op_program_empty_blocks() {
    if std::env::var("OP_PROGRAM_PATH").is_err() {
        eprintln!("SKIP: OP_PROGRAM_PATH not set.");
        return;
    }

    let mut test = DerivationTest::new();

    test.l1.emit_empty_block();
    test.l1.emit_empty_block();

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

    let status = run_op_program(&config, &rollup_config, &l1_chain_config)
        .await
        .expect("op-program should execute");
    servers.stop();

    assert!(status.success(), "op-program should exit 0 (claim match), got: {status}");
}

/// Integration test: run kona-host against empty (deposit-only) L2 blocks.
///
/// Requires: `KONA_HOST_PATH`
#[tokio::test]
#[ignore]
async fn test_kona_host_empty_blocks() {
    if std::env::var("KONA_HOST_PATH").is_err() {
        eprintln!("SKIP: KONA_HOST_PATH not set.");
        return;
    }

    let mut test = DerivationTest::new();

    test.l1.emit_empty_block();
    test.l1.emit_empty_block();

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

    assert!(status.success(), "kona-host should exit 0 (claim match), got: {status}");
}

/// Integration test: run op-program with a user transfer submitted as a singular batch.
///
/// Requires: `OP_PROGRAM_PATH`
#[tokio::test]
#[ignore]
async fn test_op_program_with_batch() {
    use alloy_consensus::{SignableTransaction, TxEip1559};
    use alloy_primitives::{Address, TxKind, U256};
    use alloy_signer::SignerSync;
    use op_alloy_consensus::OpTxEnvelope;

    if std::env::var("OP_PROGRAM_PATH").is_err() {
        eprintln!("SKIP: OP_PROGRAM_PATH not set.");
        return;
    }

    let mut test = DerivationTest::new();

    test.l1.emit_empty_block();
    test.l1.emit_empty_block();

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
    test.l1.emit_empty_block();

    let servers = test.serve().await.unwrap();
    let config = run_config_from_test(&test, &servers);
    let rollup_config = test.config.rollup_config();
    let l1_chain_config = test.config.l1_chain_config();

    let status = run_op_program(&config, &rollup_config, &l1_chain_config)
        .await
        .expect("op-program should execute");
    servers.stop();

    assert!(status.success(), "op-program should exit 0 (claim match), got: {status}");
}

/// Integration test: run kona-host with a user transfer submitted as a singular batch.
///
/// Requires: `KONA_HOST_PATH`
#[tokio::test]
#[ignore]
async fn test_kona_host_with_batch() {
    use alloy_consensus::{SignableTransaction, TxEip1559};
    use alloy_primitives::{Address, TxKind, U256};
    use alloy_signer::SignerSync;
    use op_alloy_consensus::OpTxEnvelope;

    if std::env::var("KONA_HOST_PATH").is_err() {
        eprintln!("SKIP: KONA_HOST_PATH not set.");
        return;
    }

    let mut test = DerivationTest::new();

    test.l1.emit_empty_block();
    test.l1.emit_empty_block();

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
    test.l1.emit_empty_block();

    let servers = test.serve().await.unwrap();
    let config = run_config_from_test(&test, &servers);
    let rollup_config = test.config.rollup_config();
    let l1_chain_config = test.config.l1_chain_config();

    let status = run_kona_host(&config, &rollup_config, &l1_chain_config)
        .await
        .expect("kona-host should execute");
    servers.stop();

    assert!(status.success(), "kona-host should exit 0 (claim match), got: {status}");
}
