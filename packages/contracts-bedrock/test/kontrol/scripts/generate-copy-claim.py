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
header = f'''// Generated claim, not an assumed execution summary.
// Artifact SHA256: {hashlib.sha256(artifact).hexdigest()}
// Runtime SHA256: {hashlib.sha256(code).hexdigest()}
requires "foundry.md"

module WITHDRAWAL-COPY-LOOP
    imports FOUNDRY

'''

def render_claim(name, step_case=None):
    final_index = "I +Int 32" if step_case else "?FINALINDEX"
    memory_before = "MU" if step_case else "(#if I ==Int 0 #then MU #else maxInt(MU, (DEST +Int I +Int 31) /Int 32) #fi)"
    memory_after = "#memoryUsageUpdate(MU, DEST +Int I, 32)" if step_case else "?FINALMEMORYUSED"
    extra_requires = f"       andBool I <Int LENGTH\n       andBool {step_case}\n" if step_case else ""
    ensures = "" if step_case else """      ensures END <=Int ?FINALINDEX andBool ?FINALINDEX <=Int END
       andBool ?FINALMEMORYUSED <=Int
         (#if ?FINALINDEX ==Int 0 #then MU #else maxInt(MU, (DEST +Int ?FINALINDEX +Int 31) /Int 32) #fi)
       andBool (#if ?FINALINDEX ==Int 0 #then MU #else maxInt(MU, (DEST +Int ?FINALINDEX +Int 31) /Int 32) #fi)
         <=Int ?FINALMEMORYUSED
"""
    attributes = "" if step_case else "      [circularity]\n"
    return f'''    claim [{name}]:
      <k> #execute => #execute ... </k>
      <program> {literal} </program>
      <jumpDests> #computeValidJumpDests({literal}) </jumpDests>
      <pc> {head} => {head if step_case else end} </pc>
      <wordStack> (I => {final_index}) : SRC : DEST : LENGTH : WS </wordStack>
      <localMem>
        LM [ DEST := #range(LM, SRC, I) ]
          => LM [ DEST := #range(LM, SRC, {final_index}) ]
      </localMem>
      <memoryUsed>
        {memory_before}
          => {memory_after}
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
{extra_requires}{ensures}{attributes}'''

# Each finite step is independent until its completed native graph is verified.
cases = {
    "after-end": "lengthBytes(LM) <=Int DEST +Int I",
    "across-end": "DEST +Int I <Int lengthBytes(LM) andBool lengthBytes(LM) <=Int DEST +Int I +Int 32",
    "within-buffer": "DEST +Int I +Int 32 <Int lengthBytes(LM)",
}
args.output.write_text(header + render_claim("word-copy") + "\n" +
                       "\n".join(render_claim("word-copy-step-" + name, case) for name, case in cases.items()) +
                       "endmodule\n")
print(json.dumps({"artifact": str(args.artifact), "claim": str(args.output),
                  "head": head, "exit": end, "jumpBytes": width,
                  "runtimeSha256": hashlib.sha256(code).hexdigest()}))
