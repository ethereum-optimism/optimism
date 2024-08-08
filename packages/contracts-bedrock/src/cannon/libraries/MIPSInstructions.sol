// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { MIPSMemory } from "src/cannon/libraries/MIPSMemory.sol";
import { MIPSState as st } from "src/cannon/libraries/MIPSState.sol";

library MIPSInstructions {
    uint64 private constant HALF_MASK = uint64(~uint16(0));
    uint64 private constant WORD_MASK = uint64(~uint32(0));
    uint64 private constant DOUBLEWORD_MASK = ~uint64(0);

    /// @param _pc The program counter.
    /// @param _memRoot The current memory root.
    /// @param _insnProofOffset The calldata offset of the memory proof for the current instruction.
    /// @return insn_ The current 32-bit instruction at the pc.
    /// @return opcode_ The opcode value parsed from insn_.
    /// @return fun_ The function value parsed from insn_.
    function getInstructionDetails(
        uint64 _pc,
        bytes32 _memRoot,
        uint256 _insnProofOffset
    )
        internal
        pure
        returns (uint32 insn_, uint64 opcode_, uint64 fun_)
    {
        unchecked {
            insn_ = MIPSMemory.readMem(_memRoot, _pc, _insnProofOffset);
            opcode_ = insn_ >> 26; // First 6-bits
            fun_ = insn_ & 0x3f; // Last 6-bits

            return (insn_, opcode_, fun_);
        }
    }

    /// @notice Execute core MIPS step logic.
    /// @notice _cpu The CPU scalar fields.
    /// @notice _registers The CPU registers.
    /// @notice _memRoot The current merkle root of the memory.
    /// @notice _memProofOffset The offset in calldata specify where the memory merkle proof is located.
    /// @param _insn The current 32-bit instruction at the pc.
    /// @param _opcode The opcode value parsed from insn_.
    /// @param _fun The function value parsed from insn_.
    /// @return newMemRoot_ The updated merkle root of memory after any modifications, may be unchanged.
    function execMipsCoreStepLogic(
        st.CpuScalars memory _cpu,
        uint64[32] memory _registers,
        bytes32 _memRoot,
        uint256 _memProofOffset,
        uint32 _insn,
        uint64 _opcode,
        uint64 _fun
    )
        internal
        returns (bytes32 newMemRoot_)
    {
        unchecked {
            newMemRoot_ = _memRoot;

            // j-type j/jal
            if (_opcode == 2 || _opcode == 3) {
                // Take top 4 bits of the next PC (its 256 MB region), and concatenate with the 26-bit offset
                uint64 target = (_cpu.nextPC & 0xFFFFFFFFF0000000) | (_insn & 0x03FFFFFF) << 2;
                handleJump(_cpu, _registers, _opcode == 2 ? 0 : 31, target);
                return newMemRoot_;
            }

            // register fetch
            uint64 rs = 0; // source register 1 value
            uint64 rt = 0; // source register 2 / temp value
            uint64 rtReg = (_insn >> 16) & 0x1F;

            // R-type or I-type (stores rt)
            rs = _registers[(_insn >> 21) & 0x1F];
            uint64 rdReg = rtReg;

            if (_opcode == 0 || _opcode == 0x1c) {
                // R-type (stores rd)
                rt = _registers[rtReg];
                rdReg = (_insn >> 11) & 0x1F;
            } else if (_opcode < 0x20) {
                // rt is SignExtImm
                // don't sign extend for andi, ori, xori
                if (_opcode == 0xC || _opcode == 0xD || _opcode == 0xe) {
                    // ZeroExtImm
                    rt = _insn & HALF_MASK;
                } else {
                    // SignExtImm
                    rt = signExtend(_insn & HALF_MASK, 16);
                }
            } else if (_opcode >= 0x28 || _opcode == 0x22 || _opcode == 0x26 || _opcode == 0x1a || _opcode == 0x1b) {
                // store rt value with store
                rt = _registers[rtReg];

                // store actual rt with lwl, lwr, ldl, and ldr
                rdReg = rtReg;
            }

            if ((_opcode >= 4 && _opcode < 8) || _opcode == 1) {
                handleBranch({
                    _cpu: _cpu,
                    _registers: _registers,
                    _opcode: _opcode,
                    _insn: _insn,
                    _rtReg: rtReg,
                    _rs: rs
                });
                return newMemRoot_;
            }

            uint64 storeAddr = ~uint64(0);
            // memory fetch (all I-type)
            // we do the load for stores also
            uint64 mem = 0;
            if (_opcode >= 0x20) {
                // M[R[rs]+SignExtImm]
                rs += signExtend(_insn & HALF_MASK, 16);
                uint64 addr = rs & 0xFFFFFFFFFFFFFFF8;
                mem = MIPSMemory.readMemDoubleword(_memRoot, addr, _memProofOffset);
                if (_opcode >= 0x28 && _opcode != 0x30 && _opcode != 0x34 && _opcode != 0x37) {
                    // store
                    storeAddr = addr;
                    // store opcodes don't write back to a register
                    rdReg = 0;
                }
            }

            // ALU
            uint64 val = executeMipsInstruction(_insn, _opcode, _fun, rs, rt, mem);

            if (_opcode == 0 && _fun >= 8 && _fun < 0x20) {
                if (_fun == 8 || _fun == 9) {
                    // jr/jalr
                    handleJump(_cpu, _registers, _fun == 8 ? 0 : rdReg, rs);
                    return newMemRoot_;
                }

                if (_fun == 0xa) {
                    // movz
                    handleRd(_cpu, _registers, rdReg, rs, rt == 0);
                    return newMemRoot_;
                }
                if (_fun == 0xb) {
                    // movn
                    handleRd(_cpu, _registers, rdReg, rs, rt != 0);
                    return newMemRoot_;
                }

                // lo and hi registers, as well as mips64 shift ops
                // can write back
                if (_fun >= 0x10 && _fun < 0x20) {
                    handleHiLo({ _cpu: _cpu, _registers: _registers, _fun: _fun, _rs: rs, _rt: rt, _storeReg: rdReg });

                    return newMemRoot_;
                }
            }

            // stupid sc, write a 1 to rt
            if ((_opcode == 0x38 || _opcode == 0x3c) && rtReg != 0) {
                _registers[rtReg] = 1;
            }

            // write memory
            if (storeAddr != ~uint64(0)) {
                newMemRoot_ = MIPSMemory.writeMemDoubleword(storeAddr, _memProofOffset, val);
            }

            // write back the value to destination register
            handleRd(_cpu, _registers, rdReg, val, true);

            return newMemRoot_;
        }
    }

    /// @notice Execute an instruction.
    function executeMipsInstruction(
        uint32 _insn,
        uint64 _opcode,
        uint64 _fun,
        uint64 _rs,
        uint64 _rt,
        uint64 _mem
    )
        internal
        returns (uint64 out_)
    {
        unchecked {
            if (_opcode == 0 || (_opcode >= 8 && _opcode < 0xF) || _opcode == 0x18 || _opcode == 0x19) {
                assembly {
                    // transform ArithLogI to SPECIAL
                    switch _opcode
                    // addi
                    case 0x8 { _fun := 0x20 }
                    // addiu
                    case 0x9 { _fun := 0x21 }
                    // stli
                    case 0xA { _fun := 0x2A }
                    // sltiu
                    case 0xB { _fun := 0x2B }
                    // andi
                    case 0xC { _fun := 0x24 }
                    // ori
                    case 0xD { _fun := 0x25 }
                    // xori
                    case 0xE { _fun := 0x26 }
                    // daddi
                    case 0x18 { _fun := 0x2c }
                    // daddiu
                    case 0x19 { _fun := 0x2d }
                }

                // sll
                if (_fun == 0x00) {
                    uint64 res = (_rt & WORD_MASK) << ((_insn >> 6) & 0x1F);
                    return signExtend(res, 32);
                }
                // srl
                else if (_fun == 0x02) {
                    uint64 res = (_rt & WORD_MASK) >> ((_insn >> 6) & 0x1F);
                    return signExtend(res, 32);
                }
                // sra
                else if (_fun == 0x03) {
                    uint32 shamt = (_insn >> 6) & 0x1F;
                    return signExtend((_rt & WORD_MASK) >> shamt, 32 - shamt);
                }
                // sllv
                else if (_fun == 0x04) {
                    uint64 res = (_rt & WORD_MASK) << (_rs & 0x1F);
                    return signExtend(res, 32);
                }
                // srlv
                else if (_fun == 0x6) {
                    uint64 res = (_rt & WORD_MASK) >> (_rs & 0x1F);
                    return signExtend(res, 32);
                }
                // srav
                else if (_fun == 0x07) {
                    return signExtend((_rt & WORD_MASK) >> _rs, 32 - _rs);
                }
                // functs in range [0x8, 0x1f] are handled specially by other functions
                // Explicitly enumerate each funct in range to reduce code diff against Go Vm
                // jr
                else if (_fun == 0x08) {
                    return _rs;
                }
                // jalr
                else if (_fun == 0x09) {
                    return _rs;
                }
                // movz
                else if (_fun == 0x0a) {
                    return _rs;
                }
                // movn
                else if (_fun == 0x0b) {
                    return _rs;
                }
                // syscall
                else if (_fun == 0x0c) {
                    return _rs;
                }
                // 0x0d - break not supported
                // sync
                else if (_fun == 0x0f) {
                    return _rs;
                }
                // mfhi
                else if (_fun == 0x10) {
                    return _rs;
                }
                // mthi
                else if (_fun == 0x11) {
                    return _rs;
                }
                // mflo
                else if (_fun == 0x12) {
                    return _rs;
                }
                // mtlo
                else if (_fun == 0x13) {
                    return _rs;
                }
                // dsllv
                else if (_fun == 0x14) {
                    return _rs;
                }
                // dsrlv
                else if (_fun == 0x16) {
                    return _rs;
                }
                // dsrav
                else if (_fun == 0x17) {
                    return _rs;
                }
                // mult
                else if (_fun == 0x18) {
                    return _rs;
                }
                // multu
                else if (_fun == 0x19) {
                    return _rs;
                }
                // div
                else if (_fun == 0x1a) {
                    return _rs;
                }
                // divu
                else if (_fun == 0x1b) {
                    return _rs;
                }
                // dmult
                else if (_fun == 0x1c) {
                    return _rs;
                }
                // dmultu
                else if (_fun == 0x1d) {
                    return _rs;
                }
                // ddiv
                else if (_fun == 0x1e) {
                    return _rs;
                }
                // ddivu
                else if (_fun == 0x1f) {
                    return _rs;
                }
                // The rest includes transformed R-type arith imm instructions
                // add
                else if (_fun == 0x20) {
                    return signExtend(uint32(_rs) + uint32(_rt), 32);
                }
                // addu
                else if (_fun == 0x21) {
                    return signExtend(uint32(_rs) + uint32(_rt), 32);
                }
                // sub
                else if (_fun == 0x22) {
                    return signExtend(uint32(_rs) - uint32(_rt), 32);
                }
                // subu
                else if (_fun == 0x23) {
                    return signExtend(uint32(_rs) - uint32(_rt), 32);
                }
                // and
                else if (_fun == 0x24) {
                    return (_rs & _rt);
                }
                // or
                else if (_fun == 0x25) {
                    return (_rs | _rt);
                }
                // xor
                else if (_fun == 0x26) {
                    return (_rs ^ _rt);
                }
                // nor
                else if (_fun == 0x27) {
                    return ~(_rs | _rt);
                }
                // slti
                else if (_fun == 0x2a) {
                    return int64(_rs) < int64(_rt) ? 1 : 0;
                }
                // sltiu
                else if (_fun == 0x2b) {
                    return _rs < _rt ? 1 : 0;
                }
                // dadd
                else if (_fun == 0x2c) {
                    return _rs + _rt;
                }
                // daddu
                else if (_fun == 0x2d) {
                    return _rs + _rt;
                }
                // dsub
                else if (_fun == 0x2e) {
                    return _rs - _rt;
                }
                // dsubu
                else if (_fun == 0x2f) {
                    return _rs - _rt;
                }
                // dsll
                else if (_fun == 0x38) {
                    return _rt << ((_insn >> 6) & 0x1f);
                }
                // dsrl
                else if (_fun == 0x3a) {
                    return _rt >> ((_insn >> 6) & 0x1f);
                }
                // dsra
                else if (_fun == 0x3b) {
                    return uint64(int64(_rt) >> ((_insn >> 6) & 0x1f));
                }
                // dsll32
                else if (_fun == 0x3c) {
                    return _rt << (((_insn >> 6) & 0x1f) + 32);
                }
                // dsrl32
                else if (_fun == 0x3e) {
                    return _rt >> (((_insn >> 6) & 0x1f) + 32);
                }
                // dsra32
                else if (_fun == 0x3f) {
                    return uint64(int64(_rt) >> (((_insn >> 6) & 0x1f) + 32));
                }
            } else {
                // SPECIAL2
                if (_opcode == 0x1C) {
                    // mul
                    if (_fun == 0x2) {
                        return signExtend(uint32(int32(int64(_rs)) * int32(int64(_rt))), 32);
                    }
                    // clz, clo
                    else if (_fun == 0x20 || _fun == 0x21) {
                        if (_fun == 0x20) {
                            _rs = ~_rs;
                        }
                        uint32 i = 0;
                        while (_rs & 0x80000000 != 0) {
                            i++;
                            _rs <<= 1;
                        }
                        return i;
                    }
                }
                // lui
                else if (_opcode == 0x0F) {
                    return signExtend(_rt << 16, 32);
                }
                // lb
                else if (_opcode == 0x20) {
                    return signExtend((_mem >> (56 - (_rs & 7) * 8)) & 0xFF, 8);
                }
                // lh
                else if (_opcode == 0x21) {
                    return signExtend((_mem >> (48 - (_rs & 6) * 8)) & HALF_MASK, 16);
                }
                // lwl
                else if (_opcode == 0x22) {
                    uint64 val = _mem << ((_rs & 3) * 8);
                    uint64 mask = uint64(uint32(WORD_MASK) << ((uint32(_rs) & 3) * 8));
                    return signExtend(((_rt & ~mask) | val) & WORD_MASK, 32);
                }
                // lw
                else if (_opcode == 0x23) {
                    uint64 res = (_mem >> (32 - ((_rs & 0x4) << 3))) & WORD_MASK;
                    return signExtend(res, 32);
                }
                // lbu
                else if (_opcode == 0x24) {
                    return (_mem >> (56 - (_rs & 7) * 8)) & 0xFF;
                }
                //  lhu
                else if (_opcode == 0x25) {
                    return (_mem >> (48 - (_rs & 6) * 8)) & HALF_MASK;
                }
                //  lwr
                else if (_opcode == 0x26) {
                    uint64 val = _mem >> (24 - (_rs & 3) * 8);
                    uint64 mask = WORD_MASK >> (24 - (_rs & 3) * 8);
                    return signExtend((_rt & ~mask) | val, 32);
                }
                //  sb
                else if (_opcode == 0x28) {
                    uint64 val = (_rt & 0xFF) << (56 - (_rs & 7) * 8);
                    uint64 mask = DOUBLEWORD_MASK ^ uint64(0xFF << (56 - (_rs & 7) * 8));
                    return (_mem & mask) | val;
                }
                //  sh
                else if (_opcode == 0x29) {
                    uint64 sl = 48 - ((_rs & 6) * 8);
                    uint64 val = (_rt & HALF_MASK) << sl;
                    uint64 mask = DOUBLEWORD_MASK ^ (HALF_MASK << sl);
                    return (_mem & mask) | val;
                }
                //  swl
                else if (_opcode == 0x2a) {
                    uint64 sr = (_rs & 3) * 8;
                    uint64 val = ((_rt & WORD_MASK) >> sr) << (32 - ((_rs & 0x4) * 8));
                    uint64 mask = (WORD_MASK >> sr) << (32 - ((_rs & 0x4) * 8));
                    return (_mem & ~mask) | val;
                }
                //  sw
                else if (_opcode == 0x2b) {
                    uint64 sl = 32 - ((_rs & 4) * 8);
                    uint64 val = (_rt & WORD_MASK) << sl;
                    uint64 mask = DOUBLEWORD_MASK ^ (WORD_MASK << sl);
                    return (_mem & mask) | val;
                }
                //  swr
                else if (_opcode == 0x2e) {
                    uint64 sl = 24 - ((_rs & 3) << 3);
                    uint64 val = ((_rt & WORD_MASK) << sl) << (32 - ((_rs & 4) << 3));
                    uint64 mask = uint64(uint32(WORD_MASK) << uint32(sl)) << (32 - ((_rs & 4) << 3));
                    return (_mem & ~mask) | (val & mask);
                }
                //  ll
                else if (_opcode == 0x30) {
                    return signExtend((_mem >> (32 - ((_rs & 0x4) * 8))) & WORD_MASK, 32);
                }
                // sc
                else if (_opcode == 0x38) {
                    uint64 sl = 32 - ((_rs & 4) * 8);
                    uint64 val = (_rt & WORD_MASK) << sl;
                    uint64 mask = DOUBLEWORD_MASK ^ (WORD_MASK << sl);
                    return (_mem & mask) | val;
                }
                // MIPS64

                // ldl
                else if (_opcode == 0x1A) {
                    uint64 sl = (_rs & 7) * 8;
                    uint64 val = _mem << sl;
                    uint64 mask = DOUBLEWORD_MASK << sl;
                    return (_rt & ~mask) | val;
                }
                // ldr
                else if (_opcode == 0x1B) {
                    uint64 sr = 56 - ((_rs & 7) * 8);
                    uint64 val = _mem >> sr;
                    uint64 mask = DOUBLEWORD_MASK << (64 - sr);
                    return (_rt & mask) | val;
                }
                // lwu
                else if (_opcode == 0x27) {
                    return (_mem >> (32 - ((_rs & 4) * 8))) & WORD_MASK;
                }
                // sdl
                else if (_opcode == 0x2C) {
                    uint64 sr = (_rs & 7) * 8;
                    uint64 val = _rt >> sr;
                    uint64 mask = DOUBLEWORD_MASK >> sr;
                    return (_mem & ~mask) | val;
                }
                // sdr
                else if (_opcode == 0x2D) {
                    uint64 sl = 56 - ((_rs & 7) * 8);
                    uint64 val = _rt << sl;
                    uint64 mask = DOUBLEWORD_MASK << sl;
                    return (_mem & ~mask) | val;
                }
                // lld
                else if (_opcode == 0x34) {
                    return _mem;
                }
                // ld
                else if (_opcode == 0x37) {
                    return _mem;
                }
                // scd
                else if (_opcode == 0x3C) {
                    uint64 sl = (_rs & 7) * 8;
                    uint64 val = _rt << sl;
                    uint64 mask = DOUBLEWORD_MASK << sl;
                    return (_mem & ~mask) | val;
                }
                // sd
                else if (_opcode == 0x3F) {
                    uint64 sl = (_rs & 7) * 8;
                    uint64 val = _rt << sl;
                    uint64 mask = DOUBLEWORD_MASK << sl;
                    return (_mem & ~mask) | val;
                }
            }
            revert("invalid instruction");
        }
    }

    /// @notice Extends the value leftwards with its most significant bit (sign extension).
    function signExtend(uint64 _dat, uint64 _idx) internal pure returns (uint64 out_) {
        unchecked {
            bool isSigned = (_dat >> (_idx - 1)) != 0;
            uint256 signed = ((1 << (64 - _idx)) - 1) << _idx;
            uint256 mask = (uint64(1) << _idx) - 1;
            return uint64(_dat & mask | (isSigned ? signed : 0));
        }
    }

    /// @notice Handles a branch instruction, updating the MIPS state PC where needed.
    /// @param _cpu Holds the state of cpu scalars pc, nextPC, hi, lo.
    /// @param _registers Holds the current state of the cpu registers.
    /// @param _opcode The opcode of the branch instruction.
    /// @param _insn The instruction to be executed.
    /// @param _rtReg The register to be used for the branch.
    /// @param _rs The register to be compared with the branch register.
    function handleBranch(
        st.CpuScalars memory _cpu,
        uint64[32] memory _registers,
        uint64 _opcode,
        uint32 _insn,
        uint64 _rtReg,
        uint64 _rs
    )
        internal
        pure
    {
        unchecked {
            bool shouldBranch = false;

            if (_cpu.nextPC != _cpu.pc + 4) {
                revert("branch in delay slot");
            }

            // beq/bne: Branch on equal / not equal
            if (_opcode == 4 || _opcode == 5) {
                uint64 rt = _registers[_rtReg];
                shouldBranch = (_rs == rt && _opcode == 4) || (_rs != rt && _opcode == 5);
            }
            // blez: Branches if instruction is less than or equal to zero
            else if (_opcode == 6) {
                shouldBranch = int64(_rs) <= 0;
            }
            // bgtz: Branches if instruction is greater than zero
            else if (_opcode == 7) {
                shouldBranch = int64(_rs) > 0;
            }
            // bltz/bgez: Branch on less than zero / greater than or equal to zero
            else if (_opcode == 1) {
                // regimm
                uint32 rtv = ((_insn >> 16) & 0x1F);
                if (rtv == 0) {
                    shouldBranch = int64(_rs) < 0;
                }
                if (rtv == 1) {
                    shouldBranch = int64(_rs) >= 0;
                }
            }

            // Update the state's previous PC
            uint64 prevPC = _cpu.pc;

            // Execute the delay slot first
            _cpu.pc = _cpu.nextPC;

            // If we should branch, update the PC to the branch target
            // Otherwise, proceed to the next instruction
            if (shouldBranch) {
                _cpu.nextPC = prevPC + 4 + (signExtend(_insn & HALF_MASK, 16) << 2);
            } else {
                _cpu.nextPC = _cpu.nextPC + 4;
            }
        }
    }

    /// @notice Handles HI and LO register instructions.
    /// @param _cpu Holds the state of cpu scalars pc, nextPC, hi, lo.
    /// @param _registers Holds the current state of the cpu registers.
    /// @param _fun The function code of the instruction.
    /// @param _rs The value of the RS register.
    /// @param _rt The value of the RT register.
    /// @param _storeReg The register to store the result in.
    function handleHiLo(
        st.CpuScalars memory _cpu,
        uint64[32] memory _registers,
        uint64 _fun,
        uint64 _rs,
        uint64 _rt,
        uint64 _storeReg
    )
        internal
        pure
    {
        unchecked {
            uint64 val = 0;

            // mfhi: Move the contents of the HI register into the destination
            if (_fun == 0x10) {
                val = _cpu.hi;
            }
            // mthi: Move the contents of the source into the HI register
            else if (_fun == 0x11) {
                _cpu.hi = _rs;
            }
            // mflo: Move the contents of the LO register into the destination
            else if (_fun == 0x12) {
                val = _cpu.lo;
            }
            // mtlo: Move the contents of the source into the LO register
            else if (_fun == 0x13) {
                _cpu.lo = _rs;
            }
            // mult: Multiplies `rs` by `rt` and stores the result in HI and LO registers
            else if (_fun == 0x18) {
                uint64 acc = uint64(int64(int32(uint32(_rs))) * int64(int32(uint32(_rt))));
                _cpu.hi = signExtend(uint32(acc >> 32), 32);
                _cpu.lo = signExtend(uint32(acc), 32);
            }
            // multu: Unsigned multiplies `rs` by `rt` and stores the result in HI and LO registers
            else if (_fun == 0x19) {
                uint64 acc = uint64(uint32(_rs)) * uint64(uint32(_rt));
                _cpu.hi = signExtend(uint32(acc >> 32), 32);
                _cpu.lo = signExtend(uint32(acc), 32);
            }
            // div: Divides `rs` by `rt`.
            // Stores the quotient in LO
            // And the remainder in HI
            else if (_fun == 0x1a) {
                if (int32(uint32(_rt)) == 0) {
                    revert("MIPS: division by zero");
                }
                _cpu.hi = signExtend(uint32(int32(uint32(_rs)) % int32(uint32(_rt))), 32);
                _cpu.lo = signExtend(uint32(int32(uint32(_rs)) / int32(uint32(_rt))), 32);
            }
            // divu: Unsigned divides `rs` by `rt`.
            // Stores the quotient in LO
            // And the remainder in HI
            else if (_fun == 0x1b) {
                if (_rt == 0) {
                    revert("MIPS: division by zero");
                }
                _cpu.hi = signExtend(uint32(_rs) % uint32(_rt), 32);
                _cpu.lo = signExtend(uint32(_rs) / uint32(_rt), 32);
            }
            // dsllv: Shifts `rt` left by the number of bits in `rs`
            else if (_fun == 0x14) {
                val = _rt << (_rs & 0x3F);
            }
            // dsrlv: Shifts `rt` right by the number of bits in `rs`
            else if (_fun == 0x16) {
                val = _rt >> (_rs & 0x3F);
            }
            // dsrav: Shifts `rt` right with an arithmetic shift by the number of bits in `rs`
            else if (_fun == 0x17) {
                val = uint64(int64(_rt) >> (_rs & 0x3F));
            }
            // dmult: Multiplies `rs` by `rt` and stores the result in HI and LO registers
            else if (_fun == 0x1c) {
                // TODO: Signed multiplication needs go i128
                uint128 acc = uint128(_rs) * uint128(_rt);
                _cpu.hi = uint64(acc >> 64);
                _cpu.lo = uint64(acc);
            }
            // dmultu: Unsigned multiplies `rs` by `rt` and stores the result in HI and LO registers
            else if (_fun == 0x1d) {
                uint128 acc = uint128(_rs) * uint128(_rt);
                _cpu.hi = uint64(acc >> 64);
                _cpu.lo = uint64(acc);
            }
            // ddiv: Divides `rs` by `rt`
            else if (_fun == 0x1e) {
                if (_rt == 0) {
                    revert("MIPS: division by zero");
                }
                _cpu.hi = uint64(int64(_rs) % int64(_rt));
                _cpu.lo = uint64(int64(_rs) / int64(_rt));
            }
            // ddivu: Unsigned divides `rs` by `rt`
            else if (_fun == 0x1f) {
                if (_rt == 0) {
                    revert("MIPS: division by zero");
                }
                _cpu.hi = _rs % _rt;
                _cpu.lo = _rs / _rt;
            }

            // Store the result in the destination register, if applicable
            if (_storeReg != 0) {
                _registers[_storeReg] = val;
            }

            // Update the PC
            _cpu.pc = _cpu.nextPC;
            _cpu.nextPC = _cpu.nextPC + 4;
        }
    }

    /// @notice Handles a jump instruction, updating the MIPS state PC where needed.
    /// @param _cpu Holds the state of cpu scalars pc, nextPC, hi, lo.
    /// @param _registers Holds the current state of the cpu registers.
    /// @param _linkReg The register to store the link to the instruction after the delay slot instruction.
    /// @param _dest The destination to jump to.
    function handleJump(
        st.CpuScalars memory _cpu,
        uint64[32] memory _registers,
        uint64 _linkReg,
        uint64 _dest
    )
        internal
        pure
    {
        unchecked {
            if (_cpu.nextPC != _cpu.pc + 4) {
                revert("jump in delay slot");
            }

            // Update the next PC to the jump destination.
            uint64 prevPC = _cpu.pc;
            _cpu.pc = _cpu.nextPC;
            _cpu.nextPC = _dest;

            // Update the link-register to the instruction after the delay slot instruction.
            if (_linkReg != 0) {
                _registers[_linkReg] = prevPC + 8;
            }
        }
    }

    /// @notice Handles a storing a value into a register.
    /// @param _cpu Holds the state of cpu scalars pc, nextPC, hi, lo.
    /// @param _registers Holds the current state of the cpu registers.
    /// @param _storeReg The register to store the value into.
    /// @param _val The value to store.
    /// @param _conditional Whether or not the store is conditional.
    function handleRd(
        st.CpuScalars memory _cpu,
        uint64[32] memory _registers,
        uint64 _storeReg,
        uint64 _val,
        bool _conditional
    )
        internal
        pure
    {
        unchecked {
            // The destination register must be valid.
            require(_storeReg < 32, "valid register");

            // Never write to reg 0, and it can be conditional (movz, movn).
            if (_storeReg != 0 && _conditional) {
                _registers[_storeReg] = _val;
            }

            // Update the PC.
            _cpu.pc = _cpu.nextPC;
            _cpu.nextPC = _cpu.nextPC + 4;
        }
    }
}
