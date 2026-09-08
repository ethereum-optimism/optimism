"""Generate an unproved word-copy claim from the exact compiled runtime.

This emits a claim, never a rewrite rule. Its applicability conditions must be
established by any caller before an independently proved result can be used.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("artifact", type=Path)
parser.add_argument("output", type=Path)
args = parser.parse_args()
artifact = args.artifact.read_bytes()
code = bytes.fromhex(json.loads(artifact)["deployedBytecode"]["object"].removeprefix("0x"))

# Match the complete loop and both destinations, not an incidental opcode prefix.
matches = []
for width in (2, 3):
    push = re.escape(bytes([0x5f + width]))
    immediate = b"(.{" + str(width).encode() + b"})"
    pattern = re.compile(
        rb"\x5b\x83\x81\x10\x15" + push + immediate
        + rb"\x57\x81\x81\x01\x51\x83\x82\x01\x52\x60\x20\x01" + push + immediate + rb"\x56",
        re.DOTALL,
    )
    matches.extend(
        (match, width) for match in pattern.finditer(code)
        if int.from_bytes(match[2], "big") == match.start()
        and int.from_bytes(match[1], "big") == match.end()
        and code[match.end():match.end() + 1] == b"\x5b"
    )
if len(matches) != 1:
    raise ValueError(f"Expected one complete word-copy loop, found {len(matches)}")
match, width = matches[0]
head, end = match.span()
literal = 'b"' + "".join(f"\\x{byte:02x}" for byte in code) + '"'
args.output.write_text(f'''// Generated claim, not an assumed execution summary.
// Artifact SHA256: {hashlib.sha256(artifact).hexdigest()}
// Runtime SHA256: {hashlib.sha256(code).hexdigest()}
requires "foundry.md"

module WITHDRAWAL-COPY-LOOP
    imports FOUNDRY

    claim [word-copy]:
      <k> #execute => #execute ... </k>
      <program> {literal} </program>
      <jumpDests> #computeValidJumpDests({literal}) </jumpDests>
      <pc> {head} => {end} </pc>
      <wordStack> (I => END) : SRC : DEST : LENGTH : WS </wordStack>
      <localMem>
        LM [ DEST := #range(LM, SRC, I) ]
          => LM [ DEST := #range(LM, SRC, END) ]
      </localMem>
      <memoryUsed>
        (#if I ==Int 0 #then MU #else maxInt(MU, (DEST +Int I +Int 31) /Int 32) #fi)
          => (#if END ==Int 0 #then MU #else maxInt(MU, (DEST +Int END +Int 31) /Int 32) #fi)
      </memoryUsed>
      <gas> #gas(G) => #gas(?FINALGAS) </gas>
      <useGas> true </useGas>
      <stackChecks> true </stackChecks>
      <schedule> CANCUN </schedule>
      requires 0 <=Int LENGTH andBool LENGTH <Int 2 ^Int 64
       andBool LENGTH <=Int END andBool END <Int LENGTH +Int 32
       andBool END modInt 32 ==Int 0
       andBool 0 <=Int I andBool I <=Int END andBool I modInt 32 ==Int 0
       andBool 0 <=Int SRC andBool SRC <Int 2 ^Int 256
       andBool 0 <=Int DEST andBool DEST <Int 2 ^Int 256
       andBool SRC +Int END <=Int DEST
       andBool DEST +Int END <=Int 2 ^Int 256
       andBool DEST <=Int lengthBytes(LM)
       andBool 0 <=Int MU
       andBool #sizeWordStack(WS) <=Int 1017
       andBool #sizeWordStack(WS, 3) <Int 1024
       andBool #sizeWordStack(WS, 4) <Int 1024
       andBool #sizeWordStack(WS, 5) <Int 1024
       andBool #sizeWordStack(WS, 6) <Int 1024
      [circularity]
endmodule
''')
print(json.dumps({"artifact": str(args.artifact), "claim": str(args.output),
                  "head": head, "exit": end, "jumpBytes": width,
                  "runtimeSha256": hashlib.sha256(code).hexdigest()}))
