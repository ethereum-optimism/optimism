// SPDX-License-Identifier: MIT
pragma solidity ^0.7.3;

// https://inst.eecs.berkeley.edu/~cs61c/resources/MIPS_Green_Sheet.pdf
// https://uweb.engr.arizona.edu/~ece369/Resources/spim/MIPSReference.pdf

// This is a separate contract from the challenge contract
// Anyone can use it to validate a MIPS state transition
// First, to prepare, you call AddMerkleState, which adds valid state nodes in the stateHash. 
// If you are using the Preimage oracle, you call AddPreimage
// Then, you call Step. Step will revert if state is missing. If all state is present, it will return the next hash

contract MIPS {
  // This state is global
  mapping(bytes32 => mapping (uint32 => uint64)) public state;
  mapping(bytes32 => bytes) public preimage;

  function AddPreimage(bytes calldata anything) public {
    preimage[keccak256((anything))] = anything;
  }

  function AddMerkleState(bytes32 stateHash, uint32 addr, uint32 value, string calldata proof) public {
    // TODO: check proof
    state[stateHash][addr] = (1 << 32) | value;
  }

  uint32 constant REG_OFFSET = 0xc0000000;
  uint32 constant REG_PC = REG_OFFSET + 21*4;

  function getState(bytes32 stateHash, uint32 addr) public view returns (uint32) {
    if (addr == REG_OFFSET) {
      // zero register is always 0
      return 0;
    }
    assert(addr & 3 == 0);     // aligned access only
    uint64 ret = state[stateHash][addr];
    assert((ret >> 32) == 1);  // was set
    return uint32(ret);
  }


  // compute the next state
  // will revert if any input state is missing
  function Step(bytes32 stateHash) public view returns (bytes32) {
    // instruction fetch
    uint32 pc = getState(stateHash, REG_PC);
    uint32 insn = getState(stateHash, pc);
    uint32 opcode = insn >> 26; // 6-bits

    // decode

    // register fetch
    uint32 rs;
    uint32 rt;
    if (opcode != 2 && opcode != 3) {   // j and jal have no register fetch
      // R-type or I-type (stores rt)
      rs = getState(stateHash, REG_OFFSET + ((insn >> 19) & 0x7C));
      if (opcode == 0) {
        // R-type (stores rd)
        rt = getState(stateHash, REG_OFFSET + ((insn >> 14) & 0x7C));
      }
    }

    // memory fetch (all I-type)
    // we do the load for stores also
    uint32 mem;
    if (opcode >= 0x20) {
      // M[R[rs]+SignExtImm]
      uint32 SignExtImm = insn&0xFFFF | (insn&0x8000 != 0 ? 0xFFFF0000 : 0);
      mem = getState(stateHash, (rs + SignExtImm) & 0xFFFFFFFC);
    }

    // execute
    execute(insn, rs, rt, mem);

    // write back

  }

  function execute(uint32 insn, uint32 rs, uint32 rt, uint32 mem) public pure returns (uint32) {
    uint32 opcode = insn >> 26;    // 6-bits
    uint32 func = insn & 0x3f; // 6-bits
    // TODO: deref the immed into a register
    if (opcode == 0) {
      uint32 shamt = (insn >> 6) & 0x1f;
      // R-type (ArithLog)
      if (func == 0x20 || func == 0x21) { return rs+rt;   // add or addu
      } else if (func == 0x24) { return rs&rt;            // and
      } else if (func == 0x27) { return ~(rs|rt);         // nor
      } else if (func == 0x25) { return (rs|rt);          // or
      } else if (func == 0x22 || func == 0x23) {
        return rs-rt;   // sub or subu
      } else if (func == 0x2a) {
        return int32(rs)<int32(rt) ? 1 : 0; // slt
      } else if (func == 0x26) {
        return rs<rt ? 1 : 0;            // sltu
      // Shift and ShiftV
      } else if (func == 0x00) { return rt << shamt;      // sll
      } else if (func == 0x04) { return rt << rs;         // sllv
      } else if (func == 0x03) { return rt >> shamt;      // sra
      } else if (func == 0x07) { return rt >> rs;         // srav
      } else if (func == 0x02) { return rt >> shamt;      // srl
      } else if (func == 0x06) { return rt >> rs;         // srlv
      }
    } else if (func == 0x20) { return mem;   // lb
    } else if (func == 0x24) { return mem;   // lbu
    } else if (func == 0x21) { return mem;   // lh
    } else if (func == 0x25) { return mem;   // lhu
    } else if (func == 0x23) { return mem;   // lw
    } else if (func&0x3c == 0x28) { return rt;  // sb, sh, sw
    }
  }

}
