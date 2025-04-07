mod common;

use std::time::Duration;

use testcontainers::core::client::docker_client_instance;

use crate::common::RollupBoostTestHarnessBuilder;

#[tokio::test]
async fn remote_builder_down() -> eyre::Result<()> {
    let harness = RollupBoostTestHarnessBuilder::new("remote_builder_down")
        .build()
        .await?;
    let mut block_generator = harness.block_generator().await?;

    for _ in 0..3 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_builder(),
            "Block creator should be the builder"
        );
    }

    // stop the builder
    let client = docker_client_instance().await?;
    client.pause_container(harness.builder.id()).await?;
    tokio::time::sleep(Duration::from_secs(2)).await;

    // create 3 new blocks that are processed by the l2 builder
    for _ in 0..3 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(block_creator.is_l2(), "Block creator should be l2");
    }

    client.unpause_container(harness.builder.id()).await?;

    // Sleep briefly to allow the builder to sync
    tokio::time::sleep(Duration::from_secs(2)).await;

    // create 3 new blocks that are processed by the l2 builder because the builder is not synced with the previous 3 blocks
    for _ in 0..3 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_builder(),
            "Block creator should be builder"
        );
    }

    Ok(())
}
