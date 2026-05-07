//! Same probe as ../spec_check (Go) and ../spec_check_google (cgo), but using
//! the Rust `brotli` crate's high-level Reader API. Reads the compressed bytes
//! as hex from argv[1] so we can feed it the exact same bytes the Go probes
//! produced.

use std::io::Read;

fn main() {
    let arg = std::env::args().nth(1).expect("usage: spec-check-rust <hex>");
    let compressed = hex::decode(arg.trim()).expect("hex");
    println!("compressed: {} bytes", compressed.len());

    println!();
    println!("{:<10} {:<25} {:<10} {:<10}", "trunc_at", "result", "out_bytes", "extra");
    println!("{}", "-".repeat(70));

    for i in (1..compressed.len()).rev() {
        let truncated = &compressed[..i];

        // Decompressor::new + read_to_end is the high-level Reader API. The
        // low-level streaming API is what kona uses (BrotliDecompressStream).
        let mut decoder = brotli::Decompressor::new(truncated, 4096);
        let mut output = Vec::new();
        let result = decoder.read_to_end(&mut output);

        let (status, extra) = match &result {
            Ok(_) => ("Ok", "  ⚠ accepted truncated stream"),
            Err(e) => {
                let kind = e.kind();
                let s = format!("{:?}", kind);
                (Box::leak(s.into_boxed_str()) as &str, "")
            }
        };
        println!("{:<10} {:<25} {:<10} {}", i, status, output.len(), extra);
    }
}
