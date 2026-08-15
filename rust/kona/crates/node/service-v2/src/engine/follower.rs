//! Adapter from semantic follower operations to the existing Engine API task primitives.

use crate::engine::{EngineDriver, EngineError, EngineResult, SafeChainUpdate};
use alloy_rpc_types_engine::PayloadStatusEnum;
use async_trait::async_trait;
use kona_engine::{
    ConsolidateInput, ConsolidateTask, EngineClient as RawEngineClient, EngineState,
    EngineSyncState, EngineTaskExt, FinalizeBlockId, FinalizeTask, InsertTask, InsertTaskError,
};
use kona_genesis::RollupConfig;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;

/// A semantic engine driver for nodes that acquire unsafe blocks from the network.
///
/// This is the first compatibility adapter between the V2 service and the existing Engine API
/// task primitives. It executes operations directly instead of using the V1 actor and priority
/// task queue. Local payload construction is intentionally unavailable until the V2 sequencing
/// workflow can preserve the build/seal/publication boundaries.
#[derive(Debug)]
pub struct FollowerEngineDriver<Client> {
    client: Arc<Client>,
    config: Arc<RollupConfig>,
    state: EngineState,
}

impl<Client> FollowerEngineDriver<Client> {
    /// Creates a follower engine driver from an existing engine state.
    pub const fn new(client: Arc<Client>, config: Arc<RollupConfig>, state: EngineState) -> Self {
        Self { client, config, state }
    }
}

#[async_trait]
impl<Client> EngineDriver for FollowerEngineDriver<Client>
where
    Client: RawEngineClient + core::fmt::Debug + 'static,
{
    async fn build_unsafe(
        &mut self,
        _attributes: OpAttributesWithParent,
    ) -> EngineResult<OpExecutionPayloadEnvelope> {
        Err(EngineError::SequencingDisabled)
    }

    async fn import_unsafe(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
    ) -> EngineResult<L2BlockInfo> {
        match InsertTask::new(self.client.clone(), self.config.clone(), payload, false)
            .execute(&mut self.state)
            .await
        {
            Err(InsertTaskError::UnexpectedPayloadStatus(
                status @ PayloadStatusEnum::Invalid { .. },
            )) => Err(EngineError::InvalidUnsafePayload(status.to_string())),
            Err(error) => Err(EngineError::driver(error)),
            Ok(block) => Ok(block),
        }
    }

    async fn update_safe(&mut self, update: SafeChainUpdate) -> EngineResult<L2BlockInfo> {
        let input = match update {
            SafeChainUpdate::Attributes(attributes) => ConsolidateInput::Attributes(attributes),
            SafeChainUpdate::Block(block) => ConsolidateInput::BlockInfo(block),
        };

        ConsolidateTask::new(self.client.clone(), self.config.clone(), input)
            .execute(&mut self.state)
            .await
            .map_err(EngineError::driver)?;

        Ok(self.state.sync_state.safe_head())
    }

    async fn update_finalized(&mut self, block: L2BlockInfo) -> EngineResult<()> {
        FinalizeTask::new(
            self.client.clone(),
            self.config.clone(),
            FinalizeBlockId::ByHash(block.block_info.id()),
        )
        .execute(&mut self.state)
        .await
        .map_err(EngineError::driver)
    }

    fn state(&self) -> EngineSyncState {
        self.state.sync_state
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_engine::test_utils::MockEngineClient;

    #[tokio::test]
    async fn follower_rejects_local_payload_construction() {
        let config = Arc::new(RollupConfig::default());
        let client = Arc::new(MockEngineClient::new(config.clone()));
        let mut driver = FollowerEngineDriver::new(client, config, EngineState::default());

        let error = driver
            .build_unsafe(OpAttributesWithParent::new(
                Default::default(),
                L2BlockInfo::default(),
                None,
                false,
            ))
            .await
            .unwrap_err();

        assert_eq!(error, EngineError::SequencingDisabled);
        assert_eq!(driver.state(), EngineSyncState::default());
    }
}
