use alloy_consensus::Transaction;
use alloy_eips::Decodable2718;
use alloy_primitives::{Address, TxHash, U256};
use alloy_provider::Provider;
use macros::rb_test;
use op_alloy_consensus::OpTxEnvelope;
use std::time::Duration;

use crate::{
    args::{FlashblocksArgs, OpRbuilderArgs},
    tests::{
        BlockTransactionsExt, BundleOpts, ChainDriver, FLASHBLOCKS_NUMBER_ADDRESS, LocalInstance,
        TransactionBuilderExt, flashblocks_number_contract::FlashblocksNumber,
    },
};

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 2000,
    flashblocks: FlashblocksArgs {
        enabled: true,
        flashblocks_port: 1239,
        flashblocks_addr: "127.0.0.1".into(),
        flashblocks_block_time: 200,
        flashblocks_leeway_time: 100,
        flashblocks_fixed: false,
        ..Default::default()
    },
    ..Default::default()
})]
async fn smoke_dynamic_base(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let flashblocks_listener = rbuilder.spawn_flashblocks_listener();

    // We align out block timestamps with current unix timestamp
    for _ in 0..10 {
        for _ in 0..5 {
            // send a valid transaction
            let _ = driver
                .create_transaction()
                .random_valid_transfer()
                .send()
                .await?;
        }
        let block = driver.build_new_block_with_current_timestamp(None).await?;
        assert_eq!(block.transactions.len(), 8, "Got: {:?}", block.transactions); // 5 normal txn + deposit + 2 builder txn
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;
    }

    let flashblocks = flashblocks_listener.get_flashblocks();
    assert_eq!(110, flashblocks.len());

    flashblocks_listener.stop().await
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    flashblocks: FlashblocksArgs {
        enabled: true,
        flashblocks_port: 1239,
        flashblocks_addr: "127.0.0.1".into(),
        flashblocks_block_time: 200,
        flashblocks_leeway_time: 100,
        flashblocks_fixed: false,
        ..Default::default()
    },
    ..Default::default()
})]
async fn smoke_dynamic_unichain(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let flashblocks_listener = rbuilder.spawn_flashblocks_listener();

    // We align out block timestamps with current unix timestamp
    for _ in 0..10 {
        for _ in 0..5 {
            // send a valid transaction
            let _ = driver
                .create_transaction()
                .random_valid_transfer()
                .send()
                .await?;
        }
        let block = driver.build_new_block_with_current_timestamp(None).await?;
        assert_eq!(block.transactions.len(), 8, "Got: {:?}", block.transactions); // 5 normal txn + deposit + 2 builder txn
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;
    }

    let flashblocks = flashblocks_listener.get_flashblocks();
    assert_eq!(60, flashblocks.len());

    flashblocks_listener.stop().await
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    flashblocks: FlashblocksArgs {
        enabled: true,
        flashblocks_port: 1239,
        flashblocks_addr: "127.0.0.1".into(),
        flashblocks_block_time: 200,
        flashblocks_leeway_time: 50,
        flashblocks_fixed: true,
        ..Default::default()
    },
    ..Default::default()
})]
async fn smoke_classic_unichain(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let flashblocks_listener = rbuilder.spawn_flashblocks_listener();

    // We align out block timestamps with current unix timestamp
    for _ in 0..10 {
        for _ in 0..5 {
            // send a valid transaction
            let _ = driver
                .create_transaction()
                .random_valid_transfer()
                .send()
                .await?;
        }
        let block = driver.build_new_block().await?;
        assert_eq!(block.transactions.len(), 8, "Got: {:?}", block.transactions); // 5 normal txn + deposit + 2 builder txn
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;
    }

    let flashblocks = flashblocks_listener.get_flashblocks();
    assert_eq!(60, flashblocks.len());

    flashblocks_listener.stop().await
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 2000,
    flashblocks: FlashblocksArgs {
        enabled: true,
        flashblocks_port: 1239,
        flashblocks_addr: "127.0.0.1".into(),
        flashblocks_block_time: 200,
        flashblocks_leeway_time: 50,
        flashblocks_fixed: true,
        ..Default::default()
    },
    ..Default::default()
})]
async fn smoke_classic_base(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let flashblocks_listener = rbuilder.spawn_flashblocks_listener();

    // We align out block timestamps with current unix timestamp
    for _ in 0..10 {
        for _ in 0..5 {
            // send a valid transaction
            let _ = driver
                .create_transaction()
                .random_valid_transfer()
                .send()
                .await?;
        }
        let block = driver.build_new_block().await?;
        assert_eq!(block.transactions.len(), 8, "Got: {:?}", block.transactions); // 5 normal txn + deposit + 2 builder txn
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;
    }

    let flashblocks = flashblocks_listener.get_flashblocks();
    assert_eq!(110, flashblocks.len());

    flashblocks_listener.stop().await
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    flashblocks: FlashblocksArgs {
        enabled: true,
        flashblocks_port: 1239,
        flashblocks_addr: "127.0.0.1".into(),
        flashblocks_block_time: 200,
        flashblocks_leeway_time: 100,
        flashblocks_fixed: false,
        ..Default::default()
    },
    ..Default::default()
})]
async fn unichain_dynamic_with_lag(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let flashblocks_listener = rbuilder.spawn_flashblocks_listener();

    // We align out block timestamps with current unix timestamp
    for i in 0..9 {
        for _ in 0..5 {
            // send a valid transaction
            let _ = driver
                .create_transaction()
                .random_valid_transfer()
                .send()
                .await?;
        }
        let block = driver
            .build_new_block_with_current_timestamp(Some(Duration::from_millis(i * 100)))
            .await?;
        assert_eq!(
            block.transactions.len(),
            8,
            "Got: {:#?}",
            block.transactions
        ); // 5 normal txn + deposit + 2 builder txn
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;
    }

    let flashblocks = flashblocks_listener.get_flashblocks();
    assert_eq!(34, flashblocks.len());

    flashblocks_listener.stop().await
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    flashblocks: FlashblocksArgs {
        enabled: true,
        flashblocks_port: 1239,
        flashblocks_addr: "127.0.0.1".into(),
        flashblocks_block_time: 200,
        flashblocks_leeway_time: 0,
        flashblocks_fixed: false,
        ..Default::default()
    },
    ..Default::default()
})]
async fn dynamic_with_full_block_lag(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let flashblocks_listener = rbuilder.spawn_flashblocks_listener();

    for _ in 0..5 {
        // send a valid transaction
        let _ = driver
            .create_transaction()
            .random_valid_transfer()
            .send()
            .await?;
    }
    let block = driver
        .build_new_block_with_current_timestamp(Some(Duration::from_millis(999)))
        .await?;
    // We could only produce block with deposits + builder tx because of short time frame
    assert_eq!(block.transactions.len(), 2);

    let flashblocks = flashblocks_listener.get_flashblocks();
    assert_eq!(1, flashblocks.len());

    flashblocks_listener.stop().await
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashblocks: FlashblocksArgs {
        enabled: true,
        flashblocks_port: 1239,
        flashblocks_addr: "127.0.0.1".into(),
        flashblocks_block_time: 200,
        flashblocks_leeway_time: 100,
        flashblocks_fixed: false,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashblock_min_filtering(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let flashblocks_listener = rbuilder.spawn_flashblocks_listener();

    // Create two transactions and set their tips so that while ordinarily
    // tx2 would come before tx1 because its tip is bigger, now tx1 comes
    // first because it has a lower minimum flashblock requirement.
    let tx1 = driver
        .create_transaction()
        .random_valid_transfer()
        .with_bundle(BundleOpts::default().with_flashblock_number_min(0))
        .with_max_priority_fee_per_gas(0)
        .send()
        .await?;

    let tx2 = driver
        .create_transaction()
        .random_valid_transfer()
        .with_bundle(BundleOpts::default().with_flashblock_number_min(3))
        .with_max_priority_fee_per_gas(10)
        .send()
        .await?;

    let _block1 = driver.build_new_block_with_current_timestamp(None).await?;

    // Check that tx1 comes before tx2
    let tx1_hash = *tx1.tx_hash();
    let tx2_hash = *tx2.tx_hash();
    let tx1_pos = flashblocks_listener
        .find_transaction_flashblock(&tx1_hash)
        .unwrap();
    let tx2_pos = flashblocks_listener
        .find_transaction_flashblock(&tx2_hash)
        .unwrap();

    assert!(
        tx1_pos < tx2_pos,
        "tx {tx1_hash:?} does not come before {tx2_hash:?}"
    );

    let flashblocks = flashblocks_listener.get_flashblocks();
    assert_eq!(6, flashblocks.len());

    flashblocks_listener.stop().await
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashblocks: FlashblocksArgs {
        enabled: true,
        flashblocks_port: 1239,
        flashblocks_addr: "127.0.0.1".into(),
        flashblocks_block_time: 200,
        flashblocks_leeway_time: 100,
        flashblocks_fixed: false,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashblock_max_filtering(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let flashblocks_listener = rbuilder.spawn_flashblocks_listener();

    // Since we cannot directly trigger flashblock creation in tests, we
    // instead fill up the gas of flashblocks so that our tx with the
    // flashblock_number_max parameter set is properly delayed, simulating
    // the scenario where we'd sent the tx after the flashblock max number
    // had passed.
    let call = driver
        .provider()
        .raw_request::<(i32, i32), bool>("miner_setMaxDASize".into(), (0, 100 * 3))
        .await?;
    assert!(call, "miner_setMaxDASize should be executed successfully");

    let _fit_tx_1 = driver
        .create_transaction()
        .with_max_priority_fee_per_gas(50)
        .send()
        .await?;

    let tx1 = driver
        .create_transaction()
        .random_valid_transfer()
        .with_bundle(BundleOpts::default().with_flashblock_number_max(1))
        .send()
        .await?;

    let block = driver.build_new_block_with_current_timestamp(None).await?;
    assert!(!block.includes(tx1.tx_hash()));
    assert!(
        flashblocks_listener
            .find_transaction_flashblock(tx1.tx_hash())
            .is_none()
    );

    let flashblocks = flashblocks_listener.get_flashblocks();
    assert_eq!(6, flashblocks.len());

    flashblocks_listener.stop().await
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashblocks: FlashblocksArgs {
        enabled: true,
        flashblocks_port: 1239,
        flashblocks_addr: "127.0.0.1".into(),
        flashblocks_block_time: 200,
        flashblocks_leeway_time: 100,
        flashblocks_fixed: false,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashblock_min_max_filtering(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let flashblocks_listener = rbuilder.spawn_flashblocks_listener();

    let tx1 = driver
        .create_transaction()
        .random_valid_transfer()
        .with_bundle(
            BundleOpts::default()
                .with_flashblock_number_max(2)
                .with_flashblock_number_min(2),
        )
        .send()
        .await?;

    let _block = driver.build_new_block_with_current_timestamp(None).await?;

    // It ends up in the 2nd flashblock
    assert_eq!(
        2,
        flashblocks_listener
            .find_transaction_flashblock(tx1.tx_hash())
            .unwrap(),
        "Transaction should be in the 2nd flashblock"
    );

    let flashblocks = flashblocks_listener.get_flashblocks();
    assert_eq!(6, flashblocks.len(), "Flashblocks length should be 6");

    flashblocks_listener.stop().await
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    flashblocks: FlashblocksArgs {
        enabled: true,
        flashblocks_port: 1239,
        flashblocks_addr: "127.0.0.1".into(),
        flashblocks_block_time: 200,
        flashblocks_leeway_time: 100,
        flashblocks_fixed: false,
        flashblocks_disable_state_root: true,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashblocks_no_state_root_calculation(rbuilder: LocalInstance) -> eyre::Result<()> {
    use alloy_primitives::B256;

    let driver = rbuilder.driver().await?;

    // Send a transaction to ensure block has some activity
    let _tx = driver
        .create_transaction()
        .random_valid_transfer()
        .send()
        .await?;

    // Build a block with current timestamp (not historical) and disable_state_root: true
    let block = driver.build_new_block_with_current_timestamp(None).await?;

    // Verify that flashblocks are still produced (block should have transactions)
    assert!(
        block.transactions.len() > 2,
        "Block should contain transactions"
    ); // deposit + builder tx + user tx

    // Verify that state root is not calculated (should be zero)
    assert_eq!(
        block.header.state_root,
        B256::ZERO,
        "State root should be zero when disable_state_root is true"
    );

    Ok(())
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashblocks: FlashblocksArgs {
        flashblocks_number_contract_address: Some(FLASHBLOCKS_NUMBER_ADDRESS),
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashblocks_number_contract_builder_tx(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let flashblocks_listener = rbuilder.spawn_flashblocks_listener();
    let provider = rbuilder.provider().await?;

    // Deploy flashblocks number contract which will be in flashblocks 1
    let deploy_tx = driver
        .create_transaction()
        .deploy_flashblock_number_contract()
        .with_bundle(BundleOpts::default())
        .send()
        .await?;

    // Create valid transactions for flashblocks 2-4
    let user_transactions = create_flashblock_transactions(&driver, 2..5).await?;

    // Build block with deploy tx in first flashblock, and a random valid transfer in every other flashblock
    let block = driver.build_new_block_with_current_timestamp(None).await?;

    // Verify contract deployment
    let receipt = provider
        .get_transaction_receipt(*deploy_tx.tx_hash())
        .await?
        .expect("flashblock number contract deployment not mined");
    let contract_address = receipt
        .inner
        .contract_address
        .expect("contract receipt does not contain flashblock number contract address");
    assert_eq!(
        contract_address, FLASHBLOCKS_NUMBER_ADDRESS,
        "Flashblocks number contract address mismatch"
    );

    // Verify first block structure
    assert_eq!(block.transactions.len(), 10);
    let txs = block
        .transactions
        .as_transactions()
        .expect("transactions not in block");

    // Verify builder txs (should be regular since builder tx is not registered yet)
    verify_builder_txs(
        &txs,
        &[1, 2, 4, 6, 8],
        Some(Address::ZERO),
        "Should have regular builder tx",
    );

    // Verify deploy tx position
    assert_eq!(
        txs[3].inner.inner.tx_hash(),
        *deploy_tx.tx_hash(),
        "Deploy tx not in correct position"
    );

    // Verify user transactions
    verify_user_tx_hashes(&txs, &[5, 7, 9], &user_transactions);

    // Initialize contract
    let init_tx = driver
        .create_transaction()
        .init_flashblock_number_contract(true)
        .with_to(FLASHBLOCKS_NUMBER_ADDRESS)
        .with_bundle(BundleOpts::default())
        .send()
        .await?;

    // Mine initialization
    driver.build_new_block_with_current_timestamp(None).await?;
    provider
        .get_transaction_receipt(*init_tx.tx_hash())
        .await?
        .expect("init tx not mined");

    // Create user transactions for flashblocks 1 - 5
    let user_transactions = create_flashblock_transactions(&driver, 1..5).await?;

    // Build second block after initialization which will call the flashblock number contract
    // with builder registered
    let block = driver.build_new_block_with_current_timestamp(None).await?;
    assert_eq!(block.transactions.len(), 10);
    let txs = block
        .transactions
        .as_transactions()
        .expect("transactions not in block");

    // Fallback block should have regular builder tx after deposit tx
    assert_eq!(
        txs[1].to(),
        Some(Address::ZERO),
        "Fallback block should have regular builder tx"
    );

    // Other builder txs should call the contract
    verify_builder_txs(
        &txs,
        &[2, 4, 6, 8],
        Some(contract_address),
        "Should call flashblocks contract",
    );

    // Verify user transactions, 3 blocks in total built
    verify_user_tx_hashes(&txs, &[3, 5, 7, 9], &user_transactions);

    // Verify flashblock number incremented correctly
    let contract = FlashblocksNumber::new(contract_address, provider.clone());
    let current_number = contract.getFlashblockNumber().call().await?;
    assert_eq!(
        current_number,
        U256::from(7),
        "Flashblock number not incremented correctly"
    );

    // Verify flashblocks
    let flashblocks = flashblocks_listener.get_flashblocks();
    assert_eq!(flashblocks.len(), 15);

    // Verify builder tx in each flashblock
    for (i, flashblock) in flashblocks.iter().enumerate() {
        // In fallback blocks, builder tx is the 2nd tx (index 1)
        // In regular flashblocks, builder tx is the 1st tx (index 0)
        let is_fallback = i % 5 == 0;
        let tx_index = if is_fallback { 1 } else { 0 };

        let tx_bytes = flashblock.diff.transactions.get(tx_index).expect(&format!(
            "Flashblock {} should have tx at index {}",
            i, tx_index
        ));
        let tx = OpTxEnvelope::decode_2718(&mut tx_bytes.as_ref())
            .expect("failed to decode transaction");

        let expected_to = if i < 7 || i == 10 {
            Some(Address::ZERO)
        } else {
            Some(contract_address)
        };

        assert_eq!(
            tx.to(),
            expected_to,
            "Flashblock {} builder tx (at index {}) should have to = {:?}",
            i,
            tx_index,
            expected_to
        );
    }

    flashblocks_listener.stop().await?;
    Ok(())
}

// Helper to create transactions for flashblocks
async fn create_flashblock_transactions(
    driver: &ChainDriver,
    range: std::ops::Range<u64>,
) -> eyre::Result<Vec<TxHash>> {
    let mut txs = Vec::new();
    for i in range {
        let tx = driver
            .create_transaction()
            .random_valid_transfer()
            .with_bundle(BundleOpts::default().with_flashblock_number_min(i))
            .send()
            .await?;
        txs.push(*tx.tx_hash());
    }
    Ok(txs)
}

// Helper to verify builder transactions
fn verify_builder_txs(
    block_txs: &[impl Transaction],
    indices: &[usize],
    expected_to: Option<Address>,
    msg: &str,
) {
    for &idx in indices {
        assert_eq!(block_txs[idx].to(), expected_to, "{} at index {}", msg, idx);
    }
}

// Helper to verify transaction matches
fn verify_user_tx_hashes(
    block_txs: &[impl AsRef<OpTxEnvelope>],
    indices: &[usize],
    expected_txs: &[TxHash],
) {
    for (i, &idx) in indices.iter().enumerate() {
        assert_eq!(
            *block_txs[idx].as_ref().tx_hash(),
            expected_txs[i],
            "Transaction at index {} doesn't match",
            idx
        );
    }
}
