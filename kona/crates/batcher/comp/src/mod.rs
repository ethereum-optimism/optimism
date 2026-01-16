//! Contains compression and decompression primitives for Optimism.

#[cfg(feature = "std")]
mod variant;
#[cfg(feature = "std")]
pub use variant::VariantCompressor;

mod config;
pub use config::Config;

mod types;
pub use types::{CompressionAlgo, CompressorError, CompressorResult, CompressorType};

mod zlib;
pub use zlib::{ZlibCompressor, compress_zlib, decompress_zlib};

mod brotli;
#[cfg(feature = "std")]
pub use brotli::{BrotliCompressionError, compress_brotli, BrotliCompressor};

mod traits;
pub use traits::{ChannelCompressor, CompressorWriter};

#[cfg(feature = "std")]
mod shadow;
#[cfg(feature = "std")]
pub use shadow::ShadowCompressor;

#[cfg(feature = "std")]
mod ratio;
#[cfg(feature = "std")]
pub use ratio::RatioCompressor;
