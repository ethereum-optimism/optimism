// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IPreimageOracle } from "./interfaces/IPreimageOracle.sol";
import { PreimageKeyLib } from "./PreimageKeyLib.sol";

/// @title MIPS
/// @notice The MIPS contract emulates a single MIPS instruction.
///         Note that delay slots are isolated instructions:
///         the nextPC in the state pre-schedules where the VM jumps next.
///         The Step input is a packed VM state, with binary-merkle-tree
///         witness data for memory reads/writes.
///         The Step outputs a keccak256 hash of the packed VM State,
///         and logs the resulting state for offchain usage.
/// @dev https://inst.eecs.berkeley.edu/~cs61c/resources/MIPS_Green_Sheet.pdf
/// @dev https://www.cs.cmu.edu/afs/cs/academic/class/15740-f97/public/doc/mips-isa.pdf
///      (page A-177)
/// @dev https://uweb.engr.arizona.edu/~ece369/Resources/spim/MIPSReference.pdf
/// @dev https://en.wikibooks.org/wiki/MIPS_Assembly/Instruction_Formats
/// @dev https://github.com/golang/go/blob/master/src/syscall/zerrors_linux_mips.go
///      MIPS linux kernel errors used by Go runtime
contract MIPS {

    /// @notice Stores the VM state.
    ///         Total state size: 32 + 32 + 6 * 4 + 1 + 1 + 8 + 32 * 4 = 226 bytes
    ///         If nextPC != pc + 4, then the VM is executing a branch/jump delay slot.
    struct State {
        bytes32 memRoot;
        bytes32 preimageKey;
        uint32 preimageOffset;
        uint32 pc;
        uint32 nextPC;
        uint32 lo;
        uint32 hi;
        uint32 heap;
        uint8 exitCode;
        bool exited;
        uint64 step;
        uint32[32] registers;
    }

    /// @notice Start of the data segment.
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

    /// @notice The pre-image oracle.
    IPreimageOracle public oracle;

    /// @notice Extends the value leftwards with its most significant bit (sign extension).
    function SE(uint32 _dat, uint32 _idx) internal pure returns (uint32) {
        unchecked {
            bool isSigned = (_dat >> (_idx - 1)) != 0;
            uint256 signed = ((1 << (32 - _idx)) - 1) << _idx;
            uint256 mask = (1 << _idx) - 1;
            return uint32(_dat & mask | (isSigned ? signed : 0));
        }
    }

    /// @notice Computes the hash of the MIPS state.
    /// @return out_ The hash of the MIPS state.
    function outputState() internal returns (bytes32 out_) {
        assembly {
            // copies 'size' bytes, right-aligned in word at 'from', to 'to', incl. trailing data
            function copyMem(from, to, size) -> fromOut, toOut {
                mstore(to, mload(add(from, sub(32, size))))
                fromOut := add(from, 32)
                toOut := add(to, size)
            }

            // From points to the MIPS State
            let from := 0x80

            // Copy to the free memory pointer
            let start := mload(0x40)
            let to := start

            // Copy state to free memory
            from, to := copyMem(from, to, 32)  // memRoot
            from, to := copyMem(from, to, 32)  // preimageKey
            from, to := copyMem(from, to, 4)   // preimageOffset
            from, to := copyMem(from, to, 4)   // pc
            from, to := copyMem(from, to, 4)   // nextPC
            from, to := copyMem(from, to, 4)   // lo
            from, to := copyMem(from, to, 4)   // hi
            from, to := copyMem(from, to, 4)   // heap
            from, to := copyMem(from, to, 1)   // exitCode
            from, to := copyMem(from, to, 1)   // exited
            from, to := copyMem(from, to, 8)   // step
            from := add(from, 32)              // offset to registers

            // Copy registers
            for { let i := 0 } lt(i, 32) { i := add(i, 1) } {
                from, to := copyMem(from, to, 4)
            }

            // Clean up end of memory
            mstore(to, 0)

            // Log the resulting MIPS state, for debugging
            log0(start, sub(to, start))

            // Compute the hash of the resulting MIPS state
            out_ := keccak256(start, sub(to, start))
        }
    }

    /// @notice Handles a syscall.
    function handleSyscall() internal returns (bytes32 out_) {
        unchecked {
            // Load state from memory
            State memory state;
            assembly {
                state := 0x80
            }

            // Load the syscall number from the registers
            uint32 syscall_no = state.registers[2];
            uint32 v0 = 0;
            uint32 v1 = 0;

            // Load the syscall arguments from the registers
            uint32 a0 = state.registers[4];
            uint32 a1 = state.registers[5];
            uint32 a2 = state.registers[6];

            // mmap: Allocates a page from the heap.
            if (syscall_no == 4090) {
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
            }
            // brk: Returns a fixed address for the program break at 0x40000000
            else if (syscall_no == 4045) {
                v0 = BRK_START;
            }
            // clone (not supported) returns 1
            else if (syscall_no == 4120) {
                v0 = 1;
            }
            // exit group: Sets the Exited and ExitCode states to true and argument 0.
            else if (syscall_no == 4246) {
                state.exited = true;
                state.exitCode = uint8(a0);
                return outputState();
            }
            // read: Like Linux read syscall. Splits unaligned reads into aligned reads.
            else if (syscall_no == 4003) {
                // args: a0 = fd, a1 = addr, a2 = count
                // returns: v0 = read, v1 = err code
                if (a0 == FD_STDIN) {
                    // Leave v0 and v1 zero: read nothing, no error
                }
                // pre-image oracle read
                else if (a0 == FD_PREIMAGE_READ) {
                    // verify proof 1 is correct, and get the existing memory.
                    uint32 mem = readMem(a1 & 0xFFffFFfc, 1); // mask the addr to align it to 4 bytes
                    bytes32 preimageKey = state.preimageKey;
                    // If the preimage key is a local key, localize it in the context of the caller.
                    if (uint8(preimageKey[0]) == 1) {
                        preimageKey = PreimageKeyLib.localize(preimageKey);
                    }
                    (bytes32 dat, uint256 datLen) = oracle.readPreimage(preimageKey, state.preimageOffset);

                    // Transform data for writing to memory
                    // We use assembly for more precise ops, and no var count limit
                    assembly {
                        let alignment := and(a1, 3) // the read might not start at an aligned address
                        let space := sub(4, alignment) // remaining space in memory word
                        if lt(space, datLen) { datLen := space } // if less space than data, shorten data
                        if lt(a2, datLen) { datLen := a2 } // if requested to read less, read less
                        dat := shr(sub(256, mul(datLen, 8)), dat) // right-align data
                        dat := shl(mul(sub(sub(4, datLen), alignment), 8), dat) // position data to insert into memory word
                        let mask := sub(shl(mul(sub(4, alignment), 8), 1), 1) // mask all bytes after start
                        let suffixMask := sub(shl(mul(sub(sub(4, alignment), datLen), 8), 1), 1) // mask of all bytes starting from end, maybe none
                        mask := and(mask, not(suffixMask)) // reduce mask to just cover the data we insert
                        mem := or(and(mem, not(mask)), dat) // clear masked part of original memory, and insert data
                    }

                    // Write memory back
                    writeMem(a1 & 0xFFffFFfc, 1, mem);
                    state.preimageOffset += uint32(datLen);
                    v0 = uint32(datLen);
                }
                // hint response
                else if (a0 == FD_HINT_READ) {
                    // Don't read into memory, just say we read it all
                    // The result is ignored anyway
                    v0 = a2;
                }
                else {
                    v0 = 0xFFffFFff;
                    v1 = EBADF;
                }
            }
            // write: like Linux write syscall. Splits unaligned writes into aligned writes.
            else if (syscall_no == 4004) {
                // args: a0 = fd, a1 = addr, a2 = count
                // returns: v0 = written, v1 = err code
                if (a0 == FD_STDOUT || a0 == FD_STDERR || a0 == FD_HINT_WRITE) {
                    v0 = a2; // tell program we have written everything
                }
                // pre-image oracle
                else if (a0 == FD_PREIMAGE_WRITE) {
                    uint32 mem = readMem(a1 & 0xFFffFFfc, 1); // mask the addr to align it to 4 bytes
                    bytes32 key = state.preimageKey;

                    // Construct pre-image key from memory
                    // We use assembly for more precise ops, and no var count limit
                    assembly {
                        let alignment := and(a1, 3) // the read might not start at an aligned address
                        let space := sub(4, alignment) // remaining space in memory word
                        if lt(space, a2) { a2 := space } // if less space than data, shorten data
                        key := shl(mul(a2, 8), key) // shift key, make space for new info
                        let mask := sub(shl(mul(a2, 8), 1), 1) // mask for extracting value from memory
                        mem := and(shr(mul(sub(space, a2), 8), mem), mask) // align value to right, mask it
                        key := or(key, mem) // insert into key
                    }

                    // Write pre-image key to oracle
                    state.preimageKey = key;
                    state.preimageOffset = 0; // reset offset, to read new pre-image data from the start
                    v0 = a2;
                }
                else {
                    v0 = 0xFFffFFff;
                    v1 = EBADF;
                }
            }
            // fcntl: Like linux fcntl syscall, but only supports minimal file-descriptor control commands,
            // to retrieve the file-descriptor R/W flags.
            else if (syscall_no == 4055) { // fcntl
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

            // Write the results back to the state registers
            state.registers[2] = v0;
            state.registers[7] = v1;

            // Update the PC and nextPC
            state.pc = state.nextPC;
            state.nextPC = state.nextPC + 4;

            out_ = outputState();
        }
    }

    /// @notice Handles a branch instruction, updating the MIPS state PC where needed.
    /// @param _opcode The opcode of the branch instruction.
    /// @param _insn The instruction to be executed.
    /// @param _rtReg The register to be used for the branch.
    /// @param _rs The register to be compared with the branch register.
    /// @return out_ The hashed MIPS state.
    function handleBranch(uint32 _opcode, uint32 _insn, uint32 _rtReg, uint32 _rs) internal returns (bytes32 out_) {
        unchecked {
            // Load state from memory
            State memory state;
            assembly {
                state := 0x80
            }

            bool shouldBranch = false;

            if (state.nextPC != state.pc+4) {
                revert("branch in delay slot");
            }

            // beq/bne: Branch on equal / not equal
            if (_opcode == 4 || _opcode == 5) {
                uint32 rt = state.registers[_rtReg];
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
            uint32 prevPC = state.pc;

            // Execute the delay slot first
            state.pc = state.nextPC;

            // If we should branch, update the PC to the branch target
            // Otherwise, proceed to the next instruction
            if (shouldBranch) {
                state.nextPC = prevPC + 4 + (SE(_insn & 0xFFFF, 16) << 2);
            } else {
                state.nextPC = state.nextPC + 4;
            }

            // Return the hash of the resulting state
            out_ = outputState();
        }
    }

    /// @notice Handles HI and LO register instructions.
    /// @param _func The function code of the instruction.
    /// @param _rs The value of the RS register.
    /// @param _rt The value of the RT register.
    /// @param _storeReg The register to store the result in.
    /// @return out_ The hash of the resulting MIPS state.
    function handleHiLo(uint32 _func, uint32 _rs, uint32 _rt, uint32 _storeReg) internal returns (bytes32 out_) {
        unchecked {
            // Load state from memory
            State memory state;
            assembly {
                state := 0x80
            }

            uint32 val;

            // mfhi: Move the contents of the HI register into the destination
            if (_func == 0x10) {
                val = state.hi;
            }
            // mthi: Move the contents of the source into the HI register
            else if (_func == 0x11) {
                state.hi = _rs;
            }
            // mflo: Move the contents of the LO register into the destination
            else if (_func == 0x12) {
                val = state.lo;
            }
            // mtlo: Move the contents of the source into the LO register
            else if (_func == 0x13) {
                state.lo = _rs;
            }
            // mult: Multiplies `rs` by `rt` and stores the result in HI and LO registers
            else if (_func == 0x18) {
                uint64 acc = uint64(int64(int32(_rs)) * int64(int32(_rt)));
                state.hi = uint32(acc >> 32);
                state.lo = uint32(acc);
            }
            // multu: Unsigned multiplies `rs` by `rt` and stores the result in HI and LO registers
            else if (_func == 0x19) {
                uint64 acc = uint64(uint64(_rs) * uint64(_rt));
                state.hi = uint32(acc >> 32);
                state.lo = uint32(acc);
            }
            // div: Divides `rs` by `rt`.
            // Stores the quotient in LO
            // And the remainder in HI
            else if (_func == 0x1a) {
                state.hi = uint32(int32(_rs) % int32(_rt));
                state.lo = uint32(int32(_rs) / int32(_rt));
            }
            // divu: Unsigned divides `rs` by `rt`.
            // Stores the quotient in LO
            // And the remainder in HI
            else if (_func == 0x1b) {
                state.hi = _rs % _rt;
                state.lo = _rs / _rt;
            }

            // Store the result in the destination register, if applicable
            if (_storeReg != 0) {
                state.registers[_storeReg] = val;
            }

            // Update the PC
            state.pc = state.nextPC;
            state.nextPC = state.nextPC + 4;

            // Return the hash of the resulting state
            out_ = outputState();
        }
    }

    /// @notice Handles a jump instruction, updating the MIPS state PC where needed.
    /// @param _linkReg The register to store the link to the instruction after the delay slot instruction.
    /// @param _dest The destination to jump to.
    /// @return out_ The hashed MIPS state.
    function handleJump(uint32 _linkReg, uint32 _dest) internal returns (bytes32 out_) {
        unchecked {
            // Load state from memory.
            State memory state;
            assembly {
                state := 0x80
            }

            if (state.nextPC != state.pc+4) {
                revert("jump in delay slot");
            }

            // Update the next PC to the jump destination.
            uint32 prevPC = state.pc;
            state.pc = state.nextPC;
            state.nextPC = _dest;

            // Update the link-register to the instruction after the delay slot instruction.
            if (_linkReg != 0) {
                state.registers[_linkReg] = prevPC + 8;
            }

            // Return the hash of the resulting state.
            out_ = outputState();
        }
    }

    /// @notice Handles a storing a value into a register.
    /// @param _storeReg The register to store the value into.
    /// @param _val The value to store.
    /// @param _conditional Whether or not the store is conditional.
    /// @return out_ The hashed MIPS state.
    function handleRd(uint32 _storeReg, uint32 _val, bool _conditional) internal returns (bytes32 out_) {
        unchecked {
            // Load state from memory.
            State memory state;
            assembly {
                state := 0x80
            }

            // The destination register must be valid.
            require(_storeReg < 32, "valid register");

            // Never write to reg 0, and it can be conditional (movz, movn).
            if (_storeReg != 0 && _conditional) {
                state.registers[_storeReg] = _val;
            }

            // Update the PC.
            state.pc = state.nextPC;
            state.nextPC = state.nextPC + 4;

            // Return the hash of the resulting state.
            out_ = outputState();
        }
    }

    /// @notice Computes the offset of the proof in the calldata.
    /// @param _proofIndex The index of the proof in the calldata.
    /// @return offset_ The offset of the proof in the calldata.
    function proofOffset(uint8 _proofIndex) internal pure returns (uint256 offset_) {
        unchecked {
            // A proof of 32 bit memory, with 32-byte leaf values, is (32-5)=27 bytes32 entries.
            // And the leaf value itself needs to be encoded as well. And proof.offset == 358
            offset_ = 358 + (uint256(_proofIndex) * (28 * 32));
            uint256 s = 0;
            assembly { s := calldatasize() }
            require(s >= (offset_ + 28 * 32), "check that there is enough calldata");
            return offset_;
        }
    }

    /// @notice Reads a 32-bit value from memory.
    /// @param _addr The address to read from.
    /// @param _proofIndex The index of the proof in the calldata.
    /// @return out_ The hashed MIPS state.
    function readMem(uint32 _addr, uint8 _proofIndex) internal pure returns (uint32 out_) {
        unchecked {
            // Compute the offset of the proof in the calldata.
            uint256 offset = proofOffset(_proofIndex);

            assembly {
                // Validate the address alignement.
                if and(_addr, 3) {
                    revert(0, 0)
                }

                // Load the leaf value.
                let leaf := calldataload(offset)
                offset := add(offset, 32)

                // Convenience function to hash two nodes together in scratch space.
                function hashPair(a, b) -> h {
                    mstore(0, a)
                    mstore(32, b)
                    h := keccak256(0, 64)
                }

                // Start with the leaf node.
                // Work back up by combining with siblings, to reconstruct the root.
                let path := shr(5, _addr)
                let node := leaf
                for { let i := 0 } lt(i, 27) { i := add(i, 1) } {
                    let sibling := calldataload(offset)
                    offset := add(offset, 32)
                    switch and(shr(i, path), 1)
                    case 0 {
                        node := hashPair(node, sibling)
                    } case 1 {
                        node := hashPair(sibling, node)
                    }
                }

                // Load the memory root from the first field of state.
                let memRoot := mload(0x80)

                // Verify the root matches.
                if iszero(eq(node, memRoot)) {
                    mstore(0, 0x0badf00d)
                    revert(0, 32)
                }

                // Bits to shift = (32 - 4 - (addr % 32)) * 8
                let shamt := shl(3, sub(sub(32, 4), and(_addr, 31)))
                out_ := and(shr(shamt, leaf), 0xFFffFFff)
            }
        }
    }

    /// @notice Writes a 32-bit value to memory.
    ///         This function first overwrites the part of the leaf.
    ///         Then it recomputes the memory merkle root.
    /// @param _addr The address to write to.
    /// @param _proofIndex The index of the proof in the calldata.
    /// @param _val The value to write.
    function writeMem(uint32 _addr, uint8 _proofIndex, uint32 _val) internal pure {
        unchecked {
            // Compute the offset of the proof in the calldata.
            uint256 offset = proofOffset(_proofIndex);

            assembly {
                // Validate the address alignement.
                if and(_addr, 3) {
                    revert(0, 0)
                }

                // Load the leaf value.
                let leaf := calldataload(offset)
                let shamt := shl(3, sub(sub(32, 4), and(_addr, 31)))

                // Mask out 4 bytes, and OR in the value
                leaf := or(and(leaf, not(shl(shamt, 0xFFffFFff))), shl(shamt, _val))
                offset := add(offset, 32)

                // Convenience function to hash two nodes together in scratch space.
                function hashPair(a, b) -> h {
                    mstore(0, a)
                    mstore(32, b)
                    h := keccak256(0, 64)
                }

                // Start with the leaf node.
                // Work back up by combining with siblings, to reconstruct the root.
                let path := shr(5, _addr)
                let node := leaf
                for { let i := 0 } lt(i, 27) { i := add(i, 1) } {
                    let sibling := calldataload(offset)
                    offset := add(offset, 32)
                    switch and(shr(i, path), 1)
                    case 0 {
                        node := hashPair(node, sibling)
                    } case 1 {
                        node := hashPair(sibling, node)
                    }
                }

                // Store the new memory root in the first field of state.
                mstore(0x80, node)
            }
        }
    }

    /// @notice Executes a single step of the vm.
    ///         Will revert if any required input state is missing.
    function step(bytes calldata stateData, bytes calldata proof) public returns (bytes32) {
        unchecked {
            State memory state;

            // Packed calldata is ~6 times smaller than state size
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

                // Unpack state from calldata into memory
                let c := stateData.offset  // calldata offset
                let m := 0x80              // mem offset
                c, m := putField(c, m, 32) // memRoot
                c, m := putField(c, m, 32) // preimageKey
                c, m := putField(c, m, 4)  // preimageOffset
                c, m := putField(c, m, 4)  // pc
                c, m := putField(c, m, 4)  // nextPC
                c, m := putField(c, m, 4)  // lo
                c, m := putField(c, m, 4)  // hi
                c, m := putField(c, m, 4)  // heap
                c, m := putField(c, m, 1)  // exitCode
                c, m := putField(c, m, 1)  // exited
                c, m := putField(c, m, 8)  // step

                // Unpack register calldata into memory
                mstore(m, add(m, 32))      // offset to registers
                m := add(m, 32)
                for { let i := 0 } lt(i, 32) { i := add(i, 1) } {
                    c, m := putField(c, m, 4)
                }
            }

            // Don't change state once exited
            if (state.exited) {
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
                return handleJump(opcode == 2 ? 0 : 31, SE(insn & 0x03FFFFFF, 26) << 2);
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
                    rt = insn & 0xFFFF;
                } else {
                    // SignExtImm
                    rt = SE(insn & 0xFFFF, 16);
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
    }

    /// @notice Execute an instruction.
    function execute(uint32 insn, uint32 rs, uint32 rt, uint32 mem) internal pure returns (uint32) {
        unchecked {
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
                        // jr/jalr/div + others
                        if (func >= 0x08) {
                            return rs;
                        }
                        // sll: Logical Shift Left
                        else if (func == 0x00) {
                            return rt << shamt;
                        }
                        // srl: Logical Shift Right
                        else if (func == 0x02) {
                            return rt >> shamt;
                        }
                        // sra: Arithmetic Shift Right
                        else if (func == 0x03) {
                            return SE(rt >> shamt, 32 - shamt);
                        }
                        // sllv: Variable Logical Shift Left
                        else if (func == 0x04) {
                            return rt << (rs & 0x1F);
                        }
                        // srlv: Variable Logical Shift Right
                        else if (func == 0x06) {
                            return rt >> (rs & 0x1F);
                        }
                        // srav: Variable Arithmetic Shift Right
                        else if (func == 0x07) {
                            return SE(rt >> rs, 32 - rs);
                        }
                    }

                    // R-type (ArithLog)
                    // 0x10-0x13 = mfhi, mthi, mflo, mtlo
                    // add or addu
                    if (func == 0x20 || func == 0x21) {
                        return rs + rt;
                    }
                    // sub or subu
                    else if (func == 0x22 || func == 0x23) {
                        return rs - rt;
                    }
                    // and
                    else if (func == 0x24) {
                        return rs & rt;
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
                    // slt: Set to 1 if less than
                    else if (func == 0x2a) {
                        return int32(rs) < int32(rt) ? 1 : 0;
                    }
                    // sltu: Set to 1 if less than unsigned
                    else if (func == 0x2B) {
                        return rs<rt ? 1 : 0;
                    }
                }
                // lui: Load Upper Immediate
                else if (opcode == 0xf) {
                    return rt << 16;
                }
                // SPECIAL2
                else if (opcode == 0x1c) {
                    // mul
                    if (func == 2) {
                        return uint32(int32(rs) * int32(rt));
                    }
                    // clo
                    if (func == 0x20 || func == 0x21) {
                        if (func == 0x20) {
                            rs = ~rs;
                        }
                        uint32 i = 0;
                        while (rs&0x80000000 != 0) {
                            i++;
                            rs <<= 1;
                        }
                        return i;
                    }
                }
            }
            else if (opcode < 0x28) {
                // lb
                if (opcode == 0x20) {
                    return SE((mem >> (24 - (rs & 3) * 8)) & 0xFF, 8);
                }
                // lh
                else if (opcode == 0x21) {
                    return SE((mem >> (16 - (rs & 2) * 8)) & 0xFFFF, 16);
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
                // lhu
                else if (opcode == 0x25) {
                    return (mem >> (16 - (rs & 2) * 8)) & 0xFFFF;
                }
                // lwr
                else if (opcode == 0x26) {
                    uint32 val = mem >> (24 - (rs & 3) * 8);
                    uint32 mask = uint32(0xFFFFFFFF) >> (24 - (rs & 3) * 8);
                    return (rt & ~mask) | val;
                }
            }
            // sb
            else if (opcode == 0x28) {
                uint32 val = (rt & 0xFF) << (24 - (rs & 3) * 8);
                uint32 mask = 0xFFFFFFFF ^ uint32(0xFF << (24 - (rs & 3) * 8));
                return (mem & mask) | val;
            }
            // sh
            else if (opcode == 0x29) {
                uint32 val = (rt & 0xFFFF) << (16 - (rs & 2) * 8);
                uint32 mask = 0xFFFFFFFF ^ uint32(0xFFFF << (16 - (rs & 2) * 8));
                return (mem & mask) | val;
            }
            // swl
            else if (opcode == 0x2a) {
                uint32 val = rt >> ((rs & 3) * 8);
                uint32 mask = uint32(0xFFFFFFFF) >> ((rs & 3) * 8);
                return (mem & ~mask) | val;
            }
            // sw
            else if (opcode == 0x2b) {
                return rt;
            }
            // swr
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
            }

            revert("invalid instruction");
        }
    }
}
