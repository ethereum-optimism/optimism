#!/usr/bin/env python3
import os
import sys
import math
import struct
import binascii
import traceback
from elftools.elf.elffile import ELFFile
from capstone import *
md = Cs(CS_ARCH_MIPS, CS_MODE_32 + CS_MODE_BIG_ENDIAN)
tracelevel = int(os.getenv("TRACE", 0))

from termcolor import colored, cprint
from hexdump import hexdump
from unicorn import *
from unicorn.mips_const import *
from rangetree import RangeTree

mu = Uc(UC_ARCH_MIPS, UC_MODE_32 + UC_MODE_BIG_ENDIAN)

# memory trie
# register trie

mregs = [UC_MIPS_REG_AT, UC_MIPS_REG_V0, UC_MIPS_REG_V1, UC_MIPS_REG_A0, UC_MIPS_REG_A1, UC_MIPS_REG_A2, UC_MIPS_REG_A3]
regs = ["at", "v0", "v1", "a0", "a1", "a2", "a3"]

heap_start = 0x20000000 # 0x20000000-0x30000000
# input oracle              @ 0x30000000
# output oracle             @ 0x30000800
# preimage oracle (write)   @ 0x30001000
# preimage oracle (read)    @ 0x31000000-0x32000000 (16 MB)

brk_start = 0x40000000  # 0x40000000-0x80000000
stack_start = 0x7FFFF000

# hmm, very slow
icount = 0
bcount = 0

instrumenting = False
instrumenting_all = False
instructions_seen = set()
def hook_code_simple(uc, address, size, user_data):
  global icount, bcount
  #assert size == 4
  try:
    newicount = size//4
    if bcount%10000 == 0 or instrumenting_all:
      if size > 0:
        dat = next(md.disasm(uc.mem_read(address, size), address))
      else:
        dat = "EMPTY BASIC BLOCK?!?"
      #instructions_seen.add(dat.mnemonic)
      #print(sorted(list(instructions_seen)))
      print("%10d(%2d): %8x %-80s %s" % (icount, newicount, address, r[address] if address in r else "UNKNOWN", dat))
    icount += newicount
    bcount += 1
    return True
  except Exception as e:
    raise e
  except:
    raise Exception

def start_instrumenting():
  global instrumenting, instrumenting_all
  if not instrumenting:
    if tracelevel >= 2:
      mu.hook_add(UC_HOOK_CODE, hook_code_simple, user_data=mu)
    elif tracelevel == 1:
      mu.hook_add(UC_HOOK_BLOCK, hook_code_simple, user_data=mu)
    if tracelevel >= 4:
      instrumenting_all = True
    instrumenting = True

