#!/usr/bin/env python3
import os
import sys
import binascii
import struct
from termcolor import colored
from unicorn import *
from unicorn.mips_const import *
mu = Uc(UC_ARCH_MIPS, UC_MODE_32 + UC_MODE_BIG_ENDIAN)
from capstone import *
md = Cs(CS_ARCH_MIPS, CS_MODE_32 + CS_MODE_BIG_ENDIAN)

# heap (256 MB) @ 0x20000000
heap_start = 0x20000000 # 0x20000000-0x30000000
# brk (1024 MB) @ 0x40000000
brk_start = 0x40000000  # 0x40000000-0x80000000

has_input_oracle = False

def hook_interrupt(uc, intno, user_data):
  global heap_start
  if intno != 17:
    print("interrupt", intno)
    raise unicorn.UcError(0)
  syscall_no = uc.reg_read(UC_MIPS_REG_V0)

  """
  pc = uc.reg_read(UC_MIPS_REG_PC)
  inst = struct.unpack(">I", uc.mem_read(pc, 4))[0]
  print("syscall %d at %x" % (syscall_no, pc-4))
  """

  v0 = 0
  if syscall_no == 4020:
    oracle_hash = binascii.hexlify(uc.mem_read(0x30001000, 0x20)).decode('utf-8')
    try:
      dat = open("/tmp/eth/0x"+oracle_hash, "rb").read()
      #print("oracle:", oracle_hash, len(dat))
      uc.mem_write(0x31000000, struct.pack(">I", len(dat)))
      uc.mem_write(0x31000004, dat)
    except FileNotFoundError:
      # oracle not found
      uc.mem_write(0x31000000, struct.pack(">I", 0))
  elif syscall_no == 4004:
    # write
    fd = uc.reg_read(UC_MIPS_REG_A0)
    buf = uc.reg_read(UC_MIPS_REG_A1)
    count = uc.reg_read(UC_MIPS_REG_A2)
    os.write(fd, uc.mem_read(buf, count))
  elif syscall_no == 4090:
    a0 = uc.reg_read(UC_MIPS_REG_A0)
    sz = uc.reg_read(UC_MIPS_REG_A1)
    if a0 == 0:
      v0 = heap_start
      heap_start += sz
    else:
      v0 = a0
    print("malloced", hex(v0), hex(a0), hex(sz))
  elif syscall_no == 4045:
    v0 = brk_start
    print("brk", hex(v0))
  elif syscall_no == 4120:
    v0 = 1
    print("clone not supported")

  uc.reg_write(UC_MIPS_REG_V0, v0)
  uc.reg_write(UC_MIPS_REG_A3, 0)

mu.hook_add(UC_HOOK_INTR, hook_interrupt)

# load memory
dat = open("/tmp/minigeth.bin", "rb").read()
mu.mem_map(0, len(dat))
mu.mem_write(0, dat)

# heap (256 MB) @ 0x20000000
# oracle @ 0x30000000
# brk @ 0x40000000
mu.mem_map(heap_start, 0x60000000)
if len(sys.argv) > 1:
  inputs = open("/tmp/eth/"+sys.argv[1], "rb").read()
  mu.mem_write(0x30000000, inputs)

def hook_mem_invalid(uc, access, address, size, value, user_data):
  global has_input_oracle
  pc = uc.reg_read(UC_MIPS_REG_PC)
  if pc == 0xdead0000:
    compare_hash = binascii.hexlify(mu.mem_read(0x30000800, 0x20))
    print("compare", compare_hash)
    os._exit(0)
  print("UNMAPPED MEMORY:", access, hex(address), size, "at", hex(pc))
  return False
mu.hook_add(UC_HOOK_MEM_FETCH_UNMAPPED, hook_mem_invalid)

gtf = open("/tmp/gethtrace")

# tracer
step = 0
def hook_code_simple(uc, address, size, user_data):
  global step
  pc = uc.reg_read(UC_MIPS_REG_PC)
  assert address == pc
  assert size == 4

  inst = struct.unpack(">I", uc.mem_read(pc, 4))[0]
  regs = []
  # starting at AT
  for i in range(3,12):
    regs.append(uc.reg_read(i))

  rr = ' '.join(["%08X" % x for x in regs])
  ss = "%7d %8X %08X : " % (step, pc, inst) + rr
  sgt = gtf.readline().strip("\n")
  if ss != sgt:
    dat = next(md.disasm(uc.mem_read(address, size), address))
    print(dat)
    print(colored(ss, 'green'))
    print(colored(sgt, 'red'))
    os._exit(0)
  else:
    if step % 1000 == 0:
      print(ss)

  step += 1

mu.hook_add(UC_HOOK_CODE, hook_code_simple)

mu.emu_start(0, -1)
