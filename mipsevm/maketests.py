#!/usr/bin/env python3
import os
from capstone import *
md = Cs(CS_ARCH_MIPS, CS_MODE_32 + CS_MODE_BIG_ENDIAN)
from elftools.elf.elffile import ELFFile
os.makedirs("/tmp/mips", exist_ok=True)
path = "/Users/kafka/fun/mips/mips-gcc-4.8.1/bin/"

# Tests from:
# https://github.com/grantae/OpenMIPS/blob/master/software/test/macro/tests/addiu/src/os/khi/addiu.asm

for d in os.listdir("test/"):
  if not d.endswith(".asm"):
    continue
  print("building", d)
  # which mips is go
  os.system("%s/mips-elf-as -defsym big_endian=1 -march=mips32 -o /tmp/mips/%s test/%s" % (path, d, d))
  elffile = ELFFile(open("/tmp/mips/"+d, "rb"))
  #print(elffile)
  for sec in elffile.iter_sections():
    #print(sec, sec.name, sec.data())
    if sec.name == ".test":
      with open("test/bin/"+(d.replace(".asm", ".bin")), "wb") as f:
        # jump to 0xdead0000 when done
        data = b"\x24\x1f\xde\xad\x00\x1f\xfc\x00" + sec.data()
        for dd in md.disasm(data, 0):
          print(dd)
        f.write(data)


