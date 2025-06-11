use super::primitives::FlashblocksPayloadV1;
use crate::flashblocks::metrics::FlashblocksWsInboundMetrics;
use futures::StreamExt;
use tokio::sync::mpsc;
use tokio_tungstenite::{connect_async, tungstenite::Message};
use tracing::{error, info};
use url::Url;

pub struct FlashblocksReceiverService {
    url: Url,
    sender: mpsc::Sender<FlashblocksPayloadV1>,
    reconnect_ms: u64,
    metrics: FlashblocksWsInboundMetrics,
}

impl FlashblocksReceiverService {
    pub fn new(url: Url, sender: mpsc::Sender<FlashblocksPayloadV1>, reconnect_ms: u64) -> Self {
        Self {
            url,
            sender,
            reconnect_ms,
            metrics: Default::default(),
        }
    }

    pub async fn run(self) {
        loop {
            if let Err(e) = self.connect_and_handle().await {
                error!("Flashblocks receiver connection error, retrying in 5 seconds: {e}");
                self.metrics.reconnect_attempts.increment(1);
                self.metrics.connection_status.set(0);
                tokio::time::sleep(std::time::Duration::from_millis(self.reconnect_ms)).await;
            } else {
                break;
            }
        }
    }

    async fn connect_and_handle(&self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let (ws_stream, _) = connect_async(self.url.as_str()).await?;
        let (_, mut read) = ws_stream.split();

        info!("Connected to Flashblocks receiver at {}", self.url);
        self.metrics.connection_status.set(1);

        while let Some(msg) = read.next().await {
            if let Message::Text(text) = msg? {
                self.metrics.messages_received.increment(1);
                if let Ok(flashblocks_msg) = serde_json::from_str::<FlashblocksPayloadV1>(&text) {
                    self.sender.send(flashblocks_msg).await?;
                }
            }
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use futures::SinkExt;
    use tokio::sync::watch;
    use tokio_tungstenite::{accept_async, tungstenite::Utf8Bytes};

    use super::*;
    use std::net::{SocketAddr, TcpListener};

    async fn start(
        addr: SocketAddr,
    ) -> eyre::Result<(
        watch::Sender<bool>,
        mpsc::Sender<FlashblocksPayloadV1>,
        url::Url,
    )> {
        let (term_tx, mut term_rx) = watch::channel(false);
        let (send_tx, mut send_rx) = mpsc::channel::<FlashblocksPayloadV1>(100);

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
                                    Ok(mut ws_stream) => {
                                        loop {
                                            tokio::select! {
                                                Some(msg) = send_rx.recv() => {
                                                    let serialized = serde_json::to_string(&msg).unwrap();
                                                    let utf8_bytes = Utf8Bytes::from(serialized);

                                                    ws_stream.send(Message::Text(utf8_bytes)).await.unwrap();
                                                },
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

        Ok((term_tx, send_tx, url))
    }

    #[tokio::test]
    async fn test_flashblocks_receiver_service() -> eyre::Result<()> {
        let addr = "127.0.0.1:8080".parse::<SocketAddr>().unwrap();
        let (term, send_msg, url) = start(addr).await?;

        let (tx, mut rx) = mpsc::channel(100);

        let service = FlashblocksReceiverService::new(url, tx, 100);
        let _ = tokio::spawn(async move {
            service.run().await;
        });

        // Send a message to the websocket server
        send_msg
            .send(FlashblocksPayloadV1::default())
            .await
            .expect("Failed to send message");

        let msg = rx.recv().await.expect("Failed to receive message");
        assert_eq!(msg, FlashblocksPayloadV1::default());

        // Drop the websocket server and start another one with the same address
        // The FlashblocksReceiverService should reconnect to the new server
        term.send(true).unwrap();

        // sleep for 1 second to ensure the server is dropped
        tokio::time::sleep(std::time::Duration::from_secs(1)).await;

        // start a new server with the same address
        let (term, send_msg, _url) = start(addr).await?;
        send_msg
            .send(FlashblocksPayloadV1::default())
            .await
            .expect("Failed to send message");

        let msg = rx.recv().await.expect("Failed to receive message");
        assert_eq!(msg, FlashblocksPayloadV1::default());
        term.send(true).unwrap();

        Ok(())
    }
}
