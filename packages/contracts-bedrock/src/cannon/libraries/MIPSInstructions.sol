// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { MIPSState as st } from "src/cannon/libraries/MIPSState.sol";

library MIPSInstructions {
    /// @notice Execute an instruction.
    function executeMipsInstruction(
        uint32 _insn,
        uint32 _rs,
        uint32 _rt,
        uint32 _mem
    )
        internal
        pure
        returns (uint32 out_)
    {
        unchecked {
            uint32 opcode = _insn >> 26; // 6-bits

            if (opcode == 0 || (opcode >= 8 && opcode < 0xF)) {
                uint32 func = _insn & 0x3f; // 6-bits
                assembly {
                    // transform ArithLogI to SPECIAL
                    switch opcode
                    // addi
                    case 0x8 { func := 0x20 }
                    // addiu
                    case 0x9 { func := 0x21 }
                    // stli
                    case 0xA { func := 0x2A }
                    // sltiu
                    case 0xB { func := 0x2B }
                    // andi
                    case 0xC { func := 0x24 }
                    // ori
                    case 0xD { func := 0x25 }
                    // xori
                    case 0xE { func := 0x26 }
                }

                // sll
                if (func == 0x00) {
                    return _rt << ((_insn >> 6) & 0x1F);
                }
                // srl
                else if (func == 0x02) {
                    return _rt >> ((_insn >> 6) & 0x1F);
                }
                // sra
                else if (func == 0x03) {
                    uint32 shamt = (_insn >> 6) & 0x1F;
                    return signExtend(_rt >> shamt, 32 - shamt);
                }
                // sllv
                else if (func == 0x04) {
                    return _rt << (_rs & 0x1F);
                }
                // srlv
                else if (func == 0x6) {
                    return _rt >> (_rs & 0x1F);
                }
                // srav
                else if (func == 0x07) {
                    return signExtend(_rt >> _rs, 32 - _rs);
                }
                // functs in range [0x8, 0x1b] are handled specially by other functions
                // Explicitly enumerate each funct in range to reduce code diff against Go Vm
                // jr
                else if (func == 0x08) {
                    return _rs;
                }
                // jalr
                else if (func == 0x09) {
                    return _rs;
                }
                // movz
                else if (func == 0x0a) {
                    return _rs;
                }
                // movn
                else if (func == 0x0b) {
                    return _rs;
                }
                // syscall
                else if (func == 0x0c) {
                    return _rs;
                }
                // 0x0d - break not supported
                // sync
                else if (func == 0x0f) {
                    return _rs;
                }
                // mfhi
                else if (func == 0x10) {
                    return _rs;
                }
                // mthi
                else if (func == 0x11) {
                    return _rs;
                }
                // mflo
                else if (func == 0x12) {
                    return _rs;
                }
                // mtlo
                else if (func == 0x13) {
                    return _rs;
                }
                // mult
                else if (func == 0x18) {
                    return _rs;
                }
                // multu
                else if (func == 0x19) {
                    return _rs;
                }
                // div
                else if (func == 0x1a) {
                    return _rs;
                }
                // divu
                else if (func == 0x1b) {
                    return _rs;
                }
                // The rest includes transformed R-type arith imm instructions
                // add
                else if (func == 0x20) {
                    return (_rs + _rt);
                }
                // addu
                else if (func == 0x21) {
                    return (_rs + _rt);
                }
                // sub
                else if (func == 0x22) {
                    return (_rs - _rt);
                }
                // subu
                else if (func == 0x23) {
                    return (_rs - _rt);
                }
                // and
                else if (func == 0x24) {
                    return (_rs & _rt);
                }
                // or
                else if (func == 0x25) {
                    return (_rs | _rt);
                }
                // xor
                else if (func == 0x26) {
                    return (_rs ^ _rt);
                }
                // nor
                else if (func == 0x27) {
                    return ~(_rs | _rt);
                }
                // slti
                else if (func == 0x2a) {
                    return int32(_rs) < int32(_rt) ? 1 : 0;
                }
                // sltiu
                else if (func == 0x2b) {
                    return _rs < _rt ? 1 : 0;
                } else {
                    revert("invalid instruction");
                }
            } else {
                // SPECIAL2
                if (opcode == 0x1C) {
                    uint32 func = _insn & 0x3f; // 6-bits
                    // mul
                    if (func == 0x2) {
                        return uint32(int32(_rs) * int32(_rt));
                    }
                    // clz, clo
                    else if (func == 0x20 || func == 0x21) {
                        if (func == 0x20) {
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
                else if (opcode == 0x0F) {
                    return _rt << 16;
                }
                // lb
                else if (opcode == 0x20) {
                    return signExtend((_mem >> (24 - (_rs & 3) * 8)) & 0xFF, 8);
                }
                // lh
                else if (opcode == 0x21) {
                    return signExtend((_mem >> (16 - (_rs & 2) * 8)) & 0xFFFF, 16);
                }
                // lwl
                else if (opcode == 0x22) {
                    uint32 val = _mem << ((_rs & 3) * 8);
                    uint32 mask = uint32(0xFFFFFFFF) << ((_rs & 3) * 8);
                    return (_rt & ~mask) | val;
                }
                // lw
                else if (opcode == 0x23) {
                    return _mem;
                }
                // lbu
                else if (opcode == 0x24) {
                    return (_mem >> (24 - (_rs & 3) * 8)) & 0xFF;
                }
                //  lhu
                else if (opcode == 0x25) {
                    return (_mem >> (16 - (_rs & 2) * 8)) & 0xFFFF;
                }
                //  lwr
                else if (opcode == 0x26) {
                    uint32 val = _mem >> (24 - (_rs & 3) * 8);
                    uint32 mask = uint32(0xFFFFFFFF) >> (24 - (_rs & 3) * 8);
                    return (_rt & ~mask) | val;
                }
                //  sb
                else if (opcode == 0x28) {
                    uint32 val = (_rt & 0xFF) << (24 - (_rs & 3) * 8);
                    uint32 mask = 0xFFFFFFFF ^ uint32(0xFF << (24 - (_rs & 3) * 8));
                    return (_mem & mask) | val;
                }
                //  sh
                else if (opcode == 0x29) {
                    uint32 val = (_rt & 0xFFFF) << (16 - (_rs & 2) * 8);
                    uint32 mask = 0xFFFFFFFF ^ uint32(0xFFFF << (16 - (_rs & 2) * 8));
                    return (_mem & mask) | val;
                }
                //  swl
                else if (opcode == 0x2a) {
                    uint32 val = _rt >> ((_rs & 3) * 8);
                    uint32 mask = uint32(0xFFFFFFFF) >> ((_rs & 3) * 8);
                    return (_mem & ~mask) | val;
                }
                //  sw
                else if (opcode == 0x2b) {
                    return _rt;
                }
                //  swr
                else if (opcode == 0x2e) {
                    uint32 val = _rt << (24 - (_rs & 3) * 8);
                    uint32 mask = uint32(0xFFFFFFFF) << (24 - (_rs & 3) * 8);
                    return (_mem & ~mask) | val;
                }
                // ll
                else if (opcode == 0x30) {
                    return _mem;
                }
                // sc
                else if (opcode == 0x38) {
                    return _rt;
                } else {
                    revert("invalid instruction");
                }
            }
            revert("invalid instruction");
        }
    }

    /// @notice Extends the value leftwards with its most significant bit (sign extension).
    function signExtend(uint32 _dat, uint32 _idx) internal pure returns (uint32 out_) {
        unchecked {
            bool isSigned = (_dat >> (_idx - 1)) != 0;
            uint256 signed = ((1 << (32 - _idx)) - 1) << _idx;
            uint256 mask = (1 << _idx) - 1;
            return uint32(_dat & mask | (isSigned ? signed : 0));
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
        uint32[32] memory _registers,
        uint32 _opcode,
        uint32 _insn,
        uint32 _rtReg,
        uint32 _rs
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
                uint32 rt = _registers[_rtReg];
                shouldBranch = (_rs == rt && _opcode == 4) || (_rs != rt && _opcode == 5);
            }
            // blez: Branches if instruction is less than or equal to zero
            else if (_opcode == 6) {
                shouldBranch = int32(_rs) <= 0;
            }
            // bgtz: Branches if instruction is greater than zero
            else if (_opcode == 7) {
                shouldBranch = int32(_rs) > 0;
            }
            // bltz/bgez: Branch on less than zero / greater than or equal to zero
            else if (_opcode == 1) {
                // regimm
                uint32 rtv = ((_insn >> 16) & 0x1F);
                if (rtv == 0) {
                    shouldBranch = int32(_rs) < 0;
                }
                if (rtv == 1) {
                    shouldBranch = int32(_rs) >= 0;
                }
            }

            // Update the state's previous PC
            uint32 prevPC = _cpu.pc;

            // Execute the delay slot first
            _cpu.pc = _cpu.nextPC;

            // If we should branch, update the PC to the branch target
            // Otherwise, proceed to the next instruction
            if (shouldBranch) {
                _cpu.nextPC = prevPC + 4 + (signExtend(_insn & 0xFFFF, 16) << 2);
            } else {
                _cpu.nextPC = _cpu.nextPC + 4;
            }
        }
    }

    /// @notice Handles HI and LO register instructions.
    /// @param _cpu Holds the state of cpu scalars pc, nextPC, hi, lo.
    /// @param _registers Holds the current state of the cpu registers.
    /// @param _func The function code of the instruction.
    /// @param _rs The value of the RS register.
    /// @param _rt The value of the RT register.
    /// @param _storeReg The register to store the result in.
    function handleHiLo(
        st.CpuScalars memory _cpu,
        uint32[32] memory _registers,
        uint32 _func,
        uint32 _rs,
        uint32 _rt,
        uint32 _storeReg
    )
        internal
        pure
    {
        unchecked {
            uint32 val = 0;

            // mfhi: Move the contents of the HI register into the destination
            if (_func == 0x10) {
                val = _cpu.hi;
            }
            // mthi: Move the contents of the source into the HI register
            else if (_func == 0x11) {
                _cpu.hi = _rs;
            }
            // mflo: Move the contents of the LO register into the destination
            else if (_func == 0x12) {
                val = _cpu.lo;
            }
            // mtlo: Move the contents of the source into the LO register
            else if (_func == 0x13) {
                _cpu.lo = _rs;
            }
            // mult: Multiplies `rs` by `rt` and stores the result in HI and LO registers
            else if (_func == 0x18) {
                uint64 acc = uint64(int64(int32(_rs)) * int64(int32(_rt)));
                _cpu.hi = uint32(acc >> 32);
                _cpu.lo = uint32(acc);
            }
            // multu: Unsigned multiplies `rs` by `rt` and stores the result in HI and LO registers
            else if (_func == 0x19) {
                uint64 acc = uint64(uint64(_rs) * uint64(_rt));
                _cpu.hi = uint32(acc >> 32);
                _cpu.lo = uint32(acc);
            }
            // div: Divides `rs` by `rt`.
            // Stores the quotient in LO
            // And the remainder in HI
            else if (_func == 0x1a) {
                if (int32(_rt) == 0) {
                    revert("MIPS: division by zero");
                }
                _cpu.hi = uint32(int32(_rs) % int32(_rt));
                _cpu.lo = uint32(int32(_rs) / int32(_rt));
            }
            // divu: Unsigned divides `rs` by `rt`.
            // Stores the quotient in LO
            // And the remainder in HI
            else if (_func == 0x1b) {
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
        uint32[32] memory _registers,
        uint32 _linkReg,
        uint32 _dest
    )
        internal
        pure
    {
        unchecked {
            if (_cpu.nextPC != _cpu.pc + 4) {
                revert("jump in delay slot");
            }

            // Update the next PC to the jump destination.
            uint32 prevPC = _cpu.pc;
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
        uint32[32] memory _registers,
        uint32 _storeReg,
        uint32 _val,
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
