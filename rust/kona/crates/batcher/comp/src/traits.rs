//! Contains the core `Compressor` trait.

use crate::CompressorResult;
use alloc::vec::Vec;

/// Compressor Writer
///
/// A trait that expands the standard library `Write` trait to include
/// compression-specific methods and return [CompressorResult] instead of
/// standard library `Result`.
#[allow(clippy::len_without_is_empty)]
pub trait CompressorWriter {
    /// Writes the given data to the compressor.
    fn write(&mut self, data: &[u8]) -> CompressorResult<usize>;

    /// Flushes the buffer.
    fn flush(&mut self) -> CompressorResult<()>;

    /// Closes the compressor.
    fn close(&mut self) -> CompressorResult<()>;

    /// Resets the compressor.
    fn reset(&mut self);

    /// Returns the length of the compressed data.
    fn len(&self) -> usize;

    /// Reads the compressed data into the given buffer.
    /// Returns the number of bytes read.
    fn read(&mut self, buf: &mut [u8]) -> CompressorResult<usize>;
}

/// Channel Compressor
///
/// A compressor for channels.
pub trait ChannelCompressor: CompressorWriter {
    /// Returns the compressed data buffer.
    fn get_compressed(&self) -> Vec<u8>;
}
