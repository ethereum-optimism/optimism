//! Admin RPC Module

use crate::AdminApiServer;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadError;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_node_actors::{engine, sequencer::Handle};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadError};

/// The admin RPC server.
#[derive(Debug)]
pub struct AdminRpc {
    sequencer: Option<Handle>,
    conductor_enabled: bool,
    engine: engine::Handle,
}

impl AdminRpc {
    /// Constructs the admin RPC server.
    pub const fn new(
        sequencer: Option<Handle>,
        conductor_enabled: bool,
        engine: engine::Handle,
    ) -> Self {
        Self { sequencer, conductor_enabled, engine }
    }

    fn sequencer(&self) -> RpcResult<&Handle> {
        self.sequencer.as_ref().ok_or_else(|| ErrorObject::from(ErrorCode::MethodNotFound))
    }
}

#[async_trait]
impl AdminApiServer for AdminRpc {
    async fn admin_post_unsafe_payload(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> RpcResult<()> {
        payload.check_block_hash().map_err(|err| {
            let message = match &err {
                OpPayloadError::Eth(PayloadError::BlockHash { execution, consensus }) => {
                    format!(
                        "payload has bad block hash: {consensus}, actual block hash is: {execution}"
                    )
                }
                _ => format!("invalid payload: {err}"),
            };
            ErrorObject::owned(ErrorCode::InvalidParams.code(), message, None::<()>)
        })?;
        self.engine
            .insert_unsafe_block(payload)
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_sequencer_active(&self) -> RpcResult<bool> {
        self.sequencer()?.is_active().map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_start_sequencer(&self) -> RpcResult<()> {
        self.sequencer()?.start().await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_stop_sequencer(&self) -> RpcResult<B256> {
        self.sequencer()?.stop().await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_conductor_enabled(&self) -> RpcResult<bool> {
        self.sequencer()?;
        Ok(self.conductor_enabled)
    }

    async fn admin_override_leader(&self) -> RpcResult<()> {
        self.sequencer()?
            .override_leader()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_reset_derivation_pipeline(&self) -> RpcResult<()> {
        self.engine
            .reset()
            .await
            .map(|_| ())
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rpc_types_engine::{
        ExecutionPayloadV1, ForkchoiceUpdated, PayloadId, PayloadStatus, PayloadStatusEnum,
    };
    use kona_derive::test_utils::TestAttributesBuilder;
    use kona_engine::{
        NoopBlockSink,
        test_utils::{RpcMock, test_engine_client},
    };
    use kona_genesis::RollupConfig;
    use kona_node_actors::{
        EngineClientResult, L1OriginSelectorError, OriginSelector, SequencerEngineClient,
        sequencer::{self, ConductorClient},
    };
    use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
    use std::{
        future::{Future, ready},
        sync::Arc,
        time::Duration,
    };

    /// Completes the sequencer's startup reset and reports a fixed unsafe head.
    #[derive(Debug)]
    struct IdleEngine;

    #[async_trait]
    impl SequencerEngineClient for IdleEngine {
        async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
            Ok(())
        }

        async fn start_build_block(
            &self,
            _attributes: OpAttributesWithParent,
        ) -> EngineClientResult<PayloadId> {
            panic!("a full signer queue must pause building")
        }

        async fn seal_and_canonicalize_block(
            &self,
            _payload_id: PayloadId,
            _attributes: OpAttributesWithParent,
        ) -> EngineClientResult<OpExecutionPayloadEnvelope> {
            panic!("a full signer queue must pause building")
        }

        async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo> {
            Ok(L2BlockInfo {
                block_info: BlockInfo { hash: B256::repeat_byte(42), ..Default::default() },
                ..Default::default()
            })
        }
    }

    /// Returns a handle to an engine that never runs.
    fn idle_engine() -> engine::Handle {
        engine::Builder::new(engine::Capacity::try_from(1).unwrap()).handle()
    }

    /// Returns an engine whose execution and L1 clients complete one reset.
    fn resetting_engine()
    -> (engine::Handle, impl Future<Output = Result<(), engine::ActorError>> + Send + 'static) {
        let block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse =
            Default::default();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.inner.hash_slow();
        let config = Arc::new(config);
        let (client, l2) = test_engine_client(config.clone());
        let l1 = RpcMock::default();
        for _ in 0..3 {
            l2.expect("eth_getBlockByNumber", &block);
        }
        l1.expect(
            "eth_getBlockByHash",
            alloy_rpc_types_eth::Block::<alloy_rpc_types_eth::Transaction>::default(),
        );
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
        let builder = engine::Builder::new(engine::Capacity::try_from(1).unwrap());
        let handle = builder.handle();
        let task = builder.build(Arc::new(client), l1.provider(), config, Arc::new(NoopBlockSink));
        (handle, task)
    }

    #[derive(Debug)]
    struct PausedSigner;

    #[async_trait]
    impl sequencer::Signer for PausedSigner {
        type Error = std::convert::Infallible;

        async fn send(&self, _payload: OpExecutionPayloadEnvelope) -> Result<(), Self::Error> {
            panic!("a full signer queue must pause building")
        }

        fn has_capacity(&self) -> bool {
            false
        }
    }

    #[derive(Debug)]
    struct UnusedOriginSelector;

    #[async_trait]
    impl OriginSelector for UnusedOriginSelector {
        async fn next_l1_origin(
            &mut self,
            _unsafe_head: L2BlockInfo,
        ) -> Result<BlockInfo, L1OriginSelectorError> {
            panic!("a full signer queue must pause building")
        }
    }

    fn sequencer(
        conductor_enabled: bool,
    ) -> (Handle, impl Future<Output = Result<(), sequencer::ActorError>> + Send + 'static) {
        let builder = sequencer::Builder::new(sequencer::Capacity::try_from(1).unwrap());
        let handle = builder.handle();
        let task = builder.build(
            TestAttributesBuilder { attributes: vec![] },
            IdleEngine,
            UnusedOriginSelector,
            Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
            conductor_enabled
                .then(|| ConductorClient::new_http("http://localhost:1".parse().unwrap())),
            PausedSigner,
        );
        (handle, task)
    }

    #[tokio::test]
    async fn reads_published_state_without_queueing_and_rejects_closed_publisher() {
        let (handle, task) = sequencer(true);
        // Queue a command without running the actor, leaving the mailbox full.
        tokio::select! {
            biased;
            result = handle.start() => panic!("command completed before startup: {result:?}"),
            _ = ready(()) => {}
        }
        let rpc = AdminRpc::new(Some(handle), true, idle_engine());
        assert!(!rpc.admin_sequencer_active().await.unwrap());
        assert!(rpc.admin_conductor_enabled().await.unwrap());

        drop(task);
        assert_eq!(
            rpc.admin_sequencer_active().await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
        assert!(rpc.admin_conductor_enabled().await.unwrap());
    }

    #[tokio::test]
    async fn sequencer_methods_are_unavailable_on_validators() {
        let rpc = AdminRpc::new(None, false, idle_engine());
        for result in [
            rpc.admin_sequencer_active().await.map(|_| ()),
            rpc.admin_conductor_enabled().await.map(|_| ()),
            rpc.admin_start_sequencer().await,
            rpc.admin_stop_sequencer().await.map(|_| ()),
            rpc.admin_override_leader().await,
        ] {
            assert_eq!(result.unwrap_err().code(), ErrorCode::MethodNotFound.code());
        }
    }

    #[tokio::test]
    async fn commands_return_actor_results_and_map_failures() {
        let (handle, task) = sequencer(false);
        let task = tokio::spawn(task);
        let rpc = AdminRpc::new(Some(handle), false, idle_engine());
        assert_eq!(rpc.admin_stop_sequencer().await.unwrap(), B256::repeat_byte(42));
        assert_eq!(
            rpc.admin_override_leader().await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
        rpc.admin_start_sequencer().await.unwrap();
        assert!(rpc.admin_sequencer_active().await.unwrap());

        task.abort();
        assert!(task.await.unwrap_err().is_cancelled());
        assert_eq!(
            rpc.admin_start_sequencer().await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
    }

    #[tokio::test]
    async fn reads_initial_state_before_build() {
        let builder = sequencer::Builder::new(sequencer::Capacity::try_from(1).unwrap());
        let rpc = AdminRpc::new(Some(builder.handle()), false, idle_engine());
        assert!(!rpc.admin_sequencer_active().await.unwrap());
        assert!(!rpc.admin_conductor_enabled().await.unwrap());
    }

    #[tokio::test]
    async fn resets_engine_on_sequencers_and_validators_and_waits_for_acknowledgement() {
        for is_sequencer in [true, false] {
            let builder = sequencer::Builder::new(sequencer::Capacity::try_from(1).unwrap());
            let handle = builder.handle();
            // Reset must bypass even a full sequencer command queue.
            tokio::select! {
                biased;
                result = handle.start() => panic!("command completed before startup: {result:?}"),
                _ = ready(()) => {}
            }
            let sequencer = is_sequencer.then_some(handle);
            let (engine, engine_task) = resetting_engine();
            let rpc = AdminRpc::new(sequencer, false, engine).into_rpc();
            let reset =
                rpc.call::<_, ()>("admin_resetDerivationPipeline", jsonrpsee::rpc_params![]);
            tokio::pin!(reset);
            // The reset is queued, but the engine has not performed it.
            assert!(tokio::time::timeout(Duration::from_millis(10), &mut reset).await.is_err());
            let engine_task = tokio::spawn(engine_task);
            reset.await.unwrap();
            engine_task.abort();
        }
    }

    #[tokio::test]
    async fn maps_closed_engine_channels() {
        let builder = engine::Builder::new(engine::Capacity::try_from(1).unwrap());
        let rpc = AdminRpc::new(None, false, builder.handle());
        // The engine stops with the reset queued.
        let (result, ()) = tokio::join!(rpc.admin_reset_derivation_pipeline(), async {
            drop(builder);
        });
        assert_eq!(result.unwrap_err().code(), ErrorCode::InternalError.code());
        assert_eq!(
            rpc.admin_reset_derivation_pipeline().await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
    }

    #[tokio::test]
    async fn validates_payloads_before_queueing_them_on_the_engine() {
        // The engine never runs, so its one-slot queue fills with the first queued payload.
        let builder = engine::Builder::new(engine::Capacity::try_from(1).unwrap());
        let rpc = AdminRpc::new(None, false, builder.handle());
        let mut payload = ExecutionPayloadV1 {
            parent_hash: Default::default(),
            fee_recipient: Default::default(),
            state_root: Default::default(),
            receipts_root: Default::default(),
            logs_bloom: Default::default(),
            prev_randao: Default::default(),
            block_number: 1,
            gas_limit: 30_000_000,
            gas_used: 0,
            timestamp: 1,
            extra_data: Default::default(),
            base_fee_per_gas: Default::default(),
            block_hash: Default::default(),
            transactions: vec![],
        };
        let error = rpc
            .admin_post_unsafe_payload(OpExecutionPayloadEnvelope::V1(payload.clone()))
            .await
            .unwrap_err();
        assert_eq!(error.code(), ErrorCode::InvalidParams.code());
        assert!(error.message().starts_with("payload has bad block hash:"));

        // The rejected payload left the queue empty, so a valid one is queued at once.
        payload.block_hash = payload.clone().into_block_raw().unwrap().header.hash_slow();
        let envelope = OpExecutionPayloadEnvelope::V1(payload);
        rpc.admin_post_unsafe_payload(envelope.clone()).await.unwrap();

        // The next payload waits for room behind the first, and fails once the engine stops.
        let post = rpc.admin_post_unsafe_payload(envelope);
        tokio::pin!(post);
        tokio::select! {
            biased;
            result = &mut post => panic!("a full engine queue accepted a payload: {result:?}"),
            _ = ready(()) => {}
        }
        drop(builder);
        assert_eq!(post.await.unwrap_err().code(), ErrorCode::InternalError.code());
    }
}
