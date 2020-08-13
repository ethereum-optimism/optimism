#!/usr/bin/env python3
import binascii
import unittest
from manticore.ethereum import ManticoreEVM
from manticore.core.smtlib.expression import BoolOr

from gen_safety_checker_constants import whitelist_opcodes, blacklist_opcodes, push_opcodes, stop_opcodes

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

def print_and_get_solves(m, value):
  all_solves = []
  for state in m.ready_states:
    print(binascii.hexlify(state.solve_one(value)))
    solves = state.solve_n(value, 256)
    all_solves += solves
    print("    ", list(map(binascii.hexlify, solves)))
  return all_solves

class VerifySafetyChecker(unittest.TestCase):
  def test_all_one_byte_contracts_are_whitelisted(self):
    contract_account, m = prepare_evm()

    value = m.make_symbolic_buffer(1)
    #m.constrain(value[0] == 0x5b)

    print("running")
    contract_account.isBytecodeSafe(value)
    print("done", m.count_ready_states(), m.count_terminated_states())

    all_solves = print_and_get_solves(m, value)

    # all single bytes are whitelisted opcodes
    for x in all_solves:
      assert(x[0] in whitelist_opcodes)

    # all whitelisted opcodes should be allowed as a single byte too
    valid_ops = [x[0] for x in all_solves]
    for x in whitelist_opcodes:
      assert(x in valid_ops)

  def test_all_blacklisted_ops_follow_push_or_stop(self):
    contract_account, m = prepare_evm()

    # confirm that the only way call can be second byte is with push or stop
    value = m.make_symbolic_buffer(2)

    # all blacklisted op
    cc = value[1] == blacklist_opcodes[0]
    for bop in blacklist_opcodes[1:]:
      cc = BoolOr(cc, value[1] == bop)
    m.constrain(cc)

    print("running")
    contract_account.isBytecodeSafe(value)
    print("done", m.count_ready_states(), m.count_terminated_states())

    all_solves = print_and_get_solves(m, value)

    # all first bytes are pushes or stops
    for x in all_solves:
      assert(x[0] in push_opcodes or x[0] in stop_opcodes)

  def test_all_caller_followers_are_the_allowed_string(self):
    contract_account, m = prepare_evm()
    value = m.make_symbolic_buffer(7)
    m.constrain(value[0] == 0x33)
    contract_account.isBytecodeSafe(value)
    all_solves = print_and_get_solves(m, value)

    assert(len(all_solves) > 0)
    for x in all_solves:
      assert(x[0:6] == b"\x33\x60\x00\x90\x5a\xf1")

  def test_all_caller_followers_must_be_the_allowed_string(self):
    contract_account, m = prepare_evm()
    value = m.make_symbolic_buffer(7)
    m.constrain(value[0] == 0x33)

    m.constrain(BoolOr(BoolOr(value[1] != 0x60, value[2] != 0x00),
                BoolOr(BoolOr(value[3] != 0x90, value[4] != 0x5a),
                value[5] != 0xf1)))
    contract_account.isBytecodeSafe(value)
    all_solves = print_and_get_solves(m, value)
    assert(len(all_solves) == 0)

if __name__ == '__main__':
  unittest.main()



