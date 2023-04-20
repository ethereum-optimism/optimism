// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;
pragma experimental ABIEncoderV2;

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

  struct State {
    bytes32 memRoot;
    bytes32 preimageKey;
    uint32 preimageOffset;

    uint32[32] registers;
    uint32 pc;
    uint32 nextPC;  // State is executing a branch/jump delay slot if nextPC != pc+4
    uint32 lr;
    uint32 lo;
    uint32 hi;
    uint32 heap;
    uint8 exitCode;
    bool exited;
    uint64 step;
  }

  // total State size: 32+32+4+32*4+5*4+1+1+8 = 226 bytes

  uint32 constant public HEAP_START = 0x20000000;
  uint32 constant public BRK_START = 0x40000000;

//  event DidStep(bytes32 stateHash);
//  event DidWriteMemory(uint32 addr, uint32 value);
//  event TryReadMemory(uint32 addr);
//  event DidReadMemory(uint32 addr, uint32 value);

  function SE(uint32 dat, uint32 idx) internal pure returns (uint32) {
    bool isSigned = (dat >> (idx-1)) != 0;
    uint256 signed = ((1 << (32-idx)) - 1) << idx;
    uint256 mask = (1 << idx) - 1;
    return uint32(dat&mask | (isSigned ? signed : 0));
  }

  // will revert if any required input state is missing
  function Step(bytes32 stateHash, bytes memory stateData, bytes calldata proof) public returns (bytes32) {
    require(stateHash == keccak256(stateData), "stateHash must match input");
    State memory state = abi.decode(stateData, (State)); // TODO not efficient, need to write a "decodePacked" for State
    if(state.exited) { // don't change state once exited
      return stateHash;
    }

    uint32 pc = state.pc;

    // instruction fetch
    uint32 insn; // TODO proof the memory read against memRoot
    assembly {
      insn := shr(sub(256, 32), calldataload(add(proof.offset, 0x20)))
    }

    uint32 opcode = insn >> 26; // 6-bits
    uint32 func = insn & 0x3f; // 6-bits

    // j-type j/jal
    if (opcode == 2 || opcode == 3) {
      state.pc = state.nextPC;
      state.nextPC = SE(insn&0x03FFFFFF, 26) << 2;
      if (opcode == 3) {
        state.lr = pc+8; // set the link-register to the instr after the delay slot instruction.
      }
      return keccak256(abi.encode(state));
    }

    // register fetch
    uint32 rs; // source register
    uint32 rt; // target register
    uint32 rtReg = ((insn >> 14) & 0x7C);

    // R-type or I-type (stores rt)
    rs = state.registers[(insn >> 19) & 0x7C];
    uint32 storeReg = (insn >> 14) & 0x7C;
    if (opcode == 0 || opcode == 0x1c) {
      // R-type (stores rd)
      rt = state.registers[rtReg];
      storeReg = (insn >> 9) & 0x7C;
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
      rt = state.registers[rtReg];

      // store actual rt with lwl and lwr
      storeReg = rtReg;
    }

    if ((opcode >= 4 && opcode < 8) || opcode == 1) {
      bool shouldBranch = false;

      if (opcode == 4 || opcode == 5) {   // beq/bne
        rt = state.registers[rtReg];
        shouldBranch = (rs == rt && opcode == 4) || (rs != rt && opcode == 5);
      } else if (opcode == 6) { shouldBranch = int32(rs) <= 0; // blez
      } else if (opcode == 7) { shouldBranch = int32(rs) > 0; // bgtz
      } else if (opcode == 1) {
        // regimm
        uint32 rtv = ((insn >> 16) & 0x1F);
        if (rtv == 0) shouldBranch = int32(rs) < 0;  // bltz
        if (rtv == 1) shouldBranch = int32(rs) >= 0; // bgez
      }

      state.pc = state.nextPC; // execute the delay slot first
      if (shouldBranch) {
        state.nextPC = pc + 4 + (SE(insn&0xFFFF, 16)<<2); // then continue with the instruction the branch jumps to.
      } else {
        state.nextPC = state.nextPC + 4; // branch not taken
      }
      return keccak256(abi.encode(state));
    }


    uint32 storeAddr = 0xFF_FF_FF_FF;
    // memory fetch (all I-type)
    // we do the load for stores also
    uint32 mem;
    if (opcode >= 0x20) {
      // M[R[rs]+SignExtImm]
      rs += SE(insn&0xFFFF, 16);
      uint32 addr = rs & 0xFFFFFFFC;
      // TODO proof memory read at addr
      assembly {
        mem := and(shr(sub(256, 64), calldataload(add(proof.offset, 0x20))), 0xFFFFFFFF)
      }
      if (opcode >= 0x28 && opcode != 0x30) {
        // store
        storeAddr = addr;
      }
    }

    // ALU
    uint32 val = execute(insn, rs, rt, mem);

    // TODO: this block can be before the execute call, and then share the mem read/writing
    if (opcode == 0 && func >= 8 && func < 0x1c) {
      if (func == 8 || func == 9) {
        // jr/jalr
        state.pc = state.nextPC;
        state.nextPC = rs;
        if (func == 9) {
          state.lr = pc+8; // set the link-register to the instr after the delay slot instruction.
        }
        return keccak256(abi.encode(state));
      }

      // handle movz and movn when they don't write back
      if (func == 0xa && rt != 0) { // movz
        storeReg = 0;
      }
      if (func == 0xb && rt == 0) { // movn
        storeReg = 0;
      }

      // syscall (can read and write)
      if (func == 0xC) {
        uint32 syscall_no = state.registers[2];
        uint32 v0 = 0;

        if (syscall_no == 4090) {
          // mmap
          uint32 a0 = state.registers[4];
          if (a0 == 0) {
            uint32 sz = state.registers[5];
            uint32 hr = state.heap;
            v0 = HEAP_START + hr;
            state.heap = hr+sz;
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
          state.exited = true;
          state.exitCode = uint8(state.registers[4]);
          return keccak256(abi.encode(state));
        }
        // TODO: pre-image oracle read/write

        state.registers[2] = v0;
        state.registers[7] = 0;
      }

      // lo and hi registers
      // can write back
      if (func >= 0x10 && func < 0x1c) {
        if (func == 0x10) val = state.hi; // mfhi
        else if (func == 0x11) state.hi = rs; // mthi
        else if (func == 0x12) val = state.lo; // mflo
        else if (func == 0x13) state.lo = rs; // mtlo
        else if (func == 0x18) { // mult
          uint64 acc = uint64(int64(int32(rs))*int64(int32(rt)));
          state.hi = uint32(acc>>32);
          state.lo = uint32(acc);
        } else if (func == 0x19) { // multu
          uint64 acc = uint64(uint64(rs)*uint64(rt));
          state.hi = uint32(acc>>32);
          state.lo = uint32(acc);
        } else if (func == 0x1a) { // div
          state.hi = uint32(int32(rs)%int32(rt));
          state.lo = uint32(int32(rs)/int32(rt));
        } else if (func == 0x1b) { // divu
          state.hi = rs%rt;
          state.lo = rs/rt;
        }
      }
    }

    // stupid sc, write a 1 to rt
    if (opcode == 0x38 && rtReg != 0) {
      state.registers[rtReg] = 1;
    }

    // write back
    if (storeReg != 0) {
      state.registers[storeReg] = val;
    }

    // write memory
    if (storeAddr != 0xFF_FF_FF_FF) {
      // TODO: write back memory change.
      // Note that we already read the same memory leaf earlier.
      // We can use that to shorten the proof significantly,
      // by just walking back up to construct the root with the same witness data.
      state.memRoot = bytes32(uint256(42));
    }

    state.pc = state.nextPC;
    state.nextPC = state.nextPC + 4;

    return keccak256(abi.encode(state));
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
