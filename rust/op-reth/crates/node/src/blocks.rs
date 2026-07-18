//! Gap-free WebSocket streaming of canonical unsafe blocks.
//!
//! The server treats canonical-state notifications only as wake-ups. Every connection keeps its
//! own block-number cursor and reads all payload data from the canonical provider, so notification
//! lag or coalescing cannot create gaps in the stream.
//!
//! # Wire format
//!
//! Every WebSocket message is one binary frame with a 12-byte header:
//!
//! ```text
//! bytes 0..4   "OPBS"
//! byte  4      protocol version (currently 1)
//! byte  5      frame kind (0 = metadata, 1 = block)
//! byte  6      payload version for block frames (1..4), zero for metadata
//! byte  7      flags (bit 0 = parent beacon block root present)
//! bytes 8..12  little-endian body length
//! ```
//!
//! A metadata body is four little-endian `u64`s: chain ID, configured minimum offset, requested
//! offset, and the canonical head observed during the handshake. A block body is an optional
//! 32-byte parent beacon block root followed by the SSZ encoding of the indicated OP execution
//! payload version. Transactions therefore remain binary EIP-2718 bytes rather than JSON hex.
//! OP Engine API sidecar lists not committed in the block are implied by the protocol: blob
//! versioned hashes and post-Isthmus execution requests are both empty.

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
/// Magic prefix on every blocks stream frame.
pub const BLOCKS_WIRE_MAGIC: [u8; 4] = *b"OPBS";
/// Current blocks stream protocol version.
pub const BLOCKS_WIRE_VERSION: u8 = 1;
/// Metadata frame kind.
pub const METADATA_FRAME_KIND: u8 = 0;
/// Block frame kind.
pub const BLOCK_FRAME_KIND: u8 = 1;

const FRAME_HEADER_LEN: usize = 12;
const ROOT_PRESENT_FLAG: u8 = 1;

/// Runtime blocks server configuration.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct BlocksServerConfig {
    /// TCP address on which to serve `/blocks` `WebSocket` upgrades.
    pub addr: SocketAddr,
    /// Earliest block number accepted by the endpoint.
    pub min_offset: u64,
}

/// Metadata sent as the first frame of every accepted connection.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct BlocksMetadata {
    /// L2 chain ID.
    pub chain_id: u64,
    /// Configured minimum offset.
    pub min_offset: u64,
    /// Inclusive offset requested by the client.
    pub requested_offset: u64,
    /// Canonical unsafe head observed during the `WebSocket` handshake.
    pub head: u64,
}

/// A decoded blocks stream frame.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BlocksFrame {
    /// Initial connection metadata.
    Metadata(BlocksMetadata),
    /// Complete OP execution payload envelope.
    Block(Box<OpExecutionPayloadEnvelope>),
}

/// Error decoding a blocks stream frame.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum BlocksWireError {
    /// Frame is shorter than the fixed header.
    Truncated,
    /// Frame magic is not `OPBS`.
    InvalidMagic,
    /// The protocol version is unsupported.
    UnsupportedVersion(u8),
    /// The frame kind is unsupported.
    UnsupportedFrameKind(u8),
    /// The payload version is unsupported.
    UnsupportedPayloadVersion(u8),
    /// Reserved flags were set.
    UnsupportedFlags(u8),
    /// The declared body length does not match the `WebSocket` message.
    InvalidBodyLength,
    /// Metadata has an invalid shape.
    InvalidMetadata,
    /// A payload could not be decoded from SSZ.
    InvalidPayload(String),
}

