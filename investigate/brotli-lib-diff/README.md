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
"ascending" / offset 124. RFC 7932 defines the brotli stream as ending
at the ISLAST=1 metablock plus byte-alignment padding (§9.3) and is
silent on trailing input bytes. The reference C library
(`libbrotlidec`) returns `BROTLI_DECODER_RESULT_SUCCESS` and leaves
leftover input for the caller; both Go libraries' high-level Reader
wrappers (`cbrotli.Reader.Read` and andybalholm's equivalent) add an
explicit "no leftover input" check on top and return "excessive input"
otherwise. The Rust `brotli` crate's `BrotliDecompress` does not add
this check.

This is a wrapper-level API design choice rather than an RFC violation,
so it is **not** included in the fix patch. It is a separate
conversation with upstream about whether `BrotliDecompress` should be
strict about trailing input — and one that risks breaking existing
callers who deliberately stream multiple things through the same
Reader.

## Effect on the OP Stack consensus divergence

With the patch applied to the underlying `brotli-decompressor` crate,
the OP-Stack-context differential sweep (kona's `decompress_brotli` +
`BatchReader` vs op-node's `BatchReader` on the 20-batch fixture)
converges to **0 divergences** in loose mode:

| Sweep | Pre-patch | **Post-patch** |
|---|---|---|
| Truncate (234) — agree | 234 | **234** |
| Corrupt (235) — agree | 224 | **235** |
| Corrupt — kona_more_lenient | 8 | **0** |
| Corrupt — bytes_differ_only | 3 | **0** |

This holds even though kona's existing `decompress_brotli` doesn't
explicitly require all input to be consumed: brotli's PADDING_2 check
fires *before* the final metablock's bytes are flushed from its
internal ringbuffer to the caller's output buffer, so `written == 0`
at the error point and kona's existing `_ if written == 0 => Err` arm
correctly rejects. No changes needed in kona itself.
