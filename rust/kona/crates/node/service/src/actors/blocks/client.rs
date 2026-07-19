use futures::StreamExt;
use op_alloy_rpc_types_engine::{BlocksWireError, OpExecutionPayloadEnvelope, decode_block_frame};
use thiserror::Error;
use tokio::net::TcpStream;
use tokio_tungstenite::{
    MaybeTlsStream, WebSocketStream, connect_async_with_config,
    tungstenite::{Message, http::StatusCode, protocol::WebSocketConfig},
};
use url::Url;

/// Maximum accepted blocks stream `WebSocket` frame and message size.
///
/// This matches the maximum decompressed OP Stack P2P gossip message size. Blocks stream frames
/// contain the same unsigned payload data with a one-byte version discriminator.
pub const MAX_BLOCK_FRAME_SIZE: usize = kona_gossip::MAX_GOSSIP_SIZE;

/// A low-level client for one canonical blocks stream connection.
#[derive(Debug)]
pub struct BlocksClient {
    websocket: WebSocketStream<MaybeTlsStream<TcpStream>>,
}

impl BlocksClient {
    /// Connects to the blocks stream at the requested inclusive block number.
    ///
    /// The configured endpoint's path and query are replaced with `/blocks?start=<number>`.
    pub async fn connect(mut endpoint: Url, start: u64) -> Result<Self, BlocksClientError> {
        endpoint.set_path("/blocks");
        endpoint.set_query(None);
        endpoint.query_pairs_mut().append_pair("start", &start.to_string());

        let (websocket, _) =
            connect_async_with_config(endpoint.as_str(), Some(blocks_websocket_config()), false)
                .await?;

        Ok(Self { websocket })
    }

    /// Waits for and decodes the next binary execution payload from the stream.
    ///
    /// Ping and pong messages are handled by the `WebSocket` implementation and skipped. Any other
    /// non-binary data message terminates the client with an error.
    pub async fn next_block(&mut self) -> Result<OpExecutionPayloadEnvelope, BlocksClientError> {
        loop {
            let message = self.websocket.next().await.ok_or(BlocksClientError::StreamEnded)??;
            match message {
                Message::Binary(frame) => return Ok(decode_block_frame(&frame)?),
                Message::Ping(_) | Message::Pong(_) => {}
                Message::Close(frame) => return Err(BlocksClientError::ServerClosed(frame)),
                Message::Text(_) => return Err(BlocksClientError::UnexpectedMessage("text")),
                Message::Frame(_) => return Err(BlocksClientError::UnexpectedMessage("raw frame")),
            }
        }
    }
}

pub(super) fn blocks_websocket_config() -> WebSocketConfig {
    WebSocketConfig::default()
        .max_message_size(Some(MAX_BLOCK_FRAME_SIZE))
        .max_frame_size(Some(MAX_BLOCK_FRAME_SIZE))
}

/// An error connecting to or consuming the canonical blocks stream.
#[derive(Debug, Error)]
pub enum BlocksClientError {
    /// The `WebSocket` connection or protocol failed.
    #[error("blocks stream WebSocket error: {0}")]
    WebSocket(#[from] tokio_tungstenite::tungstenite::Error),
    /// A binary block frame was invalid.
    #[error("invalid blocks stream frame: {0}")]
    Wire(#[from] BlocksWireError),
    /// The underlying stream ended without a `WebSocket` close message.
    #[error("blocks stream ended without a close message")]
    StreamEnded,
    /// The server closed the `WebSocket` connection.
    #[error("blocks server closed the connection: {0:?}")]
    ServerClosed(Option<tokio_tungstenite::tungstenite::protocol::CloseFrame>),
    /// The server sent a message type that is invalid on the blocks stream.
    #[error("unexpected {0} message on blocks stream")]
    UnexpectedMessage(&'static str),
}

impl BlocksClientError {
    /// Returns the HTTP status from a rejected `WebSocket` handshake, if present.
    pub fn http_status(&self) -> Option<StatusCode> {
        let Self::WebSocket(tokio_tungstenite::tungstenite::Error::Http(response)) = self else {
            return None;
        };
        Some(response.status())
    }

    /// Returns whether reconnecting may recover from this error.
    pub fn is_retryable(&self) -> bool {
        match self {
            Self::StreamEnded | Self::ServerClosed(_) => true,
            Self::WebSocket(error) => match error {
                tokio_tungstenite::tungstenite::Error::ConnectionClosed |
                tokio_tungstenite::tungstenite::Error::AlreadyClosed |
                tokio_tungstenite::tungstenite::Error::Io(_) |
                tokio_tungstenite::tungstenite::Error::Tls(_) |
                tokio_tungstenite::tungstenite::Error::Protocol(
                    tokio_tungstenite::tungstenite::error::ProtocolError::ResetWithoutClosingHandshake,
                ) => true,
                tokio_tungstenite::tungstenite::Error::Http(response) => {
                    response.status().is_server_error() ||
                        response.status() == StatusCode::RANGE_NOT_SATISFIABLE
                }
                _ => false,
            },
            Self::Wire(_) | Self::UnexpectedMessage(_) => false,
        }
    }
}
