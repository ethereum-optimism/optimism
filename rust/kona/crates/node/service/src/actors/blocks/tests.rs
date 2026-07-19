use super::{client::blocks_websocket_config, *};
use crate::{EngineClientError, EngineClientResult, NetworkEngineClient, NodeActor};
use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, B256, Bloom, Bytes, U256};
use alloy_rpc_types_engine::ExecutionPayloadV1;
use async_trait::async_trait;
use backon::ExponentialBuilder;
use futures::{SinkExt, StreamExt};
use kona_engine::{EngineState, EngineSyncStateUpdate};
use kona_protocol::{BlockInfo, L2BlockInfo};
use op_alloy_rpc_types_engine::{
    BLOCK_VERSION_V4, OpExecutionPayload, OpExecutionPayloadEnvelope, encode_block_frame,
};
use std::{
    collections::BTreeMap,
    future::Future,
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
};
use tokio::{
    net::{TcpListener, TcpStream},
    sync::{RwLock, mpsc, watch},
    task::JoinHandle,
    time::{Duration, timeout},
};
use tokio_tungstenite::{
    WebSocketStream, accept_hdr_async,
    tungstenite::{
        Message,
        handshake::server::{ErrorResponse, Request, Response},
        http::StatusCode,
        protocol::{CloseFrame, frame::coding::CloseCode},
    },
};
use url::Url;

fn payload(number: u64) -> OpExecutionPayloadEnvelope {
    payload_with_hash(
        number,
        B256::repeat_byte(number.saturating_sub(1) as u8),
        B256::repeat_byte(number as u8),
    )
}

fn payload_with_hash(
    number: u64,
    parent_hash: B256,
    block_hash: B256,
) -> OpExecutionPayloadEnvelope {
    OpExecutionPayloadEnvelope {
        parent_beacon_block_root: None,
        execution_payload: OpExecutionPayload::V1(ExecutionPayloadV1 {
            parent_hash,
            fee_recipient: Address::ZERO,
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            prev_randao: B256::ZERO,
            block_number: number,
            gas_limit: 30_000_000,
            gas_used: 21_000,
            timestamp: number,
            extra_data: Bytes::new(),
            base_fee_per_gas: U256::from(7),
            block_hash,
            transactions: vec![Bytes::from(vec![number as u8])],
        }),
    }
}

fn payload_id(payload: &OpExecutionPayloadEnvelope) -> BlockNumHash {
    BlockNumHash {
        number: payload.execution_payload.block_number(),
        hash: payload.execution_payload.block_hash(),
    }
}

fn l2_block_info(block: BlockNumHash) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo { hash: block.hash, number: block.number, ..Default::default() },
        ..Default::default()
    }
}

fn engine_state(unsafe_head: BlockNumHash, safe_head: BlockNumHash) -> EngineState {
    let mut state = EngineState::default();
    state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
        unsafe_head: Some(l2_block_info(unsafe_head)),
        cross_unsafe_head: Some(l2_block_info(unsafe_head)),
        local_safe_head: Some(l2_block_info(safe_head)),
        safe_head: Some(l2_block_info(safe_head)),
        finalized_head: Some(l2_block_info(safe_head)),
    });
    state
}

async fn spawn_server<F, Fut>(start: u64, serve: F) -> (Url, JoinHandle<()>)
where
    F: FnOnce(WebSocketStream<TcpStream>) -> Fut + Send + 'static,
    Fut: Future<Output = ()> + Send + 'static,
{
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    let task = tokio::spawn(async move {
        let (stream, _) = listener.accept().await.unwrap();
        let websocket = accept_hdr_async(stream, move |request: &Request, response: Response| {
            let expected_query = format!("start={start}");
            assert_eq!(request.uri().path(), "/blocks");
            assert_eq!(request.uri().query(), Some(expected_query.as_str()));
            Ok(response)
        })
        .await
        .unwrap();
        serve(websocket).await;
    });
    (Url::parse(&format!("ws://{addr}/ignored?old=query")).unwrap(), task)
}

#[derive(Debug)]
enum ScriptResponse {
    Blocks(Vec<OpExecutionPayloadEnvelope>),
    Reject(StatusCode),
}

#[derive(Debug)]
struct ConnectionScript {
    start: u64,
    response: ScriptResponse,
}

impl ConnectionScript {
    fn blocks(start: u64, blocks: Vec<OpExecutionPayloadEnvelope>) -> Self {
        Self { start, response: ScriptResponse::Blocks(blocks) }
    }

