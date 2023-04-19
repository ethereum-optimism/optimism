// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;
import "./MIPSMemory.sol";

// https://inst.eecs.berkeley.edu/~cs61c/resources/MIPS_Green_Sheet.pdf
// https://uweb.engr.arizona.edu/~ece369/Resources/spim/MIPSReference.pdf
// https://en.wikibooks.org/wiki/MIPS_Assembly/Instruction_Formats

// https://www.cs.cmu.edu/afs/cs/academic/class/15740-f97/public/doc/mips-isa.pdf
// page A-177

// This is a separate contract from the challenge contract
// Anyone can use it to validate a MIPS state transition
// First, to prepare, you call AddMerkleState, which adds valid state nodes in the stateHash. 
// If you are using the Preimage oracle, you call AddPreimage
// Then, you call Step. Step will revert if state is missing. If all state is present, it will return the next hash

contract MIPS {
  MIPSMemory public immutable m;

  uint32 constant public REG_OFFSET = 0xc0000000;
  uint32 constant public REG_ZERO = REG_OFFSET;
  uint32 constant public REG_LR = REG_OFFSET + 0x1f*4;
  uint32 constant public REG_PC = REG_OFFSET + 0x20*4;
  uint32 constant public REG_HI = REG_OFFSET + 0x21*4;
  uint32 constant public REG_LO = REG_OFFSET + 0x22*4;
  uint32 constant public REG_HEAP = REG_OFFSET + 0x23*4;

  uint32 constant public HEAP_START = 0x20000000;
  uint32 constant public BRK_START = 0x40000000;

  constructor() {
    m = new MIPSMemory();
  }

  bool constant public debug = true;

  event DidStep(bytes32 stateHash);
  event DidWriteMemory(uint32 addr, uint32 value);
  event TryReadMemory(uint32 addr);
  event DidReadMemory(uint32 addr, uint32 value);

  function WriteMemory(bytes32 stateHash, uint32 addr, uint32 value) internal returns (bytes32) {
    if (address(m) != address(0)) {
      emit DidWriteMemory(addr, value);
      bytes32 newStateHash = m.WriteMemory(stateHash, addr, value);
      require(m.ReadMemory(newStateHash, addr) == value, "memory readback check failed");
      return newStateHash;
    }
    assembly {
      // TODO: this is actually doing an SLOAD first
      sstore(addr, value)
    }
    return stateHash;
  }

  function ReadMemory(bytes32 stateHash, uint32 addr) internal returns (uint32 ret) {
    if (address(m) != address(0)) {
      emit TryReadMemory(addr);
      ret = m.ReadMemory(stateHash, addr);
      //emit DidReadMemory(addr, ret);
      return ret;
    }
    assembly {
      ret := sload(addr)
    }
  }

  function Steps(bytes32 stateHash, uint count) public returns (bytes32) {
    for (uint i = 0; i < count; i++) {
      stateHash = Step(stateHash);
    }
    return stateHash;
  }

  function SE(uint32 dat, uint32 idx) internal pure returns (uint32) {
    bool isSigned = (dat >> (idx-1)) != 0;
    uint256 signed = ((1 << (32-idx)) - 1) << idx;
    uint256 mask = (1 << idx) - 1;
    return uint32(dat&mask | (isSigned ? signed : 0));
  }

  function handleSyscall(bytes32 stateHash) internal returns (bytes32, bool) {
    uint32 syscall_no = ReadMemory(stateHash, REG_OFFSET+2*4);
    uint32 v0 = 0;
    bool exit = false;

    if (syscall_no == 4090) {
      // mmap
      uint32 a0 = ReadMemory(stateHash, REG_OFFSET+4*4);
      if (a0 == 0) {
        uint32 sz = ReadMemory(stateHash, REG_OFFSET+5*4);
        uint32 hr = ReadMemory(stateHash, REG_HEAP);
        v0 = HEAP_START + hr;
        stateHash = WriteMemory(stateHash, REG_HEAP, hr+sz);
      } else {
        v0 = a0;
      }
    } else if (syscall_no == 4045) {
      // brk
      v0 = BRK_START;
    } else if (syscall_no == 4120) {
      // clone (not supported)
      v0 = 1;
    } else if (syscall_no == 4246) {
      // exit group
      exit = true;
    }

    stateHash = WriteMemory(stateHash, REG_OFFSET+2*4, v0);
    stateHash = WriteMemory(stateHash, REG_OFFSET+7*4, 0);
    return (stateHash, exit);
  }

  function Step(bytes32 stateHash) public returns (bytes32 newStateHash) {
    uint32 pc = ReadMemory(stateHash, REG_PC);
    if (pc == 0x5ead0000) {
      return stateHash;
    }
    newStateHash = stepPC(stateHash, pc, pc+4);
    if (address(m) != address(0)) {
      emit DidStep(newStateHash);
    }
  }

  // will revert if any required input state is missing
  function stepPC(bytes32 stateHash, uint32 pc, uint32 nextPC) internal returns (bytes32) {
    // instruction fetch
    uint32 insn = ReadMemory(stateHash, pc);

    uint32 opcode = insn >> 26; // 6-bits
    uint32 func = insn & 0x3f; // 6-bits

    // j-type j/jal
    if (opcode == 2 || opcode == 3) {
      stateHash = stepPC(stateHash, nextPC,
        SE(insn&0x03FFFFFF, 26) << 2);
      if (opcode == 3) {
        stateHash = WriteMemory(stateHash, REG_LR, pc+8);
      }
      return stateHash;
    }

    // register fetch
    uint32 storeAddr = REG_ZERO;
    uint32 rs;
    uint32 rt;
    uint32 rtReg = REG_OFFSET + ((insn >> 14) & 0x7C);

    // R-type or I-type (stores rt)
    rs = ReadMemory(stateHash, REG_OFFSET + ((insn >> 19) & 0x7C));
    storeAddr = REG_OFFSET + ((insn >> 14) & 0x7C);
    if (opcode == 0 || opcode == 0x1c) {
      // R-type (stores rd)
      rt = ReadMemory(stateHash, rtReg);
      storeAddr = REG_OFFSET + ((insn >> 9) & 0x7C);
    } else if (opcode < 0x20) {
      // rt is SignExtImm
      // don't sign extend for andi, ori, xori
      if (opcode == 0xC || opcode == 0xD || opcode == 0xe) {
        // ZeroExtImm
        rt = insn&0xFFFF;
      } else {
        // SignExtImm
        rt = SE(insn&0xFFFF, 16);
      }
    } else if (opcode >= 0x28 || opcode == 0x22 || opcode == 0x26) {
      // store rt value with store
      rt = ReadMemory(stateHash, rtReg);

      // store actual rt with lwl and lwr
      storeAddr = rtReg;
    }

    if ((opcode >= 4 && opcode < 8) || opcode == 1) {
      bool shouldBranch = false;

      if (opcode == 4 || opcode == 5) {   // beq/bne
        rt = ReadMemory(stateHash, rtReg);
        shouldBranch = (rs == rt && opcode == 4) || (rs != rt && opcode == 5);
      } else if (opcode == 6) { shouldBranch = int32(rs) <= 0; // blez
      } else if (opcode == 7) { shouldBranch = int32(rs) > 0; // bgtz
      } else if (opcode == 1) {
        // regimm
        uint32 rtv = ((insn >> 16) & 0x1F);
        if (rtv == 0) shouldBranch = int32(rs) < 0;  // bltz
        if (rtv == 1) shouldBranch = int32(rs) >= 0; // bgez
      }

      if (shouldBranch) {
        return stepPC(stateHash, nextPC,
          pc + 4 + (SE(insn&0xFFFF, 16)<<2));
      }
      // branch not taken
      return stepPC(stateHash, nextPC, nextPC+4);
    }

    // memory fetch (all I-type)
    // we do the load for stores also
    uint32 mem;
    if (opcode >= 0x20) {
      // M[R[rs]+SignExtImm]
      uint32 SignExtImm = SE(insn&0xFFFF, 16);
      rs += SignExtImm;
      uint32 addr = rs & 0xFFFFFFFC;
      mem = ReadMemory(stateHash, addr);
      if (opcode >= 0x28 && opcode != 0x30) {
        // store
        storeAddr = addr;
      }
    }

    // ALU
    uint32 val = execute(insn, rs, rt, mem);

    if (opcode == 0 && func >= 8 && func < 0x1c) {
      if (func == 8 || func == 9) {
        // jr/jalr
        stateHash = stepPC(stateHash, nextPC, rs);
        if (func == 9) {
          stateHash = WriteMemory(stateHash, REG_LR, pc+8);
        }
        return stateHash;
      }

      // handle movz and movn when they don't write back
      if (func == 0xa && rt != 0) { // movz
        storeAddr = REG_ZERO;
      }
      if (func == 0xb && rt == 0) { // movn
        storeAddr = REG_ZERO;
      }

      // syscall (can read and write)
      if (func == 0xC) {
        //revert("unhandled syscall");
        bool exit;
        (stateHash, exit) = handleSyscall(stateHash);
        if (exit) {
          nextPC = 0x5ead0000;
        }
      }

      // lo and hi registers
      // can write back
      if (func >= 0x10 && func < 0x1c) {
        if (func == 0x10) val = ReadMemory(stateHash, REG_HI); // mfhi
        else if (func == 0x11) storeAddr = REG_HI; // mthi
        else if (func == 0x12) val = ReadMemory(stateHash, REG_LO); // mflo
        else if (func == 0x13) storeAddr = REG_LO; // mtlo

        uint32 hi;
        if (func == 0x18) { // mult
          uint64 acc = uint64(int64(int32(rs))*int64(int32(rt)));
          hi = uint32(acc>>32);
          val = uint32(acc);
        } else if (func == 0x19) { // multu
          uint64 acc = uint64(uint64(rs)*uint64(rt));
          hi = uint32(acc>>32);
          val = uint32(acc);
        } else if (func == 0x1a) { // div
          hi = uint32(int32(rs)%int32(rt));
          val = uint32(int32(rs)/int32(rt));
        } else if (func == 0x1b) { // divu
          hi = rs%rt;
          val = rs/rt;
        }

        // lo/hi writeback
        if (func >= 0x18 && func < 0x1c) {
          stateHash = WriteMemory(stateHash, REG_HI, hi);
          storeAddr = REG_LO;
        }
      }
    }

    // stupid sc, write a 1 to rt
    if (opcode == 0x38 && rtReg != REG_ZERO) {
      stateHash = WriteMemory(stateHash, rtReg, 1);
    }

    // write back
    if (storeAddr != REG_ZERO) {
      stateHash = WriteMemory(stateHash, storeAddr, val);
    }

    stateHash = WriteMemory(stateHash, REG_PC, nextPC);

    return stateHash;
  }

  function execute(uint32 insn, uint32 rs, uint32 rt, uint32 mem) internal pure returns (uint32) {
    uint32 opcode = insn >> 26;    // 6-bits
    uint32 func = insn & 0x3f; // 6-bits
    // TODO: deref the immed into a register

    if (opcode < 0x20) {
      // transform ArithLogI
      // TODO: replace with table
      if (opcode >= 8 && opcode < 0xF) {
        if (opcode == 8) { func = 0x20; }        // addi
        else if (opcode == 9) { func = 0x21; }   // addiu
        else if (opcode == 0xa) { func = 0x2a; } // slti
        else if (opcode == 0xb) { func = 0x2B; } // sltiu
        else if (opcode == 0xc) { func = 0x24; } // andi
        else if (opcode == 0xd) { func = 0x25; } // ori
        else if (opcode == 0xe) { func = 0x26; } // xori
        opcode = 0;
      }

      // 0 is opcode SPECIAL
      if (opcode == 0) {
        uint32 shamt = (insn >> 6) & 0x1f;
        if (func < 0x20) {
          if (func >= 0x08) { return rs;  // jr/jalr/div + others
          // Shift and ShiftV
          } else if (func == 0x00) { return rt << shamt;      // sll
          } else if (func == 0x02) { return rt >> shamt;      // srl
          } else if (func == 0x03) { return SE(rt >> shamt, 32-shamt);      // sra
          } else if (func == 0x04) { return rt << (rs&0x1F);         // sllv
          } else if (func == 0x06) { return rt >> (rs&0x1F);         // srlv
          } else if (func == 0x07) { return SE(rt >> rs, 32-rs);     // srav
          }
        }
        // 0x10-0x13 = mfhi, mthi, mflo, mtlo
        // R-type (ArithLog)
        if (func == 0x20 || func == 0x21) { return rs+rt;   // add or addu
        } else if (func == 0x22 || func == 0x23) { return rs-rt;   // sub or subu
        } else if (func == 0x24) { return rs&rt;            // and
        } else if (func == 0x25) { return (rs|rt);          // or
        } else if (func == 0x26) { return (rs^rt);          // xor
        } else if (func == 0x27) { return ~(rs|rt);         // nor
        } else if (func == 0x2a) {
          return int32(rs)<int32(rt) ? 1 : 0; // slt
        } else if (func == 0x2B) {
          return rs<rt ? 1 : 0;               // sltu
        }
      } else if (opcode == 0xf) { return rt<<16; // lui
      } else if (opcode == 0x1c) {  // SPECIAL2
        if (func == 2) return uint32(int32(rs)*int32(rt)); // mul
        if (func == 0x20 || func == 0x21) { // clo
          if (func == 0x20) rs = ~rs;
          uint32 i = 0; while (rs&0x80000000 != 0) { i++; rs <<= 1; } return i;
        }
      }
    } else if (opcode < 0x28) {
      if (opcode == 0x20) {  // lb
        return SE((mem >> (24-(rs&3)*8)) & 0xFF, 8);
      } else if (opcode == 0x21) {  // lh
        return SE((mem >> (16-(rs&2)*8)) & 0xFFFF, 16);
      } else if (opcode == 0x22) {  // lwl
        uint32 val = mem << ((rs&3)*8);
        uint32 mask = uint32(0xFFFFFFFF) << ((rs&3)*8);
        return (rt & ~mask) | val;
      } else if (opcode == 0x23) { return mem;   // lw
      } else if (opcode == 0x24) {  // lbu
        return (mem >> (24-(rs&3)*8)) & 0xFF;
      } else if (opcode == 0x25) {  // lhu
        return (mem >> (16-(rs&2)*8)) & 0xFFFF;
      } else if (opcode == 0x26) {  // lwr
        uint32 val = mem >> (24-(rs&3)*8);
        uint32 mask = uint32(0xFFFFFFFF) >> (24-(rs&3)*8);
        return (rt & ~mask) | val;
      }
    } else if (opcode == 0x28) { // sb
      uint32 val = (rt&0xFF) << (24-(rs&3)*8);
      uint32 mask = 0xFFFFFFFF ^ uint32(0xFF << (24-(rs&3)*8));
      return (mem & mask) | val;
    } else if (opcode == 0x29) { // sh
      uint32 val = (rt&0xFFFF) << (16-(rs&2)*8);
      uint32 mask = 0xFFFFFFFF ^ uint32(0xFFFF << (16-(rs&2)*8));
      return (mem & mask) | val;
    } else if (opcode == 0x2a) {  // swl
      uint32 val = rt >> ((rs&3)*8);
      uint32 mask = uint32(0xFFFFFFFF) >> ((rs&3)*8);
      return (mem & ~mask) | val;
    } else if (opcode == 0x2b) { // sw
      return rt;
    } else if (opcode == 0x2e) {  // swr
      uint32 val = rt << (24-(rs&3)*8);
      uint32 mask = uint32(0xFFFFFFFF) << (24-(rs&3)*8);
      return (mem & ~mask) | val;
    } else if (opcode == 0x30) { return mem; // ll
    } else if (opcode == 0x38) { return rt; // sc
    }

    revert("invalid instruction");
  }
}
