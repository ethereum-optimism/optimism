# Differential test: Rust `brotli` crate vs Go brotli libraries

Pure brotli-library comparison, no OP Stack context. Hunts for inputs where
the Rust [`brotli`](https://crates.io/crates/brotli) crate's
`BrotliDecompress` accepts a stream that both Go libraries
([`andybalholm/brotli`](https://github.com/andybalholm/brotli) and
[`google/brotli` cgo bindings](https://github.com/google/brotli/tree/master/go/cbrotli))
reject as a format error.

Intended as the basis for an upstream issue against the Rust `brotli` crate.

## Layout

```
investigate/brotli-lib-diff/
├── rust_decompress/       Rust binary using brotli::BrotliDecompress (high-level fn)
├── go_decompress/         Go binary using andybalholm/brotli or google/brotli/cbrotli
├── compress/              Go helper: cbrotli encoder (used to generate test fixtures)
├── hunt.py                Sweeps single-byte XOR corruptions, finds Rust-accepts/Go-rejects cases
├── repros.json            Full set of mismatching examples produced by the latest run
└── README.md
```

## Run

```bash
( cd rust_decompress && mise exec -- cargo build --release )
mise exec -- go build -o /tmp/go-brotli-decompress ./go_decompress
mise exec -- go build -o /tmp/cbrotli-compress ./compress

mise exec -- python3 hunt.py
```

## Findings

Default fixtures (5 plaintexts × every byte offset × {0xFF, 0x01, 0x55}
single-byte XORs) yield **204 mismatching cases** where the Rust crate
returns `Ok` and both Go libraries return an error.

Error-class breakdown (from andybalholm; cbrotli emits the same code under a
different name):

| Count | Error |
|---|---|
| 203 | `PADDING_2` |
|   1 | `excessive input` |

`PADDING_2` corresponds to RFC 7932 §9.2's mandate that the reserved /
padding bits at the end of a meta-block header be zero. Both Go libraries
check this; the Rust `brotli` crate does not.

In every case, the Rust crate's decompressed output **exactly matches the
original (uncorrupted) plaintext**. The bit flip lands in a reserved /
padding region that doesn't change the encoded content, but RFC 7932
requires decoders to reject it.

Sample minimal repro (75-byte plaintext "the quick brown fox..."):

```
plaintext (hex): 74686520717569636b2062726f776e20666f78206a756d7073206f76657220746865206c617a7920646f6720747769636520666f7220726564756e64616e637920616e64206c656e677468

valid compressed:
1b4a0000c4f4a469bd79252d22b452ea830d387068b271c041761e36c6ce1384e836f22a0ce789687a04492faaf731a19b0d48b7f01f483342a59c312697a9c6be67855202

invalid (offset 13, byte 0xb4 ^ 0x01 = 0xb5):
1b4a0000c4f4a469bd79252d22b552ea830d387068b271c041761e36c6ce1384e836f22a0ce789687a04492faaf731a19b0d48b7f01f483342a59c312697a9c6be67855202

  - Rust brotli crate:  Ok, returns the original 75-byte plaintext
  - andybalholm/brotli: error "PADDING_2"
  - google/brotli cgo:  error "_ERROR_FORMAT_PADDING_2"
```

## Versions tested

- Rust: `brotli` crate `8.0.2` (depends on `brotli-decompressor 5.0.0`)
- Go: `github.com/andybalholm/brotli v1.1.0`
- Go: `github.com/google/brotli/go/cbrotli v1.1.0` (cgo to libbrotlidec)

## Implications

Any system that relies on brotli decoders agreeing on validity (e.g.
consensus-critical derivation across Go and Rust implementations) is
exposed to divergence here. A malicious encoder could craft a stream
that is well-formed RFC-wise except for a flipped padding bit; Rust
nodes accept and decompress, Go nodes reject.

## Root cause and fix

The bug is in `src/decode.rs` of the
[`brotli-decompressor`](https://github.com/dropbox/rust-brotli-decompressor)
crate (which `brotli` depends on for decoding), in the
`BROTLI_STATE_METABLOCK_DONE` arm of `BrotliDecompressStream`:

```rust
if (!bit_reader::BrotliJumpToByteBoundary(&mut s.br)) {
    result = BrotliDecoderErrorCode::BROTLI_DECODER_ERROR_FORMAT_PADDING_2;
}                                  // <-- missing `break;`
// ... falls through to BROTLI_STATE_DONE which calls WriteRingBuffer
// and unconditionally overwrites `result`, silently dropping the error.
```

The parallel padding-2 site earlier in the same function (line 2871 of
5.0.0, also `BrotliJumpToByteBoundary` after metablock-length decode)
correctly has `break;`. Only this one was missing.

`padding_2_fix.patch` in this directory is the two-line fix. To reproduce
locally:

```bash
cd forks
git clone -b 5.0.0 https://github.com/dropbox/rust-brotli-decompressor.git
( cd rust-brotli-decompressor && git apply ../../padding_2_fix.patch )
( cd ../rust_decompress && cargo build --release )  # picks up the fork via [patch.crates-io]
mise exec -- python3 ../hunt.py
```

## Validation

After applying the patch and rebuilding `rust_decompress` against the
fork (via `[patch.crates-io]` in its `Cargo.toml`):

| Run | Total mismatches | PADDING_2 | Excessive input |
|---|---|---|---|
| Stock `brotli` 8.0.2 | 204 | 203 | 1 |
| **Patched** | **1** | **0** | 1 |

All 203 PADDING_2 mismatches are eliminated. The fork's own test suite
(75 tests) still passes.

## Remaining 1 mismatch: excessive input (API design, not RFC violation)

The single residual mismatch is an `excessive input` case at fixture
"ascending" / offset 124. The C reference library returns SUCCESS even
when there are leftover input bytes after the brotli stream ends; both
Go libraries' high-level Reader wrappers (cbrotli's `Reader.Read` and
andybalholm's equivalent) explicitly check for leftover input and return
"excessive input". The Rust crate's `BrotliDecompress` does not.

This is a wrapper-level API design choice rather than an RFC violation
of the decoder itself. We did not include this in the fix patch — it
should be a separate discussion with upstream about whether
`BrotliDecompress` should be strict about trailing input.
