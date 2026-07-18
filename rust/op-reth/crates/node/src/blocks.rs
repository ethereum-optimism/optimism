//! Gap-free WebSocket streaming of canonical unsafe blocks.
//!
//! The server treats canonical-state notifications only as wake-ups. Every connection keeps its
//! own block-number cursor and reads all payload data from the canonical provider, so notification
//! lag or coalescing cannot create gaps in the stream.
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

use alloy_primitives::B256;
use futures_util::{SinkExt, Stream, StreamExt};
use metrics::{Counter, Gauge, Histogram, counter, gauge, histogram};
use op_alloy_rpc_types_engine::{
    OpExecutionData, OpExecutionPayload, OpExecutionPayloadEnvelope, OpExecutionPayloadV4,
};
use reth_primitives_traits::Block as _;
use reth_provider::{BlockReader, CanonStateSubscriptions};
use ssz::{Decode, Encode};
use std::{
    fmt,
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
/// P2P block version for an [`alloy_rpc_types_engine::ExecutionPayloadV1`].
pub const BLOCK_VERSION_V1: u8 = 0;
/// P2P block version for an [`alloy_rpc_types_engine::ExecutionPayloadV2`].
pub const BLOCK_VERSION_V2: u8 = 1;
/// P2P block version for an [`alloy_rpc_types_engine::ExecutionPayloadV3`].
pub const BLOCK_VERSION_V3: u8 = 2;
/// P2P block version for an [`OpExecutionPayloadV4`].
pub const BLOCK_VERSION_V4: u8 = 3;

/// Runtime blocks server configuration.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct BlocksServerConfig {
    /// TCP address on which to serve `/blocks` `WebSocket` upgrades.
    pub addr: SocketAddr,
    /// Earliest block number accepted by the endpoint.
    pub min_offset: u64,
}

/// Error encoding or decoding a blocks stream message.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BlocksWireError {
    /// The message does not contain a block-version byte.
    Empty,
    /// A V3/V4 payload is missing its required parent beacon block root.
    MissingParentBeaconBlockRoot(u8),
    /// The P2P block version is unsupported.
    UnsupportedPayloadVersion(u8),
    /// The message is too short for the indicated P2P block version.
    Truncated,
    /// A payload could not be decoded from SSZ.
    InvalidPayload(String),
}

impl fmt::Display for BlocksWireError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Empty => f.write_str("empty blocks stream message"),
            Self::MissingParentBeaconBlockRoot(version) => {
                write!(f, "block version {version} requires a parent beacon block root")
            }
            Self::UnsupportedPayloadVersion(version) => {
                write!(f, "unsupported P2P block version {version}")
            }
            Self::Truncated => f.write_str("truncated blocks stream message"),
            Self::InvalidPayload(error) => write!(f, "invalid execution payload: {error}"),
        }
    }
}

impl std::error::Error for BlocksWireError {}

/// Encode an envelope using the P2P payload encoding, prefixed by its P2P block version.
pub fn encode_block_frame(
    envelope: &OpExecutionPayloadEnvelope,
) -> Result<Vec<u8>, BlocksWireError> {
    let (block_version, payload, has_parent_beacon_block_root) = match &envelope.execution_payload {
        OpExecutionPayload::V1(payload) => (BLOCK_VERSION_V1, payload.as_ssz_bytes(), false),
        OpExecutionPayload::V2(payload) => (BLOCK_VERSION_V2, payload.as_ssz_bytes(), false),
        OpExecutionPayload::V3(payload) => (BLOCK_VERSION_V3, payload.as_ssz_bytes(), true),
        OpExecutionPayload::V4(payload) => (BLOCK_VERSION_V4, payload.as_ssz_bytes(), true),
    };

    let mut frame =
        Vec::with_capacity(1 + payload.len() + usize::from(has_parent_beacon_block_root) * 32);
    frame.push(block_version);
    if has_parent_beacon_block_root {
        let root = envelope
            .parent_beacon_block_root
            .ok_or(BlocksWireError::MissingParentBeaconBlockRoot(block_version))?;
        frame.extend_from_slice(root.as_slice());
    }
    frame.extend_from_slice(&payload);
    Ok(frame)
}