impl fmt::Display for BlocksWireError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Truncated => f.write_str("truncated blocks stream frame"),
            Self::InvalidMagic => f.write_str("invalid blocks stream magic"),
            Self::UnsupportedVersion(version) => {
                write!(f, "unsupported blocks stream version {version}")
            }
            Self::UnsupportedFrameKind(kind) => write!(f, "unsupported frame kind {kind}"),
            Self::UnsupportedPayloadVersion(version) => {
                write!(f, "unsupported execution payload version {version}")
            }
            Self::UnsupportedFlags(flags) => write!(f, "unsupported frame flags {flags:#x}"),
            Self::InvalidBodyLength => f.write_str("invalid blocks stream body length"),
            Self::InvalidMetadata => f.write_str("invalid blocks stream metadata"),
            Self::InvalidPayload(error) => write!(f, "invalid execution payload: {error}"),
        }
    }
}

impl std::error::Error for BlocksWireError {}

/// Encode a metadata frame.
pub fn encode_metadata_frame(metadata: BlocksMetadata) -> Vec<u8> {
    let mut body = Vec::with_capacity(32);
    body.extend_from_slice(&metadata.chain_id.to_le_bytes());
    body.extend_from_slice(&metadata.min_offset.to_le_bytes());
    body.extend_from_slice(&metadata.requested_offset.to_le_bytes());
    body.extend_from_slice(&metadata.head.to_le_bytes());
    encode_frame_header(METADATA_FRAME_KIND, 0, 0, body)
}

/// Encode a complete OP execution payload envelope as a block frame.
pub fn encode_block_frame(envelope: &OpExecutionPayloadEnvelope) -> Vec<u8> {
    let (payload_version, payload) = match &envelope.execution_payload {
        OpExecutionPayload::V1(payload) => (1, payload.as_ssz_bytes()),
        OpExecutionPayload::V2(payload) => (2, payload.as_ssz_bytes()),
        OpExecutionPayload::V3(payload) => (3, payload.as_ssz_bytes()),
        OpExecutionPayload::V4(payload) => (4, payload.as_ssz_bytes()),
    };

    let mut flags = 0;
    let mut body = Vec::with_capacity(payload.len() + 32);
    if let Some(root) = envelope.parent_beacon_block_root {
        flags |= ROOT_PRESENT_FLAG;
        body.extend_from_slice(root.as_slice());
    }
    body.extend_from_slice(&payload);
    encode_frame_header(BLOCK_FRAME_KIND, payload_version, flags, body)
}

/// Decode a metadata or block frame produced by this server.
pub fn decode_blocks_frame(frame: &[u8]) -> Result<BlocksFrame, BlocksWireError> {
    if frame.len() < FRAME_HEADER_LEN {
        return Err(BlocksWireError::Truncated);
    }
    if frame[..4] != BLOCKS_WIRE_MAGIC {
        return Err(BlocksWireError::InvalidMagic);
    }
    if frame[4] != BLOCKS_WIRE_VERSION {
        return Err(BlocksWireError::UnsupportedVersion(frame[4]));
    }

    let kind = frame[5];
    let payload_version = frame[6];
    let flags = frame[7];
    let body_len = u32::from_le_bytes(frame[8..12].try_into().expect("four byte slice")) as usize;
    if frame.len() != FRAME_HEADER_LEN + body_len {
        return Err(BlocksWireError::InvalidBodyLength);
    }
    let body = &frame[FRAME_HEADER_LEN..];

    match kind {
        METADATA_FRAME_KIND => {
            if payload_version != 0 || flags != 0 || body.len() != 32 {
                return Err(BlocksWireError::InvalidMetadata);
            }
            let read_u64 = |offset| {
                u64::from_le_bytes(body[offset..offset + 8].try_into().expect("eight byte slice"))
            };
            Ok(BlocksFrame::Metadata(BlocksMetadata {
                chain_id: read_u64(0),
                min_offset: read_u64(8),
                requested_offset: read_u64(16),
                head: read_u64(24),
            }))
        }
        BLOCK_FRAME_KIND => {
            if flags & !ROOT_PRESENT_FLAG != 0 {
                return Err(BlocksWireError::UnsupportedFlags(flags));
            }
            let (parent_beacon_block_root, payload_bytes) = if flags & ROOT_PRESENT_FLAG != 0 {
                if body.len() < 32 {
                    return Err(BlocksWireError::InvalidBodyLength);
                }
                (Some(B256::from_slice(&body[..32])), &body[32..])
            } else {
                (None, body)
            };
            let decode_error =
                |error: ssz::DecodeError| BlocksWireError::InvalidPayload(format!("{error:?}"));
            let execution_payload = match payload_version {
                1 => OpExecutionPayload::V1(
                    alloy_rpc_types_engine::ExecutionPayloadV1::from_ssz_bytes(payload_bytes)
                        .map_err(decode_error)?,
                ),
                2 => OpExecutionPayload::V2(
                    alloy_rpc_types_engine::ExecutionPayloadV2::from_ssz_bytes(payload_bytes)
                        .map_err(decode_error)?,
                ),
                3 => OpExecutionPayload::V3(
                    alloy_rpc_types_engine::ExecutionPayloadV3::from_ssz_bytes(payload_bytes)
                        .map_err(decode_error)?,
                ),
                4 => OpExecutionPayload::V4(
                    OpExecutionPayloadV4::from_ssz_bytes(payload_bytes).map_err(decode_error)?,
                ),
                version => return Err(BlocksWireError::UnsupportedPayloadVersion(version)),
            };
            Ok(BlocksFrame::Block(Box::new(OpExecutionPayloadEnvelope {
                parent_beacon_block_root,
                execution_payload,
            })))
        }
        kind => Err(BlocksWireError::UnsupportedFrameKind(kind)),
    }
}

