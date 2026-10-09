#!/usr/bin/env python3
"""Concrete mini-EVM tracer used to find the basic-block path of each branch of a function.

Supports only the opcodes on the expireMessage paths (extend as needed). Prints, per scenario,
the static calls, SSTOREs, the terminal (STOP/REVERT pc) and the list of block entry pcs; the
block pcs are the `l2tol2_block_<pc>` summaries to chain in Lean (see HOWTO.md).
Usage: python3 scripts/trace_paths.py bytecode/L2ToL2CrossDomainMessenger.runtime.hex
Needs `cast` (foundry) for keccak.
"""
import subprocess
import sys

code=bytes.fromhex(open(sys.argv[1]).read().strip()[2:])
M=2**256
def run(calldata, caller, value, rets, sloads, show=True):
    pc=0; st=[]; mem=bytearray(); rets=list(rets); rd=b''; storage=dict(sloads); blocks=[]; prev_jump=True
    def mext(n):
        nonlocal mem
        if len(mem)<n: mem+=bytes(((n+31)//32)*32-len(mem))
    while True:
        op=code[pc]
        if prev_jump or op==0x5b: blocks.append(pc)
        prev_jump=False
        if 0x60<=op<=0x7f:
            n=op-0x5f; st.append(int.from_bytes(code[pc+1:pc+1+n],'big')); pc+=1+n; continue
        name=None
        if op==0x5f: st.append(0)
        elif op==0x52: a=st.pop(); v=st.pop(); mext(a+32); mem[a:a+32]=v.to_bytes(32,'big')
        elif op==0x51: a=st.pop(); mext(a+32); st.append(int.from_bytes(mem[a:a+32],'big'))
        elif op==0x36: st.append(len(calldata))
        elif op==0x35: a=st.pop(); st.append(int.from_bytes((calldata+bytes(64))[a:a+32],'big'))
        elif op==0x10: a=st.pop(); b=st.pop(); st.append(int(a<b))
        elif op==0x11: a=st.pop(); b=st.pop(); st.append(int(a>b))
        elif op==0x12:
            a=st.pop(); b=st.pop(); sa=a-M if a>=2**255 else a; sb=b-M if b>=2**255 else b; st.append(int(sa<sb))
        elif op==0x14: a=st.pop(); b=st.pop(); st.append(int(a==b))
        elif op==0x15: a=st.pop(); st.append(int(a==0))
        elif op==0x16: a=st.pop(); b=st.pop(); st.append(a&b)
        elif op==0x17: a=st.pop(); b=st.pop(); st.append(a|b)
        elif op==0x19: a=st.pop(); st.append((~a)%M)
        elif op==0x1b: s=st.pop(); v=st.pop(); st.append((v<<s)%M)
        elif op==0x1c: s=st.pop(); v=st.pop(); st.append(v>>s)
        elif op==0x01: a=st.pop(); b=st.pop(); st.append((a+b)%M)
        elif op==0x03: a=st.pop(); b=st.pop(); st.append((a-b)%M)
        elif op==0x34: st.append(value)
        elif op==0x33: st.append(caller)
        elif op==0x5a: st.append(10**6)
        elif op==0x3d: st.append(len(rd))
        elif op==0x3e: d=st.pop(); o=st.pop(); n=st.pop(); mext(d+n); mem[d:d+n]=rd[o:o+n]
        elif op==0xfa:
            g=st.pop(); t=st.pop(); io=st.pop(); isz=st.pop(); oo=st.pop(); osz=st.pop()
            mext(io+isz); inp=bytes(mem[io:io+isz]); ok,data=rets.pop(0); rd=data; mext(oo+osz); n=min(osz,len(data)); mem[oo:oo+n]=data[:n]; st.append(int(ok))
            if show: print(f"  STATICCALL@{pc} to {t:#x} in=mem[{io:#x}:{io+isz:#x}]={inp.hex()} -> {ok}")
        elif op==0x20:
            a=st.pop(); n=st.pop(); mext(a+n)
            h=subprocess.run(['cast','keccak','0x'+bytes(mem[a:a+n]).hex()],capture_output=True,text=True).stdout.strip()
            st.append(int(h,16))
        elif op==0x54: a=st.pop(); st.append(storage.get(a,0))
        elif op==0x55: a=st.pop(); v=st.pop(); storage[a]=v; print(f"  SSTORE {a:#x} := {v:#x}")
        elif op==0xa2: st.pop(); st.pop(); st.pop(); st.pop(); print("  LOG2")
        elif op==0x56: pc=st.pop(); prev_jump=True; continue
        elif op==0x57:
            d=st.pop(); c=st.pop()
            if c: pc=d; prev_jump=True; continue
            prev_jump=True
        elif op==0x5b: pass
        elif op==0x50: st.pop()
        elif 0x80<=op<=0x8f: st.append(st[-(op-0x7f)])
        elif 0x90<=op<=0x9f: n=op-0x8f; st[-1],st[-1-n]=st[-1-n],st[-1]
        elif op==0x00: print("STOP"); break
        elif op==0xfd: print("REVERT at",pc); break
        else: print("unknown op",hex(op),"at",pc); break
        pc+=1
    print("blocks:", blocks)
    return storage
H=0x1234; t=10**9
cd=bytes.fromhex('763a1cb7')+H.to_bytes(32,'big')+t.to_bytes(32,'big')
L2CDM=0x4200000000000000000000000000000000000007
X=0xabcd
enc=X.to_bytes(32,'big')
import subprocess
slot3=int(subprocess.run(['cast','index','bytes32','0x'+H.to_bytes(32,'big').hex(),'3'],capture_output=True,text=True).stdout.strip(),16)
print("success:"); run(cd,L2CDM,0,[(1,enc),(1,enc)],{slot3:5})
print("bad caller:"); run(cd,0x99,0,[],{slot3:5})
print("mismatch:"); run(cd,L2CDM,0,[(1,enc),(1,(X+1).to_bytes(32,'big'))],{slot3:5})
print("not sent:"); run(cd,L2CDM,0,[(1,enc),(1,enc)],{})
print("too early:"); run(cd,L2CDM,0,[(1,enc),(1,enc)],{slot3:t})
print("overflow:"); run(cd,L2CDM,0,[(1,enc),(1,enc)],{slot3:M-1})
print("call1 fails:"); run(cd,L2CDM,0,[(0,b'')],{slot3:5})
print("short ret:"); run(cd,L2CDM,0,[(1,b'\x00'*31)],{slot3:5})
print("dirty addr:"); run(cd,L2CDM,0,[(1,(2**160+1).to_bytes(32,'big'))],{slot3:5})
print("value:"); run(cd,L2CDM,1,[],{slot3:5})
print("short cd:"); run(cd[:67],L2CDM,0,[],{slot3:5})
slot4=int(subprocess.run(['cast','index','bytes32','0x'+H.to_bytes(32,'big').hex(),'4'],capture_output=True,text=True).stdout.strip(),16)
print("already expired:"); run(cd,L2CDM,0,[(1,enc),(1,enc)],{slot3:t,slot4:1})