/// Decode a P2P-version-prefixed payload message produced by this server.
pub fn decode_block_frame(frame: &[u8]) -> Result<OpExecutionPayloadEnvelope, BlocksWireError> {
    let (&block_version, body) = frame.split_first().ok_or(BlocksWireError::Empty)?;
    if block_version > BLOCK_VERSION_V4 {
        return Err(BlocksWireError::UnsupportedPayloadVersion(block_version));
    }

    let (parent_beacon_block_root, payload_bytes) = if block_version >= BLOCK_VERSION_V3 {
        if body.len() < 32 {
            return Err(BlocksWireError::Truncated);
        }
        (Some(B256::from_slice(&body[..32])), &body[32..])
    } else {
        (None, body)
    };
    let decode_error =
        |error: ssz::DecodeError| BlocksWireError::InvalidPayload(format!("{error:?}"));
    let execution_payload = match block_version {
        BLOCK_VERSION_V1 => OpExecutionPayload::V1(
            alloy_rpc_types_engine::ExecutionPayloadV1::from_ssz_bytes(payload_bytes)
                .map_err(decode_error)?,
        ),
        BLOCK_VERSION_V2 => OpExecutionPayload::V2(
            alloy_rpc_types_engine::ExecutionPayloadV2::from_ssz_bytes(payload_bytes)
                .map_err(decode_error)?,
        ),
        BLOCK_VERSION_V3 => OpExecutionPayload::V3(
            alloy_rpc_types_engine::ExecutionPayloadV3::from_ssz_bytes(payload_bytes)
                .map_err(decode_error)?,
        ),
        BLOCK_VERSION_V4 => OpExecutionPayload::V4(
            OpExecutionPayloadV4::from_ssz_bytes(payload_bytes).map_err(decode_error)?,
        ),
        _ => unreachable!("unsupported versions returned above"),
    };
    Ok(OpExecutionPayloadEnvelope { parent_beacon_block_root, execution_payload })
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
    config: BlocksServerConfig,
    metrics: BlocksServerMetrics,
}

