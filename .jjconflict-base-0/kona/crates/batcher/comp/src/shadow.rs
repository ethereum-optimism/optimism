//! Contains the shadow compressor for Optimism.
//!
//! This is a port of the [ShadowCompressor][sc] from the op-batcher.
//!
//! [sc]: https://github.com/ethereum-optimism/optimism/blob/develop/op-batcher/compressor/shadow_compressor.go#L18

use crate::{CompressorError, CompressorResult, CompressorWriter, Config, VariantCompressor};

/// The largest potential blow-up in bytes we expect to see when compressing
/// arbitrary (e.g. random) data.  Here we account for a 2 byte header, 4 byte
/// digest, 5 byte eof indicator, and then 5 byte flate block header for each 16k of potential
/// data. Assuming frames are max 128k size (the current max blob size) this is 2+4+5+(5*8) = 51
/// bytes.  If we start using larger frames (e.g. should max blob size increase) a larger blowup
/// might be possible, but it would be highly unlikely, and the system still works if our
/// estimate is wrong -- we just end up writing one more tx for the overflow.
const SAFE_COMPRESSION_OVERHEAD: u64 = 51;

// The number of final bytes a `zlib.Writer` call writes to the output buffer.
const CLOSE_OVERHEAD_ZLIB: u64 = 9;

/// Shadow Compressor
///
/// The shadow compressor contains two compression buffers, one for size estimation, and
/// one for the final compressed data. The first compression buffer is flushed on every
/// write, and the second isn't, which means the final compressed data is always at least
/// smaller than the size estimation.
///
/// One exception to the rule is when the first write to the buffer is not checked against
/// the target. This allows individual blocks larger than the target to be included.
/// Notice, this will be split across multiple channel frames.
#[derive(Debug, Clone)]
pub struct ShadowCompressor {
    /// The compressor configuration.
    config: Config,
    /// The inner [VariantCompressor] that will be used to compress the data.
    compressor: VariantCompressor,
    /// The shadow compressor.
    shadow: VariantCompressor,

    /// Flags that the buffer is full.
    is_full: bool,
    /// An upper bound on the size of the compressed data.
    bound: u64,
}

impl ShadowCompressor {
    /// Creates a new [ShadowCompressor] with the given [VariantCompressor].
    pub const fn new(
        config: Config,
        compressor: VariantCompressor,
        shadow: VariantCompressor,
    ) -> Self {
        Self { config, is_full: false, compressor, shadow, bound: SAFE_COMPRESSION_OVERHEAD }
    }
}

impl From<Config> for ShadowCompressor {
    fn from(config: Config) -> Self {
        let compressor = VariantCompressor::from(config.compression_algo);
        let shadow = VariantCompressor::from(config.compression_algo);
        Self::new(config, compressor, shadow)
    }
}

impl CompressorWriter for ShadowCompressor {
    fn write(&mut self, data: &[u8]) -> CompressorResult<usize> {
        // If the buffer is full, error so the user can flush.
        if self.is_full {
            return Err(CompressorError::Full);
        }

        // Write to the shadow compressor.
        self.shadow.write(data)?;

        // The new bound increases by the length of the compressed data.
        let mut newbound = data.len() as u64;
        if newbound > self.config.target_output_size {
            // Don't flush the buffer if there's a chance we're over the size limit.
            self.shadow.flush()?;
            newbound = self.shadow.len() as u64 + CLOSE_OVERHEAD_ZLIB;
            if newbound > self.config.target_output_size {
                self.is_full = true;
                // Only error if the buffer has been written to.
                if self.compressor.len() > 0 {
                    return Err(CompressorError::Full);
                }
            }
        }

        // Update the bound and compress.
        self.bound = newbound;
        self.compressor.write(data)
    }

    fn len(&self) -> usize {
        self.compressor.len()
    }

    fn flush(&mut self) -> CompressorResult<()> {
        self.shadow.flush()
    }

    fn close(&mut self) -> CompressorResult<()> {
        self.shadow.close()
    }

    fn reset(&mut self) {
        self.compressor.reset();
        self.shadow.reset();
        self.is_full = false;
        self.bound = SAFE_COMPRESSION_OVERHEAD;
    }

    fn read(&mut self, buf: &mut [u8]) -> CompressorResult<usize> {
        self.compressor.read(buf)
    }
}
