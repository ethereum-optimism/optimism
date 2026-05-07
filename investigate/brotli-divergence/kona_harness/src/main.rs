//! Reads hex-encoded channel bytes from stdin and emits kona's behavior:
//!
//!   DECOMPRESS_RESULT=ok|err
//!   DECOMPRESS_BYTES=<hex>           (bytes returned by decompress_brotli, even on err)
//!   BATCH_COUNT=<n>                  (batches accepted by full BatchReader pipeline)
//!
//! The input must start with the channel version byte (0x01 for brotli);
//! the rest is the brotli-compressed payload.

use std::io::Read;

use kona_genesis::{HardForkConfig, MAX_RLP_BYTES_PER_CHANNEL_FJORD, RollupConfig};
use kona_protocol::{BatchReader, decompress_brotli, decompress_brotli_strict};

fn hex_encode(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push_str(&format!("{:02x}", b));
    }
    out
}

fn hex_decode(s: &str) -> Vec<u8> {
    let s = s.trim();
    let mut out = Vec::with_capacity(s.len() / 2);
    let bytes = s.as_bytes();
    for chunk in bytes.chunks_exact(2) {
        let hi = (chunk[0] as char).to_digit(16).unwrap() as u8;
        let lo = (chunk[1] as char).to_digit(16).unwrap() as u8;
        out.push((hi << 4) | lo);
    }
    out
}

fn main() {
    let mut input = String::new();
    std::io::stdin().read_to_string(&mut input).expect("read stdin");
    let channel = hex_decode(&input);

    assert!(!channel.is_empty(), "empty channel");
    assert_eq!(channel[0], 0x01, "channel version must be 0x01 (brotli)");

    let max = MAX_RLP_BYTES_PER_CHANNEL_FJORD as usize;

    // 1) Raw brotli decompression (current kona behavior).
    match decompress_brotli(&channel[1..], max) {
        Ok(bytes) => {
            println!("DECOMPRESS_RESULT=ok");
            println!("DECOMPRESS_BYTES={}", hex_encode(&bytes));
        }
        Err(_) => {
            println!("DECOMPRESS_RESULT=err");
            println!("DECOMPRESS_BYTES=");
        }
    }

    // 1b) Strict brotli decompression (Seb's proposed rule).
    match decompress_brotli_strict(&channel[1..], max) {
        Ok(bytes) => {
            println!("STRICT_RESULT=ok");
            println!("STRICT_BYTES={}", hex_encode(&bytes));
        }
        Err(_) => {
            println!("STRICT_RESULT=err");
            println!("STRICT_BYTES=");
        }
    }

    // 2) Full BatchReader pipeline: how many batches kona accepts.
    // Set Fjord active so brotli is gated open; origin_timestamp = 1 (post-Fjord).
    let cfg = RollupConfig {
        hardforks: HardForkConfig { fjord_time: Some(0), ..Default::default() },
        ..Default::default()
    };
    let mut reader = BatchReader::new(channel.clone(), max, /* origin_timestamp = */ 1);
    let mut batch_count = 0usize;
    while reader.next_batch(&cfg).is_some() {
        batch_count += 1;
    }
    println!("BATCH_COUNT={}", batch_count);
}
