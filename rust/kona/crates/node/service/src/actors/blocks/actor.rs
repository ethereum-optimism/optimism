use super::{BlocksClient, BlocksClientError};
use crate::{EngineClientError, NetworkEngineClient, NodeActor};
use async_trait::async_trait;
use thiserror::Error;

/// Actor that forwards canonical unsafe blocks from a blocks stream to Kona's engine actor.
#[derive(Debug)]
pub struct BlocksClientActor<EngineClient> {
    blocks_client: BlocksClient,
    engine_client: EngineClient,
}

impl<EngineClient> BlocksClientActor<EngineClient> {
    /// Creates a blocks client actor from an established stream and engine client.
    pub const fn new(blocks_client: BlocksClient, engine_client: EngineClient) -> Self {
        Self { blocks_client, engine_client }
    }
}

#[async_trait]
impl<EngineClient> NodeActor for BlocksClientActor<EngineClient>
where
    EngineClient: NetworkEngineClient + 'static,
{
    type Error = BlocksClientActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let block = self.blocks_client.next_block().await?;
        trace!(
            target: "blocks_client",
            number = block.execution_payload.block_number(),
            hash = %block.execution_payload.block_hash(),
            "Forwarding unsafe block from sequencer blocks stream"
        );
        self.engine_client.send_unsafe_block(block).await?;
        Ok(())
    }
}

/// An error forwarding blocks stream payloads to Kona's engine actor.
#[derive(Debug, Error)]
pub enum BlocksClientActorError {
    /// The blocks stream client failed.
    #[error(transparent)]
    Client(#[from] BlocksClientError),
    /// The payload could not be forwarded to the engine actor.
    #[error(transparent)]
    Engine(#[from] EngineClientError),
}
