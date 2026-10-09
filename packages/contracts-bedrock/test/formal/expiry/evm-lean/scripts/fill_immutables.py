#!/usr/bin/env python3
"""Fill a contract's immutable references with one value.

Usage: fill_immutables.py RUNTIME_HEX IMMUTABLE_REFERENCES_JSON VALUE

RUNTIME_HEX is the artifact's deployedBytecode.object, in which immutables are zero placeholders.
IMMUTABLE_REFERENCES_JSON is deployedBytecode.immutableReferences ({ast id: [{start, length}]}).
Every reference is overwritten with VALUE as a 32-byte big-endian word, and the filled runtime
code is printed. The script requires exactly one immutable, every reference 32 bytes long and
zero before filling, so a new immutable or a layout change fails loudly.
"""
import json
import sys

hex_path, refs_path, value = sys.argv[1], sys.argv[2], int(sys.argv[3])
code = bytearray.fromhex(open(hex_path).read().strip().removeprefix("0x"))
refs = json.load(open(refs_path))
if len(refs) != 1:
    sys.exit(f"expected exactly one immutable, found {len(refs)}")
(slots,) = refs.values()
if not slots:
    sys.exit("the immutable has no references")
word = value.to_bytes(32, "big")
for ref in slots:
    start, length = ref["start"], ref["length"]
    if length != 32:
        sys.exit(f"immutable reference at {start} is {length} bytes, expected 32")
    if start < 0 or start + 32 > len(code):
        sys.exit(f"immutable reference at {start} lies outside the {len(code)}-byte runtime")
    if any(code[start:start + 32]):
        sys.exit(f"immutable reference at {start} is not a zero placeholder")
    code[start:start + 32] = word
print("0x" + code.hex())
