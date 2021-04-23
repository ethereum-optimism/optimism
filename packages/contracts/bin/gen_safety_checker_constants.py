#!/usr/bin/env python3
# pip3 install pyevmasm
from pyevmasm import instruction_tables

#print(instruction_tables.keys())

def asm(x):
  return [instruction_tables['istanbul'][i].opcode for i in x]

push_opcodes = asm(["PUSH%d" % i for i in range(1,33)])
stop_opcodes = asm(["STOP", "JUMP", "RETURN", "INVALID"])
caller_opcodes = asm(["CALLER"])
blacklist_ops = set([
  "ADDRESS", "BALANCE", "BLOCKHASH",
  "CALL", "CALLCODE", "CHAINID", "COINBASE",
  "CREATE", "CREATE2", "DELEGATECALL", "DIFFICULTY",
  "EXTCODESIZE", "EXTCODECOPY", "EXTCODEHASH",
  "GASLIMIT", "GASPRICE", "NUMBER",
  "ORIGIN", "REVERT", "SELFBALANCE", "SELFDESTRUCT",
  "SLOAD", "SSTORE", "STATICCALL", "TIMESTAMP"])
whitelist_opcodes = []
for x in instruction_tables['istanbul']:
  if x.name not in blacklist_ops:
    whitelist_opcodes.append(x.opcode)

pushmask = 0
for x in push_opcodes:
  pushmask |= 1 << x

stopmask = 0
for x in stop_opcodes:
  stopmask |= 1 << x

stoplist = [0]*256
procmask = 0
for i in range(256):
  if i in whitelist_opcodes and \
      i not in push_opcodes and \
      i not in stop_opcodes and \
      i not in caller_opcodes:
    # can skip this opcode
    stoplist[i] = 1
  else:
    procmask |= 1 << i

# PUSH1 through PUSH4, can't skip in slow
for i in range(0x60, 0x64):
  stoplist[i] = i-0x5e
rr = "uint256[8] memory opcodeSkippableBytes = [\n"
for i in range(0, 0x100, 0x20):
  ret = "uint256(0x"
  for j in range(i, i+0x20, 1):
    ret += ("%02X" % stoplist[j])
  rr += ret+"),\n"

rr = rr[:-2] + "];"

print(rr)
print("// Mask to gate opcode specific cases")
print("uint256 opcodeGateMask = ~uint256(0x%x);" % procmask)
print("// Halting opcodes")
print("uint256 opcodeHaltingMask = ~uint256(0x%x);" % stopmask)
print("// PUSH opcodes")
print("uint256 opcodePushMask = ~uint256(0x%x);" % pushmask)

