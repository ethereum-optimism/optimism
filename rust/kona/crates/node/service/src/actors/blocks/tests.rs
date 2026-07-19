use super::{client::blocks_websocket_config, *};
use crate::{EngineClientError, EngineClientResult, NetworkEngineClient, NodeActor};
use alloy_primitives::{Address, B256, Bloom, Bytes, U256};
use alloy_rpc_types_engine::ExecutionPayloadV1;
use async_trait::async_trait;
use futures::{SinkExt, StreamExt};
use op_alloy_rpc_types_engine::{
    BLOCK_VERSION_V4, OpExecutionPayload, OpExecutionPayloadEnvelope, encode_block_frame,
};
use std::future::Future;
use tokio::{
    net::{TcpListener, TcpStream},
    sync::mpsc,
    task::JoinHandle,
    time::{Duration, timeout},
};
use tokio_tungstenite::{
    WebSocketStream, accept_hdr_async,
    tungstenite::{
        Message,
        handshake::server::{Request, Response},
        protocol::{CloseFrame, frame::coding::CloseCode},
    },
};
use url::Url;

fn payload(number: u64) -> OpExecutionPayloadEnvelope {
    OpExecutionPayloadEnvelope {
        parent_beacon_block_root: None,
        execution_payload: OpExecutionPayload::V1(ExecutionPayloadV1 {
            parent_hash: B256::repeat_byte(number.saturating_sub(1) as u8),
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
            block_hash: B256::repeat_byte(number as u8),
            transactions: vec![Bytes::from(vec![number as u8])],
        }),
    }
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
struct RecordingEngineClient {
    blocks: mpsc::UnboundedSender<OpExecutionPayloadEnvelope>,
}

#[async_trait]
impl NetworkEngineClient for RecordingEngineClient {
    async fn send_unsafe_block(&self, block: OpExecutionPayloadEnvelope) -> EngineClientResult<()> {
        self.blocks.send(block).map_err(|_| {
            EngineClientError::RequestError("recording engine channel closed".to_string())
        })
    }
}

#[tokio::test]
async fn actor_forwards_payload_to_engine() {
    let expected = payload(7);
    let frame = encode_block_frame(&expected).unwrap();
    let (endpoint, server) = spawn_server(7, |mut websocket| async move {
        websocket.send(Message::Binary(frame.into())).await.unwrap();
    })
    .await;
    let blocks_client = BlocksClient::connect(endpoint, 7).await.unwrap();
    let (block_tx, mut block_rx) = mpsc::unbounded_channel();
    let mut actor =
        BlocksClientActor::new(blocks_client, RecordingEngineClient { blocks: block_tx });

    actor.step().await.unwrap();

    assert_eq!(block_rx.recv().await.unwrap(), expected);
    server.await.unwrap();
}

#[tokio::test]
async fn actor_forwards_multiple_payloads_in_order() {
    let expected = [payload(8), payload(9), payload(10)];
    let frames =
        expected.iter().map(|payload| encode_block_frame(payload).unwrap()).collect::<Vec<_>>();
    let (endpoint, server) = spawn_server(8, |mut websocket| async move {
        for frame in frames {
            websocket.send(Message::Binary(frame.into())).await.unwrap();
        }
    })
    .await;
    let blocks_client = BlocksClient::connect(endpoint, 8).await.unwrap();
    let (block_tx, mut block_rx) = mpsc::unbounded_channel();
    let mut actor =
        BlocksClientActor::new(blocks_client, RecordingEngineClient { blocks: block_tx });

    for expected_block in expected {
        actor.step().await.unwrap();
        assert_eq!(block_rx.recv().await.unwrap(), expected_block);
    }
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
