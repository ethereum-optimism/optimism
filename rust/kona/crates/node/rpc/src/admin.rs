//! Admin RPC Module

use crate::{AdminApiServer, SequencerAdminApiServer};
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
    engine: engine::Handle,
}

impl AdminRpc {
    /// Constructs the admin RPC server.
    pub const fn new(engine: engine::Handle) -> Self {
        Self { engine }
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

    async fn admin_reset_derivation_pipeline(&self) -> RpcResult<()> {
        self.engine
            .reset()
            .await
            .map(|_| ())
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}

/// The sequencer's admin RPC server.
#[derive(Debug)]
pub struct SequencerAdminRpc {
    sequencer: Handle,
    conductor_enabled: bool,
}

impl SequencerAdminRpc {
    /// Constructs the sequencer's admin RPC server.
    pub const fn new(sequencer: Handle, conductor_enabled: bool) -> Self {
        Self { sequencer, conductor_enabled }
    }
}

#[async_trait]
impl SequencerAdminApiServer for SequencerAdminRpc {
    async fn admin_sequencer_active(&self) -> RpcResult<bool> {
        self.sequencer.is_active().map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_start_sequencer(&self) -> RpcResult<()> {
        self.sequencer.start().await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_stop_sequencer(&self) -> RpcResult<B256> {
        self.sequencer.stop().await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_conductor_enabled(&self) -> RpcResult<bool> {
        Ok(self.conductor_enabled)
    }

    async fn admin_override_leader(&self) -> RpcResult<()> {
        self.sequencer
            .override_leader()
            .await
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
    use kona_genesis::RollupConfig;
    use kona_node_actors::{
        EngineClientResult, SequencerEngineClient,
        engine::NoopBlockSink,
        origin::OriginError,
        sequencer::{self, ConductorClient},
    };
    use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
    use kona_sources::test_utils::{RpcMock, test_engine_api};
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
        engine::Builder::new(engine::EngineState::default()).handle()
    }

    /// Returns an engine whose execution client completes one reset, on an L1 chain that holds
    /// only the L1 origin of the genesis block.
    fn resetting_engine()
    -> (engine::Handle, impl Future<Output = Result<(), engine::ActorError>> + Send + 'static) {
        let block: <op_alloy_network::Optimism as alloy_provider::Network>::BlockResponse =
            Default::default();
        let mut config = RollupConfig::default();
        config.genesis.l2.hash = block.header.inner.hash_slow();
        let config = Arc::new(config);
        let (client, l2) = test_engine_api(config.clone());
        for _ in 0..3 {
            l2.expect("eth_getBlockByNumber", &block);
        }
        l2.expect(
            "engine_forkchoiceUpdatedV3",
            ForkchoiceUpdated::new(PayloadStatus::from_status(PayloadStatusEnum::Valid)),
        );
        // The genesis block's L1 origin is the default L1 block, which the L1 chain starts at.
        let l1 = kona_node_actors::l1::Builder::new(
            kona_sources::UntrustedRpc::new(RpcMock::default().provider())
                .retrying(Duration::from_secs(1)),
            BlockInfo::default(),
        );
        let builder = engine::Builder::new(engine::EngineState::default());
        let handle = builder.handle();
        let engine = builder.build(Arc::new(client), l1.handle(), config, Arc::new(NoopBlockSink));
        // The L1 chain lasts as long as the engine.
        (handle, async move {
            let _l1 = l1;
            engine.await
        })
    }

    #[derive(Debug)]
    struct PausedSigner;

    #[async_trait]
    impl sequencer::Signer for PausedSigner {
        type Error = std::convert::Infallible;

        async fn send(&self, _payload: OpExecutionPayloadEnvelope) -> Result<(), Self::Error> {
            panic!("a full signer queue must pause building")
        }

        async fn ready(&self) -> Result<(), Self::Error> {
            std::future::pending().await
        }
    }

    #[derive(Debug)]
    struct UnusedOrigin;

    impl sequencer::Origin for UnusedOrigin {
        fn next(&self, _parent: L2BlockInfo) -> Result<BlockInfo, OriginError> {
            panic!("a full signer queue must pause building")
        }
    }

    fn sequencer(
        conductor_enabled: bool,
    ) -> (Handle, impl Future<Output = Result<(), sequencer::ActorError>> + Send + 'static) {
        let builder = sequencer::Builder::new();
        let handle = builder.handle();
        let task = builder.build(
            TestAttributesBuilder { attributes: vec![] },
            IdleEngine,
            UnusedOrigin,
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
        let rpc = SequencerAdminRpc::new(handle, true);
        assert!(!rpc.admin_sequencer_active().await.unwrap());
        assert!(rpc.admin_conductor_enabled().await.unwrap());

        drop(task);
        assert_eq!(
            rpc.admin_sequencer_active().await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
        assert!(rpc.admin_conductor_enabled().await.unwrap());
    }

    #[test]
    fn sequencer_methods_are_a_separate_module_in_the_admin_namespace() {
        const SEQUENCER_METHODS: [&str; 5] = [
            "admin_sequencerActive",
            "admin_startSequencer",
            "admin_stopSequencer",
            "admin_conductorEnabled",
            "admin_overrideLeader",
        ];
        let mut admin = AdminRpc::new(idle_engine()).into_rpc();
        for method in SEQUENCER_METHODS {
            assert!(admin.method(method).is_none(), "{method} is served without a sequencer");
        }

        let builder = sequencer::Builder::new();
        admin.merge(SequencerAdminRpc::new(builder.handle(), false).into_rpc()).unwrap();
        for method in SEQUENCER_METHODS {
            assert!(admin.method(method).is_some(), "{method} is not served with a sequencer");
        }
    }

    #[tokio::test]
    async fn commands_return_actor_results_and_map_failures() {
        let (handle, task) = sequencer(false);
        let task = tokio::spawn(task);
        let rpc = SequencerAdminRpc::new(handle, false);
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
        let builder = sequencer::Builder::new();
        let rpc = SequencerAdminRpc::new(builder.handle(), false);
        assert!(!rpc.admin_sequencer_active().await.unwrap());
        assert!(!rpc.admin_conductor_enabled().await.unwrap());
    }

    #[tokio::test]
    async fn resets_engine_and_waits_for_acknowledgement() {
        let (engine, engine_task) = resetting_engine();
        let rpc = AdminRpc::new(engine).into_rpc();
        let reset = rpc.call::<_, ()>("admin_resetDerivationPipeline", jsonrpsee::rpc_params![]);
        tokio::pin!(reset);
        // The reset is queued, but the engine has not performed it.
        assert!(tokio::time::timeout(Duration::from_millis(10), &mut reset).await.is_err());
        let engine_task = tokio::spawn(engine_task);
        reset.await.unwrap();
        engine_task.abort();
    }

    #[tokio::test]
    async fn maps_closed_engine_channels() {
        let builder = engine::Builder::new(engine::EngineState::default());
        let rpc = AdminRpc::new(builder.handle());
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
        let builder = engine::Builder::new(engine::EngineState::default());
        let rpc = AdminRpc::new(builder.handle());
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