fn encode_frame_header(kind: u8, payload_version: u8, flags: u8, body: Vec<u8>) -> Vec<u8> {
    let body_len = u32::try_from(body.len()).expect("execution payload exceeds 4 GiB");
    let mut frame = Vec::with_capacity(FRAME_HEADER_LEN + body.len());
    frame.extend_from_slice(&BLOCKS_WIRE_MAGIC);
    frame.push(BLOCKS_WIRE_VERSION);
    frame.push(kind);
    frame.push(payload_version);
    frame.push(flags);
    frame.extend_from_slice(&body_len.to_le_bytes());
    frame.extend_from_slice(&body);
    frame
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
    chain_id: u64,
    metrics: BlocksServerMetrics,
}

impl<P> BlocksServer<P>
where
    P: BlockReader + CanonStateSubscriptions + Clone + Send + Sync + 'static,
{
    /// Bind a blocks server.
    pub async fn bind(
        provider: P,
        config: BlocksServerConfig,
        chain_id: u64,
    ) -> std::io::Result<Self> {
        let listener = TcpListener::bind(config.addr).await?;
        Ok(Self { listener, provider, config, chain_id, metrics: BlocksServerMetrics::default() })
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
                    let chain_id = self.chain_id;
                    tokio::spawn(async move {
                        if let Err(error) =
                            handle_connection(stream, provider, config, chain_id, metrics.clone())
                                .await
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
    head: u64,
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
    chain_id: u64,
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
    metrics.replay_distance.set(if accepted.start <= accepted.head {
        accepted.head - accepted.start + 1
    } else {
        0
    } as f64);

    let metadata = encode_metadata_frame(BlocksMetadata {
        chain_id,
        min_offset: config.min_offset,
        requested_offset: accepted.start,
        head: accepted.head,
    });
    let mut websocket = websocket;
    send_binary(&mut websocket, metadata, &metrics).await?;

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

    Ok(AcceptedRequest { start, head })
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
            });
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
    fn metadata_roundtrip() {
        let metadata =
            BlocksMetadata { chain_id: 10, min_offset: 100, requested_offset: 123, head: 456 };
        assert_eq!(
            decode_blocks_frame(&encode_metadata_frame(metadata)).unwrap(),
            BlocksFrame::Metadata(metadata)
        );
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

        for envelope in envelopes {
            assert_eq!(
                decode_blocks_frame(&encode_block_frame(&envelope)).unwrap(),
                BlocksFrame::Block(Box::new(envelope))
            );
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
            10,
        )
        .await
        .unwrap();
        let addr = server.local_addr().unwrap();
        let task = tokio::spawn(server.run());
        (addr, task)
    }

    async fn receive_frame(
        websocket: &mut WebSocketStream<tokio_tungstenite::MaybeTlsStream<TcpStream>>,
    ) -> BlocksFrame {
        let message = tokio::time::timeout(Duration::from_secs(2), websocket.next())
            .await
            .expect("server should send a frame")
            .expect("connection should remain open")
            .expect("frame should be valid WebSocket data");
        let Message::Binary(frame) = message else { panic!("expected binary frame") };
        decode_blocks_frame(&frame).unwrap()
    }

    #[tokio::test]
    async fn replays_canonical_range_in_order() {
        let provider = MockEthProvider::<OpPrimitives>::new();
        provider.extend_blocks(canonical_blocks(5, 8));
        let (addr, server) = test_server(provider, 5).await;
        let (mut websocket, _) =
            connect_async(format!("ws://{addr}/blocks?start=5")).await.unwrap();

        assert_eq!(
            receive_frame(&mut websocket).await,
            BlocksFrame::Metadata(BlocksMetadata {
                chain_id: 10,
                min_offset: 5,
                requested_offset: 5,
                head: 8,
            })
        );
        let mut previous_hash = None;
        for expected_number in 5..=8 {
            let BlocksFrame::Block(envelope) = receive_frame(&mut websocket).await else {
                panic!("expected block frame")
            };
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
        assert!(matches!(receive_frame(&mut at_head).await, BlocksFrame::Metadata(_)));
        let BlocksFrame::Block(block) = receive_frame(&mut at_head).await else {
            panic!("expected head block")
        };
        assert_eq!(block.execution_payload.block_number(), 8);

        let (mut after_head, _) =
            connect_async(format!("ws://{addr}/blocks?start=9")).await.unwrap();
        let BlocksFrame::Metadata(metadata) = receive_frame(&mut after_head).await else {
            panic!("expected metadata")
        };
        assert_eq!(metadata.requested_offset, 9);
        assert_eq!(metadata.head, 8);
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

        let _ = receive_frame(&mut early).await;
        let _ = receive_frame(&mut late).await;
        let BlocksFrame::Block(early_first) = receive_frame(&mut early).await else {
            panic!("expected block")
        };
        let BlocksFrame::Block(late_first) = receive_frame(&mut late).await else {
            panic!("expected block")
        };
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
            let BlocksFrame::Block(block) = receive_frame(&mut client).await else {
                panic!("expected block")
            };
            assert_eq!(block.execution_payload.block_number(), expected);
        }

        // Model several dropped/coalesced canonical notifications: add three blocks but send only
        // one wake-up. The connection must query the latest head and replay all three from storage.
        provider.extend_blocks(blocks[2..].iter().cloned());
        wake_tx.unbounded_send(()).unwrap();
        for expected in 7..=9 {
            let BlocksFrame::Block(block) = receive_frame(&mut client).await else {
                panic!("expected block")
            };
            assert_eq!(block.execution_payload.block_number(), expected);
        }

        stream.abort();
    }

    #[test]
    fn decoder_rejects_unknown_version_and_bad_length() {
        let mut frame = encode_metadata_frame(BlocksMetadata {
            chain_id: 1,
            min_offset: 0,
            requested_offset: 0,
            head: 0,
        });
        frame[4] = BLOCKS_WIRE_VERSION + 1;
        assert!(matches!(decode_blocks_frame(&frame), Err(BlocksWireError::UnsupportedVersion(_))));

        let mut frame = encode_metadata_frame(BlocksMetadata {
            chain_id: 1,
            min_offset: 0,
            requested_offset: 0,
            head: 0,
        });
        frame.pop();
        assert_eq!(decode_blocks_frame(&frame), Err(BlocksWireError::InvalidBodyLength));
    }
}
