"""Generate an unproved append-copy claim from the exact compiled runtime.

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

def render_claim(name, kind):
    step, entry = kind == "step", kind == "entry"
    # No requires-only endpoint/address aliases: both are expressions in the state.
    extent = "(((LENGTH +Int 31) /Int 32) *Int 32)"
    dest = "lengthBytes(LM)"
    index = "0" if entry else "I"
    final_index = "I +Int 32" if step else "?FINALINDEX"
    control = "(#next [ JUMPDEST ] ~> #execute => #execute)" if step else "#execute => #execute"
    memory_before = "MU" if step or entry else f"#memoryUsageUpdate(MU, {dest} +Int I -Int 32, 32)"
    memory_after = f"#memoryUsageUpdate(MU, I +Int {dest}, 32)" if step else "?FINALMEMORYUSED"
    final_memory = f"#memoryUsageUpdate(MU, {dest} +Int ?FINALINDEX -Int 32, 32)"
    if entry:
        final_memory = f"(#if ?FINALINDEX ==Int 0 #then MU #else {final_memory} #fi)"
    # The independent step is stronger: MU is any integer. In particular it can
    # match the native counter expression of the positive-prefix invariant.
    extra_requires = "       andBool I <Int LENGTH\n" if step else "       andBool 0 <=Int MU\n"
    if kind == "positive":
        extra_requires += "       andBool 0 <Int I\n"
    ensures = "" if step else f"""      ensures {extent} <=Int ?FINALINDEX andBool ?FINALINDEX <=Int {extent}
       andBool ?FINALMEMORYUSED <=Int {final_memory}
       andBool {final_memory} <=Int ?FINALMEMORYUSED
"""
    attributes = {
        "step": "",
        "positive": "      [circularity, depends(WITHDRAWAL-COPY-LOOP.word-copy-append-step)]\n",
        "entry": "      [depends(WITHDRAWAL-COPY-LOOP.word-copy-append-step,WITHDRAWAL-COPY-LOOP.word-copy-append-positive)]\n",
    }[kind]
    return f'''    claim [{name}]:
      <k> {control} ... </k>
      <program> {literal} </program>
      <jumpDests> #computeValidJumpDests({literal}) </jumpDests>
      <pc> {head} => {head if step else end} </pc>
      <wordStack> ({index} => {final_index}) : SRC : {dest} : LENGTH : WS </wordStack>
      <localMem>
        LM +Bytes #range(LM, SRC, {index})
          => LM +Bytes #range(LM, SRC, {final_index})
      </localMem>
      <memoryUsed> {memory_before} => {memory_after} </memoryUsed>
      <gas> #gas(G) => #gas(?FINALGAS) </gas>
      <useGas> true </useGas>
      <stackChecks> true </stackChecks>
      <schedule> CANCUN </schedule>
      requires 0 <=Int LENGTH andBool LENGTH <Int 2 ^Int 64
       andBool LENGTH <=Int {extent} andBool {extent} <Int LENGTH +Int 32
       andBool {extent} modInt 32 ==Int 0
       andBool 0 <=Int {index} andBool {index} <=Int {extent} andBool {index} modInt 32 ==Int 0
       andBool 0 <=Int SRC andBool SRC <Int 2 ^Int 256
       andBool 0 <=Int {dest} andBool {dest} <Int 2 ^Int 256
       andBool SRC +Int {extent} <=Int {dest}
       andBool {dest} +Int {extent} <=Int 2 ^Int 256
       andBool #sizeWordStack(WS) <=Int 1017
       andBool #sizeWordStack(WS, 3) <Int 1024
       andBool #sizeWordStack(WS, 4) <Int 1024
       andBool #sizeWordStack(WS, 5) <Int 1024
       andBool #sizeWordStack(WS, 6) <Int 1024
{extra_requires}{ensures}{attributes}'''

# Separate zero entry from positive-prefix induction; together they retain the
# previous domain and exact counter result without a conditional invariant cell.
args.output.write_text(header + "\n".join((
    render_claim("word-copy-append-step", "step"),
    render_claim("word-copy-append-positive", "positive"),
    render_claim("word-copy-append", "entry"),
)) + "endmodule\n")
print(json.dumps({"artifact": str(args.artifact), "claim": str(args.output),
                  "head": head, "exit": end, "jumpBytes": width,
                  "runtimeSha256": hashlib.sha256(code).hexdigest()}))
