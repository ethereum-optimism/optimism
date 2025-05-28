use super::common::RollupBoostTestHarnessBuilder;

#[tokio::test]
async fn no_tx_pool() -> eyre::Result<()> {
    let harness = RollupBoostTestHarnessBuilder::new("no_tx_pool")
        .build()
        .await?;
    let mut block_generator = harness.block_generator().await?;

    // start creating 5 empty blocks which are processed by the L2 builder
    for _ in 0..5 {
        let (_block, block_creator) = block_generator.generate_block(true).await?;
        assert!(block_creator.is_l2(), "Block creator should be l2");
    }

    // process 5 more non empty blocks which are processed by the builder.
    // The builder should be on sync because it has received the new payload requests from rollup-boost.
    for _ in 0..5 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_builder(),
            "Block creator should be the builder"
        );
    }

    Ok(())
}
