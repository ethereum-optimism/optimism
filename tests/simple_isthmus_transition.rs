use common::RollupBoostTestHarnessBuilder;

mod common;

#[tokio::test]
async fn test_integration_simple_isthmus_transition() -> eyre::Result<()> {
    let harness = RollupBoostTestHarnessBuilder::new("simple_isthmus_transition")
        .with_isthmus_block(5)
        .build()
        .await?;
    let mut block_generator = harness.block_generator().await?;
    block_generator.generate_builder_blocks(10).await?;

    Ok(())
}
