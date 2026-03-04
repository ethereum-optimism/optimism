//! Raw `WebSocket` broadcast server for flashblocks.
//!
//! Broadcasts flashblocks to external subscribers (op-conductor, RPC providers)
//! over a plain `WebSocket` connection. Messages are JSON-encoded
//! [`FlashBlock`](crate::FlashBlock) payloads — matching the wire format that
//! rollup-boost historically served.

use crate::FlashBlock;
use futures_util::{SinkExt, StreamExt};
use std::{net::SocketAddr, sync::Arc};
use tokio::{net::TcpListener, sync::broadcast};
use tokio_tungstenite::tungstenite::Message;
use tracing::{debug, info, warn};

/// Starts a raw `WebSocket` server that broadcasts flashblocks to all connected clients.
///
/// Each connecting client receives JSON-encoded [`FlashBlock`] messages from the
/// broadcast channel. This preserves wire compatibility with op-conductor, which
/// connects to a generic WS endpoint and passes messages through.
///
/// The server runs until the broadcast channel's sender is dropped (node shutdown).
pub async fn serve(addr: SocketAddr, flashblock_tx: broadcast::Sender<Arc<FlashBlock>>) {
    let listener = match TcpListener::bind(addr).await {
        Ok(l) => l,
        Err(e) => {
            warn!(target: "flashblocks::ws_server", %addr, %e, "failed to bind flashblock WS server");
            return;
        }
    };

    info!(target: "flashblocks::ws_server", %addr, "flashblock WS server listening");

    loop {
        let (stream, peer) = match listener.accept().await {
            Ok(conn) => conn,
            Err(e) => {
                warn!(target: "flashblocks::ws_server", %e, "failed to accept connection");
                continue;
            }
        };

        let rx = flashblock_tx.subscribe();
        tokio::spawn(handle_client(stream, rx, peer));
    }
}

