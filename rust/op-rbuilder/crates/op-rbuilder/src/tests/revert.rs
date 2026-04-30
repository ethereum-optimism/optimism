use alloy_provider::{PendingTransactionBuilder, Provider};
use macros::{if_flashblocks, if_standard, rb_test};
use op_alloy_network::Optimism;

use crate::{
    args::OpRbuilderArgs,
    primitives::bundle::MAX_BLOCK_RANGE_BLOCKS,
    tests::{
        BlockTransactionsExt, BundleOpts, ChainDriver, ChainDriverExt, LocalInstance, ONE_ETH,
        OpRbuilderArgsTestExt, TransactionBuilderExt,
    },
};

/// This test ensures that the transactions that get reverted and not included in the block,
/// are eventually dropped from the pool once their block range is reached.
/// This test creates N transactions with different block ranges.
#[rb_test(args = OpRbuilderArgs {
    enable_revert_protection: true,
    ..Default::default()
})]
async fn monitor_transaction_gc(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let accounts = driver.fund_accounts(10, ONE_ETH).await?;
    let latest_block_number = driver.latest().await?.header.number;

    // send 10 bundles with different block ranges
    let mut pending_txn = Vec::new();

    for i in 0..accounts.len() {
        let txn = driver
            .create_transaction()
            .random_reverting_transaction()
            .with_signer(accounts[i].clone())
            .with_bundle(
                BundleOpts::default().with_block_number_max(latest_block_number + i as u64 + 1),
            )
            .send()
            .await?;
        pending_txn.push(txn);
    }

    // generate 10 blocks
    for i in 0..10 {
        let generated_block = driver.build_new_block_with_current_timestamp(None).await?;

        if_standard! {
            // standard builder blocks should only include two transactions (deposit + builder)
            assert_eq!(generated_block.transactions.len(), 2);
        }

        if_flashblocks! {
            // flashblocks should include three transactions (deposit + 2 builder txs)
            assert_eq!(generated_block.transactions.len(), 3);
        }

        // since we created the 10 transactions with increasing block ranges, as we generate blocks
        // one transaction will be gc on each block.
        // transactions from [0, i] should be dropped, transactions from [i+1, 10] should be queued
        for j in 0..=i {
            assert!(rbuilder.pool().is_dropped(*pending_txn[j].tx_hash()));
        }
        for j in i + 1..10 {
            assert!(rbuilder.pool().is_pending(*pending_txn[j].tx_hash()));
        }
    }

    Ok(())
}

/// If revert protection is disabled, the transactions that revert are included in the block.
#[rb_test]
async fn disabled(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;

    for _ in 0..10 {
        let valid_tx = driver
            .create_transaction()
            .random_valid_transfer()
            .send()
            .await?;

        let reverting_tx = driver
            .create_transaction()
            .random_reverting_transaction()
            .send()
            .await?;
        let block = driver.build_new_block().await?;

        assert!(block.includes(valid_tx.tx_hash()));
        assert!(block.includes(reverting_tx.tx_hash()));
    }

    Ok(())
}

/// If revert protection is disabled, it should not be possible to send a revert bundle
/// since the revert RPC endpoint is not available.
#[rb_test]
async fn disabled_bundle_endpoint_error(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;

    let res = driver
        .create_transaction()
        .with_bundle(BundleOpts::default())
        .send()
        .await;

    assert!(
        res.is_err(),
        "Expected error because method is not available"
    );
    Ok(())
}

