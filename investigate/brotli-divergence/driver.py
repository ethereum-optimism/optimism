#!/usr/bin/env python3
"""Differential driver: feed truncated/corrupted brotli channels to both
harnesses and report any divergence."""

import os
import subprocess
import sys

KONA = os.path.join(os.path.dirname(__file__), "kona_harness/target/release/kona-brotli-harness")
OPNODE = "/tmp/opnode-brotli-harness"
GEN = "/tmp/gen-channel"


def run(binary, hex_input):
    p = subprocess.run([binary], input=hex_input, capture_output=True, text=True, timeout=60)
    out = {}
    for line in p.stdout.splitlines():
        if "=" in line:
            k, v = line.split("=", 1)
            out[k] = v
    return out, p.returncode, p.stderr


def gen_channel(n=3):
    p = subprocess.run([GEN, "-n", str(n)], capture_output=True, text=True, check=True)
    return p.stdout.strip()


def parse_hex(s):
    return bytes.fromhex(s)


def to_hex(b):
    return b.hex()


def truncate(channel_hex, byte_len):
    """Keep `byte_len` bytes total (including the leading 0x01 version byte)."""
    b = parse_hex(channel_hex)
    return to_hex(b[:byte_len])


def corrupt(channel_hex, offset, value=0xFF):
    """Flip a single byte in the brotli payload (offset is into the full channel)."""
    b = bytearray(parse_hex(channel_hex))
    b[offset] ^= value
    return to_hex(bytes(b))


def classify(kona, opnode):
    """Return a category label for the divergence."""
    kc = int(kona.get("BATCH_COUNT", "0"))
    oc = int(opnode.get("BATCH_COUNT", "0"))
    if kc == oc and kona.get("DECOMPRESS_BYTES") == opnode.get("DECOMPRESS_BYTES"):
        return None  # agreement
    if kc > oc:
        return "kona_more_lenient"  # the original issue's direction
    if kc < oc:
        return "opnode_more_lenient"
    # batch counts equal but bytes differ — observable only if downstream cares
    return "bytes_differ_only"


def classify_strict(kona, opnode):
    """Compare strict-mode results between sides. Both ok with same bytes = agree;
    both err = agree (both reject); otherwise = divergence."""
    kr = kona.get("STRICT_RESULT")
    or_ = opnode.get("STRICT_RESULT")
    if kr == "err" and or_ == "err":
        return None  # both reject — parity
    if kr == "ok" and or_ == "ok":
        if kona.get("STRICT_BYTES") == opnode.get("STRICT_BYTES"):
            return None  # both accept same bytes — parity
        return "strict_bytes_differ"
    if kr == "ok" and or_ == "err":
        return "strict_kona_accepts_opnode_rejects"
    return "strict_opnode_accepts_kona_rejects"


def diff_results(label, kona, opnode, verbose=True):
    cat = classify(kona, opnode)
    cat_strict = classify_strict(kona, opnode)
    if verbose and cat is not None:
        print(f"DIVERGENCE [{cat}] @ {label}")
    return cat, cat_strict


def main():
    n_batches = int(os.environ.get("N_BATCHES", "3"))
    full = gen_channel(n=n_batches)
    full_len = len(full) // 2
    print(f"Full channel: {full_len} bytes, {full}")

    # Sanity: full input must agree.
    k, _, _ = run(KONA, full)
    o, _, _ = run(OPNODE, full)
    cat, cat_strict = diff_results("FULL", k, o, verbose=False)
    assert cat is None and cat_strict is None, f"harnesses disagree on full input: loose={cat} strict={cat_strict}"
    print(f"Sanity OK: full channel produces BATCH_COUNT={k['BATCH_COUNT']} on both sides (loose + strict).")
    print()

    def sweep(name, generate):
        print(f"=== {name} ===")
        loose_cats = {}
        strict_cats = {}
        strict_examples = {}
        for label, hex_in in generate():
            k, _, _ = run(KONA, hex_in)
            o, _, _ = run(OPNODE, hex_in)
            cat, cat_strict = diff_results(label, k, o, verbose=False)
            loose_cats[cat] = loose_cats.get(cat, 0) + 1
            strict_cats[cat_strict] = strict_cats.get(cat_strict, 0) + 1
            if cat_strict and cat_strict not in strict_examples:
                strict_examples[cat_strict] = (label, k, o)
        print(f"Loose categories:  {loose_cats}")
        print(f"Strict categories: {strict_cats}")
        for cat, (label, k, o) in strict_examples.items():
            print(f"--- strict example [{cat}] @ {label} ---")
            print(f"  kona:   loose={k.get('DECOMPRESS_RESULT')} bc={k.get('BATCH_COUNT')} | strict={k.get('STRICT_RESULT')} strict_bytes_len={len(k.get('STRICT_BYTES','') or '')//2}")
            print(f"  opnode: loose={o.get('DECOMPRESS_RESULT')} bc={o.get('BATCH_COUNT')} | strict={o.get('STRICT_RESULT')} strict_bytes_len={len(o.get('STRICT_BYTES','') or '')//2}")
        print()

    sweep("Truncation sweep", lambda: ((f"truncate len={n}", truncate(full, n)) for n in range(2, full_len)))
    sweep("Single-byte corruption sweep", lambda: ((f"corrupt offset={off}", corrupt(full, off)) for off in range(1, full_len)))


if __name__ == "__main__":
    main()