    const fn reject(start: u64, status: StatusCode) -> Self {
        Self { start, response: ScriptResponse::Reject(status) }
    }
}

async fn spawn_scripted_server(scripts: Vec<ConnectionScript>) -> (Url, JoinHandle<()>) {
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    let task = tokio::spawn(async move {
        for script in scripts {
            let (stream, _) = listener.accept().await.unwrap();
            let expected_start = script.start;
            let rejection = match script.response {
                ScriptResponse::Reject(status) => Some(status),
                ScriptResponse::Blocks(_) => None,
            };
            let websocket = accept_hdr_async(
                stream,
                move |request: &Request, response: Response| -> Result<Response, ErrorResponse> {
                    let expected_query = format!("start={expected_start}");
                    assert_eq!(request.uri().path(), "/blocks");
                    assert_eq!(request.uri().query(), Some(expected_query.as_str()));
                    if let Some(status) = rejection {
                        return Err(Response::builder()
                            .status(status)
                            .body(Some(status.to_string()))
                            .unwrap());
                    }
                    Ok(response)
                },
            )
            .await;

            match script.response {
                ScriptResponse::Blocks(blocks) => {
                    let mut websocket = websocket.unwrap();
                    for block in blocks {
                        let frame = encode_block_frame(&block).unwrap();
                        websocket.send(Message::Binary(frame.into())).await.unwrap();
                    }
                }
                ScriptResponse::Reject(_) => assert!(websocket.is_err()),
            }
        }
    });
    (Url::parse(&format!("ws://{addr}")).unwrap(), task)
}

#[derive(Debug)]
struct TestChain {
    blocks: BTreeMap<u64, BlockNumHash>,
    latest: BlockNumHash,
    safe: Option<BlockNumHash>,
}

#[derive(Clone, Debug)]
struct TestLocalProvider {
    chain: Arc<RwLock<TestChain>>,
}

impl TestLocalProvider {
    fn new(blocks: impl IntoIterator<Item = BlockNumHash>, safe: Option<BlockNumHash>) -> Self {
        let blocks =
            blocks.into_iter().map(|block| (block.number, block)).collect::<BTreeMap<_, _>>();
        let latest = *blocks.last_key_value().expect("test chain must not be empty").1;
        Self { chain: Arc::new(RwLock::new(TestChain { blocks, latest, safe })) }
    }

    async fn apply(&self, block: BlockNumHash) {
        let mut chain = self.chain.write().await;
        chain.blocks.retain(|number, _| *number < block.number);
        chain.blocks.insert(block.number, block);
        chain.latest = block;
    }
}

#[async_trait]
impl BlocksClientLocalProvider for TestLocalProvider {
    async fn block_by_number(
        &self,
        number: u64,
    ) -> Result<Option<BlockNumHash>, BlocksClientLocalProviderError> {
        Ok(self.chain.read().await.blocks.get(&number).copied())
    }

    async fn latest_block(&self) -> Result<BlockNumHash, BlocksClientLocalProviderError> {
        Ok(self.chain.read().await.latest)
    }

    async fn safe_block(&self) -> Result<Option<BlockNumHash>, BlocksClientLocalProviderError> {
        Ok(self.chain.read().await.safe)
    }
}

#[derive(Debug)]
struct RecordingEngineClient {
    blocks: mpsc::UnboundedSender<OpExecutionPayloadEnvelope>,
    provider: TestLocalProvider,
    state_tx: watch::Sender<EngineState>,
    auto_apply: Arc<AtomicBool>,
}

#[async_trait]
impl NetworkEngineClient for RecordingEngineClient {
    async fn send_unsafe_block(&self, block: OpExecutionPayloadEnvelope) -> EngineClientResult<()> {
        self.blocks.send(block.clone()).map_err(|_| {
            EngineClientError::RequestError("recording engine channel closed".to_string())
        })?;
        if self.auto_apply.load(Ordering::Relaxed) {
            apply_block(&self.provider, &self.state_tx, payload_id(&block)).await;
        }
        Ok(())
    }
}

async fn apply_block(
    provider: &TestLocalProvider,
    state_tx: &watch::Sender<EngineState>,
    block: BlockNumHash,
) {
    provider.apply(block).await;
    state_tx.send_modify(|state| {
        state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
            unsafe_head: Some(l2_block_info(block)),
            cross_unsafe_head: Some(l2_block_info(block)),
            ..Default::default()
        });
    });
}