/// Test the behaviour of the revert protection bundle, if the bundle **does not** revert
/// the transaction is included in the block. If the bundle reverts, the transaction
/// is not included in the block and tried again for the next bundle range blocks
/// when it will be dropped from the pool.
#[rb_test(args = OpRbuilderArgs {
    enable_revert_protection: true,
    ..Default::default()
})]
async fn bundle(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let _ = driver.build_new_block().await?; // Block 1

    // Test 1: Bundle does not revert
    let valid_bundle = driver
        .create_transaction()
        .random_valid_transfer()
        .with_bundle(BundleOpts::default())
        .send()
        .await?;

    let block2 = driver.build_new_block().await?; // Block 2
    assert!(
        block2
            .transactions
            .hashes()
            .includes(valid_bundle.tx_hash())
    );

    let bundle_opts = BundleOpts::default().with_block_number_max(4);

    let reverted_bundle = driver
        .create_transaction()
        .random_reverting_transaction()
        .with_bundle(bundle_opts)
        .send()
        .await?;

    // Test 2: Bundle reverts. It is not included in the block
    let block3 = driver.build_new_block().await?; // Block 3
    assert!(!block3.includes(reverted_bundle.tx_hash()));

    // After the block the transaction is still pending in the pool
    assert!(rbuilder.pool().is_pending(*reverted_bundle.tx_hash()));

    // Test 3: Chain progresses beyond the bundle range. The transaction is dropped from the pool
    driver.build_new_block().await?; // Block 4
    assert!(rbuilder.pool().is_dropped(*reverted_bundle.tx_hash()));

    driver.build_new_block().await?; // Block 5
    assert!(rbuilder.pool().is_dropped(*reverted_bundle.tx_hash()));

    Ok(())
}

/// Test the behaviour of the revert protection bundle with a min block number.
#[rb_test(args = OpRbuilderArgs {
    enable_revert_protection: true,
    ..Default::default()
})]
async fn bundle_min_block_number(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;

    // The bundle is valid when the min block number is equal to the current block
    let bundle_with_min_block = driver
        .create_transaction()
        .with_revert() // the transaction reverts but it is included in the block
        .with_reverted_hash()
        .with_bundle(BundleOpts::default().with_block_number_min(2))
        .send()
        .await?;

    let block = driver.build_new_block().await?; // Block 1, bundle still not valid
    assert!(!block.includes(bundle_with_min_block.tx_hash()));

    let block = driver.build_new_block().await?; // Block 2, bundle is valid
    assert!(block.includes(bundle_with_min_block.tx_hash()));

    // Send a bundle with a match of min and max block number
    let bundle_with_min_and_max_block = driver
        .create_transaction()
        .with_revert()
        .with_reverted_hash()
        .with_bundle(
            BundleOpts::default()
                .with_block_number_max(4)
                .with_block_number_min(4),
        )
        .send()
        .await?;

    let block = driver.build_new_block().await?; // Block 3, bundle still not valid
    assert!(!block.includes(bundle_with_min_and_max_block.tx_hash()));

    let block = driver.build_new_block().await?; // Block 4, bundle is valid
    assert!(block.includes(bundle_with_min_and_max_block.tx_hash()));

    Ok(())
}

/// Test the behaviour of the revert protection bundle with a min timestamp.
#[rb_test(args = OpRbuilderArgs {
    enable_revert_protection: true,
    ..Default::default()
})]
async fn bundle_min_timestamp(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let initial_timestamp = driver.latest().await?.header.timestamp;

    // The bundle is valid when the min timestamp is equal to the current block's timestamp
    let bundle_with_min_timestamp = driver
        .create_transaction()
        .with_revert() // the transaction reverts but it is included in the block
        .with_reverted_hash()
        .with_bundle(BundleOpts::default().with_min_timestamp(initial_timestamp + 2))
        .send()
        .await?;

    // Each block advances the timestamp by block_time_secs which is 1 when chain_block_time isn't set
    let block = driver.build_new_block().await?; // Block 1, initial_timestamp + 1
    assert!(!block.includes(bundle_with_min_timestamp.tx_hash()));

    let block = driver.build_new_block().await?; // Block 2, initial_timestamp + 2, so bundle is valid
    assert!(block.includes(bundle_with_min_timestamp.tx_hash()));

    Ok(())
}

