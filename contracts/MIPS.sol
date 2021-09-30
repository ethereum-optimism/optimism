// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;

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

interface IMIPSMemory {
  function ReadMemory(bytes32 stateHash, uint32 addr) external view returns (uint32);
  function ReadBytes32(bytes32 stateHash, uint32 addr) external view returns (bytes32);
  function WriteMemory(bytes32 stateHash, uint32 addr, uint32 val) external pure returns (bytes32);
}

contract MIPS {
  IMIPSMemory public immutable m;

  uint32 constant public REG_OFFSET = 0xc0000000;
  uint32 constant public REG_PC = REG_OFFSET + 0x20*4;

  constructor(IMIPSMemory _m) {
    m = _m;
  }

  function Steps(bytes32 stateHash, uint count) public view returns (bytes32) {
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

    /*if (idx == 16) {
      return dat&0xFFFF | (dat&0x8000 != 0 ? 0xFFFF0000 : 0);
    } else if (idx == 8) {
      return dat&0xFF | (dat&0x80 != 0 ? 0xFFFFFF00 : 0);
    }
    return dat;*/
  }

  // will revert if any required input state is missing
  function Step(bytes32 stateHash) public view returns (bytes32) {
    // instruction fetch
    uint32 pc = m.ReadMemory(stateHash, REG_PC);
    if (pc == 0xdead0000) {
      return stateHash;
    }
    uint32 insn = m.ReadMemory(stateHash, pc);
    uint32 opcode = insn >> 26; // 6-bits
    uint32 func = insn & 0x3f; // 6-bits

    // decode

    // register fetch
    uint32 storeAddr = 0xFFFFFFFF;
    uint32 rs;
    uint32 rt;
    if (opcode != 2 && opcode != 3) {   // J-type: j and jal have no register fetch
      // R-type or I-type (stores rt)
      rs = m.ReadMemory(stateHash, REG_OFFSET + ((insn >> 19) & 0x7C));
      storeAddr = REG_OFFSET + ((insn >> 14) & 0x7C);
      if (opcode == 0) {
        // R-type (stores rd)
        rt = m.ReadMemory(stateHash, REG_OFFSET + ((insn >> 14) & 0x7C));
        storeAddr = REG_OFFSET + ((insn >> 9) & 0x7C);
      } else if (opcode < 0x20) {
        // rt is SignExtImm
        if (opcode == 0xC || opcode == 0xD) {
          // ZeroExtImm
          rt = insn&0xFFFF;
        } else {
          // SignExtImm
          rt = SE(insn&0xFFFF, 16);
        }
      } else if (opcode >= 0x28 || opcode == 0x22 || opcode == 0x26) {
        // store rt value with store
        rt = m.ReadMemory(stateHash, REG_OFFSET + ((insn >> 14) & 0x7C));

        // store actual rt with lwl and lwr
        storeAddr = REG_OFFSET + ((insn >> 14) & 0x7C);
      }
    }

    // memory fetch (all I-type)
    // we do the load for stores also
    uint32 mem;
    if (opcode >= 0x20) {
      // M[R[rs]+SignExtImm]
      uint32 SignExtImm = SE(insn&0xFFFF, 16);
      rs += SignExtImm;
      uint32 addr = rs & 0xFFFFFFFC;
      mem = m.ReadMemory(stateHash, addr);
      if (opcode >= 0x28) {
        // store
        storeAddr = addr;
      }
    }

    if (opcode == 0 && func == 8) {
      // jr
      storeAddr = REG_PC;
    }

    // execute
    uint32 val = execute(insn, rs, rt, mem);

    // write back
    if (storeAddr != 0xFFFFFFFF) {
      // does this ever not happen?
      stateHash = m.WriteMemory(stateHash, storeAddr, val);
    }
    if (storeAddr != REG_PC) {
      stateHash = m.WriteMemory(stateHash, REG_PC, pc+4);
    }

    return stateHash;
  }

  // TODO: move pure testable stuff to LibMIPS.sol
  function execute(uint32 insn, uint32 rs, uint32 rt, uint32 mem) public pure returns (uint32) {
    uint32 opcode = insn >> 26;    // 6-bits
    uint32 func = insn & 0x3f; // 6-bits
    // TODO: deref the immed into a register

    // transform ArithLogI
    // TODO: replace with table
    if (opcode == 8) { opcode = 0; func = 0x20; }        // addi
    else if (opcode == 9) { opcode = 0; func = 0x21; }   // addiu
    else if (opcode == 0xa) { opcode = 0; func = 0x2a; } // slti
    else if (opcode == 0xb) { opcode = 0; func = 0x2B; } // sltiu
    else if (opcode == 0xc) { opcode = 0; func = 0x24; } // andi
    else if (opcode == 0xd) { opcode = 0; func = 0x25; } // ori
    else if (opcode == 0xe) { opcode = 0; func = 0x26; } // xori

    // 0 is opcode SPECIAL
    if (opcode == 0) {
      uint32 shamt = (insn >> 6) & 0x1f;
      // Shift and ShiftV
      if (func == 0x00) { return rt << shamt;      // sll
      } else if (func == 0x02) { return rt >> shamt;      // srl
      } else if (func == 0x03) { return SE(rt >> shamt, 32-shamt);      // sra
      } else if (func == 0x04) { return rt << rs;         // sllv
      } else if (func == 0x06) { return rt >> rs;         // srlv
      } else if (func == 0x07) { return SE(rt >> rs, 32-rs);     // srav
      } else if (func == 0x08) { return rs;               // jr
      } else if (func == 0x09) { return rs;               // jalr
      // 0x10-0x13 = mfhi, mthi, mflo, mtlo
      // R-type (ArithLog)
      } else if (func == 0x20 || func == 0x21) { return rs+rt;   // add or addu
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
    } else if (opcode == 0x20) {  // lb
      return SE((mem >> (24-(rs&3)*8)) & 0xFF, 8);
    } else if (opcode == 0x21) {  // lh
      return SE((mem >> (16-(rs&2)*8)) & 0xFFFF, 16);
    } else if (opcode == 0x22) {  // lwl
      return mem&0xFFFF0000 | rt&0xFFFF;
    } else if (opcode == 0x23) { return mem;   // lw
    } else if (opcode == 0x24) {  // lbu
      return (mem >> (24-(rs&3)*8)) & 0xFF;
    } else if (opcode == 0x25) {  // lhu
      return (mem >> (16-(rs&2)*8)) & 0xFFFF;
    } else if (opcode == 0x26) {  // lwr
      return rt&0xFFFF0000 | mem&0xFFFF;
    } else if (opcode&0x3c == 0x28) { return rt;  // sb, sh, sw
    } else if (opcode == 0xf) { return rt<<16; // lui
    }
  }

}
