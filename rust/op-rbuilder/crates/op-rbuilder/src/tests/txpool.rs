use crate::tests::{
    BlockTransactionsExt, ChainDriverExt, LocalInstance, ONE_ETH, default_node_config,
};
use macros::rb_test;
use reth::args::TxPoolArgs;
use reth_node_builder::NodeConfig;
use reth_optimism_chainspec::OpChainSpec;

/// This test ensures that pending pool custom limit is respected and priority tx would be included even when pool if full.
#[rb_test(
    config = NodeConfig::<OpChainSpec> {
        txpool: TxPoolArgs {
            pending_max_count: 50,
            ..Default::default()
        },
        ..default_node_config()
    },
    standard
)]
async fn pending_pool_limit(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let accounts = driver.fund_accounts(50, ONE_ETH).await?;

    // Send 50 txs from different addrs
    let acc_no_priority = accounts.first().unwrap();
    let acc_with_priority = accounts.last().unwrap();

    for _ in 0..50 {
        let _ = driver
            .create_transaction()
            .with_signer(*acc_no_priority)
            .send()
            .await?;
    }

    assert_eq!(
        rbuilder.pool().pending_count(),
        50,
        "Pending pool must contain at max 50 txs {:?}",
        rbuilder.pool().pending_count()
    );

    // Send 10 txs that should be included in the block
    let mut txs = Vec::new();
    for _ in 0..10 {
        let tx = driver
            .create_transaction()
            .with_signer(*acc_with_priority)
            .with_max_priority_fee_per_gas(10)
            .send()
            .await?;
        txs.push(*tx.tx_hash());
    }

    assert_eq!(
        rbuilder.pool().pending_count(),
        50,
        "Pending pool must contain at max 50 txs {:?}",
        rbuilder.pool().pending_count()
    );

    // After we try building block our reverting tx would be removed and other tx will move to queue pool
    let block = driver.build_new_block().await?;

    // Ensure that 10 extra txs got included
    assert!(block.includes(&txs));

    Ok(())
}
