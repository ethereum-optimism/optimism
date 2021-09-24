#!/usr/bin/env python3
from elftools.elf.elffile import ELFFile

from hexdump import hexdump
from unicorn import *
from unicorn.mips_const import *

mu = Uc(UC_ARCH_MIPS, UC_MODE_32 + UC_MODE_BIG_ENDIAN)

cnt = 0
def hook_code(uc, address, size, user_data):
  global cnt
  cnt += 1
  if cnt == 1000:
    raise Exception("too many instructions")
  try:
    print(">>> Tracing instruction at 0x%x, instruction size = %u" % (address, size))
    #print('    code hook: pc=%08x sp=%08x' % (
    #  uc.reg_read(UC_MIPS_REG_PC),
    #  uc.reg_read(UC_MIPS_REG_SP)
    #  ))
  except:
    raise Exception("ctrl-c")

elf = open("go-ethereum", "rb")
data = elf.read()
elf.seek(0)

rte = data.find(b"\x08\x02\x2c\x95")
print(hex(rte))

elffile = ELFFile(elf)
#for seg in elffile.iter_segments():
#  print(seg.header, dir(seg.header))
entry = elffile.header.e_entry
print("entrypoint: %x" % entry)


SIZE = 16*1024*1024

mu.mem_map(0, SIZE)
mu.mem_write(0x10000, data)

hexdump(mu.mem_read(entry, 0x10))

mu.reg_write(UC_MIPS_REG_SP, SIZE-4)

mu.hook_add(UC_HOOK_CODE, hook_code, user_data=mu)
mu.emu_start(entry, SIZE)
