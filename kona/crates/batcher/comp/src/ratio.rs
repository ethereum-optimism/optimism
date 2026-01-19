//! Contains the ratio compressor for Optimism.
//!
//! This is a port of the [RatioCompressor][rc] from the op-batcher.
//!
//! [rc]: https://github.com/ethereum-optimism/optimism/blob/develop/op-batcher/compressor/ratio_compressor.go#L7

use crate::{CompressorResult, CompressorWriter, Config, VariantCompressor};

/// Ratio Compressor
///
/// The ratio compressor uses the target size and a compression ration parameter
/// to determine how much data can be written to the compressor before it's
/// considered full. The full calculation is as follows:
///
/// full = uncompressedLength * approxCompRatio >= targetFrameSize * targetNumFrames
///
/// The ratio compressor wraps a [VariantCompressor] which dispatches to the
/// appropriate compression algorithm (ZLIB or Brotli).
#[derive(Debug, Clone)]
pub struct RatioCompressor {
    /// The compressor configuration.
    config: Config,
    /// The amount of data currently in the compressor.
    lake: u64,
    /// The inner [VariantCompressor] that will be used to compress the data.
    compressor: VariantCompressor,
}

impl RatioCompressor {
    /// Create a new [RatioCompressor] with the given [VariantCompressor].
    pub const fn new(config: Config, compressor: VariantCompressor) -> Self {
        Self { config, lake: 0, compressor }
    }

    /// Calculates the input threshold in bytes.
    pub fn input_threshold(&self) -> usize {
        let target_frame_size = self.config.target_output_size;
        let approx_comp_ratio = self.config.approx_compr_ratio;

        (target_frame_size as f64 / approx_comp_ratio) as usize
    }

    /// Returns if the compressor is full (exceeds the input threshold).
    pub fn is_full(&self) -> bool {
        self.lake >= self.input_threshold() as u64
    }
}

impl From<Config> for RatioCompressor {
    fn from(config: Config) -> Self {
        let compressor = VariantCompressor::from(config.compression_algo);
        Self::new(config, compressor)
    }
}

impl CompressorWriter for RatioCompressor {
    fn write(&mut self, data: &[u8]) -> CompressorResult<usize> {
        match self.compressor.write(data) {
            Ok(n) => {
                self.lake += n as u64;
                Ok(n)
            }
            Err(e) => Err(e),
        }
    }

    fn flush(&mut self) -> CompressorResult<()> {
        self.compressor.flush()
    }

    fn close(&mut self) -> CompressorResult<()> {
        self.compressor.close()
    }

    fn reset(&mut self) {
        self.compressor.reset();
        self.lake = 0;
    }

    fn len(&self) -> usize {
        self.compressor.len()
    }

    fn read(&mut self, buf: &mut [u8]) -> CompressorResult<usize> {
        self.compressor.read(buf)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{CompressionAlgo, CompressorType};

    #[test]
    fn test_input_threshold() {
        let config = Config {
            target_output_size: 1024,
            approx_compr_ratio: 0.5,
            compression_algo: CompressionAlgo::Zlib,
            kind: CompressorType::Ratio,
        };

        let inner = VariantCompressor::from(CompressionAlgo::Zlib);
        let compressor = RatioCompressor::new(config, inner);
        assert_eq!(compressor.input_threshold(), 2048);
    }

    #[test]
    fn test_ratio_compressor() {
        let config = Config {
            target_output_size: 1024,
            approx_compr_ratio: 0.5,
            compression_algo: CompressionAlgo::Zlib,
            kind: CompressorType::Ratio,
        };

        let inner = VariantCompressor::from(CompressionAlgo::Zlib);
        let mut compressor = RatioCompressor::new(config, inner);

        assert!(!compressor.is_full());
        compressor.write(&[0; 2048]).unwrap();
        assert!(compressor.is_full());
        assert_eq!(compressor.len(), 18);

        let mut buf = [];
        compressor.read(&mut buf).unwrap();
        assert_eq!(buf.len(), 0);

        compressor.flush().unwrap();

        compressor.reset();
        assert!(!compressor.is_full());
        assert_eq!(compressor.len(), 0);

        compressor.close().unwrap();
    }
}
