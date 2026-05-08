//! Reads hex-encoded brotli-compressed bytes from stdin, calls
//! `brotli::BrotliDecompress` (the high-level convenience function from the
//! `brotli` crate), and prints:
//!
//!   STATUS=ok|err
//!   ERR=<error display string if err>
//!   OUTPUT=<hex>
//!
//! No size limit, no streaming wrapper logic — purely the library function.

use std::io::Read;

fn hex_encode(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        out.push_str(&format!("{:02x}", b));
    }
    out
}

fn main() {
    let mut hex_input = String::new();
    std::io::stdin().read_to_string(&mut hex_input).expect("read stdin");
    let compressed = hex::decode(hex_input.trim()).expect("invalid hex");

    let mut output: Vec<u8> = Vec::new();
    let mut reader = &compressed[..];
    let result = brotli::BrotliDecompress(&mut reader, &mut output);

    match result {
        Ok(()) => {
            println!("STATUS=ok");
            println!("OUTPUT={}", hex_encode(&output));
        }
        Err(e) => {
            println!("STATUS=err");
            println!("ERR={}", e);
            // Per Read+Write contract, `output` may still contain partial bytes
            // written before the error. Emit them too so we can compare.
            println!("OUTPUT={}", hex_encode(&output));
        }
    }
}