/// Test the range limits for the revert protection bundle.
#[rb_test(args = OpRbuilderArgs {
    enable_revert_protection: true,
    ..Default::default()
})]
async fn bundle_range_limits(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let _ = driver.build_new_block().await?; // Block 1
    let _ = driver.build_new_block().await?; // Block 2

    async fn send_bundle(
        driver: &ChainDriver,
        bundle: BundleOpts,
    ) -> eyre::Result<PendingTransactionBuilder<Optimism>> {
        driver.create_transaction().with_bundle(bundle).send().await
    }

    // Max block cannot be a past block
    assert!(
        send_bundle(&driver, BundleOpts::default().with_block_number_max(1))
            .await
            .is_err()
    );

    // Bundles are valid if their max block in in between the current block and the max block range
    let current_block = 2;
    let next_valid_block = current_block + 1;

    for i in next_valid_block..next_valid_block + MAX_BLOCK_RANGE_BLOCKS {
        assert!(
            send_bundle(&driver, BundleOpts::default().with_block_number_max(i))
                .await
                .is_ok()
        );
    }

    // A bundle with a block out of range is invalid
    assert!(
        send_bundle(
            &driver,
            BundleOpts::default()
                .with_block_number_max(next_valid_block + MAX_BLOCK_RANGE_BLOCKS + 1)
        )
        .await
        .is_err()
    );

    // A bundle with a min block number higher than the max block is invalid
    assert!(
        send_bundle(
            &driver,
            BundleOpts::default()
                .with_block_number_max(1)
                .with_block_number_min(2)
        )
        .await
        .is_err()
    );

    // A bundle with a min block number lower or equal to the current block is valid
    assert!(
        send_bundle(
            &driver,
            BundleOpts::default()
                .with_block_number_max(next_valid_block)
                .with_block_number_min(current_block)
        )
        .await
        .is_ok()
    );
    assert!(
        send_bundle(
            &driver,
            BundleOpts::default().with_block_number_max(next_valid_block)
        )
        .await
        .is_ok()
    );

    // A bundle with a min block equal to max block is valid
    assert!(
        send_bundle(
            &driver,
            BundleOpts::default()
                .with_block_number_max(next_valid_block)
                .with_block_number_min(next_valid_block)
        )
        .await
        .is_ok()
    );

    // Test min-only cases (no max specified)
    // A bundle with only min block that's within the default range is valid
    let default_max = current_block + MAX_BLOCK_RANGE_BLOCKS;
    assert!(
        send_bundle(
            &driver,
            BundleOpts::default().with_block_number_min(current_block)
        )
        .await
        .is_ok()
    );
    assert!(
        send_bundle(
            &driver,
            BundleOpts::default().with_block_number_min(default_max - 1)
        )
        .await
        .is_ok()
    );
    assert!(
        send_bundle(
            &driver,
            BundleOpts::default().with_block_number_min(default_max)
        )
        .await
        .is_ok()
    );

    // A bundle with only min block that exceeds the default max range is invalid
    assert!(
        send_bundle(
            &driver,
            BundleOpts::default().with_block_number_min(default_max + 1)
        )
        .await
        .is_err()
    );

    Ok(())
}

/// If a transaction reverts and was sent as a normal transaction through the eth_sendRawTransaction
/// bundle, the transaction should be included in the block.
/// This behaviour is the same as the 'disabled' test.
#[rb_test(args = OpRbuilderArgs {
    enable_revert_protection: true,
    ..Default::default()
})]
async fn allow_reverted_transactions_without_bundle(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;

    for _ in 0..10 {
        let valid_tx = driver
            .create_transaction()
            .random_valid_transfer()
            .send()
            .await?;
        let reverting_tx = driver
            .create_transaction()
            .random_reverting_transaction()
            .send()
            .await?;
        let block = driver.build_new_block().await?;

        assert!(block.includes(valid_tx.tx_hash()));
        assert!(block.includes(reverting_tx.tx_hash()));
    }

    Ok(())
}

/// If a transaction reverts and gets dropped it, the eth_getTransactionReceipt should return
/// an error message that it was dropped.
#[rb_test(args = OpRbuilderArgs {
    enable_revert_protection: true,
    ..OpRbuilderArgs::test_default()
})]
async fn check_transaction_receipt_status_message(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let provider = rbuilder.provider().await?;

    let reverting_tx = driver
        .create_transaction()
        .random_reverting_transaction()
        .with_bundle(BundleOpts::default().with_block_number_max(3))
        .send()
        .await?;
    let tx_hash = reverting_tx.tx_hash();

    let _ = driver.build_new_block().await?;
    let receipt = provider.get_transaction_receipt(*tx_hash).await?;
    assert!(receipt.is_none());

    let _ = driver.build_new_block().await?;
    let receipt = provider.get_transaction_receipt(*tx_hash).await?;
    assert!(receipt.is_none());

    // Dropped
    let _ = driver.build_new_block().await?;
    let receipt = provider.get_transaction_receipt(*tx_hash).await;

    assert!(receipt.is_err());

    Ok(())
}
