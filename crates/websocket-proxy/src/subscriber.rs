use crate::metrics::Metrics;
use axum::http::Uri;
use backoff::{backoff::Backoff, ExponentialBackoff};
use futures::StreamExt;
use std::sync::Arc;
use std::time::Duration;
use tokio::select;
use tokio_tungstenite::tungstenite::Error::ConnectionClosed;
use tokio_tungstenite::tungstenite::Message;
use tokio_tungstenite::{connect_async, tungstenite::Error};
use tokio_util::sync::CancellationToken;
use tracing::{error, info, trace, warn};

pub struct WebsocketSubscriber<F>
where
    F: Fn(String) + Send + Sync + 'static,
{
    uri: Uri,
    handler: F,
    backoff: ExponentialBackoff,
    metrics: Arc<Metrics>,
}

impl<F> WebsocketSubscriber<F>
where
    F: Fn(String) + Send + Sync + 'static,
{
    pub fn new(uri: Uri, handler: F, max_interval: u64, metrics: Arc<Metrics>) -> Self {
        let backoff = ExponentialBackoff {
            initial_interval: Duration::from_secs(1),
            max_interval: Duration::from_secs(max_interval),
            max_elapsed_time: None, // Will retry indefinitely
            ..Default::default()
        };

        Self {
            uri,
            handler,
            backoff,
            metrics,
        }
    }

    pub async fn run(&mut self, token: CancellationToken) {
        info!(
            message = "starting upstream subscription",
            uri = self.uri.to_string()
        );
        loop {
            select! {
                _ = token.cancelled() => {
                    info!(
                        message = "cancelled upstream subscription",
                        uri = self.uri.to_string()
                    );
                    return;
                }
                result = self.connect_and_listen() => {
                    match result {
                        Ok(()) => {
                            info!(
                                message = "upstream connection closed",
                                uri = self.uri.to_string()
                            );
                        }
                        Err(e) => {
                            error!(
                                message = "upstream websocket error",
                                uri = self.uri.to_string(),
                                error = e.to_string()
                            );
                            self.metrics.upstream_errors.increment(1);
                            // Decrement the active connections count when connection fails
                            self.metrics.upstream_connections.decrement(1);

                            if let Some(duration) = self.backoff.next_backoff() {
                                warn!(
                                    message = "reconnecting",
                                    uri = self.uri.to_string(),
                                    seconds = duration.as_secs()
                                );
                                select! {
                                    _ = token.cancelled() => {
                                        info!(
                                            message = "cancelled subscriber during backoff",
                                            uri = self.uri.to_string()
                                        );
                                        return
                                    }
                                    _ = tokio::time::sleep(duration) => {}
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    async fn connect_and_listen(&mut self) -> Result<(), Error> {
        info!(
            message = "connecting to websocket",
            uri = self.uri.to_string()
        );

        // Increment connection attempts counter for metrics
        self.metrics.upstream_connection_attempts.increment(1);

        // Modified connection with success/failure metrics tracking
        let (ws_stream, _) = match connect_async(&self.uri).await {
            Ok(connection) => {
                // Track successful connections
                self.metrics.upstream_connection_successes.increment(1);
                connection
            }
            Err(e) => {
                // Track failed connections
                self.metrics.upstream_connection_failures.increment(1);
                return Err(e);
            }
        };

        info!(
            message = "websocket connection established",
            uri = self.uri.to_string()
        );

        // Increment active connections counter
        self.metrics.upstream_connections.increment(1);
        // Reset backoff timer on successful connection
        self.backoff.reset();

        let (_, mut read) = ws_stream.split();

        while let Some(message) = read.next().await {
            self.handle_message(message).await?;
        }

        Ok(())
    }

    async fn handle_message(&self, message: Result<Message, Error>) -> Result<(), Error> {
        let msg = match message {
            Ok(msg) => msg,
            Err(e) => {
                error!(
                    message = "error receiving message",
                    uri = self.uri.to_string(),
                    error = e.to_string()
                );
                return Err(e);
            }
        };

        match msg {
            Message::Text(text) => {
                trace!(
                    message = "received text message",
                    uri = self.uri.to_string(),
                    payload = text.as_str()
                );
                self.metrics
                    .message_received_from_upstream(self.uri.to_string().as_str());
                (self.handler)(text.to_string());
            }
            Message::Binary(data) => {
                warn!(
                    message = "received binary message, unsupported",
                    uri = self.uri.to_string(),
                    size = data.len()
                );
            }
            Message::Close(_) => {
                info!(
                    message = "received close frame from upstream",
                    uri = self.uri.to_string()
                );
                return Err(ConnectionClosed);
            }
            _ => {}
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::metrics::Metrics;
    use axum::http::Uri;
    use futures::SinkExt;
    use std::net::SocketAddr;
    use std::sync::{Arc, Mutex};
    use tokio::net::{TcpListener, TcpStream};
    use tokio::sync::broadcast;
    use tokio::time::{sleep, timeout, Duration};
    use tokio_tungstenite::{accept_async, tungstenite::Message};

    struct MockServer {
        addr: SocketAddr,
        message_sender: broadcast::Sender<String>,
        shutdown: CancellationToken,
    }

    impl MockServer {
        async fn new() -> Self {
            let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
            let addr = listener.local_addr().unwrap();
            let (tx, _) = broadcast::channel::<String>(100);
            let shutdown = CancellationToken::new();
            let shutdown_clone = shutdown.clone();
            let tx_clone = tx.clone();

            tokio::spawn(async move {
                loop {
                    select! {
                        _ = shutdown_clone.cancelled() => {
                            break;
                        }
                        accept_result = listener.accept() => {
                            match accept_result {
                                Ok((stream, _)) => {
                                    let tx = tx_clone.clone();
                                    let shutdown = shutdown_clone.clone();
                                    tokio::spawn(async move {
                                        Self::handle_connection(stream, tx, shutdown).await;
                                    });
                                }
                                Err(e) => {
                                    eprintln!("Failed to accept: {}", e);
                                    break;
                                }
                            }
                        }
                    }
                }
            });

            Self {
                addr,
                message_sender: tx,
                shutdown,
            }
        }

        async fn handle_connection(
            stream: TcpStream,
            tx: broadcast::Sender<String>,
            shutdown: CancellationToken,
        ) {
            let ws_stream = match accept_async(stream).await {
                Ok(ws_stream) => ws_stream,
                Err(e) => {
                    eprintln!("Failed to accept websocket: {}", e);
                    return;
                }
            };

            let (mut ws_sender, _) = ws_stream.split();

            let mut rx = tx.subscribe();

            loop {
                select! {
                    _ = shutdown.cancelled() => {
                        break;
                    }
                    msg = rx.recv() => {
                        match msg {
                            Ok(text) => {
                                if let Err(e) = ws_sender.send(Message::Text(text.into())).await {
                                    eprintln!("Error sending message: {}", e);
                                    break;
                                }
                            }
                            Err(_) => {
                                break;
                            }
                        }
                    }
                }
            }
        }

        async fn send_message(
            &self,
            msg: &str,
        ) -> Result<usize, broadcast::error::SendError<String>> {
            self.message_sender.send(msg.to_string())
        }

        async fn shutdown(self) {
            self.shutdown.cancel();
        }

        fn uri(&self) -> Uri {
            format!("ws://{}", self.addr)
                .parse()
                .expect("Failed to parse URI")
        }
    }

    #[tokio::test]
    async fn test_multiple_subscribers_single_listener() {
        // Create two mock servers
        let server1 = MockServer::new().await;
        let server2 = MockServer::new().await;

        // Create a receiver for the messages
        let received_messages = Arc::new(Mutex::new(Vec::new()));
        let received_clone = received_messages.clone();

        // Create a listener function that will be shared by both subscribers
        let listener = move |data: String| {
            if let Ok(mut messages) = received_clone.lock() {
                messages.push(data);
            }
        };

        // Create metrics
        let metrics = Arc::new(Metrics::default());

        // Create cancellation token
        let token = CancellationToken::new();
        let token_clone1 = token.clone();
        let token_clone2 = token.clone();

        // Create and run the first subscriber
        let uri1 = server1.uri();
        let listener_clone1 = listener.clone();
        let metrics_clone1 = metrics.clone();

        let mut subscriber1 =
            WebsocketSubscriber::new(uri1.clone(), listener_clone1, 5, metrics_clone1);

        // Create and run the second subscriber
        let uri2 = server2.uri();
        let listener_clone2 = listener.clone();
        let metrics_clone2 = metrics.clone();

        let mut subscriber2 =
            WebsocketSubscriber::new(uri2.clone(), listener_clone2, 5, metrics_clone2);

        // Spawn tasks for subscribers
        let task1 = tokio::spawn(async move {
            subscriber1.run(token_clone1).await;
        });

        let task2 = tokio::spawn(async move {
            subscriber2.run(token_clone2).await;
        });

        // Wait for connections to establish
        sleep(Duration::from_millis(500)).await;

        // Send different messages from each server
        let _ = server1.send_message("Message from server 1").await;
        let _ = server2.send_message("Message from server 2").await;

        // Wait for messages to be processed
        sleep(Duration::from_millis(500)).await;

        // Send more messages to ensure continuous operation
        let _ = server1.send_message("Another message from server 1").await;
        let _ = server2.send_message("Another message from server 2").await;

        // Wait for messages to be processed
        sleep(Duration::from_millis(500)).await;

        // Cancel the token to shut down subscribers
        token.cancel();

        // Wait for tasks to complete
        let _ = timeout(Duration::from_secs(1), task1).await;
        let _ = timeout(Duration::from_secs(1), task2).await;

        // Shutdown the mock servers
        server1.shutdown().await;
        server2.shutdown().await;

        // Verify that messages were received
        let messages = match received_messages.lock() {
            Ok(guard) => guard,
            Err(poisoned) => poisoned.into_inner(),
        };

        assert_eq!(messages.len(), 4);

        // Check that we received messages from both servers
        assert!(messages.contains(&"Message from server 1".to_string()));
        assert!(messages.contains(&"Message from server 2".to_string()));
        assert!(messages.contains(&"Another message from server 1".to_string()));
        assert!(messages.contains(&"Another message from server 2".to_string()));

        assert!(!messages.is_empty());
    }
}