fn test_actor(
    endpoint: Url,
    provider: TestLocalProvider,
    initial_state: EngineState,
) -> (
    BlocksClientActor<RecordingEngineClient, TestLocalProvider>,
    mpsc::UnboundedReceiver<OpExecutionPayloadEnvelope>,
    watch::Sender<EngineState>,
    Arc<AtomicBool>,
) {
    let (block_tx, block_rx) = mpsc::unbounded_channel();
    let (state_tx, state_rx) = watch::channel(initial_state);
    let auto_apply = Arc::new(AtomicBool::new(true));
    let engine_client = RecordingEngineClient {
        blocks: block_tx,
        provider: provider.clone(),
        state_tx: state_tx.clone(),
        auto_apply: auto_apply.clone(),
    };
    let actor = BlocksClientActor::new(
        BlocksClientConfig::new(endpoint),
        engine_client,
        provider,
        state_rx,
        0,
    )
    .with_backoff(
        ExponentialBuilder::default()
            .with_min_delay(Duration::ZERO)
            .with_max_delay(Duration::ZERO)
            .without_max_times(),
    );
    (actor, block_rx, state_tx, auto_apply)
}

#[tokio::test]
async fn actor_forwards_payload_to_engine() {
    let anchor = payload(6);
    let expected = payload(7);
    let (endpoint, server) = spawn_scripted_server(vec![ConnectionScript::blocks(
        6,
        vec![anchor.clone(), expected.clone()],
    )])
    .await;
    let anchor_id = payload_id(&anchor);
    let provider = TestLocalProvider::new([anchor_id], Some(anchor_id));
    let (mut actor, mut block_rx, _, _) =
        test_actor(endpoint, provider, engine_state(anchor_id, anchor_id));

    actor.step().await.unwrap();

    assert_eq!(block_rx.recv().await.unwrap(), expected);
    server.await.unwrap();
}

#[tokio::test]
async fn actor_forwards_multiple_payloads_in_order() {
    let anchor = payload(7);
    let expected = [payload(8), payload(9), payload(10)];
    let mut streamed = vec![anchor.clone()];
    streamed.extend(expected.iter().cloned());
    let (endpoint, server) =
        spawn_scripted_server(vec![ConnectionScript::blocks(7, streamed)]).await;
    let anchor_id = payload_id(&anchor);
    let provider = TestLocalProvider::new([anchor_id], Some(anchor_id));
    let (mut actor, mut block_rx, _, _) =
        test_actor(endpoint, provider, engine_state(anchor_id, anchor_id));

    for expected_block in expected {
        actor.step().await.unwrap();
        assert_eq!(block_rx.recv().await.unwrap(), expected_block);
    }
    server.await.unwrap();
}

#[tokio::test]
async fn actor_skips_payload_already_canonical_locally() {
    let block = payload(6);
    let (endpoint, server) = spawn_scripted_server(vec![ConnectionScript::blocks(
        6,
        vec![block.clone(), block.clone()],
    )])
    .await;
    let block_id = payload_id(&block);
    let provider = TestLocalProvider::new([block_id], Some(block_id));
    let (mut actor, mut block_rx, _, _) =
        test_actor(endpoint, provider, engine_state(block_id, block_id));

    actor.step().await.unwrap();

    assert!(block_rx.try_recv().is_err());
    server.await.unwrap();
}

#[tokio::test]
async fn actor_reconnects_from_last_applied_block() {
    let block6 = payload(6);
    let block7 = payload(7);
    let block8 = payload(8);
    let (endpoint, server) = spawn_scripted_server(vec![
        ConnectionScript::blocks(6, vec![block6.clone(), block7.clone()]),
        ConnectionScript::blocks(7, vec![block7.clone(), block8.clone()]),
    ])
    .await;
    let block6_id = payload_id(&block6);
    let provider = TestLocalProvider::new([block6_id], Some(block6_id));
    let (mut actor, mut block_rx, _, _) =
        test_actor(endpoint, provider, engine_state(block6_id, block6_id));

    actor.step().await.unwrap();
    assert_eq!(block_rx.recv().await.unwrap(), block7);
    actor.step().await.unwrap();
    assert_eq!(block_rx.recv().await.unwrap(), block8);
    server.await.unwrap();
}

