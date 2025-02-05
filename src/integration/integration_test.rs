#[cfg(test)]
mod tests {
    use crate::integration::{RollupBoostTestHarness, SimpleBlockGenerator};

    #[tokio::test]
    async fn test_integration() -> eyre::Result<()> {
        let harness = RollupBoostTestHarness::new().await?;

        let mut block_generator = SimpleBlockGenerator::new(&harness.engine_api);
        block_generator.init().await?;

        for _ in 0..10 {
            let block = block_generator.generate_block().await?;
            println!("Generated block: {:?}", block);
        }
        Ok(())
    }
}
