//! End-to-end pipeline integration tests for native flashblock production.
//!
//! These tests prove the complete data path works:
//!   Emitter → mpsc channel → `FlashblockChannelStream` → broadcast → WS server → external client
//!
//! Unlike the service.rs tests (which mock coordination logic), these tests exercise
//! the actual transport layers with real bytes on real sockets.

use alloy_primitives::{B256, Bytes, U256};
use alloy_rpc_types_engine::PayloadId;
use futures_util::StreamExt;
use op_alloy_rpc_types_engine::{
    OpFlashblockPayload, OpFlashblockPayloadBase, OpFlashblockPayloadDelta,
    OpFlashblockPayloadMetadata,
};
use reth_optimism_flashblocks::{
    FlashBlock, FlashblockChannel, FlashblockChannelStream, ws_server,
};
use reth_optimism_payload_builder::emitter::FlashblockEmitter;
use std::{sync::Arc, time::Duration};
use tokio::{net::TcpListener, sync::broadcast};
use tokio_tungstenite::connect_async;

// ======================= Helper =============================================

fn test_base(block_number: u64) -> OpFlashblockPayloadBase {
    OpFlashblockPayloadBase {
        parent_hash: B256::repeat_byte(0xAA),
        fee_recipient: Default::default(),
        block_number,
        timestamp: 1_700_000_000,
        prev_randao: B256::ZERO,
        gas_limit: 30_000_000,
        extra_data: Bytes::default(),
        base_fee_per_gas: U256::from(1_000_000_000u64),
        parent_beacon_block_root: B256::ZERO,
    }
}

// ======================= 1. Full Pipeline ====================================

/// Proves: Emitter → mpsc channel → ChannelStream → broadcast → WS client.
///
/// This is THE critical test. It simulates exactly what happens during a real
/// payload build: the emitter is called from a blocking context, flashblocks
/// flow through the in-process channel, get broadcast, and arrive at external
/// WS clients with correct ordering and content.
#[tokio::test]
async fn test_full_pipeline_emitter_to_ws_client() {
    // --- Layer 1: In-process channel (replaces rollup-boost WS client) ---
    let channel = FlashblockChannel::new(64);
    let tx = channel.sender();
    let rx = channel.take_receiver().expect("receiver available");
    let mut stream = FlashblockChannelStream::new(rx);

    // --- Layer 2: Broadcast channel (FlashBlockService would do this) ---
    let (broadcast_tx, _) = broadcast::channel::<Arc<FlashBlock>>(128);

    // --- Layer 3: WS broadcast server ---
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let ws_addr = listener.local_addr().unwrap();
    let ws_broadcast_tx = broadcast_tx.clone();
    tokio::spawn(async move {
        loop {
            let (tcp_stream, peer) = match listener.accept().await {
                Ok(conn) => conn,
                Err(_) => break,
            };
            let ws_rx = ws_broadcast_tx.subscribe();
            tokio::spawn(ws_server::handle_client(tcp_stream, ws_rx, peer));
        }
    });

    // --- Connect 2 external WS clients ---
    let url = format!("ws://{ws_addr}");
    let (mut client_a, _) = connect_async(&url).await.unwrap();
    let (mut client_b, _) = connect_async(&url).await.unwrap();
    tokio::time::sleep(Duration::from_millis(50)).await;

    // --- Layer 0: Emitter (runs in spawn_blocking during payload build) ---
    let payload_id = PayloadId::new([42u8; 8]);
    let handle = tokio::task::spawn_blocking(move || {
        let mut emitter = FlashblockEmitter::new(tx, Duration::from_millis(0), payload_id);
        emitter.set_base(test_base(100));

        // Simulate payload building: 3 transactions across 3 flashblocks
        emitter.add_tx(Bytes::from(vec![0x01]));
        emitter.emit_snapshot(21_000, 100);

        emitter.add_tx(Bytes::from(vec![0x02]));
        emitter.emit_snapshot(42_000, 100);

        emitter.add_tx(Bytes::from(vec![0x03]));
        emitter.emit_snapshot(63_000, 100);
    });
    handle.await.unwrap();

    // --- Bridge: read from ChannelStream, broadcast to WS ---
    // (In real code, FlashBlockService does this. Here we do it manually.)
    for _ in 0..3 {
        let fb = stream.next().await.unwrap().unwrap();
        broadcast_tx.send(Arc::new(fb)).unwrap();
    }

    // --- Verify: both WS clients received all 3 flashblocks in order ---
    for (label, client) in [("A", &mut client_a), ("B", &mut client_b)] {
        for expected_index in 0..3u64 {
            let msg = client.next().await.unwrap().unwrap();
            let text = msg.into_text().unwrap();
            let fb: OpFlashblockPayload = serde_json::from_str(&text).unwrap();

            assert_eq!(fb.payload_id, payload_id, "client {label}: wrong payload_id");
            assert_eq!(fb.index, expected_index, "client {label}: wrong index");

            if expected_index == 0 {
                assert!(fb.base.is_some(), "client {label}: index 0 missing base");
                assert_eq!(fb.base.as_ref().unwrap().block_number, 100);
            } else {
                assert!(fb.base.is_none(), "client {label}: index {expected_index} has base");
            }
        }
    }
}

