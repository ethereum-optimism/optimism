#!/usr/bin/env python3
import os
import sys
import tempfile
from capstone import *
from elftools.elf.elffile import ELFFile
md = Cs(CS_ARCH_MIPS, CS_MODE_32 + CS_MODE_BIG_ENDIAN)

def maketest(d, out):
  with tempfile.NamedTemporaryFile() as nf:
    path = "/Users/kafka/fun/mips/mips-gcc-4.8.1/bin/"
    print("building", d, "->", out)
    # which mips is go
    ret = os.system("%s/mips-elf-as -defsym big_endian=1 -march=mips32r2 -o %s %s" % (path, nf.name, d))
    assert(ret == 0)
    nf.seek(0)
    elffile = ELFFile(nf)
    #print(elffile)
    for sec in elffile.iter_sections():
      #print(sec, sec.name, sec.data())
      if sec.name == ".test":
        with open(out, "wb") as f:
          # jump to 0xdead0000 when done
          #data = b"\x24\x1f\xde\xad\x00\x1f\xfc\x00" + sec.data()
          data = sec.data()
          for dd in md.disasm(data, 0):
            print(dd)
          f.write(data)

if __name__ == "__main__":
  os.makedirs("/tmp/mips", exist_ok=True)
  if len(sys.argv) > 2:
    maketest(sys.argv[1], sys.argv[2])
  else:
    for d in os.listdir("test/"):
      if not d.endswith(".asm"):
        continue
      maketest("test/"+d, "test/bin/"+(d.replace(".asm", ".bin")))