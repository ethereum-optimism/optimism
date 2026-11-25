//! Tests for the [`AuthorityActor`].

use super::AuthorityActor;
use crate::NodeActor;
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    http_client::{HttpClient, HttpClientBuilder},
    server::{Server, ServerHandle},
};
use kona_engine::{EngineActorRequest, EngineState, FinalizeBlockId, VerifierHead};
use kona_genesis::RollupConfig;
use kona_rpc::InteropAuthorityApiServer;
use std::{sync::Arc, time::Duration};
use tokio::sync::{mpsc, watch};

/// An authority answering fixed heads.
struct FixedAuthority {
    verified: VerifierHead,
    finalized: VerifierHead,
}

#[async_trait]
impl InteropAuthorityApiServer for FixedAuthority {
    async fn verified_head(&self, _chain_id: u64) -> RpcResult<VerifierHead> {
        Ok(self.verified)
    }

    async fn finalized_head(&self, _chain_id: u64) -> RpcResult<VerifierHead> {
        Ok(self.finalized)
    }

    async fn denied_blocks(&self, _chain_id: u64, _from: u64) -> RpcResult<Vec<BlockNumHash>> {
        Ok(Vec::new())
    }
}

async fn serve(authority: FixedAuthority) -> (HttpClient, ServerHandle) {
    let server = Server::builder().build("127.0.0.1:0").await.unwrap();
    let url = format!("http://{}", server.local_addr().unwrap());
    let handle = server.start(authority.into_rpc());
    (HttpClientBuilder::default().build(url).unwrap(), handle)
}

fn actor(
    authority: HttpClient,
) -> (AuthorityActor<HttpClient>, mpsc::Receiver<EngineActorRequest>) {
    let (engine_request_tx, engine_request_rx) = mpsc::channel(8);
    let (_, engine_state_rx) = watch::channel(EngineState::default());
    let rollup = Arc::new(RollupConfig { block_time: 2, ..Default::default() });
    (
        AuthorityActor::new(
            authority,
            rollup,
            engine_request_tx,
            engine_state_rx,
            Duration::from_millis(1),
        ),
        engine_request_rx,
    )
}

/// One poll hands the engine the verified head to promote and the finalized block, pinned by
/// hash, to finalize.
#[tokio::test]
async fn poll_forwards_the_verified_and_finalized_heads() {
    let safe = BlockNumHash { number: 9, hash: B256::repeat_byte(9) };
    let finalized = BlockNumHash { number: 4, hash: B256::repeat_byte(4) };
    let (client, _server) = serve(FixedAuthority {
        verified: VerifierHead::Verified { block: safe, timestamp: 18 },
        finalized: VerifierHead::Verified { block: finalized, timestamp: 8 },
    })
    .await;
    let (mut actor, mut requests) = actor(client);

    actor.step().await.unwrap();

    match requests.try_recv().unwrap() {
        EngineActorRequest::ProcessVerifiedSafeHead(head) => {
            assert_eq!(*head, VerifierHead::Verified { block: safe, timestamp: 18 })
        }
        other => panic!("expected the verified head, got {other:?}"),
    }
    match requests.try_recv().unwrap() {
        EngineActorRequest::ProcessFinalizedL2Block(target) => {
            assert_eq!(*target, FinalizeBlockId::ByHash(finalized))
        }
        other => panic!("expected the finalized block, got {other:?}"),
    }
    assert!(requests.try_recv().is_err());
}

/// An unreachable authority moves nothing, so the engine holds its labels.
#[tokio::test]
async fn unreachable_authority_sends_the_engine_nothing() {
    let (client, server) = serve(FixedAuthority {
        verified: VerifierHead::PreActivation,
        finalized: VerifierHead::PreActivation,
    })
    .await;
    server.stop().unwrap();
    server.stopped().await;
    let (mut actor, mut requests) = actor(client);

    actor.step().await.unwrap();

    assert!(requests.try_recv().is_err());
}