tfd = 10
files = {}
fcnt = 0
step = 0
def hook_interrupt(uc, intno, user_data):
  global heap_start, fcnt, files, tfd, step
  pc = uc.reg_read(UC_MIPS_REG_PC)
  if intno == 17:
    syscall_no = uc.reg_read(UC_MIPS_REG_V0)
    #print("step:%d pc:%0x v0:%d" % (step, pc, syscall_no))
    step += 1
    uc.reg_write(UC_MIPS_REG_V0, 0)
    uc.reg_write(UC_MIPS_REG_A3, 0)

    if syscall_no == 4020:
      oracle_hash = binascii.hexlify(uc.mem_read(0x30001000, 0x20)).decode('utf-8')
      dat = open("/tmp/eth/0x"+oracle_hash, "rb").read()
      #print("oracle:", oracle_hash, len(dat))
      uc.mem_write(0x31000000, struct.pack(">I", len(dat)))
      uc.mem_write(0x31000004, dat)
      return True

    if syscall_no == 4004:
      # write
      fd = uc.reg_read(UC_MIPS_REG_A0)
      buf = uc.reg_read(UC_MIPS_REG_A1)
      count = uc.reg_read(UC_MIPS_REG_A2)
      #print("write(%d, %x, %d)" % (fd, buf, count))
      if fd == 1:
        # stdout
        os.write(fd, colored(uc.mem_read(buf, count).decode('utf-8'), 'green').encode('utf-8'))
      elif fd == 2:
        # stderr
        os.write(fd, colored(uc.mem_read(buf, count).decode('utf-8'), 'red').encode('utf-8'))
      else:
        os.write(fd, uc.mem_read(buf, count))
      uc.reg_write(UC_MIPS_REG_A3, 0)
      if fd == 2:
        start_instrumenting()
      return True

    if syscall_no == 4218:
      # madvise
      return
    elif syscall_no == 4194:
      # rt_sigaction
      return
    elif syscall_no == 4195:
      # rt_sigprocmask
      return
    elif syscall_no == 4055:
      # fcntl
      return
    elif syscall_no == 4220:
      # fcntl64
      return
    elif syscall_no == 4249:
      # epoll_ctl
      return
    elif syscall_no == 4263:
      # clock_gettime
      return
    elif syscall_no == 4326:
      # epoll_create1
      return
    elif syscall_no == 4328:
      # pipe2
      return
    elif syscall_no == 4206:
      # sigaltstack
      return
    elif syscall_no == 4222:
      # gettid
      return
    elif syscall_no == 4166:
      # nanosleep
      return

    if syscall_no == 4005:
      filename = uc.reg_read(UC_MIPS_REG_A0)
      print('open("%s")' % uc.mem_read(filename, 0x100).split(b"\x00")[0].decode('utf-8'))
      uc.reg_write(UC_MIPS_REG_V0, 4)
    elif syscall_no == 4045:
      print("brk", hex(brk_start))
      uc.reg_write(UC_MIPS_REG_V0, brk_start)
    elif syscall_no == 4288:
      dfd = uc.reg_read(UC_MIPS_REG_A0)
      filename = uc.reg_read(UC_MIPS_REG_A1)
      filename = uc.mem_read(filename, 0x100).split(b"\x00")[0].decode('utf-8')
      files[tfd] = open(filename, "rb")
      uc.reg_write(UC_MIPS_REG_V0, tfd)
      print('openat("%s") = %d' % (filename, tfd))
      tfd += 1
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
    elif syscall_no == 4006:
      fd = uc.reg_read(UC_MIPS_REG_A0)
      if fd >= 10:
        #print("close(%d)" % fd)
        files[fd].close()
        del files[fd]
      uc.reg_write(UC_MIPS_REG_V0, 0)
    elif syscall_no == 4003:
      fd = uc.reg_read(UC_MIPS_REG_A0)
      buf = uc.reg_read(UC_MIPS_REG_A1)
      count = uc.reg_read(UC_MIPS_REG_A2)
      # changing this works if we want smaller oracle
      #count = 4
      if fd == 4:
        val = b"2097152\n"
        uc.mem_write(buf, val)
        print("read", fd, hex(buf), count)
        uc.reg_write(UC_MIPS_REG_V0, len(val))
      else:
        ret = files[fd].read(count)
        uc.mem_write(buf, ret)
        #print("read", fd, hex(buf), count, len(ret))
        uc.reg_write(UC_MIPS_REG_V0, len(ret))
    elif syscall_no == 4246:
      a0 = uc.reg_read(UC_MIPS_REG_A0)
      if icount > 0:
        print("exit(%d) ran %.2f million instructions, %d binary searches" % (a0, icount/1_000_000, math.ceil(math.log2(icount))))
      else:
        print("exit(%d)" % a0)
      sys.stdout.flush()
      sys.stderr.flush()
      #os._exit(a0)
    elif syscall_no == 4090 or syscall_no == 4210:
      a0 = uc.reg_read(UC_MIPS_REG_A0)
      a1 = uc.reg_read(UC_MIPS_REG_A1)
      a2 = uc.reg_read(UC_MIPS_REG_A2)
      a3 = uc.reg_read(UC_MIPS_REG_A3)
      a4 = uc.reg_read(UC_MIPS_REG_T0)
      a5 = uc.reg_read(UC_MIPS_REG_T1)
      print("mmap", syscall_no, hex(a0), hex(a1), hex(a2), hex(a3), hex(a4), hex(a5), "at", hex(heap_start))
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
    return True

  print("interrupt", intno, hex(pc))
  if intno != 17:
    raise unicorn.UcError(0)
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

#elf = open("test", "rb")
elf = open("go-ethereum", "rb")
data = elf.read()
elf.seek(0)

#rte = data.find(b"\x08\x02\x2c\x95")
#print(hex(rte))

