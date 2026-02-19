//! This module contains a rudimentary channel between two file descriptors, using [`crate::io`]
//! for reading and writing from the file descriptors.

use crate::{FileDescriptor, io};
use alloc::boxed::Box;
use async_trait::async_trait;
use core::{
    cell::RefCell,
    cmp::Ordering,
    future::Future,
    pin::Pin,
    task::{Context, Poll},
};
use kona_preimage::{
    Channel,
    errors::{ChannelError, ChannelResult},
};

/// [`FileChannel`] is a handle for one end of a bidirectional channel.
#[derive(Debug, Clone, Copy)]
pub struct FileChannel {
    /// File descriptor to read from
    read_handle: FileDescriptor,
    /// File descriptor to write to
    write_handle: FileDescriptor,
}

impl FileChannel {
    /// Create a new [`FileChannel`] from two file descriptors.
    pub const fn new(read_handle: FileDescriptor, write_handle: FileDescriptor) -> Self {
        Self { read_handle, write_handle }
    }

    /// Returns a copy of the [`FileDescriptor`] used for the read end of the channel.
    pub const fn read_handle(&self) -> FileDescriptor {
        self.read_handle
    }

    /// Returns a copy of the [`FileDescriptor`] used for the write end of the channel.
    pub const fn write_handle(&self) -> FileDescriptor {
        self.write_handle
    }
}

#[async_trait]
impl Channel for FileChannel {
    async fn read(&self, buf: &mut [u8]) -> ChannelResult<usize> {
        io::read(self.read_handle, buf).map_err(|_| ChannelError::Closed)
    }

    async fn read_exact(&self, buf: &mut [u8]) -> ChannelResult<usize> {
        ReadFuture::new(*self, buf).await.map_err(|_| ChannelError::Closed)
    }

    async fn write(&self, buf: &[u8]) -> ChannelResult<usize> {
        WriteFuture::new(*self, buf).await.map_err(|_| ChannelError::Closed)
    }
}

/// A future that reads from a channel, returning [`Poll::Ready`] when the buffer is full.
struct ReadFuture<'a> {
    /// The channel to read from
    channel: FileChannel,
    /// The buffer to read into
    buf: RefCell<&'a mut [u8]>,
    /// The number of bytes read so far
    read: usize,
}

impl<'a> ReadFuture<'a> {
    /// Create a new [`ReadFuture`] from a channel and a buffer.
    #[allow(clippy::missing_const_for_fn)]
    fn new(channel: FileChannel, buf: &'a mut [u8]) -> Self {
        Self { channel, buf: RefCell::new(buf), read: 0 }
    }
}

impl Future for ReadFuture<'_> {
    type Output = ChannelResult<usize>;

    fn poll(mut self: Pin<&mut Self>, ctx: &mut Context<'_>) -> Poll<Self::Output> {
        let mut buf = self.buf.borrow_mut();
        let buf_len = buf.len();
        let chunk_read = io::read(self.channel.read_handle, &mut buf[self.read..])
            .map_err(|_| ChannelError::Closed)?;

        // Drop the borrow on self.
        drop(buf);

        self.read += chunk_read;

        match self.read.cmp(&buf_len) {
            Ordering::Greater | Ordering::Equal => Poll::Ready(Ok(self.read)),
            Ordering::Less => {
                // Register the current task to be woken up when it can make progress
                ctx.waker().wake_by_ref();
                Poll::Pending
            }
        }
    }
}

/// A future that writes to a channel, returning [`Poll::Ready`] when the full buffer has been
/// written.
struct WriteFuture<'a> {
    /// The channel to write to
    channel: FileChannel,
    /// The buffer to write
    buf: &'a [u8],
    /// The number of bytes written so far
    written: usize,
}

impl<'a> WriteFuture<'a> {
    /// Create a new [`WriteFuture`] from a channel and a buffer.
    const fn new(channel: FileChannel, buf: &'a [u8]) -> Self {
        Self { channel, buf, written: 0 }
    }
}

impl Future for WriteFuture<'_> {
    type Output = ChannelResult<usize>;

    fn poll(mut self: Pin<&mut Self>, ctx: &mut Context<'_>) -> Poll<Self::Output> {
        match io::write(self.channel.write_handle(), &self.buf[self.written..]) {
            Ok(n) => {
                self.written += n;

                match self.written.cmp(&self.buf.len()) {
                    Ordering::Equal | Ordering::Greater => {
                        // Finished writing
                        Poll::Ready(Ok(self.written))
                    }
                    Ordering::Less => {
                        // Register the current task to be woken up when it can make progress
                        ctx.waker().wake_by_ref();
                        Poll::Pending
                    }
                }
            }
            Err(_) => Poll::Ready(Err(ChannelError::Closed)),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_get_read_handle() {
        let read_handle = FileDescriptor::StdIn;
        let write_handle = FileDescriptor::StdOut;
        let chan = FileChannel::new(read_handle, write_handle);
        let ref_read_handle = chan.read_handle();
        assert_eq!(read_handle, ref_read_handle);
    }
}
