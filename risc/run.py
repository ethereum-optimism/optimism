#!/usr/bin/env python3
import os
import struct
from elftools.elf.elffile import ELFFile

from hexdump import hexdump
from unicorn import *
from unicorn.mips_const import *

mu = Uc(UC_ARCH_MIPS, UC_MODE_32 + UC_MODE_BIG_ENDIAN)

mregs = [UC_MIPS_REG_AT, UC_MIPS_REG_V0, UC_MIPS_REG_V1, UC_MIPS_REG_A0, UC_MIPS_REG_A1, UC_MIPS_REG_A2, UC_MIPS_REG_A3]
regs = ["at", "v0", "v1", "a0", "a1", "a2", "a3"]

def hook_interrupt(uc, intno, user_data):
  pc = uc.reg_read(UC_MIPS_REG_PC)
  if intno == 17:
    syscall_no = uc.reg_read(UC_MIPS_REG_V0)
    if syscall_no == 4004:
      # write
      fd = uc.reg_read(UC_MIPS_REG_A0)
      buf = uc.reg_read(UC_MIPS_REG_A1)
      count = uc.reg_read(UC_MIPS_REG_A2)
      os.write(fd, uc.mem_read(buf, count))
      return True

    print("syscall", syscall_no, hex(pc))
    if syscall_no == 4005:
      filename = uc.reg_read(UC_MIPS_REG_A0)
      print('open("%s")' % uc.mem_read(filename, 0x100).split(b"\x00")[0].decode('utf-8'))
    else:
      jj = []
      for i,r in zip(mregs, regs):
        jj += "%s: %8x " % (r, uc.reg_read(i))
      print(''.join(jj))
    return True

  print("interrupt", intno, hex(pc))
  if intno == 22:
    raise Exception
  return True

cnt = 0
def hook_code(uc, address, size, user_data):
  global cnt
  cnt += 1

  """
  dat = mu.mem_read(address, size)
  if dat == "\x0c\x00\x00\x00" or dat == "\x00\x00\x00\x0c":
    raise Exception("syscall")
  """
  
  #if cnt == 2000:
  #  raise Exception("too many instructions")
  try:
    print(">>> Tracing instruction at 0x%x, instruction size = %u" % (address, size))
    """
    jj = []
    for i in range(16):
      jj += "r%d: %x " % (i, uc.reg_read(i))
    print(''.join(jj))
    """
    #print('    code hook: pc=%08x sp=%08x' % (
    #  uc.reg_read(UC_MIPS_REG_PC),
    #  uc.reg_read(UC_MIPS_REG_SP)
    #  ))
  except:
    raise Exception("ctrl-c")

elf = open("test", "rb")
data = elf.read()
elf.seek(0)

#rte = data.find(b"\x08\x02\x2c\x95")
#print(hex(rte))

SIZE = 16*1024*1024
mu.mem_map(0, SIZE)

elffile = ELFFile(elf)
for seg in elffile.iter_segments():
  print(seg.header, hex(seg.header.p_vaddr))
  mu.mem_write(seg.header.p_vaddr, seg.data())

entry = elffile.header.e_entry
print("entrypoint: %x" % entry)
#hexdump(mu.mem_read(entry, 0x10))

mu.reg_write(UC_MIPS_REG_SP, SIZE-0x2000)

# http://articles.manugarg.com/aboutelfauxiliaryvectors.html
mu.mem_write(SIZE-0x2000, struct.pack(">IIIIIII", 1, SIZE-0x1000, 0, SIZE-0x1000, 0, SIZE-0x1000, 0))
#hexdump(mu.mem_read(SIZE-0x2000, 0x100))

# nop osinit
#mu.mem_write(0x44524, b"\x03\xe0\x00\x08\x00\x00\x00\x00")

#mu.hook_add(UC_HOOK_BLOCK, hook_code, user_data=mu)
#mu.hook_add(UC_HOOK_CODE, hook_code, user_data=mu)

mu.hook_add(UC_HOOK_INTR, hook_interrupt)
#mu.hook_add(UC_HOOK_INSN, hook_interrupt, None, 1, 0, 0x0c000000)
mu.emu_start(entry, SIZE)