#[tokio::test]
async fn actor_does_not_advance_cursor_before_block_is_applied() {
    let block6 = payload(6);
    let block7 = payload(7);
    let block8 = payload(8);
    let (endpoint, server) = spawn_scripted_server(vec![
        ConnectionScript::blocks(6, vec![block6.clone(), block7.clone()]),
        ConnectionScript::blocks(7, vec![block7.clone(), block8.clone()]),
    ])
    .await;
    let block6_id = payload_id(&block6);
    let provider = TestLocalProvider::new([block6_id], Some(block6_id));
    let (actor, mut block_rx, state_tx, auto_apply) =
        test_actor(endpoint, provider.clone(), engine_state(block6_id, block6_id));
    auto_apply.store(false, Ordering::Relaxed);

    let first_step = tokio::spawn(async move {
        let mut actor = actor;
        actor.step().await.unwrap();
        actor
    });
    let received = block_rx.recv().await.unwrap();
    assert_eq!(received, block7);
    assert!(!first_step.is_finished());

    apply_block(&provider, &state_tx, payload_id(&received)).await;
    let mut actor = first_step.await.unwrap();
    auto_apply.store(true, Ordering::Relaxed);
    actor.step().await.unwrap();
    assert_eq!(block_rx.recv().await.unwrap(), block8);
    server.await.unwrap();
}

#[tokio::test]
async fn actor_walks_back_to_common_ancestor() {
    let a1 = payload_with_hash(1, B256::ZERO, B256::repeat_byte(0x11));
    let a2 = payload_with_hash(2, payload_id(&a1).hash, B256::repeat_byte(0x22));
    let a3 = payload_with_hash(3, payload_id(&a2).hash, B256::repeat_byte(0x33));
    let b2 = payload_with_hash(2, payload_id(&a1).hash, B256::repeat_byte(0x42));
    let b3 = payload_with_hash(3, payload_id(&b2).hash, B256::repeat_byte(0x43));
    let (endpoint, server) = spawn_scripted_server(vec![
        ConnectionScript::blocks(3, vec![b3]),
        ConnectionScript::blocks(2, vec![b2.clone()]),
        ConnectionScript::blocks(1, vec![a1.clone(), b2.clone()]),
    ])
    .await;
    let a1_id = payload_id(&a1);
    let a3_id = payload_id(&a3);
    let provider = TestLocalProvider::new([a1_id, payload_id(&a2), a3_id], Some(a1_id));
    let (mut actor, mut block_rx, _, _) =
        test_actor(endpoint, provider, engine_state(a3_id, a1_id));

    actor.step().await.unwrap();

    assert_eq!(block_rx.recv().await.unwrap(), b2);
    server.await.unwrap();
}

#[tokio::test]
async fn actor_reanchors_when_server_reorgs_during_handshake() {
    let a1 = payload_with_hash(1, B256::ZERO, B256::repeat_byte(0x11));
    let a2 = payload_with_hash(2, payload_id(&a1).hash, B256::repeat_byte(0x22));
    let a3 = payload_with_hash(3, payload_id(&a2).hash, B256::repeat_byte(0x33));
    let b2 = payload_with_hash(2, payload_id(&a1).hash, B256::repeat_byte(0x42));
    let (endpoint, server) = spawn_scripted_server(vec![
        ConnectionScript::blocks(3, vec![b2.clone()]),
        ConnectionScript::blocks(1, vec![a1.clone(), b2.clone()]),
    ])
    .await;
    let a1_id = payload_id(&a1);
    let a3_id = payload_id(&a3);
    let provider = TestLocalProvider::new([a1_id, payload_id(&a2), a3_id], Some(a1_id));
    let (mut actor, mut block_rx, _, _) =
        test_actor(endpoint, provider, engine_state(a3_id, a1_id));

    actor.step().await.unwrap();

    assert_eq!(block_rx.recv().await.unwrap(), b2);
    server.await.unwrap();
}

#[tokio::test]
async fn actor_fails_on_bad_request_without_retrying() {
    let block = payload(5);
    let block_id = payload_id(&block);
    let (endpoint, server) =
        spawn_scripted_server(vec![ConnectionScript::reject(5, StatusCode::BAD_REQUEST)]).await;
    let provider = TestLocalProvider::new([block_id], Some(block_id));
    let (mut actor, _, _, _) = test_actor(endpoint, provider, engine_state(block_id, block_id));

    assert!(matches!(
        actor.step().await.unwrap_err(),
        BlocksClientActorError::Client(error)
            if error.http_status() == Some(StatusCode::BAD_REQUEST)
    ));
    server.await.unwrap();
}

