#[cfg(test)]
mod tests {
    use crate::integration::RollupBoostTestHarness;

    #[tokio::test]
    async fn test_integration_simple() -> eyre::Result<()> {
        let harness = RollupBoostTestHarness::new("test_integration_simple").await?;
        let mut block_generator = harness.get_block_generator().await;

        for _ in 0..5 {
            let (_block, block_creator) = block_generator.generate_block(false).await?;
            assert!(
                block_creator.is_builder(),
                "Block creator should be the builder"
            );

            // wait for 1 second
            tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
        }

        Ok(())
    }

    #[tokio::test]
    async fn test_integration_no_tx_pool() -> eyre::Result<()> {
        let harness = RollupBoostTestHarness::new("test_integration_no_tx_pool").await?;

        let mut block_generator = harness.get_block_generator().await;

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
}
