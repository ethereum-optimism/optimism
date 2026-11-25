"""Usage: count-pinned-vkey.py ELF VKEY

Prints how many times ELF embeds the bytes32 VKEY, as `pinned_vkey_occurrences` in
op-zk-proposer counts it (digest from `unpack_vkey_bytes32`, crates/range-vkeys).
"""

import re
import sys


def pinned_bytes(vkey: str) -> bytes:
    digits = vkey[2:] if vkey.startswith("0x") else vkey
    if not re.fullmatch(r"[0-9a-fA-F]{64}", digits) or int(digits, 16) >> 248:
        raise ValueError("not a bytes32 SP1 vkey")
    packed = int(digits, 16)
    words = [(packed >> (31 * (7 - i))) & 0x7FFF_FFFF for i in range(8)]
    return b"".join(word.to_bytes(4, "little") for word in words)


def main() -> int:
    if len(sys.argv) != 3:
        print(f"ERROR: {__doc__}", file=sys.stderr)
        return 2
    try:
        needle = pinned_bytes(sys.argv[2])
    except ValueError as err:
        print(f"ERROR: {sys.argv[2]}: {err}", file=sys.stderr)
        return 2
    with open(sys.argv[1], "rb") as elf:
        # Lookahead counts overlapping matches, like Rust's `windows`.
        print(len(re.findall(b"(?=" + re.escape(needle) + b")", elf.read())))
    return 0


if __name__ == "__main__":
    sys.exit(main())
