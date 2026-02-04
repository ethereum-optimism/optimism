//! Native implementation of the [Channel] trait, backed by [`async_channel`]'s unbounded
//! channel primitives.

use crate::{
    Channel,
    errors::{ChannelError, ChannelResult},
};
use async_channel::{Receiver, Sender, unbounded};
use async_trait::async_trait;
use std::io::Result;

/// A bidirectional channel, allowing for synchronized communication between two parties.
#[derive(Debug, Clone)]
pub struct BidirectionalChannel {
    /// The client handle of the channel.
    pub client: NativeChannel,
    /// The host handle of the channel.
    pub host: NativeChannel,
}

impl BidirectionalChannel {
    /// Creates a [`BidirectionalChannel`] instance.
    pub fn new() -> Result<Self> {
        let (bw, ar) = unbounded();
        let (aw, br) = unbounded();

        Ok(Self {
            client: NativeChannel { read: ar, write: aw },
            host: NativeChannel { read: br, write: bw },
        })
    }
}

/// A channel with a receiver and sender.
#[derive(Debug, Clone)]
pub struct NativeChannel {
    /// The receiver of the channel.
    pub(crate) read: Receiver<Vec<u8>>,
    /// The sender of the channel.
    pub(crate) write: Sender<Vec<u8>>,
}

#[async_trait]
impl Channel for NativeChannel {
    async fn read(&self, buf: &mut [u8]) -> ChannelResult<usize> {
        let data = self.read.recv().await.map_err(|_| ChannelError::Closed)?;
        let len = data.len().min(buf.len());
        buf[..len].copy_from_slice(&data[..len]);
        Ok(len)
    }

    async fn read_exact(&self, buf: &mut [u8]) -> ChannelResult<usize> {
        let data = self.read.recv().await.map_err(|_| ChannelError::Closed)?;
        buf[..].copy_from_slice(&data[..]);
        Ok(buf.len())
    }

    async fn write(&self, buf: &[u8]) -> ChannelResult<usize> {
        self.write.send(buf.to_vec()).await.map_err(|_| ChannelError::Closed)?;
        Ok(buf.len())
    }
}
