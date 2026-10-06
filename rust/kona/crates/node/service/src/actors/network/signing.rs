//! Bounded signing attempts outside the network event loop.

use alloy_primitives::Address;
use alloy_signer::Signature;
use kona_sources::{BlockSignerError, BlockSignerHandler, RemoteSignerError};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::time::{Duration, sleep, timeout};

#[derive(Debug)]
pub(super) struct SignedBlock {
    pub block: OpExecutionPayloadEnvelope,
    pub address: Address,
    pub signature: Signature,
}

/// Gossip is time sensitive: retry a transient signer outage briefly, then let subsequent
/// payloads proceed. Configuration and signature errors still reach the chain's supervisor.
pub(super) async fn sign_block(
    signer: &BlockSignerHandler,
    block: OpExecutionPayloadEnvelope,
    chain_id: u64,
    address: Address,
) -> Result<Option<SignedBlock>, BlockSignerError> {
    for attempt in 0..3 {
        match timeout(
            Duration::from_secs(2),
            signer.sign_block(block.payload_hash(), chain_id, address),
        )
        .await
        {
            Ok(Ok(signature)) => return Ok(Some(SignedBlock { block, address, signature })),
            Ok(Err(BlockSignerError::Remote(RemoteSignerError::SigningRPCError(err)))) => {
                warn!(target: "network", ?err, attempt, "Remote signer unavailable");
            }
            Ok(Err(err)) => return Err(err),
            Err(_) => warn!(target: "network", attempt, "Remote signing attempt timed out"),
        }
        if attempt < 2 {
            sleep(Duration::from_millis(100 << attempt)).await;
        }
    }
    warn!(target: "network", "Skipping gossip for payload after signer outage");
    Ok(None)
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_signer::SignerSync;
    use jsonrpsee::{RpcModule, server::ServerBuilder, types::ErrorObjectOwned};
    use kona_sources::RemoteSigner;
    use std::sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    };

    #[tokio::test]
    async fn signer_outage_is_bounded_and_later_payloads_recover() {
        let local = alloy_signer_local::PrivateKeySigner::random();
        let address = local.address();
        let block = OpExecutionPayloadEnvelope::V1(
            alloy_rpc_types_engine::ExecutionPayloadV1::from_block_slow(&alloy_consensus::Block::<
                op_alloy_consensus::OpTxEnvelope,
            >::default()),
        );
        let signature = local.sign_hash_sync(&block.payload_hash().signature_message(10)).unwrap();
        let calls = Arc::new(AtomicUsize::new(0));
        let server = ServerBuilder::default().build("127.0.0.1:0").await.unwrap();
        let endpoint = format!("http://{}", server.local_addr().unwrap()).parse().unwrap();
        let mut module = RpcModule::new(calls.clone());
        module.register_method("health_status", |_, _, _| "ok").unwrap();
        module
            .register_method("opsigner_signBlockPayload", move |_, calls, _| {
                if calls.fetch_add(1, Ordering::SeqCst) < 3 {
                    Err(ErrorObjectOwned::owned(-32000, "temporarily unavailable", None::<()>))
                } else {
                    Ok(serde_json::json!({"signature": signature.to_string()}))
                }
            })
            .unwrap();
        let handle = server.start(module);
        let signer = BlockSignerHandler::Remote(
            RemoteSigner {
                endpoint,
                address,
                client_cert: None,
                ca_cert: None,
                headers: Default::default(),
            }
            .start()
            .await
            .unwrap(),
        );
        assert!(sign_block(&signer, block.clone(), 10, address).await.unwrap().is_none());
        assert_eq!(calls.load(Ordering::SeqCst), 3);
        let recovered = sign_block(&signer, block, 10, address).await.unwrap().unwrap();
        assert_eq!(recovered.signature, signature);
        assert_eq!(calls.load(Ordering::SeqCst), 4);
        handle.stop().unwrap();
    }
}
