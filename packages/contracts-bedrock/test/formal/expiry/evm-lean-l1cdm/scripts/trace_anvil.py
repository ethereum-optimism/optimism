#!/usr/bin/env python3
"""Run relayUndeliveredMessage on anvil with mocks and print the opcode-level path.

Used to find the basic-block path of each branch (the `l1cdm_block_<pc>` summaries to chain in
Lean) and the exact call inputs. Needs `anvil` (foundry) on PATH.

    python3 scripts/trace_anvil.py bytecode/L1CrossDomainMessenger.runtime.hex [scenario]

Scenarios: success (default), nointerop, notmessenger, unauthorized, badsender. The script exits
non-zero unless the outcome is the expected one: `success` must succeed and make exactly one
`depositTransaction` call whose input is byte-for-byte `depositCd` of `L1cdmEvm/Spec.lean`
(rebuilt here from the constants below); every other scenario must revert.

Mock constants (keep in sync with `L1cdmEvm/Concrete.lean` and `L1cdmEvm/Spec.lean` when
retargeting): EXPORTER (= Predeploys.UNDELIVERED_MESSAGE_EXPORTER), L2CDM, the storage slots
204/205/207/252/254, msgNonce 7, DEPOSIT_GAS (= Spec.depositGasLimit), EXPIRE_GAS
(= EXPIRE_MESSAGE_GAS_LIMIT), the view selectors in SEL.

The mocks are the same bytecodes as `L1cdmEvm/Concrete.lean`:
  * generic mock: returns the word `SLOAD(selector)` (so each view returns what its storage says),
  * portal mock: like the generic mock, but `depositTransaction` stores keccak256(calldata) at slot 0.
"""
import json
import subprocess
import sys
import time
import urllib.request

RT = open(sys.argv[1]).read().strip()
SCEN = sys.argv[2] if len(sys.argv) > 2 else "success"
PORT = 18546
URL = f"http://127.0.0.1:{PORT}"

GENERIC = "5f3560e01c545f5260205ff3"
PORTAL = "5f3560e01c8063e9e05c4214601657545f5260205ff35b50365f5f37365f205f5500"

SELF = "0x" + "10" * 20      # A's L1CrossDomainMessenger (the code under test)
SC_A = "0x" + "11" * 20      # A's SystemConfig
P_A = "0x" + "12" * 20       # A's OptimismPortal
LB = "0x" + "13" * 20        # A's ETHLockbox
CALLER = "0x" + "20" * 20    # B's L1CrossDomainMessenger (msg.sender)
P_B = "0x" + "21" * 20       # B's portal
SC_B = "0x" + "22" * 20      # B's SystemConfig
EXPORTER = 0x4200000000000000000000000000000000000030
L2TOL2 = 0x4200000000000000000000000000000000000023
DEPOSIT_GAS = 412835
EXPIRE_GAS = 100000
NONCE = 7
L2CDM = 0x4200000000000000000000000000000000000007

SEL = {
    "isFeatureEnabled": 0x47af267b, "portal": 0x6425666b, "systemConfig": 0x33d7e2bd,
    "l1CrossDomainMessenger": 0xa7119869, "ethLockbox": 0xb682c444,
    "authorizedPortals": 0x0fd11077, "xDomainMessageSender": 0x6e296e45,
}


def rpc(method, params):
    req = urllib.request.Request(URL, json.dumps({"jsonrpc": "2.0", "id": 1, "method": method,
                                                  "params": params}).encode(),
                                 {"Content-Type": "application/json"})
    r = json.load(urllib.request.urlopen(req))
    if "error" in r:
        raise RuntimeError(r["error"])
    return r["result"]


def w(x):
    return "0x" + format(x, "064x")


def setup(s):
    rpc("anvil_setCode", [SELF, RT])
    for a in (SC_A, LB, CALLER, P_B, SC_B):
        rpc("anvil_setCode", [a, "0x" + GENERIC])
    rpc("anvil_setCode", [P_A, "0x" + PORTAL])
    st = lambda a, k, v: rpc("anvil_setStorageAt", [a, w(k), w(v)])
    st(SELF, 252, int(P_A, 16)); st(SELF, 254, int(SC_A, 16)); st(SELF, 207, L2CDM)
    st(SELF, 205, NONCE); st(SELF, 204, 0x000000000000000000000000000000000000dEaD)
    st(SC_A, SEL["isFeatureEnabled"], 0 if s == "nointerop" else 1)
    st(CALLER, SEL["portal"], int(P_B, 16))
    st(CALLER, SEL["xDomainMessageSender"], 0x99 if s == "badsender" else EXPORTER)
    st(P_B, SEL["systemConfig"], int(SC_B, 16))
    st(SC_B, SEL["l1CrossDomainMessenger"], 0x98 if s == "notmessenger" else int(CALLER, 16))
    st(P_A, SEL["ethLockbox"], int(LB, 16))
    st(LB, SEL["authorizedPortals"], 0 if s == "unauthorized" else 1)


