//! Admin RPC Module

use crate::AdminApiServer;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadError;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::{EngineActorRequest, ResetRequest};
use kona_node_actors::sequencer::Handle;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadError};
use tokio::sync::mpsc;

/// The admin RPC server.
#[derive(Debug)]
pub struct AdminRpc {
    sequencer: Option<Handle>,
    conductor_enabled: bool,
    engine: mpsc::Sender<EngineActorRequest>,
    unsafe_payloads: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

impl AdminRpc {
    /// Constructs the admin RPC server.
    pub const fn new(
        sequencer: Option<Handle>,
        conductor_enabled: bool,
        engine: mpsc::Sender<EngineActorRequest>,
        unsafe_payloads: mpsc::Sender<OpExecutionPayloadEnvelope>,
    ) -> Self {
        Self { sequencer, conductor_enabled, engine, unsafe_payloads }
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
        self.unsafe_payloads
            .send(payload)
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
        let (result_tx, mut result_rx) = mpsc::channel(1);
        self.engine
            .send(EngineActorRequest::Reset(Box::new(ResetRequest { result_tx })))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
        result_rx
            .recv()
            .await
            .ok_or_else(|| ErrorObject::from(ErrorCode::InternalError))?
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use kona_derive::test_utils::TestAttributesBuilder;
    use kona_genesis::RollupConfig;
    use kona_node_actors::{
        L1OriginSelectorError, OriginSelector, QueuedSequencerEngineClient,
        UnsafePayloadGossipClient, UnsafePayloadGossipClientError,
        sequencer::{self, ConductorClient},
    };
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use std::{
        future::{Future, ready},
        sync::Arc,
    };
    use tokio::sync::watch;

    #[derive(Debug)]
    struct PausedGossip;

    #[async_trait]
    impl UnsafePayloadGossipClient for PausedGossip {
        async fn schedule_execution_payload_gossip(
            &self,
            _payload: OpExecutionPayloadEnvelope,
        ) -> Result<(), UnsafePayloadGossipClientError> {
            panic!("a full gossip queue must pause building")
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
            panic!("a full gossip queue must pause building")
        }
    }

    fn sequencer(
        conductor_enabled: bool,
    ) -> (
        Handle,
        impl Future<Output = Result<(), sequencer::ActorError>> + Send + 'static,
        mpsc::Receiver<EngineActorRequest>,
    ) {
        let builder = sequencer::Builder::new(sequencer::Capacity::try_from(1).unwrap());
        let handle = builder.handle();
        let (engine_actor_request_tx, requests) = mpsc::channel(1);
        let (_, unsafe_head_rx) = watch::channel(L2BlockInfo {
            block_info: BlockInfo { hash: B256::repeat_byte(42), ..Default::default() },
            ..Default::default()
        });
        let task = builder.build(
            TestAttributesBuilder { attributes: vec![] },
            QueuedSequencerEngineClient { engine_actor_request_tx, unsafe_head_rx },
            UnusedOriginSelector,
            Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
            conductor_enabled
                .then(|| ConductorClient::new_http("http://localhost:1".parse().unwrap())),
            PausedGossip,
        );
        (handle, task, requests)
    }

    async fn acknowledge_startup(requests: &mut mpsc::Receiver<EngineActorRequest>) {
        let EngineActorRequest::Reset(request) = requests.recv().await.unwrap() else {
            panic!("expected initial engine reset");
        };
        request.result_tx.send(Ok(())).await.unwrap();
    }

    #[tokio::test]
    async fn reads_published_state_without_queueing_and_rejects_closed_publisher() {
        let (handle, task, _requests) = sequencer(true);
        // Queue a command without running the actor, leaving the mailbox full.
        tokio::select! {
            biased;
            result = handle.start() => panic!("command completed before startup: {result:?}"),
            _ = ready(()) => {}
        }
        let rpc = AdminRpc::new(Some(handle), true, mpsc::channel(1).0, mpsc::channel(1).0);
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
        let (tx, _) = mpsc::channel(1);
        let rpc = AdminRpc::new(None, false, mpsc::channel(1).0, tx);
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
        let (handle, task, mut requests) = sequencer(false);
        let task = tokio::spawn(task);
        acknowledge_startup(&mut requests).await;
        let rpc = AdminRpc::new(Some(handle), false, mpsc::channel(1).0, mpsc::channel(1).0);
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
        let rpc =
            AdminRpc::new(Some(builder.handle()), false, mpsc::channel(1).0, mpsc::channel(1).0);
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
            let (engine_tx, mut engine_rx) = mpsc::channel(1);
            let rpc = AdminRpc::new(sequencer, false, engine_tx, mpsc::channel(1).0).into_rpc();
            let reset =
                rpc.call::<_, ()>("admin_resetDerivationPipeline", jsonrpsee::rpc_params![]);
            tokio::pin!(reset);
            let request = tokio::select! {
                result = &mut reset => panic!("reset returned before engine acknowledgement: {result:?}"),
                request = engine_rx.recv() => request.unwrap(),
            };
            let EngineActorRequest::Reset(request) = request else {
                panic!("expected engine reset");
            };
            // Receiving the request is insufficient: the engine must acknowledge it.
            assert!(
                tokio::time::timeout(std::time::Duration::from_millis(10), &mut reset)
                    .await
                    .is_err()
            );
            request.result_tx.send(Ok(())).await.unwrap();
            reset.await.unwrap();
        }
    }

    #[tokio::test]
    async fn maps_engine_reset_failures_and_closed_channels() {
        let (engine_tx, mut engine_rx) = mpsc::channel(1);
        let rpc = AdminRpc::new(None, false, engine_tx, mpsc::channel(1).0);
        let (result, ()) = tokio::join!(rpc.admin_reset_derivation_pipeline(), async {
            let EngineActorRequest::Reset(request) = engine_rx.recv().await.unwrap() else {
                panic!("expected engine reset");
            };
            request
                .result_tx
                .send(Err(kona_engine::EngineRequestError::ResetForkchoiceError(
                    "reset failed".to_owned(),
                )))
                .await
                .unwrap();
        });
        assert_eq!(result.unwrap_err().code(), ErrorCode::InternalError.code());
        let (result, ()) = tokio::join!(rpc.admin_reset_derivation_pipeline(), async {
            drop(engine_rx.recv().await.unwrap());
        });
        assert_eq!(result.unwrap_err().code(), ErrorCode::InternalError.code());
        drop(engine_rx);
        assert_eq!(
            rpc.admin_reset_derivation_pipeline().await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
    }

    #[tokio::test]
    async fn validates_payloads_before_forwarding_on_validators() {
        let (tx, mut rx) = mpsc::channel(1);
        let rpc = AdminRpc::new(None, false, mpsc::channel(1).0, tx);
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
        assert!(matches!(rx.try_recv(), Err(mpsc::error::TryRecvError::Empty)));

        payload.block_hash = payload.clone().into_block_raw().unwrap().header.hash_slow();
        let envelope = OpExecutionPayloadEnvelope::V1(payload);
        rpc.admin_post_unsafe_payload(envelope.clone()).await.unwrap();
        assert_eq!(rx.recv().await.unwrap(), envelope);
        drop(rx);
        assert_eq!(
            rpc.admin_post_unsafe_payload(envelope).await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
    }
}
