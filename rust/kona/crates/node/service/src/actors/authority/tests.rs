//! Tests for the [`AuthorityActor`].

use super::{AppliedEpoch, AuthorityActor};
use crate::NodeActor;
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    http_client::{HttpClient, HttpClientBuilder},
    server::{Server, ServerHandle},
};
use kona_engine::{EngineActorRequest, EngineState, VerifierHead, VerifierStatus};
use kona_genesis::RollupConfig;
use kona_rpc::InteropAuthorityApiServer;
use std::{
    path::PathBuf,
    sync::{Arc, Mutex},
    time::Duration,
};
use tokio::sync::{mpsc, watch};

const EPOCH: B256 = B256::repeat_byte(0xe1);

/// An authority answering a fixed status, recording the height denied blocks were asked from.
struct FixedAuthority {
    status: VerifierStatus,
    denied_from: Arc<Mutex<Option<u64>>>,
}

#[async_trait]
impl InteropAuthorityApiServer for FixedAuthority {
    async fn chain_status(&self, _chain_id: u64, denied_from: u64) -> RpcResult<VerifierStatus> {
        *self.denied_from.lock().unwrap() = Some(denied_from);
        Ok(self.status.clone())
    }
}

fn status(heads: bool) -> VerifierStatus {
    let head = |number: u64| VerifierHead::Verified {
        block: BlockNumHash { number, hash: B256::repeat_byte(number as u8) },
        timestamp: 2 * number,
        l1: BlockNumHash { number: 3, hash: B256::repeat_byte(3) },
    };
    VerifierStatus {
        epoch: EPOCH,
        verified: heads.then(|| head(9)),
        finalized: heads.then(|| head(4)),
        denied: vec![BlockNumHash { number: 11, hash: B256::repeat_byte(11) }],
    }
}

async fn serve(status: VerifierStatus) -> (HttpClient, ServerHandle, Arc<Mutex<Option<u64>>>) {
    let denied_from = Arc::new(Mutex::new(None));
    let authority = FixedAuthority { status, denied_from: denied_from.clone() };
    let server = Server::builder().build("127.0.0.1:0").await.unwrap();
    let url = format!("http://{}", server.local_addr().unwrap());
    let handle = server.start(authority.into_rpc());
    (HttpClientBuilder::default().build(url).unwrap(), handle, denied_from)
}

fn actor(
    authority: HttpClient,
    epoch_path: Option<PathBuf>,
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
            AppliedEpoch::load(epoch_path).unwrap(),
        ),
        engine_request_rx,
    )
}

/// A node that applied the authority's epoch hands the engine the denied blocks above its
/// finalized head, then the verified head to promote and the finalized head, which the engine
/// bounds by its own L1 finality.
#[tokio::test]
async fn poll_forwards_the_denied_blocks_and_the_verified_and_finalized_heads() {
    let (client, _server, denied_from) = serve(status(true)).await;
    let (mut actor, mut requests) = actor(client, None);
    actor.applied.store(EPOCH).unwrap();

    actor.step().await.unwrap();

    assert_eq!(*denied_from.lock().unwrap(), Some(1), "asked from above the finalized genesis");
    match requests.try_recv().unwrap() {
        EngineActorRequest::ProcessDeniedBlocks(blocks) => assert_eq!(blocks, status(true).denied),
        other => panic!("expected the denied blocks, got {other:?}"),
    }
    match requests.try_recv().unwrap() {
        EngineActorRequest::ProcessVerifiedSafeHead(head) => {
            assert_eq!(Some(*head), status(true).verified)
        }
        other => panic!("expected the verified head, got {other:?}"),
    }
    match requests.try_recv().unwrap() {
        EngineActorRequest::ProcessVerifiedFinalizedHead(head) => {
            assert_eq!(Some(*head), status(true).finalized)
        }
        other => panic!("expected the finalized head, got {other:?}"),
    }
    assert!(requests.try_recv().is_err());
}

/// An authority that answers no heads yet still hands the engine its denied blocks, and moves no
/// label.
#[tokio::test]
async fn authority_without_heads_forwards_only_the_denied_blocks() {
    let (client, _server, _) = serve(status(false)).await;
    let (mut actor, mut requests) = actor(client, None);
    actor.applied.store(EPOCH).unwrap();

    actor.step().await.unwrap();

    match requests.try_recv().unwrap() {
        EngineActorRequest::ProcessDeniedBlocks(blocks) => assert_eq!(blocks, status(true).denied),
        other => panic!("expected the denied blocks, got {other:?}"),
    }
    assert!(requests.try_recv().is_err());
}

/// An unreachable authority moves no label and sends the engine nothing: it keeps the last denied
/// blocks it heard.
#[tokio::test]
async fn unreachable_authority_moves_no_label() {
    let (client, server, _) = serve(status(true)).await;
    server.stop().unwrap();
    server.stopped().await;
    let (mut actor, mut requests) = actor(client, None);

    actor.step().await.unwrap();

    assert!(requests.try_recv().is_err());
}

/// On another epoch than the one applied, also on none, the node first derives again from its
/// finalized head with the new denied blocks, and records the epoch only once the engine moved
/// its labels back, in its file, so a restart on the same epoch derives on without undoing again.
#[tokio::test]
async fn another_epoch_derives_again_before_anything_else() {
    let dir = tempfile::tempdir().unwrap();
    let path = dir.path().join("authority-epoch");
    let (client, _server, _) = serve(status(true)).await;
    let (mut actor, mut requests) = actor(client.clone(), Some(path.clone()));
    let applied = actor.applied.watch();

    let engine = tokio::spawn(async move {
        match requests.recv().await.unwrap() {
            EngineActorRequest::UndoVerifications(request) => {
                assert_eq!(request.denied, status(true).denied);
                assert_eq!(*applied.borrow(), None, "the epoch waits for the undo");
                request.done.send(()).await.unwrap();
            }
            other => panic!("expected the undo, got {other:?}"),
        }
        requests
    });
    actor.step().await.unwrap();
    let mut requests = engine.await.unwrap();

    assert!(requests.try_recv().is_err(), "nothing else of the answer is followed");
    assert_eq!(actor.applied.get(), Some(EPOCH));
    let (mut restarted, mut requests) = actor_with(client, path);
    restarted.step().await.unwrap();
    assert!(matches!(requests.try_recv().unwrap(), EngineActorRequest::ProcessDeniedBlocks(_)));
}

/// An undo the engine did not confirm records no epoch: the next poll undoes again.
#[tokio::test]
async fn an_unconfirmed_undo_records_no_epoch() {
    let (client, _server, _) = serve(status(true)).await;
    let (mut actor, mut requests) = actor(client, None);
    let engine = tokio::spawn(async move {
        let request = requests.recv().await.unwrap();
        drop(request);
        requests
    });

    actor.step().await.unwrap();
    engine.await.unwrap();

    assert_eq!(actor.applied.get(), None);
}

fn actor_with(
    client: HttpClient,
    path: PathBuf,
) -> (AuthorityActor<HttpClient>, mpsc::Receiver<EngineActorRequest>) {
    actor(client, Some(path))
}