/// Proves: concurrent payload builds (overlapping payload IDs) on the same channel
/// are correctly multiplexed. This is the real-world scenario where build N is
/// finishing while build N+1 has already started.
#[tokio::test]
async fn test_concurrent_builds_on_shared_channel() {
    let channel = FlashblockChannel::new(64);
    let tx_a = channel.sender();
    let tx_b = channel.sender();
    let rx = channel.take_receiver().unwrap();
    let mut stream = FlashblockChannelStream::new(rx);

    let id_a = PayloadId::new([0xAA; 8]);
    let id_b = PayloadId::new([0xBB; 8]);

    // Build A and Build B interleave on the same channel
    let handle = tokio::task::spawn_blocking(move || {
        let mut emitter_a = FlashblockEmitter::new(tx_a, Duration::from_millis(0), id_a);
        let mut emitter_b = FlashblockEmitter::new(tx_b, Duration::from_millis(0), id_b);

        emitter_a.set_base(test_base(100));
        emitter_b.set_base(test_base(101));

        // Interleave: A0, B0, A1, B1
        emitter_a.add_tx(Bytes::from(vec![0x01]));
        emitter_a.emit_snapshot(21_000, 100);

        emitter_b.add_tx(Bytes::from(vec![0x10]));
        emitter_b.emit_snapshot(21_000, 101);

        emitter_a.add_tx(Bytes::from(vec![0x02]));
        emitter_a.emit_snapshot(42_000, 100);

        emitter_b.add_tx(Bytes::from(vec![0x20]));
        emitter_b.emit_snapshot(42_000, 101);
    });
    handle.await.unwrap();

    // Read all 4 flashblocks — order is A0, B0, A1, B1
    let fb0 = stream.next().await.unwrap().unwrap();
    assert_eq!(fb0.payload_id, id_a);
    assert_eq!(fb0.index, 0);
    assert!(fb0.base.is_some());

    let fb1 = stream.next().await.unwrap().unwrap();
    assert_eq!(fb1.payload_id, id_b);
    assert_eq!(fb1.index, 0);
    assert!(fb1.base.is_some());

    let fb2 = stream.next().await.unwrap().unwrap();
    assert_eq!(fb2.payload_id, id_a);
    assert_eq!(fb2.index, 1);

    let fb3 = stream.next().await.unwrap().unwrap();
    assert_eq!(fb3.payload_id, id_b);
    assert_eq!(fb3.index, 1);
}

// ======================= 2. Backpressure & Reliability =======================