#[tokio::test]
async fn actor_fails_when_required_history_is_unavailable() {
    let block = payload(5);
    let block_id = payload_id(&block);
    let (endpoint, server) =
        spawn_scripted_server(vec![ConnectionScript::reject(5, StatusCode::GONE)]).await;
    let provider = TestLocalProvider::new([block_id], Some(block_id));
    let (mut actor, _, _, _) = test_actor(endpoint, provider, engine_state(block_id, block_id));

    assert!(matches!(
        actor.step().await.unwrap_err(),
        BlocksClientActorError::HistoryUnavailable { block_number: 5 }
    ));
    server.await.unwrap();
}

#[tokio::test]
async fn actor_reanchors_from_safe_head_after_future_offset() {
    let a1 = payload_with_hash(1, B256::ZERO, B256::repeat_byte(0x11));
    let a2 = payload_with_hash(2, payload_id(&a1).hash, B256::repeat_byte(0x22));
    let a3 = payload_with_hash(3, payload_id(&a2).hash, B256::repeat_byte(0x33));
    let b2 = payload_with_hash(2, payload_id(&a1).hash, B256::repeat_byte(0x42));
    let (endpoint, server) = spawn_scripted_server(vec![
        ConnectionScript::reject(3, StatusCode::RANGE_NOT_SATISFIABLE),
        ConnectionScript::blocks(1, vec![a1.clone(), b2.clone()]),
    ])
    .await;
    let a1_id = payload_id(&a1);
    let a3_id = payload_id(&a3);
    let provider = TestLocalProvider::new([a1_id, payload_id(&a2), a3_id], Some(a1_id));
    let (mut actor, mut block_rx, _, _) =
        test_actor(endpoint, provider, engine_state(a3_id, a1_id));

    actor.step().await.unwrap();

    assert_eq!(block_rx.recv().await.unwrap(), b2);
    server.await.unwrap();
}

#[tokio::test]
async fn client_rejects_text_message() {
    let (endpoint, server) = spawn_server(1, |mut websocket| async move {
        websocket.send(Message::Text("not a block".into())).await.unwrap();
    })
    .await;
    let mut client = BlocksClient::connect(endpoint, 1).await.unwrap();

    assert!(matches!(
        client.next_block().await.unwrap_err(),
        BlocksClientError::UnexpectedMessage("text")
    ));
    server.await.unwrap();
}

#[tokio::test]
async fn client_rejects_unsupported_payload_version() {
    let (endpoint, server) = spawn_server(2, |mut websocket| async move {
        websocket.send(Message::Binary(vec![BLOCK_VERSION_V4 + 1].into())).await.unwrap();
    })
    .await;
    let mut client = BlocksClient::connect(endpoint, 2).await.unwrap();

    assert!(matches!(
        client.next_block().await.unwrap_err(),
        BlocksClientError::Wire(
            op_alloy_rpc_types_engine::BlocksWireError::UnsupportedPayloadVersion(_)
        )
    ));
    server.await.unwrap();
}

#[tokio::test]
async fn client_answers_ping_and_continues_to_next_block() {
    let expected = payload(11);
    let frame = encode_block_frame(&expected).unwrap();
    let ping = b"blocks-ping".to_vec();
    let (endpoint, server) = spawn_server(11, move |mut websocket| async move {
        websocket.send(Message::Ping(ping.clone().into())).await.unwrap();
        let response = timeout(Duration::from_secs(2), websocket.next())
            .await
            .expect("client should answer ping")
            .expect("connection should remain open")
            .expect("pong should be valid");
        assert_eq!(response, Message::Pong(ping.into()));
        websocket.send(Message::Binary(frame.into())).await.unwrap();
    })
    .await;
    let mut client = BlocksClient::connect(endpoint, 11).await.unwrap();

    assert_eq!(client.next_block().await.unwrap(), expected);
    server.await.unwrap();
}

#[tokio::test]
async fn client_reports_server_close() {
    let close = CloseFrame { code: CloseCode::Normal, reason: "done".into() };
    let expected_close = close.clone();
    let (endpoint, server) = spawn_server(12, |mut websocket| async move {
        websocket.send(Message::Close(Some(close))).await.unwrap();
    })
    .await;
    let mut client = BlocksClient::connect(endpoint, 12).await.unwrap();

    assert!(matches!(
        client.next_block().await.unwrap_err(),
        BlocksClientError::ServerClosed(Some(frame)) if frame == expected_close
    ));
    server.await.unwrap();
}

#[test]
fn client_limits_websocket_frames_and_messages() {
    let config = blocks_websocket_config();
    assert_eq!(config.max_frame_size, Some(MAX_BLOCK_FRAME_SIZE));
    assert_eq!(config.max_message_size, Some(MAX_BLOCK_FRAME_SIZE));
}
