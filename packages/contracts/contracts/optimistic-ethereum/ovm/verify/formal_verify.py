#!/usr/bin/env python3
import binascii
import unittest
from manticore.ethereum import ManticoreEVM

def get_contract_src():
  # remove ContractResolver and replace console.log with assert(false);
  with open("../SafetyChecker.sol") as f:
    contract_src = f.read().split("\n")
  contract_src = [('' if 'import' in x else x) for x in contract_src]
  contract_src = [(x.replace(" is ContractResolver", "") if 'is ContractResolver' in x else x) for x in contract_src]
  contract_src = [("" if 'ContractResolver(_addressResolver)' in x else x) for x in contract_src]
  contract_src = [('assert(false);' if 'console.log' in x else x) for x in contract_src]
  contract_src = '\n'.join(contract_src)
  return contract_src

def prepare_evm():
  m = ManticoreEVM()
  user_account = m.create_account(balance=10000000)
  contract_account = m.solidity_create_contract(get_contract_src(),
    owner=user_account, balance=0, args=[0])
  return contract_account, m

class VerifySafetyChecker(unittest.TestCase):
  def test_all_one_byte_contracts_are_whitelisted(self):
    contract_account, m = prepare_evm()

    value = m.make_symbolic_buffer(1)
    #m.constrain(value[0] == 0x5b)

    print("running")
    contract_account.isBytecodeSafe(value)
    print("done", m.count_ready_states(), m.count_terminated_states())

    for state in m.ready_states:
      print(binascii.hexlify(state.solve_one(value)))
      print("    ", list(map(binascii.hexlify, state.solve_n(value, 256))))

  def test_all_bare_calls_follow_push_or_stop(self):
    contract_account, m = prepare_evm()

    # confirm that the only way call can be second byte is with push or stop
    value = m.make_symbolic_buffer(2)
    m.constrain(value[1] == 0xf1)

    print("running")
    contract_account.isBytecodeSafe(value)
    print("done", m.count_ready_states(), m.count_terminated_states())

    for state in m.ready_states:
      print(binascii.hexlify(state.solve_one(value)))
      print("    ", list(map(binascii.hexlify, state.solve_n(value, 256))))


if __name__ == '__main__':
  unittest.main()

