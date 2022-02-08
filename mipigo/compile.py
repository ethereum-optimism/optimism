#!/usr/bin/env python3
import os
import sys
import struct
import hashlib
from rangetree import RangeTree
from elftools.elf.elffile import ELFFile

def load_minigeth(fn="minigeth"):
  elf = open(fn, "rb")
  data = elf.read()
  elf.seek(0)

  elffile = ELFFile(elf)

  end_addr = 0
  for seg in elffile.iter_segments():
    end_addr = max(end_addr, seg.header.p_vaddr + seg.header.p_memsz)

  # program memory (16 MB)
  prog_size = (end_addr+0xFFF) & ~0xFFF
  prog_dat = bytearray(prog_size)
  print("malloced 0x%x for program" % prog_size)

  for seg in elffile.iter_segments():
    print(seg.header, hex(seg.header.p_vaddr))
    prog_dat[seg.header.p_vaddr:seg.header.p_vaddr+len(seg.data())] = seg.data()

  entry = elffile.header.e_entry
  print("entrypoint: 0x%x" % entry)

  # moved to MIPS
  sf = os.path.join(os.path.dirname(os.path.abspath(__file__)), "startup", "startup.bin")
  start = open(sf, "rb").read() + struct.pack(">I", entry)
  prog_dat[:len(start)] = start
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
          prog_dat[symbol['st_value']:symbol['st_value']+8] = b"\x03\xe0\x00\x08\x00\x00\x00\x00"
          found += 1
    except Exception:
      #traceback.print_exc()
      pass

  #assert(found == 2)
  return prog_dat, prog_size, r


if __name__ == "__main__":
  fn = "minigeth"
  if len(sys.argv) > 1:
    fn = sys.argv[1]

  prog_dat, prog_size, _ = load_minigeth(fn)
  print("compiled %d bytes with md5 %s" % (prog_size, hashlib.md5(prog_dat).hexdigest()))

  with open(fn+".bin", "wb") as f:
    f.write(prog_dat)