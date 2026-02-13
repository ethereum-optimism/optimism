use super::metrics::FlashblocksWsInboundMetrics;
use crate::FlashblocksWebsocketConfig;
use backoff::ExponentialBackoff;
use backoff::backoff::Backoff;
use bytes::Bytes;
use futures::{SinkExt, StreamExt};
use lru::LruCache;
use op_alloy_rpc_types_engine::OpFlashblockPayload;
use std::io::ErrorKind::TimedOut;
use std::num::NonZeroUsize;
use std::sync::Arc;
use std::sync::Mutex;
use std::time::Duration;
use tokio::{sync::mpsc, time::interval};
use tokio_tungstenite::tungstenite::Error::Io;
use tokio_tungstenite::{connect_async, tungstenite::Message};
use tokio_util::sync::CancellationToken;
use tracing::{error, info};
use url::Url;

const MAXIMUM_PINGS: NonZeroUsize = NonZeroUsize::new(60).expect("positive number always non zero");

#[derive(Debug, thiserror::Error)]
enum FlashblocksReceiverError {
    #[error("WebSocket connection failed: {0}")]
    Connection(#[from] tokio_tungstenite::tungstenite::Error),

    #[error("Ping failed")]
    PingFailed,

    #[error("Pong timeout")]
    PongTimeout,

    #[error("Websocket haven't return the message")]
    MessageMissing,

    #[error("Connection error: {0}")]
    ConnectionError(String),

    #[error("Connection closed")]
    ConnectionClosed,

    #[error("Task panicked: {0}")]
    TaskPanic(String),

    #[error("Failed to send message to sender: {0}")]
    SendError(#[from] Box<tokio::sync::mpsc::error::SendError<OpFlashblockPayload>>),

    #[error("Ping mutex poisoned")]
    MutexPoisoned,
}

pub struct FlashblocksReceiverService {
    url: Url,
    sender: mpsc::Sender<OpFlashblockPayload>,
    websocket_config: FlashblocksWebsocketConfig,
    metrics: FlashblocksWsInboundMetrics,
}

impl FlashblocksReceiverService {
    pub fn new(
        url: Url,
        sender: mpsc::Sender<OpFlashblockPayload>,
        websocket_config: FlashblocksWebsocketConfig,
    ) -> Self {
        Self {
            url,
            sender,
            websocket_config,
            metrics: Default::default(),
        }
    }

    pub async fn run(self) {
        let mut backoff = self.websocket_config.backoff();
        let timeout = Duration::from_millis(
            self.websocket_config
                .flashblock_builder_ws_connect_timeout_ms,
        );

        info!("FlashblocksReceiverService starting reconnection loop");
        loop {
            if let Err(e) = self.connect_and_handle(&mut backoff, timeout).await {
                let interval = backoff
                    .next_backoff()
                    .unwrap_or_else(|| {
                        error!("Backoff returned None despite max_elapsed_time=None, using max_interval as fallback");
                        self.websocket_config.max_interval()
                    });
                error!(
                    "Flashblocks receiver connection error, retrying in {}ms: {}",
                    interval.as_millis(),
                    e
                );
                self.metrics.reconnect_attempts.increment(1);
                self.metrics.connection_status.set(0);
                tokio::time::sleep(interval).await;
            } else {
                // connect_and_handle should never return Ok(())
                error!("Builder websocket connection has stopped. Invariant is broken.");
                self.metrics.connection_status.set(0);
            }
        }
    }

    async fn connect_and_handle(
        &self,
        backoff: &mut ExponentialBackoff,
        timeout: Duration,
    ) -> Result<(), FlashblocksReceiverError> {
        // Timeout is used to ensure we won't get stuck in case some TCP frames go missing
        let (ws_stream, _) = tokio::time::timeout(timeout, connect_async(self.url.as_str()))
            .await
            .map_err(|_| FlashblocksReceiverError::Connection(Io(TimedOut.into())))??;
        let (mut write, mut read) = ws_stream.split();

        info!("Connected to Flashblocks receiver at {}", self.url);
        self.metrics.connection_status.set(1);

        let cancel_token = CancellationToken::new();
        let cancel_for_ping = cancel_token.clone();

        // LRU cache with capacity of 60 pings - automatically evicts oldest entries
        let ping_cache = Arc::new(Mutex::new(LruCache::new(MAXIMUM_PINGS)));
        let pong_cache = ping_cache.clone();
        let mut ping_interval = interval(self.websocket_config.ping_interval());
        let ping_task = tokio::spawn(async move {
            loop {
                tokio::select! {
                    _ = ping_interval.tick() => {
                        let uuid = uuid::Uuid::now_v7();
                        if write.send(Message::Ping(Bytes::copy_from_slice(uuid.as_bytes().as_slice()))).await.is_err() {
                            return Err(FlashblocksReceiverError::PingFailed);
                        }
                        match ping_cache.lock() {
                            Ok(mut cache) => {
                                cache.put(uuid, ());
                            }
                            Err(_) => {
                                return Err(FlashblocksReceiverError::MutexPoisoned);
                            }
                        }
                    }
                    _ = cancel_for_ping.cancelled() => {
                        tracing::debug!("Ping task cancelled");
                        if let Err(e) = write.close().await {
                            tracing::warn!("Failed to close builder ws connection: {}", e);
                        }
                        return Ok(());
                    }
                }
            }
        });

        let sender = self.sender.clone();
        let metrics = self.metrics.clone();

        let pong_timeout = self.websocket_config.pong_interval();
        let message_handle = tokio::spawn(async move {
            let mut pong_interval = interval(pong_timeout);
            // We await here because first tick executes immediately
            pong_interval.tick().await;
            loop {
                tokio::select! {
                    result = read.next() => {
                        match result {
                            Some(Ok(msg)) => match msg {
                                Message::Text(text) => {
                                    metrics.messages_received.increment(1);
                                    match serde_json::from_str::<OpFlashblockPayload>(&text) {
                                        Ok(flashblocks_msg) => sender.send(flashblocks_msg).await.map_err(|e| {
                                                FlashblocksReceiverError::SendError(Box::new(e))
                                            })?,
                                        Err(e) => error!("Failed to process flashblock, error: {e}")
                                    }
                                }
                                Message::Close(_) => {
                                    return Err(FlashblocksReceiverError::ConnectionClosed);
                                }
                                Message::Pong(data) => {
                                    match uuid::Uuid::from_slice(data.as_ref()) {
                                        Ok(uuid) => {
                                            match pong_cache.lock() {
                                                Ok(mut cache) => {
                                                    if cache.pop(&uuid).is_some() {
                                                        pong_interval.reset();
                                                    } else {
                                                        tracing::warn!("Received pong with unknown data:{}", uuid);
                                                    }
                                                }
                                                Err(_) => {
                                                    return Err(FlashblocksReceiverError::MutexPoisoned);
                                                }
                                            }
                                        }
                                        Err(e) => {
                                            tracing::warn!("Failed to parse pong: {e}");
                                        }
                                    }
                                }
                                Message::Ping(_) => {},
                                msg => {
                                    tracing::warn!("Received unexpected message: {:?}", msg);
                                }
                            },
                            Some(Err(e)) => {
                                return Err(FlashblocksReceiverError::ConnectionError(e.to_string()));
                            }
                            None => {
                                return Err(FlashblocksReceiverError::MessageMissing);
                            }
                        }
                    },
                    _ = pong_interval.tick() => {
                        return Err(FlashblocksReceiverError::PongTimeout);
                    }
                };
            }
        });

        let connection_start = std::time::Instant::now();

        let result = tokio::select! {
            result = message_handle => {
                result.map_err(|e| FlashblocksReceiverError::TaskPanic(e.to_string()))?
            },
            result = ping_task => {
                result.map_err(|e| FlashblocksReceiverError::TaskPanic(e.to_string()))?
            },
        };

        cancel_token.cancel();

        // Only reset backoff if connection was stable for the max_interval set
        // This prevents rapid reconnection loops when a proxy accepts and immediately drops connections
        if connection_start.elapsed() >= backoff.max_interval {
            backoff.reset();
        }
        result
    }
}

#[cfg(test)]
mod tests {
    use futures::SinkExt;
    use tokio::sync::watch;
    use tokio_tungstenite::{accept_async, tungstenite::Utf8Bytes};

    use super::*;
    use std::net::{SocketAddr, TcpListener};
    use std::sync::atomic::{AtomicBool, Ordering};

    async fn start(
        addr: SocketAddr,
    ) -> eyre::Result<(
        watch::Sender<bool>,
        mpsc::Sender<OpFlashblockPayload>,
        mpsc::Receiver<()>,
        url::Url,
    )> {
        let (term_tx, mut term_rx) = watch::channel(false);
        let (send_tx, mut send_rx) = mpsc::channel::<OpFlashblockPayload>(100);
        let (send_ping_tx, send_ping_rx) = mpsc::channel::<()>(100);

        let listener = TcpListener::bind(addr)?;
        let url = Url::parse(&format!("ws://{addr}"))?;

        listener
            .set_nonblocking(true)
            .expect("Failed to set TcpListener socket to non-blocking");

        let listener = tokio::net::TcpListener::from_std(listener)
            .expect("Failed to convert TcpListener to tokio TcpListener");

        tokio::spawn(async move {
            loop {
                tokio::select! {
                    _ = term_rx.changed() => {
                        if *term_rx.borrow() {
                            return;
                        }
                    }

                    result = listener.accept() => {
                        match result {
                            Ok((connection, _addr)) => {
                                match accept_async(connection).await {
                                    Ok(ws_stream) => {
                                        let (mut write, mut read) = ws_stream.split();

                                        loop {
                                            tokio::select! {
                                                Some(msg) = send_rx.recv() => {
                                                    let serialized = serde_json::to_string(&msg).expect("message serialized");
                                                    let utf8_bytes = Utf8Bytes::from(serialized);

                                                    write.send(Message::Text(utf8_bytes)).await.expect("message sent");
                                                },
                                                msg = read.next() => {
                                                    match msg {
                                                        // we need to read for the library to handle pong messages
                                                        Some(Ok(Message::Ping(_))) => {
                                                            send_ping_tx.send(()).await.expect("ping notification sent");
                                                        },
                                                        _ => {}
                                                    }
                                                }
                                                _ = term_rx.changed() => {
                                                    if *term_rx.borrow() {
                                                        return;
                                                    }
                                                }
                                            }
                                        }
                                    }
                                    Err(e) => {
                                        eprintln!("Failed to accept WebSocket connection: {}", e);
                                    }
                                }
                            }
                            Err(e) => {
                                // Optionally break or continue based on error type
                                if e.kind() == std::io::ErrorKind::Interrupted {
                                    break;
                                }
                            }
                        }
                    }
                }
            }
        });

        Ok((term_tx, send_tx, send_ping_rx, url))
    }

    async fn start_ping_server(
        addr: SocketAddr,
        send_pongs: Arc<AtomicBool>,
    ) -> eyre::Result<(watch::Receiver<bool>, mpsc::Receiver<Bytes>, url::Url)> {
        let (term_tx, term_rx) = watch::channel(false);
        let (send_ping_tx, send_ping_rx) = mpsc::channel(100);

        let listener = TcpListener::bind(addr)?;
        let url = Url::parse(&format!("ws://{addr}"))?;

        listener
            .set_nonblocking(true)
            .expect("can set TcpListener socket to non-blocking");

        let listener = tokio::net::TcpListener::from_std(listener)
            .expect("can convert TcpListener to tokio TcpListener");

        tokio::spawn(async move {
            loop {
                let result = listener.accept().await;
                match result {
                    Ok((connection, _addr)) => {
                        match accept_async(connection).await {
                            Ok(ws_stream) => {
                                let (_, mut read) = ws_stream.split();
                                loop {
                                    if send_pongs.load(Ordering::Relaxed) {
                                        let msg = read.next().await;
                                        match msg {
                                            // we need to read for the library to handle pong messages
                                            Some(Ok(Message::Ping(data))) => {
                                                send_ping_tx
                                                    .send(data)
                                                    .await
                                                    .expect("ping data sent");
                                            }
                                            Some(Err(_)) => {
                                                break;
                                            }
                                            _ => {}
                                        }
                                    } else {
                                        tokio::time::sleep(tokio::time::Duration::from_millis(1))
                                            .await;
                                    }
                                }
                            }
                            Err(e) => {
                                eprintln!("Failed to accept WebSocket connection: {}", e);
                            }
                        }
                    }
                    Err(e) => {
                        // Optionally break or continue based on error type
                        if e.kind() == std::io::ErrorKind::Interrupted {
                            break;
                        }
                    }
                }
                // If we have broken from the loop it means reconnection occurred
                term_tx.send(true).expect("channel is up");
            }
        });

        Ok((term_rx, send_ping_rx, url))
    }

    #[tokio::test]
    async fn test_flashblocks_receiver_service() -> eyre::Result<()> {
        let addr = "127.0.0.1:8080"
            .parse::<SocketAddr>()
            .expect("valid socket address");
        let (term, send_msg, _, url) = start(addr).await?;

        let (tx, mut rx) = mpsc::channel(100);

        let config = FlashblocksWebsocketConfig {
            flashblock_builder_ws_initial_reconnect_ms: 100,
            flashblock_builder_ws_max_reconnect_ms: 100,
            flashblock_builder_ws_ping_interval_ms: 500,
            flashblock_builder_ws_pong_timeout_ms: 2000,
            flashblock_builder_ws_connect_timeout_ms: 5000,
        };
        let service = FlashblocksReceiverService::new(url, tx, config);
        let _ = tokio::spawn(async move {
            service.run().await;
        });

        // Send a message to the websocket server
        send_msg
            .send(OpFlashblockPayload::default())
            .await
            .expect("message sent to websocket server");

        let msg = rx.recv().await.expect("message received from websocket");
        assert_eq!(msg, OpFlashblockPayload::default());

        // Drop the websocket server and start another one with the same address
        // The FlashblocksReceiverService should reconnect to the new server
        term.send(true).expect("termination signal sent");

        // sleep for 1 second to ensure the server is dropped
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;

        // start a new server with the same address
        let (term, send_msg, _, _url) = start(addr).await?;
        send_msg
            .send(OpFlashblockPayload::default())
            .await
            .expect("message sent to websocket server");

        let msg = rx.recv().await.expect("message received from websocket");
        assert_eq!(msg, OpFlashblockPayload::default());
        term.send(true).expect("termination signal sent");

        Ok(())
    }

    #[tokio::test]
    async fn test_flashblocks_receiver_service_ping_pong() -> eyre::Result<()> {
        // test that if the builder is not sending any messages back, the service will send
        // ping messages to test the connection periodically

        let addr = "127.0.0.1:8081"
            .parse::<SocketAddr>()
            .expect("valid socket address");
        let send_pongs = Arc::new(AtomicBool::new(true));
        let (term, mut ping_rx, url) = start_ping_server(addr, send_pongs.clone()).await?;
        let config = FlashblocksWebsocketConfig {
            flashblock_builder_ws_initial_reconnect_ms: 100,
            flashblock_builder_ws_max_reconnect_ms: 1000,
            flashblock_builder_ws_ping_interval_ms: 500,
            flashblock_builder_ws_pong_timeout_ms: 2000,
            flashblock_builder_ws_connect_timeout_ms: 5000,
        };

        let (tx, _rx) = mpsc::channel(100);
        let service = FlashblocksReceiverService::new(url, tx, config);
        let _ = tokio::spawn(async move {
            service.run().await;
        });

        // even if we do not send any messages, we should receive pings to keep the connection alive
        for _ in 0..5 {
            ping_rx.recv().await.expect("ping received");
        }
        // Check that server hasn't reconnected because we have answered to pongs
        let reconnected = term.has_changed().expect("channel not closed");
        assert!(!reconnected, "not reconnected when we answered to pings");

        send_pongs.store(false, Ordering::Relaxed);
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;
        send_pongs.store(true, Ordering::Relaxed);
        // This sleep is to ensure that we will try to read socket and realise it closed
        tokio::time::sleep(std::time::Duration::from_millis(100)).await;
        // One second is not enough to break the connection
        let reconnected = term.has_changed().expect("channel not closed");
        assert!(!reconnected, "have reconnected before deadline is reached");

        send_pongs.store(false, Ordering::Relaxed);
        tokio::time::sleep(std::time::Duration::from_secs(3)).await;
        send_pongs.store(true, Ordering::Relaxed);
        // This sleep is to ensure that we will try to read socket and realise it closed
        tokio::time::sleep(std::time::Duration::from_millis(100)).await;

        // 3 seconds will cause reconnect
        let reconnected = term.has_changed().expect("channel not closed");
        assert!(reconnected, "haven't reconnected after deadline is reached");
        Ok(())
    }

    /// Starts a TCP server that accepts connections but never completes the WebSocket handshake.
    /// This simulates a stuck connection during the handshake phase.
    async fn start_stuck_server(addr: SocketAddr) -> eyre::Result<(watch::Sender<bool>, url::Url)> {
        let (term_tx, mut term_rx) = watch::channel(false);

        let listener = TcpListener::bind(addr)?;
        let url = Url::parse(&format!("ws://{addr}"))?;

        listener
            .set_nonblocking(true)
            .expect("can set TcpListener socket to non-blocking");

        let listener = tokio::net::TcpListener::from_std(listener)
            .expect("can convert TcpListener to tokio TcpListener");

        tokio::spawn(async move {
            // Store connections to keep them alive without responding
            let mut held_connections: Vec<tokio::net::TcpStream> = Vec::new();
            loop {
                tokio::select! {
                    _ = term_rx.changed() => {
                        if *term_rx.borrow() {
                            return;
                        }
                    }
                    result = listener.accept() => {
                        if let Ok((connection, _addr)) = result {
                            // Accept the TCP connection but never complete the WebSocket handshake
                            // Keep the connection alive by storing it
                            held_connections.push(connection);
                        }
                    }
                }
            }
        });

        Ok((term_tx, url))
    }

    #[tokio::test]
    async fn test_flashblocks_receiver_service_connect_timeout() -> eyre::Result<()> {
        // Test that if the WebSocket handshake hangs, the service will timeout
        let addr = "127.0.0.1:8082"
            .parse::<SocketAddr>()
            .expect("valid socket address");

        let (term, url) = start_stuck_server(addr).await?;

        let config = FlashblocksWebsocketConfig {
            flashblock_builder_ws_initial_reconnect_ms: 100,
            flashblock_builder_ws_max_reconnect_ms: 200,
            flashblock_builder_ws_ping_interval_ms: 500,
            flashblock_builder_ws_pong_timeout_ms: 2000,
            // Set a 1 second timeout for connection attempts
            flashblock_builder_ws_connect_timeout_ms: 1000,
        };

        let (tx, _rx) = mpsc::channel(100);
        let service = FlashblocksReceiverService::new(url, tx, config);

        let timeout =
            std::time::Duration::from_millis(config.flashblock_builder_ws_connect_timeout_ms);
        let mut backoff = config.backoff();

        // Call connect_and_handle directly - it should timeout and return an error
        let result = service.connect_and_handle(&mut backoff, timeout).await;

        assert!(
            result.is_err(),
            "connect_and_handle should return error on timeout"
        );

        // Verify it's a connection error (timeout is wrapped as connection error)
        let err = result.unwrap_err();
        assert!(
            matches!(err, FlashblocksReceiverError::Connection(_)),
            "expected Connection error, got: {:?}",
            err
        );

        term.send(true).expect("termination signal sent");
        Ok(())
    }
}
