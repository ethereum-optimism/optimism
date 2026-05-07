# Brotli decompression divergence between kona and op-node

Investigation for [#19333](https://github.com/ethereum-optimism/optimism/issues/19333) Bug 1.
The original claim was that kona's `decompress_brotli` returning `Ok(partial)` on
`NeedsMoreInput` / `ResultFailure` causes kona to accept brotli channels that op-node
rejects, creating a Go-vs-Rust derivation divergence.

This worktree builds a differential test harness that runs both implementations
on identical inputs (truncated and single-byte-corrupted brotli channels) and
quantifies the divergence under three rules:

- **Loose** — current kona behavior (return `Ok(partial)` on partial decode)
- **Strict + cap** — reject `NeedsMoreInput`/`ResultFailure`, but accept output truncated at `MAX_RLP_BYTES_PER_CHANNEL`
- **Strict + reject-on-cap** — reject `NeedsMoreInput`/`ResultFailure`, AND reject if output exceeds `MAX_RLP_BYTES_PER_CHANNEL` (Seb's proposed future-fork rule)

## Layout

```
investigate/brotli-divergence/
├── kona_harness/        Rust binary using kona_protocol::decompress_brotli + BatchReader
├── opnode_harness/      Go binary using op-node's BatchReader (andybalholm/brotli + RLP)
├── gen_channel/         Go helper: emits a brotli-compressed channel of N SingularBatches
├── spec_check/          Go probe: tests andybalholm/brotli RFC 7932 compliance on truncated input
├── driver.py            Sweep driver: truncates / corrupts the channel and diffs both harnesses
└── README.md
```

The Rust strict variant lives at
`rust/kona/crates/protocol/protocol/src/brotli.rs::decompress_brotli_strict`
(also re-exported from `kona_protocol`). It is throwaway — the worktree is
not intended to merge.

## Run

```bash
# build harnesses
( cd kona_harness && mise exec -- cargo build --release )
( cd .. && mise exec -- go build -o /tmp/opnode-brotli-harness ./opnode_harness )
( cd .. && mise exec -- go build -o /tmp/gen-channel ./gen_channel )

# differential sweep (loose + strict + future-rule)
N_BATCHES=20 python3 driver.py
```

## Results

20-batch fixture (236-byte channel, ~1860-byte plaintext, MAX_RLP_BYTES_PER_CHANNEL = 100 MB):

| Sweep | Mode | Agree | op-node-more-lenient | kona-more-lenient |
|---|---|---|---|---|
| Truncate (234 cases) | Loose | 65 | 152 | 0 |
| Truncate | Strict (either) | 56 | 178 | 0 |
| Corrupt (235 cases) | Loose | 174 | 37 | 8 |
| Corrupt | Strict (either) | 184 | 50 | 1 |

**Strict mode shrinks but does not eliminate divergence.** The two strict variants (with-cap-acceptance vs reject-on-cap) produce identical results on this fixture because the cap is never triggered; the divergences live entirely in the non-cap path.

## Root cause

Probing andybalholm/brotli directly (`spec_check/main.go`) on a known-valid
51-byte brotli stream truncated by N bytes:

```
trunc_at  result            out_bytes
36-51     <nil>  ⚠           varies     ← accepts truncated stream
≤35       ErrUnexpectedEOF  0           (correctly rejects)
```

`andybalholm/brotli` reports clean `<nil>` on truncations of the trailing
1–16 bytes — i.e., when the truncation falls inside or after the final
meta-block but before the `ISLAST=1` marker is seen. **Per RFC 7932 §1.5
and §9, a brotli decoder must detect end-of-stream via the `ISLAST` flag
in the meta-block header. Reporting clean termination on a stream that
never reaches `ISLAST=1` is a spec violation.**

The Rust `brotli` crate correctly returns `NeedsMoreInput` on the same
inputs.

The corruption cases at offset=2 (header bytes) suggest a separate
andybalholm leniency on reserved-bit checks, but the truncation bug
dominates the divergence count.

## Conclusion

- The original issue's mechanism (kona's wrapper returning `Ok(partial)`) is real but minor in this fixture (8/235 corruption cases).
- The dominant cause is **andybalholm/brotli accepting RFC-invalid streams**, which makes op-node accept channels kona rejects.
- A wrapper-level strict rule on kona alone shifts the divergence direction but doesn't eliminate it.
- Even Seb's proposed future-fork derivation rule (output ≤ MAX, brotli fully terminated, no leftover input) cannot achieve parity while one library deviates from RFC.

## Recommended path forward

1. **Use RFC 7932-compliant brotli libraries on both sides.** Confirm Rust's `brotli` crate is compliant (preliminary results suggest yes); replace `andybalholm/brotli` in op-node with `google/brotli` Go bindings (cgo to the reference C implementation), or another library verified against RFC.
2. **Optionally adopt the stricter derivation rule** (output ≤ MAX, full brotli termination, no leftover input). With RFC-compliant libraries, this rule achieves parity AND tightens the spec.
3. **Report the andybalholm truncation bug upstream** with the `spec_check/main.go` repro so the wider Go ecosystem benefits.
