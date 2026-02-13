use crate::client::ClientConnection;
use crate::metrics::Metrics;
use axum::extract::ws::Message;
use futures::stream::StreamExt;
use futures::SinkExt;
use std::sync::Arc;
use std::time::Instant;
use tokio::sync::broadcast::error::RecvError;
use tokio::sync::broadcast::Sender;
use tokio::time::{interval, Duration};
use tracing::{debug, info, trace, warn};

#[derive(Clone)]
pub struct Registry {
    sender: Sender<Message>,
    metrics: Arc<Metrics>,
    ping_enabled: bool,
    pong_timeout_ms: u64,
}

impl Registry {
    pub fn new(
        sender: Sender<Message>,
        metrics: Arc<Metrics>,
        ping_enabled: bool,
        pong_timeout_ms: u64,
    ) -> Self {
        Self {
            sender,
            metrics,
            ping_enabled,
            pong_timeout_ms,
        }
    }

    pub async fn subscribe(&self, client: ClientConnection) {
        info!(message = "subscribing client", client = client.id());

        let mut receiver = self.sender.subscribe();
        let metrics = self.metrics.clone();
        metrics.new_connections.increment(1);

        let client_id = client.id();
        let (mut ws_sender, ws_receiver) = client.websocket.split();

        let (disconnect_client_tx, mut disconnect_client_rx) = tokio::sync::oneshot::channel();
        let client_reader = self.start_reader(ws_receiver, client_id.clone(), disconnect_client_tx);

        loop {
            tokio::select! {
                broadcast_result = receiver.recv() => {
                    match broadcast_result {
                        Ok(msg) => {
                            if let Err(e) = ws_sender.send(msg.clone()).await {
                                warn!(
                                    message = "failed to send data to client",
                                    client = client_id,
                                    error = e.to_string()
                                );
                                metrics.failed_messages.increment(1);
                                break;
                            }
                            trace!(message = "message sent to client", client = client_id);
                            metrics.record_message_sent(&msg);
                        }
                        Err(RecvError::Closed) => {
                            info!(message = "upstream connection closed", client = client_id);
                            break;
                        }
                        Err(RecvError::Lagged(_)) => {
                            info!(message = "client is lagging", client = client_id);
                            metrics.lagged_connections.increment(1);
                            break;
                        }
                    }
                }

                _ = &mut disconnect_client_rx => {
                    debug!(message = "client reader signaled disconnect", client = client_id);
                    break;
                }
            }
        }

        client_reader.abort();
        metrics.closed_connections.increment(1);

        info!(message = "client disconnected", client = client_id);
    }

    fn start_reader(
        &self,
        ws_receiver: futures::stream::SplitStream<axum::extract::ws::WebSocket>,
        client_id: String,
        disconnect_client_tx: tokio::sync::oneshot::Sender<()>,
    ) -> tokio::task::JoinHandle<()> {
        let ping_enabled = self.ping_enabled;
        let pong_timeout_ms = self.pong_timeout_ms;
        let metrics = self.metrics.clone();

        tokio::spawn(async move {
            let mut ws_receiver = ws_receiver;
            let mut last_pong = Instant::now();
            let mut timeout_checker = interval(Duration::from_millis(pong_timeout_ms / 4));
            let pong_timeout = Duration::from_millis(pong_timeout_ms);

            loop {
                tokio::select! {
                    msg = ws_receiver.next() => {
                        match msg {
                            Some(Ok(Message::Pong(_))) => {
                                if ping_enabled {
                                    trace!(message = "received pong from client", client = client_id);
                                    last_pong = Instant::now();
                                }
                            }
                            Some(Ok(Message::Close(_))) => {
                                trace!(message = "received close from client", client = client_id);
                                let _ = disconnect_client_tx.send(());
                                return;
                            }
                            Some(Err(e)) => {
                                trace!(
                                    message = "error receiving from client",
                                    client = client_id,
                                    error = e.to_string()
                                );
                                let _ = disconnect_client_tx.send(());
                                return;
                            }
                            None => {
                                trace!(message = "client connection closed", client = client_id);
                                let _ = disconnect_client_tx.send(());
                                return;
                            }
                            _ => {}
                        }
                    }

                    _ = timeout_checker.tick() => {
                        if ping_enabled && last_pong.elapsed() > pong_timeout  {
                            debug!(
                                message = "client pong timeout, disconnecting",
                                client = client_id,
                                elapsed_ms = last_pong.elapsed().as_millis()
                            );
                            metrics.client_pong_disconnects.increment(1);
                            let _ = disconnect_client_tx.send(());
                            return;
                        }
                    }
                }
            }
        })
    }
}