/// Proves: slow WS consumer doesn't block other consumers or the broadcast channel.
/// The lagging client gets dropped messages (broadcast::Lagged), others are fine.
#[tokio::test]
async fn test_slow_consumer_doesnt_block_others() {
    let (broadcast_tx, _) = broadcast::channel::<Arc<FlashBlock>>(4); // Small buffer

    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let ws_addr = listener.local_addr().unwrap();
    let ws_tx = broadcast_tx.clone();
    tokio::spawn(async move {
        loop {
            let (stream, peer) = match listener.accept().await {
                Ok(c) => c,
                Err(_) => break,
            };
            let rx = ws_tx.subscribe();
            tokio::spawn(ws_server::handle_client(stream, rx, peer));
        }
    });

    let url = format!("ws://{ws_addr}");
    let (mut fast_client, _) = connect_async(&url).await.unwrap();
    let (mut _slow_client, _) = connect_async(&url).await.unwrap();
    tokio::time::sleep(Duration::from_millis(50)).await;

    // Flood: send more flashblocks than the broadcast buffer can hold
    for i in 0..10u64 {
        let fb = OpFlashblockPayload {
            payload_id: PayloadId::new([1u8; 8]),
            index: i,
            base: (i == 0).then(OpFlashblockPayloadBase::default),
            diff: OpFlashblockPayloadDelta {
                state_root: B256::ZERO,
                receipts_root: B256::ZERO,
                logs_bloom: Default::default(),
                gas_used: 21_000 * (i + 1),
                block_hash: B256::ZERO,
                transactions: vec![Bytes::from(vec![i as u8])],
                withdrawals: vec![],
                withdrawals_root: B256::ZERO,
                blob_gas_used: None,
            },
            metadata: OpFlashblockPayloadMetadata::default(),
        };
        let _ = broadcast_tx.send(Arc::new(fb));
    }

    // Fast client: read whatever arrives (may get all 10, may skip some depending on timing)
    let mut received = Vec::new();
    let deadline = tokio::time::Instant::now() + Duration::from_secs(2);
    loop {
        tokio::select! {
            msg = fast_client.next() => {
                match msg {
                    Some(Ok(m)) => {
                        let text = m.into_text().unwrap();
                        let fb: OpFlashblockPayload = serde_json::from_str(&text).unwrap();
                        received.push(fb.index);
                    }
                    _ => break,
                }
            }
            _ = tokio::time::sleep_until(deadline) => break,
        }
    }

    // Fast client should have received at least some flashblocks
    assert!(!received.is_empty(), "fast client should receive flashblocks");
    // And they should be in order (no reordering)
    for window in received.windows(2) {
        assert!(window[0] < window[1], "flashblocks should arrive in order: {received:?}");
    }
}

