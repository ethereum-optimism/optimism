//! Gap-free WebSocket streaming of canonical unsafe blocks.
//!
//! Every connection keeps its own block-number cursor and reads all payload data from the
//! canonical provider, so notification lag or coalescing cannot create gaps in the stream. Commit
//! notifications only wake the cursor. Reorg notifications rewind it to the common ancestor so
//! the replacement canonical blocks are sent using their original block numbers.
//!
//! # Wire format
//!
//! Every WebSocket binary message is one block. The first byte is the P2P block version (`0` = V1,
//! `1` = V2, `2` = V3, `3` = V4). The remaining bytes are the exact uncompressed, unsigned payload
//! bytes used by OP Stack P2P gossip: the SSZ execution payload for V1/V2, and the 32-byte parent
//! beacon block root followed by the SSZ execution payload for V3/V4. P2P transports these same
//! payload bytes with a sequencer signature and Snappy compression.
//!
//! The version byte replaces the P2P topic as the payload schema discriminator. WebSocket already
//! provides message framing, so no magic, message kind, flags, or body length are needed.
//!
//! After a reorg, a connection may receive a block number it has already received. The first such
//! replacement block builds on the common ancestor. Clients must rewind their unsafe chain to that
//! parent before applying the replacement block.

mod source;

use self::source::CanonicalBlockStream;
use futures_util::{SinkExt, Stream, StreamExt};
use metrics::{Counter, Gauge, Histogram, counter, gauge, histogram};
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, encode_block_frame};
use reth_provider::{BlockReader, CanonStateSubscriptions};
use std::{
    net::SocketAddr,
    time::{Duration, Instant},
};
use tokio::net::{TcpListener, TcpStream};
use tokio_tungstenite::{
    WebSocketStream, accept_hdr_async,
    tungstenite::{
        Message,
        handshake::server::{ErrorResponse, Request, Response},
        http::{StatusCode, header::CONTENT_TYPE},
    },
};

/// Default address for the blocks server.
pub const DEFAULT_BLOCKS_SERVER_ADDR: SocketAddr =
    SocketAddr::new(std::net::IpAddr::V4(std::net::Ipv4Addr::LOCALHOST), 8548);
/// Runtime blocks server configuration.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct BlocksServerConfig {
    /// TCP address on which to serve `/blocks` `WebSocket` upgrades.
    pub addr: SocketAddr,
}

#[derive(Debug, Clone)]
struct BlocksServerMetrics {
    active_connections: Gauge,
    blocks_sent: Counter,
    bytes_sent: Counter,
    requested_offset: Gauge,
    current_offset: Gauge,
    replay_distance: Gauge,
    database_read_latency: Histogram,
    websocket_write_latency: Histogram,
    missing_block_errors: Counter,
}

impl Default for BlocksServerMetrics {
    fn default() -> Self {
        Self {
            active_connections: gauge!("optimism_blocks.active_connections"),
            blocks_sent: counter!("optimism_blocks.blocks_sent"),
            bytes_sent: counter!("optimism_blocks.bytes_sent"),
            requested_offset: gauge!("optimism_blocks.requested_offset"),
            current_offset: gauge!("optimism_blocks.current_offset"),
            replay_distance: gauge!("optimism_blocks.replay_distance"),
            database_read_latency: histogram!("optimism_blocks.database_read_latency"),
            websocket_write_latency: histogram!("optimism_blocks.websocket_write_latency"),
            missing_block_errors: counter!("optimism_blocks.missing_block_errors"),
        }
    }
}

impl BlocksServerMetrics {
    fn handshake_failure(&self, reason: &'static str) {
        counter!("optimism_blocks.handshake_failures", "reason" => reason).increment(1);
    }

    fn stream_terminated(&self, reason: &'static str) {
        counter!("optimism_blocks.stream_terminations", "reason" => reason).increment(1);
    }
}

struct ActiveConnectionGuard(Gauge);

impl Drop for ActiveConnectionGuard {
    fn drop(&mut self) {
        self.0.decrement(1.0);
    }
}

/// Bound blocks server. Bind this before spawning [`Self::run`] so address conflicts fail node
/// startup rather than being hidden in a background task.
#[derive(Debug)]
pub struct BlocksServer<P> {
    listener: TcpListener,
    provider: P,
    metrics: BlocksServerMetrics,
}

