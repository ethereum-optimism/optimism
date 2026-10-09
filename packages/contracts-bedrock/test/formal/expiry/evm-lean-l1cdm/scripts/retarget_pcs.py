#!/usr/bin/env python3
"""Rename pcs in the trace files after a layout-changing retarget.

    python3 scripts/retarget_pcs.py <old-runtime.hex> <new-runtime.hex> [--dry-run]

Aligns the two disassemblies (`cast disassemble`) on their opcode sequences, ignoring PUSH2
operands (jump targets move with the layout), and rewrites, in L1cdmEvm/{Outer*,Inner*,Tails,Spec}.lean,
every `l1cdm_block_<pc>` name and every decimal literal that is an old instruction pc at or after
the first moved pc, through the old->new pc map. Prints the shift segments. Stack-shape changes
(an extra local, ...) are not handled: fix those by hand, then `lake build`. Run it exactly once
per retarget, on the trace files of the old target (it is not idempotent).
Needs `cast` (foundry) on PATH.
"""
import difflib
import glob
import os
import re
import subprocess
import sys


def disasm(path):
    hexcode = open(path).read().strip()
    out = subprocess.run(["cast", "disassemble", hexcode], check=True, capture_output=True,
                         text=True).stdout
    ins = []
    for line in out.splitlines():
        if ": " not in line:
            continue
        pc, rest = line.strip().split(": ", 1)
        ins.append((int(pc, 16), re.sub(r"PUSH2 0x[0-9a-f]+", "PUSH2 J", rest)))
    return ins


def main():
    old, new = disasm(sys.argv[1]), disasm(sys.argv[2])
    dry = "--dry-run" in sys.argv
    sm = difflib.SequenceMatcher(a=[x[1] for x in old], b=[x[1] for x in new], autojunk=False)
    pcmap = {}
    for a, b, size in sm.get_matching_blocks():
        for k in range(size):
            pcmap[old[a + k][0]] = new[b + k][0]
    moved = [pc for pc in sorted(pcmap) if pcmap[pc] != pc]
    if not moved:
        print("no pc moved")
        return
    first = moved[0]
    prev = None
    for pc in sorted(pcmap):
        if pcmap[pc] - pc != prev:
            prev = pcmap[pc] - pc
            print(f"from old pc {pc}: shift {prev:+d}")
    unmatched = len(old) - len(pcmap)
    print(f"{unmatched} old instructions unmatched (inserted/changed code)")
    here = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "L1cdmEvm")
    files = sorted(glob.glob(os.path.join(here, "Outer*.lean")) + glob.glob(os.path.join(here, "Inner*.lean"))
                   + [os.path.join(here, "Tails.lean"), os.path.join(here, "Spec.lean")])
    top = max(pcmap)
    for f in files:
        s = open(f).read()
        t = re.sub(r"l1cdm_block_(\d+)", lambda m: f"l1cdm_block_{pcmap.get(int(m.group(1)), int(m.group(1)))}", s)

        def lit(m):
            n = int(m.group(1))
            return str(pcmap[n]) if first <= n <= top and n in pcmap else m.group(1)
        t = re.sub(r"(?<![\w.])(\d+)(?![\w.])", lit, t)
        if t != s:
            print(("would rewrite " if dry else "rewrote ") + os.path.relpath(f))
            if not dry:
                open(f, "w").write(t)


if __name__ == "__main__":
    main()
