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

    // Debug: print deposit tx details
    let l2_head = test.l2.head();
    if let op_alloy_consensus::OpTxEnvelope::Deposit(dep) = &l2_head.transactions[0] {
        let input = &dep.inner().input;
        eprintln!("=== Deposit tx ===");
        eprintln!("  calldata len: {}", input.len());
        eprintln!("  gas_limit:    {}", dep.inner().gas_limit);
        let zero_bytes = input.iter().filter(|&&b| b == 0).count();
        let non_zero_bytes = input.len() - zero_bytes;
        eprintln!("  zero bytes:   {}", zero_bytes);
        eprintln!("  non_zero:     {}", non_zero_bytes);
        let standard_gas = 21000 + 4 * zero_bytes as u64 + 16 * non_zero_bytes as u64;
        let floor_gas = 21000 + 10 * zero_bytes as u64 + 40 * non_zero_bytes as u64;
        eprintln!("  standard intrinsic: {}", standard_gas);
        eprintln!("  EIP-7623 floor:     {}", floor_gas);
    }
    let l2_header = l2_head.header.inner();
    eprintln!("=== Framework L2 block 1 ===");
    eprintln!("  block_hash:    {:?}", l2_head.header.hash());
    eprintln!("  parent_hash:   {:?}", l2_header.parent_hash);
    eprintln!("  state_root:    {:?}", l2_header.state_root);
    eprintln!("  receipts_root: {:?}", l2_header.receipts_root);
    eprintln!("  tx_root:       {:?}", l2_header.transactions_root);
    eprintln!("  gas_used:      {}", l2_header.gas_used);
    eprintln!("  timestamp:     {}", l2_header.timestamp);
    eprintln!("  basefee:       {:?}", l2_header.base_fee_per_gas);
    eprintln!("  beacon_root:   {:?}", l2_header.parent_beacon_block_root);
    eprintln!("  extra_data:    {:?}", l2_header.extra_data);
    eprintln!("  withdrawals:   {:?}", l2_header.withdrawals_root);
    eprintln!("  blob_gas_used: {:?}", l2_header.blob_gas_used);
    eprintln!("  excess_blob:   {:?}", l2_header.excess_blob_gas);
    eprintln!("  number:        {}", l2_header.number);
    eprintln!("  gas_limit:     {}", l2_header.gas_limit);
    // Print receipt details
    for (i, receipt) in l2_head.receipts.iter().enumerate() {
        eprintln!("  receipt[{i}]: gas={} type={:?}", receipt.cumulative_gas_used(), receipt.tx_type());
        if let op_alloy_consensus::OpReceiptEnvelope::Deposit(rwb) = receipt {
            eprintln!("    deposit_nonce: {:?}", rwb.receipt.deposit_nonce);
            eprintln!("    deposit_receipt_version: {:?}", rwb.receipt.deposit_receipt_version);
        }
    }
    // Print output root
    let output_root = test.expected_output_root();
    eprintln!("  output_root:   {:?}", output_root);

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
        max_fee_per_gas: 1,
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
        max_fee_per_gas: 1,
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