# program memory (16 MB)
prog_size = (len(data)+0xFFF) & ~0xFFF
mu.mem_map(0, prog_size)
print("malloced 0x%x for program" % prog_size)

# heap (256 MB) @ 0x20000000
mu.mem_map(heap_start, 256*1024*1024)

# brk (1024 MB) @ 0x40000000
mu.mem_map(brk_start, 1024*1024*1024)

# input oracle
mu.mem_map(0x30000000, 0x2000000)

if len(sys.argv) > 1:
  inputs = open("/tmp/eth/"+sys.argv[1], "rb").read()
else:
  inputs = open("/tmp/eth/13284469", "rb").read()
mu.mem_write(0x30000000, inputs)

# regs at 0xC0000000 in merkle

elffile = ELFFile(elf)
for seg in elffile.iter_segments():
  print(seg.header, hex(seg.header.p_vaddr))
  mu.mem_write(seg.header.p_vaddr, seg.data())

entry = elffile.header.e_entry
print("entrypoint: 0x%x" % entry)
#hexdump(mu.mem_read(entry, 0x10))

"""
mu.reg_write(UC_MIPS_REG_SP, stack_start-0x2000)

# http://articles.manugarg.com/aboutelfauxiliaryvectors.html
_AT_PAGESZ = 6
mu.mem_write(stack_start-0x2000, struct.pack(">IIIIII",
  0,  # argc
  0,  # argv
  0,  # envp
  _AT_PAGESZ, 0x1000, 0)) # auxv
mu.mem_write(stack_start-0x400, b"GOGC=off\x00")
"""

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
      if symbol.name == "runtime.load_g":
        # hardware?
        mu.mem_write(symbol['st_value'], b"\x03\xe0\x00\x08\x00\x00\x00\x00")
        found += 1
      if symbol.name == "runtime.save_g":
        # hardware?
        mu.mem_write(symbol['st_value'], b"\x03\xe0\x00\x08\x00\x00\x00\x00")
        found += 1
      if symbol.name == "_cgo_sys_thread_start":
        mu.mem_write(symbol['st_value'], b"\x03\xe0\x00\x08\x00\x00\x00\x00")
      if symbol.name == "github.com/ethereum/go-ethereum/oracle.Halt":
         #00400000: 2004dead ; <input:0> li $a0, 57005
        # 00400004: 00042400 ; <input:1> sll $a0, $a0, 16
        # 00400008: 00800008 ; <input:2> jr $a0
        mu.mem_write(symbol['st_value'], b"\x20\x04\xde\xad\x00\x04\x24\x00\x00\x80\x00\x08")
        found += 1
  except Exception:
    #traceback.print_exc()
    pass

assert(found == 4)
#mu.hook_add(UC_HOOK_BLOCK, hook_code, user_data=mu)

died_well = False

def hook_mem_invalid(uc, access, address, size, value, user_data):
  global died_well
  pc = uc.reg_read(UC_MIPS_REG_PC)
  if pc == 0xDEAD0000:
    died_well = True
    return False
  print("UNMAPPED MEMORY:", access, hex(address), size, "at", hex(pc))
  return False
mu.hook_add(UC_HOOK_MEM_READ_UNMAPPED | UC_HOOK_MEM_WRITE_UNMAPPED, hook_mem_invalid)
mu.hook_add(UC_HOOK_MEM_FETCH_UNMAPPED, hook_mem_invalid)

mu.hook_add(UC_HOOK_INTR, hook_interrupt)
#mu.hook_add(UC_HOOK_INSN, hook_interrupt, None, 1, 0, 0x0c000000)

if tracelevel >= 3:
  start_instrumenting()

with open("/tmp/minigeth.bin", "wb") as f:
  f.write(mu.mem_read(0, prog_size))

if os.getenv("COMPILE", None) == "1":
  exit(0)

# why do i need this?
#mu.mem_map(0xfffe0000, 0x20000)

try:
  mu.emu_start(entry, -1)
except unicorn.UcError:
  pass

if not died_well:
  raise Exception("program exitted early")

real_hash = binascii.hexlify(inputs[-0x40:-0x20])
compare_hash = binascii.hexlify(mu.mem_read(0x30000800, 0x20))
print("compare", real_hash, "to computed", compare_hash)

if real_hash != compare_hash:
  raise Exception("wrong hash")
