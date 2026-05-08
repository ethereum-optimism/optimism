#!/usr/bin/env python3
"""Hunt for corrupted brotli streams where the Rust `brotli` crate's
`BrotliDecompress` accepts (returns Ok) but both Go libraries reject.

Generates compressed brotli streams from a few representative plaintexts,
sweeps single-byte XOR corruptions across every offset, runs all three
decoders, and prints the cases where Rust accepts and Go rejects (the
direction the upstream issue would target).
"""

import os
import subprocess
import sys

ROOT = os.path.dirname(os.path.abspath(__file__))
RUST = os.path.join(ROOT, "rust_decompress/target/release/rust-brotli-decompress")
GO = "/tmp/go-brotli-decompress"


def run(cmd, stdin=None):
    p = subprocess.run(cmd, input=stdin, capture_output=True, text=True, check=False)
    out = {}
    for line in p.stdout.splitlines():
        if "=" in line:
            k, v = line.split("=", 1)
            out[k] = v
    return out


def go_compress(plaintext_hex):
    """Use cbrotli (reference C library) as the encoder."""
    p = subprocess.run(["/tmp/cbrotli-compress", plaintext_hex], capture_output=True, text=True, check=True)
    return p.stdout.strip()


def hunt_for_plaintext(label, plaintext_bytes):
    plaintext_hex = plaintext_bytes.hex()
    compressed_hex = go_compress(plaintext_hex)
    compressed = bytes.fromhex(compressed_hex)
    print(f"\n=== plaintext: {label} ({len(plaintext_bytes)}b plain → {len(compressed)}b compressed) ===")
    print(f"compressed_hex: {compressed_hex}")

    sanity_rust = run([RUST], stdin=compressed_hex)
    sanity_andy = run([GO, "--impl=andybalholm"], stdin=compressed_hex)
    sanity_cbrotli = run([GO, "--impl=cbrotli"], stdin=compressed_hex)
    if not (sanity_rust["STATUS"] == sanity_andy["STATUS"] == sanity_cbrotli["STATUS"] == "ok"):
        print("  WARN: sanity check failed on uncorrupted input")

    found = []
    for offset in range(len(compressed)):
        for xor_byte in (0xFF, 0x01, 0x55):
            corrupted = bytearray(compressed)
            corrupted[offset] ^= xor_byte
            corrupted_hex = corrupted.hex()

            r = run([RUST], stdin=corrupted_hex)
            a = run([GO, "--impl=andybalholm"], stdin=corrupted_hex)
            c = run([GO, "--impl=cbrotli"], stdin=corrupted_hex)

            rust_ok = r.get("STATUS") == "ok"
            andy_ok = a.get("STATUS") == "ok"
            cbrotli_ok = c.get("STATUS") == "ok"

            # The interesting direction: Rust accepts, BOTH Go libs reject.
            if rust_ok and not andy_ok and not cbrotli_ok:
                rust_out = r.get("OUTPUT", "")
                # Skip cases where Rust output is empty (less surprising;
                # 0-byte "Ok" is debatable but downstream may still treat it
                # as a valid empty stream).
                if not rust_out:
                    continue
                found.append({
                    "offset": offset,
                    "xor": xor_byte,
                    "corrupted_hex": corrupted_hex,
                    "rust_output_len": len(rust_out) // 2,
                    "andy_err": a.get("ERR", ""),
                    "cbrotli_err": c.get("ERR", ""),
                })

    print(f"  found {len(found)} mismatches (Rust accepts, both Go libs reject)")
    return found


def main():
    fixtures = [
        ("ascii-text", b"the quick brown fox jumps over the lazy dog twice for redundancy and length"),
        ("repeated-byte", bytes([0xAA] * 200)),
        ("ascending", bytes(range(256))),
        ("structured-rlp-like", b"\xb8\x40" + b"\x00" * 64 + b"\xa0" + bytes(range(32))),
        ("zeros-1k", bytes(1024)),
    ]

    all_found = []
    for label, plain in fixtures:
        all_found += [(label, m) for m in hunt_for_plaintext(label, plain)]

    print(f"\n\n=== TOTAL: {len(all_found)} mismatching examples ===")
    # Tally errors so we know what error class dominates
    err_counts = {}
    for _, m in all_found:
        err = m["andy_err"]
        err_counts[err] = err_counts.get(err, 0) + 1
    print("\nError classes (from andybalholm — cbrotli reports the same kind):")
    for err, n in sorted(err_counts.items(), key=lambda kv: -kv[1]):
        print(f"  {n:>4}  {err}")

    if all_found:
        print("\nMinimal repros (Rust BrotliDecompress accepts, both Go libs reject):\n")
        seen_labels = {}
        for label, m in all_found:
            seen_labels.setdefault(label, [])
            seen_labels[label].append(m)
        for label, ms in seen_labels.items():
            print(f"--- fixture: {label} (showing up to 2 of {len(ms)}) ---")
            for m in ms[:2]:
                print(f"  offset={m['offset']:>3}  xor=0x{m['xor']:02x}  rust_out_len={m['rust_output_len']}")
                print(f"    corrupted_hex: {m['corrupted_hex']}")
                print(f"    andybalholm:   {m['andy_err']}")
                print(f"    cbrotli:       {m['cbrotli_err']}")

        # Save the full set to repros.json for future reference / issue drafting
        import json
        repros_path = os.path.join(ROOT, "repros.json")
        with open(repros_path, "w") as f:
            json.dump([{"fixture": label, **m} for label, m in all_found], f, indent=2)
        print(f"\nFull set written to {repros_path}")


if __name__ == "__main__":
    main()
