use crate::{FlashblocksApi, cache::FlashblocksCache};
use alloy_primitives::{Address, TxHash, U256};
use futures_util::StreamExt;
use jsonrpsee::core::async_trait;
use op_alloy_network::Optimism;
use reth_optimism_chainspec::OpChainSpec;
use reth_rpc_eth_api::{RpcBlock, RpcReceipt};
use rollup_boost::FlashblocksPayloadV1;
use std::{io::Read, sync::Arc};
use tokio::sync::mpsc;
use tokio_tungstenite::{connect_async, tungstenite::Message};
use tracing::{debug, error, info};
use url::Url;

#[derive(Clone)]
pub struct FlashblocksOverlay {
    url: Url,
    cache: FlashblocksCache,
}

impl FlashblocksOverlay {
    pub fn new(url: Url, chain_spec: Arc<OpChainSpec>) -> Self {
        Self {
            url,
            cache: FlashblocksCache::new(chain_spec),
        }
    }

    pub fn start(&mut self) -> eyre::Result<()> {
        let url = self.url.clone();
        let (sender, mut receiver) = mpsc::channel(100);

        tokio::spawn(async move {
            let mut backoff = std::time::Duration::from_secs(1);
            const MAX_BACKOFF: std::time::Duration = std::time::Duration::from_secs(10);

            loop {
                match connect_async(url.as_str()).await {
                    Ok((ws_stream, _)) => {
                        info!("WebSocket connection established");
                        let (_write, mut read) = ws_stream.split();

                        while let Some(msg) = read.next().await {
                            debug!("Received message: {:?}", msg);

                            match msg {
                                Ok(Message::Binary(bytes)) => match try_decode_message(&bytes) {
                                    Ok(payload) => {
                                        info!("Received payload: {:?}", payload);

                                        let _ = sender
                                            .send(InternalMessage::NewPayload(payload))
                                            .await
                                            .map_err(|e| {
                                                error!("failed to send payload to channel: {}", e);
                                            });
                                    }
                                    Err(e) => {
                                        error!("failed to parse fb message: {}", e);
                                    }
                                },
                                Ok(Message::Close(e)) => {
                                    error!("WebSocket connection closed: {:?}", e);
                                    break;
                                }
                                Err(e) => {
                                    error!("WebSocket connection error: {}", e);
                                    break;
                                }
                                _ => {}
                            }
                        }
                    }
                    Err(e) => {
                        error!(
                            "WebSocket connection error, retrying in {:?}: {}",
                            backoff, e
                        );
                        tokio::time::sleep(backoff).await;
                        // Double the backoff time, but cap at MAX_BACKOFF
                        backoff = std::cmp::min(backoff * 2, MAX_BACKOFF);
                        continue;
                    }
                }
            }
        });

        let cache_cloned = self.cache.clone();
        tokio::spawn(async move {
            while let Some(message) = receiver.recv().await {
                match message {
                    InternalMessage::NewPayload(payload) => {
                        if let Err(e) = cache_cloned.process_payload(payload) {
                            error!("failed to process payload: {}", e);
                        }
                    }
                }
            }
        });

        Ok(())
    }

    pub fn process_payload(&self, payload: FlashblocksPayloadV1) -> eyre::Result<()> {
        self.cache.process_payload(payload)
    }
}

enum InternalMessage {
    NewPayload(FlashblocksPayloadV1),
}

fn try_decode_message(bytes: &[u8]) -> eyre::Result<FlashblocksPayloadV1> {
    let text = try_parse_message(bytes)?;

    let payload: FlashblocksPayloadV1 = match serde_json::from_str(&text) {
        Ok(m) => m,
        Err(e) => {
            return Err(eyre::eyre!("failed to parse message: {}", e));
        }
    };

    Ok(payload)
}

fn try_parse_message(bytes: &[u8]) -> eyre::Result<String> {
    if let Ok(text) = String::from_utf8(bytes.to_vec()) {
        if text.trim_start().starts_with("{") {
            return Ok(text);
        }
    }

    let mut decompressor = brotli::Decompressor::new(bytes, 4096);
    let mut decompressed = Vec::new();
    decompressor.read_to_end(&mut decompressed)?;

    let text = String::from_utf8(decompressed)?;
    Ok(text)
}

#[async_trait]
impl FlashblocksApi for FlashblocksOverlay {
    async fn block_by_number(&self, full: bool) -> Option<RpcBlock<Optimism>> {
        self.cache.get_block(full)
    }

    async fn get_transaction_receipt(&self, tx_hash: TxHash) -> Option<RpcReceipt<Optimism>> {
        self.cache.get_receipt(&tx_hash)
    }

    async fn get_balance(&self, address: Address) -> Option<U256> {
        self.cache.get_balance(address)
    }

    async fn get_transaction_count(&self, address: Address) -> Option<u64> {
        self.cache.get_transaction_count(address)
    }
}
