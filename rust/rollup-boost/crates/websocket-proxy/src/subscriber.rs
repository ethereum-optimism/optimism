use crate::metrics::Metrics;
use axum::http::Uri;
use backoff::{backoff::Backoff, ExponentialBackoff};
use futures::{SinkExt, StreamExt};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::select;
use tokio::sync::oneshot;
use tokio_tungstenite::tungstenite::Error::ConnectionClosed;
use tokio_tungstenite::tungstenite::Message;
use tokio_tungstenite::{connect_async, tungstenite::Error};
use tokio_util::bytes;
use tokio_util::sync::CancellationToken;
use tracing::{error, info, trace, warn};

#[derive(Debug, Clone)]
pub struct SubscriberOptions {
    pub max_backoff_interval: Duration,
    pub backoff_initial_interval: Duration,
    pub ping_interval: Duration,
    pub pong_timeout: Duration,
    pub initial_grace_period: Duration,
}

impl SubscriberOptions {
    pub fn with_max_backoff_interval(mut self, max_backoff_interval: Duration) -> Self {
        self.max_backoff_interval = max_backoff_interval;
        self
    }

    pub fn with_ping_interval(mut self, ping_interval: Duration) -> Self {
        self.ping_interval = ping_interval;
        self
    }

    pub fn with_pong_timeout(mut self, pong_timeout: Duration) -> Self {
        self.pong_timeout = pong_timeout;
        self
    }

    pub fn with_backoff_initial_interval(mut self, backoff_initial_interval: Duration) -> Self {
        self.backoff_initial_interval = backoff_initial_interval;
        self
    }

    pub fn with_initial_grace_period(mut self, initial_grace_period: Duration) -> Self {
        self.initial_grace_period = initial_grace_period;
        self
    }
}

impl Default for SubscriberOptions {
    fn default() -> Self {
        Self {
            max_backoff_interval: Duration::from_millis(20000),
            backoff_initial_interval: Duration::from_millis(500),
            ping_interval: Duration::from_millis(2000),
            pong_timeout: Duration::from_millis(4000),
            initial_grace_period: Duration::from_secs(5),
        }
    }
}

pub struct WebsocketSubscriber<F>
where
    F: Fn(Vec<u8>) + Send + Sync + 'static,
{
    uri: Uri,
    handler: F,
    backoff: ExponentialBackoff,
    metrics: Arc<Metrics>,
    options: SubscriberOptions,
}

