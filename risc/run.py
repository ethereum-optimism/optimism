#!/usr/bin/env python3
import os
import struct
from elftools.elf.elffile import ELFFile

from termcolor import colored, cprint
from hexdump import hexdump
from unicorn import *
from unicorn.mips_const import *

mu = Uc(UC_ARCH_MIPS, UC_MODE_32 + UC_MODE_BIG_ENDIAN)

mregs = [UC_MIPS_REG_AT, UC_MIPS_REG_V0, UC_MIPS_REG_V1, UC_MIPS_REG_A0, UC_MIPS_REG_A1, UC_MIPS_REG_A2, UC_MIPS_REG_A3]
regs = ["at", "v0", "v1", "a0", "a1", "a2", "a3"]

SIZE = 16*1024*1024
heap_start = 16*1024*1024

fcnt = 0
def hook_interrupt(uc, intno, user_data):
  global heap_start, fcnt
  pc = uc.reg_read(UC_MIPS_REG_PC)
  if intno == 17:
    syscall_no = uc.reg_read(UC_MIPS_REG_V0)
    uc.reg_write(UC_MIPS_REG_V0, 0)
    if syscall_no == 4004:
      # write
      fd = uc.reg_read(UC_MIPS_REG_A0)
      buf = uc.reg_read(UC_MIPS_REG_A1)
      count = uc.reg_read(UC_MIPS_REG_A2)
      if fd == 1:
        # stdout
        os.write(fd, colored(uc.mem_read(buf, count).decode('utf-8'), 'green').encode('utf-8'))
      elif fd == 2:
        # stderr
        os.write(fd, colored(uc.mem_read(buf, count).decode('utf-8'), 'red').encode('utf-8'))
      else:
        os.write(fd, uc.mem_read(buf, count))
      uc.reg_write(UC_MIPS_REG_A3, 0)
      return True

    if syscall_no == 4005:
      filename = uc.reg_read(UC_MIPS_REG_A0)
      print('open("%s")' % uc.mem_read(filename, 0x100).split(b"\x00")[0].decode('utf-8'))
      # open fd=4
      uc.reg_write(UC_MIPS_REG_V0, 4)
    elif syscall_no == 4238:
      addr = uc.reg_read(UC_MIPS_REG_A0)
      print("futex", hex(addr))
      uc.mem_write(addr, b"\x00\x00\x00\x01")
      #raise Exception("not support")
      uc.reg_write(UC_MIPS_REG_V0, 1)
      uc.reg_write(UC_MIPS_REG_A3, 0)
      fcnt += 1
      if fcnt == 20:
        raise Exception("too much futex")
      return True
    elif syscall_no == 4120:
      print("clone not supported")
      #uc.reg_write(UC_MIPS_REG_V0, -1)
      uc.reg_write(UC_MIPS_REG_V0, 1238238)
      uc.reg_write(UC_MIPS_REG_A3, 0)
      return True
    elif syscall_no == 4194:
      # rt_sigaction	
      return
    elif syscall_no == 4263:
      print("clock gettime")
      #raise Exception
    elif syscall_no == 4003:
      fd = uc.reg_read(UC_MIPS_REG_A0)
      buf = uc.reg_read(UC_MIPS_REG_A1)
      count = uc.reg_read(UC_MIPS_REG_A2)
      print("read", fd, hex(buf), count)
      uc.mem_write(buf, b"16384\n\x00")
      uc.reg_write(UC_MIPS_REG_V0, 6)
    elif syscall_no == 4246:
      print("EXIT")
      return False
    elif syscall_no == 4090:
      a0 = uc.reg_read(UC_MIPS_REG_A0)
      a1 = uc.reg_read(UC_MIPS_REG_A1)
      a2 = uc.reg_read(UC_MIPS_REG_A2)
      a3 = uc.reg_read(UC_MIPS_REG_A3)
      a4 = uc.reg_read(UC_MIPS_REG_T0)
      a5 = uc.reg_read(UC_MIPS_REG_T1)
      print("mmap", hex(a0), hex(a1), hex(a2), hex(a3), hex(a4), hex(a5), "at", hex(heap_start))
      if a0 == 0:
        print("malloced new")
        #mu.mem_map(heap_start, a1)
        uc.reg_write(UC_MIPS_REG_V0, heap_start)
        heap_start += a1
      else:
        uc.reg_write(UC_MIPS_REG_V0, a0)
    else:
      print("syscall", syscall_no, hex(pc))
      jj = []
      for i,r in zip(mregs, regs):
        jj += "%s: %8x " % (r, uc.reg_read(i))
      print(''.join(jj))
    uc.reg_write(UC_MIPS_REG_A3, 0)
    return True

  print("interrupt", intno, hex(pc))
  if intno != 17:
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
#elf = open("go-ethereum", "rb")
data = elf.read()
elf.seek(0)

#rte = data.find(b"\x08\x02\x2c\x95")
#print(hex(rte))

mu.mem_map(0, SIZE)

# heap
mu.mem_map(heap_start, 0x10000000)

elffile = ELFFile(elf)
for seg in elffile.iter_segments():
  print(seg.header, hex(seg.header.p_vaddr))
  mu.mem_write(seg.header.p_vaddr, seg.data())

entry = elffile.header.e_entry
print("entrypoint: %x" % entry)
#hexdump(mu.mem_read(entry, 0x10))

mu.reg_write(UC_MIPS_REG_SP, SIZE-0x2000)

# http://articles.manugarg.com/aboutelfauxiliaryvectors.html
_AT_PAGESZ = 6
mu.mem_write(SIZE-0x2000, struct.pack(">IIIIIIII", 1, SIZE-0x1000, 0, SIZE-0x1000, 0,
  _AT_PAGESZ, 0x1000, 0))
#mu.mem_write(SIZE-0x1000, b"GOMAXPROCS=1\x00")
#hexdump(mu.mem_read(SIZE-0x2000, 0x100))

# nop osinit
#mu.mem_write(0x44524, b"\x03\xe0\x00\x08\x00\x00\x00\x00")

for section in elffile.iter_sections():
  try:
    for nsym, symbol in enumerate(section.iter_symbols()):
      if symbol.name == "runtime.gcenable":
        print(nsym, symbol.name)
        # nop gcenable
        mu.mem_write(symbol['st_value'], b"\x03\xe0\x00\x08\x00\x00\x00\x00")
  except Exception:
    pass


#mu.hook_add(UC_HOOK_BLOCK, hook_code, user_data=mu)
#mu.hook_add(UC_HOOK_CODE, hook_code, user_data=mu)

mu.hook_add(UC_HOOK_INTR, hook_interrupt)
#mu.hook_add(UC_HOOK_INSN, hook_interrupt, None, 1, 0, 0x0c000000)
mu.emu_start(entry, SIZE)