impl<P> BlocksServer<P>
where
    P: BlockReader + CanonStateSubscriptions + Clone + Send + Sync + 'static,
{
    /// Bind a blocks server.
    pub async fn bind(provider: P, config: BlocksServerConfig) -> std::io::Result<Self> {
        let listener = TcpListener::bind(config.addr).await?;
        Ok(Self { listener, provider, metrics: BlocksServerMetrics::default() })
    }

    /// Return the actual local address (useful when binding port zero in tests).
    pub fn local_addr(&self) -> std::io::Result<SocketAddr> {
        self.listener.local_addr()
    }

    /// Accept connections forever. Each connection has an independent database cursor and task.
    pub async fn run(self) {
        loop {
            match self.listener.accept().await {
                Ok((stream, peer)) => {
                    let provider = self.provider.clone();
                    let metrics = self.metrics.clone();
                    tokio::spawn(async move {
                        if let Err(error) =
                            handle_connection(stream, provider, metrics.clone()).await
                        {
                            metrics.stream_terminated(error.reason);
                            tracing::debug!(target: "reth::blocks", %peer, reason = error.reason, error = %error.message, "Blocks stream terminated");
                        }
                    });
                }
                Err(error) => {
                    tracing::warn!(target: "reth::blocks", %error, "Failed to accept blocks stream connection");
                    tokio::time::sleep(Duration::from_millis(100)).await;
                }
            }
        }
    }
}

#[derive(Debug)]
struct AcceptedRequest {
    start: u64,
}

#[derive(Debug)]
struct StreamTermination {
    reason: &'static str,
    message: String,
}

impl StreamTermination {
    fn new(reason: &'static str, message: impl Into<String>) -> Self {
        Self { reason, message: message.into() }
    }
}

async fn handle_connection<P>(
    stream: TcpStream,
    provider: P,
    metrics: BlocksServerMetrics,
) -> Result<(), StreamTermination>
where
    P: BlockReader + CanonStateSubscriptions + Clone + Send + Sync + 'static,
{
    // Subscribe before observing the initial head. A notification racing with replay is therefore
    // queued. Blocks still come from the canonical database; a reorg notification only supplies
    // the common ancestor at which the per-connection cursor must restart.
    let notifications = provider.canonical_state_stream();
    let mut accepted = None;
    let websocket = accept_hdr_async(stream, |request: &Request, response: Response| {
        let request = validate_handshake(request, &provider, &metrics)?;
        accepted = Some(request);
        Ok(response)
    })
    .await
    .map_err(|error| StreamTermination::new("handshake", error.to_string()))?;
    let accepted = accepted.ok_or_else(|| {
        StreamTermination::new("handshake", "WebSocket handshake callback was not invoked")
    })?;

    metrics.active_connections.increment(1.0);
    let _active_guard = ActiveConnectionGuard(metrics.active_connections.clone());
    metrics.requested_offset.set(accepted.start as f64);

    let mut websocket = websocket;
    let blocks =
        CanonicalBlockStream::new(provider, notifications, accepted.start, metrics.clone());
    stream_blocks(&mut websocket, blocks, &metrics).await
}

fn validate_handshake<P>(
    request: &Request,
    provider: &P,
    metrics: &BlocksServerMetrics,
) -> Result<AcceptedRequest, ErrorResponse>
where
    P: BlockReader,
{
    if request.uri().path() != "/blocks" {
        metrics.handshake_failure("not_found");
        return Err(handshake_error(StatusCode::NOT_FOUND, "not found"));
    }

    let start = match parse_start(request.uri().query()) {
        Ok(start) => start,
        Err(message) => {
            metrics.handshake_failure("invalid_start");
            return Err(handshake_error(StatusCode::BAD_REQUEST, message));
        }
    };
    let head = match provider.best_block_number() {
        Ok(head) => head,
        Err(error) => {
            metrics.handshake_failure("provider_unavailable");
            return Err(handshake_error(
                StatusCode::SERVICE_UNAVAILABLE,
                format!("canonical head unavailable: {error}"),
            ));
        }
    };

    if start > head.saturating_add(1) {
        metrics.handshake_failure("future_offset");
        return Err(handshake_error(
            StatusCode::RANGE_NOT_SATISFIABLE,
            format!("start {start} is beyond head {} plus one", head),
        ));
    }
    if start <= head {
        match provider.block_by_number(start) {
            Ok(Some(_)) => {}
            Ok(None) => {
                metrics.handshake_failure("unavailable_offset");
                return Err(handshake_error(
                    StatusCode::GONE,
                    format!("block {start} is unavailable"),
                ));
            }
            Err(error) => {
                metrics.handshake_failure("provider_unavailable");
                return Err(handshake_error(
                    StatusCode::SERVICE_UNAVAILABLE,
                    format!("failed to read block {start}: {error}"),
                ));
            }
        }
    }

    Ok(AcceptedRequest { start })
}

