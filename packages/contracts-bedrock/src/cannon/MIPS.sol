// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/ISemver.sol";
import { IPreimageOracle } from "./interfaces/IPreimageOracle.sol";
import { PreimageKeyLib } from "./PreimageKeyLib.sol";
import { MIPSInstructions as ins } from "src/cannon/libraries/MIPSInstructions.sol";
import { MIPSState as st } from "src/cannon/libraries/MIPSState.sol";

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
contract MIPS is ISemver {
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
    uint32 public constant BRK_START = 0x40000000;

    /// @notice The semantic version of the MIPS contract.
    /// @custom:semver 1.0.1
    string public constant version = "1.1.0-beta.3";

    uint32 internal constant FD_STDIN = 0;
    uint32 internal constant FD_STDOUT = 1;
    uint32 internal constant FD_STDERR = 2;
    uint32 internal constant FD_HINT_READ = 3;
    uint32 internal constant FD_HINT_WRITE = 4;
    uint32 internal constant FD_PREIMAGE_READ = 5;
    uint32 internal constant FD_PREIMAGE_WRITE = 6;

    uint32 internal constant EBADF = 0x9;
    uint32 internal constant EINVAL = 0x16;

    /// @notice The preimage oracle contract.
    IPreimageOracle internal immutable ORACLE;

    /// @param _oracle The address of the preimage oracle contract.
    constructor(IPreimageOracle _oracle) {
        ORACLE = _oracle;
    }

    /// @notice Getter for the pre-image oracle contract.
    /// @return oracle_ The IPreimageOracle contract.
    function oracle() external view returns (IPreimageOracle oracle_) {
        oracle_ = ORACLE;
    }

    /// @notice Computes the hash of the MIPS state.
    /// @return out_ The hashed MIPS state.
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
            from, to := copyMem(from, to, 32) // memRoot
            from, to := copyMem(from, to, 32) // preimageKey
            from, to := copyMem(from, to, 4) // preimageOffset
            from, to := copyMem(from, to, 4) // pc
            from, to := copyMem(from, to, 4) // nextPC
            from, to := copyMem(from, to, 4) // lo
            from, to := copyMem(from, to, 4) // hi
            from, to := copyMem(from, to, 4) // heap
            let exitCode := mload(from)
            from, to := copyMem(from, to, 1) // exitCode
            let exited := mload(from)
            from, to := copyMem(from, to, 1) // exited
            from, to := copyMem(from, to, 8) // step
            from := add(from, 32) // offset to registers

            // Copy registers
            for { let i := 0 } lt(i, 32) { i := add(i, 1) } { from, to := copyMem(from, to, 4) }

            // Clean up end of memory
            mstore(to, 0)

            // Log the resulting MIPS state, for debugging
            log0(start, sub(to, start))

            // Determine the VM status
            let status := 0
            switch exited
            case 1 {
                switch exitCode
                // VMStatusValid
                case 0 { status := 0 }
                // VMStatusInvalid
                case 1 { status := 1 }
                // VMStatusPanic
                default { status := 2 }
            }
            // VMStatusUnfinished
            default { status := 3 }

            // Compute the hash of the resulting MIPS state and set the status byte
            out_ := keccak256(start, sub(to, start))
            out_ := or(and(not(shl(248, 0xFF)), out_), shl(248, status))
        }
    }

    /// @notice Handles a syscall.
    /// @param _localContext The local key context for the preimage oracle.
    /// @return out_ The hashed MIPS state.
    function handleSyscall(bytes32 _localContext) internal returns (bytes32 out_) {
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
                if (sz & 4095 != 0) {
                    // adjust size to align with page size
                    sz += 4096 - (sz & 4095);
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
                        preimageKey = PreimageKeyLib.localize(preimageKey, _localContext);
                    }
                    (bytes32 dat, uint256 datLen) = ORACLE.readPreimage(preimageKey, state.preimageOffset);

                    // Transform data for writing to memory
                    // We use assembly for more precise ops, and no var count limit
                    assembly {
                        let alignment := and(a1, 3) // the read might not start at an aligned address
                        let space := sub(4, alignment) // remaining space in memory word
                        if lt(space, datLen) { datLen := space } // if less space than data, shorten data
                        if lt(a2, datLen) { datLen := a2 } // if requested to read less, read less
                        dat := shr(sub(256, mul(datLen, 8)), dat) // right-align data
                        dat := shl(mul(sub(sub(4, datLen), alignment), 8), dat) // position data to insert into memory
                            // word
                        let mask := sub(shl(mul(sub(4, alignment), 8), 1), 1) // mask all bytes after start
                        let suffixMask := sub(shl(mul(sub(sub(4, alignment), datLen), 8), 1), 1) // mask of all bytes
                            // starting from end, maybe none
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
                } else {
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
                } else {
                    v0 = 0xFFffFFff;
                    v1 = EBADF;
                }
            }
            // fcntl: Like linux fcntl syscall, but only supports minimal file-descriptor control commands,
            // to retrieve the file-descriptor R/W flags.
            else if (syscall_no == 4055) {
                // fcntl
                // args: a0 = fd, a1 = cmd
                if (a1 == 3) {
                    // F_GETFL: get file descriptor flags
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

    /// @notice Computes the offset of the proof in the calldata.
    /// @param _proofIndex The index of the proof in the calldata.
    /// @return offset_ The offset of the proof in the calldata.
    function proofOffset(uint8 _proofIndex) internal pure returns (uint256 offset_) {
        unchecked {
            // A proof of 32 bit memory, with 32-byte leaf values, is (32-5)=27 bytes32 entries.
            // And the leaf value itself needs to be encoded as well. And proof.offset == 420
            offset_ = 420 + (uint256(_proofIndex) * (28 * 32));
            uint256 s = 0;
            assembly {
                s := calldatasize()
            }
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
                if and(_addr, 3) { revert(0, 0) }

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
                    case 0 { node := hashPair(node, sibling) }
                    case 1 { node := hashPair(sibling, node) }
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
                if and(_addr, 3) { revert(0, 0) }

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
                    case 0 { node := hashPair(node, sibling) }
                    case 1 { node := hashPair(sibling, node) }
                }

                // Store the new memory root in the first field of state.
                mstore(0x80, node)
            }
        }
    }

    /// @notice Executes a single step of the vm.
    ///         Will revert if any required input state is missing.
    /// @param _stateData The encoded state witness data.
    /// @param _proof The encoded proof data for leaves within the MIPS VM's memory.
    /// @param _localContext The local key context for the preimage oracle. Optional, can be set as a constant
    ///                      if the caller only requires one set of local keys.
    function step(bytes calldata _stateData, bytes calldata _proof, bytes32 _localContext) public returns (bytes32) {
        unchecked {
            State memory state;

            // Packed calldata is ~6 times smaller than state size
            assembly {
                if iszero(eq(state, 0x80)) {
                    // expected state mem offset check
                    revert(0, 0)
                }
                if iszero(eq(mload(0x40), shl(5, 48))) {
                    // expected memory check
                    revert(0, 0)
                }
                if iszero(eq(_stateData.offset, 132)) {
                    // 32*4+4=132 expected state data offset
                    revert(0, 0)
                }
                if iszero(eq(_proof.offset, 420)) {
                    // 132+32+256=420 expected proof offset
                    revert(0, 0)
                }

                function putField(callOffset, memOffset, size) -> callOffsetOut, memOffsetOut {
                    // calldata is packed, thus starting left-aligned, shift-right to pad and right-align
                    let w := shr(shl(3, sub(32, size)), calldataload(callOffset))
                    mstore(memOffset, w)
                    callOffsetOut := add(callOffset, size)
                    memOffsetOut := add(memOffset, 32)
                }

                // Unpack state from calldata into memory
                let c := _stateData.offset // calldata offset
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

                // Unpack register calldata into memory
                mstore(m, add(m, 32)) // offset to registers
                m := add(m, 32)
                for { let i := 0 } lt(i, 32) { i := add(i, 1) } { c, m := putField(c, m, 4) }
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
                // Take top 4 bits of the next PC (its 256 MB region), and concatenate with the 26-bit offset
                uint32 target = (state.nextPC & 0xF0000000) | (insn & 0x03FFFFFF) << 2;
                return handleJumpAndReturnOutput(state, opcode == 2 ? 0 : 31, target);
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
                    rt = ins.signExtend(insn & 0xFFFF, 16);
                }
            } else if (opcode >= 0x28 || opcode == 0x22 || opcode == 0x26) {
                // store rt value with store
                rt = state.registers[rtReg];

                // store actual rt with lwl and lwr
                rdReg = rtReg;
            }

            if ((opcode >= 4 && opcode < 8) || opcode == 1) {
                st.CpuScalars memory cpu = getCpuScalars(state);

                ins.handleBranch({
                    _cpu: cpu,
                    _registers: state.registers,
                    _opcode: opcode,
                    _insn: insn,
                    _rtReg: rtReg,
                    _rs: rs
                });
                setStateCpuScalars(state, cpu);

                return outputState();
            }

            uint32 storeAddr = 0xFF_FF_FF_FF;
            // memory fetch (all I-type)
            // we do the load for stores also
            uint32 mem;
            if (opcode >= 0x20) {
                // M[R[rs]+SignExtImm]
                rs += ins.signExtend(insn & 0xFFFF, 16);
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
            // Note: swr outputs more than 4 bytes without the mask 0xffFFffFF
            uint32 val = ins.executeMipsInstruction(insn, rs, rt, mem) & 0xffFFffFF;

            uint32 func = insn & 0x3f; // 6-bits
            if (opcode == 0 && func >= 8 && func < 0x1c) {
                if (func == 8 || func == 9) {
                    // jr/jalr
                    return handleJumpAndReturnOutput(state, func == 8 ? 0 : rdReg, rs);
                }

                if (func == 0xa) {
                    // movz
                    return handleRdAndReturnOutput(state, rdReg, rs, rt == 0);
                }
                if (func == 0xb) {
                    // movn
                    return handleRdAndReturnOutput(state, rdReg, rs, rt != 0);
                }

                // syscall (can read and write)
                if (func == 0xC) {
                    return handleSyscall(_localContext);
                }

                // lo and hi registers
                // can write back
                if (func >= 0x10 && func < 0x1c) {
                    st.CpuScalars memory cpu = getCpuScalars(state);

                    ins.handleHiLo({
                        _cpu: cpu,
                        _registers: state.registers,
                        _func: func,
                        _rs: rs,
                        _rt: rt,
                        _storeReg: rdReg
                    });

                    setStateCpuScalars(state, cpu);
                    return outputState();
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
            return handleRdAndReturnOutput(state, rdReg, val, true);
        }
    }

    function handleJumpAndReturnOutput(
        State memory _state,
        uint32 _linkReg,
        uint32 _dest
    )
        internal
        returns (bytes32 out_)
    {
        st.CpuScalars memory cpu = getCpuScalars(_state);

        ins.handleJump({ _cpu: cpu, _registers: _state.registers, _linkReg: _linkReg, _dest: _dest });

        setStateCpuScalars(_state, cpu);
        return outputState();
    }

    function handleRdAndReturnOutput(
        State memory _state,
        uint32 _storeReg,
        uint32 _val,
        bool _conditional
    )
        internal
        returns (bytes32 out_)
    {
        st.CpuScalars memory cpu = getCpuScalars(_state);

        ins.handleRd({
            _cpu: cpu,
            _registers: _state.registers,
            _storeReg: _storeReg,
            _val: _val,
            _conditional: _conditional
        });

        setStateCpuScalars(_state, cpu);
        return outputState();
    }

    function getCpuScalars(State memory _state) internal pure returns (st.CpuScalars memory) {
        return st.CpuScalars({ pc: _state.pc, nextPC: _state.nextPC, lo: _state.lo, hi: _state.hi });
    }

    function setStateCpuScalars(State memory _state, st.CpuScalars memory _cpu) internal pure {
        _state.pc = _cpu.pc;
        _state.nextPC = _cpu.nextPC;
        _state.lo = _cpu.lo;
        _state.hi = _cpu.hi;
    }
}
