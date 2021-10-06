#!/usr/bin/env python3
import struct
from rangetree import RangeTree
from elftools.elf.elffile import ELFFile

from unicorn import *
from unicorn.mips_const import *

def load_minigeth(mu):
  elf = open("go-ethereum", "rb")
  data = elf.read()
  elf.seek(0)

  elffile = ELFFile(elf)

  end_addr = 0
  for seg in elffile.iter_segments():
    end_addr = max(end_addr, seg.header.p_vaddr + seg.header.p_memsz)

  # program memory (16 MB)
  prog_size = (end_addr+0xFFF) & ~0xFFF
  mu.mem_map(0, prog_size)
  print("malloced 0x%x for program" % prog_size)

  for seg in elffile.iter_segments():
    print(seg.header, hex(seg.header.p_vaddr))
    mu.mem_write(seg.header.p_vaddr, seg.data())

  entry = elffile.header.e_entry
  print("entrypoint: 0x%x" % entry)

  # moved to MIPS
  start = open("startup.bin", "rb").read() + struct.pack(">I", entry)
  mu.mem_write(0, start)
  entry = 0

  r = RangeTree()
  found = 0
  for section in elffile.iter_sections():
    try:
      for nsym, symbol in enumerate(section.iter_symbols()):
        ss = symbol['st_value']
        se = ss+symbol['st_size']
        if ss != se:
          try:
            r[ss:se] = symbol.name
          except KeyError:
            continue
        #print(nsym, symbol.name, symbol['st_value'], symbol['st_size'])
        if symbol.name == "runtime.gcenable":
          print(nsym, symbol.name)
          # nop gcenable
          mu.mem_write(symbol['st_value'], b"\x03\xe0\x00\x08\x00\x00\x00\x00")
          found += 1
        if symbol.name == "github.com/ethereum/go-ethereum/oracle.Halt":
          #00400000: 2004dead ; <input:0> li $a0, 57005
          # 00400004: 00042400 ; <input:1> sll $a0, $a0, 16
          # 00400008: 00800008 ; <input:2> jr $a0
          mu.mem_write(symbol['st_value'], b"\x20\x04\xde\xad\x00\x04\x24\x00\x00\x80\x00\x08")
          found += 1
    except Exception:
      #traceback.print_exc()
      pass

  assert(found == 2)

  return prog_size, r


if __name__ == "__main__":
  mu = Uc(UC_ARCH_MIPS, UC_MODE_32 + UC_MODE_BIG_ENDIAN)

  prog_size, _ = load_minigeth(mu)
  print("compiled %d bytes" % prog_size)

  with open("minigeth.bin", "wb") as f:
    f.write(mu.mem_read(0, prog_size))