fn parse_start(query: Option<&str>) -> Result<u64, &'static str> {
    let query = query.ok_or("missing required start query parameter")?;
    let mut start = None;
    for parameter in query.split('&') {
        let Some((key, value)) = parameter.split_once('=') else {
            return Err("malformed query parameter");
        };
        if key == "start" {
            if start.is_some() {
                return Err("duplicate start query parameter");
            }
            start = Some(value.parse().map_err(|_| "start must be an unsigned block number")?);
        }
    }
    start.ok_or("missing required start query parameter")
}

fn handshake_error(status: StatusCode, message: impl Into<String>) -> ErrorResponse {
    Response::builder()
        .status(status)
        .header(CONTENT_TYPE, "text/plain; charset=utf-8")
        .body(Some(message.into()))
        .expect("valid static handshake response")
}

async fn stream_blocks<S>(
    websocket: &mut WebSocketStream<TcpStream>,
    mut blocks: S,
    metrics: &BlocksServerMetrics,
) -> Result<(), StreamTermination>
where
    S: Stream<Item = Result<OpExecutionPayloadEnvelope, StreamTermination>> + Unpin,
{
    loop {
        tokio::select! {
            block = blocks.next() => {
                let envelope = block.ok_or_else(|| {
                    StreamTermination::new("blocks_source_closed", "canonical block stream closed")
                })??;
                let block_number = envelope.execution_payload.block_number();
                let frame = encode_block_frame(&envelope)
                    .map_err(|error| StreamTermination::new("encoding", error.to_string()))?;
                send_binary(websocket, frame, metrics).await?;
                metrics.blocks_sent.increment(1);
                metrics.current_offset.set(block_number as f64);
            }
            message = websocket.next() => {
                match message {
                    Some(Ok(Message::Close(_))) | None => {
                        return Err(StreamTermination::new("client_disconnect", "client disconnected"))
                    }
                    Some(Ok(Message::Ping(payload))) => {
                        let started = Instant::now();
                        websocket.send(Message::Pong(payload)).await.map_err(|error| {
                            StreamTermination::new("websocket", error.to_string())
                        })?;
                        metrics.websocket_write_latency.record(started.elapsed().as_secs_f64());
                    }
                    Some(Ok(_)) => {}
                    Some(Err(error)) => {
                        return Err(StreamTermination::new("websocket", error.to_string()))
                    }
                }
            }
        }
    }
}

