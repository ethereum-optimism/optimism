// SPDX-License-Identifier: MIT
pragma solidity 0.7.6;

interface IPreimageOracle {
  function readPreimage(bytes32 key, uint256 offset) external view returns (bytes32 dat, uint256 datLen);
}

// https://inst.eecs.berkeley.edu/~cs61c/resources/MIPS_Green_Sheet.pdf
// https://uweb.engr.arizona.edu/~ece369/Resources/spim/MIPSReference.pdf
// https://en.wikibooks.org/wiki/MIPS_Assembly/Instruction_Formats

// https://www.cs.cmu.edu/afs/cs/academic/class/15740-f97/public/doc/mips-isa.pdf
// page A-177

// MIPS linux kernel errors used by Go runtime:
// https://github.com/golang/go/blob/master/src/syscall/zerrors_linux_mips.go

// This MIPS contract emulates a single MIPS instruction.
//
// Note that delay slots are isolated instructions:
// the nextPC in the state pre-schedules where the VM jumps next.
//
// The Step input is a packed VM state, with binary-merkle-tree witness data for memory reads/writes.
// The Step outputs a keccak256 hash of the packed VM State, and logs the resulting state for offchain usage.
contract MIPS {

  struct State {
    bytes32 memRoot;
    bytes32 preimageKey;
    uint32 preimageOffset;
    uint32 pc;
    uint32 nextPC;  // State is executing a branch/jump delay slot if nextPC != pc+4
    uint32 lo;
    uint32 hi;
    uint32 heap;
    uint8 exitCode;
    bool exited;
    uint64 step;
    uint32[32] registers;
  }

  // total State size: 32+32+6*4+1+1+8+32*4 = 226 bytes

  uint32 constant public BRK_START = 0x40000000;

  uint32 constant FD_STDIN = 0;
  uint32 constant FD_STDOUT = 1;
  uint32 constant FD_STDERR = 2;
  uint32 constant FD_HINT_READ = 3;
  uint32 constant FD_HINT_WRITE = 4;
  uint32 constant FD_PREIMAGE_READ = 5;
  uint32 constant FD_PREIMAGE_WRITE = 6;

  uint32 constant EBADF = 0x9;
  uint32 constant EINVAL = 0x16;

  IPreimageOracle public oracle;

  function SE(uint32 dat, uint32 idx) internal pure returns (uint32) {
    bool isSigned = (dat >> (idx-1)) != 0;
    uint256 signed = ((1 << (32-idx)) - 1) << idx;
    uint256 mask = (1 << idx) - 1;
    return uint32(dat&mask | (isSigned ? signed : 0));
  }

  function outputState() internal returns (bytes32 out) {
    assembly {
      // copies 'size' bytes, right-aligned in word at 'from', to 'to', incl. trailing data
      function copyMem(from, to, size) -> fromOut, toOut {
        mstore(to, mload(add(from, sub(32, size))))
        fromOut := add(from, 32)
        toOut := add(to, size)
      }
      let from := 0x80 // state
      let start := mload(0x40) // free mem ptr
      let to := start
      from, to := copyMem(from, to, 32) // memRoot
      from, to := copyMem(from, to, 32) // preimageKey
      from, to := copyMem(from, to, 4) // preimageOffset
      from, to := copyMem(from, to, 4) // pc
      from, to := copyMem(from, to, 4) // nextPC
      from, to := copyMem(from, to, 4) // lo
      from, to := copyMem(from, to, 4) // hi
      from, to := copyMem(from, to, 4) // heap
      from, to := copyMem(from, to, 1) // exitCode
      from, to := copyMem(from, to, 1) // exited
      from, to := copyMem(from, to, 8) // step
      from := add(from, 32) // offset to registers
      for { let i := 0 } lt(i, 32) { i := add(i, 1) } { from, to := copyMem(from, to, 4) } // registers
      mstore(to, 0) // clean up end
      log0(start, sub(to, start)) // log the resulting MIPS state, for debugging
      out := keccak256(start, sub(to, start))
    }
    return out;
  }

  function handleSyscall() internal returns (bytes32) {
    State memory state;
    assembly {
      state := 0x80
    }
    uint32 syscall_no = state.registers[2];
    uint32 v0 = 0;
    uint32 v1 = 0;

    uint32 a0 = state.registers[4];
    uint32 a1 = state.registers[5];
    uint32 a2 = state.registers[6];

    if (syscall_no == 4090) {
      // mmap
      uint32 sz = a1;
      if (sz&4095 != 0) { // adjust size to align with page size
        sz += 4096 - (sz&4095);
      }
      if (a0 == 0) {
        v0 = state.heap;
        state.heap += sz;
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
      state.exitCode = uint8(a0);
      return outputState();
    } else if (syscall_no == 4003) { // read
      // args: a0 = fd, a1 = addr, a2 = count
      // returns: v0 = read, v1 = err code
      if (a0 == FD_STDIN) {
        // leave v0 and v1 zero: read nothing, no error
      } else if (a0 == FD_PREIMAGE_READ) { // pre-image oracle
        // verify proof 1 is correct, and get the existing memory.
        uint32 mem = readMem(a1 & 0xFFffFFfc, 1); // mask the addr to align it to 4 bytes
        (bytes32 dat, uint256 datLen) = oracle.readPreimage(state.preimageKey, state.preimageOffset);
        assembly { // assembly for more precise ops, and no var count limit
          let alignment := and(a1, 3) // the read might not start at an aligned address
          let space := sub(4, alignment) // remaining space in memory word
          if lt(space, datLen) { datLen := space } // if less space than data, shorten data
          if lt(a2, datLen) { datLen := a2 } // if requested to read less, read less
          dat := shr(sub(256, mul(datLen, 8)), dat) // right-align data
          dat := shl(mul(alignment, 8), dat) // position data to insert into memory word
          let mask := sub(shl(mul(sub(4, alignment), 8), 1), 1) // mask all bytes after start
          let suffixMask := sub(shl(mul(add(sub(4, alignment), datLen), 8), 1), 1) // mask of all bytes starting from end, maybe none
          mask := and(mask, not(suffixMask)) // reduce mask to just cover the data we insert
          mem := or(and(mem, not(mask)), dat) // clear masked part of original memory, and insert data
        }
        writeMem(a1 & 0xFFffFFfc, 1, mem);
        state.preimageOffset += uint32(datLen);
        v0 = uint32(datLen);
      } else if (a0 == FD_HINT_READ) { // hint response
        // don't actually read into memory, just say we read it all, we ignore the result anyway
        v0 = a2;
      } else {
        v0 = 0xFFffFFff;
        v1 = EBADF;
      }
    } else if (syscall_no == 4004) { // write
      // args: a0 = fd, a1 = addr, a2 = count
      // returns: v0 = written, v1 = err code
      if (a0 == FD_STDOUT || a0 == FD_STDERR || a0 == FD_HINT_WRITE) {
        v0 = a2; // tell program we have written everything
      } else if (a0 == FD_PREIMAGE_WRITE) { // pre-image oracle
        uint32 mem = readMem(a1 & 0xFFffFFfc, 1); // mask the addr to align it to 4 bytes
        bytes32 key = state.preimageKey;
        assembly { // assembly for more precise ops, and no var count limit
          let alignment := and(a1, 3) // the read might not start at an aligned address
          let space := sub(4, alignment) // remaining space in memory word
          if lt(space, a2) { a2 := space } // if less space than data, shorten data
          key := shl(mul(a2, 8), key) // shift key, make space for new info
          let mask := sub(shl(mul(a2, 8), 1), 1) // mask for extracting value from memory
          mem := and(shr(mul(sub(space, a2), 8), mem), mask) // align value to right, mask it
          key := or(key, mem) // insert into key
        }
        state.preimageKey = key;
        state.preimageOffset = 0; // reset offset, to read new pre-image data from the start
        v0 = a2;
      } else {
        v0 = 0xFFffFFff;
        v1 = EBADF;
      }
    } else if (syscall_no == 4055) { // fcntl
      // args: a0 = fd, a1 = cmd
      if (a1 == 3) { // F_GETFL: get file descriptor flags
        if (a0 == FD_STDIN || a0 == FD_PREIMAGE_READ || a0 == FD_HINT_READ) {
          v0 = 0; // O_RDONLY
        } else if (a0 == FD_STDOUT || a0 == FD_STDERR || a0 == FD_PREIMAGE_WRITE || a0 == FD_HINT_WRITE) {
          v0 = 1; // O_WRONLY
        } else {
          v0 = 0xFFffFFff;
          v1 = EBADF;
        }
      } else {
        v0 = 0xFFffFFff;
        v1 = EINVAL; // cmd not recognized by this kernel
      }
    }

    state.registers[2] = v0;
    state.registers[7] = v1;

    state.pc = state.nextPC;
    state.nextPC = state.nextPC + 4;

    return outputState();
  }

  function handleBranch(uint32 opcode, uint32 insn, uint32 rtReg, uint32 rs) internal returns (bytes32) {
    State memory state;
    assembly {
      state := 0x80
    }
    bool shouldBranch = false;

    if (opcode == 4 || opcode == 5) {   // beq/bne
      uint32 rt = state.registers[rtReg];
      shouldBranch = (rs == rt && opcode == 4) || (rs != rt && opcode == 5);
    } else if (opcode == 6) { shouldBranch = int32(rs) <= 0; // blez
    } else if (opcode == 7) { shouldBranch = int32(rs) > 0; // bgtz
    } else if (opcode == 1) {
      // regimm
      uint32 rtv = ((insn >> 16) & 0x1F);
      if (rtv == 0) shouldBranch = int32(rs) < 0;  // bltz
      if (rtv == 1) shouldBranch = int32(rs) >= 0; // bgez
    }

    uint32 prevPC = state.pc;
    state.pc = state.nextPC; // execute the delay slot first
    if (shouldBranch) {
      state.nextPC = prevPC + 4 + (SE(insn&0xFFFF, 16)<<2); // then continue with the instruction the branch jumps to.
    } else {
      state.nextPC = state.nextPC + 4; // branch not taken
    }

    return outputState();
  }

  function handleHiLo(uint32 func, uint32 rs, uint32 rt, uint32 storeReg) internal returns (bytes32) {
    State memory state;
    assembly {
      state := 0x80
    }
    uint32 val;
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

    if (storeReg != 0) {
      state.registers[storeReg] = val;
    }

    state.pc = state.nextPC;
    state.nextPC = state.nextPC + 4;

    return outputState();
  }

  function handleJump(uint32 linkReg, uint32 dest) internal returns (bytes32) {
    State memory state;
    assembly {
      state := 0x80
    }
    uint32 prevPC = state.pc;
    state.pc = state.nextPC;
    state.nextPC = dest;
    if (linkReg != 0) {
      state.registers[linkReg] = prevPC+8; // set the link-register to the instr after the delay slot instruction.
    }
    return outputState();
  }

  function handleRd(uint32 storeReg, uint32 val, bool conditional) internal returns (bytes32) {
    State memory state;
    assembly {
      state := 0x80
    }
    require(storeReg < 32, "valid register");
    // never write to reg 0, and it can be conditional (movz, movn)
    if (storeReg != 0 && conditional) {
      state.registers[storeReg] = val;
    }

    state.pc = state.nextPC;
    state.nextPC = state.nextPC + 4;

    return outputState();
  }

  function proofOffset(uint8 proofIndex) internal pure returns (uint256 offset) {
    // A proof of 32 bit memory, with 32-byte leaf values, is (32-5)=27 bytes32 entries.
    // And the leaf value itself needs to be encoded as well. And proof.offset == 358
    offset = 358 + (uint256(proofIndex) * (28*32));
    uint256 s = 0;
    assembly { s := calldatasize() }
    require(s >= (offset + 28*32), "check that there is enough calldata");
    return offset;
  }

  function readMem(uint32 addr, uint8 proofIndex) internal pure returns (uint32 out) {
    uint256 offset = proofOffset(proofIndex);
    assembly {
      if and(addr, 3) { revert(0, 0) } // quick addr alignment check
      let leaf := calldataload(offset)
      offset := add(offset, 32)
      function hashPair(a, b) -> h {
        mstore(0, a) // use scratch space slots to hash the two nodes together
        mstore(32, b)
        h := keccak256(0, 64)
      }
      let path := shr(5, addr)
      let node := leaf // starting from the leaf node, work back up by combining with siblings, to reconstruct the root
      for { let i := 0 } lt(i, 27) { i := add(i, 1) } {
        let sibling := calldataload(offset)
        offset := add(offset, 32)
        switch and(shr(i, path), 1)
        case 0 { node := hashPair(node, sibling) }
        case 1 { node := hashPair(sibling, node) }
      }
      let memRoot := mload(0x80) // load memRoot, first field of state
      if iszero(eq(node, memRoot)) { // verify the root matches
        mstore(0, 0x0badf00d)
        revert(0, 32)
      }
      // bits to shift = (32 - 4 - (addr % 32)) * 8
      let shamt := shl(3, sub(sub(32, 4), and(addr, 31)))
      out := and(shr(shamt, leaf), 0xFFffFFff)
    }
    return out;
  }

  // writeMem writes the value by first overwriting the part of the leaf, and then recomputing the memory merkle root.
  function writeMem(uint32 addr, uint8 proofIndex, uint32 value) internal pure {
    uint256 offset = proofOffset(proofIndex);
    assembly {
      if and(addr, 3) { revert(0, 0) } // quick addr alignment check
      let leaf := calldataload(offset)
      let shamt := shl(3, sub(sub(32, 4), and(addr, 31)))
      // mask out 4 bytes, and OR in the value
      leaf := or(and(leaf, not(shl(shamt, 0xFFffFFff))), shl(shamt, value))
      offset := add(offset, 32)
      function hashPair(a, b) -> h {
        mstore(0, a) // use scratch space slots to hash the two nodes together
        mstore(32, b)
        h := keccak256(0, 64)
      }
      let path := shr(5, addr)
      let node := leaf // starting from the leaf node, work back up by combining with siblings, to reconstruct the root
      for { let i := 0 } lt(i, 27) { i := add(i, 1) } {
        let sibling := calldataload(offset)
        offset := add(offset, 32)
        switch and(shr(i, path), 1)
        case 0 { node := hashPair(node, sibling) }
        case 1 { node := hashPair(sibling, node) }
      }
      mstore(0x80, node) // store new memRoot, first field of state
    }
  }

  // will revert if any required input state is missing
  function Step(bytes calldata stateData, bytes calldata proof) public returns (bytes32) {
    State memory state;
    // packed data is ~6 times smaller
    assembly {
      if iszero(eq(state, 0x80)) { // expected state mem offset check
        revert(0,0)
      }
      if iszero(eq(mload(0x40), mul(32, 48))) { // expected memory check
        revert(0,0)
      }
      if iszero(eq(stateData.offset, 100)) { // 32*3+4=100 expected state data offset
        revert(0,0)
      }
      if iszero(eq(proof.offset, 358)) { // 100+32+226=358 expected proof offset
        revert(0,0)
      }
      function putField(callOffset, memOffset, size) -> callOffsetOut, memOffsetOut {
        // calldata is packed, thus starting left-aligned, shift-right to pad and right-align
        let w := shr(shl(3, sub(32, size)), calldataload(callOffset))
        mstore(memOffset, w)
        callOffsetOut := add(callOffset, size)
        memOffsetOut := add(memOffset, 32)
      }
      let c := stateData.offset // calldata offset
      let m := 0x80 // mem offset
      c, m := putField(c, m, 32) // memRoot
      c, m := putField(c, m, 32) // preimageKey
      c, m := putField(c, m, 4) // preimageOffset
      c, m := putField(c, m, 4) // pc
      c, m := putField(c, m, 4) // nextPC
      c, m := putField(c, m, 4) // lo
      c, m := putField(c, m, 4) // hi
      c, m := putField(c, m, 4) // heap
      c, m := putField(c, m, 1) // exitCode
      c, m := putField(c, m, 1) // exited
      c, m := putField(c, m, 8) // step
      mstore(m, add(m, 32)) // offset to registers
      m := add(m, 32)
      for { let i := 0 } lt(i, 32) { i := add(i, 1) } { c, m := putField(c, m, 4) } // registers
    }
    if(state.exited) { // don't change state once exited
      return outputState();
    }
    state.step += 1;

    // instruction fetch
    uint32 insn = readMem(state.pc, 0);

    uint32 opcode = insn >> 26; // 6-bits

    // j-type j/jal
    if (opcode == 2 || opcode == 3) {
      // TODO(CLI-4136): likely bug in original code: MIPS spec says this should be in the "current" region;
      // a 256 MB aligned region (i.e. use top 4 bits of branch delay slot (pc+4))
      return handleJump(opcode == 2 ? 0 : 31, SE(insn&0x03FFFFFF, 26) << 2);
    }

    // register fetch
    uint32 rs; // source register 1 value
    uint32 rt; // source register 2 / temp value
    uint32 rtReg = (insn >> 16) & 0x1F;

    // R-type or I-type (stores rt)
    rs = state.registers[(insn >> 21) & 0x1F];
    uint32 rdReg = rtReg;
    if (opcode == 0 || opcode == 0x1c) {
      // R-type (stores rd)
      rt = state.registers[rtReg];
      rdReg = (insn >> 11) & 0x1F;
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
      rdReg = rtReg;
    }

    if ((opcode >= 4 && opcode < 8) || opcode == 1) {
      return handleBranch(opcode, insn, rtReg, rs);
    }

    uint32 storeAddr = 0xFF_FF_FF_FF;
    // memory fetch (all I-type)
    // we do the load for stores also
    uint32 mem;
    if (opcode >= 0x20) {
      // M[R[rs]+SignExtImm]
      rs += SE(insn&0xFFFF, 16);
      uint32 addr = rs & 0xFFFFFFFC;
      mem = readMem(addr, 1);
      if (opcode >= 0x28 && opcode != 0x30) {
        // store
        storeAddr = addr;
        // store opcodes don't write back to a register
        rdReg = 0;
      }
    }

    // ALU
    uint32 val = execute(insn, rs, rt, mem) & 0xffFFffFF; // swr outputs more than 4 bytes without the mask

    uint32 func = insn & 0x3f; // 6-bits
    if (opcode == 0 && func >= 8 && func < 0x1c) {
      if (func == 8 || func == 9) { // jr/jalr
        return handleJump(func == 8 ? 0 : rdReg, rs);
      }

      if (func == 0xa) { // movz
        return handleRd(rdReg, rs, rt == 0);
      }
      if (func == 0xb) { // movn
        return handleRd(rdReg, rs, rt != 0);
      }

      // syscall (can read and write)
      if (func == 0xC) {
        return handleSyscall();
      }

      // lo and hi registers
      // can write back
      if (func >= 0x10 && func < 0x1c) {
        return handleHiLo(func, rs, rt, rdReg);
      }
    }

    // stupid sc, write a 1 to rt
    if (opcode == 0x38 && rtReg != 0) {
      state.registers[rtReg] = 1;
    }

    // write memory
    if (storeAddr != 0xFF_FF_FF_FF) {
      writeMem(storeAddr, 1, val);
    }

    // write back the value to destination register
    return handleRd(rdReg, val, true);
  }

  function execute(uint32 insn, uint32 rs, uint32 rt, uint32 mem) internal pure returns (uint32) {
    uint32 opcode = insn >> 26;    // 6-bits
    uint32 func = insn & 0x3f; // 6-bits
    // TODO(CLI-4136): deref the immed into a register

    if (opcode < 0x20) {
      // transform ArithLogI
      // TODO(CLI-4136): replace with table
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
