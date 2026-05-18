use crate::{
    args::OpRbuilderArgs,
    gas_limiter::args::GasLimiterArgs,
    tests::{ChainDriverExt, LocalInstance, TransactionBuilderExt},
};
use macros::rb_test;
use std::collections::HashSet;
use tracing::info;

/// Integration test for the gas limiter functionality.
/// Tests that gas limits are properly enforced during actual block building
/// and transaction execution.
#[rb_test(args = OpRbuilderArgs {
    gas_limiter: GasLimiterArgs {
        gas_limiter_enabled: true,
        max_gas_per_address: 200000,  // 200k gas per address - low for testing
        refill_rate_per_block: 100000,  // 100k gas refill per block
        cleanup_interval: 100,
    },
    ..Default::default()
})]
async fn gas_limiter_blocks_excessive_usage(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;

    // Fund some accounts for testing
    let funded_accounts = driver
        .fund_accounts(2, 10_000_000_000_000_000_000u128)
        .await?; // 10 ETH each

    // These transactions should not be throttled
    let tx1 = driver
        .create_transaction()
        .with_signer(funded_accounts[0])
        .random_valid_transfer()
        .send()
        .await?;

    let tx2 = driver
        .create_transaction()
        .with_signer(funded_accounts[1])
        .random_valid_transfer()
        .send()
        .await?;

    // Build block and verify inclusion
    let block = driver.build_new_block_with_current_timestamp(None).await?;
    let tx_hashes: HashSet<_> = block.transactions.hashes().collect();

    assert!(tx_hashes.contains(tx1.tx_hash()), "tx1 should be included");
    assert!(tx_hashes.contains(tx2.tx_hash()), "tx2 should be included");

    // Send multiple big transactions from the same address - these should hit the gas limiter
    let mut sent_txs = Vec::new();
    for i in 0..5 {
        let big_tx = driver
            .create_transaction()
            .with_signer(funded_accounts[0])
            .random_big_transaction()
            .send()
            .await?;
        sent_txs.push(*big_tx.tx_hash());
        info!(
            "Sent big transaction {} from address {}",
            i + 1,
            funded_accounts[0].address
        );
    }

    // Meanwhile, the other address should not be throttled
    let legit_tx = driver
        .create_transaction()
        .with_signer(funded_accounts[1])
        .random_big_transaction()
        .send()
        .await?;

    let block = driver.build_new_block_with_current_timestamp(None).await?;
    let tx_hashes: HashSet<_> = block.transactions.hashes().collect();

    let included_count = sent_txs.iter().filter(|tx| tx_hashes.contains(*tx)).count();

    // With gas limiting, we shouldn't get all 5 big transactions from the same
    // address. We do this imprecise count because we haven't built a way of
    // sending a tx that uses an exact amount of gas.
    assert!(
        included_count < 5,
        "Gas limiter should have rejected some transactions, included: {}/5",
        included_count
    );
    assert!(
        included_count > 0,
        "Gas limiter should have allowed at least one transaction"
    );

    assert!(
        tx_hashes.contains(legit_tx.tx_hash()),
        "Transaction from different address should be included"
    );

    // After building new blocks, the limited address should get more capacity
    for _ in 0..3 {
        let _block = driver.build_new_block_with_current_timestamp(None).await?;
    }

    let tx_after_refill = driver
        .create_transaction()
        .with_signer(funded_accounts[0])
        .random_valid_transfer()
        .send()
        .await?;

    let refill_block = driver.build_new_block_with_current_timestamp(None).await?;
    let refill_tx_hashes: HashSet<_> = refill_block.transactions.hashes().collect();

    assert!(
        refill_tx_hashes.contains(tx_after_refill.tx_hash()),
        "Transaction should succeed after refill"
    );

    Ok(())
}
