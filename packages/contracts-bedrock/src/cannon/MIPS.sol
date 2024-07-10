// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/ISemver.sol";
import { IPreimageOracle } from "./interfaces/IPreimageOracle.sol";
import { PreimageKeyLib } from "./PreimageKeyLib.sol";
import { MIPSInstructions as ins } from "src/cannon/libraries/MIPSInstructions.sol";
import { MIPSSyscalls as sys } from "src/cannon/libraries/MIPSSyscalls.sol";
import { MIPSState as st } from "src/cannon/libraries/MIPSState.sol";
import { MIPSMemory } from "src/cannon/libraries/MIPSMemory.sol";

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
    string public constant version = "1.1.0-beta.4";

    /// @notice The preimage oracle contract.
    IPreimageOracle internal immutable ORACLE;

    // The offset of the start of proof calldata (_proof.offset) in the step() function
    uint256 internal constant STEP_PROOF_OFFSET = 420;

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

            // Load the syscall numbers and args from the registers
            (uint32 syscall_no, uint32 a0, uint32 a1, uint32 a2) = sys.getSyscallArgs(state.registers);

            uint32 v0 = 0;
            uint32 v1 = 0;

            if (syscall_no == sys.SYS_MMAP) {
                (v0, v1, state.heap) = sys.handleSysMmap(a0, a1, state.heap);
            } else if (syscall_no == sys.SYS_BRK) {
                // brk: Returns a fixed address for the program break at 0x40000000
                v0 = BRK_START;
            } else if (syscall_no == sys.SYS_CLONE) {
                // clone (not supported) returns 1
                v0 = 1;
            } else if (syscall_no == sys.SYS_EXIT_GROUP) {
                // exit group: Sets the Exited and ExitCode states to true and argument 0.
                state.exited = true;
                state.exitCode = uint8(a0);
                return outputState();
            } else if (syscall_no == sys.SYS_READ) {
                (v0, v1, state.preimageOffset, state.memRoot) = sys.handleSysRead({
                    _a0: a0,
                    _a1: a1,
                    _a2: a2,
                    _preimageKey: state.preimageKey,
                    _preimageOffset: state.preimageOffset,
                    _localContext: _localContext,
                    _oracle: ORACLE,
                    _proofOffset: MIPSMemory.memoryProofOffset(STEP_PROOF_OFFSET, 1),
                    _memRoot: state.memRoot
                });
            } else if (syscall_no == sys.SYS_WRITE) {
                (v0, v1, state.preimageKey, state.preimageOffset) = sys.handleSysWrite({
                    _a0: a0,
                    _a1: a1,
                    _a2: a2,
                    _preimageKey: state.preimageKey,
                    _preimageOffset: state.preimageOffset,
                    _proofOffset: MIPSMemory.memoryProofOffset(STEP_PROOF_OFFSET, 1),
                    _memRoot: state.memRoot
                });
            } else if (syscall_no == sys.SYS_FCNTL) {
                (v0, v1) = sys.handleSysFcntl(a0, a1);
            }

            st.CpuScalars memory cpu = getCpuScalars(state);
            sys.handleSyscallUpdates(cpu, state.registers, v0, v1);
            setStateCpuScalars(state, cpu);

            out_ = outputState();
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
                if iszero(eq(_proof.offset, STEP_PROOF_OFFSET)) {
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
            uint256 insnProofOffset = MIPSMemory.memoryProofOffset(STEP_PROOF_OFFSET, 0);
            uint32 insn = MIPSMemory.readMem(state.memRoot, state.pc, insnProofOffset);
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
                uint256 memProofOffset = MIPSMemory.memoryProofOffset(STEP_PROOF_OFFSET, 1);
                mem = MIPSMemory.readMem(state.memRoot, addr, memProofOffset);
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
                uint256 memProofOffset = MIPSMemory.memoryProofOffset(STEP_PROOF_OFFSET, 1);
                state.memRoot = MIPSMemory.writeMem(storeAddr, memProofOffset, val);
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
