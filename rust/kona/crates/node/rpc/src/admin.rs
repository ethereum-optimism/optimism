//! Engine-owned admin RPC operations.

use crate::{AdminApiServer, EngineAdmin};
use alloy_rpc_types_engine::PayloadError;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpPayloadError};

/// Admin operations available to both validators and sequencers.
#[derive(Debug)]
pub struct AdminRpc<E> {
    engine: E,
}

impl<E: EngineAdmin> AdminRpc<E> {
    /// Constructs the handler with the engine capability that owns its operations.
    pub const fn new(engine: E) -> Self {
        Self { engine }
    }
}

#[async_trait]
impl<E: EngineAdmin + 'static> AdminApiServer for AdminRpc<E> {
    async fn admin_post_unsafe_payload(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "admin_postUnsafePayload");
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
            .post_unsafe_payload(payload)
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn admin_reset_derivation_pipeline(&self) -> RpcResult<()> {
        self.engine
            .reset_derivation_pipeline()
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use jsonrpsee::core::server::MethodsError;
    use op_alloy_consensus::OpBlock;
    use tokio::sync::mpsc;

    #[derive(Debug)]
    struct Engine(mpsc::Sender<OpExecutionPayloadEnvelope>);

    #[async_trait]
    impl EngineAdmin for Engine {
        type Error = mpsc::error::SendError<OpExecutionPayloadEnvelope>;

        async fn post_unsafe_payload(
            &self,
            payload: OpExecutionPayloadEnvelope,
        ) -> Result<(), Self::Error> {
            self.0.send(payload).await
        }

        async fn reset_derivation_pipeline(&self) -> Result<(), Self::Error> {
            Ok(())
        }
    }

    fn valid_payload() -> ExecutionPayloadV1 {
        let mut payload = ExecutionPayloadV1::from_block_slow(&OpBlock::default());
        payload.block_hash = payload.clone().into_block_raw().unwrap().header.hash_slow();
        payload
    }

    #[tokio::test]
    async fn unsafe_payload_validation_precedes_admission() {
        let (sender, mut receiver) = mpsc::channel(1);
        let rpc = AdminRpc::new(Engine(sender)).into_rpc();
        let mut payload = valid_payload();
        let valid = OpExecutionPayloadEnvelope::V1(payload.clone());
        let actual_hash = payload.block_hash;
        payload.block_hash = B256::repeat_byte(0xaa);
        let invalid_hash = payload.block_hash;

        let error = rpc
            .call::<_, ()>(
                "admin_postUnsafePayload",
                jsonrpsee::rpc_params![OpExecutionPayloadEnvelope::V1(payload)],
            )
            .await
            .unwrap_err();
        let MethodsError::JsonRpc(error) = error else {
            panic!("expected a JSON-RPC error, got {error}");
        };
        assert_eq!(error.code(), ErrorCode::InvalidParams.code());
        assert_eq!(
            error.message(),
            format!(
                "payload has bad block hash: {invalid_hash}, actual block hash is: {actual_hash}"
            )
        );
        assert!(matches!(receiver.try_recv(), Err(mpsc::error::TryRecvError::Empty)));

        rpc.call::<_, ()>("admin_postUnsafePayload", jsonrpsee::rpc_params![valid.clone()])
            .await
            .unwrap();
        assert_eq!(receiver.try_recv().unwrap(), valid);
    }

    #[tokio::test]
    async fn closed_engine_admission_returns_internal_error() {
        let (sender, receiver) = mpsc::channel(1);
        drop(receiver);
        let rpc = AdminRpc::new(Engine(sender)).into_rpc();
        let error = rpc
            .call::<_, ()>(
                "admin_postUnsafePayload",
                jsonrpsee::rpc_params![OpExecutionPayloadEnvelope::V1(valid_payload())],
            )
            .await
            .unwrap_err();
        assert!(
            matches!(error, MethodsError::JsonRpc(error) if error.code() == ErrorCode::InternalError.code())
        );
    }
}
