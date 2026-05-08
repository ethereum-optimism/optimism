#!/usr/bin/env python3
"""For each of the residual `kona_more_lenient` corruption cases (loose mode,
post-wrapper-fix), feed the same corrupted brotli payload to all three decoders
and print a side-by-side comparison so we can spot which library is the odd
one out.
"""

import os
import subprocess
import sys

ROOT = os.path.dirname(os.path.abspath(__file__))
KONA = os.path.join(ROOT, "kona_harness/target/release/kona-brotli-harness")
OPNODE = "/tmp/opnode-brotli-harness"
GEN = "/tmp/gen-channel"
RUST_PROBE = os.path.join(ROOT, "spec_check_rust/target/release/spec-check-rust")


def run_capture(cmd, stdin=None):
    p = subprocess.run(cmd, input=stdin, capture_output=True, text=True, check=False)
    return p.stdout, p.stderr, p.returncode


def parse_kv(out):
    d = {}
    for line in out.splitlines():
        if "=" in line:
            k, v = line.split("=", 1)
            d[k] = v
    return d


def gen_channel(n=20):
    out, _, _ = run_capture([GEN, "-n", str(n)])
    return out.strip()


def corrupt_channel(channel_hex, offset, value=0xFF):
    b = bytearray(bytes.fromhex(channel_hex))
    b[offset] ^= value
    return bytes(b).hex()


def find_kona_more_lenient(channel_hex):
    """Run the corruption sweep, return offsets where kona_more_lenient (loose mode)."""
    full_len = len(channel_hex) // 2
    offsets = []
    for off in range(1, full_len):
        c = corrupt_channel(channel_hex, off)
        ko, _, _ = run_capture([KONA], stdin=c)
        oo, _, _ = run_capture([OPNODE], stdin=c)
        kd = parse_kv(ko)
        od = parse_kv(oo)
        try:
            kc = int(kd.get("BATCH_COUNT", "0"))
            oc = int(od.get("BATCH_COUNT", "0"))
        except ValueError:
            continue
        if kc > oc:
            offsets.append(off)
    return offsets


def probe_brotli_payload(brotli_hex):
    """Feed bytes to all three decoders; return (rust_status, rust_bytes, andy_status, andy_bytes, google_status, google_bytes)."""
    # Rust probe: takes hex via argv. For a single value (not truncation sweep),
    # spec_check_rust does a sweep over truncations, so we pad with one extra byte
    # and look at the trunc=full_len row, OR we feed a 1-byte-shorter input (effectively).
    # Simpler: write a one-shot call.
    # For now, run kona's harness which calls the low-level brotli (channel = 01 + payload).
    channel = "01" + brotli_hex
    ko, _, _ = run_capture([KONA], stdin=channel)
    kd = parse_kv(ko)

    oo, _, _ = run_capture([OPNODE], stdin=channel)
    od = parse_kv(oo)

    # Now use spec_check_google for the google/brotli view on the brotli payload only
    gp, _, _ = run_capture(["go", "run", os.path.join(ROOT, "spec_check_google"), brotli_hex])
    # spec_check_google sweeps truncations; full-length row is the first (largest trunc value)
    # but we want to know the result on the FULL payload (no truncation). The probe runs
    # truncations of the input, so we need to call it with brotli_hex as the input and read
    # the row corresponding to len(brotli_hex)/2 - 1 (since it sweeps trunc len-1 down to 1).
    # Simpler: write a one-shot caller.
    google_status, google_bytes = brotli_oneshot_google(brotli_hex)
    rust_status, rust_bytes = brotli_oneshot_rust(brotli_hex)

    return {
        "kona_loose": (kd.get("DECOMPRESS_RESULT"), kd.get("DECOMPRESS_BYTES", "")),
        "kona_strict": (kd.get("STRICT_RESULT"), kd.get("STRICT_BYTES", "")),
        "kona_batches": kd.get("BATCH_COUNT", "0"),
        "opnode": (od.get("DECOMPRESS_RESULT"), od.get("DECOMPRESS_BYTES", "")),
        "opnode_batches": od.get("BATCH_COUNT", "0"),
        "andybalholm": (od.get("DECOMPRESS_RESULT"), od.get("DECOMPRESS_BYTES", "")),
        "google_brotli": (google_status, google_bytes),
        "rust_crate": (rust_status, rust_bytes),
    }