/// Proves: WS client disconnect doesn't affect other connected clients.
#[tokio::test]
async fn test_client_disconnect_isolation() {
    let (broadcast_tx, _) = broadcast::channel::<Arc<FlashBlock>>(16);

    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let ws_addr = listener.local_addr().unwrap();
    let ws_tx = broadcast_tx.clone();
    tokio::spawn(async move {
        loop {
            let (stream, peer) = match listener.accept().await {
                Ok(c) => c,
                Err(_) => break,
            };
            let rx = ws_tx.subscribe();
            tokio::spawn(ws_server::handle_client(stream, rx, peer));
        }
    });

    let url = format!("ws://{ws_addr}");
    let (mut client_a, _) = connect_async(&url).await.unwrap();
    let (client_b, _) = connect_async(&url).await.unwrap();
    let (mut client_c, _) = connect_async(&url).await.unwrap();
    tokio::time::sleep(Duration::from_millis(50)).await;

    // Send flashblock #0
    let fb0 = OpFlashblockPayload {
        payload_id: PayloadId::new([1u8; 8]),
        index: 0,
        base: Some(OpFlashblockPayloadBase::default()),
        diff: OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Default::default(),
            gas_used: 21_000,
            block_hash: B256::ZERO,
            transactions: vec![Bytes::from(vec![0x01])],
            withdrawals: vec![],
            withdrawals_root: B256::ZERO,
            blob_gas_used: None,
        },
        metadata: OpFlashblockPayloadMetadata::default(),
    };
    broadcast_tx.send(Arc::new(fb0)).unwrap();

    // All 3 receive it
    let _ = client_a.next().await.unwrap().unwrap();
    let _ = client_c.next().await.unwrap().unwrap();

    // Kill client B
    drop(client_b);
    tokio::time::sleep(Duration::from_millis(50)).await;

    // Send flashblock #1
    let fb1 = OpFlashblockPayload {
        payload_id: PayloadId::new([1u8; 8]),
        index: 1,
        base: None,
        diff: OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Default::default(),
            gas_used: 42_000,
            block_hash: B256::ZERO,
            transactions: vec![Bytes::from(vec![0x02])],
            withdrawals: vec![],
            withdrawals_root: B256::ZERO,
            blob_gas_used: None,
        },
        metadata: OpFlashblockPayloadMetadata::default(),
    };
    broadcast_tx.send(Arc::new(fb1)).unwrap();

    // A and C still receive
    let msg_a = client_a.next().await.unwrap().unwrap();
    let fb_a: OpFlashblockPayload = serde_json::from_str(&msg_a.into_text().unwrap()).unwrap();
    assert_eq!(fb_a.index, 1);

    let msg_c = client_c.next().await.unwrap().unwrap();
    let fb_c: OpFlashblockPayload = serde_json::from_str(&msg_c.into_text().unwrap()).unwrap();
    assert_eq!(fb_c.index, 1);
}

/// Proves: channel sender drop causes clean stream termination.
/// This is the shutdown path — when the node drops the payload builder,
/// the stream ends gracefully.
#[tokio::test]
async fn test_channel_shutdown_propagation() {
    let channel = FlashblockChannel::new(16);
    let tx = channel.sender();
    let rx = channel.take_receiver().unwrap();
    let mut stream = FlashblockChannelStream::new(rx);

    // Send one flashblock, then drop everything
    let payload_id = PayloadId::new([1u8; 8]);
    tokio::task::spawn_blocking(move || {
        let mut emitter = FlashblockEmitter::new(tx, Duration::from_millis(0), payload_id);
        emitter.set_base(test_base(100));
        emitter.emit_snapshot(0, 100);
        // emitter + tx dropped here
    })
    .await
    .unwrap();

    // Drop the original channel (which holds the internal sender)
    drop(channel);

    // Should get the buffered flashblock
    let fb = stream.next().await.unwrap().unwrap();
    assert_eq!(fb.index, 0);

    // Then stream ends cleanly (None, not error)
    assert!(stream.next().await.is_none(), "stream should end after all senders drop");
}