async fn handle_client(
    stream: tokio::net::TcpStream,
    mut rx: broadcast::Receiver<Arc<FlashBlock>>,
    peer: SocketAddr,
) {
    let ws = match tokio_tungstenite::accept_async(stream).await {
        Ok(ws) => ws,
        Err(e) => {
            debug!(target: "flashblocks::ws_server", %peer, %e, "WS handshake failed");
            return;
        }
    };

    debug!(target: "flashblocks::ws_server", %peer, "client connected");
    let (mut sink, mut incoming) = ws.split();

    loop {
        tokio::select! {
            // Forward flashblocks to the client
            result = rx.recv() => {
                match result {
                    Ok(fb) => {
                        let json = match serde_json::to_string(&*fb) {
                            Ok(j) => j,
                            Err(e) => {
                                warn!(target: "flashblocks::ws_server", %peer, %e, "JSON serialization failed");
                                continue;
                            }
                        };
                        if sink.send(Message::Text(json.into())).await.is_err() {
                            break; // Client disconnected
                        }
                    }
                    Err(broadcast::error::RecvError::Lagged(n)) => {
                        debug!(target: "flashblocks::ws_server", %peer, skipped = n, "client lagged, skipping");
                    }
                    Err(broadcast::error::RecvError::Closed) => {
                        break; // Sender dropped = shutdown
                    }
                }
            }
            // Handle incoming messages (ping/pong, close)
            msg = incoming.next() => {
                match msg {
                    Some(Ok(Message::Ping(data))) => {
                        let _ = sink.send(Message::Pong(data)).await;
                    }
                    Some(Ok(Message::Close(_)) | Err(_)) | None => break,
                    _ => {} // Ignore other messages
                }
            }
        }
    }

    debug!(target: "flashblocks::ws_server", %peer, "client disconnected");
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{B256, Bytes};
    use op_alloy_rpc_types_engine::{
        OpFlashblockPayload, OpFlashblockPayloadBase, OpFlashblockPayloadDelta,
        OpFlashblockPayloadMetadata,
    };
    use tokio_tungstenite::connect_async;

    fn test_flashblock(index: u64) -> FlashBlock {
        OpFlashblockPayload {
            payload_id: alloy_rpc_types_engine::PayloadId::new([1u8; 8]),
            index,
            base: if index == 0 {
                Some(OpFlashblockPayloadBase::default())
            } else {
                None
            },
            diff: OpFlashblockPayloadDelta {
                state_root: B256::ZERO,
                receipts_root: B256::ZERO,
                logs_bloom: Default::default(),
                gas_used: 21000 * (index + 1),
                block_hash: B256::ZERO,
                transactions: vec![Bytes::from(vec![index as u8])],
                withdrawals: vec![],
                withdrawals_root: B256::ZERO,
                blob_gas_used: None,
            },
            metadata: OpFlashblockPayloadMetadata::default(),
        }
    }

    #[tokio::test]
    async fn test_ws_server_broadcast() {
        let (tx, _) = broadcast::channel::<Arc<FlashBlock>>(16);
        let addr: SocketAddr = "127.0.0.1:0".parse().unwrap();
        let listener = TcpListener::bind(addr).await.unwrap();
        let bound_addr = listener.local_addr().unwrap();

        // Spawn server manually using the bound listener
        let tx_clone = tx.clone();
        tokio::spawn(async move {
            loop {
                let (stream, peer) = match listener.accept().await {
                    Ok(conn) => conn,
                    Err(_) => break,
                };
                let rx = tx_clone.subscribe();
                tokio::spawn(handle_client(stream, rx, peer));
            }
        });

        // Connect 3 clients
        let url = format!("ws://{bound_addr}");
        let (mut c1, _) = connect_async(&url).await.unwrap();
        let (mut c2, _) = connect_async(&url).await.unwrap();
        let (mut c3, _) = connect_async(&url).await.unwrap();

        // Give connections time to establish
        tokio::time::sleep(std::time::Duration::from_millis(50)).await;

        // Broadcast 5 flashblocks
        for i in 0..5u64 {
            tx.send(Arc::new(test_flashblock(i))).unwrap();
        }

        // Each client should receive all 5
        for client in [&mut c1, &mut c2, &mut c3] {
            for i in 0..5u64 {
                let msg = client.next().await.unwrap().unwrap();
                let text = msg.into_text().unwrap();
                let fb: FlashBlock = serde_json::from_str(&text).unwrap();
                assert_eq!(fb.index, i);
            }
        }
    }

    #[tokio::test]
    async fn test_ws_server_client_disconnect() {
        let (tx, _) = broadcast::channel::<Arc<FlashBlock>>(16);
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let bound_addr = listener.local_addr().unwrap();

        let tx_clone = tx.clone();
        tokio::spawn(async move {
            loop {
                let (stream, peer) = match listener.accept().await {
                    Ok(conn) => conn,
                    Err(_) => break,
                };
                let rx = tx_clone.subscribe();
                tokio::spawn(handle_client(stream, rx, peer));
            }
        });

        let url = format!("ws://{bound_addr}");
        let (mut c1, _) = connect_async(&url).await.unwrap();
        let (mut c2, _) = connect_async(&url).await.unwrap();

        tokio::time::sleep(std::time::Duration::from_millis(50)).await;

        // Send one flashblock
        tx.send(Arc::new(test_flashblock(0))).unwrap();

        // Both receive it
        let _ = c1.next().await.unwrap().unwrap();
        let _ = c2.next().await.unwrap().unwrap();

        // Disconnect c1
        drop(c1);
        tokio::time::sleep(std::time::Duration::from_millis(50)).await;

        // Send another — c2 should still receive
        tx.send(Arc::new(test_flashblock(1))).unwrap();
        let msg = c2.next().await.unwrap().unwrap();
        let fb: FlashBlock = serde_json::from_str(&msg.into_text().unwrap()).unwrap();
        assert_eq!(fb.index, 1);
    }

    #[tokio::test]
    async fn test_ws_server_wire_format() {
        let (tx, _) = broadcast::channel::<Arc<FlashBlock>>(16);
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let bound_addr = listener.local_addr().unwrap();

        let tx_clone = tx.clone();
        tokio::spawn(async move {
            loop {
                let (stream, peer) = match listener.accept().await {
                    Ok(conn) => conn,
                    Err(_) => break,
                };
                let rx = tx_clone.subscribe();
                tokio::spawn(handle_client(stream, rx, peer));
            }
        });

        let url = format!("ws://{bound_addr}");
        let (mut client, _) = connect_async(&url).await.unwrap();
        tokio::time::sleep(std::time::Duration::from_millis(50)).await;

        let original = test_flashblock(0);
        tx.send(Arc::new(original.clone())).unwrap();

        let msg = client.next().await.unwrap().unwrap();
        let text = msg.into_text().unwrap();

        // Verify it round-trips as OpFlashblockPayload
        let deserialized: OpFlashblockPayload = serde_json::from_str(&text).unwrap();
        assert_eq!(deserialized.index, original.index);
        assert_eq!(deserialized.diff.gas_used, original.diff.gas_used);
        assert_eq!(deserialized.diff.transactions.len(), original.diff.transactions.len());
    }
}