impl<F> WebsocketSubscriber<F>
where
    F: Fn(Vec<u8>) + Send + Sync + 'static,
{
    pub fn new(uri: Uri, handler: F, metrics: Arc<Metrics>, options: SubscriberOptions) -> Self {
        let backoff = ExponentialBackoff {
            initial_interval: options.backoff_initial_interval,
            max_interval: options.max_backoff_interval,
            max_elapsed_time: None,
            ..Default::default()
        };

        Self {
            uri,
            handler,
            backoff,
            metrics,
            options,
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

        self.metrics.upstream_connection_attempts.increment(1);

        let (ws_stream, _) = match connect_async(&self.uri).await {
            Ok(connection) => {
                self.metrics.upstream_connection_successes.increment(1);
                connection
            }
            Err(e) => {
                self.metrics.upstream_connection_failures.increment(1);
                return Err(e);
            }
        };

        info!(
            message = "websocket connection established",
            uri = self.uri.to_string()
        );

        self.metrics.upstream_connections.increment(1);
        // Reset backoff timer on successful connection
        self.backoff.reset();

        let (mut write, mut read) = ws_stream.split();

        let (ping_error_tx, mut ping_error_rx) = oneshot::channel();
        let options = self.options.clone();
        let mut pong_deadline = Instant::now() + options.initial_grace_period;

        let ping_task = tokio::spawn(async move {
            let mut interval = tokio::time::interval(options.ping_interval);
            loop {
                interval.tick().await;
                if let Err(e) = write.send(Message::Ping(bytes::Bytes::new())).await {
                    error!(
                        message = "failed to send ping to upstream",
                        error = e.to_string()
                    );
                    let _ = ping_error_tx.send(e);
                    break;
                }
            }
        });

        let mut deadline_check = tokio::time::interval(self.options.pong_timeout / 4);

        let result = loop {
            select! {
                _ = deadline_check.tick() => {
                    if Instant::now() >= pong_deadline {
                        error!(
                            message = "pong timeout from upstream",
                            uri = self.uri.to_string()
                        );
                        break Err(ConnectionClosed);
                    }
                }
                Ok(ping_err) = &mut ping_error_rx => {
                    break Err(ping_err);
                }
                message = read.next() => {
                    let Some(msg) = message else {
                        break Ok(());
                    };
                    if let Err(e) = self.handle_message(msg, &mut pong_deadline, options.pong_timeout).await {
                        break Err(e);
                    }
                }
            }
        };

        ping_task.abort();
        result
    }

    async fn handle_message(
        &self,
        message: Result<Message, Error>,
        pong_deadline: &mut Instant,
        pong_timeout: Duration,
    ) -> Result<(), Error> {
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
                (self.handler)(text.as_bytes().to_vec());
            }
            Message::Binary(data) => {
                trace!(
                    message = "received binary message",
                    uri = self.uri.to_string(),
                    payload = ?data.as_ref()
                );
                self.metrics
                    .message_received_from_upstream(self.uri.to_string().as_str());
                (self.handler)(data.as_ref().to_vec());
            }
            Message::Pong(_) => {
                trace!(
                    message = "received pong from upstream",
                    uri = self.uri.to_string()
                );
                *pong_deadline = Instant::now() + pong_timeout;
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
    use tokio_tungstenite::accept_async;

    struct MockServer {
        addr: SocketAddr,
        message_sender: broadcast::Sender<Vec<u8>>,
        shutdown: CancellationToken,
    }

    impl MockServer {
        async fn new() -> Self {
            let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
            let addr = listener.local_addr().unwrap();
            let (tx, _) = broadcast::channel::<Vec<u8>>(100);
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
            tx: broadcast::Sender<Vec<u8>>,
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
                            Ok(data) => {
                                if let Err(e) = ws_sender.send(data.into()).await {
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
            msg: &[u8],
        ) -> Result<usize, broadcast::error::SendError<Vec<u8>>> {
            self.message_sender.send(msg.to_vec())
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
    async fn test_ping_pong_reconnection() {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let addr = listener.local_addr().unwrap();
        let uri: Uri = format!("ws://{}", addr).parse().unwrap();

        let shutdown = CancellationToken::new();
        let shutdown_server = shutdown.clone();

        let connection_count = Arc::new(std::sync::atomic::AtomicU32::new(0));
        let connection_count_server = connection_count.clone();

        tokio::spawn(async move {
            loop {
                select! {
                    _ = shutdown_server.cancelled() => break,
                    accept_result = listener.accept() => {
                        if let Ok((stream, _)) = accept_result {
                            connection_count_server.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
                            let shutdown_inner = shutdown_server.clone();
                            tokio::spawn(async move {
                                let _ws_stream = match accept_async(stream).await {
                                    Ok(ws) => ws,
                                    Err(_) => return,
                                };


                                // Become completely unresponsive - don't read any messages
                                select! {
                                    _ = shutdown_inner.cancelled() => return
                                }
                            });
                        }
                    }
                }
            }
        });

        let listener_fn = move |_data: Vec<u8>| {
            // Handler for received messages - not needed for this test
        };

        let options = SubscriberOptions::default()
            .with_backoff_initial_interval(Duration::from_millis(100))
            .with_ping_interval(Duration::from_millis(100))
            .with_pong_timeout(Duration::from_millis(200))
            .with_initial_grace_period(Duration::from_millis(50));

        let mut subscriber =
            WebsocketSubscriber::new(uri, listener_fn, Arc::new(Metrics::default()), options);

        let subscriber_task = {
            let token_clone = shutdown.clone();
            tokio::spawn(async move {
                subscriber.run(token_clone).await;
            })
        };

        // This needs to take into account the poll interval, pong deadline and the backoff interval.
        sleep(Duration::from_secs(1)).await;

        let connections = connection_count.load(std::sync::atomic::Ordering::SeqCst);
        assert!(
            connections >= 2,
            "Expected at least 2 connection attempts due to ping timeout, got {}",
            connections
        );

        shutdown.cancel();
        let _ = timeout(Duration::from_secs(1), subscriber_task).await;
    }

    #[tokio::test]
    async fn test_multiple_subscribers_single_listener() {
        let server1 = MockServer::new().await;
        let server2 = MockServer::new().await;

        let received_messages = Arc::new(Mutex::new(Vec::new()));
        let received_clone = received_messages.clone();

        let listener = move |data: Vec<u8>| {
            if let Ok(mut messages) = received_clone.lock() {
                messages.push(data);
            }
        };

        let metrics = Arc::new(Metrics::default());

        let token = CancellationToken::new();
        let token_clone1 = token.clone();
        let token_clone2 = token.clone();

        let uri1 = server1.uri();
        let listener_clone1 = listener.clone();
        let metrics_clone1 = metrics.clone();

        let mut subscriber1 = WebsocketSubscriber::new(
            uri1.clone(),
            listener_clone1,
            metrics_clone1,
            SubscriberOptions::default(),
        );

        let uri2 = server2.uri();
        let listener_clone2 = listener.clone();
        let metrics_clone2 = metrics.clone();

        let mut subscriber2 = WebsocketSubscriber::new(
            uri2.clone(),
            listener_clone2,
            metrics_clone2,
            SubscriberOptions::default(),
        );

        let task1 = tokio::spawn(async move {
            subscriber1.run(token_clone1).await;
        });

        let task2 = tokio::spawn(async move {
            subscriber2.run(token_clone2).await;
        });

        sleep(Duration::from_millis(500)).await;

        let _ = server1
            .send_message("Message from server 1".as_bytes())
            .await;
        let _ = server2
            .send_message("Message from server 2".as_bytes())
            .await;

        sleep(Duration::from_millis(500)).await;

        let _ = server1
            .send_message("Another message from server 1".as_bytes())
            .await;
        let _ = server2
            .send_message("Another message from server 2".as_bytes())
            .await;

        // Wait for messages to be processed
        sleep(Duration::from_millis(500)).await;

        // Cancel the token to shut down subscribers
        token.cancel();
        let _ = timeout(Duration::from_secs(1), task1).await;
        let _ = timeout(Duration::from_secs(1), task2).await;

        server1.shutdown().await;
        server2.shutdown().await;

        let messages = match received_messages.lock() {
            Ok(guard) => guard,
            Err(poisoned) => poisoned.into_inner(),
        };

        assert_eq!(messages.len(), 4);

        assert!(messages.contains(&"Message from server 1".as_bytes().to_vec()));
        assert!(messages.contains(&"Message from server 2".as_bytes().to_vec()));
        assert!(messages.contains(&"Another message from server 1".as_bytes().to_vec()));
        assert!(messages.contains(&"Another message from server 2".as_bytes().to_vec()));

        assert!(!messages.is_empty());
    }
}