impl<P> BlocksServer<P>
where
    P: BlockReader + CanonStateSubscriptions + Clone + Send + Sync + 'static,
{
    /// Bind a blocks server.
    pub async fn bind(provider: P, config: BlocksServerConfig) -> std::io::Result<Self> {
        let listener = TcpListener::bind(config.addr).await?;
        Ok(Self { listener, provider, config, metrics: BlocksServerMetrics::default() })
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
                    let config = self.config;
                    tokio::spawn(async move {
                        if let Err(error) =
                            handle_connection(stream, provider, config, metrics.clone()).await
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
    config: BlocksServerConfig,
    metrics: BlocksServerMetrics,
) -> Result<(), StreamTermination>
where
    P: BlockReader + CanonStateSubscriptions + Clone + Send + Sync + 'static,
{
    // Subscribe before observing the initial head. A notification racing with replay is therefore
    // queued and only acts as a wake-up; the canonical database remains the source of truth.
    let mut notifications = provider.canonical_state_stream();
    let mut accepted = None;
    let websocket = accept_hdr_async(stream, |request: &Request, response: Response| {
        let request = validate_handshake(request, &provider, config, &metrics)?;
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
    stream_blocks(&mut websocket, &provider, &mut notifications, accepted.start, &metrics).await
}

fn validate_handshake<P>(
    request: &Request,
    provider: &P,
    config: BlocksServerConfig,
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

    if start < config.min_offset {
        metrics.handshake_failure("before_minimum");
        return Err(handshake_error(
            StatusCode::GONE,
            format!("start {start} is before minimum offset {}", config.min_offset),
        ));
    }
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

async fn stream_blocks<P, S, N>(
    websocket: &mut WebSocketStream<TcpStream>,
    provider: &P,
    notifications: &mut S,
    mut next: u64,
    metrics: &BlocksServerMetrics,
) -> Result<(), StreamTermination>
where
    P: BlockReader + Send + Sync,
    S: Stream<Item = N> + Unpin,
{
    let mut previous_hash = None;

    loop {
        let head = provider.best_block_number().map_err(|error| {
            StreamTermination::new("provider", format!("failed to read canonical head: {error}"))
        })?;
        if next <= head {
            metrics.replay_distance.set(head.saturating_sub(next).saturating_add(1) as f64);
        } else {
            metrics.replay_distance.set(0.0);
        }

        while next <= head {
            let read_started = Instant::now();
            let block = provider.block_by_number(next).map_err(|error| {
                StreamTermination::new(
                    "provider",
                    format!("failed to read canonical block {next}: {error}"),
                )
            })?;
            metrics.database_read_latency.record(read_started.elapsed().as_secs_f64());
            let block = match block {
                Some(block) => block,
                None => {
                    metrics.missing_block_errors.increment(1);
                    return Err(StreamTermination::new(
                        "missing_block",
                        format!("canonical block {next} is unavailable"),
                    ));
                }
            };

            let ethereum_block = block.into_ethereum_block();
            let execution_data = OpExecutionData::from_block_slow(&ethereum_block);
            if execution_data.block_number() != next {
                return Err(StreamTermination::new(
                    "invalid_block_number",
                    format!(
                        "requested canonical block {next}, provider returned {}",
                        execution_data.block_number()
                    ),
                ));
            }
            if let Some(previous_hash) = previous_hash &&
                execution_data.parent_hash() != previous_hash
            {
                return Err(StreamTermination::new(
                    "parent_mismatch",
                    format!(
                        "block {next} parent {} does not match previously sent block {previous_hash}",
                        execution_data.parent_hash()
                    ),
                ));
            }

            let block_hash = execution_data.block_hash();
            let frame = encode_block_frame(&OpExecutionPayloadEnvelope {
                parent_beacon_block_root: execution_data.parent_beacon_block_root(),
                execution_payload: execution_data.payload,
            })
            .map_err(|error| StreamTermination::new("encoding", error.to_string()))?;
            send_binary(websocket, frame, metrics).await?;
            metrics.blocks_sent.increment(1);
            metrics.current_offset.set(next as f64);
            previous_hash = Some(block_hash);
            next = next.checked_add(1).ok_or_else(|| {
                StreamTermination::new("offset_overflow", "stream reached block number u64::MAX")
            })?;
        }

        // Recheck after replay before waiting. If the head advanced during the last database read
        // or WebSocket write, continue immediately. Otherwise the pre-existing subscription makes
        // any advancement racing with this check visible below.
        let rechecked_head = provider.best_block_number().map_err(|error| {
            StreamTermination::new("provider", format!("failed to recheck canonical head: {error}"))
        })?;
        if next <= rechecked_head {
            continue;
        }

        tokio::select! {
            notification = notifications.next() => {
                if notification.is_none() {
                    return Err(StreamTermination::new(
                        "notifications_closed",
                        "canonical state notification stream closed",
                    ))
                }
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
    use super::*;
    use alloy_consensus::{Block, BlockBody, Header};
    use alloy_primitives::{Address, Bloom, Bytes, U256, b256};
    use alloy_rpc_types_engine::{ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3};
    use reth_optimism_primitives::{OpBlock, OpPrimitives};
    use reth_provider::test_utils::MockEthProvider;
    use tokio::task::JoinHandle;
    use tokio_tungstenite::{accept_async, connect_async, tungstenite::Error as WebSocketError};

    fn payload_v1(number: u64) -> ExecutionPayloadV1 {
        ExecutionPayloadV1 {
            parent_hash: b256!("0101010101010101010101010101010101010101010101010101010101010101"),
            fee_recipient: Address::ZERO,
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Bloom::ZERO,
            prev_randao: B256::ZERO,
            block_number: number,
            gas_limit: 30_000_000,
            gas_used: 21_000,
            timestamp: 1,
            extra_data: Bytes::new(),
            base_fee_per_gas: U256::from(7),
            block_hash: b256!("0202020202020202020202020202020202020202020202020202020202020202"),
            transactions: vec![Bytes::from_static(&[0x01, 0x02, 0x03])],
        }
    }

    #[test]
    fn payload_versions_roundtrip() {
        let v1 = payload_v1(1);
        let v2 = ExecutionPayloadV2 { payload_inner: v1.clone(), withdrawals: Vec::new() };
        let v3 =
            ExecutionPayloadV3 { payload_inner: v2.clone(), blob_gas_used: 0, excess_blob_gas: 0 };
        let root = b256!("0303030303030303030303030303030303030303030303030303030303030303");
        let envelopes = [
            OpExecutionPayloadEnvelope {
                parent_beacon_block_root: None,
                execution_payload: OpExecutionPayload::V1(v1),
            },
            OpExecutionPayloadEnvelope {
                parent_beacon_block_root: None,
                execution_payload: OpExecutionPayload::V2(v2),
            },
            OpExecutionPayloadEnvelope {
                parent_beacon_block_root: Some(root),
                execution_payload: OpExecutionPayload::V3(v3.clone()),
            },
            OpExecutionPayloadEnvelope {
                parent_beacon_block_root: Some(root),
                execution_payload: OpExecutionPayload::V4(OpExecutionPayloadV4 {
                    payload_inner: v3,
                    withdrawals_root: B256::ZERO,
                }),
            },
        ];

        for (expected_version, envelope) in envelopes.into_iter().enumerate() {
            let frame = encode_block_frame(&envelope).unwrap();
            assert_eq!(frame[0], expected_version as u8);
            assert_eq!(&frame[1..], envelope.as_ssz_bytes());
            assert_eq!(decode_block_frame(&frame).unwrap(), envelope);
        }
    }

    #[test]
    fn parses_required_start() {
        assert_eq!(parse_start(Some("start=42")).unwrap(), 42);
        assert_eq!(parse_start(Some("other=x&start=7")).unwrap(), 7);
        assert!(parse_start(None).is_err());
        assert!(parse_start(Some("start=nope")).is_err());
        assert!(parse_start(Some("start=1&start=2")).is_err());
    }

    fn canonical_blocks(first: u64, last: u64) -> Vec<(B256, OpBlock)> {
        let mut parent_hash = B256::ZERO;
        (first..=last)
            .map(|number| {
                let block = Block {
                    header: Header {
                        parent_hash,
                        number,
                        gas_limit: 30_000_000,
                        base_fee_per_gas: Some(7),
                        timestamp: number,
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

    async fn test_server(
        provider: MockEthProvider<OpPrimitives>,
        min_offset: u64,
    ) -> (SocketAddr, JoinHandle<()>) {
        let server = BlocksServer::bind(
            provider,
            BlocksServerConfig { addr: "127.0.0.1:0".parse().unwrap(), min_offset },
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
        let (addr, server) = test_server(provider, 5).await;
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
        let (addr, server) = test_server(provider, 5).await;

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
        let (addr, server) = test_server(provider, 5).await;

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
        let (addr, server) = test_server(provider, 5).await;
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

        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        let accept = tokio::spawn(async move {
            let (stream, _) = listener.accept().await.unwrap();
            accept_async(stream).await.unwrap()
        });
        let (mut client, _) = connect_async(format!("ws://{addr}/blocks")).await.unwrap();
        let mut server_websocket = accept.await.unwrap();
        let (wake_tx, mut wake_rx) = futures::channel::mpsc::unbounded::<()>();
        let stream_provider = provider.clone();
        let stream = tokio::spawn(async move {
            stream_blocks(
                &mut server_websocket,
                &stream_provider,
                &mut wake_rx,
                5,
                &BlocksServerMetrics::default(),
            )
            .await
        });

        for expected in 5..=6 {
            let block = receive_frame(&mut client).await;
            assert_eq!(block.execution_payload.block_number(), expected);
        }

        // Model several dropped/coalesced canonical notifications: add three blocks but send only
        // one wake-up. The connection must query the latest head and replay all three from storage.
        provider.extend_blocks(blocks[2..].iter().cloned());
        wake_tx.unbounded_send(()).unwrap();
        for expected in 7..=9 {
            let block = receive_frame(&mut client).await;
            assert_eq!(block.execution_payload.block_number(), expected);
        }

        stream.abort();
    }

    #[test]
    fn decoder_rejects_unknown_version_and_truncated_messages() {
        assert_eq!(decode_block_frame(&[]), Err(BlocksWireError::Empty));
        assert_eq!(
            decode_block_frame(&[BLOCK_VERSION_V4 + 1]),
            Err(BlocksWireError::UnsupportedPayloadVersion(BLOCK_VERSION_V4 + 1))
        );
        assert_eq!(decode_block_frame(&[BLOCK_VERSION_V3]), Err(BlocksWireError::Truncated));
    }
}
