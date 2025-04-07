use common::RollupBoostTestHarnessBuilder;

mod common;

#[tokio::test]
async fn test_integration_simple() -> eyre::Result<()> {
    let harness = RollupBoostTestHarnessBuilder::new("simple").build().await?;
    let mut block_generator = harness.block_generator().await?;

    for _ in 0..5 {
        let (_block, block_creator) = block_generator.generate_block(false).await?;
        assert!(
            block_creator.is_builder(),
            "Block creator should be the builder"
        );
    }

    Ok(())
}
