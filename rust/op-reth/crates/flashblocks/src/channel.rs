//! In-process flashblock channel transport.
//!
//! Provides [`FlashblockChannel`] for sharing a channel between the flashblock
//! producer (payload builder) and consumer ([`FlashBlockService`](crate::FlashBlockService)),
//! and [`FlashblockChannelStream`] which adapts the receiving end into the
//! `Stream<Item = eyre::Result<FlashBlock>>` interface that `FlashBlockService` expects.

use crate::FlashBlock;
use futures_util::Stream;
use op_alloy_rpc_types_engine::OpFlashblockPayload;
use std::{
    fmt,
    pin::Pin,
    sync::{Arc, Mutex},
    task::{Context, Poll},
};
use tokio::sync::mpsc;

/// Shared flashblock channel connecting the payload builder (producer)
/// to [`FlashBlockService`](crate::FlashBlockService) (consumer).
///
/// The sender is clone-able (one clone per payload build). The receiver
/// can only be taken once via [`take_receiver`](Self::take_receiver) —
/// it is consumed when building the [`FlashblockChannelStream`].
#[derive(Clone)]
pub struct FlashblockChannel {
    tx: mpsc::Sender<OpFlashblockPayload>,
    rx: Arc<Mutex<Option<mpsc::Receiver<OpFlashblockPayload>>>>,
}

impl FlashblockChannel {
    /// Creates a new channel with the given buffer capacity.
    ///
    /// Capacity 64 is recommended: enough to buffer ~3 concurrent builds
    /// × ~10 flashblocks per build + headroom.
    pub fn new(capacity: usize) -> Self {
        let (tx, rx) = mpsc::channel(capacity);
        Self { tx, rx: Arc::new(Mutex::new(Some(rx))) }
    }

    /// Returns a clone of the sender for use by the payload builder.
    pub fn sender(&self) -> mpsc::Sender<OpFlashblockPayload> {
        self.tx.clone()
    }

    /// Takes the receiver. Returns `None` on subsequent calls.
    pub fn take_receiver(&self) -> Option<mpsc::Receiver<OpFlashblockPayload>> {
        self.rx.lock().unwrap().take()
    }
}

impl fmt::Debug for FlashblockChannel {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("FlashblockChannel")
            .field("tx", &self.tx)
            .field("rx_taken", &self.rx.lock().unwrap().is_none())
            .finish()
    }
}

/// Wraps a `tokio::sync::mpsc::Receiver` as a `Stream<Item = eyre::Result<FlashBlock>>`
/// for consumption by [`FlashBlockService`](crate::FlashBlockService).
///
/// Unlike the `WebSocket` stream this replaces, this stream is never expected to
/// end during normal operation — sender drop means node shutdown.
pub struct FlashblockChannelStream {
    rx: mpsc::Receiver<OpFlashblockPayload>,
}

impl FlashblockChannelStream {
    /// Creates a new stream wrapping the channel receiver.
    pub const fn new(rx: mpsc::Receiver<OpFlashblockPayload>) -> Self {
        Self { rx }
    }
}

impl Stream for FlashblockChannelStream {
    type Item = eyre::Result<FlashBlock>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        match self.rx.poll_recv(cx) {
            Poll::Ready(Some(payload)) => Poll::Ready(Some(Ok(payload))),
            Poll::Ready(None) => Poll::Ready(None),
            Poll::Pending => Poll::Pending,
        }
    }
}

impl fmt::Debug for FlashblockChannelStream {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("FlashblockChannelStream").finish()
    }
}

// SAFETY: FlashblockChannelStream contains only a tokio mpsc::Receiver which is Unpin.
impl Unpin for FlashblockChannelStream {}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{B256, Bytes};
    use futures_util::StreamExt;
    use op_alloy_rpc_types_engine::{
        OpFlashblockPayload, OpFlashblockPayloadBase, OpFlashblockPayloadDelta,
        OpFlashblockPayloadMetadata,
    };

    fn test_payload(index: u64) -> OpFlashblockPayload {
        OpFlashblockPayload {
            payload_id: alloy_rpc_types_engine::PayloadId::new([1u8; 8]),
            index,
            base: (index == 0).then(OpFlashblockPayloadBase::default),
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
    async fn test_native_channel_integration() {
        let channel = FlashblockChannel::new(16);
        let tx = channel.sender();
        let rx = channel.take_receiver().expect("receiver should be available");
        let mut stream = FlashblockChannelStream::new(rx);

        tx.send(test_payload(0)).await.unwrap();
        tx.send(test_payload(1)).await.unwrap();

        let fb0 = stream.next().await.unwrap().unwrap();
        assert_eq!(fb0.index, 0);
        assert!(fb0.base.is_some());

        let fb1 = stream.next().await.unwrap().unwrap();
        assert_eq!(fb1.index, 1);
        assert!(fb1.base.is_none());
    }

    #[tokio::test]
    async fn test_channel_persists_across_sender_drops() {
        let channel = FlashblockChannel::new(16);
        let tx1 = channel.sender();
        let tx2 = channel.sender();
        let rx = channel.take_receiver().expect("receiver should be available");
        let mut stream = FlashblockChannelStream::new(rx);

        // Send from first clone
        tx1.send(test_payload(0)).await.unwrap();
        // Drop first clone
        drop(tx1);

        // Original sender still works
        tx2.send(test_payload(1)).await.unwrap();

        let fb0 = stream.next().await.unwrap().unwrap();
        assert_eq!(fb0.index, 0);
        let fb1 = stream.next().await.unwrap().unwrap();
        assert_eq!(fb1.index, 1);
    }

    #[test]
    fn test_receiver_can_only_be_taken_once() {
        let channel = FlashblockChannel::new(16);
        assert!(channel.take_receiver().is_some());
        assert!(channel.take_receiver().is_none());
    }

    #[tokio::test]
    async fn test_stream_ends_when_all_senders_dropped() {
        let channel = FlashblockChannel::new(16);
        let tx = channel.sender();
        let rx = channel.take_receiver().expect("receiver should be available");
        let mut stream = FlashblockChannelStream::new(rx);

        tx.send(test_payload(0)).await.unwrap();
        drop(tx);
        // Drop the internal sender too
        drop(channel);

        // Should get the buffered message
        let fb = stream.next().await.unwrap().unwrap();
        assert_eq!(fb.index, 0);

        // Stream should end (None)
        assert!(stream.next().await.is_none());
    }
}