def brotli_oneshot_google(brotli_hex):
    """Decompress full brotli payload using google/brotli, return (status, hex)."""
    code = '''
package main
import (
  "bytes"; "encoding/hex"; "fmt"; "io"; "os"
  "github.com/google/brotli/go/cbrotli"
)
func main() {
  in, _ := hex.DecodeString(os.Args[1])
  br := cbrotli.NewReader(bytes.NewReader(in))
  out, err := io.ReadAll(br)
  br.Close()
  status := "Ok"
  if err != nil { status = err.Error() }
  fmt.Println(status)
  fmt.Println(hex.EncodeToString(out))
}
'''
    # Write to temp file and run
    tmp = "/tmp/_google_oneshot.go"
    with open(tmp, "w") as f:
        f.write(code)
    p, _, _ = run_capture(["go", "run", tmp, brotli_hex])
    lines = p.splitlines()
    return (lines[0] if lines else "<err>", lines[1] if len(lines) > 1 else "")


def brotli_oneshot_rust(brotli_hex):
    """Decompress full brotli payload using Rust brotli crate (high-level Reader)."""
    # spec_check_rust runs a sweep starting at len-1 down to 1. The first row reports trunc=len-1.
    # For the FULL input we need a separate one-shot. Write a tiny binary inline.
    code = '''
use std::io::Read;
fn main() {
    let arg = std::env::args().nth(1).unwrap();
    let bytes = hex::decode(arg).unwrap();
    let mut decoder = brotli::Decompressor::new(&bytes[..], 4096);
    let mut output = Vec::new();
    let result = decoder.read_to_end(&mut output);
    let status = match result {
        Ok(_) => "Ok".to_string(),
        Err(e) => format!("{:?}", e.kind()),
    };
    println!("{}", status);
    let mut hex_out = String::with_capacity(output.len() * 2);
    for b in &output {
        hex_out.push_str(&format!("{:02x}", b));
    }
    println!("{}", hex_out);
}
'''
    # Just reuse spec_check_rust binary — it does a sweep but the trunc=len-1 case
    # might differ from the full input. Actually let's just call the binary with one extra
    # zero-byte appended and look at trunc=original_len:
    p, _, _ = run_capture([RUST_PROBE, brotli_hex])
    # spec_check_rust does sweep from len(input)-1 down to 1. We want the FULL input result.
    # That doesn't exist in its output. Workaround: append a sentinel byte so the sweep
    # includes a row where trunc == original len.
    sentinel = brotli_hex + "00"
    p2, _, _ = run_capture([RUST_PROBE, sentinel])
    target_trunc = len(brotli_hex) // 2  # original length, == sentinel_len - 1
    for line in p2.splitlines():
        parts = line.split()
        if len(parts) < 3:
            continue
        try:
            t = int(parts[0])
        except ValueError:
            continue
        if t == target_trunc:
            status = parts[1]
            n_bytes = int(parts[2])
            return (status, f"({n_bytes} bytes)")
    return ("<not found>", "")


def main():
    print("Generating channel...")
    channel = gen_channel(20)
    print(f"channel: {len(channel)//2} bytes")

    print("Finding kona_more_lenient corruption offsets...")
    offsets = find_kona_more_lenient(channel)
    print(f"Found {len(offsets)} cases: {offsets}")

    if not offsets:
        print("No kona_more_lenient cases — nothing to investigate.")
        return

    print()
    print(f"{'offset':<8} {'rust_crate':<22} {'andybalholm':<22} {'google_cbrotli':<22} {'kona_loose':<15} {'kona_batches':<8} {'opnode_batches':<8}")
    print("-" * 130)

    for off in offsets:
        c = corrupt_channel(channel, off)
        brotli_payload = c[2:]  # strip the 0x01 version byte
        info = probe_brotli_payload(brotli_payload)

        rust_s, rust_b = info["rust_crate"]
        andy_s, andy_b = info["andybalholm"]  # note: this uses op-node's harness output
        goog_s, goog_b = info["google_brotli"]

        # short status
        def short(s):
            if s and len(s) > 18:
                return s[:18] + "…"
            return s or "?"

        rust_n = rust_b.replace("(", "").replace(" bytes)", "") if "(" in rust_b else len(rust_b) // 2
        andy_n = len(andy_b) // 2
        goog_n = len(goog_b) // 2

        kona_loose_s = info["kona_loose"][0]
        kona_n = len(info["kona_loose"][1]) // 2

        print(f"{off:<8} {short(rust_s):<22} {short(andy_s):<22} {short(goog_s):<22} {kona_loose_s:<15} {info['kona_batches']:<8} {info['opnode_batches']:<8}")
        print(f"         rust_bytes={rust_n}  andy_bytes={andy_n}  google_bytes={goog_n}  kona_bytes={kona_n}")


if __name__ == "__main__":
    main()
