#!/usr/bin/env python3
"""Concrete mini-EVM tracer for SuperchainETHBridge.refundETH (finds the block path of each branch).

Supports the opcodes on the refundETH paths. Prints, per scenario, the external calls (with exact
input bytes), the keccak inputs, SSTOREs, the CREATE (value and init code), the terminal pc and the
list of block entry pcs; the block pcs are the `ethbridge_block_<pc>` summaries to chain in Lean.

Usage: python3 scripts/trace_paths.py bytecode/SuperchainETHBridge.runtime.hex [--ops]
Needs `cast` (foundry) for keccak.
"""
import subprocess
import sys

code = bytes.fromhex(open(sys.argv[1]).read().strip().removeprefix("0x"))
SHOW_OPS = "--ops" in sys.argv
M = 2**256
SELF = 0x4200000000000000000000000000000000000024
L2L2 = 0x4200000000000000000000000000000000000023
ETHLIQ = 0x4200000000000000000000000000000000000025


def keccak(b: bytes) -> int:
    h = subprocess.run(["cast", "keccak", "0x" + b.hex()], capture_output=True, text=True).stdout.strip()
    return int(h, 16)


def run(calldata, caller, value, rets, storage, chainid=10, extcode=1, create_ok=True):
    pc = 0; st = []; mem = bytearray(); rets = list(rets); rd = b""; storage = dict(storage)
    blocks = []; prev_jump = True

    def mext(n):
        nonlocal mem
        if len(mem) < n:
            mem += bytes(((n + 31) // 32) * 32 - len(mem))

    while True:
        op = code[pc]
        if prev_jump or op == 0x5B:
            blocks.append(pc)
        prev_jump = False
        if SHOW_OPS:
            print(f"    {pc:5d} {op:02x} stack={[hex(x) for x in st[-6:]]}")
        if 0x60 <= op <= 0x7F:
            n = op - 0x5F; st.append(int.from_bytes(code[pc + 1:pc + 1 + n], "big")); pc += 1 + n; continue
        if op == 0x5F: st.append(0)
        elif op == 0x52: a = st.pop(); v = st.pop(); mext(a + 32); mem[a:a + 32] = v.to_bytes(32, "big")
        elif op == 0x51: a = st.pop(); mext(a + 32); st.append(int.from_bytes(mem[a:a + 32], "big"))
        elif op == 0x36: st.append(len(calldata))
        elif op == 0x35: a = st.pop(); st.append(int.from_bytes((calldata + bytes(64))[a:a + 32], "big"))
        elif op == 0x10: a = st.pop(); b = st.pop(); st.append(int(a < b))
        elif op == 0x11: a = st.pop(); b = st.pop(); st.append(int(a > b))
        elif op == 0x12:
            a = st.pop(); b = st.pop(); sa = a - M if a >= 2**255 else a; sb = b - M if b >= 2**255 else b
            st.append(int(sa < sb))
        elif op == 0x13:
            a = st.pop(); b = st.pop(); sa = a - M if a >= 2**255 else a; sb = b - M if b >= 2**255 else b
            st.append(int(sa > sb))
        elif op == 0x14: a = st.pop(); b = st.pop(); st.append(int(a == b))
        elif op == 0x15: a = st.pop(); st.append(int(a == 0))
        elif op == 0x16: a = st.pop(); b = st.pop(); st.append(a & b)
        elif op == 0x17: a = st.pop(); b = st.pop(); st.append(a | b)
        elif op == 0x19: a = st.pop(); st.append((~a) % M)
        elif op == 0x1B: s = st.pop(); v = st.pop(); st.append((v << s) % M)
        elif op == 0x1C: s = st.pop(); v = st.pop(); st.append(v >> s)
        elif op == 0x01: a = st.pop(); b = st.pop(); st.append((a + b) % M)
        elif op == 0x03: a = st.pop(); b = st.pop(); st.append((a - b) % M)
        elif op == 0x34: st.append(value)
        elif op == 0x33: st.append(caller)
        elif op == 0x30: st.append(SELF)
        elif op == 0x46: st.append(chainid)
        elif op == 0x5A: st.append(10**6)
        elif op == 0x3D: st.append(len(rd))
        elif op == 0x3B: a = st.pop(); st.append(extcode); print(f"  EXTCODESIZE@{pc} {a:#x}")
        elif op == 0x39:
            d = st.pop(); o = st.pop(); n = st.pop(); mext(d + n); mem[d:d + n] = (code + bytes(n))[o:o + n]
            print(f"  CODECOPY@{pc} mem[{d:#x}:{d + n:#x}] := code[{o:#x}:{o + n:#x}]")
        elif op == 0x3E: d = st.pop(); o = st.pop(); n = st.pop(); mext(d + n); mem[d:d + n] = rd[o:o + n]
        elif op in (0xFA, 0xF1):
            g = st.pop(); t = st.pop(); v = st.pop() if op == 0xF1 else 0
            io = st.pop(); isz = st.pop(); oo = st.pop(); osz = st.pop()
            mext(io + isz); inp = bytes(mem[io:io + isz]); ok, data = rets.pop(0); rd = data
            mext(oo + osz); n = min(osz, len(data)); mem[oo:oo + n] = data[:n]; st.append(int(ok))
            print(f"  {'STATICCALL' if op == 0xFA else 'CALL'}@{pc} to {t:#x} value={v} "
                  f"in=mem[{io:#x}:{io + isz:#x}]={inp.hex()} out=[{oo:#x}+{osz}] -> {ok}")
        elif op == 0xF0:
            v = st.pop(); o = st.pop(); n = st.pop(); mext(o + n); init = bytes(mem[o:o + n])
            print(f"  CREATE@{pc} value={v} init=mem[{o:#x}:{o + n:#x}] ({n} bytes)={init.hex()}")
            st.append(0xC0FFEE if create_ok else 0); rd = b""
        elif op == 0x20:
            a = st.pop(); n = st.pop(); mext(a + n)
            print(f"  KECCAK@{pc} mem[{a:#x}:{a + n:#x}] ({n} bytes)={bytes(mem[a:a + n]).hex()}")
            st.append(keccak(bytes(mem[a:a + n])))
        elif op == 0x54: a = st.pop(); st.append(storage.get(a, 0))
        elif op == 0x55: a = st.pop(); v = st.pop(); storage[a] = v; print(f"  SSTORE@{pc} {a:#x} := {v:#x}")
        elif 0xA0 <= op <= 0xA4:
            o = st.pop(); n = st.pop(); topics = [st.pop() for _ in range(op - 0xA0)]
            print(f"  LOG{op - 0xA0}@{pc} topics={[hex(t) for t in topics]} data={bytes(mem[o:o + n]).hex()}")
        elif op == 0x56: pc = st.pop(); prev_jump = True; continue
        elif op == 0x57:
            d = st.pop(); c = st.pop()
            if c: pc = d; prev_jump = True; continue
            prev_jump = True
        elif op == 0x5B: pass
        elif op == 0x50: st.pop()
        elif 0x80 <= op <= 0x8F: st.append(st[-(op - 0x7F)])
        elif 0x90 <= op <= 0x9F: n = op - 0x8F; st[-1], st[-1 - n] = st[-1 - n], st[-1]
        elif op == 0x00: print(f"  STOP at {pc}"); break
        elif op == 0xFD: print(f"  REVERT at {pc}"); break
        elif op == 0xFE: print(f"  INVALID at {pc}"); break
        else: print("  unknown op", hex(op), "at", pc); break
        pc += 1
    print("  blocks:", blocks)
    return storage


def w(x): return x.to_bytes(32, "big")


DEST, NONCE, FROM, TO, AMT = 901, 7, 0xF0F0, 0x7070, 10**18
cd = bytes.fromhex("e17a776b") + w(DEST) + w(NONCE) + w(FROM) + w(TO) + w(AMT)
T, F = w(1), w(0)
if __name__ == "__main__":
    print("success:"); run(cd, 0x99, 0, [(1, T), (1, b"")], {})
    print("not expired:"); run(cd, 0x99, 0, [(1, F)], {})
    refunded_slot = None
    print("expired call fails:"); run(cd, 0x99, 0, [(0, b"")], {})
    print("dirty bool:"); run(cd, 0x99, 0, [(1, w(2))], {})
    print("short return:"); run(cd, 0x99, 0, [(1, b"\x01")], {})
    print("mint fails:"); run(cd, 0x99, 0, [(1, T), (0, b"")], {})
    print("no ETHLiquidity code:"); run(cd, 0x99, 0, [(1, T)], {}, extcode=0)
    print("create fails:"); run(cd, 0x99, 0, [(1, T), (1, b"")], {}, create_ok=False)
    print("value:"); run(cd, 0x99, 1, [], {})
    print("short cd:"); run(cd[:163], 0x99, 0, [], {})
    print("dirty from:"); run(cd[:68] + w(2**160 + 1) + cd[100:], 0x99, 0, [], {})
