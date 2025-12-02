//! Test Utilities for the compression crate.

use crate::{ChannelCompressor, CompressorError, CompressorResult, CompressorWriter};
use alloc::vec::Vec;
use alloy_primitives::Bytes;

/// A Mock compressor for testing.
#[derive(Debug, Clone, Default)]
pub struct MockCompressor {
    /// Compressed bytes
    pub compressed: Option<Bytes>,
    /// Whether to throw a read error.
    pub read_error: bool,
}

impl CompressorWriter for MockCompressor {
    fn write(&mut self, data: &[u8]) -> CompressorResult<usize> {
        let data = data.to_vec();
        let written = data.len();
        self.compressed = Some(Bytes::from(data));
        Ok(written)
    }

    fn flush(&mut self) -> CompressorResult<()> {
        Ok(())
    }

    fn close(&mut self) -> CompressorResult<()> {
        Ok(())
    }

    fn reset(&mut self) {
        self.compressed = None;
    }

    fn len(&self) -> usize {
        self.compressed.as_ref().map(|b| b.len()).unwrap_or(0)
    }

    fn read(&mut self, buf: &mut [u8]) -> CompressorResult<usize> {
        if self.read_error {
            return Err(CompressorError::Full);
        }
        let len = self.compressed.as_ref().map(|b| b.len()).unwrap_or(0);
        buf[..len].copy_from_slice(self.compressed.as_ref().unwrap());
        Ok(len)
    }
}

impl ChannelCompressor for MockCompressor {
    fn get_compressed(&self) -> Vec<u8> {
        self.compressed.as_ref().unwrap().to_vec()
    }
}