async fn send_binary(
    websocket: &mut WebSocketStream<TcpStream>,
    frame: Vec<u8>,
    metrics: &BlocksServerMetrics,
) -> Result<(), StreamTermination> {
    let bytes = frame.len();
    let started = Instant::now();
    websocket
        .send(Message::Binary(frame.into()))
        .await
        .map_err(|error| StreamTermination::new("websocket", error.to_string()))?;
    metrics.websocket_write_latency.record(started.elapsed().as_secs_f64());
    metrics.bytes_sent.increment(bytes as u64);
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::{source::CanonicalUpdate, *};
    use alloy_consensus::{Block, BlockBody, Header};
    use alloy_primitives::{B256, Bytes};
    use op_alloy_rpc_types_engine::decode_block_frame;
    use reth_optimism_primitives::{OpBlock, OpPrimitives};
    use reth_provider::test_utils::MockEthProvider;
    use tokio::task::JoinHandle;
    use tokio_tungstenite::{connect_async, tungstenite::Error as WebSocketError};

    #[test]
    fn parses_required_start() {
        assert_eq!(parse_start(Some("start=42")).unwrap(), 42);
        assert_eq!(parse_start(Some("other=x&start=7")).unwrap(), 7);
        assert!(parse_start(None).is_err());
        assert!(parse_start(Some("start=nope")).is_err());
        assert!(parse_start(Some("start=1&start=2")).is_err());
    }

    fn canonical_blocks(first: u64, last: u64) -> Vec<(B256, OpBlock)> {
        canonical_blocks_from(first, last, B256::ZERO, Bytes::new())
    }

    fn canonical_blocks_from(
        first: u64,
        last: u64,
        mut parent_hash: B256,
        extra_data: Bytes,
    ) -> Vec<(B256, OpBlock)> {
        (first..=last)
            .map(|number| {
                let block = Block {
                    header: Header {
                        parent_hash,
                        number,
                        gas_limit: 30_000_000,
                        base_fee_per_gas: Some(7),
                        timestamp: number,
                        extra_data: extra_data.clone(),
                        ..Default::default()
                    },
                    body: BlockBody::default(),
                };
                let hash = block.header.hash_slow();
                parent_hash = hash;
                (hash, block)
            })
            .collect()
    }

    async fn test_server(provider: MockEthProvider<OpPrimitives>) -> (SocketAddr, JoinHandle<()>) {
        let server = BlocksServer::bind(
            provider,
            BlocksServerConfig { addr: "127.0.0.1:0".parse().unwrap() },
        )
        .await
        .unwrap();
        let addr = server.local_addr().unwrap();
        let task = tokio::spawn(server.run());
        (addr, task)
    }

    async fn receive_frame(
        websocket: &mut WebSocketStream<tokio_tungstenite::MaybeTlsStream<TcpStream>>,
    ) -> OpExecutionPayloadEnvelope {
        let message = tokio::time::timeout(Duration::from_secs(2), websocket.next())
            .await
            .expect("server should send a frame")
            .expect("connection should remain open")
            .expect("frame should be valid WebSocket data");
        let Message::Binary(frame) = message else { panic!("expected binary frame") };
        decode_block_frame(&frame).unwrap()
    }

    #[tokio::test]
    async fn replays_canonical_range_in_order() {
        let provider = MockEthProvider::<OpPrimitives>::new();
        provider.extend_blocks(canonical_blocks(5, 8));
        let (addr, server) = test_server(provider).await;
        let (mut websocket, _) =
            connect_async(format!("ws://{addr}/blocks?start=5")).await.unwrap();

        let mut previous_hash = None;
        for expected_number in 5..=8 {
            let envelope = receive_frame(&mut websocket).await;
            assert_eq!(envelope.execution_payload.block_number(), expected_number);
            if let Some(previous_hash) = previous_hash {
                assert_eq!(envelope.execution_payload.parent_hash(), previous_hash);
            }
            previous_hash = Some(envelope.execution_payload.block_hash());
        }
        server.abort();
    }

    #[tokio::test]
    async fn accepts_head_and_head_plus_one() {
        let provider = MockEthProvider::<OpPrimitives>::new();
        provider.extend_blocks(canonical_blocks(5, 8));
        let (addr, server) = test_server(provider).await;

        let (mut at_head, _) = connect_async(format!("ws://{addr}/blocks?start=8")).await.unwrap();
        let block = receive_frame(&mut at_head).await;
        assert_eq!(block.execution_payload.block_number(), 8);

        // A successful upgrade is sufficient to show that head + 1 is accepted. The mock
        // provider's notification stream closes immediately, so it cannot model waiting here.
        let (_after_head, _) = connect_async(format!("ws://{addr}/blocks?start=9")).await.unwrap();
        server.abort();
    }

    async fn handshake_status(addr: SocketAddr, target: &str) -> StatusCode {
        match connect_async(format!("ws://{addr}{target}")).await.unwrap_err() {
            WebSocketError::Http(response) => response.status(),
            error => panic!("expected HTTP handshake error, got {error}"),
        }
    }

    #[tokio::test]
    async fn rejects_invalid_unavailable_and_future_offsets() {
        let provider = MockEthProvider::<OpPrimitives>::new();
        let blocks = canonical_blocks(5, 8);
        // Keep the header for block 6 while omitting its body to emulate pruned/unavailable data.
        provider.add_header(blocks[1].0, blocks[1].1.header.clone());
        provider.add_block(blocks[3].0, blocks[3].1.clone());
        let (addr, server) = test_server(provider).await;

        assert_eq!(handshake_status(addr, "/blocks").await, StatusCode::BAD_REQUEST);
        assert_eq!(
            handshake_status(addr, "/blocks?start=not-a-number").await,
            StatusCode::BAD_REQUEST
        );
        assert_eq!(handshake_status(addr, "/blocks?start=4").await, StatusCode::GONE);
        assert_eq!(handshake_status(addr, "/blocks?start=6").await, StatusCode::GONE);
        assert_eq!(
            handshake_status(addr, "/blocks?start=10").await,
            StatusCode::RANGE_NOT_SATISFIABLE
        );
        server.abort();
    }

    #[tokio::test]
    async fn independent_subscribers_replay_different_offsets() {
        let provider = MockEthProvider::<OpPrimitives>::new();
        provider.extend_blocks(canonical_blocks(5, 8));
        let (addr, server) = test_server(provider).await;
        let (mut early, _) = connect_async(format!("ws://{addr}/blocks?start=5")).await.unwrap();
        let (mut late, _) = connect_async(format!("ws://{addr}/blocks?start=7")).await.unwrap();

        let early_first = receive_frame(&mut early).await;
        let late_first = receive_frame(&mut late).await;
        assert_eq!(early_first.execution_payload.block_number(), 5);
        assert_eq!(late_first.execution_payload.block_number(), 7);
        server.abort();
    }

    #[tokio::test]
    async fn one_notification_replays_every_new_database_block() {
        let provider = MockEthProvider::<OpPrimitives>::new();
        let blocks = canonical_blocks(5, 9);
        provider.extend_blocks(blocks[..2].iter().cloned());
        let (wake_tx, wake_rx) = futures::channel::mpsc::unbounded::<CanonicalUpdate>();
        let mut source = CanonicalBlockStream::for_test(
            provider.clone(),
            wake_rx,
            5,
            BlocksServerMetrics::default(),
        );

        for expected in 5..=6 {
            let block = source.next().await.unwrap().unwrap();
            assert_eq!(block.execution_payload.block_number(), expected);
        }

        // Model several dropped/coalesced canonical notifications: add three blocks but send only
        // one wake-up. The source must query the latest head and replay all three from storage.
        provider.extend_blocks(blocks[2..].iter().cloned());
        wake_tx.unbounded_send(CanonicalUpdate::Wake).unwrap();
        for expected in 7..=9 {
            let block = source.next().await.unwrap().unwrap();
            assert_eq!(block.execution_payload.block_number(), expected);
        }
    }

    #[tokio::test]
    async fn reorg_replays_replacement_blocks_from_fork() {
        let provider = MockEthProvider::<OpPrimitives>::new();
        let old_chain = canonical_blocks(5, 8);
        provider.extend_blocks(old_chain.iter().cloned());
        let (update_tx, update_rx) = futures::channel::mpsc::unbounded::<CanonicalUpdate>();
        let mut source = CanonicalBlockStream::for_test(
            provider.clone(),
            update_rx,
            5,
            BlocksServerMetrics::default(),
        );

        for expected in 5..=8 {
            let block = source.next().await.unwrap().unwrap();
            assert_eq!(block.execution_payload.block_number(), expected);
        }

        let fork_hash = old_chain[1].0;
        let replacement = canonical_blocks_from(7, 9, fork_hash, Bytes::from_static(&[0x42]));
        provider.blocks.lock().retain(|_, block| block.header.number <= 6);
        provider.headers.lock().retain(|_, header| header.number <= 6);
        provider.extend_blocks(replacement.iter().cloned());
        update_tx.unbounded_send(CanonicalUpdate::Reorg { fork_number: 6, fork_hash }).unwrap();

        for (index, expected_number) in (7..=9).enumerate() {
            let block = source.next().await.unwrap().unwrap();
            assert_eq!(block.execution_payload.block_number(), expected_number);
            assert_eq!(block.execution_payload.block_hash(), replacement[index].0);
            if index == 0 {
                assert_eq!(block.execution_payload.parent_hash(), fork_hash);
                assert_ne!(block.execution_payload.block_hash(), old_chain[2].0);
            }
        }
    }
}
