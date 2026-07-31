use crate::tests::{ChainDriverExt, LocalInstance, framework::ONE_ETH};
use alloy_consensus::Transaction;
use futures::{StreamExt, future::join_all, stream};
use macros::rb_test;

/// This test ensures that the transactions are ordered by fee priority in the block.
/// This version of the test is only applicable to the standard builder because in flashblocks
/// the transaction order is commited by the block after each flashblock is produced,
/// so the order is only going to hold within one flashblock, but not the entire block.
#[rb_test(standard)]
async fn fee_priority_ordering(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let accounts = driver.fund_accounts(10, ONE_ETH).await?;

    let latest_block = driver.latest().await?;
    let base_fee = latest_block
        .header
        .base_fee_per_gas
        .expect("Base fee should be present in the latest block");

    // generate transactions with randomized tips
    let txs = join_all(accounts.iter().map(|signer| {
        driver
            .create_transaction()
            .with_signer(*signer)
            .with_max_priority_fee_per_gas(rand::random_range(1..50))
            .send()
    }))
    .await
    .into_iter()
    .collect::<eyre::Result<Vec<_>>>()?
    .into_iter()
    .map(|tx| *tx.tx_hash())
    .collect::<Vec<_>>();

    driver.build_new_block().await?;

    // verify all transactions are included in the block
    assert!(
        stream::iter(txs.iter())
            .all(|tx_hash| async {
                driver
                    .latest_full()
                    .await
                    .expect("Failed to fetch latest block")
                    .transactions
                    .hashes()
                    .any(|hash| hash == *tx_hash)
            })
            .await,
        "not all transactions included in the block"
    );

    // verify all transactions are ordered by fee priority
    let txs_tips = driver
        .latest_full()
        .await?
        .into_transactions_vec()
        .into_iter()
        .skip(1) // skip the deposit transaction
        .take(txs.len()) // skip the last builder transaction
        .map(|tx| tx.effective_tip_per_gas(base_fee as u64))
        .rev() // we want to check descending order
        .collect::<Vec<_>>();

    assert!(
        txs_tips.is_sorted(),
        "Transactions not ordered by fee priority"
    );

    Ok(())
}