def expected_deposit(H, t):
    """`depositCd` of L1cdmEvm/Spec.lean: depositTransaction(otherMessenger, 0, gas, false,
    relayMessage(versionedNonce, SELF, L2ToL2CrossDomainMessenger, 0, 100000, expireMessage(H, t)))."""
    ww = lambda x: x.to_bytes(32, "big")
    expire = bytes.fromhex("763a1cb7") + ww(H) + ww(t)
    relay = (bytes.fromhex("d764ad0b") + ww((1 << 240) | NONCE) + ww(int(SELF, 16)) + ww(L2TOL2)
             + ww(0) + ww(EXPIRE_GAS) + ww(0xc0) + ww(len(expire)) + expire + bytes(28))
    return (bytes.fromhex("e9e05c42") + ww(L2CDM) + ww(0) + ww(DEPOSIT_GAS) + ww(0) + ww(0xa0)
            + ww(len(relay)) + relay + bytes(28))


def main():
    anvil = subprocess.Popen(["anvil", "--port", str(PORT), "--hardfork", "cancun", "--silent"])
    try:
        time.sleep(1.5)
        setup(SCEN)
        H, t = 0x1234, 10**9
        data = "0x372293c3" + format(H, "064x") + format(t, "064x")
        tr = rpc("debug_traceCall", [{"from": CALLER, "to": SELF, "data": data, "gas": hex(3_000_000)},
                                     "latest", {"enableMemory": True}])
        print("failed:", tr["failed"], "gas:", tr["gas"])
        logs = tr["structLogs"]
        prev_depth, blocks, jump = None, {}, True
        deposits = []
        for i, l in enumerate(logs):
            d = l["depth"]
            if d != prev_depth:
                jump = True
            if jump or l["op"] == "JUMPDEST":
                blocks.setdefault(d, []).append(l["pc"])
            jump = l["op"] in ("JUMP", "JUMPI", "CALL", "STATICCALL")
            if l["op"] in ("CALL", "STATICCALL", "LOG2", "SSTORE", "RETURN", "REVERT", "STOP"):
                stk = [int(x, 16) for x in l["stack"]][::-1]
                mem = bytes.fromhex("".join(x.removeprefix("0x") for x in l.get("memory", [])))
                info = ""
                if l["op"] in ("CALL", "STATICCALL"):
                    k = 3 if l["op"] == "CALL" else 2
                    io, isz = stk[k], stk[k + 1]
                    info = f"to={stk[1]:#x} in=[{io:#x}+{isz}] {mem[io:io+isz].hex()}"
                    if l["op"] == "CALL":
                        info = f"value={stk[2]} " + info
                        if stk[1] == int(P_A, 16) and mem[io:io + 4].hex() == "e9e05c42":
                            deposits.append(mem[io:io + isz])
                elif l["op"] == "LOG2":
                    info = f"data=[{stk[0]:#x}+{stk[1]}] topics={stk[2]:#x},{stk[3]:#x}"
                elif l["op"] == "SSTORE":
                    info = f"slot={stk[0]:#x} val={stk[1]:#x}"
                print(f"  d{d} pc={l['pc']} {l['op']} {info}")
            prev_depth = d
        for d, b in sorted(blocks.items()):
            print(f"depth {d} blocks ({len(b)}):", b)
        if SCEN == "success":
            exp = expected_deposit(H, t)
            ok = (not tr["failed"]) and deposits == [exp]
            print(f"deposit calls: {len(deposits)}; input == depositCd ({len(exp)} bytes):",
                  bool(deposits) and deposits[0] == exp)
        else:
            ok = tr["failed"]
        if not ok:
            print(f"UNEXPECTED OUTCOME for scenario {SCEN}", file=sys.stderr)
            sys.exit(1)
        print(f"OK: scenario {SCEN} behaves as expected")
    finally:
        anvil.terminate()


if __name__ == "__main__":
    main()
