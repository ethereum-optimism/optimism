#!/usr/bin/env python3
import binascii
from manticore.ethereum import ManticoreEVM

# remove ContractResolver and replace console.log with assert(false);
contract_src=open("../SafetyChecker.sol").read().split("\n")
contract_src = [('' if 'import' in x else x) for x in contract_src]
contract_src = [(x.replace(" is ContractResolver", "") if 'is ContractResolver' in x else x) for x in contract_src]
contract_src = [("" if 'ContractResolver(_addressResolver)' in x else x) for x in contract_src]
contract_src = [('assert(false);' if 'console.log' in x else x) for x in contract_src]
contract_src = '\n'.join(contract_src)

m = ManticoreEVM()
#m.verbosity(3)

user_account = m.create_account(balance=10000000)
contract_account = m.solidity_create_contract(contract_src, owner=user_account, balance=0, args=[0])

# confirm that the only way call can be second byte is with push or stop
value = m.make_symbolic_buffer(1)
#m.constrain(value[0] == 0x5b)

print("running")
contract_account.isBytecodeSafe(value)

print("done", m.count_ready_states(), m.count_terminated_states())
for state in m.ready_states:
  print(binascii.hexlify(state.solve_one(value)))


  print("    ", list(map(binascii.hexlify, state.solve_n(value, 256))))


#m.constrain(value[2] == 0x33)
#m.constrain(value[0] == 0x33)
#m.constrain(value[1] != 0x60)

