// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @notice Execute an instruction.
function executeMipsInstruction(uint32 insn, uint32 rs, uint32 rt, uint32 mem) pure returns (uint32 out) {
    unchecked {
        uint32 opcode = insn >> 26; // 6-bits

        if (opcode == 0 || (opcode >= 8 && opcode < 0xF)) {
            uint32 func = insn & 0x3f; // 6-bits
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
                return rt << ((insn >> 6) & 0x1F);
            }
            // srl
            else if (func == 0x02) {
                return rt >> ((insn >> 6) & 0x1F);
            }
            // sra
            else if (func == 0x03) {
                uint32 shamt = (insn >> 6) & 0x1F;
                return signExtend(rt >> shamt, 32 - shamt);
            }
            // sllv
            else if (func == 0x04) {
                return rt << (rs & 0x1F);
            }
            // srlv
            else if (func == 0x6) {
                return rt >> (rs & 0x1F);
            }
            // srav
            else if (func == 0x07) {
                return signExtend(rt >> rs, 32 - rs);
            }
            // functs in range [0x8, 0x1b] are handled specially by other functions
            // Explicitly enumerate each funct in range to reduce code diff against Go Vm
            // jr
            else if (func == 0x08) {
                return rs;
            }
            // jalr
            else if (func == 0x09) {
                return rs;
            }
            // movz
            else if (func == 0x0a) {
                return rs;
            }
            // movn
            else if (func == 0x0b) {
                return rs;
            }
            // syscall
            else if (func == 0x0c) {
                return rs;
            }
            // 0x0d - break not supported
            // sync
            else if (func == 0x0f) {
                return rs;
            }
            // mfhi
            else if (func == 0x10) {
                return rs;
            }
            // mthi
            else if (func == 0x11) {
                return rs;
            }
            // mflo
            else if (func == 0x12) {
                return rs;
            }
            // mtlo
            else if (func == 0x13) {
                return rs;
            }
            // mult
            else if (func == 0x18) {
                return rs;
            }
            // multu
            else if (func == 0x19) {
                return rs;
            }
            // div
            else if (func == 0x1a) {
                return rs;
            }
            // divu
            else if (func == 0x1b) {
                return rs;
            }
            // The rest includes transformed R-type arith imm instructions
            // add
            else if (func == 0x20) {
                return (rs + rt);
            }
            // addu
            else if (func == 0x21) {
                return (rs + rt);
            }
            // sub
            else if (func == 0x22) {
                return (rs - rt);
            }
            // subu
            else if (func == 0x23) {
                return (rs - rt);
            }
            // and
            else if (func == 0x24) {
                return (rs & rt);
            }
            // or
            else if (func == 0x25) {
                return (rs | rt);
            }
            // xor
            else if (func == 0x26) {
                return (rs ^ rt);
            }
            // nor
            else if (func == 0x27) {
                return ~(rs | rt);
            }
            // slti
            else if (func == 0x2a) {
                return int32(rs) < int32(rt) ? 1 : 0;
            }
            // sltiu
            else if (func == 0x2b) {
                return rs < rt ? 1 : 0;
            } else {
                revert("invalid instruction");
            }
        } else {
            // SPECIAL2
            if (opcode == 0x1C) {
                uint32 func = insn & 0x3f; // 6-bits
                // mul
                if (func == 0x2) {
                    return uint32(int32(rs) * int32(rt));
                }
                // clz, clo
                else if (func == 0x20 || func == 0x21) {
                    if (func == 0x20) {
                        rs = ~rs;
                    }
                    uint32 i = 0;
                    while (rs & 0x80000000 != 0) {
                        i++;
                        rs <<= 1;
                    }
                    return i;
                }
            }
            // lui
            else if (opcode == 0x0F) {
                return rt << 16;
            }
            // lb
            else if (opcode == 0x20) {
                return signExtend((mem >> (24 - (rs & 3) * 8)) & 0xFF, 8);
            }
            // lh
            else if (opcode == 0x21) {
                return signExtend((mem >> (16 - (rs & 2) * 8)) & 0xFFFF, 16);
            }
            // lwl
            else if (opcode == 0x22) {
                uint32 val = mem << ((rs & 3) * 8);
                uint32 mask = uint32(0xFFFFFFFF) << ((rs & 3) * 8);
                return (rt & ~mask) | val;
            }
            // lw
            else if (opcode == 0x23) {
                return mem;
            }
            // lbu
            else if (opcode == 0x24) {
                return (mem >> (24 - (rs & 3) * 8)) & 0xFF;
            }
            //  lhu
            else if (opcode == 0x25) {
                return (mem >> (16 - (rs & 2) * 8)) & 0xFFFF;
            }
            //  lwr
            else if (opcode == 0x26) {
                uint32 val = mem >> (24 - (rs & 3) * 8);
                uint32 mask = uint32(0xFFFFFFFF) >> (24 - (rs & 3) * 8);
                return (rt & ~mask) | val;
            }
            //  sb
            else if (opcode == 0x28) {
                uint32 val = (rt & 0xFF) << (24 - (rs & 3) * 8);
                uint32 mask = 0xFFFFFFFF ^ uint32(0xFF << (24 - (rs & 3) * 8));
                return (mem & mask) | val;
            }
            //  sh
            else if (opcode == 0x29) {
                uint32 val = (rt & 0xFFFF) << (16 - (rs & 2) * 8);
                uint32 mask = 0xFFFFFFFF ^ uint32(0xFFFF << (16 - (rs & 2) * 8));
                return (mem & mask) | val;
            }
            //  swl
            else if (opcode == 0x2a) {
                uint32 val = rt >> ((rs & 3) * 8);
                uint32 mask = uint32(0xFFFFFFFF) >> ((rs & 3) * 8);
                return (mem & ~mask) | val;
            }
            //  sw
            else if (opcode == 0x2b) {
                return rt;
            }
            //  swr
            else if (opcode == 0x2e) {
                uint32 val = rt << (24 - (rs & 3) * 8);
                uint32 mask = uint32(0xFFFFFFFF) << (24 - (rs & 3) * 8);
                return (mem & ~mask) | val;
            }
            // ll
            else if (opcode == 0x30) {
                return mem;
            }
            // sc
            else if (opcode == 0x38) {
                return rt;
            } else {
                revert("invalid instruction");
            }
        }
        revert("invalid instruction");
    }
}

/// @notice Extends the value leftwards with its most significant bit (sign extension).
function signExtend(uint32 _dat, uint32 _idx) pure returns (uint32 out_) {
    unchecked {
        bool isSigned = (_dat >> (_idx - 1)) != 0;
        uint256 signed = ((1 << (32 - _idx)) - 1) << _idx;
        uint256 mask = (1 << _idx) - 1;
        return uint32(_dat & mask | (isSigned ? signed : 0));
    }
}
