use super::primitives::FlashblocksPayloadV1;
use futures::StreamExt;
use tokio::sync::mpsc;
use tokio_tungstenite::{connect_async, tungstenite::Message};
use tracing::{error, info};
use url::Url;

pub struct FlashblocksReceiverService {
    url: Url,
    sender: mpsc::Sender<FlashblocksPayloadV1>,
    reconnect_ms: u64,
}

impl FlashblocksReceiverService {
    pub fn new(url: Url, sender: mpsc::Sender<FlashblocksPayloadV1>, reconnect_ms: u64) -> Self {
        Self {
            url,
            sender,
            reconnect_ms,
        }
    }

    pub async fn run(self) {
        loop {
            if let Err(e) = self.connect_and_handle().await {
                error!("Flashblocks receiver connection error, retrying in 5 seconds: {e}");
                tokio::time::sleep(std::time::Duration::from_secs(self.reconnect_ms)).await;
            } else {
                break;
            }
        }
    }

    async fn connect_and_handle(&self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let (ws_stream, _) = connect_async(self.url.as_str()).await?;
        let (_, mut read) = ws_stream.split();

        info!("Connected to Flashblocks receiver at {}", self.url);

        while let Some(msg) = read.next().await {
            if let Message::Text(text) = msg? {
                if let Ok(flashblocks_msg) = serde_json::from_str::<FlashblocksPayloadV1>(&text) {
                    self.sender.send(flashblocks_msg).await?;
                }
            }
        }

        Ok(())
    }
}
