#!/usr/bin/env python3
from elftools.elf.elffile import ELFFile

from hexdump import hexdump
from unicorn import *
from unicorn.mips_const import *

mu = Uc(UC_ARCH_MIPS, UC_MODE_32 + UC_MODE_BIG_ENDIAN)

def hook_interrupt(uc, intno, user_data):
  print("interrupt", intno)
  raise Exception

cnt = 0
def hook_code(uc, address, size, user_data):
  global cnt
  cnt += 1
  dat = mu.mem_read(address, size)
  if dat == "\x0c\x00\x00\x00" or dat == "\x00\x00\x00\x0c":
    raise Exception("syscall")
  
  #if cnt == 2000:
  #  raise Exception("too many instructions")
  try:
    print(">>> Tracing instruction at 0x%x, instruction size = %u" % (address, size))
    jj = []
    for i in range(16):
      jj += "r%d: %x " % (i, uc.reg_read(i))
    print(''.join(jj))
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


#mu.mem_write(0x10000, data)

hexdump(mu.mem_read(entry, 0x10))

mu.reg_write(UC_MIPS_REG_SP, SIZE-0x1000)

#mu.hook_add(UC_HOOK_BLOCK, hook_code, user_data=mu)
mu.hook_add(UC_HOOK_CODE, hook_code, user_data=mu)

mu.hook_add(UC_HOOK_INTR, hook_interrupt)
#mu.hook_add(UC_HOOK_INSN, hook_interrupt, None, 1, 0, 0x0c000000)
mu.emu_start(entry, SIZE)
