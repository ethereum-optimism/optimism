use crate::client::ClientConnection;
use crate::metrics::Metrics;
use std::sync::Arc;
use tokio::sync::broadcast::error::RecvError;
use tokio::sync::broadcast::Sender;
use tracing::{info, trace, warn};

#[derive(Clone)]
pub struct Registry {
    sender: Sender<String>,
    metrics: Arc<Metrics>,
}

impl Registry {
    pub fn new(sender: Sender<String>, metrics: Arc<Metrics>) -> Self {
        Self { sender, metrics }
    }

    pub async fn subscribe(&self, mut client: ClientConnection) {
        info!(message = "subscribing client", client = client.id());

        let mut receiver = self.sender.subscribe();
        let metrics = self.metrics.clone();
        metrics.new_connections.increment(1);

        tokio::spawn(async move {
            loop {
                match receiver.recv().await {
                    Ok(msg) => match client.send(msg.clone()).await {
                        Ok(_) => {
                            trace!(message = "message sent to client", client = client.id());
                            metrics.sent_messages.increment(1);
                        }
                        Err(e) => {
                            warn!(
                                message = "failed to send data to client",
                                client = client.id(),
                                error = e.to_string()
                            );
                            metrics.failed_messages.increment(1);
                            break;
                        }
                    },
                    Err(RecvError::Closed) => {
                        info!(message = "upstream connection closed", client = client.id());
                        break;
                    }
                    Err(RecvError::Lagged(_)) => {
                        info!(message = "client is lagging", client = client.id());
                        metrics.lagged_connections.increment(1);
                        break;
                    }
                }
            }

            metrics.closed_connections.increment(1);
            info!(message = "client disconnected", client = client.id());
        });
    }
}
