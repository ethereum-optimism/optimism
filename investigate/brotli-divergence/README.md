# Brotli decompression divergence between kona and op-node

Investigation for [#19333](https://github.com/ethereum-optimism/optimism/issues/19333) Bug 1.
The original claim was that kona's `decompress_brotli` returning `Ok(partial)` on
`NeedsMoreInput` / `ResultFailure` causes kona to accept brotli channels that op-node
rejects, creating a Go-vs-Rust derivation divergence.

This worktree builds a differential test harness that runs both implementations
on identical truncated/corrupted brotli channels and quantifies the divergence
under three rules:

- **Loose** — current kona behavior (`Ok(partial)` on partial decode)
- **Strict + cap** — reject `NeedsMoreInput`/`ResultFailure`, but accept output truncated at `MAX_RLP_BYTES_PER_CHANNEL`
- **Strict + reject-on-cap** — reject `NeedsMoreInput`/`ResultFailure`, AND reject if output exceeds `MAX_RLP_BYTES_PER_CHANNEL` (the proposed future-fork rule)

It also probes three brotli libraries directly to isolate where the divergence
comes from:

- Rust `brotli` crate (used by kona)
- Go `andybalholm/brotli` (used by op-node)
- Go `github.com/google/brotli/go/cbrotli` (cgo to the reference C implementation)

## Layout

```
investigate/brotli-divergence/
├── kona_harness/        Rust binary using kona_protocol::decompress_brotli + BatchReader
├── opnode_harness/      Go binary using op-node's BatchReader (andybalholm + RLP)
├── gen_channel/         Go helper: emits a brotli-compressed channel of N SingularBatches
├── spec_check/          Go probe: andybalholm/brotli on truncated streams
├── spec_check_google/   Go probe: cgo to google/brotli reference C library
├── spec_check_rust/     Rust probe: brotli crate (high-level Reader API)
├── spec_compare.py      Side-by-side: feeds same compressed bytes to all three decoders
├── driver.py            Differential sweep: truncate + corrupt + diff harnesses
└── README.md
```

The Rust `decompress_brotli_strict` variant lives at
`rust/kona/crates/protocol/protocol/src/brotli.rs`. It is throwaway —
the worktree is not intended to merge.

## Run

```bash
( cd kona_harness && mise exec -- cargo build --release )
mise exec -- go build -o /tmp/opnode-brotli-harness ./opnode_harness
mise exec -- go build -o /tmp/gen-channel ./gen_channel

# Differential sweep on a 20-batch fixture
N_BATCHES=20 python3 driver.py

# Decoder-level comparison across all three libraries
python3 spec_compare.py
```

## Findings

### 1. The dominant cause was a kona wrapper bug, not a library mismatch

`kona::decompress_brotli` had a subtle bug: when both `available_in` and
`available_out` reach 0 in the same iteration of the decompression loop,
`BrotliDecompressStream` returns `NeedsMoreInput` (priority over
`NeedsMoreOutput`). Kona's loop treated this as "done" and returned the
partial output buffer, even though brotli might have produced more bytes
given more output space.

Go's `bufio`/`io.ReadAll` indirection avoids this by always re-reading
when input is consumed, which gives brotli more output space on the next
call.

Fix in this branch (`brotli.rs`): treat `NeedsMoreInput && available_out == 0`
as the `NeedsMoreOutput` arm — grow the buffer and retry. Brotli will signal
`NeedsMoreInput` again with `available_out > 0` only when truly out of input.

### 2. Three brotli libraries agree on truncated streams; the Rust crate's *high-level Reader* doesn't

`spec_compare.py` feeds the same compressed bytes to all three decoders
and compares: with the kona-wrapper fix applied, **all three produce
byte-identical output** at every truncation point.

The earlier impression that "andybalholm is buggy / non-RFC" was caused
by comparing against the Rust crate's high-level Reader API
(`brotli::Decompressor::read_to_end`), which surfaces an `InvalidData`
error on truncated streams *but still returns the same number of decoded
bytes*. The underlying state machine (the low-level `BrotliDecompressStream`
that kona uses) matches the C reference library — both signal
`NeedsMoreInput` on truncated streams and produce the same partial bytes.

Switching op-node from `andybalholm/brotli` to `google/brotli` does **not**
change behavior on these inputs. Both Go libraries report clean `io.EOF`
identically, and both produce the same partial bytes that kona produces
(after the wrapper fix).

### 3. Differential sweep results

20-batch fixture (236-byte channel, ~1864-byte plaintext, MAX = 100 MB):

| Sweep | Mode | Agree | op-node-more-lenient | kona-more-lenient | bytes differ only |
|---|---|---|---|---|---|
| Truncate (234) | Loose, kona unfixed | 65 | 152 | 0 | 17 |
| Truncate (234) | **Loose, kona fixed** | **234** | 0 | 0 | 0 |
| Corrupt (235) | Loose, kona unfixed | 174 | 37 | 8 | 16 |
| Corrupt (235) | **Loose, kona fixed** | **224** | 0 | 8 | 3 |

The wrapper fix removes **all** truncation divergences. The remaining
8 + 3 = 11 corruption-only cases come from genuine differences between
the Rust `brotli` crate and the Go libraries on *corrupted* (not
truncated) input — these are the only cases the original issue report's
"kona accepts what op-node rejects" mechanism actually applies to.

### 4. Strict mode is a separate problem

Both proposed strict variants (with-cap-acceptance and reject-on-cap)
still leave 178 truncate divergences and ~50 corruption divergences,
because the Go reader signals `io.EOF` (clean termination) at points
where kona's strict path returns `NeedsMoreInput` error. Both libraries
have the same underlying state machine; the difference is in how the
*Reader wrapper* surfaces "stream not finished" to its caller.

For strict-mode parity, op-node would need to query
`BrotliDecoderIsFinished` after the final read, not rely on
`io.EOF` from `io.ReadAll`. This is a Go-side fix.

## Recommended path forward

1. **Land the kona wrapper fix.** It removes ~95 % of the divergence in
   this fixture without any spec change or library swap. Spec-correct
   per RFC 7932 (don't drop output bytes that brotli could still flush).

2. **For the residual 11 corruption cases**, decide whether they matter.
   These are inputs where the Rust crate's low-level state machine and
   the C reference disagree on whether a corrupted stream is recoverable.
   Honest batchers don't produce corrupted brotli; the cases require a
   malicious or buggy submitter to surface. Possible further actions:
   - Document as "adversarial-only" and rely on dispute mechanics.
   - Add the corruption-fuzz cases to a CI fixture so future library
     updates don't regress.

3. **If you want strict-mode parity** (Seb's proposed future-fork rule:
   output ≤ MAX, brotli fully terminated, no leftover input), apply
   the strict rule to *both* sides:
   - Kona: already implemented in this branch (`decompress_brotli_strict`).
   - Op-node: replace `io.ReadAll` with a wrapper that verifies
     `BrotliDecoderIsFinished` and zero leftover input bytes after EOF.

Library swap (`andybalholm` → `google/brotli`) is **not** required —
the Go libraries already behave identically.