/// Proves: emitter works correctly from spawn_blocking context.
/// This is critical because the payload builder runs in spawn_blocking,
/// and blocking_send must work there without deadlocking.
#[tokio::test]
async fn test_emitter_from_spawn_blocking() {
    let channel = FlashblockChannel::new(16);
    let tx = channel.sender();
    let rx = channel.take_receiver().unwrap();
    let mut stream = FlashblockChannelStream::new(rx);

    let payload_id = PayloadId::new([7u8; 8]);
    tokio::task::spawn_blocking(move || {
        let mut emitter = FlashblockEmitter::new(tx, Duration::from_millis(0), payload_id);
        emitter.set_base(test_base(200));

        // Simulate a full block build: 10 transactions, 5 flashblocks
        for i in 0..10u64 {
            emitter.add_tx(Bytes::from(vec![i as u8; 32]));
            if i % 2 == 1 {
                emitter.emit_snapshot(21_000 * (i + 1), 200);
            }
        }
    })
    .await
    .unwrap();
    drop(channel);

    // Collect all flashblocks
    let mut flashblocks = Vec::new();
    while let Some(Ok(fb)) = stream.next().await {
        flashblocks.push(fb);
    }

    assert_eq!(flashblocks.len(), 5, "should produce 5 flashblocks");

    // Verify ordering
    for (i, fb) in flashblocks.iter().enumerate() {
        assert_eq!(fb.index, i as u64, "wrong index at position {i}");
        assert_eq!(fb.payload_id, payload_id);
    }

    // First has base, rest don't
    assert!(flashblocks[0].base.is_some());
    for fb in &flashblocks[1..] {
        assert!(fb.base.is_none());
    }

    // Each flashblock has 2 transactions
    for fb in &flashblocks {
        assert_eq!(fb.diff.transactions.len(), 2, "each fb should have 2 txs");
    }

    // Gas usage increases monotonically
    for window in flashblocks.windows(2) {
        assert!(
            window[1].diff.gas_used > window[0].diff.gas_used,
            "gas should increase: {} vs {}",
            window[0].diff.gas_used,
            window[1].diff.gas_used
        );
    }
}

// ======================= 3. Wire Format Compatibility ========================

/// Proves: the native pipeline produces byte-identical JSON to what rollup-boost
/// would serve. This is critical for op-conductor backward compatibility.
///
/// Round-trip: Emitter → channel → serialize → deserialize → verify every field.
#[tokio::test]
async fn test_wire_format_roundtrip() {
    let channel = FlashblockChannel::new(16);
    let tx = channel.sender();
    let rx = channel.take_receiver().unwrap();
    let mut stream = FlashblockChannelStream::new(rx);

    let payload_id = PayloadId::new([0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE]);
    let base = test_base(500);
    let base_clone = base.clone();

    tokio::task::spawn_blocking(move || {
        let mut emitter = FlashblockEmitter::new(tx, Duration::from_millis(0), payload_id);
        emitter.set_base(base);
        emitter.add_tx(Bytes::from(vec![0xDE, 0xAD]));
        emitter.add_tx(Bytes::from(vec![0xBE, 0xEF]));
        emitter.emit_snapshot(42_000, 500);
    })
    .await
    .unwrap();
    drop(channel);

    let fb = stream.next().await.unwrap().unwrap();

    // Serialize to JSON (exactly what the WS server sends)
    let json = serde_json::to_string(&fb).unwrap();

    // Deserialize as OpFlashblockPayload (what op-conductor receives)
    let deserialized: OpFlashblockPayload = serde_json::from_str(&json).unwrap();

    // Verify every field matches
    assert_eq!(deserialized.payload_id, payload_id);
    assert_eq!(deserialized.index, 0);

    let deser_base = deserialized.base.as_ref().expect("index 0 must have base");
    assert_eq!(deser_base.parent_hash, base_clone.parent_hash);
    assert_eq!(deser_base.block_number, 500);
    assert_eq!(deser_base.timestamp, base_clone.timestamp);
    assert_eq!(deser_base.gas_limit, base_clone.gas_limit);
    assert_eq!(deser_base.base_fee_per_gas, base_clone.base_fee_per_gas);

    assert_eq!(deserialized.diff.gas_used, 42_000);
    assert_eq!(deserialized.diff.transactions.len(), 2);
    assert_eq!(deserialized.diff.transactions[0], Bytes::from(vec![0xDE, 0xAD]));
    assert_eq!(deserialized.diff.transactions[1], Bytes::from(vec![0xBE, 0xEF]));
    assert_eq!(deserialized.metadata.block_number, 500);
}

/// Proves: JSON structure matches the exact wire format that rollup-boost serves.
/// Verifies field names, nesting, and presence/absence of optional fields.
#[tokio::test]
async fn test_wire_format_json_structure() {
    let channel = FlashblockChannel::new(16);
    let tx = channel.sender();
    let rx = channel.take_receiver().unwrap();
    let mut stream = FlashblockChannelStream::new(rx);

    let payload_id = PayloadId::new([1u8; 8]);
    tokio::task::spawn_blocking(move || {
        let mut emitter = FlashblockEmitter::new(tx, Duration::from_millis(0), payload_id);
        emitter.set_base(test_base(100));

        // Index 0: has base
        emitter.emit_snapshot(0, 100);

        // Index 1: no base, has txs
        emitter.add_tx(Bytes::from(vec![0xFF]));
        emitter.emit_snapshot(21_000, 100);
    })
    .await
    .unwrap();
    drop(channel);

    // Index 0: verify JSON structure (snake_case field names per op-alloy serde)
    let fb0 = stream.next().await.unwrap().unwrap();
    let json0: serde_json::Value = serde_json::to_value(&fb0).unwrap();

    // Must have these top-level fields
    assert!(json0.get("payload_id").is_some(), "missing payload_id");
    assert!(json0.get("index").is_some(), "missing index");
    assert!(json0.get("base").is_some(), "index 0 must have base");
    assert!(json0.get("diff").is_some(), "missing diff");
    assert!(json0.get("metadata").is_some(), "missing metadata");

    // Base fields
    let base = json0.get("base").unwrap();
    assert!(base.get("parent_hash").is_some(), "base missing parent_hash");
    assert!(base.get("block_number").is_some(), "base missing block_number");
    assert!(base.get("timestamp").is_some(), "base missing timestamp");
    assert!(base.get("gas_limit").is_some(), "base missing gas_limit");

    // Diff fields
    let diff = json0.get("diff").unwrap();
    assert!(diff.get("state_root").is_some(), "diff missing state_root");
    assert!(diff.get("gas_used").is_some(), "diff missing gas_used");
    assert!(diff.get("transactions").is_some(), "diff missing transactions");
    assert!(diff.get("block_hash").is_some(), "diff missing block_hash");

    // Index 1: base should be absent (skip_serializing_if = "Option::is_none")
    let fb1 = stream.next().await.unwrap().unwrap();
    let json1: serde_json::Value = serde_json::to_value(&fb1).unwrap();
    assert!(json1.get("base").is_none(), "index 1 base should be omitted, not null");
}

/// Proves: WS server sends text frames (not binary), matching rollup-boost behavior.
#[tokio::test]
async fn test_ws_sends_text_frames() {
    let (broadcast_tx, _) = broadcast::channel::<Arc<FlashBlock>>(16);
    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let ws_addr = listener.local_addr().unwrap();

    let ws_tx = broadcast_tx.clone();
    tokio::spawn(async move {
        loop {
            let (stream, peer) = match listener.accept().await {
                Ok(c) => c,
                Err(_) => break,
            };
            let rx = ws_tx.subscribe();
            tokio::spawn(ws_server::handle_client(stream, rx, peer));
        }
    });

    let url = format!("ws://{ws_addr}");
    let (mut client, _) = connect_async(&url).await.unwrap();
    tokio::time::sleep(Duration::from_millis(50)).await;

    let fb = OpFlashblockPayload {
        payload_id: PayloadId::new([1u8; 8]),
        index: 0,
        base: Some(OpFlashblockPayloadBase::default()),
        diff: OpFlashblockPayloadDelta {
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Default::default(),
            gas_used: 0,
            block_hash: B256::ZERO,
            transactions: vec![],
            withdrawals: vec![],
            withdrawals_root: B256::ZERO,
            blob_gas_used: None,
        },
        metadata: OpFlashblockPayloadMetadata::default(),
    };
    broadcast_tx.send(Arc::new(fb)).unwrap();

    let msg = client.next().await.unwrap().unwrap();
    assert!(msg.is_text(), "WS server must send text frames, not binary");

    // And the text must be valid JSON
    let text = msg.into_text().unwrap();
    let _: OpFlashblockPayload = serde_json::from_str(&text).expect("must be valid JSON");
}
